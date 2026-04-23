# Sales Manager Weekly Report

This document defines the weekly report the Sales Manager should receive in HerbieGo.

The report should help the player answer one core question before acting:

`How do we shape demand and protect revenue quality without creating backlog and service promises the plant cannot credibly support?`

This is an MVP-facing document. It keeps the report grounded in the current price, backlog, shipment, and customer-sentiment model while leaving room for richer market features later.

## Role Scope

- Role: `Sales Manager`
- Status: `MVP role`
- Primary decision horizon: next round's demand impact and the near-term backlog horizon after that
- Main tradeoff: revenue capture versus price discipline, service credibility, and manageable backlog

## What This Report Should Support

The weekly sales report should help the player decide whether to:

- raise or lower price by product
- protect margin instead of chasing more volume
- cool demand when backlog or service risk is already too high
- press for more growth when the plant can credibly support it
- escalate when demand opportunity is real but operations cannot keep up

The report should not behave like a passive sales dashboard. It should help the Sales Manager make the next pricing and demand-shaping decision with the plant's visible constraints in mind.

## Recommended Report Structure

Present the report in this order.

## 1. Executive Summary

Start with a short summary panel that highlights the biggest commercial risks and opportunities.

Recommended summary items:

- whether backlog is healthy, overloaded, or too thin
- which product has the highest near-term revenue opportunity
- whether service reliability is supporting or hurting future demand
- whether recent pricing appears too aggressive or too weak
- which customer or product segment is most at risk of dissatisfaction

Why first:

- the Sales Manager should immediately know whether to pursue growth, protect margin, or restore credibility

## 2. Revenue And Demand Pipeline

This is the first detailed section because Sales needs to see whether pricing is producing useful demand.

Use one row per product:

| Product | Current Price | Recent Revenue Signal | Demand Signal | Booked Or Accepted Demand | Commercial Risk | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `example_product` | Active market price. | Recent shipped-revenue signal. | Strong, moderate, or weak demand. | Current demand queue or recent intake. | `Low`, `Medium`, or `High`. | Main interpretation. |

Key fields:

- current market price by product
- recent revenue signal
- demand response to current pricing
- accepted or booked demand signal
- comparison between revenue growth and demand quality

MVP note:

- the current MVP centers on price-driven demand and customer backlog rather than a full opportunity pipeline
- "pipeline" should therefore mean visible demand pressure and likely near-term order flow, not a complex CRM forecast

Decision value:

- helps Sales decide whether current pricing is producing the right kind of demand

## 3. Backlog And Fulfillment Pressure

Sales must understand whether demand is still supportable by the plant.

Use one row per product or customer-product segment:

| Backlog Area | Current Queue | Age Or Delay Signal | Fulfillment Risk | Customer Impact | Notes |
| --- | --- | --- | --- | --- | --- |
| `example_backlog` | Current backlog load. | Fresh, aging, or critical. | `Low`, `Medium`, or `High`. | Sentiment or revenue risk. | Main reason for concern. |

Key fields:

- backlog volume by product
- backlog age or expiry risk
- shipment pace versus incoming demand
- on-time delivery or equivalent service signal
- whether the current backlog is healthy demand or accumulating unkept promises

Decision value:

- helps Sales distinguish demand worth pursuing from demand that will degrade customer trust if accepted too aggressively

## 4. Customer Sentiment And Service Credibility

This section shows whether recent plant performance is making future sales easier or harder.

Recommended fields:

- customer sentiment trend
- service misses or backlog expiry events
- lead-time credibility signal
- returns, complaints, or visible downstream dissatisfaction signals

Use this table:

| Customer Or Signal | Current Status | Why It Matters | Warning Trigger | Likely Response |
| --- | --- | --- | --- | --- |
| `example_signal` | Current condition. | Why Sales should care. | What should worry the player. | Reasonable action. |

Decision value:

- helps Sales protect future demand quality instead of only chasing this week's volume signal

## 5. Price And Margin Quality

Sales should know whether current pricing is protecting economic value or only inflating weak volume.

Recommended fields:

- average selling price by product
- price trend versus recent demand response
- gross-margin quality signal if visible at the role-report level
- whether recent demand gains seem driven by discounting rather than strong commercial position

Use one row per product:

| Product | ASP Or Price Signal | Demand Response | Margin Quality Signal | Pricing Interpretation |
| --- | --- | --- | --- | --- |
| `example_product` | Current price quality. | How demand reacted. | Healthy or weakening. | Raise, hold, or lower cautiously. |

Decision value:

- helps Sales choose between growth, restraint, and repricing

## 6. Market And Mix Watchlist

This section surfaces where the next demand shift is most likely to matter.

Recommended fields:

- demand by product
- which product is becoming harder or easier to sell
- whether customer mix is pushing the plant toward a more or less supportable backlog
- lost-opportunity reasons when visible, such as price, lead time, or service credibility

Future expansion note:

- richer market-share, channel, and acquisition-cost measures can be added later
- the MVP version should stay anchored to product demand, service credibility, and price response

Decision value:

- helps Sales look one step ahead instead of reacting only to the current backlog

## 7. Sales Decision Prompts

End the report with 3 to 5 plain-language prompts that convert the report into a pricing decision.

Recommended prompts:

- Is the plant currently constrained enough that price should rise rather than demand rise?
- Which backlog is healthy demand, and which backlog is becoming a credibility problem?
- Are we winning low-quality volume that weakens margin or service?
- Which product has room for more demand without creating operational damage?
- What should be escalated to Production or Finance before another round of promises is made?

Why this matters:

- the report should guide the next sales action, not just summarize recent outcomes

## Visibility Guidance

This report should combine plant-wide commercial state with sales-specific interpretation.

Plant-wide inputs:

- customer backlog
- customer sentiment
- finished-goods availability
- recent shipments and service results
- active revenue targets and visible plant constraints

Role-focused interpretation:

- pricing posture
- demand quality assessment
- backlog-credibility assessment
- recommendations for where to grow, hold, or cool demand

The report must not reveal hidden current-turn actions from Procurement, Production, or Finance before round resolution.

## MVP Versus Future Expansion

| Report Element | MVP Now | Future Expansion |
| --- | --- | --- |
| Price by product | Yes | Yes |
| Revenue signal | Yes | Yes |
| Backlog by product | Yes | Yes |
| Customer sentiment | Yes | Yes |
| Shipment and service signal | Yes | Yes |
| Lead-time credibility signal | Limited | Yes |
| Lost-sales reason tracking | Limited | Yes |
| Market-share analysis | No | Yes |
| Customer acquisition cost | No | Yes |
| Channel or segment pipeline analysis | No | Yes |

## Example Decisions This Report Should Enable

- `Raise price`: moderate demand when backlog or service pressure is already too high
- `Hold price`: keep demand steady because backlog and service remain healthy
- `Lower price selectively`: stimulate demand where the plant has capacity and credibility to support it
- `Protect backlog quality`: stop chasing new volume when current commitments are already too fragile

## Design Guardrails

When contributors implement or refine this report, they should:

- make backlog quality and service credibility visible early
- connect pricing to operational reality rather than treating demand as free
- distinguish healthy demand growth from dangerous promise accumulation
- keep the MVP report honest about which commercial mechanics are not yet modeled
