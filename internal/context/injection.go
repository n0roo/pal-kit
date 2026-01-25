package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/session"
)

// SessionContext represents context for session start
type SessionContext struct {
	ProjectOverview string     `json:"project_overview"`
	ActivePorts     []PortInfo `json:"active_ports"`
	PendingPorts    []PortInfo `json:"pending_ports"`
	Conventions     []string   `json:"conventions"`
	WorkflowType    string     `json:"workflow_type"`
	RecentDecisions []string   `json:"recent_decisions"`
	Suggestions     []string   `json:"suggestions"`
}

// PortInfo represents port information
type PortInfo struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Status string `json:"status"`
}

// GenerateSessionContext creates context for session start
func GenerateSessionContext(database *db.DB, projectRoot string) (*SessionContext, error) {
	ctx := &SessionContext{
		Suggestions: []string{},
	}

	// Extract project overview from CLAUDE.md
	claudeMD := filepath.Join(projectRoot, "CLAUDE.md")
	if content, err := os.ReadFile(claudeMD); err == nil {
		ctx.ProjectOverview = extractProjectOverview(string(content))
	}

	// Get ports
	portSvc := port.NewService(database)
	
	runningPorts, err := portSvc.List("running", 10)
	if err == nil {
		for _, p := range runningPorts {
			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}
			ctx.ActivePorts = append(ctx.ActivePorts, PortInfo{
				ID:     p.ID,
				Title:  title,
				Status: p.Status,
			})
		}
	}

	pendingPorts, err := portSvc.List("pending", 10)
	if err == nil {
		for _, p := range pendingPorts {
			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}
			ctx.PendingPorts = append(ctx.PendingPorts, PortInfo{
				ID:     p.ID,
				Title:  title,
				Status: p.Status,
			})
		}
	}

	// Get recent decisions from events
	sessionSvc := session.NewService(database)
	events, err := sessionSvc.GetEvents("", "decision", 5)
	if err == nil {
		for _, e := range events {
			// EventData is JSON string, parse it
			var data map[string]interface{}
			if json.Unmarshal([]byte(e.EventData), &data) == nil {
				if msg, ok := data["message"].(string); ok {
					ctx.RecentDecisions = append(ctx.RecentDecisions, msg)
				}
			}
		}
	}

	// Generate suggestions
	if len(ctx.ActivePorts) == 0 && len(ctx.PendingPorts) > 0 {
		ctx.Suggestions = append(ctx.Suggestions, 
			fmt.Sprintf("대기 중인 포트가 %d개 있습니다. pal_port_start로 작업을 시작하세요.", len(ctx.PendingPorts)))
	}
	if len(ctx.ActivePorts) == 0 && len(ctx.PendingPorts) == 0 {
		ctx.Suggestions = append(ctx.Suggestions,
			"활성 포트가 없습니다. 새 포트를 생성하거나 기존 포트를 시작하세요.")
	}

	return ctx, nil
}

// extractProjectOverview extracts project overview from CLAUDE.md
func extractProjectOverview(content string) string {
	lines := strings.Split(content, "\n")
	var overview strings.Builder
	inOverview := false
	lineCount := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			inOverview = true
			overview.WriteString(line + "\n")
			continue
		}
		if inOverview {
			if strings.HasPrefix(line, "## ") && lineCount > 5 {
				break
			}
			overview.WriteString(line + "\n")
			lineCount++
			if lineCount > 20 {
				break
			}
		}
	}

	return strings.TrimSpace(overview.String())
}

// PortContext represents context for port start
type PortContext struct {
	PortID        string   `json:"port_id"`
	PortTitle     string   `json:"port_title"`
	PortSpec      string   `json:"port_spec"`
	AgentPrompt   string   `json:"agent_prompt,omitempty"`
	Conventions   []string `json:"conventions,omitempty"`
	RelatedDocs   []string `json:"related_docs,omitempty"`
	Checklist     []string `json:"checklist,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty"`
	TokenEstimate int      `json:"token_estimate"`
}

// GeneratePortContext creates context for port start
func GeneratePortContext(database *db.DB, projectRoot, portID string) (*PortContext, error) {
	portSvc := port.NewService(database)
	p, err := portSvc.Get(portID)
	if err != nil {
		return nil, fmt.Errorf("포트 조회 실패: %w", err)
	}

	ctx := &PortContext{
		PortID: p.ID,
	}

	if p.Title.Valid {
		ctx.PortTitle = p.Title.String
	} else {
		ctx.PortTitle = p.ID
	}

	// Load port spec if file path exists
	if p.FilePath.Valid && p.FilePath.String != "" {
		specPath := p.FilePath.String
		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(projectRoot, specPath)
		}
		if content, err := os.ReadFile(specPath); err == nil {
			ctx.PortSpec = string(content)
			ctx.TokenEstimate = len(content) / 4 // rough token estimate
		}
	}

	// Load checklist from port spec
	ctx.Checklist = extractChecklist(ctx.PortSpec)

	return ctx, nil
}

// extractChecklist extracts checklist items from port spec
func extractChecklist(spec string) []string {
	var checklist []string
	lines := strings.Split(spec, "\n")
	inChecklist := false

	for _, line := range lines {
		if strings.Contains(line, "체크리스트") || strings.Contains(line, "완료 기준") {
			inChecklist = true
			continue
		}
		if inChecklist {
			if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "---") {
				break
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]") {
				item := strings.TrimPrefix(line, "- [ ]")
				item = strings.TrimPrefix(item, "- [x]")
				item = strings.TrimSpace(item)
				if item != "" {
					checklist = append(checklist, item)
				}
			}
		}
	}

	return checklist
}

// RecoveryContext represents context for compact recovery
type RecoveryContext struct {
	CheckpointSummary string   `json:"checkpoint_summary"`
	ActivePort        string   `json:"active_port"`
	ActivePortTitle   string   `json:"active_port_title"`
	PortProgress      string   `json:"port_progress"`
	PendingTasks      []string `json:"pending_tasks"`
	RecentFiles       []string `json:"recent_files"`
	KeyDecisions      []string `json:"key_decisions"`
	RecoveryPrompt    string   `json:"recovery_prompt"`
}

// GenerateRecoveryContext creates context for compact recovery
func GenerateRecoveryContext(database *db.DB, sessionID string) (*RecoveryContext, error) {
	ctx := &RecoveryContext{
		PendingTasks:   []string{},
		RecentFiles:    []string{},
		KeyDecisions:   []string{},
	}

	sessionSvc := session.NewService(database)
	portSvc := port.NewService(database)

	// Find active port
	runningPorts, err := portSvc.List("running", 1)
	if err == nil && len(runningPorts) > 0 {
		p := runningPorts[0]
		ctx.ActivePort = p.ID
		if p.Title.Valid {
			ctx.ActivePortTitle = p.Title.String
		}

		// Load port spec for progress
		if p.FilePath.Valid {
			ctx.PortProgress = fmt.Sprintf("포트 '%s' 작업 중", ctx.ActivePortTitle)
		}
	}

	// Get recent file change events
	events, err := sessionSvc.GetEvents(sessionID, "file_change", 5)
	if err == nil {
		for _, e := range events {
			var data map[string]interface{}
			if json.Unmarshal([]byte(e.EventData), &data) == nil {
				if file, ok := data["file"].(string); ok {
					ctx.RecentFiles = append(ctx.RecentFiles, file)
				}
			}
		}
	}

	// Get recent decision events
	decisionEvents, err := sessionSvc.GetEvents(sessionID, "decision", 3)
	if err == nil {
		for _, e := range decisionEvents {
			var data map[string]interface{}
			if json.Unmarshal([]byte(e.EventData), &data) == nil {
				if msg, ok := data["message"].(string); ok {
					ctx.KeyDecisions = append(ctx.KeyDecisions, msg)
				}
			}
		}
	}

	// Generate recovery prompt
	ctx.RecoveryPrompt = generateRecoveryPrompt(ctx)

	return ctx, nil
}

// generateRecoveryPrompt creates a recovery prompt for Claude
func generateRecoveryPrompt(ctx *RecoveryContext) string {
	var prompt strings.Builder
	prompt.WriteString("## Compact 복구\n\n")

	if ctx.ActivePort != "" {
		prompt.WriteString(fmt.Sprintf("**활성 포트:** %s (%s)\n\n", ctx.ActivePortTitle, ctx.ActivePort))
	}

	if ctx.PortProgress != "" {
		prompt.WriteString(fmt.Sprintf("**진행 상황:** %s\n\n", ctx.PortProgress))
	}

	if len(ctx.RecentFiles) > 0 {
		prompt.WriteString("**최근 수정 파일:**\n")
		for _, f := range ctx.RecentFiles {
			prompt.WriteString(fmt.Sprintf("- %s\n", f))
		}
		prompt.WriteString("\n")
	}

	if len(ctx.KeyDecisions) > 0 {
		prompt.WriteString("**주요 결정:**\n")
		for _, d := range ctx.KeyDecisions {
			prompt.WriteString(fmt.Sprintf("- %s\n", d))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("위 컨텍스트를 참고하여 작업을 계속하세요.\n")

	return prompt.String()
}

// UpdateClaudeMD updates CLAUDE.md with PAL Kit section
func UpdateClaudeMD(projectRoot string) error {
	claudeMD := filepath.Join(projectRoot, "CLAUDE.md")
	
	// Read existing content
	existingContent := ""
	if content, err := os.ReadFile(claudeMD); err == nil {
		existingContent = string(content)
	}

	// Generate PAL section
	palSection := generatePALSection()

	// Check if PAL section already exists
	startMarker := "<!-- PAL-KIT-START -->"
	endMarker := "<!-- PAL-KIT-END -->"

	if strings.Contains(existingContent, startMarker) {
		// Update existing section
		startIdx := strings.Index(existingContent, startMarker)
		endIdx := strings.Index(existingContent, endMarker)
		if endIdx > startIdx {
			newContent := existingContent[:startIdx] + startMarker + "\n" + palSection + "\n" + existingContent[endIdx:]
			return os.WriteFile(claudeMD, []byte(newContent), 0644)
		}
	}

	// Append new section
	newContent := existingContent + "\n\n" + startMarker + "\n" + palSection + "\n" + endMarker + "\n"
	return os.WriteFile(claudeMD, []byte(newContent), 0644)
}

// generatePALSection generates PAL Kit section for CLAUDE.md
func generatePALSection() string {
	return fmt.Sprintf(`## PAL Kit 연동

이 프로젝트는 PAL Kit으로 관리됩니다.
마지막 업데이트: %s

### 사용 가능한 MCP 도구
- ` + "`pal_status`" + ` - 현재 상태 확인
- ` + "`pal_port_start`" + ` - 작업 시작
- ` + "`pal_port_end`" + ` - 작업 완료 (자동 검증)
- ` + "`pal_checkpoint`" + ` - 체크포인트 관리
- ` + "`pal_escalate`" + ` - 에스컬레이션

### 워크플로우
1. 작업 시작 시 ` + "`pal_status`" + `로 상태 확인
2. ` + "`pal_port_start`" + `로 포트 시작
3. 작업 완료 시 ` + "`pal_port_end`" + ` 호출
4. 자동 체크리스트 검증 후 완료/블록 처리
`, time.Now().Format("2006-01-02 15:04"))
}
