package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var wrathRank = spellData.Wrath.Highest()

func (druid *Druid) registerWrathSpell() {
	druid.Wrath = druid.RegisterSpell(Humanoid|Moonkin, core.SpellConfig{
		ActionID:       core.ActionID{SpellID: wrathRank.ID},
		SpellSchool:    wrathRank.SpellSchool(),
		DefenseType:    wrathRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskSpellDamage,
		ClassSpellMask: DruidSpellWrath,
		Flags:          core.SpellFlagAPL,
		MissileSpeed:   float64(wrathRank.Speed),
		Rank:           wrathRank.RankNumber(),

		ManaCost: core.ManaCostOptions{
			FlatCost: int32(wrathRank.Cost()),
		},

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      wrathRank.GCD(),
				CastTime: wrathRank.CastTime(),
			},
		},

		BonusCoefficient: wrathRank.DamageEffect().Coeff(),
		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		MaxRange:         float64(wrathRank.MaxRange),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, wrathRank.DamageEffect().Average(core.CharacterLevel), spell.OutcomeMagicHitAndCrit)

			spell.WaitTravelTime(sim, func(sim *core.Simulation) {
				spell.DealDamage(sim, result)
			})
		},
	})
}
