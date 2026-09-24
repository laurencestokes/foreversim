package paladin

import (
	"github.com/wowsims/forever/sim/common/shared"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// The damage spell each rank fires on a hit. They carry no rank subtext, so no table holds them.
var sealOfRighteousnessProcIDs = map[int32]int32{1: 25742, 2: 25740, 3: 25739, 4: 25738, 5: 25737, 6: 25736, 7: 25735, 8: 25713}

// Seal of Righteousness
// https://www.wowhead.com/forever/spell=20293
//
// Fills the Paladin with holy spirit for 30 sec, granting each melee attack an additional X to Y
// Holy damage. Slower weapons cause more Holy damage per swing. Only one Seal can be active on
// the Paladin at any one time.
//
// Unleashing this Seal's energy will cause Holy damage to an enemy.
//
// The row's Direct is the judgement's dummy the seal's tooltip renders each hit from: the number
// and the coefficient the per-hit damage scales on. The hit itself is a function of weapon speed:
// the value per hundred, times the swing speed, times 0.85 for a one-hander or 1.2 for a
// two-hander, the formula the Classic sim settled on from testing.
func (paladin *Paladin) registerSealOfRighteousness(row shared.SpellData) {
	judgeRow := spellData.JudgementOfRighteousness.BySpellID(int32(effectAt(row, 2).Value))

	// The judgement is Melee in SpellCategories and carries No Active Defense: it rolls hit and
	// crit on the melee table and cannot be dodged, parried or blocked.
	judgement := paladin.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: judgeRow.SpellID},
		SpellSchool:    judgeRow.SpellSchool,
		DefenseType:    judgeRow.DefenseType,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagBinary,
		ClassSpellMask: SpellMaskJudgementOfRighteousness,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: judgeRow.Direct.BonusCoefficient(),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, directDamage(sim, judgeRow), spell.OutcomeMeleeSpecialNoBlockDodgeParry)
		},
	})

	value := shared.SpellDataMin(row.Direct)
	procSpell := paladin.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: sealOfRighteousnessProcIDs[row.Rank]},
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial,
		// The damage spells are procs, so auras without Can Proc From Procs never hear them. They
		// carry no Suppress Weapon Procs, unlike their TBC rows: a "Chance on hit" weapon effect
		// rolls on the seal's hit as well as on the swing, and the game shows both procs landing.
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagPassiveSpell | core.SpellFlagProc,
		ClassSpellMask: SpellMaskSealOfRighteousnessProc,

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: shared.SpellDataCoef(row.Direct),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			mh := paladin.MainHand()
			handMultiplier := core.TernaryFloat64(mh.HandType == proto.HandType_HandTypeTwoHand, 1.2, 0.85)
			baseDamage := value / 100 * handMultiplier * mh.SwingSpeed
			result := spell.CalcDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialCritOnly)
			dealAfterBatch(sim, spell, result)
		},
	})

	aura := paladin.makeSealExclusive(paladin.MakeProcTriggerAura(core.ProcTrigger{
		Name:            sealLabel("Seal of Righteousness", paladin, row),
		ActionID:        core.ActionID{SpellID: row.SpellID},
		MetricsActionID: core.ActionID{SpellID: row.SpellID},
		Duration:        sealDuration,
		Callback:        core.CallbackOnSpellHitDealt,
		ProcMask:        core.ProcMaskMeleeWhiteHit,
		Outcome:         core.OutcomeLanded,
		Handler: func(sim *core.Simulation, _ *core.Spell, result *core.SpellResult) {
			procSpell.Cast(sim, result.Target)
		},
	}))

	paladin.registerSealSpell(&sealConfig{
		row:       row,
		classMask: SpellMaskSealOfRighteousness,
		aura:      aura,
		judgement: judgement,
		echoID:    echoOfRighteousnessID,
		echo: func(sim *core.Simulation, target *core.Unit) {
			procSpell.Cast(sim, target)
		},
	})
}
