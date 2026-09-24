package rogue

import (
	"github.com/wowsims/forever/sim/core"
)

var exposeArmorRank = spellData.ExposeArmor.Highest()

// Forever repurposes Improved Expose Armor: the client states an energy cost reduction
// (SPELLMOD_COST -5/-10) and a dummy of 1/2, which our Forever sim reads as combo points handed
// back on a full five point spend. The armor the debuff removes no longer scales with the talent.
func (rogue *Rogue) registerExposeArmorSpell() {
	rogue.ExposeArmorAuras = rogue.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.ExposeArmorAura(target, rogue.ComboPoints)
	})

	cpMetrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: 14169})
	pointsBack := spellData.ImprovedExposeArmor.EffectAt(2).ValueAt(rogue.Talents.ImprovedExposeArmor)

	rogue.ExposeArmor = rogue.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: exposeArmorRank.ID},
		SpellSchool:    exposeArmorRank.SpellSchool(),
		DefenseType:    exposeArmorRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | SpellFlagFinisher | core.SpellFlagAPL,
		MetricSplits:   6,
		ClassSpellMask: RogueSpellExposeArmor,
		MaxRange:       core.MaxMeleeRange,

		EnergyCost: core.EnergyCostOptions{
			Cost:          int32(exposeArmorRank.Cost()),
			Refund:        exposeArmorRank.MissRefund(),
			RefundMetrics: rogue.EnergyRefundMetrics,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: exposeArmorRank.GCD(),
			},
			IgnoreHaste: true,
			ModifyCast: func(sim *core.Simulation, spell *core.Spell, cast *core.Cast) {
				spell.SetMetricsSplit(rogue.ComboPoints())
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return rogue.ComboPoints() > 0 && rogue.CanApplyExposeArmorAura(target)
		},

		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)

			comboPoints := rogue.ComboPoints()
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if result.Landed() {
				rogue.ExposeArmorAuras.Get(target).Activate(sim)
				rogue.ApplyFinisher(sim, spell)
				if pointsBack > 0 && comboPoints == 5 {
					rogue.AddComboPoints(sim, int32(pointsBack), cpMetrics)
				}
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},

		RelatedAuraArrays: rogue.ExposeArmorAuras.ToMap(),
	})
}

// core's Expose Armor debuff is worth 410 armor per combo point. Our Forever sim reads 450 off
// the beta client for the rank the table gives, but the number lives on the shared raid debuff,
// so it is left alone here rather than moved under every other class at the same time.
func (rogue *Rogue) GetExposeArmorValue() float64 {
	return 410.0 * float64(rogue.ComboPoints())
}

func (rogue *Rogue) CanApplyExposeArmorAura(target *core.Unit) bool {
	aura := rogue.ExposeArmorAuras.Get(target)
	if curActive := aura.ExclusiveEffects[0].Category.GetActiveEffect(); curActive != nil {
		return rogue.GetExposeArmorValue() >= curActive.Priority
	}
	return true
}
