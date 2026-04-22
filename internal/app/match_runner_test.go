package app_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestMatchRunnerPlaysMultipleResolvedRounds(t *testing.T) {
	starter := scenario.Starter()
	initial := starter.InitialState("starter-match", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", Provider: "ollama", ModelName: "gemma4:e4b"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", Provider: "openrouter", ModelName: "openai/gpt-5-mini"},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", Provider: "openai", ModelName: "gpt-5-mini"},
	})
	initial.RoundFlow.AIRevealDelaySeconds = 15

	phaseCounts := map[domain.RoundPhase]int{}
	runner := app.MatchRunner{
		Collector: app.RoundCollector{
			Players: scriptedPlayers(),
		},
		Resolver: engine.NewResolver(starter.ResolverOptions()),
		Random:   seeded.New(1),
		OnState: func(state domain.MatchState) {
			phaseCounts[state.RoundFlow.Phase]++
		},
	}

	final, results, err := runner.Play(context.Background(), initial, 3)
	if err != nil {
		t.Fatalf("Play() error = %v", err)
	}

	if got := len(results); got != 3 {
		t.Fatalf("len(results) = %d, want 3", got)
	}
	if got := final.CurrentRound; got != 4 {
		t.Fatalf("final.CurrentRound = %d, want 4", got)
	}
	if got := len(final.History.RecentRounds); got != 3 {
		t.Fatalf("history rounds = %d, want 3", got)
	}
	if final.RoundFlow.Phase != domain.RoundPhaseCollecting {
		t.Fatalf("final phase = %q, want collecting", final.RoundFlow.Phase)
	}
	if got := len(final.RoundFlow.WaitingOnRoles); got != 4 {
		t.Fatalf("waiting roles = %d, want 4", got)
	}
	if phaseCounts[domain.RoundPhaseCollecting] != 3 {
		t.Fatalf("collecting states emitted = %d, want 3", phaseCounts[domain.RoundPhaseCollecting])
	}
	if phaseCounts[domain.RoundPhaseResolving] != 3 {
		t.Fatalf("resolving states emitted = %d, want 3", phaseCounts[domain.RoundPhaseResolving])
	}
	if phaseCounts[domain.RoundPhaseRevealed] != 3 {
		t.Fatalf("revealed states emitted = %d, want 3", phaseCounts[domain.RoundPhaseRevealed])
	}

	for index, result := range results {
		if got := len(result.Round.Actions); got != 4 {
			t.Fatalf("round %d actions = %d, want 4", index+1, got)
		}
		if got := len(result.Round.Commentary); got != 4 {
			t.Fatalf("round %d commentary = %d, want 4", index+1, got)
		}
		if len(result.Round.Timeline) == 0 {
			t.Fatalf("round %d timeline is empty", index+1)
		}
	}
}

func scriptedPlayers() map[domain.RoleID]ports.Player {
	return map[domain.RoleID]ports.Player{
		domain.RoleProcurementManager: scriptPlayer(func(request ports.RoundRequest) domain.ActionSubmission {
			return domain.ActionSubmission{
				Action: domain.RoleAction{Procurement: &domain.ProcurementAction{}},
				Commentary: domain.CommentaryRecord{
					Body: fmt.Sprintf("Round %d procurement protects cash for the bottleneck.", request.RoleView.Round),
				},
			}
		}),
		domain.RoleProductionManager: scriptPlayer(func(request ports.RoundRequest) domain.ActionSubmission {
			return domain.ActionSubmission{
				Action: domain.RoleAction{
					Production: &domain.ProductionAction{
						CapacityAllocation: []domain.CapacityAllocation{
							{WorkstationID: "assembly", ProductID: "pump", Capacity: 2},
						},
					},
				},
				Commentary: domain.CommentaryRecord{
					Body: fmt.Sprintf("Round %d production favors throughput over local utilization.", request.RoleView.Round),
				},
			}
		}),
		domain.RoleSalesManager: scriptPlayer(func(request ports.RoundRequest) domain.ActionSubmission {
			return domain.ActionSubmission{
				Action: domain.RoleAction{
					Sales: &domain.SalesAction{
						ProductOffers: []domain.ProductOffer{
							{ProductID: "pump", UnitPrice: 14},
							{ProductID: "valve", UnitPrice: 9},
						},
					},
				},
				Commentary: domain.CommentaryRecord{
					Body: fmt.Sprintf("Round %d sales holds price while backlog is visible.", request.RoleView.Round),
				},
			}
		}),
		domain.RoleFinanceController: scriptPlayer(func(request ports.RoundRequest) domain.ActionSubmission {
			targets := request.RoleView.ActiveTargets
			targets.EffectiveRound = request.RoleView.Round + 1
			return domain.ActionSubmission{
				Action: domain.RoleAction{
					Finance: &domain.FinanceAction{NextRoundTargets: targets},
				},
				Commentary: domain.CommentaryRecord{
					Body: fmt.Sprintf("Round %d finance keeps targets stable to compare tradeoffs.", request.RoleView.Round),
				},
			}
		}),
	}
}

type scriptPlayer func(ports.RoundRequest) domain.ActionSubmission

func (p scriptPlayer) SubmitRound(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
	return p(request), nil
}

func (scriptPlayer) RecoverFromNonResponse(_ context.Context, request ports.RoundRequest, cause error) (domain.ActionSubmission, error) {
	return domain.ActionSubmission{}, fmt.Errorf("role %q did not respond: %w", request.Assignment.RoleID, cause)
}
