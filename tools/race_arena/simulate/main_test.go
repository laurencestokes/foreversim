package main

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
	gp "google.golang.org/protobuf/proto"
)

func TestRequestUsesArenaBaseline(t *testing.T) {
	player := &proto.Player{Name: "Preset", Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome}
	request := makeRequest(player, 180, proto.MobType_MobTypeMechanical, 50000)
	if request.SimOptions.RandomSeed != 101 || request.SimOptions.Iterations != 50000 || request.Encounter.Duration != 180 {
		t.Fatal("request has a different simulation policy")
	}
	if !gp.Equal(request.Raid.Buffs, core.FullRaidBuffs) || !gp.Equal(request.Raid.Debuffs, core.FullDebuffs) || !gp.Equal(request.Raid.Parties[0].Buffs, core.FullPartyBuffs) || !gp.Equal(request.Raid.Parties[0].Players[0].Buffs, core.FullIndividualBuffs) {
		t.Fatal("request omits full baseline buffs or debuffs")
	}
	if player.Buffs != nil {
		t.Fatal("request construction changed the exported player")
	}
	if !gp.Equal(request.Encounter.Targets[0], core.FreshDefaultTargetConfig()) {
		t.Fatal("request changes the target defaults")
	}
}

func TestFingerprintIgnoresNamesAndStatVectorZeroTails(t *testing.T) {
	first := []byte(`{"raid":{"parties":[{"players":[{"name":"A","bonusStats":{"stats":[1,0],"pseudoStats":[2,0]}}]}]},"encounter":{"targets":[{"name":"Target","stats":[3,0]}]}}`)
	second := []byte(`{"encounter":{"targets":[{"name":"Renamed","stats":[3]}]},"raid":{"parties":[{"players":[{"name":"B","bonusStats":{"stats":[1],"pseudoStats":[2,0,0]}}]}]}}`)
	if fingerprint(first) != fingerprint(second) {
		t.Fatal("display names or zero tails change the fingerprint")
	}
	changed := []byte(`{"raid":{"parties":[{"players":[{"bonusStats":{"stats":[1],"pseudoStats":[2,1]}}]}]},"encounter":{"targets":[{"stats":[3]}]}}`)
	if fingerprint(first) == fingerprint(changed) {
		t.Fatal("a nonzero stat does not change the fingerprint")
	}
}

func TestRequestsKeepReactionRangeAcrossArenaEncounters(t *testing.T) {
	player := &proto.Player{Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc, ReactionTimeMs: 150, ReactionTimeMaxMs: 350}
	for _, seconds := range []int{90, 180, 360} {
		for _, target := range []proto.MobType{proto.MobType_MobTypeMechanical, proto.MobType_MobTypeElemental, proto.MobType_MobTypeBeast} {
			request := makeRequest(player, seconds, target, 50000)
			if request.Encounter.Duration != float64(seconds) || request.Encounter.Targets[0].MobType != target {
				t.Fatalf("request does not use the selected arena encounter: %ds / %s", seconds, target)
			}
			saved := request.Raid.Parties[0].Players[0]
			if saved.ReactionTimeMs != player.ReactionTimeMs || saved.ReactionTimeMaxMs != player.ReactionTimeMaxMs {
				t.Fatal("request changed the preset's reaction-time range")
			}
		}
	}
}

func TestRequestFingerprintChangesWithSimulationSettings(t *testing.T) {
	request := makeRequest(&proto.Player{Class: proto.Class_ClassMage, Race: proto.Race_RaceGnome}, 180, proto.MobType_MobTypeMechanical, 50000)
	first, err := protojson.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	request.Raid.Parties[0].Players[0].Race = proto.Race_RaceHuman
	second, err := protojson.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint(first) == fingerprint(second) {
		t.Fatal("race does not change the fingerprint")
	}
}
