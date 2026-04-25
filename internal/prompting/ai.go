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
	lines = append(lines, "# Role")
	lines = append(lines, fmt.Sprintf("You are a %s working for the HerbieGo Manufacturing plant.", briefing.DisplayName))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("In your role, your objective is to %s.", roleObjective(briefing)))
	lines = append(lines, "")
	lines = append(lines, "As part of your role, you have the following incentives:")
	for _, item := range briefing.HiddenIncentives {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "")
	lines = append(lines, "You also have a number of responsibilities towards the plant and rest of the operations. These are:")
	for _, item := range briefing.PublicResponsibilities {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "")
	lines = append(lines, "# Instructions")
	lines = append(lines, "Follow the following steps to")
	lines = append(lines, "- review the weekly plant report")
	lines = append(lines, "- review weekly department report")
	lines = append(lines, "- review action log of the previous weeks")
	lines = append(lines, "- take a decision on what actions should be taken this week to accomplish your objectives")
	lines = append(lines, "")
	lines = append(lines, "The decision you take should follow these principles:")
	for _, item := range briefing.DecisionPrinciples {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "")
	lines = append(lines, "# Context")
	lines = append(lines, fmt.Sprintf("You will be triggered at the start of each week with information on the last week performance. You are in charge of making decisions on %s.", roleDecisionScope(briefing.RoleID)))
	lines = append(lines, "")
	lines = append(lines, "# Tools")
	lines = append(lines, "You have the following tools available to you:")
	lines = append(lines, "")
	for _, section := range groupedToolSections(request.Tools) {
		lines = append(lines, section...)
	}
	lines = append(lines, "# Response")
	lines = append(lines, "")
	lines = append(lines, "Once you have your decision, communicate it to the rest of the team in JSON format according to the example below.")
	lines = append(lines, "")
	lines = append(lines, mustJSON(map[string]any{
		"contract_version": request.ContractVersion,
		"match_id":         request.MatchID,
		"round":            request.Round,
		"role_id":          request.RoleID,
		"action":           exampleAction(request.RoleID, request.RoundView.ActiveTargets),
		"commentary": map[string]any{
			"public_summary": "Explain the decision in one short player-facing sentence.",
			"focus_tags":     []string{"throughput"},
		},
	}))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("You have the following options for actions to take: %s.", strings.Join(briefing.AllowedActionSummary, "; ")))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Return either one lookup tool request or one final JSON decision for contract version %s.", request.ContractVersion))
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

	if request.PriorAIResponse != "" {
		sections = append(sections, "## Prior Tool Call\n"+request.PriorAIResponse)
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

func roleObjective(briefing ports.RoleBriefing) string {
	if len(briefing.DecisionPrinciples) > 0 {
		return strings.ToLower(strings.TrimSuffix(briefing.DecisionPrinciples[0], "."))
	}
	return "support plant-wide profitability and throughput"
}

func roleDecisionScope(roleID domain.RoleID) string {
	switch roleID {
	case domain.RoleProcurementManager:
		return "procurement orders for parts and supplier coverage"
	case domain.RoleProductionManager:
		return "production releases and capacity allocation across workstations"
	case domain.RoleSalesManager:
		return "product pricing and market offers"
	case domain.RoleFinanceController:
		return "next-round financial targets and budget guardrails"
	default:
		return "the decisions assigned to your role"
	}
}

func groupedToolSections(tools []ports.LookupToolSpec) [][]string {
	byCategory := map[string][]ports.LookupToolSpec{
		"Parts":     nil,
		"Products":  nil,
		"Vendors":   nil,
		"Customers": nil,
	}

	for _, tool := range tools {
		for _, category := range toolCategories(tool.Name) {
			byCategory[category] = append(byCategory[category], tool)
		}
	}

	order := []string{"Parts", "Products", "Vendors", "Customers"}
	sections := make([][]string, 0, len(order))
	for _, category := range order {
		lines := []string{"## " + category}
		toolsForCategory := byCategory[category]
		if len(toolsForCategory) == 0 {
			lines = append(lines, "No dedicated lookup tool is available in this category yet.")
			lines = append(lines, "")
			sections = append(sections, lines)
			continue
		}

		for _, tool := range toolsForCategory {
			lines = append(lines, fmt.Sprintf("- `%s`: %s", tool.Name, tool.Description))
			if len(tool.Arguments) > 0 {
				lines = append(lines, "  Arguments:")
				for _, argument := range tool.Arguments {
					required := "optional"
					if argument.Required {
						required = "required"
					}
					lines = append(lines, fmt.Sprintf("  - %s (%s): %s", argument.Name, required, argument.Description))
				}
			}
		}
		lines = append(lines, "")
		sections = append(sections, lines)
	}

	return sections
}

func toolCategories(toolName string) []string {
	switch toolName {
	case "show_product_bom":
		return []string{"Parts", "Products"}
	case "show_product_route":
		return []string{"Products"}
	case "list_valid_suppliers":
		return []string{"Parts", "Vendors"}
	case "show_customer_demand_profile":
		return []string{"Customers"}
	default:
		return []string{"Parts"}
	}
}
