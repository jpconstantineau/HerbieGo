package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/adapters/ai"
	"github.com/jpconstantineau/herbiego/internal/adapters/ai/ollama"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/human"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/llm"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
	"github.com/jpconstantineau/herbiego/internal/ports"
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
	controller := newTerminalController(runtime.Scenario, os.Stdin, os.Stdout)
	players, err := buildPlayers(runtime, controller)
	if err != nil {
		fmt.Fprintf(os.Stderr, "startup rejected:\n%v\n", err)
		os.Exit(1)
	}

	collector := app.RoundCollector{
		Players: players,
	}

	printRuntimeSummary(runtime)

	actions, err := collector.Collect(context.Background(), runtime.InitialMatch, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "round collection failed:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Collected submissions:")
	for _, action := range actions {
		fmt.Fprintf(os.Stdout, "%s\n", strings.Join(summarizeCollectedSubmission(action), "\n"))
	}

	resolver := engine.NewResolver(runtime.Scenario.ResolverOptions())
	result, err := resolver.ResolveRound(runtime.InitialMatch, actions, runtime.Random)
	if err != nil {
		fmt.Fprintf(os.Stderr, "round resolution failed:\n%v\n", err)
		os.Exit(1)
	}

	printRoundResolution(result)
}

func buildPlayers(runtime app.Runtime, controller *terminalController) (map[domain.RoleID]ports.Player, error) {
	ollamaClient, err := ollama.New()
	if err != nil {
		return nil, fmt.Errorf("configure ollama adapter: %w", err)
	}
	decisionClient := ai.NewRoutingClient(map[string]ports.DecisionClient{
		string(app.ProviderOllama): ollamaClient,
	})
	orchestrator := app.NewAIOrchestrator(runtime.Scenario, decisionClient)

	players := make(map[domain.RoleID]ports.Player, len(runtime.InitialMatch.Roles))
	for _, assignment := range runtime.InitialMatch.Roles {
		if assignment.IsHuman {
			players[assignment.RoleID] = human.New(controller.submitRound)
			continue
		}
		if !decisionClient.SupportsProvider(assignment.Provider) {
			return nil, fmt.Errorf("role %q uses unsupported AI provider %q", assignment.RoleID, assignment.Provider)
		}
		players[assignment.RoleID] = llm.New(orchestrator.SubmitRound)
	}
	return players, nil
}

func printRuntimeSummary(runtime app.Runtime) {
	fmt.Fprintf(
		os.Stdout,
		"HerbieGo runtime initialized (env=%s, human_players=%d, seed=%d)\nscenario: %s (%s)\nmatch: %s round=%d cash=%d debt=%d backlog=%d\nroles: %v\n",
		runtime.Config.Environment,
		runtime.Config.HumanPlayers,
		runtime.Config.Random.Seed,
		runtime.Scenario.ID,
		runtime.Scenario.DisplayName,
		runtime.InitialMatch.MatchID,
		runtime.InitialMatch.CurrentRound,
		runtime.InitialMatch.Plant.Cash,
		runtime.InitialMatch.Plant.Debt,
		len(runtime.InitialMatch.Plant.Backlog),
		runtime.RoleSummaries(),
	)
}

func summarizeCollectedSubmission(action domain.ActionSubmission) []string {
	lines := []string{
		fmt.Sprintf("- %s", displayRoleName(action.RoleID)),
	}
	for _, line := range summarizeAction(action.Action) {
		lines = append(lines, fmt.Sprintf("  %s", line))
	}
	lines = append(lines, fmt.Sprintf("  Commentary: %s", action.Commentary.Body))
	return lines
}

func printRoundResolution(result engine.Result) {
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Round %d resolved.\n", result.Round.Round)
	fmt.Fprintf(
		os.Stdout,
		"Next round %d | Cash %d | Debt %d | Backlog %d | Revenue %d | Profit %d\n",
		result.NextState.CurrentRound,
		result.NextState.Plant.Cash,
		result.NextState.Plant.Debt,
		len(result.NextState.Plant.Backlog),
		result.Round.Metrics.ThroughputRevenue,
		result.Round.Metrics.RoundProfit,
	)

	fmt.Fprintln(os.Stdout, "Commentary:")
	for _, note := range result.Round.Commentary {
		fmt.Fprintf(os.Stdout, "- %s: %s\n", displayRoleName(note.RoleID), note.Body)
	}

	fmt.Fprintln(os.Stdout, "Events:")
	for _, event := range result.Round.Events {
		fmt.Fprintf(os.Stdout, "- %s\n", event.Summary)
	}
}
