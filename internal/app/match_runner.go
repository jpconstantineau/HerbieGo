package app

import (
	"context"
	"fmt"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

// MatchRunner drives the MVP match loop by collecting every role's simultaneous
// action, resolving the round, and carrying the resulting state forward.
type MatchRunner struct {
	Collector RoundCollector
	Resolver  *engine.Resolver
	Random    ports.RandomSource
	OnState   func(domain.MatchState)
	OnRound   func(engine.Result)
}

// Play advances the match for the requested number of rounds.
func (r MatchRunner) Play(ctx context.Context, initial domain.MatchState, rounds int) (domain.MatchState, []engine.Result, error) {
	if rounds <= 0 {
		return domain.MatchState{}, nil, fmt.Errorf("app: match rounds must be positive")
	}
	if r.Resolver == nil {
		return domain.MatchState{}, nil, fmt.Errorf("app: match resolver is not configured")
	}
	if r.Random == nil {
		return domain.MatchState{}, nil, fmt.Errorf("app: match random source is not configured")
	}

	state := prepareCollectionState(initial.Clone())
	previous := previousAcceptedActions(state)
	results := make([]engine.Result, 0, rounds)

	for range rounds {
		r.emitState(state)

		collector := r.Collector
		collector.OnRoundFlow = func(flow domain.RoundFlowState) {
			progress := state.Clone()
			progress.RoundFlow = flow.Clone()
			r.emitState(progress)
		}

		actions, err := collector.Collect(ctx, state, previous)
		if err != nil {
			return state, results, err
		}

		resolving := state.Clone()
		resolving.RoundFlow = roundFlowFor(state.Roles, actions, domain.RoundPhaseResolving, state.RoundFlow.AIRevealDelaySeconds)
		r.emitState(resolving)

		result, err := r.Resolver.ResolveRound(state, actions, r.Random)
		if err != nil {
			return state, results, err
		}

		revealed := result.NextState.Clone()
		revealed.RoundFlow = roundFlowFor(state.Roles, actions, domain.RoundPhaseRevealed, state.RoundFlow.AIRevealDelaySeconds)
		result.NextState = revealed.Clone()
		results = append(results, result)
		for _, action := range result.Round.Actions {
			previous[action.RoleID] = action.Clone()
		}
		r.emitRound(result)
		r.emitState(revealed)

		state = prepareCollectionState(revealed)
	}

	return state.Clone(), results, nil
}

func (r MatchRunner) emitState(state domain.MatchState) {
	if r.OnState != nil {
		r.OnState(state.Clone())
	}
}

func (r MatchRunner) emitRound(result engine.Result) {
	if r.OnRound != nil {
		r.OnRound(engine.Result{
			NextState: result.NextState.Clone(),
			Round:     result.Round.Clone(),
		})
	}
}

func prepareCollectionState(state domain.MatchState) domain.MatchState {
	state.RoundFlow = domain.RoundFlowState{
		Phase:                domain.RoundPhaseCollecting,
		WaitingOnRoles:       roleIDs(state.Roles),
		ProviderWaitingRoles: nil,
		AIRevealDelaySeconds: state.RoundFlow.AIRevealDelaySeconds,
	}
	return state
}

func roundFlowFor(assignments []domain.RoleAssignment, actions []domain.ActionSubmission, phase domain.RoundPhase, revealDelaySeconds int) domain.RoundFlowState {
	submitted := make([]domain.RoleID, 0, len(actions))
	submittedSet := make(map[domain.RoleID]bool, len(actions))
	for _, action := range actions {
		submitted = append(submitted, action.RoleID)
		submittedSet[action.RoleID] = true
	}

	waiting := make([]domain.RoleID, 0, max(len(assignments)-len(submitted), 0))
	for _, assignment := range assignments {
		if !submittedSet[assignment.RoleID] {
			waiting = append(waiting, assignment.RoleID)
		}
	}

	return domain.RoundFlowState{
		Phase:                phase,
		SubmittedRoles:       submitted,
		WaitingOnRoles:       waiting,
		ProviderWaitingRoles: nil,
		AIRevealDelaySeconds: revealDelaySeconds,
	}
}

func roleIDs(assignments []domain.RoleAssignment) []domain.RoleID {
	ids := make([]domain.RoleID, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.RoleID)
	}
	return ids
}

func previousAcceptedActions(state domain.MatchState) map[domain.RoleID]domain.ActionSubmission {
	previous := make(map[domain.RoleID]domain.ActionSubmission, len(state.Roles))
	for _, round := range state.History.RecentRounds {
		for _, action := range round.Actions {
			previous[action.RoleID] = action.Clone()
		}
	}
	return previous
}
