# Procurement Manager Gameplay Playbook

This playbook teaches a human player or AI agent how to perform the `Procurement Manager` role in realistic weekly MVP play.

## Mission

Your job is to keep future production supplied without turning the plant's cash into unnecessary or mistimed inventory. Good procurement play protects flow first, but it does so with judgment: not every low-stock signal deserves a panic buy, and not every cheap bulk opportunity is worth the cash it consumes.

## What You Control

In the current MVP, you control:

- which part types you order
- how much of each part you order
- the rationale you attach to your decision

You do not control:

- current-round finance targets, which were set last round
- production releases or capacity allocation
- sales pricing or demand shaping
- same-round arrival of newly ordered parts

Critical MVP constraint:

- parts ordered this round arrive at the end of the next round, so procurement is always acting ahead of visible shortages

## Core Tension

You are balancing four things at once:

- supply continuity
- input cost
- cash usage
- inventory risk

The role becomes dangerous when you optimize only one of them.

Examples:

- buying too little protects cash now but risks starving production next round
- buying too much protects one shortage while trapping cash and increasing holding risk
- chasing low unit cost can leave the plant with the wrong inventory at the wrong time

## What You Should Read First Each Turn

Read the procurement report in this order:

1. Executive summary
2. Raw-material inventory status
3. Open purchase orders and receipts
4. Price and spend conditions
5. Supplier health and reliability, if modeled

Questions to answer before deciding:

- Which part is most likely to stop production next round?
- Which parts are already sufficiently covered by in-transit supply?
- Is the biggest risk shortage, cash pressure, or overbuy?
- Which purchase best protects the plant's real bottleneck?

## Top Metrics To Watch

- days or rounds of cover by critical part
- stock-out risk
- in-transit coverage
- purchase spend versus active finance target
- raw-material inventory exposure

Interpret them as a group, not in isolation.

Examples:

- low cover with low in-transit supply is urgent
- low cover with a receipt already arriving next round may not require more buying
- healthy cover plus high inventory exposure is often a warning against more purchasing

## Weekly Decision Checklist

Use this checklist every turn:

1. Identify the parts that could block next-round production.
2. Check whether those parts are already covered by in-transit orders.
3. Compare the likely shortage cost with the cash cost of buying now.
4. Prioritize the parts that protect the plant's most important output.
5. Cut or defer lower-value buys if cash is tight.
6. Sanity-check whether your plan creates obvious overbuy or duplication.
7. Write a short rationale that explains the tradeoff you are accepting.

## Trigger Guide

### When parts coverage is low

Default response:

- reorder the part that most directly protects near-term production

Escalate response when:

- multiple critical parts are simultaneously exposed
- finance pressure makes it impossible to protect all shortages

Bad response:

- buying every exposed part equally without prioritizing the true risk

### When backlog is rising

What it often means for Procurement:

- demand pressure may soon hit the parts that feed the most important products

Reasonable response:

- bias purchasing toward the parts behind the most commercially important output

Bad response:

- treating all backlog growth as a reason to buy large quantities of everything

### When cash is tight

Default response:

- protect only the purchases with the clearest flow benefit

Reasonable tactics:

- reduce buffer on lower-priority parts
- avoid duplicate or speculative buys
- coordinate with Finance if protecting flow requires an intentional overrun posture

Bad response:

- slashing all buys equally until a predictable shortage appears

### When inventory is already excessive

Default response:

- pause or shrink purchases that are not protecting the next real constraint

Ask:

- is this inventory supporting visible demand, or only making us feel safe?

Bad response:

- continuing to buy simply because price looks attractive or a supplier feels unreliable in the abstract

## Good Tradeoffs Vs Bad Tradeoffs

Good tradeoffs:

- spending more on one critical part to avoid a production stop
- accepting selective exposure on low-priority parts to protect cash
- keeping a modest buffer where lead-time delay would be very costly

Bad tradeoffs:

- buying large volumes of everything to avoid making a hard priority call
- using cheap unit price as the main reason for a large order
- protecting Procurement's local scorecard while increasing plant-wide inventory drag

## Common Failure Modes

- panic buying without prioritization
- buying against fear rather than visible plant need
- ignoring in-transit coverage and duplicating orders
- protecting unit cost while creating the wrong inventory mix
- assuming Production can always consume what Procurement buys

## Working With Other Roles

### Finance Controller

You need Finance when:

- protecting flow requires spending that will clearly strain the active target
- inventory exposure is already high and you need to defend a purchase anyway

What to share:

- which part is the real shortage risk
- what the shortage would cost in lost output or backlog damage
- why a cheaper or smaller buy is not actually safer

### Production Manager

You need Production to understand:

- which parts truly feed the bottleneck
- which shortages would make the next plan unrealistic

What to share:

- exposed parts
- expected arrivals
- where procurement protection is strong versus weak

### Sales Manager

You need Sales context when:

- backlog is shifting heavily toward one product
- commercial pressure is changing what inventory matters most

What to share:

- where material coverage cannot credibly support the demand story

### Future Quality And Logistics Roles

Once those roles exist, Procurement should also coordinate on:

- incoming-material quality risk
- warehouse congestion and inventory aging

## Example Decision Patterns

### Safe Decision

Situation:

- one critical part has low cover
- in-transit supply is not enough
- cash is acceptable

Decision:

- place a focused replenishment order for that part and keep the rest of the buy plan disciplined

Why it works:

- it protects flow without creating broad overbuy

### Aggressive Decision

Situation:

- backlog is rising and one product is becoming commercially important
- several parts are exposed, but one drives the real bottleneck risk

Decision:

- concentrate spend heavily on the parts behind that product and accept thinner buffers elsewhere

Why it can be right:

- it aligns procurement with the plant's most valuable near-term output

Main risk:

- lower-priority items may become next round's problem if demand broadens

### Risky Decision

Situation:

- prices look favorable
- inventory is already high
- cash is tightening

Decision:

- place a large buy across multiple parts because it seems efficient on unit cost

Why it is risky:

- it protects Procurement's local comfort while increasing cash pressure and inventory drag

## Notes For Human Briefings And AI Role Instructions

Reusable guidance:

- protect future flow, not just low purchase price
- prioritize shortages instead of buying broadly
- treat cash as a real constraint, not an afterthought
- avoid solving uncertainty by building the wrong inventory

Good procurement reasoning should sound like:

- `This order protects the next real production risk.`
- `I am accepting selective exposure elsewhere to preserve cash.`
- `I am not buying this part yet because in-transit coverage is already enough.`
