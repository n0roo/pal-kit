package history

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Event represents a history event
type Event struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	EventType   string    `json:"event_type"`
	EventData   string    `json:"event_data"`
	CreatedAt   time.Time `json:"created_at"`
	// Joined fields from sessions
	ProjectRoot string `json:"project_root,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
}

// EventDetail represents parsed event with details
type EventDetail struct {
	Event
	ParsedData map[string]interface{} `json:"parsed_data,omitempty"`
	Status     string                 `json:"status"` // success, error, warning, info
}

// Filter represents query filters for history
type Filter struct {
	SessionID   string    `json:"session_id,omitempty"`
	EventType   string    `json:"event_type,omitempty"`
	ProjectRoot string    `json:"project_root,omitempty"`
	Status      string    `json:"status,omitempty"`
	StartDate   time.Time `json:"start_date,omitempty"`
	EndDate     time.Time `json:"end_date,omitempty"`
	Search      string    `json:"search,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Offset      int       `json:"offset,omitempty"`
}

// Service handles history operations
type Service struct {
	db *db.DB
}

// NewService creates a new history service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// List returns events with optional filters
func (s *Service) List(filter Filter) ([]EventDetail, int, error) {
	// Build query
	query := `
		SELECT
			e.id, e.session_id, e.event_type, COALESCE(e.event_data, ''), e.created_at,
			COALESCE(s.project_root, ''), COALESCE(s.project_name, '')
		FROM session_events e
		LEFT JOIN sessions s ON e.session_id = s.id
		WHERE 1=1
	`
	countQuery := `
		SELECT COUNT(*)
		FROM session_events e
		LEFT JOIN sessions s ON e.session_id = s.id
		WHERE 1=1
	`

	var args []interface{}
	var conditions []string

	if filter.SessionID != "" {
		conditions = append(conditions, "e.session_id = ?")
		args = append(args, filter.SessionID)
	}

	if filter.EventType != "" {
		conditions = append(conditions, "e.event_type = ?")
		args = append(args, filter.EventType)
	}

	if filter.ProjectRoot != "" {
		// Support both project_root (full path) and project_name (short name)
		conditions = append(conditions, "(s.project_root = ? OR s.project_name = ?)")
		args = append(args, filter.ProjectRoot, filter.ProjectRoot)
	}

	if !filter.StartDate.IsZero() {
		conditions = append(conditions, "e.created_at >= ?")
		args = append(args, filter.StartDate.Format("2006-01-02 15:04:05"))
	}

	if !filter.EndDate.IsZero() {
		conditions = append(conditions, "e.created_at <= ?")
		args = append(args, filter.EndDate.Format("2006-01-02 15:04:05"))
	}

	if filter.Search != "" {
		conditions = append(conditions, "(e.event_type LIKE ? OR e.event_data LIKE ? OR e.session_id LIKE ?)")
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Add conditions to queries
	if len(conditions) > 0 {
		condStr := " AND " + strings.Join(conditions, " AND ")
		query += condStr
		countQuery += condStr
	}

	// Get total count
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("카운트 조회 실패: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY e.created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	} else {
		query += " LIMIT 100" // default limit
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	// Execute query
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("이벤트 조회 실패: %w", err)
	}
	defer rows.Close()

	var events []EventDetail
	for rows.Next() {
		var e Event
		var projectRoot, projectName sql.NullString

		if err := rows.Scan(
			&e.ID, &e.SessionID, &e.EventType, &e.EventData, &e.CreatedAt,
			&projectRoot, &projectName,
		); err != nil {
			return nil, 0, err
		}

		if projectRoot.Valid {
			e.ProjectRoot = projectRoot.String
		}
		if projectName.Valid {
			e.ProjectName = projectName.String
		}

		detail := EventDetail{Event: e}

		// Parse event_data JSON
		if e.EventData != "" {
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(e.EventData), &parsed); err == nil {
				detail.ParsedData = parsed
			}
		}

		// Determine status based on event type
		detail.Status = determineStatus(e.EventType, detail.ParsedData)

		events = append(events, detail)
	}

	return events, total, nil
}

// GetEventTypes returns all unique event types
func (s *Service) GetEventTypes() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT event_type FROM session_events ORDER BY event_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, nil
}

// GetProjects returns all unique projects from sessions
func (s *Service) GetProjects() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT project_name
		FROM sessions
		WHERE project_name IS NOT NULL AND project_name != ''
		ORDER BY project_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// GetStats returns history statistics
func (s *Service) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total events
	var total int
	s.db.QueryRow(`SELECT COUNT(*) FROM session_events`).Scan(&total)
	stats["total_events"] = total

	// Events by type
	rows, err := s.db.Query(`
		SELECT event_type, COUNT(*)
		FROM session_events
		GROUP BY event_type
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byType := make(map[string]int)
	for rows.Next() {
		var t string
		var count int
		rows.Scan(&t, &count)
		byType[t] = count
	}
	stats["by_type"] = byType

	// Events today
	var today int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM session_events
		WHERE DATE(created_at) = DATE('now')
	`).Scan(&today)
	stats["today"] = today

	return stats, nil
}

// determineStatus determines event status based on type and data
func determineStatus(eventType string, data map[string]interface{}) string {
	switch eventType {
	case "session_start":
		return "info"
	case "session_end":
		if data != nil {
			if reason, ok := data["reason"].(string); ok {
				if reason == "error" {
					return "error"
				}
			}
		}
		return "success"
	case "compact":
		return "info"
	case "port_start":
		return "info"
	case "port_end":
		return "success"
	case "error":
		return "error"
	case "warning":
		return "warning"
	default:
		return "info"
	}
}

// ExportJSON exports events to JSON format
func (s *Service) ExportJSON(filter Filter) ([]byte, error) {
	events, _, err := s.List(filter)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(events, "", "  ")
}

// ExportCSV exports events to CSV format
func (s *Service) ExportCSV(filter Filter) (string, error) {
	events, _, err := s.List(filter)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("id,session_id,event_type,status,project,created_at,event_data\n")

	for _, e := range events {
		// Escape CSV fields
		eventData := strings.ReplaceAll(e.EventData, "\"", "\"\"")
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s,%s,\"%s\"\n",
			e.ID,
			e.SessionID,
			e.EventType,
			e.Status,
			e.ProjectName,
			e.CreatedAt.Format(time.RFC3339),
			eventData,
		))
	}

	return sb.String(), nil
}
