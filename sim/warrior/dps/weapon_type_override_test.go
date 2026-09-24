package dps

import (
	"testing"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Notched Shortsword (id 727): a plain one-handed Sword, present in the real generated database,
// so the "as Axe"/"as Mace" tests exercise a real item rather than a synthetic one.
const testSwordItemID = 727

// Axe Specialization is a permanent, build-phase racial aura (see racials.go), so once it
// activates it stays up for the whole (fixed-duration) fight.
const (
	axeSpecializationSpellID   = 20574
	swordSpecializationSpellID = 20597
)

func weaponTypeOverrideRaidResult(t *testing.T, race proto.Race, weaponTypeOverride proto.WeaponType) *proto.RaidSimResult {
	t.Helper()

	equipment := weaponsOnly(testSwordItemID, 0)
	equipment.Items[proto.ItemSlot_ItemSlotMainHand].WeaponTypeOverride = weaponTypeOverride
	rotation := core.GetAplRotation("../../../ui/specs/warrior/dps/apls", "dps_reck").Rotation

	player := core.WithSpec(&proto.Player{
		Race:          race,
		Class:         proto.Class_ClassWarrior,
		Equipment:     equipment,
		Consumables:   &proto.ConsumesSpec{},
		TalentsString: DpsTalents,
		Rotation:      rotation,
	}, DefaultOptions)
	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid:       raid,
		Encounter:  core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{Iterations: 5, RandomSeed: 101},
	})
	if result.Error != nil {
		t.Fatal(result.Error.Message)
	}
	return result
}

// The racial aura is registered on the character (and so shows up in metrics with 0 uptime)
// whether or not it ever actually activates, so "did it apply" means uptime > 0, not merely
// being present in the list.
func auraUptime(result *proto.RaidSimResult, spellID int32) float64 {
	for _, aura := range result.RaidMetrics.Parties[0].Players[0].Auras {
		if aura.Id.GetSpellId() == spellID {
			return aura.UptimeSecondsAvg
		}
	}
	return 0
}

// An Orc with a real sword (no override) should not get Axe Specialization: this is the control
// for TestOrcGetsAxeSpecializationFromWeaponTypeOverride below.
func TestOrcSwordDoesNotGetAxeSpecialization(t *testing.T) {
	result := weaponTypeOverrideRaidResult(t, proto.Race_RaceOrc, proto.WeaponType_WeaponTypeUnknown)
	if uptime := auraUptime(result, axeSpecializationSpellID); uptime != 0 {
		t.Fatalf("Axe Specialization should not apply to a Sword, but was up %v sec", uptime)
	}
}

// Relabelling the Orc's sword as an Axe (debug override) should pick up Axe Specialization
// (spell 20574, +1% crit with axes) exactly as if an actual axe were equipped.
func TestOrcGetsAxeSpecializationFromWeaponTypeOverride(t *testing.T) {
	result := weaponTypeOverrideRaidResult(t, proto.Race_RaceOrc, proto.WeaponType_WeaponTypeAxe)
	uptime := auraUptime(result, axeSpecializationSpellID)
	if uptime != core.LongDuration {
		t.Fatalf("Axe Specialization is a permanent racial, expected %v sec uptime once overridden to Axe, got %v", core.LongDuration, uptime)
	}
}

// A Human with a real sword (no override) gets Sword Specialization: the control for
// TestHumanLosesSwordSpecializationFromWeaponTypeOverride below.
func TestHumanSwordGetsSwordSpecialization(t *testing.T) {
	result := weaponTypeOverrideRaidResult(t, proto.Race_RaceHuman, proto.WeaponType_WeaponTypeUnknown)
	uptime := auraUptime(result, swordSpecializationSpellID)
	if uptime != core.LongDuration {
		t.Fatalf("Sword Specialization is a permanent racial, expected %v sec uptime for a Human wielding a real sword, got %v", core.LongDuration, uptime)
	}
}

// Relabelling the Human's sword as a Mace (debug override) should lose the 2% Sword
// Specialization crit bonus, since the racial now sees a Mace instead of a Sword.
func TestHumanLosesSwordSpecializationFromWeaponTypeOverride(t *testing.T) {
	result := weaponTypeOverrideRaidResult(t, proto.Race_RaceHuman, proto.WeaponType_WeaponTypeMace)
	if uptime := auraUptime(result, swordSpecializationSpellID); uptime != 0 {
		t.Fatalf("Sword Specialization should not apply once the sword is overridden to Mace, but was up %v sec", uptime)
	}
}
