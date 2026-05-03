package app

import (
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

// Runtime contains the process dependencies created during startup.
type Runtime struct {
	Config       Config
	Logger       *slog.Logger
	Random       ports.RandomSource
	Scenario     scenario.Definition
	InitialMatch domain.MatchState
}

var runtimeTimeNow = time.Now

// Bootstrap loads startup configuration and constructs process dependencies.
func Bootstrap(options BootstrapOptions) (Runtime, error) {
	cfg, err := LoadConfig(options.ConfigPath)
	if err != nil {
		return Runtime{}, fmt.Errorf("load runtime config: %w", err)
	}

	catalog, err := LoadLLMCatalog(resolveLLMCatalogPath(options.ConfigPath, options.LLMCatalogPath))
	if err != nil {
		return Runtime{}, fmt.Errorf("load llm catalog: %w", err)
	}
	cfg.WithLLMCatalog(catalog)

	cfg = cfg.ApplyOverrides(options)
	return NewRuntime(cfg)
}

// NewRuntime validates runtime config and constructs startup dependencies.
func NewRuntime(cfg Config) (Runtime, error) {
	cfg.normalize()
	selected, ok := scenario.Lookup(cfg.ScenarioID)
	if !ok {
		return Runtime{}, fmt.Errorf("resolve scenario %q: scenario is not registered", cfg.ScenarioID)
	}
	if err := cfg.ValidateForRoles(selected.Setup.RoleRoster); err != nil {
		return Runtime{}, fmt.Errorf("validate runtime config: %w", err)
	}

	initialMatch := selected.InitialState(runtimeMatchID(cfg, selected.ID), runtimeRoles(cfg, selected))
	initialMatch.RoundFlow.AIRevealDelaySeconds = cfg.UI.AIRevealDelaySeconds

	return Runtime{
		Config:       cfg,
		Logger:       newProcessLogger(),
		Random:       seeded.New(cfg.Random.Seed),
		Scenario:     selected,
		InitialMatch: initialMatch,
	}, nil
}

func runtimeMatchID(cfg Config, scenarioID domain.ScenarioID) domain.MatchID {
	if cfg.MatchID != "" {
		return cfg.MatchID
	}

	return domain.MatchID(fmt.Sprintf(
		"%s-match-%d-%d",
		scenarioID,
		cfg.Random.Seed,
		runtimeTimeNow().UTC().UnixNano(),
	))
}

// RoleSummaries returns a stable summary of role runtime assignments.
func (r Runtime) RoleSummaries() []string {
	summaries := make([]string, 0, len(r.Config.Roles))
	for _, roleID := range r.Scenario.Setup.RoleRoster {
		roleCfg, ok := r.Config.Roles[roleID]
		if !ok {
			continue
		}

		summary := fmt.Sprintf("%s=%s", roleID, roleCfg.Kind)
		if roleCfg.Provider != "" || roleCfg.Model != "" {
			summary = fmt.Sprintf("%s[%s:%s]", summary, roleCfg.Provider, roleCfg.Model)
		}

		summaries = append(summaries, summary)
	}

	slices.Sort(summaries)
	return summaries
}

func runtimeRoles(cfg Config, selected scenario.Definition) []domain.RoleAssignment {
	roles := make([]domain.RoleAssignment, 0, len(cfg.Roles))
	for _, roleID := range selected.Setup.RoleRoster {
		roleCfg, ok := cfg.Roles[roleID]
		if !ok {
			continue
		}

		roles = append(roles, domain.RoleAssignment{
			RoleID:    roleID,
			PlayerID:  fmt.Sprintf("%s-player", roleID),
			IsHuman:   roleCfg.Kind == PlayerKindHuman,
			Provider:  string(roleCfg.Provider),
			ModelName: roleCfg.Model,
		})
	}
	return roles
}
