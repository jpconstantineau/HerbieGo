package actionschema

import (
	"fmt"
	"slices"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type ValueKind string

const (
	ValueKindChoice  ValueKind = "choice"
	ValueKindInteger ValueKind = "integer"
	ValueKindText    ValueKind = "text"
)

type Option struct {
	Value       string
	Label       string
	Description string
}

type OptionSource struct {
	Static            []Option
	DependencyFieldID string
	Dependent         map[string][]Option
}

func (s OptionSource) Options(row map[string]string) []Option {
	if s.DependencyFieldID == "" {
		return slices.Clone(s.Static)
	}
	if row == nil {
		return nil
	}
	return slices.Clone(s.Dependent[row[s.DependencyFieldID]])
}

type ColumnSpec struct {
	ID          string
	Label       string
	Help        string
	Placeholder string
	Kind        ValueKind
	Required    bool
	Options     OptionSource
}

type CollectionSpec struct {
	AddLabel  string
	EmptyText string
	Columns   []ColumnSpec
}

type FieldSpec struct {
	ID          string
	Label       string
	Help        string
	Placeholder string
	Kind        ValueKind
	Required    bool
	Multiline   bool
	Options     OptionSource
	Collection  *CollectionSpec
}

type RoleSchema struct {
	RoleID          domain.RoleID
	RequiredAction  string
	JSONSchemaName  string
	AllowedSummary  []string
	ValidationRules []string
	Fields          []FieldSpec
}

type CatalogSource interface {
	Parts() []scenario.Part
	Products() []scenario.Product
	Workstations() []scenario.Workstation
}

func Build(definition scenario.Definition, roleID domain.RoleID, _ domain.RoundView) RoleSchema {
	return buildFromPartsProductsWorkstations(definition.ProductionModel.Parts, definition.ProductionModel.Products, definition.ProductionModel.Workstations, roleID)
}

func BuildFromCatalog(source CatalogSource, roleID domain.RoleID, _ domain.RoundView) RoleSchema {
	return buildFromPartsProductsWorkstations(source.Parts(), source.Products(), source.Workstations(), roleID)
}

func buildFromPartsProductsWorkstations(parts []scenario.Part, products []scenario.Product, workstations []scenario.Workstation, roleID domain.RoleID) RoleSchema {
	switch roleID {
	case domain.RoleProcurementManager:
		return procurementSchema(parts)
	case domain.RoleProductionManager:
		return productionSchema(products, workstations)
	case domain.RoleSalesManager:
		return salesSchema(products)
	case domain.RoleFinanceController:
		return financeSchema()
	default:
		return RoleSchema{RoleID: roleID}
	}
}

func procurementSchema(parts []scenario.Part) RoleSchema {
	partOptions := make([]Option, 0, len(parts))
	partSuppliers := make(map[string][]Option, len(parts))
	for _, part := range parts {
		partOptions = append(partOptions, Option{
			Value:       string(part.ID),
			Label:       part.DisplayName,
			Description: fmt.Sprintf("Default supplier %s", part.SupplierID),
		})
		suppliers := make([]Option, 0, len(part.AlternateSuppliers)+1)
		suppliers = append(suppliers, Option{
			Value:       string(part.SupplierID),
			Label:       string(part.SupplierID),
			Description: "Default supplier",
		})
		for _, supplier := range part.AlternateSuppliers {
			suppliers = append(suppliers, Option{
				Value:       string(supplier.ID),
				Label:       string(supplier.ID),
				Description: fmt.Sprintf("Lead time %d round(s)", supplier.LeadTimeRounds),
			})
		}
		partSuppliers[string(part.ID)] = suppliers
	}

	return RoleSchema{
		RoleID:         domain.RoleProcurementManager,
		RequiredAction: "procurement",
		JSONSchemaName: "ProcurementAction",
		AllowedSummary: []string{
			"Return only procurement orders.",
			"Each order chooses a part, a valid supplier for that part, and a non-negative quantity.",
			"Use an empty orders list for a deliberate no-op.",
		},
		ValidationRules: []string{
			"Populate only action.procurement.",
			"orders entries require part_id, supplier_id, and quantity.",
			"supplier_id must be valid for the selected part.",
			"quantity must be non-negative.",
		},
		Fields: []FieldSpec{
			{
				ID:    "orders",
				Label: "Orders",
				Help:  "Add one or more purchase orders. Each row should select a part, choose a valid supplier for that part, and enter a quantity.",
				Collection: &CollectionSpec{
					AddLabel:  "Add order",
					EmptyText: "No purchase orders configured.",
					Columns: []ColumnSpec{
						{ID: "part_id", Label: "Part", Help: "Choose the part to order.", Kind: ValueKindChoice, Required: true, Options: OptionSource{Static: partOptions}},
						{ID: "supplier_id", Label: "Supplier", Help: "Choose a supplier that can provide the selected part.", Kind: ValueKindChoice, Required: true, Options: OptionSource{DependencyFieldID: "part_id", Dependent: partSuppliers}},
						{ID: "quantity", Label: "Quantity", Help: "Enter a whole-number quantity that is 0 or greater.", Placeholder: "0", Kind: ValueKindInteger, Required: true},
					},
				},
			},
			commentaryField(),
		},
	}
}

func productionSchema(products []scenario.Product, workstations []scenario.Workstation) RoleSchema {
	productOptions := productOptions(products)
	workstationOptions := workstationOptions(workstations)

	return RoleSchema{
		RoleID:         domain.RoleProductionManager,
		RequiredAction: "production",
		JSONSchemaName: "ProductionAction",
		AllowedSummary: []string{
			"Return production releases, capacity allocations, and optional overtime allocations.",
			"Each release chooses a product and quantity.",
			"Each capacity allocation chooses a workstation, product, and non-negative capacity.",
			"Each overtime allocation chooses a workstation and non-negative capacity.",
		},
		ValidationRules: []string{
			"Populate only action.production.",
			"releases entries require product_id and quantity.",
			"capacity_allocation entries require workstation_id, product_id, and capacity.",
			"overtime entries are optional; each requires workstation_id and capacity.",
			"quantities and capacity must be non-negative.",
		},
		Fields: []FieldSpec{
			{
				ID:    "releases",
				Label: "Releases",
				Help:  "Add production releases for the products the plant should push through the line this round.",
				Collection: &CollectionSpec{
					AddLabel:  "Add release",
					EmptyText: "No production releases configured.",
					Columns: []ColumnSpec{
						{ID: "product_id", Label: "Product", Help: "Choose the product to release.", Kind: ValueKindChoice, Required: true, Options: OptionSource{Static: productOptions}},
						{ID: "quantity", Label: "Quantity", Help: "Enter a whole-number quantity that is 0 or greater.", Placeholder: "0", Kind: ValueKindInteger, Required: true},
					},
				},
			},
			{
				ID:    "capacity_allocation",
				Label: "Capacity allocation",
				Help:  "Allocate workstation capacity to products for this round.",
				Collection: &CollectionSpec{
					AddLabel:  "Add allocation",
					EmptyText: "No capacity allocations configured.",
					Columns: []ColumnSpec{
						{ID: "workstation_id", Label: "Workstation", Help: "Choose the workstation receiving the capacity.", Kind: ValueKindChoice, Required: true, Options: OptionSource{Static: workstationOptions}},
						{ID: "product_id", Label: "Product", Help: "Choose the product consuming that capacity.", Kind: ValueKindChoice, Required: true, Options: OptionSource{Static: productOptions}},
						{ID: "capacity", Label: "Capacity", Help: "Enter a whole-number capacity allocation that is 0 or greater.", Placeholder: "0", Kind: ValueKindInteger, Required: true},
					},
				},
			},
			{
				ID:    "overtime",
				Label: "Overtime",
				Help:  "Add optional overtime allocations for workstations that need extra capacity.",
				Collection: &CollectionSpec{
					AddLabel:  "Add overtime",
					EmptyText: "No overtime allocations configured.",
					Columns: []ColumnSpec{
						{ID: "workstation_id", Label: "Workstation", Help: "Choose the workstation using overtime.", Kind: ValueKindChoice, Required: true, Options: OptionSource{Static: workstationOptions}},
						{ID: "capacity", Label: "Capacity", Help: "Enter overtime capacity that is 0 or greater.", Placeholder: "0", Kind: ValueKindInteger, Required: true},
					},
				},
			},
			commentaryField(),
		},
	}
}

func salesSchema(products []scenario.Product) RoleSchema {
	productOptions := productOptions(products)
	return RoleSchema{
		RoleID:         domain.RoleSalesManager,
		RequiredAction: "sales",
		JSONSchemaName: "SalesAction",
		AllowedSummary: []string{
			"Return only product offers.",
			"Each offer chooses a product and a non-negative unit price.",
			"Use an empty product_offers list for a deliberate no-op.",
		},
		ValidationRules: []string{
			"Populate only action.sales.",
			"product_offers entries require product_id and unit_price.",
			"unit_price must be non-negative.",
		},
		Fields: []FieldSpec{
			{
				ID:    "product_offers",
				Label: "Offers",
				Help:  "Set the customer-facing offer price for one or more products.",
				Collection: &CollectionSpec{
					AddLabel:  "Add offer",
					EmptyText: "No product offers configured.",
					Columns: []ColumnSpec{
						{ID: "product_id", Label: "Product", Help: "Choose the product being offered.", Kind: ValueKindChoice, Required: true, Options: OptionSource{Static: productOptions}},
						{ID: "unit_price", Label: "Unit price", Help: "Enter a whole-number unit price that is 0 or greater.", Placeholder: "0", Kind: ValueKindInteger, Required: true},
					},
				},
			},
			commentaryField(),
		},
	}
}

func financeSchema() RoleSchema {
	return RoleSchema{
		RoleID:         domain.RoleFinanceController,
		RequiredAction: "finance",
		JSONSchemaName: "FinanceAction",
		AllowedSummary: []string{
			"Return only finance next_round_targets.",
			"Provide every target field in the next_round_targets object.",
			"A safe no-op repeats the currently active targets.",
		},
		ValidationRules: []string{
			"Populate only action.finance.",
			"Provide next_round_targets exactly once.",
			"Each target field must be present and non-negative.",
		},
		Fields: []FieldSpec{
			integerField("procurement_budget", "Procurement budget", "Whole-number procurement budget for the next round."),
			integerField("production_spend_budget", "Production budget", "Whole-number production-spend budget for the next round."),
			integerField("revenue_target", "Revenue target", "Whole-number revenue target for the next round."),
			integerField("cash_floor_target", "Cash floor", "Whole-number cash floor target for the next round."),
			integerField("debt_ceiling_target", "Debt ceiling", "Whole-number debt ceiling target for the next round."),
			commentaryField(),
		},
	}
}

func commentaryField() FieldSpec {
	return FieldSpec{
		ID:          "commentary",
		Label:       "Commentary",
		Help:        "Required public commentary shown after the round is revealed.",
		Placeholder: "Explain your reasoning for this round.",
		Kind:        ValueKindText,
		Required:    true,
		Multiline:   true,
	}
}

func integerField(id, label, help string) FieldSpec {
	return FieldSpec{
		ID:          id,
		Label:       label,
		Help:        help,
		Placeholder: "0",
		Kind:        ValueKindInteger,
		Required:    true,
	}
}

func productOptions(products []scenario.Product) []Option {
	options := make([]Option, 0, len(products))
	for _, product := range products {
		options = append(options, Option{
			Value:       string(product.ID),
			Label:       product.DisplayName,
			Description: fmt.Sprintf("%d route step(s)", len(product.Route)),
		})
	}
	return options
}

func workstationOptions(workstations []scenario.Workstation) []Option {
	options := make([]Option, 0, len(workstations))
	for _, workstation := range workstations {
		options = append(options, Option{
			Value:       string(workstation.ID),
			Label:       workstation.DisplayName,
			Description: fmt.Sprintf("%d capacity per round", workstation.CapacityPerRound),
		})
	}
	return options
}
