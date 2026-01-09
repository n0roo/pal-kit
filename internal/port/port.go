package port

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Port represents a work unit specification
type Port struct {
	ID          string
	Title       sql.NullString
	Status      string
	SessionID   sql.NullString
	FilePath    sql.NullString
	CreatedAt   time.Time
	StartedAt   sql.NullTime
	CompletedAt sql.NullTime
}

// Status constants
const (
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusComplete = "complete"
	StatusFailed   = "failed"
	StatusBlocked  = "blocked"
)

// ValidStatuses lists all valid port statuses
var ValidStatuses = []string{StatusPending, StatusRunning, StatusComplete, StatusFailed, StatusBlocked}

// Service handles port operations
type Service struct {
	db *db.DB
}

// NewService creates a new port service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Create creates a new port
func (s *Service) Create(id, title, filePath string) error {
	var titleNull, filePathNull sql.NullString

	if title != "" {
		titleNull = sql.NullString{String: title, Valid: true}
	}
	if filePath != "" {
		filePathNull = sql.NullString{String: filePath, Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO ports (id, title, file_path, status)
		VALUES (?, ?, ?, 'pending')
	`, id, titleNull, filePathNull)

	if err != nil {
		return fmt.Errorf("포트 생성 실패: %w", err)
	}
	return nil
}

// UpdateStatus updates port status
func (s *Service) UpdateStatus(id, status string) error {
	// 상태 유효성 검사
	valid := false
	for _, s := range ValidStatuses {
		if s == status {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("유효하지 않은 상태: %s (가능: %v)", status, ValidStatuses)
	}

	// 상태에 따른 추가 필드 업데이트
	var query string
	switch status {
	case StatusRunning:
		query = `UPDATE ports SET status = ?, started_at = CURRENT_TIMESTAMP WHERE id = ?`
	case StatusComplete, StatusFailed:
		query = `UPDATE ports SET status = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`
	default:
		query = `UPDATE ports SET status = ? WHERE id = ?`
	}

	result, err := s.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("상태 업데이트 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", id)
	}

	return nil
}

// AssignSession assigns a session to a port
func (s *Service) AssignSession(portID, sessionID string) error {
	result, err := s.db.Exec(`
		UPDATE ports SET session_id = ? WHERE id = ?
	`, sessionID, portID)

	if err != nil {
		return fmt.Errorf("세션 할당 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", portID)
	}

	return nil
}

// Get retrieves a port by ID
func (s *Service) Get(id string) (*Port, error) {
	var p Port
	err := s.db.QueryRow(`
		SELECT id, title, status, session_id, file_path, created_at, started_at, completed_at
		FROM ports WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Title, &p.Status, &p.SessionID, &p.FilePath,
		&p.CreatedAt, &p.StartedAt, &p.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// List returns ports with optional filters
func (s *Service) List(status string, limit int) ([]Port, error) {
	query := `
		SELECT id, title, status, session_id, file_path, created_at, started_at, completed_at
		FROM ports
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
		return nil, fmt.Errorf("포트 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var ports []Port
	for rows.Next() {
		var p Port
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Status, &p.SessionID, &p.FilePath,
			&p.CreatedAt, &p.StartedAt, &p.CompletedAt,
		); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}

	return ports, nil
}

// Delete removes a port
func (s *Service) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM ports WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("포트 삭제 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", id)
	}

	return nil
}

// Summary returns port statistics
func (s *Service) Summary() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM ports GROUP BY status`)
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
