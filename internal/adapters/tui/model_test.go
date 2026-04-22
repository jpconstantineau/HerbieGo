package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
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
	initial.RoundFlow.SubmittedRoles = []domain.RoleID{domain.RoleProcurementManager}
	initial.RoundFlow.WaitingOnRoles = []domain.RoleID{
		domain.RoleProductionManager,
		domain.RoleSalesManager,
		domain.RoleFinanceController,
	}
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

	model := NewModel(scenario.Starter(), testStateSource{snapshot: initial})
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
		"Mode: action entry",
		"Navigate: [1 action] | 2 lookup | 3 report | 4 feed | 5",
		"archive | [/] cycle",
		"cycle",
		"Action entry for Procurement Manager",
		"View: draft, review, and submit a private turn without",
		"leaving the center workspace",
		"Editing flow",
		"Orders: housing=2, seal_kit=1",
		"Plant Stats",
		"Command Bar",
		"Procurement Manager",
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
		"  Player action intake",
		"    1. Finance Controller: Margins improved.",
		"  Round simulation",
		"    1. Event: Shipped two valves.",
		"[R3] 0 events | 1 commentary",
		"  Player action intake",
		"    1. Production Manager: Assembly stayed constrained.",
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
	model := NewModel(scenario.Starter(), testStateSource{
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
	if switchedShell.workspace != workspaceScenarioLookup {
		t.Fatalf("workspace = %v, want %v", switchedShell.workspace, workspaceScenarioLookup)
	}

	reset, _ := switchedShell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	resetShell := reset.(Model)
	if resetShell.workspace != workspaceActionEntry {
		t.Fatalf("workspace = %v, want %v", resetShell.workspace, workspaceActionEntry)
	}

	feed, _ := resetShell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	feedShell := feed.(Model)
	if feedShell.workspace != workspaceScenarioLookup {
		t.Fatalf("workspace = %v, want %v", feedShell.workspace, workspaceScenarioLookup)
	}

	archive, _ := feedShell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	archiveShell := archive.(Model)
	if archiveShell.workspace != workspaceRoleReport {
		t.Fatalf("workspace = %v, want %v", archiveShell.workspace, workspaceRoleReport)
	}

	history, _ := archiveShell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	historyShell := history.(Model)
	if historyShell.workspace != workspaceRoundFeed {
		t.Fatalf("workspace = %v, want %v", historyShell.workspace, workspaceRoundFeed)
	}

	archiveView, _ := historyShell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	archiveViewShell := archiveView.(Model)
	if archiveViewShell.workspace != workspaceHistoryArchive {
		t.Fatalf("workspace = %v, want %v", archiveViewShell.workspace, workspaceHistoryArchive)
	}
}

func TestModelRendersScenarioLookupWorkspaceAndSupportsBrowsing(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.width = 120
	shell.height = 32

	lookup, _ := shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	shell = lookup.(Model)
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'d'}},
		{Type: tea.KeyDown},
	} {
		next, _ := shell.Update(key)
		shell = next.(Model)
	}

	view := shell.View()
	for _, want := range []string{
		"Mode: scenario lookup",
		"Navigate: 1 action | [2 lookup] | 3 report | 4 feed | 5",
		"archive | [/] cycle",
		"Scenario lookups for Prairie Pump Starter Plant",
		"same canonical scenario lookup surface",
		"used by AI tool calls",
		"Lookup tabs: v suppliers | r routes | b bom | [d demand]",
		"Customer demand (2/6)",
		"Customer: AgriWorks (agriworks)",
		"Product: Valve (valve)",
		"Tool parity: show_customer_demand_profile(customer_id,",
		"product_id)",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("scenario lookup view missing %q\n%s", want, view)
		}
	}
}

func TestModelSupportsHumanActionDraftReviewAndSubmit(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("housing=2")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyDown},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("Ordering only what assembly can absorb.")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'r'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
	} {
		next, _ := shell.Update(key)
		shell = next.(Model)
	}

	draft := shell.currentDraft()
	if draft.stage != draftStageSubmitted {
		t.Fatalf("draft.stage = %v, want %v", draft.stage, draftStageSubmitted)
	}
	if got := len(shell.effectiveRoundFlow().SubmittedRoles); got != 1 {
		t.Fatalf("submitted roles = %d, want 1", got)
	}
	if shell.workspace != workspaceRoundFeed {
		t.Fatalf("workspace = %v, want %v", shell.workspace, workspaceRoundFeed)
	}

	shell.width = 120
	shell.height = 32
	view := shell.View()
	for _, want := range []string{
		"Mode: round feed",
		"Submissions received: 1/4",
		"Waiting on: Production Manager, Sales Manager, Finance",
		"Controller",
		"Current-turn actions remain hidden until every role is",
		"collected and the round resolves.",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("submitted round-feed view missing %q\n%s", want, view)
		}
	}
	for _, unwanted := range []string{
		"Order 2 of housing from forgeco",
		"Ordering only what assembly can absorb.",
	} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("round feed leaked %q\n%s", unwanted, view)
		}
	}
}

func TestModelSubmitForwardsLockedDraftToLiveHook(t *testing.T) {
	var submitted domain.ActionSubmission
	model := NewModelWithSubmit(
		scenario.Starter(),
		testStateSource{snapshot: scenario.Starter().InitialState("starter-match", starterAssignments())},
		func(action domain.ActionSubmission) error {
			submitted = action.Clone()
			return nil
		},
	)

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("housing=2")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyDown},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("Ordering only what assembly can absorb.")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'r'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
	} {
		next, _ := shell.Update(key)
		shell = next.(Model)
	}

	if submitted.MatchID != "starter-match" {
		t.Fatalf("submitted.MatchID = %q, want starter-match", submitted.MatchID)
	}
	if submitted.Round != 1 {
		t.Fatalf("submitted.Round = %d, want 1", submitted.Round)
	}
	if submitted.RoleID != domain.RoleProcurementManager {
		t.Fatalf("submitted.RoleID = %q, want procurement_manager", submitted.RoleID)
	}
	if got := submitted.Commentary.Body; got != "Ordering only what assembly can absorb." {
		t.Fatalf("submitted.Commentary.Body = %q, want submitted commentary", got)
	}
	if submitted.Action.Procurement == nil || len(submitted.Action.Procurement.Orders) != 1 {
		t.Fatalf("submitted.Action.Procurement = %#v, want one order", submitted.Action.Procurement)
	}
}

func TestModelAdvancesAcrossHumanRolesAndHidesLockedDrafts(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", multiHumanAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("housing=2")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyDown},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("Buying only what we can use.")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'r'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
	} {
		next, _ := shell.Update(key)
		shell = next.(Model)
	}

	if got := shell.selectedAssignment().RoleID; got != domain.RoleSalesManager {
		t.Fatalf("selected role = %q, want sales_manager", got)
	}
	if shell.workspace != workspaceActionEntry {
		t.Fatalf("workspace = %v, want %v", shell.workspace, workspaceActionEntry)
	}

	shell.width = 120
	shell.height = 32
	view := shell.View()
	for _, want := range []string{
		"Action entry for Sales Manager",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("multi-human view missing %q\n%s", want, view)
		}
	}

	shell.selectedRole = 0
	privateView := shell.View()
	for _, want := range []string{
		"Submission locked for this round.",
		"Locked human entries stay private in multi-human games.",
	} {
		if !strings.Contains(privateView, want) {
			t.Fatalf("private locked view missing %q\n%s", want, privateView)
		}
	}
	for _, unwanted := range []string{
		"Order 2 of housing from forgeco",
		"Buying only what we can use.",
	} {
		if strings.Contains(privateView, unwanted) {
			t.Fatalf("private locked view leaked %q\n%s", unwanted, privateView)
		}
	}
}

func TestModelSwitchesToRoundFeedAfterFinalHumanSubmission(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", multiHumanAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.drafts[domain.RoleProcurementManager] = actionDraft{
		stage: draftStageSubmitted,
		submission: &domain.ActionSubmission{
			Action:     domain.RoleAction{Procurement: &domain.ProcurementAction{}},
			Commentary: domain.CommentaryRecord{Body: "Already locked."},
		},
	}
	shell.selectedRole = 2

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("pump=14")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyDown},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("Holding price steady.")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'r'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
	} {
		next, _ := shell.Update(key)
		shell = next.(Model)
	}

	if shell.workspace != workspaceRoundFeed {
		t.Fatalf("workspace = %v, want %v", shell.workspace, workspaceRoundFeed)
	}
	view := shell.View()
	for _, want := range []string{
		"Mode: round feed",
		"Submissions received: 2/4",
		"Waiting on: Production Manager, Finance Controller",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("final human submission view missing %q\n%s", want, view)
		}
	}
}

func TestRoundFeedKeepsCurrentTurnSubmissionHidden(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.drafts[domain.RoleProcurementManager] = actionDraft{
		stage: draftStageSubmitted,
		submission: &domain.ActionSubmission{
			Action: domain.RoleAction{
				Procurement: &domain.ProcurementAction{
					Orders: []domain.PurchaseOrderIntent{{PartID: "housing", SupplierID: "forgeco", Quantity: 2}},
				},
			},
			Commentary: domain.CommentaryRecord{Body: "Hidden until reveal."},
		},
	}
	shell.workspace = workspaceRoundFeed
	shell.width = 120
	shell.height = 32

	view := shell.View()
	for _, want := range []string{
		"Mode: round feed",
		"Navigate: 1 action | 2 lookup | 3 report | [4 feed] | 5",
		"archive | [/] cycle",
		"cycle",
		"Submissions received: 1/4",
		"Waiting on: Production Manager, Sales Manager, Finance",
		"Controller",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("round feed missing %q\n%s", want, view)
		}
	}
	for _, unwanted := range []string{
		"Order 2 of housing from forgeco",
		"Hidden until reveal.",
	} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("round feed leaked %q\n%s", unwanted, view)
		}
	}
}

func TestFinanceActionEntrySeedsCurrentTargetsAsDefaults(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.selectedRole = 3

	submission, err := shell.buildSubmissionDraft(actionDraft{
		financeProcurement: "21",
		financeProduction:  "17",
		financeRevenue:     "33",
		financeCashFloor:   "9",
		financeDebtCeiling: "14",
		commentary:         "Keeping target posture stable while we learn the plant.",
	})
	if err != nil {
		t.Fatalf("buildSubmissionDraft() error = %v", err)
	}

	targets := submission.Action.Finance.NextRoundTargets
	if targets.EffectiveRound != 2 {
		t.Fatalf("targets.EffectiveRound = %d, want 2", targets.EffectiveRound)
	}
	if targets.ProcurementBudget != 21 || targets.ProductionSpendBudget != 17 || targets.RevenueTarget != 33 || targets.CashFloorTarget != 9 || targets.DebtCeilingTarget != 14 {
		t.Fatalf("targets = %+v, want explicit values copied into next-round targets", targets)
	}
}

func TestModelResubscribesToStateUpdates(t *testing.T) {
	updates := make(chan domain.MatchState, 1)
	model := NewModel(scenario.Starter(), testStateSource{
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
	model := NewModel(scenario.Starter(), testStateSource{
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
		"Navigate: [1 action] | 2 lookup | 3 report | 4 feed | 5",
		"archive | [/] cycle",
		"Action entry for Procurement Manager",
		"Orders: housing=2, seal_kit=1",
		"Cash: 24",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("compact View() missing %q\n%s", want, view)
		}
	}
}

func TestModelViewFitsWithinTerminalWidth(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
	}{
		{name: "wide", width: 160, height: 40},
		{name: "compact", width: 96, height: 22},
		{name: "stacked", width: 68, height: 18},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := NewModel(scenario.Starter(), testStateSource{
				snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
			})

			loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
			shell := loaded.(Model)
			shell.width = tc.width
			shell.height = tc.height

			view := shell.View()
			lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
			for index, line := range lines {
				if got := lipgloss.Width(line); got > tc.width {
					t.Fatalf("line %d width = %d, want at most %d\n%s", index+1, got, tc.width, view)
				}
			}
		})
	}
}

func TestModelRoleReportShowsCompanySnapshot(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.workspace = workspaceRoleReport
	shell.width = 120
	shell.height = 30

	view := shell.View()
	for _, want := range []string{
		"Mode: role report",
		"1 action | 2 lookup | [3 report] | 4 feed | 5",
		"archive | [/] cycle",
		"cycle",
		"Role report for Procurement Manager",
		"Company snapshot",
		"Inventory value:",
		"Tracked product financial summaries:",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("role report missing %q\n%s", want, view)
		}
	}
}

func TestModelRoundFeedExplainsResolvingAndRevealedStates(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.width = 120
	shell.height = 30
	shell.workspace = workspaceRoundFeed

	shell.state.RoundFlow.Phase = domain.RoundPhaseResolving
	resolvingView := shell.View()
	for _, want := range []string{
		"View: active round context and recent resolved feed",
		"Current phase: resolving simultaneous turn",
		"The round is resolving.",
		"All current-turn actions are locked in.",
		"The plant is resolving simultaneous decisions before",
		"reveal.",
	} {
		if !strings.Contains(resolvingView, want) {
			t.Fatalf("resolving view missing %q\n%s", want, resolvingView)
		}
	}

	shell.state.RoundFlow.Phase = domain.RoundPhaseRevealed
	shell.state.RoundFlow.AIRevealDelaySeconds = 15
	shell.state.Roles = []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: false},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", IsHuman: false},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", IsHuman: false},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", IsHuman: false},
	}
	revealedView := shell.View()
	for _, want := range []string{
		"Current phase: revealed round results",
		"The round has been revealed.",
		"Round results are now visible in the resolved history",
		"below.",
		"AI-only rounds hold the reveal for 15 seconds before",
		"advancing.",
	} {
		if !strings.Contains(revealedView, want) {
			t.Fatalf("revealed view missing %q\n%s", want, revealedView)
		}
	}
}

func TestModelRoundFeedShowsResolvedStarterHistory(t *testing.T) {
	starter := scenario.Starter()
	state := starter.InitialState("starter-match", starterAssignments())
	resolver := engine.NewResolver(starter.ResolverOptions())

	result, err := resolver.ResolveRound(state, []domain.ActionSubmission{
		{
			ActionID: "prod-1",
			MatchID:  state.MatchID,
			Round:    state.CurrentRound,
			RoleID:   domain.RoleProductionManager,
			Action: domain.RoleAction{
				Production: &domain.ProductionAction{
					CapacityAllocation: []domain.CapacityAllocation{
						{WorkstationID: "assembly", ProductID: "pump", Capacity: 2},
					},
				},
			},
			Commentary: domain.CommentaryRecord{Body: "Finishing inherited pump WIP before releasing more work."},
		},
		{
			ActionID: "sales-1",
			MatchID:  state.MatchID,
			Round:    state.CurrentRound,
			RoleID:   domain.RoleSalesManager,
			Action: domain.RoleAction{
				Sales: &domain.SalesAction{
					ProductOffers: []domain.ProductOffer{
						{ProductID: "pump", UnitPrice: 14},
						{ProductID: "valve", UnitPrice: 9},
					},
				},
			},
			Commentary: domain.CommentaryRecord{Body: "Holding starter prices while we clear inherited backlog."},
		},
	}, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	next := result.NextState
	next.RoundFlow.Phase = domain.RoundPhaseRevealed
	next.RoundFlow.SubmittedRoles = []domain.RoleID{
		domain.RoleProductionManager,
		domain.RoleSalesManager,
	}
	next.RoundFlow.WaitingOnRoles = []domain.RoleID{
		domain.RoleProcurementManager,
		domain.RoleFinanceController,
	}

	model := NewModel(starter, testStateSource{snapshot: next})
	loaded, _ := model.Update(stateLoadedMsg{state: next})
	shell := loaded.(Model)
	shell.workspace = workspaceRoundFeed
	shell.width = 120
	shell.height = 32

	view := shell.View()
	for _, want := range []string{
		"Current phase: revealed round results",
		"Recent resolved rounds (1 shown)",
		"[R1] 20 events | 2 commentary",
		"Sales Manager: Holding starter prices while we",
		"clear inherited backlog.",
		"Event: Shipped 2 pump to northbuild",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("revealed round feed missing %q\n%s", want, view)
		}
	}
}

func TestModelDepartmentsPaneShowsBrailleSpinnerForProviderWaits(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.width = 120
	shell.height = 32
	shell.state.RoundFlow.ProviderWaitingRoles = []domain.RoleID{domain.RoleProductionManager}

	view := shell.View()
	if !strings.Contains(view, "Production Manager "+providerSpinnerFrames[0]+" [AI]") {
		t.Fatalf("departments view missing provider spinner\n%s", view)
	}
	if strings.Contains(view, "Procurement Manager "+providerSpinnerFrames[0]+" [Human]") {
		t.Fatalf("departments view showed spinner for unrelated human role\n%s", view)
	}
}

func TestModelArchiveShowsRetainedHistorySummaries(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.workspace = workspaceHistoryArchive
	shell.width = 120
	shell.height = 30
	shell.state.History.RecentRounds = []domain.RoundRecord{
		{
			Round:   1,
			Actions: []domain.ActionSubmission{{ActionID: "a-1"}},
			Events: []domain.RoundEvent{
				{Summary: "Assembly shipped one pump."},
			},
			Commentary: []domain.CommentaryRecord{
				{RoleID: domain.RoleSalesManager, Body: "Demand stayed healthy."},
			},
			Metrics: domain.PlantMetrics{RoundProfit: 18, NetCashChange: 7},
		},
	}

	view := shell.View()
	for _, want := range []string{
		"Mode: history archive",
		"1 action | 2 lookup | 3 report | 4 feed | [5",
		"archive] | [/] cycle",
		"cycle",
		"Rounds retained: 1",
		"Use this view for older rounds and per-round summaries",
		"rather than the current feed.",
		"[R1] 1 actions | 1 events | 1 commentary | profit 18 |",
		"net cash 7",
		"Player action intake",
		"1. Sales Manager: Demand stayed healthy.",
		"Round simulation",
		"1. Event: Assembly shipped one pump.",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("archive view missing %q\n%s", want, view)
		}
	}
}

func TestModelScrollsRoundFeedHistoryWithKeyboard(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.workspace = workspaceRoundFeed
	shell.focusedPane = paneHistory
	shell.width = 140
	shell.height = 24
	shell.state.History.RecentRounds = []domain.RoundRecord{
		longRoundRecord(3, 18),
	}

	initialView := shell.View()
	if !strings.Contains(initialView, "Event 01 for round 03") {
		t.Fatalf("initial round feed view missing first event\n%s", initialView)
	}
	if strings.Contains(initialView, "Event 18 for round 03") {
		t.Fatalf("initial round feed view unexpectedly showed final event\n%s", initialView)
	}

	scrolled, _ := shell.Update(tea.KeyMsg{Type: tea.KeyEnd})
	scrolledShell := scrolled.(Model)
	scrolledShell.width = shell.width
	scrolledShell.height = shell.height

	scrolledView := scrolledShell.View()
	if !strings.Contains(scrolledView, "Event 18 for round 03") {
		t.Fatalf("scrolled round feed view missing final event\n%s", scrolledView)
	}
}

func TestModelScrollsArchiveHistoryWithMouseWheel(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.workspace = workspaceHistoryArchive
	shell.focusedPane = paneHistory
	shell.width = 140
	shell.height = 24
	shell.state.History.RecentRounds = []domain.RoundRecord{
		longRoundRecord(1, 1),
		longRoundRecord(2, 1),
		longRoundRecord(3, 1),
		longRoundRecord(4, 1),
	}

	initialView := shell.View()
	if !strings.Contains(initialView, "[R4] 1 actions | 1 events | 1 commentary") {
		t.Fatalf("initial archive view missing newest round\n%s", initialView)
	}
	if strings.Contains(initialView, "[R1] 1 actions | 1 events | 1 commentary") {
		t.Fatalf("initial archive view unexpectedly showed oldest round\n%s", initialView)
	}

	var next tea.Model = shell
	for i := 0; i < 8; i++ {
		next, _ = next.(Model).Update(tea.MouseMsg(tea.MouseEvent{
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		}))
	}
	scrolledShell := next.(Model)
	scrolledShell.width = shell.width
	scrolledShell.height = shell.height

	scrolledView := scrolledShell.View()
	if !strings.Contains(scrolledView, "[R1] 1 actions | 1 events | 1 commentary") {
		t.Fatalf("scrolled archive view missing oldest round\n%s", scrolledView)
	}
}

func longRoundRecord(round, eventCount int) domain.RoundRecord {
	events := make([]domain.RoundEvent, 0, eventCount)
	for index := 1; index <= eventCount; index++ {
		events = append(events, domain.RoundEvent{Summary: "Event " + twoDigit(index) + " for round " + twoDigit(round)})
	}

	return domain.RoundRecord{
		Round:   domain.RoundNumber(round),
		Actions: []domain.ActionSubmission{{ActionID: domain.ActionID("a-" + twoDigit(round))}},
		Events:  events,
		Commentary: []domain.CommentaryRecord{
			{RoleID: domain.RoleSalesManager, Body: "Commentary for round " + twoDigit(round)},
		},
		Metrics: domain.PlantMetrics{RoundProfit: domain.Money(round), NetCashChange: domain.Money(round)},
	}
}

func twoDigit(value int) string {
	return fmt.Sprintf("%02d", value)
}

func starterAssignments() []domain.RoleAssignment {
	return []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", IsHuman: false, Provider: "ollama"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", IsHuman: false, Provider: "openrouter"},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", IsHuman: false, Provider: "openai"},
	}
}

func multiHumanAssignments() []domain.RoleAssignment {
	return []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", IsHuman: false, Provider: "ollama"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", IsHuman: true},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", IsHuman: false, Provider: "openai"},
	}
}
