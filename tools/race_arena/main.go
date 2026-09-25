// Assembles the race arena's per-spec lists into the one file the race tier list page reads,
// and sorts every race on every list into a tier.
//
// Each spec's TestRaceArena (sim/arenalib/race_arena.go) writes its own lists, because `go test`
// runs packages concurrently and a shared file would be a race. This merges them, measures each
// race against its list's best, and records which commit produced the numbers: as with the build
// arena, the committed file is the cache and the commit is its key.
//
// Usage: go run ./tools/race_arena <race-arena-out-dir> [ui/app/race_arena/results.json]
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// The client build the racials were read from (sim/core/racials.go, docs/forever_rules.md). A
// constant rather than something the sim reports, because the sim does not know it either: it is
// where a person read the numbers.
const clientBuild = "1.60.1.70009"

// The fight every list is run in: the leaderboard's one target (core.MakeSingleTargetEncounter).
const (
	fightSeconds = 180
	targetLevel  = 63
	targetArmor  = 3731
)

// A tier and how far behind its list's best race a race may be, in percent, to sit in it. The
// one place the thresholds are set: the page reads them back out of the results file rather than
// keeping a copy that could disagree.
type tier struct {
	Name string `json:"name"`
	// Inclusive. Absent on the last tier, which takes everything further behind.
	MaxBehind *float64 `json:"maxBehind,omitempty"`
}

func upTo(percent float64) *float64 { return &percent }

var tiers = []tier{
	{"S", upTo(0.5)},
	{"A", upTo(1.0)},
	{"B", upTo(1.5)},
	{"C", upTo(2.5)},
	{"D", nil},
}

// The tier for a race this far behind its list's best, in percent. Scored on the figure the page
// prints (two decimals), so a race shown 0.50% behind is never in a different tier from what
// "within 0.5%" says.
func tierFor(behind float64) string {
	shown := roundTo(behind, 2)
	for _, t := range tiers {
		if t.MaxBehind == nil || shown <= *t.MaxBehind {
			return t.Name
		}
	}
	return tiers[len(tiers)-1].Name
}

// Mirrors sim/arenalib.RaceList and RaceRow. Duplicated rather than imported, as tools/arena
// does, because that package is a test harness.
type rawList struct {
	Spec           string   `json:"spec"`
	Class          string   `json:"class"`
	Build          string   `json:"build"`
	Talents        string   `json:"talents"`
	Gear           string   `json:"gear"`
	Rotation       string   `json:"rotation"`
	Consumables    string   `json:"consumables"`
	Target         string   `json:"target"`
	Iterations     int      `json:"iterations"`
	KeptWeaponType bool     `json:"keptWeaponType"`
	WeaponType     string   `json:"weaponType"`
	Races          []rawRow `json:"races"`
}

type rawRow struct {
	Race          string  `json:"race"`
	Dps           float64 `json:"dps"`
	Error         float64 `json:"error"`
	Relabelled    string  `json:"relabelled"`
	WeaponRacial  bool    `json:"weaponRacial"`
	Shipped       float64 `json:"shipped"`
	RelabelledDps float64 `json:"relabelledDps"`
}

// What the page renders.
type race struct {
	Race string  `json:"race"`
	Dps  float64 `json:"dps"`
	// Percent behind the list's best race, two decimals.
	Behind float64 `json:"behind"`
	Tier   string  `json:"tier"`
	// The weapon type the race's weapons were relabelled to, when that beat the gear as shipped.
	Relabelled string `json:"relabelled,omitempty"`
	// Which of the race's racials are in play on this list.
	Racials string `json:"racials"`
}

type list struct {
	Spec           string `json:"spec"`
	Class          string `json:"class"`
	Build          string `json:"build"`
	Talents        string `json:"talents"`
	Gear           string `json:"gear"`
	Rotation       string `json:"rotation"`
	Consumables    string `json:"consumables"`
	Target         string `json:"target"`
	KeptWeaponType bool   `json:"keptWeaponType,omitempty"`
	// The type every race's weapons were run as, when not the one they ship as.
	WeaponType string `json:"weaponType,omitempty"`
	Races      []race `json:"races"`
}

type output struct {
	Commit      string    `json:"commit"`
	Generated   time.Time `json:"generated"`
	Iterations  int       `json:"iterations"`
	ClientBuild string    `json:"clientBuild"`
	Fight       fight     `json:"fight"`
	Tiers       []tier    `json:"tiers"`
	Noise       noise     `json:"noise"`
	Lists       []list    `json:"lists"`
}

type fight struct {
	Seconds     int `json:"seconds"`
	TargetLevel int `json:"targetLevel"`
	TargetArmor int `json:"targetArmor"`
}

// How far a race's number can be trusted, as a percentage of its list's best: the largest
// standard error of any race's mean, and twice the standard error of the difference between two
// races that far out. The second is conservative - every race meets the same random rolls, which
// makes a real difference steadier than two independent errors would suggest.
type noise struct {
	MaxError      float64 `json:"maxError"`
	MaxDifference float64 `json:"maxDifference"`
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fail("usage: go run ./tools/race_arena <race-arena-out-dir> [ui/app/race_arena/results.json]")
	}
	inDir, outPath := os.Args[1], "ui/app/race_arena/results.json"
	if len(os.Args) == 3 {
		outPath = os.Args[2]
	}

	raw := []rawList{}
	entries, err := os.ReadDir(inDir)
	if err != nil {
		fail("%s: %s", inDir, err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var lists []rawList
		read(filepath.Join(inDir, entry.Name()), &lists)
		raw = append(raw, lists...)
	}
	if len(raw) == 0 {
		fail("no lists in %s, so there is nothing to publish", inDir)
	}

	out := output{
		Commit:      commit(),
		Generated:   time.Now().UTC().Truncate(time.Second),
		ClientBuild: clientBuild,
		Fight:       fight{fightSeconds, targetLevel, targetArmor},
		Tiers:       tiers,
	}
	for _, r := range raw {
		if out.Iterations != 0 && r.Iterations != out.Iterations {
			// The page states one iteration count and one noise figure for everything on it.
			fail("%s / %s ran %d iterations and another list ran %d; rerun them together", r.Spec, r.Build, r.Iterations, out.Iterations)
		}
		out.Iterations = r.Iterations
		scored, worst := score(r)
		out.Lists = append(out.Lists, scored)
		out.Noise.MaxError = max(out.Noise.MaxError, worst)
	}
	out.Noise.MaxDifference = roundTo(2*math.Sqrt2*out.Noise.MaxError, 3)
	out.Noise.MaxError = roundTo(out.Noise.MaxError, 3)

	// Grouped by class for the page, then by spec; each spec's lists stay in its page's order.
	sort.SliceStable(out.Lists, func(i, j int) bool {
		a, b := out.Lists[i], out.Lists[j]
		if a.Class != b.Class {
			return a.Class < b.Class
		}
		return a.Spec < b.Spec
	})

	encoded, err := json.MarshalIndent(out, "", "\t")
	if err != nil {
		fail("%s", err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fail("%s", err)
	}
	if err := os.WriteFile(outPath, append(encoded, '\n'), 0o644); err != nil {
		fail("%s", err)
	}
	fmt.Printf("%d lists at %d iterations -> %s\n", len(out.Lists), out.Iterations, outPath)
}

// Measures each race against the list's best and places it in a tier. Returns the list and the
// largest standard error on it, as a percentage of the best.
func score(r rawList) (list, float64) {
	if len(r.Races) == 0 {
		fail("%s / %s / %s has no races", r.Spec, r.Build, r.Target)
	}
	rows := append([]rawRow(nil), r.Races...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Dps > rows[j].Dps })
	best := rows[0].Dps
	if best <= 0 {
		fail("%s / %s / %s: no damage", r.Spec, r.Build, r.Target)
	}

	scored := list{
		Spec:           r.Spec,
		Class:          r.Class,
		Build:          r.Build,
		Talents:        r.Talents,
		Gear:           r.Gear,
		Rotation:       r.Rotation,
		Consumables:    r.Consumables,
		Target:         r.Target,
		KeptWeaponType: r.KeptWeaponType,
		WeaponType:     r.WeaponType,
	}
	worst := 0.0
	for _, row := range rows {
		behind := roundTo((1-row.Dps/best)*100, 2)
		scored.Races = append(scored.Races, race{
			Race:       row.Race,
			Dps:        roundTo(row.Dps, 1),
			Behind:     behind,
			Tier:       tierFor(behind),
			Relabelled: row.Relabelled,
			Racials:    racials(row.Race, r.Class, r.Target, row.WeaponRacial),
		})
		worst = max(worst, row.Error/best*100)
	}
	return scored, worst
}

// The racials in play for a race on a list, in a few words. Static text: which pieces a race
// has is fixed, and only the weapon, the class and the target decide which of them count.
// Survival and utility racials are left out, because nothing here measures them.
func racials(raceName string, class string, target string, weaponRacial bool) string {
	parts := []string{}
	add := func(when bool, part string) {
		if when {
			parts = append(parts, part)
		}
	}
	manaUser := class != "Warrior" && class != "Rogue"
	switch raceName {
	case "Human":
		add(weaponRacial, "sword crit")
		add(manaUser, "+5% Spirit")
	case "Dwarf":
		add(weaponRacial, "mace crit")
		add(target == "Beast", "Big Game Hunter")
	case "Night Elf":
		add(true, "Elune's Light")
	case "Gnome":
		eureka := class == "Warrior" || class == "Rogue" || class == "Warlock" || class == "Mage" || class == "Priest"
		add(eureka, "Eureka!")
		add(true, "Expansive Mind")
	case "Orc":
		add(true, "Blood Fury")
		add(weaponRacial, "axe crit")
	case "Undead":
		add(true, "Touch of the Grave")
	case "Tauren":
		add(true, "Endurance (+1% hit)")
	case "Troll":
		add(true, "Berserking")
		add(target == "Beast", "Beast Slaying")
	case "Skyborne":
		add(true, "Wind Blessed (+1% haste)")
		add(target == "Elemental", "Elemental Insight")
	}
	if len(parts) == 0 {
		return "base stats only"
	}
	return strings.Join(parts, " + ")
}

func roundTo(value float64, places int) float64 {
	scale := math.Pow(10, float64(places))
	return math.Round(value*scale) / scale
}

// The sim the numbers came from. RACE_ARENA_COMMIT names it where git cannot (a container
// mounted over a worktree, whose .git points outside it). Empty when neither can, rather than
// fatal, so the merge still works when somebody runs it by hand.
func commit() string {
	if named := os.Getenv("RACE_ARENA_COMMIT"); named != "" {
		return named
	}
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func read(path string, into any) {
	data, err := os.ReadFile(path)
	if err != nil {
		fail("%s: %s", path, err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		fail("%s: %s", path, err)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
