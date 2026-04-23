# Finance Controller Weekly Report

This document defines the weekly report the Finance Controller should receive in HerbieGo.

The report should help the player answer one core question before acting:

`How do we protect cash, debt tolerance, and economic quality next round without starving the plant of the support it actually needs?`

This is an MVP-facing document. It keeps the report grounded in the current finance model while leaving room for richer accounting and planning features later.

## Role Scope

- Role: `Finance Controller`
- Status: `MVP role`
- Primary decision horizon: next round and the near-term cash outlook beyond it
- Main tradeoff: liquidity and budget discipline versus throughput support and future profitability

## What This Report Should Support

The weekly finance report should help the player decide whether to:

- tighten next-round budgets
- deliberately support procurement or production spend
- push for better revenue quality rather than just more volume
- protect cash and debt headroom before the plant hits a hard limit
- challenge overbuy, overproduction, or weak-margin growth

The report should not read like a passive accounting statement. It should help Finance choose next-round targets that are strict, realistic, and strategically useful.

## Recommended Report Structure

Present the report in this order.

## 1. Executive Summary

Start with a short summary panel that highlights the biggest financial risks facing the next round.

Recommended summary items:

- ending cash position and whether it is trending toward the floor
- debt level versus the active ceiling
- gross-margin signal and whether revenue quality is weakening
- inventory exposure tying up cash
- whether current spending pressure is coming from useful flow support or low-value buildup

Why first:

- the Finance Controller should immediately understand whether the plant needs restraint, support, or selective intervention

## 2. Cash And Debt Position

This is the most critical section because it shows whether the plant still has room to operate.

Use a compact table like this:

| Signal | Current Value | Trend | Risk Level | Why It Matters |
| --- | --- | --- | --- | --- |
| `example_signal` | Current state. | Improving, flat, or worsening. | `Low`, `Medium`, or `High`. | Decision relevance. |

Key fields:

- beginning and ending cash balance
- current debt level
- distance to the debt ceiling
- projected near-term cash pressure from already visible conditions
- largest expected cash drains, such as material spend, operating spend, or carrying costs

Decision value:

- helps Finance decide whether next-round targets should protect survival, maintain balance, or support growth

## 3. Flash P&L And Margin Quality

This section shows whether the plant's recent economic performance is healthy or misleading.

Recommended fields:

- shipped revenue
- cost of goods sold signal
- gross margin amount and margin percentage
- operating expense estimate
- operating income or equivalent round-profit signal

Use this table:

| Measure | Current Value | Healthy Signal | Warning Trigger | Likely Follow-Up |
| --- | --- | --- | --- | --- |
| `example_measure` | Current result. | What good looks like. | What should concern Finance. | What Finance may adjust. |

MVP note:

- the report can use simple round-level economics rather than full accounting precision
- the purpose is decision support, not perfect financial statements

Decision value:

- helps Finance distinguish strong revenue from weak-quality revenue

## 4. Budget Pressure And Spend Drivers

Finance sets next-round targets, so it needs to know where operating pressure is actually coming from.

Recommended fields:

- procurement spend relative to target
- production support spend relative to target
- whether overruns were throughput-protecting or low-value
- visible reasons for spend pressure, such as shortage recovery, overtime, or inventory buildup

Use this table:

| Area | Current Spend Signal | Target Pressure | Main Cause | Finance Interpretation |
| --- | --- | --- | --- | --- |
| `example_area` | Current status. | `Low`, `Medium`, or `High`. | Main driver. | Tighten, support, or investigate. |

Decision value:

- helps Finance avoid generic cuts and instead respond to the real source of pressure

## 5. Inventory And Working-Capital Exposure

Cash problems often hide inside inventory rather than the income line.

Recommended fields:

- raw-material inventory exposure
- work-in-progress exposure
- finished-goods exposure
- backlog-supported inventory versus idle inventory
- signs that the plant is tying up cash without improving service or throughput

Use one row per inventory class:

| Inventory Class | Current Exposure | Why It Exists | Healthy Signal | Warning Signal |
| --- | --- | --- | --- | --- |
| `example_class` | Current level. | Main reason. | Value-creating support. | Cash trap or buildup signal. |

Decision value:

- helps Finance challenge overbuy and overproduction without undermining truly necessary support

## 6. Finance Decision Prompts

End the report with 3 to 5 plain-language prompts that convert the report into next-round target decisions.

Recommended prompts:

- Is the next round's biggest risk cash survival, debt headroom, weak margin, or service collapse?
- Which spending pressure is protecting profitable flow, and which is only creating buildup?
- Is inventory acting like a strategic buffer or a cash trap?
- Should Finance tighten targets broadly, or only in one area?
- What support should Finance still allow because cutting it would harm the plant more than it helps?

Why this matters:

- the report should drive the next finance submission, not just explain last round's numbers

## Visibility Guidance

This report should combine plant-wide financial state with finance-specific interpretation.

Plant-wide inputs:

- cash and debt
- visible inventory levels
- visible backlog and shipment outcomes
- active budgets and recent spend signals
- recent plant metrics relevant to revenue and output

Role-focused interpretation:

- budget realism
- liquidity and debt risk assessment
- margin-quality assessment
- recommendations for where to tighten or deliberately support the plant

The report must not reveal hidden current-turn actions from Procurement, Production, or Sales before round resolution.

## MVP Versus Future Expansion

| Report Element | MVP Now | Future Expansion |
| --- | --- | --- |
| Cash position | Yes | Yes |
| Debt versus ceiling | Yes | Yes |
| Round-level revenue and profit signal | Yes | Yes |
| Procurement and production budget pressure | Yes | Yes |
| Inventory exposure by class | Yes | Yes |
| Detailed accrual accounting | Limited | Yes |
| Accounts receivable aging | No | Yes |
| Accounts payable scheduling | Limited | Yes |
| Full variance analysis by cost center | Limited | Yes |
| Rich long-range cash forecasting | Limited | Yes |

## Example Decisions This Report Should Enable

- `Tighten selectively`: cut a weak-spend area while preserving the plant's most valuable support
- `Support deliberately`: allow more procurement or production room because the alternative would damage profitable flow
- `Protect liquidity`: prioritize cash and debt headroom over lower-value growth
- `Challenge inventory buildup`: pressure the plant to stop turning cash into low-value stock

## Design Guardrails

When contributors implement or refine this report, they should:

- make survival and liquidity risks obvious early
- separate useful spend from low-value spend
- connect financial pressure to plant behavior rather than isolated accounting labels
- keep the MVP report honest about its simplified finance model
