# MVP Game Design

This document defines the first playable version of HerbieGo. It is the acceptance target for roadmap issue `#1`: define the MVP game rules and scope before simulation code spreads across the project.

## MVP Goal

The MVP should create a short, replayable plant-management game where role-specific decisions create tension between local optimization and total plant performance.

The first playable version intentionally focuses on:

- one plant scenario
- three operating roles plus one finance control role
- simultaneous hidden turns
- 52 weekly rounds per match
- two finished products
- two purchased parts per product
- two required workstations in the production route
- one compact and deterministic round-resolution model

The MVP does not yet try to simulate decades of evolution, deep supplier networks, multiple plants, or networked multiplayer.

## Roles In MVP

The MVP includes four player roles plus plant-owned resolution logic.

### Procurement Manager

Responsibilities:

- buy parts for future production
- protect against shortages
- manage procurement spend within budget

Legal actions each round:

- place purchase orders for each part type
- choose order quantity per part type
- attach a short rationale/comment

### Production Manager

Responsibilities:

- convert part inventory into finished products
- allocate finite workstation capacity
- fulfill the plant production plan

Legal actions each round:

- choose how many units of each product to release into production
- assign workstation capacity between products
- attach a short rationale/comment

### Sales Manager

Responsibilities:

- set prices
- choose how much demand to pursue
- convert finished goods into revenue

Legal actions each round:

- set market price for each product
- set offer quantity for each customer-product offer
- attach a short rationale/comment

### Finance Controller

Responsibilities:

- manage short-term liquidity pressure
- set next-round budget limits and operating targets
- surface financial tradeoffs to the rest of the plant

Legal actions each round:

- set next-round budget for procurement
- set next-round budget for production overtime/capacity spend
- set next-round sales target for revenue
- set next-round plant target for cash floor or debt ceiling
- attach a short rationale/comment

Budget rules:

- finance budgets are soft targets
- procurement and production may exceed the active budget by up to `10%`
- spending beyond `110%` of the active budget is illegal and must be trimmed by the plant

### Plant Role

The plant is not a player, but it is a rules-owning entity that resolves all actions.

The plant enforces:

- inventory cannot go negative
- workstation capacity is finite
- finished products can only be sold if they exist in inventory
- part purchases and operating costs can push cash into short-term debt
- debt cannot exceed the current debt ceiling

## Starter Scenario

The MVP uses one fixed starter scenario so the first implementation can stay deterministic and easy to reason about.

### Products

#### Product A: `Pump`

- purchased parts: `Housing`, `Seal Kit`
- route: `Fabrication -> Assembly`

#### Product B: `Valve`

- purchased parts: `Body`, `Fastener Kit`
- route: `Fabrication -> Assembly`

### Customers

The game engine manages multiple customers. The Sales Manager sells to those customers instead of selling into one anonymous market.

The MVP should start with a small fixed customer list, such as:

- `NorthBuild`
- `PrairieFlow`
- `AgriWorks`

Each customer has:

- product-specific base demand
- product-specific price sensitivity
- a sentiment score that affects future demand
- a backlog record for unshipped orders

### Workstations

#### Workstation 1: `Fabrication`

- finite capacity per round
- converts purchased parts into fabricated subassemblies

#### Workstation 2: `Assembly`

- finite capacity per round
- converts fabricated subassemblies into finished goods

### Initial Assumptions

The initial implementation should start with visible scenario constants similar to the following:

- `Pump` and `Valve` each require exactly one unit of each of their two purchased parts
- both products require capacity at both workstations
- `Pump` and `Valve` can consume different amounts of workstation time
- each part, product, workstation capacity, and cash amount is represented as an integer

Exact numeric values belong in scenario data, not hard-coded into the rules text.

## Round Structure

One round represents one week of plant operation.

### Phase 1: Broadcast state

Each player receives:

- current week number
- current cash and debt
- current parts inventory
- current shop floor inventory and work-in-progress by product and workstation stage
- current finished goods inventory
- current customer backlog by customer and product
- current customer sentiment by customer
- workstation capacities for the round
- active budgets and targets set by finance from the previous round
- recent round log and player commentary

### Phase 2: Hidden action selection

All players choose actions simultaneously. Current-turn choices remain hidden until resolution starts.

### Phase 3: Finance target update

The Finance Controller's action creates the budgets and targets that will apply in the next round, not the current round.

This keeps the turn deterministic and prevents finance from retroactively vetoing same-round actions.

### Phase 4: Procurement resolution

The plant resolves purchase orders.

Rules:

- each ordered part type becomes an in-transit purchase order instead of immediate inventory
- purchase spend reduces cash immediately
- procurement may spend beyond cash on hand if the plant remains within the debt ceiling
- if an order would breach the debt ceiling, the order is reduced or rejected until the rule is satisfied

Purchased parts arrive at the end of the next round. In MVP, all suppliers use the same one-round lead time. Supplier-specific lead times are deferred.

### Phase 5: Parts receipt update

The plant receives any purchase orders whose one-round lead time has completed.

Rules:

- arriving parts move from in-transit supply into parts inventory
- received parts become available for production after receipt
- the event log records which orders arrived this week

### Phase 6: Production resolution

The plant resolves production actions using on-hand parts inventory, carried work-in-progress, and finite workstation capacity.

Rules:

- production cannot consume more parts than exist
- production cannot use more workstation capacity than exists at either workstation
- released units first consume required purchased parts and enter shop floor inventory at `Fabrication`
- units that complete `Fabrication` move into intermediate work-in-progress before `Assembly`
- units that complete `Assembly` move into finished goods inventory
- if the Production Manager requests more units than can be advanced, the plant advances the maximum legal quantity
- consumed parts are removed from parts inventory
- any overtime or operating cost is charged to cash and may create debt within the debt ceiling

Shop floor inventory and work-in-progress carry from one round to the next until they are completed, blocked, or scrapped by a future ruleset.

### Phase 7: Sales resolution

The plant converts pricing decisions into customer-level demand, applies backlog rules, then ships from finished goods inventory.

Rules:

- the Sales Manager sets one market price per product for all customers
- each customer may receive a separate offer quantity per product
- demand depends on market price set by the Sales Manager, current customer sentiment, and customer-specific demand
- new accepted demand is added to that customer-product backlog
- shipments draw down backlog from oldest to newest
- realized shipments cannot exceed finished goods inventory
- shipped units reduce finished goods inventory and increase cash through revenue
- backlog entries that remain unshipped for more than `2` rounds become lost sales and are removed from backlog
- expired backlog reduces the affected customer's sentiment, making future sales harder

### Phase 8: End-of-round finance update

The plant applies round-end financial effects.

Rules:

- short-term debt accrues interest or carrying cost
- inventory may incur holding cost
- the event log records key state changes and player commentary
- finance budgets and targets chosen this round become active for the next round

## Economic And Market Logic

The MVP uses a deliberately compact economic model.

### Cash And Debt

- cash may go below zero
- negative cash is treated as short-term debt
- debt may not exceed the active debt ceiling set by finance
- if a player action would push debt past the allowed ceiling, the plant trims or rejects the action

### Inventory Rules

- parts inventory cannot go negative
- shop floor inventory and work-in-progress cannot go negative
- finished goods inventory cannot go negative
- no action can sell, consume, or move more units than exist

### Capacity Rules

- every workstation has finite capacity per round
- each product consumes defined capacity units at each workstation
- the plant computes the maximum feasible production from capacity and parts availability

### Demand Rules

Each customer-product pair has:

- a reference price
- a base demand
- a price sensitivity
- a sentiment modifier

The first implementation should use a simple demand function:

`realized_demand = max(0, (base_demand * sentiment_modifier) - price_sensitivity * (price - reference_price))`

The plant may round demand to an integer after applying the formula.

### Backlog And Sentiment Rules

- accepted demand that is not shipped immediately becomes backlog
- backlog is tracked by customer, product, quantity, and order age in rounds
- the plant ages backlog at the end of each round
- backlog older than `2` rounds expires into lost sales
- each expired backlog event reduces that customer's sentiment score
- sentiment recovers very slowly over time
- sentiment recovers more quickly through reliable service and on-time shipment behavior
- lower sentiment biases future purchase decisions downward until the customer is won back through better service

### Cost Rules

The first implementation should track:

- part purchase cost
- optional production operating cost
- inventory holding cost
- debt carrying cost

### Budget Rules

- finance budgets apply starting next round
- procurement and production are expected to stay at or below budget
- the plant allows spending up to `110%` of budget
- the plant trims any action that would exceed the `110%` hard cap
- budget misses inside the allowed range are logged as soft-target misses

## Action Vocabulary

This section is the canonical answer to "what can a player do in one MVP round?"

### Procurement Manager

- `order_part(part_id, quantity)`

### Production Manager

- `release_product(product_id, quantity)`
- `allocate_capacity(workstation_id, product_id, capacity_units)`

### Sales Manager

- `set_price(product_id, unit_price)`
- `set_offer_quantity(customer_id, product_id, quantity)`

### Finance Controller

- `set_procurement_budget(amount)`
- `set_production_budget(amount)`
- `set_sales_target(amount)`
- `set_debt_ceiling(amount)`

All roles may also submit:

- `comment(text)`

## Metrics Tracked In MVP

The plant should expose shared metrics every round.

- cash
- short-term debt
- revenue
- procurement spend
- production volume by product
- shop floor inventory by product and workstation stage
- units sold by product
- backlog by customer and product
- parts inventory by part
- finished goods inventory by product
- workstation utilization by workstation
- lost sales by product
- customer sentiment by customer
- holding cost
- debt cost
- round profit
- cumulative profit

Role dashboards may emphasize subsets of those metrics, but the source data stays shared.

## Win, Loss, And Score Model

The MVP should use a shared plant score rather than separate role-specific victory conditions.

### Shared plant score

At the end of a match:

- primary score: cumulative profit
- tie-breaker 1: total fulfilled demand
- tie-breaker 2: lower ending debt
- tie-breaker 3: lower ending inventory value

### Failure conditions

The plant immediately loses if:

- debt exceeds the active debt ceiling and cannot be corrected by trimming actions
- a mandatory round cannot be resolved legally

The first playable version uses a fixed-length match of `52` rounds, with each round representing one week.

## Rules And Logic By Role

### Procurement Manager Logic

- can buy any part type in any non-negative integer quantity
- cannot force inventory negative
- is constrained by current debt ceiling and next-round budget pressure

### Production Manager Logic

- can only produce defined products
- must respect available part inventory
- must manage partially completed units already on the shop floor
- must respect workstation capacity at both workstations
- cannot create finished products directly without consuming required parts

### Sales Manager Logic

- controls demand through market-wide product pricing and customer-specific offer quantity
- cannot ship more than available inventory
- works against price sensitivity, customer sentiment, and customer-specific demand
- must manage backlog quality because delayed orders damage future sales
- cannot backdate sales into the current round after seeing production results

### Finance Controller Logic

- does not directly cancel same-round actions
- shapes the next round through budgets and targets
- can tighten or loosen debt tolerance and spending guidance

### Plant Logic

- resolves all player actions deterministically
- applies legality checks in the same order every round
- allows temporary debt but never negative inventory
- never allows capacity overuse
- logs both outcomes and player rationale

## What Is Intentionally Deferred

The MVP does not yet include:

- networked multiplayer
- long-term era progression
- multiple scenarios or plants
- supplier-specific lead times
- machine failures and stochastic disruptions
- separate hidden personal victory conditions
- maintenance, engineering, HR, or distribution roles
- quality defects, scrap, or rework
- save/load persistence
- a fully fleshed out balancing model

## Acceptance Criteria

Issue `#1` is complete when:

- a contributor can describe the exact phases of one round
- a contributor can list every legal action for each MVP role
- a contributor can explain how two products, two-part BOMs, and two workstations interact
- a contributor can explain one-round procurement lead time, backlog aging, and customer sentiment effects
- a contributor knows the hard constraints: finite capacity, no negative inventory, debt allowed within limits
- a future implementation can turn this document into deterministic code without first inventing new rules

## Open Questions

These questions are narrow enough to answer later without blocking implementation, but they are worth confirming before balancing the simulation:

1. How quickly should passive customer sentiment recovery occur relative to the stronger recovery from reliable service?
2. Should customer offers be locked to one product each week, or can the Sales Manager set offers for every customer-product pair every round?
