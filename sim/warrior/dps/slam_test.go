package dps

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Build 70009: Slam's cooldown is 18s and Improved Slam 2/2 takes 3s off it.
func TestImprovedSlamShortensSlamCooldown(t *testing.T) {
	for talents, want := range map[string]time.Duration{
		ArmsTalents:              15 * time.Second,
		"32305213132515001-5502": 18 * time.Second, // Improved Slam untaken
	} {
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid: core.SinglePlayerRaidProto(&proto.Player{
				Race: proto.Race_RaceOrc, Class: proto.Class_ClassWarrior,
				Equipment: &proto.EquipmentSpec{}, TalentsString: talents, Spec: DefaultOptions,
			}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{RandomSeed: 1},
		}, simsignals.CreateSignals())
		sim.Reset()
		slam := sim.Raid.Parties[0].Players[0].(*DpsWarrior).GetSpell(core.ActionID{SpellID: 11605})
		if slam == nil || slam.CD.Duration != want {
			t.Errorf("%s: Slam cooldown %v, want %v", talents, slam.CD.Duration, want)
		}
	}
}
