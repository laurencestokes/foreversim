package mage

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Frostfire Bolt was added upstream after our Eureka mask was written. Check the
// damage, periodic damage, mana discount and charge use, not just mask membership.
func TestEurekaFrostfireBolt(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: core.SinglePlayerRaidProto(&proto.Player{
			Name: "Gnome", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome,
			Equipment: &proto.EquipmentSpec{},
			Spec:      &proto.Player_Mage{Mage: &proto.Mage{Options: &proto.Mage_Options{ClassOptions: &proto.MageOptions{}}}},
			Rotation:  &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	mage := sim.Raid.Parties[0].Players[0].(MageAgent).GetMage()
	spell := mage.GetSpell(core.ActionID{SpellID: spellData.FrostfireBolt.Highest().ID})
	target := mage.CurrentTarget
	dot := spell.Dot(target)
	dot.Apply(sim)
	baseDamage := spell.CalcDamage(sim, target, 100, spell.OutcomeAlwaysHit).Damage
	baseTick := dot.CalcSnapshotDamage(sim, target, dot.OutcomeTick).Damage
	baseCost := spell.Cost.GetCurrentCost()
	if !mage.GetSpell(core.ActionID{SpellID: 1259817}).Cast(sim, target) {
		t.Fatal("Eureka did not cast")
	}
	for _, tc := range []struct {
		name      string
		got, want float64
	}{
		{"bolt", spell.CalcDamage(sim, target, 100, spell.OutcomeAlwaysHit).Damage, baseDamage * 1.1},
		{"active dot", dot.CalcSnapshotDamage(sim, target, dot.OutcomeTick).Damage, baseTick * 1.1},
		{"mana", spell.Cost.GetCurrentCost(), baseCost * 0.9},
	} {
		if !core.WithinToleranceFloat64(tc.want, tc.got, 1e-6) {
			t.Errorf("%s with Eureka: got %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	if !spell.Cast(sim, target) {
		t.Fatal("Frostfire Bolt did not cast")
	}
	aura := mage.GetAura("Eureka!")
	for sim.CurrentTime < 5*time.Second && aura.GetStacks() == 3 {
		sim.Step()
	}
	if aura.GetStacks() != 2 {
		t.Fatalf("Frostfire Bolt left %d charges, want 2", aura.GetStacks())
	}
	aura.Deactivate(sim)
	if got := dot.CalcSnapshotDamage(sim, target, dot.OutcomeTick).Damage; !core.WithinToleranceFloat64(baseTick, got, 1e-6) {
		t.Errorf("active dot kept Eureka after expiration: got %v, want %v", got, baseTick)
	}
}
