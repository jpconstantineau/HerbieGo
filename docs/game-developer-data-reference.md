# Game Developer Data Reference

This document is the developer-facing source of truth for HerbieGo data metadata across:

- data catalog
- data dictionary
- metric lineage
- report lineage
- role and report delivery coverage
- implementation gaps and follow-up roadmap

It is grounded in the current repository state as implemented in:

- `internal/domain`
- `internal/engine`
- `internal/scenario`
- `internal/projection`
- `internal/app`
- `internal/adapters/tui`
- the current role, report, and playbook documentation under `docs/`

This document distinguishes carefully between:

- implemented runtime behavior
- documented MVP intent that is not fully implemented yet
- forward-looking future-role documentation

## 1. Current Runtime Scope

### Playable roles implemented in code

- `procurement_manager`
- `production_manager`
- `sales_manager`
- `finance_controller`

### Future roles documented but not implemented in the action roster

- `quality_manager`
- `logistics_and_warehouse_manager`
- `maintenance_manager`
- `plant_manager`

### Runtime report surfaces implemented today

- shared `RoundView` projection for every current MVP role
- shared `RoleRoundReport` with `CompanywidePerformanceReport` and `DepartmentPerformanceReport`
- Bubble Tea TUI panes consuming `RoundView` and `RoleRoundReport`
- AI prompt assembly consuming `RoundView`, `RoleRoundReport`, and a hard-coded role briefing

### Important design truth

The repository already contains richer report-design documents than the runtime currently supports. The current executable path uses a compact report projection, not the full report structures described under `docs/reports/`.

## 2. Data Catalog

## 2.1 Static Scenario Catalog

These records are authored in scenario definitions and are effectively the seed catalog for a match.

| Domain | Record | Source | Purpose |
| --- | --- | --- | --- |
| Scenario | `Definition` | `internal/scenario/scenario.go` | Binds setup, starting conditions, market model, and production model together. |
| Match setup | `MatchSetup` | `internal/scenario/scenario.go` | Declares the role roster for the scenario. |
| Starting conditions | `StartingConditions` | `internal/scenario/scenario.go` | Seeds starting budgets, plant state, and customer state. |
| Products | `Product` | `internal/scenario/scenario.go` | Defines canonical product IDs, BOM, route, and base unit cost. |
| Parts | `Part` | `internal/scenario/scenario.go` | Defines canonical part IDs, display names, default cost, and supplier mapping. |
| Workstations | `Workstation` | `internal/scenario/scenario.go` | Defines route stations, capacity, and per-unit production cost. |
| Customers | `CustomerMarket` | `internal/scenario/scenario.go` | Defines customer demand by product. |
| Demand profiles | `DemandProfile` | `internal/scenario/scenario.go` | Defines `reference_price`, `base_demand`, and `price_sensitivity`. |
| Bottleneck assumption | `BottleneckAssumption` | `internal/scenario/scenario.go` | Documents the intended bottleneck for the scenario. |

### Static lookup surface available to AI tooling

The current lookup tool surface exposes:

- `list_valid_suppliers`
- `show_product_route`
- `show_product_bom`
- `show_customer_demand_profile`

This is a partial catalog surface. It is useful for prompts, but it is not yet a complete developer or player-facing metadata catalog.

## 2.2 Runtime State Catalog

These records hold the active match state.

| Domain | Record | Source | Purpose |
| --- | --- | --- | --- |
| Match | `MatchState` | `internal/domain/model.go` | Canonical in-memory match snapshot. |
| Role assignment | `RoleAssignment` | `internal/domain/model.go` | Connects roles to a human or AI controller. |
| Round flow | `RoundFlowState` | `internal/domain/model.go` | Tracks hidden-turn collection and reveal state. |
| Plant | `PlantState` | `internal/domain/model.go` | Core operational and financial plant snapshot. |
| Parts inventory | `PartInventory` | `internal/domain/model.go` | On-hand purchased inventory. |
| WIP inventory | `WIPInventory` | `internal/domain/model.go` | Product units by route stage. |
| Finished goods | `FinishedInventory` | `internal/domain/model.go` | Shippable completed units. |
| In-transit supply | `SupplyLot` | `internal/domain/model.go` | Ordered material not yet received. |
| Workstations | `WorkstationState` | `internal/domain/model.go` | Active capacity pools and usage. |
| Backlog | `BacklogEntry` | `internal/domain/model.go` | Accepted but unshipped demand. |
| Customers | `CustomerState` | `internal/domain/model.go` | Customer sentiment plus customer-specific backlog. |
| Targets | `BudgetTargets` | `internal/domain/model.go` | Finance-owned next-round targets and active soft limits. |
| Metrics | `PlantMetrics` | `internal/domain/model.go` | Current derived plant metrics. |
| History | `RoundHistory` / `RoundRecord` | `internal/domain/model.go` | Append-only resolved-round history. |

## 2.3 Event Catalog

These event types are currently defined and may be appended to a resolved round:

- `budget_activated`
- `purchase_order_placed`
- `supply_arrived`
- `production_released`
- `work_advanced`
- `finished_goods_produced`
- `demand_realized`
- `shipment_completed`
- `backlog_created`
- `backlog_expired`
- `customer_sentiment_moved`
- `cash_changed`
- `metric_snapshot`
- `rule_adjustment`

These events are the most important public lineage bridge between engine resolution and downstream reporting.

## 2.4 Projection Catalog

These records are derived from runtime state and history for human and AI consumption.

| Projection | Source | Purpose |
| --- | --- | --- |
| `RoundView` | `internal/projection/round_view.go` | Canonical role-facing round context. |
| `RoundHistoryEntry` | `internal/projection/round_view.go` | Historical round summaries included in the view. |
| `RoleRoundReport` | `internal/projection/role_report.go` | Role report delivered alongside the round view. |
| `CompanywidePerformanceReport` | `internal/projection/role_report.go` | Compact company snapshot for all roles. |
| `DepartmentPerformanceReport` | `internal/projection/role_report.go` | Compact role-specific metrics and notes. |

## 2.5 Presentation Catalog

| Surface | Source | Delivered data |
| --- | --- | --- |
| TUI departments pane | `internal/adapters/tui/model.go` | role list, bonus reminder, department detail lines |
| TUI history workspace | `internal/adapters/tui/model.go` | `RoundView.RecentRounds`, timeline, commentary, events |
| TUI report workspace | `internal/adapters/tui/model.go` | `RoleRoundReport.Companywide`, `RoleRoundReport.Department` |
| TUI stats pane | `internal/adapters/tui/model.go` | `RoundView.Metrics`, `PlantState`, targets, department key metrics |
| AI prompts | `internal/prompting/ai.go` and `internal/app/ai_orchestrator.go` | `RoundView`, `RoleRoundReport`, hard-coded role briefing, previous action, lookup tools |

## 3. Data Dictionary

## 3.1 Canonical identifiers

| Identifier | Meaning |
| --- | --- |
| `MatchID` | Unique match instance identifier. |
| `ScenarioID` | Scenario definition identifier. |
| `RoundNumber` | Current or historical weekly turn number. |
| `RoleID` | Canonical playable role identifier. |
| `ActorID` | Either a role actor or the `plant_system`. |
| `ProductID` | Canonical product key such as `pump` or `valve`. |
| `PartID` | Canonical purchased-material key such as `housing`. |
| `CustomerID` | Canonical customer key such as `northbuild`. |
| `SupplierID` | Canonical supplier key such as `forgeco`. |
| `WorkstationID` | Canonical route-stage key such as `fabrication`. |
| `MetricID` | Report metric key used in compact department reports. |
| `EventID` | Stable event identifier within a round history. |
| `ActionID` | Stable player submission identifier. |

## 3.2 Operational and financial state

| Field | Meaning |
| --- | --- |
| `Plant.Cash` | Current cash after procurement, production, shipments, and round-end costs. |
| `Plant.Debt` | Short-term debt created when spend exceeds cash. |
| `Plant.DebtCeiling` | Hard legal debt limit used by spend-cap checks. |
| `PartsInventory.OnHandQty` | Available purchased units by part. |
| `WIPInventory.Quantity` | Product units currently at a workstation stage. |
| `FinishedInventory.OnHandQty` | Finished goods immediately available to ship. |
| `SupplyLot.Quantity` | Material units already ordered but not yet received. |
| `WorkstationState.CapacityPerRound` | Total available station capacity this round. |
| `WorkstationState.CapacityUsed` | Capacity consumed during the current resolution. |
| `BacklogEntry.Quantity` | Accepted demand not yet shipped. |
| `BacklogEntry.AgeInRounds` | Number of completed rounds backlog has survived. |
| `CustomerState.Sentiment` | Demand modifier affected by service performance. |
| `BudgetTargets.*` | Active or future finance targets used as soft guidance and hard trim thresholds. |

## 3.3 Derived metrics

| Metric | Meaning |
| --- | --- |
| `ThroughputRevenue` | Shipment revenue recognized in the resolved round. |
| `OperatingExpense` | Procurement spend + production spend + holding cost + debt service cost. |
| `ProcurementSpend` | Spend recognized from accepted purchase orders in the round. |
| `ProductionSpend` | Spend recognized from accepted capacity advancement in the round. |
| `HoldingCost` | Round-end carrying cost on inventory value. |
| `DebtServiceCost` | Round-end carrying cost on outstanding debt. |
| `InventoryValue` | End-of-round parts + WIP + finished goods book value. |
| `NetCashChange` | Round revenue minus round operating expense. |
| `RoundProfit` | Round revenue minus round operating expense. |
| `OnTimeShipmentRate` | `shipped_units / (shipped_units + backlog_units + lost_sales_units)`. |
| `BacklogUnits` | End-of-round backlog quantity. |
| `LostSalesUnits` | Backlog units expired in the round. |
| `PartsOnHandUnits` | End-of-round on-hand part units. |
| `FinishedGoodsUnits` | End-of-round finished-goods units. |
| `ProductionOutputUnits` | Units completed into finished goods during the round. |

## 3.4 Compact report metric IDs

These `MetricID` values exist today in `DepartmentPerformanceReport.KeyMetrics`:

- `ordered_parts`
- `parts_on_hand`
- `wip_units`
- `output_units`
- `sales_pipeline`
- `throughput_revenue`
- `margin`
- `cash_position`

Important limitation:

These are presentation-specific metric IDs. They are not yet a governed, versioned report-schema catalog with field definitions, owners, thresholds, or compatibility guarantees.

## 4. Code-To-Report Lineage

## 4.1 Resolution flow

The current runtime lineage is:

1. `scenario.Definition.InitialState` seeds the first `MatchState`.
2. `engine.Resolver.ResolveRound` validates actions and resolves:
- procurement
- supply receipt
- production
- sales
- scenario world update demand creation
- finance target activation for next round
- round-end carrying and debt costs
3. `roundPhase.computeMetrics` derives `PlantMetrics`.
4. `MatchState.History` appends the resolved `RoundRecord`.
5. `projection.BuildRoundView` projects state and recent history into `RoundView`.
6. `projection.BuildRoleRoundReport` derives a compact companywide and department report.
7. TUI and AI prompt assembly present `RoundView` and `RoleRoundReport` to the role controller.

## 4.2 Event lineage by business area

| Business area | Code path | Primary events emitted | Downstream consumers |
| --- | --- | --- | --- |
| Procurement | `roundPhase.resolveProcurement` | `purchase_order_placed`, `cash_changed`, `rule_adjustment` | History feed, company financial summary, procurement report notes, AI prompt context |
| Supply receipt | `roundPhase.receiveSupply` | `supply_arrived` | History feed, procurement reasoning context |
| Production release and advancement | `roundPhase.resolveProduction` | `production_released`, `work_advanced`, `finished_goods_produced`, `rule_adjustment`, `cash_changed` | History feed, production/company reports, AI prompt context |
| Sales and shipment | `roundPhase.resolveSales` | `shipment_completed`, `cash_changed` | History feed, company financial summary, sales/company reports |
| Demand generation | `scenario.Definition.applyDemand` | `demand_realized`, `backlog_created` | New sales summary, backlog views, sales/company reports |
| Backlog aging and sentiment | `roundPhase.finalizeRound` | `backlog_expired`, `customer_sentiment_moved` | History feed, customer sentiment state, sales report inputs |
| Metric close | `roundPhase.finalizeRound` | `metric_snapshot` | Round history, TUI history/archive, prompt context |

## 4.3 Metric lineage

| Metric | Calculated from | Code path | Presented to |
| --- | --- | --- | --- |
| `ThroughputRevenue` | Sum of shipment quantity x unit price | `resolveSales` -> `stats.revenue` -> `computeMetrics` | TUI stats pane, Sales dept report, company financial summary, all AI roles via `RoundView.Metrics` |
| `ProcurementSpend` | Sum of accepted PO quantity x unit cost | `resolveProcurement` -> `stats.procurementSpend` -> `computeMetrics` | All AI roles via `RoundView.Metrics`; not directly surfaced in TUI/report detail today |
| `ProductionSpend` | Sum of accepted capacity advance x cost per capacity unit | `resolveProduction` -> `stats.productionSpend` -> `computeMetrics` | All AI roles via `RoundView.Metrics`; not directly surfaced in TUI/report detail today |
| `HoldingCost` | End-of-round carrying cost on parts + WIP + finished inventory | `applyRoundOperatingCosts` -> `computeMetrics` | All AI roles via `RoundView.Metrics`; not directly surfaced in TUI/report detail today |
| `DebtServiceCost` | End-of-round carrying cost on debt | `applyRoundOperatingCosts` -> `computeMetrics` | All AI roles via `RoundView.Metrics`; not directly surfaced in TUI/report detail today |
| `InventoryValue` | Sum of book value across parts, WIP, finished goods | `computeMetrics` | Company inventory summary, all AI roles via `RoundView.Metrics`; not directly shown in TUI stats pane |
| `NetCashChange` | Revenue minus operating expense | `computeMetrics` | Historical archive summaries and AI prompt context; not directly shown in role report |
| `RoundProfit` | Revenue minus operating expense | `computeMetrics` | TUI stats pane, Finance dept report, all AI roles via `RoundView.Metrics` |
| `OnTimeShipmentRate` | `shipped / demand_units` | `computeMetrics` | All AI roles via `RoundView.Metrics`; not directly shown in current TUI role report |
| `BacklogUnits` | Sum of end-of-round backlog quantities | `computeMetrics` | All AI roles via `RoundView.Metrics`; only indirectly surfaced in sales/company summaries |
| `LostSalesUnits` | Sum of expired backlog quantities | `finalizeRound` -> `computeMetrics` | All AI roles via `RoundView.Metrics`; not directly surfaced in current TUI role report |
| `PartsOnHandUnits` | Sum of part inventory quantities | `computeMetrics` | TUI stats pane, Procurement dept report, all AI roles |
| `FinishedGoodsUnits` | Sum of finished goods quantities | `computeMetrics` | TUI stats pane, Production dept detail line, all AI roles |
| `ProductionOutputUnits` | Sum of finished goods produced in the round | `resolveProduction` -> `stats.producedUnits` -> `computeMetrics` | TUI stats/prompt context, Production dept report, company produced summary |

## 4.4 Companywide report field lineage

| Report field | Derived from | Code path | Delivered to |
| --- | --- | --- | --- |
| `NewSales` | `demand_realized` events in latest round | `newSalesSummary` | All current MVP roles |
| `UnshippedSales` | Current `Plant.Backlog` grouped by product | `backlogSummary` | All current MVP roles |
| `SalesAtRisk` | Backlog entries with `AgeInRounds >= 2` | `backlogAtRiskSummary` | All current MVP roles |
| `ProductsProducedLastWeek` | `finished_goods_produced` events in latest round | `producedSummary` | All current MVP roles |
| `CurrentInventoryLevels` | End-of-round value of parts + WIP + finished goods | `inventorySummary` | All current MVP roles |
| `Financials` | Product-level aggregation of shipments, production release material cost, finished-goods inventory cost | `financialSummary` | All current MVP roles |

## 4.5 Department report lineage

| Role | Current department metrics | Derived from | Delivered to |
| --- | --- | --- | --- |
| Procurement | `ordered_parts`, `parts_on_hand` | `InTransitSupply`, `Metrics.PartsOnHandUnits` | TUI report pane, TUI stats pane, AI prompt |
| Production | `wip_units`, `output_units` | `Plant.WIPInventory`, `Metrics.ProductionOutputUnits` | TUI report pane, TUI stats pane, AI prompt |
| Sales | `sales_pipeline`, `throughput_revenue` | `Plant.Backlog`, `Metrics.ThroughputRevenue` | TUI report pane, TUI stats pane, AI prompt |
| Finance | `margin`, `cash_position` | `Metrics.RoundProfit`, `Plant.Cash` | TUI report pane, TUI stats pane, AI prompt |

## 5. Role, Report, and Delivery Coverage

## 5.1 Current MVP role coverage

| Role | Role card | Weekly report doc | Gameplay playbook | Runtime action support | Runtime report support | AI briefing source |
| --- | --- | --- | --- | --- | --- | --- |
| Procurement Manager | Yes | Yes | Yes | Yes | Partial | Hard-coded, not doc-driven |
| Production Manager | Yes | Yes | Yes | Yes | Partial | Hard-coded, not doc-driven |
| Sales Manager | Yes | Yes | Yes | Yes | Partial | Hard-coded, not doc-driven |
| Finance Controller | Yes | Yes | No | Yes | Partial | Hard-coded, not doc-driven |

## 5.2 Future role coverage

| Role | Role card | Weekly report doc | Gameplay playbook | Runtime action support | Runtime report support | Metric lineage support |
| --- | --- | --- | --- | --- | --- | --- |
| Quality Manager | No | Yes | No | No | No | No |
| Logistics and Warehouse Manager | No | Yes | No | No | No | No |
| Maintenance Manager | No | Yes | No | No | No | No |
| Plant Manager | No | Yes | No | No | No | No |

## 5.3 Current report delivery reality

The runtime currently delivers:

- one shared company snapshot to every MVP role
- one compact department metric block per MVP role
- no full report sections matching the detailed report-design docs
- no role-specific trend tables
- no per-customer report views
- no future-role report delivery

## 6. Confirmed Gaps

## 6.1 Documentation and metadata governance gaps

- There was no single developer-facing data catalog, data dictionary, and lineage reference before this document.
- The repo does not yet maintain a governed report schema catalog that defines each report field, owner, formula, and consumer.
- The current AI role briefing is hard-coded in Go rather than sourced from the role cards, playbooks, or report docs.

## 6.2 MVP code-versus-doc gaps

- The current `RoleRoundReport` shape is much smaller than the MVP weekly report specifications for Procurement, Production, Sales, and Finance.
- The current TUI report workspace renders compact summaries, not the structured sections described in `docs/reports/*.md`.
- Several metrics promised in `docs/mvp-game-design.md` are not present as stable derived metrics or surfaced report fields, including:
- workstation utilization by workstation
- production volume by product as a canonical metric set
- units sold by product as a canonical metric set
- lost sales by product
- cumulative profit
- explicit beginning-versus-ending cash trends
- The action vocabulary in `docs/mvp-game-design.md` still mentions `set_offer_quantity(customer_id, product_id, quantity)` for Sales, but the runtime only implements product pricing via `ProductOffer`.

## 6.3 MVP report implementation gaps by role

### Procurement Manager

Missing or partial relative to the report doc:

- per-part inventory status table
- expected burn-rate estimate
- rounds-of-cover calculation
- open-order and arrival table
- spend-condition analysis
- supplier health section

### Production Manager

Missing or partial relative to the report doc:

- per-product target versus feasible output
- per-workstation capacity and bottleneck table
- part-readiness and starvation table
- production spend-pressure section
- quality or rework signal placeholder

### Sales Manager

Missing or partial relative to the report doc:

- current price by product in the report itself
- per-product demand and revenue pipeline table
- backlog aging by product or customer-product segment
- customer sentiment trend section
- price and margin quality section

### Finance Controller

Missing or partial relative to the report doc:

- cash trend section
- debt-headroom trend section
- flash P&L table
- budget-pressure and spend-driver table
- inventory and working-capital table
- decision-prompt sections derived from live metrics

## 6.4 Future-role implementation gaps

For Quality, Logistics/Warehouse, Maintenance, and Plant Manager:

- no canonical role ID in the runtime role roster
- no action envelope or validation rules
- no report projection
- no dedicated metric lineage
- no TUI role view
- no AI briefing source
- no simulation mechanics generating the report data those docs assume

## 7. Roadmap

The roadmap below groups the confirmed gaps into implementation-sized issues.

| Theme | Scope | Status |
| --- | --- | --- |
| Developer metadata governance | Establish catalog, dictionary, lineage, and a maintained schema source | `#178` |
| MVP action and metric contract alignment | Reconcile docs with implemented actions and metric coverage | `#179` |
| Metric granularity expansion | Add product, customer, workstation, and trend-level derived metrics | `#180` |
| Procurement report implementation | Build report-grade Procurement projection from live data | `#181` |
| Production report implementation | Build report-grade Production projection from live data | `#182` |
| Sales report implementation | Build report-grade Sales projection from live data | `#183` |
| Finance report implementation | Build report-grade Finance projection from live data | `#184` |
| Runtime briefing/report single-sourcing | Replace hard-coded runtime briefings with doc-aligned structured data | `#185` |
| Future-role implementation planning | Define canonical data prerequisites for Quality, Logistics, Maintenance, Plant Manager | `#186` |

## 8. Related Existing Issues

The following issues already provide useful context and should not be duplicated:

- `#96` Procurement report intent
- `#97` Production report intent
- `#98` Sales report intent
- `#95` Finance report intent
- `#99` Quality report intent
- `#100` Logistics and Warehouse report intent
- `#101` Maintenance report intent
- `#102` Plant Manager report intent
- `#103` MVP versus future-role documentation scope
- `#104` through `#111` gameplay playbooks
- `#112` annotated sample turns
- `#114` visibility rules
- `#115` KPI thresholds
- `#116` single-source role briefing spec
- `#117` report template structure
- `#142` invariant audit

## 8.1 New issues filed from this audit

- `#178` Create governed report schema catalog from the developer data reference
- `#179` Reconcile MVP action vocabulary and metric contract with the executable runtime
- `#180` Add report-grade derived metrics by product, customer, workstation, and trend window
- `#181` Implement Procurement Manager report projection from live engine data
- `#182` Implement Production Manager report projection from live engine data
- `#183` Implement Sales Manager report projection from live engine data
- `#184` Implement Finance Controller report projection from live engine data
- `#185` Replace hard-coded role briefings with structured doc-aligned metadata
- `#186` Define future-role data prerequisites and placeholder schemas for Quality, Logistics, Maintenance, and Plant Manager

## 9. Guidance For Contributors

When adding or changing gameplay data, ask these questions in order:

1. Is this a static scenario catalog field, mutable match-state field, event payload, or derived metric?
2. What is the canonical identifier and vocabulary for it?
3. Which engine step creates or mutates it?
4. Which report field or prompt section consumes it?
5. Which role actually sees it?
6. Is the data implemented now, documented as MVP intent, or only future-role design?
7. Does the change need a new event, a new derived metric, a new projection field, or all three?

The safest pattern is:

- scenario/static catalog
- engine event emission
- derived metric calculation
- projection field
- role/report presentation
- documentation update

That sequence keeps lineage explicit and helps prevent silent report drift.
