package app

import "testing"

func TestNewRuntimeSeedsDefaultStarterMatch(t *testing.T) {
	runtime, err := NewRuntime(Config{
		Environment:  "test",
		HumanPlayers: 1,
		Random: RandomConfig{
			Seed: 9,
		},
		LLMCatalog: LLMCatalog{
			Entries: []LLMCatalogEntry{
				{Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/", APISDKType: APISDKTypeOllama},
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
	if got := runtime.InitialMatch.MatchID; got != "starter-match" {
		t.Fatalf("InitialMatch.MatchID = %q, want starter-match", got)
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
}
