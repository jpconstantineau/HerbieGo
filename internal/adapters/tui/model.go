package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/projection"
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

type stateLoadedMsg struct {
	state domain.MatchState
}

type stateStreamClosedMsg struct{}

// Model is the Bubble Tea shell for the round-based gameplay UI.
type Model struct {
	scenarioName string
	source       StateSource
	updates      <-chan domain.MatchState
	state        domain.MatchState
	selectedRole int
	focusedPane  int
	width        int
	height       int
	status       string
	streamClosed bool
}

// NewModel constructs the main gameplay shell model.
func NewModel(scenarioName string, source StateSource) Model {
	return Model{
		scenarioName: scenarioName,
		source:       source,
		updates:      source.Updates(),
		status:       "Loading round state...",
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
		switch typed.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.focusedPane = (m.focusedPane + 1) % 4
			m.status = fmt.Sprintf("Focused %s pane", paneName(m.focusedPane))
		case "shift+tab":
			m.focusedPane = (m.focusedPane + 3) % 4
			m.status = fmt.Sprintf("Focused %s pane", paneName(m.focusedPane))
		case "left", "h", "p":
			m.moveRole(-1)
		case "right", "l", "n":
			m.moveRole(1)
		}
		return m, nil
	case stateLoadedMsg:
		m.state = typed.state.Clone()
		m.selectedRole = clampRoleIndex(m.selectedRole, len(m.state.Roles))
		m.status = fmt.Sprintf("Round %d loaded for %s", m.state.CurrentRound, m.roleTitle())
		return m, waitForUpdateCmd(m.updates)
	case stateStreamClosedMsg:
		m.streamClosed = true
		if strings.TrimSpace(m.status) == "" {
			m.status = "State stream closed"
		}
		return m, nil
	}

	return m, nil
}

// View renders the four-pane shell.
func (m Model) View() string {
	if m.state.MatchID == "" {
		return "Loading HerbieGo shell..."
	}

	width := fallbackDimension(m.width, 120)
	height := fallbackDimension(m.height, 32)
	commandHeight := commandBarHeight(height)
	contentHeight := max(height-commandHeight, 6)
	layout := chooseLayout(width, contentHeight)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderContentArea(layout, width, contentHeight),
		m.renderCommandBar(width, commandHeight),
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
	m.status = fmt.Sprintf("Selected %s", m.roleTitle())
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
	return projection.BuildRoundView(m.state, assignment.RoleID)
}

func (m Model) selectedRoleReport() domain.RoleRoundReport {
	assignment := m.selectedAssignment()
	return projection.BuildRoleRoundReport(m.state, assignment.RoleID)
}

func (m Model) renderDepartmentsPane(width, height int) string {
	report := m.selectedRoleReport()

	lines := []string{
		fmt.Sprintf("Scenario: %s", m.scenarioName),
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
		lines = append(lines, fmt.Sprintf("%s %s [%s]", cursor, displayRoleName(assignment.RoleID), controller))
	}

	if report.BonusReminder != "" {
		lines = append(lines, "", wrapLine("Bonus: "+report.BonusReminder, width-4))
	}
	for _, detail := range report.Department.DetailLines {
		lines = append(lines, wrapLine("- "+detail, width-4))
	}

	return renderPane("Departments", lines, width, height, m.focusedPane == paneDepartments)
}

func (m Model) renderHistoryPane(width, height int) string {
	view := m.selectedRoleView()

	lines := []string{
		fmt.Sprintf("Round %d for %s", view.Round, m.roleTitle()),
		"Presentation: merged round feed",
	}
	if len(view.RecentRounds) == 0 {
		lines = append(lines, "No prior rounds recorded yet.", "Resolved events and role commentary will appear here after round one.")
	} else {
		lines = append(lines, "")
	}
	for _, entry := range historyFeedEntries(view.RecentRounds) {
		lines = append(lines, wrapLine(entry, width-4))
	}

	return renderPane("History", lines, width, height, m.focusedPane == paneHistory)
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
	status := fmt.Sprintf("Mode: inspect | Focus: %s | Role: %s | Round: %d", paneName(m.focusedPane), m.roleTitle(), m.state.CurrentRound)
	if detail := strings.TrimSpace(m.statusLine()); detail != "" {
		status += " | " + detail
	}

	lines := []string{
		"Inspect mode | tab/shift+tab focus panes | left/right cycle roles | q quit",
		wrapLine(status, width-4),
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
	border := lipgloss.RoundedBorder()
	borderColor := lipgloss.Color("62")
	label := title
	if focused {
		borderColor = lipgloss.Color("205")
		label = title + " [focus]"
	}

	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(border).
		BorderForeground(borderColor).
		Padding(0, 1)

	content := fitLines(lines, max(height-2, 1))
	return style.Render(label + "\n" + strings.Join(content, "\n"))
}

func fitLines(lines []string, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}
	if len(lines) <= maxLines {
		return lines
	}
	if maxLines == 1 {
		return []string{"..."}
	}

	fitted := append([]string{}, lines[:maxLines-1]...)
	return append(fitted, "...")
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
	switch layout {
	case layoutStacked:
		departmentsHeight, historyHeight, statsHeight := stackedPaneHeights(height)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderDepartmentsPane(width, departmentsHeight),
			m.renderHistoryPane(width, historyHeight),
			m.renderStatsPane(width, statsHeight),
		)
	case layoutCompact:
		topHeight, historyHeight := compactPaneHeights(height)
		leftWidth, rightWidth := splitWidth(width)
		top := lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderDepartmentsPane(leftWidth, topHeight),
			m.renderStatsPane(rightWidth, topHeight),
		)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			top,
			m.renderHistoryPane(width, historyHeight),
		)
	default:
		leftWidth, centerWidth, rightWidth := paneWidths(width)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderDepartmentsPane(leftWidth, height),
			m.renderHistoryPane(centerWidth, height),
			m.renderStatsPane(rightWidth, height),
		)
	}
}

func compactPaneHeights(totalHeight int) (int, int) {
	top := max(totalHeight/3, 7)
	if history := totalHeight - top; history >= 8 {
		return top, history
	}
	return max(totalHeight/2, 7), max(totalHeight-totalHeight/2, 7)
}

func stackedPaneHeights(totalHeight int) (int, int, int) {
	base := max(totalHeight/3, 5)
	remaining := totalHeight - base
	history := max(remaining/2, 6)
	stats := max(totalHeight-base-history, 5)
	if base+history+stats > totalHeight {
		stats = max(totalHeight-base-history, 4)
	}
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
	case totalHeight < 24:
		return 4
	default:
		return 5
	}
}

func fallbackDimension(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
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

func paneName(index int) string {
	switch index {
	case paneDepartments:
		return "departments"
	case paneHistory:
		return "history"
	case paneStats:
		return "plant stats"
	case paneCommandBar:
		return "command bar"
	default:
		return "unknown"
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
		if len(round.Events) == 0 && len(round.Commentary) == 0 {
			lines = append(lines, "  No visible history.")
			continue
		}

		for _, event := range round.Events {
			lines = append(lines, fmt.Sprintf("  Event: %s", event.Summary))
		}
		for _, commentary := range round.Commentary {
			lines = append(lines, fmt.Sprintf("  %s: %s", displayRoleName(commentary.RoleID), commentary.Body))
		}
	}

	return lines
}
