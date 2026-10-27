// Runs each exported preset with every legal race and writes the lists consumed by tools/race_arena.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/wowsims/forever/sim"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
	gp "google.golang.org/protobuf/proto"
)

type build struct {
	Spec, Class, Build, Gear, Rotation, Consumables string
	Player                                          json.RawMessage
	Setup                                           json.RawMessage
}

type row struct {
	Race   string  `json:"race"`
	Dps    float64 `json:"dps"`
	Error  float64 `json:"error"`
	Commit string  `json:"commit,omitempty"`
}

type list struct {
	Spec        string          `json:"spec"`
	Class       string          `json:"class"`
	Build       string          `json:"build"`
	Gear        string          `json:"gear"`
	Rotation    string          `json:"rotation"`
	Consumables string          `json:"consumables"`
	Talents     string          `json:"talents"`
	Target      string          `json:"target"`
	Seconds     int             `json:"seconds"`
	Iterations  int             `json:"iterations"`
	Races       []row           `json:"races"`
	Setup       json.RawMessage `json:"setup"`
}

func main() {
	input := flag.String("input", "race-arena-input.json", "exported preset players")
	out := flag.String("out", "race-arena-out", "output directory")
	iterations := flag.Int("iterations", 50000, "iterations per race")
	workers := flag.Int("workers", runtime.NumCPU(), "concurrent simulations")
	durations := flag.String("durations", "90,180,360", "fight lengths in seconds, comma separated")
	onlyRace := flag.String("race", "", "only simulate this race, for example Troll")
	sourceCommit := flag.String("commit", "", "simulator revision, required for a filtered run")
	flag.Parse()
	if *iterations < 1 || *workers < 1 {
		panic("iterations and workers must be positive")
	}
	if *onlyRace != "" && *sourceCommit == "" {
		panic("a filtered run requires -commit to record its simulator revision")
	}
	data, err := os.ReadFile(*input)
	check(err)
	var builds []build
	check(json.Unmarshal(data, &builds))
	if len(builds) == 0 {
		panic("no preset builds")
	}
	sim.RegisterAll()
	targets := []proto.MobType{proto.MobType_MobTypeMechanical, proto.MobType_MobTypeElemental, proto.MobType_MobTypeBeast}
	fightLengths := []int{}
	for _, field := range strings.Split(*durations, ",") {
		seconds, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil || seconds <= 0 {
			panic("durations must be positive whole seconds: " + *durations)
		}
		fightLengths = append(fightLengths, seconds)
	}
	lists := []list{}
	type job struct {
		list, row int
		player    *proto.Player
		target    proto.MobType
		seconds   int
	}
	jobs := []job{}
	for _, b := range builds {
		if len(b.Setup) == 0 {
			panic("missing setup snapshot: run tools/race_arena/snapshot.mjs before simulating")
		}
		player := &proto.Player{}
		check(protojson.Unmarshal(b.Player, player))
		if len(player.GetEquipment().GetItems()) == 0 || len(player.GetRotation().GetPriorityList()) == 0 {
			panic("incomplete build: " + b.Build)
		}
		for _, seconds := range fightLengths {
			for _, target := range targets {
				l := list{Spec: b.Spec, Class: b.Class, Build: b.Build, Gear: b.Gear, Rotation: b.Rotation,
					Consumables: b.Consumables, Talents: player.TalentsString, Target: strings.TrimPrefix(target.String(), "MobType"), Seconds: seconds, Iterations: *iterations, Setup: b.Setup}
				for _, race := range core.ClassRaceCapabilities[player.Class] {
					if *onlyRace != "" && !strings.EqualFold(raceName(race), *onlyRace) {
						continue
					}
					p := gp.Clone(player).(*proto.Player)
					p.Race = race
					p.Buffs = gp.Clone(core.FullIndividualBuffs).(*proto.IndividualBuffs)
					l.Races = append(l.Races, row{Race: raceName(race), Commit: *sourceCommit})
					jobs = append(jobs, job{len(lists), len(l.Races) - 1, p, target, seconds})
				}
				if len(l.Races) == 0 {
					if *onlyRace != "" {
						continue
					}
					panic("no legal races: " + b.Class)
				}
				lists = append(lists, l)
			}
		}
	}
	if len(jobs) == 0 {
		panic("no legal matchups for race: " + *onlyRace)
	}
	queue := make(chan job)
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range queue {
				target := core.FreshDefaultTargetConfig()
				target.MobType = j.target
				encounter := core.MakeSingleTargetEncounter(0)
				encounter.Targets = []*proto.Target{target}
				encounter.Duration = float64(j.seconds)
				result := core.RunRaidSim(&proto.RaidSimRequest{
					Raid: &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{j.player}, Buffs: gp.Clone(core.FullPartyBuffs).(*proto.PartyBuffs)}},
						Buffs: gp.Clone(core.FullRaidBuffs).(*proto.RaidBuffs), Debuffs: gp.Clone(core.FullDebuffs).(*proto.Debuffs)},
					Encounter:  encounter,
					SimOptions: &proto.SimOptions{Iterations: int32(*iterations), RandomSeed: 101},
				})
				if result.Error != nil {
					panic(fmt.Sprintf("%s / %s: %s", lists[j.list].Build, j.player.Race, result.Error.Message))
				}
				metrics := result.RaidMetrics.Parties[0].Players[0].Dps
				if metrics.Avg <= 0 || math.IsNaN(metrics.Avg) || math.IsInf(metrics.Avg, 0) {
					panic("invalid damage: " + lists[j.list].Build)
				}
				r := &lists[j.list].Races[j.row]
				r.Dps, r.Error = metrics.Avg, metrics.Stdev/math.Sqrt(float64(*iterations))
				fmt.Printf("%s / %s / %s / %ds / %s: %.1f DPS\n", lists[j.list].Spec, lists[j.list].Build, lists[j.list].Target, j.seconds, r.Race, r.Dps)
			}
		}()
	}
	for _, j := range jobs {
		queue <- j
	}
	close(queue)
	wg.Wait()
	check(os.MkdirAll(*out, 0755))
	data, err = json.MarshalIndent(lists, "", "  ")
	check(err)
	check(os.WriteFile(filepath.Join(*out, "presets.json"), append(data, '\n'), 0644))
	fmt.Printf("Wrote %d lists from %d builds and %d simulations\n", len(lists), len(builds), len(jobs))
}

func raceName(r proto.Race) string {
	switch r {
	case proto.Race_RaceNightElf:
		return "Night Elf"
	case proto.Race_RaceHighOrderSkyborne:
		return "High Order Skyborne"
	case proto.Race_RaceWindshaperSkyborne:
		return "Windshaper Skyborne"
	}
	return strings.TrimPrefix(r.String(), "Race")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
