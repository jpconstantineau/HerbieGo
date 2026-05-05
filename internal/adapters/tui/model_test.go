package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type fakeScenarioReader struct {
	name         string
	parts        []scenario.Part
	products     []scenario.Product
	workstations []scenario.Workstation
	demandRefs   []scenario.DemandProfileReference
}

func (f fakeScenarioReader) ScenarioDisplayName() string {
	return f.name
}

func (f fakeScenarioReader) Parts() []scenario.Part {
	return append([]scenario.Part(nil), f.parts...)
}

func (f fakeScenarioReader) Products() []scenario.Product {
	return append([]scenario.Product(nil), f.products...)
}

func (f fakeScenarioReader) Workstations() []scenario.Workstation {
	return append([]scenario.Workstation(nil), f.workstations...)
}

func (f fakeScenarioReader) DemandProfileReferences() []scenario.DemandProfileReference {
	return append([]scenario.DemandProfileReference(nil), f.demandRefs...)
}

func (f fakeScenarioReader) ListValidSuppliers(partID domain.PartID) (scenario.ValidSuppliersLookup, error) {
	return scenario.ValidSuppliersLookup{
		PartID:      partID,
		DisplayName: "Mock Part",
		Suppliers:   []domain.SupplierID{"mock-supplier"},
	}, nil
}

func (f fakeScenarioReader) ShowProductRoute(productID domain.ProductID) (scenario.ProductRouteLookup, error) {
	return scenario.ProductRouteLookup{
		ProductID:    productID,
		DisplayName:  "Mock Product",
		Route:        []domain.WorkstationID{"mock-station"},
		BottleneckID: "mock-station",
	}, nil
}

func (f fakeScenarioReader) ShowProductBOM(productID domain.ProductID) (scenario.ProductBOMLookup, error) {
	return scenario.ProductBOMLookup{
		ProductID:    productID,
		DisplayName:  "Mock Product",
		BOM:          []domain.BOMLine{{PartID: "mock-part", Quantity: 1}},
		BaseUnitCost: 3,
	}, nil
}

func (f fakeScenarioReader) ShowCustomerDemandProfile(customerID domain.CustomerID, productID domain.ProductID) (scenario.CustomerDemandProfileLookup, error) {
	return scenario.CustomerDemandProfileLookup{
		CustomerID:       customerID,
		CustomerName:     "Mock Customer",
		ProductID:        productID,
		ProductName:      "Mock Product",
		ReferencePrice:   9,
		BaseDemand:       4,
		PriceSensitivity: 1,
	}, nil
}

type testStateSource struct {
	snapshot  domain.MatchState
	snapshots []domain.MatchState
	updates   <-chan domain.MatchState
}

func (s testStateSource) Snapshot() domain.MatchState {
	return s.snapshot.Clone()
}

func (s testStateSource) Updates() <-chan domain.MatchState {
	return s.updates
}

func (s testStateSource) StateSnapshots() []domain.MatchState {
	snapshots := s.snapshots
	if len(snapshots) == 0 {
		snapshots = []domain.MatchState{s.snapshot}
	}
	cloned := make([]domain.MatchState, len(snapshots))
	for i := range snapshots {
		cloned[i] = snapshots[i].Clone()
	}
	return cloned
}

type testDebugSource struct {
	records []ports.AICallRecord
}

func (s testDebugSource) Records() []ports.AICallRecord {
	cloned := make([]ports.AICallRecord, len(s.records))
	copy(cloned, s.records)
	return cloned
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
		"No purchase orders configured. Press a to add a row.",
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

func TestModelRendersWithScenarioReaderInsteadOfConcreteDefinition(t *testing.T) {
	initial := scenario.Starter().InitialState("starter-match", starterAssignments())
	model := NewModel(fakeScenarioReader{
		name: "Mock Scenario",
		parts: []scenario.Part{
			{ID: "mock-part", DisplayName: "Mock Part", SupplierID: "mock-supplier"},
		},
		products: []scenario.Product{
			{ID: "mock-product", DisplayName: "Mock Product"},
		},
		workstations: []scenario.Workstation{
			{ID: "mock-station", DisplayName: "Mock Station"},
		},
		demandRefs: []scenario.DemandProfileReference{
			{CustomerID: "mock-customer", CustomerName: "Mock Customer", ProductID: "mock-product", ProductName: "Mock Product"},
		},
	}, testStateSource{snapshot: initial})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.width = 120
	shell.height = 32

	view := shell.View()
	if !strings.Contains(view, "Scenario: Mock Scenario") {
		t.Fatalf("View() did not render mock scenario reader name\n%s", view)
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

	shifted, _ := shell.Update(tea.KeyMsg{Type: tea.KeyDown})
	shiftedShell := shifted.(Model)
	if got := shiftedShell.roleTitle(); got != "Production Manager" {
		t.Fatalf("roleTitle() = %q, want Production Manager", got)
	}

	focused, _ := shiftedShell.Update(tea.KeyMsg{Type: tea.KeyTab})
	focusedShell := focused.(Model)
	if focusedShell.focusedPane != paneHistory {
		t.Fatalf("focusedPane = %d, want %d", focusedShell.focusedPane, paneHistory)
	}

	unchanged, _ := focusedShell.Update(tea.KeyMsg{Type: tea.KeyDown})
	unchangedShell := unchanged.(Model)
	if got := unchangedShell.roleTitle(); got != "Production Manager" {
		t.Fatalf("roleTitle() with history focus = %q, want Production Manager", got)
	}

	switched, _ := unchangedShell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
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
	shell.focusedPane = paneHistory

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
	shell.focusedPane = paneHistory
	shell = submitProcurementDraft(shell, "Ordering only what assembly can absorb.")

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
	shell.focusedPane = paneHistory
	shell = submitProcurementDraft(shell, "Ordering only what assembly can absorb.")

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
	shell.focusedPane = paneHistory
	shell = submitProcurementDraft(shell, "Buying only what we can use.")

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
	shell.focusedPane = paneHistory
	shell = submitSalesDraft(shell, "Holding price steady.")

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
	draft := shell.currentDraftForRole(domain.RoleFinanceController)
	draft.form.Values["procurement_budget"] = formFieldValue{Scalar: "21"}
	draft.form.Values["production_spend_budget"] = formFieldValue{Scalar: "17"}
	draft.form.Values["revenue_target"] = formFieldValue{Scalar: "33"}
	draft.form.Values["cash_floor_target"] = formFieldValue{Scalar: "9"}
	draft.form.Values["debt_ceiling_target"] = formFieldValue{Scalar: "14"}
	draft.form.Values["commentary"] = formFieldValue{Scalar: "Keeping target posture stable while we learn the plant."}

	submission, err := shell.buildSubmissionDraft(draft)
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
		"Editing flow",
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

func TestCommandBarHintsFollowFocusedPane(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.width = 120
	shell.height = 32

	departmentsView := shell.View()
	if !strings.Contains(departmentsView, "departments: up/down") || !strings.Contains(departmentsView, "select role") {
		t.Fatalf("departments hint missing\n%s", departmentsView)
	}

	shell.focusedPane = paneHistory
	historyView := shell.View()
	if !strings.Contains(historyView, "center workspace: up/down") || !strings.Contains(historyView, "move fields or rows, left/right move columns, a add row, x remove row") {
		t.Fatalf("history hint missing action-entry controls\n%s", historyView)
	}

	shell.workspace = workspaceScenarioLookup
	lookupView := shell.View()
	if !strings.Contains(lookupView, "center workspace: v/r/b/d") || !strings.Contains(lookupView, "switch lookup tabs, up/down browse entries") {
		t.Fatalf("history hint missing lookup controls\n%s", lookupView)
	}

	shell.focusedPane = paneStats
	statsView := shell.View()
	if !strings.Contains(statsView, "plant stats: read-only") || !strings.Contains(statsView, "summary") {
		t.Fatalf("stats hint missing\n%s", statsView)
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
		"Core decision:",
		"Inventory risk",
		"Inventory value:",
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
		"[R1] 23 events | 2 commentary",
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

func TestProviderSpinnerCompletesOneCycleEveryTwoSeconds(t *testing.T) {
	if got, want := providerSpinnerFrameInterval, 250*time.Millisecond; got != want {
		t.Fatalf("providerSpinnerFrameInterval = %s, want %s", got, want)
	}
}

func TestModelIgnoresStaleSpinnerTicks(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.state.RoundFlow.ProviderWaitingRoles = []domain.RoleID{domain.RoleProductionManager}
	shell.spinnerActive = true
	shell.spinnerGen = 2

	next, cmd := shell.Update(spinnerTickMsg{generation: 1})
	if cmd != nil {
		t.Fatalf("stale spinner tick unexpectedly scheduled follow-up work")
	}
	if got := next.(Model).spinnerFrame; got != 0 {
		t.Fatalf("spinnerFrame = %d after stale tick, want 0", got)
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

func TestModelDebugWorkspaceFiltersRecordsBySelectedRole(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.debugLog = testDebugSource{records: []ports.AICallRecord{
		{
			RoleID:       domain.RoleProcurementManager,
			Round:        1,
			Attempt:      1,
			Provider:     "ollama",
			Model:        "llama3",
			SystemPrompt: "Procurement system prompt",
			UserPrompt:   "Procurement user prompt",
			RawResponse:  `{"action":{"procurement":{"orders":[]}},"commentary":{"public_summary":"ok","focus_tags":["supply"]}}`,
			Valid:        true,
		},
		{
			RoleID:       domain.RoleSalesManager,
			Round:        2,
			Attempt:      1,
			Provider:     "openrouter",
			Model:        "gpt",
			SystemPrompt: "Sales system prompt",
			UserPrompt:   "Sales user prompt",
			RawResponse:  `{"action":{"sales":{"product_offers":[]}},"commentary":{"public_summary":"ok","focus_tags":["pricing"]}}`,
			Valid:        true,
		},
	}}
	shell.workspace = workspaceDebug
	shell.width = 120
	shell.height = 30
	shell.ensureDebugSelection()
	shell.debugExpanded[debugRoundNodeID(1)] = true
	shell.debugExpanded[debugAttemptNodeID(1, 1)] = true

	view := shell.View()
	for _, want := range []string{
		"Debug inspector for Procurement Manager",
		"Prompt/response traces for Procurement Manager (1",
		"total)",
		"Round 1 (1 tries)",
		"Try 1 - Success",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("procurement debug view missing %q\n%s", want, view)
		}
	}
	for _, unwanted := range []string{
		"Round 2 (1 tries)",
		"Sales system prompt",
	} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("procurement debug view leaked %q\n%s", unwanted, view)
		}
	}

	shell.focusedPane = paneDepartments
	next, _ := shell.Update(tea.KeyMsg{Type: tea.KeyDown})
	next, _ = next.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	salesShell := next.(Model)
	salesShell.width = shell.width
	salesShell.height = shell.height
	salesShell.debugExpanded[debugRoundNodeID(2)] = true

	salesView := salesShell.View()
	if !strings.Contains(salesView, "Debug inspector for Sales Manager") || !strings.Contains(salesView, "Round 2 (1 tries)") {
		t.Fatalf("sales debug view did not follow role selection\n%s", salesView)
	}
	if strings.Contains(salesView, "Procurement system prompt") {
		t.Fatalf("sales debug view leaked procurement request\n%s", salesView)
	}
}

func TestModelDebugTreeNavigationExpandsAndCollapses(t *testing.T) {
	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})

	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.debugLog = testDebugSource{records: []ports.AICallRecord{
		{
			RoleID:       domain.RoleProcurementManager,
			Round:        1,
			Attempt:      1,
			Provider:     "ollama",
			Model:        "llama3",
			SystemPrompt: "System prompt body",
			UserPrompt:   "User prompt body",
			RawResponse:  `{"bad_json":`,
			ErrorMessage: "response failed validation",
		},
	}}
	shell.workspace = workspaceDebug
	shell.focusedPane = paneHistory
	shell.width = 120
	shell.height = 30
	shell.ensureDebugSelection()

	if got := shell.debugSelected; got != "debug:traces" {
		t.Fatalf("initial debugSelected = %q, want root trace node", got)
	}

	next, _ := shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = next.(Model)
	if shell.debugSelected != debugRoundNodeID(1) {
		t.Fatalf("enter on trace root did not select round child: selected=%q", shell.debugSelected)
	}

	next, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = next.(Model)
	if !shell.debugExpanded[debugRoundNodeID(1)] || shell.debugSelected != debugAttemptNodeID(1, 1) {
		t.Fatalf("enter on round did not expand/select first attempt: expanded=%v selected=%q", shell.debugExpanded[debugRoundNodeID(1)], shell.debugSelected)
	}

	next, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = next.(Model)
	if !shell.debugExpanded[debugAttemptNodeID(1, 1)] || shell.debugSelected != debugStatusNodeID(1, 1) {
		t.Fatalf("enter on attempt did not expand/select status child: expanded=%v selected=%q", shell.debugExpanded[debugAttemptNodeID(1, 1)], shell.debugSelected)
	}

	view := shell.View()
	for _, want := range []string{
		"Try status details",
		"Recorded details:",
		"response failed validation",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expanded debug view missing %q\n%s", want, view)
		}
	}

	next, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEsc})
	shell = next.(Model)
	if shell.debugExpanded[debugAttemptNodeID(1, 1)] || shell.debugSelected != debugAttemptNodeID(1, 1) {
		t.Fatalf("esc on status details did not collapse to attempt: expanded=%v selected=%q", shell.debugExpanded[debugAttemptNodeID(1, 1)], shell.debugSelected)
	}

	next, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEsc})
	shell = next.(Model)
	if shell.debugExpanded[debugRoundNodeID(1)] || shell.debugSelected != debugRoundNodeID(1) {
		t.Fatalf("esc on attempt did not collapse to round: expanded=%v selected=%q", shell.debugExpanded[debugRoundNodeID(1)], shell.debugSelected)
	}
}

func TestModelDebugInspectorShowsActionInspectionAndStateDiff(t *testing.T) {
	before := scenario.Starter().InitialState("starter-match", starterAssignments())
	after := before.Clone()
	after.CurrentRound = 2
	after.Plant.Cash = 31
	after.Plant.Debt = 2
	after.Plant.Backlog = after.Plant.Backlog[:1]
	after.Metrics.RoundProfit = 7
	after.Metrics.NetCashChange = 7
	after.Metrics.PartsOnHandUnits = 3
	after.Metrics.FinishedGoodsUnits = 1
	round := domain.RoundRecord{
		Round: 1,
		Actions: []domain.ActionSubmission{
			{
				ActionID: "proc-1",
				MatchID:  before.MatchID,
				Round:    1,
				RoleID:   domain.RoleProcurementManager,
				Action: domain.RoleAction{
					Procurement: &domain.ProcurementAction{
						Orders: []domain.PurchaseOrderIntent{{PartID: "housing", SupplierID: "forgeco", Quantity: 2}},
					},
				},
				Commentary: domain.CommentaryRecord{Body: "Buying only what assembly can absorb."},
			},
		},
		Metrics: after.Metrics,
	}
	after.History.RecentRounds = []domain.RoundRecord{round}

	model := NewModel(scenario.Starter(), testStateSource{
		snapshot:  after,
		snapshots: []domain.MatchState{before, after},
	})
	loaded, _ := model.Update(stateLoadedMsg{state: after})
	shell := loaded.(Model)
	shell.workspace = workspaceDebug
	shell.width = 120
	shell.height = 30
	shell.debugExpanded["debug:inspection"] = true
	shell.debugExpanded[debugInspectionRoundNodeID(1)] = true

	view := shell.View()
	for _, want := range []string{
		"Round inspections (1 retained)",
		"Action inspection for Procurement Manager",
		"Order 2 of housing from forgeco",
		"Commentary: Buying only what assembly can absorb.",
		"State transition summary",
		"Cash: 24 -> 31 (+7)",
		"Debt: 0 -> 2 (+2)",
		"Round profit: 7",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("debug inspector missing %q\n%s", want, view)
		}
	}
}

func TestModelDebugTreeKeepsSelectionWhenNewRecordsArrive(t *testing.T) {
	debugLog := app.NewDebugLog(10)
	debugLog.Append(ports.AICallRecord{
		RoleID:       domain.RoleProcurementManager,
		Round:        1,
		Attempt:      1,
		Provider:     "ollama",
		Model:        "llama3",
		SystemPrompt: "System one",
		UserPrompt:   "User one",
		RawResponse:  `{"action":{"procurement":{"orders":[]}},"commentary":{"public_summary":"ok","focus_tags":["supply"]}}`,
		Valid:        true,
	})

	model := NewModel(scenario.Starter(), testStateSource{
		snapshot: scenario.Starter().InitialState("starter-match", starterAssignments()),
	})
	loaded, _ := model.Update(stateLoadedMsg{state: model.source.Snapshot()})
	shell := loaded.(Model)
	shell.debugLog = debugLog
	shell.workspace = workspaceDebug
	shell.focusedPane = paneHistory
	shell.width = 120
	shell.height = 24
	shell.ensureDebugSelection()
	shell.debugExpanded[debugRoundNodeID(1)] = true
	shell.debugSelected = debugAttemptNodeID(1, 1)

	debugLog.Append(ports.AICallRecord{
		RoleID:       domain.RoleProcurementManager,
		Round:        1,
		Attempt:      2,
		Provider:     "ollama",
		Model:        "llama3",
		SystemPrompt: "System two",
		UserPrompt:   "User two",
		RawResponse:  `{"action":{"procurement":{"orders":[]}},"commentary":{"public_summary":"retry ok","focus_tags":["supply"]}}`,
		Valid:        true,
	})

	next, _ := shell.Update(stateLoadedMsg{state: shell.state})
	updated := next.(Model)
	updated.width = shell.width
	updated.height = shell.height

	if got := updated.debugSelected; got != debugAttemptNodeID(1, 1) {
		t.Fatalf("debugSelected after update = %q, want first attempt preserved", got)
	}
	view := updated.View()
	if !strings.Contains(view, "Try 2 - Success") {
		t.Fatalf("updated debug view missing new attempt\n%s", view)
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

func submitProcurementDraft(model Model, commentary string) Model {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'a'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("2")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyDown},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune(commentary)},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'r'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
	} {
		next, _ := model.Update(key)
		model = next.(Model)
	}
	return model
}

func submitSalesDraft(model Model, commentary string) Model {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'a'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'l'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("14")},
		{Type: tea.KeyEnter},
		{Type: tea.KeyDown},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune(commentary)},
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune{'r'}},
		{Type: tea.KeyRunes, Runes: []rune{'s'}},
	} {
		next, _ := model.Update(key)
		model = next.(Model)
	}
	return model
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
