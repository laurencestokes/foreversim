package main

import (
	"math"
	"testing"
)

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
			{Race: "Orc", Dps: 1000, Error: 0.3, WeaponRacial: true},
			{Race: "Human", Dps: 994, Error: 0.2, Relabelled: "Sword", WeaponRacial: true},
		},
	})
	want := []struct {
		race, tier, racials string
		behind              float64
	}{
		{"Orc", "S", "Blood Fury + axe crit", 0},
		{"Human", "A", "sword crit", 0.6},
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
		weapon              bool
		want                string
	}{
		{"Skyborne", "Mage", "Mechanical", false, "Wind Blessed (+1% haste)"},
		{"Skyborne", "Mage", "Elemental", false, "Wind Blessed (+1% haste) + Elemental Insight"},
		{"Dwarf", "Warrior", "Beast", true, "mace crit + Big Game Hunter"},
		{"Dwarf", "Priest", "Mechanical", false, "base stats only"},
		{"Gnome", "Mage", "Mechanical", false, "Eureka! + Expansive Mind"},
		{"Gnome", "Priest", "Mechanical", false, "Eureka! + Expansive Mind"},
		{"Gnome", "Warlock", "Mechanical", false, "Eureka! + Expansive Mind"},
		{"Human", "Rogue", "Mechanical", false, "base stats only"},
	}
	for _, c := range cases {
		if got := racials(c.race, c.class, c.target, c.weapon); got != c.want {
			t.Errorf("%s %s vs %s: %q, want %q", c.race, c.class, c.target, got, c.want)
		}
	}
}
