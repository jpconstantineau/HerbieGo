package main

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/jpconstantineau/herbiego/internal/adapters/persistence/sqlite"
	"github.com/jpconstantineau/herbiego/internal/app"
	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

type persistentStore interface {
	ports.MatchStateStore
	Close() error
	AppendAICall(matchID domain.MatchID, record ports.AICallRecord) error
}

func resolveMatchPersistence(runtime app.Runtime, sqlitePath, resumeMatchID string) (persistentStore, domain.MatchState, error) {
	initial := runtime.InitialMatch.Clone()
	if sqlitePath == "" {
		if resumeMatchID != "" {
			return nil, domain.MatchState{}, errors.New("resume-match-id requires sqlite-db")
		}
		return nil, initial, nil
	}

	store, err := sqlite.NewStore(sqlite.Options{
		Path:               sqlitePath,
		RecentHistoryLimit: 10,
	})
	if err != nil {
		return nil, domain.MatchState{}, fmt.Errorf("open sqlite store: %w", err)
	}

	if resumeMatchID == "" {
		return store, initial, nil
	}

	current, err := store.CurrentState(domain.MatchID(resumeMatchID))
	if err != nil {
		_ = store.Close()
		return nil, domain.MatchState{}, fmt.Errorf("resume match %q: %w", resumeMatchID, err)
	}
	return store, current, nil
}

func configureAICallPersistence(debugLog *app.DebugLog, logger *slog.Logger, store persistentStore, matchID domain.MatchID) {
	debugLog.SetSink(func(record ports.AICallRecord) {
		if err := store.AppendAICall(matchID, record); err != nil {
			loggerOrDiscard(logger).Error(
				"persist ai call record failed",
				"match_id", matchID,
				"role_id", record.RoleID,
				"round", record.Round,
				"attempt", record.Attempt,
				"error", err,
			)
		}
	})
}

func loggerOrDiscard(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default()
}
