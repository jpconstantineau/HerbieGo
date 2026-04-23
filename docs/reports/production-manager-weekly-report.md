# Production Manager Weekly Report

This document defines the weekly report the Production Manager should receive in HerbieGo.

The report should help the player answer one core question before acting:

`How do we convert the parts and capacity we actually have into the most useful finished output this round?`

This is an MVP-facing document. It keeps the report grounded in the current simulation while leaving room for richer production mechanics later.

## Role Scope

- Role: `Production Manager`
- Status: `MVP role`
- Primary decision horizon: this round and the next 1 to 2 rounds
- Main tradeoff: output and service protection versus capacity limits, operating spend, and work-in-progress congestion

## What This Report Should Support

The weekly production report should help the player decide whether to:

- prioritize one product over another
- change release quantities based on part availability
- protect a bottleneck workstation
- accept more work-in-progress now or keep flow tighter
- justify overtime or other capacity spend when service risk is rising

The report should focus on flow, throughput, and constraint awareness rather than presenting a generic manufacturing scorecard.

## Recommended Report Structure

Present the report in this order.

## 1. Executive Summary

Start with a short summary panel that highlights the biggest production risks for the round.

Recommended summary items:

- products most at risk of missing output targets
- the workstation currently acting as the main bottleneck
- whether part availability or capacity is the tighter constraint
- work-in-progress that is accumulating without enough downstream capacity
- whether planned production pressure is likely to exceed active finance targets

Why first:

- the Production Manager should know immediately where the plant is constrained before reading deeper detail

## 2. Throughput And Output Status

This is the first detailed section because it shows whether Production is turning available capacity into finished goods.

Use one row per product:

| Product | Target Output | Expected Feasible Output | Finished This Round | WIP Carried | Service Risk | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `example_product` | Desired output. | What the plant can likely finish legally. | Actual completed units. | Units still in progress. | `Low`, `Medium`, or `High`. | Main reason for risk. |

Key fields:

- target versus actual output
- expected feasible output given visible parts and capacity
- finished-goods completions
- work-in-progress carried into or out of the round
- backlog or service pressure tied to each product

Decision value:

- helps Production decide which product mix is realistic and which promises are no longer supportable without tradeoffs

## 3. Capacity And Bottleneck View

This section should show where the route is constrained.

Use one row per workstation:

| Workstation | Capacity Available | Capacity Planned Or Used | Utilization Signal | Queue Or WIP Pressure | Bottleneck Status |
| --- | --- | --- | --- | --- | --- |
| `example_workstation` | Total available capacity. | Planned or consumed capacity. | `Low`, `Medium`, or `High`. | Downstream or upstream pressure. | `Primary`, `Secondary`, or `Not a bottleneck`. |

Key fields:

- available capacity by workstation
- capacity already consumed or expected to be consumed
- utilization level
- where work-in-progress is piling up
- which workstation limits plant throughput this round

MVP note:

- the current MVP uses fixed per-round workstation capacity rather than detailed scheduling
- line balancing, setup penalties, and explicit sequencing are future extensions

Decision value:

- supports product-priority changes, capacity allocation choices, and bottleneck protection

## 4. Material Readiness And Starvation Risk

Production decisions are only legal if the required parts actually exist.

Use one row per product or critical part dependency:

| Product Or Part | On Hand | Near-Term Need | Shortage Risk | Likely Production Impact | Notes |
| --- | --- | --- | --- | --- | --- |
| `example_dependency` | Visible inventory. | Expected requirement. | `Low`, `Medium`, or `High`. | What production loses if short. | Main cause or mitigation. |

Key fields:

- parts on hand for each product path
- shortages that will block release quantities
- risk of idle capacity caused by part shortages
- visible supply receipts already known to the plant

Decision value:

- prevents impossible release plans and makes procurement-driven risk visible before Production acts

## 5. Productivity And Spend Pressure

This section connects output decisions to finance pressure without turning Production into the Finance Controller.

Recommended fields:

- overtime or extra capacity spend already used
- expected spend impact of the planned production choice
- idle-capacity signal
- flow losses caused by shortages or downstream blockage

Use this table:

| Signal | Current Value | Why It Matters | Warning Trigger | Likely Response |
| --- | --- | --- | --- | --- |
| `example_signal` | Current status. | Why Production watches it. | What should worry the player. | Reasonable action. |

MVP note:

- detailed labor efficiency, shift scheduling, and downtime accounting are limited in the current MVP
- the report should still show whether capacity is being used productively or wasted

Decision value:

- helps Production judge when extra spend is helping flow versus when it is only increasing cost

## 6. Quality And Rework Pressure

Production should understand when output problems are really quality or process problems.

Recommended fields:

- first-pass yield or equivalent quality signal
- scrap or rework signal
- repeated trouble spots by product or workstation
- whether poor output quality is reducing usable finished goods

Future expansion note:

- deeper defect analysis and a dedicated Quality Manager are post-MVP concerns
- the report layout should still leave room for quality-aware production decisions

Decision value:

- helps Production avoid confusing raw activity with usable output

## 7. Production Decision Prompts

End the report with 3 to 5 plain-language prompts that help the player turn the report into action.

Recommended prompts:

- Which product should get the bottleneck first this round?
- What is the highest-output legal plan with the parts we already have?
- Where is work-in-progress building up without enough downstream capacity?
- Is overtime protecting profitable flow or only masking a planning problem?
- Which shortfall should be escalated to Procurement, Sales, or Finance before it gets worse?

Why this matters:

- the report should guide a production decision, not just explain what happened last round

## Visibility Guidance

This report should combine plant-wide state with production-focused interpretation.

Plant-wide inputs:

- visible part inventory
- visible work-in-progress
- visible finished goods inventory
- workstation capacities
- active finance targets
- already revealed backlog and service signals

Role-focused interpretation:

- feasible output assessment
- bottleneck identification
- capacity-allocation priorities
- production-specific escalation needs

The report must not reveal hidden current-turn actions from Procurement, Sales, or Finance before round resolution.

## MVP Versus Future Expansion

| Report Element | MVP Now | Future Expansion |
| --- | --- | --- |
| Target versus actual output | Yes | Yes |
| WIP by route stage | Yes | Yes |
| Workstation capacity view | Yes | Yes |
| Part starvation risk | Yes | Yes |
| Overtime or production spend pressure | Yes | Yes |
| Detailed labor efficiency rate | Limited | Yes |
| Full OEE breakdown | Limited | Yes |
| Setup-time and sequencing loss | No | Yes |
| Maintenance-driven downtime analysis | No | Yes |
| Detailed scrap and rework root-cause analysis | Limited | Yes |

## Example Decisions This Report Should Enable

- `Shift the mix`: favor the product that best uses constrained capacity this round
- `Protect the bottleneck`: reduce low-value releases that would only create more upstream congestion
- `Use overtime selectively`: spend more to protect critical output or delivery reliability
- `Escalate a shortage`: push Procurement or Finance when production risk is driven by missing parts or budget pressure

## Design Guardrails

When contributors implement or refine this report, they should:

- show feasible output, not just desired output
- make the active bottleneck obvious
- connect work-in-progress to downstream consequences
- distinguish usable finished output from raw activity
- keep the MVP report honest about which shop-floor mechanics are not yet modeled
