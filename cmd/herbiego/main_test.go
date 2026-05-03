package main

import (
	"context"
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/player/human"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/llm"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestBuildPlayersCreatesMixedHumanAndAIPlayers(t *testing.T) {
	runtime := app.Runtime{
		Config: app.Config{
			HumanPlayers: 1,
			Roles: map[domain.RoleID]app.RoleConfig{
				domain.RoleProductionManager:  {Kind: app.PlayerKindHuman, Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/v1/", APISDKType: app.APISDKTypeOpenAI},
				domain.RoleProcurementManager: {Kind: app.PlayerKindAI, Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/v1/", APISDKType: app.APISDKTypeOpenAI},
				domain.RoleSalesManager:       {Kind: app.PlayerKindAI, Provider: "ollama-cloud", Model: "gemma4:e4b", URL: "https://ollama.com/api/v1/", APISDKType: app.APISDKTypeOpenAI, APIKey: "test-ollama-cloud-key"},
				domain.RoleFinanceController:  {Kind: app.PlayerKindAI, Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/v1/", APISDKType: app.APISDKTypeOpenAI},
			},
		},
		Scenario: scenario.Starter(),
		InitialMatch: domain.MatchState{
			Roles: []domain.RoleAssignment{
				{RoleID: domain.RoleProductionManager, IsHuman: true},
				{RoleID: domain.RoleProcurementManager, IsHuman: false, Provider: "ollama-localhost", ModelName: "gemma4:e4b"},
				{RoleID: domain.RoleSalesManager, IsHuman: false, Provider: "ollama-cloud", ModelName: "gemma4:e4b"},
				{RoleID: domain.RoleFinanceController, IsHuman: false, Provider: "ollama-localhost", ModelName: "gemma4:e4b"},
			},
		},
	}

	players, _, err := buildPlayersWithHumanSubmit(runtime.Config, runtime.Scenario, runtime.InitialMatch, func(context.Context, ports.RoundRequest) (domain.ActionSubmission, error) {
		return domain.ActionSubmission{}, nil
	}, runtime.Logger)
	if err != nil {
		t.Fatalf("buildPlayersWithHumanSubmit() error = %v, want nil", err)
	}
	if _, ok := players[domain.RoleProductionManager].(*human.Player); !ok {
		t.Fatalf("production player type = %T, want *human.Player", players[domain.RoleProductionManager])
	}
	if _, ok := players[domain.RoleProcurementManager].(*llm.Player); !ok {
		t.Fatalf("procurement player type = %T, want *llm.Player", players[domain.RoleProcurementManager])
	}
}

func TestBuildPlayersRejectsUnsupportedAIProvider(t *testing.T) {
	runtime := app.Runtime{
		Config: app.Config{
			HumanPlayers: 0,
			Roles: map[domain.RoleID]app.RoleConfig{
				domain.RoleProductionManager: {Kind: app.PlayerKindAI, Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: app.APISDKTypeOpenAI},
			},
		},
		Scenario: scenario.Starter(),
		InitialMatch: domain.MatchState{
			Roles: []domain.RoleAssignment{
				{RoleID: domain.RoleProductionManager, IsHuman: false, Provider: "openrouter", ModelName: "openai/gpt-5-mini"},
			},
		},
	}

	players, _, err := buildPlayersWithHumanSubmit(runtime.Config, runtime.Scenario, runtime.InitialMatch, func(context.Context, ports.RoundRequest) (domain.ActionSubmission, error) {
		return domain.ActionSubmission{}, nil
	}, runtime.Logger)
	if err != nil {
		t.Fatalf("buildPlayersWithHumanSubmit() error = %v, want nil", err)
	}
	if _, ok := players[domain.RoleProductionManager].(*llm.Player); !ok {
		t.Fatalf("production player type = %T, want *llm.Player", players[domain.RoleProductionManager])
	}
}

func TestResolveScenarioForMatchUsesMatchScenarioID(t *testing.T) {
	definition := scenario.NewDefinition(
		"cmd-runtime-test-scenario",
		"Command Runtime Test Scenario",
		"Ensures match execution resolves the scenario from the match record.",
		scenario.MatchSetup{
			ID:          "single-role",
			DisplayName: "Single Role",
			RoleRoster:  []domain.RoleID{domain.RoleSalesManager},
		},
		scenario.StartingConditions{
			ID:          "start",
			DisplayName: "Start",
			StartingTargets: domain.BudgetTargets{
				EffectiveRound:        1,
				RevenueTarget:         20,
				ProcurementBudget:     1,
				ProductionSpendBudget: 1,
				CashFloorTarget:       1,
				DebtCeilingTarget:     1,
			},
		},
		scenario.MarketModel{
			ID:                "market",
			DisplayName:       "Market",
			DemandAssumptions: scenario.DemandAssumptions{BacklogExpiryRounds: 1},
		},
		scenario.ProductionModel{
			ID:          "production",
			DisplayName: "Production",
		},
		scenario.FinanceModel{
			ID:                    "finance",
			DisplayName:           "Finance",
			ReceivableDelayRounds: 1,
			PayableDelayRounds:    1,
			PayrollDelayRounds:    1,
		},
	)
	if err := scenario.Register(definition); err != nil && err.Error() != `scenario "cmd-runtime-test-scenario" already registered` {
		t.Fatalf("scenario.Register() error = %v", err)
	}

	resolved, err := resolveScenarioForMatch(domain.MatchState{
		MatchID:    "match-1",
		ScenarioID: definition.ID,
	})
	if err != nil {
		t.Fatalf("resolveScenarioForMatch() error = %v", err)
	}
	if got := resolved.ID; got != definition.ID {
		t.Fatalf("resolved.ID = %q, want %q", got, definition.ID)
	}
}

func TestResolveScenarioForMatchRejectsUnknownScenario(t *testing.T) {
	_, err := resolveScenarioForMatch(domain.MatchState{
		MatchID:    "match-404",
		ScenarioID: "missing-scenario",
	})
	if err == nil {
		t.Fatal("resolveScenarioForMatch() error = nil, want unknown-scenario validation")
	}
	if !strings.Contains(err.Error(), `resolve scenario "missing-scenario"`) {
		t.Fatalf("resolveScenarioForMatch() error = %v, want missing-scenario context", err)
	}
}
