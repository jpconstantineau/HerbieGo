package actionschema

import (
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestBuildProcurementSchemaIncludesDependentSupplierOptions(t *testing.T) {
	spec := Build(scenario.Starter(), domain.RoleProcurementManager, domain.RoundView{})
	if spec.RequiredAction != "procurement" {
		t.Fatalf("RequiredAction = %q, want procurement", spec.RequiredAction)
	}
	if len(spec.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(spec.Fields))
	}

	orders := spec.Fields[0]
	if orders.Collection == nil {
		t.Fatalf("orders.Collection = nil, want collection metadata")
	}

	var supplierColumn ColumnSpec
	for _, column := range orders.Collection.Columns {
		if column.ID == "supplier_id" {
			supplierColumn = column
			break
		}
	}
	if supplierColumn.ID == "" {
		t.Fatalf("supplier_id column not found in procurement orders schema")
	}

	options := supplierColumn.Options.Options(map[string]string{"part_id": "housing"})
	if len(options) != 2 {
		t.Fatalf("len(options for housing) = %d, want 2", len(options))
	}
	if options[0].Value != "forgeco" || options[1].Value != "prairiefast" {
		t.Fatalf("housing supplier options = %#v, want forgeco/prairiefast", options)
	}
}

func TestBuildProductionSchemaIncludesOvertimeCollection(t *testing.T) {
	spec := Build(scenario.Starter(), domain.RoleProductionManager, domain.RoundView{})
	if spec.RequiredAction != "production" {
		t.Fatalf("RequiredAction = %q, want production", spec.RequiredAction)
	}

	foundOvertime := false
	for _, field := range spec.Fields {
		if field.ID == "overtime" {
			foundOvertime = true
			if field.Collection == nil || len(field.Collection.Columns) != 2 {
				t.Fatalf("overtime collection columns = %#v, want workstation and capacity", field.Collection)
			}
		}
	}
	if !foundOvertime {
		t.Fatalf("production schema missing overtime collection")
	}
}

func TestBuildFinanceSchemaKeepsScalarTargetsAndCommentary(t *testing.T) {
	spec := Build(scenario.Starter(), domain.RoleFinanceController, domain.RoundView{})
	if spec.RequiredAction != "finance" {
		t.Fatalf("RequiredAction = %q, want finance", spec.RequiredAction)
	}
	if len(spec.Fields) != 6 {
		t.Fatalf("len(Fields) = %d, want 6", len(spec.Fields))
	}
	if got := spec.Fields[len(spec.Fields)-1].ID; got != "commentary" {
		t.Fatalf("last field ID = %q, want commentary", got)
	}
}

func TestValidateRoleActionRejectsSupplierOutsidePartChoices(t *testing.T) {
	spec := Build(scenario.Starter(), domain.RoleProcurementManager, domain.RoundView{})
	errs := ValidateRoleAction(spec, domain.RoleAction{
		Procurement: &domain.ProcurementAction{
			Orders: []domain.PurchaseOrderIntent{{
				PartID:     "housing",
				SupplierID: "sealworks",
				Quantity:   2,
			}},
		},
	}, domain.RoundView{})
	if len(errs) == 0 {
		t.Fatalf("ValidateRoleAction() = nil, want supplier validation error")
	}
	if errs[0].Path != "action.procurement.orders[0].supplier_id" {
		t.Fatalf("errs[0].Path = %q, want supplier_id path", errs[0].Path)
	}
}

func TestValidateRoleActionRejectsFinanceTargetsForWrongRound(t *testing.T) {
	spec := Build(scenario.Starter(), domain.RoleFinanceController, domain.RoundView{})
	errs := ValidateRoleAction(spec, domain.RoleAction{
		Finance: &domain.FinanceAction{
			NextRoundTargets: domain.BudgetTargets{
				EffectiveRound:        4,
				ProcurementBudget:     1,
				ProductionSpendBudget: 1,
				RevenueTarget:         1,
				CashFloorTarget:       1,
				DebtCeilingTarget:     1,
			},
		},
	}, domain.RoundView{ActiveTargets: domain.BudgetTargets{EffectiveRound: 1}})
	if len(errs) == 0 {
		t.Fatalf("ValidateRoleAction() = nil, want effective_round validation error")
	}
	if errs[0].Path != "action.finance.next_round_targets.effective_round" {
		t.Fatalf("errs[0].Path = %q, want effective_round path", errs[0].Path)
	}
}
