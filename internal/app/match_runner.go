package app

import (
	"context"
	"fmt"
	"log/slog"

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
	Store     ports.MatchStateStore
	OnState   func(domain.MatchState)
	OnRound   func(engine.Result)
	Logger    *slog.Logger
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

	state := initial.Clone()
	previous := previousAcceptedActions(state)
	results := make([]engine.Result, 0, rounds)
	logger := loggerOrDiscard(r.Logger).With(
		"component", "match_runner",
		"match_id", state.MatchID,
		"scenario_id", state.ScenarioID,
	)
	logger.Info("match play started", "rounds_requested", rounds, "starting_round", state.CurrentRound)

	if r.Store != nil {
		if err := r.Store.CreateMatch(state); err != nil {
			logger.Error("match store create failed", "error", err)
			return domain.MatchState{}, nil, fmt.Errorf("app: create match in store: %w", err)
		}
		logger.Info("match store initialized")
	}

	state = prepareCollectionState(state)

	for range rounds {
		roundLogger := logger.With("round", state.CurrentRound)
		roundLogger.Info("round started")
		r.emitState(state)

		collector := r.Collector
		collector.OnRoundFlow = func(flow domain.RoundFlowState) {
			progress := state.Clone()
			progress.RoundFlow = flow.Clone()
			r.emitState(progress)
		}

		actions, err := collector.Collect(ctx, state, previous)
		if err != nil {
			roundLogger.Error("round collection failed", "error", err)
			return state, results, err
		}

		resolving := state.Clone()
		resolving.RoundFlow = roundFlowFor(state.Roles, actions, domain.RoundPhaseResolving, state.RoundFlow.AIRevealDelaySeconds)
		r.emitState(resolving)

		result, err := r.Resolver.ResolveRound(state, actions, r.Random)
		if err != nil {
			roundLogger.Error("round resolution failed", "error", err)
			return state, results, err
		}

		revealed := result.NextState.Clone()
		revealed.RoundFlow = roundFlowFor(state.Roles, actions, domain.RoundPhaseRevealed, state.RoundFlow.AIRevealDelaySeconds)
		result.NextState = revealed.Clone()
		if r.Store != nil {
			committed, err := r.Store.CommitRound(state.MatchID, result.NextState, result.Round)
			if err != nil {
				roundLogger.Error("round persistence failed", "error", err)
				return state, results, fmt.Errorf("app: commit round %d: %w", result.Round.Round, err)
			}
			revealed = committed.Clone()
			result.NextState = committed.Clone()
			roundLogger.Info("round persisted")
		}
		results = append(results, result)
		for _, action := range result.Round.Actions {
			previous[action.RoleID] = action.Clone()
		}
		r.emitRound(result)
		r.emitState(revealed)
		roundLogger.Info(
			"round completed",
			"action_count", len(result.Round.Actions),
			"next_round", revealed.CurrentRound,
			"cash", revealed.Plant.Cash,
			"debt", revealed.Plant.Debt,
			"backlog_count", len(revealed.Plant.Backlog),
			"round_profit", revealed.Metrics.RoundProfit,
		)

		state = prepareCollectionState(revealed)
	}

	logger.Info("match play completed", "final_round", state.CurrentRound, "resolved_rounds", len(results))
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
