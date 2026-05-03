package app

import (
	"testing"
	"time"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestNewRuntimeSeedsGeneratedStarterMatch(t *testing.T) {
	originalNow := runtimeTimeNow
	runtimeTimeNow = func() time.Time {
		return time.Date(2026, time.May, 3, 17, 30, 45, 123456789, time.UTC)
	}
	t.Cleanup(func() {
		runtimeTimeNow = originalNow
	})

	runtime, err := NewRuntime(Config{
		Environment:  "test",
		ScenarioID:   scenario.StarterID,
		HumanPlayers: 1,
		UI: UIConfig{
			AIRevealDelaySeconds: 12,
		},
		Random: RandomConfig{
			Seed: 9,
		},
		LLMCatalog: LLMCatalog{
			Entries: []LLMCatalogEntry{
				{Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/v1/", APISDKType: APISDKTypeOpenAI},
				{Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: APISDKTypeOpenAI},
			},
		},
		RoleConfigs: []RoleConfigFile{
			{RoleID: "procurement_manager", Provider: "ollama-localhost"},
			{RoleID: "production_manager", Provider: "ollama-localhost"},
			{RoleID: "sales_manager", Provider: "openrouter"},
			{RoleID: "finance_controller", Provider: "ollama-localhost"},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	if got := runtime.Scenario.ID; got != "starter" {
		t.Fatalf("Scenario.ID = %q, want starter", got)
	}
	if got := runtime.InitialMatch.MatchID; got != "starter-match-9-1777829445123456789" {
		t.Fatalf("InitialMatch.MatchID = %q, want generated deterministic id", got)
	}
	if got := runtime.InitialMatch.MatchID; got == "starter-match" {
		t.Fatalf("InitialMatch.MatchID = %q, want generated id instead of hardcoded literal", got)
	}
	if got := runtime.InitialMatch.CurrentRound; got != 1 {
		t.Fatalf("InitialMatch.CurrentRound = %d, want 1", got)
	}
	if got := len(runtime.InitialMatch.Roles); got != 4 {
		t.Fatalf("InitialMatch.Roles len = %d, want 4", got)
	}
	if !runtime.InitialMatch.Roles[1].IsHuman {
		t.Fatalf("production role IsHuman = false, want true")
	}
	if got := runtime.InitialMatch.Plant.Cash; got != 24 {
		t.Fatalf("InitialMatch.Plant.Cash = %d, want 24", got)
	}
	if got := len(runtime.InitialMatch.Plant.Backlog); got != 2 {
		t.Fatalf("InitialMatch.Plant.Backlog len = %d, want 2", got)
	}
	if got := runtime.InitialMatch.RoundFlow.Phase; got != "collecting" {
		t.Fatalf("InitialMatch.RoundFlow.Phase = %q, want collecting", got)
	}
	if got := runtime.InitialMatch.RoundFlow.AIRevealDelaySeconds; got != 12 {
		t.Fatalf("InitialMatch.RoundFlow.AIRevealDelaySeconds = %d, want 12", got)
	}
	if got := len(runtime.InitialMatch.RoundFlow.WaitingOnRoles); got != 4 {
		t.Fatalf("InitialMatch.RoundFlow.WaitingOnRoles len = %d, want 4", got)
	}
	if runtime.Logger == nil {
		t.Fatal("runtime.Logger = nil, want process logger")
	}
}

func TestNewRuntimeUsesConfiguredMatchIDWhenProvided(t *testing.T) {
	runtime, err := NewRuntime(Config{
		Environment:  "test",
		ScenarioID:   scenario.StarterID,
		MatchID:      "fixture-match-42",
		HumanPlayers: 1,
		UI: UIConfig{
			AIRevealDelaySeconds: 12,
		},
		Random: RandomConfig{
			Seed: 9,
		},
		LLMCatalog: LLMCatalog{
			Entries: []LLMCatalogEntry{
				{Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/v1/", APISDKType: APISDKTypeOpenAI},
				{Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: APISDKTypeOpenAI},
			},
		},
		RoleConfigs: []RoleConfigFile{
			{RoleID: "procurement_manager", Provider: "ollama-localhost"},
			{RoleID: "production_manager", Provider: "ollama-localhost"},
			{RoleID: "sales_manager", Provider: "openrouter"},
			{RoleID: "finance_controller", Provider: "ollama-localhost"},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v, want nil", err)
	}

	if got := runtime.InitialMatch.MatchID; got != "fixture-match-42" {
		t.Fatalf("InitialMatch.MatchID = %q, want configured match id", got)
	}
}

func TestNewRuntimeUsesRegisteredScenarioFromConfig(t *testing.T) {
	definition := scenario.NewDefinition(
		"runtime-test-scenario",
		"Runtime Test Scenario",
		"Ensures runtime selection comes from the scenario registry.",
		scenario.MatchSetup{
			ID:          "single-role",
			DisplayName: "Single Role",
			RoleRoster:  []domain.RoleID{domain.RoleSalesManager},
		},
		scenario.StartingConditions{
			ID:          "test-start",
			DisplayName: "Test Start",
			StartingTargets: domain.BudgetTargets{
				EffectiveRound:        1,
				RevenueTarget:         77,
				DebtCeilingTarget:     5,
				ProcurementBudget:     1,
				ProductionSpendBudget: 1,
				CashFloorTarget:       1,
			},
			StartingPlant: domain.PlantState{
				Cash:        99,
				DebtCeiling: 5,
			},
			Customers: []scenario.CustomerSeed{
				{ID: "runtime-test-customer", DisplayName: "Runtime Test Customer", Sentiment: 5, PaymentDelayRounds: 1},
			},
		},
		scenario.MarketModel{
			ID:          "test-market",
			DisplayName: "Test Market",
			Customers: []scenario.CustomerMarket{
				{
					ID:          "runtime-test-customer",
					DisplayName: "Runtime Test Customer",
					DemandByProduct: map[domain.ProductID]scenario.DemandProfile{
						"runtime-test-product": {ReferencePrice: 10, BaseDemand: 1, PriceSensitivity: 1},
					},
				},
			},
			DemandAssumptions: scenario.DemandAssumptions{BacklogExpiryRounds: 1},
		},
		scenario.ProductionModel{
			ID:          "test-production",
			DisplayName: "Test Production",
			Products: []scenario.Product{
				{ID: "runtime-test-product", DisplayName: "Runtime Test Product", BaseUnitCost: 4},
			},
		},
		scenario.FinanceModel{
			ID:                    "test-finance",
			DisplayName:           "Test Finance",
			ReceivableDelayRounds: 1,
			PayableDelayRounds:    1,
			PayrollDelayRounds:    1,
		},
	)
	if err := scenario.Register(definition); err != nil && err.Error() != `scenario "runtime-test-scenario" already registered` {
		t.Fatalf("scenario.Register() error = %v", err)
	}

	runtime, err := NewRuntime(Config{
		Environment:  "test",
		ScenarioID:   definition.ID,
		HumanPlayers: 0,
		UI: UIConfig{
			AIRevealDelaySeconds: 12,
		},
		Random: RandomConfig{
			Seed: 9,
		},
		LLMCatalog: LLMCatalog{
			Entries: []LLMCatalogEntry{
				{Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: APISDKTypeOpenAI},
			},
		},
		RoleConfigs: []RoleConfigFile{
			{RoleID: domain.RoleSalesManager, Provider: "openrouter"},
		},
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	if got := runtime.Scenario.ID; got != definition.ID {
		t.Fatalf("Scenario.ID = %q, want %q", got, definition.ID)
	}
	if got := runtime.InitialMatch.ScenarioID; got != definition.ID {
		t.Fatalf("InitialMatch.ScenarioID = %q, want %q", got, definition.ID)
	}
	if got := len(runtime.InitialMatch.Roles); got != 1 {
		t.Fatalf("InitialMatch.Roles len = %d, want 1", got)
	}
	if got := runtime.InitialMatch.Plant.Cash; got != 99 {
		t.Fatalf("InitialMatch.Plant.Cash = %d, want 99", got)
	}
}
