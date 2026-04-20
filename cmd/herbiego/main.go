package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jpconstantineau/herbiego/internal/app"
)

func main() {
	var (
		configPath   = flag.String("config", "herbiego.yaml", "path to YAML runtime configuration")
		humanPlayers = flag.Int("human-players", -1, "override the number of human-controlled roles; use 0 for an AI-only test run")
		seed         = flag.Uint64("seed", 0, "override the deterministic runtime seed")
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

	fmt.Fprintf(
		os.Stdout,
		"HerbieGo runtime initialized (env=%s, human_players=%d, seed=%d)\nroles: %v\n",
		runtime.Config.Environment,
		runtime.Config.HumanPlayers,
		runtime.Config.Random.Seed,
		runtime.RoleSummaries(),
	)
}
