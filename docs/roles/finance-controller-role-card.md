# Finance Controller Role Card

## Document Metadata

- Role name: `Finance Controller`
- Status: `MVP`
- Canonical ID: `finance_controller`
- Last updated: `2026-04-22`
- Related issues: `#94`, `#95`, `#103`

## I. Identity And Objectives (The "Who")

### Description And Core Goal

The Finance Controller protects the plant's short-term financial health by setting next-round budgets and operating targets. In strong play, this role preserves liquidity and spending discipline without starving the plant of the material and capacity support it needs to remain profitable.

This role's natural local bias is to treat lower spend, tighter budgets, and cleaner short-term financials as success even when those choices weaken throughput, service reliability, or future profit.

### Success And Failure Criteria

Success signals:

- cash and debt stay inside tolerable ranges without repeated emergency intervention
- next-round targets discipline waste without making plant execution impossible
- spending support is directed toward flow that protects profitable output
- Finance explains tradeoffs clearly instead of treating all spending as equally bad

Failure signals:

- budgets are set so tight that Procurement or Production cannot support credible plant flow
- finance targets protect optics while backlog, shortages, or lost sales worsen
- cash risk is ignored until the plant is already in distress
- repeated cost cutting damages throughput more than it improves financial health

Key tradeoff:

- local finance success can look good on short-term cost control while still hurting plant-wide throughput, service, and long-run profitability

### Key Performance Indicators (KPIs)

| KPI | Why It Matters | Healthy Signal | Warning Signal | Typical Decision Trigger |
| --- | --- | --- | --- | --- |
| Ending cash position | Shows whether the plant is preserving enough liquidity. | Cash remains above the desired floor with room for routine operation. | Cash approaches the floor or trends down too quickly. | Tighten targets, defer weaker spending, or escalate collections and margin pressure. |
| Debt versus debt ceiling | Measures how close the plant is to its hard financial guardrail. | Debt remains well inside the allowed limit. | Debt pressure approaches the ceiling or removes flexibility. | Tighten procurement or production support selectively. |
| Gross margin signal | Connects revenue quality to operating choices. | Margin remains healthy relative to recent plant performance. | Revenue is rising but margin quality is deteriorating. | Reevaluate pricing, material spend, or output mix support. |
| Inventory exposure | Cash in inventory can become a hidden financial trap. | Inventory supports flow without obvious overbuild. | Raw material or finished goods exposure rises without matching value creation. | Discourage overbuy or overproduction. |
| Budget realism | Finance targets must be challenging but still executable. | Other roles can work within targets with only selective tension. | Budgets are so tight they force predictable failure or illegal trimming. | Reset targets to something strategically strict but feasible. |

## II. Operational Domain (The "What")

### Responsibilities

- set next-round budgets for procurement and production support spending
- set next-round revenue and cash or debt targets
- interpret cash, debt, margin, and inventory pressure
- surface financial tradeoffs that other roles may underweight

### Decision Levers

| Lever | MVP Or Future | What The Player Changes | Immediate Tradeoff | Likely Downstream Effect |
| --- | --- | --- | --- | --- |
| Procurement budget target | `MVP` | Sets next-round spending guidance for material buying. | Lower spend protects cash but raises shortage risk. | Changes future material flexibility and cash exposure. |
| Production spend budget target | `MVP` | Sets next-round guidance for overtime or capacity-related spend. | Tighter control lowers cost but can restrict throughput support. | Changes how much production flexibility exists next round. |
| Revenue target | `MVP` | Signals how much commercial performance Finance expects. | Higher target encourages growth but may stress the plant. | Changes tension with Sales and output planning. |
| Cash floor or debt ceiling target | `MVP` | Sets the financial guardrail for future actions. | More protection reduces financial risk but limits operating freedom. | Changes how aggressively the plant can spend through strain. |
| Capital structure and financing tools | `Future Role` | Uses richer instruments beyond the MVP's simple debt model. | More financial flexibility versus more complexity. | Expands finance strategy after MVP. |

### Constraints

- Finance targets apply to the next round, not retroactively to the current one
- active budgets are soft targets with a `110%` hard trim rule enforced by the plant
- Finance does not directly control procurement orders, production releases, or sales prices
- the role must reason from visible plant state, not hidden current-turn actions
- over-tightening can create operational failure that looks like savings at first

## III. Ecosystem (The "How")

### Synergies

- Procurement Manager: better financial framing helps distinguish strategic buying from careless overbuy
- Production Manager: throughput-aware cost discipline helps support the right output rather than generic activity
- Sales Manager: stronger revenue quality and realistic growth pressure improve the usefulness of finance targets

Shared information that matters most:

- cash and debt trends
- backlog and service pressure
- inventory exposure
- recent margin and spend patterns

### Conflicts

- Procurement Manager: Procurement wants more safety stock while Finance wants tighter cash control
- Production Manager: Production may need spend support while Finance resists avoidable cost growth
- Sales Manager: Sales may push for demand and price moves that look risky on cash or margin grounds

Escalation guidance:

- negotiate when the disagreement is about normal risk tolerance or timing
- escalate when current financial guardrails would clearly force operational failure or when operating pressure would clearly breach financial survival limits

### Reports And Data

| Information | Visibility | Why The Role Needs It | Shared Before Reveal? | Shared After Resolution? |
| --- | --- | --- | --- | --- |
| Cash and debt position | Plant-wide | Core view of financial survivability. | Yes | Yes |
| Inventory exposure | Plant-wide | Shows where cash is tied up without immediate return. | Yes | Yes |
| Revenue, backlog, and service signals | Plant-wide | Helps Finance judge whether to support growth or apply restraint. | Yes | Yes |
| Recent spending and cost signals | Plant-wide | Shows whether the plant's financial posture is improving or degrading. | Yes | Yes |
| Finance target rationale | Role-specific interpretation | Explains why a stricter or looser target is strategically justified. | Yes | Yes |
| Current-turn finance target submission | Hidden current-turn action | Must stay hidden until reveal. | No | Yes |

## IV. Dynamic Variables (The "What If")

### Opportunities

- supporting selective spend when the plant can convert it into profitable flow
- tightening targets before a cash problem becomes acute
- using financial discipline to reduce unproductive inventory or low-value effort

### Events

Common shocks for this role:

- cash falls faster than expected
- debt approaches the ceiling
- margin quality deteriorates even while revenue looks healthy
- inventory grows without translating into useful throughput or service

Reasonable responses:

- tighten one target while deliberately protecting the most valuable flow
- support spending that is clearly throughput-positive
- push Sales toward better revenue quality instead of just more demand
- push Procurement or Production away from overbuild and low-value spend

Overreaction examples:

- treating all spending as equally harmful
- forcing budgets so low that the plant cannot sustain credible operation
- protecting short-term cash while allowing larger lost-profit problems to grow

## V. Additional Aspects To Consider

### Information Asymmetry

The Finance Controller usually sees liquidity pressure, debt risk, and the cumulative cost of bad local decisions more clearly than the rest of the plant. This role is often first to notice that apparent operational success is not translating into durable financial health.

That privileged view does not include hidden current-turn actions from Procurement, Production, or Sales.

### Resource Ownership

- next-round spending targets
- debt tolerance and cash protection posture
- financial risk framing for the whole plant
- pressure on inventory and margin quality

### Risk Profile

| Risk | Earliest Warning Sign | What Happens If Ignored | Typical Mitigation |
| --- | --- | --- | --- |
| Liquidity squeeze | Cash trends toward the floor faster than expected. | The plant loses flexibility and routine decisions become survival decisions. | Tighten weaker spend and protect the highest-value flow. |
| Debt-ceiling stress | Debt rises near the cap while spending pressure remains high. | Procurement or production actions may be trimmed at the worst moment. | Reset targets, reduce lower-value commitments, or escalate. |
| False savings from over-cutting | Spend falls while service, throughput, or shortages worsen. | Short-term optics improve but total performance degrades. | Reintroduce selective support where it protects profitable flow. |
| Inventory-heavy cash trap | Money is tied up in stock that is not relieving the right constraint. | Cash weakens without meaningful operational payoff. | Push against overbuy and overproduction. |

## VI. Relationships

| Other Role | Primary Synergy | Primary Conflict | Information To Share | When To Escalate |
| --- | --- | --- | --- | --- |
| Procurement Manager | Better budget framing helps Procurement buy what matters most. | Procurement may seek more coverage than cash can safely support. | Cash pressure, budget intent, and the cost of overbuy versus shortage. | When material protection requires a deliberate exception to normal financial discipline. |
| Production Manager | Throughput-aware budgets help support the output that matters most. | Production may want spend support that looks expensive in the short term. | Budget logic, cash limits, and the difference between useful and wasteful spend. | When tighter control would clearly choke the bottleneck or critical service. |
| Sales Manager | Better revenue quality improves the whole plant's financial position. | Sales may pursue demand that raises risk without enough margin or service credibility. | Revenue target intent, cash implications, and margin tradeoffs. | When commercial pressure would materially worsen cash, margin, or service stability. |

## Notes For Role Briefings And Playbooks

For a short role briefing, keep the focus on mission, legal actions, and the local bias toward short-term cost and cash discipline.

For a longer gameplay playbook, expand:

- how to read the finance report in order
- when to tighten versus deliberately support spending
- how to distinguish productive investment from low-value cost growth
- how to avoid starving the plant while still protecting financial survival
