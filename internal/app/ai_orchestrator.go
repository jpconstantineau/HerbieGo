package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/actionschema"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/prompting"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

const (
	defaultAIMaxAttempts   = 3
	defaultAIMaxCommentary = 280
	defaultAIMaxFocusTags  = 4
	defaultAIMaxToolCalls  = 2
)

// AIOrchestrator assembles provider-neutral decision requests, executes them
// through a narrow transport client, and validates the returned JSON contract.
type AIOrchestrator struct {
	Client       ports.DecisionClient
	Scenario     scenario.Definition
	MaxAttempts  int
	MaxToolCalls int
	DebugLog     *DebugLog
	Logger       *slog.Logger
}

func NewAIOrchestrator(definition scenario.Definition, client ports.DecisionClient) AIOrchestrator {
	return AIOrchestrator{
		Client:       client,
		Scenario:     definition,
		MaxAttempts:  defaultAIMaxAttempts,
		MaxToolCalls: defaultAIMaxToolCalls,
	}
}

// SubmitRound adapts the shared round request into the AI decision runner.
func (o AIOrchestrator) SubmitRound(ctx context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
	submission, _, err := o.Decide(ctx, o.BuildRequest(request))
	return submission, err
}

// BuildRequest converts the round request into the canonical provider-neutral
// AI decision contract.
func (o AIOrchestrator) BuildRequest(request ports.RoundRequest) ports.AIDecisionRequest {
	roundView := request.RoleView.Clone()
	actionSchema := actionschema.Build(o.Scenario, request.Assignment.RoleID, roundView)
	return ports.AIDecisionRequest{
		MatchID:             request.RoleView.MatchID,
		Round:               request.RoleView.Round,
		RoleID:              request.Assignment.RoleID,
		Provider:            request.Assignment.Provider,
		Model:               request.Assignment.ModelName,
		Briefing:            roleBriefing(o.Scenario, request.Assignment.RoleID),
		RoundView:           roundView,
		RoleReport:          request.RoleReport.Clone(),
		SharedActionSurface: actionSchema,
		AllowedActions:      allowedActionSchema(o.Scenario, request.Assignment.RoleID),
		Tools:               lookupTools(),
		ResponseSpec: ports.ResponseFormatSpec{
			RequireJSONOnly:     true,
			AllowMarkdownFences: true,
			MaxCommentaryChars:  defaultAIMaxCommentary,
			MaxFocusTags:        defaultAIMaxFocusTags,
		},
		PreviousAction: clonePreviousForDecision(request.PreviousAcceptedAction),
	}
}

// Decide executes the AI request with deterministic retries and fallback.
func (o AIOrchestrator) Decide(ctx context.Context, request ports.AIDecisionRequest) (domain.ActionSubmission, ports.AIDecisionAudit, error) {
	if o.Client == nil {
		return domain.ActionSubmission{}, ports.AIDecisionAudit{}, fmt.Errorf("app: ai decision runner: decision client is not configured")
	}

	if err := validateDecisionRequest(request); err != nil {
		return domain.ActionSubmission{}, ports.AIDecisionAudit{}, fmt.Errorf("app: ai decision runner: %w", err)
	}

	attempts := max(1, o.MaxAttempts)
	audit := ports.AIDecisionAudit{}
	retryContext := request.RetryContext
	toolCalls := 0
	actionSchema := actionschema.Build(o.Scenario, request.RoleID, request.RoundView)
	logger := loggerOrDiscard(o.Logger).With(
		"component", "ai_orchestrator",
		"match_id", request.MatchID,
		"round", request.Round,
		"role_id", request.RoleID,
		"provider", request.Provider,
		"model", request.Model,
	)

	for attempt := range attempts {
		audit.AttemptCount = attempt + 1
		logger.Debug("ai decision attempt started", "attempt", attempt+1)

		providerRequest := ports.ProviderDecisionRequest{
			Provider:            request.Provider,
			Model:               request.Model,
			SystemPrompt:        prompting.BuildSystemPrompt(request),
			UserPrompt:          prompting.BuildUserPrompt(request, retryContext),
			RequireJSONOnly:     request.ResponseSpec.RequireJSONOnly,
			AllowMarkdownFences: request.ResponseSpec.AllowMarkdownFences,
		}

		result, err := o.Client.RequestDecision(ctx, providerRequest)
		if err != nil {
			o.appendDebugRecord(ports.AICallRecord{
				RoleID:       request.RoleID,
				Round:        request.Round,
				Attempt:      attempt + 1,
				Provider:     request.Provider,
				Model:        request.Model,
				SystemPrompt: providerRequest.SystemPrompt,
				UserPrompt:   providerRequest.UserPrompt,
				ErrorMessage: err.Error(),
			})
			logger.Error("ai decision request failed", "attempt", attempt+1, "error", err)
			return domain.ActionSubmission{}, audit, fmt.Errorf("app: ai decision runner: request decision: %w", err)
		}

		response, toolCall, validationErrors, parseErr := parseAndValidateDecision(result, request, actionSchema)
		isToolCall := parseErr == nil && toolCall != nil
		valid := parseErr == nil && !isToolCall && len(validationErrors) == 0

		o.appendDebugRecord(ports.AICallRecord{
			RoleID:       request.RoleID,
			Round:        request.Round,
			Attempt:      attempt + 1,
			Provider:     request.Provider,
			Model:        request.Model,
			SystemPrompt: providerRequest.SystemPrompt,
			UserPrompt:   providerRequest.UserPrompt,
			RawResponse:  result.RawResponse,
			IsToolCall:   isToolCall,
			Valid:        valid,
			ErrorMessage: debugErrorMessage(parseErr, validationErrors),
		})

		if parseErr == nil {
			if toolCall != nil {
				toolCalls++
				if toolCalls > max(0, o.MaxToolCalls) && o.MaxToolCalls > 0 {
					validationErrors = []ports.ValidationError{{Path: "tool_call", Message: fmt.Sprintf("tool call budget exceeded; at most %d tool calls allowed", o.MaxToolCalls)}}
					logger.Warn("ai tool call budget exceeded", "attempt", attempt+1, "tool_call_count", toolCalls, "tool_call_budget", o.MaxToolCalls)
				} else {
					toolResult, toolErr := o.executeToolCall(*toolCall)
					if toolErr != nil {
						validationErrors = []ports.ValidationError{{Path: "tool_call", Message: toolErr.Error()}}
						logger.Warn("ai tool call failed", "attempt", attempt+1, "tool_name", toolCall.ToolName, "error", toolErr)
					} else {
						request.ToolResults = append(request.ToolResults, toolResult)
						request.PriorAIResponse = result.RawResponse
						retryContext = nil
						audit.ValidationErrors = nil
						logger.Debug("ai tool call completed", "attempt", attempt+1, "tool_name", toolCall.ToolName, "tool_call_count", toolCalls)
						continue
					}
				}
			} else {
				logger.Info("ai decision completed", "attempt_count", attempt+1, "tool_call_count", toolCalls)
				return responseToSubmission(response, request), audit, nil
			}
		}

		audit.ValidationErrors = validationErrors
		logger.Debug("ai decision validation failed", "attempt", attempt+1, "validation_error_count", len(validationErrors), "parse_error", parseErr != nil)
		retryContext = &ports.RetryFeedback{
			Attempt:          attempt + 1,
			ValidationErrors: slices.Clone(validationErrors),
			LastRawResponse:  result.RawResponse,
		}
	}

	submission, reason := fallbackSubmission(request)
	audit.UsedFallback = true
	audit.FallbackReason = reason
	logger.Warn("ai decision fell back", "attempt_count", audit.AttemptCount, "tool_call_count", toolCalls, "fallback_reason", reason)
	return submission, audit, nil
}

func (o AIOrchestrator) appendDebugRecord(record ports.AICallRecord) {
	if o.DebugLog != nil {
		o.DebugLog.Append(record)
	}
}

func debugErrorMessage(parseErr error, validationErrors []ports.ValidationError) string {
	if parseErr != nil {
		return parseErr.Error()
	}
	if len(validationErrors) == 0 {
		return ""
	}
	msgs := make([]string, 0, len(validationErrors))
	for _, ve := range validationErrors {
		msgs = append(msgs, ve.Path+": "+ve.Message)
	}
	return strings.Join(msgs, "; ")
}

func validateDecisionRequest(request ports.AIDecisionRequest) error {
	var errs []error

	if request.MatchID == "" {
		errs = append(errs, fmt.Errorf("match id must not be empty"))
	}
	if request.Round <= 0 {
		errs = append(errs, fmt.Errorf("round must be positive"))
	}
	if request.RoleID == "" {
		errs = append(errs, fmt.Errorf("role id must not be empty"))
	}
	if strings.TrimSpace(request.Provider) == "" {
		errs = append(errs, fmt.Errorf("provider must not be empty"))
	}
	if strings.TrimSpace(request.Model) == "" {
		errs = append(errs, fmt.Errorf("model must not be empty"))
	}
	if len(errs) == 0 {
		return nil
	}
	return errorsJoin(errs...)
}

func parseAndValidateDecision(result ports.ProviderDecisionResult, request ports.AIDecisionRequest, schema actionschema.RoleSchema) (ports.AIDecisionResponse, *ports.LookupToolCall, []ports.ValidationError, error) {
	if result.StructuredResponse != nil {
		return validateStructuredDecision(*result.StructuredResponse, request, schema)
	}

	raw := result.RawResponse
	payload, err := extractJSONObject(raw)
	if err != nil {
		validationErrors := []ports.ValidationError{{Path: "$", Message: err.Error()}}
		return ports.AIDecisionResponse{}, nil, validationErrors, err
	}

	var toolEnvelope struct {
		ToolCall *ports.LookupToolCall `json:"tool_call"`
	}
	if err := json.Unmarshal([]byte(payload), &toolEnvelope); err == nil && toolEnvelope.ToolCall != nil {
		validationErrors := validateToolCall(*toolEnvelope.ToolCall, request.Tools)
		if len(validationErrors) > 0 {
			return ports.AIDecisionResponse{}, nil, validationErrors, fmt.Errorf("tool call failed validation")
		}
		return ports.AIDecisionResponse{}, toolEnvelope.ToolCall, nil, nil
	}

	var response ports.AIDecisionResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		validationErrors := []ports.ValidationError{{Path: "$", Message: fmt.Sprintf("invalid JSON payload: %v", err)}}
		return ports.AIDecisionResponse{}, nil, validationErrors, err
	}

	validationErrors := validateDecisionResponse(response, request, schema)
	if len(validationErrors) > 0 {
		return ports.AIDecisionResponse{}, nil, validationErrors, fmt.Errorf("response failed validation")
	}

	return response, nil, nil, nil
}

func validateStructuredDecision(envelope ports.AIDecisionEnvelope, request ports.AIDecisionRequest, schema actionschema.RoleSchema) (ports.AIDecisionResponse, *ports.LookupToolCall, []ports.ValidationError, error) {
	if envelope.ToolCall != nil {
		validationErrors := validateToolCall(*envelope.ToolCall, request.Tools)
		if len(validationErrors) > 0 {
			return ports.AIDecisionResponse{}, nil, validationErrors, fmt.Errorf("tool call failed validation")
		}
		return ports.AIDecisionResponse{}, envelope.ToolCall, nil, nil
	}

	response := envelope.DecisionResponse()
	validationErrors := validateDecisionResponse(response, request, schema)
	if len(validationErrors) > 0 {
		return ports.AIDecisionResponse{}, nil, validationErrors, fmt.Errorf("response failed validation")
	}

	return response, nil, nil, nil
}

func extractJSONObject(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("response did not contain a recoverable JSON object")
	}

	if strings.HasPrefix(trimmed, "```") {
		firstLineEnd := strings.Index(trimmed, "\n")
		if firstLineEnd == -1 {
			return "", fmt.Errorf("response did not contain a recoverable JSON object")
		}
		trimmed = strings.TrimSpace(trimmed[firstLineEnd+1:])
		if end := strings.LastIndex(trimmed, "```"); end >= 0 {
			trimmed = strings.TrimSpace(trimmed[:end])
		}
	}

	start := strings.IndexByte(trimmed, '{')
	if start < 0 {
		return "", fmt.Errorf("response did not contain a recoverable JSON object")
	}

	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(trimmed); i++ {
		switch trimmed[i] {
		case '\\':
			if inString {
				escaped = !escaped
			}
		case '"':
			if !escaped {
				inString = !inString
			}
			escaped = false
		case '{':
			if !inString {
				depth++
			}
			escaped = false
		case '}':
			if !inString {
				depth--
				if depth == 0 {
					return trimmed[start : i+1], nil
				}
			}
			escaped = false
		default:
			escaped = false
		}
	}

	return "", fmt.Errorf("response did not contain a recoverable JSON object")
}

func validateDecisionResponse(response ports.AIDecisionResponse, request ports.AIDecisionRequest, schema actionschema.RoleSchema) []ports.ValidationError {
	var errs []ports.ValidationError

	payloadCount := 0
	if response.Action.Procurement != nil {
		payloadCount++
	}
	if response.Action.Production != nil {
		payloadCount++
	}
	if response.Action.Sales != nil {
		payloadCount++
	}
	if response.Action.Finance != nil {
		payloadCount++
	}
	if payloadCount != 1 {
		errs = append(errs, ports.ValidationError{Path: "action", Message: "must contain exactly one populated role action payload"})
	}

	errs = append(errs, translateValidationErrors(actionschema.ValidateRoleAction(schema, response.Action, request.RoundView))...)

	summary := strings.TrimSpace(response.Commentary.PublicSummary)
	if summary == "" {
		errs = append(errs, ports.ValidationError{Path: "commentary.public_summary", Message: "must not be empty"})
	}
	if len(summary) > request.ResponseSpec.MaxCommentaryChars {
		errs = append(errs, ports.ValidationError{Path: "commentary.public_summary", Message: fmt.Sprintf("must be at most %d characters", request.ResponseSpec.MaxCommentaryChars)})
	}
	if len(response.Commentary.FocusTags) > 0 && len(response.Commentary.FocusTags) > request.ResponseSpec.MaxFocusTags {
		errs = append(errs, ports.ValidationError{Path: "commentary.focus_tags", Message: fmt.Sprintf("must contain at most %d tags", request.ResponseSpec.MaxFocusTags)})
	}
	for i, tag := range response.Commentary.FocusTags {
		if strings.TrimSpace(tag) == "" {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("commentary.focus_tags[%d]", i), Message: "must not be empty"})
		}
	}

	return errs
}

func validateToolCall(call ports.LookupToolCall, tools []ports.LookupToolSpec) []ports.ValidationError {
	var errs []ports.ValidationError
	if strings.TrimSpace(call.ToolName) == "" {
		errs = append(errs, ports.ValidationError{Path: "tool_call.tool_name", Message: "must not be empty"})
		return errs
	}

	var selected *ports.LookupToolSpec
	for i := range tools {
		if tools[i].Name == call.ToolName {
			selected = &tools[i]
			break
		}
	}
	if selected == nil {
		return []ports.ValidationError{{Path: "tool_call.tool_name", Message: fmt.Sprintf("must match one of the available tools; got %q", call.ToolName)}}
	}

	for _, argument := range selected.Arguments {
		if argument.Required && strings.TrimSpace(call.Arguments[argument.Name]) == "" {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("tool_call.arguments.%s", argument.Name), Message: "must not be empty"})
		}
	}
	return errs
}

func responseToSubmission(response ports.AIDecisionResponse, request ports.AIDecisionRequest) domain.ActionSubmission {
	return domain.ActionSubmission{
		MatchID: request.MatchID,
		Round:   request.Round,
		RoleID:  request.RoleID,
		Action:  response.Action.Clone(),
		Commentary: domain.CommentaryRecord{
			RoleID:     request.RoleID,
			Visibility: domain.CommentaryPublic,
			Body:       strings.TrimSpace(response.Commentary.PublicSummary),
		},
	}
}

func fallbackSubmission(request ports.AIDecisionRequest) (domain.ActionSubmission, string) {
	if request.PreviousAction != nil {
		reused := request.PreviousAction.Clone()
		reused.MatchID = request.MatchID
		reused.Round = request.Round
		reused.RoleID = request.RoleID
		reused.Commentary = domain.CommentaryRecord{
			MatchID:    request.MatchID,
			Round:      request.Round,
			RoleID:     request.RoleID,
			Visibility: domain.CommentaryPublic,
			Body:       "Previous action reused after invalid AI output.",
		}
		return reused, "previous accepted action reused after invalid AI output"
	}

	return domain.ActionSubmission{
		MatchID: request.MatchID,
		Round:   request.Round,
		RoleID:  request.RoleID,
		Action:  safeNoOpAction(request.RoleID, request.RoundView.ActiveTargets),
		Commentary: domain.CommentaryRecord{
			MatchID:    request.MatchID,
			Round:      request.Round,
			RoleID:     request.RoleID,
			Visibility: domain.CommentaryPublic,
			Body:       "Safe no-op submitted after invalid AI output.",
		},
	}, "safe no-op submitted after invalid AI output"
}

func safeNoOpAction(roleID domain.RoleID, targets domain.BudgetTargets) domain.RoleAction {
	switch roleID {
	case domain.RoleProcurementManager:
		return domain.RoleAction{Procurement: &domain.ProcurementAction{Orders: []domain.PurchaseOrderIntent{}}}
	case domain.RoleProductionManager:
		return domain.RoleAction{Production: &domain.ProductionAction{
			Releases:           []domain.ProductionRelease{},
			CapacityAllocation: []domain.CapacityAllocation{},
		}}
	case domain.RoleSalesManager:
		return domain.RoleAction{Sales: &domain.SalesAction{ProductOffers: []domain.ProductOffer{}}}
	case domain.RoleFinanceController:
		return domain.RoleAction{Finance: &domain.FinanceAction{NextRoundTargets: targets}}
	default:
		return domain.RoleAction{}
	}
}

func roleBriefing(definition scenario.Definition, roleID domain.RoleID) ports.RoleBriefing {
	schema := actionschema.Build(definition, roleID, domain.RoundView{})
	switch roleID {
	case domain.RoleProcurementManager:
		return ports.RoleBriefing{
			RoleID:                 roleID,
			DisplayName:            "Procurement Manager",
			PublicResponsibilities: []string{"Secure materials required for operations.", "Control input cost.", "Protect the plant from shortages.", "Build reliable supplier coverage."},
			HiddenIncentives:       []string{"Favor bulk buys and lower unit prices even when inventory and cash risk rise."},
			DecisionPrinciples:     []string{"Protect supply continuity for bottleneck flow.", "Avoid buying material the plant cannot use soon.", "Stay aware of active budget targets and plant cash."},
			AllowedActionSummary:   slices.Clone(schema.AllowedSummary),
		}
	case domain.RoleProductionManager:
		return ports.RoleBriefing{
			RoleID:                 roleID,
			DisplayName:            "Production Manager",
			PublicResponsibilities: []string{"Maximize production output.", "Keep machines and labor utilized.", "Manage work-in-progress through the shop floor.", "Meet production commitments."},
			HiddenIncentives:       []string{"Keep resources busy and local output high even when WIP or bottlenecks worsen."},
			DecisionPrinciples:     []string{"Favor plant throughput over local utilization theater.", "Release only work that can move through the route.", "Keep WIP under control at the bottleneck."},
			AllowedActionSummary:   slices.Clone(schema.AllowedSummary),
		}
	case domain.RoleSalesManager:
		return ports.RoleBriefing{
			RoleID:                 roleID,
			DisplayName:            "Sales Manager",
			PublicResponsibilities: []string{"Grow revenue.", "Capture demand.", "Maintain customer relationships.", "Push the plant toward market opportunity."},
			HiddenIncentives:       []string{"Favor booked demand and strong promises even when capacity or delivery reliability suffer."},
			DecisionPrinciples:     []string{"Protect profitable throughput, not just order count.", "Consider backlog and delivery risk before chasing demand.", "Set prices that fit current operational reality."},
			AllowedActionSummary:   slices.Clone(schema.AllowedSummary),
		}
	case domain.RoleFinanceController:
		return ports.RoleBriefing{
			RoleID:                 roleID,
			DisplayName:            "Finance Controller",
			PublicResponsibilities: []string{"Monitor cash, cost, and financial performance.", "Highlight waste and overspending.", "Protect the business from financially dangerous decisions.", "Provide visibility into profit drivers."},
			HiddenIncentives:       []string{"Favor short-term cost discipline even when it can damage throughput or resilience."},
			DecisionPrinciples:     []string{"Preserve liquidity without starving profitable flow.", "Set next-round targets that balance cash, debt, and throughput.", "Treat cost cuts that harm throughput as risky."},
			AllowedActionSummary:   slices.Clone(schema.AllowedSummary),
		}
	default:
		return ports.RoleBriefing{RoleID: roleID, DisplayName: string(roleID)}
	}
}

func allowedActionSchema(definition scenario.Definition, roleID domain.RoleID) ports.AllowedActionSchema {
	spec := actionschema.Build(definition, roleID, domain.RoundView{})
	return ports.AllowedActionSchema{
		RoleID:         roleID,
		RequiredAction: spec.RequiredAction,
		JSONSchemaName: spec.JSONSchemaName,
		Rules:          slices.Clone(spec.ValidationRules),
	}
}

func clonePreviousForDecision(previous *domain.ActionSubmission) *domain.ActionSubmission {
	if previous == nil {
		return nil
	}
	cloned := previous.Clone()
	return &cloned
}

func translateValidationErrors(errs []actionschema.ValidationError) []ports.ValidationError {
	out := make([]ports.ValidationError, 0, len(errs))
	for _, err := range errs {
		out = append(out, ports.ValidationError{Path: err.Path, Message: err.Message})
	}
	return out
}

func errorsJoin(errs ...error) error {
	return errors.Join(errs...)
}

func lookupTools() []ports.LookupToolSpec {
	return scenario.LookupTools()
}

func (o AIOrchestrator) executeToolCall(call ports.LookupToolCall) (ports.LookupToolResult, error) {
	return o.Scenario.ExecuteLookup(call)
}
