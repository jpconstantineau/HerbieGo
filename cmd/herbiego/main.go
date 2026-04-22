package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jpconstantineau/herbiego/internal/adapters/tui"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/engine"
)

func main() {
	var (
		configPath   = flag.String("config", "herbiego.yaml", "path to YAML runtime configuration")
		humanPlayers = flag.Int("human-players", -1, "override the number of human-controlled roles; use 0 for an AI-only test run")
		rounds       = flag.Int("rounds", 3, "number of rounds to play before exiting")
		seed         = flag.Uint64("seed", 0, "override the deterministic runtime seed")
		inspect      = flag.Bool("inspect", false, "open the TUI inspector for the initial match instead of playing rounds")
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

	if *inspect {
		if err := tui.Run(runtime.Scenario, runtime.InitialMatch); err != nil {
			fmt.Fprintf(os.Stderr, "tui failed:\n%v\n", err)
			os.Exit(1)
		}
		return
	}

	controller := newTerminalController(runtime.Scenario, os.Stdin, os.Stdout)
	players, err := buildPlayers(runtime, controller)
	if err != nil {
		fmt.Fprintf(os.Stderr, "player setup failed:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "HerbieGo MVP match: %s\n", runtime.Scenario.DisplayName)
	fmt.Fprintf(os.Stdout, "Roles: %v\n", runtime.RoleSummaries())
	fmt.Fprintf(os.Stdout, "Playing %d rounds. Use -human-players=0 for an unattended AI-only run.\n", *rounds)

	runner := app.MatchRunner{
		Collector: app.RoundCollector{Players: players},
		Resolver:  engine.NewResolver(runtime.Scenario.ResolverOptions()),
		Random:    runtime.Random,
		OnRound: func(result engine.Result) {
			renderRoundOutcome(os.Stdout, result)
		},
	}

	final, _, err := runner.Play(context.Background(), runtime.InitialMatch, *rounds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "match failed:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "\nMatch complete after round %d. Next round would be %d.\n", final.CurrentRound-1, final.CurrentRound)
	fmt.Fprintf(os.Stdout, "Final cash %d | debt %d | backlog %d | profit %d\n",
		final.Plant.Cash,
		final.Plant.Debt,
		len(final.Plant.Backlog),
		final.Metrics.RoundProfit,
	)
}
