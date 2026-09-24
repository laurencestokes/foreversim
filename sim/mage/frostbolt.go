package mage

import (
	"github.com/wowsims/forever/sim/core"
)

func (mage *Mage) registerFrostboltSpell() {
	frostboltRank := spellData.Frostbolt.Highest()

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: frostboltRank.ID},
		SpellSchool:    frostboltRank.SpellSchool(),
		DefenseType:    frostboltRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: MageSpellFrostbolt,
		MissileSpeed:   float64(frostboltRank.Speed),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(frostboltRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      frostboltRank.GCD(),
				CastTime: frostboltRank.CastTime(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: frostboltRank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, frostboltRank.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
