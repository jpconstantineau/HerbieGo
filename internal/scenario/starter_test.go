package scenario_test

import (
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestStarterInitialStateProvidesKnownPlayableSetup(t *testing.T) {
	starter := scenario.Starter()

	state := starter.InitialState("match-16", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "proc"},
		{RoleID: domain.RoleProductionManager, PlayerID: "prod"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales"},
		{RoleID: domain.RoleFinanceController, PlayerID: "fin"},
	})

	if got := state.ScenarioID; got != scenario.StarterID {
		t.Fatalf("ScenarioID = %q, want %q", got, scenario.StarterID)
	}
	if got := state.CurrentRound; got != 1 {
		t.Fatalf("CurrentRound = %d, want 1", got)
	}
	if got := state.Plant.Cash; got != 24 {
		t.Fatalf("Plant.Cash = %d, want 24", got)
	}
	if got := state.Plant.DebtCeiling; got != 15 {
		t.Fatalf("Plant.DebtCeiling = %d, want 15", got)
	}
	if got := len(state.Plant.PartsInventory); got != 4 {
		t.Fatalf("PartsInventory len = %d, want 4", got)
	}
	if got := len(state.Customers); got != 3 {
		t.Fatalf("Customers len = %d, want 3", got)
	}
	if got := len(state.Plant.Backlog); got != 2 {
		t.Fatalf("Backlog len = %d, want 2", got)
	}
	if got := state.Plant.Workstations[1].WorkstationID; got != "assembly" {
		t.Fatalf("bottleneck workstation = %q, want assembly", got)
	}
}

func TestStarterResolverOptionsApplyScenarioHooks(t *testing.T) {
	starter := scenario.Starter()
	state := starter.InitialState("match-16", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "proc"},
		{RoleID: domain.RoleProductionManager, PlayerID: "prod"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales"},
		{RoleID: domain.RoleFinanceController, PlayerID: "fin"},
	})
	state.Plant.Backlog = nil
	for index := range state.Customers {
		state.Customers[index].Backlog = nil
	}

	resolver := engine.NewResolver(starter.ResolverOptions())
	actions := []domain.ActionSubmission{
		{
			ActionID: "proc-1",
			MatchID:  state.MatchID,
			Round:    state.CurrentRound,
			RoleID:   domain.RoleProcurementManager,
			Action: domain.RoleAction{
				Procurement: &domain.ProcurementAction{
					Orders: []domain.PurchaseOrderIntent{
						{PartID: "housing", SupplierID: "forgeco", Quantity: 1},
					},
				},
			},
		},
		{
			ActionID: "prod-1",
			MatchID:  state.MatchID,
			Round:    state.CurrentRound,
			RoleID:   domain.RoleProductionManager,
			Action: domain.RoleAction{
				Production: &domain.ProductionAction{},
			},
		},
		{
			ActionID: "sales-1",
			MatchID:  state.MatchID,
			Round:    state.CurrentRound,
			RoleID:   domain.RoleSalesManager,
			Action: domain.RoleAction{
				Sales: &domain.SalesAction{
					ProductOffers: []domain.ProductOffer{
						{ProductID: "pump", UnitPrice: 14},
						{ProductID: "valve", UnitPrice: 9},
					},
				},
			},
		},
		{
			ActionID: "fin-1",
			MatchID:  state.MatchID,
			Round:    state.CurrentRound,
			RoleID:   domain.RoleFinanceController,
			Action: domain.RoleAction{
				Finance: &domain.FinanceAction{
					NextRoundTargets: state.ActiveTargets,
				},
			},
		},
	}

	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.NextState.Plant.InTransitSupply[0].UnitCost; got != 3 {
		t.Fatalf("housing unit cost = %d, want 3", got)
	}
	if got := len(result.NextState.Plant.Backlog); got == 0 {
		t.Fatal("starter scenario demand hook did not create any backlog")
	}
	if !containsDemandEvent(result.Round.Events) {
		t.Fatalf("Round.Events missing demand event: %#v", result.Round.Events)
	}
}

func containsDemandEvent(events []domain.RoundEvent) bool {
	for _, event := range events {
		if event.Type == domain.EventDemandRealized {
			return true
		}
	}
	return false
}
