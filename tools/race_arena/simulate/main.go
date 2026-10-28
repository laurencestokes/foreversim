// Runs each exported preset with every legal race and writes the lists consumed by tools/race_arena.
package main

import (
	"crypto/sha256"
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
	"time"

	"github.com/wowsims/forever/sim"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
	gp "google.golang.org/protobuf/proto"
)

type build struct {
	ConsumableNames                                 []consumableName
	ID, Name, Status                                string
	DevGear                                         bool
	Caveats                                         []string
	Spec, Class, Build, Gear, Rotation, Consumables string
	Player                                          json.RawMessage
	Setup                                           json.RawMessage
}

type row struct {
	RaceValue      string  `json:"raceValue,omitempty"`
	Iterations     int     `json:"iterations,omitempty"`
	ElapsedSeconds float64 `json:"elapsedSeconds,omitempty"`
	Race           string  `json:"race"`
	Dps            float64 `json:"dps"`
	Error          float64 `json:"error"`
	Commit         string  `json:"commit,omitempty"`
}

type list struct {
	ConsumableNames  []consumableName `json:"consumableNames"`
	ElapsedSeconds   float64          `json:"elapsedSeconds,omitempty"`
	BuildID          string           `json:"buildId,omitempty"`
	InputFingerprint string           `json:"inputFingerprint,omitempty"`
	Request          json.RawMessage  `json:"request,omitempty"`
	ComputedAt       string           `json:"computedAt,omitempty"`
	Spec             string           `json:"spec"`
	Class            string           `json:"class"`
	Build            string           `json:"build"`
	Gear             string           `json:"gear"`
	Rotation         string           `json:"rotation"`
	Consumables      string           `json:"consumables"`
	Talents          string           `json:"talents"`
	Target           string           `json:"target"`
	Seconds          int              `json:"seconds"`
	Iterations       int              `json:"iterations"`
	Races            []row            `json:"races"`
	Setup            json.RawMessage  `json:"setup"`
}

func main() {
	input := flag.String("input", "race-arena-input.json", "exported preset players")
	out := flag.String("out", "race-arena-out", "output directory")
	iterations := flag.Int("iterations", 50000, "iterations per race")
	workers := flag.Int("workers", runtime.NumCPU(), "concurrent simulations")
	durations := flag.String("durations", "90,180,360", "fight lengths in seconds, comma separated")
	onlyRace := flag.String("race", "", "only simulate this race, for example Troll")
	sourceCommit := flag.String("commit", "", "simulator revision, required for a filtered run")
	manifestPath := flag.String("manifest", "", "write resolved requests and legal races without running simulations")
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
	manifest := manifest{SchemaVersion: 1, Commit: *sourceCommit, Iterations: *iterations, Builds: []manifestBuild{}}
	for _, b := range builds {
		player := &proto.Player{}
		check(protojson.Unmarshal(b.Player, player))
		request := makeRequest(player, 180, proto.MobType_MobTypeMechanical, *iterations)
		requestJSON, err := protojson.Marshal(request)
		check(err)
		entry := manifestBuild{ID: b.ID, InputFingerprint: fingerprint(requestJSON), Spec: b.Spec, Class: b.Class, Name: b.Build, Gear: b.Gear, Rotation: b.Rotation, Status: b.Status, DevGear: b.DevGear, Caveats: b.Caveats, ConsumableNames: b.ConsumableNames, Setup: b.Setup, Request: requestJSON}
		if entry.Caveats == nil {
			entry.Caveats = []string{}
		}
		for _, race := range core.ClassRaceCapabilities[player.Class] {
			entry.LegalRaces = append(entry.LegalRaces, legalRace{Name: raceName(race), Value: race.String()})
		}
		manifest.Builds = append(manifest.Builds, entry)
	}
	if *manifestPath != "" {
		if *sourceCommit == "" {
			panic("a manifest requires -commit")
		}
		for _, entry := range manifest.Builds {
			if entry.ID == "" {
				panic("missing stable build ID: run tools/race_arena/snapshot.mjs")
			}
		}
		data, err := json.MarshalIndent(manifest, "", "  ")
		check(err)
		check(os.WriteFile(*manifestPath, append(data, '\n'), 0644))
		return
	}
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
	for buildIndex, b := range builds {
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
					Consumables: b.Consumables, Talents: player.TalentsString, Target: strings.TrimPrefix(target.String(), "MobType"), Seconds: seconds, Iterations: *iterations, Setup: b.Setup,
					BuildID: b.ID, InputFingerprint: manifest.Builds[buildIndex].InputFingerprint, Request: manifest.Builds[buildIndex].Request, ConsumableNames: b.ConsumableNames}
				for _, race := range core.ClassRaceCapabilities[player.Class] {
					if *onlyRace != "" && !strings.EqualFold(raceName(race), *onlyRace) {
						continue
					}
					p := gp.Clone(player).(*proto.Player)
					p.Race = race
					p.Buffs = gp.Clone(core.FullIndividualBuffs).(*proto.IndividualBuffs)
					l.Races = append(l.Races, row{Race: raceName(race), RaceValue: race.String(), Iterations: *iterations, Commit: *sourceCommit})
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
				started := time.Now()
				result := core.RunRaidSim(makeRequest(j.player, j.seconds, j.target, *iterations))
				if result.Error != nil {
					panic(fmt.Sprintf("%s / %s: %s", lists[j.list].Build, j.player.Race, result.Error.Message))
				}
				metrics := result.RaidMetrics.Parties[0].Players[0].Dps
				if metrics.Avg <= 0 || math.IsNaN(metrics.Avg) || math.IsInf(metrics.Avg, 0) {
					panic("invalid damage: " + lists[j.list].Build)
				}
				r := &lists[j.list].Races[j.row]
				r.Dps, r.Error = metrics.Avg, metrics.Stdev/math.Sqrt(float64(*iterations))
				r.ElapsedSeconds = time.Since(started).Seconds()
				fmt.Printf("%s / %s / %s / %ds / %s: %.1f DPS\n", lists[j.list].Spec, lists[j.list].Build, lists[j.list].Target, j.seconds, r.Race, r.Dps)
			}
		}()
	}
	for _, j := range jobs {
		queue <- j
	}
	close(queue)
	wg.Wait()
	computedAt := time.Now().UTC().Format(time.RFC3339)
	for i := range lists {
		lists[i].ComputedAt = computedAt
		for _, row := range lists[i].Races {
			lists[i].ElapsedSeconds += row.ElapsedSeconds
		}
	}
	check(os.MkdirAll(*out, 0755))
	data, err = json.MarshalIndent(lists, "", "  ")
	check(err)
	check(os.WriteFile(filepath.Join(*out, "presets.json"), append(data, '\n'), 0644))
	fmt.Printf("Wrote %d lists from %d builds and %d simulations\n", len(lists), len(builds), len(jobs))
}

type legalRace struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type consumableName struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type manifestBuild struct {
	ConsumableNames  []consumableName `json:"consumableNames"`
	ID               string           `json:"id"`
	InputFingerprint string           `json:"inputFingerprint"`
	Spec             string           `json:"spec"`
	Class            string           `json:"class"`
	Name             string           `json:"name"`
	Gear             string           `json:"gear"`
	Rotation         string           `json:"rotation"`
	Status           string           `json:"status"`
	DevGear          bool             `json:"devGear"`
	Caveats          []string         `json:"caveats"`
	Setup            json.RawMessage  `json:"setup"`
	Request          json.RawMessage  `json:"request"`
	LegalRaces       []legalRace      `json:"legalRaces"`
}

type manifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	Commit        string          `json:"commit"`
	Iterations    int             `json:"iterations"`
	Builds        []manifestBuild `json:"builds"`
}

func makeRequest(player *proto.Player, seconds int, mobType proto.MobType, iterations int) *proto.RaidSimRequest {
	player = gp.Clone(player).(*proto.Player)
	player.Buffs = gp.Clone(core.FullIndividualBuffs).(*proto.IndividualBuffs)
	target := core.FreshDefaultTargetConfig()
	target.MobType = mobType
	encounter := core.MakeSingleTargetEncounter(0)
	encounter.Targets = []*proto.Target{target}
	encounter.Duration = float64(seconds)
	return &proto.RaidSimRequest{
		Raid:      &proto.Raid{Parties: []*proto.Party{{Players: []*proto.Player{player}, Buffs: gp.Clone(core.FullPartyBuffs).(*proto.PartyBuffs)}}, Buffs: gp.Clone(core.FullRaidBuffs).(*proto.RaidBuffs), Debuffs: gp.Clone(core.FullDebuffs).(*proto.Debuffs)},
		Encounter: encounter, SimOptions: &proto.SimOptions{Iterations: int32(iterations), RandomSeed: 101},
	}
}

func fingerprint(requestJSON []byte) string {
	var request map[string]any
	check(json.Unmarshal(requestJSON, &request))
	raid := request["raid"].(map[string]any)
	for _, party := range raid["parties"].([]any) {
		for _, player := range party.(map[string]any)["players"].([]any) {
			delete(player.(map[string]any), "name")
		}
	}
	for _, target := range request["encounter"].(map[string]any)["targets"].([]any) {
		delete(target.(map[string]any), "name")
	}
	trimStatVectors(request)
	data, err := json.Marshal(request)
	check(err)
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

func trimStatVectors(value any) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if key == "stats" || key == "pseudoStats" {
				if vector, ok := child.([]any); ok {
					numeric := true
					for _, entry := range vector {
						if _, ok := entry.(float64); !ok {
							numeric = false
							break
						}
					}
					if !numeric {
						trimStatVectors(child)
						continue
					}
					for len(vector) > 0 && vector[len(vector)-1] == float64(0) {
						vector = vector[:len(vector)-1]
					}
					value[key] = vector
					continue
				}
			}
			trimStatVectors(child)
		}
	case []any:
		for _, child := range value {
			trimStatVectors(child)
		}
	}
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
