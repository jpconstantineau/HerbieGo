package engine_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
)

func TestResolverProducesDeterministicRoundResults(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		WorldUpdate: func(ctx *engine.WorldUpdateContext) error {
			bonus := ctx.Random.IntN(3)
			ctx.State.Plant.Backlog = append(ctx.State.Plant.Backlog, domain.BacklogEntry{
				CustomerID:  "cust-2",
				ProductID:   "widget",
				Quantity:    domain.Units(bonus + 1),
				OriginRound: ctx.State.CurrentRound,
			})
			ctx.AppendEvent(domain.EventDemandRealized, domain.ActorPlantSystem, "World update added demand", map[string]any{
				"customer_id": "cust-2",
				"product_id":  "widget",
				"quantity":    bonus + 1,
			})
			return nil
		},
	})

	state := fixtureState()
	actions := fixtureActions()

	first, err := resolver.ResolveRound(state, actions, seeded.New(99))
	if err != nil {
		t.Fatalf("ResolveRound() first error = %v", err)
	}

	second, err := resolver.ResolveRound(state, actions, seeded.New(99))
	if err != nil {
		t.Fatalf("ResolveRound() second error = %v", err)
	}

	if !reflect.DeepEqual(first.NextState, second.NextState) {
		t.Fatalf("NextState mismatch:\nfirst: %#v\nsecond: %#v", first.NextState, second.NextState)
	}
	if !reflect.DeepEqual(first.Round, second.Round) {
		t.Fatalf("Round mismatch:\nfirst: %#v\nsecond: %#v", first.Round, second.Round)
	}
}

func TestResolverExecutesRoundPhasesAndSchedulesNextTargets(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})

	result, err := resolver.ResolveRound(fixtureState(), fixtureActions(), seeded.New(7))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if result.NextState.CurrentRound != 3 {
		t.Fatalf("CurrentRound = %d, want 3", result.NextState.CurrentRound)
	}
	if result.NextState.ActiveTargets.EffectiveRound != 3 {
		t.Fatalf("ActiveTargets.EffectiveRound = %d, want 3", result.NextState.ActiveTargets.EffectiveRound)
	}
	if result.NextState.Plant.Cash != 24 {
		t.Fatalf("Plant.Cash = %d, want 24", result.NextState.Plant.Cash)
	}
	if result.NextState.Plant.Debt != 0 {
		t.Fatalf("Plant.Debt = %d, want 0", result.NextState.Plant.Debt)
	}
	if got := finishedQty(result.NextState.Plant.FinishedInventory, "widget"); got != 1 {
		t.Fatalf("FinishedInventory(widget) = %d, want 1", got)
	}
	if got := partQty(result.NextState.Plant.PartsInventory, "housing"); got != 4 {
		t.Fatalf("PartsInventory(housing) = %d, want 4", got)
	}
	if len(result.NextState.Plant.InTransitSupply) != 1 {
		t.Fatalf("InTransitSupply len = %d, want 1", len(result.NextState.Plant.InTransitSupply))
	}
	if got := backlogQty(result.NextState.Plant.Backlog, "cust-1", "widget"); got != 0 {
		t.Fatalf("Backlog(cust-1/widget) = %d, want 0", got)
	}
	if len(result.NextState.History.RecentRounds) != 1 {
		t.Fatalf("History len = %d, want 1", len(result.NextState.History.RecentRounds))
	}
	if result.Round.Metrics.ThroughputRevenue != 18 {
		t.Fatalf("ThroughputRevenue = %d, want 18", result.Round.Metrics.ThroughputRevenue)
	}
	if result.Round.Metrics.BacklogUnits != 0 {
		t.Fatalf("BacklogUnits = %d, want 0", result.Round.Metrics.BacklogUnits)
	}
	if result.Round.Metrics.ProductionOutputUnits != 2 {
		t.Fatalf("ProductionOutputUnits = %d, want 2", result.Round.Metrics.ProductionOutputUnits)
	}

	eventTypes := make([]domain.RoundEventType, len(result.Round.Events))
	for index, event := range result.Round.Events {
		eventTypes[index] = event.Type
	}

	for _, want := range []domain.RoundEventType{
		domain.EventBudgetActivated,
		domain.EventPurchaseOrderPlaced,
		domain.EventSupplyArrived,
		domain.EventProductionReleased,
		domain.EventFinishedGoodsProduced,
		domain.EventShipmentCompleted,
		domain.EventMetricSnapshot,
	} {
		if !slices.Contains(eventTypes, want) {
			t.Fatalf("Round.Events missing %q in %#v", want, eventTypes)
		}
	}
}

func TestResolverUsesWorldUpdateHookAfterSales(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		WorldUpdate: func(ctx *engine.WorldUpdateContext) error {
			ctx.State.Customers[0].Sentiment += 2
			ctx.AppendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "World update adjusted sentiment", map[string]any{
				"customer_id": string(ctx.State.Customers[0].CustomerID),
				"sentiment":   ctx.State.Customers[0].Sentiment,
			})
			return nil
		},
	})

	result, err := resolver.ResolveRound(fixtureState(), fixtureActions(), seeded.New(5))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if result.NextState.Customers[0].Sentiment != 7 {
		t.Fatalf("Customer sentiment = %d, want 7", result.NextState.Customers[0].Sentiment)
	}
	if result.Round.Events[len(result.Round.Events)-2].Type != domain.EventRuleAdjustment {
		t.Fatalf("World update event order = %#v, want rule adjustment before metric snapshot", result.Round.Events[len(result.Round.Events)-2].Type)
	}
}

func TestResolverUsesScenarioProcurementTermsAndRoutingHooks(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProcurementTerms: func(ctx engine.ProcurementTermsContext) engine.ProcurementTerms {
			if ctx.Order.PartID != "housing" {
				return engine.ProcurementTerms{UnitCost: 1}
			}
			return engine.ProcurementTerms{
				UnitCost:             2,
				MinimumOrderQuantity: 5,
			}
		},
		ProductionRoute: func(ctx engine.ProductionRouteContext) engine.ProductionRouteStep {
			switch ctx.CurrentStationID {
			case "fabrication":
				return engine.ProductionRouteStep{NextWorkstationID: "paint"}
			case "paint":
				return engine.ProductionRouteStep{NextWorkstationID: "assembly"}
			default:
				return engine.ProductionRouteStep{Finished: true}
			}
		},
	})

	state := fixtureState()
	state.ActiveTargets.ProcurementBudget = 20
	state.Plant.Workstations = []domain.WorkstationState{
		{WorkstationID: "fabrication", DisplayName: "Fabrication", CapacityPerRound: 3},
		{WorkstationID: "paint", DisplayName: "Paint", CapacityPerRound: 2},
		{WorkstationID: "assembly", DisplayName: "Assembly", CapacityPerRound: 3},
	}

	actions := fixtureActions()
	actions[1].Action.Production.CapacityAllocation = []domain.CapacityAllocation{
		{WorkstationID: "fabrication", ProductID: "widget", Capacity: 2},
	}

	result, err := resolver.ResolveRound(state, actions, seeded.New(13))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.NextState.Plant.InTransitSupply[0].Quantity; got != 5 {
		t.Fatalf("procurement MOQ quantity = %d, want 5", got)
	}
	if got := result.NextState.Plant.InTransitSupply[0].UnitCost; got != 2 {
		t.Fatalf("procurement unit cost = %d, want 2", got)
	}
	if got := wipQty(result.NextState.Plant.WIPInventory, "widget", "paint"); got != 2 {
		t.Fatalf("WIP paint quantity = %d, want 2", got)
	}
}

func fixtureState() domain.MatchState {
	return domain.MatchState{
		MatchID:      "match-1",
		ScenarioID:   "starter",
		CurrentRound: 2,
		ActiveTargets: domain.BudgetTargets{
			EffectiveRound:        2,
			ProcurementBudget:     6,
			ProductionSpendBudget: 10,
			RevenueTarget:         20,
			CashFloorTarget:       0,
			DebtCeilingTarget:     5,
		},
		Plant: domain.PlantState{
			Cash:        10,
			Debt:        0,
			DebtCeiling: 5,
			PartsInventory: []domain.PartInventory{
				{PartID: "housing", OnHandQty: 1},
			},
			WIPInventory: []domain.WIPInventory{
				{ProductID: "widget", WorkstationID: "assembly", Quantity: 1},
			},
			FinishedInventory: []domain.FinishedInventory{
				{ProductID: "widget", OnHandQty: 1},
			},
			InTransitSupply: []domain.SupplyLot{
				{
					PurchaseOrderID: "po-1",
					SupplierID:      "supplier-a",
					PartID:          "housing",
					Quantity:        3,
					UnitCost:        1,
					OrderedRound:    1,
					ArrivalRound:    2,
				},
			},
			Workstations: []domain.WorkstationState{
				{WorkstationID: "fabrication", DisplayName: "Fabrication", CapacityPerRound: 3},
				{WorkstationID: "assembly", DisplayName: "Assembly", CapacityPerRound: 2},
			},
			Backlog: []domain.BacklogEntry{
				{CustomerID: "cust-1", ProductID: "widget", Quantity: 2, OriginRound: 1, AgeInRounds: 1},
			},
		},
		Customers: []domain.CustomerState{
			{CustomerID: "cust-1", DisplayName: "NorthBuild", Sentiment: 5},
			{CustomerID: "cust-2", DisplayName: "PrairieFlow", Sentiment: 4},
		},
	}
}

func fixtureActions() []domain.ActionSubmission {
	return []domain.ActionSubmission{
		{
			ActionID: "proc-1",
			MatchID:  "match-1",
			Round:    2,
			RoleID:   domain.RoleProcurementManager,
			Action: domain.RoleAction{
				Procurement: &domain.ProcurementAction{
					Orders: []domain.PurchaseOrderIntent{
						{PartID: "housing", SupplierID: "supplier-a", Quantity: 4},
					},
				},
			},
			Commentary: domain.CommentaryRecord{
				Body: "Restocking for next week.",
			},
		},
		{
			ActionID: "prod-1",
			MatchID:  "match-1",
			Round:    2,
			RoleID:   domain.RoleProductionManager,
			Action: domain.RoleAction{
				Production: &domain.ProductionAction{
					Releases: []domain.ProductionRelease{
						{ProductID: "widget", Quantity: 2},
					},
					CapacityAllocation: []domain.CapacityAllocation{
						{WorkstationID: "fabrication", ProductID: "widget", Capacity: 2},
						{WorkstationID: "assembly", ProductID: "widget", Capacity: 2},
					},
				},
			},
		},
		{
			ActionID: "sales-1",
			MatchID:  "match-1",
			Round:    2,
			RoleID:   domain.RoleSalesManager,
			Action: domain.RoleAction{
				Sales: &domain.SalesAction{
					ProductOffers: []domain.ProductOffer{
						{ProductID: "widget", UnitPrice: 9},
					},
				},
			},
		},
		{
			ActionID: "fin-1",
			MatchID:  "match-1",
			Round:    2,
			RoleID:   domain.RoleFinanceController,
			Action: domain.RoleAction{
				Finance: &domain.FinanceAction{
					NextRoundTargets: domain.BudgetTargets{
						ProcurementBudget:     7,
						ProductionSpendBudget: 11,
						RevenueTarget:         24,
						CashFloorTarget:       2,
						DebtCeilingTarget:     6,
					},
				},
			},
		},
	}
}

func finishedQty(items []domain.FinishedInventory, productID domain.ProductID) domain.Units {
	for _, item := range items {
		if item.ProductID == productID {
			return item.OnHandQty
		}
	}
	return 0
}

func partQty(items []domain.PartInventory, partID domain.PartID) domain.Units {
	for _, item := range items {
		if item.PartID == partID {
			return item.OnHandQty
		}
	}
	return 0
}

func backlogQty(items []domain.BacklogEntry, customerID domain.CustomerID, productID domain.ProductID) domain.Units {
	for _, item := range items {
		if item.CustomerID == customerID && item.ProductID == productID {
			return item.Quantity
		}
	}
	return 0
}

func wipQty(items []domain.WIPInventory, productID domain.ProductID, workstationID domain.WorkstationID) domain.Units {
	for _, item := range items {
		if item.ProductID == productID && item.WorkstationID == workstationID {
			return item.Quantity
		}
	}
	return 0
}
