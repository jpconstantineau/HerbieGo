package main

import (
	"fmt"
	"os"

	"github.com/jpconstantineau/herbiego/internal/app"
)

func main() {
	runtime, err := app.BootstrapFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(
		os.Stdout,
		"HerbieGo runtime initialized (env=%s, seed=%d)\nroles: %v\n",
		runtime.Config.Environment,
		runtime.Config.Random.Seed,
		runtime.RoleSummaries(),
	)
}
