package core

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

func init() {
	RegisterAgentFactory(
		proto.Player_ElementalShaman{},
		proto.Spec_SpecElementalShaman,
		NewFakeElementalShaman,
		func(player *proto.Player, spec interface{}) {
			playerSpec, ok := spec.(*proto.Player_ElementalShaman)
			if !ok {
				panic("Invalid spec value for Elemental Shaman!")
			}
			player.Spec = playerSpec
		},
	)
}

type FakeAgent struct {
	Spell *Spell
	Dot   *Dot
	Character
	Init func()
}

func (fa *FakeAgent) GetCharacter() *Character {
	return &fa.Character
}

func (fa *FakeAgent) Initialize() {
	if fa.Init != nil {
		fa.Init()
	}
}

func (fa *FakeAgent) ApplyTalents()                  {}
func (fa *FakeAgent) Reset(_ *Simulation)            {}
func (fa *FakeAgent) OnGCDReady(_ *Simulation)       {}
func (fa *FakeAgent) OnEncounterStart(_ *Simulation) {}

func NewFakeElementalShaman(char *Character, _ *proto.Player, _ *proto.Raid) Agent {
	fa := &FakeAgent{
		Character: *char,
	}

	fa.Init = func() {
		fa.Spell = fa.RegisterSpell(SpellConfig{
			ActionID:    ActionID{SpellID: 42},
			SpellSchool: SpellSchoolShadow,
			ProcMask:    ProcMaskSpellDamage,
			Flags:       SpellFlagIgnoreResists,
			Cast:        CastConfig{},

			BonusCritPercent: 3,
			DamageMultiplier: 1.5,
			ThreatMultiplier: 1,

			Dot: DotConfig{
				Aura: Aura{
					Label: "fakedot",
				},
				NumberOfTicks:       6,
				TickLength:          time.Second * 3,
				AffectedByCastSpeed: false,
				BonusCoefficient:    1,

				OnSnapshot: func(sim *Simulation, target *Unit, dot *Dot) {
					dot.Snapshot(target, 100)
				},
				OnTick: func(sim *Simulation, target *Unit, dot *Dot) {
					dot.CalcAndDealPeriodicSnapshotDamage(sim, target, dot.OutcomeTick)
				},
			},

			ApplyEffects: func(sim *Simulation, target *Unit, spell *Spell) {
				result := spell.CalcOutcome(sim, target, spell.OutcomeMagicHit)
				if result.Landed() {
					spell.Dot(target).Apply(sim)
				}
				spell.DealOutcome(sim, result)
			},
		})
		fa.Dot = fa.Spell.CurDot()
	}

	return fa
}

func SetupFakeSim() *Simulation {
	sim := NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{
			RandomSeed: 100,
		},
		Raid: &proto.Raid{
			Parties: []*proto.Party{
				{
					Players: []*proto.Player{
						{
							Name:      "Caster",
							Class:     proto.Class_ClassShaman,
							Buffs:     &proto.IndividualBuffs{},
							Spec:      &proto.Player_ElementalShaman{},
							Equipment: &proto.EquipmentSpec{},
						},
					},
					Buffs: &proto.PartyBuffs{},
				},
			},
		},
		Encounter: &proto.Encounter{
			Targets: []*proto.Target{
				{Name: "target", Level: 60, MobType: proto.MobType_MobTypeDemon},
			},
			Duration: 180,
		},
	}, simsignals.CreateSignals())
	sim.Reset()

	return sim
}

func expectDotTickDamage(t *testing.T, sim *Simulation, dot *Dot, expectedDamage float64) {
	damageBefore := dot.Spell.SpellMetrics[0].TotalDamage
	dot.TickOnce(sim)
	damageAfter := dot.Spell.SpellMetrics[0].TotalDamage
	delta := damageAfter - damageBefore

	if !WithinToleranceFloat64(expectedDamage, delta, 0.01) {
		t.Fatalf("Incorrect tick damage applied: Expected: %0.3f, Actual: %0.3f", expectedDamage, delta)
	}
}

func TestDotSnapshot(t *testing.T) {
	sim := SetupFakeSim()
	fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)

	fa.Dot.Apply(sim)
	expectDotTickDamage(t, sim, fa.Dot, 150) // (100) * 1.5
}

// Forever's dots recalculate every tick from the caster's CURRENT spell power/attack power and
// damage bonuses instead of snapshotting them at application (see docs/forever_rules.md and
// core.DynamicDoTs). A spell power buff gained mid-dot should raise the very next tick, and losing
// it again should lower the tick right back down, all without needing to reapply the dot.
func TestDotSnapshotSpellDamage(t *testing.T) {
	sim := SetupFakeSim()
	fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)

	fa.Dot.Apply(sim)
	expectDotTickDamage(t, sim, fa.Dot, 150) // (100) * 1.5

	// A spell power buff gained mid-dot raises the next tick immediately.
	fa.GetCharacter().AddStatDynamic(sim, stats.SpellDamage, 100)
	expectDotTickDamage(t, sim, fa.Dot, 300) // (100 + 100) * 1.5

	// Losing the buff again lowers the tick right back down.
	fa.GetCharacter().AddStatDynamic(sim, stats.SpellDamage, -100)
	expectDotTickDamage(t, sim, fa.Dot, 150) // (100) * 1.5
}

// With DynamicDoTs turned off, behavior matches the old, purely snapshotted engine: a buff gained
// mid-dot has no effect on its ticks until the dot is reapplied.
func TestDotSnapshotSpellDamageDisabled(t *testing.T) {
	DynamicDoTs = false
	defer func() { DynamicDoTs = true }()

	sim := SetupFakeSim()
	fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)

	fa.Dot.Apply(sim)
	expectDotTickDamage(t, sim, fa.Dot, 150) // (100) * 1.5

	// Spell power shouldn't get applied because dot was already snapshot.
	fa.GetCharacter().AddStatDynamic(sim, stats.SpellDamage, 100)
	expectDotTickDamage(t, sim, fa.Dot, 150) // (100) * 1.5

	fa.Dot.Deactivate(sim)
	fa.Dot.Apply(sim)
	expectDotTickDamage(t, sim, fa.Dot, 300) // (100 + 100) * 1.5
}

func TestDotSnapshotSpellMultiplier(t *testing.T) {
	sim := SetupFakeSim()
	fa := sim.Raid.Parties[0].Players[0].(*FakeAgent)
	spell := fa.GetCharacter().Spellbook[0]
	spell.DamageMultiplier *= 2

	fa.Dot.Apply(sim)
	expectDotTickDamage(t, sim, fa.Dot, 300) // (100) * 1.5 * 2
}
