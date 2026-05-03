package main

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jpconstantineau/herbiego/internal/adapters/persistence/sqlite"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
	"github.com/jpconstantineau/herbiego/internal/scenario"
)

func TestResolveMatchPersistenceRequiresSQLiteForResume(t *testing.T) {
	runtime := app.Runtime{InitialMatch: scenario.Starter().InitialState("match-1", nil)}

	_, _, err := resolveMatchPersistence(runtime, "", "match-1")
	if err == nil {
		t.Fatal("resolveMatchPersistence() error = nil, want sqlite-db validation")
	}
}

func TestResolveMatchPersistenceLoadsResumedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resume.db")
	store, err := sqlite.NewStore(sqlite.Options{Path: path, RecentHistoryLimit: 10})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	initial := domain.MatchState{
		MatchID:      "match-27",
		ScenarioID:   "starter",
		CurrentRound: 4,
		Plant:        domain.PlantState{Cash: 31},
	}
	if err := store.CreateMatch(initial); err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	runtime := app.Runtime{
		InitialMatch: scenario.Starter().InitialState("fresh-match", nil),
	}
	resolvedStore, state, err := resolveMatchPersistence(runtime, path, "match-27")
	if err != nil {
		t.Fatalf("resolveMatchPersistence() error = %v", err)
	}
	t.Cleanup(func() { _ = resolvedStore.Close() })

	if got := state.MatchID; got != "match-27" {
		t.Fatalf("MatchID = %q, want resumed match", got)
	}
	if got := state.CurrentRound; got != 4 {
		t.Fatalf("CurrentRound = %d, want 4", got)
	}
}

func TestConfigureAICallPersistenceWritesRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai-debug.db")
	store, err := sqlite.NewStore(sqlite.Options{Path: path, RecentHistoryLimit: 10})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	match := domain.MatchState{
		MatchID:      "match-28",
		ScenarioID:   "starter",
		CurrentRound: 1,
	}
	if err := store.CreateMatch(match); err != nil {
		t.Fatalf("CreateMatch() error = %v", err)
	}

	debugLog := app.NewDebugLog(10)
	var logs bytes.Buffer
	configureAICallPersistence(debugLog, slog.New(slog.NewTextHandler(&logs, nil)), store, match.MatchID)

	debugLog.Append(ports.AICallRecord{
		RoleID:       domain.RoleSalesManager,
		Round:        1,
		Attempt:      1,
		Provider:     "openrouter",
		Model:        "openai/gpt-5-mini",
		SystemPrompt: "system",
		UserPrompt:   "user",
		RawResponse:  "{}",
		Valid:        true,
	})

	records, err := store.AICallRecords(match.MatchID)
	if err != nil {
		t.Fatalf("AICallRecords() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("AICallRecords len = %d, want 1", len(records))
	}
}
