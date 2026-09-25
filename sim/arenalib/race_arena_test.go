package arenalib

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Items under test ids of their own, so the rule is checked without the generated database.
const (
	testOneHandAxe = 990201 + iota
	testTwoHandAxe
	testHeldOffHand
	testShield
	testBow
)

func addRelabelTestItems() {
	weapon := func(id int32, weaponType proto.WeaponType, hand proto.HandType) *proto.SimItem {
		return &proto.SimItem{
			Id:             id,
			Name:           "Race arena test item",
			Type:           proto.ItemType_ItemTypeWeapon,
			WeaponType:     weaponType,
			HandType:       hand,
			ScalingOptions: map[int32]*proto.ScalingItemProperties{0: {}},
		}
	}
	core.AddToDatabase(&proto.SimDatabase{Items: []*proto.SimItem{
		weapon(testOneHandAxe, proto.WeaponType_WeaponTypeAxe, proto.HandType_HandTypeOneHand),
		weapon(testTwoHandAxe, proto.WeaponType_WeaponTypeAxe, proto.HandType_HandTypeTwoHand),
		weapon(testHeldOffHand, proto.WeaponType_WeaponTypeOffHand, proto.HandType_HandTypeOffHand),
		weapon(testShield, proto.WeaponType_WeaponTypeShield, proto.HandType_HandTypeOffHand),
		{
			Id:               testBow,
			Name:             "Race arena test bow",
			Type:             proto.ItemType_ItemTypeRanged,
			RangedWeaponType: proto.RangedWeaponType_RangedWeaponTypeBow,
			ScalingOptions:   map[int32]*proto.ScalingItemProperties{0: {}},
		},
	}})
}

func gearOf(ids ...int32) *proto.EquipmentSpec {
	gear := &proto.EquipmentSpec{}
	for _, id := range ids {
		gear.Items = append(gear.Items, &proto.ItemSpec{Id: id})
	}
	return gear
}

func overrides(gear *proto.EquipmentSpec) []proto.WeaponType {
	out := []proto.WeaponType{}
	for _, item := range gear.Items {
		out = append(out, item.WeaponTypeOverride)
	}
	return out
}

// The weapon a race's racial is tried on has to be one the class could hold as that type, and
// nothing but a swung weapon is ever relabelled.
func TestRelabelWeapons(t *testing.T) {
	addRelabelTestItems()
	const (
		none  = proto.WeaponType_WeaponTypeUnknown
		sword = proto.WeaponType_WeaponTypeSword
		mace  = proto.WeaponType_WeaponTypeMace
		axe   = proto.WeaponType_WeaponTypeAxe
	)

	// Dual wield: both hands go, and the ranged weapon stays.
	gear := relabelWeapons(gearOf(testOneHandAxe, testOneHandAxe, testBow), proto.Class_ClassWarrior, sword)
	if got := overrides(gear); got[0] != sword || got[1] != sword || got[2] != none {
		t.Errorf("warrior dual wield to swords: %v", got)
	}
	if !carries(gear, sword) || carries(gear, axe) {
		t.Error("carries should read the relabelled type, not the item's own")
	}

	// A one-hander beside a held off-hand or a shield: only the weapon goes.
	gear = relabelWeapons(gearOf(testOneHandAxe, testHeldOffHand), proto.Class_ClassShaman, mace)
	if got := overrides(gear); got[0] != mace || got[1] != none {
		t.Errorf("shaman axe and off-hand to maces: %v", got)
	}
	gear = relabelWeapons(gearOf(testOneHandAxe, testShield), proto.Class_ClassWarrior, mace)
	if got := overrides(gear); got[0] != mace || got[1] != none {
		t.Errorf("warrior axe and shield to maces: %v", got)
	}

	// A two-hander only goes to a type the class wields two-handed: a paladin's two-handed sword
	// is fine, a priest's two-handed mace is not a thing.
	if gear := relabelWeapons(gearOf(testTwoHandAxe), proto.Class_ClassPaladin, sword); gear == nil || overrides(gear)[0] != sword {
		t.Error("a paladin can wield a two-handed sword")
	}
	if relabelWeapons(gearOf(testTwoHandAxe), proto.Class_ClassPriest, mace) != nil {
		t.Error("a priest cannot wield a two-handed mace")
	}

	// A class that cannot use the type at all gets nothing, and neither does gear already of it.
	if relabelWeapons(gearOf(testOneHandAxe), proto.Class_ClassMage, axe) != nil {
		t.Error("a mage cannot use an axe")
	}
	if relabelWeapons(gearOf(testOneHandAxe, testOneHandAxe), proto.Class_ClassWarrior, axe) != nil {
		t.Error("axes relabelled as axes change nothing and need no second run")
	}

	// A one-hander never becomes a polearm or a staff, which only come two-handed.
	if relabelWeapons(gearOf(testOneHandAxe), proto.Class_ClassWarrior, proto.WeaponType_WeaponTypePolearm) != nil {
		t.Error("a one-handed polearm does not exist")
	}

	// The input is never written to: the shipped gear is the other run.
	shipped := gearOf(testOneHandAxe)
	relabelWeapons(shipped, proto.Class_ClassWarrior, sword)
	if shipped.Items[0].WeaponTypeOverride != none {
		t.Error("relabelling wrote through to the shipped gear")
	}
}

// The Skyborne are one row: both halves share every racial, and the half that runs is one the
// class can actually be.
func TestRaceCandidatesMergeSkyborne(t *testing.T) {
	for class, want := range map[proto.Class]proto.Race{
		proto.Class_ClassWarrior: proto.Race_RaceHighOrderSkyborne,
		proto.Class_ClassMage:    proto.Race_RaceHighOrderSkyborne,
		proto.Class_ClassShaman:  proto.Race_RaceWindshaperSkyborne,
	} {
		skyborne := []proto.Race{}
		for _, race := range raceCandidates(class) {
			if raceName(race) == "Skyborne" {
				skyborne = append(skyborne, race)
			}
		}
		if len(skyborne) != 1 || skyborne[0] != want {
			t.Errorf("%s: Skyborne rows %v, want one, %s", class, skyborne, want)
		}
	}
	if got, want := len(raceCandidates(proto.Class_ClassWarlock)), len(core.ClassRaceCapabilities[proto.Class_ClassWarlock]); got != want {
		t.Errorf("a class with no Skyborne keeps all %d races, got %d", want, got)
	}
}
