package domain

import "slices"

// RoleID identifies a playable role in a match.
type RoleID string

const (
	RoleProcurementManager RoleID = "procurement_manager"
	RoleProductionManager  RoleID = "production_manager"
	RoleSalesManager       RoleID = "sales_manager"
	RoleFinanceController  RoleID = "finance_controller"
)

var canonicalRoles = []RoleID{
	RoleProcurementManager,
	RoleProductionManager,
	RoleSalesManager,
	RoleFinanceController,
}

// CanonicalRoles returns the MVP role identifiers in stable order.
func CanonicalRoles() []RoleID {
	return slices.Clone(canonicalRoles)
}
