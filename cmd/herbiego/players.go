package main

import (
	"fmt"

	"github.com/jpconstantineau/herbiego/internal/adapters/ai"
	"github.com/jpconstantineau/herbiego/internal/adapters/ai/ollama"
	"github.com/jpconstantineau/herbiego/internal/adapters/ai/openai"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/human"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/llm"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

func buildPlayersWithHumanSubmit(runtime app.Runtime, submit human.SubmitFunc) (map[domain.RoleID]ports.Player, error) {
	providers, err := buildDecisionClients(runtime.Config)
	if err != nil {
		return nil, err
	}
	decisionClient := ai.NewRoutingClient(providers)
	orchestrator := app.NewAIOrchestrator(runtime.Scenario, decisionClient)

	players := make(map[domain.RoleID]ports.Player, len(runtime.InitialMatch.Roles))
	for _, assignment := range runtime.InitialMatch.Roles {
		if assignment.IsHuman {
			players[assignment.RoleID] = human.New(submit)
			continue
		}
		if !decisionClient.SupportsProvider(assignment.Provider) {
			return nil, fmt.Errorf("role %q uses unsupported AI provider %q", assignment.RoleID, assignment.Provider)
		}
		players[assignment.RoleID] = llm.New(orchestrator.SubmitRound)
	}
	return players, nil
}

func buildDecisionClients(cfg app.Config) (map[string]ports.DecisionClient, error) {
	clients := make(map[string]ports.DecisionClient)
	for _, roleCfg := range cfg.Roles {
		providerName := string(roleCfg.Provider)
		if providerName == "" {
			continue
		}
		if _, ok := clients[providerName]; ok {
			continue
		}

		switch roleCfg.APISDKType {
		case app.APISDKTypeOllama:
			client, err := ollama.New(ollama.WithBaseURL(roleCfg.URL))
			if err != nil {
				return nil, fmt.Errorf("configure provider %q: %w", roleCfg.Provider, err)
			}
			clients[providerName] = client
		case app.APISDKTypeOpenAI:
			client, err := openai.New(
				openai.WithBaseURL(roleCfg.URL),
				openai.WithAPIKey(roleCfg.APIKey),
			)
			if err != nil {
				return nil, fmt.Errorf("configure provider %q: %w", roleCfg.Provider, err)
			}
			clients[providerName] = client
		default:
			return nil, fmt.Errorf("provider %q uses unsupported api_sdk_type %q", roleCfg.Provider, roleCfg.APISDKType)
		}
	}
	return clients, nil
}
