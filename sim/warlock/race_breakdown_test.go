package warlock

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Where each race's warlock DPS comes from, for each of the page's community builds, run on
// demand:
//
//	RACE_BREAKDOWN=1 go test --tags=with_db ./sim/warlock/ -run TestRaceBreakdown -v
//
// The page's defaults: Pre-BiS gear, buffs, debuffs and consumables, a neutral level 63 target,
// 180 s. The main hand is retyped (a dagger for everyone, a stat-identical one-handed sword for the
// Human's Sword Specialization), the held off-hand stays. The baseline is an Orc with Blood Fury off:
// no other Orc racial touches a warlock's damage (Axe Specialization needs an axe, which a warlock
// cannot use). See core.RacialBreakdown for how each column is measured. RACE_BREAKDOWN_ITERATIONS
// overrides the iteration count; RACE_BREAKDOWN_OUT writes the tables.
func TestRaceBreakdown(t *testing.T) {
	if os.Getenv("RACE_BREAKDOWN") == "" {
		t.Skip("set RACE_BREAKDOWN=1 to run the race breakdown")
	}
	iterations := int32(30000)
	if s := os.Getenv("RACE_BREAKDOWN_ITERATIONS"); s != "" {
		fmt.Sscan(s, &iterations)
	}

	// The page's builds (ui/specs/warlock/dps/presets.ts): talents, rotation and demon.
	succubus := &proto.WarlockOptions{Armor: proto.WarlockOptions_DemonArmor, CurseOptions: proto.WarlockOptions_Elements, Summon: proto.WarlockOptions_Succubus}
	sacrificedImp := &proto.WarlockOptions{Armor: proto.WarlockOptions_DemonArmor, CurseOptions: proto.WarlockOptions_Elements, Summon: proto.WarlockOptions_Imp, SacrificeSummon: true}
	builds := []struct {
		name, talents, rotation string
		options                 *proto.WarlockOptions
	}{
		{"Deep Affliction 35/0/16", "2535002013521105--05000551", "affliction", succubus},
		{"DS/Ruin Pandemic 24/11/16", "25220010135201-0025003001-05500051", "ds_ruin", sacrificedImp},
		{"Shadow and Flame 13/11/27", "25501-0025003001-055035510010002", "destruction", sacrificedImp},
		{"Demonic Pact 2/31/18", "113-0005003221220311351-0550005", "demonic_pact", succubus},
	}

	var report strings.Builder
	fmt.Fprintf(&report, "# Warlock racial breakdown\n\n")
	for _, build := range builds {
		fmt.Fprintf(&report, "%s\n", core.RacialBreakdown(core.RacialBreakdownConfig{
			Title:          build.name + ", Pre-BiS, 180 s, neutral level 63 target",
			Races:          core.ClassRaceCapabilities[proto.Class_ClassWarlock],
			BaselineRace:   proto.Race_RaceOrc,
			Gear:           core.GetGearSet("../../ui/specs/warlock/dps/gear_sets", "prebis").GearSet,
			NeutralWeapons: []proto.WeaponType{proto.WeaponType_WeaponTypeDagger},
			WeaponRacials: map[proto.Race]proto.WeaponType{
				proto.Race_RaceHuman: proto.WeaponType_WeaponTypeSword,
			},
			CooldownRacials: map[proto.Race]int32{
				proto.Race_RaceOrc:   20572,   // Blood Fury
				proto.Race_RaceTroll: 20554,   // Berserking
				proto.Race_RaceGnome: 1259821, // Eureka!
			},
			Rotation:   core.GetAplRotation("../../ui/specs/warlock/dps/apls", build.rotation).Rotation,
			Iterations: iterations,
			Run: func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) (float64, float64) {
				return runWarlock(t, race, gear, build.talents, build.options, rotation, bonus, iterations)
			},
		}))
	}

	fmt.Println(report.String())
	if out := os.Getenv("RACE_BREAKDOWN_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(report.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func runWarlock(t *testing.T, race proto.Race, gear *proto.EquipmentSpec, talents string, options *proto.WarlockOptions, rotation *proto.APLRotation, bonus *proto.UnitStats, iterations int32) (float64, float64) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassWarlock,
		Race:               race,
		Equipment:          gear,
		TalentsString:      talents,
		Rotation:           rotation,
		BonusStats:         bonus,
		Profession1:        proto.Profession_Enchanting,
		Profession2:        proto.Profession_Tailoring,
		ReactionTimeMs:     200,
		ChannelClipDelayMs: 150,
		DistanceFromTarget: 25,
		Buffs:              &proto.IndividualBuffs{GreaterBlessingOfKings: true, GreaterBlessingOfWisdom: true},
		Consumables: &proto.ConsumesSpec{
			FlaskId:            13512, // Flask of Supreme Power
			SpellPowerElixirId: 13454, // Greater Arcane Elixir
			SchoolElixirId:     9264,  // Elixir of Shadow Power
			GuardianElixirId:   20007, // Mageblood Elixir
			ZanzaId:            8423,  // Cerebral Cortex Compound
			AlcoholId:          21151, // Rumsey Rum Black Label
			FoodId:             18254, // Runn Tum Tuber Surprise
			PotId:              13444, // Major Mana Potion
			ConjuredId:         12662, // Demonic Rune
		},
	}, &proto.Player_Warlock{Warlock: &proto.Warlock{Options: &proto.Warlock_Options{ClassOptions: options}}})

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player,
			&proto.PartyBuffs{MoonkinAura: true},
			&proto.RaidBuffs{
				ArcaneBrilliance:   true,
				PrayerOfSpirit:     true,
				GiftOfTheWild:      true,
				PrayerOfFortitude:  true,
				FireResistanceAura: true,
			},
			&proto.Debuffs{ExposeArmor: true, FaerieFire: true, JudgementOfWisdom: true, SunderArmor: true}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 101},
	})
	if result.Error != nil {
		t.Fatalf("%v: %s", race, result.Error.Message)
	}
	return result.RaidMetrics.Dps.Avg, result.RaidMetrics.Dps.Stdev
}

// What a gnome warlock's Eureka! is worth, and how much of it is the mana cut and how much the 10%
// damage, run on demand:
//
//	EUREKA_SPLIT=1 go test --tags=with_db ./sim/warlock/ -run TestEurekaSplit -v
func TestEurekaSplit(t *testing.T) {
	if os.Getenv("EUREKA_SPLIT") == "" {
		t.Skip("set EUREKA_SPLIT=1 to run the Eureka! split")
	}
	iterations := int32(50000)
	if s := os.Getenv("EUREKA_SPLIT_ITERATIONS"); s != "" {
		fmt.Sscan(s, &iterations)
	}
	succubus := &proto.WarlockOptions{Armor: proto.WarlockOptions_DemonArmor, CurseOptions: proto.WarlockOptions_Elements, Summon: proto.WarlockOptions_Succubus}
	sacrificedImp := &proto.WarlockOptions{Armor: proto.WarlockOptions_DemonArmor, CurseOptions: proto.WarlockOptions_Elements, Summon: proto.WarlockOptions_Imp, SacrificeSummon: true}
	builds := []struct {
		label, talents, rotation string
		options                  *proto.WarlockOptions
	}{
		{"Warlock, Deep Affliction (Pre-BiS)", "2535002013521105--05000551", "affliction", succubus},
		{"Warlock, DS/Ruin Pandemic (Pre-BiS)", "25220010135201-0025003001-05500051", "ds_ruin", sacrificedImp},
		{"Warlock, Shadow and Flame (Pre-BiS)", "25501-0025003001-055035510010002", "destruction", sacrificedImp},
		{"Warlock, Demonic Pact (Pre-BiS)", "113-0005003221220311351-0550005", "demonic_pact", succubus},
	}
	report := "# Eureka! split, warlock (180 s, neutral level 63 target)\n\n" + core.EurekaSplitHeader
	for _, build := range builds {
		report += core.EurekaSplit(core.EurekaSplitConfig{
			Label:      build.label,
			SpellID:    1259821,
			Gear:       core.GetGearSet("../../ui/specs/warlock/dps/gear_sets", "prebis").GearSet,
			Rotation:   core.GetAplRotation("../../ui/specs/warlock/dps/apls", build.rotation).Rotation,
			Iterations: iterations,
			Run: func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) (float64, float64) {
				return runWarlock(t, race, gear, build.talents, build.options, rotation, bonus, iterations)
			},
		})
	}
	fmt.Println(report)
	if out := os.Getenv("EUREKA_SPLIT_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(report), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
