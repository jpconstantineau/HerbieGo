package prompting

import (
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

func TestBuildSystemPromptUsesRoundViewEntitiesInExamples(t *testing.T) {
	request := exampleRequest()

	procurement := BuildSystemPrompt(withRole(request, domain.RoleProcurementManager))
	if !strings.Contains(procurement, `"part_id": "casing"`) {
		t.Fatalf("procurement prompt = %q, want dynamic part id", procurement)
	}
	if !strings.Contains(procurement, `"supplier_id": "rapidparts"`) {
		t.Fatalf("procurement prompt = %q, want dynamic supplier id", procurement)
	}
	if strings.Contains(procurement, `"part_id": "housing"`) || strings.Contains(procurement, `"supplier_id": "forgeco"`) {
		t.Fatalf("procurement prompt = %q, want no starter procurement ids", procurement)
	}

	production := BuildSystemPrompt(withRole(request, domain.RoleProductionManager))
	if !strings.Contains(production, `"product_id": "compressor"`) {
		t.Fatalf("production prompt = %q, want dynamic product id", production)
	}
	if !strings.Contains(production, `"workstation_id": "milling"`) {
		t.Fatalf("production prompt = %q, want dynamic workstation id", production)
	}
	if strings.Contains(production, `"product_id": "pump"`) || strings.Contains(production, `"workstation_id": "fabrication"`) {
		t.Fatalf("production prompt = %q, want no starter production ids", production)
	}

	sales := BuildSystemPrompt(withRole(request, domain.RoleSalesManager))
	if !strings.Contains(sales, `"product_id": "compressor"`) {
		t.Fatalf("sales prompt = %q, want dynamic sales product id", sales)
	}
	if strings.Contains(sales, `"product_id": "pump"`) {
		t.Fatalf("sales prompt = %q, want no starter sales ids", sales)
	}
	if !strings.Contains(sales, "## Tool Catalog") {
		t.Fatalf("sales prompt = %q, want generic tool catalog section", sales)
	}
	for _, unwanted := range []string{"## Parts", "## Products", "## Vendors", "## Customers"} {
		if strings.Contains(sales, unwanted) {
			t.Fatalf("sales prompt = %q, want no hardcoded tool categories like %q", sales, unwanted)
		}
	}
}

func TestBuildUserPromptUsesRoundViewEntitiesInDecisionAndToolExamples(t *testing.T) {
	request := withRole(exampleRequest(), domain.RoleSalesManager)

	prompt := BuildUserPrompt(request, nil)
	if !strings.Contains(prompt, `"product_id": "compressor"`) {
		t.Fatalf("user prompt = %q, want dynamic product id", prompt)
	}
	if !strings.Contains(prompt, `"customer_id": "metrofab"`) {
		t.Fatalf("user prompt = %q, want dynamic customer id in tool example", prompt)
	}
	if strings.Contains(prompt, `"product_id": "pump"`) || strings.Contains(prompt, `"customer_id": "northbuild"`) {
		t.Fatalf("user prompt = %q, want no starter tool/example ids", prompt)
	}
}

func withRole(request ports.AIDecisionRequest, roleID domain.RoleID) ports.AIDecisionRequest {
	request.RoleID = roleID
	request.Briefing = ports.RoleBriefing{
		RoleID:                 roleID,
		DisplayName:            string(roleID),
		PublicResponsibilities: []string{"Help the plant."},
		HiddenIncentives:       []string{"Protect local goals."},
		DecisionPrinciples:     []string{"Protect throughput."},
		AllowedActionSummary:   []string{"Return valid JSON."},
	}
	request.AllowedActions = ports.AllowedActionSchema{
		RoleID:         roleID,
		RequiredAction: string(roleID),
		JSONSchemaName: "ExampleAction",
		Rules:          []string{"Use live round-view ids."},
	}
	return request
}

func exampleRequest() ports.AIDecisionRequest {
	return ports.AIDecisionRequest{
		RoundView: domain.RoundView{
			Plant: domain.PlantState{
				PartsInventory: []domain.PartInventory{
					{PartID: "casing", OnHandQty: 2, UnitCost: 3},
				},
				WIPInventory: []domain.WIPInventory{
					{ProductID: "compressor", WorkstationID: "milling", Quantity: 1, UnitCost: 9},
				},
				FinishedInventory: []domain.FinishedInventory{
					{ProductID: "filter", OnHandQty: 1, UnitCost: 7},
				},
				Workstations: []domain.WorkstationState{
					{WorkstationID: "milling", CapacityPerRound: 4},
					{WorkstationID: "packout", CapacityPerRound: 3},
				},
				Backlog: []domain.BacklogEntry{
					{CustomerID: "metrofab", ProductID: "compressor", Quantity: 2, OriginRound: 1},
				},
			},
			Customers: []domain.CustomerState{
				{CustomerID: "metrofab", DisplayName: "MetroFab", Sentiment: 6},
			},
			Suppliers: []domain.SupplierState{
				{SupplierID: "rapidparts", DisplayName: "RapidParts", ReliabilityScore: 95},
			},
			ActiveTargets: domain.BudgetTargets{
				EffectiveRound:        3,
				ProcurementBudget:     22,
				ProductionSpendBudget: 18,
				RevenueTarget:         40,
				CashFloorTarget:       10,
				DebtCeilingTarget:     20,
			},
		},
		Tools: []ports.LookupToolSpec{
			{
				Name:        "show_product_route",
				Description: "Return the ordered workstation route for one product.",
				Arguments:   []ports.LookupToolArgument{{Name: "product_id", Required: true}},
			},
			{
				Name:        "show_customer_demand_profile",
				Description: "Return demand settings for one customer/product pair.",
				Arguments: []ports.LookupToolArgument{
					{Name: "customer_id", Required: true},
					{Name: "product_id", Required: true},
				},
			},
		},
		ResponseSpec: ports.ResponseFormatSpec{
			RequireJSONOnly:     true,
			AllowMarkdownFences: true,
			MaxCommentaryChars:  280,
			MaxFocusTags:        4,
		},
	}
}
