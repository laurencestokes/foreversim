package hunter

import (
	"github.com/wowsims/forever/sim/core"
)

func (hunter *Hunter) registerSerpentStingSpell() {
	rank := spellData.SerpentSting.Highest()
	tick := rank.PeriodicEffect()
	tickLength := tick.Period()
	tickDamage := tick.Average(core.CharacterLevel)
	numberOfTicks := int32(rank.Duration() / tickLength)

	// The beta client carries no spell power coefficient on Serpent Sting at all, so Classic's
	// stands: the full-duration 1.0 split across the ticks.
	spellCoeff := 1.0 / float64(numberOfTicks)

	// forever-hunter beta-changes wiki (2026-09-23): Serpent Sting scales with 15% of ranged attack
	// power over its full duration - 3% a tick over 5 ticks - and, being a dynamic dot, each tick
	// reads current RAP rather than what the hunter had when it landed.
	// https://github.com/classic-hunter/forever-hunter/wiki/Forever-Beta-Changes
	const rapCoeffPerTick = 0.03

	hunter.SerpentSting = hunter.RegisterRangedSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: rank.ID},
		SpellSchool:    rank.SpellSchool(),
		DefenseType:    rank.DefenseTypeCore(),
		ClassSpellMask: HunterSpellSerpentSting,
		ProcMask:       core.ProcMaskRangedSpecial,
		Flags:          core.SpellFlagAPL | core.SpellFlagPoison,
		MissileSpeed:   float64(rank.Speed),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(rank.Cost()),
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Serpent Sting",
				Tag:   "Sting",
			},
			NumberOfTicks:    numberOfTicks,
			TickLength:       tickLength,
			BonusCoefficient: spellCoeff,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.Snapshot(target, tickDamage+rapCoeffPerTick*dot.Spell.RangedAttackPower(target))
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				// The RAP share isn't spell power, so it doesn't go through BonusCoefficient/
				// BonusDamage; refresh the raw base by hand each tick so it tracks current RAP.
				dot.SnapshotRawBaseDamage = tickDamage + rapCoeffPerTick*dot.Spell.RangedAttackPower(target)
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, rank.TickOutcome(dot))
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeRangedHitNoHitCounter)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealOutcome(sim, result)

				// Serpent Sting is our only sting, so there is no other sting of ours to knock off; another
				// hunter's stays up beside it (it used to be removed, and two hunters kept undoing each other).
				if result.Landed() {
					spell.Dot(target).Apply(sim)
				}
			})
		},
	})
}
