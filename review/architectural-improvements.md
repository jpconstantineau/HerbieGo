# Architectural Improvements

This document identifies architectural gaps and improvement opportunities in HerbieGo. The goal is to enable new features to be added with minimal cross-cutting changes and to improve long-term maintainability. No application code changes are proposed here — these are design-level recommendations for developer consideration.

Each item includes: the current state, the problem it causes, and a recommended direction.

---

## 1. Role Registry — Eliminate Switch-on-RoleID Fan-Out

### Current State

`domain.CanonicalRoles()` returns a fixed hardcoded list. Role-specific behavior is scattered across multiple `switch roleID` blocks in:
- `internal/prompting/ai.go` — `exampleAction`, `roleObjective`, `roleDecisionScope`, `groupedToolSections`, `toolCategories`
- `internal/projection/role_report.go` — `buildDepartmentPerformanceReport`, `bonusReminder`
- `internal/app/config.go` — `preferredHumanRoleOrder`, `validateRoleConfig`
- `internal/adapters/player/llm/player.go` — `safeNoOpAction`

Adding a new role (e.g., Plant Manager) today requires identifying and updating all of these switch statements across four packages.

### Problem

This is a violation of the Open/Closed Principle. The codebase is closed for extension without modification. Each new role is a multi-package change that is easy to get incomplete.

### Recommended Direction

Introduce a `RoleDescriptor` or `RoleProfile` value type that bundles all role-specific behavior in one place:

```
RoleDescriptor {
  RoleID
  DisplayName
  Briefing (RoleBriefing)
  PreferredAsHuman bool / HumanPriority int
  DepartmentReportBuilder func(MatchState) DepartmentPerformanceReport
  SafeNoOpAction func(RoleView) RoleAction
  ExampleAction func(BudgetTargets) RoleAction
  DecisionScope string
}
```

Register all descriptors in one location (e.g., `domain/roles.go` or a new `internal/roles` package). All switch-on-roleID blocks become lookups against the registry. New roles are added by registering a new descriptor — no other files need to change.

---

## 2. Scenario Registry — Decouple Active Scenario from Bootstrap

### Current State

`app.NewRuntime` calls `scenario.Default()`, which always returns the single `Starter` scenario. The active scenario is hardwired at process startup with no way to select or switch scenarios at runtime without recompiling.

### Problem

- Adding a second scenario requires editing `app.Bootstrap` or `app.NewRuntime`.
- There is no multi-scenario selection path (config, flag, lobby) even though the game's long-term vision includes many scenarios.
- Tests that want a custom scenario must create a `Runtime` by hand.

### Recommended Direction

Introduce a `ScenarioRegistry` (or simply a `map[ScenarioID]Definition`) in the scenario package, populated at init time. `app.Config` gains a `ScenarioID` field. `Bootstrap` looks up the scenario from the registry instead of calling `scenario.Default()` directly. This decouples scenario selection from the wiring code and makes it possible to expose a scenario picker in the TUI or CLI without touching the app layer.

---

## 3. Provider Factory — Remove SDK-Type Switch from Entry Point

### Current State

`cmd/herbiego/players.go` contains a `switch roleCfg.APISDKType` that constructs concrete AI provider clients:
```go
switch roleCfg.APISDKType {
case app.APISDKTypeOllama:
    // ollama.New(...)
case app.APISDKTypeOpenAI:
    // openai.New(...)
default:
    return nil, fmt.Errorf(...)
}
```

### Problem

- Adding a new AI provider (e.g., Anthropic, Google Gemini, AWS Bedrock) requires editing `cmd/herbiego/players.go`, `app/config.go` (add new `APISDKType` constant), and the adapter test files.
- The construction logic lives in `cmd`, not in the `adapters/ai` package where it belongs.

### Recommended Direction

Move the construction logic into `internal/adapters/ai`. Define a `ProviderFactory` or a registration function:

```go
// In adapters/ai
type ClientFactory func(cfg ClientConfig) (ports.DecisionClient, error)

var factories = map[app.APISDKType]ClientFactory{}

func Register(sdkType app.APISDKType, factory ClientFactory) { ... }
func Build(cfg ClientConfig) (ports.DecisionClient, error) { ... }
```

Each provider package registers itself. The wiring in `cmd` shrinks to a single `ai.Build(cfg)` call. New providers are added by writing one adapter package and one `Register` call.

---

## 4. MatchStateStore Integration — Persist Through the Game Loop

### Current State

`internal/ports/state_store.go` defines `MatchStateStore` and `internal/adapters/persistence/memory` implements it. However, `MatchRunner.Play` never uses the store. State is carried entirely in the local variable inside the loop. The store is effectively unused in the live game path.

### Problem

- There is no persistence of completed rounds or match state across process restarts.
- The `SQLite` adapter (`adapters/persistence/sqlite`) is a placeholder — no implementation exists.
- Replay, resume, and audit features cannot be built without first integrating the store.
- The `memory.Store` is tested in isolation but its interface contract is not exercised in the game loop.

### Recommended Direction

Add an optional `Store ports.MatchStateStore` field to `MatchRunner`. When present, `Play` calls `store.CommitRound(matchID, nextState, round)` after each resolved round. This is a single insertion point — no other layers need to change. The SQLite implementation can then be completed and wired in when persistence is needed.

---

## 5. Prompting — Decouple Scenario-Specific Content from Prompt Templates

### Current State

`internal/prompting/ai.go` embeds game-specific strings directly:
- `exampleAction` returns hardcoded part IDs (`"housing"`, `"forgeco"`, `"pump"`, `"valve"`) pulled from the Starter scenario
- `groupedToolSections` hardcodes the category names `"Parts"`, `"Products"`, `"Vendors"`, `"Customers"`
- `toolCategories` maps tool names to categories with a hardcoded switch

When a second scenario with different products or parts is added, the example actions will be incorrect for that scenario's vocabulary.

### Recommended Direction

Pass scenario-derived example action data into `BuildSystemPrompt` and `BuildUserPrompt` rather than hardcoding it. The `AIDecisionRequest` already carries `RoundView` (which contains active products and suppliers); an example action generator can derive appropriate example values from that context. Alternatively, make the example action part of the `AllowedActionSchema` or `RoleBriefing` so the calling layer (which knows the scenario) provides it.

---

## 6. Demand and Market Model — Make World Update Composable

### Current State

`scenario.go`'s `applyDemand` method and `currentOfferPrices` helper embed the demand realization formula directly. The formula is:

```
demandScore = baseDemand * (sentiment + 2) - priceSensitivity * max(0, offeredPrice - referencePrice)
realizedUnits = max(0, demandScore / 5)
```

This is a single baked-in demand model for all scenarios.

### Problem

- Adding an alternative demand model (e.g., seasonal demand, demand shocks, competitor pricing) requires forking `applyDemand` or adding branches inside it.
- There is no way to compose or replace only the demand calculation without replacing the entire `WorldUpdateHook`.

### Recommended Direction

Define a `DemandModel` interface or function type:

```go
type DemandCalculator func(profile DemandProfile, sentiment int, offeredPrice Money) Units
```

Make `MarketModel` carry the calculator, defaulting to the current formula. Scenarios can override the calculator without changing any engine or scenario infrastructure. This also makes it straightforward to test demand model variants in isolation.

---

## 7. Round Metrics — Separate Accumulation from Domain

### Current State

`resolutionStats` is a private struct inside `engine/resolver.go` that accumulates metrics during resolution. At the end of resolution, its fields are mapped into `domain.PlantMetrics`. Both structs contain similar fields. Any new KPI requires adding it to both structs plus wiring it through the mapping.

### Problem

Duplication between `resolutionStats` and `domain.PlantMetrics` creates a maintenance burden. It is easy to add a metric to one and forget the other.

### Recommended Direction

Consider making `domain.PlantMetrics` itself the accumulator, or introduce a builder/accumulator pattern that produces a `PlantMetrics` directly, eliminating the mapping step. Alternatively, generate the mapping via a single helper function that is the only place both types are referenced together, making omissions immediately visible.

---

## 8. Commentary Visibility — Prepare for Role-Scoped Private Commentary

### Current State

`domain.CommentaryVisibility` has only one defined value (`CommentaryPublic`). The declaration comment says: _"CommentaryPublic is the only MVP visibility class; all stored commentary is public after reveal."_

The `CommentaryRecord` struct carries a `Visibility` field, and `commentaryVisibility()` in `app/round_collection.go` defaults all commentary to public.

### Problem

The game design explicitly involves hidden incentives and role-scoped information. Private (intra-role or to-facilitator) commentary is a natural near-term feature. However:
- `BuildRoundView` does not filter commentary by role or visibility.
- `projection/round_view.go` copies all commentary into `RoundView.RecentCommentary` without filtering.
- Adding filtered visibility today requires changes to projection, the domain view type, and potentially the store.

### Recommended Direction

Define the additional visibility constants now (`CommentaryPrivate`, `CommentaryFacilitator`, etc.) even if they are not yet used. Add a `FilterCommentary(roleID RoleID) RoundView` method or a visibility filter in `BuildRoundView`. This keeps the schema stable and makes the future feature a small targeted change rather than a structural addition.

---

## 9. MatchID Generation — Remove Hardcoded `"starter-match"` String

### Current State

`app.NewRuntime` calls:
```go
initialMatch := starter.InitialState("starter-match", runtimeRoles(cfg))
```

### Problem

Every process run uses the same match ID. If persistence is ever added, this will cause `CreateMatch` to fail with "match already exists" on the second run. It also makes it impossible to distinguish match records in any audit or replay system.

### Recommended Direction

Generate a unique match ID at startup — e.g., a UUID, a timestamp-based ID, or a hash of `(scenarioID + seed + timestamp)`. The ID could also be made configurable via `Config` or `BootstrapOptions` to support deterministic test fixtures.

---

## 10. Config Validation Coupling to Domain

### Current State

`app.Config.Validate()` calls `domain.CanonicalRoles()` to enforce that exactly the canonical roles are configured. `Config.normalize()` also calls `domain.CanonicalRoles()` when iterating roles.

### Problem

If roles ever become scenario-configurable or the canonical role set varies by scenario, this validation becomes incorrect. The config layer should not be the enforcer of domain role identity.

### Recommended Direction

Pass the expected role set as a parameter to validation, or validate role completeness in the runtime layer (which knows both the config and the scenario's `MatchSetup.RoleRoster`) rather than in config alone. This makes `Config` itself scenario-agnostic.

---

## 11. Error Handling — Adopt Sentinel Error Types

### Current State

Most errors are created inline with `fmt.Errorf("...")`. Two sentinel errors exist: `ports.ErrNonResponsive` and `memory.ErrMatchNotFound`. Error detection in `llm/player.go` uses both `errors.Is` and a string-contains fallback:

```go
func nonResponsive(err error) bool {
    return errors.Is(err, ports.ErrNonResponsive) ||
        errors.Is(err, context.DeadlineExceeded) ||
        strings.Contains(strings.ToLower(err.Error()), "timeout")
}
```

### Problem

- String-based error detection is fragile and breaks with error message changes or localization.
- Callers cannot programmatically distinguish between "player timed out" and "AI provider returned a 500 error" without inspecting error messages.
- Adding structured error handling (retry policies, alerting, user-facing messages) is harder without typed errors.

### Recommended Direction

Define sentinel or typed errors for the main failure categories in `ports` and in the adapter packages. Replace string-contains checks with `errors.Is` / `errors.As`. A simple `var ErrProviderTimeout = errors.New(...)` in the AI adapter packages would be sufficient for the most common case.

---

## 12. TUI Dependency on Scenario

### Current State

`internal/adapters/tui/model.go` imports `internal/scenario` to display scenario-specific names and call scenario lookup methods. The `Model` struct holds a `scenario.Definition` field.

### Problem

The TUI is coupled to a concrete scenario type. This means the TUI cannot be easily tested with a mock scenario, and it creates an indirect dependency from the presentation layer back through the scenario layer into domain.

### Recommended Direction

Define a `ScenarioReader` interface in the TUI package (or in `ports`) containing only the methods the TUI needs (e.g., `Parts()`, `Products()`, `Workstations()`, `LookupTools()`, `ExecuteLookup()`). The TUI depends on the interface; the caller provides the concrete `scenario.Definition` at wiring time. This follows the Dependency Inversion Principle and makes the TUI unit-testable without a fully constructed scenario.

---

## 13. Single Active Scenario Per Runtime

### Current State

`Runtime` has a single `Scenario scenario.Definition` field. A match is always started with `scenario.Default()`.

### Problem

The game vision includes many scenarios and a long-running simulation. There is no mechanism to:
- Select a scenario at match creation time
- Run multiple matches with different scenarios concurrently
- Support a lobby or match-creation flow

### Recommended Direction

This is partly addressed by items 2 (Scenario Registry) and 9 (MatchID generation). The additional step is to ensure `MatchRunner` receives the scenario to use rather than having `Runtime` own it exclusively. Making `Resolver` (or its `Options`) part of the match record (not the runtime) enables multiple simultaneous matches with different scenarios.

---

## 14. AI Prompt Examples Tied to Starter Scenario Vocabulary

### Current State

System and user prompts include hardcoded JSON examples with Starter scenario entity IDs (`"housing"`, `"forgeco"`, `"pump"`, `"valve"`). These are rendered regardless of which scenario is active.

### Problem

When a second scenario is added with different products and parts, the AI will be shown an example that references entities that do not exist in the active scenario. This may confuse the model and produce invalid actions (referencing non-existent part IDs).

### Recommended Direction

Generate the example action JSON from the active scenario's catalog — picking the first available product and supplier IDs rather than hardcoding them. This is a small change to `prompting/ai.go` that takes example entity IDs from `request.RoundView` or from `AllowedActionSchema.Rules`, both of which are already in the `AIDecisionRequest`.

---

## 15. Lack of Observability / Structured Logging

### Current State

There is no logging, tracing, or metrics instrumentation in the codebase. Errors surface through Go error returns; there is no structured event log visible to operators outside the game's own event stream.

### Problem

- AI decision retries, fallbacks, and tool call counts are captured in `AIDecisionAudit` but are not persisted or surfaced outside the current call.
- There is no way to diagnose slow or failing AI provider calls without reading source code.
- Future features such as operator dashboards, performance analytics, or alerting on AI fallback rates have no foundation.

### Recommended Direction

Inject a `slog.Logger` (standard library in Go 1.21+) into key types (`AIOrchestrator`, `RoundCollector`, `MatchRunner`). Log significant events at `Debug` or `Info` level: round completion, AI retries, fallbacks, provider errors, tool calls. Keep log calls behind the interface so tests can use a `slog.DiscardHandler`. This is a low-cost change that pays dividends immediately in debugging and long-term in observability.
