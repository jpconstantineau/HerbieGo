package memory_test

import (
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/persistence/memory"
	"github.com/jpconstantineau/herbiego/internal/domain"
)

func TestStoreCommitRoundExposesCurrentStateAndTimelines(t *testing.T) {
	store := memory.NewStore(memory.Options{RecentHistoryLimit: 2})
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
		Events: []domain.RoundEvent{
			{
				EventID: "event-1",
				MatchID: "match-1",
				Round:   1,
				Type:    domain.EventBudgetActivated,
				ActorID: domain.ActorPlantSystem,
				Summary: "Activated round targets",
				Payload: map[string]any{"budget": 10},
			},
		},
		Commentary: []domain.CommentaryRecord{
			{
				CommentaryID: "comment-1",
				MatchID:      "match-1",
				Round:        1,
				ActorID:      domain.ActorPlantSystem,
				RoleID:       domain.RoleFinanceController,
				Visibility:   domain.CommentaryPublic,
				Body:         "Finance targets are now active.",
			},
		},
	}
	nextState := domain.MatchState{
		MatchID:      "match-1",
		ScenarioID:   "starter",
		CurrentRound: 2,
		Plant: domain.PlantState{
			Cash: 25,
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
	if state.History.RecentRounds[0].Round != 1 {
		t.Fatalf("History round = %d, want 1", state.History.RecentRounds[0].Round)
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
	if got := timeline[0].Payload["budget"]; got != 10 {
		t.Fatalf("timeline payload budget = %v, want 10", got)
	}

	commentary, err := store.Commentary(initial.MatchID)
	if err != nil {
		t.Fatalf("Commentary() error = %v", err)
	}
	if len(commentary) != 1 {
		t.Fatalf("commentary len = %d, want 1", len(commentary))
	}
}

func TestStoreTrimsCurrentHistoryWindowButKeepsAppendOnlyTimeline(t *testing.T) {
	store := memory.NewStore(memory.Options{RecentHistoryLimit: 2})
	initial := domain.MatchState{
		MatchID:      "match-1",
		ScenarioID:   "starter",
		CurrentRound: 1,
	}

	if err := store.CreateMatch(initial); err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}

	for round := range 3 {
		roundNumber := domain.RoundNumber(round + 1)
		_, err := store.CommitRound(initial.MatchID, domain.MatchState{
			MatchID:      initial.MatchID,
			ScenarioID:   initial.ScenarioID,
			CurrentRound: roundNumber + 1,
		}, domain.RoundRecord{
			Round: roundNumber,
			Events: []domain.RoundEvent{
				{
					EventID: domain.EventID("event-" + string(rune('1'+round))),
					MatchID: initial.MatchID,
					Round:   roundNumber,
					Type:    domain.EventMetricSnapshot,
					ActorID: domain.ActorPlantSystem,
				},
			},
		})
		if err != nil {
			t.Fatalf("CommitRound(%d) error = %v", roundNumber, err)
		}
	}

	state, err := store.CurrentState(initial.MatchID)
	if err != nil {
		t.Fatalf("CurrentState() error = %v", err)
	}

	if len(state.History.RecentRounds) != 2 {
		t.Fatalf("History len = %d, want 2", len(state.History.RecentRounds))
	}
	if state.History.RecentRounds[0].Round != 2 || state.History.RecentRounds[1].Round != 3 {
		t.Fatalf("History rounds = %#v, want [2 3]", []domain.RoundNumber{
			state.History.RecentRounds[0].Round,
			state.History.RecentRounds[1].Round,
		})
	}

	timeline, err := store.EventTimeline(initial.MatchID)
	if err != nil {
		t.Fatalf("EventTimeline() error = %v", err)
	}
	if len(timeline) != 3 {
		t.Fatalf("timeline len = %d, want 3", len(timeline))
	}
}
