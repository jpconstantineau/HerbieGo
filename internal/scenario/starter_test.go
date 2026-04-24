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
	if got := state.Customers[0].PaymentDelayRounds; got != 2 {
		t.Fatalf("NorthBuild payment delay = %d, want 2", got)
	}
	if got := state.Customers[1].PaymentDelayRounds; got != 1 {
		t.Fatalf("PrairieFlow payment delay = %d, want 1", got)
	}
	if got := len(state.Suppliers); got != 4 {
		t.Fatalf("Suppliers len = %d, want 4", got)
	}
	if got := len(state.Plant.Backlog); got != 2 {
		t.Fatalf("Backlog len = %d, want 2", got)
	}
	if got := state.Plant.Workstations[1].WorkstationID; got != "assembly" {
		t.Fatalf("bottleneck workstation = %q, want assembly", got)
	}
	if got := state.Plant.Workstations[1].StressBufferUnits; got != 1 {
		t.Fatalf("assembly stress buffer = %d, want 1", got)
	}
	if got := state.Plant.Workstations[1].LaborCapacityPerRound; got != 3 {
		t.Fatalf("assembly labor capacity = %d, want 3", got)
	}
	if got := state.Plant.Workstations[1].OvertimeCostPerCapacityUnit; got != 3 {
		t.Fatalf("assembly overtime cost = %d, want 3", got)
	}
	if got := starter.ProductionModel.Bottleneck.WorkstationID; got != "assembly" {
		t.Fatalf("ProductionModel.Bottleneck = %q, want assembly", got)
	}
	if got := starter.StartingConditions.ID; got != "prairie_bootstrap" {
		t.Fatalf("StartingConditions.ID = %q, want prairie_bootstrap", got)
	}
	if got := starter.MarketModel.ID; got != "regional_weekly_demand" {
		t.Fatalf("MarketModel.ID = %q, want regional_weekly_demand", got)
	}
	if got := starter.ProductionModel.ID; got != "two_stage_pump_valve_line" {
		t.Fatalf("ProductionModel.ID = %q, want two_stage_pump_valve_line", got)
	}
	if got := starter.FinanceModel.ID; got != "net30-lite-weekly" {
		t.Fatalf("FinanceModel.ID = %q, want net30-lite-weekly", got)
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
	if got := result.NextState.Plant.InTransitSupply[0].ArrivalRound; got != 3 {
		t.Fatalf("housing arrival round = %d, want 3", got)
	}
	if got := len(result.NextState.Plant.Backlog); got == 0 {
		t.Fatal("starter scenario demand hook did not create any backlog")
	}
	if !containsDemandEvent(result.Round.Events) {
		t.Fatalf("Round.Events missing demand event: %#v", result.Round.Events)
	}
}

func TestStarterAlternateSupplierChangesLeadTimeAndCost(t *testing.T) {
	starter := scenario.Starter()
	state := starter.InitialState("match-19", []domain.RoleAssignment{
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
	result, err := resolver.ResolveRound(state, []domain.ActionSubmission{
		{
			ActionID: "proc-1",
			MatchID:  state.MatchID,
			Round:    state.CurrentRound,
			RoleID:   domain.RoleProcurementManager,
			Action: domain.RoleAction{
				Procurement: &domain.ProcurementAction{
					Orders: []domain.PurchaseOrderIntent{
						{PartID: "housing", SupplierID: "prairiefast", Quantity: 1},
					},
				},
			},
		},
	}, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	lot := result.NextState.Plant.InTransitSupply[0]
	if got := lot.SupplierID; got != "prairiefast" {
		t.Fatalf("SupplierID = %q, want prairiefast", got)
	}
	if got := lot.UnitCost; got != 5 {
		t.Fatalf("UnitCost = %d, want 5", got)
	}
	if got := lot.ArrivalRound; got != 2 {
		t.Fatalf("ArrivalRound = %d, want 2", got)
	}
}

func TestStarterSelectsSupplierBehaviorAtMatchCreation(t *testing.T) {
	first := scenario.Starter().InitialState("match-20", nil)
	second := scenario.Starter().InitialState("match-20", nil)
	other := scenario.Starter().InitialState("match-21", nil)

	if len(first.Suppliers) == 0 {
		t.Fatal("starter state should seed supplier behavior")
	}
	if first.Suppliers[0].BehaviorID != second.Suppliers[0].BehaviorID {
		t.Fatalf("same match should keep supplier behavior stable: %q vs %q", first.Suppliers[0].BehaviorID, second.Suppliers[0].BehaviorID)
	}
	if first.Suppliers[0].BehaviorID == "" {
		t.Fatal("supplier behavior id should not be empty")
	}
	_ = other
}

func TestScenarioComponentsCanBeSelectedIndependently(t *testing.T) {
	starting := scenario.StarterStartingConditions()
	starting.ID = "cash_heavy_bootstrap"
	starting.StartingPlant.Cash = 40

	market := scenario.StarterMarketModel()
	market.ID = "quiet_market"
	for customerIndex := range market.Customers {
		for productID, profile := range market.Customers[customerIndex].DemandByProduct {
			profile.BaseDemand = 1
			market.Customers[customerIndex].DemandByProduct[productID] = profile
		}
	}

	production := scenario.StarterProductionModel()
	production.ID = "wide_open_capacity"
	production.Workstations[1].CapacityPerRound = 8

	definition := scenario.NewDefinition(
		"mixed-starter",
		"Mixed Starter",
		"Composed from independently selected scenario components.",
		scenario.StarterMatchSetup(),
		starting,
		market,
		production,
		scenario.StarterFinanceModel(),
	)

	if got := definition.StartingConditions.ID; got != "cash_heavy_bootstrap" {
		t.Fatalf("StartingConditions.ID = %q, want cash_heavy_bootstrap", got)
	}
	if got := definition.MarketModel.ID; got != "quiet_market" {
		t.Fatalf("MarketModel.ID = %q, want quiet_market", got)
	}
	if got := definition.ProductionModel.ID; got != "wide_open_capacity" {
		t.Fatalf("ProductionModel.ID = %q, want wide_open_capacity", got)
	}

	state := definition.InitialState("mixed-match", nil)
	if got := state.Plant.Cash; got != 40 {
		t.Fatalf("Plant.Cash = %d, want 40", got)
	}

	resolver := engine.NewResolver(definition.ResolverOptions())
	result, err := resolver.ResolveRound(state, []domain.ActionSubmission{
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
	}, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.NextState.Plant.Workstations[1].CapacityPerRound; got != 8 {
		t.Fatalf("assembly capacity = %d, want 8", got)
	}
	if len(result.NextState.Plant.Backlog) == 0 {
		t.Fatal("composed scenario should still generate backlog")
	}
}

func TestStarterDemandFallsWhenPumpPriceRises(t *testing.T) {
	starter := scenario.Starter()
	state := starter.InitialState("match-17", []domain.RoleAssignment{
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

	lowPriceResult, err := resolver.ResolveRound(state, []domain.ActionSubmission{
		starterSalesAction(state, 12, 9),
	}, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() low price error = %v", err)
	}

	highPriceResult, err := resolver.ResolveRound(state, []domain.ActionSubmission{
		starterSalesAction(state, 18, 9),
	}, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() high price error = %v", err)
	}

	lowDemand := demandQuantityForProduct(lowPriceResult.Round.Events, "pump")
	highDemand := demandQuantityForProduct(highPriceResult.Round.Events, "pump")
	if lowDemand <= highDemand {
		t.Fatalf("pump demand at low price = %d, want > high price demand %d", lowDemand, highDemand)
	}

	if got := backlogQuantityForProduct(lowPriceResult.NextState.Plant.Backlog, "pump"); got != lowDemand {
		t.Fatalf("low price pump backlog = %d, want %d", got, lowDemand)
	}
	if got := backlogQuantityForProduct(highPriceResult.NextState.Plant.Backlog, "pump"); got != highDemand {
		t.Fatalf("high price pump backlog = %d, want %d", got, highDemand)
	}
}

func TestStarterExpiredBacklogBecomesLostSalesAndReducesSentiment(t *testing.T) {
	starter := scenario.Starter()
	state := starter.InitialState("match-18", []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "proc"},
		{RoleID: domain.RoleProductionManager, PlayerID: "prod"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales"},
		{RoleID: domain.RoleFinanceController, PlayerID: "fin"},
	})
	state.CurrentRound = 3
	state.Plant.Backlog = []domain.BacklogEntry{
		{CustomerID: "northbuild", ProductID: "pump", Quantity: 2, OriginRound: 1, AgeInRounds: 2},
	}
	state.Plant.FinishedInventory = nil
	for index := range state.Customers {
		state.Customers[index].Backlog = nil
	}
	state.Customers[0].Backlog = []domain.BacklogEntry{
		{CustomerID: "northbuild", ProductID: "pump", Quantity: 2, OriginRound: 1, AgeInRounds: 2},
	}

	resolver := engine.NewResolver(starter.ResolverOptions())
	result, err := resolver.ResolveRound(state, []domain.ActionSubmission{
		starterSalesAction(state, 100, 100),
	}, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.Round.Metrics.LostSalesUnits; got != 2 {
		t.Fatalf("LostSalesUnits = %d, want 2", got)
	}
	if got := result.NextState.Customers[0].Sentiment; got != 5 {
		t.Fatalf("northbuild sentiment = %d, want 5", got)
	}
	if got := len(result.NextState.Plant.Backlog); got != 0 {
		t.Fatalf("Plant.Backlog len = %d, want 0 after expiry without new demand", got)
	}
	if !containsEvent(result.Round.Events, domain.EventBacklogExpired) {
		t.Fatalf("Round.Events missing %q: %#v", domain.EventBacklogExpired, result.Round.Events)
	}
	if !containsEvent(result.Round.Events, domain.EventCustomerSentimentMoved) {
		t.Fatalf("Round.Events missing %q: %#v", domain.EventCustomerSentimentMoved, result.Round.Events)
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

func containsEvent(events []domain.RoundEvent, eventType domain.RoundEventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func starterSalesAction(state domain.MatchState, pumpPrice, valvePrice domain.Money) domain.ActionSubmission {
	return domain.ActionSubmission{
		ActionID: "sales-1",
		MatchID:  state.MatchID,
		Round:    state.CurrentRound,
		RoleID:   domain.RoleSalesManager,
		Action: domain.RoleAction{
			Sales: &domain.SalesAction{
				ProductOffers: []domain.ProductOffer{
					{ProductID: "pump", UnitPrice: pumpPrice},
					{ProductID: "valve", UnitPrice: valvePrice},
				},
			},
		},
	}
}

func demandQuantityForProduct(events []domain.RoundEvent, productID domain.ProductID) domain.Units {
	total := domain.Units(0)
	for _, event := range events {
		if event.Type != domain.EventDemandRealized || event.Payload["product_id"] != string(productID) {
			continue
		}
		quantity, ok := event.Payload["quantity"].(int)
		if !ok {
			continue
		}
		total += domain.Units(quantity)
	}
	return total
}

func backlogQuantityForProduct(backlog []domain.BacklogEntry, productID domain.ProductID) domain.Units {
	total := domain.Units(0)
	for _, entry := range backlog {
		if entry.ProductID == productID {
			total += entry.Quantity
		}
	}
	return total
}
