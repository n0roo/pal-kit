package session

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Session represents a work session
type Session struct {
	ID                string
	PortID            sql.NullString
	Title             sql.NullString
	Status            string
	StartedAt         time.Time
	EndedAt           sql.NullTime
	JSONLPath         sql.NullString
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheCreateTokens int64
	CostUSD           float64
	CompactCount      int
	LastCompactAt     sql.NullTime
}

// Service handles session operations
type Service struct {
	db *db.DB
}

// NewService creates a new session service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Start creates a new session
func (s *Service) Start(id, portID, title string) error {
	var portIDNull, titleNull sql.NullString
	
	if portID != "" {
		portIDNull = sql.NullString{String: portID, Valid: true}
	}
	if title != "" {
		titleNull = sql.NullString{String: title, Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO sessions (id, port_id, title, status)
		VALUES (?, ?, ?, 'running')
	`, id, portIDNull, titleNull)
	
	if err != nil {
		return fmt.Errorf("세션 생성 실패: %w", err)
	}
	return nil
}

// End marks a session as ended
func (s *Service) End(id string) error {
	result, err := s.db.Exec(`
		UPDATE sessions 
		SET status = 'complete', ended_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'running'
	`, id)
	
	if err != nil {
		return fmt.Errorf("세션 종료 실패: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("세션 '%s'을(를) 찾을 수 없거나 이미 종료됨", id)
	}
	
	return nil
}

// UpdateStatus updates session status
func (s *Service) UpdateStatus(id, status string) error {
	result, err := s.db.Exec(`
		UPDATE sessions SET status = ? WHERE id = ?
	`, status, id)
	
	if err != nil {
		return fmt.Errorf("상태 업데이트 실패: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("세션 '%s'을(를) 찾을 수 없습니다", id)
	}
	
	return nil
}

// Get retrieves a session by ID
func (s *Service) Get(id string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(`
		SELECT id, port_id, title, status, started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at
		FROM sessions WHERE id = ?
	`, id).Scan(
		&sess.ID, &sess.PortID, &sess.Title, &sess.Status, &sess.StartedAt, &sess.EndedAt,
		&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
		&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("세션 '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}
	
	return &sess, nil
}

// List returns sessions with optional filters
func (s *Service) List(activeOnly bool, limit int) ([]Session, error) {
	query := `
		SELECT id, port_id, title, status, started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at
		FROM sessions
	`
	
	if activeOnly {
		query += ` WHERE status = 'running'`
	}
	
	query += ` ORDER BY started_at DESC`
	
	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("세션 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(
			&sess.ID, &sess.PortID, &sess.Title, &sess.Status, &sess.StartedAt, &sess.EndedAt,
			&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
			&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// IncrementCompact increments compact count for a session
func (s *Service) IncrementCompact(id string) error {
	_, err := s.db.Exec(`
		UPDATE sessions 
		SET compact_count = compact_count + 1, last_compact_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// UpdateUsage updates token usage for a session
func (s *Service) UpdateUsage(id string, input, output, cacheRead, cacheCreate int64, cost float64) error {
	_, err := s.db.Exec(`
		UPDATE sessions 
		SET input_tokens = ?, output_tokens = ?, cache_read_tokens = ?,
		    cache_create_tokens = ?, cost_usd = ?
		WHERE id = ?
	`, input, output, cacheRead, cacheCreate, cost, id)
	return err
}
