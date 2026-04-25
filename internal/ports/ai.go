package ports

import (
	"context"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

const AIDecisionContractVersion = "herbiego.ai.v1"

// RoleBriefing is the shared human-readable role description used for UI
// briefings and AI system instructions.
type RoleBriefing struct {
	RoleID                 domain.RoleID
	DisplayName            string
	PublicResponsibilities []string
	HiddenIncentives       []string
	DecisionPrinciples     []string
	AllowedActionSummary   []string
}

// AIDecisionRequest is the canonical provider-neutral input to the AI runner.
type AIDecisionRequest struct {
	ContractVersion string
	MatchID         domain.MatchID
	Round           domain.RoundNumber
	RoleID          domain.RoleID
	Provider        string
	Model           string
	Briefing        RoleBriefing
	RoundView       domain.RoundView
	RoleReport      domain.RoleRoundReport
	AllowedActions  AllowedActionSchema
	Tools           []LookupToolSpec
	ToolResults     []LookupToolResult
	ResponseSpec    ResponseFormatSpec
	RetryContext    *RetryFeedback
	PreviousAction  *domain.ActionSubmission
}

// AllowedActionSchema narrows the action payload to the active role.
type AllowedActionSchema struct {
	RoleID         domain.RoleID
	RequiredAction string
	JSONSchemaName string
	Rules          []string
}

// ResponseFormatSpec captures the stable response-format requirements.
type ResponseFormatSpec struct {
	RequireJSONOnly     bool
	AllowMarkdownFences bool
	MaxCommentaryChars  int
	MaxFocusTags        int
}

// RetryFeedback carries concise path-based validation failures into retries.
type RetryFeedback struct {
	Attempt          int
	ValidationErrors []ValidationError
	LastRawResponse  string
}

// ValidationError describes a single validation failure.
type ValidationError struct {
	Path    string
	Message string
}

// LookupToolSpec describes one read-only game lookup that an AI role may call
// before returning a final decision.
type LookupToolSpec struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Arguments   []LookupToolArgument `json:"arguments"`
}

// LookupToolArgument describes one tool argument.
type LookupToolArgument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// LookupToolCall is the JSON envelope used when a model requests a lookup.
type LookupToolCall struct {
	ToolName  string            `json:"tool_name"`
	Arguments map[string]string `json:"arguments"`
}

// LookupToolResult captures one executed tool lookup for prompt reuse and audit.
type LookupToolResult struct {
	ToolName  string            `json:"tool_name"`
	Arguments map[string]string `json:"arguments"`
	Result    any               `json:"result"`
}

// AIDecisionResponse is the strict JSON contract returned by the model.
type AIDecisionResponse struct {
	ContractVersion string             `json:"contract_version"`
	MatchID         domain.MatchID     `json:"match_id"`
	Round           domain.RoundNumber `json:"round"`
	RoleID          domain.RoleID      `json:"role_id"`
	Action          domain.RoleAction  `json:"action"`
	Commentary      AICommentary       `json:"commentary"`
}

// AICommentary is the public explanation returned by an AI role.
type AICommentary struct {
	PublicSummary string   `json:"public_summary"`
	FocusTags     []string `json:"focus_tags"`
}

// ProviderDecisionRequest is the provider-facing transport request built from
// the canonical AI decision contract.
type ProviderDecisionRequest struct {
	Provider            string
	Model               string
	SystemPrompt        string
	UserPrompt          string
	RequireJSONOnly     bool
	AllowMarkdownFences bool
}

// ProviderDecisionResult carries the raw model output back to the app layer.
type ProviderDecisionResult struct {
	RawResponse string
}

// DecisionClient executes provider requests without owning game semantics.
type DecisionClient interface {
	RequestDecision(ctx context.Context, request ProviderDecisionRequest) (ProviderDecisionResult, error)
}

// AIPlayerRunner decides one action through the shared AI orchestration path.
type AIPlayerRunner interface {
	Decide(ctx context.Context, request AIDecisionRequest) (domain.ActionSubmission, AIDecisionAudit, error)
}

// AICallRecord captures a single AI provider request/response exchange for debug inspection.
type AICallRecord struct {
	RoleID       domain.RoleID
	Round        domain.RoundNumber
	Attempt      int
	Provider     string
	Model        string
	SystemPrompt string
	UserPrompt   string
	RawResponse  string
	Valid        bool
	ErrorMessage string
}

// AIDecisionAudit captures retry and fallback behavior for debugging.
type AIDecisionAudit struct {
	AttemptCount     int
	UsedFallback     bool
	FallbackReason   string
	ValidationErrors []ValidationError
}
