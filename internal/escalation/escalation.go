package escalation

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Escalation represents an escalation request
type Escalation struct {
	ID          int64
	FromSession sql.NullString
	FromPort    sql.NullString
	Issue       string
	Status      string
	CreatedAt   time.Time
	ResolvedAt  sql.NullTime
}

// Status constants
const (
	StatusOpen      = "open"
	StatusResolved  = "resolved"
	StatusDismissed = "dismissed"
)

// Service handles escalation operations
type Service struct {
	db *db.DB
}

// NewService creates a new escalation service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Create creates a new escalation
func (s *Service) Create(issue, sessionID, portID string) (int64, error) {
	var sessionNull, portNull sql.NullString

	if sessionID != "" {
		sessionNull = sql.NullString{String: sessionID, Valid: true}
	}
	if portID != "" {
		portNull = sql.NullString{String: portID, Valid: true}
	}

	result, err := s.db.Exec(`
		INSERT INTO escalations (issue, from_session, from_port, status)
		VALUES (?, ?, ?, 'open')
	`, issue, sessionNull, portNull)

	if err != nil {
		return 0, fmt.Errorf("에스컬레이션 생성 실패: %w", err)
	}

	return result.LastInsertId()
}

// Resolve marks an escalation as resolved
func (s *Service) Resolve(id int64) error {
	result, err := s.db.Exec(`
		UPDATE escalations 
		SET status = 'resolved', resolved_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'open'
	`, id)

	if err != nil {
		return fmt.Errorf("에스컬레이션 해결 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("에스컬레이션 #%d을(를) 찾을 수 없거나 이미 처리됨", id)
	}

	return nil
}

// Dismiss marks an escalation as dismissed
func (s *Service) Dismiss(id int64) error {
	result, err := s.db.Exec(`
		UPDATE escalations 
		SET status = 'dismissed', resolved_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'open'
	`, id)

	if err != nil {
		return fmt.Errorf("에스컬레이션 무시 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("에스컬레이션 #%d을(를) 찾을 수 없거나 이미 처리됨", id)
	}

	return nil
}

// Get retrieves an escalation by ID
func (s *Service) Get(id int64) (*Escalation, error) {
	var e Escalation
	err := s.db.QueryRow(`
		SELECT id, from_session, from_port, issue, status, created_at, resolved_at
		FROM escalations WHERE id = ?
	`, id).Scan(&e.ID, &e.FromSession, &e.FromPort, &e.Issue, &e.Status, &e.CreatedAt, &e.ResolvedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("에스컬레이션 #%d을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// List returns escalations with optional filters
func (s *Service) List(status string, limit int) ([]Escalation, error) {
	query := `
		SELECT id, from_session, from_port, issue, status, created_at, resolved_at
		FROM escalations
	`

	var args []interface{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}

	query += ` ORDER BY created_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("에스컬레이션 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var escalations []Escalation
	for rows.Next() {
		var e Escalation
		if err := rows.Scan(&e.ID, &e.FromSession, &e.FromPort, &e.Issue, &e.Status, &e.CreatedAt, &e.ResolvedAt); err != nil {
			return nil, err
		}
		escalations = append(escalations, e)
	}

	return escalations, nil
}

// OpenCount returns the number of open escalations
func (s *Service) OpenCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM escalations WHERE status = 'open'`).Scan(&count)
	return count, err
}

// Summary returns escalation statistics
func (s *Service) Summary() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM escalations GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		summary[status] = count
	}

	return summary, nil
}
