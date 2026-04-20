package app

import (
	"fmt"
	"slices"

	"github.com/jpconstantineau/herbiego/internal/adapters/random/seeded"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

// Runtime contains the process dependencies created during startup.
type Runtime struct {
	Config Config
	Random ports.RandomSource
}

// BootstrapFromEnv loads runtime configuration and constructs process dependencies.
func BootstrapFromEnv() (Runtime, error) {
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		return Runtime{}, fmt.Errorf("load runtime config: %w", err)
	}

	return NewRuntime(cfg)
}

// NewRuntime validates runtime config and constructs startup dependencies.
func NewRuntime(cfg Config) (Runtime, error) {
	if err := cfg.Validate(); err != nil {
		return Runtime{}, fmt.Errorf("validate runtime config: %w", err)
	}

	return Runtime{
		Config: cfg,
		Random: seeded.New(cfg.Random.Seed),
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
		if roleCfg.Kind == PlayerKindAI {
			summary = fmt.Sprintf("%s[%s:%s]", summary, roleCfg.Provider, roleCfg.Model)
		}

		summaries = append(summaries, summary)
	}

	slices.Sort(summaries)
	return summaries
}
