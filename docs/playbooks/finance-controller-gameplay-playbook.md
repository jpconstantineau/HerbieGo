# Finance Controller Gameplay Playbook

This playbook teaches a human player or AI agent how to perform the `Finance Controller` role in realistic weekly MVP play.

## Mission

Your job is to protect the plant's financial survivability without starving it of the support it needs to keep producing profitably. Good finance play sets next-round targets that are strict, realistic, and throughput-aware. Bad finance play improves short-term optics while making the plant weaker.

## What You Control

In the current MVP, you control:

- next-round procurement budget target
- next-round production support spend target
- next-round revenue target
- next-round cash floor or debt ceiling target
- the rationale you attach to your decision

You do not control:

- current-round procurement orders
- current-round production releases or capacity allocation
- current-round sales pricing
- retroactive veto of same-round actions

Critical MVP constraint:

- your submission affects the next round, not the current one

## Core Tension

You are balancing:

- cash preservation
- budget discipline
- throughput support
- service stability
- long-term profitability

The role becomes dangerous when you treat all spending as equally bad.

Examples:

- tighter budgets can protect cash or they can starve profitable flow
- strong revenue can still be unhealthy if margin quality and inventory exposure are deteriorating
- low spending can look disciplined while forcing the plant into shortages, backlog decay, or missed profit

## What You Should Read First Each Turn

Read the finance report in this order:

1. Executive summary
2. Cash and debt position
3. Flash P&L and margin quality
4. Budget pressure and spend drivers
5. Inventory and working-capital exposure

Questions to answer before deciding:

- Is the next real risk cash survival, debt headroom, weak margin, or service collapse?
- Which spend is protecting profitable flow, and which spend is mostly waste or buildup?
- Is inventory supporting the plant or trapping cash?
- Should next round be tighter overall, or only in one specific area?
- What should Finance still support because cutting it would damage the plant more than it helps?

## Top Metrics To Watch

- ending cash position
- debt versus debt ceiling
- gross margin signal
- inventory exposure
- budget realism

Interpret them together.

Examples:

- low cash plus rising dead inventory usually calls for tighter discipline
- low cash plus a high-value shortage risk may justify selective support
- strong revenue with weakening margin quality is not a sign to relax automatically

## Weekly Decision Checklist

Use this checklist every turn:

1. Check whether cash and debt risk are rising or stable.
2. Identify which current spend is useful and which is low-value.
3. Check whether inventory is helping the plant or trapping working capital.
4. Decide where next-round targets should tighten and where they should still allow support.
5. Sanity-check that your budgets are strict but still executable.
6. Ask whether your target set protects total plant performance or only short-term optics.
7. Write a short rationale that explains the tradeoff you are accepting.

## Liquidity Vs Throughput Tradeoffs

Good tradeoffs:

- tightening low-value spend while still protecting the bottleneck
- allowing selective procurement or production support to avoid a much bigger downstream loss
- pushing Sales toward better revenue quality instead of weak volume

Bad tradeoffs:

- forcing every function to cut equally regardless of plant impact
- protecting cash this week by allowing shortages or backlog failure next week
- treating inventory growth as harmless because it does not hit the P&L immediately

## Trigger Guide

### When cash is tight

Default response:

- tighten weaker-spend areas first

Reasonable actions:

- reduce speculative or low-value purchases
- discourage output that creates inventory without service gain
- keep support for the most profitable flow when possible

Bad response:

- cutting every category so hard that the plant cannot execute a credible next round

### When debt pressure is rising

Default response:

- protect headroom before the plant hits a hard wall

Reasonable actions:

- tighten targets where overbuy or overproduction is visible
- raise the bar for spend that lacks a clear throughput payoff

Bad response:

- ignoring debt stress because current output still looks acceptable

### When margin quality is weakening

Default response:

- ask whether the plant is winning weak revenue, buying bad inputs, or supporting the wrong output mix

Reasonable actions:

- push Sales toward better pricing discipline
- push Procurement and Production away from low-value buildup

Bad response:

- cutting useful operating support without diagnosing the real cause

### When inventory is already too high

Default response:

- pressure the plant to stop converting cash into low-value stock

Reasonable actions:

- tighten procurement where coverage is already excessive
- challenge production of goods the plant cannot ship credibly

Bad response:

- allowing inventory to keep growing because short-term service looks easier that way

## Common Failure Modes

- treating all spend as equally harmful
- setting impossible targets that only guarantee trimming and failure
- focusing on cost optics instead of economic quality
- ignoring inventory because it is less visible than cash
- under-supporting profitable throughput at the exact moment the plant needs it most

## Working With Other Roles

### Procurement Manager

You need Procurement when:

- material protection requires cash you do not want to spend
- inventory is already heavy but shortage risk is also real

What to share:

- whether Finance sees this as justified support or dangerous overbuy
- what evidence would justify an exception

### Production Manager

You need Production when:

- overtime or support spend might protect valuable throughput
- low-value activity is being confused with real output

What to share:

- what Finance is still willing to support
- what spend now looks economically unjustified

### Sales Manager

You need Sales when:

- revenue quality is deteriorating
- demand pressure is worsening service or margin problems

What to share:

- whether Finance needs stronger pricing discipline
- when volume is becoming financially weak rather than helpful

### Future Plant Manager Role

Once that role exists, Finance should also coordinate on:

- plant-level prioritization when financial protection and operational pressure conflict

## Example Decision Patterns

### Safe Decision

Situation:

- cash is stable
- debt is manageable
- one spend area is visibly bloated without helping flow

Decision:

- tighten that specific budget while keeping support for the plant's strongest output

Why it works:

- it improves discipline without indiscriminately weakening the plant

### Aggressive Decision

Situation:

- cash is tightening
- inventory is already high
- margin quality is weakening

Decision:

- materially tighten procurement and production support targets except where the bottleneck clearly needs protection

Why it can be right:

- it stops working-capital deterioration before survival pressure becomes severe

Main risk:

- if Finance misidentifies the useful spend, the plant can be starved at the wrong place

### Risky Decision

Situation:

- cash looks stressed
- backlog is commercially important
- one targeted support decision could protect profitable flow

Decision:

- cut all categories equally because discipline feels safer

Why it is risky:

- it preserves short-term caution while allowing a larger throughput or service failure to unfold

## Notes For Human Briefings And AI Role Instructions

Reusable guidance:

- protect liquidity, but not blindly
- distinguish productive support from low-value spend
- do not starve the bottleneck to improve short-term optics
- inventory is a financial decision, not just an operations detail

Good finance reasoning should sound like:

- `I am tightening the spend that adds little value while still protecting profitable flow.`
- `This target is strict but still executable.`
- `I am not cutting this support because the downstream loss would be bigger than the savings.`
