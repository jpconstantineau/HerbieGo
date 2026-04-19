# ADR 0001: Initial Architecture And Package Boundaries

- Status: Accepted
- Date: 2026-04-19
- Deciders: HerbieGo maintainers
- Related roadmap issue: `#3`

## Context

Issue `#3` asks for an architecture decision record that makes package ownership explicit before implementation spreads across the repository.

The project already has two important design anchors:

- [MVP Game Design](../mvp-game-design.md)
- [Canonical Domain Model](../domain-model.md)

Those documents define what the game should do and which domain records are canonical. This ADR defines how implementation should be split into Go packages so contributors can place code consistently without creating circular dependencies or coupling the simulation to the UI.

## Decision Summary

HerbieGo will use a layered architecture built around a UI-independent deterministic engine.

- `internal/domain` owns shared game vocabulary and canonical data shapes.
- `internal/scenario` owns match setup data, scenario catalogs, and scenario-specific tuning values.
- `internal/engine` owns deterministic rules, round resolution, legality checks, and metric calculation.
- `internal/app` owns use-case orchestration between the engine and the outside world.
- `internal/ports` owns interfaces for time, randomness, persistence, player input, and model providers.
- `internal/projection` owns read models derived from domain state for TUI screens, debugging views, and prompts.
- `internal/adapters/...` owns concrete implementations for TUI, SQLite, OpenRouter, Ollama, seeded randomness, and other infrastructure.
- `cmd/...` owns process startup and dependency wiring only.

This keeps the simulation core reusable from tests, CLI tools, future HTTP APIs, and alternate UIs.

## Proposed Package Layout

```text
cmd/
  herbiego/

internal/
  app/
  domain/
  scenario/
  engine/
  ports/
  projection/
  adapters/
    ai/
      ollama/
      openrouter/
    persistence/
      sqlite/
    player/
      human/
      llm/
    random/
      seeded/
    tui/
```

This structure is intentionally small for the first implementation. New packages should be added only when they establish a stable boundary, not to mirror every concept one-for-one.

## Package Responsibilities

### `cmd/herbiego`

Owns:

- executable entrypoint
- configuration loading
- dependency construction
- process lifecycle and shutdown

Must not own:

- game rules
- prompt assembly rules
- SQL queries beyond adapter setup
- TUI business logic beyond startup

### `internal/domain`

Owns:

- canonical structs and enums from the domain-model ADR target
- domain identifiers and value types
- stable round, event, action, metrics, and state records

Must not own:

- persistence annotations tied to one storage engine
- Bubble Tea types
- LLM provider request and response types
- random number generation
- round resolution logic

### `internal/scenario`

Owns:

- scenario definitions and catalogs
- starting inventory, customer mix, workstation capacities, and economic constants
- scenario-specific rule parameters consumed by the engine
- scenario metadata used for match setup and future scenario selection

Must not own:

- mutable match state
- UI rendering logic
- persistence transport details
- round resolution logic

The scenario package exists so the game can grow beyond one starter plant without overloading `internal/domain` with content data.

### `internal/engine`

Owns:

- deterministic round resolution
- legality checks and action trimming
- world update sequencing
- metric calculation
- event emission
- construction of the next canonical state from current state plus inputs

Must not own:

- terminal rendering
- direct database access
- HTTP clients
- provider-specific prompt formatting
- wall-clock time lookups

### `internal/app`

Owns:

- use cases such as start match, collect actions, resolve round, save snapshot, and build round views
- orchestration across engine, projections, persistence, and player gateways
- transaction boundaries at the application layer

Must not own:

- low-level transport code
- deterministic rules that belong in `internal/engine`
- provider-specific SDK logic

### `internal/ports`

Owns interfaces consumed by `internal/app` and `internal/engine`, such as:

- `ActionSource`
- `MatchRepository`
- `EventRepository`
- `RandomSource`
- `Clock`
- `PromptDecisionProvider`

Ports should be small and named by capability, not by technology.

### `internal/projection`

Owns:

- `RoundView` builders from canonical state and recent history
- TUI-specific read models
- prompt-context projections for AI players
- debugging and inspection views derived from domain records

Must not own:

- mutation of canonical state
- rule decisions
- adapter I/O

This is where UI projections are built. The engine emits canonical state and events; the projection layer turns them into read models for presentation.

### `internal/adapters/...`

Own concrete infrastructure:

- `internal/adapters/tui`: Bubble Tea models, key handling, render layout
- `internal/adapters/persistence/sqlite`: repositories and schema mapping
- `internal/adapters/ai/openrouter`: OpenRouter client implementation
- `internal/adapters/ai/ollama`: Ollama client implementation
- `internal/adapters/player/human`: human input bridge
- `internal/adapters/player/llm`: AI player action collection via prompt/response flow
- `internal/adapters/random/seeded`: seeded PRNG implementation

Adapters may depend on external libraries. Core packages should not.

## Allowed Dependencies

The allowed dependency graph is:

```text
cmd/herbiego
  -> internal/app
  -> internal/adapters/*

internal/app
  -> internal/domain
  -> internal/scenario
  -> internal/engine
  -> internal/ports
  -> internal/projection

internal/engine
  -> internal/domain
  -> internal/scenario
  -> internal/ports

internal/projection
  -> internal/domain
  -> internal/scenario

internal/adapters/*
  -> internal/app
  -> internal/domain
  -> internal/scenario
  -> internal/ports
  -> internal/projection
```

Explicit rules:

- `internal/domain` depends on nothing inside `internal/`.
- `internal/scenario` may depend on `internal/domain` for canonical identifiers and record shapes.
- `internal/engine` may depend on `internal/domain` and narrow interfaces in `internal/ports`.
- `internal/engine` may read immutable scenario definitions from `internal/scenario`.
- `internal/projection` may depend on `internal/domain` and `internal/scenario` when scenario metadata affects presentation.
- `internal/app` may coordinate all core packages, but adapters must stay outside it.
- Adapters implement ports; ports must never import adapters.
- TUI code must never be imported by `internal/engine` or `internal/domain`.
- Persistence code must never define alternate copies of domain records.
- Provider adapters must return domain-facing decisions through ports, not leak SDK types upward.

## Determinism Boundary

Determinism lives in `internal/engine`.

That means:

- given the same input state, submitted actions, scenario data, and random draws, the engine must produce the same next state, events, and metrics
- scenario definitions are treated as immutable inputs to the engine, not mutable game state
- rule ordering is fixed and documented in one place
- legality checks and trimming are engine responsibilities, not UI or adapter responsibilities
- projections must be pure transforms of canonical records

Practical guidance:

- represent all MVP economic quantities as integers in domain types
- keep event ordering stable
- prefer explicit phase functions instead of hidden side effects
- record enough round metadata to replay or test a round deterministically

## Randomness Injection

Randomness must not be created ad hoc inside rules with package-global state such as `math/rand` defaults.

Instead:

- `internal/engine` consumes randomness through a `ports.RandomSource`
- the seeded implementation lives in `internal/adapters/random/seeded`
- application code injects the random source when creating a match or resolving a round
- seeds used for a match or round should be persisted alongside match metadata so simulations can be replayed

This lets the engine stay deterministic in tests while still supporting stochastic post-MVP scenarios.

Recommended shape:

```go
type RandomSource interface {
    Intn(n int) int
    Float64() float64
}
```

For MVP, the design should prefer deterministic scenario constants first and use injected randomness only where the rules explicitly call for it.

## Scenario Package Guidance

`internal/scenario` should hold content that changes the starting conditions or rule parameters of a match without changing the shared domain vocabulary.

Examples of values that belong there:

- product catalogs and BOM choices used by one scenario
- customer portfolios and price-sensitivity profiles
- workstation capacity baselines
- debt ceilings, holding-cost settings, and demand-tuning constants
- optional scenario flags that enable post-MVP mechanics once rules exist

Examples of post-MVP scenarios the package should be able to host:

- a high-volume commodity plant with thin margins and aggressive price sensitivity
- a custom-engineering plant with long lead times, low volume, and high backlog risk
- a constrained plant centered around one chronic bottleneck workstation
- a disruption-heavy scenario with unreliable supply and volatile demand
- a growth-phase scenario that introduces new products and management methods over time

Those examples are intentionally content-driven variations, not reasons to fork the engine package layout.

## UI And Prompt Projections

UI projections are built in `internal/projection`, not in the engine and not directly inside Bubble Tea components.

Why:

- the TUI and AI prompts both need shaped read models derived from the same canonical state
- a dedicated projection package keeps naming aligned with the domain model
- alternate front ends can reuse the same projection builders

Examples of projection outputs:

- round summary pane data
- metrics sidebar data
- event log rows
- role-specific action forms
- AI prompt context windows derived from `RoundView`

The TUI adapter renders projection outputs. It should not compute rules, mutate state, or invent alternate state naming.

## Feature Placement Guide

Use this checklist when adding a feature:

- If it changes game vocabulary or canonical record shape, start in `internal/domain`.
- If it changes starting conditions, scenario catalogs, or tunable scenario constants, place it in `internal/scenario`.
- If it changes how a round resolves, legality is enforced, or metrics are computed, change `internal/engine`.
- If it is a workflow that coordinates repositories, players, and projections, place it in `internal/app`.
- If it is a read-only view model for TUI, debugging, or prompts, place it in `internal/projection`.
- If it talks to SQLite, Bubble Tea, OpenRouter, Ollama, or another external library, place it in an adapter.
- If it is only an interface that core logic consumes, place it in `internal/ports`.

Examples:

- adding backlog expiry rules: `internal/engine`
- adding a new `CustomerSentimentMoved` field to a canonical event payload: `internal/domain`
- adding a "supplier disruption" scenario pack: `internal/scenario`
- adding a "finance dashboard" sidebar model: `internal/projection`
- adding an Ollama streaming client: `internal/adapters/ai/ollama`
- adding a "resume saved match" use case: `internal/app`

## Consequences

Positive:

- contributors get a clear home for new code before package drift begins
- the game engine stays reusable and testable without the TUI
- UI and AI integrations share projections instead of duplicating mapping logic
- randomness and time become controllable dependencies instead of hidden globals
- adapter churn stays outside the simulation core

Tradeoffs:

- simple features may touch several packages because the boundaries are explicit
- some early wrappers may feel thin until more implementation exists
- contributors must resist the temptation to put convenience logic in adapters

## Status Review Trigger

Revisit this ADR when any of the following become true:

- the project adds a network API or multiplayer service boundary
- multiple UIs need incompatible projection models
- post-MVP rules require a richer scenario or simulation subpackage split
- persistence needs event sourcing or snapshotting beyond the initial repository interfaces

Until then, contributors should treat this ADR as the default package placement guide.
