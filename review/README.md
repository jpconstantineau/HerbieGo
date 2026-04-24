# Review README

This folder contains developer-facing architectural documentation produced by a code review pass. No application code was changed.

## Contents

| File | Description |
|---|---|
| [`architecture-overview.md`](architecture-overview.md) | Full description of the current architecture: layers, packages, key types, data flows, concurrency model, and configuration |
| [`architectural-improvements.md`](architectural-improvements.md) | Identified design gaps and recommended improvements to reduce the cost of adding features and improve long-term maintainability |

## Quick Summary

HerbieGo is well-structured. The hexagonal architecture is clear, the domain model is clean, and the port/adapter separation is sound. The main areas where the architecture will start to show friction as the game grows are:

1. **Role extensibility** — Adding a new role today requires touching `switch roleID` blocks in four separate packages. A role registry would make this a single-location change.
2. **Scenario extensibility** — Only one scenario can be active at runtime, and it is hardwired. A scenario registry and config-driven selection are needed before a second scenario can be added cleanly.
3. **Provider extensibility** — Adding an AI provider requires editing the entry-point wiring (`cmd/herbiego/players.go`). A provider factory registry in the adapter layer would isolate this.
4. **Persistence not integrated** — The `MatchStateStore` port and memory implementation exist but are not wired into the live game loop. One field addition to `MatchRunner` would connect them.
5. **Prompt content tied to Starter scenario** — Example actions in AI prompts reference hardcoded Starter scenario entity IDs. These should be derived from the active scenario's catalog.

See [`architectural-improvements.md`](architectural-improvements.md) for the full list of 15 improvements with current state, problem statement, and recommended direction for each.
