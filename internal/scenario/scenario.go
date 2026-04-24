package scenario

import (
	"fmt"
	"hash/fnv"
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
	FinanceModel        FinanceModel
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
	ID                 domain.CustomerID
	DisplayName        string
	Sentiment          int
	PaymentDelayRounds int
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
	Suppliers    []Supplier
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
	ID                   domain.PartID
	DisplayName          string
	UnitCost             domain.Money
	SupplierID           domain.SupplierID
	LeadTimeRounds       int
	MinimumOrderQuantity domain.Units
	PriceBreaks          []SupplierPriceBreak
	AlternateSuppliers   []SupplierOption
}

type SupplierOption struct {
	ID                   domain.SupplierID
	UnitCost             domain.Money
	LeadTimeRounds       int
	MinimumOrderQuantity domain.Units
	PriceBreaks          []SupplierPriceBreak
}

type SupplierPriceBreak struct {
	MinimumQuantity domain.Units
	UnitCost        domain.Money
}

type Supplier struct {
	ID             domain.SupplierID
	DisplayName    string
	BehaviorScript []SupplierBehavior
}

type SupplierBehavior struct {
	ID                 string
	DisplayName        string
	OnTimeDeliveryPct  int
	LateDeliveryRounds int
}

type Workstation struct {
	ID                          domain.WorkstationID
	DisplayName                 string
	CapacityPerRound            domain.CapacityUnits
	CostPerUnit                 domain.Money
	StressBufferUnits           domain.CapacityUnits
	StressPenaltyPerExcessUnit  domain.CapacityUnits
	LaborCapacityPerRound       domain.CapacityUnits
	LaborCostPerCapacityUnit    domain.Money
	OvertimeCostPerCapacityUnit domain.Money
}

type BottleneckAssumption struct {
	WorkstationID domain.WorkstationID
	Summary       string
}

type FinanceModel struct {
	ID                    string
	DisplayName           string
	Description           string
	ReceivableDelayRounds int
	PayableDelayRounds    int
	PayrollDelayRounds    int
}

type DemandAssumptions struct {
	BacklogExpiryRounds int
}

func NewDefinition(id domain.ScenarioID, displayName, description string, setup MatchSetup, starting StartingConditions, market MarketModel, production ProductionModel, finance FinanceModel) Definition {
	return Definition{
		ID:                  id,
		DisplayName:         displayName,
		Description:         description,
		Setup:               setup,
		StartingConditions:  starting,
		MarketModel:         market,
		ProductionModel:     production,
		FinanceModel:        finance,
		DefaultHistoryLimit: 10,
	}
}

func (d Definition) InitialState(matchID domain.MatchID, roles []domain.RoleAssignment) domain.MatchState {
	plant := d.StartingConditions.StartingPlant.Clone()
	plant.Workstations = d.productionWorkstationState()

	customers := make([]domain.CustomerState, 0, len(d.StartingConditions.Customers))
	for _, customer := range d.StartingConditions.Customers {
		customers = append(customers, domain.CustomerState{
			CustomerID:         customer.ID,
			DisplayName:        customer.DisplayName,
			Sentiment:          customer.Sentiment,
			PaymentDelayRounds: customer.PaymentDelayRounds,
			Backlog:            customerBacklog(plant.Backlog, customer.ID),
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
		Suppliers:     d.supplierState(matchID),
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
				return engine.ProcurementTerms{}
			}
			supplierState, ok := supplierStateByID(ctx.State.Suppliers, ctx.Order.SupplierID)
			if !ok {
				return engine.ProcurementTerms{}
			}
			supplier, ok := part.supplier(ctx.Order.SupplierID)
			if !ok {
				return engine.ProcurementTerms{}
			}
			return engine.ProcurementTerms{
				UnitCost:             supplier.unitCostForQuantity(ctx.Order.Quantity),
				MinimumOrderQuantity: supplier.MinimumOrderQuantity,
				LeadTimeRounds:       supplier.LeadTimeRounds,
				OnTimeDeliveryPct:    supplierState.OnTimeDeliveryPct,
				LateDeliveryRounds:   supplierState.LateDeliveryRounds,
				KnownSupplier:        true,
			}
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
		ReceivableDelayRounds: d.FinanceModel.ReceivableDelayRounds,
		PayableDelayRounds:    d.FinanceModel.PayableDelayRounds,
		PayrollDelayRounds:    d.FinanceModel.PayrollDelayRounds,
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
		fmt.Sprintf("finance_model=%s", d.FinanceModel.ID),
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
			WorkstationID:               workstation.ID,
			DisplayName:                 workstation.DisplayName,
			CapacityPerRound:            workstation.CapacityPerRound,
			EffectiveCapacityPerRound:   workstation.CapacityPerRound,
			StressBufferUnits:           workstation.StressBufferUnits,
			StressPenaltyPerExcessUnit:  workstation.StressPenaltyPerExcessUnit,
			LaborCapacityPerRound:       workstation.LaborCapacityPerRound,
			LaborCostPerCapacityUnit:    workstation.LaborCostPerCapacityUnit,
			OvertimeCostPerCapacityUnit: workstation.OvertimeCostPerCapacityUnit,
		})
	}
	return items
}

func (p Part) suppliers() []SupplierOption {
	items := []SupplierOption{
		{
			ID:                   p.SupplierID,
			UnitCost:             p.UnitCost,
			LeadTimeRounds:       normalizedLeadTime(p.LeadTimeRounds),
			MinimumOrderQuantity: p.MinimumOrderQuantity,
			PriceBreaks:          slices.Clone(p.PriceBreaks),
		},
	}
	items = append(items, slices.Clone(p.AlternateSuppliers)...)
	return items
}

func (p Part) supplier(supplierID domain.SupplierID) (SupplierOption, bool) {
	for _, item := range p.suppliers() {
		if item.ID == supplierID {
			item.LeadTimeRounds = normalizedLeadTime(item.LeadTimeRounds)
			return item, true
		}
	}
	return SupplierOption{}, false
}

func (o SupplierOption) unitCostForQuantity(quantity domain.Units) domain.Money {
	unitCost := o.UnitCost
	bestBreak := domain.Units(0)
	for _, item := range o.PriceBreaks {
		if item.MinimumQuantity <= 0 || quantity < item.MinimumQuantity || item.MinimumQuantity < bestBreak {
			continue
		}
		bestBreak = item.MinimumQuantity
		unitCost = item.UnitCost
	}
	return unitCost
}

func (d Definition) supplierState(matchID domain.MatchID) []domain.SupplierState {
	suppliers := d.ProductionModel.Suppliers
	items := make([]domain.SupplierState, 0, len(suppliers))
	for _, supplier := range suppliers {
		behavior := supplier.selectBehavior(matchID)
		items = append(items, domain.SupplierState{
			SupplierID:         supplier.ID,
			DisplayName:        supplier.DisplayName,
			BehaviorID:         behavior.ID,
			BehaviorDisplay:    behavior.DisplayName,
			OnTimeDeliveryPct:  normalizedOnTimePct(behavior.OnTimeDeliveryPct),
			LateDeliveryRounds: normalizedLateRounds(behavior.LateDeliveryRounds),
			ReliabilityScore:   normalizedOnTimePct(behavior.OnTimeDeliveryPct),
		})
	}
	return items
}

func (s Supplier) selectBehavior(matchID domain.MatchID) SupplierBehavior {
	if len(s.BehaviorScript) == 0 {
		return SupplierBehavior{ID: "steady", DisplayName: "Steady", OnTimeDeliveryPct: 100}
	}
	index := supplierBehaviorIndex(matchID, s.ID, len(s.BehaviorScript))
	behavior := s.BehaviorScript[index]
	behavior.OnTimeDeliveryPct = normalizedOnTimePct(behavior.OnTimeDeliveryPct)
	behavior.LateDeliveryRounds = normalizedLateRounds(behavior.LateDeliveryRounds)
	return behavior
}

func supplierBehaviorIndex(matchID domain.MatchID, supplierID domain.SupplierID, length int) int {
	if length <= 1 {
		return 0
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(matchID))
	_, _ = hasher.Write([]byte(":"))
	_, _ = hasher.Write([]byte(supplierID))
	return int(hasher.Sum32() % uint32(length))
}

func supplierStateByID(items []domain.SupplierState, supplierID domain.SupplierID) (domain.SupplierState, bool) {
	for _, item := range items {
		if item.SupplierID == supplierID {
			return item, true
		}
	}
	return domain.SupplierState{}, false
}

func normalizedLeadTime(rounds int) int {
	if rounds <= 0 {
		return 1
	}
	return rounds
}

func normalizedOnTimePct(pct int) int {
	if pct < 0 {
		return 100
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func normalizedLateRounds(rounds int) int {
	if rounds < 0 {
		return 0
	}
	return rounds
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
