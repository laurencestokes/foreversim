package warrior

import (
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/dbcenums"
	"github.com/wowsims/forever/sim/core/proto"
)

func (warrior *Warrior) registerBattleShout() {
	battleShoutRank := spellData.BattleShout.Highest()
	baseAttackPower := battleShoutRank.Effect(dbcenums.A_MOD_ATTACK_POWER, 0).Average(core.CharacterLevel)
	attackPower := func() float64 {
		return baseAttackPower + core.TernaryFloat64(warrior.HasBsT2, core.BattleShoutWrathBonus, 0)
	}

	auras := warrior.NewAllyAuraArray(func(unit *core.Unit) *core.Aura {
		aura := core.BattleShoutAura(unit, true, baseAttackPower, battleShoutRank.Duration())
		aura.BuildPhase = core.Ternary(warrior.DefaultShout == proto.WarriorShout_WarriorShoutBattle, core.CharacterBuildPhaseBuffs, core.CharacterBuildPhaseNone)
		return aura.ApplyOnGain(func(aura *core.Aura, sim *core.Simulation) {
			if ee := aura.ExclusiveEffects[0]; ee.Priority != attackPower() {
				ee.SetPriority(sim, attackPower())
			}
		})
	})
	selfAura := auras.Get(&warrior.Unit)

	warrior.BattleShout = warrior.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: battleShoutRank.ID},
		ClassSpellMask: SpellMaskBattleShout,
		SpellSchool:    battleShoutRank.SpellSchool(),
		Flags:          core.SpellFlagAPL | core.SpellFlagHelpful,
		ProcMask:       core.ProcMaskEmpty,

		RageCost: core.RageCostOptions{
			Cost: int32(battleShoutRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: battleShoutRank.GCD(),
			},
			IgnoreHaste: true,
		},

		ThreatMultiplier: 1,
		// TODO: Manual review needed -- spell 25289 carries no threat effect; none is modelled until
		// measured in game.
		FlatThreatBonus: battleShoutRank.FindEffect(dbcenums.E_THREAT, 0, 0).Average(core.CharacterLevel),

		// Battle Shout is a single-aura exclusive category: a stronger one from another source (the
		// party buff) blocks ours, so casting would only burn rage every GCD.
		ExtraCastCondition: func(sim *core.Simulation, _ *core.Unit) bool {
			active := selfAura.ExclusiveEffects[0].Category.GetActiveEffect()
			return active == nil || active.Priority <= attackPower()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealOutcome(sim, target, spell.OutcomeAlwaysHit)
			// The exclusive check runs before OnGain sets the value, so give it the real one first.
			for _, aura := range auras {
				if aura != nil {
					aura.ExclusiveEffects[0].SetPriority(sim, attackPower())
				}
			}
			auras.ActivateAllPlayers(sim)
		},

		RelatedAuraArrays: auras.ToMap(),
	})
}
