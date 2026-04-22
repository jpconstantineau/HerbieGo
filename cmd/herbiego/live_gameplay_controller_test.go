package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jpconstantineau/herbiego/internal/adapters/player/human"
	"github.com/jpconstantineau/herbiego/internal/adapters/player/llm"
	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestLiveGameplayControllerDeliversHumanSubmissionAndStreamsRoundPhases(t *testing.T) {
	starter := scenario.Starter()
	initial := starter.InitialState("starter-match", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", Provider: "ollama", ModelName: "gemma"},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", Provider: "ollama", ModelName: "gemma"},
	})

	controller := newLiveGameplayController(initial)
	now := time.Date(2026, time.April, 21, 20, 0, 0, 0, time.UTC)
	runner := app.MatchRunner{
		Collector: app.RoundCollector{
			Now: func() time.Time { return now },
			Players: map[domain.RoleID]ports.Player{
				domain.RoleProcurementManager: human.New(controller.SubmitRound),
				domain.RoleProductionManager: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
					return domain.ActionSubmission{
						Action: domain.RoleAction{
							Production: &domain.ProductionAction{
								CapacityAllocation: []domain.CapacityAllocation{
									{WorkstationID: "assembly", ProductID: "pump", Capacity: 2},
								},
							},
						},
						Commentary: domain.CommentaryRecord{Body: "Clearing inherited pump WIP first."},
					}, nil
				}),
				domain.RoleSalesManager: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
					return domain.ActionSubmission{
						Action: domain.RoleAction{
							Sales: &domain.SalesAction{
								ProductOffers: []domain.ProductOffer{
									{ProductID: "pump", UnitPrice: 14},
									{ProductID: "valve", UnitPrice: 9},
								},
							},
						},
						Commentary: domain.CommentaryRecord{Body: "Holding price to ship starter backlog."},
					}, nil
				}),
				domain.RoleFinanceController: llm.New(func(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
					return domain.ActionSubmission{
						Action: domain.RoleAction{
							Finance: &domain.FinanceAction{
								NextRoundTargets: domain.BudgetTargets{
									EffectiveRound:        2,
									ProcurementBudget:     18,
									ProductionSpendBudget: 12,
									RevenueTarget:         34,
									CashFloorTarget:       10,
									DebtCeilingTarget:     15,
								},
							},
						},
						Commentary: domain.CommentaryRecord{Body: "Keeping spend tight during the first reveal."},
					}, nil
				}),
			},
		},
		Resolver: engine.NewResolver(starter.ResolverOptions()),
		Random:   seeded.New(1),
		OnState:  controller.Publish,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() {
		defer controller.Close()
		_, _, err := runner.Play(ctx, initial, 1)
		result <- err
	}()

	controller.Submit(domain.ActionSubmission{
		MatchID: initial.MatchID,
		Round:   initial.CurrentRound,
		RoleID:  domain.RoleProcurementManager,
		Action: domain.RoleAction{
			Procurement: &domain.ProcurementAction{
				Orders: []domain.PurchaseOrderIntent{{PartID: "housing", SupplierID: "forgeco", Quantity: 1}},
			},
		},
		Commentary: domain.CommentaryRecord{Body: "Buying only what the bottleneck can absorb."},
	})

	states := collectStates(t, controller.Updates(), 3)
	if err := <-result; err != nil {
		t.Fatalf("runner.Play() error = %v", err)
	}

	if got := states[0].RoundFlow.Phase; got != domain.RoundPhaseCollecting {
		t.Fatalf("first phase = %q, want collecting", got)
	}
	if got := states[1].RoundFlow.Phase; got != domain.RoundPhaseResolving {
		t.Fatalf("second phase = %q, want resolving", got)
	}
	if got := states[2].RoundFlow.Phase; got != domain.RoundPhaseRevealed {
		t.Fatalf("third phase = %q, want revealed", got)
	}
	if got := states[2].History.RecentRounds; len(got) != 1 {
		t.Fatalf("revealed history len = %d, want 1", len(got))
	}

	commentary := states[2].History.RecentRounds[0].Commentary
	if len(commentary) == 0 || commentary[0].RoleID != domain.RoleProcurementManager {
		t.Fatalf("revealed commentary = %#v, want procurement commentary first", commentary)
	}
	if !strings.Contains(commentary[0].Body, "bottleneck") {
		t.Fatalf("commentary body = %q, want submitted human commentary", commentary[0].Body)
	}
}

func collectStates(t *testing.T, updates <-chan domain.MatchState, want int) []domain.MatchState {
	t.Helper()

	states := make([]domain.MatchState, 0, want)
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()

	for len(states) < want {
		select {
		case state, ok := <-updates:
			if !ok {
				t.Fatalf("updates closed after %d states, want %d", len(states), want)
			}
			states = append(states, state.Clone())
		case <-timeout.C:
			t.Fatalf("timed out after collecting %d/%d states", len(states), want)
		}
	}

	return states
}
