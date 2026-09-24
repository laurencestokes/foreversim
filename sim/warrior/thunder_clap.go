package warrior

import (
	"github.com/wowsims/forever/sim/core"
)

func (warrior *Warrior) registerThunderClap() {
	thunderClapRank := spellData.ThunderClap.Highest()
	thunderClapBaseDamage := thunderClapRank.DamageEffect().Average(core.CharacterLevel)
	thunderClapSlow := thunderClapRank.Effects[1].Percent()

	auras := warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		// The priority below rescales core's flat 20% for the Conqueror's set bonus.
		return core.ThunderClapAura(target).ApplyOnGain(func(aura *core.Aura, sim *core.Simulation) {
			// The slow is -20 (a fraction of -0.2) and adds to the time between attacks; Conqueror's
			// 5 piece (26110, all effects +50%) makes it 30%.
			speedMultiplier := 1 - thunderClapSlow*(1+warrior.thunderClapEffectBonus)
			if ee := aura.ExclusiveEffects[0]; ee.Priority != speedMultiplier {
				ee.SetPriority(sim, speedMultiplier)
			}
		})
	})

	warrior.RegisterSpell(core.SpellConfig{
		ActionID:    core.ActionID{SpellID: thunderClapRank.ID},
		SpellSchool: thunderClapRank.SpellSchool(),
		// Thunder Clap is Physical but Magic in SpellCategories: it rolls on the spell hit table
		// (logs show full resists next to armor mitigation) and crits on spell crit chance for
		// 1.5x. Warriors have no base spell crit, so logs without Totem of Wrath show none
		// (0 of 799 landed hits from 6 prot warriors on fresh.warcraftlogs.com, 2026-09-14).
		DefenseType:    thunderClapRank.DefenseTypeCore(),
		ProcMask:       core.ProcMaskRangedSpecial,
		Flags:          core.SpellFlagAPL | core.SpellFlagBinary,
		ClassSpellMask: SpellMaskThunderClap,

		RageCost: core.RageCostOptions{
			Cost: int32(thunderClapRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: thunderClapRank.GCD(),
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: cooldownOf(thunderClapRank),
			},
		},

		DamageMultiplier: 1,
		// Not in the client table; our Classic value until measured in game.
		ThreatMultiplier: 2.5,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.StanceMatches(BattleStance | DefensiveStance)
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			results := spell.CalcCleaveDamage(sim, target, int32(thunderClapRank.MaxTargets), thunderClapBaseDamage, spell.OutcomeMagicHitAndCrit)
			warrior.CastNormalizedSweepingStrikesAttack(results, sim)

			for _, result := range results {
				if result.Landed() {
					auras.Get(result.Target).Activate(sim)
				}
				spell.DealDamage(sim, result)
			}
		},

		RelatedAuraArrays: auras.ToMap(),
	})
}
