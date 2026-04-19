# ADR 0002: Simultaneous Action Collection And Resolution Flow

- Status: Accepted
- Date: 2026-04-19
- Deciders: HerbieGo maintainers
- Related roadmap issue: `#4`

## Context

Issue `#4` asks for a concrete specification of how HerbieGo collects hidden simultaneous actions, validates them, resolves them, and reveals outcomes.

The project already defines:

- [MVP Game Design](../mvp-game-design.md)
- [Canonical Domain Model](../domain-model.md)
- [ADR 0001: Initial Architecture And Package Boundaries](0001-initial-architecture.md)

Those documents establish the MVP round phases, shared data vocabulary, and package ownership. What remains is the exact turn contract contributors can implement without guessing about timing, legality checks, or event ordering.

This ADR defines the authoritative round lifecycle for simultaneous hidden turns in the MVP.

## Decision Summary

HerbieGo resolves one round through six deterministic stages:

1. build and broadcast round views from the current canonical state
2. collect exactly one hidden `ActionSubmission` per role for the active round
3. validate submission shape and role ownership before any game rules run
4. freeze the round input set and resolve actions in a fixed plant-owned order
5. append public outcome events and round metadata in deterministic order as outcomes occur
6. reveal only the resolved round record after the full round is committed

Key decisions:

- Current-turn actions remain hidden from all player roles until round resolution is complete.
- Validation is split into submission validation and resolution validation.
- Missing or illegal intents never stop the round if the plant can deterministically trim them into a legal no-op or partial outcome.
- Resolution order is fixed as start-of-round activation of previously scheduled targets, then procurement, supply receipt, production, sales, and end-of-round aging plus next-round target scheduling.
- Event ordering and round metadata ordering are stable and append-only so UI, persistence, replay, and tests can agree on the same round history.

## Round Lifecycle

### Stage 1: Build Round Snapshot

At the start of round `R`, the application builds one `RoundView` per player role from the same canonical `MatchState`.

Each view must reflect:

- plant state as of the end of round `R-1`
- currently active budgets and targets whose `EffectiveRound == R`
- recent events and commentary already revealed from prior rounds
- no current-round actions from any role

This stage is pure projection work. No rules are executed and no hidden state is created beyond the round-collection session metadata.

### Stage 2: Hidden Action Collection

The application opens a collection window for round `R` and requests one `ActionSubmission` from each assigned role:

- `procurement_manager`
- `production_manager`
- `sales_manager`
- `finance_controller`

Collection rules:

- each submission must include `MatchID`, `Round`, `RoleID`, one matching action payload, and optional public commentary
- each role may have at most one active submission for the round
- resubmission before lock is allowed by replacing that role's prior submission
- players never receive another role's current-round submission contents
- the plant does not begin outcome resolution until collection is closed and the input set is frozen

Collection behavior differs by player type:

- human players block round collection until they submit, with the option to resubmit a previously provided response before round freeze
- AI-controlled roles should use a timeout
- if an AI role times out, the previous accepted action for that role is reused by default
- if an AI role returns an unparsable response before timeout, the game should log the parsing error and retry after telling the agent what parsing problem occurred

Partial real-time resolution remains out of scope for MVP.

### Stage 3: Submission Validation

Each submission is validated immediately on receipt for structural correctness, without revealing it to other players and without mutating plant state.

Submission validation checks:

- `MatchID` matches the active match
- `Round` equals the current round
- `RoleID` belongs to an assigned role in the match
- only the payload matching `RoleID` is populated
- numeric inputs are integers and non-negative where the action contract requires non-negative quantities or prices
- referenced product, part, customer, and workstation identifiers exist in scenario/domain data

Outcomes:

- structurally valid submissions are stored as accepted hidden inputs
- structurally invalid submissions are rejected back to the submitting player for correction while collection remains open

Recommended round-record behavior:

- store accepted submissions as round metadata once they enter the frozen input set
- do not emit public rejection events for transient pre-lock validation failures

This keeps the public event log focused on resolved plant outcomes rather than every draft edit a player made during collection.

### Stage 4: Input Freeze And Resolution Validation

Once collection closes, the application freezes one final accepted submission per role into the round input set. From this point on, actions cannot be edited for round `R`.

The engine then performs resolution validation against the frozen set and current state. This validation answers "what is legal to execute now?" rather than "is the payload well formed?"

Resolution validation may:

- accept an intent as-is
- trim quantities or spending down to the maximum legal amount
- convert an intent into a no-op if nothing legal can be executed

The engine must prefer deterministic trimming over hard failure whenever the rules text already says the plant "reduces", "rejects", or "advances the maximum legal quantity".

If a round still cannot be resolved legally after deterministic trimming, the match enters the MVP failure condition described in the game design.

## Deterministic Resolution Order

The engine resolves the frozen round input set in this exact order.

### 1. Start-Of-Round Target Activation

Budgets and targets with `EffectiveRound == R` are already active when round `R` begins and must be treated as immutable inputs throughout the rest of the round.

If this activation is represented in the round record, emit `budget_activated` before any player-action outcome events.

### 2. Procurement Resolution

Resolve `ProcurementAction` first.

Rules:

- calculate requested purchase orders from the frozen procurement submission
- enforce procurement budget soft limit and `110%` hard cap using the active targets for round `R`
- enforce debt ceiling using current cash/debt plus already accepted procurement spend in this phase
- trim or reject order lines deterministically when the budget cap or debt ceiling would be breached
- accepted lines create `SupplyLot` records with arrival at round `R+1`
- cash and debt effects apply immediately in this phase

Why first:

- production in the same round cannot consume newly ordered parts because purchased parts arrive later
- procurement spending must influence round cash/debt before later phases

### 3. Supply Receipt Update

Before production uses parts, receive any prior `SupplyLot` with `ArrivalRound == R`.

Rules:

- move arrived lots from `InTransitSupply` into `PartsInventory`
- preserve stable ordering by existing lot order, then part identifier
- received parts become available for production in round `R`

This makes one-round lead time precise: orders placed in round `R-1` are usable in production during round `R`.

### 4. Production Resolution

Resolve `ProductionAction` against:

- parts inventory after receipts
- carried `WIPInventory`
- workstation capacities for round `R`
- production budget cap active for round `R`
- current cash/debt state after procurement

Rules:

- compute the maximum legal advances from parts, route stage, and capacity constraints
- accept both `release_product` and `allocate_capacity` intents even when they do not align perfectly
- when release intent exceeds available downstream advancement, the excess accepted work remains in `WIPInventory`
- inventory limits such as safety stock or maximum holding capacity are post-MVP concerns and do not block this behavior
- never create negative parts inventory, negative WIP, negative finished goods, or capacity overuse
- apply production spending and any resulting debt changes during this phase

Production never sees same-round sales demand before resolving. Sales adapts to whatever finished goods exist after this step.

### 5. Sales Resolution

Resolve `SalesAction` after finished goods inventory is known for the round.

Rules:

- pricing decisions from round `R` influence new demand created for round `R+1`, not retroactive same-round demand
- current round shipments use backlog and available finished goods after production
- shipments are allocated deterministically from oldest backlog to newest backlog
- if finished goods are insufficient, backlog remains in age order with unshipped remainder
- revenue, cash, and shipment metrics apply during this phase

This ordering preserves the MVP rule that sales cannot backdate same-round promises after seeing production outcomes, while still letting the plant ship available goods against existing backlog.

### 6. End-Of-Round Aging, Metrics, And Next-Round Targets

After operational phases complete, the engine performs end-of-round updates:

- age backlog entries
- expire backlog older than the MVP limit
- apply customer sentiment changes caused by expiry or reliable service
- apply holding cost and debt carrying cost
- calculate final `PlantMetrics` for round `R`
- persist finance targets submitted in round `R` as the `BudgetTargets` whose `EffectiveRound == R+1`

These next-round finance targets do not affect procurement or production decisions already resolved in round `R`.

## Conflict Handling Rules

Simultaneous hidden turns create conflicting intents by design. The plant resolves conflicts through shared-state priority, not by player seniority.

### General Rule

When two or more actions compete for the same constrained resource, the engine resolves them according to the phase order above and trims later effects against the state produced by earlier effects.

### Specific MVP Conflicts

Procurement versus debt ceiling:

- procurement orders are processed in deterministic line order
- once the debt ceiling or procurement hard cap would be exceeded, remaining excess quantity is trimmed or rejected

Production versus parts and capacity:

- requested releases and capacity allocations are both accepted as valid intent
- the engine advances as much work as legal through the route in round `R`
- any accepted release that cannot advance far enough this round remains as increased `WIPInventory`

Sales versus limited finished goods:

- existing backlog has priority according to oldest-first shipment order
- if finished goods run out, later backlog remains open

Finance versus same-round operating actions:

- finance does not veto same-round procurement or production with a fresh round-`R` submission
- finance only changes targets for round `R+1`

### Deterministic Tie-Breaking

When a rule needs stable ordering among peer items of the same type, use canonical identifier order unless a stronger domain rule already exists.

Recommended MVP ordering rules:

- purchase order intents: by the order they appear in the accepted submission
- supply receipt lots: by `ArrivalRound`, then existing lot order
- backlog shipment: by oldest `OriginRound`, then `CustomerID`, then `ProductID`
- event emission within a phase: by the order outcomes are applied to canonical state

The engine must not rely on Go map iteration order for any visible outcome.

## Event Emission Rules

The round record is append-only and must reflect actual resolved outcomes, not speculative intent.

### Required Principles

- public events are emitted only from plant-owned resolution steps
- event order matches state mutation order
- all public events for round `R` are revealed together after the round commits
- accepted action envelopes belong to round metadata, not the public event stream
- each materially visible trim, rejection, or adjustment should produce an explicit `rule_adjustment` event
- final round metrics should be captured with `metric_snapshot`

### Typical Event Sequence

An MVP round will usually emit public events in an order like:

1. `budget_activated` if round `R` starts with newly effective targets
2. `purchase_order_placed` for each accepted procurement line
3. `cash_changed` for procurement spend effects
4. `supply_arrived` for receipts entering parts inventory
5. `production_released`, `work_advanced`, and `finished_goods_produced` during production
6. `cash_changed` for production operating cost if applicable
7. `demand_realized`, `shipment_completed`, and `backlog_created` during sales processing
8. `cash_changed` for realized revenue and finance adjustments
9. `backlog_expired` and `customer_sentiment_moved` during end-of-round aging
10. `metric_snapshot` after all state mutation is complete

Not every round will emit every event type.

### Commentary Reveal

Player commentary and accepted action intent are stored with the round input set as round metadata, then revealed only with the completed round record.

Reveal rule:

- commentary for round `R` becomes visible only after round `R` has fully resolved and committed

This preserves simultaneous hidden play while keeping the social explanation layer intact.

## Failure And Recovery Rules

The MVP should keep round processing deterministic and robust.

- A missing human submission blocks round freeze by default.
- An AI role that times out reuses its previous accepted action by default.
- An AI role that returns an unparsable response should be retried with the parsing error surfaced to that agent.
- A structurally invalid submission is rejected before freeze and does not enter the round record until corrected.
- A structurally valid but economically illegal submission is frozen, then trimmed or converted into a no-op during resolution.
- Only irrecoverable rule contradictions after deterministic trimming should trigger the match failure condition.

This lets UI and AI contributors implement collection and engine code without debating whether malformed input, over-budget requests, and impossible production plans are the same class of problem. They are not.

## Implementation Guidance

Contributors implementing the engine and UI should treat the following as the minimum contract:

- `internal/app` owns the collection window, submission replacement, AI timeout and retry orchestration, and round freeze orchestration
- `internal/engine` owns resolution validation, deterministic phase execution, trimming, and event emission
- `internal/projection` reveals only prior-round history plus the viewer's current round input draft, never another role's current-round action
- persistence should store the frozen action set, accepted-action metadata, emitted events, commentary, and post-round metrics as one coherent round record

Suggested engine shape:

```go
type RoundResolver interface {
    ResolveRound(state MatchState, actions []ActionSubmission) RoundRecord
}
```

Practical phase helpers:

- `validateSubmission`
- `freezeRoundInputs`
- `resolveProcurement`
- `receiveSupply`
- `resolveProduction`
- `resolveSales`
- `finalizeRound`

## Consequences

Positive:

- UI and engine contributors can agree on exactly when actions are hidden, frozen, and revealed
- legality trimming behavior is explicit instead of left to adapter discretion
- event history becomes deterministic enough for replay, tests, and prompt context
- finance timing is unambiguous for both players and implementers

Tradeoffs:

- collection blocks on all roles in MVP, which may slow future asynchronous play modes
- deterministic trimming requires more explicit engine code than simply rejecting whole actions
- some event sequences will feel verbose, but that verbosity is useful for debugging and explainability

## Status Review Trigger

Revisit this ADR when any of the following become true:

- the project adds turn timers, auto-submit defaults, or partial missing-player resolution
- hidden/private commentary or per-role secret information classes are introduced
- same-round finance vetoes or interrupt-style reactions are added
- post-MVP rules require interleaving market events inside player-action resolution

Until then, contributors should treat this ADR as the canonical answer for simultaneous hidden turn collection and resolution semantics.
