package main

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jpconstantineau/herbiego/internal/adapters/persistence/sqlite"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type startMenuAction int

const (
	startMenuActionStartNewGame startMenuAction = iota
	startMenuActionResumeGame
	startMenuActionLoadGame
	startMenuActionSaveGame
	startMenuActionExit
)

type startMenuState struct {
	ActiveSession     *activeSession
	SaveSlots         []sqlite.SaveSlotSummary
	SelectedLoadIndex int
	SelectedSaveIndex int
	StoreEnabled      bool
	StatusText        string
}

func (s *startMenuState) clampSelections() {
	if len(s.SaveSlots) == 0 {
		s.SelectedLoadIndex = 0
		s.SelectedSaveIndex = 0
		return
	}
	if s.SelectedLoadIndex < 0 {
		s.SelectedLoadIndex = 0
	}
	if s.SelectedLoadIndex >= len(s.SaveSlots) {
		s.SelectedLoadIndex = len(s.SaveSlots) - 1
	}
	saveChoiceCount := len(s.SaveSlots) + 1
	if s.SelectedSaveIndex < 0 {
		s.SelectedSaveIndex = 0
	}
	if s.SelectedSaveIndex >= saveChoiceCount {
		s.SelectedSaveIndex = saveChoiceCount - 1
	}
}

type startMenuResult struct {
	Action   startMenuAction
	Config   app.Config
	State    startMenuState
	SlotName string
}

type startMenuModel struct {
	config     app.Config
	scenarios  []domain.ScenarioID
	state      startMenuState
	cursor     int
	width      int
	height     int
	result     startMenuResult
	cancelled  bool
	statusLine string
}

func newStartMenuModel(cfg app.Config, state startMenuState) startMenuModel {
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
	state.clampSelections()

	status := state.StatusText
	if strings.TrimSpace(status) == "" {
		status = "Use up/down to move, left/right to change menu selections, and enter to confirm."
	}

	return startMenuModel{
		config:     normalized,
		scenarios:  availableScenarios,
		state:      state,
		statusLine: status,
		result: startMenuResult{
			Action: startMenuActionExit,
			Config: normalized,
			State:  state,
		},
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
			m.result = startMenuResult{
				Action: startMenuActionExit,
				Config: m.config,
				State:  m.state,
			}
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
	disabledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	lines := []string{
		titleStyle.Render("HerbieGo Start Menu"),
		subtitleStyle.Render("Start a new match, resume the current session, or browse SQL-backed save slots."),
		"",
	}

	for index, item := range m.items() {
		line := "  " + item.label
		if !item.enabled {
			line = disabledStyle.Render(line)
		} else if index == m.cursor {
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
		width = 112
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
	case menuItemLoadGame:
		m.shiftLoadSelection(delta)
	case menuItemSaveGame:
		m.shiftSaveSelection(delta)
	}
}

func (m *startMenuModel) activateSelected() bool {
	item := m.items()[m.cursor]
	if !item.enabled {
		m.statusLine = item.disabledHint
		return false
	}

	switch item.kind {
	case menuItemStartNewGame:
		m.result = startMenuResult{Action: startMenuActionStartNewGame, Config: m.config, State: m.state}
		return true
	case menuItemResumeGame:
		m.result = startMenuResult{Action: startMenuActionResumeGame, Config: m.config, State: m.state}
		return true
	case menuItemLoadGame:
		m.result = startMenuResult{Action: startMenuActionLoadGame, Config: m.config, State: m.state, SlotName: m.selectedLoadSlotName()}
		return true
	case menuItemSaveGame:
		m.result = startMenuResult{Action: startMenuActionSaveGame, Config: m.config, State: m.state, SlotName: m.selectedSaveSlotName()}
		return true
	case menuItemExit:
		m.result = startMenuResult{Action: startMenuActionExit, Config: m.config, State: m.state}
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

func (m *startMenuModel) shiftLoadSelection(delta int) {
	if len(m.state.SaveSlots) == 0 {
		m.statusLine = "No saved games are available yet."
		return
	}
	count := len(m.state.SaveSlots)
	m.state.SelectedLoadIndex = (m.state.SelectedLoadIndex + delta + count) % count
	slot := m.state.SaveSlots[m.state.SelectedLoadIndex]
	m.statusLine = fmt.Sprintf("Selected save slot %s at round %d.", slot.SlotName, slot.CurrentRound)
}

func (m *startMenuModel) shiftSaveSelection(delta int) {
	choiceCount := len(m.state.SaveSlots) + 1
	if choiceCount <= 0 {
		return
	}
	m.state.SelectedSaveIndex = (m.state.SelectedSaveIndex + delta + choiceCount) % choiceCount
	if slotName := m.selectedSaveSlotName(); slotName != "" {
		m.statusLine = fmt.Sprintf("Save target set to slot %s.", slotName)
		return
	}
	m.statusLine = "Save target set to a new slot."
}

func (m startMenuModel) selectedLoadSlotName() string {
	if len(m.state.SaveSlots) == 0 {
		return ""
	}
	return m.state.SaveSlots[m.state.SelectedLoadIndex].SlotName
}

func (m startMenuModel) selectedSaveSlotName() string {
	if len(m.state.SaveSlots) == 0 || m.state.SelectedSaveIndex >= len(m.state.SaveSlots) {
		return ""
	}
	return m.state.SaveSlots[m.state.SelectedSaveIndex].SlotName
}

type menuItemKind int

const (
	menuItemStartNewGame menuItemKind = iota
	menuItemResumeGame
	menuItemLoadGame
	menuItemSaveGame
	menuItemScenario
	menuItemRole
	menuItemExit
)

type menuItem struct {
	kind         menuItemKind
	title        string
	label        string
	roleID       domain.RoleID
	enabled      bool
	disabledHint string
}

func (m startMenuModel) items() []menuItem {
	items := []menuItem{
		{
			kind:    menuItemStartNewGame,
			title:   "Start New Game",
			label:   m.startGameLabel(),
			enabled: true,
		},
		{
			kind:         menuItemResumeGame,
			title:        "Resume Current Game",
			label:        m.resumeGameLabel(),
			enabled:      m.state.ActiveSession != nil,
			disabledHint: "No current session is available yet.",
		},
		{
			kind:         menuItemLoadGame,
			title:        "Load Saved Game",
			label:        m.loadGameLabel(),
			enabled:      m.state.StoreEnabled && len(m.state.SaveSlots) > 0,
			disabledHint: m.loadDisabledHint(),
		},
		{
			kind:         menuItemSaveGame,
			title:        "Save Current Game",
			label:        m.saveGameLabel(),
			enabled:      m.state.StoreEnabled && m.state.ActiveSession != nil,
			disabledHint: m.saveDisabledHint(),
		},
		{
			kind:    menuItemScenario,
			title:   "Scenario",
			label:   fmt.Sprintf("Scenario: %s", scenarioTitle(m.config.ScenarioID)),
			enabled: true,
		},
	}

	for _, roleID := range selectedScenarioRoster(m.config.ScenarioID) {
		roleCfg, ok := m.config.Roles[roleID]
		if !ok {
			roleCfg = defaultRoleConfig(m.config)
		}
		items = append(items, menuItem{
			kind:    menuItemRole,
			title:   displayRoleName(roleID),
			label:   fmt.Sprintf("%s: %s", displayRoleName(roleID), roleControlLabel(roleCfg)),
			roleID:  roleID,
			enabled: true,
		})
	}

	items = append(items, menuItem{
		kind:    menuItemExit,
		title:   "Exit",
		label:   "Exit",
		enabled: true,
	})
	return items
}

func (m startMenuModel) startGameLabel() string {
	if m.state.ActiveSession == nil {
		return "Start Game"
	}
	return "Start New Game"
}

func (m startMenuModel) resumeGameLabel() string {
	if m.state.ActiveSession == nil {
		return "Resume Current Game: unavailable"
	}
	state := m.state.ActiveSession.state
	return fmt.Sprintf("Resume Current Game: %s round %d cash %d", scenarioTitle(state.ScenarioID), state.CurrentRound, state.Plant.Cash)
}

func (m startMenuModel) loadGameLabel() string {
	if !m.state.StoreEnabled {
		return "Load Saved Game: requires -sqlite-db"
	}
	if len(m.state.SaveSlots) == 0 {
		return "Load Saved Game: no saved games"
	}
	slot := m.state.SaveSlots[m.state.SelectedLoadIndex]
	return fmt.Sprintf("Load Saved Game: %s | %s | round %d | cash %d", slot.SlotName, scenarioTitle(slot.ScenarioID), slot.CurrentRound, slot.Cash)
}

func (m startMenuModel) saveGameLabel() string {
	if !m.state.StoreEnabled {
		return "Save Current Game: requires -sqlite-db"
	}
	if m.state.ActiveSession == nil {
		return "Save Current Game: unavailable until a session exists"
	}
	if slotName := m.selectedSaveSlotName(); slotName != "" {
		return fmt.Sprintf("Save Current Game: overwrite slot %s", slotName)
	}
	return "Save Current Game: create new slot"
}

func (m startMenuModel) loadDisabledHint() string {
	if !m.state.StoreEnabled {
		return "Load saved game requires -sqlite-db."
	}
	return "No saved games are available yet."
}

func (m startMenuModel) saveDisabledHint() string {
	if !m.state.StoreEnabled {
		return "Save current game requires -sqlite-db."
	}
	return "Return from gameplay first so there is a current session to save."
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
