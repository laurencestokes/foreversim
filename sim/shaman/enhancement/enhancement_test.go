package enhancement

import (
	"testing"

	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

func init() {
	RegisterEnhancementShaman()
	common.RegisterAllEffects()
}

func TestEnhancement(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassShaman,
			Race:       proto.Race_RaceTroll,
			OtherRaces: []proto.Race{proto.Race_RaceOrc, proto.Race_RaceDwarf},
			SpecOptions: core.SpecOptionsCombo{Label: "Standard", SpecOptions: &proto.Player_EnhancementShaman{
				EnhancementShaman: &proto.EnhancementShaman{
					Options: &proto.EnhancementShaman_Options{
						SyncType:    proto.ShamanSyncType_Auto,
						ImbueOh:     proto.ShamanImbue_WindfuryWeapon,
						ImbueOhSwap: proto.ShamanImbue_WindfuryWeapon,
						ClassOptions: &proto.ShamanOptions{
							ImbueMh:        proto.ShamanImbue_WindfuryWeapon,
							ImbueMhSwap:    proto.ShamanImbue_WindfuryWeapon,
							ShieldProcrate: 0.0,
						},
					},
				},
			}},
			// Naked, for the same reason as Elemental and the merged Mage port.
			GearSet:  core.GearSetCombo{Label: "Naked", GearSet: &proto.EquipmentSpec{}},
			Talents:  DefaultTalents,
			Rotation: core.GetAplRotation("../../../ui/specs/shaman/enhancement/apls", "forever"),
			ItemFilter: core.ItemFilter{
				WeaponTypes: []proto.WeaponType{
					proto.WeaponType_WeaponTypeAxe,
					proto.WeaponType_WeaponTypeDagger,
					proto.WeaponType_WeaponTypeFist,
					proto.WeaponType_WeaponTypeMace,
					proto.WeaponType_WeaponTypeOffHand,
				},
				ArmorType: proto.ArmorType_ArmorTypeMail,
				RangedWeaponTypes: []proto.RangedWeaponType{
					proto.RangedWeaponType_RangedWeaponTypeTotem,
				},
			},
		},
	}))
}

// The community build our Forever sim ranks Enhancement with.
const DefaultTalents = "5505301-053030031005112251"
