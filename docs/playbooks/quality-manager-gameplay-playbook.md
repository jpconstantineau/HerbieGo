# Quality Manager Gameplay Playbook (Future Role)

This playbook defines how a human player or AI agent would likely perform the future `Quality Manager` role in realistic HerbieGo play once that role is implemented.

## Future Role Status

The `Quality Manager` is not currently implemented in the MVP action roster.

This document is forward-looking design guidance for future expansion, contributor onboarding, and AI-role planning. It describes what the role would likely control, monitor, and optimize if explicit quality mechanics are added later.

## Mission

Your job would be to protect the customer and the plant from preventable quality failure without turning quality into blind bureaucracy. Good quality play catches risk early, contains damage quickly, and improves the process where it matters most. Bad quality play either lets defects escape or blocks flow indiscriminately without improving the system.

## What This Role Would Likely Control

If implemented, the `Quality Manager` would likely control:

- inspection intensity by product, supplier, or process area
- containment actions such as holds, quarantine, or release restrictions
- escalation of supplier and process-quality issues
- prioritization of corrective-action and root-cause work
- recommendations on whether throughput pressure should yield to customer protection

This role would likely not control directly:

- production scheduling or labor allocation
- procurement spending authority
- sales pricing and demand posture
- plant-wide priority calls beyond quality-specific escalation

Critical future-role constraint:

- quality decisions should improve customer protection and process stability, not become a blanket excuse to stop output whenever uncertainty appears

## Core Tension

You would be balancing:

- customer protection
- internal process control
- throughput and schedule pressure
- cost of containment and rework
- long-term quality credibility

The role becomes dangerous at both extremes.

Examples:

- weak containment can protect short-term output while allowing customer damage to spread
- aggressive holds can reduce escapes while freezing too much healthy flow
- repeated firefighting can feel active while the underlying defect source remains untouched

## What You Should Read First Each Turn

Read the quality report in this order:

1. Executive summary
2. Yield and defect analysis
3. Customer escape and external failure
4. Supplier and incoming-material quality
5. Containment, compliance, and audit watchlist

Questions to answer before deciding:

- Is the main risk internal waste, customer escape, supplier instability, or compliance exposure?
- Which product, process, or supplier is creating the most business damage?
- Is this a week for monitoring, containment, or root-cause escalation?
- Are we underreacting to a growing defect source or overreacting to noise?
- What short-term throughput loss is justified to prevent a much larger downstream failure?

## Top Signals To Watch

- first-pass yield and defect trend
- scrap or rework pattern
- return, complaint, or RMA signal
- supplier incoming-quality instability
- open holds, quarantine, or overdue corrective actions

Interpret them together.

Examples:

- worsening internal yield with no customer escapes yet may still justify early containment
- rising RMAs usually mean quality has already moved beyond a local process problem
- repeat supplier failures should not be treated as isolated production bad luck

## Weekly Decision Checklist

Use this checklist every turn:

1. Identify the biggest source of current quality risk.
2. Decide whether the risk is internal, external, incoming-material driven, or compliance related.
3. Check whether immediate containment is needed to protect customers.
4. Decide whether inspection should tighten, hold, or relax.
5. Escalate the issue to the function that owns the likely root cause.
6. Sanity-check that your response improves protection more than it harms healthy flow.
7. Write a short rationale that explains the tradeoff you are accepting.

## Quality Vs Throughput Tradeoffs

Good tradeoffs:

- accepting a temporary hold on suspect material to avoid wider customer damage
- tightening inspection around a rising defect source instead of slowing every flow equally
- allowing controlled output to continue where risk is understood and contained

Bad tradeoffs:

- blocking broad production without evidence that the risk is widespread
- ignoring repeated low-level signals because current shipments are still moving
- treating all defects as equally important regardless of customer or compliance risk

## Trigger Guide

### When internal defects are rising

Default response:

- tighten control around the affected process or product before defects become normalized

Reasonable actions:

- raise inspection intensity selectively
- escalate a targeted root-cause investigation
- warn Production that throughput pressure is creating unstable output

Bad response:

- waiting for customer failures before acting

### When customer complaints or RMAs are repeating

Default response:

- prioritize customer protection over local output convenience

Reasonable actions:

- contain suspect output
- investigate whether the source is process, supplier, or release discipline
- coordinate with Sales on customer-risk posture

Bad response:

- letting shipments continue unchanged because backlog pressure feels urgent

### When supplier-quality issues are recurring

Default response:

- treat the issue as a cross-role quality and procurement problem, not a one-off plant nuisance

Reasonable actions:

- escalate incoming inspection
- identify the affected material families
- push Procurement for supplier corrective action or source review

Bad response:

- absorbing bad incoming quality silently inside the plant until scrap and rework explode

### When audit or compliance risk is rising

Default response:

- slow down enough to stay credible and compliant

Reasonable actions:

- place targeted holds
- prioritize corrective action closure
- make the compliance risk visible to Plant leadership

Bad response:

- hiding exposure because the plant is under schedule pressure

## Common Failure Modes

- using quality language to block work without clear risk evidence
- tolerating repeat escapes because throughput targets feel more urgent
- treating supplier defects as purely a production problem
- measuring quality only by internal scrap while missing customer harm
- over-focusing on detection while under-investing in root-cause elimination

## Working With Other Roles

### Production Manager

You need Production when:

- output pressure is colliding with process stability
- defect patterns suggest poor process discipline or rushed execution

What to share:

- where quality risk is concentrated
- what containment is necessary
- what process behavior appears to be driving defects

### Procurement Manager

You need Procurement when:

- incoming-material quality is unstable
- supplier behavior is creating internal disruption or customer risk

What to share:

- what supplier or material family is failing
- whether the issue justifies tighter receiving control or escalation

### Sales Manager

You need Sales when:

- customer-facing failure is rising
- service promises need to reflect quality containment reality

What to share:

- whether customer protection requires shipment caution
- when sales pressure could worsen trust damage

### Finance Controller

You need Finance when:

- containment, rework, or corrective action has meaningful cost impact
- the plant is under pressure to underreact because money is tight

What to share:

- why containment cost is still cheaper than customer failure
- where underinvestment is creating bigger downstream risk

### Future Plant Manager

Once that role exists, Quality should also coordinate on:

- plant-wide tradeoffs between customer protection, throughput, and credibility

## Example Decision Patterns

### Safe Decision

Situation:

- one product shows a mild defect increase
- customer escapes are still limited
- the source appears localized

Decision:

- tighten inspection and launch targeted root-cause work on that flow only

Why it works:

- it increases protection without spreading disruption across the whole plant

### Aggressive Decision

Situation:

- complaint rates are rising
- suspect lots are still moving
- production pressure remains high

Decision:

- quarantine the exposed output and force rapid cross-role escalation

Why it can be right:

- it accepts short-term pain to stop a larger customer and brand failure

Main risk:

- if the hold is too broad or poorly targeted, the plant may lose healthy throughput unnecessarily

### Risky Decision

Situation:

- supplier defects are recurring
- scrap and rework are climbing
- backlog pressure feels intense

Decision:

- keep shipping and rely on downstream firefighting instead of tighter control

Why it is risky:

- it preserves short-term flow by allowing the underlying quality problem to spread

## Likely Future Mechanics Needed

To support this role well, future simulation design would likely need:

- explicit defect and yield tracking
- customer returns or complaint signals
- supplier-quality differentiation
- hold or quarantine mechanics
- corrective-action or root-cause workflow signals
- some way to express inspection intensity and release discipline

## Notes For Human Briefings And AI Role Instructions

Reusable guidance:

- protect the customer without freezing healthy flow
- treat repeat defects as system signals, not isolated bad luck
- contain targeted risk early instead of accepting broad damage later
- quality credibility is built by judgment, not blanket restriction

Good quality reasoning should sound like:

- `I am tightening control where the defect signal is real instead of slowing everything equally.`
- `This hold protects the customer from a risk that is no longer acceptable.`
- `I am escalating the likely root cause rather than absorbing the problem downstream.`
