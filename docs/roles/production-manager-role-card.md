# Production Manager Role Card

## Document Metadata

- Role name: `Production Manager`
- Status: `MVP`
- Canonical ID: `production_manager`
- Last updated: `2026-04-22`
- Related issues: `#92`, `#97`, `#103`

## I. Identity And Objectives (The "Who")

### Description And Core Goal

The Production Manager turns available parts and finite workstation capacity into finished goods. In strong play, this role protects plant throughput by making realistic release and capacity choices that move the right products through the bottleneck at the right time.

This role's natural local bias is to treat high utilization and high local output as success even when extra work-in-progress, overproduction, or low-value releases make the overall plant less effective.

### Success And Failure Criteria

Success signals:

- the bottleneck is kept working on useful output rather than starved or clogged work
- finished goods are produced where they relieve the most service pressure
- work-in-progress stays controlled instead of piling up without downstream capacity
- production spend is used intentionally rather than as a reflexive response to every shortfall

Failure signals:

- releases exceed realistic parts or capacity constraints
- work-in-progress accumulates while finished output fails to improve
- the plant protects utilization while missing the most important shipments
- overtime or extra capacity spend rises without a clear throughput payoff

Key tradeoff:

- local production success can look strong on activity and utilization while still hurting plant-wide inventory, cash, or service performance

### Key Performance Indicators (KPIs)

| KPI | Why It Matters | Healthy Signal | Warning Signal | Typical Decision Trigger |
| --- | --- | --- | --- | --- |
| Feasible output versus target | Shows whether the plan matches the plant's real constraints. | Requested output is close to what parts and capacity can actually support. | Target volume materially exceeds visible feasibility. | Reduce releases, change priorities, or escalate constraints. |
| Bottleneck utilization on useful work | Measures whether the plant's main constraint is protected. | The bottleneck is busy on the highest-value available work. | The bottleneck is starved, blocked, or consumed by low-value mix. | Reallocate capacity or shift product priority. |
| Work-in-progress accumulation | Reveals when Production is creating congestion instead of flow. | WIP remains controlled relative to downstream capacity. | WIP grows round after round without corresponding completions. | Tighten releases or protect downstream capacity. |
| Part-starvation risk | Production decisions fail if parts are missing. | Critical parts cover the intended near-term mix. | One missing part blocks planned output. | Cut releases, change the mix, or escalate to Procurement. |
| Production spend pressure | Connects output choices to finance guardrails. | Extra spend is selective and linked to meaningful service or throughput gain. | Overtime or added capacity spend rises without clear benefit. | Reduce low-value output or justify support spending explicitly. |

## II. Operational Domain (The "What")

### Responsibilities

- convert available parts inventory into finished goods
- allocate finite workstation capacity between products
- interpret where the plant is constrained and where flow is getting stuck
- explain production tradeoffs clearly when throughput, cost, and service goals conflict

### Decision Levers

| Lever | MVP Or Future | What The Player Changes | Immediate Tradeoff | Likely Downstream Effect |
| --- | --- | --- | --- | --- |
| Release quantity by product | `MVP` | Chooses how many units of each product to push into production. | More release can protect output or create excess WIP. | Changes parts consumption, WIP levels, and finished output. |
| Capacity allocation by product | `MVP` | Chooses how workstation time is divided between products. | Prioritizing one product delays another. | Changes which backlog pressure is relieved first. |
| Bottleneck protection | `MVP` | Keeps the most constrained workstation focused on the highest-value work. | Some lower-priority work waits longer. | Improves throughput and reduces wasted effort. |
| Overtime or added operating spend posture | `MVP` | Decides when extra spend is worth the output benefit. | More spend now versus more service risk later. | Can protect shipments or simply increase cost if used poorly. |
| Detailed sequencing and setup strategy | `Future Role` | Optimizes line order, changeovers, and dispatching. | Lower flexibility versus lower lost time. | Changes effective capacity once detailed scheduling exists. |

### Constraints

- production can only consume parts that already exist in inventory
- workstation capacity is finite each round
- active finance targets create soft spend guardrails with a hard trim point
- current-turn actions remain hidden until resolution
- the role does not control customer demand, procurement orders, or next-round finance targets directly

## III. Ecosystem (The "How")

### Synergies

- Procurement Manager: good coordination keeps the bottleneck fed with the right parts
- Sales Manager: shared backlog and service priorities help Production focus on the most valuable output
- Finance Controller: clear throughput reasoning helps distinguish productive support spending from waste

Shared information that matters most:

- visible parts availability
- visible backlog and service risk
- workstation capacity and WIP pressure
- active budgets and operating targets

### Conflicts

- Procurement Manager: Production wants reliable near-term material flow while Procurement may favor cheaper or slower supply
- Sales Manager: Sales wants more fulfilled demand while Production must respect physical and material limits
- Finance Controller: Production may need extra spend or buffer while Finance pushes tighter control

Escalation guidance:

- negotiate when the disagreement is about mix, timing, or acceptable short-term tradeoffs
- escalate when feasible output cannot support committed demand or when cost control would clearly starve the bottleneck

### Reports And Data

| Information | Visibility | Why The Role Needs It | Shared Before Reveal? | Shared After Resolution? |
| --- | --- | --- | --- | --- |
| Parts inventory by part | Plant-wide | Determines what can legally be released. | Yes | Yes |
| WIP by product and stage | Plant-wide | Shows where work is stuck or advancing. | Yes | Yes |
| Finished goods inventory | Plant-wide | Helps judge whether more output is urgently needed. | Yes | Yes |
| Workstation capacities | Plant-wide | Defines the round's physical production limit. | Yes | Yes |
| Active finance targets | Plant-wide | Shows when extra spend is likely to trigger finance tension. | Yes | Yes |
| Production bottleneck assessment | Role-specific interpretation | Helps Production explain why one mix is better than another. | Yes | Yes |
| Current-turn production intent | Hidden current-turn action | Must stay hidden until reveal. | No | Yes |

## IV. Dynamic Variables (The "What If")

### Opportunities

- shifting the mix toward the product that best uses the bottleneck
- reducing low-value releases that only create more congestion
- using selective overtime to protect meaningful throughput or delivery reliability

### Events

Common shocks for this role:

- a critical part shortage cuts the feasible plan
- one product's backlog becomes much more urgent than the other
- the bottleneck shifts because one workstation becomes much tighter
- finance pressure makes support spending harder to justify

Reasonable responses:

- shrink releases to the legal, high-value plan
- move capacity toward the most important output
- escalate shortages or impossible service expectations early
- accept lower local utilization if it protects better total flow

Overreaction examples:

- pushing more units simply to keep resources busy
- creating WIP everywhere instead of protecting the bottleneck
- using overtime to hide a bad mix decision

## V. Additional Aspects To Consider

### Information Asymmetry

The Production Manager usually sees flow disruption and congestion earlier than the rest of the plant. This role is often first to notice when the problem is not absolute demand, but where limited capacity and material availability are colliding.

That privileged view does not include hidden current-turn actions from Procurement, Sales, or Finance.

### Resource Ownership

- workstation capacity allocation
- release quantities by product
- exposure to work-in-progress congestion
- production operating spend posture

### Risk Profile

| Risk | Earliest Warning Sign | What Happens If Ignored | Typical Mitigation |
| --- | --- | --- | --- |
| Bottleneck misuse | The main constrained workstation is serving the wrong mix or sitting starved. | Total throughput stays low even though effort stays high. | Reallocate capacity and protect the best flow. |
| WIP congestion | Released work rises faster than completions. | The floor clogs, visibility drops, and finished output stalls. | Tighten releases and focus downstream completion. |
| Part-starved production plan | One missing part undermines planned output. | The plan becomes partly illegal or yields much less than expected. | Reduce releases or escalate the shortage quickly. |
| Costly output chasing | Overtime or support spend rises without real service improvement. | Finance pressure grows while plant performance barely changes. | Reserve extra spend for meaningful throughput or delivery gains. |

## VI. Relationships

| Other Role | Primary Synergy | Primary Conflict | Information To Share | When To Escalate |
| --- | --- | --- | --- | --- |
| Procurement Manager | Reliable parts flow keeps the bottleneck productive. | Procurement may choose cost or timing that leaves Production exposed. | Critical part shortages, true near-term needs, and high-risk mixes. | When missing parts materially change the feasible production plan. |
| Sales Manager | Clear service priorities help Production choose the right output mix. | Sales may want more than the plant can legally or profitably make. | Feasible output, constrained products, and likely backlog relief. | When customer promises exceed visible production feasibility. |
| Finance Controller | Throughput-aware spending logic helps Production justify selective support spending. | Finance may push targets that protect cost but starve useful output. | Spend tradeoffs, bottleneck impact, and the cost of under-supporting flow. | When cost discipline would clearly damage throughput or service resilience. |

## Notes For Role Briefings And Playbooks

For a short role briefing, keep the focus on mission, legal actions, and the local bias toward utilization and output.

For a longer gameplay playbook, expand:

- how to read the production report in order
- how to distinguish feasible output from desired output
- when to protect the bottleneck instead of chasing total activity
- when overtime is strategically justified versus wasteful
