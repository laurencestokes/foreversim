package warrior

import (
	"github.com/wowsims/forever/sim/core"
)

// TODO: The core Demoralizing Shout aura still takes Booming Voice and Improved Demoralizing
// Shout points. In the client Booming Voice (12321) widens the radius only, the improved talent
// does not exist, and the shout states -205 attack power for 45 seconds (11556). Pending the
// shared shout aura rework, both are passed as 0.
func (warrior *Warrior) registerDemoralizingShout() {
	// TODO: Ingame research needed if this adds flat threat
	demoralizingShoutRank := spellData.DemoralizingShout.Highest()

	warrior.DemoralizingShoutAuras = warrior.NewEnemyAuraArray(func(target *core.Unit) *core.Aura {
		return core.DemoralizingShoutAura(target, 0, 0)
	})

	warrior.DemoralizingShout = warrior.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: demoralizingShoutRank.ID},
		SpellSchool:    demoralizingShoutRank.SpellSchool(),
		DefenseType:    demoralizingShoutRank.DefenseTypeCore(),
		ClassSpellMask: SpellMaskDemoralizingShout,
		ProcMask:       core.ProcMaskEmpty,
		Flags:          core.SpellFlagAPL,

		RageCost: core.RageCostOptions{
			Cost: int32(demoralizingShoutRank.Cost()),
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: demoralizingShoutRank.GCD(),
			},
			IgnoreHaste: true,
		},

		// Not in the client table; our Classic value until measured in game.
		ThreatMultiplier: 0.4,
		FlatThreatBonus:  0.4 * 2 * 54,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			for _, aoeTarget := range sim.Encounter.ActiveTargetUnits {
				result := spell.CalcAndDealOutcome(sim, aoeTarget, spell.OutcomeMagicHit)
				if result.Landed() {
					warrior.DemoralizingShoutAuras.Get(aoeTarget).Activate(sim)
				}
			}
		},

		RelatedAuraArrays: warrior.DemoralizingShoutAuras.ToMap(),
	})
}
