package sqlite

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/jpconstantineau/herbiego/internal/domain"
	"github.com/jpconstantineau/herbiego/internal/ports"
)

var ErrSaveSlotNotFound = errors.New("save slot not found")

type Options struct {
	Path               string
	RecentHistoryLimit int
}

type SaveSlotSummary struct {
	SlotName     string
	MatchID      domain.MatchID
	ScenarioID   domain.ScenarioID
	CurrentRound domain.RoundNumber
	Cash         domain.Money
	SavedAt      time.Time
}

type Store struct {
	db           *sql.DB
	historyLimit int
}

func NewStore(options Options) (*Store, error) {
	if options.Path == "" {
		return nil, errors.New("sqlite store: path must not be empty")
	}

	db, err := sql.Open("sqlite", options.Path)
	if err != nil {
		return nil, fmt.Errorf("sqlite store: open database: %w", err)
	}

	store := &Store{
		db:           db,
		historyLimit: normalizedHistoryLimit(options.RecentHistoryLimit),
	}
	if err := store.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) CreateMatch(initial domain.MatchState) error {
	if initial.MatchID == "" {
		return errors.New("sqlite store: match id must not be empty")
	}

	state := initial.Clone()
	history := state.History.Recent(s.historyLimit)
	rounds := history.RecentRounds
	state.History = history

	stateJSON, err := marshalJSON(state)
	if err != nil {
		return fmt.Errorf("sqlite store: marshal initial state: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("sqlite store: begin create match: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.Exec(`
		INSERT INTO matches(match_id, scenario_id, current_state_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, initial.MatchID, initial.ScenarioID, stateJSON, now, now); err != nil {
		return fmt.Errorf("sqlite store: create match %q: %w", initial.MatchID, err)
	}
	if _, err := tx.Exec(`
		INSERT INTO state_snapshots(match_id, current_round, state_json)
		VALUES (?, ?, ?)
	`, initial.MatchID, state.CurrentRound, stateJSON); err != nil {
		return fmt.Errorf("sqlite store: insert initial snapshot for %q: %w", initial.MatchID, err)
	}

	if err := insertRounds(tx, initial.MatchID, rounds); err != nil {
		return err
	}
	if err := insertEvents(tx, initial.MatchID, flattenEvents(rounds)); err != nil {
		return err
	}
	if err := insertCommentary(tx, initial.MatchID, flattenCommentary(rounds)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite store: commit create match: %w", err)
	}
	return nil
}

func (s *Store) CurrentState(matchID domain.MatchID) (domain.MatchState, error) {
	var stateJSON string
	err := s.db.QueryRow(`SELECT current_state_json FROM matches WHERE match_id = ?`, matchID).Scan(&stateJSON)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.MatchState{}, ports.ErrMatchNotFound
	case err != nil:
		return domain.MatchState{}, fmt.Errorf("sqlite store: current state for %q: %w", matchID, err)
	}

	var state domain.MatchState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: decode current state for %q: %w", matchID, err)
	}
	return state, nil
}

func (s *Store) CommitRound(matchID domain.MatchID, nextState domain.MatchState, round domain.RoundRecord) (domain.MatchState, error) {
	if matchID == "" {
		return domain.MatchState{}, errors.New("sqlite store: match id must not be empty")
	}
	if nextState.MatchID != matchID {
		return domain.MatchState{}, fmt.Errorf("sqlite store: next state match id %q does not match %q", nextState.MatchID, matchID)
	}
	if round.Round <= 0 {
		return domain.MatchState{}, fmt.Errorf("sqlite store: round %d must be positive", round.Round)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: begin commit round: %w", err)
	}
	defer tx.Rollback()

	var last sql.NullInt64
	err = tx.QueryRow(`SELECT MAX(round_number) FROM rounds WHERE match_id = ?`, matchID).Scan(&last)
	if err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: select latest round: %w", err)
	}
	if last.Valid && int64(round.Round) <= last.Int64 {
		return domain.MatchState{}, fmt.Errorf("sqlite store: round %d must be greater than last committed round %d", round.Round, last.Int64)
	}

	next := nextState.Clone()
	committedRound := round.Clone()
	stateJSON, err := marshalStateWithRecentHistory(next, s.historyLimit)
	if err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: marshal next state: %w", err)
	}
	roundJSON, err := marshalJSON(committedRound)
	if err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: marshal round: %w", err)
	}

	result, err := tx.Exec(`
		UPDATE matches
		SET current_state_json = ?, updated_at = ?
		WHERE match_id = ?
	`, stateJSON, time.Now().UTC().Format(time.RFC3339Nano), matchID)
	if err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: update current state: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: rows affected for update: %w", err)
	}
	if rowsAffected == 0 {
		return domain.MatchState{}, ports.ErrMatchNotFound
	}

	if _, err := tx.Exec(`
		INSERT INTO rounds(match_id, round_number, round_json)
		VALUES (?, ?, ?)
	`, matchID, round.Round, roundJSON); err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: insert round %d: %w", round.Round, err)
	}
	if _, err := tx.Exec(`
		INSERT INTO state_snapshots(match_id, current_round, state_json)
		VALUES (?, ?, ?)
		ON CONFLICT(match_id, current_round) DO UPDATE SET state_json = excluded.state_json
	`, matchID, next.CurrentRound, stateJSON); err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: insert snapshot for round %d: %w", next.CurrentRound, err)
	}
	if err := insertEvents(tx, matchID, committedRound.Events); err != nil {
		return domain.MatchState{}, err
	}
	if err := insertCommentary(tx, matchID, committedRound.Commentary); err != nil {
		return domain.MatchState{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.MatchState{}, fmt.Errorf("sqlite store: commit round transaction: %w", err)
	}

	next.History = next.History.Recent(s.historyLimit)
	return next, nil
}

func (s *Store) Round(matchID domain.MatchID, roundNumber domain.RoundNumber) (domain.RoundRecord, error) {
	var roundJSON string
	err := s.db.QueryRow(`
		SELECT round_json
		FROM rounds
		WHERE match_id = ? AND round_number = ?
	`, matchID, roundNumber).Scan(&roundJSON)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, stateErr := s.CurrentState(matchID); stateErr != nil {
			return domain.RoundRecord{}, stateErr
		}
		return domain.RoundRecord{}, fmt.Errorf("sqlite store: round %d not found", roundNumber)
	case err != nil:
		return domain.RoundRecord{}, fmt.Errorf("sqlite store: fetch round %d: %w", roundNumber, err)
	}

	var round domain.RoundRecord
	if err := json.Unmarshal([]byte(roundJSON), &round); err != nil {
		return domain.RoundRecord{}, fmt.Errorf("sqlite store: decode round %d: %w", roundNumber, err)
	}
	return round, nil
}

func (s *Store) EventTimeline(matchID domain.MatchID) ([]domain.RoundEvent, error) {
	rows, err := s.db.Query(`
		SELECT event_json
		FROM events
		WHERE match_id = ?
		ORDER BY id ASC
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("sqlite store: query event timeline: %w", err)
	}
	defer rows.Close()

	var events []domain.RoundEvent
	for rows.Next() {
		var encoded string
		if err := rows.Scan(&encoded); err != nil {
			return nil, fmt.Errorf("sqlite store: scan event row: %w", err)
		}
		var event domain.RoundEvent
		if err := json.Unmarshal([]byte(encoded), &event); err != nil {
			return nil, fmt.Errorf("sqlite store: decode event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite store: iterate event timeline: %w", err)
	}
	if len(events) == 0 {
		if _, err := s.CurrentState(matchID); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (s *Store) Commentary(matchID domain.MatchID) ([]domain.CommentaryRecord, error) {
	rows, err := s.db.Query(`
		SELECT commentary_json
		FROM commentary
		WHERE match_id = ?
		ORDER BY id ASC
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("sqlite store: query commentary timeline: %w", err)
	}
	defer rows.Close()

	var commentary []domain.CommentaryRecord
	for rows.Next() {
		var encoded string
		if err := rows.Scan(&encoded); err != nil {
			return nil, fmt.Errorf("sqlite store: scan commentary row: %w", err)
		}
		var record domain.CommentaryRecord
		if err := json.Unmarshal([]byte(encoded), &record); err != nil {
			return nil, fmt.Errorf("sqlite store: decode commentary: %w", err)
		}
		commentary = append(commentary, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite store: iterate commentary timeline: %w", err)
	}
	if len(commentary) == 0 {
		if _, err := s.CurrentState(matchID); err != nil {
			return nil, err
		}
	}
	return commentary, nil
}

func (s *Store) AppendAICall(matchID domain.MatchID, record ports.AICallRecord) error {
	if _, err := s.CurrentState(matchID); err != nil {
		return err
	}

	recordJSON, err := marshalJSON(record)
	if err != nil {
		return fmt.Errorf("sqlite store: marshal ai call record: %w", err)
	}
	if _, err := s.db.Exec(`
		INSERT INTO ai_call_records(match_id, record_json)
		VALUES (?, ?)
	`, matchID, recordJSON); err != nil {
		return fmt.Errorf("sqlite store: insert ai call record: %w", err)
	}
	return nil
}

func (s *Store) AICallRecords(matchID domain.MatchID) ([]ports.AICallRecord, error) {
	rows, err := s.db.Query(`
		SELECT record_json
		FROM ai_call_records
		WHERE match_id = ?
		ORDER BY id ASC
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("sqlite store: query ai call records: %w", err)
	}
	defer rows.Close()

	var records []ports.AICallRecord
	for rows.Next() {
		var encoded string
		if err := rows.Scan(&encoded); err != nil {
			return nil, fmt.Errorf("sqlite store: scan ai call row: %w", err)
		}
		var record ports.AICallRecord
		if err := json.Unmarshal([]byte(encoded), &record); err != nil {
			return nil, fmt.Errorf("sqlite store: decode ai call record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite store: iterate ai call records: %w", err)
	}
	if len(records) == 0 {
		if _, err := s.CurrentState(matchID); err != nil {
			return nil, err
		}
	}
	return records, nil
}

func (s *Store) StateSnapshots(matchID domain.MatchID) ([]domain.MatchState, error) {
	rows, err := s.db.Query(`
		SELECT state_json
		FROM state_snapshots
		WHERE match_id = ?
		ORDER BY current_round ASC
	`, matchID)
	if err != nil {
		return nil, fmt.Errorf("sqlite store: query state snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []domain.MatchState
	for rows.Next() {
		var encoded string
		if err := rows.Scan(&encoded); err != nil {
			return nil, fmt.Errorf("sqlite store: scan state snapshot: %w", err)
		}
		var state domain.MatchState
		if err := json.Unmarshal([]byte(encoded), &state); err != nil {
			return nil, fmt.Errorf("sqlite store: decode state snapshot: %w", err)
		}
		snapshots = append(snapshots, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite store: iterate state snapshots: %w", err)
	}
	if len(snapshots) == 0 {
		if _, err := s.CurrentState(matchID); err != nil {
			return nil, err
		}
	}
	return snapshots, nil
}

func (s *Store) SaveSlot(slotName string, matchID domain.MatchID) (SaveSlotSummary, error) {
	if slotName == "" {
		return SaveSlotSummary{}, errors.New("sqlite store: save slot name must not be empty")
	}
	state, err := s.CurrentState(matchID)
	if err != nil {
		return SaveSlotSummary{}, err
	}

	now := time.Now().UTC()
	if _, err := s.db.Exec(`
		INSERT INTO save_slots(slot_name, match_id, scenario_id, current_round, cash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(slot_name) DO UPDATE SET
			match_id = excluded.match_id,
			scenario_id = excluded.scenario_id,
			current_round = excluded.current_round,
			cash = excluded.cash,
			updated_at = excluded.updated_at
	`, slotName, matchID, state.ScenarioID, state.CurrentRound, state.Plant.Cash, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		return SaveSlotSummary{}, fmt.Errorf("sqlite store: save slot %q: %w", slotName, err)
	}

	return SaveSlotSummary{
		SlotName:     slotName,
		MatchID:      matchID,
		ScenarioID:   state.ScenarioID,
		CurrentRound: state.CurrentRound,
		Cash:         state.Plant.Cash,
		SavedAt:      now,
	}, nil
}

func (s *Store) ListSaveSlots() ([]SaveSlotSummary, error) {
	rows, err := s.db.Query(`
		SELECT slot_name, match_id, scenario_id, current_round, cash, updated_at
		FROM save_slots
		ORDER BY updated_at DESC, slot_name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("sqlite store: query save slots: %w", err)
	}
	defer rows.Close()

	var slots []SaveSlotSummary
	for rows.Next() {
		var (
			slot       SaveSlotSummary
			updatedAt  string
			roundValue int
			cashValue  int
		)
		if err := rows.Scan(&slot.SlotName, &slot.MatchID, &slot.ScenarioID, &roundValue, &cashValue, &updatedAt); err != nil {
			return nil, fmt.Errorf("sqlite store: scan save slot: %w", err)
		}
		slot.CurrentRound = domain.RoundNumber(roundValue)
		slot.Cash = domain.Money(cashValue)
		slot.SavedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("sqlite store: parse save slot time: %w", err)
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite store: iterate save slots: %w", err)
	}
	return slots, nil
}

func (s *Store) LoadSaveSlot(slotName string) (domain.MatchState, SaveSlotSummary, error) {
	if slotName == "" {
		return domain.MatchState{}, SaveSlotSummary{}, errors.New("sqlite store: save slot name must not be empty")
	}

	var (
		summary    SaveSlotSummary
		updatedAt  string
		roundValue int
		cashValue  int
	)
	err := s.db.QueryRow(`
		SELECT slot_name, match_id, scenario_id, current_round, cash, updated_at
		FROM save_slots
		WHERE slot_name = ?
	`, slotName).Scan(&summary.SlotName, &summary.MatchID, &summary.ScenarioID, &roundValue, &cashValue, &updatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return domain.MatchState{}, SaveSlotSummary{}, ErrSaveSlotNotFound
	case err != nil:
		return domain.MatchState{}, SaveSlotSummary{}, fmt.Errorf("sqlite store: load save slot %q: %w", slotName, err)
	}

	summary.CurrentRound = domain.RoundNumber(roundValue)
	summary.Cash = domain.Money(cashValue)
	summary.SavedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return domain.MatchState{}, SaveSlotSummary{}, fmt.Errorf("sqlite store: parse save slot time: %w", err)
	}

	state, err := s.CurrentState(summary.MatchID)
	if err != nil {
		return domain.MatchState{}, SaveSlotSummary{}, err
	}
	return state, summary, nil
}

func (s *Store) initSchema() error {
	if _, err := s.db.Exec(`
		PRAGMA foreign_keys = ON;
		CREATE TABLE IF NOT EXISTS matches (
			match_id TEXT PRIMARY KEY,
			scenario_id TEXT NOT NULL,
			current_state_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS rounds (
			match_id TEXT NOT NULL,
			round_number INTEGER NOT NULL,
			round_json TEXT NOT NULL,
			PRIMARY KEY(match_id, round_number)
		);
		CREATE TABLE IF NOT EXISTS state_snapshots (
			match_id TEXT NOT NULL,
			current_round INTEGER NOT NULL,
			state_json TEXT NOT NULL,
			PRIMARY KEY(match_id, current_round)
		);
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id TEXT NOT NULL,
			event_json TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS commentary (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id TEXT NOT NULL,
			commentary_json TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS ai_call_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id TEXT NOT NULL,
			record_json TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS save_slots (
			slot_name TEXT PRIMARY KEY,
			match_id TEXT NOT NULL,
			scenario_id TEXT NOT NULL,
			current_round INTEGER NOT NULL,
			cash INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("sqlite store: initialize schema: %w", err)
	}
	return nil
}

func insertRounds(tx *sql.Tx, matchID domain.MatchID, rounds []domain.RoundRecord) error {
	for _, round := range rounds {
		encoded, err := marshalJSON(round)
		if err != nil {
			return fmt.Errorf("sqlite store: marshal round %d: %w", round.Round, err)
		}
		if _, err := tx.Exec(`
			INSERT INTO rounds(match_id, round_number, round_json)
			VALUES (?, ?, ?)
		`, matchID, round.Round, encoded); err != nil {
			return fmt.Errorf("sqlite store: insert round %d: %w", round.Round, err)
		}
	}
	return nil
}

func insertEvents(tx *sql.Tx, matchID domain.MatchID, events []domain.RoundEvent) error {
	for _, event := range events {
		encoded, err := marshalJSON(event)
		if err != nil {
			return fmt.Errorf("sqlite store: marshal event %q: %w", event.EventID, err)
		}
		if _, err := tx.Exec(`
			INSERT INTO events(match_id, event_json)
			VALUES (?, ?)
		`, matchID, encoded); err != nil {
			return fmt.Errorf("sqlite store: insert event %q: %w", event.EventID, err)
		}
	}
	return nil
}

func insertCommentary(tx *sql.Tx, matchID domain.MatchID, commentary []domain.CommentaryRecord) error {
	for _, record := range commentary {
		encoded, err := marshalJSON(record)
		if err != nil {
			return fmt.Errorf("sqlite store: marshal commentary %q: %w", record.CommentaryID, err)
		}
		if _, err := tx.Exec(`
			INSERT INTO commentary(match_id, commentary_json)
			VALUES (?, ?)
		`, matchID, encoded); err != nil {
			return fmt.Errorf("sqlite store: insert commentary %q: %w", record.CommentaryID, err)
		}
	}
	return nil
}

func marshalStateWithRecentHistory(nextState domain.MatchState, limit int) (string, error) {
	state := nextState.Clone()
	state.History = state.History.Recent(limit)
	return marshalJSON(state)
}

func marshalJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizedHistoryLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	return limit
}

func flattenEvents(rounds []domain.RoundRecord) []domain.RoundEvent {
	var events []domain.RoundEvent
	for _, round := range rounds {
		for _, event := range round.Events {
			events = append(events, event.Clone())
		}
	}
	return events
}

func flattenCommentary(rounds []domain.RoundRecord) []domain.CommentaryRecord {
	var commentary []domain.CommentaryRecord
	for _, round := range rounds {
		for _, record := range round.Commentary {
			commentary = append(commentary, record.Clone())
		}
	}
	return commentary
}
