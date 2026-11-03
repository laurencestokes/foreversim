package rogue

import (
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
		TalentsString: talents, TalentLayout: proto.TalentLayout_TalentLayoutLaunch,
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
	if !rogue.PoisedAura.IsActive() {
		t.Fatal("Poised is absent at maximum Energy")
	}
	fullCost := rogue.SinisterStrike.Cost.GetCurrentCost() + 20
	before := rogue.CurrentEnergy()
	if !rogue.SinisterStrike.Cast(sim, rogue.CurrentTarget) {
		t.Fatal("the opener did not cast")
	}
	spent := before - rogue.CurrentEnergy()
	if spent != fullCost-20 || rogue.PoisedAura.IsActive() {
		t.Fatalf("opener spent %g Energy with Poised active=%t", spent, rogue.PoisedAura.IsActive())
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
		{"Boot Knife", "005302212-32-005230311321021151", 25, 10 * time.Second, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sim, rogue := newLaunchRogue(t, tc.talents, nil)
			rogue.PoisedAura.Deactivate(sim)
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
			if !rogue.Shank.ProcMask.Matches(core.ProcMaskMeleeOHSpecial) {
				t.Fatal("Shank does not use the off-hand proc mask")
			}
			if tc.name == "Flash of Steel" && rogue.Shank.DamageMultiplier != 0.75 {
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
		if item.GetAction().GetCastSpell().GetSpellId().GetOtherId() == proto.OtherAction_OtherActionShank {
			shank = item
			break
		}
	}
	if shank == nil {
		t.Fatal("the Rogue preset has no Shank action")
	}
	const bootKnife = "005302212-32-005230311321021151"
	for _, tc := range []struct {
		name, talents string
		points        int32
		energy        float64
		wantCast      bool
	}{
		{"no Boot Knife at five points", DefaultTalents, 5, 60, true},
		{"Boot Knife at three points", bootKnife, 3, 60, true},
		{"Boot Knife at four points", bootKnife, 4, 60, false},
		{"one rank of Boot Knife at four points", "005302212-32-005230311321021111", 4, 60, false},
		{"no Boot Knife above the Energy limit", DefaultTalents, 5, 61, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			player := core.WithSpec(&proto.Player{
				Class: proto.Class_ClassRogue, Race: proto.Race_RaceHuman,
				Equipment:     core.GetGearSet("../../ui/specs/rogue/dps/gear_sets", "p1_muti").GearSet,
				TalentsString: tc.talents, TalentLayout: proto.TalentLayout_TalentLayoutLaunch,
				Rotation: &proto.APLRotation{Type: proto.APLRotation_TypeAPL, PriorityList: []*proto.APLListItem{shank}},
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
	sim, rogue := newLaunchRogue(t, "00530221250551031--0052303003", &proto.ConsumesSpec{MhWeaponImbueId: 8928, OhWeaponImbueId: 8985})
	rogue.PoisedAura.Deactivate(sim)
	rogue.AddComboPoints(sim, 3, rogue.NewComboPointMetrics(core.ActionID{}))
	rogue.SpendEnergy(sim, 70, rogue.NewEnergyMetrics(core.ActionID{}))
	before := rogue.ModifiedProcChance(rogue.InstantPoison.ClassFlags, 0.2)
	if !rogue.Shank.Cast(sim, rogue.CurrentTarget) || !rogue.VenomAura.IsActive() {
		t.Fatal("Shank did not activate Venom")
	}
	after := rogue.ModifiedProcChance(rogue.InstantPoison.ClassFlags, 0.2)
	if math.Abs(after-before-0.2) > 1e-9 || rogue.ComboPoints() != 3 || rogue.VenomAura.Duration != 6*time.Second {
		t.Errorf("Venom changed chance by %g, leaves %d points and lasts %s", after-before, rogue.ComboPoints(), rogue.VenomAura.Duration)
	}
	if rogue.GetSpell(core.ActionID{SpellID: 1310703}) != nil {
		t.Error("Venom is still a castable finisher")
	}
	t.Logf("poisonChanceBefore=%g after=%g VenomDuration=%s comboPoints=%d", before, after, rogue.VenomAura.Duration, rogue.ComboPoints())
}

func TestBloodSplatterExtendsBothBleeds(t *testing.T) {
	ticks := func(talents string) (int32, int32) {
		sim, rogue := newLaunchRogue(t, talents, nil)
		rogue.StealthAura.Activate(sim)
		if !rogue.Garrote.Cast(sim, rogue.CurrentTarget) {
			t.Fatal("Garrote did not cast from Stealth")
		}
		garroteTicks := rogue.Garrote.Dot(rogue.CurrentTarget).BaseTickCount
		sim.CurrentTime += rogue.GCD.TimeToReady(sim)
		rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
		rogue.AddComboPoints(sim, 5, rogue.NewComboPointMetrics(core.ActionID{}))
		if !rogue.Rupture.Cast(sim, rogue.CurrentTarget) {
			t.Fatal("Rupture did not cast with combo points")
		}
		return garroteTicks, rogue.Rupture.Dot(rogue.CurrentTarget).BaseTickCount
	}
	g0, r0 := ticks("")
	g1, r1 := ticks("005302212")
	if g1-g0 != 3 || r1-r0 != 3 {
		t.Errorf("extra bleed ticks: Garrote=%d Rupture=%d", g1-g0, r1-r0)
	}
	t.Logf("GarroteTicks=%d→%d RuptureTicks=%d→%d", g0, g1, r0, r1)
}

func TestGhostlyStrikeOpensOneStealthAbility(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "005302212-32-005230311321021151", nil)
	rogue.PoisedAura.Deactivate(sim)
	if rogue.Ambush.CanCast(sim, rogue.CurrentTarget) || rogue.Garrote.CanCast(sim, rogue.CurrentTarget) {
		t.Fatal("a stealth opener is available without Stealth or Ghostly Strike")
	}
	if !rogue.GhostlyStrike.Cast(sim, rogue.CurrentTarget) {
		t.Fatal("Ghostly Strike did not cast")
	}
	sim.CurrentTime += rogue.GCD.TimeToReady(sim)
	rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
	if !rogue.Garrote.Cast(sim, rogue.CurrentTarget) || rogue.GhostlyStrikeAura.IsActive() {
		t.Fatal("Ghostly Strike did not grant and spend the opener window")
	}
	sim.CurrentTime += rogue.GCD.TimeToReady(sim)
	rogue.AddEnergy(sim, rogue.MaximumEnergy(), rogue.NewEnergyMetrics(core.ActionID{}))
	if rogue.Ambush.CanCast(sim, rogue.CurrentTarget) {
		t.Error("the spent Ghostly Strike window permits another opener")
	}
	t.Log("Ghostly Strike granted Garrote outside Stealth and the cast consumed the window")
}

func TestDeathMarkStoresBleedsTwiceAndDoesNotStoreItself(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "005302212-32-005230311321021151", nil)
	rogue.PoisedAura.Deactivate(sim)
	target := rogue.CurrentTarget
	rogue.AddComboPoints(sim, 5, rogue.NewComboPointMetrics(core.ActionID{}))
	if !rogue.DeathMark.Cast(sim, target) || rogue.DeathMark.ComboPointMetrics().ActualGainForCurrentIteration() != -5 {
		t.Fatalf("Death Mark did not cast and spend its combo points: remaining=%d", rogue.ComboPoints())
	}
	// Deal one ordinary hit and one bleed tick through the class spells' normal result dispatch.
	direct := rogue.SinisterStrike.CalcAndDealDamage(sim, target, 20, rogue.SinisterStrike.OutcomeAlwaysHit).Damage
	bleed := rogue.Rupture.CalcAndDealPeriodicDamage(sim, target, 20, rogue.Rupture.Dot(target).OutcomeTick).Damage
	coup := 1.2
	expected := (direct + 2*bleed) * 0.6 * coup * coup
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
	cap := (156 + 245*5 + 0.04*rogue.DeathMark.MeleeAttackPower(target)*5) * coup
	before := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage
	dot.TickOnce(sim)
	if got := rogue.DeathMark.SpellMetrics[target.UnitIndex].TotalDamage - before; math.Abs(got-cap) > 1e-6 {
		t.Errorf("Death Mark dealt %g at its cap, want %g", got, cap)
	}
	t.Logf("ordinaryHit=%g bleedTick=%g markDamage=%g repeatedDamage=%g cap=%g", direct, bleed, first, second, cap)
}

func TestHemorrhageAmplifiesBothCasterBleeds(t *testing.T) {
	sim, rogue := newLaunchRogue(t, "005302212-32-005230311321021151", nil)
	rogue.PoisedAura.Deactivate(sim)
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
	sim, rogue := newLaunchRogue(t, "00530221250551031--0052303003", nil)
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
			Talents: DefaultTalents,
			OtherTalentSets: []core.TalentsCombo{
				{
					Label:   "Subtlety BS",
					Talents: "005302212-32-005230311321021151",
				},
				{
					Label:   "Asn Muti",
					Talents: "00530221250551031--0052303003",
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
