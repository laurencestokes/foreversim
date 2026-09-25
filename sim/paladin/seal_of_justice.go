package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

// Seal of Justice
// https://www.wowhead.com/forever/spell=20164
//
// Fills the Paladin with the spirit of justice for 30 sec, giving each melee attack a chance to
// stun for 2 sec. Only one Seal can be active on the Paladin at any one time.
//
// Unleashing this Seal's energy will judge an enemy for 10 sec, preventing them from fleeing.
// Your melee strikes will refresh the spell's duration. Only one Judgement per Paladin can be
// active at any one time.
//
// Raid bosses are immune to the stun and never flee, so the seal is the aura and the judgement
// is the debuff, nothing more. Twist of Light still names it, so replacing it leaves an Echo of
// Justice with nothing to replay.
func (paladin *Paladin) registerSealOfJustice(row shared.SpellData) {
	judgementRow := spellData.SealOfJusticeTriggered.BySpellID(int32(effectAt(row, 2).Value))
	judgementAuras := paladin.newJudgementAuras(func(target *core.Unit) *core.Aura {
		return target.GetOrRegisterAura(core.Aura{
			Label:    "Judgement of Justice",
			ActionID: core.ActionID{SpellID: judgementRow.SpellID},
			Tag:      buffs.JudgementAuraTag,
			Duration: judgementRow.Duration,
		})
	})

	// Melee in SpellCategories and Always Hit: the debuff lands without a roll.
	judgement := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: judgementRow.SpellID},
		SpellSchool:    judgementRow.SpellSchool,
		DefenseType:    judgementRow.DefenseType,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagMeleeMetrics,
		ClassSpellMask: SpellMaskJudgementOfJustice,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
			judgementAuras.Get(target).Activate(sim)
		},

		RelatedAuraArrays: judgementAuras.ToMap(),
	})

	aura := paladin.makeSealExclusive(paladin.RegisterAura(core.Aura{
		Label:    sealLabel("Seal of Justice", paladin, row),
		ActionID: core.ActionID{SpellID: row.SpellID},
		Duration: sealDuration,
	}))

	paladin.registerSealSpell(&sealConfig{
		row:       row,
		classMask: SpellMaskSealOfJustice,
		aura:      aura,
		judgement: judgement,
		echoID:    echoOfJusticeID,
	})
}
