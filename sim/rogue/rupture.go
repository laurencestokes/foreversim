package rogue

import (
	"github.com/wowsims/forever/sim/core"
)

// Was the TBC rank-6 id 26867, which the level squish removed. Derived from the table so
// it follows the data instead of naming a rank that may not exist.
var RuptureSpellID = spellData.Rupture.Highest().ID

var ruptureRank = spellData.Rupture.ByID(RuptureSpellID)

func (rogue *Rogue) registerRupture() {
	tick := ruptureRank.PeriodicEffect()
	tickLength := tick.Period()
	tickDamage := tick.Average(core.CharacterLevel)
	baseTickCount := int32(ruptureRank.Duration() / tickLength)

	// The beta client cut the per combo point step with the tick (rank 6: 60 + 8 -> 35 + 4.73).
	// The table carries the 35; the step sits on a dummy effect the generator reads as 0.
	const damagePerComboPoint = 4.73
	apCoeffByComboPoint := []float64{0, 0.01, 0.02, 0.03, 0.03, 0.03}

	// Combo points spent and the Hemorrhage bonus are fixed at cast; only the attack power share
	// is dynamic, so both are captured here and reused by OnTick to rebuild the raw base with
	// current attack power every tick.
	var comboPointsAtCast int32
	var hemoMultiplierAtCast float64
	rawBaseDamage := func(meleeAttackPower float64) float64 {
		return (tickDamage + damagePerComboPoint*float64(comboPointsAtCast) + apCoeffByComboPoint[comboPointsAtCast]*meleeAttackPower) * hemoMultiplierAtCast
	}

	rogue.Rupture = rogue.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: ruptureRank.ID},
		SpellSchool:    ruptureRank.SpellSchool(),
		DefenseType:    ruptureRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | SpellFlagFinisher | core.SpellFlagAPL,
		MetricSplits:   6,
		ClassSpellMask: RogueSpellRupture,
		MaxRange:       core.MaxMeleeRange,

		EnergyCost: core.EnergyCostOptions{
			Cost:          int32(ruptureRank.Cost()),
			Refund:        ruptureRank.MissRefund(),
			RefundMetrics: rogue.EnergyRefundMetrics,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: ruptureRank.GCD(),
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

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label: "Rupture",
				Tag:   RogueBleedTag,
			},
			NumberOfTicks: 0, // Set dynamically
			TickLength:    tickLength,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				comboPointsAtCast = rogue.ComboPoints()
				hemoMultiplierAtCast = core.TernaryFloat64(rogue.isHemorrhaging(target), HemorrhageRuptureMultiplier, 1)
				dot.SnapshotPhysical(target, rawBaseDamage(dot.Spell.MeleeAttackPower(target)))
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.SnapshotRawBaseDamage = rawBaseDamage(dot.Spell.MeleeAttackPower(target))
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, ruptureRank.TickOutcome(dot))
			},
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if result.Landed() {
				dot := spell.Dot(target)
				dot.BaseTickCount = baseTickCount + rogue.ComboPoints()
				dot.Apply(sim)
				rogue.ApplyFinisher(sim, spell)
			} else {
				spell.IssueRefund(sim)
			}
			spell.DealOutcome(sim, result)
		},
	})
}
