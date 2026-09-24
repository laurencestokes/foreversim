package warlock

import (
	"testing"

	"github.com/wowsims/forever/sim/arenalib"
	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func init() {
	RegisterWarlock()
	common.RegisterAllEffects()
}

// The community builds our Forever sim ranks: Deep Affliction 35/0/16 keeps its Succubus out, and
// Shadow and Flame 13/11/27 sacrifices one before the pull for the Fire damage Forever's Demonic
// Sacrifice leaves behind.
var AfflictionTalents = "2535002013521105--05000551"
var DestructionTalents = "25501-0025003001-055035510010002"

func TestAffliction(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		warlockSuite("affliction", AfflictionTalents, &proto.WarlockOptions{
			Summon:          proto.WarlockOptions_Succubus,
			SacrificeSummon: false,
			Armor:           proto.WarlockOptions_DemonArmor,
			CurseOptions:    proto.WarlockOptions_Elements,
		}),
	}))
}

func TestDestruction(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		warlockSuite("destruction", DestructionTalents, &proto.WarlockOptions{
			Summon:          proto.WarlockOptions_Succubus,
			SacrificeSummon: true,
			Armor:           proto.WarlockOptions_DemonArmor,
			CurseOptions:    proto.WarlockOptions_Elements,
		}),
	}))
}

func warlockSuite(apl string, talents string, options *proto.WarlockOptions) core.CharacterSuiteConfig {
	return core.CharacterSuiteConfig{
		Class:      proto.Class_ClassWarlock,
		Race:       proto.Race_RaceOrc,
		OtherRaces: []proto.Race{proto.Race_RaceGnome},
		SpecOptions: core.SpecOptionsCombo{Label: "Warlock", SpecOptions: &proto.Player_Warlock{
			Warlock: &proto.Warlock{
				Options: &proto.Warlock_Options{
					ClassOptions: options,
				},
			},
		}},
		// Naked: the generated item database does not carry the pre-raid set our Forever sim tests
		// with yet, and gives the rest TBC-shaped stats.
		GearSet:  core.GearSetCombo{Label: "Naked", GearSet: &proto.EquipmentSpec{}},
		Talents:  talents,
		Rotation: core.GetAplRotation("../../ui/specs/warlock/dps/apls", apl),
		ItemFilter: core.ItemFilter{
			WeaponTypes: []proto.WeaponType{
				proto.WeaponType_WeaponTypeDagger,
				proto.WeaponType_WeaponTypeStaff,
				proto.WeaponType_WeaponTypeSword,
			},
			ArmorType: proto.ArmorType_ArmorTypeCloth,
			RangedWeaponTypes: []proto.RangedWeaponType{
				proto.RangedWeaponType_RangedWeaponTypeWand,
			},
			EnchantBlacklist: []int32{2673, 3225, 3273},
			IDBlacklist:      []int32{28556},
		},
	}
}

// One demon for every rotation: the Succubus, kept out, as the Affliction builds run it.
func TestArena(t *testing.T) {
	arenalib.Run(t, arenalib.Spec{
		Dir:   "warlock",
		UI:    "warlock/dps",
		Class: proto.Class_ClassWarlock,
		Race:  proto.Race_RaceOrc,
		SpecOptions: &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{
			ClassOptions: &proto.WarlockOptions{
				Summon:       proto.WarlockOptions_Succubus,
				Armor:        proto.WarlockOptions_DemonArmor,
				CurseOptions: proto.WarlockOptions_Elements,
			},
		}}},
		Role:               arenalib.Caster,
		DistanceFromTarget: 30,
	})
}
