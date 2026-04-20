package app

import (
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}

	if cfg.Environment != defaultEnvironment {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, defaultEnvironment)
	}

	if cfg.Random.Seed != defaultRandomSeed {
		t.Fatalf("Random.Seed = %d, want %d", cfg.Random.Seed, defaultRandomSeed)
	}

	for _, roleID := range domain.CanonicalRoles() {
		roleCfg, ok := cfg.Roles[roleID]
		if !ok {
			t.Fatalf("missing role config for %s", roleID)
		}

		if roleCfg.Kind != PlayerKindHuman {
			t.Fatalf("role %s kind = %q, want %q", roleID, roleCfg.Kind, PlayerKindHuman)
		}
	}
}

func TestLoadConfigFromEnvAIRoleMapping(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("HERBIEGO_ENV", "test")
	t.Setenv("HERBIEGO_RANDOM_SEED", "42")
	t.Setenv("HERBIEGO_ROLE_FINANCE_CONTROLLER_KIND", "ai")
	t.Setenv("HERBIEGO_ROLE_FINANCE_CONTROLLER_PROVIDER", "ollama")
	t.Setenv("HERBIEGO_ROLE_FINANCE_CONTROLLER_MODEL", "llama3.2:3b")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}

	finance := cfg.Roles[domain.RoleFinanceController]
	if finance.Kind != PlayerKindAI {
		t.Fatalf("finance kind = %q, want %q", finance.Kind, PlayerKindAI)
	}

	if finance.Provider != ProviderOllama {
		t.Fatalf("finance provider = %q, want %q", finance.Provider, ProviderOllama)
	}

	if finance.Model != "llama3.2:3b" {
		t.Fatalf("finance model = %q, want %q", finance.Model, "llama3.2:3b")
	}
}

func TestLoadConfigFromEnvReportsValidationErrors(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("HERBIEGO_RANDOM_SEED", "nope")
	t.Setenv("HERBIEGO_ROLE_SALES_MANAGER_KIND", "bot")
	t.Setenv("HERBIEGO_ROLE_FINANCE_CONTROLLER_KIND", "ai")
	t.Setenv("HERBIEGO_ROLE_FINANCE_CONTROLLER_PROVIDER", "invalid")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatal("LoadConfigFromEnv() error = nil, want validation error")
	}

	message := err.Error()
	for _, want := range []string{
		"HERBIEGO_RANDOM_SEED must be an unsigned integer",
		"HERBIEGO_ROLE_SALES_MANAGER_KIND must be \"human\" or \"ai\"",
		"HERBIEGO_ROLE_FINANCE_CONTROLLER_PROVIDER must be one of",
		"HERBIEGO_ROLE_FINANCE_CONTROLLER_MODEL is required",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not contain %q", message, want)
		}
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HERBIEGO_ENV", "")
	t.Setenv("HERBIEGO_RANDOM_SEED", "")

	for _, roleID := range domain.CanonicalRoles() {
		prefix := envRolePrefix(roleID)
		t.Setenv(prefix+"_KIND", "")
		t.Setenv(prefix+"_PROVIDER", "")
		t.Setenv(prefix+"_MODEL", "")
	}
}
