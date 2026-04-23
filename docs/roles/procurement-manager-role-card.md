# Procurement Manager Role Card

## Document Metadata

- Role name: `Procurement Manager`
- Status: `MVP`
- Canonical ID: `procurement_manager`
- Last updated: `2026-04-22`
- Related issues: `#91`, `#96`, `#103`

## I. Identity And Objectives (The "Who")

### Description And Core Goal

The Procurement Manager protects the plant from material shortages while trying to keep input spending disciplined. In strong play, this role keeps future production fed without trapping too much cash in early or oversized purchases.

This role's natural local bias is to treat low unit cost and high material coverage as success even when those choices increase inventory, tie up cash, or crowd the plant with the wrong parts.

### Success And Failure Criteria

Success signals:

- production is rarely starved by missing purchased parts
- critical parts arrive before they block the bottleneck
- input spending stays within or near finance targets without emergency firefighting
- supplier choices improve reliability without causing avoidable overbuy

Failure signals:

- stock-outs force idle production time or missed shipments
- large purchases protect one risk while creating damaging cash pressure elsewhere
- the plant accumulates slow-moving or excessive raw-material inventory
- procurement reacts too late and repeatedly pays premium prices to recover

Key tradeoff:

- local procurement success can look good on cost-per-unit or coverage while still hurting plant-wide cash, inventory, or flow

### Key Performance Indicators (KPIs)

| KPI | Why It Matters | Healthy Signal | Warning Signal | Typical Decision Trigger |
| --- | --- | --- | --- | --- |
| Days of cover by critical part | Shows whether production can keep running through the next rounds. | Coverage exceeds visible near-term need plus a small buffer. | Coverage falls below the known lead-time window. | Reorder, expedite, or shift buying priority. |
| Stock-out count | Measures whether Procurement is protecting flow at all. | Zero stock-outs on production-critical parts. | Any repeated or expanding shortage pattern. | Panic buy, escalate, or rebalance purchase priorities. |
| In-transit coverage | Shows whether already ordered supply is enough to protect next-round production. | Open orders cover near-term need without duplication. | Critical parts are still exposed even after in-transit orders. | Add orders or change order timing. |
| Purchase spend versus active target | Keeps procurement choices grounded in finance pressure. | Spend remains inside active targets or uses overruns intentionally. | Replenishment plan would materially exceed the active budget. | Reduce order quantity, phase buys, or escalate tradeoff. |
| Raw-material inventory exposure | Reveals when Procurement is solving shortages by overbuying. | Inventory supports flow without long aging tails. | Excess stock grows while demand or output stays constrained. | Pause buys, redirect cash, or accept higher risk selectively. |

## II. Operational Domain (The "What")

### Responsibilities

- buy the purchased parts the plant needs for future production
- protect against shortages before they shut down flow
- interpret supply risk, lead-time risk, and cash tradeoffs
- explain procurement choices to the rest of the plant through clear rationale

### Decision Levers

| Lever | MVP Or Future | What The Player Changes | Immediate Tradeoff | Likely Downstream Effect |
| --- | --- | --- | --- | --- |
| Place purchase orders by part | `MVP` | Chooses which parts to buy this round. | Spending cash now to protect future flow. | Changes future part availability and debt pressure. |
| Set order quantity by part | `MVP` | Chooses how much of each part to buy. | More coverage versus more inventory and cash exposure. | Can prevent shortages or create overstock. |
| Prioritize scarce budget across parts | `MVP` | Concentrates spend on the most important shortages. | Some parts remain exposed. | Protects the bottleneck while accepting selective risk elsewhere. |
| Supplier choice and sourcing strategy | `Future Role` | Chooses among suppliers with different cost and reliability profiles. | Lower price may mean higher delay or quality risk. | Changes lead-time stability, quality, and total supply risk. |
| Bulk-buy strategy | `Future Role` | Buys ahead intentionally to exploit discounts or hedge risk. | Ties up cash and raises inventory exposure. | Can lower cost or worsen plant-wide cash pressure. |

### Constraints

- finance sets next-round procurement budgets and debt tolerance
- purchased parts arrive after lead time rather than instantly
- current-turn actions are hidden until round resolution
- the role cannot consume parts directly or force production sequencing
- overbuying can protect local supply while damaging cash and inventory health

## III. Ecosystem (The "How")

### Synergies

- Production Manager: alignment reduces idle time and keeps bottlenecks fed
- Sales Manager: shared demand visibility helps avoid both shortages and useless inventory
- Finance Controller: realistic budget guidance helps Procurement separate justified protection buys from careless cash burn

Shared information that matters most:

- visible demand and backlog pressure
- part coverage and in-transit supply
- active budgets, cash pressure, and debt tolerance

### Conflicts

- Finance Controller: Procurement wants more safety stock while Finance wants tighter cash and lower spend
- Production Manager: Procurement may prefer cheaper, slower, or larger buys while Production needs reliable near-term material flow
- Sales Manager: aggressive demand growth can create supply promises Procurement cannot safely support

Escalation guidance:

- negotiate when the disagreement is about normal timing, quantity, or acceptable buffer
- escalate when the plant faces likely stock-out, debt stress, or repeated promise-versus-supply mismatch

### Reports And Data

| Information | Visibility | Why The Role Needs It | Shared Before Reveal? | Shared After Resolution? |
| --- | --- | --- | --- | --- |
| Parts inventory by part | Plant-wide | Shows what can support near-term production. | Yes | Yes |
| In-transit purchase orders | Plant-wide | Shows which shortages are already covered. | Yes | Yes |
| Active finance targets | Plant-wide | Defines current spending pressure and debt guardrails. | Yes | Yes |
| Backlog and demand pressure | Plant-wide | Helps judge which parts protect the most important output. | Yes | Yes |
| Procurement shortage commentary | Role-specific interpretation | Explains why one part is riskier than another. | Yes | Yes |
| Current-turn purchase order intent | Hidden current-turn action | Must stay hidden until reveal. | No | Yes |

## IV. Dynamic Variables (The "What If")

### Opportunities

- buying ahead before a visible shortage becomes urgent
- concentrating limited budget on the parts that protect the bottleneck
- using a selective bulk buy when cash is healthy and demand is credible

### Events

Common shocks for this role:

- demand spike that outpaces current part coverage
- supplier delay or missed receipt
- sudden cash squeeze or tighter finance target
- chronic overbuy pattern that exposes the plant to inventory drag

Reasonable responses:

- reorder earlier or higher for critical parts
- accept selective shortages on lower-value items
- pause noncritical buys to protect cash
- escalate when supply risk and finance limits cannot both be satisfied

Overreaction examples:

- panic buying everything at once
- treating every shortage as equally urgent
- protecting unit cost while ignoring near-term flow risk

## V. Additional Aspects To Consider

### Information Asymmetry

The Procurement Manager usually sees supply fragility earlier and more clearly than the rest of the plant. This role is often first to notice that on-hand stock, in-transit supply, and lead-time assumptions no longer support visible demand.

That privileged view does not include hidden current-turn actions from other players.

### Resource Ownership

- procurement spend
- future parts availability
- exposure to raw-material inventory
- part-specific supply risk

### Risk Profile

| Risk | Earliest Warning Sign | What Happens If Ignored | Typical Mitigation |
| --- | --- | --- | --- |
| Critical part shortage | Days of cover fall below visible need. | Production idles and backlog grows. | Reorder, reprioritize spend, or escalate. |
| Cash-consuming overbuy | Inventory grows faster than credible consumption. | Finance pressure rises and flexibility falls. | Delay buys, reduce buffers, or phase orders. |
| False confidence from in-transit supply | One delayed receipt removes all coverage. | The plant discovers the shortage too late to respond cheaply. | Maintain buffer or diversify future sourcing. |
| Cheap but unreliable supply choice | Unit cost looks good while service gets less stable. | Production firefights and premium recovery spend grows. | Accept higher cost for reliability when risk is material. |

## VI. Relationships

| Other Role | Primary Synergy | Primary Conflict | Information To Share | When To Escalate |
| --- | --- | --- | --- | --- |
| Production Manager | Protect the bottleneck with the right parts at the right time. | Production wants certainty while Procurement may chase cheaper or slower supply. | Critical shortages, expected arrivals, and part-priority tradeoffs. | When shortages will materially change the feasible production plan. |
| Sales Manager | Demand visibility helps Procurement buy what the plant is most likely to need. | Sales can create demand pressure that exceeds credible supply coverage. | Backlog pressure, likely short parts, and product-risk implications. | When customer commitments depend on parts that cannot be covered safely. |
| Finance Controller | Better cash guidance helps Procurement separate strategic buys from reckless buying. | Finance may push tighter spend than supply protection safely allows. | Spend forecast, shortage cost, and inventory exposure. | When protecting flow requires exceeding a normal spending posture. |

## Notes For Role Briefings And Playbooks

For a short role briefing, keep the focus on mission, legal actions, and the local bias toward larger or cheaper buys.

For a longer gameplay playbook, expand:

- how to read the procurement report in order
- when to accept shortage risk instead of spending more
- when a bulk buy is smart versus dangerous
- how to negotiate with Production, Sales, and Finance when supply and cash cannot both be optimized
