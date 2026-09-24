package warrior

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
)

func (warrior *Warrior) registerSunderArmor() {
	// The client supplies Sunder Armor's flat threat per rank: 405/608/810/1013 for ranks 2-5.
	//
	// TODO: rank 1 reads a flat threat of 1, which looks like placeholder data next to the
	// rest of the ladder. Harmless while this pins the highest rank, but worth confirming.
	sunderArmorRank := spellData.SunderArmor.Highest()

	actionId := core.ActionID{SpellID: sunderArmorRank.ID}

	// core.SunderArmorAura carries the client's rank 5 (450 armor a stack). The id is set to the
	// rank's so an APL can watch the stacks by the spell it casts.
	warrior.SunderArmorAuras = warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		aura := core.SunderArmorAura(target)
		aura.ActionID = actionId
		return aura
	})

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:       actionId,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		ClassSpellMask: SpellMaskSunderArmor,
		MaxRange:       core.MaxMeleeRange,

		RageCost: core.RageCostOptions{
			Cost:   int32(sunderArmorRank.Cost()),
			Refund: sunderArmorRank.MissRefund(),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: sunderArmorRank.GCD(),
			},
			IgnoreHaste: true,
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.CanApplySunderAura(target)
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		FlatThreatBonus:  sunderArmorRank.FindEffect(dbcenums.E_THREAT, 0, 0).Average(core.CharacterLevel),

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcOutcome(sim, target, spell.OutcomeMeleeSpecialHit)

			if result.Landed() {
				aura := warrior.SunderArmorAuras.Get(target)
				aura.Activate(sim)
				aura.AddStack(sim)
			} else {
				spell.IssueRefund(sim)
			}

			spell.DealOutcome(sim, result)
		},

		RelatedAuraArrays: warrior.SunderArmorAuras.ToMap(),
	})
}

func (warrior *Warrior) CanApplySunderAura(target *core.Unit) bool {
	return warrior.SunderArmorAuras.Get(target).IsActive() || !warrior.SunderArmorAuras.Get(target).ExclusiveEffects[0].Category.AnyActive()
}
