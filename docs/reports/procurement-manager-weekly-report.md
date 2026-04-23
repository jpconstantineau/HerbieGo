# Procurement Manager Weekly Report

This document defines the weekly report the Procurement Manager should receive in HerbieGo.

The report should help the player answer one core question before acting:

`Will the plant have the right parts, at the right time, without creating avoidable cash pressure or inventory risk?`

This is an MVP-facing document. It separates:

- what the Procurement Manager should see in the current MVP
- what can be added later as supplier, quality, and logistics mechanics expand

## Role Scope

- Role: `Procurement Manager`
- Status: `MVP role`
- Primary decision horizon: next round and the 1 to 3 rounds after that
- Main tradeoff: shortage protection versus cash discipline and overbuy risk

## What This Report Should Support

The weekly procurement report should help the player decide whether to:

- place normal replenishment orders
- increase buys to protect against expected shortages
- defer or reduce orders to protect cash
- pay more for reliability or speed
- accept higher inventory in exchange for lower supply risk

The report is not just an inventory snapshot. It should make supply risk visible early enough for the Procurement Manager to act before Production is starved.

## Recommended Report Structure

Present the report in this order.

## 1. Executive Summary

Start with a short summary panel that highlights the most urgent procurement risks.

Recommended summary items:

- parts at immediate shortage risk
- parts with less than one supplier-lead-time of cover
- total spend already committed in transit
- next-round arrivals already scheduled
- whether current procurement behavior is likely to breach finance targets or create debt pressure

Why first:

- the player should understand the plant's supply risk in seconds before reading line-item detail

## 2. Raw Material Inventory Status

This is the most important section in the report.

Use one row per part:

| Part | On Hand | Expected Burn Rate | Days Or Rounds Of Cover | Shortage Risk | Notes |
| --- | --- | --- | --- | --- | --- |
| `example_part` | Current inventory. | Expected near-term usage. | Coverage estimate. | `Low`, `Medium`, or `High`. | Reason for concern. |

Key fields:

- on-hand stock
- expected burn rate from visible backlog, prior rounds, scenario demand, or published plant planning assumptions
- days or rounds of cover
- stock-out events or near-stock-out warnings
- impact of a shortage on the plant, such as idle production time or backlog growth

Decision value:

- tells Procurement where a normal replenishment order is enough and where a panic buy may be justified

## 3. Open Purchase Orders And Receipts

This section shows what is already on the way and when it should become usable.

Use one row per open order or aggregated row per part:

| Part | In Transit Quantity | Expected Arrival Round | Spend Committed | Late Risk | Notes |
| --- | --- | --- | --- | --- | --- |
| `example_part` | Ordered but not received. | Planned receipt timing. | Cash already committed. | `Low`, `Medium`, or `High`. | Delay or dependency note. |

Key fields:

- in-transit quantity
- expected arrival round
- total cash already committed to open orders
- any receipt likely to arrive too late for projected demand

MVP note:

- the current MVP uses a shared one-round lead time for suppliers
- late deliveries and supplier-specific lead-time variation are future extensions, but the report structure should leave room for them

Decision value:

- prevents duplicate buying
- shows whether next round is already covered by orders in flight

## 4. Price And Spend Conditions

This section helps Procurement weigh supply protection against finance pressure.

Recommended fields:

- last purchase price by part
- standard or target purchase price
- purchase price variance
- projected spend if the player replenishes to a target coverage level
- warning when a purchase would materially increase debt or consume most of the active budget

Use this table:

| Part | Last Price | Target Price | Variance | Reorder Spend Estimate | Spend Pressure |
| --- | --- | --- | --- | --- | --- |
| `example_part` | Last paid cost. | Budgeted or expected cost. | Positive or negative variance. | Cost to restore target cover. | `Low`, `Medium`, or `High`. |

Future expansion note:

- market price trends and supplier-specific pricing can be added later without changing the basic layout

Decision value:

- helps Procurement justify normal buying, delayed buying, or an intentional bulk buy

## 5. Supplier Health And Reliability

This section becomes more important as the simulation adds supplier differentiation.

Recommended fields:

- lead-time reliability
- quality rejection rate
- missed-delivery count
- volume discount status
- reasons a supplier may no longer be a safe default choice

Use this table:

| Supplier | Primary Part | Reliability Signal | Quality Signal | Cost Signal | Recommended Watch Item |
| --- | --- | --- | --- | --- | --- |
| `example_supplier` | Main supplied item. | Delivery performance. | Rejection or defect concern. | Cost trend. | What to monitor next. |

MVP note:

- the current MVP does not yet model supplier-specific reliability, quality rejections, or discount ladders in detail
- this section can begin as placeholder guidance and become richer later

Decision value:

- supports vendor swapping, risk balancing, and future quality-aware buying decisions

## 6. Procurement Decision Prompts

End the report with 3 to 5 plain-language prompts that help the player turn data into action.

Recommended prompts:

- Which part is most likely to stop production next round?
- Which order protects flow at the lowest cash cost?
- Is there any part where we are buying too early and tying up cash?
- Are we relying on an in-transit order that leaves no buffer if anything slips?
- Is this a week for protection buying, disciplined delay, or selective expediting?

Why this matters:

- HerbieGo asks players to make a decision, not just read a dashboard

## Visibility Guidance

This report should combine plant-wide and role-specific information.

Plant-wide inputs:

- current part inventory
- in-transit supply
- active finance targets
- recent production consumption patterns and other already revealed plant-state signals

Role-focused interpretation:

- shortage risk assessment
- reorder priorities
- supplier-risk commentary
- procurement-specific spend tradeoffs

The report must not reveal hidden current-turn actions submitted by other players before round resolution.

## MVP Versus Future Expansion

| Report Element | MVP Now | Future Expansion |
| --- | --- | --- |
| On-hand inventory | Yes | Yes |
| In-transit supply by arrival round | Yes | Yes |
| Expected part burn rate | Yes | Yes |
| Procurement budget pressure | Yes | Yes |
| Supplier-specific lead-time variance | No | Yes |
| Late delivery exceptions | No | Yes |
| Supplier quality rejection rate | No | Yes |
| Volume discount progress | No | Yes |
| Alternate supplier recommendations | Limited | Yes |

## Example Decisions This Report Should Enable

- `Panic buy`: spend more than usual to prevent a production stoppage next round
- `Disciplined hold`: delay a purchase because inventory and in-transit coverage are already sufficient
- `Bulk play`: buy ahead intentionally when the cost case is strong and cash allows it
- `Vendor swap`: move away from a risky supplier once supplier quality and reliability become modeled

## Design Guardrails

When contributors implement or refine this report, they should:

- prioritize shortage visibility before cost detail
- connect each section to a realistic procurement decision
- avoid turning the report into a generic ERP dump
- keep the MVP version honest about which supplier mechanics do and do not exist yet
