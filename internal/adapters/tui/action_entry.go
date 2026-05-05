package tui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/actionschema"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/projection"
)

type draftStage int

const (
	draftStageEditing draftStage = iota
	draftStageReview
	draftStageSubmitted
)

type actionDraft struct {
	stage      draftStage
	form       actionFormModel
	status     string
	errorText  string
	submission *domain.ActionSubmission
}

func (m *Model) handleActionEntryKey(msg tea.KeyMsg) bool {
	if m.workspace != workspaceActionEntry || m.focusedPane != paneHistory {
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

	if draft.form.Editing {
		return m.handleEditingKey(msg, assignment.RoleID)
	}

	switch msg.String() {
	case "up", "k":
		draft.form.MoveUp()
		draft.status = fmt.Sprintf("Focused %s", m.currentFocusLabel(draft))
		m.drafts[assignment.RoleID] = draft
		return true
	case "down", "j":
		draft.form.MoveDown()
		draft.status = fmt.Sprintf("Focused %s", m.currentFocusLabel(draft))
		m.drafts[assignment.RoleID] = draft
		return true
	case "left", "h":
		draft.form.MoveLeft()
		draft.status = fmt.Sprintf("Focused %s", m.currentFocusLabel(draft))
		m.drafts[assignment.RoleID] = draft
		return true
	case "right", "l":
		draft.form.MoveRight()
		draft.status = fmt.Sprintf("Focused %s", m.currentFocusLabel(draft))
		m.drafts[assignment.RoleID] = draft
		return true
	case "a":
		field := draft.form.currentField()
		if field != nil && field.Collection != nil {
			draft.form.AddRow()
			draft.status = fmt.Sprintf("Added %s row", strings.ToLower(field.Label))
			draft.errorText = ""
			m.drafts[assignment.RoleID] = draft
			return true
		}
	case "x":
		field := draft.form.currentField()
		if field != nil && field.Collection != nil && draft.form.RemoveRow() {
			draft.status = fmt.Sprintf("Removed %s row", strings.ToLower(field.Label))
			draft.errorText = ""
			m.drafts[assignment.RoleID] = draft
			return true
		}
	case "enter", "e":
		if draft.form.BeginEdit() {
			draft.errorText = ""
			draft.status = fmt.Sprintf("Editing %s", m.currentFocusLabel(draft))
			m.drafts[assignment.RoleID] = draft
			return true
		}
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
			submission.MatchID = m.state.MatchID
			submission.Round = m.state.CurrentRound
			submission.RoleID = assignment.RoleID
			if m.submit != nil {
				if err := m.submit(submission); err != nil {
					draft.errorText = err.Error()
					draft.status = "Submit blocked until the live match accepts the action"
					m.drafts[assignment.RoleID] = draft
					return true
				}
			}
			draft.stage = draftStageSubmitted
			draft.submission = &submission
			draft.status = "Submission locked for this round"
			draft.errorText = ""
			m.drafts[assignment.RoleID] = draft
			m.advanceAfterSubmission(assignment.RoleID)
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
	if draft.form.editingChoice() {
		switch msg.String() {
		case "up", "k", "left", "h":
			if draft.form.CycleEditingChoice(-1) {
				m.drafts[roleID] = draft
			}
			return true
		case "down", "j", "right", "l":
			if draft.form.CycleEditingChoice(1) {
				m.drafts[roleID] = draft
			}
			return true
		}
	}
	switch msg.Type {
	case tea.KeyEsc:
		draft.form.CancelEdit()
		draft.status = "Edit cancelled"
		m.drafts[roleID] = draft
		return true
	case tea.KeyEnter:
		if draft.form.CommitEdit() {
			draft.status = fmt.Sprintf("Saved %s", m.currentFocusLabel(draft))
			draft.errorText = ""
			m.drafts[roleID] = draft
		}
		return true
	case tea.KeyBackspace:
		if draft.form.Backspace() {
			m.drafts[roleID] = draft
		}
		return true
	case tea.KeySpace:
		if draft.form.TypeRunes(" ") {
			m.drafts[roleID] = draft
		}
		return true
	case tea.KeyRunes:
		var builder strings.Builder
		for _, r := range msg.Runes {
			if unicode.IsPrint(r) {
				builder.WriteRune(r)
			}
		}
		if draft.form.TypeRunes(builder.String()) {
			m.drafts[roleID] = draft
		}
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
		lines = append(lines, "", "Previous accepted commentary", wrapLine(previous.Commentary.Body, paneTextWidth(width)))
	}

	if draft.stage == draftStageReview {
		lines = append(lines, "", "Review draft", "Press s to submit and lock, or b to return to editing.")
		if draft.submission != nil {
			for _, line := range summarizeSubmission(*draft.submission) {
				lines = append(lines, wrapLine("- "+line, paneTextWidth(width)))
			}
		}
		lines = append(lines, m.renderReviewComparisons(width, assignment.RoleID, draft)...)
		return append(lines, m.renderDraftStatus(width, draft)...)
	}

	if draft.stage == draftStageSubmitted {
		if m.hideSubmittedHumanDetails(assignment.RoleID) {
			lines = append(lines,
				"",
				"Submission locked for this round.",
				waitingOnSummary(m.effectiveRoundFlow().WaitingOnRoles),
				"Locked human entries stay private in multi-human games.",
			)
			return append(lines, m.renderDraftStatus(width, draft)...)
		}

		lines = append(lines,
			"",
			"Submission locked for this round.",
			waitingOnSummary(m.effectiveRoundFlow().WaitingOnRoles),
			"Current-turn details stay out of the shared round feed until reveal.",
		)
		if draft.submission != nil {
			lines = append(lines, "")
			for _, line := range summarizeSubmission(*draft.submission) {
				lines = append(lines, wrapLine("- "+line, paneTextWidth(width)))
			}
		}
		return append(lines, m.renderDraftStatus(width, draft)...)
	}

	lines = append(lines,
		"",
		"Editing flow",
		"Use up/down to move between fields or table rows, left/right to move across table columns, enter to edit and save, a to add rows, x to remove rows, esc to cancel, and r to review.",
		"Choice editors keep focus while editing so arrows can change the selected option before enter commits it.",
	)
	for index, field := range draft.form.Schema.Fields {
		fieldCursor := " "
		if index == draft.form.FieldIndex {
			fieldCursor = ">"
		}
		lines = append(lines, fmt.Sprintf("%s %s", fieldCursor, field.Label))
		lines = append(lines, wrapLine("  "+field.Help, paneTextWidth(width)))
		if field.Collection == nil {
			lines = append(lines, wrapLine("  "+m.renderScalarFieldValue(draft, field), paneTextWidth(width)))
			continue
		}
		lines = append(lines, wrapLine("  "+draft.form.currentCollectionSummary(field), paneTextWidth(width)))
		active := index == draft.form.FieldIndex
		lines = append(lines, draft.form.renderedCollectionTable(field, paneTextWidth(width), active))
		lines = append(lines, "")
	}

	return append(lines, m.renderDraftStatus(width, draft)...)
}

func (m Model) renderScalarFieldValue(draft actionDraft, field actionschema.FieldSpec) string {
	value := draft.form.displayScalar(field)
	if strings.TrimSpace(value) == "" {
		value = field.Placeholder
	}
	return value
}

func (m Model) renderDraftStatus(width int, draft actionDraft) []string {
	var lines []string
	if strings.TrimSpace(draft.status) != "" {
		lines = append(lines, "", wrapLine("Status: "+draft.status, paneTextWidth(width)))
	}
	if strings.TrimSpace(draft.errorText) != "" {
		lines = append(lines, wrapLine("Validation: "+draft.errorText, paneTextWidth(width)))
	}
	return lines
}

func (m Model) currentDraft() actionDraft {
	return m.currentDraftForRole(m.selectedAssignment().RoleID)
}

func (m *Model) advanceAfterSubmission(roleID domain.RoleID) {
	if nextIndex, ok := m.nextPendingHumanRole(roleID); ok {
		m.selectedRole = nextIndex
		m.workspace = workspaceActionEntry
		m.status = fmt.Sprintf("%s submitted. Moved to %s for entry.", displayRoleName(roleID), m.roleTitle())
		return
	}

	m.workspace = workspaceRoundFeed
	m.status = fmt.Sprintf("%s submitted. All human entries are locked; switched to the round feed.", displayRoleName(roleID))
}

func (m Model) currentFocusLabel(draft actionDraft) string {
	field := draft.form.currentField()
	if field == nil {
		return "field"
	}
	if field.Collection == nil {
		return field.Label
	}
	column := draft.form.currentColumn()
	if column == nil {
		return field.Label
	}
	return fmt.Sprintf("%s row %d %s", field.Label, draft.form.RowIndex+1, column.Label)
}

func (m Model) buildSubmissionDraft(draft actionDraft) (domain.ActionSubmission, error) {
	assignment := m.selectedAssignment()
	action, err := m.parseRoleAction(assignment.RoleID, draft.form)
	if err != nil {
		return domain.ActionSubmission{}, err
	}
	if err := actionschema.FirstError(actionschema.ValidateRoleAction(draft.form.Schema, action, m.selectedRoleView())); err != nil {
		return domain.ActionSubmission{}, err
	}
	commentary := strings.TrimSpace(draft.form.Values["commentary"].Scalar)
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

func (m Model) parseRoleAction(roleID domain.RoleID, form actionFormModel) (domain.RoleAction, error) {
	switch roleID {
	case domain.RoleProcurementManager:
		orders, err := parseOrderRows(form.Values["orders"].Rows)
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{Procurement: &domain.ProcurementAction{Orders: orders}}, nil
	case domain.RoleProductionManager:
		releases, err := parseReleaseRows(form.Values["releases"].Rows)
		if err != nil {
			return domain.RoleAction{}, err
		}
		allocations, err := parseAllocationRows(form.Values["capacity_allocation"].Rows)
		if err != nil {
			return domain.RoleAction{}, err
		}
		overtime, err := parseOvertimeRows(form.Values["overtime"].Rows)
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{Production: &domain.ProductionAction{
			Releases:           releases,
			CapacityAllocation: allocations,
			Overtime:           overtime,
		}}, nil
	case domain.RoleSalesManager:
		offers, err := parseOfferRows(form.Values["product_offers"].Rows)
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{Sales: &domain.SalesAction{ProductOffers: offers}}, nil
	case domain.RoleFinanceController:
		targets, err := parseFinanceTargets(form, m.selectedRoleView())
		if err != nil {
			return domain.RoleAction{}, err
		}
		return domain.RoleAction{Finance: &domain.FinanceAction{NextRoundTargets: targets}}, nil
	default:
		return domain.RoleAction{}, fmt.Errorf("unsupported role %q", roleID)
	}
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

func (m Model) nextPendingHumanRole(submittedRoleID domain.RoleID) (int, bool) {
	if len(m.state.Roles) == 0 {
		return 0, false
	}

	start := clampRoleIndex(m.selectedRole, len(m.state.Roles))
	for step := 1; step <= len(m.state.Roles); step++ {
		index := (start + step) % len(m.state.Roles)
		assignment := m.state.Roles[index]
		if !assignment.IsHuman || assignment.RoleID == submittedRoleID {
			continue
		}
		if m.currentDraftForRole(assignment.RoleID).stage == draftStageSubmitted {
			continue
		}
		return index, true
	}
	return 0, false
}

func (m Model) currentDraftForRole(roleID domain.RoleID) actionDraft {
	draft, ok := m.drafts[roleID]
	if ok {
		return draft
	}

	schema := actionschema.BuildFromCatalog(m.scenario, roleID, projection.BuildRoundView(m.state, roleID))
	draft = actionDraft{
		form: newActionFormModel(schema),
	}
	if previous := m.previousAcceptedAction(roleID); previous != nil {
		seedDraftFromSubmission(&draft.form, *previous)
		return draft
	}
	if roleID == domain.RoleFinanceController {
		view := projection.BuildRoundView(m.state, roleID)
		targets := view.ActiveTargets
		draft.form.Values["procurement_budget"] = formFieldValue{Scalar: strconv.Itoa(int(targets.ProcurementBudget))}
		draft.form.Values["production_spend_budget"] = formFieldValue{Scalar: strconv.Itoa(int(targets.ProductionSpendBudget))}
		draft.form.Values["revenue_target"] = formFieldValue{Scalar: strconv.Itoa(int(targets.RevenueTarget))}
		draft.form.Values["cash_floor_target"] = formFieldValue{Scalar: strconv.Itoa(int(targets.CashFloorTarget))}
		draft.form.Values["debt_ceiling_target"] = formFieldValue{Scalar: strconv.Itoa(int(targets.DebtCeilingTarget))}
	}
	return draft
}

func (m Model) hideSubmittedHumanDetails(roleID domain.RoleID) bool {
	return m.selectedAssignment().IsHuman && humanRoleCount(m.state.Roles) > 1 && m.currentDraftForRole(roleID).stage == draftStageSubmitted
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

func seedDraftFromSubmission(form *actionFormModel, submission domain.ActionSubmission) {
	if form == nil {
		return
	}

	if action := submission.Action.Procurement; action != nil {
		rows := make([]map[string]string, 0, len(action.Orders))
		for _, order := range action.Orders {
			rows = append(rows, map[string]string{
				"part_id":     string(order.PartID),
				"supplier_id": string(order.SupplierID),
				"quantity":    strconv.Itoa(int(order.Quantity)),
			})
		}
		form.Values["orders"] = formFieldValue{Rows: rows}
		form.syncTable("orders")
	}

	if action := submission.Action.Production; action != nil {
		releases := make([]map[string]string, 0, len(action.Releases))
		for _, release := range action.Releases {
			releases = append(releases, map[string]string{
				"product_id": string(release.ProductID),
				"quantity":   strconv.Itoa(int(release.Quantity)),
			})
		}
		form.Values["releases"] = formFieldValue{Rows: releases}
		form.syncTable("releases")

		allocations := make([]map[string]string, 0, len(action.CapacityAllocation))
		for _, allocation := range action.CapacityAllocation {
			allocations = append(allocations, map[string]string{
				"workstation_id": string(allocation.WorkstationID),
				"product_id":     string(allocation.ProductID),
				"capacity":       strconv.Itoa(int(allocation.Capacity)),
			})
		}
		form.Values["capacity_allocation"] = formFieldValue{Rows: allocations}
		form.syncTable("capacity_allocation")

		overtime := make([]map[string]string, 0, len(action.Overtime))
		for _, allocation := range action.Overtime {
			overtime = append(overtime, map[string]string{
				"workstation_id": string(allocation.WorkstationID),
				"capacity":       strconv.Itoa(int(allocation.Capacity)),
			})
		}
		form.Values["overtime"] = formFieldValue{Rows: overtime}
		form.syncTable("overtime")
	}

	if action := submission.Action.Sales; action != nil {
		rows := make([]map[string]string, 0, len(action.ProductOffers))
		for _, offer := range action.ProductOffers {
			rows = append(rows, map[string]string{
				"product_id": string(offer.ProductID),
				"unit_price": strconv.Itoa(int(offer.UnitPrice)),
			})
		}
		form.Values["product_offers"] = formFieldValue{Rows: rows}
		form.syncTable("product_offers")
	}

	if action := submission.Action.Finance; action != nil {
		targets := action.NextRoundTargets
		form.Values["procurement_budget"] = formFieldValue{Scalar: strconv.Itoa(int(targets.ProcurementBudget))}
		form.Values["production_spend_budget"] = formFieldValue{Scalar: strconv.Itoa(int(targets.ProductionSpendBudget))}
		form.Values["revenue_target"] = formFieldValue{Scalar: strconv.Itoa(int(targets.RevenueTarget))}
		form.Values["cash_floor_target"] = formFieldValue{Scalar: strconv.Itoa(int(targets.CashFloorTarget))}
		form.Values["debt_ceiling_target"] = formFieldValue{Scalar: strconv.Itoa(int(targets.DebtCeilingTarget))}
	}

	form.Values["commentary"] = formFieldValue{Scalar: submission.Commentary.Body}
}

func (m Model) renderReviewComparisons(width int, roleID domain.RoleID, draft actionDraft) []string {
	lines := []string{"", "Decision support"}

	lines = append(lines, "Compare with past orders")
	if previous := m.previousAcceptedAction(roleID); previous != nil {
		for _, line := range summarizeAction(previous.Action) {
			lines = append(lines, wrapLine("- "+line, paneTextWidth(width)))
		}
	} else {
		lines = append(lines, "No prior accepted action is available for this role.")
	}

	lines = append(lines, "", "Past performance")
	for _, line := range m.reviewPerformanceHighlights() {
		lines = append(lines, wrapLine("- "+line, paneTextWidth(width)))
	}

	lines = append(lines, "", "Expected guidance")
	for _, line := range m.expectedGuidance(roleID, draft) {
		lines = append(lines, wrapLine("- "+line, paneTextWidth(width)))
	}

	return lines
}

func (m Model) reviewPerformanceHighlights() []string {
	report := m.selectedRoleReport()
	var lines []string
	for _, section := range report.Department.Sections {
		for _, metric := range section.Metrics {
			lines = append(lines, fmt.Sprintf("%s: %d %s", metric.Label, metric.Metric.Value, metric.Metric.DisplayUnit))
			if len(lines) == 3 {
				return lines
			}
		}
		for _, warning := range section.Warnings {
			lines = append(lines, warning.Headline)
			if len(lines) == 3 {
				return lines
			}
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "No role-specific performance highlights are available yet.")
	}
	return lines
}

func (m Model) expectedGuidance(roleID domain.RoleID, draft actionDraft) []string {
	switch roleID {
	case domain.RoleProcurementManager:
		return m.expectedProcurementGuidance(draft)
	case domain.RoleProductionManager:
		return m.expectedProductionGuidance(draft)
	case domain.RoleSalesManager:
		return m.expectedSalesGuidance(draft)
	case domain.RoleFinanceController:
		return m.expectedFinanceGuidance(draft)
	default:
		return []string{"No expected guidance is configured for this role."}
	}
}

func (m Model) expectedProcurementGuidance(_ actionDraft) []string {
	backlogDemand := make(map[domain.ProductID]int)
	for _, entry := range m.state.Plant.Backlog {
		backlogDemand[entry.ProductID] += int(entry.Quantity)
	}

	finishedByProduct := make(map[domain.ProductID]int)
	for _, item := range m.state.Plant.FinishedInventory {
		finishedByProduct[item.ProductID] += int(item.OnHandQty)
	}

	inTransitByPart := make(map[domain.PartID]int)
	for _, item := range m.state.Plant.InTransitSupply {
		inTransitByPart[item.PartID] += int(item.Quantity)
	}

	partsOnHand := make(map[domain.PartID]int)
	for _, item := range m.state.Plant.PartsInventory {
		partsOnHand[item.PartID] += int(item.OnHandQty)
	}

	requiredByPart := make(map[domain.PartID]int)
	for _, product := range m.scenario.Products() {
		gap := backlogDemand[product.ID] - finishedByProduct[product.ID]
		if gap <= 0 {
			continue
		}
		for _, line := range product.BOM {
			requiredByPart[line.PartID] += gap * int(line.Quantity)
		}
	}

	var lines []string
	for _, part := range m.scenario.Parts() {
		expected := requiredByPart[part.ID] - partsOnHand[part.ID] - inTransitByPart[part.ID]
		if expected <= 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("Expected %s order is roughly %d based on backlog demand, BOM coverage, on-hand parts, and inbound supply.", part.ID, expected))
	}
	if len(lines) == 0 {
		lines = append(lines, "Current backlog, BOM, inventory, and inbound supply do not point to an urgent additional part order.")
	}
	return lines
}

func (m Model) expectedProductionGuidance(_ actionDraft) []string {
	backlogByProduct := make(map[domain.ProductID]int)
	for _, entry := range m.state.Plant.Backlog {
		backlogByProduct[entry.ProductID] += int(entry.Quantity)
	}
	finishedByProduct := make(map[domain.ProductID]int)
	for _, item := range m.state.Plant.FinishedInventory {
		finishedByProduct[item.ProductID] += int(item.OnHandQty)
	}
	wipByProduct := make(map[domain.ProductID]int)
	for _, item := range m.state.Plant.WIPInventory {
		wipByProduct[item.ProductID] += int(item.Quantity)
	}

	var lines []string
	for _, product := range m.scenario.Products() {
		gap := backlogByProduct[product.ID] - finishedByProduct[product.ID] - wipByProduct[product.ID]
		if gap <= 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("Expected %s release is roughly %d based on backlog pressure minus finished goods and WIP already in the line.", product.ID, gap))
	}
	if len(lines) == 0 {
		lines = append(lines, "Backlog pressure does not currently justify more release beyond existing finished goods and WIP.")
	}
	return lines
}

func (m Model) expectedSalesGuidance(_ actionDraft) []string {
	backlogByProduct := make(map[domain.ProductID]int)
	for _, entry := range m.state.Plant.Backlog {
		backlogByProduct[entry.ProductID] += int(entry.Quantity)
	}
	finishedByProduct := make(map[domain.ProductID]int)
	for _, item := range m.state.Plant.FinishedInventory {
		finishedByProduct[item.ProductID] += int(item.OnHandQty)
	}

	var lines []string
	for _, product := range m.scenario.Products() {
		gap := backlogByProduct[product.ID] - finishedByProduct[product.ID]
		if gap > 0 {
			lines = append(lines, fmt.Sprintf("%s demand already exceeds finished-goods coverage by %d unit(s); avoid aggressive demand expansion until operations catch up.", product.ID, gap))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s has finished-goods coverage against visible backlog, so measured demand expansion is supportable if service remains stable.", product.ID))
	}
	if len(lines) == 0 {
		lines = append(lines, "No sales guidance is available because no visible product demand is loaded.")
	}
	return lines
}

func (m Model) expectedFinanceGuidance(_ actionDraft) []string {
	nextRoundCashPressure := sumCommitmentAmount(m.state.Plant.Payables) - sumCommitmentAmount(m.state.Plant.Receivables)
	lines := []string{
		fmt.Sprintf("Current targets already anchor procurement at %d and production at %d for next round.", m.state.ActiveTargets.ProcurementBudget, m.state.ActiveTargets.ProductionSpendBudget),
	}
	if nextRoundCashPressure > 0 {
		lines = append(lines, fmt.Sprintf("Visible commitments are net cash-negative by %d, so budget tightening deserves consideration.", nextRoundCashPressure))
	} else {
		lines = append(lines, "Visible commitments are not net cash-negative, so holding current targets is defensible if throughput needs support.")
	}
	if m.state.Plant.Debt >= m.state.Plant.DebtCeiling {
		lines = append(lines, "Debt is already at or above the configured ceiling, so any growth posture needs explicit liquidity justification.")
	}
	return lines
}

func sumCommitmentAmount(items []domain.CashCommitment) int {
	total := 0
	for _, item := range items {
		total += int(item.Amount)
	}
	return total
}

func parseOrderRows(rows []map[string]string) ([]domain.PurchaseOrderIntent, error) {
	orders := make([]domain.PurchaseOrderIntent, 0, len(rows))
	for index, row := range rows {
		partID := domain.PartID(strings.TrimSpace(row["part_id"]))
		if partID == "" {
			return nil, fmt.Errorf("orders row %d: part is required", index+1)
		}
		supplierID := domain.SupplierID(strings.TrimSpace(row["supplier_id"]))
		if supplierID == "" {
			return nil, fmt.Errorf("orders row %d: supplier is required", index+1)
		}
		quantity, err := parseNonNegativeInt(row["quantity"])
		if err != nil {
			return nil, fmt.Errorf("orders row %d: %w", index+1, err)
		}
		orders = append(orders, domain.PurchaseOrderIntent{
			PartID:     partID,
			SupplierID: supplierID,
			Quantity:   domain.Units(quantity),
		})
	}
	return orders, nil
}

func parseReleaseRows(rows []map[string]string) ([]domain.ProductionRelease, error) {
	releases := make([]domain.ProductionRelease, 0, len(rows))
	for index, row := range rows {
		productID := domain.ProductID(strings.TrimSpace(row["product_id"]))
		if productID == "" {
			return nil, fmt.Errorf("releases row %d: product is required", index+1)
		}
		quantity, err := parseNonNegativeInt(row["quantity"])
		if err != nil {
			return nil, fmt.Errorf("releases row %d: %w", index+1, err)
		}
		releases = append(releases, domain.ProductionRelease{ProductID: productID, Quantity: domain.Units(quantity)})
	}
	return releases, nil
}

func parseAllocationRows(rows []map[string]string) ([]domain.CapacityAllocation, error) {
	allocations := make([]domain.CapacityAllocation, 0, len(rows))
	for index, row := range rows {
		workstationID := domain.WorkstationID(strings.TrimSpace(row["workstation_id"]))
		if workstationID == "" {
			return nil, fmt.Errorf("capacity allocation row %d: workstation is required", index+1)
		}
		productID := domain.ProductID(strings.TrimSpace(row["product_id"]))
		if productID == "" {
			return nil, fmt.Errorf("capacity allocation row %d: product is required", index+1)
		}
		capacity, err := parseNonNegativeInt(row["capacity"])
		if err != nil {
			return nil, fmt.Errorf("capacity allocation row %d: %w", index+1, err)
		}
		allocations = append(allocations, domain.CapacityAllocation{
			WorkstationID: workstationID,
			ProductID:     productID,
			Capacity:      domain.CapacityUnits(capacity),
		})
	}
	return allocations, nil
}

func parseOvertimeRows(rows []map[string]string) ([]domain.OvertimeAllocation, error) {
	overtime := make([]domain.OvertimeAllocation, 0, len(rows))
	for index, row := range rows {
		workstationID := domain.WorkstationID(strings.TrimSpace(row["workstation_id"]))
		if workstationID == "" {
			return nil, fmt.Errorf("overtime row %d: workstation is required", index+1)
		}
		capacity, err := parseNonNegativeInt(row["capacity"])
		if err != nil {
			return nil, fmt.Errorf("overtime row %d: %w", index+1, err)
		}
		overtime = append(overtime, domain.OvertimeAllocation{
			WorkstationID: workstationID,
			Capacity:      domain.CapacityUnits(capacity),
		})
	}
	return overtime, nil
}

func parseOfferRows(rows []map[string]string) ([]domain.ProductOffer, error) {
	offers := make([]domain.ProductOffer, 0, len(rows))
	for index, row := range rows {
		productID := domain.ProductID(strings.TrimSpace(row["product_id"]))
		if productID == "" {
			return nil, fmt.Errorf("offers row %d: product is required", index+1)
		}
		unitPrice, err := parseNonNegativeInt(row["unit_price"])
		if err != nil {
			return nil, fmt.Errorf("offers row %d: %w", index+1, err)
		}
		offers = append(offers, domain.ProductOffer{ProductID: productID, UnitPrice: domain.Money(unitPrice)})
	}
	return offers, nil
}

func parseFinanceTargets(form actionFormModel, view domain.RoundView) (domain.BudgetTargets, error) {
	procurement, err := parseNonNegativeInt(form.Values["procurement_budget"].Scalar)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("procurement budget: %w", err)
	}
	production, err := parseNonNegativeInt(form.Values["production_spend_budget"].Scalar)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("production budget: %w", err)
	}
	revenue, err := parseNonNegativeInt(form.Values["revenue_target"].Scalar)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("revenue target: %w", err)
	}
	cashFloor, err := parseNonNegativeInt(form.Values["cash_floor_target"].Scalar)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("cash floor: %w", err)
	}
	debtCeiling, err := parseNonNegativeInt(form.Values["debt_ceiling_target"].Scalar)
	if err != nil {
		return domain.BudgetTargets{}, fmt.Errorf("debt ceiling: %w", err)
	}

	return domain.BudgetTargets{
		EffectiveRound:        view.ActiveTargets.EffectiveRound + 1,
		ProcurementBudget:     domain.Money(procurement),
		ProductionSpendBudget: domain.Money(production),
		RevenueTarget:         domain.Money(revenue),
		CashFloorTarget:       domain.Money(cashFloor),
		DebtCeilingTarget:     domain.Money(debtCeiling),
	}, nil
}

func parseNonNegativeInt(raw string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0, fmt.Errorf("enter a whole number that is 0 or greater")
	}
	return value, nil
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
		lines := make([]string, 0, len(action.Production.Releases)+len(action.Production.CapacityAllocation)+len(action.Production.Overtime))
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
		for _, overtime := range action.Production.Overtime {
			lines = append(lines, fmt.Sprintf("Use %d overtime capacity at %s", overtime.Capacity, overtime.WorkstationID))
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
