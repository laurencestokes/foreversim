package database

import (
	"slices"
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/tools/database/dbc"
)

func keepConsumableLists(t *testing.T) {
	allow, deny, overrides := ConsumableAllowList, ConsumableDenyList, ConsumableOverrides
	t.Cleanup(func() {
		ConsumableAllowList, ConsumableDenyList, ConsumableOverrides = allow, deny, overrides
	})
}

func TestAnAllowListedOrOverriddenConsumableShipsUnsupported(t *testing.T) {
	withDBCInputs(t)
	keepConsumableLists(t)

	blastingCharge := dbc.GetDBC().Consumables[221325]
	routing := RouteConsumable(&blastingCharge, dbc.GetDBC())
	if routing.Supported() {
		t.Fatal("221325 routes as supported, so this proves nothing")
	}
	if ShipsConsumable(routing) {
		t.Error("221325 ships while nothing names it")
	}

	ConsumableAllowList = append(slices.Clone(ConsumableAllowList), 221325)
	if !ShipsConsumable(routing) {
		t.Error("221325 is dropped although ConsumableAllowList names it")
	}

	ConsumableAllowList = slices.DeleteFunc(slices.Clone(ConsumableAllowList), func(id int32) bool { return id == 221325 })
	ConsumableOverrides = append(slices.Clone(ConsumableOverrides), &proto.Consumable{Id: 221325, Name: "EZ-Splode Blasting Charge"})
	if !ShipsConsumable(routing) {
		t.Error("221325 is dropped although ConsumableOverrides names it")
	}
}

func TestTheDenyListRemovesAConsumable(t *testing.T) {
	keepConsumableLists(t)

	sapper := &proto.Consumable{Id: 10646, Name: "Goblin Sapper Charge"}
	if !KeepConsumable(sapper) {
		t.Fatal("10646 is filtered out while no list names it")
	}

	ConsumableDenyList = append(slices.Clone(ConsumableDenyList), 10646)
	if KeepConsumable(sapper) {
		t.Error("10646 is kept although ConsumableDenyList names it")
	}

	ConsumableAllowList = append(slices.Clone(ConsumableAllowList), 10646)
	if !KeepConsumable(sapper) {
		t.Error("10646 is filtered out although ConsumableAllowList names it, which wins over the deny list")
	}
}

func TestACustomConsumableTheGameFilesLackIsKept(t *testing.T) {
	custom := &proto.Consumable{Id: 999001, Name: "Custom Elixir", Type: proto.ConsumableType_ConsumableTypeElixir}
	if !KeepConsumable(custom) {
		t.Error("a consumable ConsumableOverrides adds is filtered out")
	}
}

// A consumable the catalogue carries takes the game files' icon; one only ConsumableOverrides adds
// takes the icon it states.
func TestEveryShippedConsumableGetsAnIcon(t *testing.T) {
	db := NewWowDatabase()
	db.Consumables[10646] = &proto.Consumable{Id: 10646, Name: "Goblin Sapper Charge"}
	db.Consumables[990001] = &proto.Consumable{Id: 990001, Name: "Test Override", Icon: "inv_misc_questionmark"}
	catalogue := []dbc.Consumable{{Id: 10646, IconFileDataID: 1}}

	AddConsumableIcons(db, catalogue, &dbc.DBC{}, map[int]string{1: "Interface/Icons/INV_Gizmo_08.blp"})

	if got := db.ItemIcons[10646].GetIcon(); got != "inv_gizmo_08" {
		t.Errorf("10646's icon is %q, want the game files' inv_gizmo_08", got)
	}
	if got := db.ItemIcons[990001].GetIcon(); got != "inv_misc_questionmark" {
		t.Errorf("the override's icon is %q, want the one it states", got)
	}
}
