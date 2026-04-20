package ports

import "github.com/jpconstantineau/herbiego/internal/domain"

// MatchStateStore persists and serves canonical match state for the round being collected,
// plus append-only event and commentary timelines.
type MatchStateStore interface {
	CreateMatch(initial domain.MatchState) error
	CurrentState(matchID domain.MatchID) (domain.MatchState, error)
	CommitRound(matchID domain.MatchID, nextState domain.MatchState, round domain.RoundRecord) (domain.MatchState, error)
	Round(matchID domain.MatchID, round domain.RoundNumber) (domain.RoundRecord, error)
	EventTimeline(matchID domain.MatchID) ([]domain.RoundEvent, error)
	Commentary(matchID domain.MatchID) ([]domain.CommentaryRecord, error)
}
