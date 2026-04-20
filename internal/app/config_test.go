package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/domain"
)

func TestLoadConfigDefaultsFromYAML(t *testing.T) {
	configPath := writeConfigFile(t, `
environment: local
random:
  seed: 7
human_players: 1
roles:
  - role_id: procurement_manager
    provider: ollama
    model: llama3.2:3b
  - role_id: production_manager
    provider: ollama
    model: llama3.2:3b
  - role_id: sales_manager
    provider: openrouter
    model: openai/gpt-5-mini
  - role_id: finance_controller
    provider: ollama
    model: llama3.2:3b
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Environment != "local" {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, "local")
	}

	if cfg.Random.Seed != 7 {
		t.Fatalf("Random.Seed = %d, want 7", cfg.Random.Seed)
	}

	if cfg.HumanPlayers != 1 {
		t.Fatalf("HumanPlayers = %d, want 1", cfg.HumanPlayers)
	}

	if cfg.Roles[domain.RoleProductionManager].Kind != PlayerKindHuman {
		t.Fatalf("production kind = %q, want %q", cfg.Roles[domain.RoleProductionManager].Kind, PlayerKindHuman)
	}

	if cfg.Roles[domain.RoleProcurementManager].Kind != PlayerKindAI {
		t.Fatalf("procurement kind = %q, want %q", cfg.Roles[domain.RoleProcurementManager].Kind, PlayerKindAI)
	}
}

func TestConfigApplyOverridesSupportsAITestMode(t *testing.T) {
	configPath := writeConfigFile(t, `
roles:
  - role_id: procurement_manager
    provider: ollama
    model: llama3.2:3b
  - role_id: production_manager
    provider: ollama
    model: llama3.2:3b
  - role_id: sales_manager
    provider: openrouter
    model: openai/gpt-5-mini
  - role_id: finance_controller
    provider: ollama
    model: llama3.2:3b
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	humanPlayers := 0
	seed := uint64(42)
	cfg = cfg.ApplyOverrides(BootstrapOptions{
		HumanPlayersOverride: &humanPlayers,
		SeedOverride:         &seed,
	})

	if cfg.HumanPlayers != 0 {
		t.Fatalf("HumanPlayers = %d, want 0", cfg.HumanPlayers)
	}

	if cfg.Random.Seed != 42 {
		t.Fatalf("Random.Seed = %d, want 42", cfg.Random.Seed)
	}

	for _, roleID := range domain.CanonicalRoles() {
		if cfg.Roles[roleID].Kind != PlayerKindAI {
			t.Fatalf("role %s kind = %q, want %q", roleID, cfg.Roles[roleID].Kind, PlayerKindAI)
		}
	}
}

func TestLoadConfigReportsValidationErrors(t *testing.T) {
	configPath := writeConfigFile(t, `
human_players: 5
roles:
  - role_id: procurement_manager
    provider: ""
    model: ""
`)

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want validation error")
	}

	message := err.Error()
	for _, want := range []string{
		"human_players must be between 0 and 4",
		"roles must include exactly 4 canonical roles",
		"procurement_manager provider must not be empty",
		"procurement_manager model must not be empty",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not contain %q", message, want)
		}
	}
}

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "herbiego.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	return path
}
