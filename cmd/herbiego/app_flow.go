package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

type activeSession struct {
	runtime app.Runtime
	state   domain.MatchState
}

func runApplication(ctx context.Context, baseRuntime app.Runtime, resumedState domain.MatchState, hasResumedState bool, store persistentStore, rounds int, persistAIDebug bool) error {
	if err := runSplashScreen(); err != nil {
		return fmt.Errorf("run splash screen: %w", err)
	}

	menuConfig := cloneConfig(baseRuntime.Config)
	var current *activeSession
	menuState := startMenuState{
		StoreEnabled: store != nil,
	}
	if hasResumedState {
		current = &activeSession{runtime: baseRuntime, state: resumedState.Clone()}
	}

	for {
		if store != nil {
			slots, err := store.ListSaveSlots()
			if err != nil {
				return fmt.Errorf("list save slots: %w", err)
			}
			menuState.SaveSlots = slots
			menuState.clampSelections()
		}

		if hasResumedState && current != nil {
			exitIntent, latest, err := runLiveGameplay(ctx, current.runtime, current.state, store, rounds, persistAIDebug)
			if err != nil {
				return err
			}
			current.state = latest.Clone()
			menuConfig = cloneConfig(current.runtime.Config)
			hasResumedState = false
			if exitIntent != gameplayExitToMenu {
				return nil
			}
			menuState.ActiveSession = current
			menuState.StatusText = currentSessionStatus(current.state)
			continue
		}

		menuState.ActiveSession = current
		result, err := runStartMenu(menuConfig, menuState)
		if err != nil {
			return fmt.Errorf("run start menu: %w", err)
		}
		menuConfig = result.Config
		menuState = result.State

		switch result.Action {
		case startMenuActionExit:
			return nil
		case startMenuActionSaveGame:
			if current == nil || store == nil {
				menuState.StatusText = "Saving requires an active session and SQLite persistence."
				continue
			}

			slotName := result.SlotName
			if strings.TrimSpace(slotName) == "" {
				slotName = defaultSaveSlotName(current.state)
			}
			summary, err := store.SaveSlot(slotName, current.state.MatchID)
			if err != nil {
				menuState.StatusText = fmt.Sprintf("Save failed: %v", err)
				continue
			}
			menuState.StatusText = fmt.Sprintf("Saved round %d to slot %s.", summary.CurrentRound, summary.SlotName)
			continue
		case startMenuActionLoadGame:
			if store == nil {
				menuState.StatusText = "Load saved game requires -sqlite-db."
				continue
			}
			state, summary, err := store.LoadSaveSlot(result.SlotName)
			if err != nil {
				menuState.StatusText = fmt.Sprintf("Load failed: %v", err)
				continue
			}
			runtime, err := runtimeForLoadedState(menuConfig, state)
			if err != nil {
				return fmt.Errorf("build runtime for save slot %q: %w", summary.SlotName, err)
			}
			current = &activeSession{runtime: runtime, state: state.Clone()}
		case startMenuActionResumeGame:
			if current == nil {
				menuState.StatusText = "No current session is available to resume."
				continue
			}
		case startMenuActionStartNewGame:
			runtime, err := app.NewRuntime(runtimeConfigFromMenu(menuConfig))
			if err != nil {
				menuState.StatusText = fmt.Sprintf("Start failed: %v", err)
				continue
			}
			current = &activeSession{runtime: runtime, state: runtime.InitialMatch.Clone()}
		default:
			continue
		}

		if current == nil {
			continue
		}

		exitIntent, latest, err := runLiveGameplay(ctx, current.runtime, current.state, store, rounds, persistAIDebug)
		if err != nil {
			return err
		}
		current.state = latest.Clone()
		menuConfig = cloneConfig(current.runtime.Config)
		menuState.ActiveSession = current
		menuState.StatusText = currentSessionStatus(current.state)
		if exitIntent != gameplayExitToMenu {
			return nil
		}
	}
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
