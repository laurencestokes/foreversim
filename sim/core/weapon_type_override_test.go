package core

import (
	"testing"

	"github.com/wowsims/forever/sim/core/proto"
)

// Registers a plain one-handed sword and a shield under freshly-minted test IDs so these tests
// don't depend on (or collide with) the real generated database.
func addWeaponTypeOverrideTestItems() (swordID, shieldID, chestID int32) {
	const testSwordID, testShieldID, testChestID = 990101, 990102, 990103
	addToDatabase(&proto.SimDatabase{
		Items: []*proto.SimItem{
			{
				Id:             testSwordID,
				Name:           "Test Sword",
				Type:           proto.ItemType_ItemTypeWeapon,
				WeaponType:     proto.WeaponType_WeaponTypeSword,
				HandType:       proto.HandType_HandTypeOneHand,
				WeaponSpeed:    2.6,
				ScalingOptions: map[int32]*proto.ScalingItemProperties{0: {WeaponDamageMin: 10, WeaponDamageMax: 20}},
			},
			{
				Id:             testShieldID,
				Name:           "Test Shield",
				Type:           proto.ItemType_ItemTypeWeapon,
				WeaponType:     proto.WeaponType_WeaponTypeShield,
				HandType:       proto.HandType_HandTypeOffHand,
				ScalingOptions: map[int32]*proto.ScalingItemProperties{0: {}},
			},
			{
				Id:             testChestID,
				Name:           "Test Chest",
				Type:           proto.ItemType_ItemTypeChest,
				ScalingOptions: map[int32]*proto.ScalingItemProperties{0: {}},
			},
		},
	})
	return testSwordID, testShieldID, testChestID
}

// The override exists purely to relabel the type key that racials/talents/abilities read;
// everything else about the item (damage, speed, hand) must be untouched.
func TestWeaponTypeOverrideRelabelsTypeOnly(t *testing.T) {
	swordID, _, _ := addWeaponTypeOverrideTestItems()

	plain := NewItem(ItemSpec{ID: swordID})
	overridden := NewItem(ItemSpec{ID: swordID, WeaponTypeOverride: proto.WeaponType_WeaponTypeAxe})

	if overridden.WeaponType != proto.WeaponType_WeaponTypeAxe {
		t.Fatalf("expected overridden weapon type Axe, got %v", overridden.WeaponType)
	}
	if overridden.WeaponTypeOverride != proto.WeaponType_WeaponTypeAxe {
		t.Fatalf("expected WeaponTypeOverride to be recorded on the item, got %v", overridden.WeaponTypeOverride)
	}
	if overridden.HandType != plain.HandType {
		t.Fatalf("hand type must not change: plain=%v overridden=%v", plain.HandType, overridden.HandType)
	}
	if overridden.WeaponDamageMin != plain.WeaponDamageMin || overridden.WeaponDamageMax != plain.WeaponDamageMax {
		t.Fatal("weapon damage must not change from a type override")
	}
	if overridden.SwingSpeed != plain.SwingSpeed {
		t.Fatal("swing speed must not change from a type override")
	}
	if overridden.Stats != plain.Stats {
		t.Fatal("stats must not change from a type override")
	}
}

func TestWeaponTypeOverrideUnsetKeepsItemsOwnType(t *testing.T) {
	swordID, _, _ := addWeaponTypeOverrideTestItems()

	item := NewItem(ItemSpec{ID: swordID, WeaponTypeOverride: proto.WeaponType_WeaponTypeUnknown})
	if item.WeaponType != proto.WeaponType_WeaponTypeSword {
		t.Fatalf("unset override should leave the item's own type, got %v", item.WeaponType)
	}
	if item.WeaponTypeOverride != proto.WeaponType_WeaponTypeUnknown {
		t.Fatalf("expected no override recorded, got %v", item.WeaponTypeOverride)
	}
}

// Off-hand items and shields aren't "weapon type" specializations in the usual sense, and
// relabelling into/out of them would confuse dual-wield and hand-type logic, so both directions
// are excluded.
func TestWeaponTypeOverrideExcludesOffHandAndShield(t *testing.T) {
	swordID, shieldID, _ := addWeaponTypeOverrideTestItems()

	// A shield can't be relabelled as a sword.
	shield := NewItem(ItemSpec{ID: shieldID, WeaponTypeOverride: proto.WeaponType_WeaponTypeSword})
	if shield.WeaponType != proto.WeaponType_WeaponTypeShield {
		t.Fatalf("shield's own type must not be overridden, got %v", shield.WeaponType)
	}

	// A sword can't be relabelled into a shield or an off-hand.
	asShield := NewItem(ItemSpec{ID: swordID, WeaponTypeOverride: proto.WeaponType_WeaponTypeShield})
	if asShield.WeaponType != proto.WeaponType_WeaponTypeSword {
		t.Fatalf("sword must not be relabelled as Shield, got %v", asShield.WeaponType)
	}
	asOffHand := NewItem(ItemSpec{ID: swordID, WeaponTypeOverride: proto.WeaponType_WeaponTypeOffHand})
	if asOffHand.WeaponType != proto.WeaponType_WeaponTypeSword {
		t.Fatalf("sword must not be relabelled as OffHand, got %v", asOffHand.WeaponType)
	}
}

func TestWeaponTypeOverrideIgnoredForNonWeapons(t *testing.T) {
	_, _, chestID := addWeaponTypeOverrideTestItems()

	chest := NewItem(ItemSpec{ID: chestID, WeaponTypeOverride: proto.WeaponType_WeaponTypeAxe})
	if chest.WeaponType != proto.WeaponType_WeaponTypeUnknown {
		t.Fatalf("override on a non-weapon item must be a no-op, got weapon type %v", chest.WeaponType)
	}
}

// The override is carried on Item.WeaponTypeOverride purely so it survives a save/share round
// trip through the proto; it has to come back out the same way meta_gem_disabled does.
func TestWeaponTypeOverrideSurvivesProtoRoundTrip(t *testing.T) {
	swordID, _, _ := addWeaponTypeOverrideTestItems()

	item := NewItem(ItemSpec{ID: swordID, WeaponTypeOverride: proto.WeaponType_WeaponTypeAxe})
	itemSpecProto := item.ToItemSpecProto()
	if itemSpecProto.GetWeaponTypeOverride() != proto.WeaponType_WeaponTypeAxe {
		t.Fatalf("ToItemSpecProto lost the override, got %v", itemSpecProto.GetWeaponTypeOverride())
	}

	roundTripped := NewItem(ProtoToEquipmentSpec(&proto.EquipmentSpec{Items: []*proto.ItemSpec{itemSpecProto}})[0])
	if roundTripped.WeaponType != proto.WeaponType_WeaponTypeAxe {
		t.Fatalf("override did not survive an ItemSpec proto round trip, got %v", roundTripped.WeaponType)
	}
	if roundTripped.WeaponTypeOverride != proto.WeaponType_WeaponTypeAxe {
		t.Fatalf("WeaponTypeOverride did not survive an ItemSpec proto round trip, got %v", roundTripped.WeaponTypeOverride)
	}
}
