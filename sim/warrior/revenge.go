package warrior

import (
	"time"

	"github.com/wowsims/forever/sim/core"
)

func (warrior *Warrior) registerRevenge() {
	// TODO: Manual review needed -- spell 25288 states "a high amount of threat" with no number;
	// none is modelled until measured in game.
	revengeRank := spellData.Revenge.Highest()

	actionID := core.ActionID{SpellID: revengeRank.ID}

	// TODO: In-game test needed
	aura := warrior.RegisterAura(core.Aura{
		Label:    "Revenge",
		Duration: 5 * time.Second,
		ActionID: actionID,
	})

	warrior.MakeProcTriggerAura(core.ProcTrigger{
		Name:               "Revenge - Trigger",
		TriggerImmediately: true,
		Outcome:            core.OutcomeBlock | core.OutcomeDodge | core.OutcomeParry,
		Callback:           core.CallbackOnSpellHitTaken,
		Handler: func(sim *core.Simulation, spell *core.Spell, result *core.SpellResult) {
			aura.Activate(sim)
		},
	})

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:       actionID,
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics | core.SpellFlagAPL,
		ClassSpellMask: SpellMaskRevenge,
		MaxRange:       core.MaxMeleeRange,

		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: revengeRank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: cooldownOf(revengeRank),
			},
		},

		RageCost: core.RageCostOptions{
			Cost:   int32(revengeRank.Cost()),
			Refund: revengeRank.MissRefund(),
		},

		DamageMultiplier: 1,
		// Not in the client table; our Classic value until measured in game.
		ThreatMultiplier: 2.25,
		FlatThreatBonus:  2.25 * 2 * 60,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.StanceMatches(DefensiveStance) && aura.IsActive()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			// Rank 6 rolls 138-168; the generator stores the centre of the range (153), so the range
			// stays ours.
			baseDamage := sim.Roll(138, 168)
			result := spell.CalcAndDealDamage(sim, target, baseDamage, spell.OutcomeMeleeSpecialHitAndCrit)
			aura.Deactivate(sim)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},

		RelatedSelfBuff: aura,
	})
}
