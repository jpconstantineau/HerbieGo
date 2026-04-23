package scenario

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
)

type Definition struct {
	ID                  domain.ScenarioID
	DisplayName         string
	Description         string
	Setup               MatchSetup
	StartingConditions  StartingConditions
	MarketModel         MarketModel
	ProductionModel     ProductionModel
	DefaultHistoryLimit int
}

type MatchSetup struct {
	ID          string
	DisplayName string
	RoleRoster  []domain.RoleID
}

type StartingConditions struct {
	ID              string
	DisplayName     string
	Description     string
	StartingTargets domain.BudgetTargets
	StartingPlant   domain.PlantState
	Customers       []CustomerSeed
}

type CustomerSeed struct {
	ID          domain.CustomerID
	DisplayName string
	Sentiment   int
}

type MarketModel struct {
	ID                string
	DisplayName       string
	Description       string
	Customers         []CustomerMarket
	DemandAssumptions DemandAssumptions
}

type CustomerMarket struct {
	ID              domain.CustomerID
	DisplayName     string
	DemandByProduct map[domain.ProductID]DemandProfile
}

type DemandProfile struct {
	ReferencePrice   domain.Money
	BaseDemand       domain.Units
	PriceSensitivity int
}

type ProductionModel struct {
	ID           string
	DisplayName  string
	Description  string
	Products     []Product
	Parts        []Part
	Workstations []Workstation
	Bottleneck   BottleneckAssumption
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

func NewDefinition(id domain.ScenarioID, displayName, description string, setup MatchSetup, starting StartingConditions, market MarketModel, production ProductionModel) Definition {
	return Definition{
		ID:                  id,
		DisplayName:         displayName,
		Description:         description,
		Setup:               setup,
		StartingConditions:  starting,
		MarketModel:         market,
		ProductionModel:     production,
		DefaultHistoryLimit: 10,
	}
}

func (d Definition) InitialState(matchID domain.MatchID, roles []domain.RoleAssignment) domain.MatchState {
	plant := d.StartingConditions.StartingPlant.Clone()
	plant.Workstations = d.productionWorkstationState()

	customers := make([]domain.CustomerState, 0, len(d.StartingConditions.Customers))
	for _, customer := range d.StartingConditions.Customers {
		customers = append(customers, domain.CustomerState{
			CustomerID:  customer.ID,
			DisplayName: customer.DisplayName,
			Sentiment:   customer.Sentiment,
			Backlog:     customerBacklog(plant.Backlog, customer.ID),
		})
	}

	return domain.MatchState{
		MatchID:      matchID,
		ScenarioID:   d.ID,
		CurrentRound: 1,
		Roles:        slices.Clone(roles),
		RoundFlow: domain.RoundFlowState{
			Phase:          domain.RoundPhaseCollecting,
			WaitingOnRoles: roleIDs(roles),
		},
		Plant:         plant,
		Customers:     customers,
		ActiveTargets: d.StartingConditions.StartingTargets,
	}
}

func (d Definition) ResolverOptions() engine.Options {
	partByID := d.partsByID()
	productByID := d.productsByID()
	workstationByID := d.workstationsByID()

	return engine.Options{
		HistoryLimit:        d.DefaultHistoryLimit,
		BacklogExpiryRounds: d.MarketModel.DemandAssumptions.BacklogExpiryRounds,
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
	return []string{
		fmt.Sprintf("scenario=%s (%s)", d.ID, d.DisplayName),
		fmt.Sprintf("setup=%s", d.Setup.ID),
		fmt.Sprintf("starting_conditions=%s", d.StartingConditions.ID),
		fmt.Sprintf("market_model=%s", d.MarketModel.ID),
		fmt.Sprintf("production_model=%s", d.ProductionModel.ID),
		fmt.Sprintf("bottleneck=%s", d.ProductionModel.Bottleneck.WorkstationID),
		fmt.Sprintf("roles=%s", strings.Join(roleNames(d.Setup.RoleRoster), ", ")),
	}
}

func (d Definition) partsByID() map[domain.PartID]Part {
	items := make(map[domain.PartID]Part, len(d.ProductionModel.Parts))
	for _, item := range d.ProductionModel.Parts {
		items[item.ID] = item
	}
	return items
}

func (d Definition) productsByID() map[domain.ProductID]Product {
	items := make(map[domain.ProductID]Product, len(d.ProductionModel.Products))
	for _, item := range d.ProductionModel.Products {
		items[item.ID] = item
	}
	return items
}

func (d Definition) workstationsByID() map[domain.WorkstationID]Workstation {
	items := make(map[domain.WorkstationID]Workstation, len(d.ProductionModel.Workstations))
	for _, item := range d.ProductionModel.Workstations {
		items[item.ID] = item
	}
	return items
}

func (d Definition) customersByID() map[domain.CustomerID]CustomerMarket {
	items := make(map[domain.CustomerID]CustomerMarket, len(d.MarketModel.Customers))
	for _, item := range d.MarketModel.Customers {
		items[item.ID] = item
	}
	return items
}

func (d Definition) productionWorkstationState() []domain.WorkstationState {
	items := make([]domain.WorkstationState, 0, len(d.ProductionModel.Workstations))
	for _, workstation := range d.ProductionModel.Workstations {
		items = append(items, domain.WorkstationState{
			WorkstationID:    workstation.ID,
			DisplayName:      workstation.DisplayName,
			CapacityPerRound: workstation.CapacityPerRound,
		})
	}
	return items
}

func (d Definition) applyDemand(ctx *engine.WorldUpdateContext) error {
	if ctx == nil || ctx.State == nil {
		return nil
	}

	customerByID := d.customersByID()
	priceByProduct := currentOfferPrices(ctx.Round, d.ProductionModel.Products)

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

func roleIDs(assignments []domain.RoleAssignment) []domain.RoleID {
	ids := make([]domain.RoleID, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.RoleID)
	}
	return ids
}
