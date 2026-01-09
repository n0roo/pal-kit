package pipeline

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Pipeline represents a port execution pipeline
type Pipeline struct {
	ID          string
	Name        string
	SessionID   sql.NullString
	Status      string
	CreatedAt   time.Time
	StartedAt   sql.NullTime
	CompletedAt sql.NullTime
}

// PipelinePort represents a port in a pipeline
type PipelinePort struct {
	PipelineID string
	PortID     string
	GroupOrder int
	Status     string
}

// PortDependency represents a dependency between ports
type PortDependency struct {
	PortID    string
	DependsOn string
}

// Status constants
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusComplete  = "complete"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusSkipped   = "skipped"
)

// Service handles pipeline operations
type Service struct {
	db *db.DB
}

// NewService creates a new pipeline service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Create creates a new pipeline
func (s *Service) Create(id, name, sessionID string) error {
	var sessionNull sql.NullString
	if sessionID != "" {
		sessionNull = sql.NullString{String: sessionID, Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO pipelines (id, name, session_id, status)
		VALUES (?, ?, ?, 'pending')
	`, id, name, sessionNull)

	if err != nil {
		return fmt.Errorf("파이프라인 생성 실패: %w", err)
	}
	return nil
}

// AddPort adds a port to a pipeline
func (s *Service) AddPort(pipelineID, portID string, groupOrder int) error {
	_, err := s.db.Exec(`
		INSERT INTO pipeline_ports (pipeline_id, port_id, group_order, status)
		VALUES (?, ?, ?, 'pending')
	`, pipelineID, portID, groupOrder)

	if err != nil {
		return fmt.Errorf("포트 추가 실패: %w", err)
	}
	return nil
}

// AddDependency adds a dependency between ports
func (s *Service) AddDependency(portID, dependsOn string) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO port_dependencies (port_id, depends_on)
		VALUES (?, ?)
	`, portID, dependsOn)

	if err != nil {
		return fmt.Errorf("의존성 추가 실패: %w", err)
	}
	return nil
}

// Get retrieves a pipeline by ID
func (s *Service) Get(id string) (*Pipeline, error) {
	var p Pipeline
	err := s.db.QueryRow(`
		SELECT id, name, session_id, status, created_at, started_at, completed_at
		FROM pipelines WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.SessionID, &p.Status, &p.CreatedAt, &p.StartedAt, &p.CompletedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("파이프라인 '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// List returns all pipelines
func (s *Service) List(status string, limit int) ([]Pipeline, error) {
	query := `SELECT id, name, session_id, status, created_at, started_at, completed_at FROM pipelines`
	
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
		return nil, err
	}
	defer rows.Close()

	var pipelines []Pipeline
	for rows.Next() {
		var p Pipeline
		if err := rows.Scan(&p.ID, &p.Name, &p.SessionID, &p.Status, &p.CreatedAt, &p.StartedAt, &p.CompletedAt); err != nil {
			return nil, err
		}
		pipelines = append(pipelines, p)
	}
	return pipelines, nil
}

// GetPorts returns ports in a pipeline ordered by group
func (s *Service) GetPorts(pipelineID string) ([]PipelinePort, error) {
	rows, err := s.db.Query(`
		SELECT pipeline_id, port_id, group_order, status
		FROM pipeline_ports
		WHERE pipeline_id = ?
		ORDER BY group_order, port_id
	`, pipelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []PipelinePort
	for rows.Next() {
		var pp PipelinePort
		if err := rows.Scan(&pp.PipelineID, &pp.PortID, &pp.GroupOrder, &pp.Status); err != nil {
			return nil, err
		}
		ports = append(ports, pp)
	}
	return ports, nil
}

// GetGroups returns ports grouped by group_order
func (s *Service) GetGroups(pipelineID string) (map[int][]PipelinePort, error) {
	ports, err := s.GetPorts(pipelineID)
	if err != nil {
		return nil, err
	}

	groups := make(map[int][]PipelinePort)
	for _, p := range ports {
		groups[p.GroupOrder] = append(groups[p.GroupOrder], p)
	}
	return groups, nil
}

// GetDependencies returns dependencies for a port
func (s *Service) GetDependencies(portID string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT depends_on FROM port_dependencies WHERE port_id = ?
	`, portID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []string
	for rows.Next() {
		var dep string
		if err := rows.Scan(&dep); err != nil {
			return nil, err
		}
		deps = append(deps, dep)
	}
	return deps, nil
}

// UpdateStatus updates pipeline status
func (s *Service) UpdateStatus(id, status string) error {
	var query string
	switch status {
	case StatusRunning:
		query = `UPDATE pipelines SET status = ?, started_at = CURRENT_TIMESTAMP WHERE id = ?`
	case StatusComplete, StatusFailed, StatusCancelled:
		query = `UPDATE pipelines SET status = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`
	default:
		query = `UPDATE pipelines SET status = ? WHERE id = ?`
	}

	_, err := s.db.Exec(query, status, id)
	return err
}

// UpdatePortStatus updates a port status within a pipeline
func (s *Service) UpdatePortStatus(pipelineID, portID, status string) error {
	_, err := s.db.Exec(`
		UPDATE pipeline_ports SET status = ? WHERE pipeline_id = ? AND port_id = ?
	`, status, pipelineID, portID)
	return err
}

// GetProgress returns pipeline progress
func (s *Service) GetProgress(pipelineID string) (completed, total int, err error) {
	err = s.db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN status = 'complete' THEN 1 END),
			COUNT(*)
		FROM pipeline_ports WHERE pipeline_id = ?
	`, pipelineID).Scan(&completed, &total)
	return
}

// Delete removes a pipeline
func (s *Service) Delete(id string) error {
	// 파이프라인 포트 먼저 삭제
	s.db.Exec(`DELETE FROM pipeline_ports WHERE pipeline_id = ?`, id)
	
	_, err := s.db.Exec(`DELETE FROM pipelines WHERE id = ?`, id)
	return err
}

// RemovePort removes a port from a pipeline
func (s *Service) RemovePort(pipelineID, portID string) error {
	_, err := s.db.Exec(`
		DELETE FROM pipeline_ports WHERE pipeline_id = ? AND port_id = ?
	`, pipelineID, portID)
	return err
}

// CanExecutePort checks if all dependencies are complete
func (s *Service) CanExecutePort(pipelineID, portID string) (bool, []string, error) {
	deps, err := s.GetDependencies(portID)
	if err != nil {
		return false, nil, err
	}

	if len(deps) == 0 {
		return true, nil, nil
	}

	// 각 의존성의 상태 확인
	var pending []string
	for _, dep := range deps {
		var status string
		err := s.db.QueryRow(`
			SELECT status FROM pipeline_ports WHERE pipeline_id = ? AND port_id = ?
		`, pipelineID, dep).Scan(&status)
		
		if err != nil || status != StatusComplete {
			pending = append(pending, dep)
		}
	}

	return len(pending) == 0, pending, nil
}
