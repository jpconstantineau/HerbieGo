package scenario_test

import (
	"reflect"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestStarterLookupsExposeCanonicalScenarioData(t *testing.T) {
	definition := scenario.Starter()

	tools := scenario.LookupTools()
	if len(tools) != 4 {
		t.Fatalf("LookupTools() len = %d, want 4", len(tools))
	}

	suppliers, err := definition.ListValidSuppliers("housing")
	if err != nil {
		t.Fatalf("ListValidSuppliers() error = %v", err)
	}
	if !reflect.DeepEqual(suppliers.Suppliers, []domain.SupplierID{"forgeco", "prairiefast"}) {
		t.Fatalf("Suppliers = %#v, want [forgeco prairiefast]", suppliers.Suppliers)
	}

	route, err := definition.ShowProductRoute("pump")
	if err != nil {
		t.Fatalf("ShowProductRoute() error = %v", err)
	}
	if !reflect.DeepEqual(route.Route, []domain.WorkstationID{"fabrication", "assembly"}) {
		t.Fatalf("Route = %#v, want fabrication->assembly", route.Route)
	}

	bom, err := definition.ShowProductBOM("valve")
	if err != nil {
		t.Fatalf("ShowProductBOM() error = %v", err)
	}
	if len(bom.BOM) != 2 || bom.BOM[0].PartID != "body" || bom.BOM[1].PartID != "fastener_kit" {
		t.Fatalf("BOM = %#v, want valve starter parts", bom.BOM)
	}

	demand, err := definition.ShowCustomerDemandProfile("northbuild", "pump")
	if err != nil {
		t.Fatalf("ShowCustomerDemandProfile() error = %v", err)
	}
	if demand.ReferencePrice != 14 || demand.BaseDemand != 3 || demand.PriceSensitivity != 1 {
		t.Fatalf("Demand profile = %+v, want starter northbuild/pump profile", demand)
	}
}

func TestStarterExecuteLookupMatchesAIToolContract(t *testing.T) {
	definition := scenario.Starter()

	result, err := definition.ExecuteLookup(ports.LookupToolCall{
		ToolName: "show_customer_demand_profile",
		Arguments: map[string]string{
			"customer_id": "prairieflow",
			"product_id":  "valve",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteLookup() error = %v", err)
	}

	want := map[string]any{
		"customer_id":       domain.CustomerID("prairieflow"),
		"customer_name":     "PrairieFlow",
		"product_id":        domain.ProductID("valve"),
		"reference_price":   domain.Money(8),
		"base_demand":       domain.Units(3),
		"price_sensitivity": 1,
	}
	if !reflect.DeepEqual(result.Result, want) {
		t.Fatalf("Result = %#v, want %#v", result.Result, want)
	}
}
