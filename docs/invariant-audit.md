# Invariant Audit

This audit accompanies [Simulation Invariants](simulation-invariants.md) and records the current state of enforcement in the MVP engine.

## Scope Reviewed

The audit focused on:

- [internal/engine/resolver.go](/C:/GIT/HerbieGo/internal/engine/resolver.go)
- [internal/scenario/scenario.go](/C:/GIT/HerbieGo/internal/scenario/scenario.go)
- [internal/scenario/starter.go](/C:/GIT/HerbieGo/internal/scenario/starter.go)
- [docs/mvp-game-design.md](/C:/GIT/HerbieGo/docs/mvp-game-design.md)
- [docs/domain-model.md](/C:/GIT/HerbieGo/docs/domain-model.md)

## Findings

### Rules already enforced

- Negative inventory is prevented by trimming releases, capacity advances, and shipments to available state.
- Procurement and production spending are trimmed to budget and debt-constrained spend capacity.
- Revenue is recognized only when finished goods ship.
- Purchase orders move through `InTransitSupply` before parts inventory.
- Production route transitions are controlled by the scenario route hook.
- Backlog expiration generates lost-sales and sentiment events.
- Finance targets are recorded for the next round instead of retroactively changing the current one.

### Rules only partially enforced

- Inventory value is projected into metrics, but there is no standalone reconciliation assertion to prove mass-balance across every state bucket.
- Budget overspend inside the permitted `110%` window is allowed, but soft-target misses are not separately surfaced in events or reports.
- Finance targets include revenue and cash-floor guidance, but those targets are not used as enforceable plant rules today.
- The event stream records many automatic adjustments, but there is no dedicated anomaly report that summarizes invariant-related trims, rejections, or reconciliation mismatches for a round.

### Confirmed gaps and discrepancies

- The engine had a hardcoded backlog-expiry threshold even though scenarios already define `BacklogExpiryRounds`.
- Finance-set debt ceiling targets were stored in `ActiveTargets` but were not flowing into the next-round `Plant.DebtCeiling`.
- The MVP design doc says sales pricing changes affect customer demand in the subsequent round, while the current scenario demand model uses prices from the round being resolved.
- The MVP design doc still lists `set_offer_quantity(customer_id, product_id, quantity)` for Sales, but the current domain model exposes price-setting only.
- The MVP design doc says soft budget misses should be logged, but the resolver currently trims only hard violations.

## Resolved in the issue `#142` implementation

- backlog expiry now honors the scenario-configured expiry window
- finance-set debt ceiling targets now flow into next-round plant state
- tests now cover both behaviors

## Follow-up work

The follow-up issues filed from this audit should cover:

- `#143` scenario-configured backlog expiry
- `#144` next-round debt ceiling activation
- `#145` pricing-timing semantics versus current demand resolution
- `#146` Sales action vocabulary mismatch between docs and code
- `#147` missing soft-budget-miss reporting
- future reconciliation checks or anomaly summaries beyond the scope of this change set

## Clarifications that remain open

No unresolved clarification blocked the current implementation work.

There is one design choice worth keeping visible for future discussion:

- if Finance lowers the next-round debt ceiling below the debt carried out of the current round, the project still needs an explicit product decision on whether to auto-clamp, reject the finance target, force immediate loss, or allow a temporarily out-of-policy opening state with a warning
