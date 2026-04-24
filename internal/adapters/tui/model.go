package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/projection"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

const (
	paneDepartments = iota
	paneHistory
	paneStats
	paneCommandBar
)

const (
	layoutWide layoutMode = iota
	layoutCompact
	layoutStacked
)

type layoutMode int

const (
	workspaceActionEntry workspaceMode = iota
	workspaceScenarioLookup
	workspaceRoleReport
	workspaceRoundFeed
	workspaceHistoryArchive
)

type workspaceMode int

type stateLoadedMsg struct {
	state domain.MatchState
}

type stateStreamClosedMsg struct{}

type spinnerTickMsg struct {
	generation int
}

// StatusMsg updates the command-bar status from outside the Bubble Tea model.
type StatusMsg struct {
	Text string
}

type SubmitFunc func(domain.ActionSubmission) error

// Model is the Bubble Tea shell for the round-based gameplay UI.
type Model struct {
	scenario      scenario.Definition
	source        StateSource
	updates       <-chan domain.MatchState
	submit        SubmitFunc
	state         domain.MatchState
	selectedRole  int
	focusedPane   int
	workspace     workspaceMode
	width         int
	height        int
	status        string
	streamClosed  bool
	spinnerFrame  int
	spinnerActive bool
	spinnerGen    int
	historyScroll int
	drafts        map[domain.RoleID]actionDraft
	lookup        lookupBrowserState
}

// NewModel constructs the main gameplay shell model.
func NewModel(definition scenario.Definition, source StateSource) Model {
	return NewModelWithSubmit(definition, source, nil)
}

// NewModelWithSubmit constructs the gameplay shell with an optional live
// submission hook for forwarding locked human actions into the shared runner.
func NewModelWithSubmit(definition scenario.Definition, source StateSource, submit SubmitFunc) Model {
	return Model{
		scenario:  definition,
		source:    source,
		updates:   source.Updates(),
		submit:    submit,
		workspace: workspaceActionEntry,
		status:    "Loading round state...",
		drafts:    make(map[domain.RoleID]actionDraft),
	}
}

// Init loads the initial state snapshot and subscribes to future updates.
func (m Model) Init() tea.Cmd {
	return tea.Batch(loadSnapshotCmd(m.source), waitForUpdateCmd(m.updates))
}

// Update handles keyboard input, terminal resize events, and new match snapshots.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		return m, nil
	case tea.KeyMsg:
		if m.handleScenarioLookupKey(typed) {
			return m, nil
		}
		if m.handleActionEntryKey(typed) {
			return m, nil
		}
		if m.handleHistoryScrollKey(typed) {
			return m, nil
		}
		switch typed.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.focusedPane = (m.focusedPane + 1) % 4
			m.status = fmt.Sprintf("Focused %s pane", paneName(m.focusedPane))
		case "shift+tab":
			m.focusedPane = (m.focusedPane + 3) % 4
			m.status = fmt.Sprintf("Focused %s pane", paneName(m.focusedPane))
		case "]":
			m.moveWorkspace(1)
		case "[":
			m.moveWorkspace(-1)
		case "1":
			m.setWorkspace(workspaceActionEntry)
		case "2":
			m.setWorkspace(workspaceScenarioLookup)
		case "3":
			m.setWorkspace(workspaceRoleReport)
		case "4":
			m.setWorkspace(workspaceRoundFeed)
		case "5":
			m.setWorkspace(workspaceHistoryArchive)
		case "up", "k":
			if m.focusedPane == paneDepartments {
				m.moveRole(-1)
			}
		case "down", "j":
			if m.focusedPane == paneDepartments {
				m.moveRole(1)
			}
		}
		return m, nil
	case tea.MouseMsg:
		if m.handleHistoryScrollMouse(typed) {
			return m, nil
		}
		return m, nil
	case stateLoadedMsg:
		if typed.state.CurrentRound != m.state.CurrentRound {
			clear(m.drafts)
			m.historyScroll = 0
		}
		m.state = typed.state.Clone()
		m.selectedRole = clampRoleIndex(m.selectedRole, len(m.state.Roles))
		m.status = fmt.Sprintf("Round %d loaded for %s", m.state.CurrentRound, m.roleTitle())
		cmds := []tea.Cmd{waitForUpdateCmd(m.updates)}
		if m.hasProviderWaits() {
			if !m.spinnerActive {
				m.spinnerActive = true
				m.spinnerGen++
				cmds = append(cmds, m.spinnerCmd(m.spinnerGen))
			}
		} else {
			m.spinnerActive = false
			m.spinnerFrame = 0
			m.spinnerGen++
		}
		return m, tea.Batch(cmds...)
	case StatusMsg:
		if strings.TrimSpace(typed.Text) != "" {
			m.status = typed.Text
		}
		return m, nil
	case stateStreamClosedMsg:
		m.streamClosed = true
		if strings.TrimSpace(m.status) == "" || strings.HasPrefix(m.status, "Round ") {
			m.status = "Match updates complete. Inspect results and press q to exit."
		}
		return m, nil
	case spinnerTickMsg:
		if typed.generation != m.spinnerGen {
			return m, nil
		}
		if !m.hasProviderWaits() {
			m.spinnerActive = false
			m.spinnerFrame = 0
			return m, nil
		}
		m.spinnerFrame = (m.spinnerFrame + 1) % len(providerSpinnerFrames)
		return m, m.spinnerCmd(m.spinnerGen)
	}

	return m, nil
}

var providerSpinnerFrames = []string{"⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇"}

var providerSpinnerFrameInterval = (2 * time.Second) / time.Duration(len(providerSpinnerFrames))

func (m Model) spinnerCmd(generation int) tea.Cmd {
	if !m.hasProviderWaits() {
		return nil
	}
	return tea.Tick(providerSpinnerFrameInterval, func(time.Time) tea.Msg {
		return spinnerTickMsg{generation: generation}
	})
}

func (m Model) hasProviderWaits() bool {
	return len(m.effectiveRoundFlow().ProviderWaitingRoles) > 0
}

// View renders the four-pane shell.
func (m Model) View() string {
	if m.state.MatchID == "" {
		return "Loading HerbieGo shell..."
	}

	width := fallbackDimension(m.width, 120)
	height := fallbackDimension(m.height, 32)
	layout := chooseLayout(width, height)
	frame := paneStyle(false)
	totalRows := layoutPaneRows(layout) + 1
	availableContentHeight := max(height-(totalRows*frame.GetVerticalFrameSize()), totalRows)
	commandHeight := commandBarHeight(availableContentHeight)
	contentHeight := max(availableContentHeight-commandHeight, layoutPaneRows(layout))
	commandWidth := max(width-frame.GetHorizontalFrameSize(), 1)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderContentArea(layout, width, contentHeight),
		m.renderCommandBar(commandWidth, commandHeight),
	)
}

func loadSnapshotCmd(source StateSource) tea.Cmd {
	return func() tea.Msg {
		return stateLoadedMsg{state: source.Snapshot()}
	}
}

func waitForUpdateCmd(updates <-chan domain.MatchState) tea.Cmd {
	if updates == nil {
		return nil
	}

	return func() tea.Msg {
		state, ok := <-updates
		if !ok {
			return stateStreamClosedMsg{}
		}
		return stateLoadedMsg{state: state.Clone()}
	}
}

func (m *Model) moveRole(delta int) {
	roleCount := len(m.state.Roles)
	if roleCount == 0 {
		return
	}

	m.selectedRole = (m.selectedRole + delta + roleCount) % roleCount
	m.historyScroll = 0
	m.status = fmt.Sprintf("Selected %s", m.roleTitle())
}

func (m *Model) moveWorkspace(delta int) {
	modeCount := int(workspaceHistoryArchive) + 1
	m.workspace = workspaceMode((int(m.workspace) + delta + modeCount) % modeCount)
	m.historyScroll = 0
	m.status = fmt.Sprintf("Workspace switched to %s", m.workspace.label())
}

func (m *Model) setWorkspace(mode workspaceMode) {
	if m.workspace == mode {
		m.status = fmt.Sprintf("Workspace remains on %s", m.workspace.label())
		return
	}

	m.workspace = mode
	m.historyScroll = 0
	m.status = fmt.Sprintf("Workspace switched to %s", m.workspace.label())
}

func (m Model) roleTitle() string {
	if len(m.state.Roles) == 0 {
		return "No role selected"
	}
	return displayRoleName(m.selectedAssignment().RoleID)
}

func (m Model) selectedAssignment() domain.RoleAssignment {
	if len(m.state.Roles) == 0 {
		return domain.RoleAssignment{}
	}
	return m.state.Roles[clampRoleIndex(m.selectedRole, len(m.state.Roles))]
}

func (m Model) selectedRoleView() domain.RoundView {
	assignment := m.selectedAssignment()
	view := projection.BuildRoundView(m.state, assignment.RoleID)
	view.RoundFlow = m.effectiveRoundFlow()
	return view
}

func (m Model) selectedRoleReport() domain.RoleRoundReport {
	assignment := m.selectedAssignment()
	return projection.BuildRoleRoundReport(m.state, assignment.RoleID)
}

func (m Model) renderDepartmentsPane(width, height int) string {
	report := m.selectedRoleReport()

	lines := []string{
		fmt.Sprintf("Scenario: %s", m.scenario.DisplayName),
		fmt.Sprintf("Match: %s", m.state.MatchID),
		fmt.Sprintf("Mode: %s", modeLabel(m.focusedPane)),
		"",
	}
	if len(m.state.Roles) == 0 {
		lines = append(lines, "No role assignments loaded yet.")
	}
	for index, assignment := range m.state.Roles {
		cursor := " "
		if index == clampRoleIndex(m.selectedRole, len(m.state.Roles)) {
			cursor = ">"
		}
		controller := "AI"
		if assignment.IsHuman {
			controller = "Human"
		}
		waiting := ""
		if slices.Contains(m.effectiveRoundFlow().ProviderWaitingRoles, assignment.RoleID) {
			waiting = " " + providerSpinnerFrames[m.spinnerFrame%len(providerSpinnerFrames)]
		}
		lines = append(lines, fmt.Sprintf("%s %s%s [%s]", cursor, displayRoleName(assignment.RoleID), waiting, controller))
	}

	if report.BonusReminder != "" {
		lines = append(lines, "", wrapLine("Bonus: "+report.BonusReminder, paneTextWidth(width)))
	}
	for _, detail := range report.Department.DetailLines {
		lines = append(lines, wrapLine("- "+detail, paneTextWidth(width)))
	}

	return renderPane("Departments", lines, width, height, m.focusedPane == paneDepartments)
}

func (m Model) renderHistoryPane(width, height int) string {
	lines := m.renderHistoryWorkspaceLines(width)
	if m.historyWorkspaceSupportsScroll() {
		return renderScrollablePane(workspacePaneTitle(), lines, width, height, m.focusedPane == paneHistory, m.historyScroll)
	}
	return renderPane(workspacePaneTitle(), lines, width, height, m.focusedPane == paneHistory)
}

func (m Model) renderHistoryWorkspaceLines(width int) []string {
	lines := []string{
		fmt.Sprintf("Mode: %s", m.workspace.label()),
		workspaceNavigationLine(m.workspace),
	}

	switch m.workspace {
	case workspaceScenarioLookup:
		lines = append(lines, m.renderScenarioLookupWorkspace(width)...)
	case workspaceRoleReport:
		lines = append(lines, m.renderRoleReportWorkspace(width)...)
	case workspaceActionEntry:
		lines = append(lines, m.renderActionEntryWorkspace(width)...)
	case workspaceHistoryArchive:
		lines = append(lines, m.renderHistoryArchiveWorkspace(width)...)
	default:
		lines = append(lines, m.renderRoundFeedWorkspace(width)...)
	}

	return lines
}

func (m Model) renderRoundFeedWorkspace(width int) []string {
	view := m.selectedRoleView()
	recentRounds := lastRoundEntries(view.RecentRounds, 3)

	lines := []string{
		fmt.Sprintf("Round %d for %s", view.Round, m.roleTitle()),
		"View: active round context and recent resolved feed",
		fmt.Sprintf("Current phase: %s", roundPhaseLabel(view.RoundFlow.Phase)),
	}
	lines = append(lines, roundFlowSummary(view.RoundFlow, m.state.Roles)...)

	if len(recentRounds) == 0 {
		lines = append(lines, "", "No resolved rounds recorded yet.", "Recent events and role commentary will appear here after round one.")
	} else {
		lines = append(lines, "", fmt.Sprintf("Recent resolved rounds (%d shown)", len(recentRounds)))
	}
	for _, entry := range historyFeedEntries(recentRounds) {
		lines = append(lines, wrapLine(entry, paneTextWidth(width)))
	}
	return lines
}

func (m Model) renderRoleReportWorkspace(width int) []string {
	report := m.selectedRoleReport()

	lines := []string{
		fmt.Sprintf("Role report for %s", m.roleTitle()),
		"View: current briefing, company snapshot, and department metrics",
	}
	if report.BonusReminder != "" {
		lines = append(lines, wrapLine("Bonus reminder: "+report.BonusReminder, paneTextWidth(width)))
	}
	company := companywideReportLines(report.Companywide)
	if len(company) > 0 {
		lines = append(lines, "", "Company snapshot")
		for _, line := range company {
			lines = append(lines, wrapLine("- "+line, paneTextWidth(width)))
		}
	}
	if len(report.Department.KeyMetrics) > 0 {
		lines = append(lines, "", "Key metrics")
		for _, metric := range report.Department.KeyMetrics {
			lines = append(lines, fmt.Sprintf("- %s: %d %s", metric.MetricID, metric.Value, metric.DisplayUnit))
		}
	}
	if len(report.Department.DetailLines) > 0 {
		lines = append(lines, "", "Role notes")
		for _, detail := range report.Department.DetailLines {
			lines = append(lines, wrapLine("- "+detail, paneTextWidth(width)))
		}
	}
	if len(lines) == 2 {
		lines = append(lines, "No additional role report content is available yet.")
	}
	return lines
}

func (m Model) renderHistoryArchiveWorkspace(width int) []string {
	lines := []string{
		fmt.Sprintf("Archive for %s", m.roleTitle()),
		"View: retained round-by-round history and trend summaries",
	}
	if len(m.state.History.RecentRounds) == 0 {
		lines = append(lines, "No historical rounds are retained yet.")
		return lines
	}
	lines = append(lines,
		fmt.Sprintf("Rounds retained: %d", len(m.state.History.RecentRounds)),
		"Use this view for older rounds and per-round summaries rather than the current feed.",
		"",
	)
	for _, entry := range archiveEntries(m.state.History.RecentRounds) {
		lines = append(lines, wrapLine(entry, paneTextWidth(width)))
	}
	return lines
}

func (m Model) renderStatsPane(width, height int) string {
	view := m.selectedRoleView()
	report := m.selectedRoleReport()

	lines := []string{
		fmt.Sprintf("Cash: %d", view.Plant.Cash),
		fmt.Sprintf("Debt: %d / %d", view.Plant.Debt, view.Plant.DebtCeiling),
		fmt.Sprintf("Backlog: %d", len(view.Plant.Backlog)),
		workstationSummary(view.Plant.Workstations),
		fmt.Sprintf("Parts on hand: %d", view.Metrics.PartsOnHandUnits),
		fmt.Sprintf("Finished goods: %d", view.Metrics.FinishedGoodsUnits),
		fmt.Sprintf("Revenue: %d", view.Metrics.ThroughputRevenue),
		fmt.Sprintf("Profit: %d", view.Metrics.RoundProfit),
		"",
		"Targets",
		fmt.Sprintf("Procurement: %d", view.ActiveTargets.ProcurementBudget),
		fmt.Sprintf("Production: %d", view.ActiveTargets.ProductionSpendBudget),
		fmt.Sprintf("Revenue: %d", view.ActiveTargets.RevenueTarget),
		fmt.Sprintf("Cash floor: %d", view.ActiveTargets.CashFloorTarget),
		fmt.Sprintf("Debt ceiling: %d", view.ActiveTargets.DebtCeilingTarget),
	}

	if len(report.Department.KeyMetrics) > 0 {
		lines = append(lines, "", "Role metrics")
		for _, metric := range report.Department.KeyMetrics {
			lines = append(lines, fmt.Sprintf("%s: %d %s", metric.MetricID, metric.Value, metric.DisplayUnit))
		}
	}

	return renderPane("Plant Stats", lines, width, height, m.focusedPane == paneStats)
}

func (m Model) renderCommandBar(width, height int) string {
	flow := m.effectiveRoundFlow()
	status := fmt.Sprintf("Mode: inspect | Focus: %s | Workspace: %s | Role: %s | Round: %d", paneName(m.focusedPane), m.workspace.label(), m.roleTitle(), m.state.CurrentRound)
	status += " | Phase: " + roundPhaseShortLabel(flow.Phase)
	if detail := strings.TrimSpace(m.statusLine()); detail != "" {
		status += " | " + detail
	}

	lines := []string{
		focusedPaneCommandHints(m.focusedPane, m.workspace),
		wrapLine(status, paneTextWidth(width)),
	}
	return renderPane("Command Bar", lines, width, height, m.focusedPane == paneCommandBar)
}

func (m Model) statusLine() string {
	status := m.status
	if m.streamClosed {
		status += " | engine updates paused"
	}
	return status
}

func renderPane(title string, lines []string, width, height int, focused bool) string {
	label := title
	if focused {
		label = title + " [focus]"
	}
	content := fitLines(lines, max(height-1, 0))

	return paneStyle(focused).
		Width(width).
		Height(height).
		Render(label + "\n" + strings.Join(content, "\n"))
}

func renderScrollablePane(title string, lines []string, width, height int, focused bool, offset int) string {
	label := title
	if focused {
		label = title + " [focus]"
	}

	content := viewportLines(lines, max(height-1, 0), offset)
	return paneStyle(focused).
		Width(width).
		Height(height).
		Render(label + "\n" + strings.Join(content, "\n"))
}

func paneStyle(focused bool) lipgloss.Style {
	borderColor := lipgloss.Color("62")
	if focused {
		borderColor = lipgloss.Color("205")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor)
}

func fitLines(lines []string, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}

	flattened := flattenLines(lines)
	if len(flattened) <= maxLines {
		return flattened
	}
	if maxLines == 1 {
		return []string{"..."}
	}

	fitted := append([]string{}, flattened[:maxLines-1]...)
	return append(fitted, "...")
}

func flattenLines(lines []string) []string {
	flattened := make([]string, 0, len(lines))
	for _, line := range lines {
		flattened = append(flattened, strings.Split(line, "\n")...)
	}
	return flattened
}

func viewportLines(lines []string, maxLines, offset int) []string {
	if maxLines <= 0 {
		return nil
	}

	flattened := flattenLines(lines)
	if len(flattened) <= maxLines {
		return flattened
	}

	clamped := clampScrollOffset(offset, len(flattened), maxLines)
	return append([]string{}, flattened[clamped:clamped+maxLines]...)
}

func clampScrollOffset(offset, lineCount, visibleLines int) int {
	if visibleLines <= 0 || lineCount <= visibleLines {
		return 0
	}
	if offset < 0 {
		return 0
	}

	maxOffset := lineCount - visibleLines
	if offset > maxOffset {
		return maxOffset
	}
	return offset
}

func paneWidths(totalWidth int) (int, int, int) {
	left := max(totalWidth/4, 24)
	right := max(totalWidth/4, 28)
	center := totalWidth - left - right
	if center < 34 {
		center = 34
		left = max((totalWidth-center)/2, 24)
		right = totalWidth - left - center
	}
	return left, center, right
}

func layoutPaneRows(layout layoutMode) int {
	switch layout {
	case layoutStacked:
		return 3
	case layoutCompact:
		return 2
	default:
		return 1
	}
}

func chooseLayout(width, height int) layoutMode {
	switch {
	case width < 72 || height < 18:
		return layoutStacked
	case width < 118 || height < 24:
		return layoutCompact
	default:
		return layoutWide
	}
}

func (m Model) renderContentArea(layout layoutMode, width, height int) string {
	frameWidth := paneStyle(false).GetHorizontalFrameSize()

	switch layout {
	case layoutStacked:
		departmentsHeight, historyHeight, statsHeight := stackedPaneHeights(height)
		paneWidth := max(width-frameWidth, 1)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderDepartmentsPane(paneWidth, departmentsHeight),
			m.renderHistoryPane(paneWidth, historyHeight),
			m.renderStatsPane(paneWidth, statsHeight),
		)
	case layoutCompact:
		topHeight, historyHeight := compactPaneHeights(height)
		leftWidth, rightWidth := splitWidth(max(width-(2*frameWidth), 2))
		top := lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderDepartmentsPane(leftWidth, topHeight),
			m.renderStatsPane(rightWidth, topHeight),
		)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			top,
			m.renderHistoryPane(max(width-frameWidth, 1), historyHeight),
		)
	default:
		leftWidth, centerWidth, rightWidth := paneWidths(max(width-(3*frameWidth), 3))
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderDepartmentsPane(leftWidth, height),
			m.renderHistoryPane(centerWidth, height),
			m.renderStatsPane(rightWidth, height),
		)
	}
}

func compactPaneHeights(totalHeight int) (int, int) {
	if totalHeight <= 2 {
		return 1, 1
	}

	top := max(totalHeight/3, 5)
	if remaining := totalHeight - top; remaining >= 5 {
		return top, remaining
	}

	top = max(totalHeight-5, 3)
	return top, max(totalHeight-top, 1)
}

func stackedPaneHeights(totalHeight int) (int, int, int) {
	base := max(totalHeight/3, 1)
	remaining := max(totalHeight-base, 1)
	history := max(remaining/2, 1)
	stats := max(totalHeight-base-history, 1)
	return base, history, stats
}

func splitWidth(totalWidth int) (int, int) {
	left := max(totalWidth/3, 28)
	if left > totalWidth-28 {
		left = max(totalWidth/2, 24)
	}
	return left, max(totalWidth-left, 28)
}

func commandBarHeight(totalHeight int) int {
	switch {
	case totalHeight < 20:
		return 1
	case totalHeight < 24:
		return 2
	default:
		return 3
	}
}

func fallbackDimension(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func paneTextWidth(width int) int {
	return max(width-2, 1)
}

func clampRoleIndex(index, roleCount int) int {
	if roleCount <= 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= roleCount {
		return roleCount - 1
	}
	return index
}

func wrapLine(line string, width int) string {
	if width <= 0 || len(line) <= width {
		return line
	}
	return lipgloss.NewStyle().Width(width).Render(line)
}

func modeLabel(focusedPane int) string {
	return fmt.Sprintf("Inspecting %s", paneName(focusedPane))
}

func workstationSummary(workstations []domain.WorkstationState) string {
	if len(workstations) == 0 {
		return "Workstations: waiting for first telemetry"
	}
	return fmt.Sprintf("Workstations: %d online", len(workstations))
}

func roundFlowSummary(flow domain.RoundFlowState, assignments []domain.RoleAssignment) []string {
	phase := flow.Phase
	if phase == "" {
		phase = domain.RoundPhaseCollecting
	}

	lines := []string{
		roundPhaseDescription(phase),
	}

	switch phase {
	case domain.RoundPhaseCollecting:
		submitted := len(flow.SubmittedRoles)
		waiting := len(flow.WaitingOnRoles)
		total := submitted + waiting
		if total == 0 {
			total = len(assignments)
		}
		lines = append(lines,
			fmt.Sprintf("Submissions received: %d/%d", submitted, total),
			waitingOnSummary(flow.WaitingOnRoles),
			"Current-turn actions remain hidden until every role is collected and the round resolves.",
		)
	case domain.RoundPhaseResolving:
		lines = append(lines,
			"All current-turn actions are locked in.",
			"The plant is resolving simultaneous decisions before reveal.",
		)
	case domain.RoundPhaseRevealed:
		lines = append(lines,
			"Round results are now visible in the resolved history below.",
			revealDelaySummary(flow, assignments),
		)
	default:
		lines = append(lines, "Round flow is waiting for the next engine update.")
	}

	return lines
}

func waitingOnSummary(roleIDs []domain.RoleID) string {
	if len(roleIDs) == 0 {
		return "Waiting on: none"
	}

	names := make([]string, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		names = append(names, displayRoleName(roleID))
	}
	return "Waiting on: " + strings.Join(names, ", ")
}

func revealDelaySummary(flow domain.RoundFlowState, assignments []domain.RoleAssignment) string {
	if humanRoleCount(assignments) > 0 {
		return "Reveal remains visible until the next collection phase begins."
	}
	if flow.AIRevealDelaySeconds <= 0 {
		return "AI-only reveal pause is not configured."
	}
	return fmt.Sprintf("AI-only rounds hold the reveal for %d seconds before advancing.", flow.AIRevealDelaySeconds)
}

func humanRoleCount(assignments []domain.RoleAssignment) int {
	count := 0
	for _, assignment := range assignments {
		if assignment.IsHuman {
			count++
		}
	}
	return count
}

func roundPhaseLabel(phase domain.RoundPhase) string {
	switch phase {
	case domain.RoundPhaseResolving:
		return "resolving simultaneous turn"
	case domain.RoundPhaseRevealed:
		return "revealed round results"
	default:
		return "hidden simultaneous turn collection"
	}
}

func roundPhaseShortLabel(phase domain.RoundPhase) string {
	switch phase {
	case domain.RoundPhaseResolving:
		return "resolving"
	case domain.RoundPhaseRevealed:
		return "revealed"
	default:
		return "collecting"
	}
}

func roundPhaseDescription(phase domain.RoundPhase) string {
	switch phase {
	case domain.RoundPhaseResolving:
		return "The round is resolving."
	case domain.RoundPhaseRevealed:
		return "The round has been revealed."
	default:
		return "The round is waiting for simultaneous submissions."
	}
}

func paneName(index int) string {
	switch index {
	case paneDepartments:
		return "departments"
	case paneHistory:
		return "center workspace"
	case paneStats:
		return "plant stats"
	case paneCommandBar:
		return "command bar"
	default:
		return "unknown"
	}
}

func workspacePaneTitle() string {
	return "Center Workspace"
}

func (mode workspaceMode) label() string {
	switch mode {
	case workspaceActionEntry:
		return "action entry"
	case workspaceScenarioLookup:
		return "scenario lookup"
	case workspaceRoleReport:
		return "role report"
	case workspaceHistoryArchive:
		return "history archive"
	default:
		return "round feed"
	}
}

func workspaceNavigationLine(active workspaceMode) string {
	items := []workspaceMode{workspaceActionEntry, workspaceScenarioLookup, workspaceRoleReport, workspaceRoundFeed, workspaceHistoryArchive}
	labels := make([]string, 0, len(items))
	for index, mode := range items {
		label := fmt.Sprintf("%d %s", index+1, mode.shortLabel())
		if mode == active {
			label = "[" + label + "]"
		}
		labels = append(labels, label)
	}
	return "Navigate: " + strings.Join(labels, " | ") + " | [/] cycle"
}

func focusedPaneCommandHints(focusedPane int, active workspaceMode) string {
	base := "Inspect mode | tab/shift+tab focus panes | 1/2/3/4/5 switch workspace | [/] cycle | q quit"

	switch focusedPane {
	case paneDepartments:
		return base + " | departments: up/down select role"
	case paneHistory:
		return base + " | center workspace: " + workspaceInteractionHint(active)
	case paneStats:
		return base + " | plant stats: read-only summary"
	case paneCommandBar:
		return base + " | command bar: status and focused-pane shortcuts"
	default:
		return base
	}
}

func workspaceInteractionHint(active workspaceMode) string {
	switch active {
	case workspaceActionEntry:
		return "up/down move fields, enter edit/save, esc cancel, r review, s submit"
	case workspaceScenarioLookup:
		return "v/r/b/d switch lookup tabs, up/down browse entries"
	case workspaceRoleReport:
		return "role report is read-only"
	case workspaceHistoryArchive:
		return "up/down/pgup/pgdn/home/end scroll archive history"
	default:
		return "up/down/pgup/pgdn/home/end scroll round feed history"
	}
}

func (m Model) historyWorkspaceSupportsScroll() bool {
	return m.workspace == workspaceRoundFeed || m.workspace == workspaceHistoryArchive
}

func (m *Model) handleHistoryScrollKey(msg tea.KeyMsg) bool {
	if m.focusedPane != paneHistory || !m.historyWorkspaceSupportsScroll() {
		return false
	}

	switch msg.String() {
	case "up":
		m.scrollHistoryBy(-1)
	case "down":
		m.scrollHistoryBy(1)
	case "pgup":
		m.scrollHistoryBy(-m.historyPageSize())
	case "pgdown":
		m.scrollHistoryBy(m.historyPageSize())
	case "home":
		m.historyScroll = 0
	case "end":
		m.historyScroll = m.maxHistoryScrollOffset()
	default:
		return false
	}

	return true
}

func (m *Model) handleHistoryScrollMouse(msg tea.MouseMsg) bool {
	if m.focusedPane != paneHistory || !m.historyWorkspaceSupportsScroll() || !tea.MouseEvent(msg).IsWheel() {
		return false
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.scrollHistoryBy(-3)
	case tea.MouseButtonWheelDown:
		m.scrollHistoryBy(3)
	default:
		return false
	}

	return true
}

func (m *Model) scrollHistoryBy(delta int) {
	if delta == 0 {
		return
	}

	m.historyScroll = clampScrollOffset(m.historyScroll+delta, m.historyLineCount(), m.historyVisibleLineCount())
}

func (m Model) maxHistoryScrollOffset() int {
	return clampScrollOffset(m.historyLineCount(), m.historyLineCount(), m.historyVisibleLineCount())
}

func (m Model) historyPageSize() int {
	return max(m.historyVisibleLineCount()-1, 1)
}

func (m Model) historyLineCount() int {
	width, _ := m.historyPaneDimensions()
	return len(flattenLines(m.renderHistoryWorkspaceLines(width)))
}

func (m Model) historyVisibleLineCount() int {
	_, height := m.historyPaneDimensions()
	return max(height-1, 0)
}

func (m Model) historyPaneDimensions() (int, int) {
	width := fallbackDimension(m.width, 120)
	height := fallbackDimension(m.height, 32)
	layout := chooseLayout(width, height)
	frame := paneStyle(false)
	totalRows := layoutPaneRows(layout) + 1
	availableContentHeight := max(height-(totalRows*frame.GetVerticalFrameSize()), totalRows)
	commandHeight := commandBarHeight(availableContentHeight)
	contentHeight := max(availableContentHeight-commandHeight, layoutPaneRows(layout))
	frameWidth := frame.GetHorizontalFrameSize()

	switch layout {
	case layoutStacked:
		_, historyHeight, _ := stackedPaneHeights(contentHeight)
		return max(width-frameWidth, 1), historyHeight
	case layoutCompact:
		_, historyHeight := compactPaneHeights(contentHeight)
		return max(width-frameWidth, 1), historyHeight
	default:
		_, centerWidth, _ := paneWidths(max(width-(3*frameWidth), 3))
		return centerWidth, contentHeight
	}
}

func (mode workspaceMode) shortLabel() string {
	switch mode {
	case workspaceActionEntry:
		return "action"
	case workspaceScenarioLookup:
		return "lookup"
	case workspaceRoleReport:
		return "report"
	case workspaceHistoryArchive:
		return "archive"
	default:
		return "feed"
	}
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

func historyFeedEntries(rounds []domain.RoundHistoryEntry) []string {
	if len(rounds) == 0 {
		return nil
	}

	lines := make([]string, 0, len(rounds)*2)
	for _, round := range rounds {
		lines = append(lines, fmt.Sprintf("[R%d] %d events | %d commentary", round.Round, len(round.Events), len(round.Commentary)))
		timeline := historyEntryTimeline(round)
		if len(timeline) == 0 {
			lines = append(lines, "  No visible history.")
			continue
		}
		lines = append(lines, roundTimelineLines(timeline)...)
	}

	return lines
}

func archiveEntries(rounds []domain.RoundRecord) []string {
	if len(rounds) == 0 {
		return nil
	}

	lines := make([]string, 0, len(rounds)*2)
	for _, round := range slices.Backward(rounds) {
		lines = append(lines,
			fmt.Sprintf("[R%d] %d actions | %d events | %d commentary | profit %d | net cash %d",
				round.Round,
				len(round.Actions),
				len(round.Events),
				len(round.Commentary),
				round.Metrics.RoundProfit,
				round.Metrics.NetCashChange,
			),
		)
		timeline := round.CanonicalTimeline()
		if len(timeline) == 0 {
			lines = append(lines, "  No visible history.")
			continue
		}
		lines = append(lines, roundTimelineLines(timeline)...)
	}

	return lines
}

func roundTimelineLines(entries []domain.RoundTimelineEntry) []string {
	if len(entries) == 0 {
		return nil
	}

	lines := make([]string, 0, len(entries))
	var phase domain.RoundTimelinePhase
	for _, entry := range entries {
		if entry.Phase != phase {
			phase = entry.Phase
			lines = append(lines, fmt.Sprintf("  %s", timelinePhaseLabel(entry.Phase)))
		}

		switch entry.Kind {
		case domain.RoundTimelineKindCommentary:
			if entry.Commentary == nil {
				continue
			}
			lines = append(lines, fmt.Sprintf("    %d. %s: %s", entry.Sequence, displayRoleName(entry.Commentary.RoleID), entry.Commentary.Body))
		case domain.RoundTimelineKindEvent:
			if entry.Event == nil {
				continue
			}
			lines = append(lines, fmt.Sprintf("    %d. Event: %s", entry.Sequence, entry.Event.Summary))
		}
	}

	return lines
}

func historyEntryTimeline(entry domain.RoundHistoryEntry) []domain.RoundTimelineEntry {
	if len(entry.Timeline) > 0 {
		return cloneTimelineEntries(entry.Timeline)
	}

	round := domain.RoundRecord{
		Round:      entry.Round,
		Events:     entry.Events,
		Commentary: entry.Commentary,
	}
	return round.CanonicalTimeline()
}

func cloneTimelineEntries(entries []domain.RoundTimelineEntry) []domain.RoundTimelineEntry {
	cloned := make([]domain.RoundTimelineEntry, len(entries))
	for i := range entries {
		cloned[i] = entries[i].Clone()
	}
	return cloned
}

func timelinePhaseLabel(phase domain.RoundTimelinePhase) string {
	switch phase {
	case domain.RoundTimelinePhaseIntake:
		return "Player action intake"
	case domain.RoundTimelinePhaseSummary:
		return "Round summary"
	default:
		return "Round simulation"
	}
}

func companywideReportLines(report domain.CompanywidePerformanceReport) []string {
	lines := []string{
		fmt.Sprintf("Inventory value: %d", report.CurrentInventoryLevels.TotalValue),
	}

	lines = append(lines, companyMetricLine("New sales", report.NewSales))
	lines = append(lines, companyMetricLine("Unshipped sales", report.UnshippedSales))
	lines = append(lines, companyMetricLine("Sales at risk", report.SalesAtRisk))
	lines = append(lines, companyUnitMetricLine("Products produced last week", report.ProductsProducedLastWeek))
	lines = append(lines, fmt.Sprintf("Tracked product financial summaries: %d", len(report.Financials)))

	return lines
}

func companyMetricLine(label string, items []domain.ProductValueSummary) string {
	var units domain.Units
	var totalValue domain.Money
	for _, item := range items {
		units += item.Count
		totalValue += item.TotalValue
	}
	if len(items) == 0 {
		return label + ": none recorded"
	}
	return fmt.Sprintf("%s: %d units across %d products worth %d", label, units, len(items), totalValue)
}

func companyUnitMetricLine(label string, items []domain.ProductUnitSummary) string {
	var units domain.Units
	for _, item := range items {
		units += item.Count
	}
	if len(items) == 0 {
		return label + ": none recorded"
	}
	return fmt.Sprintf("%s: %d units across %d products", label, units, len(items))
}

func lastRoundEntries(rounds []domain.RoundHistoryEntry, limit int) []domain.RoundHistoryEntry {
	if len(rounds) <= limit {
		return rounds
	}
	return rounds[len(rounds)-limit:]
}
