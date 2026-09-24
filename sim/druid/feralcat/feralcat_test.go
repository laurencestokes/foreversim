package feralcat

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/arenalib"
	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/druid"
)

func init() {
	RegisterFeralCatDruid()
	common.RegisterAllEffects()
}

func TestFeralCat(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:      proto.Class_ClassDruid,
			Race:       proto.Race_RaceNightElf,
			OtherRaces: []proto.Race{proto.Race_RaceTauren},

			// Naked: the generated item database does not carry the Forever gear our sim tests
			// with yet, and gives the rest TBC-shaped stats.
			GearSet: core.GearSetCombo{Label: "Naked", GearSet: &proto.EquipmentSpec{}},

			Talents: DefaultTalents,
			OtherTalentSets: []core.TalentsCombo{
				{Label: "FeralCat", Talents: FeralCatTalents},
			},

			SpecOptions: core.SpecOptionsCombo{Label: "Standard", SpecOptions: DefaultSpecOptions},

			Rotation: core.GetAplRotation("../../../ui/specs/druid/feralcat/apls", "default"),

			Consumables: DefaultConsumables,

			Profession1: proto.Profession_Engineering,
			Profession2: proto.Profession_Enchanting,

			ItemFilter: core.ItemFilter{
				ArmorType: proto.ArmorType_ArmorTypeLeather,
				WeaponTypes: []proto.WeaponType{
					proto.WeaponType_WeaponTypeDagger,
					proto.WeaponType_WeaponTypeFist,
					proto.WeaponType_WeaponTypeMace,
					proto.WeaponType_WeaponTypeStaff,
				},
				RangedWeaponTypes: []proto.RangedWeaponType{
					proto.RangedWeaponType_RangedWeaponTypeIdol,
				},
			},

			EPReferenceStat: proto.Stat_StatAttackPower,
			StatsToWeigh: []proto.Stat{
				proto.Stat_StatAgility,
				proto.Stat_StatStrength,
				proto.Stat_StatAttackPower,
				proto.Stat_StatFeralAttackPower,
				proto.Stat_StatMeleeHitRating,
				proto.Stat_StatExpertiseRating,
				proto.Stat_StatMeleeCritRating,
				proto.Stat_StatMeleeHasteRating,
				proto.Stat_StatArmorPenetration,
			},
		},
	}))
}

// Our Forever sim's feral builds.
const DefaultTalents = "-5521002023132213051-05503"
const FeralCatTalents = "050022-5500002123032213051-052"

var DefaultSpecOptions = &proto.Player_FeralCatDruid{
	FeralCatDruid: &proto.FeralCatDruid{
		Rotation: &proto.FeralCatDruid_Rotation{
			FinishingMove:      proto.FeralCatDruid_Rotation_Rip,
			Biteweave:          true,
			RipMinComboPoints:  5,
			BiteMinComboPoints: 5,
			MangleTrick:        true,
			MaintainFaerieFire: false,
		},
		Options: &proto.FeralCatDruid_Options{},
	},
}

var DefaultConsumables = &proto.ConsumesSpec{
	PotId:            22838, // Haste Potion
	BattleElixirId:   22831, // Elixir of Major Agility
	GuardianElixirId: 32067, // Elixir of Draenic Wisdom
	FoodId:           27664, // Grilled Mudfish
	MhImbueId:        34340, // Adamantite Weightstone
	ConjuredId:       12662, // Demonic Rune
	SuperSapper:      true,
	GoblinSapper:     true,
	ScrollAgi:        true,
	ScrollStr:        true,
}

// Clearcasting (16870, one charge) makes the next ability in its mask free and is spent by it; one
// outside the mask leaves it up.
func TestClearcastingSpentByNextCostedAbility(t *testing.T) {
	sim := core.NewSim(&proto.RaidSimRequest{
		SimOptions: &proto.SimOptions{RandomSeed: 1},
		Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
			Name: "Cat", Class: proto.Class_ClassDruid, Race: proto.Race_RaceNightElf, TalentsString: DefaultTalents,
			Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{}, Spec: DefaultSpecOptions,
			Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
		}}}}},
		Encounter: core.MakeSingleTargetEncounter(0),
	}, simsignals.CreateSignals())
	sim.Reset()

	cat := sim.Raid.Parties[0].Players[0].(druid.DruidAgent).GetDruid()
	cat.ClearcastingAura.Activate(sim)

	if !cat.TigersFury.Cast(sim, cat.CurrentTarget) || !cat.ClearcastingAura.IsActive() {
		t.Fatal("Tiger's Fury, outside the mask, did not cast or spent Clearcasting")
	}

	for !cat.GCD.IsReady(sim) && sim.CurrentTime < 5*time.Second {
		sim.Step()
	}
	energy := cat.CurrentEnergy()
	if !cat.Shred.Cast(sim, cat.CurrentTarget) {
		t.Fatal("Shred did not cast")
	}
	if cat.CurrentEnergy() != energy {
		t.Errorf("Shred cost %v Energy under Clearcasting, want 0", energy-cat.CurrentEnergy())
	}
	if cat.ClearcastingAura.IsActive() {
		t.Error("Shred did not spend Clearcasting")
	}
}

// The arena entry for this spec. Without ARENA_OUT set it only checks every build's damage against the spell manifest; see sim/arenalib.
func TestArena(t *testing.T) {
	arenalib.Run(t, arenalib.Spec{
		Dir:         "feral_druid",
		UI:          "druid/feralcat",
		Class:       proto.Class_ClassDruid,
		Race:        proto.Race_RaceTauren,
		SpecOptions: DefaultSpecOptions,
		Role:        arenalib.Melee,
	})
}
