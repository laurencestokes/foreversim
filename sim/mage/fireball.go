package mage

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/spelldata"
)

func (mage *Mage) registerFireballSpell() {
	fireballRank := spellData.Fireball.Highest()
	fireballTick := fireballRank.PeriodicEffect()
	tickLength := fireballTick.Period()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: fireballRank.ID},
		SpellSchool:    fireballRank.SpellSchool(),
		DefenseType:    fireballRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: MageSpellFireball,
		MissileSpeed:   float64(fireballRank.Speed),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(fireballRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      fireballRank.GCD(),
				CastTime: fireballRank.CastTime(),
			},
		},

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "FireballDoT",
			},
			NumberOfTicks:    int32(fireballRank.Duration() / tickLength),
			TickLength:       tickLength,
			BonusCoefficient: fireballTick.Coeff(),
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.Snapshot(target, fireballTick.Average(core.CharacterLevel))
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, periodicTickOutcome(fireballRank, dot))
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: fireballRank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, fireballRank.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
				if result.Landed() {
					spell.Dot(target).Apply(sim)
				}
			})
		},
	})
}

// The outcome a periodic tick rolls: a tick that can crit where the client marks Periodic Can Crit, a
// plain tick otherwise, on the crit table the row's defense type names. A tick never rolls to hit: the
// dot did that once, when it landed. This is shared.PeriodicTickOutcome read off the store's row;
// Spell.TickOutcome is not the same thing, since its magic branches roll the hit again every tick.
func periodicTickOutcome(row *spelldata.Spell, dot *core.Dot) core.OutcomeApplier {
	magic := row.DefenseTypeCore() == core.DefenseTypeMagic
	switch {
	case row.PeriodicCanCrit() && magic:
		return dot.Spell.OutcomeTickMagicCrit
	case row.PeriodicCanCrit():
		return dot.Spell.OutcomeTickPhysicalCrit
	default:
		return dot.OutcomeTick
	}
}
