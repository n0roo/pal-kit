package port

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Port represents a work unit specification
type Port struct {
	ID           string
	Title        sql.NullString
	Status       string
	SessionID    sql.NullString
	FilePath     sql.NullString
	CreatedAt    time.Time
	StartedAt    sql.NullTime
	CompletedAt  sql.NullTime
	InputTokens  int64
	OutputTokens int64
	CostUSD      float64
	DurationSecs int64
	AgentID      sql.NullString
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
		SELECT id, title, status, session_id, file_path, created_at, started_at, completed_at,
		       input_tokens, output_tokens, cost_usd, duration_secs, agent_id
		FROM ports WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Title, &p.Status, &p.SessionID, &p.FilePath,
		&p.CreatedAt, &p.StartedAt, &p.CompletedAt,
		&p.InputTokens, &p.OutputTokens, &p.CostUSD, &p.DurationSecs, &p.AgentID,
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
		SELECT id, title, status, session_id, file_path, created_at, started_at, completed_at,
		       input_tokens, output_tokens, cost_usd, duration_secs, agent_id
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
			&p.InputTokens, &p.OutputTokens, &p.CostUSD, &p.DurationSecs, &p.AgentID,
		); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}

	return ports, nil
}

// ListBySession returns ports associated with a specific session
func (s *Service) ListBySession(sessionID string) ([]Port, error) {
	rows, err := s.db.Query(`
		SELECT id, title, status, session_id, file_path, created_at, started_at, completed_at,
		       input_tokens, output_tokens, cost_usd, duration_secs, agent_id
		FROM ports
		WHERE session_id = ?
		ORDER BY created_at DESC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("세션별 포트 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var ports []Port
	for rows.Next() {
		var p Port
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Status, &p.SessionID, &p.FilePath,
			&p.CreatedAt, &p.StartedAt, &p.CompletedAt,
			&p.InputTokens, &p.OutputTokens, &p.CostUSD, &p.DurationSecs, &p.AgentID,
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

// UpdateUsage updates token usage and cost for a port
func (s *Service) UpdateUsage(id string, inputTokens, outputTokens int64, cost float64) error {
	result, err := s.db.Exec(`
		UPDATE ports
		SET input_tokens = input_tokens + ?,
		    output_tokens = output_tokens + ?,
		    cost_usd = cost_usd + ?
		WHERE id = ?
	`, inputTokens, outputTokens, cost, id)

	if err != nil {
		return fmt.Errorf("사용량 업데이트 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", id)
	}

	return nil
}

// SetUsage sets absolute token usage and cost for a port
func (s *Service) SetUsage(id string, inputTokens, outputTokens int64, cost float64, durationSecs int64) error {
	result, err := s.db.Exec(`
		UPDATE ports
		SET input_tokens = ?,
		    output_tokens = ?,
		    cost_usd = ?,
		    duration_secs = ?
		WHERE id = ?
	`, inputTokens, outputTokens, cost, durationSecs, id)

	if err != nil {
		return fmt.Errorf("사용량 설정 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", id)
	}

	return nil
}

// SetAgentID sets the agent ID for a port
func (s *Service) SetAgentID(id, agentID string) error {
	result, err := s.db.Exec(`UPDATE ports SET agent_id = ? WHERE id = ?`, agentID, id)
	if err != nil {
		return fmt.Errorf("에이전트 ID 설정 실패: %w", err)
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

// SetDuration sets the duration in seconds for a port
func (s *Service) SetDuration(id string, durationSecs int64) error {
	result, err := s.db.Exec(`UPDATE ports SET duration_secs = ? WHERE id = ?`, durationSecs, id)
	if err != nil {
		return fmt.Errorf("duration 설정 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", id)
	}

	return nil
}

// RecordStart records port start with all related info
func (s *Service) RecordStart(portID, sessionID, agentID string) error {
	query := `
		UPDATE ports
		SET status = 'running',
		    started_at = CURRENT_TIMESTAMP,
		    session_id = ?,
		    agent_id = ?
		WHERE id = ?
	`

	var sessionIDNull, agentIDNull sql.NullString
	if sessionID != "" {
		sessionIDNull = sql.NullString{String: sessionID, Valid: true}
	}
	if agentID != "" {
		agentIDNull = sql.NullString{String: agentID, Valid: true}
	}

	result, err := s.db.Exec(query, sessionIDNull, agentIDNull, portID)
	if err != nil {
		return fmt.Errorf("포트 시작 기록 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", portID)
	}

	return nil
}

// RecordCompletion records port completion with usage stats
func (s *Service) RecordCompletion(portID string, inputTokens, outputTokens int64, cost float64) error {
	// Get port to calculate duration
	p, err := s.Get(portID)
	if err != nil {
		return err
	}

	var durationSecs int64
	if p.StartedAt.Valid {
		durationSecs = int64(time.Since(p.StartedAt.Time).Seconds())
	}

	result, err := s.db.Exec(`
		UPDATE ports
		SET status = 'complete',
		    completed_at = CURRENT_TIMESTAMP,
		    input_tokens = ?,
		    output_tokens = ?,
		    cost_usd = ?,
		    duration_secs = ?
		WHERE id = ?
	`, inputTokens, outputTokens, cost, durationSecs, portID)

	if err != nil {
		return fmt.Errorf("포트 완료 기록 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", portID)
	}

	return nil
}

// GetBySession returns ports associated with a session
func (s *Service) GetBySession(sessionID string) ([]Port, error) {
	rows, err := s.db.Query(`
		SELECT id, title, status, session_id, file_path, created_at, started_at, completed_at,
		       input_tokens, output_tokens, cost_usd, duration_secs, agent_id
		FROM ports
		WHERE session_id = ?
		ORDER BY started_at DESC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("세션별 포트 조회 실패: %w", err)
	}
	defer rows.Close()

	var ports []Port
	for rows.Next() {
		var p Port
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Status, &p.SessionID, &p.FilePath,
			&p.CreatedAt, &p.StartedAt, &p.CompletedAt,
			&p.InputTokens, &p.OutputTokens, &p.CostUSD, &p.DurationSecs, &p.AgentID,
		); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}

	return ports, nil
}

// PortStats represents port statistics including usage
type PortStats struct {
	TotalPorts      int     `json:"total_ports"`
	PendingPorts    int     `json:"pending_ports"`
	RunningPorts    int     `json:"running_ports"`
	CompletedPorts  int     `json:"completed_ports"`
	FailedPorts     int     `json:"failed_ports"`
	TotalInputTokens  int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalCostUSD    float64 `json:"total_cost_usd"`
	TotalDurationSecs int64   `json:"total_duration_secs"`
	AvgDurationSecs float64 `json:"avg_duration_secs"`
}

// GetStats returns comprehensive port statistics
func (s *Service) GetStats() (*PortStats, error) {
	stats := &PortStats{}

	// Count by status
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as running,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
		FROM ports
	`).Scan(&stats.TotalPorts, &stats.PendingPorts, &stats.RunningPorts,
		&stats.CompletedPorts, &stats.FailedPorts)
	if err != nil {
		return nil, err
	}

	// Sum usage
	err = s.db.QueryRow(`
		SELECT
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(cost_usd), 0),
			COALESCE(SUM(duration_secs), 0)
		FROM ports
	`).Scan(&stats.TotalInputTokens, &stats.TotalOutputTokens,
		&stats.TotalCostUSD, &stats.TotalDurationSecs)
	if err != nil {
		return nil, err
	}

	// Calculate average duration for completed ports
	if stats.CompletedPorts > 0 {
		err = s.db.QueryRow(`
			SELECT COALESCE(AVG(duration_secs), 0)
			FROM ports
			WHERE status = 'complete' AND duration_secs > 0
		`).Scan(&stats.AvgDurationSecs)
		if err != nil {
			stats.AvgDurationSecs = 0
		}
	}

	return stats, nil
}

// GetRecentCompleted returns recently completed ports
func (s *Service) GetRecentCompleted(limit int) ([]Port, error) {
	query := `
		SELECT id, title, status, session_id, file_path, created_at, started_at, completed_at,
		       input_tokens, output_tokens, cost_usd, duration_secs, agent_id
		FROM ports
		WHERE status = 'complete'
		ORDER BY completed_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("최근 완료 포트 조회 실패: %w", err)
	}
	defer rows.Close()

	var ports []Port
	for rows.Next() {
		var p Port
		if err := rows.Scan(
			&p.ID, &p.Title, &p.Status, &p.SessionID, &p.FilePath,
			&p.CreatedAt, &p.StartedAt, &p.CompletedAt,
			&p.InputTokens, &p.OutputTokens, &p.CostUSD, &p.DurationSecs, &p.AgentID,
		); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}

	return ports, nil
}
