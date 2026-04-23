# Cross-Role Conflicts, Negotiation, And Escalation Guide

This guide documents the recurring cross-role tensions that make HerbieGo feel like a realistic plant instead of a set of isolated local optimization puzzles.

Use this document when contributors need to:

- understand where role incentives should naturally collide
- write or refine role cards, playbooks, and AI prompts
- decide what information roles should share before and after a round
- identify when a disagreement should remain local versus escalate to plant-level coordination

## Why Cross-Role Conflict Matters

The simulation becomes interesting when every role is trying to help the plant from its own local vantage point, but their best local move is not automatically the best total-plant move.

Healthy gameplay tension should feel like this:

- Procurement wants enough material protection to keep flow stable
- Production wants feasible throughput on the most important mix
- Sales wants revenue and backlog quality without destroying service credibility
- Finance wants liquidity and discipline without starving profitable flow

Those goals are all reasonable. Conflict appears because they are not always simultaneously satisfiable.

Good documentation should therefore explain:

- what each side is trying to protect
- what evidence each side should bring into the discussion
- when one role should yield
- when the disagreement has become a plant-level problem

## MVP Conflict Matrix

| Role Link | What Each Side Is Trying To Protect | Healthy Negotiation Focus | Harmful Local Optimization Pattern | Default Escalation Trigger |
| --- | --- | --- | --- | --- |
| Procurement vs Finance | Material coverage versus cash discipline | Which purchases protect the next real bottleneck | Procurement buys broad safety stock or Finance cuts broadly without priority | A likely shortage or hard cash/debt violation cannot both be avoided |
| Procurement vs Production | Future part coverage versus near-term feasible output | Which parts truly matter to the bottleneck and next-round plan | Procurement chases cheap or broad buys while Production assumes every shortage deserves immediate coverage | The feasible production plan changes materially because supply and consumption assumptions diverge |
| Production vs Sales | Useful output and feasibility versus revenue and backlog pressure | Which product mix best protects service and commercial value | Production maximizes activity while Sales keeps amplifying demand the plant cannot support | Customer commitments or backlog pressure exceed visible physical reality |
| Production vs Finance | Throughput support versus budget discipline | Which spend protects the bottleneck and which spend is low value | Production uses spend to hide poor prioritization or Finance under-supports profitable flow | Cost control would clearly choke the bottleneck or service-critical output |
| Sales vs Finance | Revenue growth versus margin, cash, and revenue quality | Whether the plant needs more demand, better demand, or less pressure | Sales discounts into an overloaded plant or Finance blocks support for healthy growth | Revenue pursuit materially worsens cash, margin, or service stability |

## Future-Role Conflict Matrix

These tensions are design-important even though the future roles are not yet implemented in the MVP action roster.

| Role Link | Main Tension | Why It Matters | Likely Escalation Trigger |
| --- | --- | --- | --- |
| Production vs Maintenance | Output today versus reliability tomorrow | A short planned stop may protect far more future throughput than it costs | Reliability risk becomes a bigger throughput threat than the planned downtime |
| Production vs Quality | Throughput versus customer protection and process control | Shipping more can be harmful if defect escape or containment risk is rising | The plant cannot maintain both current output pressure and acceptable quality protection |
| Sales vs Logistics | Commercial urgency versus physical shipping reality | A promise is not real service unless the warehouse can move and ship it credibly | Repeated expedite pressure or outbound lateness shows demand promises exceed flow capability |
| Finance vs Plant Manager | Local financial discipline versus total plant performance | Strict cost control can be right or can become false economy at plant scale | One plant-wide constraint clearly deserves support that local finance logic resists |
| Procurement vs Quality | Material availability versus incoming-quality protection | Cheap or abundant supply is not useful if it destabilizes the plant or customer quality | Supplier-quality risk is repeatedly harming flow or customer outcomes |
| Logistics vs Production | Storage and movement reliability versus local output comfort | Overproduction can turn the warehouse into the next bottleneck | Finished-goods buildup or congestion is now harming service or throughput |

## Typical Negotiation Patterns

### Pattern 1: Protect The Real Constraint

This is the healthiest recurring negotiation pattern in the game.

Questions the roles should ask:

- What is the current bottleneck or highest-value risk?
- Which local sacrifice best protects that system-level priority?
- What is the smallest compromise that preserves plant performance?

Good example:

- Procurement accepts selective shortage risk on low-priority parts so Production can protect the real bottleneck under a tight budget

Bad example:

- every role demands full protection for its own concern and the plant spreads itself too thin

### Pattern 2: Distinguish Useful Flow From Activity

Many bad negotiations happen because one side is defending busyness instead of value.

Good questions:

- does this action improve actual throughput, service, or margin quality?
- or does it only improve a local dashboard or reduce anxiety?

Good example:

- Production reduces releases because more WIP would not improve completions

Bad example:

- Production asks for more spend simply to keep every resource busy

### Pattern 3: Protect Revenue Quality, Not Just Volume

Sales and Finance often need this pattern.

Good questions:

- are we trying to win more demand, or better demand?
- is the plant capable of serving what Sales wants to win?
- is the margin or service cost of winning this volume acceptable?

Good example:

- Sales raises price slightly to protect backlog quality when service is already fragile

Bad example:

- Sales discounts into a plant that is already overloaded because the revenue target feels urgent

### Pattern 4: Separate Short-Term Optics From Real Risk

Finance, Maintenance, and Quality especially need this pattern.

Good questions:

- does this decision solve the real problem or merely delay when it becomes visible?
- what risk gets worse if we choose the cleaner-looking short-term option?

Good example:

- Finance still supports a targeted spend because the downstream loss from starving the bottleneck would be worse

Bad example:

- Finance cuts every spend category equally and calls the result discipline

## Information To Share Before And After A Round

The game should reward sharing interpretation and constraint signals without leaking hidden current-turn choices.

Before reveal, roles should share:

- visible plant state such as inventory, backlog, capacity, cash, and service outcomes
- role-specific interpretation of what the visible state most likely means
- warnings about emerging risk, such as likely shortages, overloaded backlog, or weak margin quality
- conditional guidance such as `if service remains this weak, demand should not be pushed harder`

Before reveal, roles should not share:

- hidden current-turn actions they have already locked in
- invented certainty about what another role is about to do
- role-specific prompt content that leaks current-turn private intent

After resolution, roles should share:

- what action they took
- what tradeoff they intended to make
- what evidence supported the choice
- what downstream effect they now expect others to manage

## Escalation Triggers

Escalation should be rare enough to feel meaningful but clear enough that contributors know when local negotiation has run out of room.

Escalate when one or more of these conditions holds:

- the disagreement concerns plant survival, not just local preference
- two roles cannot both achieve their minimum acceptable outcome this round
- the visible state shows that repeated local compromise is no longer protecting the plant
- one role's local optimization is creating material harm for multiple other roles
- a constraint is plant-wide enough that no single role should decide alone

### MVP Escalation Examples

- Procurement versus Finance:
  protecting the critical part would exceed the financial guardrail, but not buying would likely idle the plant next round

- Production versus Sales:
  backlog and demand pressure are rising beyond what visible parts and capacity can support

- Production versus Finance:
  budget discipline would force the plant to under-support the bottleneck in a way that clearly reduces useful throughput

- Sales versus Finance:
  revenue growth is available, but only through pricing or demand pressure that weakens margin and service credibility

### Future-Role Escalation Examples

- Production versus Maintenance:
  the plant must choose between a short planned stop and a larger reliability collapse

- Production versus Quality:
  current throughput pressure is no longer compatible with customer protection

- Sales versus Logistics:
  commercial urgency now exceeds what the warehouse can move or ship credibly

## Examples Of Good And Bad Alignment

### Good Alignment

- Procurement and Finance agree to protect one critical material family while delaying lower-value buys
- Production and Sales agree to prioritize the product whose backlog matters most commercially
- Finance supports selective spend that protects profitable throughput rather than generic activity
- roles explain not only what they want, but what tradeoff they are willing to accept

### Bad Alignment

- each role defends its local KPI as if plant-wide performance will automatically follow
- roles demand broad protection rather than choosing a priority
- disagreement stays polite but vague long after it should have been made explicit
- a role hides behind general urgency instead of naming the actual constraint

## Guidance For Role Cards, Playbooks, And AI Prompts

When reusing this material:

- role cards should summarize the role's most important synergies and conflicts
- gameplay playbooks should show how the role should negotiate under pressure
- AI prompts should include the role's local bias and the common cross-role tensions it must reason through
- runtime briefings should emphasize the current visible constraint without leaking hidden actions

If another document implies that a role can optimize in isolation, this guide should take precedence until the design deliberately changes.
