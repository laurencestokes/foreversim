package priest

import (
	"testing"
	"time"

	"github.com/wowsims/forever/sim/arenalib"
	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
)

func init() {
	RegisterPriest()
	common.RegisterAllEffects()
}

// The community builds the rankings page runs on our Forever sim: Shadow 15/0/36 and Smite 31/17/3.
var ShadowTalents = "0253000311--550022501201302251"
var SmiteTalents = "515030031305001031-00505023002-003"

func TestShadowPriest(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		priestSuite("shadow", ShadowTalents, true),
	}))
}

func TestSmitePriest(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		priestSuite("smite", SmiteTalents, false),
	}))
}

func priestSuite(apl string, talents string, preShadowform bool) core.CharacterSuiteConfig {
	return core.CharacterSuiteConfig{
		Class:      proto.Class_ClassPriest,
		Race:       proto.Race_RaceTroll,
		OtherRaces: []proto.Race{proto.Race_RaceUndead, proto.Race_RaceNightElf},

		SpecOptions: core.SpecOptionsCombo{
			Label: "Default",
			SpecOptions: &proto.Player_DpsPriest{
				DpsPriest: &proto.DpsPriest{
					Options: &proto.DpsPriest_Options{
						ClassOptions: &proto.PriestOptions{
							Armor: proto.PriestOptions_InnerFire,
							// Off by default in the UI; on here so the pet stays covered.
							UseShadowfiend: true,
							// Begin the sim already in Shadowform so the opener does not spend a
							// GCD casting it.
							PreShadowform: preShadowform,
						},
					},
				},
			},
		},

		// Naked: the generated item database does not carry the pre-raid set our Forever sim tests
		// with yet, and gives the rest TBC-shaped stats.
		GearSet:  core.GearSetCombo{Label: "Naked", GearSet: &proto.EquipmentSpec{}},
		Talents:  talents,
		Rotation: core.GetAplRotation("../../ui/specs/priest/dps/apls", apl),

		ItemFilter: core.ItemFilter{
			WeaponTypes: []proto.WeaponType{
				proto.WeaponType_WeaponTypeDagger,
				proto.WeaponType_WeaponTypeStaff,
				proto.WeaponType_WeaponTypeMace,
				proto.WeaponType_WeaponTypeOffHand,
			},
			ArmorType: proto.ArmorType_ArmorTypeCloth,
			RangedWeaponTypes: []proto.RangedWeaponType{
				proto.RangedWeaponType_RangedWeaponTypeWand,
			},
			// Blacklist melee enchants that appear on cloth-relevant slots but are never used by
			// casters.
			EnchantBlacklist: []int32{2673, 3225, 3273},
		},
	}
}

// The gnome's Eureka! (1259823): registered only for a gnome priest, and the next covered cast
// spends one of its three charges. Its own heals are not covered (see racials.go and
// ui/sim/spells/core.json), so this checks a damaging spell, Smite.
func TestEurekaGnome(t *testing.T) {
	newPriestSim := func(race proto.Race) (*core.Simulation, *Priest) {
		sim := core.NewSim(&proto.RaidSimRequest{
			SimOptions: &proto.SimOptions{RandomSeed: 1},
			Raid: &proto.Raid{Parties: []*proto.Party{{Buffs: &proto.PartyBuffs{}, Players: []*proto.Player{{
				Name: "Priest", Class: proto.Class_ClassPriest, Race: race, TalentsString: SmiteTalents,
				Equipment: &proto.EquipmentSpec{}, Buffs: &proto.IndividualBuffs{},
				Spec:     arenaPriestOptions,
				Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL},
			}}}}},
			Encounter: core.MakeSingleTargetEncounter(0),
		}, simsignals.CreateSignals())
		sim.Reset()
		return sim, sim.Raid.Parties[0].Players[0].(PriestAgent).GetPriest()
	}

	if _, nonGnome := newPriestSim(proto.Race_RaceTroll); nonGnome.GetSpell(core.ActionID{SpellID: 1259823}) != nil {
		t.Error("a non-gnome priest should not register Eureka!")
	}

	sim, gnome := newPriestSim(proto.Race_RaceGnome)
	eureka := gnome.GetSpell(core.ActionID{SpellID: 1259823})
	if eureka == nil {
		t.Fatal("a gnome priest should register Eureka!")
	}
	if !eureka.Cast(sim, gnome.CurrentTarget) {
		t.Fatal("Eureka! did not cast")
	}
	aura := gnome.GetAura("Eureka!")
	if aura == nil || aura.GetStacks() != 3 {
		t.Fatal("Eureka! did not start with 3 charges")
	}

	smite := gnome.GetSpell(core.ActionID{SpellID: spellData.Smite.Highest().ID})
	if !smite.Cast(sim, gnome.CurrentTarget) {
		t.Fatal("Smite did not cast")
	}
	for sim.CurrentTime < 5*time.Second && aura.GetStacks() == 3 {
		sim.Step()
	}
	if got := aura.GetStacks(); got != 2 {
		t.Errorf("Eureka! has %d charges after a covered cast, want 2", got)
	}
}

// Both priests share ui/specs/priest/dps, so each takes its own talents, gear and rotation out
// of it. The site's options, which leave Inner Fire and Shadowfiend off.
func TestArenaShadow(t *testing.T) {
	arenalib.Run(t, arenaShadow)
}

func TestArenaSmite(t *testing.T) {
	arenalib.Run(t, arenaSmite)
}

// The race tier lists for both priests; skipped unless RACE_ARENA_OUT is set. See
// sim/arenalib/race_arena.go.
func TestRaceArenaShadow(t *testing.T) {
	arenalib.RunRaceArena(t, arenaShadow)
}

func TestRaceArenaSmite(t *testing.T) {
	arenalib.RunRaceArena(t, arenaSmite)
}

var arenaShadow = arenalib.Spec{
	Dir:         "shadow_priest",
	UI:          "priest/dps",
	Class:       proto.Class_ClassPriest,
	Race:        proto.Race_RaceUndead,
	SpecOptions: arenaPriestOptions,
	Role:        arenalib.Caster,
	// Mind Flay is 20 yd in the client (SpellRange 3).
	DistanceFromTarget: 20,
	Talents:            "Shadow",
	GearSets:           []string{"launch", "p0.bis", "p1.bis"},
	Rotations:          []string{"shadow"},
}

var arenaSmite = arenalib.Spec{
	Dir:                "smite_priest",
	UI:                 "priest/dps",
	Class:              proto.Class_ClassPriest,
	Race:               proto.Race_RaceUndead,
	SpecOptions:        arenaPriestOptions,
	Role:               arenalib.Caster,
	DistanceFromTarget: 30,
	Talents:            "Smite",
	GearSets:           []string{"smite_launch"},
	Rotations:          []string{"smite", "smite_lowrank"},
	// The page's default gear is the Shadow set; Smite has its own.
	RaceBuilds: map[string]arenalib.RaceBuild{
		"Smite 31/17/3": {Gear: "smite_launch"},
	},
}

var arenaPriestOptions = &proto.Player_DpsPriest{DpsPriest: &proto.DpsPriest{Options: &proto.DpsPriest_Options{
	ClassOptions: &proto.PriestOptions{},
}}}
