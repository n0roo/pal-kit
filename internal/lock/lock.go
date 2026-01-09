package lock

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Lock represents a resource lock
type Lock struct {
	Resource   string
	SessionID  string
	AcquiredAt time.Time
}

// Service handles lock operations
type Service struct {
	db *db.DB
}

// NewService creates a new lock service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Acquire attempts to acquire a lock on a resource
// Returns nil if successful, error if already locked or failed
func (s *Service) Acquire(resource, sessionID string) error {
	// 이미 잠겨있는지 확인
	var existing string
	err := s.db.QueryRow(`SELECT session_id FROM locks WHERE resource = ?`, resource).Scan(&existing)
	
	if err == nil {
		// 이미 잠김
		return fmt.Errorf("리소스 '%s'는 세션 '%s'에 의해 잠겨있습니다", resource, existing)
	}
	
	if err != sql.ErrNoRows {
		return fmt.Errorf("Lock 확인 실패: %w", err)
	}

	// Lock 획득
	_, err = s.db.Exec(`INSERT INTO locks (resource, session_id) VALUES (?, ?)`, resource, sessionID)
	if err != nil {
		return fmt.Errorf("Lock 획득 실패: %w", err)
	}

	return nil
}

// Release releases a lock on a resource
func (s *Service) Release(resource string) error {
	result, err := s.db.Exec(`DELETE FROM locks WHERE resource = ?`, resource)
	if err != nil {
		return fmt.Errorf("Lock 해제 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("리소스 '%s'에 대한 Lock이 없습니다", resource)
	}

	return nil
}

// List returns all active locks
func (s *Service) List() ([]Lock, error) {
	rows, err := s.db.Query(`SELECT resource, session_id, acquired_at FROM locks ORDER BY acquired_at`)
	if err != nil {
		return nil, fmt.Errorf("Lock 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var locks []Lock
	for rows.Next() {
		var l Lock
		if err := rows.Scan(&l.Resource, &l.SessionID, &l.AcquiredAt); err != nil {
			return nil, err
		}
		locks = append(locks, l)
	}

	return locks, nil
}

// Clear removes all locks (force cleanup)
func (s *Service) Clear() (int64, error) {
	result, err := s.db.Exec(`DELETE FROM locks`)
	if err != nil {
		return 0, fmt.Errorf("Lock 정리 실패: %w", err)
	}
	return result.RowsAffected()
}

// IsLocked checks if a resource is locked
func (s *Service) IsLocked(resource string) (bool, string, error) {
	var sessionID string
	err := s.db.QueryRow(`SELECT session_id FROM locks WHERE resource = ?`, resource).Scan(&sessionID)
	
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	
	return true, sessionID, nil
}
