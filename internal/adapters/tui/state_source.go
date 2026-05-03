package tui

import (
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

// StateSource exposes the current match snapshot plus future updates from the engine/store layer.
type StateSource interface {
	Snapshot() domain.MatchState
	Updates() <-chan domain.MatchState
}

// StateSnapshotSource exposes canonical match-state checkpoints for replay inspection.
type StateSnapshotSource interface {
	StateSnapshots() []domain.MatchState
}

// DebugSource provides read access to recorded AI provider request/response exchanges.
type DebugSource interface {
	Records() []ports.AICallRecord
}

type staticStateSource struct {
	state     domain.MatchState
	snapshots []domain.MatchState
}

func newStaticStateSource(state domain.MatchState) staticStateSource {
	return newReplayStateSource(state, []domain.MatchState{state})
}

func newReplayStateSource(state domain.MatchState, snapshots []domain.MatchState) staticStateSource {
	clonedSnapshots := make([]domain.MatchState, 0, len(snapshots))
	for _, snapshot := range snapshots {
		clonedSnapshots = append(clonedSnapshots, snapshot.Clone())
	}
	if len(clonedSnapshots) == 0 {
		clonedSnapshots = append(clonedSnapshots, state.Clone())
	}
	return staticStateSource{
		state:     state.Clone(),
		snapshots: clonedSnapshots,
	}
}

func (s staticStateSource) Snapshot() domain.MatchState {
	return s.state.Clone()
}

func (staticStateSource) Updates() <-chan domain.MatchState {
	return nil
}

func (s staticStateSource) StateSnapshots() []domain.MatchState {
	cloned := make([]domain.MatchState, len(s.snapshots))
	for i := range s.snapshots {
		cloned[i] = s.snapshots[i].Clone()
	}
	return cloned
}

type staticDebugSource struct {
	records []ports.AICallRecord
}

func newStaticDebugSource(records []ports.AICallRecord) staticDebugSource {
	cloned := make([]ports.AICallRecord, len(records))
	copy(cloned, records)
	return staticDebugSource{records: cloned}
}

func (s staticDebugSource) Records() []ports.AICallRecord {
	cloned := make([]ports.AICallRecord, len(s.records))
	copy(cloned, s.records)
	return cloned
}
