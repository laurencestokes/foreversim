package warlock

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

// The base ramps in thirds: half, the base, then one and a half. Spell power follows the caster on
// every tick and does not ramp. Amplify Curse raises the whole tick for the duration of its cast.
func (warlock *Warlock) registerBaneOfAgony() {
	rankedSpell := spellData.BaneOfAgony.ByID(11713)
	agonyTick := rankedSpell.PeriodicEffect()

	// Each target keeps the Amplify Curse bonus its cast spent. Spell power and damage multipliers
	// follow the warlock on every tick.
	type agonyCast struct{ ampMult float64 }
	castByTarget := map[int32]agonyCast{}

	config := spelldata.SpellConfig(&warlock.Unit, rankedSpell, spelldata.Magic(core.ProcMaskSpellDamage))
	config.ClassSpellMask = WarlockSpellBaneOfAgony

	config.Dot = spelldata.DotConfig(rankedSpell, agonyTick)
	config.Dot.Aura.Tag = "Affliction"

	// A cast now, with Amplify Curse if it is up.
	castNow := func() agonyCast {
		cast := agonyCast{ampMult: 1}
		if warlock.AmplifyCurseAura.IsActive() {
			cast.ampMult = warlock.AmplifyCurseMultiplier
		}
		return cast
	}

	// Dot.Apply snapshots on every application, whatever applied the Bane, and runs inside the cast's
	// effects, before the cast spends Amplify Curse.
	config.Dot.OnSnapshot = func(_ *core.Simulation, target *core.Unit, dot *core.Dot) {
		castByTarget[target.UnitIndex] = castNow()
	}

	// Sets the dot's snapshot to a tick of the cast at the ramp, with the damage multipliers of now.
	setTick := func(target *core.Unit, dot *core.Dot, cast agonyCast, ramp float64) {
		spellPowerShare := dot.BonusCoefficient * dot.Spell.BonusDamage(warlock.AttackTables[target.UnitIndex])
		dot.SnapshotBaseDamage = (agonyTick.Average()*ramp + spellPowerShare) * cast.ampMult
		dot.SnapshotAttackerMultiplier = dot.Spell.AttackerDamageMultiplier(warlock.AttackTables[target.UnitIndex], true) *
			dot.PeriodicDamageMultiplier
	}

	// A dot activated without Apply has no entry, and ticks as if cast now.
	config.Dot.OnTick = func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
		cast, ok := castByTarget[target.UnitIndex]
		if !ok {
			cast = castNow()
		}
		setTick(target, dot, cast, baneOfAgonyRamp(dot.TickCount(), dot.BaseTickCount))
		dot.CalcAndDealPeriodicSnapshotDamage(sim, target, rankedSpell.TickOutcome(dot))
	}

	// Roll to hit and apply as dot in place of any other Bane
	config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
		result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHitNoHitCounter)
		if result.Landed() {
			warlock.deactivateOtherBanes(sim, spell, target)
			spell.Dot(target).Apply(sim)
		}
		spell.DealOutcome(sim, result)
	}

	// Calculate expected damage from the average tick, the base at a ramp of one, which is what a tick is worth
	// over the whole duration. The existing cast keeps its Amplify Curse bonus; either reading uses current
	// spell power. Each tick sets the snapshot again, so setting it here costs a running dot nothing.
	config.ExpectedTickDamage = func(sim *core.Simulation, target *core.Unit, spell *core.Spell, useSnapshot bool) *core.SpellResult {
		dot := spell.Dot(target)
		cast, ok := castByTarget[target.UnitIndex]
		if !useSnapshot || !ok {
			cast = castNow()
		}
		setTick(target, dot, cast, 1)
		return dot.CalcSnapshotDamage(sim, target, spell.OutcomeExpectedMagicAlwaysHit)
	}

	warlock.BaneOfAgony = warlock.RegisterSpell(config)
	warlock.Banes = append(warlock.Banes, warlock.BaneOfAgony)
}

// Helper function to facilitate the percentage of base damage dealt per tick
func baneOfAgonyRamp(tick int32, ticks int32) float64 {
	third := max(ticks/3, 1)
	switch {
	case tick <= third:
		return 0.5
	case tick <= 2*third:
		return 1
	default:
		return 1.5
	}
}
