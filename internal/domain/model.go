package domain

import (
	"maps"
	"slices"
	"time"
)

type MatchID string
type ScenarioID string
type RoundNumber int

type ActorID string

type ProductID string
type PartID string
type CustomerID string
type SupplierID string
type WorkstationID string
type MetricID string
type EventID string
type CommentaryID string
type ActionID string

type Units int
type Money int
type CapacityUnits int
type Percentage int

const ActorPlantSystem ActorID = "plant_system"

// MatchState is the canonical in-memory snapshot for the round currently being collected.
type MatchState struct {
	MatchID       MatchID
	ScenarioID    ScenarioID
	CurrentRound  RoundNumber
	Roles         []RoleAssignment
	Plant         PlantState
	Customers     []CustomerState
	ActiveTargets BudgetTargets
	Metrics       PlantMetrics
	History       RoundHistory
}

type RoleAssignment struct {
	RoleID    RoleID
	PlayerID  string
	IsHuman   bool
	Provider  string
	ModelName string
}

type PlantState struct {
	Cash              Money
	Debt              Money
	DebtCeiling       Money
	PartsInventory    []PartInventory
	WIPInventory      []WIPInventory
	FinishedInventory []FinishedInventory
	InTransitSupply   []SupplyLot
	Workstations      []WorkstationState
	Backlog           []BacklogEntry
}

type PartInventory struct {
	PartID    PartID
	OnHandQty Units
	UnitCost  Money
}

type WIPInventory struct {
	ProductID     ProductID
	WorkstationID WorkstationID
	Quantity      Units
	UnitCost      Money
}

type FinishedInventory struct {
	ProductID ProductID
	OnHandQty Units
	UnitCost  Money
}

type SupplyLot struct {
	PurchaseOrderID string
	SupplierID      SupplierID
	PartID          PartID
	Quantity        Units
	UnitCost        Money
	OrderedRound    RoundNumber
	ArrivalRound    RoundNumber
}

type WorkstationState struct {
	WorkstationID    WorkstationID
	DisplayName      string
	CapacityPerRound CapacityUnits
	CapacityUsed     CapacityUnits
}

type ProductDefinition struct {
	ProductID   ProductID
	DisplayName string
	BOM         []BOMLine
	Route       []WorkstationID
}

type BOMLine struct {
	PartID   PartID
	Quantity Units
}

type PartDefinition struct {
	PartID      PartID
	DisplayName string
	DefaultCost Money
}

type CustomerState struct {
	CustomerID  CustomerID
	DisplayName string
	Sentiment   int
	Backlog     []BacklogEntry
}

type BudgetTargets struct {
	EffectiveRound        RoundNumber
	ProcurementBudget     Money
	ProductionSpendBudget Money
	RevenueTarget         Money
	CashFloorTarget       Money
	DebtCeilingTarget     Money
}

type PlantMetrics struct {
	ThroughputRevenue     Money
	OperatingExpense      Money
	ProcurementSpend      Money
	ProductionSpend       Money
	HoldingCost           Money
	DebtServiceCost       Money
	InventoryValue        Money
	NetCashChange         Money
	RoundProfit           Money
	OnTimeShipmentRate    Percentage
	BacklogUnits          Units
	LostSalesUnits        Units
	PartsOnHandUnits      Units
	FinishedGoodsUnits    Units
	ProductionOutputUnits Units
}

type MetricValue struct {
	MetricID    MetricID
	Value       int
	DisplayUnit string
}

type ActionSubmission struct {
	ActionID    ActionID
	MatchID     MatchID
	Round       RoundNumber
	RoleID      RoleID
	SubmittedAt time.Time
	Action      RoleAction
	Commentary  CommentaryRecord
}

type RoleAction struct {
	Procurement *ProcurementAction
	Production  *ProductionAction
	Sales       *SalesAction
	Finance     *FinanceAction
}

type ProcurementAction struct {
	Orders []PurchaseOrderIntent
}

type PurchaseOrderIntent struct {
	PartID     PartID
	SupplierID SupplierID
	Quantity   Units
}

type ProductionAction struct {
	Releases           []ProductionRelease
	CapacityAllocation []CapacityAllocation
}

type ProductionRelease struct {
	ProductID ProductID
	Quantity  Units
}

type CapacityAllocation struct {
	WorkstationID WorkstationID
	ProductID     ProductID
	Capacity      CapacityUnits
}

type SalesAction struct {
	ProductOffers []ProductOffer
}

type ProductOffer struct {
	ProductID ProductID
	UnitPrice Money
}

type FinanceAction struct {
	NextRoundTargets BudgetTargets
}

type CustomerOrder struct {
	OrderID      string
	CustomerID   CustomerID
	ProductID    ProductID
	OrderedQty   Units
	ShippedQty   Units
	UnitPrice    Money
	OrderedRound RoundNumber
}

type BacklogEntry struct {
	CustomerID  CustomerID
	ProductID   ProductID
	Quantity    Units
	OriginRound RoundNumber
	AgeInRounds int
}

type ShipmentRecord struct {
	CustomerID CustomerID
	ProductID  ProductID
	Quantity   Units
	UnitPrice  Money
	ShipRound  RoundNumber
}

type RoundRecord struct {
	Round      RoundNumber
	Actions    []ActionSubmission
	Events     []RoundEvent
	Commentary []CommentaryRecord
	Metrics    PlantMetrics
}

type RoundHistory struct {
	RecentRounds []RoundRecord
}

type CommentaryRecord struct {
	CommentaryID CommentaryID
	MatchID      MatchID
	Round        RoundNumber
	ActorID      ActorID
	RoleID       RoleID
	Visibility   CommentaryVisibility
	Body         string
}

type CommentaryVisibility string

// CommentaryPublic is the only MVP visibility class; all stored commentary is public after reveal.
const CommentaryPublic CommentaryVisibility = "public"

type RoundEvent struct {
	EventID EventID
	MatchID MatchID
	Round   RoundNumber
	Type    RoundEventType
	ActorID ActorID
	Summary string
	Payload map[string]any
}

type RoundEventType string

const (
	EventBudgetActivated        RoundEventType = "budget_activated"
	EventPurchaseOrderPlaced    RoundEventType = "purchase_order_placed"
	EventSupplyArrived          RoundEventType = "supply_arrived"
	EventProductionReleased     RoundEventType = "production_released"
	EventWorkAdvanced           RoundEventType = "work_advanced"
	EventFinishedGoodsProduced  RoundEventType = "finished_goods_produced"
	EventDemandRealized         RoundEventType = "demand_realized"
	EventShipmentCompleted      RoundEventType = "shipment_completed"
	EventBacklogCreated         RoundEventType = "backlog_created"
	EventBacklogExpired         RoundEventType = "backlog_expired"
	EventCustomerSentimentMoved RoundEventType = "customer_sentiment_moved"
	EventCashChanged            RoundEventType = "cash_changed"
	EventMetricSnapshot         RoundEventType = "metric_snapshot"
	EventRuleAdjustment         RoundEventType = "rule_adjustment"
)

type RoundView struct {
	MatchID          MatchID
	Round            RoundNumber
	ViewerRoleID     RoleID
	Plant            PlantState
	Customers        []CustomerState
	ActiveTargets    BudgetTargets
	Metrics          PlantMetrics
	RecentEvents     []RoundEvent
	RecentCommentary []CommentaryRecord
}

func (s MatchState) Clone() MatchState {
	return MatchState{
		MatchID:       s.MatchID,
		ScenarioID:    s.ScenarioID,
		CurrentRound:  s.CurrentRound,
		Roles:         slices.Clone(s.Roles),
		Plant:         s.Plant.Clone(),
		Customers:     cloneSlice(s.Customers, CustomerState.Clone),
		ActiveTargets: s.ActiveTargets,
		Metrics:       s.Metrics,
		History:       s.History.Clone(),
	}
}

func (s PlantState) Clone() PlantState {
	return PlantState{
		Cash:              s.Cash,
		Debt:              s.Debt,
		DebtCeiling:       s.DebtCeiling,
		PartsInventory:    slices.Clone(s.PartsInventory),
		WIPInventory:      slices.Clone(s.WIPInventory),
		FinishedInventory: slices.Clone(s.FinishedInventory),
		InTransitSupply:   slices.Clone(s.InTransitSupply),
		Workstations:      slices.Clone(s.Workstations),
		Backlog:           slices.Clone(s.Backlog),
	}
}

func (s CustomerState) Clone() CustomerState {
	return CustomerState{
		CustomerID:  s.CustomerID,
		DisplayName: s.DisplayName,
		Sentiment:   s.Sentiment,
		Backlog:     slices.Clone(s.Backlog),
	}
}

func (h RoundHistory) Clone() RoundHistory {
	return RoundHistory{
		RecentRounds: cloneSlice(h.RecentRounds, RoundRecord.Clone),
	}
}

func (h RoundHistory) Recent(limit int) RoundHistory {
	if limit <= 0 || len(h.RecentRounds) <= limit {
		return h.Clone()
	}

	start := len(h.RecentRounds) - limit
	return RoundHistory{
		RecentRounds: cloneSlice(h.RecentRounds[start:], RoundRecord.Clone),
	}
}

func (r RoundRecord) Clone() RoundRecord {
	return RoundRecord{
		Round:      r.Round,
		Actions:    cloneSlice(r.Actions, ActionSubmission.Clone),
		Events:     cloneSlice(r.Events, RoundEvent.Clone),
		Commentary: cloneSlice(r.Commentary, CommentaryRecord.Clone),
		Metrics:    r.Metrics,
	}
}

func (s ActionSubmission) Clone() ActionSubmission {
	return ActionSubmission{
		ActionID:    s.ActionID,
		MatchID:     s.MatchID,
		Round:       s.Round,
		RoleID:      s.RoleID,
		SubmittedAt: s.SubmittedAt,
		Action:      s.Action.Clone(),
		Commentary:  s.Commentary.Clone(),
	}
}

func (a RoleAction) Clone() RoleAction {
	return RoleAction{
		Procurement: clonePtr(a.Procurement, ProcurementAction.Clone),
		Production:  clonePtr(a.Production, ProductionAction.Clone),
		Sales:       clonePtr(a.Sales, SalesAction.Clone),
		Finance:     clonePtr(a.Finance, FinanceAction.Clone),
	}
}

func (a ProcurementAction) Clone() ProcurementAction {
	return ProcurementAction{Orders: slices.Clone(a.Orders)}
}

func (a ProductionAction) Clone() ProductionAction {
	return ProductionAction{
		Releases:           slices.Clone(a.Releases),
		CapacityAllocation: slices.Clone(a.CapacityAllocation),
	}
}

func (a SalesAction) Clone() SalesAction {
	return SalesAction{ProductOffers: slices.Clone(a.ProductOffers)}
}

func (a FinanceAction) Clone() FinanceAction {
	return FinanceAction{NextRoundTargets: a.NextRoundTargets}
}

func (r CommentaryRecord) Clone() CommentaryRecord {
	return r
}

func (e RoundEvent) Clone() RoundEvent {
	return RoundEvent{
		EventID: e.EventID,
		MatchID: e.MatchID,
		Round:   e.Round,
		Type:    e.Type,
		ActorID: e.ActorID,
		Summary: e.Summary,
		Payload: clonePayload(e.Payload),
	}
}

func (v RoundView) Clone() RoundView {
	return RoundView{
		MatchID:          v.MatchID,
		Round:            v.Round,
		ViewerRoleID:     v.ViewerRoleID,
		Plant:            v.Plant.Clone(),
		Customers:        cloneSlice(v.Customers, CustomerState.Clone),
		ActiveTargets:    v.ActiveTargets,
		Metrics:          v.Metrics,
		RecentEvents:     cloneSlice(v.RecentEvents, RoundEvent.Clone),
		RecentCommentary: cloneSlice(v.RecentCommentary, CommentaryRecord.Clone),
	}
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}

	cloned := maps.Clone(payload)
	for key, value := range cloned {
		cloned[key] = cloneAny(value)
	}

	return cloned
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return clonePayload(typed)
	case []any:
		cloned := make([]any, len(typed))
		for i := range typed {
			cloned[i] = cloneAny(typed[i])
		}
		return cloned
	default:
		return typed
	}
}

func cloneSlice[T any](items []T, clone func(T) T) []T {
	if items == nil {
		return nil
	}

	cloned := make([]T, len(items))
	for i := range items {
		cloned[i] = clone(items[i])
	}

	return cloned
}

func clonePtr[T any](item *T, clone func(T) T) *T {
	if item == nil {
		return nil
	}

	cloned := clone(*item)
	return &cloned
}
