# Single-Source Role Briefing Spec

This document defines the canonical role-briefing structure HerbieGo should use across documentation, human-facing briefings, TUI role views, and AI prompt assembly.

Use this specification when contributors need to:

- add or revise role cards
- build runtime role briefings in the TUI
- assemble AI system and role prompts
- add future roles without inventing a new briefing format each time

## Goal

The same role guidance appears in several surfaces:

- stable documentation for contributors
- human onboarding and role briefings
- TUI role views
- AI prompt content

Those surfaces do not need identical wording, but they should all draw from the same underlying role-briefing model so role intent does not drift.

## Design Principles

### Principle 1: Stable Role Identity Should Have One Home

Role mission, legal action summary, incentives, and warning patterns should not be rewritten independently for every surface.

They should be defined once in a canonical structure, then adapted as needed.

### Principle 2: Stable Briefing Content Must Be Separate From Round State

Stable role guidance describes what the role is and how it should think.

Round-specific state describes what is happening this week.

Those are both important, but they must not be stored or phrased as the same kind of content.

### Principle 3: Human And AI Surfaces Should Share Meaning, Not Always Verbatim Wording

Some fields can be reused almost word for word.

Other fields should be adapted for:

- human readability
- AI decisional clarity
- TUI space constraints

### Principle 4: Hidden Simultaneous Play Must Be Respected Everywhere

No briefing field should leak hidden current-turn state.

Round-specific prompt assembly must rely only on information the role is actually allowed to know.

## Canonical Briefing Object

Each role should have one canonical briefing object with these fields.

| Field | Required | Purpose | Stable Or Dynamic | Notes |
| --- | --- | --- | --- | --- |
| `role_id` | Yes | Canonical machine-readable identifier | Stable | Example: `production_manager` |
| `display_name` | Yes | Human-facing role name | Stable | Example: `Production Manager` |
| `implementation_status` | Yes | Whether the role is `MVP` or `Future Role` | Stable | Must be explicit for expansion roles |
| `one_sentence_mission` | Yes | Short role purpose | Stable | Reusable almost verbatim |
| `public_responsibilities` | Yes | What the role is accountable for | Stable | Human and AI safe |
| `local_incentive_or_bias` | Yes | Natural local optimization pressure | Stable | Crucial for realistic role behavior |
| `legal_action_summary` | Yes for MVP roles | What the role can actually submit in runtime play | Stable | Future roles may mark this as provisional |
| `non_controls` | Recommended | What the role does not directly control | Stable | Helps avoid hallucinated authority |
| `decision_principles` | Yes | Short list of durable reasoning rules | Stable | Works well across docs and prompts |
| `recommended_metric_focus` | Yes | Which metrics matter most to the role | Stable | Can map to reports and prompts |
| `warning_triggers` | Yes | The patterns that should cause concern | Stable | Draws from KPI and playbook guidance |
| `common_failure_modes` | Recommended | How the role hurts the plant when played badly | Stable | Strong for onboarding and prompting |
| `primary_synergies_and_conflicts` | Recommended | High-value role relationships | Stable | Supports docs and runtime hints |
| `visibility_scope` | Yes | What information class the role sees | Stable | Must align with visibility rules |
| `runtime_prompt_notes` | Recommended | Guidance on how to present state to the role at runtime | Stable | No hidden-state leakage |
| `report_template_mapping` | Recommended | What report sections feed this role best | Stable | Ties to report-template docs |
| `playbook_mapping` | Recommended | Which topics belong in the longer playbook | Stable | Prevents duplication |
| `round_context` | No, runtime only | Current-round visible state and alerts | Dynamic | Must never be stored as stable identity |

## Field Definitions

## Core Identity Fields

### `role_id`

Canonical identifier used in code, prompts, and data mapping.

Rules:

- stable over time
- snake_case
- should match domain-model vocabulary

### `display_name`

Human-facing role label.

Rules:

- should match role cards and UI labels
- should not vary by surface without strong reason

### `implementation_status`

Allowed values:

- `MVP`
- `Future Role`

Rules:

- must be visible in contributor-facing docs
- future-role surfaces should never imply current runtime support

## Guidance Fields

### `one_sentence_mission`

A short paragraph or sentence that answers:

- what the role is trying to protect
- what plant-wide failure it is most trying to avoid

Best use:

- reusable across docs, TUI briefings, and AI prompts with minimal change

### `public_responsibilities`

Short list of what the role is responsible for.

Rules:

- use plain operational language
- should be safe to show to any human or AI player assigned that role
- should not include current-round hidden specifics

### `local_incentive_or_bias`

This field captures the role's natural distortion.

Examples:

- Procurement may overvalue safety stock and cheap unit cost
- Production may overvalue utilization and activity
- Sales may overvalue bookings and revenue momentum
- Finance may overvalue short-term cost discipline

Why this field matters:

- it makes role behavior realistic
- it helps AI prompts avoid generic plant-manager reasoning when a role should think locally first

### `legal_action_summary`

Short summary of what the role can actually do in runtime play.

Rules:

- MVP roles should map directly to the canonical action vocabulary
- future roles should say `not currently implemented` or `future control surface would likely include`

### `non_controls`

Short list of what the role cannot directly command.

Why this matters:

- reduces confusion in human onboarding
- prevents AI roles from inventing authority they do not have

### `decision_principles`

Three to seven durable guidance bullets such as:

- protect the bottleneck, not just utilization
- protect revenue quality, not just order volume
- tighten weaker spend first

Best use:

- reusable verbatim in most surfaces

### `recommended_metric_focus`

Small set of metrics the role should watch first.

Rules:

- should map cleanly to report sections and prompt assembly
- should not become a long KPI dump

### `warning_triggers`

Specific patterns that should prompt caution or escalation.

Examples:

- backlog aging
- one-round supply exposure with no in-transit cover
- debt nearing ceiling
- PM backlog growth on a critical asset

### `common_failure_modes`

Short list of what bad play looks like for the role.

Best use:

- especially valuable in gameplay playbooks and AI prompts

### `primary_synergies_and_conflicts`

Summarizes:

- who the role most needs to coordinate with
- where the most common tension appears

Best use:

- role cards
- playbooks
- runtime coordination hints

### `visibility_scope`

Defines the information class available to the role.

Suggested subfields:

- `shared_pre_reveal_state`
- `role_specific_interpretation`
- `hidden_current_turn_state`
- `post_resolution_visibility`

Rules:

- must align with the visibility guide
- should never imply access to another role's hidden current-turn action

## Mapping Guidance By Surface

## Role Card

Purpose:

- contributor-facing stable role definition

Should emphasize:

- identity
- success and failure logic
- KPIs
- responsibilities
- synergies and conflicts

Should draw mainly from:

- `display_name`
- `implementation_status`
- `one_sentence_mission`
- `public_responsibilities`
- `local_incentive_or_bias`
- `legal_action_summary`
- `decision_principles`
- `recommended_metric_focus`
- `warning_triggers`
- `primary_synergies_and_conflicts`

## Gameplay Playbook

Purpose:

- practical, turn-oriented decision guidance

Should emphasize:

- what to read first each turn
- checklist logic
- trigger responses
- failure modes
- cross-role coordination under pressure

Should draw from the canonical briefing object plus richer examples.

Most relevant fields:

- `one_sentence_mission`
- `local_incentive_or_bias`
- `legal_action_summary`
- `non_controls`
- `decision_principles`
- `recommended_metric_focus`
- `warning_triggers`
- `common_failure_modes`
- `primary_synergies_and_conflicts`

## Human Runtime Briefing

Purpose:

- orient a player quickly inside the current round

Should combine:

- stable role identity
- legal actions
- the 3 to 5 most important current-round signals

Best field mix:

- `display_name`
- `one_sentence_mission`
- `legal_action_summary`
- `decision_principles`
- `warning_triggers`
- `round_context`

## TUI Role View

Purpose:

- present runtime state in a role-shaped way without violating visibility rules

Should include:

- stable header information
- current-round metrics that the role is allowed to know
- role-specific prompts or warnings

Should not include:

- another role's hidden current-turn submission

Best field mix:

- `display_name`
- `implementation_status`
- `decision_principles`
- `recommended_metric_focus`
- `warning_triggers`
- `visibility_scope`
- `round_context`

## AI Prompt Assembly

Purpose:

- give an AI role enough stable identity and current context to make a realistic role-constrained decision

Should include:

- stable mission
- local bias
- legal actions
- key metrics to watch
- warning triggers
- current visible state

Should not include:

- hidden current-turn actions from other roles
- plant-wide omniscience the human player would not get

Best field mix:

- `role_id`
- `display_name`
- `implementation_status`
- `one_sentence_mission`
- `local_incentive_or_bias`
- `legal_action_summary`
- `non_controls`
- `decision_principles`
- `recommended_metric_focus`
- `warning_triggers`
- `common_failure_modes`
- `visibility_scope`
- `round_context`

## Verbatim Reuse Guidance

These fields are usually safe to reuse nearly verbatim across docs and prompts:

- `display_name`
- `implementation_status`
- `one_sentence_mission`
- `decision_principles`
- `recommended_metric_focus`
- `warning_triggers`

These fields usually need adaptation by surface:

- `public_responsibilities`
- `legal_action_summary`
- `primary_synergies_and_conflicts`
- `runtime_prompt_notes`
- `round_context`

These fields should be especially careful in AI prompts:

- `local_incentive_or_bias`
- `common_failure_modes`
- `visibility_scope`

The aim is to preserve meaning while adjusting tone, length, and format for the destination surface.

## Example Skeleton

```yaml
role_id: production_manager
display_name: Production Manager
implementation_status: MVP
one_sentence_mission: Convert available parts and finite capacity into the most useful finished output the plant can realistically produce.
public_responsibilities:
  - turn parts into finished goods
  - allocate finite workstation capacity
  - protect plant throughput
local_incentive_or_bias: Tends to overvalue utilization and visible activity even when they do not improve total flow.
legal_action_summary:
  - release_product(product_id, quantity)
  - allocate_capacity(workstation_id, product_id, capacity_units)
non_controls:
  - procurement orders
  - sales pricing
  - retroactive finance vetoes
decision_principles:
  - protect the bottleneck
  - do not release work that cannot realistically finish
  - reserve extra spend for meaningful throughput gains
recommended_metric_focus:
  - feasible output versus target
  - bottleneck utilization on useful work
  - WIP accumulation
warning_triggers:
  - plan exceeds visible parts or capacity
  - WIP rises faster than completions
  - spend rises without throughput benefit
common_failure_modes:
  - maximizing activity instead of flow
  - using overtime to hide bad prioritization
visibility_scope:
  shared_pre_reveal_state: broad plant state
  hidden_current_turn_state: other roles' unresolved submissions
```

## Future-Role Guidance

Future roles should use the same canonical structure, but with two extra rules:

1. `implementation_status` must remain explicit everywhere.
2. `legal_action_summary` should describe likely future control surface without implying current runtime support.

Example:

- `future control surface would likely include inspection intensity and quarantine decisions`

Not:

- `choose inspection intensity this round` if the role is not implemented yet

## Contributor Checklist

When adding or changing role guidance, contributors should ask:

1. Is this stable role identity or current-round state?
2. Which canonical field does this content belong to?
3. Can this wording be reused across surfaces, or does it need adaptation?
4. Does it imply legal authority the role does not actually have?
5. Does it leak hidden current-turn information?

If those questions cannot be answered cleanly, the briefing content is probably not yet in the right shape.
