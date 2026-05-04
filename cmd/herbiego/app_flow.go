package main

import (
	"context"
	"fmt"

	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func runApplication(ctx context.Context, baseRuntime app.Runtime, resumedState domain.MatchState, hasResumedState bool, store persistentStore, rounds int, persistAIDebug bool) error {
	if err := runSplashScreen(); err != nil {
		return fmt.Errorf("run splash screen: %w", err)
	}

	menuConfig := cloneConfig(baseRuntime.Config)
	pendingState := resumedState.Clone()

	for {
		if hasResumedState {
			exitIntent, err := runLiveGameplay(ctx, baseRuntime, pendingState, store, rounds, persistAIDebug)
			if err != nil {
				return err
			}
			if exitIntent != gameplayExitToMenu {
				return nil
			}
			hasResumedState = false
			continue
		}

		action, updatedConfig, err := runStartMenu(menuConfig)
		if err != nil {
			return fmt.Errorf("run start menu: %w", err)
		}
		menuConfig = updatedConfig
		if action == startMenuActionExit {
			return nil
		}

		runtime, err := app.NewRuntime(runtimeConfigFromMenu(menuConfig))
		if err != nil {
			return fmt.Errorf("build runtime from start menu: %w", err)
		}

		exitIntent, err := runLiveGameplay(ctx, runtime, runtime.InitialMatch, store, rounds, persistAIDebug)
		if err != nil {
			return err
		}
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
