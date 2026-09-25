package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/buffs"
	"github.com/wowsims/forever/sim/core/stats"
)

// Seal of the Crusader
// https://www.wowhead.com/forever/spell=20308
//
// Fills the Paladin with the spirit of a crusader for 30 sec, granting melee attack power. The
// Paladin also attacks 40% faster, but deals less damage with each attack. Only one Seal can be
// active on the Paladin at any one time.
//
// Unleashing this Seal's energy will judge an enemy for 40 sec, increasing Holy damage taken.
// Your melee strikes will refresh the spell's duration. Only one Judgement per Paladin can be
// active at any one time.
//
// The client states no number for the damage each attack loses; every sim before this one has
// divided the white damage by the same 1.4 the speed gains, and that stays.
func (paladin *Paladin) registerSealOfTheCrusader(row shared.SpellData) {
	judgementRow := spellData.SealOfTheCrusaderTriggered.BySpellID(int32(effectAt(row, 2).Value))
	judgementAuras := paladin.newJudgementAuras(func(target *core.Unit) *core.Aura {
		return buffs.JudgementOfTheCrusaderAura(target, buffs.JudgementRank{
			SpellID: judgementRow.SpellID,
			Rank:    row.Rank,
			Value:   shared.SpellDataMin(judgementRow.Direct) + paladin.judgementOfTheCrusaderBonus,
		})
	})

	// Melee in SpellCategories and Always Hit: the debuff lands without a roll.
	judgement := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: judgementRow.SpellID},
		SpellSchool:    judgementRow.SpellSchool,
		DefenseType:    judgementRow.DefenseType,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagMeleeMetrics,
		ClassSpellMask: SpellMaskJudgementOfTheCrusader,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
			judgementAuras.Get(target).Activate(sim)
		},

		RelatedAuraArrays: judgementAuras.ToMap(),
	})

	attackPower := row.Effect(shared.A_MOD_ATTACK_POWER, 0).High() + paladin.sealOfTheCrusaderBonusAttackPower
	speed := 1 + row.Effect(shared.A_MOD_ATTACKSPEED, 0).Value/100

	aura := paladin.makeSealExclusive(paladin.RegisterAura(core.Aura{
		Label:    sealLabel("Seal of the Crusader", paladin, row),
		ActionID: core.ActionID{SpellID: row.SpellID},
		Duration: sealDuration,
	}).
		AttachStatBuff(stats.AttackPower, attackPower).
		AttachMultiplyMeleeSpeed(speed).
		AttachSpellMod(core.SpellModConfig{
			ProcMask:   core.ProcMaskMeleeMHAuto,
			Kind:       core.SpellMod_DamageDone_Pct,
			FloatValue: 1/speed - 1,
		}))

	paladin.registerSealSpell(&sealConfig{
		row:       row,
		classMask: SpellMaskSealOfTheCrusader,
		aura:      aura,
		judgement: judgement,
	})
}
