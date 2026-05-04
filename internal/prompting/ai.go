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
	examples := promptExamples(request)
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
	lines = append(lines, renderToolCatalog(request.Tools)...)
	lines = append(lines, "# Response")
	lines = append(lines, "")
	lines = append(lines, "Once you have your decision, communicate it to the rest of the team in JSON format according to the example below.")
	lines = append(lines, "")
	lines = append(lines, mustJSON(map[string]any{
		"action": examples.decisionAction,
		"commentary": map[string]any{
			"public_summary": "Explain the decision in one short player-facing sentence.",
			"focus_tags":     []string{"throughput"},
		},
	}))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("You have the following options for actions to take: %s.", strings.Join(briefing.AllowedActionSummary, "; ")))
	lines = append(lines, "")
	lines = append(lines, "Return either one lookup tool request or one final JSON decision.")
	return strings.Join(lines, "\n")
}

// BuildUserPrompt renders the canonical provider-neutral decision prompt.
func BuildUserPrompt(request ports.AIDecisionRequest, retry *ports.RetryFeedback) string {
	var sections []string
	examples := promptExamples(request)

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
			"action": examples.decisionAction,
			"commentary": map[string]any{
				"public_summary": "Explain the decision in one short player-facing sentence.",
				"focus_tags":     []string{"throughput"},
			},
		},
	}))

	if len(request.Tools) > 0 {
		sections = append(sections, "## Tool Lookup\n"+mustJSON(map[string]any{
			"available_tools":   request.Tools,
			"tool_call_example": examples.toolCallExample,
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

type promptExampleSet struct {
	decisionAction  domain.RoleAction
	toolCallExample map[string]any
}

func promptExamples(request ports.AIDecisionRequest) promptExampleSet {
	selector := exampleSelector{view: request.RoundView}
	return promptExampleSet{
		decisionAction: exampleActionForView(request.RoleID, selector),
		toolCallExample: map[string]any{
			"tool_call": selector.toolCallExample(request.Tools),
		},
	}
}

func exampleActionForView(roleID domain.RoleID, selector exampleSelector) domain.RoleAction {
	switch roleID {
	case domain.RoleProcurementManager:
		return domain.RoleAction{
			Procurement: &domain.ProcurementAction{
				Orders: []domain.PurchaseOrderIntent{{
					PartID:     selector.partID(),
					SupplierID: selector.supplierID(),
					Quantity:   1,
				}},
			},
		}
	case domain.RoleProductionManager:
		productID := selector.productID()
		return domain.RoleAction{
			Production: &domain.ProductionAction{
				Releases: []domain.ProductionRelease{{ProductID: productID, Quantity: 1}},
				CapacityAllocation: []domain.CapacityAllocation{{
					WorkstationID: selector.workstationID(),
					ProductID:     productID,
					Capacity:      1,
				}},
			},
		}
	case domain.RoleSalesManager:
		return domain.RoleAction{
			Sales: &domain.SalesAction{
				ProductOffers: []domain.ProductOffer{{ProductID: selector.productID(), UnitPrice: 14}},
			},
		}
	case domain.RoleFinanceController:
		return domain.RoleAction{
			Finance: &domain.FinanceAction{NextRoundTargets: selector.view.ActiveTargets},
		}
	default:
		return domain.RoleAction{}
	}
}

type exampleSelector struct {
	view domain.RoundView
}

func (s exampleSelector) productID() domain.ProductID {
	for _, item := range s.view.Plant.WIPInventory {
		if item.ProductID != "" {
			return item.ProductID
		}
	}
	for _, item := range s.view.Plant.FinishedInventory {
		if item.ProductID != "" {
			return item.ProductID
		}
	}
	for _, item := range s.view.Plant.Backlog {
		if item.ProductID != "" {
			return item.ProductID
		}
	}
	for _, customer := range s.view.Customers {
		for _, item := range customer.Backlog {
			if item.ProductID != "" {
				return item.ProductID
			}
		}
	}
	for _, round := range s.view.RecentRounds {
		for _, event := range round.Events {
			if productID := productIDFromPayload(event.Payload); productID != "" {
				return productID
			}
		}
	}
	return "product_id"
}

func (s exampleSelector) partID() domain.PartID {
	for _, item := range s.view.Plant.PartsInventory {
		if item.PartID != "" {
			return item.PartID
		}
	}
	for _, item := range s.view.Plant.InTransitSupply {
		if item.PartID != "" {
			return item.PartID
		}
	}
	for _, round := range s.view.RecentRounds {
		for _, event := range round.Events {
			if partID := partIDFromPayload(event.Payload); partID != "" {
				return partID
			}
		}
	}
	return "part_id"
}

func (s exampleSelector) supplierID() domain.SupplierID {
	for _, item := range s.view.Suppliers {
		if item.SupplierID != "" {
			return item.SupplierID
		}
	}
	for _, item := range s.view.Plant.InTransitSupply {
		if item.SupplierID != "" {
			return item.SupplierID
		}
	}
	for _, round := range s.view.RecentRounds {
		for _, event := range round.Events {
			if supplierID := supplierIDFromPayload(event.Payload); supplierID != "" {
				return supplierID
			}
		}
	}
	return "supplier_id"
}

func (s exampleSelector) workstationID() domain.WorkstationID {
	for _, item := range s.view.Plant.Workstations {
		if item.WorkstationID != "" {
			return item.WorkstationID
		}
	}
	for _, item := range s.view.Plant.WIPInventory {
		if item.WorkstationID != "" {
			return item.WorkstationID
		}
	}
	return "workstation_id"
}

func (s exampleSelector) customerID() domain.CustomerID {
	for _, item := range s.view.Customers {
		if item.CustomerID != "" {
			return item.CustomerID
		}
	}
	for _, item := range s.view.Plant.Backlog {
		if item.CustomerID != "" {
			return item.CustomerID
		}
	}
	for _, round := range s.view.RecentRounds {
		for _, event := range round.Events {
			if customerID := customerIDFromPayload(event.Payload); customerID != "" {
				return customerID
			}
		}
	}
	return "customer_id"
}

func (s exampleSelector) toolCallExample(tools []ports.LookupToolSpec) map[string]any {
	for _, tool := range tools {
		if tool.Name == "show_customer_demand_profile" {
			return map[string]any{
				"tool_name": tool.Name,
				"arguments": map[string]string{
					"customer_id": string(s.customerID()),
					"product_id":  string(s.productID()),
				},
			}
		}
	}

	for _, tool := range tools {
		switch tool.Name {
		case "show_product_route", "show_product_bom":
			return map[string]any{
				"tool_name": tool.Name,
				"arguments": map[string]string{
					"product_id": string(s.productID()),
				},
			}
		case "list_valid_suppliers":
			return map[string]any{
				"tool_name": tool.Name,
				"arguments": map[string]string{
					"part_id": string(s.partID()),
				},
			}
		}
	}

	return map[string]any{
		"tool_name": "show_product_route",
		"arguments": map[string]string{
			"product_id": string(s.productID()),
		},
	}
}

func productIDFromPayload(payload map[string]any) domain.ProductID {
	value, _ := payload["product_id"].(string)
	return domain.ProductID(value)
}

func partIDFromPayload(payload map[string]any) domain.PartID {
	value, _ := payload["part_id"].(string)
	return domain.PartID(value)
}

func supplierIDFromPayload(payload map[string]any) domain.SupplierID {
	value, _ := payload["supplier_id"].(string)
	return domain.SupplierID(value)
}

func customerIDFromPayload(payload map[string]any) domain.CustomerID {
	value, _ := payload["customer_id"].(string)
	return domain.CustomerID(value)
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

func renderToolCatalog(tools []ports.LookupToolSpec) []string {
	if len(tools) == 0 {
		return []string{"No scenario lookup tools are available for this match.", ""}
	}

	lines := []string{"## Tool Catalog"}
	for _, tool := range tools {
		lines = append(lines, fmt.Sprintf("- `%s`: %s", tool.Name, tool.Description))
		if len(tool.Arguments) == 0 {
			continue
		}
		lines = append(lines, "  Arguments:")
		for _, argument := range tool.Arguments {
			required := "optional"
			if argument.Required {
				required = "required"
			}
			lines = append(lines, fmt.Sprintf("  - %s (%s): %s", argument.Name, required, argument.Description))
		}
	}
	lines = append(lines, "")
	return lines
}
