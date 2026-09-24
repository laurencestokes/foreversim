package rogue

import (
	"github.com/wowsims/forever/sim/core"
)

var eviscerateRank = spellData.Eviscerate.Highest()

func (rogue *Rogue) registerEviscerate() {
	// The generator stores the average of a damage range, so the client's 54-162 arrives as 108.
	// The spread and the per combo point step are the beta client values our Forever sim runs on
	// (rank 9: 54-162 plus 170 per combo point); the table's dummy effect for the step reads 0.
	avgDamage := eviscerateRank.DamageEffect().Average(core.CharacterLevel)
	damageVariance := 108.0
	flatDamage := avgDamage - damageVariance/2
	comboDamageBonus := 170.0 + rogue.DeathmantleBonus

	rogue.Eviscerate = rogue.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: eviscerateRank.ID},
		SpellSchool:    eviscerateRank.SpellSchool(),
		DefenseType:    eviscerateRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | SpellFlagFinisher | core.SpellFlagAPL,
		MetricSplits:   6,
		ClassSpellMask: RogueSpellEviscerate,
		MaxRange:       core.MaxMeleeRange,

		EnergyCost: core.EnergyCostOptions{
			Cost:          int32(eviscerateRank.Cost()),
			Refund:        eviscerateRank.MissRefund(),
			RefundMetrics: rogue.EnergyRefundMetrics,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: eviscerateRank.GCD(),
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(rogue.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0
		},

		DamageMultiplier:         1,
		DamageMultiplierAdditive: 1,
		ThreatMultiplier:         1,

		BonusCoefficient: eviscerateRank.DamageEffect().Coeff(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			comboPoints := float64(rogue.ComboPoints())
			flatBaseDamage := flatDamage + comboDamageBonus*comboPoints

			damage := sim.Roll(flatBaseDamage, flatBaseDamage+damageVariance) +
				0.03*comboPoints*spell.MeleeAttackPower(target)

			result := spell.CalcDamage(sim, target, damage, spell.OutcomeMeleeSpecialHitAndCrit)

			if result.Landed() {
				rogue.ApplyFinisher(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}

			spell.DealDamage(sim, result)
		},
	})
}
