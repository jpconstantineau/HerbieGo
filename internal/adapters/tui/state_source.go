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

// DebugSource provides read access to recorded AI provider request/response exchanges.
type DebugSource interface {
	Records() []ports.AICallRecord
}

type staticStateSource struct {
	state domain.MatchState
}

func newStaticStateSource(state domain.MatchState) staticStateSource {
	return staticStateSource{state: state.Clone()}
}

func (s staticStateSource) Snapshot() domain.MatchState {
	return s.state.Clone()
}

func (staticStateSource) Updates() <-chan domain.MatchState {
	return nil
}
