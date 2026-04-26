# ADR 0003: AI Player Input And Output Contracts

- Status: Accepted
- Date: 2026-04-19
- Deciders: HerbieGo maintainers
- Related roadmap issue: `#5`

## Context

Issue `#5` asks for a concrete contract describing how AI-controlled roles receive context, return decisions, expose commentary, and recover from malformed responses.

The project already defines:

- [MVP Game Design](../mvp-game-design.md)
- [Canonical Domain Model](../domain-model.md)
- [ADR 0001: Initial Architecture And Package Boundaries](0001-initial-architecture.md)
- [ADR 0002: Simultaneous Action Collection And Resolution Flow](0002-simultaneous-action-collection-and-resolution.md)

Those documents define the canonical round view, the shared action envelope, and the timing rules for hidden simultaneous turns. What remains is the provider-neutral contract that the AI orchestration layer can hand to OpenAI-compatible providers such as OpenRouter or Ollama without changing game semantics.

This ADR defines that contract.

## Decision Summary

HerbieGo will use one provider-neutral AI contract with four layers:

1. a stable `RoleBriefing` shared by both human briefings and AI system instructions
2. a bounded `AIDecisionRequest` built from canonical domain projections
3. a strict `AIDecisionResponse` JSON object returned by the model
4. a deterministic validation and fallback loop owned by application code, not by provider adapters

Key decisions:

- AI roles consume the same canonical `RoundView` used by the UI rather than provider-specific prompt fields.
- AI roles must return a single machine-readable JSON response whose action payload matches the role they control.
- Commentary is structured as concise public explanation text plus machine-readable focus tags.
- Invalid model output triggers a fixed retry path with explicit validation feedback before any fallback is applied.
- Fallback behavior is deterministic: reuse the previous accepted action when available, otherwise submit a role-specific safe no-op.
- Token and window management is projection-driven, ordered, and truncation-safe so OpenRouter, Ollama, and other OpenAI-compatible providers can all receive materially equivalent context.
- AI and human-facing round context should preserve enough recent history and actual performance data to support four-round monthly reporting.

## Shared Contract Objects

The AI contract should be represented in the application and ports layers with stable structures like the following.

### Role Briefing

`RoleBriefing` is the human-readable role description that can be rendered in the TUI for a human player and also injected into an AI system prompt.

```go
type RoleBriefing struct {
    RoleID                 RoleID
    DisplayName            string
    PublicResponsibilities []string
    HiddenIncentives       []string
    DecisionPrinciples     []string
    AllowedActionSummary   []string
}
```

Rules:

- The same briefing content should be understandable without provider-specific prompt engineering.
- The briefing must not include current-round hidden actions from other roles.
- The briefing may describe incentives and priorities, but it must not redefine legal actions beyond the canonical domain model.

### AI Decision Request

`AIDecisionRequest` is the canonical input to an AI decision runner.

```go
type AIDecisionRequest struct {
    ContractVersion   string
    MatchID           MatchID
    Round             RoundNumber
    RoleID            RoleID
    Briefing          RoleBriefing
    RoundView         RoundView
    AllowedActions    AllowedActionSchema
    ResponseSpec      ResponseFormatSpec
    RetryContext      *RetryFeedback
    PreviousAction    *ActionSubmission
}
```

```go
type AllowedActionSchema struct {
    RoleID         RoleID
    RequiredAction string
    JSONSchemaName string
    Rules          []string
}

type ResponseFormatSpec struct {
    RequireJSONOnly        bool
    AllowMarkdownFences    bool
    MaxCommentaryChars     int
    MaxFocusTags           int
}

type RetryFeedback struct {
    Attempt             int
    ValidationErrors    []ValidationError
    LastRawResponse     string
}

type ValidationError struct {
    Path    string
    Message string
}
```

Request rules:

- `ContractVersion` must change only when response semantics change incompatibly.
- `RoundView` remains the source of current plant state, metrics, recent events, and recent commentary.
- `AllowedActions` narrows the action schema to the active role so the model is not asked to infer payload shape from prose alone.
- `RetryContext` is populated only on a retry after an invalid attempt.
- `PreviousAction` is supplied so fallback reuse can be explained consistently to the player and logs.

### AI Decision Response

Each AI-controlled role must return exactly one JSON object matching this shape.

```go
type AIDecisionResponse struct {
    ContractVersion string            `json:"contract_version"`
    MatchID         MatchID           `json:"match_id"`
    Round           RoundNumber       `json:"round"`
    RoleID          RoleID            `json:"role_id"`
    Action          RoleAction        `json:"action"`
    Commentary      AICommentary      `json:"commentary"`
}

type AICommentary struct {
    PublicSummary string   `json:"public_summary"`
    FocusTags     []string `json:"focus_tags"`
}
```

Required response rules:

- `contract_version`, `match_id`, `round`, and `role_id` must echo the request.
- `action` must populate only the payload corresponding to `role_id`.
- `commentary.public_summary` is required and should be short enough to reveal in the multiplayer log without further summarization.
- `commentary.focus_tags` is required and exists for UI filtering, reporting, or debugging; it must not contain private hidden information from other roles.
- `commentary.focus_tags` should summarize the primary concern driving the decision, such as `throughput`, `cash_discipline`, or `inventory_risk`.
- No free-form text may appear before or after the JSON object in the ideal path.

Recommended JSON shape by role:

- `procurement_manager` returns `action.procurement`
- `production_manager` returns `action.production`
- `sales_manager` returns `action.sales`
- `finance_controller` returns `action.finance`

### Example Response

```json
{
  "contract_version": "herbiego.ai.v1",
  "match_id": "match-001",
  "round": 4,
  "role_id": "procurement_manager",
  "action": {
    "procurement": {
      "orders": [
        {
          "part_id": "bearing_a",
          "supplier_id": "northsteel",
          "quantity": 40
        }
      ]
    }
  },
  "commentary": {
    "public_summary": "I replenished bearings to protect next round throughput while staying inside the active spend guardrails.",
    "focus_tags": ["throughput", "inventory_risk"]
  }
}
```

## Provider-Neutral Prompt Assembly

The application should assemble the request in fixed sections before handing it to a provider adapter:

1. contract header
2. role briefing
3. current round facts from `RoundView`
4. allowed action schema for the role
5. response-format instruction with JSON example
6. retry feedback, only when retrying

Provider adapters for OpenAI-compatible backends such as OpenRouter and Ollama may translate this into their preferred message transport, but they must not change the meaning of the contract.

Adapter rules:

- Adapters may choose chat-completion or generation APIs, but the application-level request and response schema stays the same.
- Adapters must not inject provider-specific decision fields into the returned payload.
- Adapters may request JSON mode or structured output features when available, but parsing must still tolerate plain-text transport around the same schema.

This keeps game semantics in HerbieGo code instead of in prompt templates hidden inside one provider integration.

## Validation Pipeline

AI responses are validated in four steps.

### 1. Transport Extraction

The decision runner extracts the first JSON object from the model output.

Rules:

- accept raw JSON directly
- accept JSON wrapped in one Markdown code fence
- reject outputs with no recoverable JSON object

### 2. Envelope Validation

Validate:

- `contract_version` matches the active version
- `match_id` matches the active match
- `round` matches the active round
- `role_id` matches the assigned role

### 3. Shape Validation

Validate:

- exactly one role action payload is populated
- the populated payload matches `role_id`
- all required fields exist
- numeric fields are integers
- commentary exists and fits the maximum length

### 4. Domain Reference Validation

Validate against known scenario/domain data:

- referenced parts, products, suppliers, customers, and workstations exist
- quantities and prices obey non-negative requirements
- no post-schema field rewriting occurs inside adapters

If all four steps pass, the response is converted into the canonical `ActionSubmission` plus `CommentaryRecord` structures already defined by the domain model.

## Deterministic Recovery Path

Malformed AI output must not create undefined behavior.

The application owns this recovery algorithm:

1. request a decision using the current `AIDecisionRequest`
2. parse and validate the result
3. if invalid and time remains, retry with `RetryContext` containing the specific validation errors
4. cap retries at a small fixed count per round attempt
5. if no valid response is produced, apply deterministic fallback

Recommended MVP defaults:

- initial attempt plus `2` retries
- one shared timeout budget for the whole decision attempt, not per retry
- validation errors should be concise path-based messages rather than long prose

Fallback rules:

- if the role has a previously accepted action from the prior round, reuse it
- if there is no previous accepted action, emit a role-specific safe no-op
- always emit a machine-readable reason in logs when fallback occurs

Recommended safe no-op defaults:

- procurement: empty `orders`
- production: empty `releases` and empty `capacity_allocation`
- sales: empty `product_offers`
- finance: repeat the currently active targets exactly into `next_round_targets`

The fallback chosen for round `R` must be independent of provider brand, latency race conditions, or non-deterministic parser behavior.

## Commentary Contract

AI commentary is part of the social and debugging layer, but it is not hidden chain-of-thought.

Rules:

- `public_summary` is the only required explanation field
- it should be `<= 280` characters in MVP
- it should describe intent in player-facing language, not raw JSON restatement
- it becomes visible only when the round record is revealed under ADR `0002`
- it should be safe to show to human and AI players after resolution without redaction

Non-goals:

- no hidden private reasoning field
- no requirement to persist full internal deliberation text
- no provider-specific reasoning token handling in the canonical contract

This keeps explainability useful without making the game depend on opaque provider-specific reasoning features.

## Token And Window Limits Strategy

The prompt builder must shrink context by projection, not by arbitrary truncation of serialized state.

### Stable Context Budget

Split the input budget into fixed bands:

- role briefing band
- current round snapshot band
- recent history band
- response schema band
- retry feedback band

Recommended MVP strategy:

- keep the full role briefing
- keep the full current-round facts needed for the role
- keep a bounded recent history window large enough to support monthly performance reasoning
- summarize older history into durable trend bullets before inclusion

### Ordering Rules

To keep truncation deterministic:

- sort repeated entities by canonical identifier unless domain rules specify another order
- serialize repeated records in the same order used by `RoundView`
- trim oldest history first
- preserve schema instructions even when history must be shortened

### History Rules

The MVP prompt builder should prefer:

- the last `4` fully revealed rounds in structured form so players and AI can reason about a monthly window
- actual performance values for those rounds rather than commentary-only history
- last-month and month-to-date performance summaries derived from those actuals
- older history collapsed into compact summary bullets or omitted entirely
- only the most relevant commentary excerpts rather than every prior message

### Hard Limits

Each provider adapter should expose its effective input budget, but the canonical strategy is:

- reserve output capacity for the JSON response
- reserve retry budget so one invalid answer does not exhaust the whole window
- fail closed into deterministic fallback if a compliant request cannot fit after projection

This means the prompt builder decides what information is essential, while provider adapters only expose practical transport limits.

## Implementation Guidance

Contributors implementing AI play should treat the following as the minimum contract:

- `internal/projection` produces the bounded `RoundView` and any history summaries
- `internal/app` assembles `AIDecisionRequest`, runs retry orchestration, and applies fallback
- `internal/ports` defines provider-neutral decision interfaces
- `internal/adapters/ai/openai` translates the shared request into OpenAI-compatible provider API calls
- adapters return raw model text or parsed JSON, but application code owns validation and conversion into domain actions

Suggested port shape:

```go
type DecisionClient interface {
    RequestDecision(ctx context.Context, request ProviderDecisionRequest) (ProviderDecisionResult, error)
}
```

```go
type AIPlayerRunner interface {
    Decide(ctx context.Context, request AIDecisionRequest) (ActionSubmission, AIDecisionAudit, error)
}
```

```go
type AIDecisionAudit struct {
    AttemptCount    int
    UsedFallback    bool
    FallbackReason  string
    ValidationErrors []ValidationError
}
```

## Consequences

Positive:

- OpenRouter, Ollama, and other OpenAI-compatible integrations can share one decision contract instead of duplicating game semantics
- human role briefings and AI role instructions come from the same source material
- invalid model output has a deterministic recovery path instead of ad hoc parser behavior
- prompt assembly remains aligned with canonical domain projections and round timing

Tradeoffs:

- strict JSON contracts are less flexible than free-form prompting
- prompt builders need deliberate context budgeting work instead of dumping raw state
- some nuanced strategy context may be omitted when the history window is aggressively bounded

## Status Review Trigger

Revisit this ADR when any of the following become true:

- the project adopts tool calling or function calling as a mandatory provider feature
- hidden/private commentary classes are introduced
- the MVP moves from one-action-per-role to multi-step negotiation or interrupt mechanics
- multimodal context such as charts or screenshots becomes part of the AI contract

Until then, contributors should treat this ADR as the canonical answer for AI player request, response, validation, and fallback semantics.
