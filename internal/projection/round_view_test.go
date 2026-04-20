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
	if len(view.RecentEvents) != 2 {
		t.Fatalf("RecentEvents len = %d, want 2", len(view.RecentEvents))
	}
	if len(view.RecentCommentary) != 1 {
		t.Fatalf("RecentCommentary len = %d, want 1", len(view.RecentCommentary))
	}
	if view.Plant.Cash != 100 {
		t.Fatalf("Plant.Cash = %d, want 100", view.Plant.Cash)
	}
}
