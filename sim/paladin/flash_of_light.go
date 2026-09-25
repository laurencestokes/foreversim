package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

var FlashOfLightRankMap = spellData.FlashOfLight

// Flash of Light
// https://www.wowhead.com/forever/spell=19943
//
// Heals a friendly target for 308.
func (paladin *Paladin) registerFlashOfLight(row shared.SpellData) {
	paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: row.SpellID},
		SpellSchool:    row.SpellSchool,
		DefenseType:    row.DefenseType,
		ProcMask:       core.ProcMaskSpellHealing,
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ClassSpellMask: SpellMaskFlashOfLight,
		Rank:           row.Rank,
		MaxRange:       row.MaxRange,

		ManaCost: manaCost(row),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      row.GCD,
				CastTime: row.CastTime,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: row.Heal.BonusCoefficient(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			heal := row.Heal.Damage(sim) + paladin.flashOfLightBonusHealing
			if target.HasActiveAuraWithTag(buffs.GreaterBlessingOfLightCategory) {
				heal += blessingOfLightFlashOfLightBonus
			}
			spell.CalcAndDealHealing(sim, target, heal, spell.OutcomeHealingCrit)
		},
	})
}
