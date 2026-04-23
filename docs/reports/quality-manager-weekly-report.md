# Quality Manager Weekly Report

This document defines the weekly report a future `Quality Manager` role would likely need in HerbieGo.

The report should help the player answer one core question before acting:

`Where is quality risk entering the plant, how much of it is escaping to the customer, and which intervention best protects the business without creating avoidable operational damage?`

## Future Role Status

The `Quality Manager` is not currently implemented in the MVP action roster. This document is forward-looking design guidance for future expansion, onboarding, and AI-role planning.

Where possible, it references signals that already have analogs in the MVP, but several sections assume richer future mechanics such as inspection policy, quarantine, supplier-quality tracking, and deeper defect attribution.

## Role Scope

- Role: `Quality Manager`
- Status: `Future Role`
- Primary decision horizon: current quality containment plus the next few rounds of customer and process impact
- Main tradeoff: customer protection and process control versus throughput, cost, and schedule pressure

## What This Report Should Support

The weekly quality report should help the player decide whether to:

- tighten or relax inspection intensity
- quarantine suspect output before it reaches customers
- escalate a supplier or process issue
- trigger root-cause investigation instead of repeated firefighting
- accept short-term throughput pain to avoid bigger downstream failure

The report should make quality risk visible early enough that the role is not reduced to reporting damage after it has already spread.

## Recommended Report Structure

Present the report in this order.

## 1. Executive Summary

Start with a short summary panel that highlights the biggest quality risks.

Recommended summary items:

- whether internal quality is stable, degrading, or in crisis
- whether customer-facing failures are rising
- which product, supplier, or process area is driving the most risk
- whether the plant is currently leaning too hard toward throughput over control
- whether immediate containment is warranted

Why first:

- the Quality Manager should know immediately whether the week calls for monitoring, containment, or escalation

## 2. Yield And Defect Analysis

This is the core of the report because it shows whether the plant is producing usable output rather than hidden waste.

Use one row per product or major process area:

| Product Or Area | First-Pass Yield | Scrap Or Rework Signal | Defect Trend | Risk Level | Notes |
| --- | --- | --- | --- | --- | --- |
| `example_area` | Current quality yield. | Main internal loss signal. | Improving, flat, or worsening. | `Low`, `Medium`, or `High`. | Main explanation. |

Key fields:

- first-pass yield
- scrap rate
- rework rate
- defect trend over recent rounds
- where defects are concentrated

Future expansion note:

- the MVP currently exposes some quality-like outcomes indirectly through production and returns signals, but not a dedicated defect-management role
- this section should become richer as explicit quality mechanics are added

Decision value:

- helps Quality decide when throughput problems are really process-quality problems

## 3. Customer Escape And External Failure

This section shows whether defects are reaching customers instead of being caught inside the plant.

Recommended fields:

- return or RMA signal
- customer complaint trend
- warranty or remediation cost signal
- repeat failure pattern by product or defect family

Use this table:

| External Signal | Current Status | Why It Matters | Warning Trigger | Likely Response |
| --- | --- | --- | --- | --- |
| `example_signal` | Current condition. | Customer and business impact. | What should worry Quality. | Contain, investigate, or escalate. |

Decision value:

- helps Quality prioritize customer protection when internal measures are failing to contain defects

## 4. Supplier And Incoming-Material Quality

Quality often needs to know whether defects begin before production even starts.

Use one row per supplier or part family:

| Supplier Or Part | Incoming Quality Signal | Rejection Trend | Operational Impact | Risk Level | Notes |
| --- | --- | --- | --- | --- | --- |
| `example_supplier` | Current incoming quality condition. | Improving, flat, or worsening. | What it disrupts. | `Low`, `Medium`, or `High`. | Main interpretation. |

Recommended fields:

- material rejection rate
- incoming defect or deviation pattern
- repeat supplier-quality issues
- effect on scrap, rework, or line stability

Decision value:

- supports supplier escalation, tighter inspection, and future coordination with Procurement

## 5. Containment, Compliance, And Audit Watchlist

Quality needs visibility into the risks that can justify slowing the plant down.

Recommended fields:

- suspect lots needing hold or quarantine
- audit or compliance status
- calibration or measurement-system risk
- open corrective actions and overdue items

Use this table:

| Watch Item | Current Status | Why It Matters | Escalation Trigger | Likely Action |
| --- | --- | --- | --- | --- |
| `example_item` | Current condition. | Business or compliance impact. | What makes it urgent. | Hold, inspect, or escalate. |

Decision value:

- helps Quality justify interventions that may temporarily reduce output but prevent larger failure

## 6. Quality Decision Prompts

End the report with 3 to 5 plain-language prompts that guide the next quality decision.

Recommended prompts:

- Is this a week for containment, correction, or watchful monitoring?
- Which defect source is hurting the business most right now?
- Are we overprotecting quality in a low-risk area, or underreacting in a high-risk one?
- Which issue belongs with Procurement, Production, or Sales instead of staying inside Quality?
- What short-term throughput loss is justified to avoid a much larger customer or compliance failure?

Why this matters:

- the report should drive a quality action, not only describe defect counts

## Visibility Guidance

This report would combine plant-wide signals with quality-specific interpretation.

Plant-wide or shared inputs:

- production-yield analogs
- shipment and return signals
- supplier or material issue signals
- backlog and customer-impact signals

Role-focused interpretation:

- whether defects are internal, external, or incoming-material driven
- whether containment is warranted
- which tradeoff between quality protection and throughput is justified

When this role exists in runtime play, its report should still respect hidden simultaneous turns and must not expose other roles' current-turn actions before reveal.

## MVP Versus Future Expansion

| Report Element | MVP Now | Future Expansion |
| --- | --- | --- |
| Yield-like production quality signals | Limited | Yes |
| Customer return signal | Limited | Yes |
| Defect categorization | No | Yes |
| Supplier incoming-quality scorecard | No | Yes |
| Quarantine and hold status | No | Yes |
| Audit and compliance tracking | No | Yes |
| Corrective-action workflow | No | Yes |

## Example Decisions This Report Should Enable

- `Tighten inspection`: accept more friction to stop customer escapes
- `Quarantine batch`: hold suspect output until the risk is understood
- `Escalate supplier quality`: push Procurement to address incoming-material instability
- `Launch root-cause work`: investigate recurring failures instead of treating symptoms each round

## Design Guardrails

When contributors implement or refine this report, they should:

- frame it clearly as future-role guidance until the role exists in-game
- connect quality signals to decisions, not just defect totals
- make customer-risk and containment logic obvious
- keep the tension between quality protection and throughput explicit
