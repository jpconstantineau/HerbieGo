package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type testStateSource struct {
	snapshot domain.MatchState
	updates  <-chan domain.MatchState
}

func (s testStateSource) Snapshot() domain.MatchState {
	return s.snapshot.Clone()
}

func (s testStateSource) Updates() <-chan domain.MatchState {
	return s.updates
}

func TestModelLoadsInitialSnapshotAndRendersShell(t *testing.T) {
	initial := scenario.Starter().InitialState("starter-match", starterAssignments())
	initial.History.RecentRounds = []domain.RoundRecord{
		{
			Round: 1,
			Events: []domain.RoundEvent{
				{Summary: "Assembly shipped one pump."},
			},
			Commentary: []domain.CommentaryRecord{
				{RoleID: domain.RoleSalesManager, Body: "Demand stayed healthy."},
			},
		},
	}

	model := NewModel("Prairie Pump Starter Plant", testStateSource{snapshot: initial})
	cmd := model.Init()
	msg := cmd()

	nextModel, nextCmd := model.Update(msg)
	if nextCmd != nil {
		t.Fatalf("expected nil follow-up cmd for static source")
	}

	shell := nextModel.(Model)
	shell.width = 120
	shell.height = 32
	view := shell.View()

	for _, want := range []string{
		"Departments [focus]",
		"Center Workspace",
		"Mode: round feed",
		"Plant Stats",
		"Command Bar",
		"Procurement Manager",
		"[R1] 1 events | 1 commentary",
		"Event: Assembly shipped one pump.",
		"Sales Manager: Demand stayed healthy.",
		"Inspect mode",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("View() missing %q\n%s", want, view)
		}
	}
}

func TestHistoryFeedEntriesMergesRoundsIntoSingleChronologicalFeed(t *testing.T) {
	entries := historyFeedEntries([]domain.RoundHistoryEntry{
		{
			Round: 2,
			Events: []domain.RoundEvent{
				{Summary: "Shipped two valves."},
			},
			Commentary: []domain.CommentaryRecord{
				{RoleID: domain.RoleFinanceController, Body: "Margins improved."},
			},
		},
		{
			Round: 3,
			Commentary: []domain.CommentaryRecord{
				{RoleID: domain.RoleProductionManager, Body: "Assembly stayed constrained."},
			},
		},
	})

	want := []string{
		"[R2] 1 events | 1 commentary",
		"  Event: Shipped two valves.",
		"  Finance Controller: Margins improved.",
		"[R3] 0 events | 1 commentary",
		"  Production Manager: Assembly stayed constrained.",
	}
	if len(entries) != len(want) {
		t.Fatalf("len(entries) = %d, want %d (%v)", len(entries), len(want), entries)
	}
	for index := range want {
		if entries[index] != want[index] {
			t.Fatalf("entries[%d] = %q, want %q", index, entries[index], want[index])
		}
	}
}

func TestModelCyclesRoleSelectionAndPaneFocus(t *testing.T) {
	model := NewModel("Prairie Pump Starter Plant", testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)

	shifted, _ := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	shiftedShell := shifted.(Model)
	if got := shiftedShell.roleTitle(); got != "Production Manager" {
		t.Fatalf("roleTitle() = %q, want Production Manager", got)
	}

	focused, _ := shiftedShell.Update(tea.KeyMsg{Type: tea.KeyTab})
	focusedShell := focused.(Model)
	if focusedShell.focusedPane != paneHistory {
		t.Fatalf("focusedPane = %d, want %d", focusedShell.focusedPane, paneHistory)
	}

	switched, _ := focusedShell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	switchedShell := switched.(Model)
	if switchedShell.workspace != workspaceHistoryArchive {
		t.Fatalf("workspace = %v, want %v", switchedShell.workspace, workspaceHistoryArchive)
	}
}

func TestModelResubscribesToStateUpdates(t *testing.T) {
	updates := make(chan domain.MatchState, 1)
	model := NewModel("Prairie Pump Starter Plant", testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
		updates:  updates,
	})

	loaded, cmd := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	if cmd == nil {
		t.Fatalf("expected follow-up subscription cmd")
	}

	next := scenario.Starter().InitialState("starter-match", starterAssignments())
	next.CurrentRound = 2
	updates <- next

	msg := cmd()
	resynced, followUp := loaded.(Model).Update(msg)
	if followUp == nil {
		t.Fatalf("expected to keep listening after update")
	}
	if got := resynced.(Model).state.CurrentRound; got != 2 {
		t.Fatalf("CurrentRound = %d, want 2", got)
	}
}

func TestModelUsesCompactLayoutAndEmptyStatesOnSmallerTerminal(t *testing.T) {
	model := NewModel("Prairie Pump Starter Plant", testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.width = 96
	shell.height = 22
	shell.state.History = domain.RoundHistory{}
	shell.state.Plant.Workstations = nil

	view := shell.View()

	for _, want := range []string{
		"Departments [focus]",
		"Plant Stats",
		"Center Workspace",
		"No prior rounds recorded yet.",
		"Workstations: waiting for first telemetry",
		"Mode: inspect | Focus: departments | Role: Procurement Manager | Round: 1",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("compact View() missing %q\n%s", want, view)
		}
	}
}

func TestModelActionWorkspaceExplainsDeferredEntrySurface(t *testing.T) {
	model := NewModel("Prairie Pump Starter Plant", testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.workspace = workspaceActionEntry
	shell.width = 120
	shell.height = 30

	view := shell.View()
	for _, want := range []string{
		"Mode: action entry",
		"Decision workspace for Procurement Manager",
		"Action entry is intentionally deferred from issue #23",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("action workspace missing %q\n%s", want, view)
		}
	}
}

func starterAssignments() []domain.RoleAssignment {
	return []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", IsHuman: false, Provider: "ollama"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", IsHuman: false, Provider: "openrouter"},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", IsHuman: false, Provider: "openai"},
	}
}
