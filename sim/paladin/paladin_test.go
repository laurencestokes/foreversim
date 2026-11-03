package paladin_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/common"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
	"github.com/wowsims/forever/sim/paladin"
	"github.com/wowsims/forever/sim/paladin/retribution"
)

func init() {
	retribution.RegisterRetributionPaladin()
	common.RegisterAllEffects()
}

// The Unsealed Wrath preset shared by the spec page and class suite.
const unsealedWrathTalents = "050250002-501-05233011201301351"

func launchPaladin(talents string) (*core.Simulation, *paladin.Paladin) {
	player := &proto.Player{
		Race: proto.Race_RaceHuman, Class: proto.Class_ClassPaladin,
		Equipment:     core.GetGearSet("../../ui/specs/paladin/retribution/gear_sets", "default").GearSet,
		TalentsString: talents, Spec: &proto.Player_RetributionPaladin{RetributionPaladin: &proto.RetributionPaladin{Options: &proto.RetributionPaladin_Options{ClassOptions: &proto.PaladinOptions{}}}}, Rotation: &proto.APLRotation{},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()
	return sim, sim.Raid.Parties[0].Players[0].(paladin.PaladinAgent).GetPaladin()
}

func withoutPaladinTalent(talents string, number int) string {
	parts := strings.Split(talents, "-")
	for i, size := range paladin.TalentTreeSizes {
		if number > size {
			number -= size
			continue
		}
		digits := []byte(parts[i])
		if number <= len(digits) {
			digits[number-1] = '0'
		}
		parts[i] = string(digits)
		break
	}
	return strings.Join(parts, "-")
}

func TestTwoHandedSpecializationIncludesHolySpellDamage(t *testing.T) {
	field := (&proto.PaladinTalents{}).ProtoReflect().Descriptor().Fields().ByName("two_handed_weapon_specialization")
	_, pal := launchPaladin(unsealedWrathTalents)
	_, base := launchPaladin(withoutPaladinTalent(unsealedWrathTalents, int(field.Number())))
	want := 1 + spelldata.Talent(20111, 3).EffectAt(1).FractionAt(pal.Talents.TwoHandedWeaponSpecialization)
	for _, school := range []stats.SchoolIndex{stats.SchoolIndexPhysical, stats.SchoolIndexHoly} {
		got := pal.PseudoStats.SchoolDamageDealtMultiplier[school] / base.PseudoStats.SchoolDamageDealtMultiplier[school]
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("school %v damage multiplier is %v, want %v", school, got, want)
		}
	}
	threat := 1 + spelldata.Talent(20111, 3).EffectAt(2).FractionAt(pal.Talents.TwoHandedWeaponSpecialization)
	if math.Abs(pal.PseudoStats.ThreatMultiplier/base.PseudoStats.ThreatMultiplier-threat) > 1e-9 {
		t.Error("the damage change lost the two-handed threat reduction")
	}
	t.Logf("Physical and Holy damage multiplier=%v", want)
}

func TestVengeanceCanReachFiveStacks(t *testing.T) {
	sim, pal := launchPaladin(unsealedWrathTalents)
	aura := pal.GetAura("Vengeance" + pal.Label)
	if aura == nil {
		t.Fatal("Vengeance is not registered")
	}
	spell := pal.GetSpell(core.ActionID{SpellID: paladin.HolyStrikeRankMap.Highest().ID})
	if spell == nil {
		t.Fatal("Holy Strike is not registered")
	}
	before := spell.DamageMultiplier
	pal.AutoAttacks.CancelAutoSwing(sim)
	attack := pal.AutoAttacks.MHAuto()
	attack.BonusCritPercent = 100
	for range 6 {
		at := sim.CurrentTime
		attack.CalcAndDealDamage(sim, pal.CurrentTarget, 100, attack.OutcomeMeleeSpecialCritOnly)
		for sim.CurrentTime < at+10*time.Millisecond {
			if sim.Step() {
				t.Fatal("the encounter ended before Vengeance resolved")
			}
		}
	}
	if aura.GetStacks() != 5 {
		t.Fatalf("Vengeance reached %d stacks, want five", aura.GetStacks())
	}
	perStack := spelldata.Talent(20049, 3).ValueAt(pal.Talents.Vengeance) / 100
	want := 1 + 5*perStack
	if got := spell.DamageMultiplier / before; math.Abs(got-want) > 1e-9 {
		t.Errorf("Vengeance damage multiplier is %v, want %v", got, want)
	}
	t.Logf("stacks=%v damage multiplier=%v", aura.GetStacks(), spell.DamageMultiplier/before)
}
