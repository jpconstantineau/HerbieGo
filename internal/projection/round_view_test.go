package projection_test

import (
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/projection"
)

func TestBuildRoundViewIncludesRecentHistoryWindow(t *testing.T) {
	state := domain.MatchState{
		MatchID:      "match-1",
		CurrentRound: 4,
		Plant: domain.PlantState{
			Cash: 100,
		},
		Customers: []domain.CustomerState{
			{CustomerID: "cust-1", DisplayName: "Acme"},
		},
		ActiveTargets: domain.BudgetTargets{
			EffectiveRound:        4,
			ProcurementBudget:     20,
			ProductionSpendBudget: 30,
		},
		Metrics: domain.PlantMetrics{
			ThroughputRevenue: 55,
		},
		History: domain.RoundHistory{
			RecentRounds: []domain.RoundRecord{
				{
					Round: 2,
					Events: []domain.RoundEvent{
						{
							EventID: "event-2",
							MatchID: "match-1",
							Round:   2,
							Type:    domain.EventSupplyArrived,
						},
					},
				},
				{
					Round: 3,
					Actions: []domain.ActionSubmission{
						{ActionID: "sales-3", RoleID: domain.RoleSalesManager},
					},
					Metrics: domain.PlantMetrics{
						RoundProfit:       12,
						ThroughputRevenue: 21,
					},
					Commentary: []domain.CommentaryRecord{
						{
							CommentaryID: "comment-3",
							MatchID:      "match-1",
							Round:        3,
							RoleID:       domain.RoleSalesManager,
							Visibility:   domain.CommentaryPublic,
							Body:         "Demand stayed strong.",
						},
					},
					Events: []domain.RoundEvent{
						{
							EventID: "event-3",
							MatchID: "match-1",
							Round:   3,
							Type:    domain.EventShipmentCompleted,
						},
					},
				},
			},
		},
	}

	view := projection.BuildRoundView(state, domain.RoleSalesManager)

	if view.Round != 4 {
		t.Fatalf("Round = %d, want 4", view.Round)
	}
	if view.ViewerRoleID != domain.RoleSalesManager {
		t.Fatalf("ViewerRoleID = %q, want %q", view.ViewerRoleID, domain.RoleSalesManager)
	}
	if len(view.RecentRounds) != 2 {
		t.Fatalf("RecentRounds len = %d, want 2", len(view.RecentRounds))
	}
	if len(view.RecentEvents) != 2 {
		t.Fatalf("RecentEvents len = %d, want 2", len(view.RecentEvents))
	}
	if len(view.RecentCommentary) != 1 {
		t.Fatalf("RecentCommentary len = %d, want 1", len(view.RecentCommentary))
	}
	if got := view.RecentRounds[1].Round; got != 3 {
		t.Fatalf("RecentRounds[1].Round = %d, want 3", got)
	}
	if got := view.RecentRounds[1].Commentary[0].Body; got != "Demand stayed strong." {
		t.Fatalf("RecentRounds[1].Commentary[0].Body = %q, want %q", got, "Demand stayed strong.")
	}
	if got := view.RecentRounds[1].Summary.CommentaryCount; got != 1 {
		t.Fatalf("RecentRounds[1].Summary.CommentaryCount = %d, want 1", got)
	}
	if got := view.RecentRounds[1].Summary.ActionCount; got != 1 {
		t.Fatalf("RecentRounds[1].Summary.ActionCount = %d, want 1", got)
	}
	if got := view.RecentRounds[1].Summary.Metrics.RoundProfit; got != 12 {
		t.Fatalf("RecentRounds[1].Summary.Metrics.RoundProfit = %d, want 12", got)
	}
	if view.Plant.Cash != 100 {
		t.Fatalf("Plant.Cash = %d, want 100", view.Plant.Cash)
	}
}

func TestBuildRoundViewLimitsStructuredHistoryToLastTenRounds(t *testing.T) {
	rounds := make([]domain.RoundRecord, 0, 12)
	for round := range 12 {
		rounds = append(rounds, domain.RoundRecord{
			Round: domain.RoundNumber(round + 1),
			Events: []domain.RoundEvent{
				{
					EventID: domain.EventID("event"),
					MatchID: "match-1",
					Round:   domain.RoundNumber(round + 1),
					Type:    domain.EventMetricSnapshot,
				},
			},
		})
	}

	view := projection.BuildRoundView(domain.MatchState{
		MatchID:      "match-1",
		CurrentRound: 13,
		History: domain.RoundHistory{
			RecentRounds: rounds,
		},
	}, domain.RoleFinanceController)

	if len(view.RecentRounds) != 10 {
		t.Fatalf("RecentRounds len = %d, want 10", len(view.RecentRounds))
	}
	if got := view.RecentRounds[0].Round; got != 3 {
		t.Fatalf("RecentRounds[0].Round = %d, want 3", got)
	}
	if got := view.RecentRounds[9].Round; got != 12 {
		t.Fatalf("RecentRounds[9].Round = %d, want 12", got)
	}
}

func TestBuildRoundViewClonesStructuredHistory(t *testing.T) {
	state := domain.MatchState{
		MatchID:      "match-1",
		CurrentRound: 2,
		History: domain.RoundHistory{
			RecentRounds: []domain.RoundRecord{
				{
					Round: 1,
					Events: []domain.RoundEvent{
						{
							EventID: "event-1",
							MatchID: "match-1",
							Round:   1,
							Type:    domain.EventDemandRealized,
							Payload: map[string]any{"quantity": 2},
						},
					},
					Commentary: []domain.CommentaryRecord{
						{
							CommentaryID: "comment-1",
							Body:         "Original",
						},
					},
				},
			},
		},
	}

	view := projection.BuildRoundView(state, domain.RoleProcurementManager)
	view.RecentRounds[0].Events[0].Payload["quantity"] = 99
	view.RecentRounds[0].Commentary[0].Body = "Changed"

	if got := state.History.RecentRounds[0].Events[0].Payload["quantity"]; got != 2 {
		t.Fatalf("state payload quantity = %#v, want 2", got)
	}
	if got := state.History.RecentRounds[0].Commentary[0].Body; got != "Original" {
		t.Fatalf("state commentary body = %q, want %q", got, "Original")
	}
}
