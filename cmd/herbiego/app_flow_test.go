package main

import (
	"context"
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
