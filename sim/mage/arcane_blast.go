package mage

import (
	"github.com/wowsims/forever/sim/core"
)

func (mage *Mage) registerArcaneBlastSpell() {
	if !mage.Talents.ArcaneBlast {
		return
	}

	arcaneBlastRank := spellData.ArcaneBlast.Highest()

	mage.ArcaneBlast = mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: arcaneBlastRank.ID},
		SpellSchool:    arcaneBlastRank.SpellSchool(),
		DefenseType:    arcaneBlastRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL,
		ClassSpellMask: MageSpellArcaneBlast,

		// The generated row carries no cost: Forever prices it at 15% of base mana (beta client
		// 1.60.1.69893), a PowerCostPct the generator does not read yet.
		ManaCost: core.ManaCostOptions{
			BaseCostPercent: 15,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      arcaneBlastRank.GCD(),
				CastTime: arcaneBlastRank.CastTime(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: arcaneBlastRank.DamageEffect().Coeff(),
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, arcaneBlastRank.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			mage.ArcaneBlastAura.Activate(sim)
			mage.ArcaneBlastAura.AddStack(sim)
		},
	})
}
