// The race tier lists: every race a class can be, on the community builds of each DPS spec,
// with everything but the race held still.
//
// A race is worth a few percent at most, less than a different gear set or rotation moves, so
// the only way to rank races is to change nothing else. Each list is one of the spec's community
// builds - the talent presets named with their point split, the ones the leaderboard and the
// rankings page run - on the page's default gear set with the rotation the page plays it with,
// in the leaderboard's own environment (consumables.go) against its one target, on one random
// seed. Only the race changes.
//
// Weapon racials are what make "identical gear" need a rule. A Human's Sword Specialization is
// worth nothing to a warrior carrying axes, so ranking races on the gear as shipped would rank
// the gear set author's taste in weapons. So each race runs twice: on the gear as shipped, and
// with its racial's weapon type laid over the same weapons by core's weapon type override, which
// relabels the type and leaves the stats, the procs and the item itself alone. The better run
// counts, and the list says which. The relabel only goes where the class could hold that weapon
// (relabelWeapons), and not at all for a build whose rotation needs the weapons it ships with.
//
// Opt in, like the leaderboard's search: without RACE_ARENA_OUT every TestRaceArena skips, so an
// ordinary `go test ./...` spends nothing here and its results cannot move. tools/race_arena
// merges what the specs write into ui/app/race_arena/results.json and scores the tiers there.
package arenalib

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	googleProto "google.golang.org/protobuf/proto"

	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
)

// Where each spec writes its lists: one file per spec, because `go test` runs packages side by
// side and a shared file would be a race.
const raceOutDirEnv = "RACE_ARENA_OUT"

// Races on one list sit tenths of a percent apart, and the narrowest tier is half a percent wide.
// At 20,000 iterations the worst race's mean was good to 0.06% (one standard error), which puts
// two standard errors of a difference at 0.17% - a third of a tier, enough to move a race sitting
// near an edge. 50,000 brings that to about 0.11%, and every list on all three targets still runs
// in about a quarter of an hour on a 16 thread machine (6 minutes at 20,000, measured 2026-09-24).
const raceIterations = int32(50000)

// Overrides raceIterations, for a quick run that proves the plumbing rather than ranks anything.
const raceIterationsEnv = "RACE_ARENA_ITERATIONS"

// The leaderboard's neutral target first, then the two creature types a racial pays extra
// against: Elementals for the Skyborne's Elemental Insight, Beasts for the Dwarf's Big Game
// Hunter and the Troll's Beast Slaying. The page leads with the neutral target; the other two
// are there for anyone whose raid is fighting one.
var raceTargets = []proto.MobType{proto.MobType_MobTypeMechanical, proto.MobType_MobTypeElemental, proto.MobType_MobTypeBeast}

// The weapon skill racials, as sim/core/racials.go applies them: crit while a weapon of the type
// is in either hand.
var weaponRacials = map[proto.Race]proto.WeaponType{
	proto.Race_RaceHuman: proto.WeaponType_WeaponTypeSword,
	proto.Race_RaceOrc:   proto.WeaponType_WeaponTypeAxe,
	proto.Race_RaceDwarf: proto.WeaponType_WeaponTypeMace,
}

// How the race arena plays one community build, where that is not the page's default gear set
// with the spec's one rotation. See Spec.RaceBuilds.
type RaceBuild struct {
	// File names under the spec's ui/specs folder, without their extensions.
	Gear     string
	Rotation string
	// The spec options the page's own build preset brings (a warlock's demon), when they are not
	// Spec.SpecOptions.
	SpecOptions interface{}
	// The rotation needs the weapons' own type - Backstab, Ambush and Mutilate need a dagger - so
	// no race's weapon type is laid over them.
	KeepWeaponType bool
	// The type the list's weapons are run as for every race, before any race tries its own: for a
	// build whose talents pay very differently by weapon type (a warrior's Weaponmaster) on a set
	// that ships a type the build would not choose. Without it the races with a weapon racial
	// would be the only ones allowed off the shipped type, and would be credited with the talent.
	WeaponType proto.WeaponType
}

// One tier list as a spec writes it. tools/race_arena's rawList mirrors this.
type RaceList struct {
	Spec        string `json:"spec"`
	Class       string `json:"class"`
	Build       string `json:"build"`
	Talents     string `json:"talents"`
	Gear        string `json:"gear"`
	Rotation    string `json:"rotation"`
	Consumables string `json:"consumables"`
	// The target's creature type. Mechanical is the leaderboard's neutral target.
	Target     string `json:"target"`
	Iterations int32  `json:"iterations"`
	// The build kept its weapons' type for every race (RaceBuild.KeepWeaponType).
	KeptWeaponType bool `json:"keptWeaponType,omitempty"`
	// The type every race's weapons were run as, when not the one they ship as (RaceBuild.WeaponType).
	WeaponType string    `json:"weaponType,omitempty"`
	Races      []RaceRow `json:"races"`
}

type RaceRow struct {
	// "Skyborne" for both halves, which share every racial a rotation uses.
	Race string  `json:"race"`
	Dps  float64 `json:"dps"`
	// The standard error of that mean.
	Error float64 `json:"error"`
	// The racial's weapon type the weapons were relabelled to, when that run beat the gear as
	// shipped. Empty when the shipped gear counted.
	Relabelled string `json:"relabelled,omitempty"`
	// Whether the race's weapon racial was active in the run that counted.
	WeaponRacial bool `json:"weaponRacial,omitempty"`
	// Both runs, when there were two, so the choice between them can be checked.
	Shipped       float64 `json:"shipped"`
	RelabelledDps float64 `json:"relabelledDps,omitempty"`
}

// A community build resolved into what a run needs.
type raceBuild struct {
	name, talents, gear, rotation string
	equipment                     *proto.EquipmentSpec
	rotationProto                 *proto.APLRotation
	specOptions                   interface{}
	keepWeaponType                bool
	weaponType                    proto.WeaponType
	// The same gear with a weapon-racial race's type laid over it, for each such race where that
	// is allowed and changes anything.
	relabelled map[proto.Race]*proto.EquipmentSpec
}

// RunRaceArena ranks every race the spec's class can be on each of its community builds, and
// writes the lists to RACE_ARENA_OUT. Without it the test skips.
func RunRaceArena(t *testing.T, spec Spec) {
	outDir := os.Getenv(raceOutDirEnv)
	if outDir == "" {
		t.Skip("set RACE_ARENA_OUT to a directory to run the race tier lists")
	}
	iterations := raceIterations
	if n, err := strconv.Atoi(os.Getenv(raceIterationsEnv)); err == nil && n > 0 {
		iterations = int32(n)
	}

	builds := raceBuilds(t, spec)
	races := raceCandidates(spec.Class)
	environment := consumesFor(spec.Role, spec.ClassImbues)

	// Every run is independent - one per build, target, race and weapon choice - so they all go
	// side by side, the way the leaderboard runs its builds (parallel.go).
	type job struct {
		build, target, race int
		gear                *proto.EquipmentSpec
		relabelled          proto.WeaponType
	}
	jobs := []job{}
	for b, build := range builds {
		for m := range raceTargets {
			for r, race := range races {
				jobs = append(jobs, job{b, m, r, build.equipment, proto.WeaponType_WeaponTypeUnknown})
				if gear := build.relabelled[race]; gear != nil {
					jobs = append(jobs, job{b, m, r, gear, weaponRacials[race]})
				}
			}
		}
	}

	started := time.Now()
	type outcome struct {
		dps, stdev float64
		err        string
	}
	outcomes := parallelMap(len(jobs), func(i int) outcome {
		j := jobs[i]
		dps, stdev, err := runRace(spec, environment, builds[j.build], races[j.race], j.gear, raceTargets[j.target], iterations)
		return outcome{dps, stdev, err}
	})

	lists := make([]RaceList, 0, len(builds)*len(raceTargets))
	listOf := map[[2]int]int{}
	rowOf := map[[3]int]int{}
	for i, j := range jobs {
		o := outcomes[i]
		build, race := builds[j.build], races[j.race]
		if o.err != "" {
			// Every race has to be on a list for the list to mean anything, so one failed run fails
			// the spec rather than publishing a list with a race missing from it.
			t.Errorf("%s / %s / %s / %s: %s", spec.Dir, build.name, raceName(race), raceTargets[j.target], o.err)
			continue
		}
		at, ok := listOf[[2]int{j.build, j.target}]
		if !ok {
			at = len(lists)
			listOf[[2]int{j.build, j.target}] = at
			lists = append(lists, RaceList{
				Spec:           spec.Dir,
				Class:          strings.TrimPrefix(spec.Class.String(), "Class"),
				Build:          build.name,
				Talents:        build.talents,
				Gear:           build.gear,
				Rotation:       build.rotation,
				Consumables:    environment.Label,
				Target:         strings.TrimPrefix(raceTargets[j.target].String(), "MobType"),
				Iterations:     iterations,
				KeptWeaponType: build.keepWeaponType,
			})
			if build.weaponType != proto.WeaponType_WeaponTypeUnknown {
				lists[at].WeaponType = strings.TrimPrefix(build.weaponType.String(), "WeaponType")
			}
		}
		list := &lists[at]
		key := [3]int{j.build, j.target, j.race}
		r, ok := rowOf[key]
		if !ok {
			r = len(list.Races)
			rowOf[key] = r
			list.Races = append(list.Races, RaceRow{Race: raceName(race), Dps: -1})
		}
		row := &list.Races[r]
		se := o.stdev / math.Sqrt(float64(iterations))
		if j.relabelled == proto.WeaponType_WeaponTypeUnknown {
			row.Shipped = o.dps
		} else {
			row.RelabelledDps = o.dps
		}
		// The better of the two runs counts. The shipped run comes first in the jobs, so a tie
		// stays with the gear as shipped, which needs no explaining.
		if o.dps > row.Dps {
			row.Dps, row.Error = o.dps, se
			row.Relabelled = ""
			row.WeaponRacial = carries(j.gear, weaponRacials[race])
			if j.relabelled != proto.WeaponType_WeaponTypeUnknown {
				row.Relabelled = strings.TrimPrefix(j.relabelled.String(), "WeaponType")
			}
		}
	}
	if t.Failed() {
		t.FailNow()
	}

	for i := range lists {
		sort.SliceStable(lists[i].Races, func(a, b int) bool { return lists[i].Races[a].Dps > lists[i].Races[b].Dps })
		best, worst := lists[i].Races[0], lists[i].Races[len(lists[i].Races)-1]
		t.Logf("%s / %s / %s: best %s %.1f, worst %s %.1f (%+.2f%%)", spec.Dir, lists[i].Build, lists[i].Target,
			best.Race, best.Dps, worst.Race, worst.Dps, (worst.Dps/best.Dps-1)*100)
	}
	write(t, filepath.Join(outDir, spec.Dir+".json"), lists)
	t.Logf("%s: %d lists, %d runs at %d iterations in %s", spec.Dir, len(lists), len(jobs), iterations, time.Since(started).Round(time.Second))
}

// Resolves the spec's community builds into what a run needs: the talents from the page's
// presets, the gear set and rotation from Spec.RaceBuilds or the page's defaults, and the
// relabelled gear each weapon-racial race may try.
func raceBuilds(t *testing.T, spec Spec) []raceBuild {
	uiDir := specDir(spec)
	// The leaderboard's own reading of the page, without the builds its search found: a searched
	// build is a result about talents, not a build anybody plays.
	community := communityTalents(t, uiDir, spec.Talents)
	if len(community) == 0 {
		t.Fatalf("%s: no community builds in %s", spec.Dir, uiDir)
	}
	for name := range spec.RaceBuilds {
		if !slices.ContainsFunc(community, func(build TalentBuild) bool { return build.Name == name }) {
			t.Fatalf("%s: RaceBuilds names %q, which is not one of the page's community builds", spec.Dir, name)
		}
	}

	gearSets := filter(namesIn(filepath.Join(uiDir, "gear_sets"), ".gear.json"), spec.GearSets)
	rotations := filter(namesIn(filepath.Join(uiDir, "apls"), ".apl.json"), spec.Rotations)
	defaultGear := pageDefaultGear(t, uiDir)

	builds := []raceBuild{}
	for _, talent := range community {
		own := spec.RaceBuilds[talent.Name]
		build := raceBuild{
			name:           talent.Name,
			talents:        talent.Talents,
			gear:           own.Gear,
			rotation:       own.Rotation,
			specOptions:    own.SpecOptions,
			keepWeaponType: own.KeepWeaponType,
			weaponType:     own.WeaponType,
			relabelled:     map[proto.Race]*proto.EquipmentSpec{},
		}
		if build.gear == "" {
			if !slices.Contains(gearSets, defaultGear) {
				t.Fatalf("%s: the page's default gear set %q is not one of this spec's %v; name one in RaceBuilds for %q", spec.Dir, defaultGear, gearSets, talent.Name)
			}
			build.gear = defaultGear
		}
		if build.rotation == "" {
			// The page picks among several in code (its autoRotation), which the arena cannot read.
			if len(rotations) != 1 {
				t.Fatalf("%s: %d rotations on file and none named for %q; say in RaceBuilds which one the page plays it with", spec.Dir, len(rotations), talent.Name)
			}
			build.rotation = rotations[0]
		}
		if build.specOptions == nil {
			build.specOptions = spec.SpecOptions
		}

		gearPath := filepath.Join(uiDir, "gear_sets", build.gear+".gear.json")
		rotationPath := filepath.Join(uiDir, "apls", build.rotation+".apl.json")
		for _, path := range []string{gearPath, rotationPath} {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("%s: %q: %s", spec.Dir, talent.Name, err)
			}
		}
		build.equipment = core.GetGearSet(filepath.Join(uiDir, "gear_sets"), build.gear).GearSet
		build.rotationProto = core.GetAplRotation(filepath.Join(uiDir, "apls"), build.rotation).Rotation

		if build.weaponType != proto.WeaponType_WeaponTypeUnknown {
			if build.keepWeaponType {
				t.Fatalf("%s: %q both keeps its weapons' type and runs them as %s", spec.Dir, talent.Name, build.weaponType)
			}
			relabelled := relabelWeapons(build.equipment, spec.Class, build.weaponType)
			if relabelled == nil {
				t.Fatalf("%s: %q cannot run %s's weapons as %s", spec.Dir, talent.Name, build.gear, build.weaponType)
			}
			build.equipment = relabelled
		}
		if !build.keepWeaponType {
			for race, weaponType := range weaponRacials {
				if gear := relabelWeapons(build.equipment, spec.Class, weaponType); gear != nil {
					build.relabelled[race] = gear
				}
			}
		}
		builds = append(builds, build)
	}
	return builds
}

// The gear set the spec's page opens with: DEFAULT_GEAR in its presets.ts, followed through the
// preset it names to the file that preset imports. Read the same way the leaderboard reads the
// community builds, and for the same reason - so the list and the page cannot disagree.
var defaultGearRef = regexp.MustCompile(`export const DEFAULT_GEAR\s*=\s*(\w+)\s*;`)

func pageDefaultGear(t *testing.T, uiDir string) string {
	source, err := os.ReadFile(filepath.Join(uiDir, "presets.ts"))
	if err != nil {
		t.Fatalf("no presets to read the default gear from: %s", err)
	}
	preset := defaultGearRef.FindSubmatch(source)
	if preset == nil {
		t.Fatalf("%s: no DEFAULT_GEAR", uiDir)
	}
	gear := regexp.MustCompile(`export const ` + string(preset[1]) + `\s*=\s*PresetUtils\.makePresetGear\(\s*'[^']*'\s*,\s*(\w+)\s*\)`).FindSubmatch(source)
	if gear == nil {
		t.Fatalf("%s: DEFAULT_GEAR is %s, which is not a gear preset", uiDir, preset[1])
	}
	file := regexp.MustCompile(`import ` + string(gear[1]) + ` from '\./gear_sets/([^']+)\.gear\.json'`).FindSubmatch(source)
	if file == nil {
		t.Fatalf("%s: %s is not imported from gear_sets", uiDir, gear[1])
	}
	return string(file[1])
}

// The races a class can be, with the two Skyborne halves as one: they share every passive racial
// and their base attributes, and their on-use racials (Read Ley Line, Skysight) are in no
// rotation, so the second would only repeat the first. The half that runs is
// the one the class can be (a mage only High Order, a shaman only Windshaper).
func raceCandidates(class proto.Class) []proto.Race {
	races := []proto.Race{}
	skyborne := false
	for _, race := range core.ClassRaceCapabilities[class] {
		if race == proto.Race_RaceHighOrderSkyborne || race == proto.Race_RaceWindshaperSkyborne {
			if skyborne {
				continue
			}
			skyborne = true
		}
		races = append(races, race)
	}
	return races
}

func raceName(race proto.Race) string {
	switch race {
	case proto.Race_RaceNightElf:
		return "Night Elf"
	case proto.Race_RaceHighOrderSkyborne, proto.Race_RaceWindshaperSkyborne:
		return "Skyborne"
	}
	return strings.TrimPrefix(race.String(), "Race")
}

// The gear with each main- and off-hand weapon relabelled to weaponType, through the weapon type
// override (stats, procs and item id unchanged), or nil when that changes nothing.
//
// Only where the class could hold the weapon as that type: a two-hander goes only to a type the
// class wields two-handed, and a one-hander never to a polearm or a staff. Held off-hands,
// shields and ranged weapons are not weapons a racial reads and are left alone. A class that
// cannot use the type at all - a mage and an axe - gets nothing, which is the point: that race's
// weapon racial is worth nothing to it in the game either.
func relabelWeapons(gear *proto.EquipmentSpec, class proto.Class, weaponType proto.WeaponType) *proto.EquipmentSpec {
	wields, ok := core.ClassWeaponTypeCapabilities[class][weaponType]
	if !ok {
		return nil
	}
	out := googleProto.Clone(gear).(*proto.EquipmentSpec)
	changed := false
	for _, spec := range out.Items {
		item := core.GetItemByID(spec.Id)
		if item == nil || !heldWeapon(item) || effectiveType(spec, item) == weaponType {
			continue
		}
		if item.HandType == proto.HandType_HandTypeTwoHand && !wields.CanUseTwoHand {
			continue
		}
		if item.HandType != proto.HandType_HandTypeTwoHand &&
			(weaponType == proto.WeaponType_WeaponTypePolearm || weaponType == proto.WeaponType_WeaponTypeStaff) {
			continue
		}
		spec.WeaponTypeOverride = weaponType
		changed = true
	}
	if !changed {
		return nil
	}
	return out
}

// Whether a main- or off-hand weapon in the gear counts as weaponType.
func carries(gear *proto.EquipmentSpec, weaponType proto.WeaponType) bool {
	if weaponType == proto.WeaponType_WeaponTypeUnknown {
		return false
	}
	for _, spec := range gear.Items {
		item := core.GetItemByID(spec.Id)
		if item != nil && heldWeapon(item) && effectiveType(spec, item) == weaponType {
			return true
		}
	}
	return false
}

// A weapon swung from the main or off hand. A ranged weapon is a different item type altogether.
func heldWeapon(item *core.Item) bool {
	return item.Type == proto.ItemType_ItemTypeWeapon &&
		item.WeaponType != proto.WeaponType_WeaponTypeOffHand && item.WeaponType != proto.WeaponType_WeaponTypeShield
}

func effectiveType(spec *proto.ItemSpec, item *core.Item) proto.WeaponType {
	if spec.WeaponTypeOverride != proto.WeaponType_WeaponTypeUnknown {
		return spec.WeaponTypeOverride
	}
	return item.WeaponType
}

// One race on one build against one target: the mean DPS and its standard deviation.
func runRace(spec Spec, environment core.BuffsCombo, build raceBuild, race proto.Race, gear *proto.EquipmentSpec, mobType proto.MobType, iterations int32) (dps float64, stdev float64, failure string) {
	// A panic is a failed run, named in the log, rather than the end of every other spec's.
	defer func() {
		if r := recover(); r != nil {
			dps, stdev, failure = 0, 0, fmt.Sprintf("panic: %.200v", r)
		}
	}()

	encounter := core.MakeSingleTargetEncounter(0)
	if encounter.Targets[0].MobType != mobType {
		// A fresh target, not the shared default: that one is read by every other run.
		target := core.FreshDefaultTargetConfig()
		target.MobType = mobType
		encounter.Targets = []*proto.Target{target}
	}

	// Cloned per run: the build's gear and rotation are shared by every race's run of it.
	result := core.RunRaidSim(&proto.RaidSimRequest{
		Raid: arenaRaid(spec, environment, race, googleProto.Clone(gear).(*proto.EquipmentSpec), build.talents,
			googleProto.Clone(build.rotationProto).(*proto.APLRotation), build.specOptions),
		Encounter: encounter,
		SimOptions: &proto.SimOptions{
			Iterations: iterations,
			RandomSeed: arenaSeed,
		},
	})
	if result.Error != nil {
		return 0, 0, fmt.Sprintf("%.200s", result.Error.Message)
	}
	if result.RaidMetrics == nil || len(result.RaidMetrics.Parties) == 0 {
		return 0, 0, "no metrics"
	}
	// The player's numbers include its pets'.
	metrics := result.RaidMetrics.Parties[0].Players[0]
	if metrics.Dps.Avg <= 0 {
		return 0, 0, "no damage"
	}
	return metrics.Dps.Avg, metrics.Dps.Stdev, ""
}
