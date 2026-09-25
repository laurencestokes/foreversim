package core

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/forever/sim/core/proto"
)

// Helpers for the racial breakdown tests (sim/<class>/..._test.go), which ask where each race's DPS
// comes from by switching its racials on one piece at a time, on identical gear.

// RetypeWeapons returns a copy of the gear whose weapons are stat-identical copies of the originals,
// registered under fresh ids, with only the weapon type changed. Held off-hands and shields keep
// theirs. Anything keyed to the original item id (a scripted proc) is not carried over.
func RetypeWeapons(gear *proto.EquipmentSpec, weaponType proto.WeaponType) *proto.EquipmentSpec {
	out := &proto.EquipmentSpec{}
	for _, spec := range gear.Items {
		copied := &proto.ItemSpec{Id: spec.Id, Enchant: spec.Enchant, RandomSuffix: spec.RandomSuffix, Gems: spec.Gems}
		item := GetItemByID(spec.Id)
		if item != nil && item.Type == proto.ItemType_ItemTypeWeapon &&
			item.WeaponType != proto.WeaponType_WeaponTypeOffHand && item.WeaponType != proto.WeaponType_WeaponTypeShield {
			newID := 900000000 + int32(weaponType)*1000000 + spec.Id
			AddToDatabase(&proto.SimDatabase{Items: []*proto.SimItem{{
				Id:               newID,
				Name:             fmt.Sprintf("%s (%s)", item.Name, strings.TrimPrefix(weaponType.String(), "WeaponType")),
				Type:             item.Type,
				ArmorType:        item.ArmorType,
				WeaponType:       weaponType,
				HandType:         item.HandType,
				RangedWeaponType: item.RangedWeaponType,
				WeaponSpeed:      item.SwingSpeed,
				QualityModifier:  item.QualityModifier,
				GemSockets:       item.GemSockets,
				SocketBonus:      item.SocketBonus.ToProtoArray(),
				Unique:           item.Unique,
				LimitCategory:    item.LimitCategory,
				ScalingOptions:   item.ScalingOptions,
				ItemEffects:      item.ItemEffects,
			}}})
			copied.Id = newID
		}
		out.Items = append(out.Items, copied)
	}
	return out
}

// RotationWithoutSpell returns a copy of the rotation with one spell switched off. Any line that
// casts it goes, and a line that would cast it under a false condition takes it out of Autocast
// Other Cooldowns, the way a No Reck rotation drops Recklessness.
func RotationWithoutSpell(rotation *proto.APLRotation, spellID int32) *proto.APLRotation {
	out := googleProto.Clone(rotation).(*proto.APLRotation)
	kept := []*proto.APLListItem{{Action: &proto.APLAction{
		Condition: &proto.APLValue{Value: &proto.APLValue_Const{Const: &proto.APLValueConst{Val: "false"}}},
		Action: &proto.APLAction_CastSpell{CastSpell: &proto.APLActionCastSpell{
			SpellId: &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: spellID}},
		}},
	}}}
	for _, item := range out.PriorityList {
		if item.GetAction().GetCastSpell().GetSpellId().GetSpellId() != spellID {
			kept = append(kept, item)
		}
	}
	out.PriorityList = kept
	return out
}

type RacialBreakdownConfig struct {
	Title string
	Races []proto.Race
	// A race none of whose racials move this spec's DPS once its cooldown racial is off. Every
	// other race is measured against it.
	BaselineRace proto.Race
	// The gear, whose weapons are retyped for each weapon type the breakdown needs.
	Gear *proto.EquipmentSpec
	// Weapon types no racial rewards, in order of preference; a race's own racial type is skipped.
	NeutralWeapons  []proto.WeaponType
	WeaponRacials   map[proto.Race]proto.WeaponType
	CooldownRacials map[proto.Race]int32
	// A race with no weapon racial, run with its cooldown racial off on each weapon type a weapon
	// racial needs, to separate what the weapon type does on its own (through talents or abilities
	// that care about it) from the racial. Defaults to BaselineRace.
	TypeReferenceRace proto.Race
	Rotation          *proto.APLRotation
	Iterations        int32
	// Runs one configuration and returns the mean and standard deviation of its DPS.
	Run func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) (float64, float64)
}

// RacialBreakdown builds each race up from the baseline one piece at a time and returns the result
// as a markdown table:
//
//   - base stats: the baseline race plus the difference in attribute offsets, as bonus stats
//   - passive racials: the race itself on a neutral weapon with its cooldown racial off, less the
//     line above; zero for a race whose passives do nothing here, which checks the method
//   - cooldown racial: the same with the cooldown back on
//   - weapon type: what moving from the neutral type to the race's own weapon type does for a race
//     with no weapon racial (TypeReferenceRace); zero unless talents or abilities care about type
//   - weapon racial: the race's own gain from that move, less the weapon type column
//
// The columns sum to the total by construction.
func RacialBreakdown(config RacialBreakdownConfig) string {
	type measure struct{ dps, se float64 }
	diff := func(a, b measure) measure {
		return measure{a.dps - b.dps, math.Sqrt(a.se*a.se + b.se*b.se)}
	}

	neutralType := func(race proto.Race) proto.WeaponType {
		for _, weaponType := range config.NeutralWeapons {
			if own, ok := config.WeaponRacials[race]; !ok || own != weaponType {
				return weaponType
			}
		}
		panic("no neutral weapon type for " + race.String())
	}

	// When the only weapon type the breakdown needs is the one the gear already carries, the gear
	// runs as it is, scripted procs and all. Otherwise every run uses the retyped copies, so no race
	// keeps a proc another loses.
	needed := map[proto.WeaponType]bool{}
	for _, race := range config.Races {
		needed[neutralType(race)] = true
		if own, ok := config.WeaponRacials[race]; ok {
			needed[own] = true
		}
	}
	asShipped := len(needed) == 1
	for _, spec := range config.Gear.Items {
		item := GetItemByID(spec.Id)
		if item != nil && item.Type == proto.ItemType_ItemTypeWeapon && item.WeaponType != proto.WeaponType_WeaponTypeOffHand &&
			item.WeaponType != proto.WeaponType_WeaponTypeShield && !needed[item.WeaponType] {
			asShipped = false
		}
	}

	weapons := map[proto.WeaponType]*proto.EquipmentSpec{}
	weaponsOf := func(weaponType proto.WeaponType) *proto.EquipmentSpec {
		if asShipped {
			return config.Gear
		}
		if weapons[weaponType] == nil {
			weapons[weaponType] = RetypeWeapons(config.Gear, weaponType)
		}
		return weapons[weaponType]
	}
	neutral := func(race proto.Race) *proto.EquipmentSpec {
		return weaponsOf(neutralType(race))
	}
	rotation := func(race proto.Race, cooldownOn bool) *proto.APLRotation {
		if id, ok := config.CooldownRacials[race]; ok && !cooldownOn {
			return RotationWithoutSpell(config.Rotation, id)
		}
		return googleProto.Clone(config.Rotation).(*proto.APLRotation)
	}
	run := func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) measure {
		dps, stdev := config.Run(race, gear, rotation, bonus)
		return measure{dps, stdev / math.Sqrt(float64(config.Iterations))}
	}

	baseline := run(config.BaselineRace, neutral(config.BaselineRace), rotation(config.BaselineRace, false), nil)

	reference := config.TypeReferenceRace
	if reference == proto.Race_RaceUnknown {
		reference = config.BaselineRace
	}
	if _, ok := config.WeaponRacials[reference]; ok {
		panic("the weapon type reference race has a weapon racial: " + reference.String())
	}
	referenceOn := map[proto.WeaponType]measure{}
	referenceRun := func(weaponType proto.WeaponType) measure {
		if _, ok := referenceOn[weaponType]; !ok {
			referenceOn[weaponType] = run(reference, weaponsOf(weaponType), rotation(reference, false), nil)
		}
		return referenceOn[weaponType]
	}

	type row struct {
		name                                     string
		total, base, passive, cooldown, weapon   measure
		weaponType                               measure
		hasCooldown, hasWeapon                   bool
		weaponWithoutCooldown, cooldownOnOwnType measure
	}
	var rows []row
	for _, race := range config.Races {
		name := strings.TrimPrefix(race.String(), "Race")
		if race == proto.Race_RaceWindshaperSkyborne && slices.Contains(config.Races, proto.Race_RaceHighOrderSkyborne) {
			continue // Shares every racial with the High Order.
		}
		if race == proto.Race_RaceHighOrderSkyborne && slices.Contains(config.Races, proto.Race_RaceWindshaperSkyborne) {
			name = "Skyborne (either)"
		}

		statsOnly, passive := baseline, baseline
		if race != config.BaselineRace {
			offsets := RaceOffsets[race].Subtract(RaceOffsets[config.BaselineRace])
			statsOnly = run(config.BaselineRace, neutral(config.BaselineRace), rotation(config.BaselineRace, false), &proto.UnitStats{Stats: offsets.ToProtoArray()})
			passive = run(race, neutral(race), rotation(race, false), nil)
		}

		_, hasCooldown := config.CooldownRacials[race]
		withCooldown := passive
		if hasCooldown {
			withCooldown = run(race, neutral(race), rotation(race, true), nil)
		}

		ownType, hasWeapon := config.WeaponRacials[race]
		full := withCooldown
		if hasWeapon {
			full = run(race, weaponsOf(ownType), rotation(race, true), nil)
		}

		r := row{
			name:        name,
			total:       diff(full, baseline),
			base:        diff(statsOnly, baseline),
			passive:     diff(passive, statsOnly),
			cooldown:    diff(withCooldown, passive),
			weapon:      diff(full, withCooldown),
			hasCooldown: hasCooldown,
			hasWeapon:   hasWeapon,
		}
		if hasWeapon {
			r.weaponType = diff(referenceRun(ownType), referenceRun(neutralType(race)))
			r.weapon = diff(r.weapon, r.weaponType)
		}
		if hasCooldown && hasWeapon {
			// The two pieces the other way round, to show how much they interact.
			weaponOnly := run(race, weaponsOf(ownType), rotation(race, false), nil)
			r.weaponWithoutCooldown = diff(weaponOnly, passive)
			r.cooldownOnOwnType = diff(full, weaponOnly)
		}
		rows = append(rows, r)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].total.dps > rows[j].total.dps })

	show := func(m measure) string {
		if math.Abs(m.dps) < 2*m.se {
			return fmt.Sprintf("%+.1f (≈0)", m.dps)
		}
		return fmt.Sprintf("%+.1f (%+.2f%%)", m.dps, m.dps/baseline.dps*100)
	}
	showIf := func(present bool, m measure) string {
		if !present {
			return "-"
		}
		return show(m)
	}

	var report strings.Builder
	fmt.Fprintf(&report, "## %s\n\n", config.Title)
	fmt.Fprintf(&report, "Baseline: %s, %.1f ± %.1f DPS (%d iterations per run). Differences under about %.1f DPS are noise, shown as ≈0.\n\n",
		strings.TrimPrefix(config.BaselineRace.String(), "Race"), baseline.dps, baseline.se, config.Iterations, 2*math.Sqrt2*baseline.se)
	fmt.Fprintf(&report, "| Race | Total vs baseline | Base stats | Passive racials | Cooldown racial | Weapon type (no racial) | Weapon racial |\n|---|---|---|---|---|---|---|\n")
	for _, r := range rows {
		fmt.Fprintf(&report, "| %s | %s | %s | %s | %s | %s | %s |\n", r.name, show(r.total), show(r.base), show(r.passive),
			showIf(r.hasCooldown, r.cooldown), showIf(r.hasWeapon, r.weaponType), showIf(r.hasWeapon, r.weapon))
	}
	for _, r := range rows {
		if r.hasCooldown && r.hasWeapon {
			fmt.Fprintf(&report, "\n%s, the other order: weapon racial alone %s, cooldown racial on top of it %s.\n", r.name, show(r.weaponWithoutCooldown), show(r.cooldownOnOwnType))
		}
	}
	return report.String()
}

type EurekaSplitConfig struct {
	Label string
	// The class's Eureka! spell, to switch it off.
	SpellID    int32
	Gear       *proto.EquipmentSpec
	Rotation   *proto.APLRotation
	Iterations int32
	// Runs one configuration and returns the mean and standard deviation of its DPS.
	Run func(race proto.Race, gear *proto.EquipmentSpec, rotation *proto.APLRotation, bonus *proto.UnitStats) (float64, float64)
}

// The header for the rows EurekaSplit returns.
const EurekaSplitHeader = "| Build | Gnome, Eureka! off | Whole Eureka! | Cost cut alone | Damage bonus alone | Overlap |\n|---|---|---|---|---|---|\n"

// EurekaSplit measures a gnome's Eureka! whole and in its two halves - the cost cut alone and the
// 10% damage alone, charges spent the same way either way - against the same gnome with Eureka!
// off, and returns a markdown row. Overlap is whole less the two halves: what the halves are worth
// only together (a cheaper ability cast sooner also takes the damage bonus).
func EurekaSplit(config EurekaSplitConfig) string {
	defer func() { EurekaParts.Cost, EurekaParts.Damage = true, true }()
	run := func(cost, damage bool, rotation *proto.APLRotation) (float64, float64) {
		EurekaParts.Cost, EurekaParts.Damage = cost, damage
		dps, stdev := config.Run(proto.Race_RaceGnome, config.Gear, rotation, nil)
		return dps, stdev / math.Sqrt(float64(config.Iterations))
	}
	off, se := run(true, true, RotationWithoutSpell(config.Rotation, config.SpellID))
	whole, _ := run(true, true, googleProto.Clone(config.Rotation).(*proto.APLRotation))
	costOnly, _ := run(true, false, googleProto.Clone(config.Rotation).(*proto.APLRotation))
	damageOnly, _ := run(false, true, googleProto.Clone(config.Rotation).(*proto.APLRotation))

	show := func(delta float64) string {
		if math.Abs(delta) < 2*math.Sqrt2*se {
			return fmt.Sprintf("%+.1f (≈0)", delta)
		}
		return fmt.Sprintf("%+.1f (%+.2f%%)", delta, delta/off*100)
	}
	return fmt.Sprintf("| %s | %.1f ± %.1f | %s | %s | %s | %s |\n", config.Label, off, se,
		show(whole-off), show(costOnly-off), show(damageOnly-off), show(whole-costOnly-damageOnly+off))
}
