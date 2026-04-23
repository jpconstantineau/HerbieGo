# Plant Manager Weekly Report

This document defines the weekly report a future `Plant Manager` role would likely need in HerbieGo.

The report should help the player answer one core question before acting:

`What is the biggest plant-wide constraint this week, and which directive best improves total performance without pushing the problem into another function?`

## Future Role Status

The `Plant Manager` is not currently implemented in the MVP action roster. This document is forward-looking design guidance for future expansion, onboarding, and AI-role planning.

This role assumes a future cross-functional control surface that can coordinate or direct other functions. The report therefore emphasizes aggregated plant insight rather than one department's local dashboard.

## Role Scope

- Role: `Plant Manager`
- Status: `Future Role`
- Primary decision horizon: immediate plant stability plus the next few rounds of coordinated tradeoff management
- Main tradeoff: local departmental success versus total plant throughput, service, cash health, and resilience

## What This Report Should Support

The weekly plant-manager report should help the player decide whether to:

- shift plant-wide priority to the most important constraint
- protect service by overriding a locally convenient but globally harmful choice
- direct more attention toward quality, maintenance, logistics, or finance pressure
- intervene when cross-role negotiation is failing
- trade short-term local pain for better total plant performance

The report should synthesize the plant's real state into a small number of actionable cross-functional decisions.

## Recommended Report Structure

Present the report in this order.

## 1. Executive Scorecard

Start with a one-page plant health summary that highlights the most important plant-wide outcomes.

Recommended summary items:

- revenue and margin signal
- throughput or output signal
- service reliability signal
- inventory exposure
- cash or debt pressure
- top plant-wide risk for the next round

Why first:

- the Plant Manager should see overall plant health before diving into any one function's interpretation

## 2. Resource And Bottleneck Report

This section should identify where the plant is actually constrained.

Use one row per major resource class or bottleneck area:

| Resource Or Constraint | Current Status | Why It Matters | Risk Level | Likely Cross-Role Impact |
| --- | --- | --- | --- | --- |
| `example_constraint` | Current condition. | Throughput or service effect. | `Low`, `Medium`, or `High`. | Who it affects next. |

Recommended fields:

- current production bottleneck
- reliability or downtime pressure
- inventory congestion
- warehouse or movement pressure
- labor or support-capacity stress where relevant

Decision value:

- helps the Plant Manager focus the organization on the true system constraint instead of the loudest local complaint

## 3. Market And Fulfillment Report

This section links the plant's internal condition to customer outcomes.

Use one row per major commercial pressure area:

| Commercial Signal | Current Status | Why It Matters | Warning Trigger | Likely Response |
| --- | --- | --- | --- | --- |
| `example_signal` | Current condition. | Customer or revenue effect. | What should worry the plant. | Direct, coordinate, or escalate. |

Recommended fields:

- order backlog pressure
- on-time-delivery signal
- customer complaint or return signal
- lead-time credibility trend
- whether service problems are driven by production, logistics, quality, or demand mismatch

Decision value:

- helps the Plant Manager separate a market problem from an internal coordination problem

## 4. Variance And Deep-Dive Report

This section should explain what changed materially and why.

Recommended fields:

- spend versus budget signal by function
- material or yield variance signal
- labor or overtime variance signal
- largest cross-functional miss of the week
- whether the root cause is supply, production, quality, maintenance, logistics, or financial discipline

Use a table like this:

| Variance Area | Current Signal | Likely Root Cause | Who Needs To Coordinate | Why It Matters |
| --- | --- | --- | --- | --- |
| `example_variance` | Current condition. | Best current explanation. | Key roles involved. | Plant-wide effect. |

Decision value:

- helps the Plant Manager intervene on causes rather than symptoms

## 5. Inter-Role Link Watchlist

The Plant Manager exists to see where local decisions are colliding.

Use one row per important tension:

| Role Link | Current Tension | What Each Side Is Optimizing | Escalation Risk | Suggested Plant-Level Focus |
| --- | --- | --- | --- | --- |
| `example_link` | Current conflict. | Local objectives in conflict. | `Low`, `Medium`, or `High`. | What the Plant Manager should push. |

Recommended examples:

- Procurement versus Finance
- Production versus Sales
- Production versus Maintenance
- Sales versus Logistics
- Quality versus Throughput pressure

Decision value:

- helps the Plant Manager issue coherent directives instead of reacting function by function

## 6. Plant Manager Decision Prompts

End the report with 3 to 5 plain-language prompts that guide the next directive.

Recommended prompts:

- What is the single biggest plant-wide constraint this week?
- Which local optimization is currently hurting the system most?
- Where should the plant accept short-term pain to avoid larger downstream failure?
- Which conflict can still be negotiated locally, and which one now needs plant-level direction?
- If only one function gets extra attention or protection this week, which one changes the total plant outcome most?

Why this matters:

- the report should drive a directive or coordination move, not just summarize departmental updates

## Visibility Guidance

This report would aggregate plant-wide state and role-specific summaries.

Plant-wide or shared inputs:

- financial, operational, and service signals
- inventory and backlog pressure
- major variance signals across functions
- cross-role tensions already visible after resolution

Role-focused interpretation:

- system bottleneck identification
- cross-functional prioritization
- escalation judgment
- recommendations for plant-wide direction

When this role exists in runtime play, its report should still respect hidden simultaneous turns and must not reveal other roles' current-turn actions before reveal.

## MVP Versus Future Expansion

| Report Element | MVP Now | Future Expansion |
| --- | --- | --- |
| Aggregated plant health summary | Limited | Yes |
| Cross-role bottleneck analysis | Limited | Yes |
| Functional variance deep dives | Limited | Yes |
| Plant-level directive support | No | Yes |
| Maintenance and quality oversight signals | No | Yes |
| Logistics and warehouse coordination signals | No | Yes |
| Full cross-functional prioritization dashboard | No | Yes |

## Example Decisions This Report Should Enable

- `Prioritize maintenance`: accept short-term output loss to restore future reliability
- `Protect service`: shift attention toward shipments or backlog at the expense of lower-value internal goals
- `Freeze or support spending selectively`: direct Finance and Procurement toward the constraint that matters most
- `Escalate quality containment`: slow the plant down before customer damage grows

## Design Guardrails

When contributors implement or refine this report, they should:

- frame it clearly as future-role guidance until the role exists in-game
- keep it aggregated and decision-oriented rather than turning it into a pile of raw departmental detail
- make cross-role links explicit
- help the Plant Manager identify the true system constraint, not just the most visible symptom
