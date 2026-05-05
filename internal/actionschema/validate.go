package actionschema

import (
	"fmt"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

type ValidationError struct {
	Path    string
	Message string
}

func ValidateRoleAction(schema RoleSchema, action domain.RoleAction, view domain.RoundView) []ValidationError {
	var errs []ValidationError

	payloadCount := 0
	if action.Procurement != nil {
		payloadCount++
	}
	if action.Production != nil {
		payloadCount++
	}
	if action.Sales != nil {
		payloadCount++
	}
	if action.Finance != nil {
		payloadCount++
	}
	if payloadCount != 1 {
		errs = append(errs, ValidationError{Path: "action", Message: "must contain exactly one populated role action payload"})
	}

	switch schema.RoleID {
	case domain.RoleProcurementManager:
		if action.Procurement == nil {
			errs = append(errs, ValidationError{Path: "action.procurement", Message: "must be populated for procurement_manager"})
			return errs
		}
		errs = append(errs, validateProcurementRows(schema, action.Procurement.Orders)...)
	case domain.RoleProductionManager:
		if action.Production == nil {
			errs = append(errs, ValidationError{Path: "action.production", Message: "must be populated for production_manager"})
			return errs
		}
		errs = append(errs, validateProductionRows(schema, action.Production)...)
	case domain.RoleSalesManager:
		if action.Sales == nil {
			errs = append(errs, ValidationError{Path: "action.sales", Message: "must be populated for sales_manager"})
			return errs
		}
		errs = append(errs, validateSalesRows(schema, action.Sales)...)
	case domain.RoleFinanceController:
		if action.Finance == nil {
			errs = append(errs, ValidationError{Path: "action.finance", Message: "must be populated for finance_controller"})
			return errs
		}
		errs = append(errs, validateFinanceTargets(action.Finance.NextRoundTargets, view)...)
	}

	return errs
}

func FirstError(errs []ValidationError) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s: %s", errs[0].Path, errs[0].Message)
}

func validateProcurementRows(schema RoleSchema, orders []domain.PurchaseOrderIntent) []ValidationError {
	var errs []ValidationError
	partOptions := staticColumnOptions(schema, "orders", "part_id")
	supplierOptions := dependentColumnOptions(schema, "orders", "supplier_id")
	for i, order := range orders {
		if !partOptions[string(order.PartID)] {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.procurement.orders[%d].part_id", i), Message: "must reference a known part in the shared action schema"})
		}
		if !supplierOptions[string(order.PartID)][string(order.SupplierID)] {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.procurement.orders[%d].supplier_id", i), Message: "must be valid for the selected part"})
		}
		if order.Quantity < 0 {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.procurement.orders[%d].quantity", i), Message: "must be non-negative"})
		}
	}
	return errs
}

func validateProductionRows(schema RoleSchema, action *domain.ProductionAction) []ValidationError {
	var errs []ValidationError
	productOptions := staticColumnOptions(schema, "releases", "product_id")
	workstationOptions := staticColumnOptions(schema, "capacity_allocation", "workstation_id")
	allocationProducts := staticColumnOptions(schema, "capacity_allocation", "product_id")
	overtimeStations := staticColumnOptions(schema, "overtime", "workstation_id")

	for i, release := range action.Releases {
		if !productOptions[string(release.ProductID)] {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.production.releases[%d].product_id", i), Message: "must reference a product visible in the shared action schema"})
		}
		if release.Quantity < 0 {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.production.releases[%d].quantity", i), Message: "must be non-negative"})
		}
	}
	for i, allocation := range action.CapacityAllocation {
		if !workstationOptions[string(allocation.WorkstationID)] {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.production.capacity_allocation[%d].workstation_id", i), Message: "must reference a workstation visible in the shared action schema"})
		}
		if !allocationProducts[string(allocation.ProductID)] {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.production.capacity_allocation[%d].product_id", i), Message: "must reference a product visible in the shared action schema"})
		}
		if allocation.Capacity < 0 {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.production.capacity_allocation[%d].capacity", i), Message: "must be non-negative"})
		}
	}
	for i, overtime := range action.Overtime {
		if !overtimeStations[string(overtime.WorkstationID)] {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.production.overtime[%d].workstation_id", i), Message: "must reference a workstation visible in the shared action schema"})
		}
		if overtime.Capacity < 0 {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.production.overtime[%d].capacity", i), Message: "must be non-negative"})
		}
	}
	return errs
}

func validateSalesRows(schema RoleSchema, action *domain.SalesAction) []ValidationError {
	var errs []ValidationError
	productOptions := staticColumnOptions(schema, "product_offers", "product_id")
	for i, offer := range action.ProductOffers {
		if !productOptions[string(offer.ProductID)] {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.sales.product_offers[%d].product_id", i), Message: "must reference a product visible in the shared action schema"})
		}
		if offer.UnitPrice < 0 {
			errs = append(errs, ValidationError{Path: fmt.Sprintf("action.sales.product_offers[%d].unit_price", i), Message: "must be non-negative"})
		}
	}
	return errs
}

func validateFinanceTargets(targets domain.BudgetTargets, view domain.RoundView) []ValidationError {
	var errs []ValidationError
	expectedRound := view.ActiveTargets.EffectiveRound + 1
	if targets.EffectiveRound != expectedRound {
		errs = append(errs, ValidationError{Path: "action.finance.next_round_targets.effective_round", Message: fmt.Sprintf("must equal %d for the next planning round", expectedRound)})
	}
	if targets.ProcurementBudget < 0 {
		errs = append(errs, ValidationError{Path: "action.finance.next_round_targets.procurement_budget", Message: "must be non-negative"})
	}
	if targets.ProductionSpendBudget < 0 {
		errs = append(errs, ValidationError{Path: "action.finance.next_round_targets.production_spend_budget", Message: "must be non-negative"})
	}
	if targets.RevenueTarget < 0 {
		errs = append(errs, ValidationError{Path: "action.finance.next_round_targets.revenue_target", Message: "must be non-negative"})
	}
	if targets.CashFloorTarget < 0 {
		errs = append(errs, ValidationError{Path: "action.finance.next_round_targets.cash_floor_target", Message: "must be non-negative"})
	}
	if targets.DebtCeilingTarget < 0 {
		errs = append(errs, ValidationError{Path: "action.finance.next_round_targets.debt_ceiling_target", Message: "must be non-negative"})
	}
	return errs
}

func staticColumnOptions(schema RoleSchema, fieldID, columnID string) map[string]bool {
	options := make(map[string]bool)
	column, ok := findColumn(schema, fieldID, columnID)
	if !ok {
		return options
	}
	for _, option := range column.Options.Static {
		options[option.Value] = true
	}
	return options
}

func dependentColumnOptions(schema RoleSchema, fieldID, columnID string) map[string]map[string]bool {
	options := make(map[string]map[string]bool)
	column, ok := findColumn(schema, fieldID, columnID)
	if !ok {
		return options
	}
	for key, items := range column.Options.Dependent {
		allowed := make(map[string]bool, len(items))
		for _, item := range items {
			allowed[item.Value] = true
		}
		options[key] = allowed
	}
	return options
}

func findColumn(schema RoleSchema, fieldID, columnID string) (ColumnSpec, bool) {
	for _, field := range schema.Fields {
		if field.ID != fieldID || field.Collection == nil {
			continue
		}
		for _, column := range field.Collection.Columns {
			if column.ID == columnID {
				return column, true
			}
		}
	}
	return ColumnSpec{}, false
}
