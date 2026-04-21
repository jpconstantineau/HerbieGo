package main

import (
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/player/human"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/llm"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestBuildPlayersCreatesMixedHumanAndAIPlayers(t *testing.T) {
	runtime := app.Runtime{
		Config: app.Config{
			HumanPlayers: 1,
		},
		Scenario: scenario.Starter(),
		InitialMatch: domain.MatchState{
			Roles: []domain.RoleAssignment{
				{RoleID: domain.RoleProductionManager, IsHuman: true},
				{RoleID: domain.RoleProcurementManager, IsHuman: false, Provider: "ollama", ModelName: "gemma4:e4b"},
				{RoleID: domain.RoleSalesManager, IsHuman: false, Provider: "ollama", ModelName: "gemma4:e4b"},
				{RoleID: domain.RoleFinanceController, IsHuman: false, Provider: "ollama", ModelName: "gemma4:e4b"},
			},
		},
	}

	players, err := buildPlayers(runtime, &terminalController{})
	if err != nil {
		t.Fatalf("buildPlayers() error = %v, want nil", err)
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
		},
		Scenario: scenario.Starter(),
		InitialMatch: domain.MatchState{
			Roles: []domain.RoleAssignment{
				{RoleID: domain.RoleProductionManager, IsHuman: false, Provider: "openrouter", ModelName: "openai/gpt-5-mini"},
			},
		},
	}

	_, err := buildPlayers(runtime, &terminalController{})
	if err == nil {
		t.Fatal("buildPlayers() error = nil, want unsupported-provider error")
	}
	if got, want := err.Error(), `role "production_manager" uses unsupported AI provider "openrouter"`; got != want {
		t.Fatalf("buildPlayers() error = %q, want %q", got, want)
	}
}
