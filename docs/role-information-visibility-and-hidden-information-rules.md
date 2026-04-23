# Role Information Visibility And Hidden-Information Rules

This guide defines what each role is allowed to know before acting, what information is shared plant-wide, and what remains hidden during simultaneous play in HerbieGo.

Use this document when contributors need to:

- design or review role reports and dashboards
- write TUI role views and briefings
- assemble AI prompt context safely
- decide whether a piece of information belongs in pre-reveal state, post-reveal logs, or role-specific interpretation

## Why This Guide Exists

Hidden simultaneous turns are a core MVP mechanic.

If contributors are inconsistent about what a role can see, the game stops feeling fair and role-specific decisions become easier or harder for the wrong reasons.

This guide therefore separates:

- plant-wide shared state
- role-specific interpretation
- hidden current-turn intent
- post-resolution visibility

## Core Visibility Principles

### Principle 1: Shared State Should Be Broad

The MVP exposes a wide shared operational picture so the game is about tradeoffs, not blind guessing.

Before acting, every current MVP role should see the current visible plant state that the rules define in the round's broadcast phase.

### Principle 2: Current-Turn Choices Stay Hidden

All players choose actions simultaneously.

A role may reason about what another role is likely to do, but it must not see that role's locked current-turn action before reveal and resolution.

### Principle 3: Role Dashboards Emphasize, They Do Not Invent

Role-specific reports should:

- emphasize the subset of shared state that matters most to that role
- add role-specific interpretation and prompts

Role-specific reports should not:

- leak hidden current-turn actions
- invent private facts that bypass the simultaneous-turn model

### Principle 4: Runtime Briefings Must Separate Stable Identity From Round State

Role identity, incentives, and legal action summaries are stable.

Round-specific metrics, alerts, and commentary are situational.

Those two content types should be assembled together carefully but not confused.

## MVP Pre-Reveal Shared State

The MVP broadcast phase already defines the baseline information every role receives at the start of a round.

Every MVP role sees:

- current week number
- current cash and debt
- current parts inventory
- current shop floor inventory and work-in-progress by product and workstation stage
- current finished goods inventory
- current customer backlog by customer and product
- current customer sentiment by customer
- workstation capacities for the round
- active budgets and targets from the previous round
- recent round log and player commentary from resolved prior rounds

This means the MVP is asymmetric mainly in interpretation and action responsibility, not in broad access to current plant state.

## MVP Visibility Matrix

| Information Category | Procurement Manager | Production Manager | Sales Manager | Finance Controller | Shared Before Reveal? | Shared After Resolution? |
| --- | --- | --- | --- | --- | --- | --- |
| Week number | Yes | Yes | Yes | Yes | Yes | Yes |
| Cash and debt | Yes | Yes | Yes | Yes | Yes | Yes |
| Parts inventory | Yes | Yes | Yes | Yes | Yes | Yes |
| Shop floor inventory and WIP | Yes | Yes | Yes | Yes | Yes | Yes |
| Finished goods inventory | Yes | Yes | Yes | Yes | Yes | Yes |
| Customer backlog by customer and product | Yes | Yes | Yes | Yes | Yes | Yes |
| Customer sentiment | Yes | Yes | Yes | Yes | Yes | Yes |
| Workstation capacities | Yes | Yes | Yes | Yes | Yes | Yes |
| Active budgets and targets from prior round | Yes | Yes | Yes | Yes | Yes | Yes |
| Recent round log and prior commentary | Yes | Yes | Yes | Yes | Yes | Yes |
| Role-specific report interpretation | Yes, own emphasis | Yes, own emphasis | Yes, own emphasis | Yes, own emphasis | Yes | Yes |
| Current-turn purchase intent | Procurement only | No | No | No | No | Yes |
| Current-turn production intent | No | Production only | No | No | No | Yes |
| Current-turn pricing or demand intent | No | No | Sales only | No | No | Yes |
| Current-turn finance target submission | No | No | No | Finance only | No | Yes |

## What Counts As Role-Specific Information

In the MVP, role-specific information should usually mean:

- prioritization cues
- decision prompts
- warnings tailored to the role
- summaries of which metrics matter most for that role

Examples:

- Procurement sees shared parts inventory, but its report highlights coverage risk, in-transit protection, and overbuy exposure
- Production sees shared WIP and capacity, but its report emphasizes bottleneck protection and feasible output
- Sales sees shared backlog and finished goods, but its report emphasizes backlog quality, sentiment, and service credibility
- Finance sees shared cash, inventory, and revenue signals, but its report emphasizes liquidity, margin quality, and budget realism

This is interpretation asymmetry, not raw-state asymmetry.

## Hidden Information During The Round

The following information must remain hidden until resolution starts:

- current-turn order quantities chosen by Procurement
- current-turn release quantities and capacity allocations chosen by Production
- current-turn prices and demand-pursuit choices chosen by Sales
- current-turn budget and target submissions chosen by Finance
- rationale text attached to those current-turn actions if the rationale would reveal the locked choice

Roles may still communicate general reasoning before reveal, but not the final hidden submission itself.

## Allowed Versus Disallowed Pre-Reveal Communication

### Allowed

- referring to visible plant state from the current round broadcast
- discussing what each role considers the main risk
- warning other roles about what visible conditions suggest
- stating conditional guidance such as `if backlog remains this fragile, demand should not be pushed harder`

### Disallowed

- revealing a locked current-turn action
- asking another role to reveal its current-turn action after submission
- exposing system-generated previews of unresolved actions
- building role dashboards that infer or leak hidden selections before reveal

## Post-Resolution Visibility

After resolution, the round log should make actions and outcomes legible enough for learning and coordination.

Post-resolution visibility should include:

- what each role submitted
- what the plant trimmed, adjusted, or rejected for legality
- the resulting state changes
- the commentary or rationale that explains why the role chose that action

This post-resolution transparency is important because simultaneous play should create uncertainty during the round, not permanent opacity after the fact.

## Commentary Visibility Rules

Commentary needs special handling because it can accidentally leak hidden state.

Before reveal:

- commentary should discuss visible conditions, risks, and tradeoffs
- commentary should avoid stating a locked action directly

After resolution:

- commentary may explain the actual action chosen
- commentary may name rejected alternatives and expected downstream effects

Design rule:

- if commentary would let another player reconstruct an unrevealed locked action with high confidence, it is too specific for pre-reveal use

## Guidance For Human Briefings, TUI Views, And AI Prompts

### Human Briefings

Human-facing role briefings may include:

- stable role purpose
- legal actions
- role-specific KPI focus
- common warning patterns

Human-facing runtime views may include:

- current shared plant state
- role-specific interpretation of that shared state

They must not include:

- hidden current-turn actions from any role

### TUI Role Views

TUI role views should:

- present the same shared source data consistently across roles
- vary emphasis, prompts, and ordering by role
- keep unresolved current-turn state hidden

TUI role views should not:

- create private operational facts that do not exist in the rules
- expose another role's in-progress or submitted hidden action

### AI Prompt Assembly

AI role prompts may include:

- stable role identity and local incentives
- legal action summary
- current round shared state
- role-specific decision prompts derived from shared state

AI role prompts must not include:

- another role's hidden current-turn action
- hidden future outcomes
- role-specific private state that the rules do not actually provide

## Future-Role Expansion Guidance

Future roles can introduce more asymmetry, but the model should remain understandable and fair.

Good future asymmetry:

- Quality sees richer defect and containment interpretation
- Logistics sees richer shipment and storage flow interpretation
- Maintenance sees richer asset-health and spare-readiness interpretation
- Plant Manager sees richer cross-role tension summaries after resolution or from explicitly shared state

Risky future asymmetry:

- giving one role hidden previews of another role's unresolved choices
- creating private raw-state silos that make cooperative reasoning impossible
- letting AI prompts receive information that human players would not

If a future role needs genuinely unique raw state, contributors should document:

- why that asymmetry improves gameplay
- what other roles can still infer from shared signals
- how the hidden-state boundary remains consistent across humans and AI

## Examples

### Allowed Example

The Sales Manager sees backlog, finished goods, and sentiment, then tells the team:

`Service credibility is weakening, so pushing more demand would be risky.`

This is allowed because it is interpretation of visible state, not a revealed hidden action.

### Disallowed Example

Before reveal, the Finance Controller tells Production:

`I already set next round's production budget below your current run rate.`

This is disallowed because it reveals a locked current-turn action.

### Allowed Post-Resolution Example

After the round resolves, Finance explains:

`I tightened the production budget because inventory and debt were both rising.`

This is allowed because the hidden-action phase is over.

## Implementation Guidance

When contributors add or revise docs, prompts, or UI surfaces, they should ask:

1. Is this stable role guidance or current round state?
2. Is the underlying information visible to all roles before reveal?
3. If it is role-specific, is it interpretation rather than leaked hidden action?
4. Would a human and an AI player receive the same class of information?

If the answer to any of those checks is unclear, this guide should be treated as authoritative until a design change is accepted.
