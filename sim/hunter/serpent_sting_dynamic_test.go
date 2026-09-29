package hunter

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/stats"
)

func TestSerpentStingDynamicRAP(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: core.SinglePlayerRaidProto(&proto.Player{
			Name: "Hunter", Class: proto.Class_ClassHunter, Race: proto.Race_RaceOrc,
			Equipment: WeaponsOnly, DistanceFromTarget: 30,
			Spec:     hunterSuite("mm", "").SpecOptions.SpecOptions.(*proto.Player_Hunter),
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()
	hunter := sim.Raid.Parties[0].Players[0].(HunterAgent).GetHunter()
	dot := hunter.SerpentSting.Dot(hunter.CurrentTarget)
	dot.Apply(sim)
	base := dot.SnapshotRawBaseDamage
	// 1,000 RAP gained after application must add 35 to the very next tick's
	// raw damage, and losing it must undo that without reapplying the sting.
	for _, tc := range []struct{ change, want float64 }{{1000, base + 35}, {-1000, base}} {
		hunter.AddStatDynamic(sim, stats.RangedAttackPower, tc.change)
		dot.TickOnce(sim)
		if !core.WithinToleranceFloat64(tc.want, dot.SnapshotRawBaseDamage, 1e-6) {
			t.Errorf("tick base after RAP change %v: got %v, want %v", tc.change, dot.SnapshotRawBaseDamage, tc.want)
		}
	}
}
