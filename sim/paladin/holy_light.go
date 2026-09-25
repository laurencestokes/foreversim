package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
)

var HolyLightRankMap = spellData.HolyLight

// What Blessing of Light adds to a Holy Light or Flash of Light on a target that carries it.
var blessingOfLightRow = spellData.GreaterBlessingOfLight.HighestRank()
var blessingOfLightHolyLightBonus = effectAt(blessingOfLightRow, 0).Value
var blessingOfLightFlashOfLightBonus = effectAt(blessingOfLightRow, 1).Value

// Holy Light
// https://www.wowhead.com/forever/spell=25292
//
// Heals a friendly target for 1580.
func (paladin *Paladin) registerHolyLight(row shared.SpellData) {
	paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: row.SpellID},
		SpellSchool:    row.SpellSchool,
		DefenseType:    row.DefenseType,
		ProcMask:       core.ProcMaskSpellHealing,
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ClassSpellMask: SpellMaskHolyLight,
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
			heal := row.Heal.Damage(sim)
			if target.HasActiveAuraWithTag(buffs.GreaterBlessingOfLightCategory) {
				heal += blessingOfLightHolyLightBonus
			}
			spell.CalcAndDealHealing(sim, target, heal, spell.OutcomeHealingCrit)
		},
	})
}
