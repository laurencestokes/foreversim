package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

// The heal each rank fires on a hit, and the heal its judgement grants attackers. Neither carries
// the seal's rank subtext, so the pairing is by hand.
var sealOfLightProcIDs = map[int32]int32{1: 20167, 2: 20333, 3: 20334, 4: 20340}
var judgementOfLightHealIDs = map[int32]int32{1: 20267, 2: 20341, 3: 20342, 4: 20343}

// Seal of Light
// https://www.wowhead.com/forever/spell=20349
//
// Fills the Paladin with divine light for 30 sec, giving each melee attack a chance to heal the
// Paladin. Only one Seal can be active on the Paladin at any one time.
//
// Unleashing this Seal's energy will judge an enemy for 40 sec, granting melee attacks made
// against the judged enemy a chance of healing the attacker. Your melee strikes will refresh the
// spell's duration. Only one Judgement per Paladin can be active at any one time.
func (paladin *Paladin) registerSealOfLight(row shared.SpellData) {
	judgementID := int32(effectAt(row, 2).Value)
	healRow := spellData.SealOfLightTriggered.BySpellID(judgementOfLightHealIDs[row.Rank])
	judgementAuras := paladin.newJudgementAuras(func(target *core.Unit) *core.Aura {
		return buffs.JudgementOfLightRankAura(target, buffs.JudgementRank{
			SpellID: judgementID,
			Rank:    row.Rank,
			Value:   shared.SpellDataMin(healRow.Heal),
		})
	})

	// Melee in SpellCategories and Always Hit: the debuff lands without a roll.
	judgement := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: judgementID},
		SpellSchool:    core.SpellSchoolHoly,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagMeleeMetrics,
		ClassSpellMask: SpellMaskJudgementOfLight,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
			judgementAuras.Get(target).Activate(sim)
		},

		RelatedAuraArrays: judgementAuras.ToMap(),
	})

	heal := shared.SpellDataMin(row.Heal)
	procSpell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: sealOfLightProcIDs[row.Rank]},
		SpellSchool:    core.SpellSchoolHoly,
		ProcMask:       core.ProcMaskSpellHealing,
		Flags:          core.SpellFlagHelpful | core.SpellFlagPassiveSpell | core.SpellFlagProc, // The heals lack Not a Proc.
		ClassSpellMask: SpellMaskSealOfLightProc,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealHealing(sim, target, heal, spell.OutcomeAlwaysHit)
		},
	})

	aura := paladin.makeSealExclusive(paladin.MakeProcTriggerAura(core.ProcTrigger{
		Name:            sealLabel("Seal of Light", paladin, row),
		ActionID:        core.ActionID{SpellID: row.SpellID},
		MetricsActionID: core.ActionID{SpellID: row.SpellID},
		Duration:        sealDuration,
		Callback:        core.CallbackOnSpellHitDealt,
		ProcMask:        core.ProcMaskMelee,
		Outcome:         core.OutcomeLanded,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			procSpell.Cast(sim, &paladin.Unit)
		},
	}))

	paladin.registerSealSpell(&sealConfig{
		row:       row,
		classMask: SpellMaskSealOfLight,
		aura:      aura,
		judgement: judgement,
	})
}
