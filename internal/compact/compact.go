package compact

import (
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Compaction represents a context compaction event
type Compaction struct {
	ID             int64
	SessionID      string
	TriggeredAt    time.Time
	TriggerType    string
	ContextSummary string
	TokensBefore   int64
}

// Service handles compaction tracking
type Service struct {
	db *db.DB
}

// NewService creates a new compact service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Record records a compaction event
func (s *Service) Record(sessionID, triggerType, summary string, tokensBefore int64) (int64, error) {
	if triggerType == "" {
		triggerType = "auto"
	}

	result, err := s.db.Exec(`
		INSERT INTO compactions (session_id, trigger_type, context_summary, tokens_before)
		VALUES (?, ?, ?, ?)
	`, sessionID, triggerType, summary, tokensBefore)

	if err != nil {
		return 0, fmt.Errorf("컴팩션 기록 실패: %w", err)
	}

	// 세션의 compact_count 업데이트
	s.db.Exec(`
		UPDATE sessions 
		SET compact_count = compact_count + 1, last_compact_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, sessionID)

	return result.LastInsertId()
}

// List returns compaction history for a session
func (s *Service) List(sessionID string, limit int) ([]Compaction, error) {
	query := `
		SELECT id, session_id, triggered_at, trigger_type, 
		       COALESCE(context_summary, ''), COALESCE(tokens_before, 0)
		FROM compactions
	`

	var args []interface{}
	if sessionID != "" {
		query += ` WHERE session_id = ?`
		args = append(args, sessionID)
	}

	query += ` ORDER BY triggered_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("컴팩션 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var compactions []Compaction
	for rows.Next() {
		var c Compaction
		if err := rows.Scan(&c.ID, &c.SessionID, &c.TriggeredAt, &c.TriggerType, &c.ContextSummary, &c.TokensBefore); err != nil {
			return nil, err
		}
		compactions = append(compactions, c)
	}

	return compactions, nil
}

// Get retrieves a specific compaction
func (s *Service) Get(id int64) (*Compaction, error) {
	var c Compaction
	err := s.db.QueryRow(`
		SELECT id, session_id, triggered_at, trigger_type, 
		       COALESCE(context_summary, ''), COALESCE(tokens_before, 0)
		FROM compactions WHERE id = ?
	`, id).Scan(&c.ID, &c.SessionID, &c.TriggeredAt, &c.TriggerType, &c.ContextSummary, &c.TokensBefore)

	if err != nil {
		return nil, fmt.Errorf("컴팩션 조회 실패: %w", err)
	}

	return &c, nil
}

// GetSessionCompactCount returns the number of compactions for a session
func (s *Service) GetSessionCompactCount(sessionID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM compactions WHERE session_id = ?`, sessionID).Scan(&count)
	return count, err
}

// Summary returns compaction statistics
func (s *Service) Summary() (map[string]interface{}, error) {
	var totalCount int
	var autoCount int
	var manualCount int

	s.db.QueryRow(`SELECT COUNT(*) FROM compactions`).Scan(&totalCount)
	s.db.QueryRow(`SELECT COUNT(*) FROM compactions WHERE trigger_type = 'auto'`).Scan(&autoCount)
	s.db.QueryRow(`SELECT COUNT(*) FROM compactions WHERE trigger_type = 'manual'`).Scan(&manualCount)

	return map[string]interface{}{
		"total":  totalCount,
		"auto":   autoCount,
		"manual": manualCount,
	}, nil
}
