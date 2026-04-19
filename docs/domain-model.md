# Canonical Domain Model

This document is the acceptance target for roadmap issue `#2`: design the shared domain types used by simulation, UI, persistence, and AI-facing prompt assembly before implementation spreads across packages.

The goal is not to lock every field forever. The goal is to lock the vocabulary, record shapes, and boundaries that other packages can safely build around.

## Design Intent

The domain model should:

- give `internal/domain` a stable first surface area
- keep engine, UI, persistence, and prompts talking about the same things with the same names
- model the MVP cleanly without blocking post-MVP expansion
- prefer append-only history and explicit events over implicit state changes

## Naming Rules

These terms are canonical and should be reused consistently in code, prompts, logs, and UI labels.

- A `role` is a player-facing responsibility such as Procurement or Sales.
- An `actor` is either a player role or the plant system.
- An `action` is an intent submitted during hidden turn collection.
- An `event` is a resolved plant outcome emitted by the system.
- `inventory` means on-hand stock owned by the plant.
- `in_transit_supply` means purchased parts that have been ordered but not yet received.
- `work_in_progress` means units already released into the route but not yet finished.
- A `backlog` entry is accepted demand that was not yet shipped.
- `metrics` are computed operational or financial measurements, not player intents.

## MVP Canonical Roles

The MVP canonical role set is:

- `procurement_manager`
- `production_manager`
- `sales_manager`
- `finance_controller`

The plant is modeled separately as a system actor:

- `plant_system`

`Plant Manager` remains a valid future role idea from the broader vision, but it is not part of the MVP role enum and should not appear in MVP action contracts or AI role assignment.

## Core Scalar And Identifier Types

The first implementation should treat these as small, explicit value types even if they initially compile down to `string` or `int`.

```go
type MatchID string
type ScenarioID string
type RoundNumber int

type RoleID string
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
```

Recommended enum values:

```go
const (
    RoleProcurementManager RoleID = "procurement_manager"
    RoleProductionManager  RoleID = "production_manager"
    RoleSalesManager       RoleID = "sales_manager"
    RoleFinanceController  RoleID = "finance_controller"
)

const (
    ActorPlantSystem ActorID = "plant_system"
)
```

## Shared Aggregate Types

These are the shared records that other packages should expect to consume or produce.

### Match And Round State

```go
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
```

```go
type RoleAssignment struct {
    RoleID    RoleID
    PlayerID  string
    IsHuman   bool
    Provider  string
    ModelName string
}
```

### Plant State

`PlantState` is the canonical simulation snapshot. UI projections and prompts should be derived from this shape rather than inventing their own names.

```go
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
```

Inventory concepts are intentionally split by state:

- `PartInventory` for purchased parts available for use
- `WIPInventory` for released but unfinished units by route stage
- `FinishedInventory` for shippable completed units
- `SupplyLot` for ordered parts not yet received

```go
type PartInventory struct {
    PartID    PartID
    OnHandQty Units
}

type WIPInventory struct {
    ProductID      ProductID
    WorkstationID  WorkstationID
    Quantity       Units
}

type FinishedInventory struct {
    ProductID ProductID
    OnHandQty Units
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
```

### Product, Part, And Customer Records

Static scenario data and dynamic state should use the same identifiers.

```go
type ProductDefinition struct {
    ProductID      ProductID
    DisplayName    string
    BOM            []BOMLine
    Route          []WorkstationID
}

type BOMLine struct {
    PartID    PartID
    Quantity  Units
}

type PartDefinition struct {
    PartID       PartID
    DisplayName  string
    DefaultCost  Money
}

type CustomerState struct {
    CustomerID      CustomerID
    DisplayName     string
    Sentiment       int
    Backlog         []BacklogEntry
}
```

### Budgets, Targets, And Metrics

Budgets are directives. Metrics are observed outcomes.

```go
type BudgetTargets struct {
    EffectiveRound          RoundNumber
    ProcurementBudget       Money
    ProductionSpendBudget   Money
    RevenueTarget           Money
    CashFloorTarget         Money
    DebtCeilingTarget       Money
}
```

```go
type PlantMetrics struct {
    ThroughputRevenue     Money
    OperatingExpense      Money
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
```

If the implementation needs a generic metric feed for UI panes or prompts, it should project from the typed fields above into:

```go
type MetricValue struct {
    MetricID    MetricID
    Value       int
    DisplayUnit string
}
```

## Action Model

Every player turn submission should use a shared envelope plus a role-specific payload.

```go
type ActionSubmission struct {
    ActionID     ActionID
    MatchID      MatchID
    Round        RoundNumber
    RoleID       RoleID
    SubmittedAt  time.Time
    Action       RoleAction
    Commentary   CommentaryRecord
}
```

```go
type RoleAction struct {
    Procurement *ProcurementAction
    Production  *ProductionAction
    Sales       *SalesAction
    Finance     *FinanceAction
}
```

Only the payload matching `RoleID` should be populated.

### Procurement

```go
type ProcurementAction struct {
    Orders []PurchaseOrderIntent
}

type PurchaseOrderIntent struct {
    PartID    PartID
    SupplierID SupplierID
    Quantity  Units
}
```

### Production

```go
type ProductionAction struct {
    Releases          []ProductionRelease
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
```

### Sales

```go
type SalesAction struct {
    ProductOffers []ProductOffer
}

type ProductOffer struct {
    ProductID  ProductID
    UnitPrice  Money
}
```

### Finance

```go
type FinanceAction struct {
    NextRoundTargets BudgetTargets
}
```

## Demand, Orders, Supply, And Shipment Records

These records are outcomes the engine owns after actions resolve.

```go
type CustomerOrder struct {
    OrderID       string
    CustomerID    CustomerID
    ProductID     ProductID
    OrderedQty    Units
    ShippedQty    Units
    UnitPrice     Money
    OrderedRound  RoundNumber
}

type BacklogEntry struct {
    CustomerID     CustomerID
    ProductID      ProductID
    Quantity       Units
    OriginRound    RoundNumber
    AgeInRounds    int
}

type ShipmentRecord struct {
    CustomerID   CustomerID
    ProductID    ProductID
    Quantity     Units
    UnitPrice    Money
    ShipRound    RoundNumber
}
```

## Event And Commentary Model

The round record is append-only. State snapshots may be persisted for convenience, but history should still be reconstructable from accepted actions, events, and scenario data.

```go
type RoundRecord struct {
    Round        RoundNumber
    Actions      []ActionSubmission
    Events       []RoundEvent
    Commentary   []CommentaryRecord
    Metrics      PlantMetrics
}

type RoundHistory struct {
    RecentRounds []RoundRecord
}
```

`Actions` are accepted round metadata revealed with the completed round record. `Events` are the public plant outcomes produced while resolving those accepted actions.

```go
type CommentaryRecord struct {
    CommentaryID CommentaryID
    MatchID      MatchID
    Round        RoundNumber
    ActorID      ActorID
    RoleID       RoleID
    Visibility   CommentaryVisibility
    Body         string
}
```

```go
type CommentaryVisibility string

const (
    CommentaryPublic CommentaryVisibility = "public"
)
```

```go
type RoundEvent struct {
    EventID     EventID
    MatchID     MatchID
    Round       RoundNumber
    Type        RoundEventType
    ActorID     ActorID
    Summary     string
    Payload     map[string]any
}
```

Recommended MVP event types:

```go
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
```

`Payload` should carry machine-readable details specific to the event, but `Type` must stay stable so UI filters, persistence, and prompts are not coupled to prose.

## Round View Contract

Prompt assembly, TUI rendering, and human-input screens should all work from the same round-view contract rather than reading simulation internals directly.

```go
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
```

This keeps AI prompts and UI panes aligned around the same data slices:

- left pane or role header: `ViewerRoleID`
- center event log: `RecentEvents` and `RecentCommentary`
- right metrics pane: `Metrics`
- action form: role-specific subset of `RoundView`

## Explicit Non-Goals For The First Domain Pass

The first domain model should not try to solve everything at once.

- No generic ECS or highly abstract resource graph
- No multi-plant network model
- No supplier-specific lead-time rules in the shared types yet
- No workstation scheduling calendar, batch, setup, maintenance, failure, or repair model in the MVP domain contract yet
- No TOC-specific bottleneck-control entities such as drum-buffer-rope buffers, planner queues, scheduler states, or expeditor workflows yet
- No hidden/private commentary visibility classes until the game actually needs them
- No provider-specific AI fields in domain records beyond stable role assignment metadata

Those concepts are expected post-MVP, but they should be added only when the rules are concrete enough to justify stable shared types.

## Stability Rules

Before adding new types elsewhere, contributors should prefer extending this model or projecting from it.

- Engine code should mutate `PlantState`, emit `RoundEvent`, and calculate `PlantMetrics`.
- UI code should render `RoundView` and avoid inventing parallel inventory or event names.
- Persistence should store snapshots and/or event history using these identifiers and record names.
- Prompt builders should serialize `RoundView`, role descriptors, and recent commentary using the same terminology in this document.

## Acceptance Checklist

Issue `#2` should be considered satisfied when:

- core structs and enums are identifiable from this document before package scaffolding starts
- the MVP role set is explicit and stable
- plant state, actions, metrics, events, and commentary use one shared vocabulary
- engine, UI, persistence, and AI prompt work can all point back to this document as the canonical source
