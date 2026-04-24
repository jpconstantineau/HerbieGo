package scenario

import "github.com/jpconstantineau/herbiego/internal/domain"

const (
	// StarterID is the default playable scenario shipped with the MVP.
	StarterID domain.ScenarioID = "starter"
)

func Default() Definition {
	return Starter()
}

func Starter() Definition {
	return NewDefinition(
		StarterID,
		"Prairie Pump Starter Plant",
		"A small pump-and-valve plant with constrained final assembly, limited cash, and enough demand pressure to create local-vs-global tension immediately.",
		StarterMatchSetup(),
		StarterStartingConditions(),
		StarterMarketModel(),
		StarterProductionModel(),
		StarterFinanceModel(),
	)
}

func StarterMatchSetup() MatchSetup {
	return MatchSetup{
		ID:          "core_four_roles",
		DisplayName: "Core Four Roles",
		RoleRoster:  domain.CanonicalRoles(),
	}
}

func StarterStartingConditions() StartingConditions {
	return StartingConditions{
		ID:          "prairie_bootstrap",
		DisplayName: "Prairie Bootstrap",
		Description: "A solvent but cash-tight plant that starts with some WIP, one finished valve, and a small backlog already waiting.",
		StartingTargets: domain.BudgetTargets{
			EffectiveRound:        1,
			ProcurementBudget:     18,
			ProductionSpendBudget: 14,
			RevenueTarget:         28,
			CashFloorTarget:       8,
			DebtCeilingTarget:     15,
		},
		StartingPlant: domain.PlantState{
			Cash:        24,
			Debt:        0,
			DebtCeiling: 15,
			PartsInventory: []domain.PartInventory{
				{PartID: "housing", OnHandQty: 2, UnitCost: 3},
				{PartID: "seal_kit", OnHandQty: 2, UnitCost: 2},
				{PartID: "body", OnHandQty: 3, UnitCost: 2},
				{PartID: "fastener_kit", OnHandQty: 3, UnitCost: 1},
			},
			WIPInventory: []domain.WIPInventory{
				{ProductID: "pump", WorkstationID: "assembly", Quantity: 2, UnitCost: 8},
			},
			FinishedInventory: []domain.FinishedInventory{
				{ProductID: "valve", OnHandQty: 1, UnitCost: 5},
			},
			Workstations: []domain.WorkstationState{
				{WorkstationID: "fabrication", DisplayName: "Fabrication", CapacityPerRound: 7},
				{WorkstationID: "assembly", DisplayName: "Assembly", CapacityPerRound: 4},
			},
			Backlog: []domain.BacklogEntry{
				{CustomerID: "northbuild", ProductID: "pump", Quantity: 2, OriginRound: 0},
				{CustomerID: "prairieflow", ProductID: "valve", Quantity: 1, OriginRound: 0},
			},
		},
		Customers: []CustomerSeed{
			{ID: "northbuild", DisplayName: "NorthBuild", Sentiment: 6, PaymentDelayRounds: 2},
			{ID: "prairieflow", DisplayName: "PrairieFlow", Sentiment: 5, PaymentDelayRounds: 1},
			{ID: "agriworks", DisplayName: "AgriWorks", Sentiment: 4, PaymentDelayRounds: 1},
		},
	}
}

func StarterMarketModel() MarketModel {
	return MarketModel{
		ID:          "regional_weekly_demand",
		DisplayName: "Regional Weekly Demand",
		Description: "Demand reacts to offered prices and customer sentiment, then rolls into next-round backlog.",
		Customers: []CustomerMarket{
			{
				ID:          "northbuild",
				DisplayName: "NorthBuild",
				DemandByProduct: map[domain.ProductID]DemandProfile{
					"pump":  {ReferencePrice: 14, BaseDemand: 3, PriceSensitivity: 1},
					"valve": {ReferencePrice: 9, BaseDemand: 2, PriceSensitivity: 1},
				},
			},
			{
				ID:          "prairieflow",
				DisplayName: "PrairieFlow",
				DemandByProduct: map[domain.ProductID]DemandProfile{
					"pump":  {ReferencePrice: 13, BaseDemand: 2, PriceSensitivity: 1},
					"valve": {ReferencePrice: 8, BaseDemand: 3, PriceSensitivity: 1},
				},
			},
			{
				ID:          "agriworks",
				DisplayName: "AgriWorks",
				DemandByProduct: map[domain.ProductID]DemandProfile{
					"pump":  {ReferencePrice: 12, BaseDemand: 2, PriceSensitivity: 1},
					"valve": {ReferencePrice: 8, BaseDemand: 2, PriceSensitivity: 1},
				},
			},
		},
		DemandAssumptions: DemandAssumptions{
			BacklogExpiryRounds: 2,
		},
	}
}

func StarterProductionModel() ProductionModel {
	return ProductionModel{
		ID:          "two_stage_pump_valve_line",
		DisplayName: "Two-Stage Pump/Valve Line",
		Description: "Two products share fabrication and assembly, with assembly intentionally sized as the tighter capacity pool.",
		Parts: []Part{
			{
				ID:                 "housing",
				DisplayName:        "Housing",
				UnitCost:           3,
				SupplierID:         "forgeco",
				LeadTimeRounds:     2,
				OnTimeDeliveryPct:  70,
				LateDeliveryRounds: 1,
				AlternateSuppliers: []SupplierOption{
					{ID: "prairiefast", UnitCost: 5, LeadTimeRounds: 1, OnTimeDeliveryPct: 100},
				},
			},
			{
				ID:                "seal_kit",
				DisplayName:       "Seal Kit",
				UnitCost:          2,
				SupplierID:        "sealworks",
				LeadTimeRounds:    1,
				OnTimeDeliveryPct: 95,
				AlternateSuppliers: []SupplierOption{
					{ID: "prairiefast", UnitCost: 3, LeadTimeRounds: 1, OnTimeDeliveryPct: 100},
				},
			},
			{
				ID:                 "body",
				DisplayName:        "Valve Body",
				UnitCost:           2,
				SupplierID:         "forgeco",
				LeadTimeRounds:     2,
				OnTimeDeliveryPct:  70,
				LateDeliveryRounds: 1,
				AlternateSuppliers: []SupplierOption{
					{ID: "prairiefast", UnitCost: 4, LeadTimeRounds: 1, OnTimeDeliveryPct: 100},
				},
			},
			{
				ID:                 "fastener_kit",
				DisplayName:        "Fastener Kit",
				UnitCost:           1,
				SupplierID:         "fastenall",
				LeadTimeRounds:     1,
				OnTimeDeliveryPct:  90,
				LateDeliveryRounds: 1,
			},
		},
		Products: []Product{
			{
				ID:          "pump",
				DisplayName: "Pump",
				BOM: []domain.BOMLine{
					{PartID: "housing", Quantity: 1},
					{PartID: "seal_kit", Quantity: 1},
				},
				Route:        []domain.WorkstationID{"fabrication", "assembly"},
				BaseUnitCost: 8,
			},
			{
				ID:          "valve",
				DisplayName: "Valve",
				BOM: []domain.BOMLine{
					{PartID: "body", Quantity: 1},
					{PartID: "fastener_kit", Quantity: 1},
				},
				Route:        []domain.WorkstationID{"fabrication", "assembly"},
				BaseUnitCost: 5,
			},
		},
		Workstations: []Workstation{
			{ID: "fabrication", DisplayName: "Fabrication", CapacityPerRound: 7, CostPerUnit: 2, StressBufferUnits: 2, StressPenaltyPerExcessUnit: 1},
			{ID: "assembly", DisplayName: "Assembly", CapacityPerRound: 4, CostPerUnit: 3, StressBufferUnits: 1, StressPenaltyPerExcessUnit: 1},
		},
		Bottleneck: BottleneckAssumption{
			WorkstationID: "assembly",
			Summary:       "Assembly is the chronic near-term bottleneck, so overselling or over-releasing work tends to inflate WIP and burn cash faster than throughput improves.",
		},
	}
}

func StarterFinanceModel() FinanceModel {
	return FinanceModel{
		ID:                    "net30-lite-weekly",
		DisplayName:           "Net-30 Lite Weekly Cash Timing",
		Description:           "Customer receipts settle on customer-specific terms, while supplier invoices settle one round later to keep near-term cash pressure visible.",
		ReceivableDelayRounds: 1,
		PayableDelayRounds:    1,
	}
}
