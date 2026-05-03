package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/player/llm"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestAIOrchestratorBuildsPromptAndParsesValidDecision(t *testing.T) {
	client := &stubDecisionClient{
		responses: []ports.ProviderDecisionResult{
			{RawResponse: `{
				"action":{"sales":{"product_offers":[{"product_id":"pump","unit_price":16}]}},
				"commentary":{"public_summary":"Holding price to protect throughput.","focus_tags":["throughput","pricing"]}
			}`},
		},
	}

	orchestrator := app.NewAIOrchestrator(scenario.Starter(), client)
	request := aiRoundRequest(domain.RoleSalesManager)

	submission, audit, err := orchestrator.Decide(context.Background(), orchestrator.BuildRequest(request))
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}

	if audit.AttemptCount != 1 {
		t.Fatalf("audit.AttemptCount = %d, want 1", audit.AttemptCount)
	}
	if audit.UsedFallback {
		t.Fatal("audit.UsedFallback = true, want false")
	}
	if got := len(client.requests); got != 1 {
		t.Fatalf("client request count = %d, want 1", got)
	}
	if got := client.requests[0].Provider; got != "openrouter" {
		t.Fatalf("Provider = %q, want openrouter", got)
	}
	if !strings.Contains(client.requests[0].SystemPrompt, "Sales Manager") {
		t.Fatalf("SystemPrompt = %q, want role briefing content", client.requests[0].SystemPrompt)
	}
	if !strings.Contains(client.requests[0].SystemPrompt, "# Role") || !strings.Contains(client.requests[0].SystemPrompt, "# Instructions") || !strings.Contains(client.requests[0].SystemPrompt, "# Tools") || !strings.Contains(client.requests[0].SystemPrompt, "# Response") {
		t.Fatalf("SystemPrompt = %q, want structured prompt sections", client.requests[0].SystemPrompt)
	}
	if !strings.Contains(client.requests[0].SystemPrompt, "## Parts") || !strings.Contains(client.requests[0].SystemPrompt, "## Products") || !strings.Contains(client.requests[0].SystemPrompt, "## Vendors") || !strings.Contains(client.requests[0].SystemPrompt, "## Customers") {
		t.Fatalf("SystemPrompt = %q, want grouped tool sections", client.requests[0].SystemPrompt)
	}
	if !strings.Contains(client.requests[0].UserPrompt, "## Allowed Action Schema") {
		t.Fatalf("UserPrompt = %q, want schema section", client.requests[0].UserPrompt)
	}
	if !strings.Contains(client.requests[0].UserPrompt, "## Tool Lookup") {
		t.Fatalf("UserPrompt = %q, want tool lookup section", client.requests[0].UserPrompt)
	}
	if submission.Action.Sales == nil {
		t.Fatal("submission.Action.Sales = nil, want populated sales payload")
	}
	if got := submission.Action.Sales.ProductOffers[0].UnitPrice; got != 16 {
		t.Fatalf("UnitPrice = %d, want 16", got)
	}
	if got := submission.Commentary.Body; got != "Holding price to protect throughput." {
		t.Fatalf("Commentary.Body = %q, want parsed commentary", got)
	}
}

func TestAIOrchestratorRetriesWithValidationFeedback(t *testing.T) {
	client := &stubDecisionClient{
		responses: []ports.ProviderDecisionResult{
			{RawResponse: `{"action":{"sales":{"product_offers":[{"product_id":"pump","unit_price":16}]}},"commentary":{"public_summary":"","focus_tags":[]}}`},
			{RawResponse: "```json\n{\"action\":{\"sales\":{\"product_offers\":[{\"product_id\":\"pump\",\"unit_price\":15}]}},\"commentary\":{\"public_summary\":\"Reducing price slightly to protect revenue without outrunning flow.\",\"focus_tags\":[\"revenue\",\"flow\"]}}\n```"},
		},
	}

	orchestrator := app.NewAIOrchestrator(scenario.Starter(), client)
	request := aiRoundRequest(domain.RoleSalesManager)

	submission, audit, err := orchestrator.Decide(context.Background(), orchestrator.BuildRequest(request))
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}

	if audit.AttemptCount != 2 {
		t.Fatalf("audit.AttemptCount = %d, want 2", audit.AttemptCount)
	}
	if len(audit.ValidationErrors) == 0 {
		t.Fatal("audit.ValidationErrors = nil, want retry validation feedback")
	}
	if got := len(client.requests); got != 2 {
		t.Fatalf("client request count = %d, want 2", got)
	}
	if !strings.Contains(client.requests[1].UserPrompt, "## Retry Feedback") {
		t.Fatalf("second UserPrompt = %q, want retry feedback section", client.requests[1].UserPrompt)
	}
	if got := submission.Action.Sales.ProductOffers[0].UnitPrice; got != 15 {
		t.Fatalf("UnitPrice = %d, want 15", got)
	}
}

func TestAIOrchestratorFallsBackAfterInvalidResponses(t *testing.T) {
	client := &stubDecisionClient{
		responses: []ports.ProviderDecisionResult{
			{RawResponse: "not json"},
			{RawResponse: `{"action":{"sales":{"product_offers":[{"product_id":"","unit_price":-1}]}},"commentary":{"public_summary":"","focus_tags":[]}}`},
			{RawResponse: `{"action":{},"commentary":{"public_summary":"","focus_tags":[]}}`},
		},
	}

	orchestrator := app.NewAIOrchestrator(scenario.Starter(), client)
	request := aiRoundRequest(domain.RoleSalesManager)
	request.PreviousAcceptedAction = &domain.ActionSubmission{
		MatchID: "match-17",
		Round:   1,
		RoleID:  domain.RoleSalesManager,
		Action: domain.RoleAction{
			Sales: &domain.SalesAction{
				ProductOffers: []domain.ProductOffer{{ProductID: "pump", UnitPrice: 13}},
			},
		},
	}

	submission, audit, err := orchestrator.Decide(context.Background(), orchestrator.BuildRequest(request))
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}

	if !audit.UsedFallback {
		t.Fatal("audit.UsedFallback = false, want true")
	}
	if got := submission.Commentary.Body; got != "Previous action reused after invalid AI output." {
		t.Fatalf("Commentary.Body = %q, want fallback reuse commentary", got)
	}
	if got := submission.Action.Sales.ProductOffers[0].UnitPrice; got != 13 {
		t.Fatalf("UnitPrice = %d, want reused price 13", got)
	}
}

func TestRoundCollectorUsesAIOrchestratorThroughLLMPlayer(t *testing.T) {
	client := &stubDecisionClient{
		responses: []ports.ProviderDecisionResult{
			{RawResponse: `{"action":{"production":{"releases":[{"product_id":"pump","quantity":1}],"capacity_allocation":[{"workstation_id":"fabrication","product_id":"pump","capacity":1}]}},"commentary":{"public_summary":"Releasing only what fabrication can move this round.","focus_tags":["throughput"]}}`},
		},
	}

	orchestrator := app.NewAIOrchestrator(scenario.Starter(), client)
	state := fixtureMatchState()
	state.Roles = []domain.RoleAssignment{
		{RoleID: domain.RoleProductionManager, PlayerID: "ai-prod", Provider: "ollama", ModelName: "llama3.2:3b"},
	}

	collector := app.RoundCollector{
		Players: map[domain.RoleID]ports.Player{
			domain.RoleProductionManager: llm.New(orchestrator.SubmitRound),
		},
	}

	actions, err := collector.Collect(context.Background(), state, nil)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if got := len(actions); got != 1 {
		t.Fatalf("len(actions) = %d, want 1", got)
	}
	if actions[0].Action.Production == nil {
		t.Fatal("actions[0].Action.Production = nil, want populated production payload")
	}
}

func TestAIOrchestratorReturnsTransportErrors(t *testing.T) {
	client := &stubDecisionClient{
		err: errors.New("provider unavailable"),
	}

	orchestrator := app.NewAIOrchestrator(scenario.Starter(), client)

	_, _, err := orchestrator.Decide(context.Background(), orchestrator.BuildRequest(aiRoundRequest(domain.RoleSalesManager)))
	if err == nil {
		t.Fatal("Decide() error = nil, want transport error")
	}
	if !strings.Contains(err.Error(), "provider unavailable") {
		t.Fatalf("error = %q, want transport cause", err)
	}
}

type stubDecisionClient struct {
	requests  []ports.ProviderDecisionRequest
	responses []ports.ProviderDecisionResult
	err       error
}

func (s *stubDecisionClient) RequestDecision(_ context.Context, request ports.ProviderDecisionRequest) (ports.ProviderDecisionResult, error) {
	s.requests = append(s.requests, request)
	if s.err != nil {
		return ports.ProviderDecisionResult{}, s.err
	}
	if len(s.responses) == 0 {
		return ports.ProviderDecisionResult{}, errors.New("unexpected request")
	}
	response := s.responses[0]
	s.responses = s.responses[1:]
	return response, nil
}

func aiRoundRequest(roleID domain.RoleID) ports.RoundRequest {
	request := ports.RoundRequest{
		Assignment: domain.RoleAssignment{
			RoleID:    roleID,
			PlayerID:  "ai-player",
			Provider:  "openrouter",
			ModelName: "openai/gpt-5-mini",
		},
		RoleView:   buildAIView(roleID),
		RoleReport: buildAIReport(roleID),
	}
	if roleID == domain.RoleProductionManager {
		request.Assignment.Provider = "ollama"
		request.Assignment.ModelName = "llama3.2:3b"
	}
	return request
}

func buildAIView(roleID domain.RoleID) domain.RoundView {
	return domain.RoundView{
		MatchID:      "match-17",
		Round:        2,
		ViewerRoleID: roleID,
		Plant: domain.PlantState{
			Cash: 24,
			WIPInventory: []domain.WIPInventory{
				{ProductID: "pump", WorkstationID: "fabrication", Quantity: 1, UnitCost: 6},
			},
			FinishedInventory: []domain.FinishedInventory{
				{ProductID: "pump", OnHandQty: 1, UnitCost: 8},
			},
			Backlog: []domain.BacklogEntry{
				{CustomerID: "northbuild", ProductID: "pump", Quantity: 2, OriginRound: 1},
			},
			Workstations: []domain.WorkstationState{
				{WorkstationID: "fabrication", CapacityPerRound: 4},
				{WorkstationID: "assembly", CapacityPerRound: 3},
			},
		},
		Customers: []domain.CustomerState{
			{CustomerID: "northbuild", DisplayName: "NorthBuild", Sentiment: 6},
		},
		ActiveTargets: domain.BudgetTargets{
			EffectiveRound:        2,
			ProcurementBudget:     18,
			ProductionSpendBudget: 14,
			RevenueTarget:         28,
			CashFloorTarget:       8,
			DebtCeilingTarget:     15,
		},
	}
}

func buildAIReport(roleID domain.RoleID) domain.RoleRoundReport {
	return domain.RoleRoundReport{
		Department: domain.DepartmentPerformanceReport{
			RoleID: roleID,
		},
		BonusReminder: "Protect plant-wide flow.",
	}
}
func TestAIOrchestratorExecutesLookupToolCallsBeforeFinalDecision(t *testing.T) {
	toolCallResponse := `{"tool_call":{"tool_name":"show_product_route","arguments":{"product_id":"pump"}}}`
	client := &stubDecisionClient{
		responses: []ports.ProviderDecisionResult{
			{RawResponse: toolCallResponse},
			{RawResponse: `{"action":{"production":{"releases":[{"product_id":"pump","quantity":1}],"capacity_allocation":[{"workstation_id":"fabrication","product_id":"pump","capacity":1}]}},"commentary":{"public_summary":"Using the route lookup to release only work that fits the line.","focus_tags":["throughput"]}}`},
		},
	}

	orchestrator := app.NewAIOrchestrator(scenario.Starter(), client)
	debugLog := app.NewDebugLog(10)
	orchestrator.DebugLog = debugLog
	request := aiRoundRequest(domain.RoleProductionManager)

	submission, audit, err := orchestrator.Decide(context.Background(), orchestrator.BuildRequest(request))
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}

	if audit.AttemptCount != 2 {
		t.Fatalf("audit.AttemptCount = %d, want 2", audit.AttemptCount)
	}
	if got := len(client.requests); got != 2 {
		t.Fatalf("client request count = %d, want 2", got)
	}
	if !strings.Contains(client.requests[1].UserPrompt, "## Tool Results") {
		t.Fatalf("second UserPrompt = %q, want tool results section", client.requests[1].UserPrompt)
	}
	if !strings.Contains(client.requests[1].UserPrompt, "## Prior Tool Call") {
		t.Fatalf("second UserPrompt = %q, want prior tool call section", client.requests[1].UserPrompt)
	}
	if !strings.Contains(client.requests[1].UserPrompt, toolCallResponse) {
		t.Fatalf("second UserPrompt = %q, want prior tool call response verbatim", client.requests[1].UserPrompt)
	}
	if got := submission.Commentary.Body; got != "Using the route lookup to release only work that fits the line." {
		t.Fatalf("Commentary.Body = %q, want final commentary", got)
	}

	records := debugLog.Records()
	if got := len(records); got != 2 {
		t.Fatalf("debug log records = %d, want 2", got)
	}
	if !records[0].IsToolCall {
		t.Fatal("records[0].IsToolCall = false, want true for tool call round")
	}
	if records[1].IsToolCall {
		t.Fatal("records[1].IsToolCall = true, want false for final decision round")
	}
	if !records[1].Valid {
		t.Fatal("records[1].Valid = false, want true for successful final decision")
	}
}

func TestAIOrchestratorUsesStructuredProviderResponsesWhenAvailable(t *testing.T) {
	client := &stubDecisionClient{
		responses: []ports.ProviderDecisionResult{
			{
				RawResponse: `{}`,
				StructuredResponse: &ports.AIDecisionEnvelope{
					Action: domain.RoleAction{
						Sales: &domain.SalesAction{
							ProductOffers: []domain.ProductOffer{{ProductID: "pump", UnitPrice: 17}},
						},
					},
					Commentary: ports.AICommentary{
						PublicSummary: "Raise price to protect constrained throughput.",
						FocusTags:     []string{"throughput", "pricing"},
					},
				},
			},
		},
	}

	orchestrator := app.NewAIOrchestrator(scenario.Starter(), client)

	submission, audit, err := orchestrator.Decide(context.Background(), orchestrator.BuildRequest(aiRoundRequest(domain.RoleSalesManager)))
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if audit.AttemptCount != 1 {
		t.Fatalf("audit.AttemptCount = %d, want 1", audit.AttemptCount)
	}
	if submission.Action.Sales == nil {
		t.Fatal("submission.Action.Sales = nil, want structured sales payload")
	}
	if got := submission.Action.Sales.ProductOffers[0].UnitPrice; got != 17 {
		t.Fatalf("UnitPrice = %d, want 17", got)
	}
	if got := submission.Commentary.Body; got != "Raise price to protect constrained throughput." {
		t.Fatalf("Commentary.Body = %q, want structured commentary", got)
	}
}
