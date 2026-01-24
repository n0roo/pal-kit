package escalation

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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

// ========================================
// v10 Extensions - Enhanced Escalation
// ========================================

// EscalationType defines the type of escalation
type EscalationType string

const (
	TypeBuildFail      EscalationType = "build_fail"
	TypeTestFail       EscalationType = "test_fail"
	TypeBlocked        EscalationType = "blocked"
	TypeQuestion       EscalationType = "question"
	TypeTokenExceeded  EscalationType = "token_exceeded"
	TypeCompactWarning EscalationType = "compact_warning"
	TypeDependencyLoop EscalationType = "dependency_loop"
	TypeManualReview   EscalationType = "manual_review"
)

// Severity defines the severity of an escalation
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// EnhancedEscalation represents an enhanced escalation with v10 fields
type EnhancedEscalation struct {
	ID          string         `json:"id"`
	FromSession string         `json:"from_session"`
	ToSession   string         `json:"to_session,omitempty"`
	FromPort    string         `json:"from_port,omitempty"`
	Type        EscalationType `json:"type"`
	Severity    Severity       `json:"severity"`
	Issue       string         `json:"issue"`
	Context     interface{}    `json:"context,omitempty"`
	Suggestion  string         `json:"suggestion,omitempty"`
	Status      string         `json:"status"`
	Resolution  string         `json:"resolution,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
}

// CreateEnhanced creates an enhanced escalation
func (s *Service) CreateEnhanced(opts EnhancedEscalationOptions) (*EnhancedEscalation, error) {
	id := uuid.New().String()
	now := time.Now()

	contextJSON, _ := json.Marshal(opts.Context)

	_, err := s.db.Exec(`
		INSERT INTO escalations (
			id, from_session, to_session, from_port, type, severity,
			issue, context, suggestion, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'open', ?)
	`, id, opts.FromSession, nullableString(opts.ToSession), nullableString(opts.FromPort),
		opts.Type, opts.Severity, opts.Issue, string(contextJSON), opts.Suggestion, now)

	if err != nil {
		return nil, fmt.Errorf("에스컬레이션 생성 실패: %w", err)
	}

	return &EnhancedEscalation{
		ID:          id,
		FromSession: opts.FromSession,
		ToSession:   opts.ToSession,
		FromPort:    opts.FromPort,
		Type:        opts.Type,
		Severity:    opts.Severity,
		Issue:       opts.Issue,
		Context:     opts.Context,
		Suggestion:  opts.Suggestion,
		Status:      StatusOpen,
		CreatedAt:   now,
	}, nil
}

// EnhancedEscalationOptions holds options for creating enhanced escalation
type EnhancedEscalationOptions struct {
	FromSession string
	ToSession   string
	FromPort    string
	Type        EscalationType
	Severity    Severity
	Issue       string
	Context     interface{}
	Suggestion  string
}

// GetEnhanced retrieves an enhanced escalation by ID
func (s *Service) GetEnhanced(id string) (*EnhancedEscalation, error) {
	var e EnhancedEscalation
	var toSession, fromPort, contextJSON, suggestion, resolution sql.NullString
	var resolvedAt sql.NullTime
	var escType, severity sql.NullString

	// Try new schema first
	err := s.db.QueryRow(`
		SELECT id, from_session, to_session, from_port, type, severity,
		       issue, context, suggestion, status, resolution, created_at, resolved_at
		FROM escalations WHERE id = ?
	`, id).Scan(&e.ID, &e.FromSession, &toSession, &fromPort, &escType, &severity,
		&e.Issue, &contextJSON, &suggestion, &e.Status, &resolution, &e.CreatedAt, &resolvedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("에스컬레이션 '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	if toSession.Valid {
		e.ToSession = toSession.String
	}
	if fromPort.Valid {
		e.FromPort = fromPort.String
	}
	if escType.Valid {
		e.Type = EscalationType(escType.String)
	}
	if severity.Valid {
		e.Severity = Severity(severity.String)
	}
	if contextJSON.Valid {
		json.Unmarshal([]byte(contextJSON.String), &e.Context)
	}
	if suggestion.Valid {
		e.Suggestion = suggestion.String
	}
	if resolution.Valid {
		e.Resolution = resolution.String
	}
	if resolvedAt.Valid {
		e.ResolvedAt = &resolvedAt.Time
	}

	return &e, nil
}

// ResolveEnhanced resolves an enhanced escalation with resolution
func (s *Service) ResolveEnhanced(id, resolution string) error {
	result, err := s.db.Exec(`
		UPDATE escalations 
		SET status = 'resolved', resolution = ?, resolved_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'open'
	`, resolution, id)

	if err != nil {
		return fmt.Errorf("에스컬레이션 해결 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("에스컬레이션 '%s'을(를) 찾을 수 없거나 이미 처리됨", id)
	}

	return nil
}

// ListBySession returns escalations for a session
func (s *Service) ListBySession(sessionID string, limit int) ([]*EnhancedEscalation, error) {
	query := `
		SELECT id, from_session, to_session, from_port, type, severity,
		       issue, context, suggestion, status, resolution, created_at, resolved_at
		FROM escalations
		WHERE from_session = ? OR to_session = ?
		ORDER BY created_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, sessionID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanEnhancedEscalations(rows)
}

// ListByType returns escalations of a specific type
func (s *Service) ListByType(escType EscalationType, status string, limit int) ([]*EnhancedEscalation, error) {
	query := `
		SELECT id, from_session, to_session, from_port, type, severity,
		       issue, context, suggestion, status, resolution, created_at, resolved_at
		FROM escalations
		WHERE type = ?
	`
	args := []interface{}{escType}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanEnhancedEscalations(rows)
}

// ListBySeverity returns escalations of a specific severity or higher
func (s *Service) ListBySeverity(minSeverity Severity, limit int) ([]*EnhancedEscalation, error) {
	severityOrder := map[Severity]int{
		SeverityLow:      1,
		SeverityMedium:   2,
		SeverityHigh:     3,
		SeverityCritical: 4,
	}

	minOrder := severityOrder[minSeverity]

	query := `
		SELECT id, from_session, to_session, from_port, type, severity,
		       issue, context, suggestion, status, resolution, created_at, resolved_at
		FROM escalations
		WHERE status = 'open'
		ORDER BY 
			CASE severity
				WHEN 'critical' THEN 4
				WHEN 'high' THEN 3
				WHEN 'medium' THEN 2
				WHEN 'low' THEN 1
				ELSE 0
			END DESC,
			created_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	escalations, err := s.scanEnhancedEscalations(rows)
	if err != nil {
		return nil, err
	}

	// Filter by minimum severity
	var filtered []*EnhancedEscalation
	for _, e := range escalations {
		if severityOrder[e.Severity] >= minOrder {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

func (s *Service) scanEnhancedEscalations(rows *sql.Rows) ([]*EnhancedEscalation, error) {
	var escalations []*EnhancedEscalation
	for rows.Next() {
		var e EnhancedEscalation
		var toSession, fromPort, contextJSON, suggestion, resolution sql.NullString
		var resolvedAt sql.NullTime
		var escType, severity sql.NullString

		err := rows.Scan(&e.ID, &e.FromSession, &toSession, &fromPort, &escType, &severity,
			&e.Issue, &contextJSON, &suggestion, &e.Status, &resolution, &e.CreatedAt, &resolvedAt)
		if err != nil {
			continue
		}

		if toSession.Valid {
			e.ToSession = toSession.String
		}
		if fromPort.Valid {
			e.FromPort = fromPort.String
		}
		if escType.Valid {
			e.Type = EscalationType(escType.String)
		}
		if severity.Valid {
			e.Severity = Severity(severity.String)
		}
		if contextJSON.Valid {
			json.Unmarshal([]byte(contextJSON.String), &e.Context)
		}
		if suggestion.Valid {
			e.Suggestion = suggestion.String
		}
		if resolution.Valid {
			e.Resolution = resolution.String
		}
		if resolvedAt.Valid {
			e.ResolvedAt = &resolvedAt.Time
		}

		escalations = append(escalations, &e)
	}

	return escalations, nil
}

// EscalationStats represents escalation statistics
type EscalationStats struct {
	Total     int            `json:"total"`
	Open      int            `json:"open"`
	Resolved  int            `json:"resolved"`
	Dismissed int            `json:"dismissed"`
	ByType    map[string]int `json:"by_type"`
	BySeverity map[string]int `json:"by_severity"`
}

// GetStats returns comprehensive escalation statistics
func (s *Service) GetStats() (*EscalationStats, error) {
	stats := &EscalationStats{
		ByType:     make(map[string]int),
		BySeverity: make(map[string]int),
	}

	// Count by status
	err := s.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END) as open,
			SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END) as resolved,
			SUM(CASE WHEN status = 'dismissed' THEN 1 ELSE 0 END) as dismissed
		FROM escalations
	`).Scan(&stats.Total, &stats.Open, &stats.Resolved, &stats.Dismissed)
	if err != nil {
		return nil, err
	}

	// Count by type
	rows, err := s.db.Query(`SELECT COALESCE(type, 'unknown'), COUNT(*) FROM escalations GROUP BY type`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var count int
			if rows.Scan(&t, &count) == nil {
				stats.ByType[t] = count
			}
		}
	}

	// Count by severity
	rows, err = s.db.Query(`SELECT COALESCE(severity, 'unknown'), COUNT(*) FROM escalations GROUP BY severity`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sev string
			var count int
			if rows.Scan(&sev, &count) == nil {
				stats.BySeverity[sev] = count
			}
		}
	}

	return stats, nil
}

func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
