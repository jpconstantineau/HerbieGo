package app

import (
	"fmt"
	"slices"

	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

// Runtime contains the process dependencies created during startup.
type Runtime struct {
	Config       Config
	Random       ports.RandomSource
	Scenario     scenario.Definition
	InitialMatch domain.MatchState
}

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
	if err := cfg.Validate(); err != nil {
		return Runtime{}, fmt.Errorf("validate runtime config: %w", err)
	}

	starter := scenario.Default()
	return Runtime{
		Config:       cfg,
		Random:       seeded.New(cfg.Random.Seed),
		Scenario:     starter,
		InitialMatch: starter.InitialState("starter-match", runtimeRoles(cfg)),
	}, nil
}

// RoleSummaries returns a stable summary of role runtime assignments.
func (r Runtime) RoleSummaries() []string {
	summaries := make([]string, 0, len(r.Config.Roles))
	for _, roleID := range domain.CanonicalRoles() {
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

func runtimeRoles(cfg Config) []domain.RoleAssignment {
	roles := make([]domain.RoleAssignment, 0, len(cfg.Roles))
	for _, roleID := range domain.CanonicalRoles() {
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
