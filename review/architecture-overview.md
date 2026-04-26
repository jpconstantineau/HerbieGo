# HerbieGo Architecture Overview

This document describes the current architecture of HerbieGo as implemented, covering layer responsibilities, key data flows, and design patterns in use.

---

## Architectural Style

HerbieGo follows a **Hexagonal Architecture** (Ports and Adapters, also called Clean Architecture). The goal is to isolate the game's core domain logic from infrastructure concerns such as the terminal UI, AI providers, and persistence backends.

```
┌────────────────────────────────────────────────────────────────────┐
│                         cmd/herbiego (main)                        │
│  CLI flags → Bootstrap → wire players → run game loop → TUI       │
└────────────────────────────────────────────────────────────────────┘
           │                                             │
           ▼                                             ▼
┌─────────────────────┐                    ┌────────────────────────┐
│    internal/app      │                    │   internal/adapters    │
│  Config, Runtime     │                    │  tui / ai / player /   │
│  MatchRunner         │◄──ports────────────│  persistence / random  │
│  RoundCollector      │                    └────────────────────────┘
│  AIOrchestrator      │
└─────────────────────┘
           │
           ▼
┌─────────────────────┐     ┌──────────────────────────┐
│   internal/engine   │     │   internal/scenario      │
│   Resolver          │◄────│   Definition (hooks)     │
└─────────────────────┘     └──────────────────────────┘
           │
           ▼
┌─────────────────────┐
│   internal/domain   │
│   MatchState, types │
└─────────────────────┘
```

---

## Package Inventory

### `internal/domain`

The **domain model** holds all typed value objects and aggregates. It has no dependencies on any other internal package.

Key types:
- `MatchState` — canonical in-memory game snapshot for the round being collected
- `RoundRecord` — immutable record of one completed round (actions, events, commentary, metrics)
- `RoleAction` — a union of the four role-specific action payloads (`Procurement`, `Production`, `Sales`, `Finance`)
- `RoundView` — player-facing projection of `MatchState` (not filtered per role in MVP — all roles see all data)
- `PlantMetrics` — all KPI fields accumulated during round resolution
- `RoundEvent` / `RoundEventType` — strongly-typed event stream emitted by the engine

**Design pattern:** Every aggregate and collection type implements `Clone()` methods using `slices.Clone` and `maps.Clone`. This allows safe immutable passing of state between concurrent goroutines without defensive copying at every call site.

---

### `internal/ports`

**Port interfaces** define the contracts between the application layer and its adapters. Nothing in this package has knowledge of any concrete implementation.

| Interface | Responsibility |
|---|---|
| `Player` | Accepts a `RoundRequest`, returns `ActionSubmission` (human or AI) |
| `DecisionClient` | Executes one provider-specific LLM call; returns raw text |
| `AIPlayerRunner` | Orchestrates the full AI decision cycle including retries |
| `MatchStateStore` | Persists and retrieves canonical match state and timelines |
| `RandomSource` | Provides deterministic, seedable random numbers |

Key types in this package:
- `AIDecisionRequest` / `AIDecisionResponse` — the versioned JSON contract exchanged with AI providers
- `ProviderDecisionRequest` / `ProviderDecisionResult` — the provider-neutral wire format
- `LookupToolSpec` / `LookupToolCall` / `LookupToolResult` — tool-use contract for AI agents
- `RoleBriefing` — shared description of a role used for both UI display and AI system prompts
- `RetryFeedback` — carries validation failures into retry attempts
- `AIDecisionAudit` — debugging record of retries and fallback use

---

### `internal/engine`

The **deterministic round resolver**. Given a `MatchState`, a slice of `ActionSubmission` values, and a `RandomSource`, `Resolver.ResolveRound` produces a new `MatchState` and a `RoundRecord`.

The engine is **purely functional from the outside**: callers pass immutable state in, receive immutable state out. Internal mutations are scoped to local `roundPhase` structs.

**Hook pattern:** The engine accepts `Options` containing function-valued hooks:
- `ProcurementTermsHook` — supplies supplier lead times, unit costs, on-time percentages
- `ProductionBOMHook` — returns parts consumed per product release
- `ProductionRouteHook` — returns the next workstation in a product's route
- `ProductionCostHook` — returns cost per capacity unit used
- `InventoryCarryingCostHook` — returns holding cost for each inventory class
- `WorldUpdateHook` — runs scenario-owned demand generation after player actions resolve

This allows `scenario.Definition` to supply all scenario-specific data without the engine importing the scenario package, maintaining clean dependency direction.

Round resolution order (within `ResolveRound`):
1. Normalize and validate actions
2. Activate budget targets if applicable
3. Resolve procurement (place purchase orders, schedule payables)
4. Receive in-transit supply (arrive or delay)
5. Resolve production (release work, advance WIP, consume capacity)
6. Resolve sales (compute shipments, schedule receivables, update backlog)
7. World update (hook) — demand realization, market events
8. Finalize round — collect receivables, pay payables, compute metrics, snapshot
9. Advance round counter, prune history to configured limit

---

### `internal/scenario`

Defines the **`scenario.Definition`** struct, the `Starter` scenario, and the lookup surface.

`Definition` groups five independently addressable sub-models:
- `MatchSetup` — role roster
- `StartingConditions` — initial plant state, customers, budgets
- `MarketModel` — customer demand profiles and backlog expiry
- `ProductionModel` — products, parts, suppliers, workstations, bottleneck assumption
- `FinanceModel` — receivable/payable delay rounds

`Definition.ResolverOptions()` builds the concrete `engine.Options` by closing over scenario data inside the hooks, so the engine never imports the scenario package.

`Definition.InitialState()` builds the first `MatchState` from starting conditions.

`lookups.go` provides both the human-browsable and AI-callable lookup surface (`ListValidSuppliers`, `ShowProductBOM`, `ShowProductRoute`, `ShowCustomerDemandProfile`, `ExecuteLookup`). The lookup tool list (`LookupTools()`) is the single source of truth used by both `AIOrchestrator.BuildRequest` and the TUI lookup browser.

---

### `internal/app`

The **application service layer**. Orchestrates game flow using domain types and port interfaces without owning infrastructure.

Key types:

**`Config` / `LLMCatalog`**: YAML-backed configuration. `Config.normalize()` builds the `map[RoleID]RoleConfig` from the flat YAML role list by joining with catalog entries. `BootstrapOptions` allow CLI overrides.

**`Runtime`**: Process-level dependency container populated once at startup:
- `Config` — validated runtime config
- `Random` — seeded deterministic random source
- `Scenario` — the active scenario definition
- `InitialMatch` — the first `MatchState` ready for play

**`MatchRunner`**: Drives the match loop. The `Play` method iterates for `N` rounds:
1. Emits collection-phase state via `OnState`
2. Delegates to `RoundCollector.Collect` (concurrent per role)
3. Emits resolving-phase state
4. Calls `engine.Resolver.ResolveRound`
5. Emits revealed-phase state
6. Calls `OnRound` callback
7. Advances to next round

**`RoundCollector`**: Collects one action per role by calling `Player.SubmitRound` in parallel using `golang.org/x/sync/errgroup`. Tracks round-flow progress (which roles have submitted, which are waiting on AI) and emits intermediate `RoundFlowState` snapshots via `OnRoundFlow`.

**`AIOrchestrator`**: Bridges the `RoundCollector` → AI path. It:
1. Builds a `ports.AIDecisionRequest` from the round request (briefings, allowed schemas, tools, response spec)
2. Builds system and user prompts via `internal/prompting`
3. Calls `ports.DecisionClient.RequestDecision`
4. Parses and validates the JSON response
5. Handles tool-call round trips (up to `MaxToolCalls`)
6. Retries on validation failure (up to `MaxAttempts`)
7. Falls back to a safe no-op if all attempts fail

---

### `internal/projection`

Stateless **read-model builders** that transform `MatchState` into player-facing views.

- `BuildRoundView` — builds `domain.RoundView` from state and a viewer role ID (MVP: no per-role information filtering)
- `BuildRoleRoundReport` — builds `domain.RoleRoundReport` with role-specific department metrics, detail lines, and a bonus reminder

These functions are pure (no side effects) and are called by `RoundCollector.Collect` before dispatching to each `Player`.

---

### `internal/prompting`

**Prompt assembly** for the AI decision cycle.

- `BuildSystemPrompt` — renders a Markdown role briefing, decision principles, tool catalog, and JSON response example
- `BuildUserPrompt` — renders the per-round context window: contract header, role briefing, round view, allowed action schema, response format spec, tool results (if any), and retry feedback (if applicable)

Both functions only depend on `ports.AIDecisionRequest`, keeping prompt logic separated from transport and game logic.

---

### `internal/adapters`

Infrastructure adapters. Each sub-package implements one or more port interfaces.

#### `adapters/ai`

- `RoutingClient` — implements `ports.DecisionClient`; dispatches to the correct concrete provider client by normalized provider name
- `adapters/ai/openai` — OpenAI-compatible `/chat/completions` client; supports API key and configurable base URL (works for OpenRouter, Ollama, and other OpenAI-compatible endpoints)
- `adapters/ai/openrouter` — placeholder package (`doc.go` only; OpenRouter is reached via the OpenAI adapter)

#### `adapters/player`

- `adapters/player/human` — wraps a `SubmitFunc` callback; forwards to the TUI controller
- `adapters/player/llm` — wraps an AI submit function; adds a fallback policy (reuse previous action or safe no-op) when the AI fails to respond

#### `adapters/persistence`

- `adapters/persistence/memory` — in-process `MatchStateStore` using a mutex-guarded map; separates recent history from the full event/commentary append log
- `adapters/persistence/sqlite` — placeholder package (`doc.go` only)

#### `adapters/tui`

Bubble Tea terminal UI. Key types:
- `Model` — the Bubble Tea model; holds state, manages pane focus, workspace mode (action entry / scenario lookup / role report / round feed / history archive), drafts, and live state subscription
- `StateSource` interface — `Snapshot() MatchState` + `Updates() <-chan MatchState`; decouples the model from whether state comes from a live game or a static snapshot
- `SubmitFunc` — `func(domain.ActionSubmission) error`; dependency-injected at construction time

#### `adapters/random/seeded`

Implements `ports.RandomSource` using a deterministic PRNG seeded from `Config.Random.Seed`.

---

### `cmd/herbiego`

The **entry point and wiring layer**.

- `main.go` — parses CLI flags, calls `app.Bootstrap`, builds players, creates `MatchRunner`, starts TUI program, and coordinates goroutine lifecycle
- `players.go` — constructs `ports.Player` implementations per role: `human.Player` for human roles, `llm.Player` wrapping `AIOrchestrator.SubmitRound` for AI roles; builds concrete `DecisionClient` instances for configured OpenAI-compatible providers
- `live_gameplay_controller.go` — the bidirectional bridge between the game loop goroutine and the TUI goroutine: implements `StateSource` (for TUI subscription), exposes `Publish` (for `MatchRunner.OnState`), and routes human action submissions through a channel

---

## Key Data Flows

### 1. Bootstrap and Startup

```
CLI flags
  → app.Bootstrap
      → LoadConfig("herbiego.yaml")
      → LoadLLMCatalog("llm.yaml")
      → Config.normalize() (join roles with catalog)
      → Config.Validate()
      → scenario.Default() (hardwired to Starter scenario)
      → Definition.InitialState(matchID, roles)
      → seeded.New(seed)
  ← Runtime{Config, Random, Scenario, InitialMatch}
```

### 2. Game Loop (one round)

```
MatchRunner.Play(ctx, initialState, rounds)
  │
  ├─ emitState(collectingPhase)
  │
  ├─ RoundCollector.Collect(ctx, state, previous)
  │    ├─ [parallel per role]
  │    │    projection.BuildRoundView(state, roleID)
  │    │    projection.BuildRoleRoundReport(state, roleID)
  │    │    Player.SubmitRound(ctx, RoundRequest)
  │    │         ├─ human.Player → liveGameplayController.SubmitRound → channel ← TUI
  │    │         └─ llm.Player → AIOrchestrator.Decide
  │    │              prompting.BuildSystemPrompt/BuildUserPrompt
  │    │              RoutingClient.RequestDecision → HTTP → AI provider
  │    │              parseAndValidateDecision
  │    │              [retry / tool calls / fallback]
  │    └─ []ActionSubmission
  │
  ├─ emitState(resolvingPhase)
  │
  ├─ engine.Resolver.ResolveRound(state, actions, random)
  │    resolveProcurement → receiveSupply → resolveProduction
  │    → resolveSales → worldUpdate(demand) → finalizeRound
  │  ← Result{NextState, Round}
  │
  ├─ emitState(revealedPhase)
  ├─ emitRound(result)
  └─ advance to next round
```

### 3. Human Player Submission (via TUI)

```
TUI Model (user types action)
  → Submit callback (SubmitFunc)
  → liveGameplayController.Submit(ActionSubmission)
  → buffered channel (submissions)

liveGameplayController.SubmitRound (called by human.Player)
  → drain channel; match by (roleID, round)
  → return ActionSubmission to RoundCollector
```

### 4. State Streaming to TUI

```
MatchRunner.OnState callback
  → liveGameplayController.Publish(MatchState)
  → buffered channel (updates, capacity 8)
  → TUI Model.Update(stateLoadedMsg)
  → re-render
```

---

## Concurrency Model

- All roles in a round are collected **in parallel** using `errgroup`.
- `liveGameplayController` is safe for concurrent access via a `sync.Mutex`.
- `memory.Store` uses a `sync.RWMutex` (though it is not currently wired into the live game loop — state is carried in `MatchRunner.Play`).
- State passed between goroutines is always deep-cloned via `Clone()`.
- The TUI runs in one goroutine; the game loop runs in a separate goroutine; they communicate through channels.

---

## Configuration Model

Two YAML files:

**`herbiego.yaml`** — runtime config:
- `environment` — label (default: `local`)
- `random.seed` — deterministic PRNG seed
- `human_players` — how many of the canonical roles are human-controlled (assigned in `preferredHumanRoleOrder`)
- `ui.ai_reveal_delay_seconds` — pause between collecting and revealing AI decisions
- `roles[]` — list of `{role_id, provider, model}` entries

**`llm.yaml`** — LLM catalog (connection metadata):
- `models[]` — list of `{provider_name, model_name, url, api_sdk_type, api_key}` entries, where `url` includes the provider-specific OpenAI-compatible base path

Config normalization joins the role list with catalog entries, then overrides the `human_players` count of roles to `PlayerKindHuman` in `preferredHumanRoleOrder` order.

---

## Testing Approach

- Unit tests exist for `domain`, `engine`, `app` (config, runtime, round collection, AI orchestration, MVP flow), `scenario`, `projection`, and the AI/TUI adapters.
- The `cmd/quality` tool runs `gofmt`, `go test ./...`, and `staticcheck`.
- No integration tests hit live AI providers; adapter tests use fake HTTP servers or hand-crafted JSON responses.
