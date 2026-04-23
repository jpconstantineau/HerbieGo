# Maintenance Manager Gameplay Playbook (Future Role)

This playbook defines how a human player or AI agent would likely perform the future `Maintenance Manager` role in realistic HerbieGo play once that role is implemented.

## Future Role Status

The `Maintenance Manager` is not currently implemented in the MVP action roster.

This document is forward-looking design guidance for future expansion, contributor onboarding, and AI-role planning. It describes what the role would likely control, monitor, and optimize if explicit maintenance and reliability mechanics are added later.

## Mission

Your job would be to protect future throughput by keeping critical assets healthy enough to support the plant's real demand, not just this week's output pressure. Good maintenance play accepts the right amount of planned pain to avoid much bigger unplanned failure later. Bad maintenance play either defers too much work until reliability collapses or overreacts in ways that stop useful production without improving the real risk.

## What This Role Would Likely Control

If implemented, the `Maintenance Manager` would likely control:

- timing and prioritization of preventive maintenance
- sequencing of reactive repair work
- recommendations for short planned downtime
- escalation of spare-parts risk and maintenance backlog
- repair-versus-replace recommendations for repeat bad actors

This role would likely not control directly:

- production mix or release decisions
- procurement budgets or supplier selection
- sales commitments
- plant-wide prioritization beyond maintenance-specific escalation

Critical future-role constraint:

- maintenance should improve reliability and protect throughput, not become a reflex to stop equipment whenever pressure rises or uncertainty appears

## Core Tension

You would be balancing:

- uptime today
- reliability tomorrow
- maintenance spend
- planned downtime
- operational pressure to keep producing

The role becomes dangerous at both extremes.

Examples:

- deferring preventive work can preserve this week's output while creating next week's failure
- forcing too much planned downtime can protect assets while starving the plant of useful throughput
- repeated repairs can look responsive while avoiding the harder decision to address chronic bad actors

## What You Should Read First Each Turn

Read the maintenance report in this order:

1. Executive summary
2. Equipment downtime and reliability
3. Preventive maintenance and workload balance
4. Spare parts and MRO readiness
5. Asset-health and performance watchlist

Questions to answer before deciding:

- Which asset is the next real threat to throughput?
- Is the plant paying too much for deferral already?
- Which work is preventive and valuable, and which work can wait safely?
- Are missing spares or repeat failures turning a manageable problem into a crisis?
- Is this the week to accept a short planned stop to avoid a much worse unplanned one?

## Top Signals To Watch

- unplanned downtime and failure trend
- mean time between failures and mean time to repair
- preventive-maintenance completion versus backlog growth
- critical spare availability
- repeat bad actors or recurring symptoms

Interpret them together.

Examples:

- rising downtime with growing PM backlog usually means deferral is already hurting the plant
- one bad actor with repeated repairs may deserve disproportionate attention
- missing spares can make even a modest failure much more dangerous

## Weekly Decision Checklist

Use this checklist every turn:

1. Identify the asset creating the biggest current or near-term reliability risk.
2. Decide whether the week calls for prevention, repair prioritization, or escalation.
3. Check whether critical spares or maintenance capacity will constrain the response.
4. Decide whether planned downtime now is better than unplanned downtime later.
5. Protect the work that most improves plant resilience, not just the loudest failure.
6. Sanity-check that your action improves total throughput rather than only near-term comfort.
7. Write a short rationale that explains the tradeoff you are accepting.

## Reliability Vs Output Tradeoffs

Good tradeoffs:

- taking a short planned stop to avoid a likely longer failure later
- prioritizing one bad actor over a long list of lower-value tasks
- escalating spare coverage before a breakdown turns into a prolonged outage

Bad tradeoffs:

- deferring all preventive work because the plant is under pressure
- treating every maintenance request as equally urgent
- spending heavily on reactive fixes without addressing the recurring source of failure

## Trigger Guide

### When downtime is rising

Default response:

- focus maintenance attention on the asset hurting throughput most

Reasonable actions:

- prioritize the highest-impact repair
- surface whether the plant is now living in reactive mode
- warn Production that reliability loss is becoming the real bottleneck

Bad response:

- spreading maintenance effort thinly across many small issues

### When PM backlog is growing

Default response:

- assume future reliability is being borrowed away unless proven otherwise

Reasonable actions:

- protect the overdue PMs tied to critical assets
- negotiate short planned stops where they meaningfully reduce risk
- challenge output pressure that is making degradation invisible

Bad response:

- continuing to defer because nothing has failed yet

### When critical spares are low

Default response:

- treat spare risk as part of uptime risk, not only as an inventory issue

Reasonable actions:

- escalate the specific spare family that would prolong downtime most
- coordinate with Procurement and Finance before the failure happens
- adjust repair priority based on what can actually be supported

Bad response:

- waiting for the breakdown before checking whether repair parts exist

### When one asset becomes a chronic bad actor

Default response:

- stop treating it like a string of unrelated incidents

Reasonable actions:

- prioritize root-cause work
- build the case for a more durable intervention
- surface the long-term cost of repeated patch repairs

Bad response:

- celebrating fast repair turnaround while the same asset keeps failing

## Common Failure Modes

- protecting this week's output by sacrificing next week's reliability
- reacting only after failure instead of acting on early warning
- letting the PM backlog grow until it becomes normal
- underestimating spare-part readiness as a throughput risk
- treating chronic bad actors as isolated repair events

## Working With Other Roles

### Production Manager

You need Production when:

- a short planned stop could protect a much larger future loss
- output pressure is forcing unhealthy deferral

What to share:

- which assets are becoming dangerous
- what maintenance action best protects useful throughput

### Procurement Manager

You need Procurement when:

- spare coverage is weak
- repair lead times or part risk threaten restoration speed

What to share:

- which spares matter most
- where missing parts could turn a failure into extended downtime

### Finance Controller

You need Finance when:

- maintenance spend needs justification
- preventive work or spare investment protects a larger downstream outcome

What to share:

- why near-term maintenance cost avoids bigger loss later
- when cutting support now would be false economy

### Future Plant Manager

Once that role exists, Maintenance should also coordinate on:

- plant-level prioritization when reliability protection and output pressure are directly in conflict

## Example Decision Patterns

### Safe Decision

Situation:

- one constrained asset is showing moderate degradation
- PM backlog is manageable
- a short planned stop is still feasible

Decision:

- take the short planned maintenance action now and protect future reliability

Why it works:

- it accepts limited disruption before the asset becomes a larger threat

### Aggressive Decision

Situation:

- downtime is rising
- one bad actor is hurting throughput repeatedly
- spare readiness exists to support a durable intervention

Decision:

- concentrate maintenance effort on the bad actor and accept visible short-term output pain

Why it can be right:

- it sacrifices some immediate production to restore a much healthier future operating position

Main risk:

- if the diagnosis is wrong, the plant absorbs the downtime without fixing the true cause

### Risky Decision

Situation:

- output pressure is high
- PM backlog is growing
- repeat failures are increasing

Decision:

- defer preventive work again and rely on reactive repair speed

Why it is risky:

- it preserves this week's output optics while increasing the odds of a much more damaging breakdown

## Likely Future Mechanics Needed

To support this role well, future simulation design would likely need:

- asset-level downtime and reliability signals
- preventive-maintenance backlog and completion tracking
- critical spare readiness
- repeat-failure or bad-actor visibility
- some representation of planned shutdown tradeoffs
- repair-versus-replace pressure for chronic problems

## Notes For Human Briefings And AI Role Instructions

Reusable guidance:

- protect reliability before failure becomes the only teacher
- planned pain is often cheaper than unplanned downtime
- not every asset deserves equal maintenance attention
- spare readiness is part of uptime protection, not a side issue

Good maintenance reasoning should sound like:

- `I am accepting a short planned stop to avoid a much worse failure later.`
- `This asset has become the real reliability threat to plant throughput.`
- `I am escalating spare risk now because waiting for the breakdown would be too late.`
