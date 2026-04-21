package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	defaultConfigPath     = "herbiego.yaml"
	defaultLLMCatalogPath = "llm.yaml"
	defaultEnvironment    = "local"
	defaultRandomSeed     = uint64(1)
)

var preferredHumanRoleOrder = []domain.RoleID{
	domain.RoleProductionManager,
	domain.RoleProcurementManager,
	domain.RoleSalesManager,
	domain.RoleFinanceController,
}

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
	LLMCatalog   LLMCatalog                   `yaml:"-"`
}

// RandomConfig controls deterministic randomness for the process.
type RandomConfig struct {
	Seed uint64 `yaml:"seed"`
}

// RoleConfig defines runtime settings for a single role.
type RoleConfig struct {
	Kind       PlayerKind
	Provider   ProviderName
	Model      string
	URL        string
	APISDKType APISDKType
	APIKey     string
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
	LLMCatalogPath       string
	HumanPlayersOverride *int
	SeedOverride         *uint64
}

// APISDKType identifies the wire protocol family an external provider uses.
type APISDKType string

const (
	APISDKTypeOllama APISDKType = "ollama"
	APISDKTypeOpenAI APISDKType = "openai"
)

// LLMCatalog stores named provider/model connection metadata loaded from llm.yaml.
type LLMCatalog struct {
	Entries []LLMCatalogEntry `yaml:"models"`
}

// LLMCatalogEntry is one editable catalog entry for a provider/model pair.
type LLMCatalogEntry struct {
	Provider   ProviderName `yaml:"provider_name"`
	Model      string       `yaml:"model_name"`
	URL        string       `yaml:"url"`
	APISDKType APISDKType   `yaml:"api_sdk_type"`
	APIKey     string       `yaml:"api_key"`
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

// LoadLLMCatalog reads provider/model connection metadata from a YAML file.
func LoadLLMCatalog(path string) (LLMCatalog, error) {
	if strings.TrimSpace(path) == "" {
		path = defaultLLMCatalogPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return LLMCatalog{}, fmt.Errorf("read LLM catalog file %q: %w", path, err)
	}

	var catalog LLMCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return LLMCatalog{}, fmt.Errorf("parse LLM catalog file %q: %w", path, err)
	}

	catalog.normalize()
	if err := catalog.Validate(); err != nil {
		return LLMCatalog{}, fmt.Errorf("validate LLM catalog file %q: %w", path, err)
	}

	return catalog, nil
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

	if err := c.LLMCatalog.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("llm catalog: %w", err))
	}

	for _, roleID := range expectedRoles {
		roleCfg, ok := c.Roles[roleID]
		if !ok {
			errs = append(errs, fmt.Errorf("role configuration missing for %s", roleID))
			continue
		}

		if err := validateRoleConfig(roleID, roleCfg, len(c.LLMCatalog.Entries) > 0); err != nil {
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
		for _, roleFile := range c.RoleConfigs {
			roleID := roleFile.RoleID
			roleCfg := RoleConfig{
				Kind:     PlayerKindAI,
				Provider: ProviderName(strings.ToLower(strings.TrimSpace(string(roleFile.Provider)))),
				Model:    strings.TrimSpace(roleFile.Model),
			}
			if entry, ok := c.LLMCatalog.Lookup(roleCfg.Provider, roleCfg.Model); ok {
				roleCfg.URL = entry.URL
				roleCfg.APISDKType = entry.APISDKType
				roleCfg.APIKey = entry.APIKey
			}
			c.Roles[roleID] = roleCfg
		}

		for _, roleID := range selectedHumanRoles(c.HumanPlayers) {
			roleCfg, ok := c.Roles[roleID]
			if !ok {
				continue
			}

			roleCfg.Kind = PlayerKindHuman
			c.Roles[roleID] = roleCfg
		}
	}
}

func selectedHumanRoles(humanPlayers int) []domain.RoleID {
	if humanPlayers <= 0 {
		return nil
	}

	if humanPlayers > len(preferredHumanRoleOrder) {
		humanPlayers = len(preferredHumanRoleOrder)
	}

	return preferredHumanRoleOrder[:humanPlayers]
}

func validateRoleConfig(roleID domain.RoleID, roleCfg RoleConfig, requireCatalog bool) error {
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

	if requireCatalog && roleCfg.Provider != "" && roleCfg.Model != "" {
		if strings.TrimSpace(roleCfg.URL) == "" {
			errs = append(errs, fmt.Errorf("%s provider/model must exist in llm catalog with a non-empty URL", roleID))
		}
		if roleCfg.APISDKType == "" {
			errs = append(errs, fmt.Errorf("%s provider/model must exist in llm catalog with a non-empty api_sdk_type", roleID))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errors.Join(errs...)
}

func (c *Config) WithLLMCatalog(catalog LLMCatalog) {
	c.LLMCatalog = catalog
}

func (c LLMCatalog) Lookup(provider ProviderName, model string) (LLMCatalogEntry, bool) {
	name := strings.ToLower(strings.TrimSpace(string(provider)))
	model = strings.TrimSpace(model)
	for _, entry := range c.Entries {
		if strings.ToLower(strings.TrimSpace(string(entry.Provider))) == name && strings.TrimSpace(entry.Model) == model {
			return entry, true
		}
	}
	return LLMCatalogEntry{}, false
}

func (c *LLMCatalog) normalize() {
	for i := range c.Entries {
		c.Entries[i].Provider = ProviderName(strings.ToLower(strings.TrimSpace(string(c.Entries[i].Provider))))
		c.Entries[i].Model = strings.TrimSpace(c.Entries[i].Model)
		c.Entries[i].URL = strings.TrimSpace(c.Entries[i].URL)
		c.Entries[i].APISDKType = APISDKType(strings.ToLower(strings.TrimSpace(string(c.Entries[i].APISDKType))))
		c.Entries[i].APIKey = strings.TrimSpace(c.Entries[i].APIKey)
	}
}

func (c LLMCatalog) Validate() error {
	var errs []error
	seen := make(map[string]bool, len(c.Entries))
	for _, entry := range c.Entries {
		key := fmt.Sprintf("%s::%s", entry.Provider, entry.Model)
		if strings.TrimSpace(string(entry.Provider)) == "" {
			errs = append(errs, errors.New("llm catalog provider_name must not be empty"))
		}
		if strings.TrimSpace(entry.Model) == "" {
			errs = append(errs, errors.New("llm catalog model_name must not be empty"))
		}
		if strings.TrimSpace(entry.URL) == "" {
			errs = append(errs, fmt.Errorf("llm catalog entry %q/%q url must not be empty", entry.Provider, entry.Model))
		}
		switch entry.APISDKType {
		case APISDKTypeOllama, APISDKTypeOpenAI:
		case "":
			errs = append(errs, fmt.Errorf("llm catalog entry %q/%q api_sdk_type must not be empty", entry.Provider, entry.Model))
		default:
			errs = append(errs, fmt.Errorf("llm catalog entry %q/%q api_sdk_type must be %q or %q", entry.Provider, entry.Model, APISDKTypeOllama, APISDKTypeOpenAI))
		}
		if seen[key] {
			errs = append(errs, fmt.Errorf("llm catalog entry %q/%q must be unique", entry.Provider, entry.Model))
		}
		seen[key] = true
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func resolveLLMCatalogPath(configPath, catalogPath string) string {
	if strings.TrimSpace(catalogPath) != "" {
		return catalogPath
	}
	if strings.TrimSpace(configPath) == "" {
		return defaultLLMCatalogPath
	}
	return filepath.Join(filepath.Dir(configPath), defaultLLMCatalogPath)
}
