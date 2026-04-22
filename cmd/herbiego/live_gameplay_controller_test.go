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

	states := collectStatesUntilPhase(t, controller.Updates(), domain.RoundPhaseRevealed)
	if err := <-result; err != nil {
		t.Fatalf("runner.Play() error = %v", err)
	}

	if got := states[0].RoundFlow.Phase; got != domain.RoundPhaseCollecting {
		t.Fatalf("first phase = %q, want collecting", got)
	}

	resolvingIndex := -1
	revealedIndex := -1
	for index, state := range states {
		switch state.RoundFlow.Phase {
		case domain.RoundPhaseResolving:
			if resolvingIndex < 0 {
				resolvingIndex = index
			}
		case domain.RoundPhaseRevealed:
			if revealedIndex < 0 {
				revealedIndex = index
			}
		}
	}
	if resolvingIndex < 0 {
		t.Fatalf("states never reached resolving: %+v", states)
	}
	if revealedIndex < 0 {
		t.Fatalf("states never reached revealed: %+v", states)
	}
	if resolvingIndex > revealedIndex {
		t.Fatalf("resolving phase arrived after revealed phase: %+v", states)
	}
	if got := states[revealedIndex].History.RecentRounds; len(got) != 1 {
		t.Fatalf("revealed history len = %d, want 1", len(got))
	}

	commentary := states[revealedIndex].History.RecentRounds[0].Commentary
	if len(commentary) == 0 || commentary[0].RoleID != domain.RoleProcurementManager {
		t.Fatalf("revealed commentary = %#v, want procurement commentary first", commentary)
	}
	if !strings.Contains(commentary[0].Body, "bottleneck") {
		t.Fatalf("commentary body = %q, want submitted human commentary", commentary[0].Body)
	}
}

func collectStatesUntilPhase(t *testing.T, updates <-chan domain.MatchState, target domain.RoundPhase) []domain.MatchState {
	t.Helper()

	states := make([]domain.MatchState, 0, 6)
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case state, ok := <-updates:
			if !ok {
				t.Fatalf("updates closed before phase %q; saw %d states", target, len(states))
			}
			states = append(states, state.Clone())
			if state.RoundFlow.Phase == target {
				return states
			}
		case <-timeout.C:
			t.Fatalf("timed out waiting for phase %q after collecting %d states", target, len(states))
		}
	}
}
