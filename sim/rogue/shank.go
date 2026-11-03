package rogue

import (
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// The November 2 development notes define Shank's cost, cooldown, weapon share and Energy gain.
// The available game files have no player Shank row, so its action identity belongs to the sim.
// Preview assumptions: the normal one-second Rogue GCD, ordinary off-hand damage and avoidance,
// no miss roll, Energy and Venom on cast, and Boot Knife on a landed attack.
func (rogue *Rogue) registerShank() {
	if rogue.Level < 24 {
		return
	}
	actionID := core.ActionID{OtherID: proto.OtherAction_OtherActionShank}
	rank := int32(1)
	energy := 15.0
	if rogue.Level >= 44 {
		rank = 2
		energy = 25
	}
	energy += []float64{0, 3, 6, 9, 12, 15}[rogue.Talents.FlashOfSteel]
	cooldown := 10*time.Second - []time.Duration{0, 400 * time.Millisecond, 800 * time.Millisecond, 1200 * time.Millisecond, 1600 * time.Millisecond, 2 * time.Second}[rogue.Talents.FlashOfSteel]
	flags := core.SpellFlagAPL | core.SpellFlagMeleeMetrics
	if rogue.Talents.BootKnife > 0 {
		flags |= SpellFlagBuilder
	}
	rogue.Shank = rogue.RegisterSpell(core.SpellConfig{
		ActionID: actionID, Rank: rank, SpellSchool: core.SpellSchoolPhysical, DefenseType: core.DefenseTypeMelee,
		ProcMask: core.ProcMaskMeleeOHSpecial, Flags: flags, ClassSpellMask: RogueSpellShank,
		CastRequirement: core.CastRequirement{Weapon: core.EquipRequirement{Hand: core.ProcMaskMeleeOH}},
		EnergyCost:      core.EnergyCostOptions{Cost: 5},
		Cast:            core.CastConfig{DefaultCast: core.Cast{GCD: time.Second}, CD: core.Cooldown{Timer: rogue.NewTimer(), Duration: cooldown}},
		BonusHitPercent: 100, DamageMultiplier: 0.5, ThreatMultiplier: 1, MaxRange: core.MaxMeleeRange,
		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			rogue.BreakStealth(sim)
			if rogue.VenomAura != nil {
				rogue.VenomAura.Activate(sim)
			}
			result := spell.CalcAndDealDamage(sim, target, rogue.OHWeaponDamage(sim, spell.MeleeAttackPower(target)), spell.OutcomeMeleeWeaponSpecialHitAndCrit)
			rogue.AddEnergy(sim, energy, spell.EnergyMetrics())
			if result.Landed() && rogue.Talents.BootKnife > 0 && sim.Proc([]float64{0, 0.2, 0.4, 0.6, 0.8, 1}[rogue.Talents.BootKnife], "Boot Knife") {
				rogue.AddComboPoints(sim, 2, spell.ComboPointMetrics())
			}
		},
	})
}
