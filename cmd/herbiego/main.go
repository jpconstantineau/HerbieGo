package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func main() {
	var (
		configPath     = flag.String("config", "herbiego.yaml", "path to YAML runtime configuration")
		humanPlayers   = flag.Int("human-players", -1, "override the number of human-controlled roles; use 0 for an AI-only test run")
		rounds         = flag.Int("rounds", 3, "number of rounds to play before exiting")
		seed           = flag.Uint64("seed", 0, "override the deterministic runtime seed")
		sqlitePath     = flag.String("sqlite-db", "", "optional SQLite database path for match persistence")
		resumeMatchID  = flag.String("resume-match-id", "", "existing persisted match id to resume from the SQLite store")
		persistAIDebug = flag.Bool("persist-ai-debug", false, "persist AI prompt/response history in SQLite when enabled")
	)
	flag.Parse()

	options := app.BootstrapOptions{
		ConfigPath: *configPath,
	}
	if *humanPlayers >= 0 {
		options.HumanPlayersOverride = humanPlayers
	}
	if *seed != 0 {
		options.SeedOverride = seed
	}

	runtime, err := app.Bootstrap(options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed:\n%v\n", err)
		os.Exit(1)
	}

	store, initialState, err := resolveMatchPersistence(runtime, *sqlitePath, *resumeMatchID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "persistence setup failed:\n%v\n", err)
		os.Exit(1)
	}
	if store != nil {
		defer store.Close()
	}

	hasResumedState := strings.TrimSpace(*resumeMatchID) != ""
	if err := runApplication(context.Background(), runtime, initialState, hasResumedState, store, *rounds, *persistAIDebug); err != nil {
		fmt.Fprintf(os.Stderr, "application failed:\n%v\n", err)
		os.Exit(1)
	}
}

func resolveScenarioForMatch(state domain.MatchState) (scenario.Definition, error) {
	definition, ok := scenario.Lookup(state.ScenarioID)
	if !ok {
		return scenario.Definition{}, fmt.Errorf("resolve scenario %q for match %q: scenario is not registered", state.ScenarioID, state.MatchID)
	}
	return definition, nil
}

type combinedDebugSource struct {
	live      *app.DebugLog
	persisted []ports.AICallRecord
}

func (s combinedDebugSource) Records() []ports.AICallRecord {
	liveRecords := s.live.Records()
	if len(s.persisted) == 0 {
		return liveRecords
	}

	records := make([]ports.AICallRecord, 0, len(s.persisted)+len(liveRecords))
	records = append(records, s.persisted...)
	records = append(records, liveRecords...)
	return records
}
