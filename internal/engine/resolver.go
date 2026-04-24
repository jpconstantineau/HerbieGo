package engine

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

type Options struct {
	HistoryLimit          int
	BacklogExpiryRounds   int
	ProcurementTerms      ProcurementTermsHook
	ProductionBOM         ProductionBOMHook
	ProductionRoute       ProductionRouteHook
	ProductionCost        ProductionCostHook
	InventoryCost         InventoryCarryingCostHook
	ReceivableDelayRounds int
	PayableDelayRounds    int
	WorldUpdate           WorldUpdateHook
}

// ProcurementTermsHook lets scenario data supply part cost, MOQ, and quantity-based pricing rules.
// The default implementation uses a unit cost fallback of 1 with no MOQ.
type ProcurementTermsHook func(ProcurementTermsContext) ProcurementTerms

type ProcurementTermsContext struct {
	State        domain.MatchState
	CurrentRound domain.RoundNumber
	Order        domain.PurchaseOrderIntent
}

type ProcurementTerms struct {
	UnitCost             domain.Money
	MinimumOrderQuantity domain.Units
	LeadTimeRounds       int
	OnTimeDeliveryPct    int
	LateDeliveryRounds   int
	KnownSupplier        bool
}

// ProductionBOMHook lets scenario data define the parts consumed when a product is released.
// The default implementation assumes products consume no explicit parts until scenario data is provided.
type ProductionBOMHook func(ProductionBOMContext) ProductionBOM

type ProductionBOMContext struct {
	State        domain.MatchState
	CurrentRound domain.RoundNumber
	ProductID    domain.ProductID
}

type ProductionBOM struct {
	KnownProduct bool
	Parts        []domain.BOMLine
}

// ProductionRouteHook lets scenario data define product-specific routing semantics.
// The default implementation advances through Plant.Workstations in their declared order.
type ProductionRouteHook func(ProductionRouteContext) ProductionRouteStep

type ProductionRouteContext struct {
	State            domain.MatchState
	CurrentRound     domain.RoundNumber
	ProductID        domain.ProductID
	CurrentStationID domain.WorkstationID
}

type ProductionRouteStep struct {
	NextWorkstationID domain.WorkstationID
	Finished          bool
}

// ProductionCostHook lets scenario data define operating expense per unit of advanced capacity.
// The default implementation charges 1 money unit per unit of advanced work.
type ProductionCostHook func(ProductionCostContext) ProductionCost

type ProductionCostContext struct {
	State         domain.MatchState
	CurrentRound  domain.RoundNumber
	ProductID     domain.ProductID
	WorkstationID domain.WorkstationID
	Quantity      domain.CapacityUnits
}

type ProductionCost struct {
	CostPerCapacityUnit domain.Money
}

// InventoryCarryingCostHook lets scenario data define carrying cost by inventory class.
// The default implementation charges a flat 10% round-end carrying cost with a minimum of 1.
type InventoryCarryingCostHook func(InventoryCarryingCostContext) domain.Money

type InventoryCarryingCostContext struct {
	State          domain.MatchState
	CurrentRound   domain.RoundNumber
	InventoryClass InventoryClass
	InventoryValue domain.Money
}

type InventoryClass string

const (
	InventoryClassParts    InventoryClass = "parts"
	InventoryClassWIP      InventoryClass = "wip"
	InventoryClassFinished InventoryClass = "finished"
)

// WorldUpdateHook is the extension point for scenario-owned demand generation, market behavior,
// and other seeded world events that should stay outside the deterministic core resolver.
type WorldUpdateHook func(*WorldUpdateContext) error

type WorldUpdateContext struct {
	State       *domain.MatchState
	Round       *domain.RoundRecord
	Random      ports.RandomSource
	AppendEvent func(eventType domain.RoundEventType, actorID domain.ActorID, summary string, payload map[string]any)
}

type Result struct {
	NextState domain.MatchState
	Round     domain.RoundRecord
}

type Resolver struct {
	historyLimit          int
	backlogExpiryRounds   int
	procurementTerms      ProcurementTermsHook
	productionBOM         ProductionBOMHook
	productionRoute       ProductionRouteHook
	productionCost        ProductionCostHook
	inventoryCost         InventoryCarryingCostHook
	receivableDelayRounds int
	payableDelayRounds    int
	worldUpdate           WorldUpdateHook
}

func NewResolver(options Options) *Resolver {
	return &Resolver{
		historyLimit:          normalizedHistoryLimit(options.HistoryLimit),
		backlogExpiryRounds:   normalizedBacklogExpiryRounds(options.BacklogExpiryRounds),
		procurementTerms:      defaultProcurementTerms(options.ProcurementTerms),
		productionBOM:         defaultProductionBOM(options.ProductionBOM),
		productionRoute:       defaultProductionRoute(options.ProductionRoute),
		productionCost:        defaultProductionCost(options.ProductionCost),
		inventoryCost:         options.InventoryCost,
		receivableDelayRounds: normalizedDelayRounds(options.ReceivableDelayRounds),
		payableDelayRounds:    normalizedDelayRounds(options.PayableDelayRounds),
		worldUpdate:           options.WorldUpdate,
	}
}

func (r *Resolver) ResolveRound(state domain.MatchState, actions []domain.ActionSubmission, random ports.RandomSource) (Result, error) {
	if state.MatchID == "" {
		return Result{}, errors.New("engine: state match id must not be empty")
	}
	if state.CurrentRound <= 0 {
		return Result{}, fmt.Errorf("engine: state round %d must be positive", state.CurrentRound)
	}

	orderedActions, err := normalizeActions(state, actions)
	if err != nil {
		return Result{}, err
	}

	nextState := state.Clone()
	nextState.Plant.Workstations = resetCapacityUsage(nextState.Plant.Workstations)

	round := domain.RoundRecord{
		Round:      state.CurrentRound,
		Actions:    cloneActions(orderedActions),
		Commentary: collectCommentary(state, orderedActions),
	}

	phase := roundPhase{
		state:                 &nextState,
		round:                 &round,
		currentRound:          state.CurrentRound,
		beginningCash:         state.Plant.Cash,
		stats:                 &resolutionStats{},
		backlogExpiryRounds:   r.backlogExpiryRounds,
		procurementTerms:      r.procurementTerms,
		productionBOM:         r.productionBOM,
		productionRoute:       r.productionRoute,
		productionCost:        r.productionCost,
		inventoryCost:         r.inventoryCost,
		receivableDelayRounds: r.receivableDelayRounds,
		payableDelayRounds:    r.payableDelayRounds,
		stressedStations:      map[domain.WorkstationID]bool{},
		random:                random,
	}

	if nextState.ActiveTargets.EffectiveRound == state.CurrentRound {
		phase.appendEvent(domain.EventBudgetActivated, domain.ActorPlantSystem, "Activated current round targets", map[string]any{
			"effective_round":     int(nextState.ActiveTargets.EffectiveRound),
			"procurement_budget":  int(nextState.ActiveTargets.ProcurementBudget),
			"production_budget":   int(nextState.ActiveTargets.ProductionSpendBudget),
			"revenue_target":      int(nextState.ActiveTargets.RevenueTarget),
			"cash_floor_target":   int(nextState.ActiveTargets.CashFloorTarget),
			"debt_ceiling_target": int(nextState.ActiveTargets.DebtCeilingTarget),
		})
	}

	phase.resolveProcurement(actionForRole(orderedActions, domain.RoleProcurementManager))
	phase.receiveSupply()
	phase.resolveProduction(actionForRole(orderedActions, domain.RoleProductionManager))
	phase.resolveSales(actionForRole(orderedActions, domain.RoleSalesManager))

	if r.worldUpdate != nil {
		if err := r.worldUpdate(&WorldUpdateContext{
			State:  &nextState,
			Round:  &round,
			Random: random,
			AppendEvent: func(eventType domain.RoundEventType, actorID domain.ActorID, summary string, payload map[string]any) {
				phase.appendEvent(eventType, actorID, summary, payload)
			},
		}); err != nil {
			return Result{}, fmt.Errorf("engine: world update: %w", err)
		}
	}

	phase.finalizeRound(actionForRole(orderedActions, domain.RoleFinanceController))
	round.Timeline = round.CanonicalTimeline()

	nextState.Metrics = round.Metrics
	nextState.CurrentRound++
	nextState.History = appendRecentRound(nextState.History, round, r.historyLimit)

	return Result{
		NextState: nextState.Clone(),
		Round:     round.Clone(),
	}, nil
}

type roundPhase struct {
	state                 *domain.MatchState
	round                 *domain.RoundRecord
	currentRound          domain.RoundNumber
	beginningCash         domain.Money
	stats                 *resolutionStats
	backlogExpiryRounds   int
	procurementTerms      ProcurementTermsHook
	productionBOM         ProductionBOMHook
	productionRoute       ProductionRouteHook
	productionCost        ProductionCostHook
	inventoryCost         InventoryCarryingCostHook
	receivableDelayRounds int
	payableDelayRounds    int
	random                ports.RandomSource
	eventSeq              int
	stressedStations      map[domain.WorkstationID]bool
}

type resolutionStats struct {
	revenue           domain.Money
	procurementSpend  domain.Money
	productionSpend   domain.Money
	payrollExpense    domain.Money
	holdingCost       domain.Money
	debtServiceCost   domain.Money
	shippedUnits      domain.Units
	producedUnits     domain.Units
	lostSalesUnits    domain.Units
	cashReceipts      domain.Money
	cashDisbursements domain.Money
}

func (p *roundPhase) resolveProcurement(action *domain.ActionSubmission) {
	if action == nil || action.Action.Procurement == nil {
		return
	}

	hardCap := cappedBudget(p.state.ActiveTargets.ProcurementBudget)
	spendUsed := domain.Money(0)

	for index, order := range action.Action.Procurement.Orders {
		if order.Quantity <= 0 {
			continue
		}

		allowed := order.Quantity
		terms := p.procurementTerms(ProcurementTermsContext{
			State:        p.state.Clone(),
			CurrentRound: p.currentRound,
			Order:        order,
		})
		if !terms.KnownSupplier {
			p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Rejected procurement order for unsupported supplier/part combination", map[string]any{
				"part_id":     string(order.PartID),
				"supplier_id": string(order.SupplierID),
				"quantity":    int(order.Quantity),
			})
			continue
		}
		if terms.UnitCost <= 0 {
			terms.UnitCost = 1
		}
		if terms.LeadTimeRounds <= 0 {
			terms.LeadTimeRounds = 1
		}
		if terms.OnTimeDeliveryPct < 0 {
			terms.OnTimeDeliveryPct = 100
		}
		if order.Quantity < terms.MinimumOrderQuantity {
			allowed = terms.MinimumOrderQuantity
		}

		if hardCap > 0 {
			remaining := hardCap - spendUsed
			if remaining <= 0 {
				allowed = 0
			} else if spendForQuantity(allowed, terms.UnitCost) > remaining {
				allowed = affordableQuantity(remaining, terms.UnitCost)
			}
		}

		affordable := availableProjectedSpendCapacity(p.state.Plant)
		if spendForQuantity(allowed, terms.UnitCost) > affordable {
			allowed = affordableQuantity(affordable, terms.UnitCost)
		}
		if allowed > 0 && allowed < terms.MinimumOrderQuantity {
			allowed = 0
		}

		if allowed != order.Quantity {
			p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Trimmed procurement order to stay within budget or debt constraints", map[string]any{
				"part_id":                string(order.PartID),
				"requested_quantity":     int(order.Quantity),
				"accepted_quantity":      int(allowed),
				"minimum_order_quantity": int(terms.MinimumOrderQuantity),
				"effective_unit_cost":    int(terms.UnitCost),
			})
		}

		if allowed <= 0 {
			p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Rejected procurement order because no legal quantity remained", map[string]any{
				"part_id":                string(order.PartID),
				"requested_quantity":     int(order.Quantity),
				"accepted_quantity":      0,
				"minimum_order_quantity": int(terms.MinimumOrderQuantity),
				"effective_unit_cost":    int(terms.UnitCost),
			})
			continue
		}

		arrivalRound := p.currentRound + domain.RoundNumber(terms.LeadTimeRounds)
		lateDelivery := false
		if terms.OnTimeDeliveryPct < 100 && terms.LateDeliveryRounds > 0 && p.random != nil {
			if p.random.IntN(100) >= terms.OnTimeDeliveryPct {
				arrivalRound += domain.RoundNumber(terms.LateDeliveryRounds)
				lateDelivery = true
			}
		}

		lot := domain.SupplyLot{
			PurchaseOrderID: fmt.Sprintf("%s-po-%02d", action.ActionID, index+1),
			SupplierID:      order.SupplierID,
			PartID:          order.PartID,
			Quantity:        allowed,
			UnitCost:        terms.UnitCost,
			OrderedRound:    p.currentRound,
			ArrivalRound:    arrivalRound,
		}
		p.state.Plant.InTransitSupply = append(p.state.Plant.InTransitSupply, lot)
		lineSpend := spendForQuantity(allowed, lot.UnitCost)
		spendUsed += lineSpend
		p.stats.procurementSpend += lineSpend

		p.appendEvent(domain.EventPurchaseOrderPlaced, domain.ActorPlantSystem, fmt.Sprintf("Placed purchase order for %d %s", allowed, order.PartID), map[string]any{
			"purchase_order_id": lot.PurchaseOrderID,
			"part_id":           string(order.PartID),
			"supplier_id":       string(order.SupplierID),
			"quantity":          int(allowed),
			"lead_time_rounds":  terms.LeadTimeRounds,
			"on_time_pct":       terms.OnTimeDeliveryPct,
			"late_delivery":     lateDelivery,
			"arrival_round":     int(lot.ArrivalRound),
			"unit_cost":         int(lot.UnitCost),
		})
	}

	if spendUsed > 0 {
		p.schedulePayable(spendUsed, fmt.Sprintf("procurement-r%d", p.currentRound))
	}
}

func (p *roundPhase) receiveSupply() {
	if len(p.state.Plant.InTransitSupply) == 0 {
		return
	}

	arrivals := make([]domain.SupplyLot, 0, len(p.state.Plant.InTransitSupply))
	remaining := make([]domain.SupplyLot, 0, len(p.state.Plant.InTransitSupply))
	for _, lot := range p.state.Plant.InTransitSupply {
		if lot.ArrivalRound == p.currentRound {
			arrivals = append(arrivals, lot)
			continue
		}
		remaining = append(remaining, lot)
	}

	p.state.Plant.InTransitSupply = remaining
	for _, lot := range arrivals {
		addPartInventory(&p.state.Plant.PartsInventory, lot.PartID, lot.Quantity, lot.UnitCost)
		p.appendEvent(domain.EventSupplyArrived, domain.ActorPlantSystem, fmt.Sprintf("Received %d %s", lot.Quantity, lot.PartID), map[string]any{
			"purchase_order_id": lot.PurchaseOrderID,
			"part_id":           string(lot.PartID),
			"quantity":          int(lot.Quantity),
			"unit_cost":         int(lot.UnitCost),
		})
	}
}

func (p *roundPhase) resolveProduction(action *domain.ActionSubmission) {
	if action == nil || action.Action.Production == nil {
		return
	}
	if len(p.state.Plant.Workstations) == 0 {
		return
	}

	firstStation := p.state.Plant.Workstations[0].WorkstationID
	for _, release := range action.Action.Production.Releases {
		if release.Quantity <= 0 {
			continue
		}

		bom := p.productionBOM(ProductionBOMContext{
			State:        p.state.Clone(),
			CurrentRound: p.currentRound,
			ProductID:    release.ProductID,
		})
		if !bom.KnownProduct {
			p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Rejected production release for unknown product", map[string]any{
				"product_id":          string(release.ProductID),
				"requested_quantity":  int(release.Quantity),
				"accepted_quantity":   0,
				"blocked_requirement": "unknown_product",
			})
			continue
		}

		allowed := release.Quantity
		if len(bom.Parts) > 0 {
			allowed = minUnits(allowed, maxBuildableQuantity(p.state.Plant.PartsInventory, bom.Parts))
			if allowed != release.Quantity {
				p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Trimmed production release to available part inventory", map[string]any{
					"product_id":         string(release.ProductID),
					"requested_quantity": int(release.Quantity),
					"accepted_quantity":  int(allowed),
				})
			}
		}
		if allowed <= 0 {
			p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Rejected production release because required parts were unavailable", map[string]any{
				"product_id":         string(release.ProductID),
				"requested_quantity": int(release.Quantity),
				"accepted_quantity":  0,
			})
			continue
		}

		materialCost := consumePartInventory(&p.state.Plant.PartsInventory, bom.Parts, allowed)
		addWIPInventory(&p.state.Plant.WIPInventory, release.ProductID, firstStation, allowed, perUnitCost(materialCost, allowed))
		p.appendEvent(domain.EventProductionReleased, domain.ActorPlantSystem, fmt.Sprintf("Released %d %s into production", allowed, release.ProductID), map[string]any{
			"product_id":     string(release.ProductID),
			"quantity":       int(allowed),
			"workstation_id": string(firstStation),
			"material_cost":  int(materialCost),
		})
	}

	hardCap := cappedBudget(p.state.ActiveTargets.ProductionSpendBudget)
	spendUsed := domain.Money(0)
	for _, allocation := range action.Action.Production.CapacityAllocation {
		if allocation.Capacity <= 0 {
			continue
		}

		wsIndex := workstationIndex(p.state.Plant.Workstations, allocation.WorkstationID)
		if wsIndex < 0 {
			p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Ignored capacity allocation for unknown workstation", map[string]any{
				"workstation_id": string(allocation.WorkstationID),
				"product_id":     string(allocation.ProductID),
			})
			continue
		}

		effectiveCapacity, capacityLoss := stressedCapacity(p.state.Plant.Workstations[wsIndex], wipUnitsAtWorkstation(p.state.Plant.WIPInventory, allocation.WorkstationID))
		p.state.Plant.Workstations[wsIndex].EffectiveCapacityPerRound = effectiveCapacity
		p.state.Plant.Workstations[wsIndex].StressCapacityLoss = capacityLoss
		if capacityLoss > 0 && !p.stressedStations[allocation.WorkstationID] {
			p.stressedStations[allocation.WorkstationID] = true
			p.appendEvent(domain.EventWorkstationStressed, domain.ActorPlantSystem, fmt.Sprintf("%s lost effective capacity to congestion", allocation.WorkstationID), map[string]any{
				"workstation_id":     string(allocation.WorkstationID),
				"effective_capacity": int(effectiveCapacity),
				"capacity_loss":      int(capacityLoss),
			})
		}
		remainingCapacity := effectiveCapacity - p.state.Plant.Workstations[wsIndex].CapacityUsed
		availableWIP := wipQuantity(p.state.Plant.WIPInventory, allocation.ProductID, allocation.WorkstationID)
		advance := minCapacity(allocation.Capacity, remainingCapacity, domain.CapacityUnits(availableWIP))
		costing := p.productionCost(ProductionCostContext{
			State:         p.state.Clone(),
			CurrentRound:  p.currentRound,
			ProductID:     allocation.ProductID,
			WorkstationID: allocation.WorkstationID,
			Quantity:      advance,
		})
		if costing.CostPerCapacityUnit <= 0 {
			costing.CostPerCapacityUnit = 1
		}
		if hardCap > 0 {
			remainingBudget := hardCap - spendUsed
			if remainingBudget <= 0 {
				advance = 0
			} else {
				advance = minCapacity(advance, domain.CapacityUnits(affordableQuantity(remainingBudget, costing.CostPerCapacityUnit)))
			}
		}
		affordable := availableSpendCapacity(p.state.Plant)
		advance = minCapacity(advance, domain.CapacityUnits(affordableQuantity(affordable, costing.CostPerCapacityUnit)))
		if advance < allocation.Capacity {
			p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Trimmed production advance to available work or capacity", map[string]any{
				"workstation_id":     string(allocation.WorkstationID),
				"product_id":         string(allocation.ProductID),
				"requested_capacity": int(allocation.Capacity),
				"accepted_capacity":  int(advance),
				"effective_capacity": int(effectiveCapacity),
			})
		}
		if advance <= 0 {
			continue
		}

		advancedCost := takeWIPInventory(&p.state.Plant.WIPInventory, allocation.ProductID, allocation.WorkstationID, domain.Units(advance))
		p.state.Plant.Workstations[wsIndex].CapacityUsed += advance
		lineSpend := spendForQuantity(domain.Units(advance), costing.CostPerCapacityUnit)
		spendUsed += lineSpend
		p.stats.productionSpend += lineSpend

		if wsIndex == len(p.state.Plant.Workstations)-1 {
			route := p.productionRoute(ProductionRouteContext{
				State:            p.state.Clone(),
				CurrentRound:     p.currentRound,
				ProductID:        allocation.ProductID,
				CurrentStationID: allocation.WorkstationID,
			})
			if route.Finished {
				addFinishedInventory(&p.state.Plant.FinishedInventory, allocation.ProductID, domain.Units(advance), perUnitCost(advancedCost, domain.Units(advance)))
				p.stats.producedUnits += domain.Units(advance)
				p.appendEvent(domain.EventFinishedGoodsProduced, domain.ActorPlantSystem, fmt.Sprintf("Finished %d %s", advance, allocation.ProductID), map[string]any{
					"product_id":     string(allocation.ProductID),
					"quantity":       int(advance),
					"workstation_id": string(allocation.WorkstationID),
					"inventory_cost": int(advancedCost),
				})
				continue
			}
		}

		route := p.productionRoute(ProductionRouteContext{
			State:            p.state.Clone(),
			CurrentRound:     p.currentRound,
			ProductID:        allocation.ProductID,
			CurrentStationID: allocation.WorkstationID,
		})
		if route.Finished {
			addFinishedInventory(&p.state.Plant.FinishedInventory, allocation.ProductID, domain.Units(advance), perUnitCost(advancedCost, domain.Units(advance)))
			p.stats.producedUnits += domain.Units(advance)
			p.appendEvent(domain.EventFinishedGoodsProduced, domain.ActorPlantSystem, fmt.Sprintf("Finished %d %s", advance, allocation.ProductID), map[string]any{
				"product_id":     string(allocation.ProductID),
				"quantity":       int(advance),
				"workstation_id": string(allocation.WorkstationID),
				"inventory_cost": int(advancedCost),
			})
			continue
		}

		nextStation := route.NextWorkstationID
		if nextStation == "" {
			p.appendEvent(domain.EventRuleAdjustment, domain.ActorPlantSystem, "Ignored production advance with no scenario route continuation", map[string]any{
				"product_id":     string(allocation.ProductID),
				"workstation_id": string(allocation.WorkstationID),
				"quantity":       int(advance),
			})
			addWIPInventory(&p.state.Plant.WIPInventory, allocation.ProductID, allocation.WorkstationID, domain.Units(advance), perUnitCost(advancedCost, domain.Units(advance)))
			continue
		}
		addWIPInventory(&p.state.Plant.WIPInventory, allocation.ProductID, nextStation, domain.Units(advance), perUnitCost(advancedCost, domain.Units(advance)))
		p.appendEvent(domain.EventWorkAdvanced, domain.ActorPlantSystem, fmt.Sprintf("Advanced %d %s to %s", advance, allocation.ProductID, nextStation), map[string]any{
			"product_id":          string(allocation.ProductID),
			"quantity":            int(advance),
			"from_workstation_id": string(allocation.WorkstationID),
			"to_workstation_id":   string(nextStation),
		})
	}

	if spendUsed > 0 {
		p.applyCashDelta(-spendUsed)
		p.stats.cashDisbursements += spendUsed
		p.appendEvent(domain.EventCashChanged, domain.ActorPlantSystem, "Production spending applied", map[string]any{
			"delta": -int(spendUsed),
			"cash":  int(p.state.Plant.Cash),
			"debt":  int(p.state.Plant.Debt),
		})
	}
}

func (p *roundPhase) resolveSales(action *domain.ActionSubmission) {
	priceByProduct := map[domain.ProductID]domain.Money{}
	if action != nil && action.Action.Sales != nil {
		for _, offer := range action.Action.Sales.ProductOffers {
			if offer.UnitPrice <= 0 {
				continue
			}
			priceByProduct[offer.ProductID] = offer.UnitPrice
		}
	}

	backlog := effectiveBacklog(*p.state)
	if len(backlog) == 0 {
		p.syncCustomerBacklog(nil)
		p.state.Plant.Backlog = nil
		return
	}

	slices.SortFunc(backlog, func(left, right domain.BacklogEntry) int {
		return cmp.Or(
			cmp.Compare(int(left.OriginRound), int(right.OriginRound)),
			cmp.Compare(string(left.CustomerID), string(right.CustomerID)),
			cmp.Compare(string(left.ProductID), string(right.ProductID)),
		)
	})

	remaining := make([]domain.BacklogEntry, 0, len(backlog))
	for _, entry := range backlog {
		onHand := finishedQuantity(p.state.Plant.FinishedInventory, entry.ProductID)
		shipped := minUnits(entry.Quantity, onHand)
		if shipped > 0 {
			takeFinishedInventory(&p.state.Plant.FinishedInventory, entry.ProductID, shipped)
			entry.Quantity -= shipped

			unitPrice := priceByProduct[entry.ProductID]
			if unitPrice <= 0 {
				unitPrice = 1
			}
			revenue := domain.Money(shipped) * unitPrice
			p.stats.revenue += revenue
			p.stats.shippedUnits += shipped
			p.scheduleReceivable(revenue, fmt.Sprintf("%s-%s-r%d", entry.CustomerID, entry.ProductID, p.currentRound), p.customerPaymentDelay(entry.CustomerID))

			p.appendEvent(domain.EventShipmentCompleted, domain.ActorPlantSystem, fmt.Sprintf("Shipped %d %s to %s", shipped, entry.ProductID, entry.CustomerID), map[string]any{
				"customer_id": string(entry.CustomerID),
				"product_id":  string(entry.ProductID),
				"quantity":    int(shipped),
				"unit_price":  int(unitPrice),
			})
		}

		if entry.Quantity > 0 {
			remaining = append(remaining, entry)
		}
	}

	p.state.Plant.Backlog = remaining
	p.syncCustomerBacklog(remaining)
}

func (p *roundPhase) finalizeRound(action *domain.ActionSubmission) {
	backlog := effectiveBacklog(*p.state)
	aged := make([]domain.BacklogEntry, 0, len(backlog))
	for _, entry := range backlog {
		entry.AgeInRounds++
		if entry.AgeInRounds > p.backlogExpiryRounds {
			p.stats.lostSalesUnits += entry.Quantity
			p.appendEvent(domain.EventBacklogExpired, domain.ActorPlantSystem, fmt.Sprintf("Expired backlog for %s/%s", entry.CustomerID, entry.ProductID), map[string]any{
				"customer_id": string(entry.CustomerID),
				"product_id":  string(entry.ProductID),
				"quantity":    int(entry.Quantity),
			})
			if customer := findCustomer(p.state.Customers, entry.CustomerID); customer != nil {
				customer.Sentiment--
				p.appendEvent(domain.EventCustomerSentimentMoved, domain.ActorPlantSystem, fmt.Sprintf("Customer %s sentiment decreased", entry.CustomerID), map[string]any{
					"customer_id": string(entry.CustomerID),
					"sentiment":   customer.Sentiment,
				})
			}
			continue
		}
		aged = append(aged, entry)
	}

	p.state.Plant.Backlog = aged
	p.syncCustomerBacklog(aged)

	if action != nil && action.Action.Finance != nil {
		targets := action.Action.Finance.NextRoundTargets
		targets.EffectiveRound = p.currentRound + 1
		p.state.ActiveTargets = targets
		p.state.Plant.DebtCeiling = targets.DebtCeilingTarget
	}

	p.collectReceivablesDue()
	p.payPayablesDue()

	holdingCost, debtCost := p.applyRoundOperatingCosts()
	p.stats.holdingCost = holdingCost
	p.stats.debtServiceCost = debtCost

	metrics := p.computeMetrics()
	p.round.Metrics = metrics
	p.appendEvent(domain.EventMetricSnapshot, domain.ActorPlantSystem, "Recorded round metrics", map[string]any{
		"operating_expense":       int(metrics.OperatingExpense),
		"round_profit":            int(metrics.RoundProfit),
		"throughput_revenue":      int(metrics.ThroughputRevenue),
		"backlog_units":           int(metrics.BacklogUnits),
		"production_output_units": int(metrics.ProductionOutputUnits),
	})
}

func (p *roundPhase) computeMetrics() domain.PlantMetrics {
	backlogUnits := domain.Units(0)
	for _, entry := range p.state.Plant.Backlog {
		backlogUnits += entry.Quantity
	}

	partsUnits := domain.Units(0)
	inventoryValue := domain.Money(0)
	for _, part := range p.state.Plant.PartsInventory {
		partsUnits += part.OnHandQty
		inventoryValue += inventoryCost(part.OnHandQty, part.UnitCost)
	}

	wipUnits := domain.Units(0)
	for _, item := range p.state.Plant.WIPInventory {
		wipUnits += item.Quantity
		inventoryValue += inventoryCost(item.Quantity, item.UnitCost)
	}

	finishedUnits := domain.Units(0)
	for _, item := range p.state.Plant.FinishedInventory {
		finishedUnits += item.OnHandQty
		inventoryValue += inventoryCost(item.OnHandQty, item.UnitCost)
	}

	capacityLossUnits := domain.Units(0)
	for _, workstation := range p.state.Plant.Workstations {
		capacityLossUnits += domain.Units(workstation.StressCapacityLoss)
	}

	operatingExpense := p.stats.procurementSpend + p.stats.productionSpend + p.stats.payrollExpense + p.stats.holdingCost + p.stats.debtServiceCost
	netCashChange := p.state.Plant.Cash - p.beginningCash
	demandUnits := p.stats.shippedUnits + backlogUnits + p.stats.lostSalesUnits
	onTime := domain.Percentage(100)
	if demandUnits > 0 {
		onTime = domain.Percentage(int(p.stats.shippedUnits) * 100 / int(demandUnits))
	}

	return domain.PlantMetrics{
		ThroughputRevenue:     p.stats.revenue,
		OperatingExpense:      operatingExpense,
		ProcurementSpend:      p.stats.procurementSpend,
		ProductionSpend:       p.stats.productionSpend,
		PayrollExpense:        p.stats.payrollExpense,
		HoldingCost:           p.stats.holdingCost,
		DebtServiceCost:       p.stats.debtServiceCost,
		InventoryValue:        inventoryValue,
		NetCashChange:         netCashChange,
		CashReceipts:          p.stats.cashReceipts,
		CashDisbursements:     p.stats.cashDisbursements,
		EndingCash:            p.state.Plant.Cash,
		RoundProfit:           p.stats.revenue - operatingExpense,
		OnTimeShipmentRate:    onTime,
		BacklogUnits:          backlogUnits,
		LostSalesUnits:        p.stats.lostSalesUnits,
		PartsOnHandUnits:      partsUnits,
		FinishedGoodsUnits:    finishedUnits,
		ProductionOutputUnits: p.stats.producedUnits,
		CapacityLossUnits:     capacityLossUnits,
	}
}

func (p *roundPhase) applyRoundOperatingCosts() (domain.Money, domain.Money) {
	partsValue := domain.Money(0)
	for _, part := range p.state.Plant.PartsInventory {
		partsValue += inventoryCost(part.OnHandQty, part.UnitCost)
	}
	wipValue := domain.Money(0)
	for _, item := range p.state.Plant.WIPInventory {
		wipValue += inventoryCost(item.Quantity, item.UnitCost)
	}
	finishedValue := domain.Money(0)
	for _, item := range p.state.Plant.FinishedInventory {
		finishedValue += inventoryCost(item.OnHandQty, item.UnitCost)
	}

	holdingCost := carryingCost(partsValue + wipValue + finishedValue)
	if p.inventoryCost != nil {
		holdingCost = p.inventoryCost(InventoryCarryingCostContext{
			State:          p.state.Clone(),
			CurrentRound:   p.currentRound,
			InventoryClass: InventoryClassParts,
			InventoryValue: partsValue,
		})
		holdingCost += p.inventoryCost(InventoryCarryingCostContext{
			State:          p.state.Clone(),
			CurrentRound:   p.currentRound,
			InventoryClass: InventoryClassWIP,
			InventoryValue: wipValue,
		})
		holdingCost += p.inventoryCost(InventoryCarryingCostContext{
			State:          p.state.Clone(),
			CurrentRound:   p.currentRound,
			InventoryClass: InventoryClassFinished,
			InventoryValue: finishedValue,
		})
	}
	debtCost := carryingCost(p.state.Plant.Debt)
	total := holdingCost + debtCost
	if total > 0 {
		p.applyCashDelta(-total)
		p.stats.cashDisbursements += total
		p.appendEvent(domain.EventCashChanged, domain.ActorPlantSystem, "Round-end carrying costs applied", map[string]any{
			"delta":             -int(total),
			"cash":              int(p.state.Plant.Cash),
			"debt":              int(p.state.Plant.Debt),
			"holding_cost":      int(holdingCost),
			"debt_service_cost": int(debtCost),
		})
	}

	return holdingCost, debtCost
}

func (p *roundPhase) collectReceivablesDue() {
	if len(p.state.Plant.Receivables) == 0 {
		return
	}

	due := make([]domain.CashCommitment, 0, len(p.state.Plant.Receivables))
	remaining := make([]domain.CashCommitment, 0, len(p.state.Plant.Receivables))
	for _, item := range p.state.Plant.Receivables {
		if item.DueRound <= p.currentRound {
			due = append(due, item)
			continue
		}
		remaining = append(remaining, item)
	}
	p.state.Plant.Receivables = remaining

	collected := sumCommitments(due)
	for _, item := range due {
		p.applyCashDelta(item.Amount)
		p.appendEvent(domain.EventReceivableCollected, domain.ActorPlantSystem, "Collected customer receivable", map[string]any{
			"commitment_id": item.CommitmentID,
			"amount":        int(item.Amount),
			"due_round":     int(item.DueRound),
			"cash":          int(p.state.Plant.Cash),
			"debt":          int(p.state.Plant.Debt),
		})
	}
	if collected > 0 {
		p.stats.cashReceipts += collected
		p.appendEvent(domain.EventCashChanged, domain.ActorPlantSystem, "Customer receipts collected", map[string]any{
			"delta": int(collected),
			"cash":  int(p.state.Plant.Cash),
			"debt":  int(p.state.Plant.Debt),
		})
	}
}

func (p *roundPhase) payPayablesDue() {
	if len(p.state.Plant.Payables) == 0 {
		return
	}

	due := make([]domain.CashCommitment, 0, len(p.state.Plant.Payables))
	remaining := make([]domain.CashCommitment, 0, len(p.state.Plant.Payables))
	for _, item := range p.state.Plant.Payables {
		if item.DueRound <= p.currentRound {
			due = append(due, item)
			continue
		}
		remaining = append(remaining, item)
	}
	p.state.Plant.Payables = remaining

	disbursed := sumCommitments(due)
	for _, item := range due {
		p.applyCashDelta(-item.Amount)
		p.appendEvent(domain.EventPayablePaid, domain.ActorPlantSystem, "Paid supplier payable", map[string]any{
			"commitment_id": item.CommitmentID,
			"amount":        int(item.Amount),
			"due_round":     int(item.DueRound),
			"cash":          int(p.state.Plant.Cash),
			"debt":          int(p.state.Plant.Debt),
		})
	}
	if disbursed > 0 {
		p.stats.cashDisbursements += disbursed
		p.appendEvent(domain.EventCashChanged, domain.ActorPlantSystem, "Supplier payments applied", map[string]any{
			"delta": -int(disbursed),
			"cash":  int(p.state.Plant.Cash),
			"debt":  int(p.state.Plant.Debt),
		})
	}
}

func (p *roundPhase) scheduleReceivable(amount domain.Money, referenceID string, delayRounds int) {
	if amount <= 0 {
		return
	}
	delayRounds = max(0, delayRounds)
	if delayRounds == 0 {
		p.applyCashDelta(amount)
		p.stats.cashReceipts += amount
		p.appendEvent(domain.EventCashChanged, domain.ActorPlantSystem, "Shipment revenue applied", map[string]any{
			"delta": int(amount),
			"cash":  int(p.state.Plant.Cash),
			"debt":  int(p.state.Plant.Debt),
		})
		return
	}

	item := domain.CashCommitment{
		CommitmentID: fmt.Sprintf("ar-r%d-%d", p.currentRound, len(p.state.Plant.Receivables)+1),
		Kind:         domain.CashCommitmentReceivable,
		Amount:       amount,
		DueRound:     p.currentRound + domain.RoundNumber(delayRounds),
		CreatedRound: p.currentRound,
		ReferenceID:  referenceID,
	}
	p.state.Plant.Receivables = append(p.state.Plant.Receivables, item)
	p.appendEvent(domain.EventReceivableScheduled, domain.ActorPlantSystem, "Scheduled customer receivable", map[string]any{
		"commitment_id": item.CommitmentID,
		"amount":        int(item.Amount),
		"due_round":     int(item.DueRound),
		"reference_id":  item.ReferenceID,
	})
}

func (p *roundPhase) customerPaymentDelay(customerID domain.CustomerID) int {
	customer := findCustomer(p.state.Customers, customerID)
	if customer == nil || customer.PaymentDelayRounds <= 0 {
		return p.receivableDelayRounds
	}
	return customer.PaymentDelayRounds
}

func (p *roundPhase) schedulePayable(amount domain.Money, referenceID string) {
	if amount <= 0 {
		return
	}
	if p.payableDelayRounds == 0 {
		p.applyCashDelta(-amount)
		p.stats.cashDisbursements += amount
		p.appendEvent(domain.EventCashChanged, domain.ActorPlantSystem, "Procurement spending applied", map[string]any{
			"delta": -int(amount),
			"cash":  int(p.state.Plant.Cash),
			"debt":  int(p.state.Plant.Debt),
		})
		return
	}

	item := domain.CashCommitment{
		CommitmentID: fmt.Sprintf("ap-r%d-%d", p.currentRound, len(p.state.Plant.Payables)+1),
		Kind:         domain.CashCommitmentPayable,
		Amount:       amount,
		DueRound:     p.currentRound + domain.RoundNumber(p.payableDelayRounds),
		CreatedRound: p.currentRound,
		ReferenceID:  referenceID,
	}
	p.state.Plant.Payables = append(p.state.Plant.Payables, item)
	p.appendEvent(domain.EventPayableScheduled, domain.ActorPlantSystem, "Scheduled supplier payable", map[string]any{
		"commitment_id": item.CommitmentID,
		"amount":        int(item.Amount),
		"due_round":     int(item.DueRound),
		"reference_id":  item.ReferenceID,
	})
}

func (p *roundPhase) syncCustomerBacklog(backlog []domain.BacklogEntry) {
	byCustomer := map[domain.CustomerID][]domain.BacklogEntry{}
	for _, entry := range backlog {
		byCustomer[entry.CustomerID] = append(byCustomer[entry.CustomerID], entry)
	}

	for index := range p.state.Customers {
		p.state.Customers[index].Backlog = slices.Clone(byCustomer[p.state.Customers[index].CustomerID])
	}
}

func (p *roundPhase) applyCashDelta(delta domain.Money) {
	if delta == 0 {
		return
	}

	if delta > 0 {
		paydown := minMoney(delta, p.state.Plant.Debt)
		p.state.Plant.Debt -= paydown
		p.state.Plant.Cash += delta - paydown
		return
	}

	spend := -delta
	if p.state.Plant.Cash >= spend {
		p.state.Plant.Cash -= spend
		return
	}

	deficit := spend - p.state.Plant.Cash
	p.state.Plant.Cash = 0
	p.state.Plant.Debt += deficit
}

func (p *roundPhase) appendEvent(eventType domain.RoundEventType, actorID domain.ActorID, summary string, payload map[string]any) {
	p.eventSeq++
	p.round.Events = append(p.round.Events, domain.RoundEvent{
		EventID: domain.EventID(fmt.Sprintf("round-%d-event-%03d", p.currentRound, p.eventSeq)),
		MatchID: p.state.MatchID,
		Round:   p.currentRound,
		Type:    eventType,
		ActorID: actorID,
		Summary: summary,
		Payload: clonePayload(payload),
	})
}

func normalizeActions(state domain.MatchState, actions []domain.ActionSubmission) ([]domain.ActionSubmission, error) {
	byRole := map[domain.RoleID]domain.ActionSubmission{}
	assignedRoles := assignedRoles(state)
	for _, action := range actions {
		if action.MatchID != state.MatchID {
			return nil, fmt.Errorf("engine: action %q match id %q does not match state %q", action.ActionID, action.MatchID, state.MatchID)
		}
		if action.Round != state.CurrentRound {
			return nil, fmt.Errorf("engine: action %q round %d does not match state round %d", action.ActionID, action.Round, state.CurrentRound)
		}
		if !slices.Contains(domain.CanonicalRoles(), action.RoleID) {
			return nil, fmt.Errorf("engine: action %q uses unsupported role %q", action.ActionID, action.RoleID)
		}
		if len(assignedRoles) > 0 && !assignedRoles[action.RoleID] {
			return nil, fmt.Errorf("engine: action %q role %q is not assigned in the current match", action.ActionID, action.RoleID)
		}
		if err := validateActionSubmission(state, action); err != nil {
			return nil, err
		}
		if _, exists := byRole[action.RoleID]; exists {
			return nil, fmt.Errorf("engine: duplicate action for role %q", action.RoleID)
		}
		byRole[action.RoleID] = action.Clone()
	}

	ordered := make([]domain.ActionSubmission, 0, len(actions))
	for _, roleID := range domain.CanonicalRoles() {
		action, ok := byRole[roleID]
		if !ok {
			continue
		}
		ordered = append(ordered, action)
		delete(byRole, roleID)
	}

	if len(byRole) > 0 {
		return nil, fmt.Errorf("engine: unsupported roles in action set: %s", strings.Join(sortedRoleNames(byRole), ", "))
	}

	return ordered, nil
}

func assignedRoles(state domain.MatchState) map[domain.RoleID]bool {
	if len(state.Roles) == 0 {
		return nil
	}

	assigned := make(map[domain.RoleID]bool, len(state.Roles))
	for _, role := range state.Roles {
		assigned[role.RoleID] = true
	}
	return assigned
}

func validateActionSubmission(state domain.MatchState, action domain.ActionSubmission) error {
	payloads := populatedPayloadNames(action.Action)
	if len(payloads) == 0 {
		return fmt.Errorf("engine: action %q role %q must include payload %q", action.ActionID, action.RoleID, action.RoleID)
	}
	if len(payloads) > 1 {
		return fmt.Errorf("engine: action %q role %q includes multiple payloads: %s", action.ActionID, action.RoleID, strings.Join(payloads, ", "))
	}
	if len(payloads) == 1 && payloads[0] != string(action.RoleID) {
		return fmt.Errorf("engine: action %q role %q includes mismatched payload %q", action.ActionID, action.RoleID, payloads[0])
	}

	switch action.RoleID {
	case domain.RoleProcurementManager:
		return validateProcurementAction(action)
	case domain.RoleProductionManager:
		return validateProductionAction(state, action)
	case domain.RoleSalesManager:
		return validateSalesAction(action)
	case domain.RoleFinanceController:
		return validateFinanceAction(action)
	default:
		return fmt.Errorf("engine: action %q uses unsupported role %q", action.ActionID, action.RoleID)
	}
}

func populatedPayloadNames(action domain.RoleAction) []string {
	var names []string
	if action.Procurement != nil {
		names = append(names, string(domain.RoleProcurementManager))
	}
	if action.Production != nil {
		names = append(names, string(domain.RoleProductionManager))
	}
	if action.Sales != nil {
		names = append(names, string(domain.RoleSalesManager))
	}
	if action.Finance != nil {
		names = append(names, string(domain.RoleFinanceController))
	}
	return names
}

func validateProcurementAction(action domain.ActionSubmission) error {
	if action.Action.Procurement == nil {
		return fmt.Errorf("engine: action %q role %q must include payload %q", action.ActionID, action.RoleID, action.RoleID)
	}

	for index, order := range action.Action.Procurement.Orders {
		if order.Quantity < 0 {
			return fmt.Errorf("engine: action %q procurement order %d quantity must be non-negative", action.ActionID, index)
		}
	}
	return nil
}

func validateProductionAction(state domain.MatchState, action domain.ActionSubmission) error {
	if action.Action.Production == nil {
		return fmt.Errorf("engine: action %q role %q must include payload %q", action.ActionID, action.RoleID, action.RoleID)
	}

	for index, release := range action.Action.Production.Releases {
		if release.Quantity < 0 {
			return fmt.Errorf("engine: action %q production release %d quantity must be non-negative", action.ActionID, index)
		}
	}
	for index, allocation := range action.Action.Production.CapacityAllocation {
		if allocation.Capacity < 0 {
			return fmt.Errorf("engine: action %q capacity allocation %d capacity must be non-negative", action.ActionID, index)
		}
		if workstationIndex(state.Plant.Workstations, allocation.WorkstationID) < 0 {
			return fmt.Errorf("engine: action %q capacity allocation %d references unknown workstation %q", action.ActionID, index, allocation.WorkstationID)
		}
	}
	return nil
}

func validateSalesAction(action domain.ActionSubmission) error {
	if action.Action.Sales == nil {
		return fmt.Errorf("engine: action %q role %q must include payload %q", action.ActionID, action.RoleID, action.RoleID)
	}

	for index, offer := range action.Action.Sales.ProductOffers {
		if offer.UnitPrice < 0 {
			return fmt.Errorf("engine: action %q sales offer %d unit price must be non-negative", action.ActionID, index)
		}
	}
	return nil
}

func validateFinanceAction(action domain.ActionSubmission) error {
	if action.Action.Finance == nil {
		return fmt.Errorf("engine: action %q role %q must include payload %q", action.ActionID, action.RoleID, action.RoleID)
	}

	targets := action.Action.Finance.NextRoundTargets
	if targets.ProcurementBudget < 0 {
		return fmt.Errorf("engine: action %q procurement budget must be non-negative", action.ActionID)
	}
	if targets.ProductionSpendBudget < 0 {
		return fmt.Errorf("engine: action %q production budget must be non-negative", action.ActionID)
	}
	if targets.RevenueTarget < 0 {
		return fmt.Errorf("engine: action %q revenue target must be non-negative", action.ActionID)
	}
	if targets.CashFloorTarget < 0 {
		return fmt.Errorf("engine: action %q cash floor target must be non-negative", action.ActionID)
	}
	if targets.DebtCeilingTarget < 0 {
		return fmt.Errorf("engine: action %q debt ceiling target must be non-negative", action.ActionID)
	}
	return nil
}

func collectCommentary(state domain.MatchState, actions []domain.ActionSubmission) []domain.CommentaryRecord {
	ordered := cloneActions(actions)
	slices.SortFunc(ordered, func(left, right domain.ActionSubmission) int {
		if compare := left.SubmittedAt.Compare(right.SubmittedAt); compare != 0 {
			return compare
		}
		if compare := cmp.Compare(actionTimelineRoleRank(left.RoleID), actionTimelineRoleRank(right.RoleID)); compare != 0 {
			return compare
		}
		return cmp.Compare(string(left.ActionID), string(right.ActionID))
	})

	commentary := make([]domain.CommentaryRecord, 0, len(ordered))
	for _, action := range ordered {
		if strings.TrimSpace(action.Commentary.Body) == "" {
			continue
		}

		record := action.Commentary.Clone()
		record.MatchID = state.MatchID
		record.Round = state.CurrentRound
		record.RoleID = action.RoleID
		if record.ActorID == "" {
			record.ActorID = domain.ActorID(action.RoleID)
		}
		if record.Visibility == "" {
			record.Visibility = domain.CommentaryPublic
		}
		if record.CommentaryID == "" {
			record.CommentaryID = domain.CommentaryID(fmt.Sprintf("%s-commentary", action.ActionID))
		}
		commentary = append(commentary, record)
	}

	return commentary
}

func cloneActions(actions []domain.ActionSubmission) []domain.ActionSubmission {
	if actions == nil {
		return nil
	}

	cloned := make([]domain.ActionSubmission, len(actions))
	for i := range actions {
		cloned[i] = actions[i].Clone()
	}
	return cloned
}

func actionTimelineRoleRank(roleID domain.RoleID) int {
	for index, canonical := range domain.CanonicalRoles() {
		if canonical == roleID {
			return index
		}
	}
	return len(domain.CanonicalRoles())
}

func actionForRole(actions []domain.ActionSubmission, roleID domain.RoleID) *domain.ActionSubmission {
	for index := range actions {
		if actions[index].RoleID == roleID {
			return &actions[index]
		}
	}
	return nil
}

func appendRecentRound(history domain.RoundHistory, round domain.RoundRecord, limit int) domain.RoundHistory {
	rounds := cloneRoundHistory(history.RecentRounds)
	rounds = append(rounds, round.Clone())
	if limit > 0 && len(rounds) > limit {
		rounds = rounds[len(rounds)-limit:]
	}
	return domain.RoundHistory{RecentRounds: rounds}
}

func cloneRoundHistory(rounds []domain.RoundRecord) []domain.RoundRecord {
	if rounds == nil {
		return nil
	}

	cloned := make([]domain.RoundRecord, len(rounds))
	for i := range rounds {
		cloned[i] = rounds[i].Clone()
	}
	return cloned
}

func effectiveBacklog(state domain.MatchState) []domain.BacklogEntry {
	if len(state.Plant.Backlog) > 0 {
		return slices.Clone(state.Plant.Backlog)
	}

	var backlog []domain.BacklogEntry
	for _, customer := range state.Customers {
		backlog = append(backlog, slices.Clone(customer.Backlog)...)
	}
	return backlog
}

func availableSpendCapacity(plant domain.PlantState) domain.Money {
	if plant.DebtCeiling <= 0 {
		return plant.Cash
	}
	return plant.Cash + max(plant.DebtCeiling-plant.Debt, 0)
}

func availableProjectedSpendCapacity(plant domain.PlantState) domain.Money {
	projectedCash, projectedDebt := projectedPlantPositionAfterCommitments(plant)
	if plant.DebtCeiling <= 0 {
		return max(projectedCash, 0)
	}
	return max(projectedCash, 0) + max(plant.DebtCeiling-projectedDebt, 0)
}

func projectedPlantPositionAfterCommitments(plant domain.PlantState) (domain.Money, domain.Money) {
	projectedCash := plant.Cash
	projectedDebt := plant.Debt

	for _, item := range plant.Payables {
		projectedCash, projectedDebt = projectCashDelta(projectedCash, projectedDebt, -item.Amount)
	}
	for _, item := range plant.Receivables {
		projectedCash, projectedDebt = projectCashDelta(projectedCash, projectedDebt, item.Amount)
	}

	return projectedCash, projectedDebt
}

func projectCashDelta(cash, debt, delta domain.Money) (domain.Money, domain.Money) {
	if delta == 0 {
		return cash, debt
	}
	if delta > 0 {
		paydown := minMoney(delta, debt)
		debt -= paydown
		cash += delta - paydown
		return cash, debt
	}

	spend := -delta
	if cash >= spend {
		return cash - spend, debt
	}

	deficit := spend - cash
	return 0, debt + deficit
}

func cappedBudget(target domain.Money) domain.Money {
	if target <= 0 {
		return 0
	}
	return target + target/10
}

func addPartInventory(items *[]domain.PartInventory, partID domain.PartID, quantity domain.Units, unitCost domain.Money) {
	if quantity <= 0 {
		return
	}

	for index := range *items {
		if (*items)[index].PartID == partID {
			if (*items)[index].UnitCost <= 0 {
				(*items)[index].OnHandQty += quantity
				(*items)[index].UnitCost = unitCost
				return
			}
			if unitCost <= 0 {
				(*items)[index].OnHandQty += quantity
				return
			}
			totalValue := inventoryCost((*items)[index].OnHandQty, (*items)[index].UnitCost) + inventoryCost(quantity, unitCost)
			(*items)[index].OnHandQty += quantity
			(*items)[index].UnitCost = perUnitCost(totalValue, (*items)[index].OnHandQty)
			return
		}
	}
	*items = append(*items, domain.PartInventory{PartID: partID, OnHandQty: quantity, UnitCost: unitCost})
}

func partQuantity(items []domain.PartInventory, partID domain.PartID) domain.Units {
	for _, item := range items {
		if item.PartID == partID {
			return item.OnHandQty
		}
	}
	return 0
}

func consumePartInventory(items *[]domain.PartInventory, bom []domain.BOMLine, quantity domain.Units) domain.Money {
	totalCost := domain.Money(0)
	for _, line := range bom {
		totalCost += takePartInventory(items, line.PartID, line.Quantity*quantity)
	}
	return totalCost
}

func takePartInventory(items *[]domain.PartInventory, partID domain.PartID, quantity domain.Units) domain.Money {
	totalCost := domain.Money(0)
	filtered := (*items)[:0]
	for _, item := range *items {
		if item.PartID == partID {
			totalCost = inventoryCost(quantity, item.UnitCost)
			item.OnHandQty -= quantity
		}
		if item.OnHandQty > 0 {
			filtered = append(filtered, item)
		}
	}
	*items = filtered
	return totalCost
}

func maxBuildableQuantity(items []domain.PartInventory, bom []domain.BOMLine) domain.Units {
	if len(bom) == 0 {
		return 0
	}

	allowed := domain.Units(-1)
	for _, line := range bom {
		if line.Quantity <= 0 {
			continue
		}
		buildable := partQuantity(items, line.PartID) / line.Quantity
		if allowed < 0 || buildable < allowed {
			allowed = buildable
		}
	}
	if allowed < 0 {
		return 0
	}
	return allowed
}

func addWIPInventory(items *[]domain.WIPInventory, productID domain.ProductID, workstationID domain.WorkstationID, quantity domain.Units, unitCost domain.Money) {
	if quantity <= 0 {
		return
	}

	for index := range *items {
		if (*items)[index].ProductID == productID && (*items)[index].WorkstationID == workstationID {
			if (*items)[index].UnitCost <= 0 {
				(*items)[index].Quantity += quantity
				(*items)[index].UnitCost = unitCost
				return
			}
			if unitCost <= 0 {
				(*items)[index].Quantity += quantity
				return
			}
			totalValue := inventoryCost((*items)[index].Quantity, (*items)[index].UnitCost) + inventoryCost(quantity, unitCost)
			(*items)[index].Quantity += quantity
			(*items)[index].UnitCost = perUnitCost(totalValue, (*items)[index].Quantity)
			return
		}
	}
	*items = append(*items, domain.WIPInventory{ProductID: productID, WorkstationID: workstationID, Quantity: quantity, UnitCost: unitCost})
}

func wipQuantity(items []domain.WIPInventory, productID domain.ProductID, workstationID domain.WorkstationID) domain.Units {
	for _, item := range items {
		if item.ProductID == productID && item.WorkstationID == workstationID {
			return item.Quantity
		}
	}
	return 0
}

func takeWIPInventory(items *[]domain.WIPInventory, productID domain.ProductID, workstationID domain.WorkstationID, quantity domain.Units) domain.Money {
	totalCost := domain.Money(0)
	filtered := (*items)[:0]
	for _, item := range *items {
		if item.ProductID == productID && item.WorkstationID == workstationID {
			totalCost = inventoryCost(quantity, item.UnitCost)
			item.Quantity -= quantity
		}
		if item.Quantity > 0 {
			filtered = append(filtered, item)
		}
	}
	*items = filtered
	return totalCost
}

func addFinishedInventory(items *[]domain.FinishedInventory, productID domain.ProductID, quantity domain.Units, unitCost domain.Money) {
	if quantity <= 0 {
		return
	}

	for index := range *items {
		if (*items)[index].ProductID == productID {
			if (*items)[index].UnitCost <= 0 {
				(*items)[index].OnHandQty += quantity
				(*items)[index].UnitCost = unitCost
				return
			}
			if unitCost <= 0 {
				(*items)[index].OnHandQty += quantity
				return
			}
			totalValue := inventoryCost((*items)[index].OnHandQty, (*items)[index].UnitCost) + inventoryCost(quantity, unitCost)
			(*items)[index].OnHandQty += quantity
			(*items)[index].UnitCost = perUnitCost(totalValue, (*items)[index].OnHandQty)
			return
		}
	}
	*items = append(*items, domain.FinishedInventory{ProductID: productID, OnHandQty: quantity, UnitCost: unitCost})
}

func finishedQuantity(items []domain.FinishedInventory, productID domain.ProductID) domain.Units {
	for _, item := range items {
		if item.ProductID == productID {
			return item.OnHandQty
		}
	}
	return 0
}

func takeFinishedInventory(items *[]domain.FinishedInventory, productID domain.ProductID, quantity domain.Units) {
	filtered := (*items)[:0]
	for _, item := range *items {
		if item.ProductID == productID {
			item.OnHandQty -= quantity
		}
		if item.OnHandQty > 0 {
			filtered = append(filtered, item)
		}
	}
	*items = filtered
}

func workstationIndex(items []domain.WorkstationState, workstationID domain.WorkstationID) int {
	for index, item := range items {
		if item.WorkstationID == workstationID {
			return index
		}
	}
	return -1
}

func resetCapacityUsage(items []domain.WorkstationState) []domain.WorkstationState {
	cloned := slices.Clone(items)
	for index := range cloned {
		cloned[index].CapacityUsed = 0
		cloned[index].EffectiveCapacityPerRound = cloned[index].CapacityPerRound
		cloned[index].StressCapacityLoss = 0
	}
	return cloned
}

func stressedCapacity(workstation domain.WorkstationState, wipUnits domain.Units) (domain.CapacityUnits, domain.CapacityUnits) {
	nominal := workstation.CapacityPerRound
	if nominal <= 0 {
		return 0, 0
	}
	if workstation.StressPenaltyPerExcessUnit <= 0 {
		return nominal, 0
	}

	threshold := domain.Units(nominal + workstation.StressBufferUnits)
	if wipUnits <= threshold {
		return nominal, 0
	}

	// Start with a linear congestion penalty so players can see the effect of excess WIP
	// before later mechanics add harsher stoppage, quality, or maintenance interactions.
	excess := domain.CapacityUnits(wipUnits - threshold)
	loss := excess * workstation.StressPenaltyPerExcessUnit
	if loss > nominal {
		loss = nominal
	}
	return nominal - loss, loss
}

func wipUnitsAtWorkstation(items []domain.WIPInventory, workstationID domain.WorkstationID) domain.Units {
	total := domain.Units(0)
	for _, item := range items {
		if item.WorkstationID == workstationID {
			total += item.Quantity
		}
	}
	return total
}

func findCustomer(customers []domain.CustomerState, customerID domain.CustomerID) *domain.CustomerState {
	for index := range customers {
		if customers[index].CustomerID == customerID {
			return &customers[index]
		}
	}
	return nil
}

func sortedRoleNames(items map[domain.RoleID]domain.ActionSubmission) []string {
	names := make([]string, 0, len(items))
	for roleID := range items {
		names = append(names, string(roleID))
	}
	slices.Sort(names)
	return names
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}

	cloned := maps.Clone(payload)
	for key, value := range cloned {
		switch typed := value.(type) {
		case map[string]any:
			cloned[key] = clonePayload(typed)
		case []any:
			items := make([]any, len(typed))
			copy(items, typed)
			cloned[key] = items
		}
	}
	return cloned
}

func normalizedHistoryLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	return limit
}

func normalizedBacklogExpiryRounds(rounds int) int {
	if rounds <= 0 {
		return 2
	}
	return rounds
}

func normalizedDelayRounds(rounds int) int {
	if rounds < 0 {
		return 0
	}
	return rounds
}

func minUnits(values ...domain.Units) domain.Units {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func defaultProcurementTerms(hook ProcurementTermsHook) ProcurementTermsHook {
	if hook != nil {
		return hook
	}

	return func(_ ProcurementTermsContext) ProcurementTerms {
		return ProcurementTerms{
			UnitCost:          1,
			LeadTimeRounds:    1,
			OnTimeDeliveryPct: 100,
			KnownSupplier:     true,
		}
	}
}

func defaultProductionBOM(hook ProductionBOMHook) ProductionBOMHook {
	if hook != nil {
		return hook
	}

	return func(_ ProductionBOMContext) ProductionBOM {
		return ProductionBOM{KnownProduct: true}
	}
}

func defaultProductionRoute(hook ProductionRouteHook) ProductionRouteHook {
	if hook != nil {
		return hook
	}

	return func(ctx ProductionRouteContext) ProductionRouteStep {
		index := workstationIndex(ctx.State.Plant.Workstations, ctx.CurrentStationID)
		if index < 0 || index == len(ctx.State.Plant.Workstations)-1 {
			return ProductionRouteStep{Finished: true}
		}
		return ProductionRouteStep{NextWorkstationID: ctx.State.Plant.Workstations[index+1].WorkstationID}
	}
}

func defaultProductionCost(hook ProductionCostHook) ProductionCostHook {
	if hook != nil {
		return hook
	}

	return func(_ ProductionCostContext) ProductionCost {
		return ProductionCost{CostPerCapacityUnit: 1}
	}
}

func spendForQuantity(quantity domain.Units, unitCost domain.Money) domain.Money {
	return domain.Money(quantity) * unitCost
}

func inventoryCost(quantity domain.Units, unitCost domain.Money) domain.Money {
	return domain.Money(quantity) * unitCost
}

func perUnitCost(totalCost domain.Money, quantity domain.Units) domain.Money {
	if quantity <= 0 {
		return 0
	}
	return totalCost / domain.Money(quantity)
}

func affordableQuantity(available domain.Money, unitCost domain.Money) domain.Units {
	if available <= 0 || unitCost <= 0 {
		return 0
	}
	return domain.Units(available / unitCost)
}

func carryingCost(amount domain.Money) domain.Money {
	if amount <= 0 {
		return 0
	}
	return max(1, (amount+9)/10)
}

func minCapacity(values ...domain.CapacityUnits) domain.CapacityUnits {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func minMoney(values ...domain.Money) domain.Money {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func sumCommitments(items []domain.CashCommitment) domain.Money {
	total := domain.Money(0)
	for _, item := range items {
		total += item.Amount
	}
	return total
}
