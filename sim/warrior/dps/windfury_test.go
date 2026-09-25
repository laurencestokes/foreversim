package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// Windfury Totem's proc is a combat enchant on the main hand: a main-hand hit
// that lands can fire it, auto or special, whether or not it deals damage, and
// no other hit can. A main-hand auto that
// procs it has spent the first charge itself and leaves one for the extra
// attack; a special leaves both.
func TestWindfuryTotemProcsOffMainHandHitsWithTheChargesTheTriggerLeaves(t *testing.T) {
	setup := func(t *testing.T) (*core.Simulation, *core.Unit, *core.Aura, *[]int32) {
		t.Helper()

		player := &proto.Player{
			Name:          "Windfury",
			Race:          proto.Race_RaceOrc,
			Class:         proto.Class_ClassWarrior,
			Equipment:     &proto.EquipmentSpec{},
			TalentsString: FuryTalents,
			Spec:          DefaultOptions,
			Rotation:      core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "dps_reck").Rotation,
		}
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{WindfuryTotem: true}, &proto.RaidBuffs{},
				&proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{RandomSeed: 1},
		}, simsignals.CreateSignals())
		sim.Reset()

		war := sim.Raid.Parties[0].Players[0].(*DpsWarrior)
		trigger := war.GetAura("Windfury Totem Trigger")
		proc := war.GetAura("Windfury Totem (External)")
		if trigger == nil || proc == nil {
			t.Fatalf("trigger %v, proc %v; want both registered", trigger, proc)
		}
		if !trigger.IsActive() {
			t.Fatal("the totem's trigger is not up at the pull")
		}

		var started []int32
		proc.ApplyOnStacksChange(func(_ *core.Aura, _ *core.Simulation, oldStacks int32, newStacks int32) {
			if oldStacks == 0 {
				started = append(started, newStacks)
			}
		})
		return sim, sim.Encounter.AllTargetUnits[0], trigger, &started
	}

	// Offers the trigger up to 200 landed hits of one kind and answers whether any of them procced.
	offer := func(sim *core.Simulation, target *core.Unit, trigger *core.Aura, started *[]int32, procMask core.ProcMask, damage float64) bool {
		spell := &core.Spell{ProcMask: procMask}
		for i := 0; i < 200 && len(*started) == 0; i++ {
			trigger.OnSpellHitDealt(trigger, sim, spell, &core.SpellResult{Target: target, Outcome: core.OutcomeHit, Damage: damage})
		}
		return len(*started) > 0
	}

	for _, row := range []struct {
		name     string
		procMask core.ProcMask
		damage   float64
		charges  int32
	}{
		{"a main-hand auto leaves one charge", core.ProcMaskMeleeMHAuto, 100, 1},
		{"a main-hand special leaves both charges", core.ProcMaskMeleeMHSpecial, 100, 2},
		{"a main-hand special that deals no damage leaves both charges", core.ProcMaskMeleeMHSpecial, 0, 2},
	} {
		t.Run(row.name, func(t *testing.T) {
			sim, target, trigger, started := setup(t)
			if !offer(sim, target, trigger, started, row.procMask, row.damage) {
				t.Fatal("200 landed hits never procced the totem")
			}
			if got := (*started)[0]; got != row.charges {
				t.Errorf("the proc started with %d charges, want %d", got, row.charges)
			}
		})
	}

	for _, row := range []struct {
		name     string
		procMask core.ProcMask
	}{
		{"an off-hand auto", core.ProcMaskMeleeOHAuto},
		{"an off-hand special", core.ProcMaskMeleeOHSpecial},
		{"a ranged auto", core.ProcMaskRangedAuto},
		{"a ranged special", core.ProcMaskRangedSpecial},
		{"a spell", core.ProcMaskSpellDamage},
	} {
		t.Run(row.name+" never procs it", func(t *testing.T) {
			sim, target, trigger, started := setup(t)
			if offer(sim, target, trigger, started, row.procMask, 100) {
				t.Errorf("%s procced the totem", row.name)
			}
		})
	}
}
