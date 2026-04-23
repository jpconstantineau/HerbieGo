# Annotated Sample Turns For MVP Roles

This guide provides annotated sample turns for each current MVP role in HerbieGo.

Use it when contributors need to:

- see how a role moves from report to action
- align human-facing docs with AI prompting
- test whether role guidance actually produces realistic decisions

Each example includes:

- the round context the role sees
- the report highlights that matter most
- the action chosen
- why the action is reasonable
- what tradeoff is being accepted
- one rejected alternative and why it was rejected
- likely downstream effects on the rest of the plant

These examples are illustrative and intentionally use compact scenario language rather than exact balanced scenario numbers.

## Procurement Manager

## Scenario A: Stable Week, One Clear Exposure

### Round Context

- current cash is healthy
- debt is low
- `Housing` coverage is thin for the next round
- `Seal Kit` coverage is healthy
- one `Housing` receipt is in transit, but not enough to cover visible near-term need
- backlog pressure on `Pump` is moderate and growing

### Report Highlights That Matter Most

- critical part coverage shows `Housing` as the only real next-round exposure
- in-transit supply helps, but does not fully close the gap
- inventory exposure is acceptable overall
- active finance target is tight enough to discourage broad buying, but not so tight that all action must stop

### Action Chosen

- `order_part(Housing, moderate_quantity)`
- `comment("Protecting next-round Pump flow with a focused Housing buy while leaving lower-risk parts untouched.")`

### Why This Is Reasonable

The role is protecting the next visible bottleneck risk without turning one shortage concern into a broad buy wave. The current state does not justify buying everything. It justifies covering the part that most directly supports likely next-round production.

### Tradeoff Accepted

- spends cash now to reduce shortage risk next round
- accepts that some lower-priority exposure will remain unprotected

### Rejected Alternative

- `order_part(Housing, large_quantity)` and `order_part(Seal Kit, large_quantity)`

Why rejected:

- it would protect against uncertainty more broadly, but the current report does not justify tying up that much cash or increasing inventory exposure on already-covered material

### Likely Downstream Effects

- Production is less likely to face a `Pump` material constraint next round
- Finance sees a defensible, focused use of cash rather than broad safety-stock growth
- if demand shifts suddenly away from `Pump`, some inventory risk still remains

## Scenario B: Stressed Week Under Cash Pressure

### Round Context

- cash is tight and debt is rising toward the ceiling
- both `Body` and `Fastener Kit` show thinning cover
- one product's backlog is commercially weaker than the other
- inventory exposure is already elevated on noncritical materials

### Report Highlights That Matter Most

- the plant cannot safely buy every exposed part
- one part family matters much more to the higher-value near-term output
- finance pressure makes a broad replenishment posture unrealistic

### Action Chosen

- `order_part(Body, targeted_quantity)`
- `comment("Concentrating scarce spend on the material with the strongest near-term throughput payoff and accepting exposure elsewhere.")`

### Why This Is Reasonable

The role is recognizing that under stress, the correct answer is not equal protection for all shortages. It is focused protection of the part that best supports the most useful output.

### Tradeoff Accepted

- reduces the risk of one important shortage
- accepts selective exposure on lower-value material because cash cannot support everything

### Rejected Alternative

- split the same spend evenly across both exposed parts

Why rejected:

- equal sharing feels fair but protects neither risk well enough

### Likely Downstream Effects

- Production will still need to manage a constrained plan
- Finance sees intentional prioritization instead of panic buying
- Sales may need to moderate demand if the unprotected product becomes commercially urgent

## Production Manager

## Scenario A: Healthy Week With A Clear Bottleneck Priority

### Round Context

- workstation capacity is tight but stable
- `Pump` backlog is more commercially important than `Valve` backlog
- parts support `Pump` well enough for a focused plan
- WIP is under control
- support spend is not currently under extreme pressure

### Report Highlights That Matter Most

- bottleneck view shows constrained capacity should favor `Pump`
- material readiness is sufficient for a meaningful `Pump`-heavy plan
- WIP and congestion do not justify broad caution yet

### Action Chosen

- `release_product(Pump, focused_quantity)`
- `release_product(Valve, smaller_quantity)`
- `allocate_capacity(Fabrication, Pump, majority_share)`
- `allocate_capacity(Assembly, Pump, majority_share)`
- `comment("Using constrained capacity on the product with the strongest service payoff while keeping Valve flow alive at a lower level.")`

### Why This Is Reasonable

The role is protecting the plant's best near-term service outcome instead of trying to treat both products equally.

### Tradeoff Accepted

- `Valve` progress slows somewhat
- total plant value improves because constrained time is pointed at the higher-payoff mix

### Rejected Alternative

- allocate capacity evenly across both products

Why rejected:

- equal allocation would be simpler and feel balanced, but it would dilute the value of the true bottleneck and relieve neither pressure as effectively

### Likely Downstream Effects

- Sales gets better support on the more valuable backlog
- Procurement gets a clearer signal about which material stream matters most
- if `Valve` demand becomes urgent next round, mix pressure may reverse

## Scenario B: High WIP, Part Mismatch, And Budget Pressure

### Round Context

- WIP has been rising for two rounds
- one critical part is missing for part of the desired mix
- the bottleneck is still busy, but completions are not improving
- finance pressure makes extra spend harder to justify

### Report Highlights That Matter Most

- feasible output is materially below the unconstrained desired plan
- one product cannot be completed cleanly because material support is weak
- WIP accumulation shows that more releases would likely worsen congestion

### Action Chosen

- cut releases on the starved product
- focus capacity on the product that can still move cleanly
- avoid extra production support spend this turn
- `comment("Reducing releases to protect completions and avoid adding more WIP where material support is already weak.")`

### Why This Is Reasonable

The role is responding to actual flow constraints rather than trying to preserve the appearance of high activity.

### Tradeoff Accepted

- lower short-term activity
- better odds of useful completions instead of more unfinished work

### Rejected Alternative

- push both products hard and use extra spend to keep activity high

Why rejected:

- the missing part and rising WIP indicate that more activity would create congestion, not better throughput

### Likely Downstream Effects

- Procurement gets a sharper shortage signal
- Finance sees that Production is not using spend to hide a weak plan
- Sales may need to manage expectations on the deprioritized product

## Sales Manager

## Scenario A: Healthy Service, Thin Backlog

### Round Context

- finished goods are available
- customer sentiment is stable
- backlog is thin
- recent service performance has been healthy
- margin is acceptable

### Report Highlights That Matter Most

- backlog pressure is lower than ideal for keeping the plant usefully loaded
- service credibility is strong enough to support some growth
- there is no sign that the plant is already overloaded

### Action Chosen

- `set_price(Pump, slightly_lower_price)`
- `set_price(Valve, hold_price)`
- `comment("Using a targeted price move to grow demand where the plant can credibly support it, without broadly discounting both products.")`

### Why This Is Reasonable

The role is using pricing to create useful demand where service capacity appears able to absorb it. The move is selective rather than broad, which helps preserve overall price quality.

### Tradeoff Accepted

- accepts some margin pressure on one product
- aims to improve future demand quality and plant loading

### Rejected Alternative

- lower price aggressively on both products

Why rejected:

- the current state supports some growth, not a full-volume chase

### Likely Downstream Effects

- future backlog should grow where the plant is most prepared to serve it
- Finance may tolerate the move if revenue quality remains healthy
- if production support weakens unexpectedly, backlog could become fragile next round

## Scenario B: Rising Backlog, Weakening Sentiment

### Round Context

- backlog is aging
- customer sentiment has started to fall
- recent shipments have not kept pace with accepted demand
- finished goods are not keeping up with commitments

### Report Highlights That Matter Most

- the plant is no longer converting accepted demand into reliable service
- expired or at-risk backlog is becoming a credibility problem
- additional volume would likely worsen future demand quality

### Action Chosen

- `set_price(Pump, higher_price)`
- `set_price(Valve, hold_or_raise_price)`
- `comment("Protecting backlog quality and service credibility by cooling demand until the plant can serve more reliably.")`

### Why This Is Reasonable

The role is recognizing that the best commercial move is not always more demand. When service credibility is under pressure, restraint can protect future revenue quality better than discounting.

### Tradeoff Accepted

- accepts slower short-term demand growth
- aims to preserve trust and backlog quality

### Rejected Alternative

- lower price to try to recover the revenue target quickly

Why rejected:

- that would add pressure to a plant that is already failing to serve existing backlog well

### Likely Downstream Effects

- Production gets breathing room to recover service
- Finance sees better revenue-quality discipline
- if the plant recovers quickly, Sales may later return to a more growth-oriented posture

## Finance Controller

## Scenario A: Stable Cash, One Clear Waste Pattern

### Round Context

- cash is stable
- debt is well inside the ceiling
- inventory is slightly elevated in one area
- margin quality is acceptable
- one spend category looks bloated without clear throughput benefit

### Report Highlights That Matter Most

- the plant does not need survival-mode tightening
- one category deserves stricter discipline
- the rest of the plant can still support useful flow if targets stay realistic

### Action Chosen

- `set_procurement_budget(tighter_amount)`
- `set_production_budget(hold_amount)`
- `set_sales_target(hold_amount)`
- `set_debt_ceiling(hold_amount)`
- `comment("Tightening the area showing low-value buildup while preserving support for the plant's useful flow.")`

### Why This Is Reasonable

Finance is acting surgically rather than broadly. The right move is not to tighten everything equally when only one area shows a weak economic pattern.

### Tradeoff Accepted

- increases discipline in one area
- keeps the rest of the plant from being starved unnecessarily

### Rejected Alternative

- cut procurement and production budgets equally

Why rejected:

- the report shows one specific weakness, not a system-wide need for blanket austerity

### Likely Downstream Effects

- Procurement should reduce low-value overbuy
- Production retains enough support to protect useful output
- the plant learns that Finance is linking targets to economic quality, not just optics

## Scenario B: Tight Cash, Rising Inventory, Fragile Margin

### Round Context

- cash is falling
- debt is nearing the ceiling
- inventory is high
- margin quality is weakening
- the plant is at risk of protecting activity that does not translate into healthy economics

### Report Highlights That Matter Most

- liquidity protection is now urgent
- inventory is consuming flexibility
- future support must become more selective

### Action Chosen

- `set_procurement_budget(materially_tighter_amount)`
- `set_production_budget(selectively_tighter_amount)`
- `set_sales_target(more_disciplined_amount)`
- `set_debt_ceiling(conservative_amount)`
- `comment("Tightening targets to protect liquidity while still leaving room for the most defensible throughput support.")`

### Why This Is Reasonable

Finance is responding to a real survival pressure, but it is still avoiding the trap of cutting every dimension blindly. The posture becomes stricter, yet still reasons about plant viability.

### Tradeoff Accepted

- next-round flexibility is reduced
- the plant gains better odds of staying inside financial guardrails

### Rejected Alternative

- keep targets unchanged so operations can continue without tension

Why rejected:

- the visible cash, debt, and inventory trend indicates that inaction would be more dangerous than discomfort

### Likely Downstream Effects

- Procurement and Production will need sharper prioritization next round
- Sales may need to protect revenue quality rather than pure volume
- if the plant misreads which spend is still useful, throughput risk could rise

## Reuse Guidance

These examples are intended to be reused in:

- onboarding docs
- AI role-prompt tests
- TUI demo content
- contributor discussions about whether role guidance is actually decision-supportive

When extending this guide, contributors should keep each example grounded in:

- the canonical MVP action vocabulary
- visible state the role is actually allowed to know
- explicit tradeoffs rather than generic management advice
