package paladin

import (
	"fmt"

	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
)

// The damage spell each rank fires on a hit. They sit in SealOfFuryTriggered in the order the
// client lists them, not by rank, so the pairing is by hand.
var sealOfFuryProcIDs = map[int32]int32{1: 1311647, 2: 1311654, 3: 20231, 4: 20415, 5: 20416, 6: 20417, 7: 20418}

// Seal of Fury
// https://www.wowhead.com/forever/spell=20423
//
// Fills the Paladin with divine fury for 30 sec, causing melee attacks to deal an additional X
// Holy damage. While a shield is equipped, each attack also grants an absorb shield equal to 50%
// of the Holy damage dealt. Only one Seal can be active on the Paladin at any one time.
//
// Unleashing this Seal's energy causes Holy damage to an enemy and taunts the target to attack
// you for 4 sec.
//
// The row's Direct is the proc spell the tooltip renders the per-hit damage from: a flat number
// with a 10% coefficient, read as the tooltip says. The seal also carries a weapon-speed dummy in
// Seal of Righteousness's shape that the tooltip never references, and it is left alone. The
// taunt has no place in the sim.
func (paladin *Paladin) registerSealOfFury(row shared.SpellData) {
	judgementRow := spellData.SealOfFuryTriggered.BySpellID(int32(effectAt(row, 2).Value))

	// Melee in SpellCategories with No Active Defense: hit and crit on the melee table, never
	// dodged, parried or blocked.
	judgement := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: judgementRow.SpellID},
		SpellSchool:    judgementRow.SpellSchool,
		DefenseType:    judgementRow.DefenseType,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagBinary,
		ClassSpellMask: SpellMaskJudgementOfFury,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: judgementRow.Direct.BonusCoefficient(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, judgementRow.Direct.Damage(sim), spell.OutcomeMeleeSpecialNoBlockDodgeParry)
		},
	})

	shieldShare := effectAt(row, 1).Value / 100
	var pendingShield float64
	shield := paladin.NewDamageAbsorptionAura(core.AbsorptionAuraConfig{
		Aura: core.Aura{
			Label:    fmt.Sprintf("Seal of Fury Shield%s Rank %d", paladin.Label, row.Rank),
			ActionID: core.ActionID{SpellID: row.SpellID}.WithTag(1),
			Duration: sealDuration,
		},
		ShieldStrengthCalculator: func(_ *core.Unit) float64 {
			return pendingShield
		},
	})
	paladin.applyImprovedSealOfFury(shield)

	damage := shared.SpellDataMin(row.Direct)
	procSpell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: sealOfFuryProcIDs[row.Rank]},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		// The damage spells carry the same flags as Seal of Righteousness's: procs themselves,
		// and no Suppress Weapon Procs, so a weapon's "Chance on hit" rolls on the seal's hit too.
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell | core.SpellFlagProc,
		ClassSpellMask: SpellMaskSealOfFuryProc,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: shared.SpellDataCoef(row.Direct),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcDamage(sim, target, damage, spell.OutcomeMeleeSpecialCritOnly)
			sim.AddPendingAction(core.NewDelayedAction(core.DelayedActionOptions{
				DoAt:     sim.CurrentTime + core.SpellBatchWindow,
				Priority: core.ActionPriorityLow,
				OnAction: func(sim *core.Simulation) {
					spell.DealDamage(sim, result)
					if paladin.PseudoStats.CanBlock && result.Damage > 0 {
						pendingShield = result.Damage * shieldShare
						shield.Activate(sim)
					}
				},
			}))
		},
	})

	aura := paladin.makeSealExclusive(paladin.MakeProcTriggerAura(core.ProcTrigger{
		Name:            sealLabel("Seal of Fury", paladin, row),
		ActionID:        core.ActionID{SpellID: row.SpellID},
		MetricsActionID: core.ActionID{SpellID: row.SpellID},
		Duration:        sealDuration,
		Callback:        core.CallbackOnSpellHitDealt,
		ProcMask:        core.ProcMaskMeleeWhiteHit,
		Outcome:         core.OutcomeLanded,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			procSpell.Cast(sim, result.Target)
		},
	}))

	paladin.registerSealSpell(&sealConfig{
		row:       row,
		classMask: SpellMaskSealOfFury,
		aura:      aura,
		judgement: judgement,
		echoID:    echoOfFuryID,
		echo: func(sim *core.Simulation, target *core.Unit) {
			procSpell.Cast(sim, target)
		},
	})
}
