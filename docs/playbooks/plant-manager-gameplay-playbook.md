# Plant Manager Gameplay Playbook (Future Role)

This playbook defines how a human player or AI agent would likely perform the future `Plant Manager` role in realistic HerbieGo play once that role is implemented.

## Future Role Status

The `Plant Manager` is not currently implemented in the MVP action roster.

This document is forward-looking design guidance for future expansion, contributor onboarding, and AI-role planning. It describes what the role would likely control, monitor, and optimize if a plant-wide coordination role is added later.

## Mission

Your job would be to improve total plant performance when local role incentives are no longer aligning on their own. Good plant-management play identifies the real system constraint, resolves the most important cross-role conflicts, and accepts the right local pain to protect the whole plant. Bad plant-management play either lets every function optimize for itself or overrides too much without understanding the real source of the problem.

## What This Role Would Likely Control

If implemented, the `Plant Manager` would likely control:

- plant-wide prioritization directives
- conflict resolution between functions
- escalation decisions when local negotiation is no longer enough
- emergency intervention when plant performance is materially at risk
- alignment of short-term targets around the most important system constraint

This role would likely not control directly:

- detailed role-specific execution inside each department
- exact purchasing quantities, production releases, or pricing decisions
- the mechanics of repair, inspection, or shipment work themselves

Critical future-role constraint:

- the Plant Manager should coordinate and prioritize, not become a generic super-role that replaces every local decision

## Core Tension

You would be balancing:

- local departmental success
- total plant throughput
- service reliability
- cash and margin health
- long-term resilience

The role becomes dangerous when you treat every visible problem as equally important or every conflict as a command problem.

Examples:

- protecting one department's target can hurt total plant performance
- forcing a plant-wide directive too early can weaken useful local judgment
- waiting too long to intervene can let cross-role friction become backlog, cost, or service damage

## What You Should Read First Each Turn

Read the plant-manager report in this order:

1. Executive scorecard
2. Resource and bottleneck report
3. Market and fulfillment report
4. Variance and deep-dive report
5. Inter-role link watchlist

Questions to answer before deciding:

- What is the single biggest plant-wide constraint this week?
- Which local optimization is helping itself while hurting the system?
- Which conflict can still be handled by negotiation, and which now needs direction?
- Where should the plant accept short-term pain to avoid larger downstream failure?
- If only one problem gets focused leadership attention, which one most changes the total outcome?

## Top Signals To Watch

- overall throughput and service signal
- backlog or fulfillment deterioration
- cash, margin, or inventory pressure
- resource or bottleneck instability
- cross-role conflicts that are repeating without resolution

Interpret them together.

Examples:

- strong local output with worsening service can still mean the plant is pointed the wrong way
- high inventory with weak flow often means the system is busy but not healthy
- repeated Finance versus Operations tension may indicate a real prioritization gap, not just interpersonal friction

## Weekly Decision Checklist

Use this checklist every turn:

1. Identify the current plant-wide constraint.
2. Identify which role link or local optimization is worsening it.
3. Decide whether coordination, prioritization, or direct escalation is needed.
4. Choose the smallest plant-level intervention that materially improves the total outcome.
5. Check what local pain the plant should accept and what pain is avoidable.
6. Sanity-check that your directive improves the system rather than just shifting blame.
7. Write a short rationale that explains the tradeoff you are accepting.

## Plant-Wide Tradeoff Management

Good tradeoffs:

- protecting the true bottleneck even if another function's local target slips
- directing the plant to slow down in one area to recover total service or reliability
- forcing clarity when one role's local success is clearly damaging the whole plant

Bad tradeoffs:

- trying to make every function equally happy
- issuing broad directives without identifying the real constraint
- overriding local teams on details that they can still handle effectively themselves

## Trigger Guide

### When backlog or service is deteriorating

Default response:

- identify whether the failure is driven by supply, production, quality, logistics, or bad commercial pressure

Reasonable actions:

- force alignment around the highest-value service risk
- push the plant away from low-value activity
- escalate if local teams are solving different problems at the same time

Bad response:

- demanding generic urgency from everyone

### When margin or cash health is weakening

Default response:

- determine whether the real cause is weak pricing, bad mix, overbuy, overproduction, or firefighting cost

Reasonable actions:

- direct support toward the actions that protect economic quality
- challenge local decisions that are creating inventory or spend without throughput payoff

Bad response:

- cutting support everywhere equally without understanding the system effect

### When one function is protecting itself at plant expense

Default response:

- intervene before the local optimization becomes normalized

Reasonable actions:

- restate the plant-wide objective clearly
- force the tradeoff into the open
- decide which outcome matters more this week

Bad response:

- letting every role continue defending its own metric while the plant drifts

### When a major future-role pressure emerges

Examples:

- quality containment versus throughput
- maintenance downtime versus current output
- logistics flow versus storage saturation

Default response:

- make the total-plant tradeoff explicit and choose deliberately

Bad response:

- pretending those cross-functional costs will resolve themselves automatically

## Common Failure Modes

- confusing plant leadership with micromanagement
- letting local negotiations continue after they have clearly stopped working
- chasing the loudest problem instead of the real constraint
- protecting optics instead of system health
- moving cost or disruption from one department to another and calling it resolution

## Working With Other Roles

### Procurement Manager

You need Procurement when:

- supply protection conflicts with inventory or cash discipline

What to share:

- which material decision matters to the total plant outcome
- what level of risk the plant should accept elsewhere

### Production Manager

You need Production when:

- throughput, WIP, or bottleneck use is no longer aligned with plant priorities

What to share:

- what the plant most needs from constrained capacity
- where local activity is no longer creating useful system output

### Sales Manager

You need Sales when:

- commercial pressure is creating unrealistic service expectations

What to share:

- what the plant can support credibly
- when revenue quality matters more than volume

### Finance Controller

You need Finance when:

- budget discipline conflicts with protecting the real bottleneck or service risk

What to share:

- what spend is system-protective rather than locally convenient
- where false economy is beginning to damage the plant

### Future Quality, Logistics, And Maintenance Roles

Once those roles exist, Plant leadership should also coordinate on:

- quality containment versus throughput
- downtime for reliability versus short-term output
- warehouse or shipping constraints versus internal production comfort

## Example Decision Patterns

### Safe Decision

Situation:

- one clear bottleneck is limiting the plant
- the roles involved mostly agree on the problem
- only a small amount of alignment is needed

Decision:

- issue a focused plant priority that protects the true constraint and leave local execution to the roles

Why it works:

- it improves coordination without overreaching

### Aggressive Decision

Situation:

- service is deteriorating
- multiple functions are optimizing locally in different directions
- the plant is at risk of a visible customer or financial miss

Decision:

- override local preferences and force the plant around the highest-value plant-wide objective

Why it can be right:

- it stops cross-functional drift before the system takes a larger hit

Main risk:

- if the diagnosis is wrong, the directive can concentrate the plant on the wrong problem

### Risky Decision

Situation:

- recurring conflicts are emerging
- no single department wants to own the tradeoff
- plant performance is becoming unstable

Decision:

- ask each function to do its best and hope the conflict resolves naturally

Why it is risky:

- it avoids hard prioritization at exactly the moment the system most needs it

## Likely Future Mechanics Needed

To support this role well, future simulation design would likely need:

- cross-role escalation or directive mechanics
- clearer plant-wide bottleneck and variance summaries
- some way to represent local-versus-global target conflict
- explicit quality, maintenance, and logistics signals once those roles exist
- plant-level decision prompts that do not leak hidden current-turn state

## Notes For Human Briefings And AI Role Instructions

Reusable guidance:

- manage the system, not the loudest department
- escalate only when local negotiation no longer protects the plant
- accept the right local pain to improve the total outcome
- do not micromanage what roles can still handle well themselves

Good plant-manager reasoning should sound like:

- `The plant-wide constraint this week is not where the loudest pressure is coming from.`
- `I am prioritizing the tradeoff that most improves total plant performance.`
- `This directive is narrow on purpose so the local roles can still execute well.`
