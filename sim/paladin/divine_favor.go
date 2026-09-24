package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

var DivineFavorRankMap = spellData.DivineFavor

// Divine Favor (talent)
// https://www.wowhead.com/forever/spell=20216
//
// When activated, gives your next Flash of Light, Holy Light, or Holy Shock spell a 100% critical
// effect chance.
func (paladin *Paladin) registerDivineFavor() {
	row := DivineFavorRankMap.HighestRank()
	actionID := core.ActionID{SpellID: row.SpellID}

	var divineFavorAura *core.Aura
	divineFavorAura = paladin.RegisterAura(core.Aura{
		Label:    "Divine Favor" + paladin.Label,
		ActionID: actionID,
		Duration: core.NeverExpires,
	}).AttachSpellMod(core.SpellModConfig{
		Kind:       core.SpellMod_BonusCrit_Percent,
		ClassMask:  SpellMaskHealingSpells | SpellMaskHolyShock,
		FloatValue: row.Effect(shared.A_ADD_FLAT_MODIFIER, int32(dbcenums.SPELLMOD_CRITICAL_CHANCE)).Value,
	}).AttachProcTrigger(core.ProcTrigger{
		CanProcFromProcs:   true, // 20216 carries the bit.
		Callback:           core.CallbackOnCastComplete,
		ClassSpellMask:     SpellMaskHealingSpells | SpellMaskHolyShock,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, _ *core.Spell, _ *core.SpellResult) {
			divineFavorAura.Deactivate(sim)
		},
	})

	divineFavor := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    row.SpellSchool,
		DefenseType:    row.DefenseType,
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ClassSpellMask: SpellMaskDivineFavor,

		ManaCost: manaCost(row),
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
			spell.RelatedSelfBuff.Activate(sim)
		},

		RelatedSelfBuff: divineFavorAura,
	})

	paladin.AddMajorCooldown(core.MajorCooldown{
		Spell: divineFavor,
		Type:  core.CooldownTypeDPS,
	})
}
