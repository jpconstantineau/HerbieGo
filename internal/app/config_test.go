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
    provider: ollama-localhost
  - role_id: production_manager
    provider: ollama-localhost
  - role_id: sales_manager
    provider: openrouter
  - role_id: finance_controller
    provider: ollama-localhost
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	catalogPath := writeSiblingFile(t, configPath, "llm.yaml", `
models:
  - provider_name: ollama-localhost
    model_name: llama3.2:3b
    url: http://localhost:11434/
    api_sdk_type: ollama
    api_key: ""
  - provider_name: openrouter
    model_name: openai/gpt-5-mini
    url: https://openrouter.ai/api/v1/
    api_sdk_type: openai
    api_key: ""
`)
	catalog, err := LoadLLMCatalog(catalogPath)
	if err != nil {
		t.Fatalf("LoadLLMCatalog() error = %v", err)
	}
	cfg.WithLLMCatalog(catalog)
	cfg = cfg.ApplyOverrides(BootstrapOptions{})

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
	if cfg.Roles[domain.RoleProcurementManager].URL != "http://localhost:11434/" {
		t.Fatalf("procurement url = %q, want localhost Ollama endpoint", cfg.Roles[domain.RoleProcurementManager].URL)
	}
	if cfg.Roles[domain.RoleSalesManager].APISDKType != APISDKTypeOpenAI {
		t.Fatalf("sales api sdk type = %q, want %q", cfg.Roles[domain.RoleSalesManager].APISDKType, APISDKTypeOpenAI)
	}
}

func TestConfigApplyOverridesSupportsAITestMode(t *testing.T) {
	configPath := writeConfigFile(t, `
roles:
  - role_id: procurement_manager
    provider: ollama-localhost
  - role_id: production_manager
    provider: ollama-localhost
  - role_id: sales_manager
    provider: openrouter
  - role_id: finance_controller
    provider: ollama-localhost
`)

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	catalogPath := writeSiblingFile(t, configPath, "llm.yaml", `
models:
  - provider_name: ollama-localhost
    model_name: llama3.2:3b
    url: http://localhost:11434/
    api_sdk_type: ollama
    api_key: ""
  - provider_name: openrouter
    model_name: openai/gpt-5-mini
    url: https://openrouter.ai/api/v1/
    api_sdk_type: openai
    api_key: ""
`)
	catalog, err := LoadLLMCatalog(catalogPath)
	if err != nil {
		t.Fatalf("LoadLLMCatalog() error = %v", err)
	}
	cfg.WithLLMCatalog(catalog)

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
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not contain %q", message, want)
		}
	}
}

func TestLoadLLMCatalogRejectsDuplicateProviderModelPairs(t *testing.T) {
	path := writeCatalogFile(t, `
models:
  - provider_name: ollama-localhost
    model_name: gemma4:e4b
    url: http://localhost:11434/
    api_sdk_type: ollama
    api_key: ""
  - provider_name: ollama-localhost
    model_name: llama3.2:3b
    url: https://ollama.com/
    api_sdk_type: ollama
    api_key: ""
`)

	_, err := LoadLLMCatalog(path)
	if err == nil {
		t.Fatal("LoadLLMCatalog() error = nil, want duplicate-entry validation error")
	}
	if !strings.Contains(err.Error(), `provider_name "ollama-localhost" must be unique`) {
		t.Fatalf("LoadLLMCatalog() error = %v, want duplicate-entry validation", err)
	}
}

func TestNewRuntimeRejectsRolesMissingCatalogEntries(t *testing.T) {
	_, err := NewRuntime(Config{
		Environment:  "test",
		HumanPlayers: 0,
		Random: RandomConfig{
			Seed: 9,
		},
		LLMCatalog: LLMCatalog{
			Entries: []LLMCatalogEntry{
				{Provider: "ollama-localhost", Model: "gemma4:e4b", URL: "http://localhost:11434/", APISDKType: APISDKTypeOllama},
			},
		},
		RoleConfigs: []RoleConfigFile{
			{RoleID: "procurement_manager", Provider: "ollama-localhost"},
			{RoleID: "production_manager", Provider: "ollama-localhost"},
			{RoleID: "sales_manager", Provider: "openrouter"},
			{RoleID: "finance_controller", Provider: "ollama-localhost"},
		},
	})
	if err == nil {
		t.Fatal("NewRuntime() error = nil, want missing-catalog validation error")
	}
	if !strings.Contains(err.Error(), "provider label") {
		t.Fatalf("NewRuntime() error = %v, want llm-catalog validation", err)
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

func writeCatalogFile(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	return writeSiblingFile(t, filepath.Join(dir, "herbiego.yaml"), "llm.yaml", contents)
}

func writeSiblingFile(t *testing.T, anchorPath, name, contents string) string {
	t.Helper()

	path := filepath.Join(filepath.Dir(anchorPath), name)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(contents)), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
