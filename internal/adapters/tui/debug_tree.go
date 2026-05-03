package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

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
	var roots []*debugTreeNode
	if inspection := m.buildRoundInspectionTree(); inspection != nil {
		roots = append(roots, inspection)
	}
	if traces := m.buildTraceTree(); traces != nil {
		roots = append(roots, traces)
	}
	return roots
}

func (m Model) buildRoundInspectionTree() *debugTreeNode {
	if len(m.state.History.RecentRounds) == 0 {
		return nil
	}

	root := &debugTreeNode{
		id:    "debug:inspection",
		title: fmt.Sprintf("Round inspections (%d retained)", len(m.state.History.RecentRounds)),
	}
	for _, round := range slices.Backward(m.state.History.RecentRounds) {
		root.children = append(root.children, m.buildRoundInspectionNode(round))
	}
	return root
}

func (m Model) buildRoundInspectionNode(round domain.RoundRecord) *debugTreeNode {
	node := &debugTreeNode{
		id:       debugInspectionRoundNodeID(round.Round),
		parentID: "debug:inspection",
		title: fmt.Sprintf(
			"Round %d (%d actions | %d events | %d commentary)",
			round.Round,
			len(round.Actions),
			len(round.Events),
			len(round.Commentary),
		),
	}

	actionNode := &debugTreeNode{
		id:       debugInspectionActionNodeID(round.Round),
		parentID: node.id,
		title:    fmt.Sprintf("Action inspection for %s", m.roleTitle()),
		body:     debugActionInspectionBody(round, m.selectedAssignment().RoleID),
	}
	stateNode := &debugTreeNode{
		id:       debugInspectionStateNodeID(round.Round),
		parentID: node.id,
		title:    "State transition summary",
		body:     m.debugStateDiffBody(round.Round, round.Metrics),
	}
	timelineNode := &debugTreeNode{
		id:       debugInspectionTimelineNodeID(round.Round),
		parentID: node.id,
		title:    "Round timeline highlights",
		body:     debugTimelineBody(round),
	}

	node.children = []*debugTreeNode{actionNode, stateNode, timelineNode}
	return node
}

func (m Model) buildTraceTree() *debugTreeNode {
	records := m.debugRoleRecords()
	if len(records) == 0 {
		return nil
	}

	root := &debugTreeNode{
		id:    "debug:traces",
		title: fmt.Sprintf("Prompt/response traces for %s (%d total)", m.roleTitle(), len(records)),
	}
	for _, roundNode := range buildTraceRoundNodes(records) {
		roundNode.parentID = root.id
		root.children = append(root.children, roundNode)
	}
	return root
}

func buildTraceRoundNodes(records []ports.AICallRecord) []*debugTreeNode {
	roundNodes := make([]*debugTreeNode, 0)
	roundIndex := make(map[domain.RoundNumber]*debugTreeNode)
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

		roundNode.children = append(roundNode.children, buildDebugAttemptNode(record, roundNode.id))
		attemptCounts[record.Round]++
	}

	slices.SortFunc(roundNodes, func(left, right *debugTreeNode) int {
		switch {
		case left.id > right.id:
			return -1
		case left.id < right.id:
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

func buildDebugAttemptNode(record ports.AICallRecord, parentID string) *debugTreeNode {
	_, statusBody := debugAttemptStatus(record)
	requestSummary := fmt.Sprintf(
		"Request summary (%s/%s, system %d chars, user %d chars)",
		record.Provider,
		record.Model,
		len(record.SystemPrompt),
		len(record.UserPrompt),
	)
	responseSummary := fmt.Sprintf(
		"Response summary (%s)",
		debugResponseSummary(record),
	)

	attemptNode := &debugTreeNode{
		id:       debugAttemptNodeID(record.Round, record.Attempt),
		parentID: parentID,
		title:    fmt.Sprintf("Try %d - %s", record.Attempt, debugAttemptOutcome(record)),
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
	return depth <= 1
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
		fmt.Sprintf("Debug inspector for %s", m.roleTitle()),
		"View: round actions, state transitions, and AI prompt/response traces in one drill-down tree",
		"",
	}

	visible := m.visibleDebugNodes()
	if len(visible) == 0 {
		if len(m.state.History.RecentRounds) == 0 && m.debugLog == nil {
			lines = append(lines, "No debug data is available yet.")
		} else {
			lines = append(lines, "No round inspection or AI trace entries are available yet for this role.")
		}
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

func debugActionInspectionBody(round domain.RoundRecord, roleID domain.RoleID) string {
	var selected *domain.ActionSubmission
	for i := range round.Actions {
		if round.Actions[i].RoleID == roleID {
			clone := round.Actions[i].Clone()
			selected = &clone
			break
		}
	}
	if selected == nil {
		return fmt.Sprintf("No submitted action for %s was recorded in round %d.", displayRoleName(roleID), round.Round)
	}

	lines := []string{
		fmt.Sprintf("Role: %s", displayRoleName(selected.RoleID)),
		fmt.Sprintf("Action ID: %s", selected.ActionID),
	}
	if !selected.SubmittedAt.IsZero() {
		lines = append(lines, "Submitted at: "+selected.SubmittedAt.UTC().Format(time.RFC3339))
	}
	lines = append(lines, "Action summary:")
	for _, summary := range summarizeAction(selected.Action) {
		lines = append(lines, "  - "+summary)
	}
	if strings.TrimSpace(selected.Commentary.Body) != "" {
		lines = append(lines, "Commentary: "+selected.Commentary.Body)
	}
	return strings.Join(lines, "\n")
}

func (m Model) debugStateDiffBody(round domain.RoundNumber, metrics domain.PlantMetrics) string {
	before, after, ok := m.roundStatePair(round)
	if !ok {
		return "State diff unavailable. This inspector needs round snapshots from the current live session or persisted replay store."
	}

	type diffLine struct {
		label string
		from  int
		to    int
	}
	diffs := []diffLine{
		{label: "Cash", from: int(before.Plant.Cash), to: int(after.Plant.Cash)},
		{label: "Debt", from: int(before.Plant.Debt), to: int(after.Plant.Debt)},
		{label: "Backlog lines", from: len(before.Plant.Backlog), to: len(after.Plant.Backlog)},
		{label: "Parts on hand units", from: int(before.Metrics.PartsOnHandUnits), to: int(after.Metrics.PartsOnHandUnits)},
		{label: "Finished goods units", from: int(before.Metrics.FinishedGoodsUnits), to: int(after.Metrics.FinishedGoodsUnits)},
		{label: "Inspection hold units", from: int(before.Metrics.InspectionHoldUnits), to: int(after.Metrics.InspectionHoldUnits)},
		{label: "Receivables open", from: len(before.Plant.Receivables), to: len(after.Plant.Receivables)},
		{label: "Payables open", from: len(before.Plant.Payables), to: len(after.Plant.Payables)},
	}

	lines := []string{
		fmt.Sprintf("State transition: round %d -> %d", before.CurrentRound, after.CurrentRound),
		fmt.Sprintf("Round profit: %d", metrics.RoundProfit),
		fmt.Sprintf("Net cash change: %d", metrics.NetCashChange),
	}
	changed := 0
	for _, diff := range diffs {
		if diff.from == diff.to {
			continue
		}
		changed++
		lines = append(lines, fmt.Sprintf("%s: %d -> %d (%+d)", diff.label, diff.from, diff.to, diff.to-diff.from))
	}
	if changed == 0 {
		lines = append(lines, "No tracked aggregate counters changed across the stored snapshots.")
	}
	return strings.Join(lines, "\n")
}

func debugTimelineBody(round domain.RoundRecord) string {
	timeline := round.CanonicalTimeline()
	if len(timeline) == 0 {
		return "No timeline entries were recorded for this round."
	}

	lines := make([]string, 0, min(len(timeline), 8)+1)
	lines = append(lines, fmt.Sprintf("Showing %d of %d timeline entries.", min(len(timeline), 8), len(timeline)))
	for _, entry := range timeline[:min(len(timeline), 8)] {
		switch entry.Kind {
		case domain.RoundTimelineKindCommentary:
			if entry.Commentary != nil {
				lines = append(lines, fmt.Sprintf("%s #%d: %s said %q", timelinePhaseLabel(entry.Phase), entry.Sequence, displayRoleName(entry.Commentary.RoleID), entry.Commentary.Body))
			}
		case domain.RoundTimelineKindEvent:
			if entry.Event != nil {
				lines = append(lines, fmt.Sprintf("%s #%d: %s", timelinePhaseLabel(entry.Phase), entry.Sequence, entry.Event.Summary))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) roundStatePair(round domain.RoundNumber) (domain.MatchState, domain.MatchState, bool) {
	snapshots := m.stateSnapshots()
	if len(snapshots) == 0 {
		return domain.MatchState{}, domain.MatchState{}, false
	}

	var before domain.MatchState
	var after domain.MatchState
	foundBefore := false
	foundAfter := false
	for _, snapshot := range snapshots {
		switch snapshot.CurrentRound {
		case round:
			before = snapshot.Clone()
			foundBefore = true
		case round + 1:
			after = snapshot.Clone()
			foundAfter = true
		}
	}
	return before, after, foundBefore && foundAfter
}

func (m Model) stateSnapshots() []domain.MatchState {
	source, ok := m.source.(StateSnapshotSource)
	if !ok {
		return nil
	}
	return source.StateSnapshots()
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

func debugInspectionRoundNodeID(round domain.RoundNumber) string {
	return fmt.Sprintf("debug:inspection:round:%06d", round)
}

func debugInspectionActionNodeID(round domain.RoundNumber) string {
	return debugInspectionRoundNodeID(round) + ":action"
}

func debugInspectionStateNodeID(round domain.RoundNumber) string {
	return debugInspectionRoundNodeID(round) + ":state"
}

func debugInspectionTimelineNodeID(round domain.RoundNumber) string {
	return debugInspectionRoundNodeID(round) + ":timeline"
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
