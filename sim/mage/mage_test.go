package mage

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/common" // imported to get item effects included.
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

func init() {
	RegisterMage()
	common.RegisterAllEffects()
}

func TestArcane(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{arcaneSuite()}))
}

func TestFFB(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{ffbSuite()}))
}

func TestFire(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{fireSuite()}))
}

func TestFrost(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{frostSuite()}))
}

func TestFingersOfFrostGivesBlizzardTicksNoShatterBonus(t *testing.T) {
	const noShatter, shatter = "--000000001000000200", "--000000001000300200"
	ticksWithout, directWithout, _ := shatterCrits(noShatter)
	ticksWith, directWith, uptimeWith := shatterCrits(shatter)

	if uptimeWith == 0 {
		t.Fatal("Fingers of Frost was never up")
	}
	if ticksWithout == 0 {
		t.Fatal("no Blizzard tick crit")
	}
	if directWith <= directWithout {
		t.Fatalf("Fire Blast gained no crits with Shatter: %d without, %d with", directWithout, directWith)
	}
	if ticksWith != ticksWithout {
		t.Errorf("Blizzard tick crits moved with Shatter: %d without, %d with", ticksWithout, ticksWith)
	}
}

func newMageSim(talents string) (*core.Simulation, *Mage) {
	suite := arcaneSuite()
	player := core.WithSpec(&proto.Player{
		Class: suite.Class, Race: suite.Race, Equipment: &proto.EquipmentSpec{},
		TalentsString: talents, Rotation: &proto.APLRotation{},
	}, suite.SpecOptions.SpecOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 600, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(*Mage)
}

func TestArcaneMissilesPreservesClearcastingWithBarrage(t *testing.T) {
	for _, rank := range []int32{arcaneMissilesTrainerRank.ID, arcaneMissilesBookRank.ID} {
		for _, barrage := range []bool{false, true} {
			t.Run(fmt.Sprintf("rank=%d/barrage=%v", rank, barrage), func(t *testing.T) {
				sim, mage := newMageSim("0000052")
				barrageAura := mage.GetAuraByID(core.ActionID{SpellID: spellData.MissileBarrageTriggered.Highest().ID})
				mage.ClearCasting.Activate(sim)
				if barrage {
					barrageAura.Activate(sim)
				}
				before := mage.CurrentMana()
				if !mage.GetSpell(core.ActionID{SpellID: rank}).Cast(sim, mage.CurrentTarget) {
					t.Fatal("Arcane Missiles could not start")
				}
				if mage.ClearCasting.IsActive() != barrage {
					t.Errorf("Clearcasting active=%v after starting channel, want %v", mage.ClearCasting.IsActive(), barrage)
				}
				if barrageAura.IsActive() {
					t.Error("Missile Barrage remains active after the channel starts")
				}
				if spent := before - mage.CurrentMana(); math.Abs(spent) > 1e-9 {
					t.Errorf("free channel spent %g mana", spent)
				}
				t.Logf("Clearcasting=%v Barrage=%v channel mana=%g", mage.ClearCasting.IsActive(), barrageAura.IsActive(), before-mage.CurrentMana())
			})
		}
	}
}

func TestArcaneMissilesCanGrantClearcasting(t *testing.T) {
	sim, mage := newMageSim("000005")
	mage.AddStatDynamic(sim, stats.SpellHitPercent, 100)
	missile := mage.GetSpell(core.ActionID{SpellID: arcaneMissilesTrainerRank.PeriodicEffect().TriggerID})
	procs := 0
	for range 100 {
		mage.ClearCasting.Deactivate(sim)
		missile.SkipCastAndApplyEffects(sim, mage.CurrentTarget)
		for end := sim.CurrentTime + 2*time.Second; sim.CurrentTime < end && !sim.Step(); {
		}
		if mage.ClearCasting.IsActive() {
			procs++
		}
	}
	if procs == 0 {
		t.Fatal("landed Arcane Missiles never granted Clearcasting")
	}
	t.Logf("Clearcasting procs from 100 missiles=%d", procs)
}

func TestFlamestrikeBurnCanIgnite(t *testing.T) {
	sim, mage := newMageSim("-005")
	mage.AddStatsDynamic(sim, stats.Stats{stats.SpellHitPercent: 100, stats.SpellCritPercent: 100})
	burn := mage.GetSpell(core.ActionID{SpellID: spellData.Flamestrike.Highest().ID, Tag: 1})
	burn.SkipCastAndApplyEffects(sim, mage.CurrentTarget)
	if mage.Ignite.Dot(mage.CurrentTarget).IsActive() {
		t.Fatal("starting the burn ignited without dealing damage")
	}
	burn.AOEDot().TickOnce(sim)
	ignite := mage.Ignite.Dot(mage.CurrentTarget)
	if !ignite.IsActive() || ignite.SnapshotBaseDamage <= 0 {
		t.Fatal("a critical Flamestrike burn tick did not ignite")
	}
	t.Logf("Ignite damage per tick from Flamestrike burn=%g", ignite.SnapshotBaseDamage)
}

// Each book rank registers beside the trainer's top rank, Arcane Missiles with its own missile. Both ranks of Frost
// Ward run Fire Ward's one category cooldown and keep a shield each, and the higher rank's Fireball burn holds a target.
func TestBookRanksBesideTrainerRanks(t *testing.T) {
	suite := arcaneSuite()
	player := core.WithSpec(&proto.Player{
		Class:     suite.Class,
		Race:      suite.Race,
		Equipment: &proto.EquipmentSpec{},
		Rotation:  &proto.APLRotation{},
	}, suite.SpecOptions.SpecOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()
	mage := sim.Raid.Parties[0].Players[0].(*Mage)
	target := mage.CurrentTarget

	spellOf := func(id int32, tag int32) *core.Spell {
		spell := mage.GetSpell(core.ActionID{SpellID: id, Tag: tag})
		if spell == nil {
			t.Fatalf("spell %d tag %d is not registered", id, tag)
		}
		return spell
	}
	for _, rank := range []*spelldata.Spell{frostboltTrainerRank, frostboltBookRank, arcaneMissilesTrainerRank, arcaneMissilesBookRank} {
		spellOf(rank.ID, 0)
	}
	spellOf(arcaneMissilesTrainerRank.PeriodicEffect().TriggerID, 0)
	spellOf(arcaneMissilesBookRank.PeriodicEffect().TriggerID, 0)

	trainerWard, bookWard := spellOf(frostWardTrainerRank.ID, 0), spellOf(frostWardBookRank.ID, 0)
	if !bookWard.Cast(sim, target) {
		t.Fatal("the book rank of Frost Ward did not cast")
	}
	if trainerWard.CD.IsReady(sim) || spellOf(spellData.FireWard.Highest().ID, 0).CD.IsReady(sim) {
		t.Error("the book rank of Frost Ward left the trainer's rank or Fire Ward ready, want one category cooldown")
	}
	if !bookWard.RelatedSelfBuff.IsActive() || trainerWard.RelatedSelfBuff.IsActive() {
		t.Error("the book rank of Frost Ward shares its shield with the trainer's rank, want a shield each")
	}

	spellOf(fireballTrainerRank.ID, 0)
	spellOf(fireballBookRank.ID, 0)
	trainerBurn := spellOf(fireballTrainerRank.ID, 1).Dot(target)
	bookBurn := spellOf(fireballBookRank.ID, 1).Dot(target)
	trainerBurn.Apply(sim)
	bookBurn.Apply(sim)
	if trainerBurn.IsActive() || !bookBurn.IsActive() {
		t.Error("the book rank's Fireball burn left the trainer's up, want it to replace the trainer's")
	}
	trainerBurn.Apply(sim)
	if trainerBurn.IsActive() || !bookBurn.IsActive() {
		t.Error("the trainer's Fireball burn landed under the book rank, want it refused")
	}
}

func shatterCrits(talents string) (tickCrits, directCrits int32, fingersOfFrostUptime float64) {
	tickID := spellData.BlizzardTriggered.Highest().ID
	directID := spellData.FireBlast.Highest().ID
	suite := frostSuite()
	rotation := fmt.Sprintf(`{"type":"TypeAPL","priorityList":[{"action":{"castSpell":{"spellId":{"spellId":%d}}}},{"action":{"castSpell":{"spellId":{"spellId":%d}}}}]}`,
		directID, spellData.Blizzard.Highest().ID)
	player := core.WithSpec(&proto.Player{
		Class:         suite.Class,
		Race:          suite.Race,
		Equipment:     suite.GearSet.GearSet,
		TalentsString: talents,
		Rotation:      core.APLRotationFromJsonString(rotation),
	}, suite.SpecOptions.SpecOptions)
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter:  &proto.Encounter{Duration: 40, Targets: []*proto.Target{core.NewDefaultTarget()}},
		SimOptions: core.DefaultSimTestOptions,
	})

	metrics := result.RaidMetrics.Parties[0].Players[0]
	for _, action := range metrics.Actions {
		for _, target := range action.Targets {
			switch action.Id.GetSpellId() {
			case tickID:
				tickCrits += target.Crits
			case directID:
				directCrits += target.Crits
			}
		}
	}
	for _, aura := range metrics.Auras {
		if aura.Id.GetSpellId() == spellData.FingersOfFrostTriggered.Highest().ID {
			fingersOfFrostUptime = aura.UptimeSecondsAvg
		}
	}
	return tickCrits, directCrits, fingersOfFrostUptime
}

func BenchmarkMage(b *testing.B) {
	b.Run("Arcane", func(b *testing.B) { core.CharacterBenchmark(b, arcaneSuite()) })
	b.Run("FFB", func(b *testing.B) { core.CharacterBenchmark(b, ffbSuite()) })
	b.Run("Fire", func(b *testing.B) { core.CharacterBenchmark(b, fireSuite()) })
	b.Run("Frost", func(b *testing.B) { core.CharacterBenchmark(b, frostSuite()) })
}

func arcaneSuite() core.CharacterSuiteConfig {
	config := mageSuite(ArcaneTalents, "arcane")
	config.OtherRotations = []core.RotationCombo{
		core.RotationWithSpellIds("arcane_book_ranks", config.Rotation, map[int32]int32{10212: 25345}),
	}
	return config
}

func ffbSuite() core.CharacterSuiteConfig {
	return mageSuite(FFBTalents, "ffb")
}

func fireSuite() core.CharacterSuiteConfig {
	config := mageSuite(FireTalents, "deepFire")
	config.OtherRotations = []core.RotationCombo{
		core.RotationWithSpellIds("deepFire_book_ranks", config.Rotation, map[int32]int32{10151: 25306, 10212: 25345}),
	}
	return config
}

func frostSuite() core.CharacterSuiteConfig {
	config := mageSuite(FrostTalents, "frost")
	config.StartingDistance = 30
	config.OtherRotations = []core.RotationCombo{
		core.RotationWithSpellIds("frost_book_ranks", config.Rotation, map[int32]int32{10181: 25304}),
	}
	return config
}

func mageSuite(talents string, apl string) core.CharacterSuiteConfig {
	return core.CharacterSuiteConfig{
		Class:      proto.Class_ClassMage,
		Race:       proto.Race_RaceHighOrderSkyborne,
		OtherRaces: []proto.Race{proto.Race_RaceGnome},
		SpecOptions: core.SpecOptionsCombo{Label: "Default", SpecOptions: &proto.Player_Mage{
			Mage: &proto.Mage{
				Options: &proto.Mage_Options{
					ClassOptions: &proto.MageOptions{
						DefaultMageArmor: proto.MageArmor_MageArmorMageArmor,
					},
				},
			},
		}},
		Consumables: DefaultConsumables,
		Profession1: proto.Profession_Engineering,
		Profession2: proto.Profession_Tailoring,
		GearSet:     core.GetGearSet("../../ui/specs/mage/dps/gear_sets", "p1Arcane"),
		Talents:     talents,
		Rotation:    core.GetAplRotation("../../ui/specs/mage/dps/apls", apl),
		ItemFilter: core.ItemFilter{
			WeaponTypes: []proto.WeaponType{
				proto.WeaponType_WeaponTypeDagger,
				proto.WeaponType_WeaponTypeSword,
				proto.WeaponType_WeaponTypeOffHand,
				proto.WeaponType_WeaponTypeStaff,
			},
			ArmorType: proto.ArmorType_ArmorTypeCloth,
			RangedWeaponTypes: []proto.RangedWeaponType{
				proto.RangedWeaponType_RangedWeaponTypeWand,
			},
			EnchantBlacklist: []int32{2673, 3225, 3273},
		},
	}
}

var ArcaneTalents = "050205223101311531-5500003-003"
var FFBTalents = "-5550000123011303-2035010311013002"
var FireTalents = "0502052001001-55500001230133031-003"
var FrostTalents = "203205-5-053500011121331251"

var DefaultConsumables = &proto.ConsumesSpec{
	FlaskId: 13512, // Flask of Supreme Power
	FoodId:  13931, // Nightfin Soup
	ElixirIds: []int32{
		13454,  // Greater Arcane Elixir
		13447,  // Elixir of the Sages
		250348, // Elixir of the Whale
		250341, // Greater Mageblood Elixir
	},
	MhImbueId:       20749,  // Brilliant Wizard Oil
	MhWeaponImbueId: 277502, // Scroll of Imbue Precision
	ResourcePotId:   13444,  // Major Mana Potion
}
