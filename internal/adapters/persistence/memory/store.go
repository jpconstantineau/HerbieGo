package memory

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

var ErrMatchNotFound = errors.New("memory store: match not found")

type Options struct {
	// RecentHistoryLimit bounds MatchState.History without trimming the full event or commentary timeline.
	RecentHistoryLimit int
}

type Store struct {
	mu           sync.RWMutex
	historyLimit int
	matches      map[domain.MatchID]storedMatch
}

type storedMatch struct {
	state      domain.MatchState
	rounds     []domain.RoundRecord
	events     []domain.RoundEvent
	commentary []domain.CommentaryRecord
}

func NewStore(options Options) *Store {
	return &Store{
		historyLimit: normalizedHistoryLimit(options.RecentHistoryLimit),
		matches:      make(map[domain.MatchID]storedMatch),
	}
}

func (s *Store) CreateMatch(initial domain.MatchState) error {
	if initial.MatchID == "" {
		return errors.New("memory store: match id must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.matches[initial.MatchID]; exists {
		return fmt.Errorf("memory store: match %q already exists", initial.MatchID)
	}

	state := initial.Clone()
	history := state.History.Recent(s.historyLimit)
	rounds := history.RecentRounds
	state.History = history

	s.matches[initial.MatchID] = storedMatch{
		state:      state,
		rounds:     cloneRoundRecords(rounds),
		events:     flattenEvents(rounds),
		commentary: flattenCommentary(rounds),
	}

	return nil
}

func (s *Store) CurrentState(matchID domain.MatchID) (domain.MatchState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	match, ok := s.matches[matchID]
	if !ok {
		return domain.MatchState{}, ErrMatchNotFound
	}

	return match.state.Clone(), nil
}

func (s *Store) CommitRound(matchID domain.MatchID, nextState domain.MatchState, round domain.RoundRecord) (domain.MatchState, error) {
	if matchID == "" {
		return domain.MatchState{}, errors.New("memory store: match id must not be empty")
	}
	if nextState.MatchID != matchID {
		return domain.MatchState{}, fmt.Errorf("memory store: next state match id %q does not match %q", nextState.MatchID, matchID)
	}
	if round.Round <= 0 {
		return domain.MatchState{}, fmt.Errorf("memory store: round %d must be positive", round.Round)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	match, ok := s.matches[matchID]
	if !ok {
		return domain.MatchState{}, ErrMatchNotFound
	}

	if len(match.rounds) > 0 {
		last := match.rounds[len(match.rounds)-1].Round
		if round.Round <= last {
			return domain.MatchState{}, fmt.Errorf("memory store: round %d must be greater than last committed round %d", round.Round, last)
		}
	}

	committedRound := round.Clone()
	match.rounds = append(match.rounds, committedRound)
	match.events = append(match.events, flattenEvents([]domain.RoundRecord{committedRound})...)
	match.commentary = append(match.commentary, flattenCommentary([]domain.RoundRecord{committedRound})...)

	match.state = nextState.Clone()
	match.state.History = domain.RoundHistory{
		RecentRounds: cloneRoundRecords(tail(match.rounds, s.historyLimit)),
	}

	s.matches[matchID] = match
	return match.state.Clone(), nil
}

func (s *Store) Round(matchID domain.MatchID, roundNumber domain.RoundNumber) (domain.RoundRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	match, ok := s.matches[matchID]
	if !ok {
		return domain.RoundRecord{}, ErrMatchNotFound
	}

	for _, round := range match.rounds {
		if round.Round == roundNumber {
			return round.Clone(), nil
		}
	}

	return domain.RoundRecord{}, fmt.Errorf("memory store: round %d not found", roundNumber)
}

func (s *Store) EventTimeline(matchID domain.MatchID) ([]domain.RoundEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	match, ok := s.matches[matchID]
	if !ok {
		return nil, ErrMatchNotFound
	}

	return cloneEvents(match.events), nil
}

// Commentary returns the full append-only public commentary timeline for the match.
func (s *Store) Commentary(matchID domain.MatchID) ([]domain.CommentaryRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	match, ok := s.matches[matchID]
	if !ok {
		return nil, ErrMatchNotFound
	}

	return slices.Clone(match.commentary), nil
}

func normalizedHistoryLimit(limit int) int {
	if limit <= 0 {
		return 10
	}

	return limit
}

func cloneRoundRecords(rounds []domain.RoundRecord) []domain.RoundRecord {
	if rounds == nil {
		return nil
	}

	cloned := make([]domain.RoundRecord, len(rounds))
	for i := range rounds {
		cloned[i] = rounds[i].Clone()
	}

	return cloned
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

func flattenEvents(rounds []domain.RoundRecord) []domain.RoundEvent {
	var events []domain.RoundEvent
	for _, round := range rounds {
		events = append(events, cloneEvents(round.Events)...)
	}

	return events
}

func flattenCommentary(rounds []domain.RoundRecord) []domain.CommentaryRecord {
	var commentary []domain.CommentaryRecord
	for _, round := range rounds {
		commentary = append(commentary, slices.Clone(round.Commentary)...)
	}

	return commentary
}

func tail[T any](items []T, limit int) []T {
	if limit <= 0 || len(items) <= limit {
		return items
	}

	return items[len(items)-limit:]
}
