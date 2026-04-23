# Role KPI Thresholds And Decision Triggers

This guide defines the key KPIs, threshold bands, and recommended decision triggers for each role in HerbieGo so reports can drive realistic action instead of passive observation.

Use this document when contributors need to:

- design role reports and dashboards
- connect metrics to gameplay decisions
- write gameplay playbooks or AI prompts
- define what healthy, concerning, and critical conditions look like for a role

## How To Use This Guide

Each KPI is described using:

- a practical interpretation
- a green, yellow, and red condition band
- a typical role response when the KPI enters each band
- what overreaction looks like
- what ignored warning signs look like

The exact numeric thresholds may vary by scenario balancing. For the MVP, contributors may use:

- absolute thresholds where the rules already make them clear
- relative thresholds such as low, moderate, or high pressure where balancing remains future work

Design rule:

- if no stable scenario number exists yet, the document should still define the condition bands clearly enough that a contributor knows how the role ought to react

## Band Semantics

| Band | Meaning | General Role Posture |
| --- | --- | --- |
| Green | Healthy or controlled | Stay disciplined and avoid unnecessary intervention |
| Yellow | Concerning or becoming fragile | Tighten attention, narrow decisions, and prepare to escalate selectively |
| Red | Critical, unstable, or clearly harmful | Act decisively, protect the highest-value outcome, and escalate when one role cannot solve it alone |

## MVP Role KPIs

## Procurement Manager

| KPI | Interpretation | Green Band | Yellow Band | Red Band | Typical Triggered Response |
| --- | --- | --- | --- | --- | --- |
| Days or rounds of cover by critical part | Whether visible supply is enough to protect near-term production | Coverage exceeds visible near-term need plus a modest buffer | Coverage is near the likely lead-time window with little slack | Coverage falls below visible need or one delay would create a shortage | Reorder or reprioritize toward the exposed part |
| In-transit protection | Whether existing open orders already solve the problem | Open orders cover visible risk without duplication | Existing orders cover only part of the likely need | Even with in-transit supply, next-round production remains exposed | Add orders or escalate the shortage risk |
| Stock-out risk | How likely the plant is to lose throughput because of missing parts | No visible production-critical shortage pattern | One or more parts are becoming exposed | A shortage is imminent or already affecting feasibility | Buy the most protective part first and escalate if budget blocks it |
| Purchase spend versus active target | Whether buying discipline still fits the finance posture | Spend remains inside target or only uses modest overage intentionally | Spend approaches the soft budget edge and tradeoffs become tighter | Spend would breach the hard cap or strain debt safety materially | Cut lower-value buys or escalate the strategic exception |
| Raw-material inventory exposure | Whether Procurement is solving risk by trapping too much cash | Inventory supports known flow without broad overbuild | Coverage is growing faster than visible need | Inventory is clearly accumulating beyond likely consumption | Pause or shrink lower-value buying |

Overreaction signs:

- buying broad safety stock because uncertainty feels uncomfortable
- treating every exposed part as equally urgent

Ignored warning signs:

- assuming one receipt will arrive exactly on time when there is no cushion left
- continuing to buy because unit cost looks attractive even while inventory exposure is rising

Cross-role dependency notes:

- yellow or red stock-out risk should immediately matter to Production
- red inventory exposure should immediately matter to Finance

## Production Manager

| KPI | Interpretation | Green Band | Yellow Band | Red Band | Typical Triggered Response |
| --- | --- | --- | --- | --- | --- |
| Feasible output versus requested output | Whether the plan matches visible material and capacity reality | Requested plan is close to what the plant can actually advance | Requested plan relies on optimistic assumptions or leaves little slack | Requested plan materially exceeds visible feasibility | Cut releases and prioritize the highest-value feasible mix |
| Bottleneck utilization on useful work | Whether the true constraint is protected | The bottleneck is occupied with the most valuable feasible work | The bottleneck is busy, but mix quality is debatable | The bottleneck is starved, blocked, or consumed by low-value work | Reallocate capacity toward the best system payoff |
| WIP accumulation | Whether activity is turning into congestion | WIP stays proportional to downstream capacity and completions | WIP is growing faster than completions | WIP is clearly piling up without service improvement | Reduce releases and protect downstream completion |
| Part-starvation risk | Whether missing material undermines the intended plan | Parts visibly support the intended mix | One missing part could force a mix change | The plan is already partly unsupported by available parts | Change the mix and escalate to Procurement |
| Production spend pressure | Whether extra operating spend is protecting flow or only activity | Extra spend is selective and clearly useful | Support spend is rising without clear proof of value | Spend is being used broadly to compensate for a weak plan | Reserve spend for bottleneck-protective actions only |

Overreaction signs:

- using overtime or support spend to chase activity instead of output
- keeping every resource busy even when the system constraint is elsewhere

Ignored warning signs:

- treating a growing WIP pile as evidence of productivity
- continuing to release work despite obvious part mismatch

Cross-role dependency notes:

- red part-starvation risk should be shared immediately with Procurement
- red spend pressure should be discussed with Finance before it becomes false support

## Sales Manager

| KPI | Interpretation | Green Band | Yellow Band | Red Band | Typical Triggered Response |
| --- | --- | --- | --- | --- | --- |
| Revenue versus target | Whether Sales is producing meaningful top-line performance | Revenue is healthy without obvious service damage | Revenue is soft or improving only at questionable price quality | Revenue is materially weak or only available through harmful backlog growth or discounting | Reconsider pricing posture and protect revenue quality |
| Backlog pressure | Whether accepted demand is still manageable and healthy | Backlog keeps the plant busy without clear aging risk | Backlog is growing faster or becoming fragile | Backlog is aging, overloaded, or visibly beyond plant support | Slow demand growth, raise price, or narrow pursuit |
| Customer sentiment | Whether the market still trusts the plant's service | Sentiment is stable or recovering | Sentiment is slipping or becoming uneven | Sentiment is clearly deteriorating because service credibility is weak | Protect service credibility over volume |
| Average selling price and margin quality | Whether demand is being won at a reasonable economic quality | Price supports healthy demand and margin | Price pressure is growing and margin discipline is weaker | Volume is being won mainly through harmful discounting | Raise price or stop chasing weak volume |
| Lost-sales or backlog-expiry risk | Whether orders are decaying before shipment | Backlog mostly converts before expiry | Aging orders are increasing | Expired backlog is becoming common and harming future demand | Stop amplifying demand and escalate the plant constraint |

Overreaction signs:

- raising price too aggressively when backlog is thin and service is healthy
- protecting only volume and ignoring demand quality

Ignored warning signs:

- treating rising backlog as automatically good news
- discounting into weak service and falling sentiment

Cross-role dependency notes:

- red backlog pressure should matter immediately to Production
- red margin-quality problems should matter immediately to Finance

## Finance Controller

| KPI | Interpretation | Green Band | Yellow Band | Red Band | Typical Triggered Response |
| --- | --- | --- | --- | --- | --- |
| Ending cash position | Whether liquidity remains comfortably above risk | Cash is stable with routine flexibility | Cash is tightening and room for error is smaller | Cash is near the floor or deteriorating fast | Tighten weaker spend while protecting the best flow |
| Debt versus debt ceiling | Whether the plant is approaching a hard financial wall | Debt remains comfortably inside the ceiling | Debt headroom is narrowing | Debt pressure is close to the cap or likely to force trims | Raise discipline and escalate where flow still needs support |
| Gross margin signal | Whether revenue quality and operating support are economically sound | Margin is stable relative to recent plant behavior | Margin is weakening or increasingly mixed | Margin erosion is material even if revenue still looks healthy | Push against weak pricing, bad mix, or low-value buildup |
| Inventory exposure | Whether cash is trapped in raw material, WIP, or finished goods | Inventory supports useful flow without obvious overbuild | Inventory is growing faster than healthy throughput or service benefit | Inventory is clearly becoming a cash trap | Push against overbuy or overproduction |
| Budget realism | Whether targets are strict but still executable | Roles can operate inside the target with selective tension | Targets are pushing strain that may still be manageable | Targets are likely to force predictable failure or hard trims | Reset the targets to something disciplined but credible |

Overreaction signs:

- cutting all categories equally because that feels fair
- treating every spend increase as equally dangerous

Ignored warning signs:

- accepting weak revenue quality because top-line revenue still looks acceptable
- leaving targets unchanged while cash and inventory trends deteriorate together

Cross-role dependency notes:

- red inventory exposure should feed back to Procurement and Production
- red margin weakness should feed back to Sales and Production mix decisions

## Future-Role KPI Guidance

These KPIs are design guidance for future-role documentation and expansion work. They are not all implemented as engine-backed metrics in the MVP.

## Quality Manager (Future Role)

| KPI | Interpretation | Green Band | Yellow Band | Red Band | Typical Triggered Response |
| --- | --- | --- | --- | --- | --- |
| First-pass yield and defect trend | Whether the process is producing stable output | Yield is stable and defect trend is controlled | Defects are rising or localized instability is emerging | Defects are worsening or escaping containment | Tighten inspection or escalate root-cause work |
| Customer escape signal | Whether defects are reaching customers | Complaints and RMAs are limited | Customer-facing failure is appearing intermittently | Customer harm is recurring or clearly rising | Contain suspect output and protect customers |
| Supplier incoming-quality risk | Whether quality failure is entering through supply | Incoming quality is stable | Supplier issues are recurring but still containable | Supplier quality is materially disrupting flow or customer outcomes | Escalate with Procurement and tighten receiving control |
| Corrective-action backlog | Whether the system is learning or only firefighting | Corrective actions are current and prioritized | Overdue actions are accumulating | High-risk actions remain open while defects repeat | Force root-cause prioritization |

Overreaction signs:

- blocking broad production without evidence of broad risk

Ignored warning signs:

- waiting for customer failures before acting on worsening internal signals

## Logistics And Warehouse Manager (Future Role)

| KPI | Interpretation | Green Band | Yellow Band | Red Band | Typical Triggered Response |
| --- | --- | --- | --- | --- | --- |
| Warehouse occupancy and overflow risk | Whether storage is still supporting usable flow | Space is healthy and movement is reliable | Congestion is building or overflow risk is increasing | Occupancy is choking movement or creating service risk | Clear obstructive inventory or escalate the space bottleneck |
| Shipping reliability | Whether outbound commitments are moving credibly | Shipments are on time with limited firefighting | Delays or expedite pressure are rising | Service failures are being driven by outbound instability | Reprioritize shipments and escalate the physical-flow risk |
| Inbound congestion | Whether receiving is still workable | Inbound flow lands cleanly | Receipts are causing noticeable staging strain | Inbound pressure is materially disrupting the warehouse | Stagger, reprioritize, or escalate inbound flow overload |
| Premium-freight pressure | Whether expediting is selective or normalized | Expedites are rare and justified | Expedites are becoming frequent | Premium freight is the default way to maintain service | Surface the root coordination failure and restrict expedites |

Overreaction signs:

- expediting broadly instead of choosing high-value service risks

Ignored warning signs:

- treating occupancy as cosmetic until it affects shipping directly

## Maintenance Manager (Future Role)

| KPI | Interpretation | Green Band | Yellow Band | Red Band | Typical Triggered Response |
| --- | --- | --- | --- | --- | --- |
| Unplanned downtime trend | Whether asset reliability is stable | Downtime is controlled and predictable | Downtime is rising or concentrated in one area | Downtime is materially threatening throughput | Focus maintenance on the highest-impact asset |
| PM completion versus backlog | Whether the plant is protecting future uptime | Preventive work is current enough to sustain reliability | PM backlog is growing | PM deferral is clearly undermining reliability | Protect critical PMs and challenge deferral |
| Bad-actor asset recurrence | Whether the same asset is failing repeatedly | No chronic asset is dominating failure time | One asset is becoming noticeably fragile | A repeat bad actor is consuming disproportionate maintenance effort | Escalate root-cause or repair-versus-replace action |
| Critical spare readiness | Whether repair work can actually be completed | Critical spares are ready | Some spare exposure is emerging | One missing spare would create prolonged downtime | Escalate spare coverage before failure happens |

Overreaction signs:

- stopping healthy throughput for low-value maintenance work

Ignored warning signs:

- celebrating fast reactive repairs while the same asset keeps failing

## Plant Manager (Future Role)

| KPI | Interpretation | Green Band | Yellow Band | Red Band | Typical Triggered Response |
| --- | --- | --- | --- | --- | --- |
| Plant-wide bottleneck stability | Whether the main constraint is understood and protected | The true constraint is stable and being managed | Constraint pressure is shifting or contested | The plant is reacting to symptoms instead of the real bottleneck | Re-center the plant on the real system constraint |
| Cross-role conflict load | Whether local negotiations are still working | Role tensions are productive and mostly self-resolving | Friction is rising and repeated tradeoffs are harder | Local optimization is repeatedly harming total plant outcomes | Escalate to plant-level prioritization |
| Service and profitability balance | Whether the plant is protecting both output value and operating health | Service and economics are broadly aligned | One side is weakening and tradeoffs are becoming sharper | The plant is protecting one side by materially damaging the other | Force explicit plant-wide prioritization |
| Coordination responsiveness | Whether the plant can still act coherently under pressure | Cross-role adjustments happen without heavy intervention | Delays or ambiguity are appearing | Repeated conflicts persist without resolution | Issue a directive or coordinate an escalation |

Overreaction signs:

- micromanaging local decisions that roles can still handle themselves

Ignored warning signs:

- letting unresolved cross-role friction continue because no one wants to choose the tradeoff

## Guidance For Reports, Playbooks, And Prompts

When contributors reuse this guide:

- reports should show the KPI, its current band, and the decision the band is meant to support
- gameplay playbooks should explain how the role should respond when a KPI moves from green to yellow or red
- AI prompts should include the role's most important trigger logic but avoid hardcoding scenario numbers that are not yet canonically balanced
- future-role docs should label clearly which KPI thresholds are design guidance rather than current engine outputs

## Clarification Rule

If a contributor needs a precise numeric threshold and the scenario data does not yet define one, they should:

1. keep the qualitative green, yellow, and red guidance from this document
2. avoid inventing false numeric precision in player-facing rules
3. note the missing balancing requirement in implementation work if exact values become necessary

That prevents the documentation from pretending the balancing problem is already solved when it is not.
