package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestModelQReturnsToMenuWhenConfigured(t *testing.T) {
	initial := scenario.Starter().InitialState("starter-match", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", Provider: "ollama", ModelName: "gemma"},
	})

	model := NewModelWithSubmitAndQuitBehavior(scenario.Starter(), testStateSource{snapshot: initial}, nil, QuitBehaviorReturnToMenu)
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := next.(Model)

	if updated.ExitIntent() != ExitIntentReturnToMenu {
		t.Fatalf("ExitIntent() = %q, want %q", updated.ExitIntent(), ExitIntentReturnToMenu)
	}
}

func TestModelQQuitsProgramByDefault(t *testing.T) {
	initial := scenario.Starter().InitialState("starter-match", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", Provider: "ollama", ModelName: "gemma"},
	})

	model := NewModelWithSubmit(scenario.Starter(), testStateSource{snapshot: initial}, nil)
	next, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := next.(Model)

	if updated.ExitIntent() != ExitIntentQuit {
		t.Fatalf("ExitIntent() = %q, want %q", updated.ExitIntent(), ExitIntentQuit)
	}
}
