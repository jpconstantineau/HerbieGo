# Simulation Invariants

This document is the canonical reference for physical, business, temporal, and role-boundary invariants in the HerbieGo simulation.

Issue `#142` asked for a stable document contributors can use when they extend the engine, add reports, or debug surprising outcomes. These rules describe what the simulation should never violate, which rules are currently enforced centrally by the plant system, and where future work is still needed.

## Purpose

The invariants in this document serve four jobs:

- protect the simulation from physically impossible state transitions
- protect the business model from internally inconsistent cash, debt, and revenue updates
- keep round timing deterministic
- clarify which roles may request an action versus which rules the plant system must enforce centrally

## Physical Flow And Mass-Balance Invariants

These invariants protect the plant from impossible material flow.

### PF-1. Inventory cannot become negative

- parts inventory cannot go below zero
- work-in-progress cannot go below zero
- finished goods inventory cannot go below zero

Current status:

- enforced for parts by trimming production releases to buildable BOM quantities
- enforced for WIP and finished goods by trimming capacity advances and shipments to available quantities

### PF-2. Production releases must consume the required upstream parts

- a unit may not be released into production unless the required BOM inputs exist
- releasing work must reduce parts inventory by the consumed amount

Current status:

- enforced when the scenario provides BOM data
- unknown products are rejected

### PF-3. Route transitions must follow a legal production route

- WIP can only advance from its current workstation to the scenario-defined next workstation
- finished goods can only be created from a route step marked as finished

Current status:

- enforced through the scenario route hook
- invalid or missing continuations generate a rule-adjustment event and leave work in place

### PF-4. Work advanced in a round cannot exceed effective capacity

- workstation usage cannot exceed `CapacityPerRound`
- work advanced at a workstation cannot exceed the WIP waiting at that workstation

Current status:

- enforced by trimming requested capacity to remaining workstation capacity and available WIP

### PF-5. Finished goods cannot be shipped unless they physically exist

- shipments draw only from finished inventory
- backlog may remain after shipment if inventory is insufficient

Current status:

- enforced by trimming each shipment to current finished-goods on-hand

### PF-6. Ordered material cannot arrive before lead time completes

- purchase orders enter `InTransitSupply`
- parts become usable only when their arrival round is reached

Current status:

- enforced with one-round lead time in the current MVP scenario model

### PF-7. Units must stay in a visible state bucket

Every unit should always be representable as one of:

- parts inventory
- WIP at a named workstation
- finished goods
- in-transit supply
- backlog
- lost sales or expired demand through explicit events

Current status:

- mostly enforced by resolver transitions and event emission
- not yet fully audited through explicit reconciliation metrics

## Financial And Business Invariants

These invariants protect the bookkeeping side of the simulation.

### FB-1. Cash, debt, and spending updates must remain internally consistent

- procurement spend reduces cash and may create debt
- production spend reduces cash and may create debt
- shipment revenue increases cash and automatically pays down debt first
- round-end carrying costs reduce cash and may create debt

Current status:

- enforced centrally through `applyCashDelta`

### FB-2. Debt-constrained spending cannot exceed the active debt ceiling

- procurement and production must be trimmed or rejected when the plant cannot legally fund them

Current status:

- enforced through available-spend calculations against `Plant.DebtCeiling`
- finance-set debt ceiling targets now flow into the next-round plant state

### FB-3. Revenue cannot be recognized without shipment

- only shipped finished goods create revenue
- price offers by themselves do not create revenue

Current status:

- enforced

### FB-4. Inventory value must reconcile with on-hand state

- parts, WIP, and finished goods each contribute to inventory value
- round metrics should match the state snapshot at the end of resolution

Current status:

- enforced for round-end metric projection
- not yet enforced through a separate reconciliation check

### FB-5. Backlog expiration and lost sales must be explicit

- expired demand must leave backlog
- lost sales must be counted
- affected customer sentiment must move in the same resolution step

Current status:

- enforced
- backlog expiry is now driven by the scenario-configured expiry window instead of a hardcoded constant

### FB-6. Finance targets are directives, not retroactive overrides

- procurement and production budgets act as soft targets with a `110%` hard cap
- finance targets shape future rounds instead of rewriting already-resolved current-round actions

Current status:

- budget hard caps are enforced
- debt ceiling targets now become part of next-round state
- revenue-target and cash-floor handling remain advisory rather than enforced rules

## Temporal Invariants

These invariants keep the turn order deterministic.

### TR-1. Current-round resolution order is stable

The MVP resolver follows this sequence:

1. activate current-round targets
2. resolve procurement
3. receive due supply
4. resolve production
5. resolve sales
6. apply scenario/world demand updates
7. finalize finance targets and round-end metrics

Current status:

- enforced by the resolver flow

### TR-2. Finance actions cannot retroactively affect the current round

- finance submissions set next-round targets
- they do not veto current-round procurement, production, or sales after seeing their outcomes

Current status:

- enforced

### TR-3. Demand and backlog aging move forward round by round

- new demand enters backlog with the current round as origin
- backlog age increases once during round finalization
- expired backlog produces explicit events

Current status:

- enforced

### TR-4. Purchase orders cannot skip the transit state

- orders placed in round `N` arrive in a later round, not immediately

Current status:

- enforced

## Role-Responsibility Boundaries

Roles may request actions inside their decision space. The plant system is still responsible for rejecting or trimming illegal actions.

### Procurement Manager

Relevant invariants:

- cannot request negative quantities
- cannot create immediate inventory from an order
- cannot spend beyond legal budget and debt constraints once the plant applies enforcement

Plant-system responsibility:

- trimming to budget and spend capacity
- preserving lead-time behavior

### Production Manager

Relevant invariants:

- cannot release more work than available BOM parts support
- cannot advance more work than available WIP and workstation capacity allow
- cannot skip route stages

Plant-system responsibility:

- route enforcement
- capacity enforcement
- material consumption

### Sales Manager

Relevant invariants:

- cannot recognize revenue without shipment
- cannot ship more finished goods than exist
- cannot directly mutate backlog or sentiment outside plant rules

Plant-system responsibility:

- demand realization
- shipment trimming
- backlog aging and expiration

### Finance Controller

Relevant invariants:

- cannot set negative budgets or targets
- cannot retroactively alter already-resolved current-round actions
- can influence next-round procurement, production, and debt policy

Plant-system responsibility:

- activating next-round targets at the right time
- enforcing budget hard caps and debt limits

### Plant/System

The plant system is the invariant owner of last resort. If a role requests an impossible action, the system must reject it, trim it, or surface the inconsistency through events and tests rather than allowing impossible state.

### Future Roles

These responsibility areas should keep the same pattern when introduced:

- Quality: defects, scrap, rework, release holds
- Maintenance: downtime, repair completion, capacity restoration
- Logistics/Warehouse: storage state, transfer legality, inbound/outbound timing
- Plant Manager: cross-functional policy, priority setting, escalation, exception handling

## Audit Summary

Issue `#142` also required an audit of the current implementation. The companion document [Invariant Audit](invariant-audit.md) records which rules are already enforced, which are partial, and which need follow-up work.

Current follow-up issues:

- `#143` scenario-configured backlog expiry
- `#144` next-round debt ceiling activation
- `#145` pricing timing alignment
- `#146` Sales action vocabulary alignment
- `#147` soft-budget-miss reporting
