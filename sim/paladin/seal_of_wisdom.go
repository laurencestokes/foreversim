package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

// The mana each rank restores on a hit, and the mana its judgement grants attackers. Neither
// carries the seal's rank subtext, so the pairing is by hand.
var sealOfWisdomProcIDs = map[int32]int32{1: 20168, 2: 20350, 3: 20351}
var judgementOfWisdomManaIDs = map[int32]int32{1: 20268, 2: 20352, 3: 20353}

// Seal of Wisdom
// https://www.wowhead.com/forever/spell=20357
//
// Fills the Paladin with divine wisdom for 30 sec, giving each melee attack a chance to restore
// mana to the Paladin. Only one Seal can be active on the Paladin at any one time.
//
// Unleashing this Seal's energy will judge an enemy for 40 sec, granting attacks and spells used
// against the judged enemy a chance to restore mana to the attacker. Your melee strikes will
// refresh the spell's duration. Only one Judgement per Paladin can be active at any one time.
func (paladin *Paladin) registerSealOfWisdom(row shared.SpellData) {
	judgementID := int32(effectAt(row, 2).Value)
	manaRow := spellData.SealOfWisdomTriggered.BySpellID(judgementOfWisdomManaIDs[row.Rank])
	judgementAuras := paladin.newJudgementAuras(func(target *core.Unit) *core.Aura {
		return buffs.JudgementOfWisdomRankAura(target, buffs.JudgementRank{
			SpellID: judgementID,
			Rank:    row.Rank,
			Value:   shared.SpellDataMin(manaRow.Energize),
		})
	})

	// Melee in SpellCategories and Always Hit: the debuff lands without a roll.
	judgement := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: judgementID},
		SpellSchool:    core.SpellSchoolHoly,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagMeleeMetrics,
		ClassSpellMask: SpellMaskJudgementOfWisdom,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
			judgementAuras.Get(target).Activate(sim)
		},

		RelatedAuraArrays: judgementAuras.ToMap(),
	})

	mana := shared.SpellDataMin(row.Energize)
	procID := core.ActionID{SpellID: sealOfWisdomProcIDs[row.Rank]}
	manaMetrics := paladin.NewManaMetrics(procID)
	procSpell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       procID,
		SpellSchool:    core.SpellSchoolHoly,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagHelpful | core.SpellFlagPassiveSpell | core.SpellFlagProc, // The restores lack Not a Proc.
		ClassSpellMask: SpellMaskSealOfWisdomProc,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			spell.Unit.AddMana(sim, mana, manaMetrics)
		},
	})

	aura := paladin.makeSealExclusive(paladin.MakeProcTriggerAura(core.ProcTrigger{
		Name:            sealLabel("Seal of Wisdom", paladin, row),
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
		classMask: SpellMaskSealOfWisdom,
		aura:      aura,
		judgement: judgement,
	})
}
