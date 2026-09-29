package rogue

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/stats"
)

func TestRuptureDynamicAPKeepsComboPointsPerTarget(t *testing.T) {
	sim, rogue := subtletySim()
	metrics := rogue.NewComboPointMetrics(core.ActionID{SpellID: RuptureSpellID})
	dots := make([]*core.Dot, 2)
	bases := make([]float64, 2)
	for i, points := range []int32{1, 4} {
		rogue.AddComboPoints(sim, points, metrics)
		dot := rogue.Rupture.Dot(rogue.Env.Encounter.AllTargetUnits[i])
		dot.BaseTickCount = 10
		dot.Apply(sim)
		dots[i], bases[i] = dot, dot.SnapshotRawBaseDamage
	}
	// One target has a one-point Rupture and the other a five-point Rupture.
	// A later AP buff must use each cast's points, not the rogue's current five.
	rogue.AddStatDynamic(sim, stats.AttackPower, 1000)
	for i, extra := range []float64{10, 30} {
		dots[i].TickOnce(sim)
		if got, want := dots[i].SnapshotRawBaseDamage, bases[i]+extra; !core.WithinToleranceFloat64(want, got, 1e-6) {
			t.Errorf("target %d: got %v, want %v", i, got, want)
		}
	}
	rogue.AddStatDynamic(sim, stats.AttackPower, -1000)
	for i, dot := range dots {
		dot.TickOnce(sim)
		if !core.WithinToleranceFloat64(bases[i], dot.SnapshotRawBaseDamage, 1e-6) {
			t.Errorf("target %d retained the expired AP buff", i)
		}
	}
}
