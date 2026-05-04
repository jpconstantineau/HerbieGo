package main

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type startMenuAction int

const (
	startMenuActionStartGame startMenuAction = iota
	startMenuActionExit
)

type startMenuModel struct {
	config     app.Config
	scenarios  []domain.ScenarioID
	cursor     int
	width      int
	height     int
	action     startMenuAction
	cancelled  bool
	statusLine string
}

func runStartMenu(cfg app.Config) (startMenuAction, app.Config, error) {
	model := newStartMenuModel(cfg)
	final, err := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return startMenuActionExit, cfg, err
	}

	menu, ok := final.(startMenuModel)
	if !ok {
		return startMenuActionExit, cfg, fmt.Errorf("unexpected start menu result type %T", final)
	}
	if menu.cancelled {
		return startMenuActionExit, menu.config, nil
	}
	return menu.action, menu.config, nil
}

func newStartMenuModel(cfg app.Config) startMenuModel {
	normalized := cloneConfig(cfg)
	if normalized.Roles == nil {
		normalized.Roles = make(map[domain.RoleID]app.RoleConfig)
	}
	availableScenarios := scenario.RegisteredIDs()
	if len(availableScenarios) == 0 {
		availableScenarios = []domain.ScenarioID{scenario.DefaultID}
	}
	if !slices.Contains(availableScenarios, normalized.ScenarioID) {
		normalized.ScenarioID = availableScenarios[0]
	}

	return startMenuModel{
		config:     normalized,
		scenarios:  availableScenarios,
		statusLine: "Use up/down to move, left/right or enter to change settings, and enter on Start Game to launch.",
	}
}

func (m startMenuModel) Init() tea.Cmd {
	return nil
}

func (m startMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		return m, nil
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "esc", "q":
			m.cancelled = true
			m.action = startMenuActionExit
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "left", "h":
			m.adjustSelected(-1)
		case "right", "l":
			m.adjustSelected(1)
		case "enter", " ":
			if m.activateSelected() {
				return m, tea.Quit
			}
		}
		return m, nil
	}

	return m, nil
}

func (m startMenuModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("223"))

	lines := []string{
		titleStyle.Render("HerbieGo Start Menu"),
		subtitleStyle.Render("Set the launch configuration, then start a match or exit."),
		"",
	}

	for index, item := range m.items() {
		line := fmt.Sprintf("  %s", item.label)
		if index == m.cursor {
			line = selectedStyle.Render("> " + item.label)
		} else {
			line = itemStyle.Render(line)
		}
		lines = append(lines, line)
	}

	lines = append(lines,
		"",
		hintStyle.Render("Controls: up/down move | left/right change | enter select"),
		statusStyle.Render(m.statusLine),
	)

	width := m.width
	if width <= 0 {
		width = 96
	}

	return lipgloss.NewStyle().
		Padding(1, 2).
		Width(width).
		Render(strings.Join(lines, "\n"))
}

func (m *startMenuModel) moveCursor(delta int) {
	itemCount := len(m.items())
	if itemCount == 0 {
		return
	}

	m.cursor = (m.cursor + delta + itemCount) % itemCount
	m.statusLine = fmt.Sprintf("Selected %s.", m.items()[m.cursor].title)
}

func (m *startMenuModel) adjustSelected(delta int) {
	if delta == 0 {
		return
	}

	item := m.items()[m.cursor]
	switch item.kind {
	case menuItemScenario:
		m.shiftScenario(delta)
	case menuItemRole:
		m.toggleRoleControl(item.roleID)
	}
}

func (m *startMenuModel) activateSelected() bool {
	item := m.items()[m.cursor]
	switch item.kind {
	case menuItemStartGame:
		m.action = startMenuActionStartGame
		return true
	case menuItemExit:
		m.action = startMenuActionExit
		return true
	case menuItemScenario:
		m.shiftScenario(1)
	case menuItemRole:
		m.toggleRoleControl(item.roleID)
	}
	return false
}

func (m *startMenuModel) shiftScenario(delta int) {
	if len(m.scenarios) == 0 {
		return
	}

	current := slices.Index(m.scenarios, m.config.ScenarioID)
	if current < 0 {
		current = 0
	}
	current = (current + delta + len(m.scenarios)) % len(m.scenarios)
	m.config.ScenarioID = m.scenarios[current]
	m.statusLine = fmt.Sprintf("Scenario set to %s.", scenarioTitle(m.config.ScenarioID))
}

func (m *startMenuModel) toggleRoleControl(roleID domain.RoleID) {
	if m.config.Roles == nil {
		m.config.Roles = make(map[domain.RoleID]app.RoleConfig)
	}
	roleCfg, ok := m.config.Roles[roleID]
	if !ok {
		roleCfg = defaultRoleConfig(m.config)
	}
	if roleCfg.Kind == app.PlayerKindHuman {
		roleCfg.Kind = app.PlayerKindAI
	} else {
		roleCfg.Kind = app.PlayerKindHuman
	}
	m.config.Roles[roleID] = roleCfg
	m.statusLine = fmt.Sprintf("%s is now %s-controlled.", displayRoleName(roleID), roleControlLabel(roleCfg))
}

type menuItemKind int

const (
	menuItemStartGame menuItemKind = iota
	menuItemScenario
	menuItemRole
	menuItemExit
)

type menuItem struct {
	kind   menuItemKind
	title  string
	label  string
	roleID domain.RoleID
}

func (m startMenuModel) items() []menuItem {
	items := []menuItem{
		{
			kind:  menuItemStartGame,
			title: "Start Game",
			label: "Start Game",
		},
		{
			kind:  menuItemScenario,
			title: "Scenario",
			label: fmt.Sprintf("Scenario: %s", scenarioTitle(m.config.ScenarioID)),
		},
	}

	for _, roleID := range selectedScenarioRoster(m.config.ScenarioID) {
		roleCfg, ok := m.config.Roles[roleID]
		if !ok {
			roleCfg = defaultRoleConfig(m.config)
		}
		items = append(items, menuItem{
			kind:   menuItemRole,
			title:  displayRoleName(roleID),
			label:  fmt.Sprintf("%s: %s", displayRoleName(roleID), roleControlLabel(roleCfg)),
			roleID: roleID,
		})
	}

	items = append(items, menuItem{
		kind:  menuItemExit,
		title: "Exit",
		label: "Exit",
	})
	return items
}

func selectedScenarioRoster(scenarioID domain.ScenarioID) []domain.RoleID {
	definition, ok := scenario.Lookup(scenarioID)
	if !ok {
		return domain.CanonicalRoles()
	}
	return slices.Clone(definition.Setup.RoleRoster)
}

func scenarioTitle(id domain.ScenarioID) string {
	definition, ok := scenario.Lookup(id)
	if !ok {
		return string(id)
	}
	return definition.DisplayName
}

func roleControlLabel(cfg app.RoleConfig) string {
	controller := "AI"
	if cfg.Kind == app.PlayerKindHuman {
		controller = "Human"
	}
	if strings.TrimSpace(string(cfg.Provider)) == "" || strings.TrimSpace(cfg.Model) == "" {
		return controller
	}
	return fmt.Sprintf("%s (%s / %s)", controller, cfg.Provider, cfg.Model)
}

func defaultRoleConfig(cfg app.Config) app.RoleConfig {
	if entry, ok := firstCatalogEntry(cfg.LLMCatalog); ok {
		return app.RoleConfig{
			Kind:       app.PlayerKindAI,
			Provider:   entry.Provider,
			Model:      entry.Model,
			URL:        entry.URL,
			APISDKType: entry.APISDKType,
			APIKey:     entry.APIKey,
		}
	}
	return app.RoleConfig{Kind: app.PlayerKindAI}
}

func firstCatalogEntry(catalog app.LLMCatalog) (app.LLMCatalogEntry, bool) {
	if len(catalog.Entries) == 0 {
		return app.LLMCatalogEntry{}, false
	}
	return catalog.Entries[0], true
}

func cloneConfig(cfg app.Config) app.Config {
	cloned := cfg
	cloned.RoleConfigs = slices.Clone(cfg.RoleConfigs)
	cloned.LLMCatalog.Entries = slices.Clone(cfg.LLMCatalog.Entries)
	if cfg.Roles != nil {
		cloned.Roles = make(map[domain.RoleID]app.RoleConfig, len(cfg.Roles))
		for roleID, roleCfg := range cfg.Roles {
			cloned.Roles[roleID] = roleCfg
		}
	}
	return cloned
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
