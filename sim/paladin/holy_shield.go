package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/stats"
)

var HolyShieldRankMap = spellData.HolyShield

// Holy Shield (talent)
// https://www.wowhead.com/forever/spell=20928
//
// Increases chance to block by 20% for 10 sec, and deals 221 Holy damage for each attack blocked
// while active. Damage caused by Holy Shield causes 20% additional threat. Each block expends a
// charge. 4 charges.
func (paladin *Paladin) registerHolyShield(row shared.SpellData) {
	actionID := core.ActionID{SpellID: row.SpellID}
	damage := shared.SpellDataMin(row.Direct)
	blockPercent := row.Effect(shared.A_MOD_BLOCK_PERCENT, 0).Value
	charges := row.ProcCharges

	procSpell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       actionID.WithTag(2),
		SpellSchool:    core.SpellSchoolHoly,
		DefenseType:    core.DefenseTypeMagic,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagNoOnCastComplete | core.SpellFlagPassiveSpell | core.SpellFlagBinary,
		ClassSpellMask: SpellMaskHolyShieldProc,

		DamageMultiplier: 1,
		ThreatMultiplier: 1.2,
		BonusCoefficient: row.Direct.BonusCoefficient(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMagicHit)
		},
	})

	var holyShieldAura *core.Aura
	holyShieldAura = paladin.RegisterAura(core.Aura{
		Label:     "Holy Shield" + paladin.Label + " " + row.GetRankLabel(),
		ActionID:  actionID,
		Duration:  row.Duration,
		MaxStacks: charges,
	}).AttachProcTrigger(core.ProcTrigger{
		Callback:           core.CallbackOnSpellHitTaken,
		Outcome:            core.OutcomeBlock,
		TriggerImmediately: true,
		Handler: func(sim *core.Simulation, spell *core.Spell, _ *core.SpellResult) {
			procSpell.Cast(sim, spell.Unit)
			holyShieldAura.RemoveStack(sim)
		},
	}).AttachStatBuff(stats.BlockPercent, blockPercent)

	// Steadfast Libram: more block value while the shield is up.
	if paladin.holyShieldBlockValueMultiplier != 1 {
		holyShieldAura.AttachMultiplicativePseudoStatBuff(&paladin.PseudoStats.BlockValueMultiplier, paladin.holyShieldBlockValueMultiplier)
	}

	paladin.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    row.SpellSchool,
		DefenseType:    row.DefenseType,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ClassSpellMask: SpellMaskHolyShield,
		Rank:           row.Rank,

		ManaCost: manaCost(row),
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: row.GCD,
			},
			CD: core.Cooldown{
				Timer:    paladin.sharedTimer(&paladin.holyShieldTimer),
				Duration: row.Cooldown,
			},
		},

		ExtraCastCondition: func(_ *core.Simulation, _ *core.Unit) bool {
			return paladin.PseudoStats.CanBlock
		},

		ApplyEffects: func(sim *core.Simulation, _ *core.Unit, _ *core.Spell) {
			holyShieldAura.Activate(sim)
			holyShieldAura.SetStacks(sim, charges)
		},

		RelatedSelfBuff: holyShieldAura,
	})
}
