package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/adapters/tui"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
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

	if err := runLiveGameplay(context.Background(), runtime, initialState, store, *rounds, *persistAIDebug); err != nil {
		fmt.Fprintf(os.Stderr, "match failed:\n%v\n", err)
		os.Exit(1)
	}
}

func runLiveGameplay(ctx context.Context, runtime app.Runtime, initialState domain.MatchState, store persistentStore, rounds int, persistAIDebug bool) error {
	controller := newLiveGameplayController(initialState)
	players, debugLog, err := buildPlayersWithHumanSubmit(runtime, initialState, controller.SubmitRound)
	if err != nil {
		return fmt.Errorf("player setup: %w", err)
	}
	if store != nil && persistAIDebug {
		configureAICallPersistence(debugLog, runtime.Logger, store, initialState.MatchID)
	}

	runner := app.MatchRunner{
		Collector: app.RoundCollector{Players: players, Logger: runtime.Logger},
		Resolver:  engine.NewResolver(runtime.Scenario.ResolverOptions()),
		Random:    runtime.Random,
		Store:     store,
		OnState:   controller.Publish,
		Logger:    runtime.Logger,
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	program := tui.NewProgram(
		runtime.Scenario,
		controller,
		controller.Submit,
		debugLog,
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	)

	runnerErr := make(chan error, 1)
	go func() {
		defer controller.Close()

		final, _, err := runner.Play(ctx, initialState, rounds)
		switch {
		case err == nil:
			program.Send(tui.StatusMsg{
				Text: fmt.Sprintf(
					"Match complete after round %d. Final cash %d, debt %d, backlog %d, profit %d. Inspect results and press q to exit.",
					final.CurrentRound-1,
					final.Plant.Cash,
					final.Plant.Debt,
					len(final.Plant.Backlog),
					final.Metrics.RoundProfit,
				),
			})
		case errors.Is(err, context.Canceled):
			program.Send(tui.StatusMsg{Text: "Match cancelled. Press q to exit."})
		default:
			program.Send(tui.StatusMsg{Text: fmt.Sprintf("Match failed: %v. Press q to exit.", err)})
		}

		runnerErr <- err
	}()

	_, err = program.Run()
	cancel()
	playErr := <-runnerErr

	if err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return fmt.Errorf("bubble tea runtime: %w", err)
	}
	if playErr != nil && !errors.Is(playErr, context.Canceled) {
		return playErr
	}
	return nil
}
