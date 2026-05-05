package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestAppShellTransitionsFromSplashToMenu(t *testing.T) {
	runtime, err := app.NewRuntime(testMenuConfig())
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	model, err := newAppShellModel(context.Background(), runtime, domain.MatchState{}, false, nil, 1, false)
	if err != nil {
		t.Fatalf("newAppShellModel() error = %v", err)
	}

	next, _ := model.Update(splashTickMsg{})
	model = unwrapShellModel(t, next)
	for model.screen != screenMenu {
		next, _ = model.Update(splashTickMsg{})
		model = unwrapShellModel(t, next)
	}

	if model.screen != screenMenu {
		t.Fatalf("screen = %v, want screenMenu", model.screen)
	}
}

func TestAppShellRoutesStartNewGameIntoGameplay(t *testing.T) {
	runtime, err := app.NewRuntime(testMenuConfig())
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	model, err := newAppShellModel(context.Background(), runtime, domain.MatchState{}, false, nil, 1, false)
	if err != nil {
		t.Fatalf("newAppShellModel() error = %v", err)
	}

	model.screen = screenMenu
	model.menu = newStartMenuModel(model.menuConfig, model.menuState)
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = unwrapShellModel(t, next)

	if model.screen != screenGameplay {
		t.Fatalf("screen = %v, want screenGameplay", model.screen)
	}
	if model.current == nil {
		t.Fatal("current session = nil, want active session after starting game")
	}
}

func TestAppShellAppliesExistingWindowSizeWhenEnteringGameplay(t *testing.T) {
	runtime, err := app.NewRuntime(testMenuConfig())
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}

	model, err := newAppShellModel(context.Background(), runtime, domain.MatchState{}, false, nil, 1, false)
	if err != nil {
		t.Fatalf("newAppShellModel() error = %v", err)
	}

	model.screen = screenMenu
	model.menu = newStartMenuModel(model.menuConfig, model.menuState)

	next, _ := model.Update(tea.WindowSizeMsg{Width: 180, Height: 50})
	model = unwrapShellModel(t, next)
	next, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = unwrapShellModel(t, next)

	if model.screen != screenGameplay || model.gameplay == nil {
		t.Fatalf("screen = %v, gameplay = %#v, want gameplay screen", model.screen, model.gameplay)
	}

	debugView := fmt.Sprintf("%#v", model.gameplay.model)
	if !containsAll(debugView, "width:180", "height:50") {
		t.Fatalf("gameplay model dimensions were not carried into the embedded shell:\n%s", debugView)
	}
}

func unwrapShellModel(t *testing.T, model tea.Model) appShellModel {
	t.Helper()

	switch typed := model.(type) {
	case appShellModel:
		return typed
	case *appShellModel:
		return *typed
	default:
		t.Fatalf("unexpected shell model type %T", model)
		return appShellModel{}
	}
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}

func TestRuntimeConfigFromMenuFiltersRolesToScenarioRoster(t *testing.T) {
	const scenarioID = domain.ScenarioID("shell-test-scenario")
	if _, exists := scenario.Lookup(scenarioID); !exists {
		err := scenario.Register(scenario.NewDefinition(
			scenarioID,
			"Shell Test Scenario",
			"Verifies runtime config role filtering.",
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

	cfg := testMenuConfig()
	cfg.ScenarioID = scenarioID
	filtered := runtimeConfigFromMenu(cfg)
	if len(filtered.Roles) != 1 {
		t.Fatalf("len(filtered.Roles) = %d, want 1", len(filtered.Roles))
	}
	if _, ok := filtered.Roles[domain.RoleSalesManager]; !ok {
		t.Fatal("filtered roles missing sales_manager")
	}
}
