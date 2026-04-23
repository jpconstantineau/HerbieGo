# MVP Role Report Template Guide

This guide defines the expected structure of MVP role reports so each role receives information that is actionable for weekly decision-making rather than a generic metric dump.

Use this document when contributors need to:

- build or revise role dashboards
- decide which report sections belong in each role's weekly view
- connect report sections to actual role decisions
- keep human-facing and AI-facing role reports aligned

## Why This Guide Exists

The role report issues define what each role wants to know.

This guide goes one level further and answers:

- what order the report should present information in
- what each section is supposed to help the player decide
- what belongs in the summary versus drill-down detail
- what should be company-wide versus role-specific emphasis

## Design Principles

### Principle 1: Reports Should Support A Weekly Decision

Every report should help the player answer one core question before acting.

If a section does not clearly support a decision or tradeoff, it should be reconsidered.

### Principle 2: Put The Most Actionable Material First

Reports should start with:

- a short executive summary
- the role's highest-value constraints and decision signals

They should not begin with long tables that require the player to infer what matters unaided.

### Principle 3: Shared Data, Role-Specific Emphasis

The underlying source data can be shared across roles.

What changes by role is:

- ordering
- interpretation
- warning prompts
- what tradeoffs the report makes most explicit

### Principle 4: Hidden Simultaneous Play Must Be Preserved

Role reports should only use:

- visible plant state
- prior resolved outcomes
- role-specific interpretation of visible state

Role reports must not reveal:

- another role's current-turn hidden action
- unresolved plant outcomes from the current round

## Shared Report Template

Every MVP role report should follow this basic shape.

1. Executive summary
2. Role-critical operating picture
3. Constraint and risk view
4. Tradeoff and pressure view
5. Decision prompts

### 1. Executive Summary

Purpose:

- tell the player what matters most this week in a few lines

Should include:

- one-line health read
- one to three biggest risks
- one to three likely decision priorities

Should avoid:

- raw metric lists without interpretation

### 2. Role-Critical Operating Picture

Purpose:

- show the operational state this role most needs in order to act

Examples:

- Procurement: material coverage and in-transit supply
- Production: feasible output, bottleneck, and WIP
- Sales: backlog health, service credibility, and finished goods
- Finance: cash, debt, margin, and inventory exposure

### 3. Constraint And Risk View

Purpose:

- show what is becoming fragile or dangerous

Should include:

- threshold-aware warnings
- red and yellow conditions
- the likely reason the risk matters

### 4. Tradeoff And Pressure View

Purpose:

- show the tensions this role must navigate, not just the raw state

Examples:

- Procurement: shortage protection versus overbuy
- Production: throughput versus WIP and spend
- Sales: revenue growth versus service credibility
- Finance: liquidity discipline versus throughput support

### 5. Decision Prompts

Purpose:

- end the report with plain-language prompts that make the player act

Good prompts:

- Which exposed part is actually worth buying now?
- What is the highest-value feasible mix this week?
- Are we winning healthy demand or dangerous backlog?
- Which spend should tighten and which still deserves support?

## Role-By-Role Templates

## Procurement Manager Report

Core decision:

`Which purchases best protect next-round production without creating avoidable cash and inventory drag?`

Recommended section order:

1. Executive summary
2. Critical part coverage
3. In-transit orders and receipt timing
4. Spend and inventory exposure
5. Supplier or reliability watchlist
6. Procurement decision prompts

### Section Guidance

| Section | Company-Wide Or Role-Focused | Mandatory Or Nice-To-Have | What The Player Should Infer | Decision Supported |
| --- | --- | --- | --- | --- |
| Executive summary | Role-focused interpretation of shared state | Mandatory | Whether the week is about shortage protection, discipline, or both | How aggressive the buy posture should be |
| Critical part coverage | Shared state with procurement emphasis | Mandatory | Which part is the next real bottleneck risk | What to buy first |
| In-transit orders and receipt timing | Shared state with procurement emphasis | Mandatory | Whether open supply already protects the risk | Whether to add, delay, or avoid duplication |
| Spend and inventory exposure | Shared plus finance-linked interpretation | Mandatory | Whether protection is becoming overbuy | Whether to shrink lower-value buys |
| Supplier or reliability watchlist | Future-enrichment leaning | Nice-to-have in MVP, stronger post-MVP | Where supply stability could undermine future cover | Whether to escalate future sourcing risk |
| Decision prompts | Role-focused | Mandatory | The tradeoff Procurement should resolve now | Final order and quantity choice |

## Production Manager Report

Core decision:

`What is the highest-value feasible production plan given visible parts, capacity, WIP, and spend pressure?`

Recommended section order:

1. Executive summary
2. Throughput and output status
3. Bottleneck and capacity view
4. Material readiness and starvation risk
5. WIP and congestion pressure
6. Production spend and support pressure
7. Production decision prompts

### Section Guidance

| Section | Company-Wide Or Role-Focused | Mandatory Or Nice-To-Have | What The Player Should Infer | Decision Supported |
| --- | --- | --- | --- | --- |
| Executive summary | Role-focused interpretation of shared state | Mandatory | Whether the week is about throughput protection, WIP control, or service catch-up | How bold the production plan should be |
| Throughput and output status | Shared with production emphasis | Mandatory | How much useful output is really moving | How much release is worth attempting |
| Bottleneck and capacity view | Shared with production emphasis | Mandatory | Where the real constraint is | Which product deserves constrained capacity |
| Material readiness and starvation risk | Shared with production emphasis | Mandatory | Whether the intended plan is materially legal and realistic | Whether the mix must change |
| WIP and congestion pressure | Shared with production emphasis | Mandatory | Whether more release would help or only clog flow | Whether to reduce or defer work |
| Production spend and support pressure | Shared plus finance-linked interpretation | Mandatory | Whether extra spend protects throughput or only activity | Whether selective overtime is justified |
| Decision prompts | Role-focused | Mandatory | What tradeoff Production must accept now | Final release and allocation choice |

## Sales Manager Report

Core decision:

`How should pricing shape future demand without creating backlog, service, or margin damage the plant cannot support?`

Recommended section order:

1. Executive summary
2. Revenue and demand pipeline
3. Backlog and fulfillment pressure
4. Customer sentiment and service credibility
5. Price and margin quality
6. Sales decision prompts

### Section Guidance

| Section | Company-Wide Or Role-Focused | Mandatory Or Nice-To-Have | What The Player Should Infer | Decision Supported |
| --- | --- | --- | --- | --- |
| Executive summary | Role-focused interpretation of shared state | Mandatory | Whether the week calls for growth, restraint, or credibility repair | Overall pricing posture |
| Revenue and demand pipeline | Shared with sales emphasis | Mandatory | Whether demand pressure is healthy or weak | Whether to pursue more demand |
| Backlog and fulfillment pressure | Shared with sales emphasis | Mandatory | Whether accepted demand is still serviceable | Whether to cool demand or protect backlog quality |
| Customer sentiment and service credibility | Shared with sales emphasis | Mandatory | Whether service misses are becoming a demand problem | Whether to prioritize trust over volume |
| Price and margin quality | Shared plus finance-linked interpretation | Mandatory | Whether revenue quality is improving or eroding | Whether to raise, hold, or lower price |
| Decision prompts | Role-focused | Mandatory | Which commercial tradeoff matters most this round | Final product pricing decision |

## Finance Controller Report

Core decision:

`Which next-round targets protect liquidity and discipline without starving the plant of the support it still needs?`

Recommended section order:

1. Executive summary
2. Cash and debt position
3. Margin and economic quality
4. Spend pressure and budget realism
5. Inventory and working-capital exposure
6. Finance decision prompts

### Section Guidance

| Section | Company-Wide Or Role-Focused | Mandatory Or Nice-To-Have | What The Player Should Infer | Decision Supported |
| --- | --- | --- | --- | --- |
| Executive summary | Role-focused interpretation of shared state | Mandatory | Whether Finance should tighten, hold, or selectively support | Overall next-round target posture |
| Cash and debt position | Shared with finance emphasis | Mandatory | How close the plant is to real liquidity stress | How strict the guardrails must be |
| Margin and economic quality | Shared with finance emphasis | Mandatory | Whether revenue and operations are creating healthy economics | Whether to push for better pricing or mix discipline |
| Spend pressure and budget realism | Shared with finance emphasis | Mandatory | Whether budgets are still executable and where waste is appearing | Which targets to tighten or protect |
| Inventory and working-capital exposure | Shared with finance emphasis | Mandatory | Whether cash is being trapped in low-value stock | Whether to challenge buying or production buildup |
| Decision prompts | Role-focused | Mandatory | Which next-round tradeoff Finance should enforce | Final budget, target, and ceiling settings |

## Summary Versus Drill-Down Guidance

Each report should distinguish between:

- summary content that drives the weekly decision
- supporting detail that explains why the summary is justified

Good summary content:

- current health read
- the top risk
- the one or two most important tradeoffs
- short prompts tied to the role's legal actions

Good drill-down content:

- row-level tables
- product-by-product or part-by-part detail
- supporting variance explanations
- secondary warnings

Bad report structure:

- burying the decision signal below large tables
- repeating the same metric in three places without a different decision purpose
- showing company-wide data without telling the role why it matters

## Human And AI Support Guidance

Reports should support both human and AI-controlled roles.

To do that well:

- section titles should imply the decision they support
- prompts should be explicit enough that an AI role can reason from them
- summaries should be short enough for a human player to scan quickly
- the report should never rely on hidden current-turn information

For AI decision requests, the report should make it easy to extract:

- the top current risks
- the metric bands that matter
- the role's most plausible action options
- the tradeoff being accepted

## Future-Role Adoption Guidance

Future roles should use the same report-template logic:

1. executive summary
2. role-critical operating picture
3. constraint and risk view
4. tradeoff and pressure view
5. decision prompts

The exact metrics will differ, but the decision-support structure should remain recognizable across roles.

## Contributor Checklist

When building or refining a report, contributors should ask:

1. What single decision is this report trying to support?
2. Does the report surface the decision-critical risk in the first screen?
3. Is each section tied to a concrete role tradeoff?
4. Is the report emphasizing shared data correctly for this role?
5. Does the report avoid leaking hidden current-turn information?

If a report cannot answer those questions clearly, it is probably still a metric dump rather than a role-support tool.
