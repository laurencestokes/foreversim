package rogue

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/wowsims/forever/sim/common"
	_ "github.com/wowsims/forever/sim/common" // imported to get item effects included.
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"github.com/wowsims/forever/sim/core/simsignals"
	"github.com/wowsims/forever/sim/core/spelldata"
	"github.com/wowsims/forever/sim/core/stats"
)

func init() {
	RegisterRogue()
	common.RegisterAllEffects()
}

func newLaunchRogue(t *testing.T, talents string, consumables *proto.ConsumesSpec) (*core.Simulation, *Rogue) {
	t.Helper()
	player := core.WithSpec(&proto.Player{
		Class: proto.Class_ClassRogue, Race: proto.Race_RaceHuman,
		Equipment:     core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1_muti").GearSet,
		TalentsString: talents, TalentLayout: proto.TalentLayout_TalentLayoutHotfix,
		Consumables: consumables, Rotation: &proto.APLRotation{},
	}, DefaultOptions)
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid:      core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()
	rogue := sim.Raid.Parties[0].Players[0].(*Rogue)
	rogue.AutoAttacks.CancelAutoSwing(sim)
	rogue.CurrentTarget.PseudoStats.BaseDodgeChance = 0
	rogue.AddStatDynamic(sim, stats.PhysicalHitPercent, 100)
	rogue.AddStatDynamic(sim, stats.PhysicalCritPercent, -100)
	rogue.AddStatDynamic(sim, stats.SpellHitPercent, 100)
	rogue.AddStatDynamic(sim, stats.SpellCritPercent, -100)
	return sim, rogue
}

func TestPoisedDiscountsOneEnergyAbility(t *testing.T) {
	sim, rogue := newLaunchRogue(t, DefaultTalents, nil)
	if !rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).IsActive() {
		t.Fatal("Poised is absent at maximum Energy")
	}
	fullCost := rogue.SinisterStrike.Cost.GetCurrentCost() + 20
	before := rogue.CurrentEnergy()
	if !rogue.SinisterStrike.Cast(sim, rogue.CurrentTarget) {
		t.Fatal("the opener did not cast")
	}
	spent := before - rogue.CurrentEnergy()
	if spent != fullCost-20 || rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).IsActive() {
		t.Fatalf("opener spent %g Energy with Poised active=%t", spent, rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).IsActive())
	}
	sim.CurrentTime += rogue.GCD.TimeToReady(sim)
	rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
	before = rogue.CurrentEnergy()
	if !rogue.SinisterStrike.Cast(sim, rogue.CurrentTarget) || before-rogue.CurrentEnergy() != fullCost {
		t.Fatal("the next ability did not pay its full Energy cost")
	}
	t.Logf("openerCost=%g nextCost=%g PoisedConsumed=true", spent, before-rogue.CurrentEnergy())
}

func TestShankUsesTheOffHandAndLaunchPassives(t *testing.T) {
	for _, tc := range []struct {
		name, talents string
		gain          float64
		cooldown      time.Duration
		points        int32
	}{
		{"Flash of Steel", DefaultTalents, 40, 8 * time.Second, 0},
		{"Boot Knife", "005302212-32-005233011321021151", 25, 10 * time.Second, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, rogue := newLaunchRogue(t, tc.talents, nil)
			rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).Deactivate(sim)
			rogue.SpendEnergy(sim, 70, rogue.NewEnergyMetrics(core.ActionID{}))
			before := rogue.CurrentEnergy()
			if !rogue.Shank.Cast(sim, rogue.CurrentTarget) {
				t.Fatal("Shank did not cast with an off-hand weapon")
			}
			if got := rogue.CurrentEnergy() - before; got != tc.gain-5 {
				t.Errorf("Shank changed Energy by %g, want %g", got, tc.gain-5)
			}
			if rogue.ComboPoints() != tc.points || rogue.Shank.CD.TimeToReady(sim) != tc.cooldown {
				t.Errorf("Shank leaves points=%d cooldown=%s", rogue.ComboPoints(), rogue.Shank.CD.TimeToReady(sim))
			}
			if rogue.Shank.ProcMask != core.ProcMaskEmpty {
				t.Fatal("Shank permits attack procs")
			}
			if tc.name == "Flash of Steel" && rogue.Shank.DamageMultiplier != shankRank.DamageEffect().Percent() {
				t.Errorf("off-hand specialization leaves Shank's damage multiplier at %g", rogue.Shank.DamageMultiplier)
			}
			t.Logf("netEnergy=%g comboPoints=%d cooldown=%s damage=%g", rogue.CurrentEnergy()-before, rogue.ComboPoints(), rogue.Shank.CD.TimeToReady(sim), rogue.Shank.SpellMetrics[rogue.CurrentTarget.UnitIndex].TotalDamage)
			rogue.Equipment[proto.ItemSlot_ItemSlotOffHand] = core.Item{}
			sim.CurrentTime += tc.cooldown
			if rogue.Shank.CanCast(sim, rogue.CurrentTarget) {
				t.Error("Shank can cast without an off-hand weapon")
			}
		})
	}
}

func TestPresetShankLeavesRoomForBootKnife(t *testing.T) {
	preset := core.GetAplRotation("../../ui/specs/rogue/dps/apls", "default").Rotation
	var shank *proto.APLListItem
	for _, item := range preset.GetPriorityList() {
		if item.GetAction().GetCastSpell().GetSpellId().GetSpellId() == shankRank.ID {
			shank = item
			break
		}
	}
	if shank == nil {
		t.Fatal("the Rogue preset has no Shank action")
	}
	const bootKnife = "005302212-32-005233011321021151"
	for _, tc := range []struct {
		name, talents string
		points        int32
		energy        float64
		wantCast      bool
	}{
		{"no Boot Knife at five points", DefaultTalents, 5, 80, true},
		{"Boot Knife at three points", bootKnife, 3, 80, true},
		{"Boot Knife at four points", bootKnife, 4, 80, false},
		{"one rank of Boot Knife at four points", "005302212-32-005233011321021111", 4, 80, false},
		{"no Boot Knife above the Energy limit", DefaultTalents, 5, 81, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			player := core.WithSpec(&proto.Player{
				Class: proto.Class_ClassRogue, Race: proto.Race_RaceHuman,
				Equipment:     core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1_muti").GearSet,
				TalentsString: tc.talents, TalentLayout: proto.TalentLayout_TalentLayoutHotfix,
				Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{shank}, ValueVariables: preset.GetValueVariables()},
			}, DefaultOptions)
			sim := core.NewSim(&proto.RaidSimRequest{
				Raid: core.SinglePlayerRaidProto(player, nil, nil, nil), Encounter: core.MakeSingleTargetEncounter(0),
				SimOptions: &proto.SimOptions{RandomSeed: 1},
			}, simsignals.CreateSignals())
			sim.Reset()
			rogue := sim.Raid.Parties[0].Players[0].(*Rogue)
			rogue.SpendEnergy(sim, rogue.CurrentEnergy()-tc.energy, rogue.NewEnergyMetrics(core.ActionID{}))
			rogue.AddComboPoints(sim, tc.points, rogue.NewComboPointMetrics(core.ActionID{}))
			rogue.Rotation.DoNextAction(sim)
			if cast := !rogue.Shank.IsReady(sim); cast != tc.wantCast {
				t.Errorf("Shank cast = %t at %d combo points and %g Energy, want %t", cast, tc.points, tc.energy, tc.wantCast)
			}
		})
	}
}

func TestVenomComesFromShankWithoutSpendingPoints(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "00530221250551031--0052330003", &proto.ConsumesSpec{MhWeaponImbueId: 8928, OhWeaponImbueId: 8985})
	rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).Deactivate(sim)
	rogue.AddComboPoints(sim, 3, rogue.NewComboPointMetrics(core.ActionID{}))
	rogue.SpendEnergy(sim, 70, rogue.NewEnergyMetrics(core.ActionID{}))
	before := rogue.ModifiedProcChance(rogue.InstantPoison.ClassFlags, 0.2)
	if !rogue.Shank.Cast(sim, rogue.CurrentTarget) || !rogue.GetAuraByID(core.ActionID{SpellID: venomBuff.ID}).IsActive() {
		t.Fatal("Shank did not activate Venom")
	}
	after := rogue.ModifiedProcChance(rogue.InstantPoison.ClassFlags, 0.2)
	if math.Abs(after-before-0.2) > 1e-9 || rogue.ComboPoints() != 3 || rogue.GetAuraByID(core.ActionID{SpellID: venomBuff.ID}).Duration != 6*time.Second {
		t.Errorf("Venom changed chance by %g, leaves %d points and lasts %s", after-before, rogue.ComboPoints(), rogue.GetAuraByID(core.ActionID{SpellID: venomBuff.ID}).Duration)
	}
	if rogue.GetSpell(core.ActionID{SpellID: 1310703}) != nil {
		t.Error("Venom is still a castable finisher")
	}
	t.Logf("poisonChanceBefore=%g after=%g VenomDuration=%s comboPoints=%d", before, after, rogue.GetAuraByID(core.ActionID{SpellID: venomBuff.ID}).Duration, rogue.ComboPoints())
}

func TestBloodSplatterExtendsBothBleeds(t *testing.T) {
	ticks := func(talents string) (int32, int32) {
		sim, rogue := newLaunchRogue(t, talents, nil)
		rogue.StealthAura.Activate(sim)
		if !rogue.Garrote.Cast(sim, rogue.CurrentTarget) {
			t.Fatal("Garrote did not cast from Stealth")
		}
		garroteTicks := rogue.Garrote.Dot(rogue.CurrentTarget).ExpectedTickCount()
		sim.CurrentTime += rogue.GCD.TimeToReady(sim)
		rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
		rogue.AddComboPoints(sim, 5, rogue.NewComboPointMetrics(core.ActionID{}))
		if !rogue.Rupture.Cast(sim, rogue.CurrentTarget) {
			t.Fatal("Rupture did not cast with combo points")
		}
		return garroteTicks, rogue.Rupture.Dot(rogue.CurrentTarget).ExpectedTickCount()
	}
	g0, r0 := ticks("")
	g1, r1 := ticks("005302212")
	if g1-g0 != 3 || r1-r0 != 3 {
		t.Errorf("extra bleed ticks: Garrote=%d Rupture=%d", g1-g0, r1-r0)
	}
	t.Logf("GarroteTicks=%d→%d RuptureTicks=%d→%d", g0, g1, r0, r1)
}

func TestGhostlyStrikeOpensOneStealthAbility(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "005302212-32-005233011321021151", nil)
	rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).Deactivate(sim)
	dodge := rogue.GetAuraByID(core.ActionID{SpellID: ghostlyStrikeRank.ID, Tag: 1})
	legacy := rogue.GetAuraByID(core.ActionID{SpellID: ghostlyStrikeRank.ID})
	if rogue.Ambush.CanCast(sim, rogue.CurrentTarget) || rogue.Garrote.CanCast(sim, rogue.CurrentTarget) {
		t.Fatal("a stealth opener is available without Stealth or Ghostly Strike")
	}
	if !rogue.GhostlyStrike.Cast(sim, rogue.CurrentTarget) {
		t.Fatal("Ghostly Strike did not cast")
	}
	if !legacy.IsActive() || !rogue.GhostlyStrikeAura.IsActive() || rogue.GhostlyStrikeAura.ActionID.SpellID != 1329326 {
		t.Fatal("Ghostly Strike did not grant its opener window")
	}
	sim.CurrentTime += rogue.GCD.TimeToReady(sim)
	rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
	if !rogue.Garrote.Cast(sim, rogue.CurrentTarget) || rogue.GhostlyStrikeAura.IsActive() || legacy.IsActive() {
		t.Fatal("Ghostly Strike did not grant and spend the opener window")
	}
	if !dodge.IsActive() || dodge.Duration != 7*time.Second {
		t.Fatal("the opener spent or shortened Ghostly Strike's dodge bonus")
	}
	sim.CurrentTime += rogue.GCD.TimeToReady(sim)
	rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
	if rogue.Ambush.CanCast(sim, rogue.CurrentTarget) {
		t.Error("the spent Ghostly Strike window permits another opener")
	}
	t.Log("Ghostly Strike granted Garrote outside Stealth and the cast consumed the window")
}

func TestDeathMarkStoresBleedsTwiceAndDoesNotStoreItself(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "005302212-32-005233011321021151", nil)
	rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).Deactivate(sim)
	target := rogue.CurrentTarget
	rogue.AddComboPoints(sim, 5, rogue.NewComboPointMetrics(core.ActionID{}))
	if !rogue.DeathMark.Cast(sim, target) || rogue.DeathMark.ComboPointMetrics().ActualGainForCurrentIteration() != -5 {
		t.Fatalf("Death Mark did not cast and spend its combo points: remaining=%d", rogue.ComboPoints())
	}
	// Deal one ordinary hit and one bleed tick through the class spells' normal result dispatch.
	direct := rogue.SinisterStrike.CalcAndDealDamage(sim, target, 20, rogue.SinisterStrike.OutcomeAlwaysHit).Damage
	bleed := rogue.Rupture.CalcAndDealPeriodicDamage(sim, target, 20, rogue.Rupture.Dot(target).OutcomeTick).Damage
	coup := spellData.CoupDeGrace.EffectAt(2).MultiplierAt(rogue.Talents.CoupDeGrace)
	expected := (direct + 2*bleed) * 0.6 * coup
	dot := rogue.DeathMark.Dot(target)
	dot.TickOnce(sim)
	first := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage
	dot.TickOnce(sim)
	second := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage - first
	if math.Abs(first-expected) > 1e-6 || math.Abs(second-first) > 1e-6 {
		t.Errorf("Death Mark dealt %g then %g, want %g on both ticks", first, second, expected)
	}
	// A fresh mark replaces its storage; a large hit is limited by the captured base/AP cap.
	sim.CurrentTime += rogue.DeathMark.CD.TimeToReady(sim)
	rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
	rogue.AddComboPoints(sim, 5, rogue.NewComboPointMetrics(core.ActionID{}))
	if !rogue.DeathMark.Cast(sim, target) {
		t.Fatal("Death Mark did not refresh")
	}
	rogue.SinisterStrike.CalcAndDealDamage(sim, target, 100000, rogue.SinisterStrike.OutcomeAlwaysHit)
	cap := (156 + 245*5 + 0.04*rogue.DeathMark.MeleeAttackPower(target)*5) * coup * (1 + rogue.GetStat(stats.PhysicalCritPercent)/100)
	before := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage
	dot.TickOnce(sim)
	if got := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage - before; math.Abs(got-cap) > 1e-6 {
		t.Errorf("Death Mark dealt %g at its cap, want %g", got, cap)
	}
	t.Logf("ordinaryHit=%g bleedTick=%g markDamage=%g repeatedDamage=%g cap=%g", direct, bleed, first, second, cap)
}

func TestHemorrhageAmplifiesBothCasterBleeds(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "005302212-32-005233011321021151", nil)
	rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).Deactivate(sim)
	target := rogue.CurrentTarget
	before := []float64{}
	for _, bleed := range []*core.Spell{rogue.Garrote, rogue.Rupture} {
		before = append(before, bleed.CalcAndDealPeriodicDamage(sim, target, 20, bleed.Dot(target).OutcomeTick).Damage)
	}
	if !rogue.Hemorrhage.Cast(sim, target) {
		t.Fatal("Hemorrhage did not cast")
	}
	for i, bleed := range []*core.Spell{rogue.Garrote, rogue.Rupture} {
		after := bleed.CalcAndDealPeriodicDamage(sim, target, 20, bleed.Dot(target).OutcomeTick).Damage
		if math.Abs(after/before[i]-1.15) > 1e-9 {
			t.Errorf("%s damage changed from %g to %g", bleed.ActionID, before[i], after)
		}
		t.Logf("%s periodicDamageBefore=%g after=%g", bleed.ActionID, before[i], after)
	}
}

func TestSealFateSharesItsCooldownAcrossMutilateStrikes(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "00530221250551031--0052330003", nil)
	rogue.AddStatDynamic(sim, stats.PhysicalCritPercent, 200)
	if !rogue.Mutilate.Cast(sim, rogue.CurrentTarget) {
		t.Fatal("Mutilate did not cast with two daggers")
	}
	if rogue.ComboPoints() != 3 {
		t.Errorf("two Mutilate critical strikes leave %d combo points", rogue.ComboPoints())
	}
	if rogue.MutilateMH.SpellMetrics[rogue.CurrentTarget.UnitIndex].Crits != 1 || rogue.MutilateOH.SpellMetrics[rogue.CurrentTarget.UnitIndex].Crits != 1 {
		t.Fatal("both Mutilate strikes must crit to exercise Seal Fate's shared cooldown")
	}
	if rogue.MutilateMH.CritMultiplierAdditive != 0.2 || rogue.MutilateOH.CritMultiplierAdditive != 0.2 {
		t.Error("Lethality does not reach both Mutilate strikes")
	}
	t.Logf("MutilateComboPoints=%d mhCritBonus=%g ohCritBonus=%g", rogue.ComboPoints(), rogue.MutilateMH.CritMultiplierAdditive, rogue.MutilateOH.CritMultiplierAdditive)
}

func TestPoisonAttackPowerShares(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "", &proto.ConsumesSpec{MhWeaponImbueId: 8928, OhWeaponImbueId: 8985})
	target := rogue.CurrentTarget
	rogue.InstantPoison.Flags |= core.SpellFlagIgnoreResists
	rogue.InstantPoison.DirectVariance = 0
	rogue.DeadlyPoison.Flags |= core.SpellFlagIgnoreResists
	if !rogue.DeadlyPoison.Cast(sim, target) {
		t.Fatal("Deadly Poison did not apply")
	}
	damage := func() (float64, float64) {
		before := rogue.InstantPoison.SpellMetrics[target.UnitIndex].TotalDamage
		rogue.InstantPoison.Cast(sim, target)
		instant := rogue.InstantPoison.SpellMetrics[target.UnitIndex].TotalDamage - before
		before = rogue.DeadlyPoison.SpellMetrics[target.UnitIndex].TotalDamage
		rogue.DeadlyPoison.Dot(target).TickOnce(sim)
		return instant, rogue.DeadlyPoison.SpellMetrics[target.UnitIndex].TotalDamage - before
	}
	i0, d0 := damage()
	rogue.AddStatDynamic(sim, stats.AttackPower, 1000)
	i1, d1 := damage()
	numTicks := float64(rogue.DeadlyPoison.Dot(target).BaseTickCount)
	if math.Abs(i1-i0-10) > 1e-9 || math.Abs((d1-d0)*numTicks-9) > 1e-9 {
		t.Errorf("Attack Power added %g Instant damage and %g Deadly damage over its duration", i1-i0, (d1-d0)*numTicks)
	}
	t.Logf("extraInstantDamage=%g extraDeadlyTickDamage=%g ticks=%g", i1-i0, d1-d0, numTicks)
}

// Each book rank registers beside the trainer's top rank. Backstab builds a combo point at either rank, Eviscerate
// spends them all at either rank and hits harder at the book rank, and both ranks of Feint run one category cooldown.
func TestBookRanksBesideTrainerRanks(t *testing.T) {
	newRogue := func(rank *spelldata.Spell) (*core.Simulation, *Rogue, *core.Spell) {
		player := core.WithSpec(&proto.Player{
			Class:     proto.Class_ClassRogue,
			Race:      proto.Race_RaceHuman,
			Equipment: core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1_muti").GearSet,
			Rotation:  &proto.APLRotation{},
		}, DefaultOptions)
		sim := core.NewSim(&proto.RaidSimRequest{
			Raid:       core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
			Encounter:  core.MakeSingleTargetEncounter(0),
			SimOptions: &proto.SimOptions{RandomSeed: 1},
		}, simsignals.CreateSignals())
		sim.Reset()
		rogue := sim.Raid.Parties[0].Players[0].(*Rogue)
		rogue.CurrentTarget.PseudoStats.BaseDodgeChance = 0
		rogue.AddStatDynamic(sim, stats.PhysicalHitPercent, 100)
		rogue.AddStatDynamic(sim, stats.PhysicalCritPercent, -100)
		spell := rogue.GetSpell(core.ActionID{SpellID: rank.ID})
		if spell == nil {
			t.Fatalf("%s is not registered", rank.RankedName())
		}
		return sim, rogue, spell
	}

	for _, rank := range []*spelldata.Spell{backstabTrainerRank, backstabBookRank} {
		sim, rogue, backstab := newRogue(rank)
		if !backstab.Cast(sim, rogue.CurrentTarget) {
			t.Fatalf("%s did not cast", rank.RankedName())
		}
		if rogue.ComboPoints() != 1 || !backstab.Matches(RogueSpellBackstab) {
			t.Errorf("%s leaves %d combo points, or lost the Backstab mask; want 1", rank.RankedName(), rogue.ComboPoints())
		}
	}

	eviscerate := func(rank *spelldata.Spell) float64 {
		sim, rogue, spell := newRogue(rank)
		target := rogue.CurrentTarget
		rogue.AddComboPoints(sim, 5, rogue.NewComboPointMetrics(core.ActionID{SpellID: 1}))
		if !spell.Cast(sim, target) {
			t.Fatalf("%s did not cast", rank.RankedName())
		}
		if rogue.ComboPoints() != 0 {
			t.Errorf("%s leaves %d combo points, want 0", rank.RankedName(), rogue.ComboPoints())
		}
		return spell.SpellMetrics[target.UnitIndex].TotalDamage
	}
	if trainer, book := eviscerate(eviscerateTrainerRank), eviscerate(eviscerateBookRank); book <= trainer {
		t.Errorf("the book rank of Eviscerate deals %.0f at 5 combo points, want more than the trainer rank's %.0f", book, trainer)
	}

	sim, rogue, trainerFeint := newRogue(feintTrainerRank)
	bookFeint := rogue.GetSpell(core.ActionID{SpellID: feintBookRank.ID})
	if bookFeint == nil {
		t.Fatalf("%s is not registered", feintBookRank.RankedName())
	}
	if !trainerFeint.Cast(sim, rogue.CurrentTarget) {
		t.Fatal("the trainer rank of Feint did not cast")
	}
	if bookFeint.CD.IsReady(sim) {
		t.Error("the trainer rank of Feint left the book rank ready, want one category cooldown")
	}
}

func TestRogue(t *testing.T) {
	rotation := core.GetAplRotation("../../ui/specs/rogue/dps/apls", "default")
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:         proto.Class_ClassRogue,
			Race:          proto.Race_RaceHuman,
			OtherRaces:    []proto.Race{proto.Race_RaceOrc},
			GearSet:       core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1_muti"),
			OtherGearSets: []core.GearSetCombo{
				//core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1"),
			},
			Talents: "005303202014-3203032201012321501",
			OtherTalentSets: []core.TalentsCombo{
				{
					Label:   "Subtlety BS",
					Talents: "005303202-3203-0352330313210011",
				},
				{
					Label:   "Asn Muti",
					Talents: "00530320251511031-30230322000103",
				},
			},
			Consumables: DefaultConsumables,
			SpecOptions: core.SpecOptionsCombo{Label: "Rogue", SpecOptions: DefaultOptions},

			Rotation: rotation,
			OtherRotations: []core.RotationCombo{
				core.RotationWithSpellIds("default_book_ranks", rotation, map[int32]int32{11300: 31016, 11281: 25300}),
			},
			ItemFilter: core.ItemFilter{
				ArmorType: proto.ArmorType_ArmorTypeLeather,
				WeaponTypes: []proto.WeaponType{
					proto.WeaponType_WeaponTypeDagger,
					proto.WeaponType_WeaponTypeFist,
					proto.WeaponType_WeaponTypeMace,
					proto.WeaponType_WeaponTypeSword,
					proto.WeaponType_WeaponTypeAxe,
				},
				HandTypes: []proto.HandType{
					proto.HandType_HandTypeMainHand,
					proto.HandType_HandTypeOffHand,
					proto.HandType_HandTypeOneHand,
				},
			},
		},
	}))
}

var DefaultOptions = &proto.Player_Rogue{
	Rogue: &proto.Rogue{
		Options: &proto.Rogue_Options{
			ClassOptions: &proto.RogueOptions{},
		},
	},
}

var DefaultTalents = "005302212-3200330202012321151-005"

var DefaultConsumables = &proto.ConsumesSpec{
	FlaskId:         274274, // Flask of Natural Aggression
	FoodId:          250072, // Swiftstrike Steak
	OffensivePotId:  250943, // Major Frenzy Potion
	ConjuredId:      7676,
	MhWeaponImbueId: 8928, // Instant Poison VI
	OhWeaponImbueId: 8985, // Deadly Poison IV
}

// Lightning Reflexes 5/5, Deflection 3/3, Riposte, Setup 3/3 and Ghostly Strike.
const tankTalents = "-005030001-300000000001"

// A rogue with the talents given, facing two targets.
func newTankSim(t *testing.T, talents string) (*core.Simulation, *Rogue, []*core.Unit) {
	t.Helper()

	player := &proto.Player{
		Name:          "Tank",
		Race:          proto.Race_RaceHuman,
		Class:         proto.Class_ClassRogue,
		Equipment:     core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1_muti").GearSet,
		TalentsString: talents,
		TalentLayout:  proto.TalentLayout_TalentLayoutHotfix,
		Spec:          DefaultOptions,
		Rotation:      &proto.APLRotation{},
	}
	sim := core.NewSim(&proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration: core.LongDuration,
			Targets:  []*proto.Target{core.NewDefaultTarget(), core.NewDefaultTarget()},
		},
		SimOptions: &proto.SimOptions{RandomSeed: 1},
	}, simsignals.CreateSignals())
	sim.Reset()

	return sim, sim.Raid.Parties[0].Players[0].(RogueAgent).GetRogue(), sim.Encounter.AllTargetUnits
}

// A trigger without TriggerImmediately runs its handler one batch window after the hit.
func passBatchWindow(t *testing.T, sim *core.Simulation) {
	t.Helper()
	until := sim.CurrentTime + core.SpellBatchWindow
	sim.AddPendingAction(&core.PendingAction{NextActionAt: until + 1, OnAction: func(*core.Simulation) {}})
	for sim.CurrentTime <= until {
		if sim.Step() {
			t.Fatal("the sim ended inside the batch window")
		}
	}
}

func auraOf(t *testing.T, rogue *Rogue, id int32) *core.Aura {
	t.Helper()
	aura := rogue.GetAuraByID(core.ActionID{SpellID: id})
	if aura == nil {
		t.Fatalf("the rogue has no aura %d", id)
	}
	return aura
}

func swingAt(sim *core.Simulation, rogue *Rogue, from *core.Unit, outcome core.HitOutcome) {
	rogue.OnSpellHitTaken(sim, &core.Spell{ProcMask: core.ProcMaskMeleeMHAuto, Unit: from},
		&core.SpellResult{Target: &rogue.Unit, Outcome: outcome})
}

// Defensive State (16092) hears melee hits taken, and Riposte's window opens on a parry only.
func TestRiposteWindowOpensOnParriedSwings(t *testing.T) {
	for _, c := range []struct {
		name    string
		outcome core.HitOutcome
		opens   bool
	}{
		{"a parried swing opens the window", core.OutcomeParry, true},
		{"a dodged swing does not open the window", core.OutcomeDodge, false},
		{"a landed swing does not open the window", core.OutcomeHit, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			sim, rogue, targets := newTankSim(t, tankTalents)
			window := auraOf(t, rogue, riposteWindow.ID)

			swingAt(sim, rogue, targets[0], c.outcome)
			passBatchWindow(t, sim)

			if got := window.IsActive(); got != c.opens {
				t.Errorf("the window is up: %v, want %v", got, c.opens)
			}
		})
	}
}

// Riposte is off the GCD, and only its own cast spends the window: a Sinister Strike leaves it up.
func TestRiposteCastsOffTheGCDAndSpendsItsWindow(t *testing.T) {
	sim, rogue, targets := newTankSim(t, tankTalents)
	window := auraOf(t, rogue, riposteWindow.ID)

	swingAt(sim, rogue, targets[0], core.OutcomeParry)
	passBatchWindow(t, sim)

	if !rogue.SinisterStrike.Cast(sim, targets[0]) {
		t.Fatal("Sinister Strike did not cast")
	}
	if !window.IsActive() {
		t.Fatal("Sinister Strike spent the Riposte window")
	}
	if rogue.GCD.IsReady(sim) {
		t.Fatal("Sinister Strike left the GCD ready")
	}
	if !rogue.Riposte.Cast(sim, targets[0]) {
		t.Fatal("Riposte did not cast while the GCD was up")
	}
	if window.IsActive() {
		t.Error("Riposte's cast left its window up")
	}
}

// Lightning Reflexes 5/5 and Deflection 3/3 add their rank's dodge and parry, and Ghostly Strike's buff its
// second effect's dodge.
func TestAvoidanceTalentsAddTheirRowsAmounts(t *testing.T) {
	_, bare, _ := newTankSim(t, "")
	sim, rogue, _ := newTankSim(t, tankTalents)

	wantDodge := spellData.LightningReflexes.Rank(5).EffectN(1).BaseValue()
	if got := rogue.GetStat(stats.DodgePercent) - bare.GetStat(stats.DodgePercent); got != wantDodge {
		t.Errorf("Lightning Reflexes adds %.2f%% dodge, want %.2f%%", got, wantDodge)
	}
	wantParry := spellData.Deflection.Rank(3).EffectN(1).BaseValue()
	if got := rogue.GetStat(stats.ParryPercent) - bare.GetStat(stats.ParryPercent); got != wantParry {
		t.Errorf("Deflection adds %.2f%% parry, want %.2f%%", got, wantParry)
	}

	buff := rogue.GetAuraByID(core.ActionID{SpellID: ghostlyStrikeRank.ID, Tag: 1})
	before := rogue.GetStat(stats.DodgePercent)
	buff.Activate(sim)
	if got, want := rogue.GetStat(stats.DodgePercent)-before, ghostlyStrikeRank.EffectN(2).BaseValue(); got != want {
		t.Errorf("Ghostly Strike's buff adds %.2f%% dodge, want %.2f%%", got, want)
	}
}

// Ghostly Strike puts its dodge buff up when it lands, and an avoided strike grants none.
func TestGhostlyStrikeBuffsOnlyWhenItLands(t *testing.T) {
	sim, rogue, targets := newTankSim(t, tankTalents)
	target := targets[0]
	buff := rogue.GetAuraByID(core.ActionID{SpellID: ghostlyStrikeRank.ID, Tag: 1})
	metrics := &rogue.GhostlyStrike.SpellMetrics[target.UnitIndex]
	avoided := func() int32 { return metrics.Misses + metrics.Dodges + metrics.Parries }

	var landedSeen, avoidedSeen bool
	for tries := 0; tries < 500 && !(landedSeen && avoidedSeen); tries++ {
		for rogue.CurrentEnergy() < rogue.GhostlyStrike.DefaultCast.Cost || !rogue.GCD.IsReady(sim) {
			if sim.Step() {
				t.Fatal("the sim ended before Ghostly Strike was castable")
			}
		}
		buff.Deactivate(sim)
		rogue.GhostlyStrike.CD.Reset()
		before := avoided()
		if !rogue.GhostlyStrike.Cast(sim, target) {
			t.Fatal("Ghostly Strike did not cast")
		}

		if avoided() > before {
			avoidedSeen = true
			if buff.IsActive() {
				t.Fatal("an avoided Ghostly Strike put its buff up")
			}
		} else {
			landedSeen = true
			if !buff.IsActive() {
				t.Fatal("a landed Ghostly Strike left its buff down")
			}
		}
	}
	if !landedSeen || !avoidedSeen {
		t.Fatalf("saw a landed strike: %v, an avoided one: %v", landedSeen, avoidedSeen)
	}
}

// Remorseless Attacks hears the killing blow: a credited kill gives Sinister Strike the crit of the
// talent's rank, a death the raid gets no kill for does not, and the strike spends it.
func TestRemorselessAttacksCritsTheStrikeAfterAKill(t *testing.T) {
	sim, rogue, _ := newTankSim(t, "01")
	dying, other := sim.Encounter.AllTargets[0], sim.Encounter.AllTargetUnits[1]
	buff := rogue.GetAuraByID(core.ActionID{SpellID: remorselessRank.ID})
	strike := rogue.SinisterStrike
	base := strike.BonusCritPercent
	bonus := spellData.RemorselessAttacks.Rank(1).EffectN(1).BaseValue()

	dying.Die(sim, 0)
	passBatchWindow(t, sim)
	if buff.IsActive() {
		t.Error("a death with no kill credit put up Remorseless")
	}

	dying.Enable(sim)
	dying.Die(sim, 1)
	passBatchWindow(t, sim)
	if got := strike.BonusCritPercent - base; !buff.IsActive() || math.Abs(got-bonus) > 1e-9 {
		t.Fatalf("after a killing blow Remorseless is up %v and adds %v crit to Sinister Strike, want up and %v", buff.IsActive(), got, bonus)
	}

	strike.SkipCastAndApplyEffects(sim, other)
	passBatchWindow(t, sim)
	if buff.IsActive() || strike.BonusCritPercent != base {
		t.Errorf("after Sinister Strike Remorseless is up %v at %v crit, want gone and %v", buff.IsActive(), strike.BonusCritPercent, base)
	}
}

// A miss, dodge or parry of the strike spends Remorseless as a landed one does, and the landed strike
// still rolls its crit with the bonus.
func TestRemorselessAttacksSpendsOnAvoidedStrikesAndCritsLandedOnes(t *testing.T) {
	sim, rogue, _ := newTankSim(t, "01")
	target := sim.Encounter.AllTargetUnits[1]
	buff := rogue.GetAuraByID(core.ActionID{SpellID: remorselessRank.ID})
	strike := rogue.SinisterStrike
	table := rogue.AttackTables[target.UnitIndex].Hand(strike)
	bonus := spellData.RemorselessAttacks.Rank(1).EffectN(1).BaseValue()
	table.BaseMissChance, table.BaseDodgeChance, table.BaseParryChance, table.BaseBlockChance = 0, 0, 0, 0

	for _, c := range []struct {
		name string
		set  func()
	}{
		{"a miss", func() { table.BaseMissChance = 1 }},
		{"a dodge", func() { table.BaseDodgeChance = 1 }},
		{"a parry", func() { table.BaseParryChance = 1 }},
	} {
		c.set()
		buff.Activate(sim)
		strike.SkipCastAndApplyEffects(sim, target)
		if buff.IsActive() || strike.BonusCritPercent != 0 {
			t.Errorf("after %s Remorseless is up %v at %v crit, want gone and 0", c.name, buff.IsActive(), strike.BonusCritPercent)
		}
		table.BaseMissChance, table.BaseDodgeChance, table.BaseParryChance = 0, 0, 0
	}

	const casts = 2000
	critRate := func(withBuff bool) float64 {
		before := strike.SpellMetrics[target.UnitIndex].Crits
		for range casts {
			buff.Deactivate(sim)
			if withBuff {
				buff.Activate(sim)
			}
			strike.SkipCastAndApplyEffects(sim, target)
		}
		return float64(strike.SpellMetrics[target.UnitIndex].Crits-before) / casts
	}
	with, without := critRate(true), critRate(false)
	if with-without < bonus/200 {
		t.Errorf("landed strikes crit %v with Remorseless and %v without, want the bonus of %v points on top", with, without, bonus)
	}
}

// Setup rolls only for the current target's dodged attack, using the launch talent's rank.
func TestSetupUsesItsLaunchChanceOnTheCurrentTargetsDodges(t *testing.T) {
	for rank := int32(1); rank <= 3; rank++ {
		t.Run(fmt.Sprint(rank), func(t *testing.T) {
			sim, rogue, targets := newTankSim(t, "--"+fmt.Sprint(rank))
			rogue.AutoAttacks.CancelAutoSwing(sim)
			aura := rogue.GetAura("Setup")
			metrics := rogue.NewComboPointMetrics(core.ActionID{})
			for _, tc := range []struct {
				name    string
				from    int
				outcome core.HitOutcome
				chance  float64
			}{
				{"current target dodge", 0, core.OutcomeDodge, float64(rank) * 0.15},
				{"other target dodge", 1, core.OutcomeDodge, 0},
				{"current target parry", 0, core.OutcomeParry, 0},
				{"current target hit", 0, core.OutcomeHit, 0},
			} {
				attempts := 100
				if tc.chance > 0 {
					attempts = 1000
				}
				gained := int32(0)
				for range attempts {
					rogue.SpendComboPoints(sim, metrics)
					if aura.Icd != nil {
						aura.Icd.Reset()
					}
					swingAt(sim, rogue, targets[tc.from], tc.outcome)
					passBatchWindow(t, sim)
					gained += rogue.ComboPoints()
				}
				got := float64(gained) / float64(attempts)
				if (tc.chance == 0 && gained != 0) || math.Abs(got-tc.chance) > 0.05 {
					t.Errorf("%s proc chance = %g, want %g", tc.name, got, tc.chance)
				}
				t.Logf("rank=%d %s: points=%d attempts=%d rate=%g", rank, tc.name, gained, attempts, got)
			}
		})
	}
}

func TestDeathMarkCapUsesCurrentPlayerMeleeCrit(t *testing.T) {
	for _, crit := range []float64{0, 50} {
		t.Run(fmt.Sprint(crit), func(t *testing.T) {
			sim, rogue := newLaunchRogue(t, "005302212-32-005233011321021151", nil)
			rogue.AddStatDynamic(sim, stats.PhysicalCritPercent, crit-rogue.GetStat(stats.PhysicalCritPercent))
			rogue.AddComboPoints(sim, 5, rogue.NewComboPointMetrics(core.ActionID{}))
			target := rogue.CurrentTarget
			if !rogue.DeathMark.Cast(sim, target) {
				t.Fatal("Death Mark did not cast")
			}
			rogue.SinisterStrike.CalcAndDealDamage(sim, target, 100000, rogue.SinisterStrike.OutcomeAlwaysHit)
			coup := spellData.CoupDeGrace.Rank(rogue.Talents.CoupDeGrace).EffectN(2).Percent() + 1
			want := (156 + 245*5 + 0.04*rogue.DeathMark.MeleeAttackPower(target)*5) * (1 + (crit+40)/100) * coup
			rogue.AddStatDynamic(sim, stats.PhysicalCritPercent, 40)
			before := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage
			rogue.DeathMark.Dot(target).TickOnce(sim)
			got := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage - before
			if math.Abs(got-want) > 1e-6 {
				t.Errorf("cap at %g%% melee crit = %g, want %g", crit, got, want)
			}
			t.Logf("applicationCrit=%g%% critAfterCast=%g%% cap=%g", crit, rogue.GetStat(stats.PhysicalCritPercent), got)
		})
	}
}

func TestDeathMarkKeepsEachTargetsComboPoints(t *testing.T) {
	sim, rogue, targets := newTankSim(t, "005302212-32-005233011321021151")
	rogue.AutoAttacks.CancelAutoSwing(sim)
	rogue.AddStatDynamic(sim, stats.PhysicalHitPercent, 100)
	for i, target := range targets {
		rogue.AttackTables[target.UnitIndex].Hand(rogue.DeathMark).BaseDodgeChance = 0
		rogue.AddComboPoints(sim, int32(1+4*i), rogue.NewComboPointMetrics(core.ActionID{}))
		rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
		rogue.DeathMark.CD.Reset()
		sim.CurrentTime += rogue.GCD.TimeToReady(sim)
		if !rogue.DeathMark.Cast(sim, target) {
			t.Fatal("Death Mark did not cast")
		}
	}
	coup := spellData.CoupDeGrace.EffectAt(2).MultiplierAt(rogue.Talents.CoupDeGrace)
	for i, target := range targets {
		dealt := rogue.SinisterStrike.CalcAndDealDamage(sim, target, 20, rogue.SinisterStrike.OutcomeAlwaysHit).Damage
		rogue.DeathMark.Dot(target).TickOnce(sim)
		got := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage
		want := dealt * deathMarkStoredPerPoint * float64(1+4*i) * coup
		if math.Abs(got-want) > 1e-6 {
			t.Errorf("target %d mark dealt %g, want %g", i, got, want)
		}
	}
}

func TestShankIgnoresAvoidanceAndSuppressesProcs(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "005302212-32-005233011321021151", nil)
	rogue.GetAuraByID(core.ActionID{SpellID: poisedBuff.ID}).Deactivate(sim)
	rogue.SpendEnergy(sim, 70, rogue.NewEnergyMetrics(core.ActionID{}))
	target := rogue.CurrentTarget
	table := rogue.AttackTables[target.UnitIndex].Hand(rogue.Shank)
	table.BaseMissChance, table.BaseDodgeChance, table.BaseParryChance, table.BaseBlockChance = 2, 2, 2, 2
	before := rogue.CurrentEnergy()
	if !rogue.Shank.Cast(sim, target) {
		t.Fatal("Shank did not cast")
	}
	metrics := rogue.Shank.SpellMetrics[target.UnitIndex]
	if metrics.Misses+metrics.Dodges+metrics.Parries+metrics.Blocks != 0 {
		t.Fatal("Shank was avoided")
	}
	want := shankRank.EffectN(2).Average() - float64(rogue.Shank.Cost.BaseCost)
	if got := rogue.CurrentEnergy() - before; math.Abs(got-want) > 1e-9 {
		t.Errorf("Shank changed Energy by %g, want %g", got, want)
	}
	if rogue.ComboPoints() != int32(bootKnifePoints.ProcEnergizeEffect().Average()) {
		t.Fatal("Shank did not grant Boot Knife points")
	}
	if rogue.Shank.ProcMask != core.ProcMaskEmpty || !rogue.Shank.Flags.Matches(core.SpellFlagSuppressWeaponProcs) {
		t.Fatal("Shank permits attack procs")
	}
	t.Logf("netEnergy=%g BootKnifePoints=%d attackProcsSuppressed=true", rogue.CurrentEnergy()-before, rogue.ComboPoints())
}

func TestMutilateHitsUseTheCapturedWeaponAndFlatDamage(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "00530221250551031--0052330003", nil)
	for _, weapon := range []*core.Weapon{rogue.AutoAttacks.MH(), rogue.AutoAttacks.OH()} {
		weapon.BaseDamageMax = weapon.BaseDamageMin
	}
	target := rogue.CurrentTarget
	expected := []float64{}
	for i, hit := range []*core.Spell{rogue.MutilateMH, rogue.MutilateOH} {
		row := mutilateCastRank.EffectN(i + 2).Trigger()
		weapon := rogue.MHNormalizedWeaponDamage(sim, hit.MeleeAttackPower(target))
		if i == 1 {
			weapon = rogue.OHNormalizedWeaponDamage(sim, hit.MeleeAttackPower(target))
		}
		expected = append(expected, hit.CalcDamage(sim, target, row.EffectN(1).BaseValue()+weapon, hit.OutcomeAlwaysHit).Damage)
		if hit.DamageMultiplier != row.EffectN(2).Percent() {
			t.Fatal("Mutilate's weapon multiplier differs from its row")
		}
	}
	if !rogue.Mutilate.Cast(sim, target) {
		t.Fatal("Mutilate did not cast")
	}
	for i, hit := range []*core.Spell{rogue.MutilateMH, rogue.MutilateOH} {
		if got := hit.SpellMetrics[target.UnitIndex].TotalDamage; math.Abs(got-expected[i]) > 1e-9 {
			t.Errorf("Mutilate hand %d dealt %g, want %g", i, got, expected[i])
		}
	}
	t.Logf("MutilateCastRank=%d flatDamage=%g weaponShare=%g mh=%g oh=%g", rogue.Mutilate.Rank, mutilateCastRank.EffectN(2).Trigger().EffectN(1).BaseValue(), rogue.MutilateMH.DamageMultiplier, expected[0], expected[1])
}

func TestSavedMutilateRankFiltersStillCast(t *testing.T) {
	for _, rank := range []int32{3, 4} {
		t.Run(fmt.Sprint(rank), func(t *testing.T) {
			id := &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: 1241584}, Rank: rank}
			rotation := &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{{Action: &proto.APLAction{
				Condition: &proto.APLValue{Value: &proto.APLValue_SpellIsKnown{SpellIsKnown: &proto.APLValueSpellIsKnown{SpellId: id}}},
				Action:    &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{SpellId: id}},
			}}}}
			player := core.WithSpec(&proto.Player{Class: proto.Class_ClassRogue, Race: proto.Race_RaceHuman, Equipment: core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1_muti").GearSet, TalentsString: "00530221250551031--0052330003", TalentLayout: proto.TalentLayout_TalentLayoutHotfix, Rotation: rotation}, DefaultOptions)
			sim := core.NewSim(&proto.RaidSimRequest{Raid: core.SinglePlayerRaidProto(player, nil, nil, nil), Encounter: core.MakeSingleTargetEncounter(0), SimOptions: &proto.SimOptions{RandomSeed: 1}}, simsignals.CreateSignals())
			sim.Reset()
			rogue := sim.Raid.Parties[0].Players[0].(*Rogue)
			rogue.Rotation.DoNextAction(sim)
			if casts := rogue.Mutilate.SpellMetrics[rogue.CurrentTarget.UnitIndex].Casts; casts != 1 {
				t.Errorf("saved rank %d cast %d times, want one", rank, casts)
			}
		})
	}
}
