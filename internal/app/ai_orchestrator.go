package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

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
	return ports.AIDecisionRequest{
		ContractVersion: ports.AIDecisionContractVersion,
		MatchID:         request.RoleView.MatchID,
		Round:           request.RoleView.Round,
		RoleID:          request.Assignment.RoleID,
		Provider:        request.Assignment.Provider,
		Model:           request.Assignment.ModelName,
		Briefing:        roleBriefing(request.Assignment.RoleID),
		RoundView:       request.RoleView.Clone(),
		RoleReport:      request.RoleReport.Clone(),
		AllowedActions:  allowedActionSchema(request.Assignment.RoleID),
		Tools:           lookupTools(),
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

	for attempt := range attempts {
		audit.AttemptCount = attempt + 1

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
			return domain.ActionSubmission{}, audit, fmt.Errorf("app: ai decision runner: request decision: %w", err)
		}

		response, toolCall, validationErrors, parseErr := parseAndValidateDecision(result, request)
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
				} else {
					toolResult, toolErr := o.executeToolCall(*toolCall)
					if toolErr != nil {
						validationErrors = []ports.ValidationError{{Path: "tool_call", Message: toolErr.Error()}}
					} else {
						request.ToolResults = append(request.ToolResults, toolResult)
						request.PriorAIResponse = result.RawResponse
						retryContext = nil
						audit.ValidationErrors = nil
						continue
					}
				}
			} else {
				return responseToSubmission(response, request), audit, nil
			}
		}

		audit.ValidationErrors = validationErrors
		retryContext = &ports.RetryFeedback{
			Attempt:          attempt + 1,
			ValidationErrors: slices.Clone(validationErrors),
			LastRawResponse:  result.RawResponse,
		}
	}

	submission, reason := fallbackSubmission(request)
	audit.UsedFallback = true
	audit.FallbackReason = reason
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

	if request.ContractVersion == "" {
		errs = append(errs, fmt.Errorf("contract version must not be empty"))
	}
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

func parseAndValidateDecision(result ports.ProviderDecisionResult, request ports.AIDecisionRequest) (ports.AIDecisionResponse, *ports.LookupToolCall, []ports.ValidationError, error) {
	if result.StructuredResponse != nil {
		return validateStructuredDecision(*result.StructuredResponse, request)
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

	validationErrors := validateDecisionResponse(response, request)
	if len(validationErrors) > 0 {
		return ports.AIDecisionResponse{}, nil, validationErrors, fmt.Errorf("response failed validation")
	}

	return response, nil, nil, nil
}

func validateStructuredDecision(envelope ports.AIDecisionEnvelope, request ports.AIDecisionRequest) (ports.AIDecisionResponse, *ports.LookupToolCall, []ports.ValidationError, error) {
	if envelope.ToolCall != nil {
		validationErrors := validateToolCall(*envelope.ToolCall, request.Tools)
		if len(validationErrors) > 0 {
			return ports.AIDecisionResponse{}, nil, validationErrors, fmt.Errorf("tool call failed validation")
		}
		return ports.AIDecisionResponse{}, envelope.ToolCall, nil, nil
	}

	response := envelope.DecisionResponse()
	validationErrors := validateDecisionResponse(response, request)
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

func validateDecisionResponse(response ports.AIDecisionResponse, request ports.AIDecisionRequest) []ports.ValidationError {
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

	switch request.RoleID {
	case domain.RoleProcurementManager:
		if response.Action.Procurement == nil {
			errs = append(errs, ports.ValidationError{Path: "action.procurement", Message: "must be populated for procurement_manager"})
		}
		errs = append(errs, validateProcurementAction(response.Action.Procurement)...)
	case domain.RoleProductionManager:
		if response.Action.Production == nil {
			errs = append(errs, ports.ValidationError{Path: "action.production", Message: "must be populated for production_manager"})
		}
		errs = append(errs, validateProductionAction(response.Action.Production, request.RoundView)...)
	case domain.RoleSalesManager:
		if response.Action.Sales == nil {
			errs = append(errs, ports.ValidationError{Path: "action.sales", Message: "must be populated for sales_manager"})
		}
		errs = append(errs, validateSalesAction(response.Action.Sales, request.RoundView)...)
	case domain.RoleFinanceController:
		if response.Action.Finance == nil {
			errs = append(errs, ports.ValidationError{Path: "action.finance", Message: "must be populated for finance_controller"})
		}
	}

	summary := strings.TrimSpace(response.Commentary.PublicSummary)
	if summary == "" {
		errs = append(errs, ports.ValidationError{Path: "commentary.public_summary", Message: "must not be empty"})
	}
	if len(summary) > request.ResponseSpec.MaxCommentaryChars {
		errs = append(errs, ports.ValidationError{Path: "commentary.public_summary", Message: fmt.Sprintf("must be at most %d characters", request.ResponseSpec.MaxCommentaryChars)})
	}
	if len(response.Commentary.FocusTags) == 0 {
		errs = append(errs, ports.ValidationError{Path: "commentary.focus_tags", Message: "must contain at least one tag"})
	}
	if len(response.Commentary.FocusTags) > request.ResponseSpec.MaxFocusTags {
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

func validateProcurementAction(action *domain.ProcurementAction) []ports.ValidationError {
	if action == nil {
		return nil
	}

	var errs []ports.ValidationError
	for i, order := range action.Orders {
		if order.PartID == "" {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.procurement.orders[%d].part_id", i), Message: "must not be empty"})
		}
		if order.SupplierID == "" {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.procurement.orders[%d].supplier_id", i), Message: "must not be empty"})
		}
		if order.Quantity < 0 {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.procurement.orders[%d].quantity", i), Message: "must be non-negative"})
		}
	}
	return errs
}

func validateProductionAction(action *domain.ProductionAction, view domain.RoundView) []ports.ValidationError {
	if action == nil {
		return nil
	}

	knownProducts := productSet(view)
	knownStations := workstationSet(view)

	var errs []ports.ValidationError
	for i, release := range action.Releases {
		if !knownProducts[release.ProductID] {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.production.releases[%d].product_id", i), Message: "must reference a product visible in the round view"})
		}
		if release.Quantity < 0 {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.production.releases[%d].quantity", i), Message: "must be non-negative"})
		}
	}
	for i, allocation := range action.CapacityAllocation {
		if !knownStations[allocation.WorkstationID] {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.production.capacity_allocation[%d].workstation_id", i), Message: "must reference a workstation visible in the round view"})
		}
		if !knownProducts[allocation.ProductID] {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.production.capacity_allocation[%d].product_id", i), Message: "must reference a product visible in the round view"})
		}
		if allocation.Capacity < 0 {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.production.capacity_allocation[%d].capacity", i), Message: "must be non-negative"})
		}
	}
	return errs
}

func validateSalesAction(action *domain.SalesAction, view domain.RoundView) []ports.ValidationError {
	if action == nil {
		return nil
	}

	knownProducts := productSet(view)
	var errs []ports.ValidationError
	for i, offer := range action.ProductOffers {
		if !knownProducts[offer.ProductID] {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.sales.product_offers[%d].product_id", i), Message: "must reference a product visible in the round view"})
		}
		if offer.UnitPrice < 0 {
			errs = append(errs, ports.ValidationError{Path: fmt.Sprintf("action.sales.product_offers[%d].unit_price", i), Message: "must be non-negative"})
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

func roleBriefing(roleID domain.RoleID) ports.RoleBriefing {
	switch roleID {
	case domain.RoleProcurementManager:
		return ports.RoleBriefing{
			RoleID:                 roleID,
			DisplayName:            "Procurement Manager",
			PublicResponsibilities: []string{"Secure materials required for operations.", "Control input cost.", "Protect the plant from shortages.", "Build reliable supplier coverage."},
			HiddenIncentives:       []string{"Favor bulk buys and lower unit prices even when inventory and cash risk rise."},
			DecisionPrinciples:     []string{"Protect supply continuity for bottleneck flow.", "Avoid buying material the plant cannot use soon.", "Stay aware of active budget targets and plant cash."},
			AllowedActionSummary:   []string{"Return only procurement orders.", "Each order needs part_id, supplier_id, and quantity.", "Use an empty orders list for a deliberate no-op."},
		}
	case domain.RoleProductionManager:
		return ports.RoleBriefing{
			RoleID:                 roleID,
			DisplayName:            "Production Manager",
			PublicResponsibilities: []string{"Maximize production output.", "Keep machines and labor utilized.", "Manage work-in-progress through the shop floor.", "Meet production commitments."},
			HiddenIncentives:       []string{"Keep resources busy and local output high even when WIP or bottlenecks worsen."},
			DecisionPrinciples:     []string{"Favor plant throughput over local utilization theater.", "Release only work that can move through the route.", "Keep WIP under control at the bottleneck."},
			AllowedActionSummary:   []string{"Return production releases, capacity allocations, and optional overtime allocations.", "Each release needs product_id and quantity.", "Each capacity allocation needs workstation_id, product_id, and capacity.", "Each overtime allocation needs workstation_id and capacity; omit overtime entirely if not needed."},
		}
	case domain.RoleSalesManager:
		return ports.RoleBriefing{
			RoleID:                 roleID,
			DisplayName:            "Sales Manager",
			PublicResponsibilities: []string{"Grow revenue.", "Capture demand.", "Maintain customer relationships.", "Push the plant toward market opportunity."},
			HiddenIncentives:       []string{"Favor booked demand and strong promises even when capacity or delivery reliability suffer."},
			DecisionPrinciples:     []string{"Protect profitable throughput, not just order count.", "Consider backlog and delivery risk before chasing demand.", "Set prices that fit current operational reality."},
			AllowedActionSummary:   []string{"Return only product offers.", "Each offer needs product_id and unit_price.", "Use an empty product_offers list for a deliberate no-op."},
		}
	case domain.RoleFinanceController:
		return ports.RoleBriefing{
			RoleID:                 roleID,
			DisplayName:            "Finance Controller",
			PublicResponsibilities: []string{"Monitor cash, cost, and financial performance.", "Highlight waste and overspending.", "Protect the business from financially dangerous decisions.", "Provide visibility into profit drivers."},
			HiddenIncentives:       []string{"Favor short-term cost discipline even when it can damage throughput or resilience."},
			DecisionPrinciples:     []string{"Preserve liquidity without starving profitable flow.", "Set next-round targets that balance cash, debt, and throughput.", "Treat cost cuts that harm throughput as risky."},
			AllowedActionSummary:   []string{"Return only finance next_round_targets.", "Provide the full next_round_targets object.", "A safe no-op repeats the currently active targets."},
		}
	default:
		return ports.RoleBriefing{RoleID: roleID, DisplayName: string(roleID)}
	}
}

func allowedActionSchema(roleID domain.RoleID) ports.AllowedActionSchema {
	switch roleID {
	case domain.RoleProcurementManager:
		return ports.AllowedActionSchema{
			RoleID:         roleID,
			RequiredAction: "procurement",
			JSONSchemaName: "ProcurementAction",
			Rules:          []string{"Populate only action.procurement.", "orders entries require part_id, supplier_id, and quantity.", "quantity must be non-negative."},
		}
	case domain.RoleProductionManager:
		return ports.AllowedActionSchema{
			RoleID:         roleID,
			RequiredAction: "production",
			JSONSchemaName: "ProductionAction",
			Rules:          []string{"Populate only action.production.", "releases entries require product_id and quantity.", "capacity_allocation entries require workstation_id, product_id, and capacity.", "overtime entries are optional; each requires workstation_id and capacity.", "quantities and capacity must be non-negative."},
		}
	case domain.RoleSalesManager:
		return ports.AllowedActionSchema{
			RoleID:         roleID,
			RequiredAction: "sales",
			JSONSchemaName: "SalesAction",
			Rules:          []string{"Populate only action.sales.", "product_offers entries require product_id and unit_price.", "unit_price must be non-negative."},
		}
	case domain.RoleFinanceController:
		return ports.AllowedActionSchema{
			RoleID:         roleID,
			RequiredAction: "finance",
			JSONSchemaName: "FinanceAction",
			Rules:          []string{"Populate only action.finance.", "Provide next_round_targets exactly once.", "Each target field must be present."},
		}
	default:
		return ports.AllowedActionSchema{RoleID: roleID}
	}
}

func productSet(view domain.RoundView) map[domain.ProductID]bool {
	products := make(map[domain.ProductID]bool)
	for _, item := range view.Plant.WIPInventory {
		products[item.ProductID] = true
	}
	for _, item := range view.Plant.FinishedInventory {
		products[item.ProductID] = true
	}
	for _, item := range view.Plant.Backlog {
		products[item.ProductID] = true
	}
	for _, customer := range view.Customers {
		for _, item := range customer.Backlog {
			products[item.ProductID] = true
		}
	}
	if len(products) == 0 {
		products["pump"] = true
	}
	return products
}

func workstationSet(view domain.RoundView) map[domain.WorkstationID]bool {
	stations := make(map[domain.WorkstationID]bool, len(view.Plant.Workstations))
	for _, item := range view.Plant.Workstations {
		stations[item.WorkstationID] = true
	}
	return stations
}

func clonePreviousForDecision(previous *domain.ActionSubmission) *domain.ActionSubmission {
	if previous == nil {
		return nil
	}
	cloned := previous.Clone()
	return &cloned
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
