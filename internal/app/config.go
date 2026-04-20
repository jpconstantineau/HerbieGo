package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	defaultConfigPath  = "herbiego.yaml"
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
	Environment  string                       `yaml:"environment"`
	Random       RandomConfig                 `yaml:"random"`
	HumanPlayers int                          `yaml:"human_players"`
	Roles        map[domain.RoleID]RoleConfig `yaml:"-"`
	RoleConfigs  []RoleConfigFile             `yaml:"roles"`
}

// RandomConfig controls deterministic randomness for the process.
type RandomConfig struct {
	Seed uint64 `yaml:"seed"`
}

// RoleConfig defines runtime settings for a single role.
type RoleConfig struct {
	Kind     PlayerKind
	Provider ProviderName
	Model    string
}

// RoleConfigFile is the editable YAML representation for a role's runtime options.
type RoleConfigFile struct {
	RoleID   domain.RoleID `yaml:"role_id"`
	Provider ProviderName  `yaml:"provider"`
	Model    string        `yaml:"model"`
}

// BootstrapOptions controls startup loading behavior.
type BootstrapOptions struct {
	ConfigPath           string
	HumanPlayersOverride *int
	SeedOverride         *uint64
}

// LoadConfig reads runtime configuration from a YAML file.
func LoadConfig(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		path = defaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	cfg := Config{
		Environment: defaultEnvironment,
		Random: RandomConfig{
			Seed: defaultRandomSeed,
		},
		HumanPlayers: 1,
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	cfg.normalize()
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config file %q: %w", path, err)
	}

	return cfg, nil
}

// ApplyOverrides applies runtime-only startup overrides after config loading.
func (c Config) ApplyOverrides(options BootstrapOptions) Config {
	if options.HumanPlayersOverride != nil {
		c.HumanPlayers = *options.HumanPlayersOverride
	}

	if options.SeedOverride != nil {
		c.Random.Seed = *options.SeedOverride
	}

	c.normalize()
	return c
}

// Validate checks that the configuration is internally consistent.
func (c *Config) Validate() error {
	c.normalize()

	var errs []error

	if strings.TrimSpace(c.Environment) == "" {
		errs = append(errs, errors.New("environment must not be empty"))
	}

	expectedRoles := domain.CanonicalRoles()
	if len(c.Roles) != len(expectedRoles) {
		errs = append(errs, fmt.Errorf("roles must include exactly %d canonical roles", len(expectedRoles)))
	}

	if c.HumanPlayers < 0 || c.HumanPlayers > len(expectedRoles) {
		errs = append(errs, fmt.Errorf("human_players must be between 0 and %d", len(expectedRoles)))
	}

	for _, roleID := range expectedRoles {
		roleCfg, ok := c.Roles[roleID]
		if !ok {
			errs = append(errs, fmt.Errorf("role configuration missing for %s", roleID))
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

func (c *Config) normalize() {
	if strings.TrimSpace(c.Environment) == "" {
		c.Environment = defaultEnvironment
	}

	if c.Random.Seed == 0 {
		c.Random.Seed = defaultRandomSeed
	}

	if c.HumanPlayers == 0 && len(c.RoleConfigs) == 0 && len(c.Roles) == 0 {
		c.HumanPlayers = 1
	}

	if c.Roles == nil {
		c.Roles = make(map[domain.RoleID]RoleConfig, len(c.RoleConfigs))
	}

	if len(c.RoleConfigs) > 0 {
		c.Roles = make(map[domain.RoleID]RoleConfig, len(c.RoleConfigs))
		for index, roleFile := range c.RoleConfigs {
			roleID := roleFile.RoleID
			roleCfg := RoleConfig{
				Kind:     PlayerKindAI,
				Provider: ProviderName(strings.ToLower(strings.TrimSpace(string(roleFile.Provider)))),
				Model:    strings.TrimSpace(roleFile.Model),
			}

			if index < c.HumanPlayers {
				roleCfg.Kind = PlayerKindHuman
			}

			c.Roles[roleID] = roleCfg
		}
	}
}

func validateRoleConfig(roleID domain.RoleID, roleCfg RoleConfig) error {
	var errs []error

	if roleCfg.Kind == "" {
		errs = append(errs, fmt.Errorf("%s kind must not be empty", roleID))
	}

	switch roleCfg.Kind {
	case PlayerKindHuman:
		// Human-controlled roles still carry minimal AI mapping so a CLI override can enable AI-only tests.
		if roleCfg.Provider == "" {
			errs = append(errs, fmt.Errorf("%s provider must not be empty", roleID))
		}
		if roleCfg.Model == "" {
			errs = append(errs, fmt.Errorf("%s model must not be empty", roleID))
		}
	case PlayerKindAI:
		if roleCfg.Provider == "" {
			errs = append(errs, fmt.Errorf("%s provider must not be empty", roleID))
		}
		if roleCfg.Model == "" {
			errs = append(errs, fmt.Errorf("%s model must not be empty", roleID))
		}
	default:
		errs = append(errs, fmt.Errorf("%s kind must be %q or %q", roleID, PlayerKindHuman, PlayerKindAI))
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}
