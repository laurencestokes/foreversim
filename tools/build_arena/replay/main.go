package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"github.com/wowsims/forever/sim"
	"github.com/wowsims/forever/sim/core"
	"github.com/wowsims/forever/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	input := flag.String("input", "", "saved RaidSimRequest JSON")
	output := flag.String("out", "", "complete RaidSimResult JSON")
	flag.Parse()
	if *input == "" || *output == "" || flag.NArg() != 0 {
		return fmt.Errorf("usage: replay -input <RaidSimRequest.json> -out <RaidSimResult.json>")
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		return fmt.Errorf("read request: %w", err)
	}
	request := &proto.RaidSimRequest{}
	if err := protojson.Unmarshal(data, request); err != nil {
		return fmt.Errorf("parse request: %w", err)
	}
	if request.GetSimOptions().GetIterations() <= 0 {
		return fmt.Errorf("request iterations must be positive")
	}
	sim.RegisterAll()
	result := core.RunRaidSim(request)
	data, err = (protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: true}).Marshal(result)
	if err != nil {
		return fmt.Errorf("serialize result: %w", err)
	}
	if err := os.WriteFile(*output, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write result: %w", err)
	}
	if result.Error != nil {
		return fmt.Errorf("simulation failed: %s", result.Error.Message)
	}
	if result.IterationsDone <= 0 || result.GetRaidMetrics().GetDps() == nil {
		return fmt.Errorf("simulation returned no completed damage metrics")
	}
	dps := result.RaidMetrics.Dps
	standardError := dps.Stdev / math.Sqrt(float64(result.IterationsDone))
	fmt.Printf("%.9f DPS, %.9f standard error, %d iterations\n", dps.Avg, standardError, result.IterationsDone)
	return nil
}
