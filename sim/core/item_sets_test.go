package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

// A set item counts toward the registered set with its set id. An item whose set id no registered set has
// matches by name only where the item carries no set id, so a reissue under the same name takes no bonus.
func TestSetBonusMatchesByNameOnlyWithoutSetID(t *testing.T) {
	const setName = "Test Raiment"
	saved := sets
	sets = []*ItemSet{{ID: 990900, Name: setName, Bonuses: map[int32]ApplySetBonus{2: func(Agent, *Aura) {}}}}
	t.Cleanup(func() { sets = saved })

	for _, tc := range []struct {
		name  string
		setID int32
		want  bool
	}{
		{"registered set id", 990900, true},
		{"no set id", 0, true},
		{"set id no registered set has", 990901, false},
	} {
		var equipment Equipment
		equipment[proto.ItemSlot_ItemSlotHead] = Item{SetName: setName, SetID: tc.setID}
		equipment[proto.ItemSlot_ItemSlotChest] = Item{SetName: setName, SetID: tc.setID}
		if got := equipment.getSetBonuses().ContainsBonus(setName, 2); got != tc.want {
			t.Errorf("%s: 2-piece bonus active = %v, want %v", tc.name, got, tc.want)
		}
	}
}

const collisionSetA, collisionSetB int32 = 990920, 990921

type setCollisionFixture struct {
	sim        *Simulation
	agent      *FakeAgent
	registered map[int32][]*Aura
	procs      map[int32]int
}

func newSetCollisionFixture(t *testing.T, main, swap []int, sameName bool) *setCollisionFixture {
	t.Helper()
	slots := []proto.ItemSlot{proto.ItemSlot_ItemSlotHead, proto.ItemSlot_ItemSlotShoulder, proto.ItemSlot_ItemSlotChest,
		proto.ItemSlot_ItemSlotWrist, proto.ItemSlot_ItemSlotHands, proto.ItemSlot_ItemSlotWaist, proto.ItemSlot_ItemSlotLegs, proto.ItemSlot_ItemSlotFeet}
	types := []proto.ItemType{proto.ItemType_ItemTypeHead, proto.ItemType_ItemTypeShoulder, proto.ItemType_ItemTypeChest,
		proto.ItemType_ItemTypeWrist, proto.ItemType_ItemTypeHands, proto.ItemType_ItemTypeWaist, proto.ItemType_ItemTypeLegs, proto.ItemType_ItemTypeFeet}
	setIDs := []int32{0, collisionSetA, collisionSetB}
	setNames := []string{"", "Collision Raiment", "Collision Raiment"}
	if !sameName {
		setNames[2] = "Other Raiment"
	}
	database := &proto.SimDatabase{}
	for group := range 3 {
		for i := range slots {
			database.Items = append(database.Items, &proto.SimItem{Id: int32(990940 + group*8 + i), Name: fmt.Sprintf("Collision fixture %d/%d", group, i),
				Type: types[i], ArmorType: proto.ArmorType_ArmorTypeCloth, SetId: setIDs[group], SetName: setNames[group], ScalingOptions: map[int32]*proto.ScalingItemProperties{0: {}}})
		}
	}
	previous := sets
	sets = nil
	t.Cleanup(func() {
		sets = previous
		dbMu.Lock()
		defer dbMu.Unlock()
		for _, item := range database.Items {
			delete(itemsByID, item.Id)
		}
	})
	if WITH_DB {
		AddToDatabase(database)
	}
	f := &setCollisionFixture{registered: map[int32][]*Aura{}, procs: map[int32]int{}}
	for group := 1; group <= 2; group++ {
		id := setIDs[group]
		bonuses := map[int32]ApplySetBonus{2: func(Agent, *Aura) {}, 4: func(_ Agent, status *Aura) {
			f.registered[id] = append(f.registered[id], status)
			status.ExposeToAPL(id)
			status.AttachProcTrigger(ProcTrigger{Name: fmt.Sprintf("Collision set %d proc", id), Callback: CallbackOnSpellHitDealt,
				ProcMask: ProcMaskSpellDamage, ProcChance: 1, TriggerImmediately: true,
				Handler: func(_ *Simulation, _ *Spell, _ *SpellResult) { f.procs[id]++ }})
		}}
		NewItemSet(ItemSet{ID: id, Name: setNames[group], Bonuses: bonuses, Slots: slots})
	}
	equipment := func(groups []int) []*proto.ItemSpec {
		items := make([]*proto.ItemSpec, NumItemSlots)
		for i := range items {
			items[i] = &proto.ItemSpec{}
		}
		for i, group := range groups {
			items[slots[i]] = &proto.ItemSpec{Id: int32(990940 + group*8 + i)}
		}
		return items
	}
	player := &proto.Player{Name: "Set Collision", Class: proto.Class_ClassShaman, Race: proto.Race_RaceTroll,
		Equipment: &proto.EquipmentSpec{Items: equipment(main)}, Database: database,
		Spec: &proto.Player_ElementalShaman{}, Rotation: &proto.APLRotation{}}
	if len(swap) > 0 {
		player.EnableItemSwap = true
		player.ItemSwap = &proto.ItemSwap{Items: equipment(swap)}
	}
	f.sim = NewSim(&proto.RaidSimRequest{Raid: SinglePlayerRaidProto(player, nil, nil, nil),
		Encounter: MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{RandomSeed: 1}}, simsignals.CreateSignals())
	f.sim.Reset()
	f.agent = f.sim.Raid.Parties[0].Players[0].(*FakeAgent)
	return f
}

func (f *setCollisionFixture) checkProcs(t *testing.T, a, b bool) {
	t.Helper()
	beforeA, beforeB := f.procs[collisionSetA], f.procs[collisionSetB]
	f.agent.Spell.CalcAndDealDamage(f.sim, f.agent.CurrentTarget, 1, f.agent.Spell.OutcomeAlwaysHit)
	for _, tc := range []struct {
		id     int32
		before int
		active bool
	}{{collisionSetA, beforeA, a}, {collisionSetB, beforeB, b}} {
		want := tc.before
		if tc.active {
			want++
		}
		if f.procs[tc.id] != want {
			t.Errorf("set %d proc count = %d, want %d", tc.id, f.procs[tc.id], want)
		}
		if registered := f.registered[tc.id]; len(registered) > 0 {
			if len(registered) != 1 {
				t.Errorf("set %d registers its four-piece bonus %d times", tc.id, len(registered))
			}
			aura := f.agent.GetAuraByID(ActionID{SpellID: tc.id})
			if aura != registered[0] || aura.IsActive() != tc.active {
				t.Errorf("set %d APL aura does not match its expected active status %v", tc.id, tc.active)
			}
		} else if tc.active {
			t.Errorf("set %d has no registered four-piece bonus", tc.id)
		}
	}
}

func TestSameNameSetBonusesDoNotPoolPiecesOrClaims(t *testing.T) {
	f := newSetCollisionFixture(t, []int{1, 1, 2, 2}, nil, true)
	if f.agent.Equipment.getSetBonuses().ContainsBonus("Collision Raiment", 4) {
		t.Fatal("two pieces of each identified set pool into a four-piece bonus")
	}
	f.checkProcs(t, false, false)

	g := newSetCollisionFixture(t, []int{2, 2, 2, 2}, nil, true)
	character := &Character{Equipment: g.agent.Equipment}
	if character.CouldHaveSetBonus(sets[0], 4) {
		t.Fatal("one identified set claims the other set's four-piece bonus")
	}
	if !character.CouldHaveSetBonus(sets[1], 4) {
		t.Fatal("the equipped set loses its four-piece bonus")
	}
	if !g.agent.Equipment.getSetBonuses().ContainsBonus("Collision Raiment", 4) {
		t.Fatal("name-only lookup loses the equipped bonus")
	}
}

func TestSameNameSetBonusesSwapByIdentity(t *testing.T) {
	for _, tc := range []struct {
		name             string
		main, swap       []int
		initial, swapped [2]bool
	}{
		{"both four-piece bonuses", []int{1, 1, 1, 1, 2, 2, 2, 2}, []int{0, 0, 0, 0}, [2]bool{true, true}, [2]bool{false, true}},
		{"four-piece into the other four-piece", []int{1, 1, 1, 1}, []int{2, 2, 2, 2}, [2]bool{true, false}, [2]bool{false, true}},
		{"two swapped pieces join two unchanged pieces", []int{1, 1, 2, 2}, []int{2, 2}, [2]bool{}, [2]bool{false, true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSetCollisionFixture(t, tc.main, tc.swap, true)
			f.checkProcs(t, tc.initial[0], tc.initial[1])
			f.sim.CurrentTime = -time.Second
			f.agent.ItemSwap.SwapItems(f.sim, proto.APLActionItemSwap_Swap1, false)
			f.checkProcs(t, tc.swapped[0], tc.swapped[1])
			f.agent.ItemSwap.SwapItems(f.sim, proto.APLActionItemSwap_Main, false)
			f.checkProcs(t, tc.initial[0], tc.initial[1])
			f.sim.Cleanup()
			f.sim.Reset()
			f.checkProcs(t, tc.initial[0], tc.initial[1])
		})
	}
}

func TestDistinctSetNamesKeepTheirStatusLabels(t *testing.T) {
	f := newSetCollisionFixture(t, []int{1, 1, 1, 1, 2, 2, 2, 2}, nil, false)
	f.checkProcs(t, true, true)
	if f.registered[collisionSetA][0].Label != "Collision Raiment 4P" || f.registered[collisionSetB][0].Label != "Other Raiment 4P" {
		t.Fatal("unrelated set status labels changed")
	}
}
