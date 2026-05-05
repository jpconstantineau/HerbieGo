package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/adapters/tui"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/engine"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type activeSession struct {
	runtime app.Runtime
	state   domain.MatchState
}

type shellScreen int

const (
	screenSplash shellScreen = iota
	screenMenu
	screenGameplay
)

type gameplayRunnerDoneMsg struct {
	err error
}

type gameplayEventsClosedMsg struct{}

type gameplayScreen struct {
	model      tui.Model
	controller *liveGameplayController
	cancel     context.CancelFunc
	events     <-chan tea.Msg
}

type appShellModel struct {
	ctx            context.Context
	baseRuntime    app.Runtime
	store          persistentStore
	rounds         int
	persistAIDebug bool
	autoResume     bool
	width          int
	height         int

	screen     shellScreen
	splash     splashModel
	menu       startMenuModel
	gameplay   *gameplayScreen
	menuConfig app.Config
	menuState  startMenuState
	current    *activeSession

	fatalErr error
}

func runApplication(ctx context.Context, baseRuntime app.Runtime, resumedState domain.MatchState, hasResumedState bool, store persistentStore, rounds int, persistAIDebug bool) error {
	model, err := newAppShellModel(ctx, baseRuntime, resumedState, hasResumedState, store, rounds, persistAIDebug)
	if err != nil {
		return err
	}

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)
	finalModel, err := program.Run()
	if err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return fmt.Errorf("run app shell: %w", err)
	}

	switch shell := finalModel.(type) {
	case appShellModel:
		return shell.fatalErr
	case *appShellModel:
		return shell.fatalErr
	default:
		return nil
	}
}

func newAppShellModel(ctx context.Context, baseRuntime app.Runtime, resumedState domain.MatchState, hasResumedState bool, store persistentStore, rounds int, persistAIDebug bool) (appShellModel, error) {
	menuConfig := cloneConfig(baseRuntime.Config)
	var current *activeSession
	menuState := startMenuState{
		StoreEnabled: store != nil,
	}
	if hasResumedState {
		current = &activeSession{runtime: baseRuntime, state: resumedState.Clone()}
	}
	if err := refreshMenuState(store, current, &menuState); err != nil {
		return appShellModel{}, err
	}

	return appShellModel{
		ctx:            ctx,
		baseRuntime:    baseRuntime,
		store:          store,
		rounds:         rounds,
		persistAIDebug: persistAIDebug,
		autoResume:     hasResumedState,
		screen:         screenSplash,
		splash:         splashModel{},
		menuConfig:     menuConfig,
		menuState:      menuState,
		current:        current,
	}, nil
}

func (m appShellModel) Init() tea.Cmd {
	return m.splash.Init()
}

func (m appShellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if typed, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = typed.Width
		m.height = typed.Height
	}

	switch m.screen {
	case screenSplash:
		return m.updateSplash(msg)
	case screenMenu:
		return m.updateMenu(msg)
	case screenGameplay:
		return m.updateGameplay(msg)
	default:
		return m, nil
	}
}

func (m appShellModel) View() string {
	switch m.screen {
	case screenSplash:
		return m.splash.View()
	case screenGameplay:
		if m.gameplay != nil {
			return m.gameplay.model.View()
		}
		return ""
	default:
		return m.menu.View()
	}
}

func (m appShellModel) updateSplash(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.splash.Update(msg)
	m.splash = next.(splashModel)
	if isQuitCmd(cmd) {
		return m.finishSplash()
	}
	return m, cmd
}

func (m appShellModel) finishSplash() (tea.Model, tea.Cmd) {
	if m.autoResume && m.current != nil {
		m.autoResume = false
		return m.enterGameplay(m.current)
	}
	m.screen = screenMenu
	m.menu = newStartMenuModel(m.menuConfig, m.menuState)
	m.applyWindowSizeToMenu()
	return m, m.menu.Init()
}

func (m appShellModel) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.menu.Update(msg)
	m.menu = next.(startMenuModel)
	if !isQuitCmd(cmd) {
		return m, cmd
	}

	result := m.menu.result
	m.menuConfig = result.Config
	m.menuState = result.State

	switch result.Action {
	case startMenuActionExit:
		return m, tea.Quit
	case startMenuActionSaveGame:
		if m.current == nil || m.store == nil {
			m.menuState.StatusText = "Saving requires an active session and SQLite persistence."
			m.menu = newStartMenuModel(m.menuConfig, m.menuState)
			return m, nil
		}
		slotName := result.SlotName
		if strings.TrimSpace(slotName) == "" {
			slotName = defaultSaveSlotName(m.current.state)
		}
		summary, err := m.store.SaveSlot(slotName, m.current.state.MatchID)
		if err != nil {
			m.menuState.StatusText = fmt.Sprintf("Save failed: %v", err)
		} else {
			m.menuState.StatusText = fmt.Sprintf("Saved round %d to slot %s.", summary.CurrentRound, summary.SlotName)
		}
		if err := refreshMenuState(m.store, m.current, &m.menuState); err != nil {
			m.fatalErr = err
			return m, tea.Quit
		}
		m.menu = newStartMenuModel(m.menuConfig, m.menuState)
		return m, nil
	case startMenuActionLoadGame:
		if m.store == nil {
			m.menuState.StatusText = "Load saved game requires -sqlite-db."
			m.menu = newStartMenuModel(m.menuConfig, m.menuState)
			return m, nil
		}
		state, summary, err := m.store.LoadSaveSlot(result.SlotName)
		if err != nil {
			m.menuState.StatusText = fmt.Sprintf("Load failed: %v", err)
			m.menu = newStartMenuModel(m.menuConfig, m.menuState)
			return m, nil
		}
		runtime, err := runtimeForLoadedState(m.menuConfig, state)
		if err != nil {
			m.fatalErr = fmt.Errorf("build runtime for save slot %q: %w", summary.SlotName, err)
			return m, tea.Quit
		}
		m.current = &activeSession{runtime: runtime, state: state.Clone()}
		return m.enterGameplay(m.current)
	case startMenuActionResumeGame:
		if m.current == nil {
			m.menuState.StatusText = "No current session is available to resume."
			m.menu = newStartMenuModel(m.menuConfig, m.menuState)
			return m, nil
		}
		return m.enterGameplay(m.current)
	case startMenuActionStartNewGame:
		runtime, err := app.NewRuntime(runtimeConfigFromMenu(m.menuConfig))
		if err != nil {
			m.menuState.StatusText = fmt.Sprintf("Start failed: %v", err)
			m.menu = newStartMenuModel(m.menuConfig, m.menuState)
			return m, nil
		}
		m.current = &activeSession{runtime: runtime, state: runtime.InitialMatch.Clone()}
		return m.enterGameplay(m.current)
	default:
		return m, nil
	}
}

func (m appShellModel) updateGameplay(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case gameplayRunnerDoneMsg:
		if m.gameplay == nil {
			return m, nil
		}
		if typed.err != nil && !errors.Is(typed.err, context.Canceled) {
			m.fatalErr = typed.err
			return m, tea.Quit
		}
		if m.current != nil {
			m.current.state = m.gameplay.controller.Snapshot()
		}
		if m.gameplay != nil && m.gameplay.events != nil {
			return m, waitForGameplayEvent(m.gameplay.events)
		}
		return m, nil
	case gameplayEventsClosedMsg:
		return m, nil
	}

	if m.gameplay == nil {
		return m, nil
	}

	next, cmd := m.gameplay.model.Update(msg)
	m.gameplay.model = next.(tui.Model)
	if !isQuitCmd(cmd) {
		return m, cmd
	}

	switch m.gameplay.model.ExitIntent() {
	case tui.ExitIntentReturnToMenu:
		if m.current != nil {
			m.current.state = m.gameplay.controller.Snapshot()
			m.menuConfig = cloneConfig(m.current.runtime.Config)
			m.menuState.StatusText = currentSessionStatus(m.current.state)
		}
		if m.gameplay.cancel != nil {
			m.gameplay.cancel()
		}
		m.gameplay = nil
		if err := refreshMenuState(m.store, m.current, &m.menuState); err != nil {
			m.fatalErr = err
			return m, tea.Quit
		}
		m.screen = screenMenu
		m.menu = newStartMenuModel(m.menuConfig, m.menuState)
		m.applyWindowSizeToMenu()
		return m, m.menu.Init()
	default:
		if m.gameplay.cancel != nil {
			m.gameplay.cancel()
		}
		m.gameplay = nil
		return m, tea.Quit
	}
}

func (m *appShellModel) enterGameplay(session *activeSession) (tea.Model, tea.Cmd) {
	screen, cmd, err := newGameplayScreen(m.ctx, session.runtime, session.state, m.store, m.rounds, m.persistAIDebug)
	if err != nil {
		m.fatalErr = err
		return m, tea.Quit
	}
	m.screen = screenGameplay
	m.gameplay = screen
	m.applyWindowSizeToGameplay()
	return m, cmd
}

func (m *appShellModel) applyWindowSizeToMenu() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	updated, _ := m.menu.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	m.menu = updated.(startMenuModel)
}

func (m *appShellModel) applyWindowSizeToGameplay() {
	if m.gameplay == nil || m.width <= 0 || m.height <= 0 {
		return
	}
	updated, _ := m.gameplay.model.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	m.gameplay.model = updated.(tui.Model)
}

func newGameplayScreen(ctx context.Context, runtime app.Runtime, initialState domain.MatchState, store persistentStore, rounds int, persistAIDebug bool) (*gameplayScreen, tea.Cmd, error) {
	definition, err := resolveScenarioForMatch(initialState)
	if err != nil {
		return nil, nil, err
	}
	liveLogger := app.NewDiscardLogger()

	stateSnapshots := []domain.MatchState{initialState.Clone()}
	if store != nil {
		persistedSnapshots, err := store.StateSnapshots(initialState.MatchID)
		if err != nil {
			return nil, nil, fmt.Errorf("load persisted state snapshots: %w", err)
		}
		stateSnapshots = persistedSnapshots
	}
	controller := newLiveGameplayController(initialState, stateSnapshots)
	players, debugLog, err := buildPlayersWithHumanSubmit(runtime.Config, definition, initialState, controller.SubmitRound, liveLogger)
	if err != nil {
		return nil, nil, fmt.Errorf("player setup: %w", err)
	}
	if store != nil && persistAIDebug {
		configureAICallPersistence(debugLog, liveLogger, store, initialState.MatchID)
	}
	debugSource := tui.DebugSource(debugLog)
	if store != nil {
		debugRecords, err := store.AICallRecords(initialState.MatchID)
		if err != nil {
			return nil, nil, fmt.Errorf("load persisted ai traces: %w", err)
		}
		debugSource = combinedDebugSource{
			live:      debugLog,
			persisted: debugRecords,
		}
	}

	gameplayModel := tui.NewModelWithSubmitDebugAndQuitBehavior(
		definition,
		controller,
		controller.Submit,
		debugSource,
		tui.QuitBehaviorReturnToMenu,
	)

	runner := app.MatchRunner{
		Collector: app.RoundCollector{Players: players, Logger: liveLogger},
		Resolver:  engine.NewResolver(definition.ResolverOptions()),
		Random:    runtime.Random,
		Store:     store,
		OnState:   controller.Publish,
		Logger:    liveLogger,
	}

	runCtx, cancel := context.WithCancel(ctx)
	events := make(chan tea.Msg, 4)
	go func() {
		defer controller.Close()
		defer close(events)

		final, _, err := runner.Play(runCtx, initialState, rounds)
		switch {
		case err == nil:
			events <- tui.StatusMsg{
				Text: fmt.Sprintf(
					"Match complete after round %d. Final cash %d, debt %d, backlog %d, profit %d. Inspect results and press q to return to the menu.",
					final.CurrentRound-1,
					final.Plant.Cash,
					final.Plant.Debt,
					len(final.Plant.Backlog),
					final.Metrics.RoundProfit,
				),
			}
		case errors.Is(err, context.Canceled):
		default:
			events <- tui.StatusMsg{Text: fmt.Sprintf("Match failed: %v. Press q to return to the menu.", err)}
		}

		events <- gameplayRunnerDoneMsg{err: err}
	}()

	screen := &gameplayScreen{
		model:      gameplayModel,
		controller: controller,
		cancel:     cancel,
		events:     events,
	}
	return screen, tea.Batch(gameplayModel.Init(), waitForGameplayEvent(events)), nil
}

func waitForGameplayEvent(events <-chan tea.Msg) tea.Cmd {
	if events == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return gameplayEventsClosedMsg{}
		}
		return msg
	}
}

func refreshMenuState(store persistentStore, current *activeSession, state *startMenuState) error {
	if state == nil {
		return nil
	}
	state.ActiveSession = current
	if current != nil && strings.TrimSpace(state.StatusText) == "" {
		state.StatusText = currentSessionStatus(current.state)
	}
	if store == nil {
		state.StoreEnabled = false
		state.SaveSlots = nil
		state.clampSelections()
		return nil
	}

	slots, err := store.ListSaveSlots()
	if err != nil {
		return fmt.Errorf("list save slots: %w", err)
	}
	state.StoreEnabled = true
	state.SaveSlots = slots
	state.clampSelections()
	return nil
}

func runtimeConfigFromMenu(cfg app.Config) app.Config {
	cloned := cloneConfig(cfg)
	definition, ok := scenario.Lookup(cloned.ScenarioID)
	if !ok {
		return cloned
	}

	filteredRoles := make(map[domain.RoleID]app.RoleConfig, len(definition.Setup.RoleRoster))
	for _, roleID := range definition.Setup.RoleRoster {
		roleCfg, ok := cloned.Roles[roleID]
		if !ok {
			roleCfg = defaultRoleConfig(cloned)
		}
		filteredRoles[roleID] = roleCfg
	}

	cloned.Roles = filteredRoles
	cloned.RoleConfigs = nil
	return cloned
}

func runtimeForLoadedState(cfg app.Config, state domain.MatchState) (app.Runtime, error) {
	runtimeCfg := cloneConfig(cfg)
	runtimeCfg.ScenarioID = state.ScenarioID
	runtimeCfg.Roles = make(map[domain.RoleID]app.RoleConfig, len(state.Roles))
	runtimeCfg.RoleConfigs = nil
	for _, assignment := range state.Roles {
		roleCfg := defaultRoleConfig(runtimeCfg)
		if assignment.IsHuman {
			roleCfg.Kind = app.PlayerKindHuman
		}
		if strings.TrimSpace(assignment.Provider) != "" {
			roleCfg.Provider = app.ProviderName(assignment.Provider)
		}
		if strings.TrimSpace(assignment.ModelName) != "" {
			roleCfg.Model = assignment.ModelName
		}
		runtimeCfg.Roles[assignment.RoleID] = roleCfg
	}

	runtime, err := app.NewRuntime(runtimeCfg)
	if err != nil {
		return app.Runtime{}, err
	}
	runtime.InitialMatch = state.Clone()
	runtime.Random = seeded.New(runtimeCfg.Random.Seed)
	return runtime, nil
}

func defaultSaveSlotName(state domain.MatchState) string {
	return fmt.Sprintf("%s-r%d-%s", state.ScenarioID, state.CurrentRound, time.Now().UTC().Format("20060102-150405"))
}

func currentSessionStatus(state domain.MatchState) string {
	return fmt.Sprintf("Current session: %s round %d cash %d.", scenarioTitle(state.ScenarioID), state.CurrentRound, state.Plant.Cash)
}

func isQuitCmd(cmd tea.Cmd) bool {
	return fmt.Sprintf("%p", cmd) == fmt.Sprintf("%p", tea.Quit)
}
