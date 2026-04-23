# Maintenance Manager Weekly Report

This document defines the weekly report a future `Maintenance Manager` role would likely need in HerbieGo.

The report should help the player answer one core question before acting:

`Where is asset reliability breaking down, and which maintenance action best protects future throughput without creating avoidable downtime or overspend?`

## Future Role Status

The `Maintenance Manager` is not currently implemented in the MVP action roster. This document is forward-looking design guidance for future expansion, onboarding, and AI-role planning.

Several concepts here build on plant constraints already implied by the MVP, such as workstation capacity and throughput loss, but the role assumes future mechanics like planned downtime, repair work, maintenance backlog, and asset reliability tracking.

## Role Scope

- Role: `Maintenance Manager`
- Status: `Future Role`
- Primary decision horizon: immediate uptime protection plus the next few rounds of reliability risk
- Main tradeoff: output today versus reliability tomorrow, maintenance spend, and planned downtime

## What This Report Should Support

The weekly maintenance report should help the player decide whether to:

- perform preventive maintenance now or delay it
- prioritize one repair over another
- escalate a bad-actor asset before it becomes a crisis
- spend more on maintenance support to avoid larger future failure
- force a short planned stop to avoid a much longer breakdown later

The report should make reliability risk visible before it appears only as unexplained throughput loss.

## Recommended Report Structure

Present the report in this order.

## 1. Executive Summary

Start with a short summary panel that highlights the biggest reliability risks.

Recommended summary items:

- whether overall asset health is stable, degrading, or critical
- which asset or line is creating the most uptime risk
- whether preventive work is being deferred unsafely
- whether spare-parts risk is increasing expected downtime
- whether a shutdown decision is becoming unavoidable

Why first:

- the role should know immediately whether the week is about prevention, repair prioritization, or emergency containment

## 2. Equipment Downtime And Reliability

This is the most important section because it shows where the plant is already losing time to asset failure.

Use one row per machine, line, or constrained asset:

| Asset | Downtime Signal | Failure Trend | Repair Burden | Risk Level | Notes |
| --- | --- | --- | --- | --- | --- |
| `example_asset` | Current lost-time condition. | Improving, flat, or worsening. | Light, moderate, or severe. | `Low`, `Medium`, or `High`. | Main interpretation. |

Recommended fields:

- unplanned downtime
- mean time between failures
- mean time to repair
- top repeat "bad actors"
- where reliability risk is most likely to hit throughput next

Decision value:

- helps Maintenance choose which asset deserves immediate attention versus watchful monitoring

## 3. Preventive Maintenance And Workload Balance

This section shows whether the plant is protecting future uptime or living only in reactive mode.

Use a table like this:

| Maintenance Signal | Current Status | Why It Matters | Warning Trigger | Likely Response |
| --- | --- | --- | --- | --- |
| `example_signal` | Current condition. | Reliability implication. | What should worry Maintenance. | Schedule, defer, or escalate. |

Recommended fields:

- preventive-maintenance completion rate
- maintenance backlog
- preventive versus reactive work mix
- overdue critical tasks
- whether maintenance capacity is being consumed by repeat failures instead of prevention

Decision value:

- helps Maintenance decide when short-term output pressure is undermining long-term asset health

## 4. Spare Parts And MRO Readiness

Maintenance cannot restore uptime if required parts are missing.

Use one row per critical spare family:

| Spare Or MRO Area | Availability Signal | Stockout Risk | Operational Impact | Notes |
| --- | --- | --- | --- | --- |
| `example_spare` | In stock, low, or exposed. | `Low`, `Medium`, or `High`. | What failure it would prolong. | Main interpretation. |

Recommended fields:

- critical spare availability
- stockout events that increased downtime
- maintenance, repair, and operations spend signal
- whether missing spares are making repair times much worse

Decision value:

- supports repair prioritization, spare escalation, and future coordination with Procurement and Finance

## 5. Asset-Health And Performance Watchlist

Maintenance often needs early warning signs before a machine fully fails.

Recommended fields:

- technical availability
- repeat symptom pattern
- energy or performance anomalies
- assets that are becoming more fragile even if still running

Use this table:

| Watch Item | Current Status | Why It Matters | Escalation Trigger | Likely Action |
| --- | --- | --- | --- | --- |
| `example_watch_item` | Current condition. | Reliability implication. | What makes it urgent. | Inspect, repair, or plan shutdown. |

Decision value:

- helps Maintenance intervene early instead of always reacting after failure

## 6. Maintenance Decision Prompts

End the report with 3 to 5 plain-language prompts that guide the next maintenance decision.

Recommended prompts:

- Which asset is most likely to cause the next major throughput loss?
- Is this a week to accept a short planned stop instead of risking a much longer failure?
- Which maintenance backlog item has become more dangerous than the plant is admitting?
- Are missing spares turning repair work into a larger risk than it should be?
- What should be escalated to Production, Procurement, or Finance before reliability risk worsens?

Why this matters:

- the report should guide a maintenance action, not just list breakdown statistics

## Visibility Guidance

This report would combine plant-wide throughput and asset signals with maintenance-specific interpretation.

Plant-wide or shared inputs:

- throughput loss associated with constrained equipment
- capacity pressure on key lines or workstations
- spend and inventory signals related to repair support

Role-focused interpretation:

- whether the plant is living off unhealthy deferral
- which assets need preventive action
- where repair, replace, or planned shutdown decisions are becoming urgent

When this role exists in runtime play, its report should still respect hidden simultaneous turns and must not expose other roles' current-turn actions before reveal.

## MVP Versus Future Expansion

| Report Element | MVP Now | Future Expansion |
| --- | --- | --- |
| Workstation-capacity pressure | Yes | Yes |
| Explicit downtime by asset | No | Yes |
| MTBF and MTTR tracking | No | Yes |
| Preventive-maintenance completion | No | Yes |
| Maintenance backlog | No | Yes |
| Critical spare readiness | No | Yes |
| Technical-availability signal | No | Yes |
| Energy-anomaly monitoring | No | Yes |

## Example Decisions This Report Should Enable

- `Shutdown call`: take a short planned stop to avoid catastrophic downtime later
- `Repair prioritization`: focus scarce maintenance effort on the asset hurting the plant most
- `Spare escalation`: push the plant to cover critical repair parts before failure occurs
- `Repair versus replace`: build the case for a larger asset decision when repeated fixes are no longer good enough

## Design Guardrails

When contributors implement or refine this report, they should:

- frame it clearly as future-role guidance until the role exists in-game
- keep the tradeoff between uptime today and reliability tomorrow explicit
- connect maintenance metrics to real throughput and cost consequences
- avoid turning the report into a generic CMMS dump without decision guidance
