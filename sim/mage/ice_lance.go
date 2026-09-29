package mage

import (
	"github.com/wowsims/forever/sim/core"
)

// Damage bonus against a target the mage counts as frozen, i.e. one held by Fingers of Frost.
const IceLanceFrozenMultiplier = 4.0

func (mage *Mage) registerIceLanceSpell() {
	if !mage.Talents.IceLance {
		return
	}

	iceLanceRank := spellData.IceLance.Highest()

	// TODO: the client's damage effect carries no spell power coefficient (the row reads 0), like the
	// few other spells whose coefficient moved off the effect row. .143 is our estimate, kept until a
	// beta log settles it. The row is a copy of Season of Discovery's reworked Ice Lance (400640: the
	// same zero, the same 20 and 5 Glaciate dummies), whose scaling a 2024-12-04 hotfix raised by 300%
	// from .143, so .429 or .572 is as likely; ui/sim/spells/mage.json has the level 20 test.
	iceLanceCoefficient := 0.143

	mage.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: iceLanceRank.ID},
		SpellSchool:    iceLanceRank.SpellSchool(),
		DefenseType:    iceLanceRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: MageSpellIceLance,
		MissileSpeed:   float64(iceLanceRank.Speed),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(iceLanceRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: iceLanceRank.GCD(),
			},
		},

		DamageMultiplier: 1,
		BonusCoefficient: iceLanceCoefficient,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, iceLanceRank.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)
			// A bonus on the whole hit rather than the base roll, so spell power is multiplied too.
			if mage.IsTargetFrozen() {
				result.Damage *= IceLanceFrozenMultiplier
			}
			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
