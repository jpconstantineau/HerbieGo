package app_test

import (
	"context"
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

func TestStarterMVPFlowCollectsMixedPlayersAndResolvesRound(t *testing.T) {
	starter := scenario.Starter()
	state := starter.InitialState("starter-match", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "procurement-player", IsHuman: true},
		{RoleID: domain.RoleProductionManager, PlayerID: "production-player", Provider: "ollama", ModelName: "gemma4:e4b"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales-player", IsHuman: true},
		{RoleID: domain.RoleFinanceController, PlayerID: "finance-player", Provider: "openrouter", ModelName: "openai/gpt-5-mini"},
	})
	now := time.Date(2026, time.April, 21, 19, 0, 0, 0, time.UTC)

	collector := app.RoundCollector{
		Now: func() time.Time { return now },
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProcurementManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Procurement: &domain.ProcurementAction{},
					},
					Commentary: domain.CommentaryRecord{Body: "Holding cash until the bottleneck needs more parts."},
				}, nil
			}),
			domain.RoleProductionManager: llm.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Production: &domain.ProductionAction{
							CapacityAllocation: []domain.CapacityAllocation{
								{WorkstationID: "assembly", ProductID: "pump", Capacity: 2},
							},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Finishing inherited pump WIP before releasing more work."},
				}, nil
			}),
			domain.RoleSalesManager: human.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Sales: &domain.SalesAction{
							ProductOffers: []domain.ProductOffer{
								{ProductID: "pump", UnitPrice: 14},
								{ProductID: "valve", UnitPrice: 9},
							},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Holding starter prices while we clear inherited backlog."},
				}, nil
			}),
			domain.RoleFinanceController: llm.New(func(_ context.Context, _ ports.RoundRequest) (domain.ActionSubmission, error) {
				return domain.ActionSubmission{
					Action: domain.RoleAction{
						Finance: &domain.FinanceAction{
							NextRoundTargets: domain.BudgetTargets{
								ProcurementBudget:     16,
								ProductionSpendBudget: 12,
								RevenueTarget:         34,
								CashFloorTarget:       10,
								DebtCeilingTarget:     15,
							},
						},
					},
					Commentary: domain.CommentaryRecord{Body: "Keeping spending tight while backlog is still monetized."},
				}, nil
			}),
		},
	}

	actions, err := collector.Collect(context.Background(), state, nil)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if got := len(actions); got != 4 {
		t.Fatalf("len(actions) = %d, want 4", got)
	}
	for _, action := range actions {
		if action.MatchID != state.MatchID {
			t.Fatalf("action.MatchID = %q, want %q", action.MatchID, state.MatchID)
		}
		if action.Round != state.CurrentRound {
			t.Fatalf("action.Round = %d, want %d", action.Round, state.CurrentRound)
		}
		if action.SubmittedAt != now {
			t.Fatalf("action.SubmittedAt = %s, want %s", action.SubmittedAt, now)
		}
		if action.ActionID == "" {
			t.Fatalf("action for %q has empty ActionID", action.RoleID)
		}
	}

	resolver := engine.NewResolver(starter.ResolverOptions())
	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.NextState.CurrentRound; got != 2 {
		t.Fatalf("CurrentRound = %d, want 2", got)
	}
	if got := result.NextState.ActiveTargets.EffectiveRound; got != 2 {
		t.Fatalf("ActiveTargets.EffectiveRound = %d, want 2", got)
	}
	if got := len(result.NextState.History.RecentRounds); got != 1 {
		t.Fatalf("History len = %d, want 1", got)
	}
	if got := result.Round.Metrics.ThroughputRevenue; got != 37 {
		t.Fatalf("ThroughputRevenue = %d, want 37", got)
	}
	if got := result.Round.Metrics.ProductionSpend; got != 6 {
		t.Fatalf("ProductionSpend = %d, want 6", got)
	}
	if got := result.Round.Metrics.ProductionOutputUnits; got != 2 {
		t.Fatalf("ProductionOutputUnits = %d, want 2", got)
	}
	if got := result.Round.Metrics.BacklogUnits; got != 17 {
		t.Fatalf("BacklogUnits = %d, want 17", got)
	}
	if !hasRoundEvent(result.Round.Events, domain.EventShipmentCompleted) {
		t.Fatalf("Round.Events missing shipment event: %#v", result.Round.Events)
	}
	if !hasRoundEvent(result.Round.Events, domain.EventDemandRealized) {
		t.Fatalf("Round.Events missing demand event: %#v", result.Round.Events)
	}
	if got := len(result.Round.Commentary); got != 4 {
		t.Fatalf("Round.Commentary len = %d, want 4", got)
	}
	if len(result.Round.Timeline) == 0 {
		t.Fatal("Round.Timeline is empty, want revealed chronology")
	}
}

func hasRoundEvent(events []domain.RoundEvent, eventType domain.RoundEventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
