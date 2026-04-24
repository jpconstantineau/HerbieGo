package engine_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

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
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: widgetBOM,
	})

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
	if result.NextState.Plant.DebtCeiling != 6 {
		t.Fatalf("Plant.DebtCeiling = %d, want 6", result.NextState.Plant.DebtCeiling)
	}
	if result.NextState.Plant.Cash != 19 {
		t.Fatalf("Plant.Cash = %d, want 19", result.NextState.Plant.Cash)
	}
	if result.NextState.Plant.Debt != 0 {
		t.Fatalf("Plant.Debt = %d, want 0", result.NextState.Plant.Debt)
	}
	if got := finishedQty(result.NextState.Plant.FinishedInventory, "widget"); got != 1 {
		t.Fatalf("FinishedInventory(widget) = %d, want 1", got)
	}
	if got := partQty(result.NextState.Plant.PartsInventory, "housing"); got != 2 {
		t.Fatalf("PartsInventory(housing) = %d, want 2", got)
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
	if result.Round.Metrics.ProcurementSpend != 4 {
		t.Fatalf("ProcurementSpend = %d, want 4", result.Round.Metrics.ProcurementSpend)
	}
	if result.Round.Metrics.ProductionSpend != 4 {
		t.Fatalf("ProductionSpend = %d, want 4", result.Round.Metrics.ProductionSpend)
	}
	if result.Round.Metrics.HoldingCost != 1 {
		t.Fatalf("HoldingCost = %d, want 1", result.Round.Metrics.HoldingCost)
	}
	if result.Round.Metrics.OperatingExpense != 9 {
		t.Fatalf("OperatingExpense = %d, want 9", result.Round.Metrics.OperatingExpense)
	}
	if result.Round.Metrics.InventoryValue != 4 {
		t.Fatalf("InventoryValue = %d, want 4", result.Round.Metrics.InventoryValue)
	}
	if result.Round.Metrics.RoundProfit != 9 {
		t.Fatalf("RoundProfit = %d, want 9", result.Round.Metrics.RoundProfit)
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

	if len(result.Round.Timeline) == 0 {
		t.Fatal("Round.Timeline is empty, want canonical round chronology")
	}
	if got := result.Round.Timeline[0].Phase; got != domain.RoundTimelinePhaseIntake {
		t.Fatalf("Round.Timeline[0].Phase = %q, want %q", got, domain.RoundTimelinePhaseIntake)
	}
	if got := result.Round.Timeline[len(result.Round.Timeline)-1].Phase; got != domain.RoundTimelinePhaseSummary {
		t.Fatalf("Round.Timeline[last].Phase = %q, want %q", got, domain.RoundTimelinePhaseSummary)
	}
}

func TestResolverOrdersCommentaryBySubmissionTimeWithinIntakePhase(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	actions := fixtureActions()
	actions[0].SubmittedAt = time.Date(2026, time.April, 21, 12, 0, 3, 0, time.UTC)
	actions[1].Commentary = domain.CommentaryRecord{Body: "Assembly is the bottleneck."}
	actions[1].SubmittedAt = time.Date(2026, time.April, 21, 12, 0, 1, 0, time.UTC)
	actions[2].Commentary = domain.CommentaryRecord{Body: "Demand is steady."}
	actions[2].SubmittedAt = time.Date(2026, time.April, 21, 12, 0, 2, 0, time.UTC)
	actions[3].SubmittedAt = time.Date(2026, time.April, 21, 12, 0, 4, 0, time.UTC)

	result, err := resolver.ResolveRound(fixtureState(), actions, seeded.New(7))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	want := []domain.RoleID{
		domain.RoleProductionManager,
		domain.RoleSalesManager,
		domain.RoleProcurementManager,
	}
	if len(result.Round.Commentary) != len(want) {
		t.Fatalf("len(Round.Commentary) = %d, want %d", len(result.Round.Commentary), len(want))
	}
	for index, roleID := range want {
		if got := result.Round.Commentary[index].RoleID; got != roleID {
			t.Fatalf("Round.Commentary[%d].RoleID = %q, want %q", index, got, roleID)
		}
		if got := result.Round.Timeline[index].Commentary.RoleID; got != roleID {
			t.Fatalf("Round.Timeline[%d].Commentary.RoleID = %q, want %q", index, got, roleID)
		}
		if got := result.Round.Timeline[index].Sequence; got != index+1 {
			t.Fatalf("Round.Timeline[%d].Sequence = %d, want %d", index, got, index+1)
		}
	}
}

func TestResolverUsesWorldUpdateHookAfterSales(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: widgetBOM,
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
	if result.Round.Events[len(result.Round.Events)-1].Type != domain.EventMetricSnapshot {
		t.Fatalf("last event type = %#v, want metric snapshot", result.Round.Events[len(result.Round.Events)-1].Type)
	}
	if result.Round.Events[len(result.Round.Events)-3].Type != domain.EventRuleAdjustment {
		t.Fatalf("World update event order = %#v, want rule adjustment before round-end cash update and metric snapshot", result.Round.Events[len(result.Round.Events)-3].Type)
	}
}

func TestResolverUsesScenarioProcurementTermsAndRoutingHooks(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProcurementTerms: func(ctx engine.ProcurementTermsContext) engine.ProcurementTerms {
			if ctx.Order.PartID != "housing" {
				return engine.ProcurementTerms{UnitCost: 1, KnownSupplier: true}
			}
			return engine.ProcurementTerms{
				UnitCost:             2,
				MinimumOrderQuantity: 5,
				KnownSupplier:        true,
			}
		},
		ProductionBOM: widgetBOM,
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

func TestResolverUsesScenarioProductionAndInventoryCostHooks(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: widgetBOM,
		ProductionCost: func(ctx engine.ProductionCostContext) engine.ProductionCost {
			switch ctx.WorkstationID {
			case "fabrication":
				return engine.ProductionCost{CostPerCapacityUnit: 2}
			case "assembly":
				return engine.ProductionCost{CostPerCapacityUnit: 3}
			default:
				return engine.ProductionCost{CostPerCapacityUnit: 1}
			}
		},
		InventoryCost: func(ctx engine.InventoryCarryingCostContext) domain.Money {
			if ctx.InventoryValue <= 0 {
				return 0
			}
			switch ctx.InventoryClass {
			case engine.InventoryClassParts:
				return 1
			case engine.InventoryClassWIP:
				return 2
			case engine.InventoryClassFinished:
				return 3
			default:
				return 0
			}
		},
	})

	result, err := resolver.ResolveRound(fixtureState(), fixtureActions(), seeded.New(7))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.Round.Metrics.ProductionSpend; got != 10 {
		t.Fatalf("ProductionSpend = %d, want 10", got)
	}
	if got := result.Round.Metrics.HoldingCost; got != 6 {
		t.Fatalf("HoldingCost = %d, want 6", got)
	}
	if got := result.Round.Metrics.OperatingExpense; got != 20 {
		t.Fatalf("OperatingExpense = %d, want 20", got)
	}
}

func TestResolverUsesLaborCapacityAndOvertime(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: widgetBOM,
	})
	state := fixtureState()
	state.Plant.Workstations = []domain.WorkstationState{
		{
			WorkstationID:               "fabrication",
			DisplayName:                 "Fabrication",
			CapacityPerRound:            3,
			EffectiveCapacityPerRound:   3,
			StressBufferUnits:           0,
			StressPenaltyPerExcessUnit:  1,
			LaborCapacityPerRound:       1,
			LaborCostPerCapacityUnit:    1,
			OvertimeCostPerCapacityUnit: 2,
		},
		{
			WorkstationID:               "assembly",
			DisplayName:                 "Assembly",
			CapacityPerRound:            2,
			EffectiveCapacityPerRound:   2,
			StressBufferUnits:           0,
			StressPenaltyPerExcessUnit:  0,
			LaborCapacityPerRound:       2,
			LaborCostPerCapacityUnit:    1,
			OvertimeCostPerCapacityUnit: 0,
		},
	}
	state.Plant.WIPInventory = []domain.WIPInventory{
		{ProductID: "widget", WorkstationID: "fabrication", Quantity: 5, UnitCost: 1},
	}
	actions := fixtureActions()
	actions[1].Action.Production.Releases = []domain.ProductionRelease{{ProductID: "widget", Quantity: 2}}
	actions[1].Action.Production.CapacityAllocation = []domain.CapacityAllocation{
		{WorkstationID: "fabrication", ProductID: "widget", Capacity: 2},
	}

	noOvertime, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() no overtime error = %v", err)
	}
	if got := wipQty(noOvertime.NextState.Plant.WIPInventory, "widget", "assembly"); got != 2 {
		t.Fatalf("WIP(widget/assembly) without overtime = %d, want 2", got)
	}
	if got := noOvertime.Round.Metrics.OvertimeUnits; got != 0 {
		t.Fatalf("OvertimeUnits without overtime = %d, want 0", got)
	}
	if got := noOvertime.Round.Metrics.LaborCost; got != 3 {
		t.Fatalf("LaborCost without overtime = %d, want 3", got)
	}

	actions[1].Action.Production.Overtime = []domain.OvertimeAllocation{
		{WorkstationID: "fabrication", Capacity: 1},
	}
	withOvertime, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() with overtime error = %v", err)
	}
	if got := wipQty(withOvertime.NextState.Plant.WIPInventory, "widget", "assembly"); got != 3 {
		t.Fatalf("WIP(widget/assembly) with overtime = %d, want 3", got)
	}
	if got := withOvertime.Round.Metrics.OvertimeUnits; got != 1 {
		t.Fatalf("OvertimeUnits with overtime = %d, want 1", got)
	}
	if got := withOvertime.Round.Metrics.OvertimeCost; got != 2 {
		t.Fatalf("OvertimeCost with overtime = %d, want 2", got)
	}
	if !containsEvent(withOvertime.Round.Events, domain.EventOvertimeApplied) {
		t.Fatalf("Round.Events missing %q: %#v", domain.EventOvertimeApplied, withOvertime.Round.Events)
	}
}

func TestResolverReducesEffectiveCapacityUnderCongestion(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: widgetBOM,
	})
	state := fixtureState()
	state.Plant.Workstations = []domain.WorkstationState{
		{
			WorkstationID:              "fabrication",
			DisplayName:                "Fabrication",
			CapacityPerRound:           3,
			EffectiveCapacityPerRound:  3,
			StressBufferUnits:          0,
			StressPenaltyPerExcessUnit: 1,
		},
		{
			WorkstationID:              "assembly",
			DisplayName:                "Assembly",
			CapacityPerRound:           2,
			EffectiveCapacityPerRound:  2,
			StressBufferUnits:          0,
			StressPenaltyPerExcessUnit: 0,
		},
	}
	state.Plant.WIPInventory = []domain.WIPInventory{
		{ProductID: "widget", WorkstationID: "fabrication", Quantity: 5, UnitCost: 1},
	}
	actions := fixtureActions()
	actions[1].Action.Production.Releases = nil
	actions[1].Action.Production.CapacityAllocation = []domain.CapacityAllocation{
		{WorkstationID: "fabrication", ProductID: "widget", Capacity: 3},
	}

	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.NextState.Plant.Workstations[0].EffectiveCapacityPerRound; got != 1 {
		t.Fatalf("fabrication effective capacity = %d, want 1", got)
	}
	if got := result.NextState.Plant.Workstations[0].StressCapacityLoss; got != 2 {
		t.Fatalf("fabrication stress loss = %d, want 2", got)
	}
	if got := result.NextState.Plant.Workstations[0].CapacityUsed; got != 1 {
		t.Fatalf("fabrication capacity used = %d, want 1", got)
	}
	if got := result.Round.Metrics.CapacityLossUnits; got != 2 {
		t.Fatalf("CapacityLossUnits = %d, want 2", got)
	}
	if !containsEvent(result.Round.Events, domain.EventWorkstationStressed) {
		t.Fatalf("Round.Events missing %q: %#v", domain.EventWorkstationStressed, result.Round.Events)
	}
}

func TestResolverLimitsOvertimeThroughProductionBudget(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: widgetBOM,
	})
	state := fixtureState()
	state.ActiveTargets.ProductionSpendBudget = 3
	state.Plant.Workstations = []domain.WorkstationState{
		{
			WorkstationID:               "fabrication",
			DisplayName:                 "Fabrication",
			CapacityPerRound:            3,
			EffectiveCapacityPerRound:   3,
			StressBufferUnits:           0,
			StressPenaltyPerExcessUnit:  1,
			LaborCapacityPerRound:       1,
			LaborCostPerCapacityUnit:    1,
			OvertimeCostPerCapacityUnit: 2,
		},
		{
			WorkstationID:               "assembly",
			DisplayName:                 "Assembly",
			CapacityPerRound:            2,
			EffectiveCapacityPerRound:   2,
			StressBufferUnits:           0,
			StressPenaltyPerExcessUnit:  0,
			LaborCapacityPerRound:       2,
			LaborCostPerCapacityUnit:    1,
			OvertimeCostPerCapacityUnit: 0,
		},
	}
	actions := fixtureActions()
	actions[1].Action.Production.Releases = []domain.ProductionRelease{{ProductID: "widget", Quantity: 2}}
	actions[1].Action.Production.CapacityAllocation = []domain.CapacityAllocation{
		{WorkstationID: "fabrication", ProductID: "widget", Capacity: 2},
	}
	actions[1].Action.Production.Overtime = []domain.OvertimeAllocation{
		{WorkstationID: "fabrication", Capacity: 1},
	}

	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := wipQty(result.NextState.Plant.WIPInventory, "widget", "assembly"); got != 2 {
		t.Fatalf("WIP(widget/assembly) = %d, want 2 because overtime should be budget-trimmed", got)
	}
	if got := result.Round.Metrics.OvertimeUnits; got != 0 {
		t.Fatalf("OvertimeUnits = %d, want 0 when production budget cannot absorb overtime", got)
	}
}

func TestResolverRequiresExplicitRolePayload(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	actions := fixtureActions()
	actions[2].Action = domain.RoleAction{}

	_, err := resolver.ResolveRound(fixtureState(), actions, seeded.New(1))
	if err == nil {
		t.Fatal("ResolveRound() error = nil, want missing payload error")
	}
	if got := err.Error(); got != `engine: action "sales-1" role "sales_manager" must include payload "sales_manager"` {
		t.Fatalf("ResolveRound() error = %q", got)
	}
}

func TestResolverRejectsMismatchedRolePayloads(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	actions := fixtureActions()
	actions[0].Action = domain.RoleAction{
		Sales: &domain.SalesAction{
			ProductOffers: []domain.ProductOffer{
				{ProductID: "widget", UnitPrice: 9},
			},
		},
	}

	_, err := resolver.ResolveRound(fixtureState(), actions, seeded.New(1))
	if err == nil {
		t.Fatal("ResolveRound() error = nil, want mismatched payload error")
	}
	if got := err.Error(); got != `engine: action "proc-1" role "procurement_manager" includes mismatched payload "sales_manager"` {
		t.Fatalf("ResolveRound() error = %q", got)
	}
}

func TestResolverRejectsMultiplePayloads(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	actions := fixtureActions()
	actions[1].Action.Sales = &domain.SalesAction{
		ProductOffers: []domain.ProductOffer{
			{ProductID: "widget", UnitPrice: 7},
		},
	}

	_, err := resolver.ResolveRound(fixtureState(), actions, seeded.New(1))
	if err == nil {
		t.Fatal("ResolveRound() error = nil, want multiple payload error")
	}
	if got := err.Error(); got != `engine: action "prod-1" role "production_manager" includes multiple payloads: production_manager, sales_manager` {
		t.Fatalf("ResolveRound() error = %q", got)
	}
}

func TestResolverRejectsUnknownWorkstationsDuringValidation(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	actions := fixtureActions()
	actions[1].Action.Production.CapacityAllocation[0].WorkstationID = "unknown"

	_, err := resolver.ResolveRound(fixtureState(), actions, seeded.New(1))
	if err == nil {
		t.Fatal("ResolveRound() error = nil, want unknown workstation error")
	}
	if got := err.Error(); got != `engine: action "prod-1" capacity allocation 0 references unknown workstation "unknown"` {
		t.Fatalf("ResolveRound() error = %q", got)
	}
}

func TestResolverRejectsNegativeValues(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	actions := fixtureActions()
	actions[2].Action.Sales.ProductOffers[0].UnitPrice = -1

	_, err := resolver.ResolveRound(fixtureState(), actions, seeded.New(1))
	if err == nil {
		t.Fatal("ResolveRound() error = nil, want negative price error")
	}
	if got := err.Error(); got != `engine: action "sales-1" sales offer 0 unit price must be non-negative` {
		t.Fatalf("ResolveRound() error = %q", got)
	}
}

func TestResolverRejectsActionsForUnassignedRoles(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	state := fixtureState()
	state.Roles = []domain.RoleAssignment{
		{RoleID: domain.RoleProcurementManager, PlayerID: "proc"},
		{RoleID: domain.RoleProductionManager, PlayerID: "prod"},
		{RoleID: domain.RoleSalesManager, PlayerID: "sales"},
	}

	_, err := resolver.ResolveRound(state, fixtureActions(), seeded.New(1))
	if err == nil {
		t.Fatal("ResolveRound() error = nil, want unassigned role error")
	}
	if got := err.Error(); got != `engine: action "fin-1" role "finance_controller" is not assigned in the current match` {
		t.Fatalf("ResolveRound() error = %q", got)
	}
}

func TestResolverEmitsRuleAdjustmentWhenProcurementOrderIsFullyRejected(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	state := fixtureState()
	state.ActiveTargets.ProcurementBudget = 1
	state.Plant.Cash = 0
	state.Plant.DebtCeiling = 0

	result, err := resolver.ResolveRound(state, fixtureActions(), seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := len(result.NextState.Plant.InTransitSupply); got != 0 {
		t.Fatalf("InTransitSupply len = %d, want 0", got)
	}

	rejected := false
	for _, event := range result.Round.Events {
		if event.Type != domain.EventRuleAdjustment {
			continue
		}
		if event.Summary == "Rejected procurement order because no legal quantity remained" {
			rejected = true
			if got := event.Payload["accepted_quantity"]; got != 0 {
				t.Fatalf("accepted_quantity = %#v, want 0", got)
			}
		}
	}
	if !rejected {
		t.Fatalf("Round.Events missing procurement rejection rule adjustment: %#v", result.Round.Events)
	}
}

func TestResolverConsumesPartsAndTrimsReleaseToAvailableBOM(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: widgetBOM,
	})
	state := fixtureState()
	state.Plant.PartsInventory = []domain.PartInventory{
		{PartID: "housing", OnHandQty: 1},
	}
	state.Plant.InTransitSupply = nil
	actions := fixtureActions()
	actions[1].Action.Production.Releases = []domain.ProductionRelease{
		{ProductID: "widget", Quantity: 3},
	}
	actions[1].Action.Production.CapacityAllocation = nil

	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := partQty(result.NextState.Plant.PartsInventory, "housing"); got != 0 {
		t.Fatalf("PartsInventory(housing) = %d, want 0", got)
	}
	if got := wipQty(result.NextState.Plant.WIPInventory, "widget", "fabrication"); got != 1 {
		t.Fatalf("WIP(fabrication/widget) = %d, want 1", got)
	}

	trimmed := false
	for _, event := range result.Round.Events {
		if event.Type == domain.EventRuleAdjustment && event.Summary == "Trimmed production release to available part inventory" {
			trimmed = true
			if got := event.Payload["accepted_quantity"]; got != 1 {
				t.Fatalf("accepted_quantity = %#v, want 1", got)
			}
		}
	}
	if !trimmed {
		t.Fatalf("Round.Events missing production trim event: %#v", result.Round.Events)
	}
}

func TestResolverRejectsUnknownProductsWhenProductionBOMMarksThemUnknown(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: func(ctx engine.ProductionBOMContext) engine.ProductionBOM {
			if ctx.ProductID == "widget" {
				return widgetBOM(ctx)
			}
			return engine.ProductionBOM{}
		},
	})
	state := fixtureState()
	actions := fixtureActions()
	actions[1].Action.Production.Releases = []domain.ProductionRelease{
		{ProductID: "mystery", Quantity: 1},
	}
	actions[1].Action.Production.CapacityAllocation = nil

	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	rejected := false
	for _, event := range result.Round.Events {
		if event.Type == domain.EventRuleAdjustment && event.Summary == "Rejected production release for unknown product" {
			rejected = true
		}
	}
	if !rejected {
		t.Fatalf("Round.Events missing unknown-product rejection: %#v", result.Round.Events)
	}
}

func TestResolverTrimsProductionToBudgetAndAvailableCash(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM: widgetBOM,
	})
	state := fixtureState()
	state.ActiveTargets.ProductionSpendBudget = 1
	actions := fixtureActions()

	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.Round.Metrics.ProductionSpend; got != 1 {
		t.Fatalf("ProductionSpend = %d, want 1", got)
	}
	if got := result.NextState.Plant.Workstations[0].CapacityUsed; got != 1 {
		t.Fatalf("fabrication capacity used = %d, want 1", got)
	}

	trimmed := false
	for _, event := range result.Round.Events {
		if event.Type == domain.EventRuleAdjustment && event.Summary == "Trimmed production advance to available work or capacity" {
			if got := event.Payload["accepted_capacity"]; got == 1 {
				trimmed = true
			}
		}
	}
	if !trimmed {
		t.Fatalf("Round.Events missing production budget trim: %#v", result.Round.Events)
	}
}

func TestResolverTracksLostSalesAndDebtServiceInMetrics(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{})
	state := fixtureState()
	state.Plant.Cash = 0
	state.Plant.Debt = 10
	state.Plant.DebtCeiling = 20
	state.Plant.Backlog = []domain.BacklogEntry{
		{CustomerID: "cust-1", ProductID: "widget", Quantity: 3, OriginRound: 1, AgeInRounds: 2},
	}
	state.Plant.FinishedInventory = nil
	state.Plant.InTransitSupply = nil

	actions := fixtureActions()
	actions[0].Action.Procurement.Orders = nil
	actions[1].Action.Production.Releases = nil
	actions[1].Action.Production.CapacityAllocation = nil
	actions[2].Action.Sales.ProductOffers = nil

	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.Round.Metrics.LostSalesUnits; got != 3 {
		t.Fatalf("LostSalesUnits = %d, want 3", got)
	}
	if got := result.Round.Metrics.DebtServiceCost; got != 1 {
		t.Fatalf("DebtServiceCost = %d, want 1", got)
	}
	if got := result.NextState.Customers[0].Sentiment; got != 4 {
		t.Fatalf("Customer sentiment = %d, want 4", got)
	}
}

func TestResolverSchedulesReceivablesPayablesAndUsesProjectedDebtCapacity(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProductionBOM:         widgetBOM,
		ReceivableDelayRounds: 1,
		PayableDelayRounds:    1,
	})

	state := fixtureState()
	state.Customers[0].PaymentDelayRounds = 2
	state.Plant.Payables = []domain.CashCommitment{
		{
			CommitmentID: "ap-1",
			Kind:         domain.CashCommitmentPayable,
			Amount:       12,
			DueRound:     3,
			CreatedRound: 2,
			ReferenceID:  "legacy-procurement",
		},
	}

	result, err := resolver.ResolveRound(state, fixtureActions(), seeded.New(7))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := result.NextState.Plant.Cash; got != 5 {
		t.Fatalf("Plant.Cash = %d, want 5", got)
	}
	if got := result.NextState.Plant.Debt; got != 0 {
		t.Fatalf("Plant.Debt = %d, want 0", got)
	}
	if got := len(result.NextState.Plant.Receivables); got != 1 {
		t.Fatalf("Receivables len = %d, want 1", got)
	}
	if got := len(result.NextState.Plant.Payables); got != 2 {
		t.Fatalf("Payables len = %d, want 2", got)
	}
	if got := result.NextState.Plant.Receivables[0].DueRound; got != 4 {
		t.Fatalf("Customer-specific receivable due round = %d, want 4", got)
	}
	if got := result.NextState.Plant.Payables[1].DueRound; got != 3 {
		t.Fatalf("Scheduled payable due round = %d, want 3", got)
	}
	if got := result.NextState.Plant.Payables[1].Amount; got != 3 {
		t.Fatalf("Trimmed payable amount = %d, want 3", got)
	}
	if got := result.Round.Metrics.CashReceipts; got != 0 {
		t.Fatalf("CashReceipts = %d, want 0", got)
	}
	if got := result.Round.Metrics.CashDisbursements; got != 5 {
		t.Fatalf("CashDisbursements = %d, want 5", got)
	}
	if got := result.Round.Metrics.NetCashChange; got != -5 {
		t.Fatalf("NetCashChange = %d, want -5", got)
	}
	if got := result.Round.Metrics.PayrollExpense; got != 0 {
		t.Fatalf("PayrollExpense = %d, want 0", got)
	}
	if got := result.Round.Metrics.RoundProfit; got != 10 {
		t.Fatalf("RoundProfit = %d, want 10", got)
	}
}

func TestResolverUsesSupplierLeadTimeAndReliabilityTerms(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		ProcurementTerms: func(ctx engine.ProcurementTermsContext) engine.ProcurementTerms {
			if ctx.Order.SupplierID == "slow" {
				return engine.ProcurementTerms{
					UnitCost:           1,
					LeadTimeRounds:     2,
					OnTimeDeliveryPct:  0,
					LateDeliveryRounds: 1,
					KnownSupplier:      true,
				}
			}
			return engine.ProcurementTerms{
				UnitCost:          3,
				LeadTimeRounds:    1,
				OnTimeDeliveryPct: 100,
				KnownSupplier:     true,
			}
		},
	})

	state := fixtureState()
	state.Suppliers = []domain.SupplierState{
		{SupplierID: "slow", DisplayName: "Slow Supply", OnTimeDeliveryPct: 0, LateDeliveryRounds: 1, ReliabilityScore: 80},
		{SupplierID: "fast", DisplayName: "Fast Supply", OnTimeDeliveryPct: 100, LateDeliveryRounds: 1, ReliabilityScore: 98},
	}
	actions := fixtureActions()
	actions[0].Action.Procurement.Orders = []domain.PurchaseOrderIntent{
		{PartID: "housing", SupplierID: "slow", Quantity: 1},
		{PartID: "housing", SupplierID: "fast", Quantity: 1},
	}

	first, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}
	if got := first.NextState.Plant.InTransitSupply[0].PromisedRound; got != 4 {
		t.Fatalf("slow supplier promised round = %d, want 4", got)
	}
	if got := first.NextState.Plant.InTransitSupply[0].ArrivalRound; got != 4 {
		t.Fatalf("slow supplier initial arrival round = %d, want 4", got)
	}

	second, err := resolver.ResolveRound(first.NextState, nil, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() second error = %v", err)
	}
	if got := len(second.NextState.Plant.InTransitSupply); got != 1 {
		t.Fatalf("InTransitSupply after fast receipt = %d, want 1", got)
	}
	if got := second.NextState.Plant.InTransitSupply[0].SupplierID; got != "slow" {
		t.Fatalf("remaining supplier = %q, want slow", got)
	}

	third, err := resolver.ResolveRound(second.NextState, nil, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() third error = %v", err)
	}
	if got := third.NextState.Plant.InTransitSupply[0].ArrivalRound; got != 5 {
		t.Fatalf("slow supplier delayed arrival round = %d, want 5", got)
	}
	if got := supplierScore(third.NextState.Suppliers, "slow"); got != 70 {
		t.Fatalf("slow supplier score = %d, want 70", got)
	}
	if !containsEvent(third.Round.Events, domain.EventSupplyDelayed) {
		t.Fatalf("Round.Events missing %q: %#v", domain.EventSupplyDelayed, third.Round.Events)
	}

	fourth, err := resolver.ResolveRound(third.NextState, nil, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() fourth error = %v", err)
	}
	if got := len(fourth.NextState.Plant.InTransitSupply); got != 0 {
		t.Fatalf("InTransitSupply after delayed receipt = %d, want 0", got)
	}
	if got := partQty(fourth.NextState.Plant.PartsInventory, "housing"); got != 6 {
		t.Fatalf("PartsInventory(housing) = %d, want 6", got)
	}
}

func TestResolverUsesConfiguredBacklogExpiryRounds(t *testing.T) {
	resolver := engine.NewResolver(engine.Options{
		BacklogExpiryRounds: 3,
	})
	state := fixtureState()
	state.Plant.Backlog = []domain.BacklogEntry{
		{CustomerID: "cust-1", ProductID: "widget", Quantity: 2, OriginRound: 1, AgeInRounds: 2},
	}
	state.Plant.FinishedInventory = nil
	state.Plant.InTransitSupply = nil

	actions := fixtureActions()
	actions[0].Action.Procurement.Orders = nil
	actions[1].Action.Production.Releases = nil
	actions[1].Action.Production.CapacityAllocation = nil
	actions[2].Action.Sales.ProductOffers = nil

	result, err := resolver.ResolveRound(state, actions, seeded.New(1))
	if err != nil {
		t.Fatalf("ResolveRound() error = %v", err)
	}

	if got := backlogQty(result.NextState.Plant.Backlog, "cust-1", "widget"); got != 2 {
		t.Fatalf("Backlog(cust-1/widget) = %d, want 2", got)
	}
	if got := result.Round.Metrics.LostSalesUnits; got != 0 {
		t.Fatalf("LostSalesUnits = %d, want 0", got)
	}

	for _, event := range result.Round.Events {
		if event.Type == domain.EventBacklogExpired {
			t.Fatalf("unexpected backlog expiry event: %#v", event)
		}
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
				{PartID: "housing", OnHandQty: 1, UnitCost: 1},
			},
			WIPInventory: []domain.WIPInventory{
				{ProductID: "widget", WorkstationID: "assembly", Quantity: 1, UnitCost: 1},
			},
			FinishedInventory: []domain.FinishedInventory{
				{ProductID: "widget", OnHandQty: 1, UnitCost: 1},
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
		Suppliers: []domain.SupplierState{
			{SupplierID: "supplier-a", DisplayName: "Supplier A", OnTimeDeliveryPct: 90, LateDeliveryRounds: 1, ReliabilityScore: 90},
			{SupplierID: "fast", DisplayName: "Fast Supply", OnTimeDeliveryPct: 100, LateDeliveryRounds: 1, ReliabilityScore: 98},
			{SupplierID: "slow", DisplayName: "Slow Supply", OnTimeDeliveryPct: 0, LateDeliveryRounds: 1, ReliabilityScore: 80},
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

func supplierScore(items []domain.SupplierState, supplierID domain.SupplierID) int {
	for _, item := range items {
		if item.SupplierID == supplierID {
			return item.ReliabilityScore
		}
	}
	return 0
}
func containsEvent(events []domain.RoundEvent, eventType domain.RoundEventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func widgetBOM(ctx engine.ProductionBOMContext) engine.ProductionBOM {
	if ctx.ProductID != "widget" {
		return engine.ProductionBOM{KnownProduct: true}
	}

	return engine.ProductionBOM{
		KnownProduct: true,
		Parts: []domain.BOMLine{
			{PartID: "housing", Quantity: 1},
		},
	}
}
