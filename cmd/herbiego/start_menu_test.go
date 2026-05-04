package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestSplashModelFinishesAfterLastFrame(t *testing.T) {
	model := splashModel{}

	for i := 0; i < len(herbieSplashFrames())-1; i++ {
		next, _ := model.Update(splashTickMsg{})
		model = next.(splashModel)
		if model.finished {
			t.Fatalf("frame %d finished early", i)
		}
	}

	next, _ := model.Update(splashTickMsg{})
	model = next.(splashModel)
	if !model.finished {
		t.Fatal("splash model did not finish on the last animation frame")
	}
}

func TestSplashFrameDelayDefaultsToThreeSecondsAcrossFrames(t *testing.T) {
	got := splashFrameDelayForFrameCount(len(herbieSplashFrames()))
	want := 500 * time.Millisecond
	if got != want {
		t.Fatalf("splashFrameDelayForFrameCount() = %v, want %v", got, want)
	}
}

func TestSplashFrameRendersExpectedCanvasHeight(t *testing.T) {
	frame := renderHerbieSplashFrame(2)
	lines := strings.Split(frame, "\n")
	if len(lines) != splashCanvasHeight {
		t.Fatalf("renderHerbieSplashFrame() line count = %d, want %d", len(lines), splashCanvasHeight)
	}
}

func TestSplashFramesKeepStableVisibleWidth(t *testing.T) {
	widths := splashFrameVisibleWidths()
	if len(widths) == 0 {
		t.Fatal("splashFrameVisibleWidths() returned no frame widths")
	}
	for _, width := range widths[1:] {
		if width != widths[0] {
			t.Fatalf("splash frame widths = %v, want all frames to share one visible width", widths)
		}
	}
}

func TestStartMenuCanToggleRoleAndLaunch(t *testing.T) {
	model := newStartMenuModel(testMenuConfig())

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = next.(startMenuModel)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = next.(startMenuModel)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(startMenuModel)

	roleCfg := model.config.Roles[domain.RoleProcurementManager]
	if roleCfg.Kind != app.PlayerKindHuman {
		t.Fatalf("procurement role kind = %q, want human after toggle", roleCfg.Kind)
	}

	model.cursor = 0
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = next.(startMenuModel)
	if model.action != startMenuActionStartGame {
		t.Fatalf("menu action = %v, want start game", model.action)
	}
}

func TestStartMenuCanSwitchScenarios(t *testing.T) {
	const scenarioID = domain.ScenarioID("menu-test-scenario")
	if _, exists := scenario.Lookup(scenarioID); !exists {
		err := scenario.Register(scenario.NewDefinition(
			scenarioID,
			"Menu Test Scenario",
			"Verifies the start menu can switch between registered scenarios.",
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
					RevenueTarget:         10,
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
		))
		if err != nil {
			t.Fatalf("scenario.Register() error = %v", err)
		}
	}

	model := newStartMenuModel(testMenuConfig())
	model.cursor = 1

	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = next.(startMenuModel)
	if model.config.ScenarioID == scenario.StarterID {
		t.Fatalf("scenario id = %q, want scenario selection to advance", model.config.ScenarioID)
	}
}

func testMenuConfig() app.Config {
	return app.Config{
		Environment: "test",
		ScenarioID:  scenario.StarterID,
		LLMCatalog: app.LLMCatalog{
			Entries: []app.LLMCatalogEntry{
				{Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: app.APISDKTypeOpenAI},
			},
		},
		Roles: map[domain.RoleID]app.RoleConfig{
			domain.RoleProcurementManager: {Kind: app.PlayerKindAI, Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: app.APISDKTypeOpenAI},
			domain.RoleProductionManager:  {Kind: app.PlayerKindAI, Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: app.APISDKTypeOpenAI},
			domain.RoleSalesManager:       {Kind: app.PlayerKindAI, Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: app.APISDKTypeOpenAI},
			domain.RoleFinanceController:  {Kind: app.PlayerKindAI, Provider: "openrouter", Model: "openai/gpt-5-mini", URL: "https://openrouter.ai/api/v1/", APISDKType: app.APISDKTypeOpenAI},
		},
	}
}
