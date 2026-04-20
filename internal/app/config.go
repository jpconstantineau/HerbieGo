package app

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

const (
	defaultEnvironment = "local"
	defaultRandomSeed  = uint64(1)
)

// PlayerKind determines how a role is controlled at runtime.
type PlayerKind string

const (
	PlayerKindHuman PlayerKind = "human"
	PlayerKindAI    PlayerKind = "ai"
)

// ProviderName identifies the model backend used for an AI-controlled role.
type ProviderName string

const (
	ProviderOllama     ProviderName = "ollama"
	ProviderOpenRouter ProviderName = "openrouter"
)

// Config holds process-level runtime configuration.
type Config struct {
	Environment string
	Random      RandomConfig
	Roles       map[domain.RoleID]RoleConfig
}

// RandomConfig controls deterministic randomness for the process.
type RandomConfig struct {
	Seed uint64
}

// RoleConfig defines runtime settings for a single role.
type RoleConfig struct {
	Kind     PlayerKind
	Provider ProviderName
	Model    string
}

// LoadConfigFromEnv reads runtime configuration from environment variables.
func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		Environment: strings.TrimSpace(os.Getenv("HERBIEGO_ENV")),
		Random: RandomConfig{
			Seed: defaultRandomSeed,
		},
		Roles: make(map[domain.RoleID]RoleConfig, len(domain.CanonicalRoles())),
	}

	if cfg.Environment == "" {
		cfg.Environment = defaultEnvironment
	}

	var errs []error

	if rawSeed := strings.TrimSpace(os.Getenv("HERBIEGO_RANDOM_SEED")); rawSeed != "" {
		seed, err := strconv.ParseUint(rawSeed, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("HERBIEGO_RANDOM_SEED must be an unsigned integer: %w", err))
		} else {
			cfg.Random.Seed = seed
		}
	}

	for _, roleID := range domain.CanonicalRoles() {
		roleCfg := loadRoleConfig(roleID)
		cfg.Roles[roleID] = roleCfg
	}

	if err := cfg.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

// Validate checks that the configuration is internally consistent.
func (c Config) Validate() error {
	var errs []error

	if strings.TrimSpace(c.Environment) == "" {
		errs = append(errs, errors.New("HERBIEGO_ENV must not be empty"))
	}

	expectedRoles := domain.CanonicalRoles()
	if len(c.Roles) != len(expectedRoles) {
		errs = append(errs, fmt.Errorf("runtime roles must include exactly %d canonical roles", len(expectedRoles)))
	}

	for _, roleID := range expectedRoles {
		roleCfg, ok := c.Roles[roleID]
		if !ok {
			errs = append(errs, fmt.Errorf("runtime role configuration missing for %s", roleID))
			continue
		}

		if err := validateRoleConfig(roleID, roleCfg); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

func loadRoleConfig(roleID domain.RoleID) RoleConfig {
	prefix := envRolePrefix(roleID)

	rawKind := strings.TrimSpace(os.Getenv(prefix + "_KIND"))
	if rawKind == "" {
		rawKind = string(PlayerKindHuman)
	}

	roleCfg := RoleConfig{
		Kind:  PlayerKind(strings.ToLower(rawKind)),
		Model: strings.TrimSpace(os.Getenv(prefix + "_MODEL")),
	}

	if provider := strings.TrimSpace(os.Getenv(prefix + "_PROVIDER")); provider != "" {
		roleCfg.Provider = ProviderName(strings.ToLower(provider))
	}

	return roleCfg
}

func validateRoleConfig(roleID domain.RoleID, roleCfg RoleConfig) error {
	var errs []error
	prefix := envRolePrefix(roleID)

	switch roleCfg.Kind {
	case PlayerKindHuman:
		if roleCfg.Provider != "" {
			errs = append(errs, fmt.Errorf("%s_PROVIDER must be unset when %s_KIND=%q", prefix, prefix, PlayerKindHuman))
		}
		if roleCfg.Model != "" {
			errs = append(errs, fmt.Errorf("%s_MODEL must be unset when %s_KIND=%q", prefix, prefix, PlayerKindHuman))
		}
	case PlayerKindAI:
		if !slices.Contains([]ProviderName{ProviderOllama, ProviderOpenRouter}, roleCfg.Provider) {
			errs = append(errs, fmt.Errorf("%s_PROVIDER must be one of %q or %q when %s_KIND=%q", prefix, ProviderOllama, ProviderOpenRouter, prefix, PlayerKindAI))
		}
		if roleCfg.Model == "" {
			errs = append(errs, fmt.Errorf("%s_MODEL is required when %s_KIND=%q", prefix, prefix, PlayerKindAI))
		}
	default:
		errs = append(errs, fmt.Errorf("%s_KIND must be %q or %q", prefix, PlayerKindHuman, PlayerKindAI))
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

func envRolePrefix(roleID domain.RoleID) string {
	return "HERBIEGO_ROLE_" + strings.ToUpper(string(roleID))
}
