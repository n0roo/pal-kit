package operator

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/session"
)

// Service provides operator functionality for session management
type Service struct {
	db          *db.DB
	projectRoot string
}

// Briefing represents a session start briefing
type Briefing struct {
	GeneratedAt    time.Time        `json:"generated_at"`
	ProjectName    string           `json:"project_name"`
	Summary        string           `json:"summary"`
	RecentSessions []SessionSummary `json:"recent_sessions"`
	RunningPorts   []PortSummary    `json:"running_ports"`
	PendingPorts   []PortSummary    `json:"pending_ports"`
	Escalations    []Escalation     `json:"escalations"`
	Recommendations []string        `json:"recommendations"`
}

// Summary represents a session end summary
type Summary struct {
	SessionID      string         `json:"session_id"`
	GeneratedAt    time.Time      `json:"generated_at"`
	Duration       time.Duration  `json:"duration"`
	DurationStr    string         `json:"duration_str"`
	PortsCompleted []PortSummary  `json:"ports_completed"`
	PortsStarted   []PortSummary  `json:"ports_started"`
	Events         []EventSummary `json:"events"`
	Usage          UsageSummary   `json:"usage"`
	ADRCandidates  []ADRCandidate `json:"adr_candidates"`
}

// SessionSummary is a brief session info
type SessionSummary struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	DurationStr string    `json:"duration_str"`
	PortCount   int       `json:"port_count"`
}

// PortSummary is a brief port info
type PortSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at,omitempty"`
}

// Escalation represents an active escalation
type Escalation struct {
	ID        int64     `json:"id"`
	PortID    string    `json:"port_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// EventSummary is a brief event info
type EventSummary struct {
	Type      string    `json:"type"`
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

// UsageSummary contains token usage info
type UsageSummary struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CacheRead    int64   `json:"cache_read_tokens"`
	CacheCreate  int64   `json:"cache_create_tokens"`
	CostUSD      float64 `json:"cost_usd"`
}

// ADRCandidate represents a potential Architecture Decision Record
type ADRCandidate struct {
	Title       string `json:"title"`
	Context     string `json:"context"`
	Decision    string `json:"decision"`
	Consequence string `json:"consequence"`
	SourceEvent string `json:"source_event"`
}

// NewService creates a new operator service
func NewService(database *db.DB, projectRoot string) *Service {
	return &Service{
		db:          database,
		projectRoot: projectRoot,
	}
}

// GenerateBriefing generates a session start briefing
func (s *Service) GenerateBriefing() (*Briefing, error) {
	sessionSvc := session.NewService(s.db)
	portSvc := port.NewService(s.db)

	briefing := &Briefing{
		GeneratedAt: time.Now(),
		ProjectName: filepath.Base(s.projectRoot),
	}

	// Recent sessions (last 5)
	sessions, err := sessionSvc.ListDetailed(false, 5)
	if err == nil {
		for _, sess := range sessions {
			title := ""
			if sess.Title.Valid {
				title = sess.Title.String
			}
			briefing.RecentSessions = append(briefing.RecentSessions, SessionSummary{
				ID:          sess.ID,
				Title:       title,
				Status:      sess.Status,
				StartedAt:   sess.StartedAt,
				DurationStr: sess.DurationStr,
			})
		}
	}

	// Running ports
	runningPorts, err := portSvc.List("running", 20)
	if err == nil {
		for _, p := range runningPorts {
			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}
			var startedAt time.Time
			if p.StartedAt.Valid {
				startedAt = p.StartedAt.Time
			}
			briefing.RunningPorts = append(briefing.RunningPorts, PortSummary{
				ID:        p.ID,
				Title:     title,
				Status:    p.Status,
				StartedAt: startedAt,
			})
		}
	}

	// Pending ports
	pendingPorts, err := portSvc.List("pending", 10)
	if err == nil {
		for _, p := range pendingPorts {
			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}
			briefing.PendingPorts = append(briefing.PendingPorts, PortSummary{
				ID:     p.ID,
				Title:  title,
				Status: p.Status,
			})
		}
	}

	// Load escalations from database
	escalations, err := s.getActiveEscalations()
	if err == nil {
		briefing.Escalations = escalations
	}

	// Generate summary and recommendations
	briefing.Summary = s.generateBriefingSummary(briefing)
	briefing.Recommendations = s.generateRecommendations(briefing)

	return briefing, nil
}

// GenerateSummary generates a session end summary
func (s *Service) GenerateSummary(sessionID string) (*Summary, error) {
	sessionSvc := session.NewService(s.db)
	portSvc := port.NewService(s.db)

	sess, err := sessionSvc.Get(sessionID)
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		SessionID:   sessionID,
		GeneratedAt: time.Now(),
	}

	// Calculate duration
	endTime := time.Now()
	if sess.EndedAt.Valid {
		endTime = sess.EndedAt.Time
	}
	summary.Duration = endTime.Sub(sess.StartedAt)
	summary.DurationStr = formatDuration(summary.Duration)

	// Get session events
	events, err := sessionSvc.GetEvents(sessionID, "", 50)
	if err == nil {
		for _, e := range events {
			summary.Events = append(summary.Events, EventSummary{
				Type:      e.EventType,
				Data:      e.EventData,
				CreatedAt: e.CreatedAt,
			})

			// Track port_end events for completed ports
			if e.EventType == "port_end" {
				portID := extractPortIDFromEvent(e.EventData)
				if portID != "" {
					p, err := portSvc.Get(portID)
					if err == nil {
						title := portID
						if p.Title.Valid {
							title = p.Title.String
						}
						summary.PortsCompleted = append(summary.PortsCompleted, PortSummary{
							ID:     portID,
							Title:  title,
							Status: p.Status,
						})
					}
				}
			}

			// Track port_start events
			if e.EventType == "port_start" {
				portID := extractPortIDFromEvent(e.EventData)
				if portID != "" {
					p, err := portSvc.Get(portID)
					if err == nil {
						title := portID
						if p.Title.Valid {
							title = p.Title.String
						}
						summary.PortsStarted = append(summary.PortsStarted, PortSummary{
							ID:     portID,
							Title:  title,
							Status: p.Status,
						})
					}
				}
			}
		}
	}

	// Usage summary
	summary.Usage = UsageSummary{
		InputTokens:  sess.InputTokens,
		OutputTokens: sess.OutputTokens,
		CacheRead:    sess.CacheReadTokens,
		CacheCreate:  sess.CacheCreateTokens,
		CostUSD:      sess.CostUSD,
	}

	// Detect ADR candidates
	summary.ADRCandidates = s.DetectADR(sessionID)

	return summary, nil
}

// DetectADR detects Architecture Decision Record candidates from session events
func (s *Service) DetectADR(sessionID string) []ADRCandidate {
	sessionSvc := session.NewService(s.db)

	events, err := sessionSvc.GetEvents(sessionID, "", 100)
	if err != nil {
		return nil
	}

	var candidates []ADRCandidate

	// Look for patterns that indicate architectural decisions
	for _, e := range events {
		// Check for escalation events (often indicate decisions)
		if e.EventType == "escalation" {
			candidates = append(candidates, ADRCandidate{
				Title:       "Escalation Decision",
				Context:     e.EventData,
				SourceEvent: e.EventType,
			})
		}

		// Check for significant port completions
		if e.EventType == "port_end" {
			portID := extractPortIDFromEvent(e.EventData)
			// Ports with certain prefixes might indicate architectural work
			if strings.HasPrefix(portID, "arch-") ||
				strings.HasPrefix(portID, "design-") ||
				strings.HasPrefix(portID, "refactor-") {
				candidates = append(candidates, ADRCandidate{
					Title:       fmt.Sprintf("Port Completion: %s", portID),
					Context:     e.EventData,
					SourceEvent: e.EventType,
				})
			}
		}
	}

	return candidates
}

// WriteBriefing writes briefing to .pal/context/session-briefing.md
func (s *Service) WriteBriefing(b *Briefing) error {
	contextDir := filepath.Join(s.projectRoot, ".pal", "context")
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return err
	}

	briefingPath := filepath.Join(contextDir, "session-briefing.md")
	content := s.formatBriefingMarkdown(b)

	return os.WriteFile(briefingPath, []byte(content), 0644)
}

// WriteSummary writes summary to .pal/sessions/{date}-{id}.md
func (s *Service) WriteSummary(sum *Summary) error {
	sessionsDir := filepath.Join(s.projectRoot, ".pal", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return err
	}

	dateStr := sum.GeneratedAt.Format("2006-01-02")
	summaryPath := filepath.Join(sessionsDir, fmt.Sprintf("%s-%s.md", dateStr, sum.SessionID))
	content := s.formatSummaryMarkdown(sum)

	return os.WriteFile(summaryPath, []byte(content), 0644)
}

// GetBriefingPath returns the path to the briefing file
func (s *Service) GetBriefingPath() string {
	return filepath.Join(s.projectRoot, ".pal", "context", "session-briefing.md")
}

// Helper functions

func (s *Service) getActiveEscalations() ([]Escalation, error) {
	rows, err := s.db.Query(`
		SELECT id, port_id, escalation_type, message, created_at
		FROM escalations
		WHERE status = 'open'
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var escalations []Escalation
	for rows.Next() {
		var e Escalation
		var portID sql.NullString
		if err := rows.Scan(&e.ID, &portID, &e.Type, &e.Message, &e.CreatedAt); err != nil {
			continue
		}
		if portID.Valid {
			e.PortID = portID.String
		}
		escalations = append(escalations, e)
	}

	return escalations, nil
}

func (s *Service) generateBriefingSummary(b *Briefing) string {
	var parts []string

	if len(b.RunningPorts) > 0 {
		parts = append(parts, fmt.Sprintf("%d running port(s)", len(b.RunningPorts)))
	}

	if len(b.PendingPorts) > 0 {
		parts = append(parts, fmt.Sprintf("%d pending port(s)", len(b.PendingPorts)))
	}

	if len(b.Escalations) > 0 {
		parts = append(parts, fmt.Sprintf("%d active escalation(s)", len(b.Escalations)))
	}

	if len(parts) == 0 {
		return "No active work items."
	}

	return strings.Join(parts, ", ")
}

func (s *Service) generateRecommendations(b *Briefing) []string {
	var recommendations []string

	// Recommend completing running ports first
	if len(b.RunningPorts) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Continue with running port: %s", b.RunningPorts[0].ID))
	}

	// Recommend addressing escalations
	if len(b.Escalations) > 0 {
		recommendations = append(recommendations,
			"Address pending escalations before starting new work")
	}

	// Recommend pending ports if nothing is running
	if len(b.RunningPorts) == 0 && len(b.PendingPorts) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Start pending port: %s", b.PendingPorts[0].ID))
	}

	return recommendations
}

func (s *Service) formatBriefingMarkdown(b *Briefing) string {
	var sb strings.Builder

	sb.WriteString("# Session Briefing\n\n")
	sb.WriteString(fmt.Sprintf("> Generated: %s\n\n", b.GeneratedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Project**: %s\n\n", b.ProjectName))
	sb.WriteString(fmt.Sprintf("**Summary**: %s\n\n", b.Summary))

	// Running ports
	if len(b.RunningPorts) > 0 {
		sb.WriteString("## Running Ports\n\n")
		for _, p := range b.RunningPorts {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", p.ID, p.Title))
		}
		sb.WriteString("\n")
	}

	// Pending ports
	if len(b.PendingPorts) > 0 {
		sb.WriteString("## Pending Ports\n\n")
		for _, p := range b.PendingPorts {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", p.ID, p.Title))
		}
		sb.WriteString("\n")
	}

	// Escalations
	if len(b.Escalations) > 0 {
		sb.WriteString("## Active Escalations\n\n")
		for _, e := range b.Escalations {
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", e.Type, e.PortID, e.Message))
		}
		sb.WriteString("\n")
	}

	// Recommendations
	if len(b.Recommendations) > 0 {
		sb.WriteString("## Recommendations\n\n")
		for _, r := range b.Recommendations {
			sb.WriteString(fmt.Sprintf("- %s\n", r))
		}
		sb.WriteString("\n")
	}

	// Recent sessions
	if len(b.RecentSessions) > 0 {
		sb.WriteString("## Recent Sessions\n\n")
		sb.WriteString("| ID | Status | Duration | Started |\n")
		sb.WriteString("|----|--------|----------|----------|\n")
		for _, sess := range b.RecentSessions {
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				sess.ID, sess.Status, sess.DurationStr,
				sess.StartedAt.Format("01/02 15:04")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (s *Service) formatSummaryMarkdown(sum *Summary) string {
	var sb strings.Builder

	sb.WriteString("# Session Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Session ID**: %s\n", sum.SessionID))
	sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", sum.DurationStr))
	sb.WriteString(fmt.Sprintf("- **Generated**: %s\n\n", sum.GeneratedAt.Format("2006-01-02 15:04:05")))

	// Usage
	sb.WriteString("## Usage\n\n")
	sb.WriteString(fmt.Sprintf("- Input tokens: %d\n", sum.Usage.InputTokens))
	sb.WriteString(fmt.Sprintf("- Output tokens: %d\n", sum.Usage.OutputTokens))
	sb.WriteString(fmt.Sprintf("- Cache read: %d\n", sum.Usage.CacheRead))
	sb.WriteString(fmt.Sprintf("- Cache create: %d\n", sum.Usage.CacheCreate))
	sb.WriteString(fmt.Sprintf("- Cost: $%.4f\n\n", sum.Usage.CostUSD))

	// Completed ports
	if len(sum.PortsCompleted) > 0 {
		sb.WriteString("## Completed Ports\n\n")
		for _, p := range sum.PortsCompleted {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", p.ID, p.Title))
		}
		sb.WriteString("\n")
	}

	// Started ports
	if len(sum.PortsStarted) > 0 {
		sb.WriteString("## Started Ports\n\n")
		for _, p := range sum.PortsStarted {
			sb.WriteString(fmt.Sprintf("- **%s**: %s (%s)\n", p.ID, p.Title, p.Status))
		}
		sb.WriteString("\n")
	}

	// ADR Candidates
	if len(sum.ADRCandidates) > 0 {
		sb.WriteString("## ADR Candidates\n\n")
		for _, adr := range sum.ADRCandidates {
			sb.WriteString(fmt.Sprintf("### %s\n\n", adr.Title))
			if adr.Context != "" {
				sb.WriteString(fmt.Sprintf("**Context**: %s\n\n", adr.Context))
			}
		}
	}

	// Events timeline
	if len(sum.Events) > 0 {
		sb.WriteString("## Event Timeline\n\n")
		for _, e := range sum.Events {
			sb.WriteString(fmt.Sprintf("- `%s` [%s] %s\n",
				e.CreatedAt.Format("15:04:05"), e.Type, truncate(e.Data, 60)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func extractPortIDFromEvent(eventData string) string {
	// Simple JSON parsing for port_id
	// Format: {"port_id":"xxx",...}
	start := strings.Index(eventData, `"port_id":"`)
	if start == -1 {
		return ""
	}
	start += len(`"port_id":"`)
	end := strings.Index(eventData[start:], `"`)
	if end == -1 {
		return ""
	}
	return eventData[start : start+end]
}

func formatDuration(d time.Duration) string {
	secs := int64(d.Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm %ds", secs/60, secs%60)
	}
	hours := secs / 3600
	mins := (secs % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
