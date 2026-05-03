package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

type debugTreeNode struct {
	id       string
	parentID string
	title    string
	body     string
	children []*debugTreeNode
}

type debugVisibleNode struct {
	node  *debugTreeNode
	depth int
}

type debugWorkspaceRender struct {
	lines             []string
	selectedLineStart int
	selectedLineCount int
}

func (m Model) debugRoleRecords() []ports.AICallRecord {
	if m.debugLog == nil {
		return nil
	}

	roleID := m.selectedAssignment().RoleID
	if roleID == "" {
		return nil
	}

	records := m.debugLog.Records()
	filtered := make([]ports.AICallRecord, 0, len(records))
	for _, record := range records {
		if record.RoleID == roleID {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func (m Model) buildDebugTree() []*debugTreeNode {
	records := m.debugRoleRecords()
	if len(records) == 0 {
		return nil
	}

	roundNodes := make([]*debugTreeNode, 0)
	roundIndex := make(map[domain.RoundNumber]*debugTreeNode)
	attemptIndex := make(map[string]*debugTreeNode)
	attemptCounts := make(map[domain.RoundNumber]int)

	for _, record := range records {
		roundNode := roundIndex[record.Round]
		if roundNode == nil {
			roundNode = &debugTreeNode{
				id:    debugRoundNodeID(record.Round),
				title: fmt.Sprintf("Round %d", record.Round),
			}
			roundIndex[record.Round] = roundNode
			roundNodes = append(roundNodes, roundNode)
		}

		attemptID := debugAttemptNodeID(record.Round, record.Attempt)
		if attemptIndex[attemptID] != nil {
			continue
		}

		attemptNode := buildDebugAttemptNode(record)
		attemptNode.parentID = roundNode.id
		roundNode.children = append(roundNode.children, attemptNode)
		attemptIndex[attemptID] = attemptNode
		attemptCounts[record.Round]++
	}

	slices.SortFunc(roundNodes, func(left, right *debugTreeNode) int {
		switch {
		case left.id < right.id:
			return -1
		case left.id > right.id:
			return 1
		default:
			return 0
		}
	})
	for _, roundNode := range roundNodes {
		round := parseRoundID(roundNode.id)
		roundNode.title = fmt.Sprintf("Round %d (%d tries)", round, attemptCounts[round])
		slices.SortFunc(roundNode.children, func(left, right *debugTreeNode) int {
			switch {
			case left.id < right.id:
				return -1
			case left.id > right.id:
				return 1
			default:
				return 0
			}
		})
	}

	return roundNodes
}

func buildDebugAttemptNode(record ports.AICallRecord) *debugTreeNode {
	_, statusBody := debugAttemptStatus(record)
	requestSummary := fmt.Sprintf(
		"Request Summary (%s/%s, system %d chars, user %d chars)",
		record.Provider,
		record.Model,
		len(record.SystemPrompt),
		len(record.UserPrompt),
	)
	responseSummary := fmt.Sprintf(
		"Response Summary (%s)",
		debugResponseSummary(record),
	)

	attemptNode := &debugTreeNode{
		id:    debugAttemptNodeID(record.Round, record.Attempt),
		title: fmt.Sprintf("Try %d - %s", record.Attempt, debugAttemptOutcome(record)),
	}
	statusNode := &debugTreeNode{
		id:       debugStatusNodeID(record.Round, record.Attempt),
		parentID: attemptNode.id,
		title:    "Try status details",
		body:     statusBody,
	}
	requestSummaryNode := &debugTreeNode{
		id:       debugRequestSummaryNodeID(record.Round, record.Attempt),
		parentID: attemptNode.id,
		title:    requestSummary,
	}
	requestDetailNode := &debugTreeNode{
		id:       debugRequestDetailNodeID(record.Round, record.Attempt),
		parentID: requestSummaryNode.id,
		title:    "Request details",
		body:     debugRequestDetails(record),
	}
	responseSummaryNode := &debugTreeNode{
		id:       debugResponseSummaryNodeID(record.Round, record.Attempt),
		parentID: attemptNode.id,
		title:    responseSummary,
	}
	responseDetailNode := &debugTreeNode{
		id:       debugResponseDetailNodeID(record.Round, record.Attempt),
		parentID: responseSummaryNode.id,
		title:    "Response details",
		body:     debugResponseDetails(record),
	}

	requestSummaryNode.children = []*debugTreeNode{requestDetailNode}
	responseSummaryNode.children = []*debugTreeNode{responseDetailNode}
	attemptNode.children = []*debugTreeNode{statusNode, requestSummaryNode, responseSummaryNode}
	return attemptNode
}

func (m Model) visibleDebugNodes() []debugVisibleNode {
	return m.flattenDebugNodes(m.buildDebugTree(), 0)
}

func (m Model) flattenDebugNodes(nodes []*debugTreeNode, depth int) []debugVisibleNode {
	visible := make([]debugVisibleNode, 0)
	for _, node := range nodes {
		visible = append(visible, debugVisibleNode{node: node, depth: depth})
		if len(node.children) == 0 || !m.isDebugNodeExpanded(node.id, depth) {
			continue
		}
		visible = append(visible, m.flattenDebugNodes(node.children, depth+1)...)
	}
	return visible
}

func (m Model) isDebugNodeExpanded(nodeID string, depth int) bool {
	expanded, ok := m.debugExpanded[nodeID]
	if ok {
		return expanded
	}
	return depth == 0
}

func (m *Model) ensureDebugSelection() {
	if m.workspace != workspaceDebug {
		return
	}

	visible := m.visibleDebugNodes()
	if len(visible) == 0 {
		m.debugSelected = ""
		return
	}

	for _, node := range visible {
		if node.node.id == m.debugSelected {
			return
		}
	}
	m.debugSelected = visible[0].node.id
}

func (m *Model) ensureDebugSelectionVisible() {
	if m.workspace != workspaceDebug {
		return
	}

	width, _ := m.historyPaneDimensions()
	rendered := m.renderDebugWorkspaceContent(width)
	visibleLines := m.historyVisibleLineCount()
	if visibleLines <= 0 || rendered.selectedLineCount == 0 {
		m.historyScroll = 0
		return
	}

	top := m.historyScroll
	bottom := top + visibleLines
	selectedTop := rendered.selectedLineStart
	selectedBottom := rendered.selectedLineStart + rendered.selectedLineCount

	switch {
	case selectedTop < top:
		m.historyScroll = selectedTop
	case selectedBottom > bottom:
		m.historyScroll = selectedBottom - visibleLines
	}

	m.historyScroll = clampScrollOffset(m.historyScroll, len(flattenLines(rendered.lines)), visibleLines)
}

func (m *Model) moveDebugSelection(delta int) {
	visible := m.visibleDebugNodes()
	if len(visible) == 0 {
		m.debugSelected = ""
		return
	}

	index := m.debugSelectionIndex(visible)
	index = max(min(index+delta, len(visible)-1), 0)
	m.debugSelected = visible[index].node.id
	m.ensureDebugSelectionVisible()
}

func (m Model) debugSelectionIndex(visible []debugVisibleNode) int {
	for index, node := range visible {
		if node.node.id == m.debugSelected {
			return index
		}
	}
	return 0
}

func (m *Model) expandDebugSelection() {
	visible := m.visibleDebugNodes()
	if len(visible) == 0 {
		return
	}

	selected := visible[m.debugSelectionIndex(visible)].node
	if len(selected.children) == 0 {
		return
	}
	m.debugExpanded[selected.id] = true
	m.debugSelected = selected.children[0].id
	m.ensureDebugSelectionVisible()
}

func (m *Model) collapseDebugSelection() {
	visible := m.visibleDebugNodes()
	if len(visible) == 0 {
		return
	}

	selected := visible[m.debugSelectionIndex(visible)].node
	if len(selected.children) == 0 {
		if selected.parentID == "" {
			return
		}
		m.debugExpanded[selected.parentID] = false
		m.debugSelected = selected.parentID
		m.ensureDebugSelectionVisible()
		return
	}

	if m.debugExpanded[selected.id] {
		m.debugExpanded[selected.id] = false
		if selected.parentID != "" {
			m.debugSelected = selected.parentID
		}
		m.ensureDebugSelectionVisible()
		return
	}

	if selected.parentID != "" {
		m.debugExpanded[selected.parentID] = false
		m.debugSelected = selected.parentID
		m.ensureDebugSelectionVisible()
	}
}

func (m *Model) pageDebugSelection(delta int) {
	if delta == 0 {
		return
	}
	m.moveDebugSelection(delta)
}

func (m *Model) moveDebugSelectionToEdge(toEnd bool) {
	visible := m.visibleDebugNodes()
	if len(visible) == 0 {
		m.debugSelected = ""
		return
	}
	if toEnd {
		m.debugSelected = visible[len(visible)-1].node.id
	} else {
		m.debugSelected = visible[0].node.id
	}
	m.ensureDebugSelectionVisible()
}

func (m Model) renderDebugWorkspaceContent(width int) debugWorkspaceRender {
	lines := []string{
		fmt.Sprintf("Debug tree for %s", m.roleTitle()),
		"View: role-scoped AI API requests and responses in a drill-down tree",
		"",
	}

	if m.debugLog == nil {
		lines = append(lines, "Debug log not available. AI logging requires a live game with AI players.")
		return debugWorkspaceRender{lines: lines}
	}

	visible := m.visibleDebugNodes()
	if len(visible) == 0 {
		lines = append(lines, "No AI API calls recorded yet for this role.")
		return debugWorkspaceRender{lines: lines}
	}

	textWidth := paneTextWidth(width)
	selectedLineStart := -1
	selectedLineCount := 0

	for _, visibleNode := range visible {
		nodeLines := renderDebugTreeNode(visibleNode, textWidth, m.isDebugNodeExpanded(visibleNode.node.id, visibleNode.depth), visibleNode.node.id == m.debugSelected)
		if visibleNode.node.id == m.debugSelected {
			selectedLineStart = len(flattenLines(lines))
			selectedLineCount = len(flattenLines(nodeLines))
		}
		lines = append(lines, nodeLines...)
	}

	if selectedLineStart < 0 {
		selectedLineStart = 0
	}

	return debugWorkspaceRender{
		lines:             lines,
		selectedLineStart: selectedLineStart,
		selectedLineCount: max(selectedLineCount, 1),
	}
}

func renderDebugTreeNode(visible debugVisibleNode, width int, expanded, selected bool) []string {
	indent := strings.Repeat("  ", visible.depth)
	prefix := " "
	if selected {
		prefix = ">"
	}

	marker := "."
	if len(visible.node.children) > 0 {
		marker = "+"
		if expanded {
			marker = "-"
		}
	}

	lines := wrapIndentedBlock(prefix+" "+indent+marker+" "+visible.node.title, width, prefix+" "+indent+"  ")
	if strings.TrimSpace(visible.node.body) == "" {
		return lines
	}

	bodyIndent := prefix + " " + indent + "    "
	lines = append(lines, wrapIndentedBlock(bodyIndent+visible.node.body, width, bodyIndent)...)
	return lines
}

func wrapIndentedBlock(text string, width int, continuation string) []string {
	parts := strings.Split(text, "\n")
	lines := make([]string, 0, len(parts))
	for _, part := range parts {
		wrapped := wrapLine(part, width)
		segments := strings.Split(wrapped, "\n")
		for index, segment := range segments {
			if index == 0 {
				lines = append(lines, segment)
				continue
			}
			lines = append(lines, continuation+strings.TrimSpace(segment))
		}
	}
	return lines
}

func debugAttemptOutcome(record ports.AICallRecord) string {
	if record.Valid {
		return "Success"
	}
	return "Failure"
}

func debugAttemptStatus(record ports.AICallRecord) (string, string) {
	if record.Valid {
		return "Success", "Accepted final response."
	}
	if record.IsToolCall {
		return "Failure", "The model requested a lookup tool call, so the attempt did not return a final decision.\n" + debugStatusErrorBody(record)
	}
	if strings.TrimSpace(record.ErrorMessage) != "" {
		return "Failure", debugFailureKind(record) + "\n" + debugStatusErrorBody(record)
	}
	return "Failure", "The attempt did not produce a valid final response."
}

func debugFailureKind(record ports.AICallRecord) string {
	if strings.TrimSpace(record.RawResponse) == "" {
		return "Transport/provider failure."
	}
	return "Parse/validation failure."
}

func debugStatusErrorBody(record ports.AICallRecord) string {
	if strings.TrimSpace(record.ErrorMessage) == "" {
		return "No additional error details were recorded."
	}
	return "Recorded details:\n" + sanitizeDebugText(record.ErrorMessage)
}

func debugRequestDetails(record ports.AICallRecord) string {
	var sections []string
	if strings.TrimSpace(record.SystemPrompt) != "" {
		sections = append(sections, "System prompt:\n"+sanitizeDebugText(record.SystemPrompt))
	}
	if strings.TrimSpace(record.UserPrompt) != "" {
		sections = append(sections, "User prompt:\n"+sanitizeDebugText(record.UserPrompt))
	}
	if len(sections) == 0 {
		return "No request payload was recorded."
	}
	return strings.Join(sections, "\n\n")
}

func debugResponseSummary(record ports.AICallRecord) string {
	switch {
	case strings.TrimSpace(record.RawResponse) == "":
		return "no response body captured"
	case record.IsToolCall:
		return fmt.Sprintf("%d chars, tool call", len(record.RawResponse))
	case record.Valid:
		return fmt.Sprintf("%d chars, valid JSON", len(record.RawResponse))
	default:
		return fmt.Sprintf("%d chars, invalid response", len(record.RawResponse))
	}
}

func debugResponseDetails(record ports.AICallRecord) string {
	if strings.TrimSpace(record.RawResponse) == "" {
		return "No response body was captured for this attempt."
	}
	return sanitizeDebugText(record.RawResponse)
}

func sanitizeDebugText(text string) string {
	return strings.ReplaceAll(text, "\t", " ")
}

func debugRoundNodeID(round domain.RoundNumber) string {
	return fmt.Sprintf("debug:round:%06d", round)
}

func debugAttemptNodeID(round domain.RoundNumber, attempt int) string {
	return fmt.Sprintf("debug:round:%06d:attempt:%03d", round, attempt)
}

func debugStatusNodeID(round domain.RoundNumber, attempt int) string {
	return debugAttemptNodeID(round, attempt) + ":status"
}

func debugRequestSummaryNodeID(round domain.RoundNumber, attempt int) string {
	return debugAttemptNodeID(round, attempt) + ":request:summary"
}

func debugRequestDetailNodeID(round domain.RoundNumber, attempt int) string {
	return debugAttemptNodeID(round, attempt) + ":request:detail"
}

func debugResponseSummaryNodeID(round domain.RoundNumber, attempt int) string {
	return debugAttemptNodeID(round, attempt) + ":response:summary"
}

func debugResponseDetailNodeID(round domain.RoundNumber, attempt int) string {
	return debugAttemptNodeID(round, attempt) + ":response:detail"
}

func parseRoundID(id string) domain.RoundNumber {
	var round int
	_, _ = fmt.Sscanf(id, "debug:round:%06d", &round)
	return domain.RoundNumber(round)
}
