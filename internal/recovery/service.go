package recovery

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0roo/pal-kit/internal/checkpoint"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/session"
)

// Service handles compact recovery
type Service struct {
	database      *db.DB
	checkpointMon *checkpoint.Monitor
	portSvc       *port.Service
	sessionSvc    *session.Service
}

// NewService creates a new recovery service
func NewService(database *db.DB) *Service {
	return &Service{
		database:      database,
		checkpointMon: checkpoint.NewMonitor(database),
		portSvc:       port.NewService(database),
		sessionSvc:    session.NewService(database),
	}
}

// RecoveryContext represents context for compact recovery
type RecoveryContext struct {
	CheckpointID   string   `json:"checkpoint_id"`
	Summary        string   `json:"summary"`
	ActivePort     string   `json:"active_port"`
	ActivePortTitle string  `json:"active_port_title,omitempty"`
	PortProgress   string   `json:"port_progress"`
	PendingTasks   []string `json:"pending_tasks"`
	RecentFiles    []string `json:"recent_files"`
	KeyDecisions   []string `json:"key_decisions"`
	RecoveryPrompt string   `json:"recovery_prompt"`
	TokensUsed     int      `json:"tokens_used,omitempty"`
	TokenBudget    int      `json:"token_budget,omitempty"`
}

// GenerateRecoveryContext creates a recovery context from the latest checkpoint
func (s *Service) GenerateRecoveryContext(sessionID string) (*RecoveryContext, error) {
	ctx := &RecoveryContext{
		PendingTasks: []string{},
		RecentFiles:  []string{},
		KeyDecisions: []string{},
	}

	// Get latest checkpoint
	cp, err := s.checkpointMon.GetLatest(sessionID)
	if err == nil && cp != nil {
		ctx.CheckpointID = cp.ID
		ctx.Summary = cp.Summary
		ctx.RecentFiles = cp.ActiveFiles
		ctx.KeyDecisions = cp.KeyPoints
		ctx.TokensUsed = cp.TokensUsed
		ctx.TokenBudget = cp.TokenBudget

		if cp.PortID != "" {
			ctx.ActivePort = cp.PortID
		}
	}

	// Get active port
	ports, err := s.portSvc.List("running", 1)
	if err == nil && len(ports) > 0 {
		p := ports[0]
		ctx.ActivePort = p.ID
		if p.Title.Valid {
			ctx.ActivePortTitle = p.Title.String
		}
		ctx.PortProgress = s.generatePortProgress(&p)
		ctx.PendingTasks = s.getRemainingTasks(&p)
	}

	// Get recent decisions from events if not from checkpoint
	if len(ctx.KeyDecisions) == 0 {
		events, err := s.sessionSvc.GetEvents(sessionID, "decision", 5)
		if err == nil {
			for _, e := range events {
				var data map[string]interface{}
				if json.Unmarshal([]byte(e.EventData), &data) == nil {
					if msg, ok := data["message"].(string); ok {
						ctx.KeyDecisions = append(ctx.KeyDecisions, msg)
					}
				}
			}
		}
	}

	// Generate recovery prompt
	ctx.RecoveryPrompt = s.generateRecoveryPrompt(ctx)

	return ctx, nil
}

// generatePortProgress generates a progress string for a port
func (s *Service) generatePortProgress(p *port.Port) string {
	// Simple progress based on port status
	switch p.Status {
	case "pending":
		return "0% - 시작 전"
	case "running":
		return "진행 중"
	case "blocked":
		return "블록됨 - 문제 해결 필요"
	case "complete":
		return "100% 완료"
	default:
		return p.Status
	}
}

// getRemainingTasks gets remaining tasks for a port
func (s *Service) getRemainingTasks(p *port.Port) []string {
	// Try to extract from port file if available
	if p.FilePath.Valid && p.FilePath.String != "" {
		// This would read the port file and extract checklist
		// For now, return empty
		return []string{}
	}
	return []string{}
}

// generateRecoveryPrompt generates a recovery prompt for Claude
func (s *Service) generateRecoveryPrompt(ctx *RecoveryContext) string {
	var sb strings.Builder

	sb.WriteString("## Compact 복구\n\n")

	if ctx.Summary != "" {
		sb.WriteString(fmt.Sprintf("**마지막 상태:** %s\n\n", ctx.Summary))
	}

	if ctx.ActivePort != "" {
		portName := ctx.ActivePort
		if ctx.ActivePortTitle != "" {
			portName = ctx.ActivePortTitle
		}
		sb.WriteString(fmt.Sprintf("**활성 포트:** %s\n", portName))
		if ctx.PortProgress != "" {
			sb.WriteString(fmt.Sprintf("**진행 상황:** %s\n", ctx.PortProgress))
		}
		sb.WriteString("\n")
	}

	if len(ctx.PendingTasks) > 0 {
		sb.WriteString("**남은 작업:**\n")
		for _, task := range ctx.PendingTasks {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", task))
		}
		sb.WriteString("\n")
	}

	if len(ctx.RecentFiles) > 0 {
		sb.WriteString("**최근 수정 파일:**\n")
		for _, file := range ctx.RecentFiles {
			sb.WriteString(fmt.Sprintf("- %s\n", file))
		}
		sb.WriteString("\n")
	}

	if len(ctx.KeyDecisions) > 0 {
		sb.WriteString("**주요 결정:**\n")
		for _, dec := range ctx.KeyDecisions {
			sb.WriteString(fmt.Sprintf("- %s\n", dec))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("위 컨텍스트를 참고하여 작업을 계속하세요.\n")

	return sb.String()
}

// RecordCompactEvent records a compact event
func (s *Service) RecordCompactEvent(sessionID string, ctx *RecoveryContext) error {
	// Record compact event in session - serialize to JSON string
	eventData := fmt.Sprintf(`{"checkpoint_id":"%s","active_port":"%s","tokens_used":%d}`,
		ctx.CheckpointID, ctx.ActivePort, ctx.TokensUsed)
	return s.sessionSvc.LogEvent(sessionID, "compact", eventData)
}

// DetectCompact detects if a message indicates a compact event
func DetectCompact(message string) bool {
	lower := strings.ToLower(message)
	compactIndicators := []string{
		"compact",
		"context window",
		"token limit",
		"context limit",
		"truncat",
		"summariz",
	}

	for _, indicator := range compactIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	return false
}

// GetRecoveryHint returns a short recovery hint
func (s *Service) GetRecoveryHint(sessionID string) (string, error) {
	ctx, err := s.GenerateRecoveryContext(sessionID)
	if err != nil {
		return "", err
	}

	hint := ""
	if ctx.ActivePort != "" {
		hint = fmt.Sprintf("작업 중: %s", ctx.ActivePortTitle)
		if ctx.ActivePortTitle == "" {
			hint = fmt.Sprintf("작업 중: %s", ctx.ActivePort)
		}
	}
	if ctx.Summary != "" {
		hint += fmt.Sprintf(" - %s", ctx.Summary)
	}

	return hint, nil
}
