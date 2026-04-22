package tui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/domain"
)

type draftStage int

const (
	draftStageEditing draftStage = iota
	draftStageReview
	draftStageSubmitted
)

type draftField int

const (
	fieldProcurementOrders draftField = iota
	fieldProductionReleases
	fieldProductionAllocations
	fieldSalesOffers
	fieldFinanceProcurementBudget
	fieldFinanceProductionBudget
	fieldFinanceRevenueTarget
	fieldFinanceCashFloor
	fieldFinanceDebtCeiling
	fieldCommentary
)

type actionDraft struct {
	stage         draftStage
	selectedField int
	editing       bool
	inputBuffer   string
	status        string
	errorText     string

	procurementOrders     string
	productionReleases    string
	productionAllocations string
	salesOffers           string
	financeProcurement    string
	financeProduction     string
	financeRevenue        string
	financeCashFloor      string
	financeDebtCeiling    string
	commentary            string

	submission *domain.ActionSubmission
}

type fieldSpec struct {
	id          draftField
	label       string
	value       string
	placeholder string
	help        string
}

func (m *Model) handleActionEntryKey(msg tea.KeyMsg) bool {
	if m.workspace != workspaceActionEntry {
		return false
	}

	assignment := m.selectedAssignment()
	if !assignment.IsHuman {
		return false
	}

	draft := m.currentDraft()
	if draft.stage == draftStageSubmitted {
		return false
	}

	if draft.editing {
		return m.handleEditingKey(msg, assignment.RoleID)
	}

	switch msg.String() {
	case "up", "k":
		if draft.selectedField > 0 {
			draft.selectedField--
			draft.status = fmt.Sprintf("Focused %s", m.actionFields()[draft.selectedField].label)
		}
		m.drafts[assignment.RoleID] = draft
		return true
	case "down", "j":
		if draft.selectedField < len(m.actionFields())-1 {
			draft.selectedField++
			draft.status = fmt.Sprintf("Focused %s", m.actionFields()[draft.selectedField].label)
		}
		m.drafts[assignment.RoleID] = draft
		return true
	case "enter", "e":
		draft.editing = true
		draft.inputBuffer = m.actionFields()[draft.selectedField].value
		draft.errorText = ""
		draft.status = fmt.Sprintf("Editing %s", m.actionFields()[draft.selectedField].label)
		m.drafts[assignment.RoleID] = draft
		return true
	case "r":
		submission, err := m.buildSubmissionDraft(draft)
		if err != nil {
			draft.errorText = err.Error()
			draft.status = "Review blocked until the draft is valid"
			m.drafts[assignment.RoleID] = draft
			return true
		}
		draft.stage = draftStageReview
		draft.errorText = ""
		draft.status = "Draft ready for review"
		draft.submission = &submission
		m.drafts[assignment.RoleID] = draft
		return true
	case "s":
		if draft.stage == draftStageReview {
			submission, err := m.buildSubmissionDraft(draft)
			if err != nil {
				draft.errorText = err.Error()
				draft.status = "Submit blocked until the draft is valid"
				m.drafts[assignment.RoleID] = draft
				return true
			}
			draft.stage = draftStageSubmitted
			draft.submission = &submission
			draft.status = "Submission locked for this round"
			draft.errorText = ""
			m.drafts[assignment.RoleID] = draft
			m.status = fmt.Sprintf("%s submitted and locked for round %d", m.roleTitle(), m.state.CurrentRound)
			return true
		}
	case "b":
		if draft.stage == draftStageReview {
			draft.stage = draftStageEditing
			draft.status = "Returned to editing"
			m.drafts[assignment.RoleID] = draft
			return true
		}
	}

	return false
}

func (m *Model) handleEditingKey(msg tea.KeyMsg, roleID domain.RoleID) bool {
	draft := m.currentDraft()
	switch msg.Type {
	case tea.KeyEsc:
		draft.editing = false
		draft.inputBuffer = ""
		draft.status = "Edit cancelled"
		m.drafts[roleID] = draft
		return true
	case tea.KeyEnter:
		spec := m.actionFields()[draft.selectedField]
		m.setDraftField(&draft, spec.id, draft.inputBuffer)
		draft.editing = false
		draft.inputBuffer = ""
		draft.status = fmt.Sprintf("Saved %s", spec.label)
		draft.errorText = ""
		m.drafts[roleID] = draft
		return true
	case tea.KeyBackspace:
		if len(draft.inputBuffer) > 0 {
			draft.inputBuffer = draft.inputBuffer[:len(draft.inputBuffer)-1]
			m.drafts[roleID] = draft
		}
		return true
	case tea.KeySpace:
		draft.inputBuffer += " "
		m.drafts[roleID] = draft
		return true
	case tea.KeyRunes:
		for _, r := range msg.Runes {
			if unicode.IsPrint(r) {
				draft.inputBuffer += string(r)
			}
		}
		m.drafts[roleID] = draft
		return true
	default:
		return true
	}
}

func (m Model) renderActionEntryWorkspace(width int) []string {
	assignment := m.selectedAssignment()
	lines := []string{
		fmt.Sprintf("Action entry for %s", m.roleTitle()),
		"View: draft, review, and submit a private turn without leaving the center workspace",
	}

	if !assignment.IsHuman {
		return append(lines,
			"",
			"This role is AI-controlled in the current match configuration.",
			"Use report/feed/archive views to inspect plant context while the AI submits through the shared player interface.",
		)
	}

	draft := m.currentDraft()
	if previous := m.previousAcceptedAction(assignment.RoleID); previous != nil {
		lines = append(lines, "", "Previous accepted commentary", wrapLine(previous.Commentary.Body, width-4))
	}

	if draft.stage == draftStageReview {
		lines = append(lines, "", "Review draft", "Press s to submit and lock, or b to return to editing.")
		if draft.submission != nil {
			for _, line := range summarizeSubmission(*draft.submission) {
				lines = append(lines, wrapLine("- "+line, width-4))
			}
		}
		return append(lines, m.renderDraftStatus(width, draft)...)
	}

	if draft.stage == draftStageSubmitted {
		lines = append(lines,
			"",
			"Submission locked for this round.",
			waitingOnSummary(m.effectiveRoundFlow().WaitingOnRoles),
			"Current-turn details stay out of the shared round feed until reveal.",
		)
		if draft.submission != nil {
			lines = append(lines, "")
			for _, line := range summarizeSubmission(*draft.submission) {
				lines = append(lines, wrapLine("- "+line, width-4))
			}
		}
		return append(lines, m.renderDraftStatus(width, draft)...)
	}

	lines = append(lines,
		"",
		"Editing flow",
		"Use up/down to move fields, enter to edit/save, esc to cancel a field edit, and r to review.",
	)
	for index, field := range m.actionFields() {
		cursor := " "
		value := field.value
		if index == draft.selectedField {
			cursor = ">"
			if draft.editing {
				value = draft.inputBuffer
			}
		}
		if strings.TrimSpace(value) == "" {
			value = field.placeholder
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s", cursor, field.label, value))
		lines = append(lines, wrapLine("  "+field.help, width-4))
	}

	return append(lines, m.renderDraftStatus(width, draft)...)
}

func (m Model) renderDraftStatus(width int, draft actionDraft) []string {
	var lines []string
	if strings.TrimSpace(draft.status) != "" {
		lines = append(lines, "", wrapLine("Status: "+draft.status, width-4))
	}
	if strings.TrimSpace(draft.errorText) != "" {
		lines = append(lines, wrapLine("Validation: "+draft.errorText, width-4))
	}
	return lines
}

func (m Model) actionFields() []fieldSpec {
	draft := m.currentDraft()
	switch m.selectedAssignment().RoleID {
	case domain.RoleProcurementManager:
		return []fieldSpec{
			{id: fieldProcurementOrders, label: "Orders", value: draft.procurementOrders, placeholder: "housing=2, seal_kit=1", help: "Comma-separated part=quantity entries. Leave blank for no orders."},
			{id: fieldCommentary, label: "Commentary", value: draft.commentary, placeholder: "Explain your reasoning for this round.", help: "Required public commentary shown after the round is revealed."},
		}
	case domain.RoleProductionManager:
		return []fieldSpec{
			{id: fieldProductionReleases, label: "Releases", value: draft.productionReleases, placeholder: "pump=2, valve=1", help: "Comma-separated product=quantity entries. Leave blank for no releases."},
			{id: fieldProductionAllocations, label: "Capacity", value: draft.productionAllocations, placeholder: "fabrication:pump=2, assembly:pump=2", help: "Comma-separated workstation:product=capacity entries. Leave blank for no allocations."},
			{id: fieldCommentary, label: "Commentary", value: draft.commentary, placeholder: "Explain your reasoning for this round.", help: "Required public commentary shown after the round is revealed."},
		}
	case domain.RoleSalesManager:
		return []fieldSpec{
			{id: fieldSalesOffers, label: "Offers", value: draft.salesOffers, placeholder: "pump=14, valve=9", help: "Comma-separated product=unit_price entries. Leave blank for no offers."},
			{id: fieldCommentary, label: "Commentary", value: draft.commentary, placeholder: "Explain your reasoning for this round.", help: "Required public commentary shown after the round is revealed."},
		}
	case domain.RoleFinanceController:
		return []fieldSpec{
			{id: fieldFinanceProcurementBudget, label: "Procurement budget", value: draft.financeProcurement, placeholder: "0", help: "Whole number budget for the next round."},
			{id: fieldFinanceProductionBudget, label: "Production budget", value: draft.financeProduction, placeholder: "0", help: "Whole number budget for the next round."},
			{id: fieldFinanceRevenueTarget, label: "Revenue target", value: draft.financeRevenue, placeholder: "0", help: "Whole number revenue target for the next round."},
			{id: fieldFinanceCashFloor, label: "Cash floor", value: draft.financeCashFloor, placeholder: "0", help: "Whole number cash floor target for the next round."},
			{id: fieldFinanceDebtCeiling, label: "Debt ceiling", value: draft.financeDebtCeiling, placeholder: "0", help: "Whole number debt ceiling target for the next round."},
			{id: fieldCommentary, label: "Commentary", value: draft.commentary, placeholder: "Explain your reasoning for this round.", help: "Required public commentary shown after the round is revealed."},
		}
	default:
		return []fieldSpec{{id: fieldCommentary, label: "Commentary", value: draft.commentary, placeholder: "Explain your reasoning for this round.", help: "Required public commentary shown after the round is revealed."}}
	}
}

func (m Model) currentDraft() actionDraft {
	roleID := m.selectedAssignment().RoleID
	draft, ok := m.drafts[roleID]
	if ok {
		return draft
	}

	draft = actionDraft{}
	if m.selectedAssignment().RoleID == domain.RoleFinanceController {
		targets := m.selectedRoleView().ActiveTargets
		draft.financeProcurement = strconv.Itoa(int(targets.ProcurementBudget))
		draft.financeProduction = strconv.Itoa(int(targets.ProductionSpendBudget))
		draft.financeRevenue = strconv.Itoa(int(targets.RevenueTarget))
		draft.financeCashFloor = strconv.Itoa(int(targets.CashFloorTarget))
		draft.financeDebtCeiling = strconv.Itoa(int(targets.DebtCeilingTarget))
	}
	return draft
}

func (m *Model) setDraftField(draft *actionDraft, field draftField, value string) {
	switch field {
	case fieldProcurementOrders:
		draft.procurementOrders = strings.TrimSpace(value)
	case fieldProductionReleases:
		draft.productionReleases = strings.TrimSpace(value)
	case fieldProductionAllocations:
		draft.productionAllocations = strings.TrimSpace(value)
	case fieldSalesOffers:
		draft.salesOffers = strings.TrimSpace(value)
	case fieldFinanceProcurementBudget:
		draft.financeProcurement = strings.TrimSpace(value)
	case fieldFinanceProductionBudget:
		draft.financeProduction = strings.TrimSpace(value)
	case fieldFinanceRevenueTarget:
		draft.financeRevenue = strings.TrimSpace(value)
	case fieldFinanceCashFloor:
		draft.financeCashFloor = strings.TrimSpace(value)
	case fieldFinanceDebtCeiling:
		draft.financeDebtCeiling = strings.TrimSpace(value)
	case fieldCommentary:
		draft.commentary = strings.TrimSpace(value)
	}
}

func (m Model) buildSubmissionDraft(draft actionDraft) (domain.ActionSubmission, error) {
	assignment := m.selectedAssignment()
	action, err := m.parseRoleAction(assignment.RoleID, draft)
	if err != nil {
		return domain.ActionSubmission{}, err
	}
	commentary := strings.TrimSpace(draft.commentary)
	if commentary == "" {
		return domain.ActionSubmission{}, fmt.Errorf("commentary is required before review or submit")
	}

	return domain.ActionSubmission{
		Action: action,
		Commentary: domain.CommentaryRecord{
			ActorID:    domain.ActorID(assignment.PlayerID),
			Visibility: domain.CommentaryPublic,
			Body:       commentary,
		},
	}, nil
}

func (m Model) parseRoleAction(roleID domain.RoleID, draft actionDraft) (domain.RoleAction, error) {
	switch roleID {
	case domain.RoleProcurementManager:
		orders, err := m.parseOrders(draft.procurementOrders)
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{Procurement: &domain.ProcurementAction{Orders: orders}}, nil
	case domain.RoleProductionManager:
		releases, err := m.parseReleases(draft.productionReleases)
		if err != nil {
			return domain.RoleAction{}, err
		}
		allocations, err := m.parseAllocations(draft.productionAllocations)
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{Production: &domain.ProductionAction{
			Releases:           releases,
			CapacityAllocation: allocations,
		}}, nil
	case domain.RoleSalesManager:
		offers, err := m.parseOffers(draft.salesOffers)
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{Sales: &domain.SalesAction{ProductOffers: offers}}, nil
	case domain.RoleFinanceController:
		targets, err := m.parseFinanceTargets(draft)
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{Finance: &domain.FinanceAction{NextRoundTargets: targets}}, nil
	default:
		return domain.RoleAction{}, fmt.Errorf("unsupported role %q", roleID)
	}
}

func (m Model) parseOrders(raw string) ([]domain.PurchaseOrderIntent, error) {
	pairs, err := parseKeyValueList(raw)
	if err != nil {
		return nil, err
	}
	orders := make([]domain.PurchaseOrderIntent, 0, len(pairs))
	for _, pair := range pairs {
		partID := domain.PartID(pair.key)
		supplierID, ok := m.supplierForPart(partID)
		if !ok {
			return nil, fmt.Errorf("unknown part %q", pair.key)
		}
		quantity := domain.Units(pair.value)
		orders = append(orders, domain.PurchaseOrderIntent{
			PartID:     partID,
			SupplierID: supplierID,
			Quantity:   quantity,
		})
	}
	return orders, nil
}

func (m Model) parseReleases(raw string) ([]domain.ProductionRelease, error) {
	pairs, err := parseKeyValueList(raw)
	if err != nil {
		return nil, err
	}
	releases := make([]domain.ProductionRelease, 0, len(pairs))
	for _, pair := range pairs {
		if !m.validProduct(domain.ProductID(pair.key)) {
			return nil, fmt.Errorf("unknown product %q", pair.key)
		}
		releases = append(releases, domain.ProductionRelease{
			ProductID: domain.ProductID(pair.key),
			Quantity:  domain.Units(pair.value),
		})
	}
	return releases, nil
}

func (m Model) parseAllocations(raw string) ([]domain.CapacityAllocation, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	entries := strings.Split(raw, ",")
	allocations := make([]domain.CapacityAllocation, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		workstationAndProduct, capacityText, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("capacity entry %q must use workstation:product=capacity", entry)
		}
		workstationText, productText, ok := strings.Cut(strings.TrimSpace(workstationAndProduct), ":")
		if !ok {
			return nil, fmt.Errorf("capacity entry %q must include workstation:product", entry)
		}
		if !m.validWorkstation(domain.WorkstationID(strings.TrimSpace(workstationText))) {
			return nil, fmt.Errorf("unknown workstation %q", strings.TrimSpace(workstationText))
		}
		productID := domain.ProductID(strings.TrimSpace(productText))
		if !m.validProduct(productID) {
			return nil, fmt.Errorf("unknown product %q", productText)
		}
		capacity, err := parseNonNegativeInt(capacityText)
		if err != nil {
			return nil, fmt.Errorf("capacity entry %q: %w", entry, err)
		}
		allocations = append(allocations, domain.CapacityAllocation{
			WorkstationID: domain.WorkstationID(strings.TrimSpace(workstationText)),
			ProductID:     productID,
			Capacity:      domain.CapacityUnits(capacity),
		})
	}
	return allocations, nil
}

func (m Model) parseOffers(raw string) ([]domain.ProductOffer, error) {
	pairs, err := parseKeyValueList(raw)
	if err != nil {
		return nil, err
	}
	offers := make([]domain.ProductOffer, 0, len(pairs))
	for _, pair := range pairs {
		productID := domain.ProductID(pair.key)
		if !m.validProduct(productID) {
			return nil, fmt.Errorf("unknown product %q", pair.key)
		}
		offers = append(offers, domain.ProductOffer{
			ProductID: productID,
			UnitPrice: domain.Money(pair.value),
		})
	}
	return offers, nil
}

func (m Model) parseFinanceTargets(draft actionDraft) (domain.BudgetTargets, error) {
	procurement, err := parseNonNegativeInt(draft.financeProcurement)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("procurement budget: %w", err)
	}
	production, err := parseNonNegativeInt(draft.financeProduction)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("production budget: %w", err)
	}
	revenue, err := parseNonNegativeInt(draft.financeRevenue)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("revenue target: %w", err)
	}
	cashFloor, err := parseNonNegativeInt(draft.financeCashFloor)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("cash floor: %w", err)
	}
	debtCeiling, err := parseNonNegativeInt(draft.financeDebtCeiling)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("debt ceiling: %w", err)
	}

	return domain.BudgetTargets{
		EffectiveRound:        m.selectedRoleView().ActiveTargets.EffectiveRound + 1,
		ProcurementBudget:     domain.Money(procurement),
		ProductionSpendBudget: domain.Money(production),
		RevenueTarget:         domain.Money(revenue),
		CashFloorTarget:       domain.Money(cashFloor),
		DebtCeilingTarget:     domain.Money(debtCeiling),
	}, nil
}

func (m Model) effectiveRoundFlow() domain.RoundFlowState {
	flow := m.state.RoundFlow.Clone()
	submittedSet := make(map[domain.RoleID]bool, len(flow.SubmittedRoles))
	for _, roleID := range flow.SubmittedRoles {
		submittedSet[roleID] = true
	}
	for roleID, draft := range m.drafts {
		if draft.stage == draftStageSubmitted {
			submittedSet[roleID] = true
		}
	}

	var submitted []domain.RoleID
	var waiting []domain.RoleID
	for _, assignment := range m.state.Roles {
		if submittedSet[assignment.RoleID] {
			submitted = append(submitted, assignment.RoleID)
			continue
		}
		waiting = append(waiting, assignment.RoleID)
	}
	flow.SubmittedRoles = submitted
	flow.WaitingOnRoles = waiting
	if flow.Phase == "" {
		flow.Phase = domain.RoundPhaseCollecting
	}
	if flow.Phase == domain.RoundPhaseCollecting && len(waiting) == 0 && len(submitted) > 0 {
		flow.Phase = domain.RoundPhaseResolving
	}
	return flow
}

func (m Model) previousAcceptedAction(roleID domain.RoleID) *domain.ActionSubmission {
	for i := len(m.state.History.RecentRounds) - 1; i >= 0; i-- {
		round := m.state.History.RecentRounds[i]
		for _, action := range round.Actions {
			if action.RoleID != roleID {
				continue
			}
			cloned := action.Clone()
			return &cloned
		}
	}
	return nil
}

func summarizeSubmission(submission domain.ActionSubmission) []string {
	lines := summarizeAction(submission.Action)
	return append(lines, "Commentary: "+submission.Commentary.Body)
}

type kvPair struct {
	key   string
	value int
}

func parseKeyValueList(raw string) ([]kvPair, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	entries := strings.Split(raw, ",")
	pairs := make([]kvPair, 0, len(entries))
	for _, entry := range entries {
		keyText, valueText, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("entry %q must use name=value", strings.TrimSpace(entry))
		}
		key := strings.TrimSpace(keyText)
		if key == "" {
			return nil, fmt.Errorf("entry %q needs a name before =", strings.TrimSpace(entry))
		}
		value, err := parseNonNegativeInt(valueText)
		if err != nil {
			return nil, fmt.Errorf("entry %q: %w", strings.TrimSpace(entry), err)
		}
		pairs = append(pairs, kvPair{key: key, value: value})
	}
	return pairs, nil
}

func parseNonNegativeInt(raw string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("enter a whole number that is 0 or greater")
	}
	return value, nil
}

func (m Model) validProduct(productID domain.ProductID) bool {
	for _, product := range m.scenario.ProductionModel.Products {
		if product.ID == productID {
			return true
		}
	}
	return false
}

func (m Model) validWorkstation(workstationID domain.WorkstationID) bool {
	for _, workstation := range m.scenario.ProductionModel.Workstations {
		if workstation.ID == workstationID {
			return true
		}
	}
	return false
}

func (m Model) supplierForPart(partID domain.PartID) (domain.SupplierID, bool) {
	for _, part := range m.scenario.ProductionModel.Parts {
		if part.ID == partID {
			return part.SupplierID, true
		}
	}
	return "", false
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
