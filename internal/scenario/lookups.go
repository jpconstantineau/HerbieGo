package scenario

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

// ValidSuppliersLookup is the human-readable supplier lookup payload shared by
// the AI tool surface and the TUI browser.
type ValidSuppliersLookup struct {
	PartID      domain.PartID
	DisplayName string
	Suppliers   []domain.SupplierID
}

// ProductRouteLookup is the shared route lookup payload.
type ProductRouteLookup struct {
	ProductID     domain.ProductID
	DisplayName   string
	Route         []domain.WorkstationID
	BottleneckID  domain.WorkstationID
	BottleneckWhy string
}

// ProductBOMLookup is the shared bill-of-materials lookup payload.
type ProductBOMLookup struct {
	ProductID    domain.ProductID
	DisplayName  string
	BOM          []domain.BOMLine
	BaseUnitCost domain.Money
}

// CustomerDemandProfileLookup is the shared customer-demand lookup payload.
type CustomerDemandProfileLookup struct {
	CustomerID       domain.CustomerID
	CustomerName     string
	ProductID        domain.ProductID
	ProductName      string
	ReferencePrice   domain.Money
	BaseDemand       domain.Units
	PriceSensitivity int
}

// DemandProfileReference identifies one browsable customer/product demand pair.
type DemandProfileReference struct {
	CustomerID   domain.CustomerID
	CustomerName string
	ProductID    domain.ProductID
	ProductName  string
}

// LookupTools defines the canonical scenario lookup surface. Keep new lookup
// tools here so AI calls and human browsing stay aligned.
func LookupTools() []ports.LookupToolSpec {
	return []ports.LookupToolSpec{
		{
			Name:        "list_valid_suppliers",
			Description: "Return the valid suppliers for one part in the active scenario.",
			Arguments: []ports.LookupToolArgument{
				{Name: "part_id", Description: "Canonical part identifier.", Required: true},
			},
		},
		{
			Name:        "show_product_route",
			Description: "Return the ordered workstation route for one product.",
			Arguments: []ports.LookupToolArgument{
				{Name: "product_id", Description: "Canonical product identifier.", Required: true},
			},
		},
		{
			Name:        "show_product_bom",
			Description: "Return the bill of materials for one product.",
			Arguments: []ports.LookupToolArgument{
				{Name: "product_id", Description: "Canonical product identifier.", Required: true},
			},
		},
		{
			Name:        "show_customer_demand_profile",
			Description: "Return the known demand settings for one customer and product pair.",
			Arguments: []ports.LookupToolArgument{
				{Name: "customer_id", Description: "Canonical customer identifier.", Required: true},
				{Name: "product_id", Description: "Canonical product identifier.", Required: true},
			},
		},
	}
}

// Part returns one canonical part definition.
func (d Definition) Part(partID domain.PartID) (Part, bool) {
	part, ok := d.partsByID()[partID]
	return part, ok
}

// Parts returns all canonical parts in stable display order.
func (d Definition) Parts() []Part {
	parts := slices.Clone(d.ProductionModel.Parts)
	slices.SortFunc(parts, func(left, right Part) int {
		if byName := cmp.Compare(left.DisplayName, right.DisplayName); byName != 0 {
			return byName
		}
		return cmp.Compare(left.ID, right.ID)
	})
	return parts
}

// Product returns one canonical product definition.
func (d Definition) Product(productID domain.ProductID) (Product, bool) {
	product, ok := d.productsByID()[productID]
	return product, ok
}

// Products returns all canonical products in stable display order.
func (d Definition) Products() []Product {
	products := slices.Clone(d.ProductionModel.Products)
	slices.SortFunc(products, func(left, right Product) int {
		if byName := cmp.Compare(left.DisplayName, right.DisplayName); byName != 0 {
			return byName
		}
		return cmp.Compare(left.ID, right.ID)
	})
	return products
}

// Workstation returns one canonical workstation definition.
func (d Definition) Workstation(workstationID domain.WorkstationID) (Workstation, bool) {
	workstation, ok := d.workstationsByID()[workstationID]
	return workstation, ok
}

// Customer returns one canonical customer definition.
func (d Definition) Customer(customerID domain.CustomerID) (CustomerMarket, bool) {
	customer, ok := d.customersByID()[customerID]
	return customer, ok
}

// DemandProfileReferences returns all customer/product demand pairs in stable
// browse order for the human lookup workspace.
func (d Definition) DemandProfileReferences() []DemandProfileReference {
	customers := d.customersByID()
	products := d.productsByID()

	refs := make([]DemandProfileReference, 0)
	for _, customer := range d.MarketModel.Customers {
		productIDs := make([]domain.ProductID, 0, len(customer.DemandByProduct))
		for productID := range customer.DemandByProduct {
			productIDs = append(productIDs, productID)
		}
		slices.Sort(productIDs)
		for _, productID := range productIDs {
			ref := DemandProfileReference{
				CustomerID:   customer.ID,
				CustomerName: customer.DisplayName,
				ProductID:    productID,
				ProductName:  string(productID),
			}
			if product, ok := products[productID]; ok {
				ref.ProductName = product.DisplayName
			}
			if canonical, ok := customers[customer.ID]; ok {
				ref.CustomerName = canonical.DisplayName
			}
			refs = append(refs, ref)
		}
	}
	slices.SortFunc(refs, func(left, right DemandProfileReference) int {
		if byCustomer := cmp.Compare(left.CustomerName, right.CustomerName); byCustomer != 0 {
			return byCustomer
		}
		if byProduct := cmp.Compare(left.ProductName, right.ProductName); byProduct != 0 {
			return byProduct
		}
		if byCustomerID := cmp.Compare(left.CustomerID, right.CustomerID); byCustomerID != 0 {
			return byCustomerID
		}
		return cmp.Compare(left.ProductID, right.ProductID)
	})
	return refs
}

// ListValidSuppliers returns the canonical supplier lookup for one part.
func (d Definition) ListValidSuppliers(partID domain.PartID) (ValidSuppliersLookup, error) {
	part, ok := d.Part(partID)
	if !ok {
		return ValidSuppliersLookup{}, fmt.Errorf("unknown part_id %q", partID)
	}
	suppliers := make([]domain.SupplierID, 0, len(part.AlternateSuppliers)+1)
	for _, supplier := range part.suppliers() {
		suppliers = append(suppliers, supplier.ID)
	}
	return ValidSuppliersLookup{
		PartID:      part.ID,
		DisplayName: part.DisplayName,
		Suppliers:   suppliers,
	}, nil
}

// ShowProductRoute returns the canonical route lookup for one product.
func (d Definition) ShowProductRoute(productID domain.ProductID) (ProductRouteLookup, error) {
	product, ok := d.Product(productID)
	if !ok {
		return ProductRouteLookup{}, fmt.Errorf("unknown product_id %q", productID)
	}
	return ProductRouteLookup{
		ProductID:     product.ID,
		DisplayName:   product.DisplayName,
		Route:         slices.Clone(product.Route),
		BottleneckID:  d.ProductionModel.Bottleneck.WorkstationID,
		BottleneckWhy: d.ProductionModel.Bottleneck.Summary,
	}, nil
}

// ShowProductBOM returns the canonical bill-of-materials lookup for one product.
func (d Definition) ShowProductBOM(productID domain.ProductID) (ProductBOMLookup, error) {
	product, ok := d.Product(productID)
	if !ok {
		return ProductBOMLookup{}, fmt.Errorf("unknown product_id %q", productID)
	}
	return ProductBOMLookup{
		ProductID:    product.ID,
		DisplayName:  product.DisplayName,
		BOM:          slices.Clone(product.BOM),
		BaseUnitCost: product.BaseUnitCost,
	}, nil
}

// ShowCustomerDemandProfile returns the canonical demand profile lookup for one
// customer/product pair.
func (d Definition) ShowCustomerDemandProfile(customerID domain.CustomerID, productID domain.ProductID) (CustomerDemandProfileLookup, error) {
	customer, ok := d.Customer(customerID)
	if !ok {
		return CustomerDemandProfileLookup{}, fmt.Errorf("unknown customer_id %q", customerID)
	}
	profile, ok := customer.DemandByProduct[productID]
	if !ok {
		return CustomerDemandProfileLookup{}, fmt.Errorf("customer %q has no demand profile for product %q", customerID, productID)
	}
	productName := string(productID)
	if product, ok := d.Product(productID); ok {
		productName = product.DisplayName
	}
	return CustomerDemandProfileLookup{
		CustomerID:       customer.ID,
		CustomerName:     customer.DisplayName,
		ProductID:        productID,
		ProductName:      productName,
		ReferencePrice:   profile.ReferencePrice,
		BaseDemand:       profile.BaseDemand,
		PriceSensitivity: profile.PriceSensitivity,
	}, nil
}

// ExecuteLookup adapts the canonical scenario lookups to the AI tool contract.
func (d Definition) ExecuteLookup(call ports.LookupToolCall) (ports.LookupToolResult, error) {
	if d.ID == "" {
		return ports.LookupToolResult{}, fmt.Errorf("scenario lookups are not configured")
	}

	result := ports.LookupToolResult{
		ToolName:  call.ToolName,
		Arguments: maps.Clone(call.Arguments),
	}

	switch call.ToolName {
	case "list_valid_suppliers":
		lookup, err := d.ListValidSuppliers(domain.PartID(call.Arguments["part_id"]))
		if err != nil {
			return ports.LookupToolResult{}, err
		}
		result.Result = map[string]any{
			"part_id":      lookup.PartID,
			"display_name": lookup.DisplayName,
			"suppliers":    slices.Clone(lookup.Suppliers),
		}
	case "show_product_route":
		lookup, err := d.ShowProductRoute(domain.ProductID(call.Arguments["product_id"]))
		if err != nil {
			return ports.LookupToolResult{}, err
		}
		result.Result = map[string]any{
			"product_id":    lookup.ProductID,
			"display_name":  lookup.DisplayName,
			"route":         slices.Clone(lookup.Route),
			"bottleneck_id": lookup.BottleneckID,
		}
	case "show_product_bom":
		lookup, err := d.ShowProductBOM(domain.ProductID(call.Arguments["product_id"]))
		if err != nil {
			return ports.LookupToolResult{}, err
		}
		result.Result = map[string]any{
			"product_id":   lookup.ProductID,
			"display_name": lookup.DisplayName,
			"bom":          slices.Clone(lookup.BOM),
		}
	case "show_customer_demand_profile":
		lookup, err := d.ShowCustomerDemandProfile(
			domain.CustomerID(call.Arguments["customer_id"]),
			domain.ProductID(call.Arguments["product_id"]),
		)
		if err != nil {
			return ports.LookupToolResult{}, err
		}
		result.Result = map[string]any{
			"customer_id":       lookup.CustomerID,
			"customer_name":     lookup.CustomerName,
			"product_id":        lookup.ProductID,
			"reference_price":   lookup.ReferencePrice,
			"base_demand":       lookup.BaseDemand,
			"price_sensitivity": lookup.PriceSensitivity,
		}
	default:
		return ports.LookupToolResult{}, fmt.Errorf("unsupported tool %q", call.ToolName)
	}

	return result, nil
}
