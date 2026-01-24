package checkpoint

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/db"
)

// Checkpoint represents a saved state
type Checkpoint struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"session_id"`
	PortID      string    `json:"port_id,omitempty"`
	TriggerType string    `json:"trigger_type"` // auto_80, auto_90, manual
	TokensUsed  int       `json:"tokens_used"`
	TokenBudget int       `json:"token_budget"`
	Summary     string    `json:"summary"`
	ActiveFiles []string  `json:"active_files"`
	KeyPoints   []string  `json:"key_points"`
	CreatedAt   time.Time `json:"created_at"`
}

// Store manages checkpoint storage
type Store struct {
	db *db.DB
}

// NewStore creates a new checkpoint store
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// EnsureTable creates the checkpoints table if it doesn't exist
func (s *Store) EnsureTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS checkpoints (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			port_id TEXT,
			trigger_type TEXT NOT NULL,
			tokens_used INTEGER,
			token_budget INTEGER,
			summary TEXT,
			active_files TEXT,
			key_points TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_checkpoints_session ON checkpoints(session_id, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_checkpoints_trigger ON checkpoints(trigger_type);
	`)
	return err
}

// Create saves a new checkpoint
func (s *Store) Create(cp *Checkpoint) error {
	if cp.ID == "" {
		cp.ID = "cp-" + uuid.New().String()[:8]
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}

	activeFilesJSON, _ := json.Marshal(cp.ActiveFiles)
	keyPointsJSON, _ := json.Marshal(cp.KeyPoints)

	_, err := s.db.Exec(`
		INSERT INTO checkpoints (id, session_id, port_id, trigger_type, tokens_used, token_budget, summary, active_files, key_points, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cp.ID, cp.SessionID, cp.PortID, cp.TriggerType, cp.TokensUsed, cp.TokenBudget, cp.Summary, string(activeFilesJSON), string(keyPointsJSON), cp.CreatedAt)

	return err
}

// GetLatest retrieves the latest checkpoint for a session
func (s *Store) GetLatest(sessionID string) (*Checkpoint, error) {
	row := s.db.QueryRow(`
		SELECT id, session_id, port_id, trigger_type, tokens_used, token_budget, summary, active_files, key_points, created_at
		FROM checkpoints
		WHERE session_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, sessionID)

	return s.scanCheckpoint(row)
}

// GetByID retrieves a checkpoint by ID
func (s *Store) GetByID(id string) (*Checkpoint, error) {
	row := s.db.QueryRow(`
		SELECT id, session_id, port_id, trigger_type, tokens_used, token_budget, summary, active_files, key_points, created_at
		FROM checkpoints
		WHERE id = ?
	`, id)

	return s.scanCheckpoint(row)
}

// List retrieves checkpoints for a session
func (s *Store) List(sessionID string, limit int) ([]*Checkpoint, error) {
	if limit == 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT id, session_id, port_id, trigger_type, tokens_used, token_budget, summary, active_files, key_points, created_at
		FROM checkpoints
		WHERE session_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkpoints []*Checkpoint
	for rows.Next() {
		cp, err := s.scanCheckpointRows(rows)
		if err != nil {
			continue
		}
		checkpoints = append(checkpoints, cp)
	}

	return checkpoints, nil
}

// GetLatestByTrigger retrieves the latest checkpoint of a specific trigger type
func (s *Store) GetLatestByTrigger(sessionID, triggerType string) (*Checkpoint, error) {
	row := s.db.QueryRow(`
		SELECT id, session_id, port_id, trigger_type, tokens_used, token_budget, summary, active_files, key_points, created_at
		FROM checkpoints
		WHERE session_id = ? AND trigger_type = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, sessionID, triggerType)

	return s.scanCheckpoint(row)
}

// HasRecentCheckpoint checks if there's a recent checkpoint of a specific type
func (s *Store) HasRecentCheckpoint(sessionID, triggerType string, within time.Duration) bool {
	cp, err := s.GetLatestByTrigger(sessionID, triggerType)
	if err != nil {
		return false
	}
	return time.Since(cp.CreatedAt) < within
}

// scanCheckpoint scans a single row into a Checkpoint
func (s *Store) scanCheckpoint(row *sql.Row) (*Checkpoint, error) {
	cp := &Checkpoint{}
	var portID sql.NullString
	var activeFilesJSON, keyPointsJSON string

	err := row.Scan(
		&cp.ID,
		&cp.SessionID,
		&portID,
		&cp.TriggerType,
		&cp.TokensUsed,
		&cp.TokenBudget,
		&cp.Summary,
		&activeFilesJSON,
		&keyPointsJSON,
		&cp.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if portID.Valid {
		cp.PortID = portID.String
	}

	json.Unmarshal([]byte(activeFilesJSON), &cp.ActiveFiles)
	json.Unmarshal([]byte(keyPointsJSON), &cp.KeyPoints)

	return cp, nil
}

// scanCheckpointRows scans rows into a Checkpoint
func (s *Store) scanCheckpointRows(rows *sql.Rows) (*Checkpoint, error) {
	cp := &Checkpoint{}
	var portID sql.NullString
	var activeFilesJSON, keyPointsJSON string

	err := rows.Scan(
		&cp.ID,
		&cp.SessionID,
		&portID,
		&cp.TriggerType,
		&cp.TokensUsed,
		&cp.TokenBudget,
		&cp.Summary,
		&activeFilesJSON,
		&keyPointsJSON,
		&cp.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("체크포인트 스캔 실패: %w", err)
	}

	if portID.Valid {
		cp.PortID = portID.String
	}

	json.Unmarshal([]byte(activeFilesJSON), &cp.ActiveFiles)
	json.Unmarshal([]byte(keyPointsJSON), &cp.KeyPoints)

	return cp, nil
}

// Delete removes a checkpoint
func (s *Store) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM checkpoints WHERE id = ?", id)
	return err
}

// DeleteOld removes checkpoints older than a specified duration
func (s *Store) DeleteOld(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result, err := s.db.Exec("DELETE FROM checkpoints WHERE created_at < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
