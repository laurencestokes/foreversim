package database

// Re-derives the families below from the client database and asserts the committed table agrees, so
// a hand-edited or stale generated file fails. Covers 8 of 809 families; regenerating and checking
// the diff is empty is the only check that reaches every row.
//
// Skips when tools/database/wowsims.db is absent, which is why CI is unaffected.

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/paladin"
)

const (
	classPaladin = 2
)

type rankFamily struct {
	Name     string
	ClassBit int
	Table    shared.SpellDataTable
}

// TODO: Holy Shock, Avenger's Shield and Vampiric Touch left this gate when their abilities were
// stubbed. Holy Shock's damage and heal chains share one name, which the generator refuses, so its
// table is by hand in sim/paladin/holy_shock.go; the other two still have no ladder the generator
// can read. Restore each row once its ability is implemented.
var rankFamilies = []rankFamily{
	{"Consecration", classPaladin, paladin.ConsecrationRankMap},
	{"Exorcism", classPaladin, paladin.ExorcismRankMap},
	{"Hammer of Wrath", classPaladin, paladin.HammerOfWrathRankMap},
	{"Holy Strike", classPaladin, paladin.HolyStrikeRankMap},
	{"Holy Wrath", classPaladin, paladin.HolyWrathRankMap},
	{"Holy Light", classPaladin, paladin.HolyLightRankMap},
	{"Flash of Light", classPaladin, paladin.FlashOfLightRankMap},
	{"Lay on Hands", classPaladin, paladin.LayOnHandsRankMap},
	{"Light's Vigil", classPaladin, paladin.LightsVigilRankMap},
	{"Holy Shield", classPaladin, paladin.HolyShieldRankMap},
	{"Seal of Righteousness", classPaladin, paladin.SealOfRighteousnessRankMap},
	{"Seal of Command", classPaladin, paladin.SealOfCommandRankMap},
	{"Seal of Light", classPaladin, paladin.SealOfLightRankMap},
	{"Seal of Wisdom", classPaladin, paladin.SealOfWisdomRankMap},
	{"Seal of the Crusader", classPaladin, paladin.SealOfTheCrusaderRankMap},
	{"Seal of Fury", classPaladin, paladin.SealOfFuryRankMap},
	{"Devotion Aura", classPaladin, paladin.DevotionAuraRankMap},
	{"Retribution Aura", classPaladin, paladin.RetributionAuraRankMap},
	{"Fire Resistance Aura", classPaladin, paladin.FireResistanceAuraRankMap},
	{"Frost Resistance Aura", classPaladin, paladin.FrostResistanceAuraRankMap},
	{"Shadow Resistance Aura", classPaladin, paladin.ShadowResistanceAuraRankMap},
}

// One value in the committed table against the same value re-derived from the database.
type comparison struct {
	Family    string
	Rank      int32
	SpellID   int32
	Field     string
	Generated float64
	Derived   float64
	Source    string
}

func (c comparison) ok() bool { return c.Generated == c.Derived }

func TestGeneratedRankTablesMatchTheDatabase(t *testing.T) {
	DatabasePath = "wowsims.db"
	if _, err := os.Stat(DatabasePath); err != nil {
		t.Skipf("no client database at %s - run `make db` from a local WoW install to enable this gate", DatabasePath)
	}

	helper, err := NewDBHelper()
	if err != nil {
		t.Fatalf("opening %s: %v", DatabasePath, err)
	}
	defer helper.Close()
	db := helper.db

	var all []comparison
	for _, fam := range rankFamilies {
		for _, row := range fam.Table {
			all = append(all, compareRow(t, db, fam, row)...)
		}
	}

	var mismatched []comparison
	for _, c := range all {
		if !c.ok() {
			mismatched = append(mismatched, c)
		}
	}
	t.Logf("%d comparisons, %d mismatched", len(all), len(mismatched))

	for _, c := range mismatched {
		t.Errorf("%s rank %d (spell %d) %s: table says %v, the database derives %v (%s)\n"+
			"    regenerate with `go run ./tools/database/gen_spelldata`, or fix DeriveRankAmount",
			c.Family, c.Rank, c.SpellID, c.Field, c.Generated, c.Derived, c.Source)
	}
}

func compareRow(t *testing.T, db *sql.DB, fam rankFamily, row shared.SpellData) []comparison {
	t.Helper()

	spell, candidates, err := RankCandidates(db, row.SpellID, fam.ClassBit)
	if err != nil {
		t.Fatalf("%s rank %d: %v", fam.Name, row.Rank, err)
	}

	base := comparison{Family: fam.Name, Rank: row.Rank, SpellID: row.SpellID}
	var out []comparison

	if row.Cost > 0 || spell.ManaCost.Valid {
		c := base
		c.Field = "Cost"
		c.Generated = float64(row.Cost)
		c.Source = "SpellPower.ManaCost"
		if spell.ManaCost.Valid {
			c.Derived = float64(NormalizePowerCost(int32(spell.ManaCost.Int64), spell.PowerType))
		}
		out = append(out, c)
	}

	// Reporting which effect reproduced each value is the point, not a nicety: it is the evidence the
	// generator's role rules get built from, and it is how a value that turns out to belong to a
	// sibling spell (Holy Shock's damage, Lay on Hands' energize) shows itself.
	coef := 0.0
	if row.Direct != nil {
		out = append(out, matchPair(base, "Direct.Min", "Direct.Max", shared.SpellDataMin(row.Direct), shared.SpellDataMax(row.Direct),
			directCandidates(candidates))...)
		coef = row.Direct.BonusCoefficient()
	}

	if row.Heal != nil {
		out = append(out, matchPair(base, "Heal.Min", "Heal.Max", shared.SpellDataMin(row.Heal), shared.SpellDataMax(row.Heal),
			directCandidates(candidates))...)
		coef = row.Heal.BonusCoefficient()
	}

	if row.Periodic != nil {
		out = append(out, matchTick(base, "DotTickDamage", shared.SpellDataMin(row.Periodic), periodicCandidates(candidates)))
		if row.Periodic.BonusCoefficient() > 0 {
			coef = row.Periodic.BonusCoefficient()
		}
	}

	// Consecration's second tick, and the coefficient that sits on it alone.
	if row.SecondaryPeriodic != nil {
		out = append(out, matchTick(base, "SecondaryTickDamage", shared.SpellDataMin(row.SecondaryPeriodic), periodicCandidates(candidates)))
		if c := row.SecondaryPeriodic.BonusCoefficient(); c > 0 {
			out = append(out, matchCoefficient(base, c, candidates))
		}
	}

	if row.Energize != nil {
		out = append(out, matchPair(base, "Energize", "", shared.SpellDataMin(row.Energize), 0,
			directCandidates(candidates))...)
	}

	if coef > 0 {
		out = append(out, matchCoefficient(base, coef, candidates))
	}

	return out
}

// Holy Shield's per-block damage lives on an aura effect (EffectAura 43) rather than on a damage
// effect, and the generator files it under Direct all the same, so any aura effect is a candidate too.
// A weapon-damage effect's points are the flat part the generator files there as well, which is
// where Holy Strike's Holy damage sits.
func directCandidates(effects []RankEffect) []RankEffect {
	var out []RankEffect
	for _, e := range effects {
		if e.Effect == dbcenums.E_SCHOOL_DAMAGE || e.Effect == dbcenums.E_HEAL || e.Effect == dbcenums.E_ENERGIZE || e.Aura != 0 || IsWeaponDamageEffect(e.Effect) {
			out = append(out, e)
		}
	}
	return out
}

func periodicCandidates(effects []RankEffect) []RankEffect {
	var out []RankEffect
	for _, e := range effects {
		if IsPeriodicAura(e.Aura) {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		out = effects
	}
	return out
}

func matchPair(base comparison, minField, maxField string, genMin, genMax float64, cands []RankEffect) []comparison {
	bestMin, bestMax, bestSrc, found := 0.0, 0.0, "no candidate effect", false
	for _, e := range cands {
		dMin, dMax := DeriveRankAmount(e, e.SpellLevel, e.MaxLevel)
		src := fmt.Sprintf("spell %d effect %d (Effect=%d, Aura=%d)", e.OwnerSpellID, e.Index, e.Effect, e.Aura)
		if dMin == genMin && (genMax == 0 || dMax == genMax) {
			out := []comparison{finishWith(base, minField, genMin, dMin, src)}
			if genMax > 0 {
				out = append(out, finishWith(base, maxField, genMax, dMax, src))
			}
			return out
		}
		// Closest candidate, so a residual reports a near-miss rather than "no match".
		if !found || math.Abs(dMin-genMin) < math.Abs(bestMin-genMin) {
			bestMin, bestMax, bestSrc, found = dMin, dMax, src, true
		}
	}

	out := []comparison{finishWith(base, minField, genMin, bestMin, bestSrc)}
	if genMax > 0 {
		out = append(out, finishWith(base, maxField, genMax, bestMax, bestSrc))
	}
	return out
}

func matchTick(base comparison, field string, genTick float64, cands []RankEffect) comparison {
	bestTick, bestSrc := 0.0, "no periodic effect"
	for _, e := range cands {
		dMin, _ := DeriveRankAmount(e, e.SpellLevel, e.MaxLevel)
		src := fmt.Sprintf("spell %d effect %d (periodic)", e.OwnerSpellID, e.Index)
		if dMin == genTick {
			return finishWith(base, field, genTick, dMin, src)
		}
		if bestSrc == "no periodic effect" || math.Abs(dMin-genTick) < math.Abs(bestTick-genTick) {
			bestTick, bestSrc = dMin, src
		}
	}
	return finishWith(base, field, genTick, bestTick, bestSrc)
}

func matchCoefficient(base comparison, genCoef float64, cands []RankEffect) comparison {
	best, bestSrc := 0.0, "no candidate effect"
	for _, e := range cands {
		if e.Coefficient <= 0 {
			continue
		}
		src := fmt.Sprintf("spell %d effect %d", e.OwnerSpellID, e.Index)
		if e.Coefficient == genCoef {
			return finishWith(base, "Coefficient", genCoef, e.Coefficient, src)
		}
		if bestSrc == "no candidate effect" || math.Abs(e.Coefficient-genCoef) < math.Abs(best-genCoef) {
			best, bestSrc = e.Coefficient, src
		}
	}
	return finishWith(base, "Coefficient", genCoef, best, bestSrc)
}

func finishWith(base comparison, field string, generated, derived float64, source string) comparison {
	c := base
	c.Field = field
	c.Generated = generated
	c.Derived = derived
	c.Source = source
	return c
}
