package tui

import (
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

// ScenarioReader defines the scenario surface the TUI needs for browse and
// validation workflows without depending on a concrete scenario.Definition.
type ScenarioReader interface {
	ScenarioDisplayName() string
	Parts() []scenario.Part
	Products() []scenario.Product
	Workstations() []scenario.Workstation
	DemandProfileReferences() []scenario.DemandProfileReference
	ListValidSuppliers(partID domain.PartID) (scenario.ValidSuppliersLookup, error)
	ShowProductRoute(productID domain.ProductID) (scenario.ProductRouteLookup, error)
	ShowProductBOM(productID domain.ProductID) (scenario.ProductBOMLookup, error)
	ShowCustomerDemandProfile(customerID domain.CustomerID, productID domain.ProductID) (scenario.CustomerDemandProfileLookup, error)
}
