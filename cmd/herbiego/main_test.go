package main

import (
	"context"
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
				domain.RoleProductionManager:  {Kind: app.PlayerKindHuman, Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/", APISDKType: app.APISDKTypeOllama},
				domain.RoleProcurementManager: {Kind: app.PlayerKindAI, Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/", APISDKType: app.APISDKTypeOllama},
				domain.RoleSalesManager:       {Kind: app.PlayerKindAI, Provider: "ollama-cloud", Model: "gemma4:e4b", URL: "https://ollama.com/api/", APISDKType: app.APISDKTypeOllama},
				domain.RoleFinanceController:  {Kind: app.PlayerKindAI, Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/", APISDKType: app.APISDKTypeOllama},
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

	players, err := buildPlayersWithHumanSubmit(runtime, func(context.Context, ports.RoundRequest) (domain.ActionSubmission, error) {
		return domain.ActionSubmission{}, nil
	})
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

	players, err := buildPlayersWithHumanSubmit(runtime, func(context.Context, ports.RoundRequest) (domain.ActionSubmission, error) {
		return domain.ActionSubmission{}, nil
	})
	if err != nil {
		t.Fatalf("buildPlayersWithHumanSubmit() error = %v, want nil", err)
	}
	if _, ok := players[domain.RoleProductionManager].(*llm.Player); !ok {
		t.Fatalf("production player type = %T, want *llm.Player", players[domain.RoleProductionManager])
	}
}
