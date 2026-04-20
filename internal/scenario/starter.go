package scenario

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
)

const (
	// StarterID is the default playable scenario shipped with the MVP.
	StarterID domain.ScenarioID = "starter"
)

type Definition struct {
	ID                  domain.ScenarioID
	DisplayName         string
	Description         string
	RoleRoster          []domain.RoleID
	Products            []Product
	Parts               []Part
	Customers           []Customer
	Workstations        []Workstation
	Bottleneck          BottleneckAssumption
	StartingTargets     domain.BudgetTargets
	StartingPlant       domain.PlantState
	DemandAssumptions   DemandAssumptions
	DebtCeiling         domain.Money
	DefaultHistoryLimit int
}

type Product struct {
	ID           domain.ProductID
	DisplayName  string
	BOM          []domain.BOMLine
	Route        []domain.WorkstationID
	BaseUnitCost domain.Money
}

type Part struct {
	ID          domain.PartID
	DisplayName string
	UnitCost    domain.Money
	SupplierID  domain.SupplierID
}

type Customer struct {
	ID              domain.CustomerID
	DisplayName     string
	StartingMood    int
	DemandByProduct map[domain.ProductID]DemandProfile
}

type DemandProfile struct {
	ReferencePrice   domain.Money
	BaseDemand       domain.Units
	PriceSensitivity int
}

type Workstation struct {
	ID               domain.WorkstationID
	DisplayName      string
	CapacityPerRound domain.CapacityUnits
	CostPerUnit      domain.Money
}

type BottleneckAssumption struct {
	WorkstationID domain.WorkstationID
	Summary       string
}

type DemandAssumptions struct {
	BacklogExpiryRounds int
}

func Default() Definition {
	return Starter()
}

func Starter() Definition {
	return Definition{
		ID:          StarterID,
		DisplayName: "Prairie Pump Starter Plant",
		Description: "A small pump-and-valve plant with constrained final assembly, limited cash, and enough demand pressure to create local-vs-global tension immediately.",
		RoleRoster:  domain.CanonicalRoles(),
		Parts: []Part{
			{ID: "housing", DisplayName: "Housing", UnitCost: 3, SupplierID: "forgeco"},
			{ID: "seal_kit", DisplayName: "Seal Kit", UnitCost: 2, SupplierID: "sealworks"},
			{ID: "body", DisplayName: "Valve Body", UnitCost: 2, SupplierID: "forgeco"},
			{ID: "fastener_kit", DisplayName: "Fastener Kit", UnitCost: 1, SupplierID: "fastenall"},
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
		Customers: []Customer{
			{
				ID:           "northbuild",
				DisplayName:  "NorthBuild",
				StartingMood: 6,
				DemandByProduct: map[domain.ProductID]DemandProfile{
					"pump":  {ReferencePrice: 14, BaseDemand: 3, PriceSensitivity: 1},
					"valve": {ReferencePrice: 9, BaseDemand: 2, PriceSensitivity: 1},
				},
			},
			{
				ID:           "prairieflow",
				DisplayName:  "PrairieFlow",
				StartingMood: 5,
				DemandByProduct: map[domain.ProductID]DemandProfile{
					"pump":  {ReferencePrice: 13, BaseDemand: 2, PriceSensitivity: 1},
					"valve": {ReferencePrice: 8, BaseDemand: 3, PriceSensitivity: 1},
				},
			},
			{
				ID:           "agriworks",
				DisplayName:  "AgriWorks",
				StartingMood: 4,
				DemandByProduct: map[domain.ProductID]DemandProfile{
					"pump":  {ReferencePrice: 12, BaseDemand: 2, PriceSensitivity: 1},
					"valve": {ReferencePrice: 8, BaseDemand: 2, PriceSensitivity: 1},
				},
			},
		},
		Workstations: []Workstation{
			{ID: "fabrication", DisplayName: "Fabrication", CapacityPerRound: 7, CostPerUnit: 2},
			{ID: "assembly", DisplayName: "Assembly", CapacityPerRound: 4, CostPerUnit: 3},
		},
		Bottleneck: BottleneckAssumption{
			WorkstationID: "assembly",
			Summary:       "Assembly is the chronic near-term bottleneck, so overselling or over-releasing work tends to inflate WIP and burn cash faster than throughput improves.",
		},
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
		DemandAssumptions: DemandAssumptions{
			BacklogExpiryRounds: 2,
		},
		DefaultHistoryLimit: 10,
	}
}

func (d Definition) InitialState(matchID domain.MatchID, roles []domain.RoleAssignment) domain.MatchState {
	customers := make([]domain.CustomerState, 0, len(d.Customers))
	for _, customer := range d.Customers {
		customers = append(customers, domain.CustomerState{
			CustomerID:  customer.ID,
			DisplayName: customer.DisplayName,
			Sentiment:   customer.StartingMood,
			Backlog:     customerBacklog(d.StartingPlant.Backlog, customer.ID),
		})
	}

	return domain.MatchState{
		MatchID:       matchID,
		ScenarioID:    d.ID,
		CurrentRound:  1,
		Roles:         slices.Clone(roles),
		Plant:         d.StartingPlant.Clone(),
		Customers:     customers,
		ActiveTargets: d.StartingTargets,
	}
}

func (d Definition) ResolverOptions() engine.Options {
	partByID := d.partsByID()
	productByID := d.productsByID()
	workstationByID := d.workstationsByID()

	return engine.Options{
		HistoryLimit: d.DefaultHistoryLimit,
		ProcurementTerms: func(ctx engine.ProcurementTermsContext) engine.ProcurementTerms {
			part, ok := partByID[ctx.Order.PartID]
			if !ok {
				return engine.ProcurementTerms{UnitCost: 1}
			}
			return engine.ProcurementTerms{UnitCost: part.UnitCost}
		},
		ProductionBOM: func(ctx engine.ProductionBOMContext) engine.ProductionBOM {
			product, ok := productByID[ctx.ProductID]
			if !ok {
				return engine.ProductionBOM{}
			}
			return engine.ProductionBOM{
				KnownProduct: true,
				Parts:        slices.Clone(product.BOM),
			}
		},
		ProductionRoute: func(ctx engine.ProductionRouteContext) engine.ProductionRouteStep {
			product, ok := productByID[ctx.ProductID]
			if !ok {
				return engine.ProductionRouteStep{}
			}

			index := slices.Index(product.Route, ctx.CurrentStationID)
			if index < 0 {
				return engine.ProductionRouteStep{}
			}
			if index == len(product.Route)-1 {
				return engine.ProductionRouteStep{Finished: true}
			}
			return engine.ProductionRouteStep{NextWorkstationID: product.Route[index+1]}
		},
		ProductionCost: func(ctx engine.ProductionCostContext) engine.ProductionCost {
			workstation, ok := workstationByID[ctx.WorkstationID]
			if !ok || workstation.CostPerUnit <= 0 {
				return engine.ProductionCost{CostPerCapacityUnit: 1}
			}
			return engine.ProductionCost{CostPerCapacityUnit: workstation.CostPerUnit}
		},
		WorldUpdate: func(ctx *engine.WorldUpdateContext) error {
			return d.applyDemand(ctx)
		},
	}
}

func (d Definition) SummaryLines() []string {
	lines := []string{
		fmt.Sprintf("scenario=%s (%s)", d.ID, d.DisplayName),
		fmt.Sprintf("bottleneck=%s", d.Bottleneck.WorkstationID),
		fmt.Sprintf("roles=%s", strings.Join(roleNames(d.RoleRoster), ", ")),
	}
	return lines
}

func (d Definition) partsByID() map[domain.PartID]Part {
	items := make(map[domain.PartID]Part, len(d.Parts))
	for _, item := range d.Parts {
		items[item.ID] = item
	}
	return items
}

func (d Definition) productsByID() map[domain.ProductID]Product {
	items := make(map[domain.ProductID]Product, len(d.Products))
	for _, item := range d.Products {
		items[item.ID] = item
	}
	return items
}

func (d Definition) workstationsByID() map[domain.WorkstationID]Workstation {
	items := make(map[domain.WorkstationID]Workstation, len(d.Workstations))
	for _, item := range d.Workstations {
		items[item.ID] = item
	}
	return items
}

func (d Definition) customersByID() map[domain.CustomerID]Customer {
	items := make(map[domain.CustomerID]Customer, len(d.Customers))
	for _, item := range d.Customers {
		items[item.ID] = item
	}
	return items
}

func (d Definition) applyDemand(ctx *engine.WorldUpdateContext) error {
	if ctx == nil || ctx.State == nil {
		return nil
	}

	customerByID := d.customersByID()
	priceByProduct := currentOfferPrices(ctx.Round, d.Products)

	for _, customerState := range ctx.State.Customers {
		customer, ok := customerByID[customerState.CustomerID]
		if !ok {
			continue
		}

		for productID, offerPrice := range priceByProduct {
			profile, ok := customer.DemandByProduct[productID]
			if !ok {
				continue
			}

			realized := realizedDemand(profile, customerState.Sentiment, offerPrice)
			if realized <= 0 {
				continue
			}

			entry := domain.BacklogEntry{
				CustomerID:  customerState.CustomerID,
				ProductID:   productID,
				Quantity:    realized,
				OriginRound: ctx.State.CurrentRound,
			}
			ctx.State.Plant.Backlog = append(ctx.State.Plant.Backlog, entry)
			appendCustomerBacklog(ctx.State, customerState.CustomerID, entry)

			ctx.AppendEvent(domain.EventDemandRealized, domain.ActorPlantSystem, fmt.Sprintf("Realized %d units of %s demand from %s", realized, productID, customerState.CustomerID), map[string]any{
				"customer_id":        string(customerState.CustomerID),
				"product_id":         string(productID),
				"quantity":           int(realized),
				"reference_price":    int(profile.ReferencePrice),
				"offered_unit_price": int(offerPrice),
				"sentiment":          customerState.Sentiment,
			})
			ctx.AppendEvent(domain.EventBacklogCreated, domain.ActorPlantSystem, fmt.Sprintf("Booked %d units of %s backlog for %s", realized, productID, customerState.CustomerID), map[string]any{
				"customer_id": string(customerState.CustomerID),
				"product_id":  string(productID),
				"quantity":    int(realized),
			})
		}
	}

	return nil
}

func currentOfferPrices(round *domain.RoundRecord, products []Product) map[domain.ProductID]domain.Money {
	prices := make(map[domain.ProductID]domain.Money, len(products))
	for _, product := range products {
		prices[product.ID] = product.BaseUnitCost * 2
	}

	if round == nil {
		return prices
	}

	for _, action := range round.Actions {
		if action.RoleID != domain.RoleSalesManager || action.Action.Sales == nil {
			continue
		}
		for _, offer := range action.Action.Sales.ProductOffers {
			if offer.UnitPrice > 0 {
				prices[offer.ProductID] = offer.UnitPrice
			}
		}
	}

	return prices
}

func realizedDemand(profile DemandProfile, sentiment int, offeredPrice domain.Money) domain.Units {
	sentimentModifier := max(1, sentiment) + 2
	demandScore := int(profile.BaseDemand)*sentimentModifier - profile.PriceSensitivity*max(0, int(offeredPrice-profile.ReferencePrice))
	return domain.Units(max(0, demandScore/5))
}

func appendCustomerBacklog(state *domain.MatchState, customerID domain.CustomerID, entry domain.BacklogEntry) {
	for index := range state.Customers {
		if state.Customers[index].CustomerID != customerID {
			continue
		}
		state.Customers[index].Backlog = append(state.Customers[index].Backlog, entry)
		return
	}
}

func customerBacklog(backlog []domain.BacklogEntry, customerID domain.CustomerID) []domain.BacklogEntry {
	items := make([]domain.BacklogEntry, 0)
	for _, entry := range backlog {
		if entry.CustomerID == customerID {
			items = append(items, entry)
		}
	}
	return items
}

func roleNames(roleIDs []domain.RoleID) []string {
	names := make([]string, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		names = append(names, string(roleID))
	}
	return names
}
