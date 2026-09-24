package dps

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// A race comparison for the DPS warrior, run on demand:
//
//	RACE_SWEEP=1 go test --tags=with_db ./sim/warrior/dps/ -run TestRaceSweep -v
//
// Every warrior race runs the same character - the page's defaults: Pre-BiS gear, DPS talents, the
// No Reck rotation, its buffs, debuffs and consumables - with the same random seed. The weapon
// racials depend on weapon type, so the "matched" scenarios swap the weapons for copies with
// identical stats and only the type changed, and each race is shown on its best type. The preset
// scenarios run the gear exactly as shipped. RACE_SWEEP_ITERATIONS overrides the iteration count.
func TestRaceSweep(t *testing.T) {
	if os.Getenv("RACE_SWEEP") == "" {
		t.Skip("set RACE_SWEEP=1 to run the race comparison")
	}
	iterations := int32(20000)
	if s := os.Getenv("RACE_SWEEP_ITERATIONS"); s != "" {
		fmt.Sscan(s, &iterations)
	}

	races := core.ClassRaceCapabilities[proto.Class_ClassWarrior]
	weaponTypes := []proto.WeaponType{proto.WeaponType_WeaponTypeSword, proto.WeaponType_WeaponTypeAxe, proto.WeaponType_WeaponTypeMace}

	p0 := core.GetGearSet("../../../ui/specs/warrior/dps/gear_sets", "p0.bis").GearSet
	armsLaunch := core.GetGearSet("../../../ui/specs/warrior/dps/gear_sets", "arms_launch").GearSet

	type scenario struct {
		label   string
		talents string
		mobType proto.MobType
		// One gear set per weapon type for the matched scenarios, or a single preset.
		gear map[proto.WeaponType]*proto.EquipmentSpec
	}
	matched := func(gear *proto.EquipmentSpec) map[proto.WeaponType]*proto.EquipmentSpec {
		out := map[proto.WeaponType]*proto.EquipmentSpec{}
		for _, wt := range weaponTypes {
			out[wt] = core.RetypeWeapons(gear, wt)
		}
		return out
	}
	preset := func(name string) map[proto.WeaponType]*proto.EquipmentSpec {
		return map[proto.WeaponType]*proto.EquipmentSpec{
			proto.WeaponType_WeaponTypeUnknown: core.GetGearSet("../../../ui/specs/warrior/dps/gear_sets", name).GearSet,
		}
	}

	scenarios := []scenario{
		{"Dual wield (Pre-BiS stats), matched weapon type, neutral boss", DpsTalents, proto.MobType_MobTypeMechanical, matched(p0)},
		{"Dual wield (Pre-BiS stats), matched weapon type, Elemental boss", DpsTalents, proto.MobType_MobTypeElemental, matched(p0)},
		{"Dual wield (Pre-BiS stats), matched weapon type, Beast boss", DpsTalents, proto.MobType_MobTypeBeast, matched(p0)},
		{"Two-hander (Launch Arms stats), Arms talents, matched weapon type, neutral boss", ArmsTalents, proto.MobType_MobTypeMechanical, matched(armsLaunch)},
		{"Preset: Pre-BiS as shipped (axe + axe), neutral boss", DpsTalents, proto.MobType_MobTypeMechanical, preset("p0.bis")},
		{"Preset: P1 BiS as shipped (sword + sword), neutral boss", DpsTalents, proto.MobType_MobTypeMechanical, preset("phase_1")},
		{"Preset: Launch as shipped (sword + axe), neutral boss", DpsTalents, proto.MobType_MobTypeMechanical, preset("launch")},
	}

	var report strings.Builder
	fmt.Fprintf(&report, "# DPS warrior race sweep (%d iterations, 180 s, one level 63 target, armor 3731)\n", iterations)

	for _, sc := range scenarios {
		type row struct {
			race   proto.Race
			best   proto.WeaponType
			dps    float64
			stderr float64
			byType map[proto.WeaponType]float64
		}
		var rows []row
		for _, race := range races {
			r := row{race: race, dps: -1, byType: map[proto.WeaponType]float64{}}
			for wt, gear := range sc.gear {
				dps, stdev := runWarrior(t, warriorRun{race: race, gear: gear, talents: sc.talents, mobType: sc.mobType}, iterations)
				r.byType[wt] = dps
				if dps > r.dps {
					r.dps, r.best, r.stderr = dps, wt, stdev/math.Sqrt(float64(iterations))
				}
			}
			rows = append(rows, r)
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].dps > rows[j].dps })

		fmt.Fprintf(&report, "\n## %s\n\n", sc.label)
		isMatched := len(sc.gear) > 1
		if isMatched {
			fmt.Fprintf(&report, "| # | Race | Best weapon | DPS | vs best | Sword | Axe | Mace |\n|---|---|---|---|---|---|---|---|\n")
		} else {
			fmt.Fprintf(&report, "| # | Race | DPS | vs best |\n|---|---|---|---|\n")
		}
		for i, r := range rows {
			delta := (r.dps/rows[0].dps - 1) * 100
			if isMatched {
				fmt.Fprintf(&report, "| %d | %s | %s | %.1f ± %.1f | %+.2f%% | %.1f | %.1f | %.1f |\n", i+1, raceName(r.race), weaponName(r.best), r.dps, r.stderr, delta,
					r.byType[proto.WeaponType_WeaponTypeSword], r.byType[proto.WeaponType_WeaponTypeAxe], r.byType[proto.WeaponType_WeaponTypeMace])
			} else {
				fmt.Fprintf(&report, "| %d | %s | %.1f ± %.1f | %+.2f%% |\n", i+1, raceName(r.race), r.dps, r.stderr, delta)
			}
		}
	}

	fmt.Println(report.String())
	if out := os.Getenv("RACE_SWEEP_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(report.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

type warriorRun struct {
	race    proto.Race
	gear    *proto.EquipmentSpec
	talents string
	mobType proto.MobType
	// Optional: the page's No Reck rotation when nil, and no bonus stats.
	rotation *proto.APLRotation
	bonus    *proto.UnitStats
}

func runWarrior(t *testing.T, run warriorRun, iterations int32) (float64, float64) {
	t.Helper()
	rotation := run.rotation
	if rotation == nil {
		rotation = pageRotation()
	}
	player := core.WithSpec(&proto.Player{
		Class:          proto.Class_ClassWarrior,
		Race:           run.race,
		Equipment:      run.gear,
		TalentsString:  run.talents,
		Consumables:    pageConsumables,
		Buffs:          &proto.IndividualBuffs{BlessingOfKings: true, BlessingOfMight: true},
		Profession1:    proto.Profession_Alchemy,
		Profession2:    proto.Profession_Engineering,
		ReactionTimeMs: 200,
		Rotation:       rotation,
		BonusStats:     run.bonus,
	}, pageOptions)

	target := core.FreshDefaultTargetConfig()
	target.MobType = run.mobType
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Targets = []*proto.Target{target}

	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player,
			&proto.PartyBuffs{BattleShout: proto.TristateEffect_TristateEffectImproved, LeaderOfThePack: proto.TristateEffect_TristateEffectRegular},
			&proto.RaidBuffs{GiftOfTheWild: proto.TristateEffect_TristateEffectImproved},
			&proto.Debuffs{CurseOfRecklessness: true, ExposeArmor: proto.TristateEffect_TristateEffectImproved, FaerieFire: proto.TristateEffect_TristateEffectRegular, GiftOfArthas: true, SunderArmor: true}),
		Encounter:  encounter,
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 101},
	})
	if result.Error != nil {
		t.Fatalf("%v: %s", run.race, result.Error.Message)
	}
	return result.RaidMetrics.Dps.Avg, result.RaidMetrics.Dps.Stdev
}

// The page's warrior options and consumables (ui/specs/warrior/dps/presets.ts, shared/presets.ts).
var pageOptions = &proto.Player_DpsWarrior{
	DpsWarrior: &proto.DpsWarrior{
		Options: &proto.DpsWarrior_Options{
			ClassOptions: &proto.WarriorOptions{
				StartingRage:  0,
				QueueDelay:    250,
				DefaultShout:  proto.WarriorShout_WarriorShoutBattle,
				DefaultStance: proto.WarriorStance_WarriorStanceBerserker,
			},
		},
	},
}

var pageConsumables = &proto.ConsumesSpec{
	BattleElixirId:    13452, // Elixir of the Mongoose
	GuardianElixirId:  3825,  // Elixir of Lesser Fortitude
	DefenseElixirId:   13445, // Elixir of Greater Defense
	StrengthBuffId:    12451, // Juju Power
	AttackPowerBuffId: 12460, // Juju Might
	ZanzaId:           8410,  // R.O.I.D.S.
	AlcoholId:         21151, // Rumsey Rum Black Label
	DragonbreathChili: true,
	FoodId:            20452, // Smoked Desert Dumplings
	PotId:             13442, // Mighty Rage Potion
	OhImbueId:         18262, // Elemental Sharpening Stone
	GoblinSapper:      true,
}

func pageRotation() *proto.APLRotation {
	return core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "dps_no_reck").Rotation
}

func raceName(race proto.Race) string {
	return strings.TrimPrefix(race.String(), "Race")
}

func weaponName(weaponType proto.WeaponType) string {
	if weaponType == proto.WeaponType_WeaponTypeUnknown {
		return "as shipped"
	}
	return strings.TrimPrefix(weaponType.String(), "WeaponType")
}

// Where each race's DPS comes from, run on demand:
//
//	RACE_BREAKDOWN=1 go test --tags=with_db ./sim/warrior/dps/ -run TestRaceBreakdown -v
//
// Dual wield on the Pre-BiS weapons' stats (retyped copies, so every race swings the same two
// weapons and only the type label changes), a neutral target, the page's defaults. The baseline is
// a Human on axes: a warrior never uses the Human's Spirit, so on axes it carries no racial. See
// core.RacialBreakdown for how each column is measured. RACE_BREAKDOWN_ITERATIONS overrides the
// iteration count; RACE_BREAKDOWN_OUT writes the table.
func TestRaceBreakdown(t *testing.T) {
	if os.Getenv("RACE_BREAKDOWN") == "" {
		t.Skip("set RACE_BREAKDOWN=1 to run the race breakdown")
	}
	iterations := int32(50000)
	if s := os.Getenv("RACE_BREAKDOWN_ITERATIONS"); s != "" {
		fmt.Sscan(s, &iterations)
	}

	report := core.RacialBreakdown(core.RacialBreakdownConfig{
		Title:        "DPS warrior, dual wield, 180 s, neutral level 63 target",
		Races:        core.ClassRaceCapabilities[proto.Class_ClassWarrior],
		BaselineRace: proto.Race_RaceHuman,
		// Troll, Berserking off: no weapon racial, so it shows what the weapon type does on its own.
		TypeReferenceRace: proto.Race_RaceTroll,
		Gear:              core.GetGearSet("../../../ui/specs/warrior/dps/gear_sets", "p0.bis").GearSet,
		NeutralWeapons:    []proto.WeaponType{proto.WeaponType_WeaponTypeSword, proto.WeaponType_WeaponTypeAxe},
		WeaponRacials: map[proto.Race]proto.WeaponType{
			proto.Race_RaceHuman: proto.WeaponType_WeaponTypeSword,
			proto.Race_RaceOrc:   proto.WeaponType_WeaponTypeAxe,
			proto.Race_RaceDwarf: proto.WeaponType_WeaponTypeMace,
		},
		CooldownRacials: map[proto.Race]int32{
			proto.Race_RaceOrc:      20572,   // Blood Fury
			proto.Race_RaceTroll:    20554,   // Berserking
			proto.Race_RaceNightElf: 1259799, // Elune's Light
			proto.Race_RaceGnome:    1259813, // Eureka!
		},
		Rotation:   pageRotation(),
		Iterations: iterations,
		Run: func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) (float64, float64) {
			return runWarrior(t, warriorRun{race: race, gear: gear, talents: DpsTalents, mobType: proto.MobType_MobTypeMechanical, rotation: rotation, bonus: bonus}, iterations)
		},
	})

	fmt.Println(report)
	if out := os.Getenv("RACE_BREAKDOWN_OUT"); out != "" {
		if err := os.WriteFile(out, []byte(report), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// What a gnome warrior's Eureka! is worth, and how much of it is the rage cut and how much the 10%
// damage, run on demand:
//
//	EUREKA_SPLIT=1 go test --tags=with_db ./sim/warrior/dps/ -run TestEurekaSplit -v
func TestEurekaSplit(t *testing.T) {
	if os.Getenv("EUREKA_SPLIT") == "" {
		t.Skip("set EUREKA_SPLIT=1 to run the Eureka! split")
	}
	iterations := int32(50000)
	if s := os.Getenv("EUREKA_SPLIT_ITERATIONS"); s != "" {
		fmt.Sscan(s, &iterations)
	}
	builds := []struct{ label, gear, talents string }{
		{"Warrior, dual wield (DPS talents, Pre-BiS)", "p0.bis", DpsTalents},
		{"Warrior, two-hander (Arms talents, Launch Arms)", "arms_launch", ArmsTalents},
	}
	report := "# Eureka! split, warrior (180 s, neutral level 63 target)\n\n" + core.EurekaSplitHeader
	for _, build := range builds {
		report += core.EurekaSplit(core.EurekaSplitConfig{
			Label:      build.label,
			SpellID:    1259813,
			Gear:       core.GetGearSet("../../../ui/specs/warrior/dps/gear_sets", build.gear).GearSet,
			Rotation:   pageRotation(),
			Iterations: iterations,
			Run: func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) (float64, float64) {
				return runWarrior(t, warriorRun{race: race, gear: gear, talents: build.talents, mobType: proto.MobType_MobTypeMechanical, rotation: rotation, bonus: bonus}, iterations)
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
