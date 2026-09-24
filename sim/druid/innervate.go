package druid

import (
	"github.com/wowsims/forever/sim/core"
)

var innervateRank = spellData.Innervate.Highest()

func (druid *Druid) registerInnervateCD() {
	innervateTarget := druid.GetUnit(druid.SelfBuffs.InnervateTarget)
	if innervateTarget == nil {
		innervateTarget = &druid.Unit
	}
	innervateTargetChar := druid.Env.Raid.GetPlayerFromUnit(innervateTarget).GetCharacter()

	actionID := core.ActionID{SpellID: innervateRank.ID, Tag: druid.Index}

	amount := 0.05
	if innervateTarget == &druid.Unit {
		// TODO: Forever drops Dreamstate; the self-cast base amount stands until we know
		// whether the effect moved onto another talent.
		amount = 0.2
	}

	innervateAura := core.InnervateAura(innervateTargetChar, amount, actionID.Tag)

	innervateSpell := druid.RegisterSpell(Humanoid|Moonkin|Tree, core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    innervateRank.SpellSchool(),
		DefenseType:    innervateRank.DefenseTypeCore(),
		ClassSpellMask: DruidSpellInnervate,
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		MaxRange:       float64(innervateRank.MaxRange),

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: innervateRank.GCD(),
			},
			CD: core.Cooldown{
				Timer:    druid.NewTimer(),
				Duration: max(innervateRank.Cooldown(), innervateRank.CategoryCooldown()),
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			// If the target already has another Innervate, don't cast.
			return !innervateTarget.HasActiveAuraWithTag(core.InnervateAuraTag)
		},
		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			innervateAura.Activate(sim)
		},
	})

	druid.AddMajorCooldown(core.MajorCooldown{
		Spell: innervateSpell.Spell,
		Type:  core.CooldownTypeMana,
		ShouldActivate: func(sim *core.Simulation, character *core.Character) bool {
			// Require manual APL usage.
			return false
		},
	})
}
