# Production Manager Gameplay Playbook

This playbook teaches a human player or AI agent how to perform the `Production Manager` role in realistic weekly MVP play.

## Mission

Your job is to convert available parts and finite workstation capacity into the most useful finished output the plant can realistically produce. Good production play is not about staying busy. It is about protecting throughput, controlling work-in-progress, and putting constrained capacity where it helps the plant most.

## What You Control

In the current MVP, you control:

- how many units of each product to release into production
- how workstation capacity is allocated between products
- the rationale you attach to your decision

You do not control:

- parts procurement this round
- customer demand or pricing
- next-round finance targets
- detailed intra-round scheduling or setup sequencing

Critical MVP constraints:

- you cannot consume parts that do not exist
- you cannot use more workstation capacity than the round allows
- if you request more than the system can legally advance, the plant trims execution to the maximum legal amount

## Core Tension

You are balancing:

- throughput
- utilization
- work-in-progress growth
- backlog relief
- budget pressure

The role becomes dangerous when you confuse activity with flow.

Examples:

- high releases can look productive while only creating WIP
- high utilization can look good while starving the real bottleneck of the right work
- more output of the wrong product can worsen service instead of improving it

## What You Should Read First Each Turn

Read the production report in this order:

1. Executive summary
2. Throughput and output status
3. Capacity and bottleneck view
4. Material readiness and starvation risk
5. Productivity and spend pressure

Questions to answer before deciding:

- What is the real bottleneck this round?
- Which product gives the best use of that constrained capacity?
- Which parts are likely to block the plan?
- Are we at risk of creating WIP that cannot complete?
- Is extra spend protecting flow or only making us feel active?

## Top Metrics To Watch

- feasible output versus target
- bottleneck utilization on useful work
- work-in-progress accumulation
- part-starvation risk
- production spend pressure

Read them together.

Examples:

- high requested output with low feasible output means the plan is unrealistic
- rising WIP with flat completions means you are feeding congestion, not flow
- a starved bottleneck is more dangerous than low utilization on a non-bottleneck station

## Weekly Decision Checklist

Use this checklist every turn:

1. Identify the current bottleneck or tightest constraint.
2. Identify which product most deserves that constrained capacity.
3. Check whether the required parts actually support that plan.
4. Reduce or defer low-value releases that would only create more WIP.
5. Align capacity allocation with the highest-value feasible output.
6. Decide whether any extra spend is justified by real throughput or service protection.
7. Write a short rationale that explains the tradeoff you are accepting.

## Bottleneck-Aware Thinking

Production wins by protecting the constrained resource, not by maximizing activity everywhere.

Good bottleneck-aware thinking:

- keep the bottleneck working on the most valuable feasible mix
- avoid feeding upstream work that cannot clear downstream
- accept lower local utilization when it improves total flow

Bad bottleneck-aware thinking:

- pushing all lines hard because idle time looks embarrassing
- allocating time evenly when one product matters more
- flooding the floor with WIP because the bottleneck is not yet visibly blocked

## Trigger Guide

### When parts are short

Default response:

- cut releases to the highest-value legal plan

Reasonable actions:

- shift mix toward the product with stronger material support
- escalate the shortage to Procurement early

Bad response:

- pretending the shortage is temporary and releasing work that cannot realistically finish

### When backlog is rising

What it often means:

- the plant needs focused output, not more generic activity

Reasonable response:

- put constrained capacity behind the backlog with the best plant-wide payoff

Bad response:

- trying to make a little of everything and relieving none of the real pressure

### When WIP is already high

Default response:

- reduce releases and protect downstream completion

Ask:

- are we moving work through, or just creating more unfinished inventory?

Bad response:

- releasing even more work because the bottleneck looks busy and that feels productive

### When finance pressure is high

Default response:

- reserve extra spend for decisions that clearly protect throughput or delivery reliability

Reasonable actions:

- cut low-value output
- avoid overtime that only masks a weak mix decision
- defend selective support spend if it protects the bottleneck meaningfully

Bad response:

- cutting all support equally and starving the plant's best output

## Good Tradeoffs Vs Bad Tradeoffs

Good tradeoffs:

- letting a noncritical line slow down to protect the real bottleneck
- reducing releases to prevent downstream congestion
- using selective overtime when it clearly protects high-value output

Bad tradeoffs:

- keeping every resource busy regardless of final throughput
- building finished goods or WIP the plant cannot ship profitably
- spending more on output that does not relieve the most important backlog

## Common Failure Modes

- over-releasing work that cannot complete
- feeding non-bottlenecks while the real constraint is underprotected
- mistaking utilization for throughput
- ignoring part shortages until the plan becomes partly impossible
- using overtime to hide poor prioritization

## Working With Other Roles

### Procurement Manager

You need Procurement when:

- a part shortage materially changes the feasible plan
- one product can only win if materials are protected next round

What to share:

- true near-term part needs
- which shortage matters most
- expected effect on feasible output

### Sales Manager

You need Sales when:

- backlog priorities are shifting
- one product's service impact is much more important than another's

What to share:

- what output is actually feasible
- which product mix protects the most important demand
- where promises now exceed physical reality

### Finance Controller

You need Finance when:

- support spend is required to protect meaningful flow
- a budget posture would clearly starve the bottleneck

What to share:

- why a spend decision protects throughput rather than generic activity
- what happens if the plant under-supports the current constraint

### Future Maintenance And Quality Roles

Once those roles exist, Production should also coordinate on:

- when downtime is worth taking to restore reliability
- when quality protection should slow or redirect output

## Example Decision Patterns

### Safe Decision

Situation:

- one product has stronger material support
- backlog is moderate
- bottleneck capacity is tight but stable

Decision:

- allocate most constrained capacity to the supported product and keep other releases modest

Why it works:

- it protects throughput without creating excess WIP

### Aggressive Decision

Situation:

- one product's backlog is commercially urgent
- parts are available
- extra spend can materially improve completions

Decision:

- concentrate capacity on that product and use selective overtime or support spend

Why it can be right:

- it protects high-value service when the plant can actually execute

Main risk:

- lower-priority output slips and may create next round's pressure

### Risky Decision

Situation:

- parts are uneven
- WIP is already high
- the bottleneck is tight

Decision:

- release large quantities of both products to keep resources busy

Why it is risky:

- it inflates activity while worsening congestion and doing little for real output

## Notes For Human Briefings And AI Role Instructions

Reusable guidance:

- protect throughput, not just utilization
- use the bottleneck deliberately
- do not release work that cannot realistically finish
- accept lower activity if it improves total plant flow

Good production reasoning should sound like:

- `This mix gives the best use of our constrained capacity.`
- `I am reducing releases because more WIP would hurt flow.`
- `I am spending more only because it protects meaningful output, not because idle time feels bad.`
