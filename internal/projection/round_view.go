package projection

import (
	"github.com/jpconstantineau/herbiego/internal/domain"
)

// BuildRoundView projects the canonical state and recent append-only history into a player-facing view.
func BuildRoundView(state domain.MatchState, viewerRoleID domain.RoleID) domain.RoundView {
	recentRounds := buildRecentRounds(state.History.Recent(10))
	recentEvents := make([]domain.RoundEvent, 0)
	recentCommentary := make([]domain.CommentaryRecord, 0)
	for _, round := range recentRounds {
		recentEvents = append(recentEvents, cloneEvents(round.Events)...)
		recentCommentary = append(recentCommentary, cloneCommentary(round.Commentary)...)
	}

	return domain.RoundView{
		MatchID:          state.MatchID,
		Round:            state.CurrentRound,
		ViewerRoleID:     viewerRoleID,
		RoundFlow:        state.RoundFlow.Clone(),
		Plant:            state.Plant.Clone(),
		Customers:        cloneCustomers(state.Customers),
		ActiveTargets:    state.ActiveTargets,
		Metrics:          state.Metrics,
		RecentRounds:     recentRounds,
		RecentEvents:     recentEvents,
		RecentCommentary: recentCommentary,
	}
}

func buildRecentRounds(history domain.RoundHistory) []domain.RoundHistoryEntry {
	if len(history.RecentRounds) == 0 {
		return nil
	}

	entries := make([]domain.RoundHistoryEntry, len(history.RecentRounds))
	for i, round := range history.RecentRounds {
		entries[i] = domain.RoundHistoryEntry{
			Round:      round.Round,
			Events:     cloneEvents(round.Events),
			Commentary: cloneCommentary(round.Commentary),
			Summary: domain.RoundSummary{
				Metrics:         round.Metrics,
				EventCount:      len(round.Events),
				CommentaryCount: len(round.Commentary),
				ActionCount:     len(round.Actions),
			},
		}
	}

	return entries
}

func cloneEvents(events []domain.RoundEvent) []domain.RoundEvent {
	if events == nil {
		return nil
	}

	cloned := make([]domain.RoundEvent, len(events))
	for i := range events {
		cloned[i] = events[i].Clone()
	}

	return cloned
}

func cloneCommentary(commentary []domain.CommentaryRecord) []domain.CommentaryRecord {
	if commentary == nil {
		return nil
	}

	cloned := make([]domain.CommentaryRecord, len(commentary))
	copy(cloned, commentary)
	return cloned
}

func cloneCustomers(customers []domain.CustomerState) []domain.CustomerState {
	if customers == nil {
		return nil
	}

	cloned := make([]domain.CustomerState, len(customers))
	for i := range customers {
		cloned[i] = customers[i].Clone()
	}

	return cloned
}
