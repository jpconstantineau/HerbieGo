package prompting

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

// BuildSystemPrompt renders the role-facing system prompt.
func BuildSystemPrompt(request ports.AIDecisionRequest) string {
	briefing := request.Briefing
	var lines []string
	lines = append(lines, "You are playing HerbieGo as one role in a hidden simultaneous-turn factory simulation.")
	lines = append(lines, fmt.Sprintf("Return either one lookup tool request or one final JSON decision for contract version %s.", request.ContractVersion))
	lines = append(lines, fmt.Sprintf("Role: %s (%s)", briefing.DisplayName, briefing.RoleID))
	lines = append(lines, "Public responsibilities:")
	for _, item := range briefing.PublicResponsibilities {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "Hidden incentives:")
	for _, item := range briefing.HiddenIncentives {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "Decision principles:")
	for _, item := range briefing.DecisionPrinciples {
		lines = append(lines, "- "+item)
	}
	return strings.Join(lines, "\n")
}

// BuildUserPrompt renders the canonical provider-neutral decision prompt.
func BuildUserPrompt(request ports.AIDecisionRequest, retry *ports.RetryFeedback) string {
	var sections []string

	sections = append(sections, "## Contract Header\n"+mustJSON(map[string]any{
		"contract_version": request.ContractVersion,
		"match_id":         request.MatchID,
		"round":            request.Round,
		"role_id":          request.RoleID,
		"provider":         request.Provider,
		"model":            request.Model,
	}))

	sections = append(sections, "## Role Briefing\n"+mustJSON(request.Briefing))
	sections = append(sections, "## Current Round Facts\n"+mustJSON(map[string]any{
		"round_view":      request.RoundView,
		"role_report":     request.RoleReport,
		"previous_action": request.PreviousAction,
	}))
	sections = append(sections, "## Allowed Action Schema\n"+mustJSON(request.AllowedActions))
	sections = append(sections, "## Response Format\n"+mustJSON(map[string]any{
		"response_spec": request.ResponseSpec,
		"decision_example": map[string]any{
			"contract_version": request.ContractVersion,
			"match_id":         request.MatchID,
			"round":            request.Round,
			"role_id":          request.RoleID,
			"action":           exampleAction(request.RoleID, request.RoundView.ActiveTargets),
			"commentary": map[string]any{
				"public_summary": "Explain the decision in one short player-facing sentence.",
				"focus_tags":     []string{"throughput"},
			},
		},
	}))

	if len(request.Tools) > 0 {
		sections = append(sections, "## Tool Lookup\n"+mustJSON(map[string]any{
			"available_tools": request.Tools,
			"tool_call_example": map[string]any{
				"tool_call": map[string]any{
					"tool_name": "show_product_route",
					"arguments": map[string]string{"product_id": "pump"},
				},
			},
			"instructions": []string{
				"If you need more catalog information, return exactly one tool_call JSON object.",
				"After you receive tool results, return the final decision JSON in the main contract format.",
				"Do not combine tool_call with the final decision in the same JSON object.",
			},
		}))
	}

	if len(request.ToolResults) > 0 {
		sections = append(sections, "## Tool Results\n"+mustJSON(request.ToolResults))
	}

	if retry != nil {
		sections = append(sections, "## Retry Feedback\n"+mustJSON(retry))
	}

	sections = append(sections, "Return JSON only. Do not add prose before or after the JSON object.")
	return strings.Join(sections, "\n\n")
}

func mustJSON(value any) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("prompting: marshal prompt section: %v", err))
	}
	return string(data)
}

func exampleAction(roleID domain.RoleID, targets domain.BudgetTargets) domain.RoleAction {
	switch roleID {
	case domain.RoleProcurementManager:
		return domain.RoleAction{
			Procurement: &domain.ProcurementAction{
				Orders: []domain.PurchaseOrderIntent{{PartID: "housing", SupplierID: "forgeco", Quantity: 1}},
			},
		}
	case domain.RoleProductionManager:
		return domain.RoleAction{
			Production: &domain.ProductionAction{
				Releases:           []domain.ProductionRelease{{ProductID: "pump", Quantity: 1}},
				CapacityAllocation: []domain.CapacityAllocation{{WorkstationID: "fabrication", ProductID: "pump", Capacity: 1}},
			},
		}
	case domain.RoleSalesManager:
		return domain.RoleAction{
			Sales: &domain.SalesAction{
				ProductOffers: []domain.ProductOffer{{ProductID: "pump", UnitPrice: 14}},
			},
		}
	case domain.RoleFinanceController:
		return domain.RoleAction{
			Finance: &domain.FinanceAction{NextRoundTargets: targets},
		}
	default:
		return domain.RoleAction{}
	}
}
