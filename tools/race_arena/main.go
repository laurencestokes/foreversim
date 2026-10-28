// Scores the race arena simulations and writes the committed tier lists.
// Usage: go run ./tools/race_arena <output-directory> [ui/app/race_arena/results.json]
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

// The fight every list is run in: the leaderboard's one target (core.MakeSingleTargetEncounter), at
// each of the runner's fight lengths. defaultSeconds is the length the page opens on.
const (
	defaultSeconds = 180
	targetLevel    = 63
	targetArmor    = 3731
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

// The lists written by tools/race_arena/simulate.
type rawList struct {
	Spec           string          `json:"spec"`
	Class          string          `json:"class"`
	Build          string          `json:"build"`
	Talents        string          `json:"talents"`
	Gear           string          `json:"gear"`
	Rotation       string          `json:"rotation"`
	Consumables    string          `json:"consumables"`
	Target         string          `json:"target"`
	Seconds        int             `json:"seconds"`
	Iterations     int             `json:"iterations"`
	KeptWeaponType bool            `json:"keptWeaponType"`
	WeaponType     string          `json:"weaponType"`
	Races          []rawRow        `json:"races"`
	Setup          json.RawMessage `json:"setup"`
}

type rawRow struct {
	Race          string  `json:"race"`
	Commit        string  `json:"commit,omitempty"`
	Dps           float64 `json:"dps"`
	Error         float64 `json:"error"`
	Relabelled    string  `json:"relabelled"`
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
	Spec        string `json:"spec"`
	Class       string `json:"class"`
	Build       string `json:"build"`
	Talents     string `json:"talents"`
	Gear        string `json:"gear"`
	Rotation    string `json:"rotation"`
	Consumables string `json:"consumables"`
	Target      string `json:"target"`
	// The fight length the list was run at, in seconds.
	Seconds        int  `json:"seconds"`
	KeptWeaponType bool `json:"keptWeaponType,omitempty"`
	// The type every race's weapons were run as, when not the one they ship as.
	WeaponType string `json:"weaponType,omitempty"`
	Races      []race `json:"races"`
}

type output struct {
	Commit      string                     `json:"commit"`
	RaceCommits map[string]string          `json:"raceCommits,omitempty"`
	Generated   time.Time                  `json:"generated"`
	Iterations  int                        `json:"iterations"`
	Fight       fight                      `json:"fight"`
	Tiers       []tier                     `json:"tiers"`
	Noise       noise                      `json:"noise"`
	Lists       []list                     `json:"lists"`
	Setups      map[string]json.RawMessage `json:"setups"`
}

type fight struct {
	// The length the page opens on, and every length the lists were run at.
	Seconds     int   `json:"seconds"`
	Durations   []int `json:"durations"`
	TargetLevel int   `json:"targetLevel"`
	TargetArmor int   `json:"targetArmor"`
}

// How far a race's number can be trusted, as a percentage of its list's best: the largest
// standard error of any race's mean, and twice the standard error of the difference between two
// races that far out, using an independent-errors approximation. A shared seed alone supplies no
// paired covariance measurement.
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
		RaceCommits: make(map[string]string),
		Generated:   time.Now().UTC().Truncate(time.Second),
		Fight:       fight{Seconds: defaultSeconds, TargetLevel: targetLevel, TargetArmor: targetArmor},
		Tiers:       tiers,
		Setups:      make(map[string]json.RawMessage),
	}
	for _, r := range raw {
		key := r.Spec + "|" + r.Build
		for _, row := range r.Races {
			revision := row.Commit
			if revision == "" {
				revision = out.Commit
			}
			if previous, exists := out.RaceCommits[row.Race]; exists && previous != revision {
				fail("%s has results from different simulator revisions; rerun every matchup for that race", row.Race)
			}
			out.RaceCommits[row.Race] = revision
		}
		if len(r.Setup) == 0 || bytes.Equal(r.Setup, []byte("null")) {
			fail("%s: missing setup snapshot", key)
		}
		if previous, exists := out.Setups[key]; exists && !bytes.Equal(previous, r.Setup) {
			fail("%s: targets have different setup snapshots", key)
		}
		out.Setups[key] = r.Setup
		if out.Iterations != 0 && r.Iterations != out.Iterations {
			// The page states one iteration count and one noise figure for everything on it.
			fail("%s / %s ran %d iterations and another list ran %d; rerun them together", r.Spec, r.Build, r.Iterations, out.Iterations)
		}
		out.Iterations = r.Iterations
		if r.Seconds <= 0 {
			fail("%s / %s / %s: no fight length; rerun with the current runner", r.Spec, r.Build, r.Target)
		}
		if !slices.Contains(out.Fight.Durations, r.Seconds) {
			out.Fight.Durations = append(out.Fight.Durations, r.Seconds)
		}
		scored, worst := score(r)
		out.Lists = append(out.Lists, scored)
		out.Noise.MaxError = max(out.Noise.MaxError, worst)
	}
	for race, revision := range out.RaceCommits {
		if revision == out.Commit {
			delete(out.RaceCommits, race)
		}
	}
	slices.Sort(out.Fight.Durations)
	if !slices.Contains(out.Fight.Durations, defaultSeconds) {
		out.Fight.Seconds = out.Fight.Durations[0]
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
		Seconds:        r.Seconds,
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
			Racials:    racials(row.Race, r.Class, r.Target),
		})
		worst = max(worst, row.Error/best*100)
	}
	return scored, worst
}

// The racials in play for a race on a list, in a few words. Static text: which pieces a race
// has is fixed, and only the class and the target decide which of them count.
// Survival and utility racials are left out, because nothing here measures them.
func racials(raceName string, class string, target string) string {
	parts := []string{}
	add := func(when bool, part string) {
		if when {
			parts = append(parts, part)
		}
	}
	manaUser := class != "Warrior" && class != "Rogue"
	switch raceName {
	case "Human":
		add(true, "Ruthless (+1% crit)")

		add(manaUser, "The Human Spirit (+10% Spirit)")
	case "Dwarf":
		add(true, "Artisan's Precision (+1% hit)")
		add(target == "Beast", "Big Game Hunter")
	case "Night Elf":
		add(true, "Elune's Light")
	case "Gnome":
		eureka := class == "Warrior" || class == "Rogue" || class == "Warlock" || class == "Mage" || class == "Priest"
		add(eureka, "Eureka!")
		add(true, "Expansive Mind")
	case "Orc":
		add(true, "Blood Fury")
		add(class == "Hunter" || class == "Warlock", "Command (+5% pet damage)")

	case "Undead":
		add(true, "Touch of the Grave")
	case "Troll":
		add(true, "Berserking")
		add(target == "Beast", "Beast Slaying")
	case "Skyborne", "High Order Skyborne", "Windshaper Skyborne":
		add(true, "Wind Blessed (+2% haste)")
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
