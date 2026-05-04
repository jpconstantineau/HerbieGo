package sqlite_test

import (
	"path/filepath"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/persistence/sqlite"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

func TestStoreCommitRoundExposesCurrentStateAndTimelines(t *testing.T) {
	store := newStore(t)
	initial := domain.MatchState{
		MatchID:      "match-1",
		ScenarioID:   "starter",
		CurrentRound: 1,
	}

	if err := store.CreateMatch(initial); err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}

	round := domain.RoundRecord{
		Round: 1,
		Events: []domain.RoundEvent{{
			EventID: "event-1",
			MatchID: "match-1",
			Round:   1,
			Type:    domain.EventBudgetActivated,
			ActorID: domain.ActorPlantSystem,
			Summary: "Activated round targets",
			Payload: map[string]any{"budget": 10},
		}},
		Commentary: []domain.CommentaryRecord{{
			CommentaryID: "comment-1",
			MatchID:      "match-1",
			Round:        1,
			ActorID:      domain.ActorPlantSystem,
			RoleID:       domain.RoleFinanceController,
			Visibility:   domain.CommentaryPublic,
			Body:         "Finance targets are now active.",
		}},
	}
	nextState := domain.MatchState{
		MatchID:      "match-1",
		ScenarioID:   "starter",
		CurrentRound: 2,
		Plant: domain.PlantState{
			Cash: 25,
		},
		History: domain.RoundHistory{
			RecentRounds: []domain.RoundRecord{round.Clone()},
		},
	}

	state, err := store.CommitRound(initial.MatchID, nextState, round)
	if err != nil {
		t.Fatalf("CommitRound() error = %v", err)
	}
	if state.CurrentRound != 2 {
		t.Fatalf("CurrentRound = %d, want 2", state.CurrentRound)
	}
	if len(state.History.RecentRounds) != 1 {
		t.Fatalf("History len = %d, want 1", len(state.History.RecentRounds))
	}

	storedRound, err := store.Round(initial.MatchID, 1)
	if err != nil {
		t.Fatalf("Round() error = %v", err)
	}
	if len(storedRound.Events) != 1 {
		t.Fatalf("stored events len = %d, want 1", len(storedRound.Events))
	}

	timeline, err := store.EventTimeline(initial.MatchID)
	if err != nil {
		t.Fatalf("EventTimeline() error = %v", err)
	}
	if len(timeline) != 1 {
		t.Fatalf("timeline len = %d, want 1", len(timeline))
	}

	commentary, err := store.Commentary(initial.MatchID)
	if err != nil {
		t.Fatalf("Commentary() error = %v", err)
	}
	if len(commentary) != 1 {
		t.Fatalf("commentary len = %d, want 1", len(commentary))
	}
}

func TestStoreCanResumeLaterFromPersistedCurrentState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resume.db")
	store, err := sqlite.NewStore(sqlite.Options{Path: path, RecentHistoryLimit: 2})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.CreateMatch(domain.MatchState{
		MatchID:      "match-2",
		ScenarioID:   "starter",
		CurrentRound: 3,
		Plant:        domain.PlantState{Cash: 30},
	}); err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := sqlite.NewStore(sqlite.Options{Path: path, RecentHistoryLimit: 2})
	if err != nil {
		t.Fatalf("NewStore() reopen error = %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	current, err := reopened.CurrentState("match-2")
	if err != nil {
		t.Fatalf("CurrentState() error = %v", err)
	}
	if got := current.CurrentRound; got != 3 {
		t.Fatalf("CurrentRound = %d, want 3", got)
	}
	if got := current.Plant.Cash; got != 30 {
		t.Fatalf("Cash = %d, want 30", got)
	}
}

func TestStorePersistsAICallRecordsWhenEnabled(t *testing.T) {
	store := newStore(t)
	initial := domain.MatchState{
		MatchID:      "match-3",
		ScenarioID:   "starter",
		CurrentRound: 1,
	}
	if err := store.CreateMatch(initial); err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}

	record := ports.AICallRecord{
		RoleID:       domain.RoleSalesManager,
		Round:        1,
		Attempt:      1,
		Provider:     "openrouter",
		Model:        "openai/gpt-5-mini",
		SystemPrompt: "system",
		UserPrompt:   "user",
		RawResponse:  "{}",
		Valid:        true,
	}
	if err := store.AppendAICall(initial.MatchID, record); err != nil {
		t.Fatalf("AppendAICall() error = %v", err)
	}

	records, err := store.AICallRecords(initial.MatchID)
	if err != nil {
		t.Fatalf("AICallRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("AICallRecords len = %d, want 1", len(records))
	}
	if got := records[0].Provider; got != "openrouter" {
		t.Fatalf("Provider = %q, want openrouter", got)
	}
}

func TestStorePersistsStateSnapshotsAcrossRounds(t *testing.T) {
	store := newStore(t)
	initial := domain.MatchState{
		MatchID:      "match-4",
		ScenarioID:   "starter",
		CurrentRound: 1,
		Plant:        domain.PlantState{Cash: 24},
	}
	if err := store.CreateMatch(initial); err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}

	nextState := initial.Clone()
	nextState.CurrentRound = 2
	nextState.Plant.Cash = 31
	if _, err := store.CommitRound(initial.MatchID, nextState, domain.RoundRecord{Round: 1}); err != nil {
		t.Fatalf("CommitRound() error = %v", err)
	}

	snapshots, err := store.StateSnapshots(initial.MatchID)
	if err != nil {
		t.Fatalf("StateSnapshots() error = %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("StateSnapshots len = %d, want 2", len(snapshots))
	}
	if got := snapshots[0].CurrentRound; got != 1 {
		t.Fatalf("snapshots[0].CurrentRound = %d, want 1", got)
	}
	if got := snapshots[1].Plant.Cash; got != 31 {
		t.Fatalf("snapshots[1].Plant.Cash = %d, want 31", got)
	}
}

func TestStoreCanSaveAndLoadNamedSlots(t *testing.T) {
	store := newStore(t)
	initial := domain.MatchState{
		MatchID:      "match-5",
		ScenarioID:   "starter",
		CurrentRound: 3,
		Plant:        domain.PlantState{Cash: 41},
	}
	if err := store.CreateMatch(initial); err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}

	summary, err := store.SaveSlot("starter-slot", initial.MatchID)
	if err != nil {
		t.Fatalf("SaveSlot() error = %v", err)
	}
	if summary.MatchID != initial.MatchID {
		t.Fatalf("SaveSlot() match id = %q, want %q", summary.MatchID, initial.MatchID)
	}

	slots, err := store.ListSaveSlots()
	if err != nil {
		t.Fatalf("ListSaveSlots() error = %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("ListSaveSlots len = %d, want 1", len(slots))
	}
	if slots[0].SlotName != "starter-slot" {
		t.Fatalf("ListSaveSlots slot name = %q, want starter-slot", slots[0].SlotName)
	}

	state, loaded, err := store.LoadSaveSlot("starter-slot")
	if err != nil {
		t.Fatalf("LoadSaveSlot() error = %v", err)
	}
	if state.MatchID != initial.MatchID {
		t.Fatalf("LoadSaveSlot() match id = %q, want %q", state.MatchID, initial.MatchID)
	}
	if loaded.CurrentRound != 3 {
		t.Fatalf("LoadSaveSlot() round = %d, want 3", loaded.CurrentRound)
	}
}

func newStore(t *testing.T) *sqlite.Store {
	t.Helper()

	store, err := sqlite.NewStore(sqlite.Options{
		Path:               filepath.Join(t.TempDir(), "match.db"),
		RecentHistoryLimit: 2,
	})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}
