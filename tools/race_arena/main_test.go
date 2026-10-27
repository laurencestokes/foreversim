package main

import (
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRaceSourceRevisions(t *testing.T) {
	t.Setenv("RACE_ARENA_COMMIT", "original")
	dir := t.TempDir()
	in := filepath.Join(dir, "raw")
	if err := os.Mkdir(in, 0755); err != nil {
		t.Fatal(err)
	}
	rows := []rawList{}
	for _, seconds := range []int{90, 180} {
		rows = append(rows, rawList{Spec: "warrior", Class: "Warrior", Build: "Fury", Seconds: seconds,
			Iterations: 50000, Setup: json.RawMessage(`{}`), Races: []rawRow{
				{Race: "Orc", Dps: 1000}, {Race: "Troll", Dps: 1010, Commit: "updated"},
			}})
	}
	write := func() {
		t.Helper()
		data, err := json.Marshal(rows)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(in, "presets.json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write()
	outPath := filepath.Join(dir, "results.json")
	if output, err := exec.Command("go", "run", ".", in, outPath).CombinedOutput(); err != nil {
		t.Fatalf("scoring failed: %v\n%s", err, output)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var result output
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.Commit != "original" || !reflect.DeepEqual(result.RaceCommits, map[string]string{"Troll": "updated"}) {
		t.Fatalf("source revisions were not preserved: %s, %v", result.Commit, result.RaceCommits)
	}
	rows[1].Races[1].Commit = ""
	write()
	output, err := exec.Command("go", "run", ".", in, outPath).CombinedOutput()
	if err == nil || !strings.Contains(string(output), "different simulator revisions") {
		t.Fatalf("an incomplete race update should fail: %v\n%s", err, output)
	}
}

// The tier edges are the published ones: S within 0.5% of the best, A to 1.0%, B to 1.5%, C to
// 2.5%, D beyond. Each edge belongs to the tier below it in the list ("within 0.5%" includes
// 0.5%), and a race is scored on the two-decimal figure the page prints.
func TestTierFor(t *testing.T) {
	cases := []struct {
		behind float64
		want   string
	}{
		{0, "S"},
		{0.49, "S"},
		{0.5, "S"},
		{0.504, "S"}, // printed as 0.50%
		{0.506, "A"}, // printed as 0.51%
		{0.51, "A"},
		{1.0, "A"},
		{1.01, "B"},
		{1.5, "B"},
		{1.51, "C"},
		{2.5, "C"},
		{2.51, "D"},
		{12, "D"},
	}
	for _, c := range cases {
		if got := tierFor(c.behind); got != c.want {
			t.Errorf("%.3f%% behind: tier %s, want %s", c.behind, got, c.want)
		}
	}
}

// The page draws one row per tier in this order and reads the thresholds from here, so they must
// rise and only the last may be open ended.
func TestTiersRiseAndEndOpen(t *testing.T) {
	previous := -1.0
	for i, tier := range tiers {
		if tier.MaxBehind == nil {
			if i != len(tiers)-1 {
				t.Fatalf("tier %s has no upper edge but is not the last", tier.Name)
			}
			continue
		}
		if *tier.MaxBehind <= previous {
			t.Fatalf("tier %s ends at %.2f%%, not above the tier before it (%.2f%%)", tier.Name, *tier.MaxBehind, previous)
		}
		previous = *tier.MaxBehind
	}
	if tiers[len(tiers)-1].MaxBehind != nil {
		t.Fatal("the last tier must take everything further behind")
	}
}

// Scoring measures every race against the list's best, whatever order the spec wrote them in.
func TestScore(t *testing.T) {
	scored, worst := score(rawList{
		Spec:   "warrior",
		Class:  "Warrior",
		Build:  "Fury 17/34/0",
		Target: "Mechanical",
		Races: []rawRow{
			{Race: "Troll", Dps: 970, Error: 0.2},
			{Race: "Orc", Dps: 1000, Error: 0.3},
			{Race: "Human", Dps: 994, Error: 0.2, Relabelled: "Sword"},
		},
	})
	want := []struct {
		race, tier, racials string
		behind              float64
	}{
		{"Orc", "S", "Blood Fury", 0},
		{"Human", "A", "Ruthless (+1% crit)", 0.6},
		{"Troll", "D", "Berserking", 3},
	}
	for i, w := range want {
		got := scored.Races[i]
		if got.Race != w.race || got.Tier != w.tier || got.Behind != w.behind || got.Racials != w.racials {
			t.Errorf("row %d: %+v, want %s %s %.2f%% %q", i, got, w.race, w.tier, w.behind, w.racials)
		}
	}
	if scored.Races[1].Relabelled != "Sword" {
		t.Error("the relabelled weapon type did not reach the page")
	}
	if math.Abs(worst-0.03) > 1e-9 {
		t.Errorf("largest standard error %.4f%% of the best, want 0.03%%", worst)
	}
}

// Situational racials only count against their own creature type, and a racial whose class has
// no model for it is not claimed.
func TestRacials(t *testing.T) {
	cases := []struct {
		race, class, target string
		want                string
	}{
		{"Skyborne", "Mage", "Mechanical", "Wind Blessed (+2% haste)"},
		{"Skyborne", "Mage", "Elemental", "Wind Blessed (+2% haste) + Elemental Insight"},
		{"Dwarf", "Warrior", "Beast", "Artisan's Precision (+1% hit) + Big Game Hunter"},
		{"Dwarf", "Priest", "Mechanical", "Artisan's Precision (+1% hit)"},
		{"Gnome", "Mage", "Mechanical", "Eureka! + Expansive Mind"},
		{"Gnome", "Priest", "Mechanical", "Eureka! + Expansive Mind"},
		{"Gnome", "Warlock", "Mechanical", "Eureka! + Expansive Mind"},
		{"Human", "Rogue", "Mechanical", "Ruthless (+1% crit)"},
		{"Human", "Mage", "Mechanical", "Ruthless (+1% crit) + The Human Spirit (+10% Spirit)"},
		{"Orc", "Hunter", "Mechanical", "Blood Fury + Command (+5% pet damage)"},
		{"Orc", "Warlock", "Mechanical", "Blood Fury + Command (+5% pet damage)"},
		{"Orc", "Warrior", "Mechanical", "Blood Fury"},
		{"Tauren", "Druid", "Mechanical", "base stats only"},
	}
	for _, c := range cases {
		if got := racials(c.race, c.class, c.target); got != c.want {
			t.Errorf("%s %s vs %s: %q, want %q", c.race, c.class, c.target, got, c.want)
		}
	}
}
