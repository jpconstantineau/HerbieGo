package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type lookupView int

const (
	lookupViewSuppliers lookupView = iota
	lookupViewRoutes
	lookupViewBOM
	lookupViewDemand
)

type lookupBrowserState struct {
	active      lookupView
	partIndex   int
	routeIndex  int
	bomIndex    int
	demandIndex int
}

func (m *Model) handleScenarioLookupKey(msg tea.KeyMsg) bool {
	if m.workspace != workspaceScenarioLookup || m.focusedPane != paneHistory {
		return false
	}

	switch msg.String() {
	case "v":
		m.lookup.active = lookupViewSuppliers
		m.status = "Scenario lookup switched to valid suppliers"
		return true
	case "r":
		m.lookup.active = lookupViewRoutes
		m.status = "Scenario lookup switched to product routes"
		return true
	case "b":
		m.lookup.active = lookupViewBOM
		m.status = "Scenario lookup switched to product BOMs"
		return true
	case "d":
		m.lookup.active = lookupViewDemand
		m.status = "Scenario lookup switched to customer demand"
		return true
	case "up", "k":
		m.shiftLookupSelection(-1)
		return true
	case "down", "j":
		m.shiftLookupSelection(1)
		return true
	default:
		return false
	}
}

func (m *Model) shiftLookupSelection(delta int) {
	switch m.lookup.active {
	case lookupViewSuppliers:
		m.lookup.partIndex = cycleIndex(m.lookup.partIndex, len(m.scenario.Parts()), delta)
		if part, ok := m.selectedLookupPart(); ok {
			m.status = fmt.Sprintf("Scenario lookup focused %s suppliers", part.DisplayName)
		}
	case lookupViewRoutes:
		m.lookup.routeIndex = cycleIndex(m.lookup.routeIndex, len(m.scenario.Products()), delta)
		if product, ok := m.selectedLookupRouteProduct(); ok {
			m.status = fmt.Sprintf("Scenario lookup focused %s route", product.DisplayName)
		}
	case lookupViewBOM:
		m.lookup.bomIndex = cycleIndex(m.lookup.bomIndex, len(m.scenario.Products()), delta)
		if product, ok := m.selectedLookupBOMProduct(); ok {
			m.status = fmt.Sprintf("Scenario lookup focused %s BOM", product.DisplayName)
		}
	case lookupViewDemand:
		m.lookup.demandIndex = cycleIndex(m.lookup.demandIndex, len(m.scenario.DemandProfileReferences()), delta)
		if ref, ok := m.selectedDemandReference(); ok {
			m.status = fmt.Sprintf("Scenario lookup focused %s / %s demand", ref.CustomerName, ref.ProductName)
		}
	}
}

func (m Model) renderScenarioLookupWorkspace(width int) []string {
	lines := []string{
		fmt.Sprintf("Scenario lookups for %s", m.scenario.DisplayName),
		"View: browse the same canonical scenario lookup surface used by AI tool calls",
		selectedLookupTabsLine(m.lookup.active),
		"Use v/r/b/d to switch lookup type and up/down to browse entries.",
	}

	switch m.lookup.active {
	case lookupViewRoutes:
		lines = append(lines, "")
		lines = append(lines, m.renderRouteLookup(width)...)
	case lookupViewBOM:
		lines = append(lines, "")
		lines = append(lines, m.renderBOMLookup(width)...)
	case lookupViewDemand:
		lines = append(lines, "")
		lines = append(lines, m.renderDemandLookup(width)...)
	default:
		lines = append(lines, "")
		lines = append(lines, m.renderSupplierLookup(width)...)
	}

	return lines
}

func (m Model) renderSupplierLookup(width int) []string {
	part, ok := m.selectedLookupPart()
	if !ok {
		return []string{"No scenario parts are available."}
	}
	lookup, err := m.scenario.ListValidSuppliers(part.ID)
	if err != nil {
		return []string{wrapLine("Lookup error: "+err.Error(), paneTextWidth(width))}
	}

	return []string{
		fmt.Sprintf("Valid suppliers (%d/%d)", m.lookup.partIndex+1, len(m.scenario.Parts())),
		fmt.Sprintf("Part: %s (%s)", lookup.DisplayName, lookup.PartID),
		"Tool parity: list_valid_suppliers(part_id)",
		fmt.Sprintf("Suppliers: %s", joinIDs(lookup.Suppliers)),
		wrapLine("Browse order: "+joinPartNames(m.scenario.Parts()), paneTextWidth(width)),
	}
}

func (m Model) renderRouteLookup(width int) []string {
	product, ok := m.selectedLookupRouteProduct()
	if !ok {
		return []string{"No scenario products are available."}
	}
	lookup, err := m.scenario.ShowProductRoute(product.ID)
	if err != nil {
		return []string{wrapLine("Lookup error: "+err.Error(), paneTextWidth(width))}
	}

	lines := []string{
		fmt.Sprintf("Product route (%d/%d)", m.lookup.routeIndex+1, len(m.scenario.Products())),
		fmt.Sprintf("Product: %s (%s)", lookup.DisplayName, lookup.ProductID),
		"Tool parity: show_product_route(product_id)",
		fmt.Sprintf("Route: %s", joinWorkstations(lookup.Route)),
		fmt.Sprintf("Bottleneck: %s", lookup.BottleneckID),
	}
	if strings.TrimSpace(lookup.BottleneckWhy) != "" {
		lines = append(lines, wrapLine("Bottleneck context: "+lookup.BottleneckWhy, paneTextWidth(width)))
	}
	lines = append(lines, wrapLine("Browse order: "+joinProductNames(m.scenario.Products()), paneTextWidth(width)))
	return lines
}

func (m Model) renderBOMLookup(width int) []string {
	product, ok := m.selectedLookupBOMProduct()
	if !ok {
		return []string{"No scenario products are available."}
	}
	lookup, err := m.scenario.ShowProductBOM(product.ID)
	if err != nil {
		return []string{wrapLine("Lookup error: "+err.Error(), paneTextWidth(width))}
	}

	lines := []string{
		fmt.Sprintf("Product BOM (%d/%d)", m.lookup.bomIndex+1, len(m.scenario.Products())),
		fmt.Sprintf("Product: %s (%s)", lookup.DisplayName, lookup.ProductID),
		"Tool parity: show_product_bom(product_id)",
		fmt.Sprintf("Base unit cost: %d", lookup.BaseUnitCost),
		"Materials:",
	}
	for _, item := range lookup.BOM {
		lines = append(lines, fmt.Sprintf("- %s x%d", item.PartID, item.Quantity))
	}
	lines = append(lines, wrapLine("Browse order: "+joinProductNames(m.scenario.Products()), paneTextWidth(width)))
	return lines
}

func (m Model) renderDemandLookup(width int) []string {
	ref, ok := m.selectedDemandReference()
	if !ok {
		return []string{"No customer demand profiles are available."}
	}
	lookup, err := m.scenario.ShowCustomerDemandProfile(ref.CustomerID, ref.ProductID)
	if err != nil {
		return []string{wrapLine("Lookup error: "+err.Error(), paneTextWidth(width))}
	}

	return []string{
		fmt.Sprintf("Customer demand (%d/%d)", m.lookup.demandIndex+1, len(m.scenario.DemandProfileReferences())),
		fmt.Sprintf("Customer: %s (%s)", lookup.CustomerName, lookup.CustomerID),
		fmt.Sprintf("Product: %s (%s)", lookup.ProductName, lookup.ProductID),
		"Tool parity: show_customer_demand_profile(customer_id, product_id)",
		fmt.Sprintf("Reference price: %d", lookup.ReferencePrice),
		fmt.Sprintf("Base demand: %d", lookup.BaseDemand),
		fmt.Sprintf("Price sensitivity: %d", lookup.PriceSensitivity),
		wrapLine("Browse order: "+joinDemandNames(m.scenario.DemandProfileReferences()), paneTextWidth(width)),
	}
}

func (m Model) selectedLookupPart() (scenario.Part, bool) {
	parts := m.scenario.Parts()
	if len(parts) == 0 {
		return scenario.Part{}, false
	}
	return parts[clampIndex(m.lookup.partIndex, len(parts))], true
}

func (m Model) selectedLookupRouteProduct() (scenario.Product, bool) {
	products := m.scenario.Products()
	if len(products) == 0 {
		return scenario.Product{}, false
	}
	return products[clampIndex(m.lookup.routeIndex, len(products))], true
}

func (m Model) selectedLookupBOMProduct() (scenario.Product, bool) {
	products := m.scenario.Products()
	if len(products) == 0 {
		return scenario.Product{}, false
	}
	return products[clampIndex(m.lookup.bomIndex, len(products))], true
}

func (m Model) selectedDemandReference() (scenario.DemandProfileReference, bool) {
	refs := m.scenario.DemandProfileReferences()
	if len(refs) == 0 {
		return scenario.DemandProfileReference{}, false
	}
	return refs[clampIndex(m.lookup.demandIndex, len(refs))], true
}

func selectedLookupTabsLine(active lookupView) string {
	items := []struct {
		view  lookupView
		label string
	}{
		{view: lookupViewSuppliers, label: "v suppliers"},
		{view: lookupViewRoutes, label: "r routes"},
		{view: lookupViewBOM, label: "b bom"},
		{view: lookupViewDemand, label: "d demand"},
	}

	labels := make([]string, 0, len(items))
	for _, item := range items {
		label := item.label
		if item.view == active {
			label = "[" + label + "]"
		}
		labels = append(labels, label)
	}
	return "Lookup tabs: " + strings.Join(labels, " | ")
}

func cycleIndex(current, total, delta int) int {
	if total <= 0 {
		return 0
	}
	return (clampIndex(current, total) + delta + total) % total
}

func clampIndex(index, total int) int {
	switch {
	case total <= 0:
		return 0
	case index < 0:
		return 0
	case index >= total:
		return total - 1
	default:
		return index
	}
}

func joinIDs[T ~string](items []T) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, string(item))
	}
	return strings.Join(names, ", ")
}

func joinWorkstations(items []domain.WorkstationID) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, string(item))
	}
	return strings.Join(parts, " -> ")
}

func joinPartNames(items []scenario.Part) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.DisplayName)
	}
	return strings.Join(names, ", ")
}

func joinProductNames(items []scenario.Product) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.DisplayName)
	}
	return strings.Join(names, ", ")
}

func joinDemandNames(items []scenario.DemandProfileReference) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.CustomerName+"/"+item.ProductName)
	}
	return strings.Join(names, ", ")
}
