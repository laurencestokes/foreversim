package rogue

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Where each race's rogue DPS comes from, for each of the page's builds, run on demand:
//
//	RACE_BREAKDOWN=1 go test --tags=with_db ./sim/rogue/ -run TestRaceBreakdown -v
//
// The page's builds (gear, talents, rotation), its buffs, debuffs and consumables, a neutral level
// 63 target, 180 s. The baseline is a Troll with Berserking off: no other Troll racial touches a
// rogue's damage on a neutral target.
//
// Weapon type is not just a label for a rogue. Backstab, Ambush and Mutilate need daggers, so the
// dagger builds keep their daggers and no weapon racial applies (none of the three is a dagger
// racial). The sword builds swap their swords for stat-identical copies, and Hack and Slash reads
// the type: axes and swords get extra attacks, maces armor penetration. So the Dwarf's move to maces
// changes the talent as well as the racial; the "weapon type" column is that talent change alone,
// measured on the Troll, and the "weapon racial" column what is left. See core.RacialBreakdown.
// RACE_BREAKDOWN_ITERATIONS overrides the iteration count; RACE_BREAKDOWN_OUT writes the tables.
func TestRaceBreakdown(t *testing.T) {
	if os.Getenv("RACE_BREAKDOWN") == "" {
		t.Skip("set RACE_BREAKDOWN=1 to run the race breakdown")
	}
	iterations := int32(30000)
	if s := os.Getenv("RACE_BREAKDOWN_ITERATIONS"); s != "" {
		fmt.Sscan(s, &iterations)
	}

	swordRacials := map[proto.Race]proto.WeaponType{
		proto.Race_RaceHuman: proto.WeaponType_WeaponTypeSword,
		proto.Race_RaceOrc:   proto.WeaponType_WeaponTypeAxe,
		proto.Race_RaceDwarf: proto.WeaponType_WeaponTypeMace,
	}
	// The page's builds (ui/specs/rogue/dps/presets.ts, BUILD_PRESETS).
	builds := []struct {
		name, gear, talents, rotation string
		neutral                       []proto.WeaponType
		weaponRacials                 map[proto.Race]proto.WeaponType
	}{
		{"Sinister Strike (swords)", "combat_sinister_strike_p2_bis", "00530310501-32003311201515231", "combat_sinister_strike",
			[]proto.WeaponType{proto.WeaponType_WeaponTypeSword, proto.WeaponType_WeaponTypeAxe}, swordRacials},
		{"IEA (swords)", "combat_sinister_strike_p2_bis", "005303125-32003311201515131", "combat_sinister_strike_iea",
			[]proto.WeaponType{proto.WeaponType_WeaponTypeSword, proto.WeaponType_WeaponTypeAxe}, swordRacials},
		{"Backstab (daggers)", "combat_backstab_p2_bis", "005302005-30230320201515231-102", "combat_backstab",
			[]proto.WeaponType{proto.WeaponType_WeaponTypeDagger}, nil},
		{"Mutilate (daggers)", "combat_backstab_p2_bis", "00530310551021051-302303202004", "forever_mutilate",
			[]proto.WeaponType{proto.WeaponType_WeaponTypeDagger}, nil},
	}

	var report strings.Builder
	fmt.Fprintf(&report, "# Rogue racial breakdown\n\n")
	for _, build := range builds {
		fmt.Fprintf(&report, "%s\n", core.RacialBreakdown(core.RacialBreakdownConfig{
			Title:          build.name + ", P2 BiS, 180 s, neutral level 63 target",
			Races:          core.ClassRaceCapabilities[proto.Class_ClassRogue],
			BaselineRace:   proto.Race_RaceTroll,
			Gear:           core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", build.gear).GearSet,
			NeutralWeapons: build.neutral,
			WeaponRacials:  build.weaponRacials,
			CooldownRacials: map[proto.Race]int32{
				proto.Race_RaceOrc:      20572,   // Blood Fury
				proto.Race_RaceTroll:    20554,   // Berserking
				proto.Race_RaceNightElf: 1259799, // Elune's Light
				proto.Race_RaceGnome:    1259812, // Eureka!
			},
			Rotation:   core.GetAplRotation("../../ui/specs/rogue/dps/apls", build.rotation).Rotation,
			Iterations: iterations,
			Run: func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) (float64, float64) {
				return runRogue(t, race, gear, build.talents, rotation, bonus, iterations)
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

func runRogue(t *testing.T, race proto.Race, gear *proto.EquipmentSpec, talents string, rotation *proto.APLRotation, bonus *proto.UnitStats, iterations int32) (float64, float64) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class:              proto.Class_ClassRogue,
		Race:               race,
		Equipment:          gear,
		TalentsString:      talents,
		Rotation:           rotation,
		BonusStats:         bonus,
		Profession1:        proto.Profession_Engineering,
		ReactionTimeMs:     200,
		DistanceFromTarget: 5,
		Buffs:              &proto.IndividualBuffs{GreaterBlessingOfKings: true, GreaterBlessingOfMight: true},
		// The page's consumables (ui/specs/rogue/dps/presets.ts).
		Consumables: &proto.ConsumesSpec{
			FlaskId:           13512, // Flask of Supreme Power
			BattleElixirId:    13452, // Elixir of the Mongoose
			StrengthBuffId:    12451, // Juju Power
			AttackPowerBuffId: 12460, // Juju Might
			ZanzaId:           8412,  // Ground Scorpok Assay
			DragonbreathChili: true,
			FoodId:            13928, // Grilled Squid
			ConjuredId:        7676,  // Thistle Tea
			MhImbueId:         26891, // Instant Poison
			OhImbueId:         27186, // Deadly Poison
			GoblinSapper:      true,
		},
	}, &proto.Player_Rogue{Rogue: &proto.Rogue{Options: &proto.Rogue_Options{ClassOptions: &proto.RogueOptions{}}}})

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player,
			&proto.PartyBuffs{
				BattleShout:     proto.TristateEffect_TristateEffectImproved,
				TrueshotAura:    true,
				LeaderOfThePack: true,
			},
			&proto.RaidBuffs{GiftOfTheWild: true, FireResistanceAura: true},
			&proto.Debuffs{FaerieFire: true, SunderArmor: true, CurseOfRecklessness: true}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 101},
	})
	if result.Error != nil {
		t.Fatalf("%v: %s", race, result.Error.Message)
	}
	return result.RaidMetrics.Dps.Avg, result.RaidMetrics.Dps.Stdev
}

// What a gnome rogue's Eureka! is worth, and how much of it is the energy cut and how much the 10%
// damage, run on demand:
//
//	EUREKA_SPLIT=1 go test --tags=with_db ./sim/rogue/ -run TestEurekaSplit -v
func TestEurekaSplit(t *testing.T) {
	if os.Getenv("EUREKA_SPLIT") == "" {
		t.Skip("set EUREKA_SPLIT=1 to run the Eureka! split")
	}
	iterations := int32(50000)
	if s := os.Getenv("EUREKA_SPLIT_ITERATIONS"); s != "" {
		fmt.Sscan(s, &iterations)
	}
	builds := []struct{ label, gear, talents, rotation string }{
		{"Rogue, Sinister Strike (swords, P2 BiS)", "combat_sinister_strike_p2_bis", "00530310501-32003311201515231", "combat_sinister_strike"},
		{"Rogue, IEA (swords, P2 BiS)", "combat_sinister_strike_p2_bis", "005303125-32003311201515131", "combat_sinister_strike_iea"},
		{"Rogue, Backstab (daggers, P2 BiS)", "combat_backstab_p2_bis", "005302005-30230320201515231-102", "combat_backstab"},
		{"Rogue, Mutilate (daggers, P2 BiS)", "combat_backstab_p2_bis", "00530310551021051-302303202004", "forever_mutilate"},
	}
	report := "# Eureka! split, rogue (180 s, neutral level 63 target)\n\n" + core.EurekaSplitHeader
	for _, build := range builds {
		report += core.EurekaSplit(core.EurekaSplitConfig{
			Label:      build.label,
			SpellID:    1259812,
			Gear:       core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", build.gear).GearSet,
			Rotation:   core.GetAplRotation("../../ui/specs/rogue/dps/apls", build.rotation).Rotation,
			Iterations: iterations,
			Run: func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) (float64, float64) {
				return runRogue(t, race, gear, build.talents, rotation, bonus, iterations)
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
