package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

var SwiftJudgementRankMap = spellData.SwiftJudgement

// Swift Judgement (talent)
// https://www.wowhead.com/forever/spell=1310994
//
// Finishes the remaining cooldown on your Judgement ability and reduces the Mana cost of your next
// Judgement by 100%.
func (paladin *Paladin) registerSwiftJudgement() {
	row := SwiftJudgementRankMap.HighestRank()
	actionID := core.ActionID{SpellID: row.SpellID}

	var freeJudgement *core.Aura
	freeJudgement = paladin.RegisterAura(core.Aura{
		Label:    "Swift Judgement" + paladin.Label,
		ActionID: actionID,
		Duration: core.NeverExpires,
	}).AttachSpellMod(core.SpellModConfig{
		Kind:       core.SpellMod_PowerCost_Pct_Add,
		ClassMask:  SpellMaskJudgement,
		FloatValue: row.Effect(shared.A_ADD_PCT_MODIFIER, int32(dbcenums.SPELLMOD_COST)).Value / 100,
	}).AttachProcTrigger(core.ProcTrigger{
		Callback:           core.CallbackOnCastComplete,
		ClassSpellMask:     SpellMaskJudgement,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			freeJudgement.Deactivate(sim)
		},
	})

	spell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    row.SpellSchool,
		DefenseType:    row.DefenseType,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ClassSpellMask: SpellMaskSwiftJudgement,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				NonEmpty: true,
			},
			CD: core.Cooldown{
				Timer:    paladin.NewTimer(),
				Duration: row.Cooldown,
			},
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			paladin.Judgement.CD.Reset()
			spell.RelatedSelfBuff.Activate(sim)
		},

		RelatedSelfBuff: freeJudgement,
	})

	// Our sim fires it as a cooldown, whenever there is a Judgement to finish and a seal to spend
	// the free cast on.
	paladin.AddMajorCooldown(core.MajorCooldown{
		Spell: spell,
		Type:  core.CooldownTypeDPS,
		ShouldActivate: func(sim *core.Simulation, _ *core.Character) bool {
			return !paladin.Judgement.CD.IsReady(sim) && paladin.activeSeal() != nil
		},
	})
}
