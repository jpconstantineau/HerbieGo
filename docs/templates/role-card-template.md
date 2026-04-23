# Role Card Template

Use this template to document a player-facing role in HerbieGo.

The goal is to keep role cards consistent across current MVP roles and future roles while staying aligned with the canonical MVP vocabulary in [MVP Game Design](../mvp-game-design.md) and [Canonical Domain Model](../domain-model.md).

## How To Use This Template

- Treat the role card as a stable identity brief, not a round-specific report.
- Use canonical role names for MVP roles: `Procurement Manager`, `Production Manager`, `Sales Manager`, and `Finance Controller`.
- Distinguish clearly between current MVP mechanics and future-role speculation.
- Describe what the role is allowed to know without exposing hidden current-turn actions from other roles.
- Keep action language consistent with the legal action vocabulary already defined by the game rules.
- Prefer concrete operational tradeoffs over generic management advice.

## Document Metadata

- Role name:
- Status: `MVP` or `Future Role`
- Canonical ID:
- Last updated:
- Related issues:

## I. Identity And Objectives (The "Who")

### Description And Core Goal

Write a short mission statement explaining why this role exists in the plant and what good performance looks like.

Prompting questions:

- What plant outcome is this role trying to improve?
- What local bias or pressure naturally shapes this role's behavior?

### Success And Failure Criteria

Document the clearest signs that the role is succeeding or failing.

Include:

- 2 to 4 success signals
- 2 to 4 failure signals
- any tradeoff where local success can still damage plant-wide performance

### Key Performance Indicators (KPIs)

List the 3 to 5 KPIs that best summarize the role's health.

Use this table:

| KPI | Why It Matters | Healthy Signal | Warning Signal | Typical Decision Trigger |
| --- | --- | --- | --- | --- |
| `example_kpi` | Explain why the role watches it. | What good looks like. | What should worry the player. | What the player may change. |

Notes:

- KPIs should support decisions, not act as a generic metric dump.
- If the role is a future role, label any KPI that depends on mechanics not yet implemented.

## II. Operational Domain (The "What")

### Responsibilities

List the role's standing responsibilities in plain language.

Focus on:

- what this role is accountable for every round
- what information it is expected to interpret
- what outcome it must explain to the rest of the plant

### Decision Levers

Document the specific levers this role can pull in gameplay.

For MVP roles, keep this tied to legal actions already defined in the rules.
For future roles, separate likely control levers from speculative mechanics.

Use this table:

| Lever | MVP Or Future | What The Player Changes | Immediate Tradeoff | Likely Downstream Effect |
| --- | --- | --- | --- | --- |
| `example_lever` | `MVP` | Describe the change. | What gets harder. | Who else feels it. |

### Constraints

Document the hard limits that prevent this role from acting freely.

Examples:

- budget caps
- finite workstation capacity
- lead times
- debt ceiling
- inventory availability
- compliance obligations
- hidden simultaneous turns

## III. Ecosystem (The "How")

### Synergies

List the roles that usually benefit from good coordination with this role.

For each synergy, note:

- what gets easier when the roles align
- what shared information matters most

### Conflicts

List the roles this role naturally clashes with.

For each conflict, explain:

- the tension in one sentence
- what each side is optimizing
- when the disagreement should be negotiated versus escalated

### Reports And Data

Describe what information this role receives and what it should share back.

Use this table:

| Information | Visibility | Why The Role Needs It | Shared Before Reveal? | Shared After Resolution? |
| --- | --- | --- | --- | --- |
| `example_information` | `Plant-wide` or `Role-specific` | Explain the decision use. | `Yes/No` | `Yes/No` |

Guidance:

- Distinguish plant-wide information from role-specific information.
- Do not imply that a role can see other players' hidden current-turn actions.
- If a report is role-specific, explain what decision it supports.

## IV. Dynamic Variables (The "What If")

### Opportunities

Describe high-upside or high-risk plays this role may be tempted to make.

Examples:

- bulk buy to reduce unit cost
- price cut to capture demand
- overtime to protect service
- spending freeze to protect cash

### Events

Document the shocks or scenario changes that force this role to adapt.

Examples:

- demand spike
- supplier delay
- cash squeeze
- quality incident
- chronic bottleneck

For each event, note:

- what first warning sign the role would see
- what immediate response is reasonable
- what overreaction would look like

## V. Additional Aspects To Consider

### Information Asymmetry

Document what this role knows earlier, better, or in more detail than others.

Keep this section compatible with hidden simultaneous play:

- describe privileged visibility, not current-turn omniscience
- separate public role identity from round-specific private context

### Resource Ownership

Describe what scarce resources this role directly influences.

Examples:

- cash
- purchase spend
- labor time
- workstation capacity
- finished goods availability
- customer commitments

### Risk Profile

List the role's nightmare scenarios and early warning signs.

Use this table:

| Risk | Earliest Warning Sign | What Happens If Ignored | Typical Mitigation |
| --- | --- | --- | --- |
| `example_risk` | First signal. | Consequence. | Likely response. |

## VI. Relationships

Document the role's relationship with every other relevant role in the game.

Use one row per role pairing:

| Other Role | Primary Synergy | Primary Conflict | Information To Share | When To Escalate |
| --- | --- | --- | --- | --- |
| `example_role` | How they help each other. | Where they clash. | What should be discussed. | What cannot stay local. |

Guidance:

- Cover all current MVP role pairings when documenting an MVP role.
- For future roles, include both MVP interactions and expected future-role interactions where useful.

## Optional Closing Section

### Notes For Role Briefings And Playbooks

Use this section when the role card should feed other docs or runtime surfaces.

Capture:

- what belongs in a short role card versus a longer gameplay playbook
- what briefing language can be reused for human and AI-facing role summaries
- what details belong in runtime reports instead of stable role identity docs
