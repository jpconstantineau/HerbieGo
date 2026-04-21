package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type terminalController struct {
	scenario scenario.Definition
	reader   *bufio.Reader
	writer   io.Writer
}

func newTerminalController(definition scenario.Definition, input io.Reader, output io.Writer) *terminalController {
	return &terminalController{
		scenario: definition,
		reader:   bufio.NewReader(input),
		writer:   output,
	}
}

func (c *terminalController) submitRound(_ context.Context, request ports.RoundRequest) (domain.ActionSubmission, error) {
	for {
		c.renderRoundIntro(request)

		action, err := c.draftAction(request)
		if err != nil {
			return domain.ActionSubmission{}, err
		}

		commentary, err := c.promptRequiredLine("Commentary", "Explain your reasoning for this round.")
		if err != nil {
			return domain.ActionSubmission{}, err
		}

		submission := domain.ActionSubmission{
			Action: action,
			Commentary: domain.CommentaryRecord{
				ActorID:    domain.ActorID(request.Assignment.PlayerID),
				Visibility: domain.CommentaryPublic,
				Body:       commentary,
			},
		}

		c.renderSubmissionDraft(request, submission)
		confirmed, err := c.promptYesNo("Submit this action", true)
		if err != nil {
			return domain.ActionSubmission{}, err
		}
		if confirmed {
			return submission, nil
		}

		fmt.Fprintln(c.writer, "Submission discarded. Let's draft it again.")
	}
}

func (c *terminalController) renderRoundIntro(request ports.RoundRequest) {
	fmt.Fprintln(c.writer)
	fmt.Fprintf(c.writer, "=== Round %d: %s ===\n", request.RoleView.Round, displayRoleName(request.Assignment.RoleID))
	fmt.Fprintf(c.writer, "Cash %d | Debt %d | Backlog %d | Revenue %d | Profit %d\n",
		request.RoleView.Plant.Cash,
		request.RoleView.Plant.Debt,
		len(request.RoleView.Plant.Backlog),
		request.RoleView.Metrics.ThroughputRevenue,
		request.RoleView.Metrics.RoundProfit,
	)
	fmt.Fprintf(c.writer, "Bonus focus: %s\n", request.RoleReport.BonusReminder)
	for _, line := range request.RoleReport.Department.DetailLines {
		fmt.Fprintf(c.writer, "- %s\n", line)
	}
	if request.PreviousAcceptedAction != nil {
		fmt.Fprintf(c.writer, "Previous commentary: %s\n", request.PreviousAcceptedAction.Commentary.Body)
	}
}

func (c *terminalController) draftAction(request ports.RoundRequest) (domain.RoleAction, error) {
	switch request.Assignment.RoleID {
	case domain.RoleProcurementManager:
		orders, err := c.promptPurchaseOrders()
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{
			Procurement: &domain.ProcurementAction{Orders: orders},
		}, nil
	case domain.RoleProductionManager:
		releases, err := c.promptProductionReleases()
		if err != nil {
			return domain.RoleAction{}, err
		}
		allocations, err := c.promptCapacityAllocations()
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{
			Production: &domain.ProductionAction{
				Releases:           releases,
				CapacityAllocation: allocations,
			},
		}, nil
	case domain.RoleSalesManager:
		offers, err := c.promptProductOffers()
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{
			Sales: &domain.SalesAction{ProductOffers: offers},
		}, nil
	case domain.RoleFinanceController:
		targets, err := c.promptBudgetTargets(request.RoleView.ActiveTargets)
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{
			Finance: &domain.FinanceAction{NextRoundTargets: targets},
		}, nil
	default:
		return domain.RoleAction{}, fmt.Errorf("unsupported role %q", request.Assignment.RoleID)
	}
}

func (c *terminalController) promptPurchaseOrders() ([]domain.PurchaseOrderIntent, error) {
	fmt.Fprintf(c.writer, "Available parts: %s\n", strings.Join(c.partOptions(), ", "))
	count, err := c.promptCount("How many purchase orders", 0)
	if err != nil {
		return nil, err
	}

	orders := make([]domain.PurchaseOrderIntent, 0, count)
	for index := range count {
		partID, err := c.promptChoice(fmt.Sprintf("Order %d part", index+1), c.partOptions())
		if err != nil {
			return nil, err
		}
		quantity, err := c.promptNonNegativeInt(fmt.Sprintf("Order %d quantity", index+1), 0)
		if err != nil {
			return nil, err
		}
		orders = append(orders, domain.PurchaseOrderIntent{
			PartID:     domain.PartID(partID),
			SupplierID: c.supplierForPart(domain.PartID(partID)),
			Quantity:   domain.Units(quantity),
		})
	}

	return orders, nil
}

func (c *terminalController) promptProductionReleases() ([]domain.ProductionRelease, error) {
	fmt.Fprintf(c.writer, "Products: %s\n", strings.Join(c.productOptions(), ", "))
	count, err := c.promptCount("How many production releases", 0)
	if err != nil {
		return nil, err
	}

	releases := make([]domain.ProductionRelease, 0, count)
	for index := range count {
		productID, err := c.promptChoice(fmt.Sprintf("Release %d product", index+1), c.productOptions())
		if err != nil {
			return nil, err
		}
		quantity, err := c.promptNonNegativeInt(fmt.Sprintf("Release %d quantity", index+1), 0)
		if err != nil {
			return nil, err
		}
		releases = append(releases, domain.ProductionRelease{
			ProductID: domain.ProductID(productID),
			Quantity:  domain.Units(quantity),
		})
	}

	return releases, nil
}

func (c *terminalController) promptCapacityAllocations() ([]domain.CapacityAllocation, error) {
	fmt.Fprintf(c.writer, "Workstations: %s\n", strings.Join(c.workstationOptions(), ", "))
	count, err := c.promptCount("How many capacity allocations", 0)
	if err != nil {
		return nil, err
	}

	allocations := make([]domain.CapacityAllocation, 0, count)
	for index := range count {
		workstationID, err := c.promptChoice(fmt.Sprintf("Allocation %d workstation", index+1), c.workstationOptions())
		if err != nil {
			return nil, err
		}
		productID, err := c.promptChoice(fmt.Sprintf("Allocation %d product", index+1), c.productOptions())
		if err != nil {
			return nil, err
		}
		capacity, err := c.promptNonNegativeInt(fmt.Sprintf("Allocation %d capacity", index+1), 0)
		if err != nil {
			return nil, err
		}
		allocations = append(allocations, domain.CapacityAllocation{
			WorkstationID: domain.WorkstationID(workstationID),
			ProductID:     domain.ProductID(productID),
			Capacity:      domain.CapacityUnits(capacity),
		})
	}

	return allocations, nil
}

func (c *terminalController) promptProductOffers() ([]domain.ProductOffer, error) {
	fmt.Fprintf(c.writer, "Products: %s\n", strings.Join(c.productOptions(), ", "))
	count, err := c.promptCount("How many product offers", 0)
	if err != nil {
		return nil, err
	}

	offers := make([]domain.ProductOffer, 0, count)
	for index := range count {
		productID, err := c.promptChoice(fmt.Sprintf("Offer %d product", index+1), c.productOptions())
		if err != nil {
			return nil, err
		}
		unitPrice, err := c.promptNonNegativeInt(fmt.Sprintf("Offer %d unit price", index+1), 0)
		if err != nil {
			return nil, err
		}
		offers = append(offers, domain.ProductOffer{
			ProductID: domain.ProductID(productID),
			UnitPrice: domain.Money(unitPrice),
		})
	}

	return offers, nil
}

func (c *terminalController) promptBudgetTargets(current domain.BudgetTargets) (domain.BudgetTargets, error) {
	fmt.Fprintln(c.writer, "Leave the defaults in place if you just want to keep the current target posture.")

	procurementBudget, err := c.promptNonNegativeInt("Next procurement budget", int(current.ProcurementBudget))
	if err != nil {
		return domain.BudgetTargets{}, err
	}
	productionBudget, err := c.promptNonNegativeInt("Next production spend budget", int(current.ProductionSpendBudget))
	if err != nil {
		return domain.BudgetTargets{}, err
	}
	revenueTarget, err := c.promptNonNegativeInt("Next revenue target", int(current.RevenueTarget))
	if err != nil {
		return domain.BudgetTargets{}, err
	}
	cashFloor, err := c.promptNonNegativeInt("Next cash floor target", int(current.CashFloorTarget))
	if err != nil {
		return domain.BudgetTargets{}, err
	}
	debtCeiling, err := c.promptNonNegativeInt("Next debt ceiling target", int(current.DebtCeilingTarget))
	if err != nil {
		return domain.BudgetTargets{}, err
	}

	return domain.BudgetTargets{
		EffectiveRound:        current.EffectiveRound + 1,
		ProcurementBudget:     domain.Money(procurementBudget),
		ProductionSpendBudget: domain.Money(productionBudget),
		RevenueTarget:         domain.Money(revenueTarget),
		CashFloorTarget:       domain.Money(cashFloor),
		DebtCeilingTarget:     domain.Money(debtCeiling),
	}, nil
}

func (c *terminalController) renderSubmissionDraft(request ports.RoundRequest, submission domain.ActionSubmission) {
	fmt.Fprintln(c.writer)
	fmt.Fprintln(c.writer, "Draft submission:")
	fmt.Fprintf(c.writer, "Role: %s\n", displayRoleName(request.Assignment.RoleID))
	for _, line := range summarizeAction(submission.Action) {
		fmt.Fprintf(c.writer, "- %s\n", line)
	}
	fmt.Fprintf(c.writer, "Commentary: %s\n", submission.Commentary.Body)
}

func summarizeAction(action domain.RoleAction) []string {
	switch {
	case action.Procurement != nil:
		if len(action.Procurement.Orders) == 0 {
			return []string{"No purchase orders"}
		}

		lines := make([]string, 0, len(action.Procurement.Orders))
		for _, order := range action.Procurement.Orders {
			lines = append(lines, fmt.Sprintf("Order %d of %s from %s", order.Quantity, order.PartID, order.SupplierID))
		}
		return lines
	case action.Production != nil:
		lines := make([]string, 0, len(action.Production.Releases)+len(action.Production.CapacityAllocation))
		if len(action.Production.Releases) == 0 {
			lines = append(lines, "No production releases")
		}
		for _, release := range action.Production.Releases {
			lines = append(lines, fmt.Sprintf("Release %d of %s", release.Quantity, release.ProductID))
		}
		if len(action.Production.CapacityAllocation) == 0 {
			lines = append(lines, "No capacity allocations")
		}
		for _, allocation := range action.Production.CapacityAllocation {
			lines = append(lines, fmt.Sprintf("Allocate %d capacity at %s for %s", allocation.Capacity, allocation.WorkstationID, allocation.ProductID))
		}
		return lines
	case action.Sales != nil:
		if len(action.Sales.ProductOffers) == 0 {
			return []string{"No product offers"}
		}

		lines := make([]string, 0, len(action.Sales.ProductOffers))
		for _, offer := range action.Sales.ProductOffers {
			lines = append(lines, fmt.Sprintf("Offer %s at %d", offer.ProductID, offer.UnitPrice))
		}
		return lines
	case action.Finance != nil:
		targets := action.Finance.NextRoundTargets
		return []string{
			fmt.Sprintf("Effective round %d", targets.EffectiveRound),
			fmt.Sprintf("Procurement budget %d", targets.ProcurementBudget),
			fmt.Sprintf("Production budget %d", targets.ProductionSpendBudget),
			fmt.Sprintf("Revenue target %d", targets.RevenueTarget),
			fmt.Sprintf("Cash floor %d", targets.CashFloorTarget),
			fmt.Sprintf("Debt ceiling %d", targets.DebtCeilingTarget),
		}
	default:
		return []string{"No action payload"}
	}
}

func (c *terminalController) promptCount(label string, defaultValue int) (int, error) {
	return c.promptNonNegativeInt(label, defaultValue)
}

func (c *terminalController) promptChoice(label string, options []string) (string, error) {
	allowed := make(map[string]string, len(options))
	for _, option := range options {
		allowed[strings.ToLower(option)] = option
	}

	for {
		value, err := c.promptLine(label, "")
		if err != nil {
			return "", err
		}
		choice, ok := allowed[strings.ToLower(value)]
		if ok {
			return choice, nil
		}
		fmt.Fprintf(c.writer, "Choose one of: %s\n", strings.Join(options, ", "))
	}
}

func (c *terminalController) promptNonNegativeInt(label string, defaultValue int) (int, error) {
	for {
		value, err := c.promptLine(label, strconv.Itoa(defaultValue))
		if err != nil {
			return 0, err
		}

		number, convErr := strconv.Atoi(value)
		if convErr == nil && number >= 0 {
			return number, nil
		}
		fmt.Fprintln(c.writer, "Enter a whole number that is 0 or greater.")
	}
}

func (c *terminalController) promptRequiredLine(label, help string) (string, error) {
	for {
		value, err := c.promptLine(label, "")
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(value) != "" {
			return value, nil
		}
		fmt.Fprintln(c.writer, help)
	}
}

func (c *terminalController) promptYesNo(label string, defaultYes bool) (bool, error) {
	defaultText := "y/N"
	if defaultYes {
		defaultText = "Y/n"
	}

	for {
		fmt.Fprintf(c.writer, "%s [%s]: ", label, defaultText)
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(line)) {
		case "":
			return defaultYes, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintln(c.writer, "Enter y or n.")
		}
	}
}

func (c *terminalController) promptLine(label, defaultValue string) (string, error) {
	if defaultValue == "" {
		fmt.Fprintf(c.writer, "%s: ", label)
	} else {
		fmt.Fprintf(c.writer, "%s [%s]: ", label, defaultValue)
	}

	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(line)
	if value == "" && defaultValue != "" {
		return defaultValue, nil
	}
	return value, nil
}

func (c *terminalController) partOptions() []string {
	options := make([]string, 0, len(c.scenario.ProductionModel.Parts))
	for _, part := range c.scenario.ProductionModel.Parts {
		options = append(options, string(part.ID))
	}
	return options
}

func (c *terminalController) productOptions() []string {
	options := make([]string, 0, len(c.scenario.ProductionModel.Products))
	for _, product := range c.scenario.ProductionModel.Products {
		options = append(options, string(product.ID))
	}
	return options
}

func (c *terminalController) workstationOptions() []string {
	options := make([]string, 0, len(c.scenario.ProductionModel.Workstations))
	for _, workstation := range c.scenario.ProductionModel.Workstations {
		options = append(options, string(workstation.ID))
	}
	return options
}

func (c *terminalController) supplierForPart(partID domain.PartID) domain.SupplierID {
	for _, part := range c.scenario.ProductionModel.Parts {
		if part.ID == partID {
			return part.SupplierID
		}
	}
	return ""
}

func displayRoleName(roleID domain.RoleID) string {
	switch roleID {
	case domain.RoleProcurementManager:
		return "Procurement Manager"
	case domain.RoleProductionManager:
		return "Production Manager"
	case domain.RoleSalesManager:
		return "Sales Manager"
	case domain.RoleFinanceController:
		return "Finance Controller"
	default:
		return string(roleID)
	}
}
