# Role Documentation Scope

This document defines which role documentation is MVP-critical for HerbieGo today and which role documentation is intentionally forward-looking for planned expansion.

Use this page whenever a contributor needs to decide:

- whether a role is currently playable
- what kind of documentation should exist for that role now
- how future-role documentation should be framed so it does not imply engine support that does not exist yet

The canonical MVP role list is defined in [MVP Game Design](./mvp-game-design.md) and [Canonical Domain Model](./domain-model.md).

## Why This Scope Exists

The current role-documentation work mixes two valid but different goals:

- documenting the roles that are already part of the MVP action roster
- exploring realistic future roles that are not yet implemented in the simulation

Both kinds of work are useful, but they should not read as if they are at the same implementation status.

## Current Role Status

| Role | Canonical Status | Currently Playable In MVP | Documentation Priority | Notes |
| --- | --- | --- | --- | --- |
| Procurement Manager | MVP role | Yes | MVP-critical | Part of the current action roster and round flow. |
| Production Manager | MVP role | Yes | MVP-critical | Part of the current action roster and round flow. |
| Sales Manager | MVP role | Yes | MVP-critical | Part of the current action roster and round flow. |
| Finance Controller | MVP role | Yes | MVP-critical | Part of the current action roster and round flow. |
| Quality Manager | Future role | No | Expansion-oriented | Valid design target, but not implemented in the MVP action roster. |
| Logistics and Warehouse Manager | Future role | No | Expansion-oriented | Valid design target, but not implemented in the MVP action roster. |
| Maintenance Manager | Future role | No | Expansion-oriented | Valid design target, but not implemented in the MVP action roster. |
| Plant Manager | Future role | No | Expansion-oriented | Valid design target, but not implemented in the MVP action roster. |

## Documentation Expectations By Role Status

### MVP Roles

For MVP roles, documentation should describe how the role works in the current playable game.

Expected emphasis:

- current legal actions
- current round visibility rules
- current reports and decision inputs
- realistic local incentives and cross-role tradeoffs
- wording that can be reused for human onboarding and AI role briefings

Documentation for MVP roles should avoid:

- implying mechanics that do not exist yet
- importing future-role control levers into the current ruleset without labeling them as post-MVP

### Future Roles

For future roles, documentation should act as design and gameplay guidance for later expansion.

Expected emphasis:

- likely mission and decision pressures
- likely reports, warning triggers, and role tensions
- likely mechanics needed to support the role well
- clear notes about what is speculative versus already supported

Documentation for future roles must say plainly that:

- the role is not currently implemented in the MVP action roster
- the document is meant to guide future design, onboarding, and prompt work
- any control surface described is provisional until the simulation supports it

## Documentation Artifact Matrix

| Artifact | MVP Roles | Future Roles | Purpose |
| --- | --- | --- | --- |
| Role card | Required | Recommended | Stable identity, objectives, KPIs, constraints, and relationship summary. |
| Gameplay playbook | Recommended | Required | Practical turn-by-turn guidance, decision checklists, and tradeoff examples. |
| Reports the role wants | Required when the role is playable | Optional early design artifact | Clarifies what information the role needs in order to act well. |
| AI briefing or prompt content | Required | Draftable | Reusable role guidance for AI-controlled players without leaking hidden current-turn state. |
| TUI briefing or role view | Required when surfaced in-game | Not required until the role exists in-game | Player-facing runtime projection of role identity and current decision context. |
| Annotated sample turns | Recommended | Optional | Shows how a role moves from report interpretation to action choice. |

## How To Frame Future-Role Documentation

When writing for a future role:

- include a visible `Future Role Status` section near the top
- use phrases such as `would likely control`, `would monitor`, or `future mechanics may include`
- call out which examples are realistic guidance versus current implementation
- avoid language that sounds like the player can select that role in today's MVP

Good framing:

- `This is a future-role playbook for a role not yet implemented in the MVP action roster.`
- `The role would likely monitor supplier quality and inspection backlog if added later.`

Bad framing:

- `The Quality Manager chooses inspection intensity each round.` unless the text also says that this is future design guidance
- `Players can use this role in the MVP.` when that is not true

## Artifact Placement Guidance

Use a structure like this:

- stable role-definition documents belong under `docs/`
- reusable templates belong under `docs/templates/`
- future-role playbooks should use file names that make their expansion status obvious

When a future-role document is added, its title should make the status clear.

Preferred examples:

- `Quality Manager Gameplay Playbook (Future Role)`
- `Plant Manager Role Card (Future Role)`

Avoid titles that imply current implementation unless the role is already in the MVP action roster.

## Guidance For Future Issues

When opening or refining role-documentation issues:

- identify whether the role is `MVP` or `Future Role`
- say whether the doc is meant to describe current gameplay or guide later design
- point back to the canonical MVP role list when there is any risk of ambiguity
- avoid acceptance criteria that imply runtime support for future-role mechanics

Issue bodies should answer these questions early:

- Is the role currently playable?
- Is the document normative for MVP behavior or exploratory for expansion?
- Which runtime surfaces, if any, should reuse this content now?

## Relationship To Other Documentation

Use this scoping page together with:

- [MVP Game Design](./mvp-game-design.md) for the current playable role roster and legal action model
- [Canonical Domain Model](./domain-model.md) for canonical role identifiers and vocabulary
- role cards for stable role identity
- gameplay playbooks for practical decision guidance
- report-design docs for what information each role needs each round

If a document conflicts with the MVP role list, the MVP game design and domain model remain authoritative until a deliberate design change is accepted.
