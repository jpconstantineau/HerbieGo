package main

import (
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestRequireAllHumanRejectsMixedRuntime(t *testing.T) {
	runtime := app.Runtime{
		Config: app.Config{
			HumanPlayers: 1,
		},
		Scenario: scenario.Starter(),
		InitialMatch: domain.MatchState{
			Roles: []domain.RoleAssignment{
				{RoleID: domain.RoleProductionManager, IsHuman: true},
				{RoleID: domain.RoleProcurementManager, IsHuman: false},
				{RoleID: domain.RoleSalesManager, IsHuman: false},
				{RoleID: domain.RoleFinanceController, IsHuman: false},
			},
		},
	}

	err := requireAllHuman(runtime)
	if err == nil {
		t.Fatal("requireAllHuman() error = nil, want mixed-runtime rejection")
	}
	if !strings.Contains(err.Error(), "-human-players=4") {
		t.Fatalf("requireAllHuman() error = %v, want flag guidance", err)
	}
}

func TestRequireAllHumanAcceptsFullyHumanRuntime(t *testing.T) {
	runtime := app.Runtime{
		Config: app.Config{
			HumanPlayers: 4,
		},
		Scenario: scenario.Starter(),
		InitialMatch: domain.MatchState{
			Roles: []domain.RoleAssignment{
				{RoleID: domain.RoleProductionManager, IsHuman: true},
				{RoleID: domain.RoleProcurementManager, IsHuman: true},
				{RoleID: domain.RoleSalesManager, IsHuman: true},
				{RoleID: domain.RoleFinanceController, IsHuman: true},
			},
		},
	}

	if err := requireAllHuman(runtime); err != nil {
		t.Fatalf("requireAllHuman() error = %v, want nil", err)
	}
}
