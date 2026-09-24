package hunter

import (
	"time"

	"github.com/wowsims/forever/sim/core"
)

type PetAbilityType int

// Pet AI doesn't use abilities immediately, so model this with a 1.6s GCD.
const PetGCD = time.Millisecond * 1600

const (
	Unknown PetAbilityType = iota

	Bite
	Claw
	LightningBreath
	Screech
	ScorpidPoison
)

func (hp *HunterPet) NewPetAbility(abilityType PetAbilityType) *core.Spell {
	switch abilityType {
	case Bite:
		return hp.newBite()
	case Claw:
		return hp.newClaw()
	case LightningBreath:
		return hp.newLightningBreath()
	case Screech:
		return hp.newScreech()
	case ScorpidPoison:
		return hp.newScorpidPoison()
	case Unknown:
		return nil
	default:
		panic("Invalid pet ability type")
	}
}

func (hp *HunterPet) newBite() *core.Spell {
	return hp.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 17261},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ClassSpellMask: HunterPetDamage,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics,
		MaxRange:       core.MaxMeleeRange,

		FocusCost: core.FocusCostOptions{
			Cost: 35,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: 10 * time.Second,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hp.IsEnabled()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, sim.Roll(81, 99), spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

func (hp *HunterPet) newClaw() *core.Spell {
	return hp.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 3009},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ClassSpellMask: HunterPetDamage,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagMeleeMetrics,
		MaxRange:       core.MaxMeleeRange,

		FocusCost: core.FocusCostOptions{
			Cost: 25,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hp.IsEnabled()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, sim.Roll(43, 59), spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

// Beta client: every rank lower than Classic's, and no more growth per level. Rank 6 is 86-98.
func (hp *HunterPet) newLightningBreath() *core.Spell {
	return hp.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 25012},
		SpellSchool:    core.SpellSchoolNature,
		DefenseType:    core.DefenseTypeMagic,
		ClassSpellMask: HunterPetDamage,
		ProcMask:       core.ProcMaskSpellDamage,
		MaxRange:       20,

		FocusCost: core.FocusCostOptions{
			Cost: 50,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		BonusCoefficient: 1,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hp.IsEnabled()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, sim.Roll(86, 98), spell.OutcomeMagicHitAndCrit)
		},
	})
}

// Demoralizing Screech in the beta client: new damage, and a 10 sec cooldown Classic did not have.
// The attack power reduction it also applies is left out.
func (hp *HunterPet) newScreech() *core.Spell {
	return hp.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 24582},
		SpellSchool:    core.SpellSchoolPhysical,
		DefenseType:    core.DefenseTypeMelee,
		ClassSpellMask: HunterPetDamage,
		ProcMask:       core.ProcMaskMeleeSpecial,
		Flags:          core.SpellFlagMeleeMetrics,
		MaxRange:       core.MaxMeleeRange,

		FocusCost: core.FocusCostOptions{
			Cost: 20,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: time.Second * 10,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hp.IsEnabled()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealDamage(sim, target, sim.Roll(24, 42), spell.OutcomeMeleeSpecialHitAndCrit)
		},
	})
}

// Beta client: 5 a tick at rank 4, down from 8.
func (hp *HunterPet) newScorpidPoison() *core.Spell {
	const baseDamageTick = 5.0

	return hp.RegisterSpell(core.SpellConfig{
		ActionID:       core.ActionID{SpellID: 24587},
		SpellSchool:    core.SpellSchoolNature,
		DefenseType:    core.DefenseTypeMelee,
		ClassSpellMask: HunterPetDamage,
		ProcMask:       core.ProcMaskMeleeMHSpecial,
		Flags:          core.SpellFlagPassiveSpell | core.SpellFlagPoison,
		MaxRange:       core.MaxMeleeRange,

		FocusCost: core.FocusCostOptions{
			Cost: 30,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: PetGCD,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    hp.NewTimer(),
				Duration: time.Second * 4,
			},
		},

		DamageMultiplier: 1,
		ThreatMultiplier: 1,

		Dot: core.DotConfig{
			Aura: core.Aura{
				Label:     "Scorpid Poison",
				MaxStacks: 5,
				Duration:  time.Second * 10,
			},
			NumberOfTicks: 5,
			TickLength:    time.Second * 2,

			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				// Each stack adds another flat tick; the multiplier no longer needs to special-case
				// the first stack since SnapshotPhysical keeps it live every tick regardless (see
				// core.Dot.computeSnapshot).
				rawBase := baseDamageTick
				if dot.GetStacks() > 1 {
					rawBase += dot.SnapshotRawBaseDamage
				}
				dot.SnapshotPhysical(target, rawBase)
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
			},
		},

		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return hp.IsEnabled()
		},

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			result := spell.CalcAndDealOutcome(sim, target, spell.OutcomeMeleeSpecialHit)
			if !result.Landed() {
				return
			}

			dot := spell.Dot(target)
			dot.Apply(sim)
			if dot.GetStacks() < dot.MaxStacks {
				dot.AddStack(sim)
				dot.TakeSnapshot(sim)
			}
		},
	})
}
