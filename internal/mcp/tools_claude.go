package mcp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/checklist"
	"github.com/n0roo/pal-kit/internal/checkpoint"
	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/rules"
)

// Claude 친화적 MCP 도구들 (LM-mcp-tools 포트)
// 사용자가 직접 pal 명령어를 호출하지 않고 Claude가 이 도구들을 호출

// pal_status 도구 스키마
var toolPalStatus = Tool{
	Name:        "pal_status",
	Description: "PAL Kit 현재 상태 조회. 세션, 활성 포트, Attention 상태를 확인합니다. 작업 시작 시 호출하세요.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`),
}

// pal_port_start 도구 스키마
var toolPalPortStart = Tool{
	Name:        "pal_port_start",
	Description: "새 작업 포트를 시작합니다. 컨텍스트와 체크리스트가 자동으로 로드됩니다.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string", "description": "포트 ID (영문, 숫자, 하이픈)"},
			"title": {"type": "string", "description": "포트 제목 (작업 설명)"},
			"description": {"type": "string", "description": "상세 설명 (optional)"}
		},
		"required": ["id", "title"]
	}`),
}

// pal_port_end 도구 스키마
var toolPalPortEnd = Tool{
	Name:        "pal_port_end",
	Description: "작업 포트를 완료합니다. 자동으로 빌드/테스트 검증이 실행됩니다.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {"type": "string", "description": "포트 ID"},
			"summary": {"type": "string", "description": "작업 요약"}
		},
		"required": ["id"]
	}`),
}

// pal_checkpoint 도구 스키마
var toolPalCheckpoint = Tool{
	Name:        "pal_checkpoint",
	Description: "체크포인트를 생성하거나 복구합니다. 주요 결정 후 저장하세요.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"action": {"type": "string", "enum": ["create", "restore", "list"], "description": "동작"},
			"id": {"type": "string", "description": "복구 시 체크포인트 ID"},
			"summary": {"type": "string", "description": "생성 시 요약"}
		},
		"required": ["action"]
	}`),
}

// pal_escalate 도구 스키마
var toolPalEscalate = Tool{
	Name:        "pal_escalate",
	Description: "문제 발생 시 에스컬레이션합니다. 사용자 개입이 필요할 때 호출하세요.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"type": {"type": "string", "enum": ["user", "architect", "blocked"], "description": "에스컬레이션 타입"},
			"issue": {"type": "string", "description": "문제 설명"},
			"context": {"type": "string", "description": "상황 컨텍스트"},
			"suggestion": {"type": "string", "description": "제안 (optional)"}
		},
		"required": ["type", "issue"]
	}`),
}

// pal_context 도구 스키마
var toolPalContext = Tool{
	Name:        "pal_context",
	Description: "컨벤션이나 문서를 조회합니다.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"type": {"type": "string", "enum": ["convention", "document", "port"], "description": "컨텍스트 타입"},
			"query": {"type": "string", "description": "검색어"},
			"limit": {"type": "integer", "description": "토큰 제한 (default: 10000)"}
		},
		"required": ["type"]
	}`),
}

// pal_session 도구 스키마 (OP-v1.0-claude-integration)
var toolPalSession = Tool{
	Name:        "pal_session",
	Description: "세션 상태와 Attention 정보를 조회합니다. 토큰 사용량, 컴팩트 횟수 등을 확인할 수 있습니다.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"session_id": {"type": "string", "description": "세션 ID (optional, 기본: 현재 세션)"},
			"include_events": {"type": "boolean", "description": "이벤트 포함 여부 (default: false)"}
		}
	}`),
}

// pal_hierarchy 도구 스키마 (OP-v1.0-claude-integration)
var toolPalHierarchy = Tool{
	Name:        "pal_hierarchy",
	Description: "세션 계층 구조를 조회합니다. 부모-자식 세션, 포트 관계를 확인할 수 있습니다.",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"session_id": {"type": "string", "description": "세션 ID (optional)"},
			"depth": {"type": "integer", "description": "계층 깊이 (default: 3)"}
		}
	}`),
}

// GetClaudeTools returns Claude-friendly tools
func GetClaudeTools() []Tool {
	return []Tool{
		toolPalStatus,
		toolPalPortStart,
		toolPalPortEnd,
		toolPalCheckpoint,
		toolPalEscalate,
		toolPalContext,
		toolPalSession,
		toolPalHierarchy,
	}
}

// StatusResult represents pal_status result
type StatusResult struct {
	Session        *SessionInfo      `json:"session,omitempty"`
	ActivePorts    []PortInfo        `json:"active_ports"`
	PendingPorts   []PortInfo        `json:"pending_ports"`
	TokenUsage     *TokenUsageInfo   `json:"token_usage,omitempty"`
	LastCheckpoint *CheckpointInfo   `json:"last_checkpoint,omitempty"`
	Suggestions    []string          `json:"suggestions"`
}

type SessionInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Status      string `json:"status"`
	ProjectRoot string `json:"project_root,omitempty"`
}

type PortInfo struct {
	ID     string `json:"id"`
	Title  string `json:"title,omitempty"`
	Status string `json:"status"`
}

type TokenUsageInfo struct {
	Used       int     `json:"used"`
	Budget     int     `json:"budget"`
	UsageRatio float64 `json:"usage_ratio"`
}

type CheckpointInfo struct {
	ID        string    `json:"id"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

// PortStartResult represents pal_port_start result
type PortStartResult struct {
	PortID      string   `json:"port_id"`
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	Checklist   []string `json:"checklist,omitempty"`
	Conventions []string `json:"conventions,omitempty"`
}

// PortEndResult represents pal_port_end result
type PortEndResult struct {
	PortID     string           `json:"port_id"`
	Status     string           `json:"status"` // complete, blocked
	Message    string           `json:"message"`
	Checklist  *ChecklistResult `json:"checklist,omitempty"`
	NextAction string           `json:"next_action,omitempty"`
}

type ChecklistResult struct {
	Passed bool              `json:"passed"`
	Items  []ChecklistItem   `json:"items"`
}

type ChecklistItem struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// EscalateResult represents pal_escalate result
type EscalateResult struct {
	EscalationID string `json:"escalation_id"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// ContextResult represents pal_context result
type ContextResult struct {
	Items       []ContextItem `json:"items"`
	TotalTokens int           `json:"total_tokens"`
}

type ContextItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Tokens  int    `json:"tokens"`
}

// toolPalStatusHandler handles pal_status tool call
func (s *Server) toolPalStatusHandler(args json.RawMessage) (interface{}, error) {
	result := &StatusResult{
		ActivePorts:  []PortInfo{},
		PendingPorts: []PortInfo{},
		Suggestions:  []string{},
	}

	// 컨텍스트 주입을 사용하여 세션 컨텍스트 생성
	sessionCtx, err := context.GenerateSessionContext(s.database, s.projectRoot)
	if err == nil {
		// 활성 포트
		for _, p := range sessionCtx.ActivePorts {
			result.ActivePorts = append(result.ActivePorts, PortInfo{
				ID:     p.ID,
				Title:  p.Title,
				Status: p.Status,
			})
		}
		// 대기 포트
		for _, p := range sessionCtx.PendingPorts {
			result.PendingPorts = append(result.PendingPorts, PortInfo{
				ID:     p.ID,
				Title:  p.Title,
				Status: p.Status,
			})
		}
		// 제안
		result.Suggestions = sessionCtx.Suggestions
	} else {
		// 폴백: 직접 포트 조회
		portSvc := port.NewService(s.database)

		runningPorts, err := portSvc.List("running", 10)
		if err == nil {
			for _, p := range runningPorts {
				title := p.ID
				if p.Title.Valid {
					title = p.Title.String
				}
				result.ActivePorts = append(result.ActivePorts, PortInfo{
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
				result.PendingPorts = append(result.PendingPorts, PortInfo{
					ID:     p.ID,
					Title:  title,
					Status: p.Status,
				})
			}
		}

		if len(result.ActivePorts) == 0 {
			result.Suggestions = append(result.Suggestions,
				"활성 포트가 없습니다. pal_port_start로 포트를 시작하세요.")
		}
		if len(result.PendingPorts) > 0 {
			result.Suggestions = append(result.Suggestions,
				fmt.Sprintf("대기 중인 포트가 %d개 있습니다.", len(result.PendingPorts)))
		}
	}

	return result, nil
}

// toolPalPortStartHandler handles pal_port_start tool call
func (s *Server) toolPalPortStartHandler(args json.RawMessage) (interface{}, error) {
	var params struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	// 포트 서비스로 포트 생성/시작
	portSvc := port.NewService(s.database)

	// 포트가 없으면 생성
	_, err := portSvc.Get(params.ID)
	if err != nil {
		// 포트 생성
		if err := portSvc.Create(params.ID, params.Title, ""); err != nil {
			return nil, fmt.Errorf("포트 생성 실패: %w", err)
		}
	}

	// 포트 상태를 running으로 변경
	if err := portSvc.UpdateStatus(params.ID, "running"); err != nil {
		return nil, fmt.Errorf("포트 시작 실패: %w", err)
	}

	// Rules 파일 자동 생성 (LM-context-injection)
	if err := rules.ActivatePortRules(s.database, s.projectRoot, params.ID); err != nil {
		// 실패해도 계속 진행 (경고만)
		fmt.Printf("Warning: rules 파일 생성 실패: %v\n", err)
	}

	// 포트 컨텍스트 생성
	portCtx, _ := context.GeneratePortContext(s.database, s.projectRoot, params.ID)

	result := &PortStartResult{
		PortID:  params.ID,
		Status:  "running",
		Message: fmt.Sprintf("포트 '%s' 시작됨", params.Title),
	}

	// 포트 컨텍스트에서 체크리스트 추가
	if portCtx != nil {
		result.Checklist = portCtx.Checklist
	}

	// pal hook port-start 호출 (호환성)
	cmd := exec.Command("pal", "hook", "port-start", params.ID)
	cmd.Dir = s.projectRoot
	cmd.Run()

	return result, nil
}

// toolPalPortEndHandler handles pal_port_end tool call
func (s *Server) toolPalPortEndHandler(args json.RawMessage) (interface{}, error) {
	var params struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	// 체크리스트 검증 실행 (L1-checklist-enforce)
	verifier := checklist.NewVerifier(s.projectRoot)
	verifyResult := verifier.QuickVerify()

	// 결과 변환
	checkResult := &ChecklistResult{
		Passed: verifyResult.Passed,
		Items:  []ChecklistItem{},
	}

	for _, r := range verifyResult.Results {
		checkResult.Items = append(checkResult.Items, ChecklistItem{
			Name:   r.Description,
			Passed: r.Passed,
			Output: r.Output,
			Error:  r.Message,
		})
	}

	// 포트 서비스
	portSvc := port.NewService(s.database)

	result := &PortEndResult{
		PortID:    params.ID,
		Checklist: checkResult,
	}

	if verifyResult.Passed {
		// 성공: 포트 완료
		portSvc.UpdateStatus(params.ID, "complete")
		result.Status = "complete"
		result.Message = "✅ 포트 완료"

		// Rules 파일 제거 (LM-context-injection)
		rules.DeactivatePortRules(s.projectRoot, params.ID)

		// pal hook port-end 호출 (호환성)
		cmd := exec.Command("pal", "hook", "port-end", params.ID)
		cmd.Dir = s.projectRoot
		cmd.Run()
	} else {
		// 실패: 포트 블록
		portSvc.UpdateStatus(params.ID, "blocked")
		result.Status = "blocked"
		result.Message = "❌ 체크리스트 검증 실패"

		// 실패 항목에 대한 수정 가이드
		var failedItems []string
		for _, r := range verifyResult.Results {
			if !r.Passed && r.Required {
				failedItems = append(failedItems, fmt.Sprintf("%s: %s", r.Description, r.Message))
			}
		}
		result.NextAction = strings.Join(failedItems, "\n")
		if len(verifyResult.BlockedBy) > 0 {
			result.NextAction += fmt.Sprintf("\n\n블록 항목: %s", strings.Join(verifyResult.BlockedBy, ", "))
		}
	}

	return result, nil
}

// runBuildCheck runs build check
func (s *Server) runBuildCheck() ChecklistItem {
	item := ChecklistItem{
		Name:   "빌드",
		Passed: true,
	}

	// Go 프로젝트인지 확인
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = s.projectRoot
	output, err := cmd.CombinedOutput()

	if err != nil {
		item.Passed = false
		item.Error = truncateOutput(string(output), 300)
	} else {
		item.Output = "성공"
	}

	return item
}

// runTestCheck runs test check
func (s *Server) runTestCheck() ChecklistItem {
	item := ChecklistItem{
		Name:   "테스트",
		Passed: true,
	}

	// Go 테스트 실행
	cmd := exec.Command("go", "test", "./...", "-v")
	cmd.Dir = s.projectRoot
	output, err := cmd.CombinedOutput()

	if err != nil {
		item.Passed = false
		item.Error = truncateOutput(string(output), 300)
	} else {
		// 테스트 통과 수 추출
		lines := strings.Split(string(output), "\n")
		passCount := 0
		for _, line := range lines {
			if strings.Contains(line, "--- PASS:") {
				passCount++
			}
		}
		item.Output = fmt.Sprintf("%d tests passed", passCount)
	}

	return item
}

// toolPalCheckpointHandler handles pal_checkpoint tool call
func (s *Server) toolPalCheckpointHandler(args json.RawMessage) (interface{}, error) {
	var params struct {
		Action    string `json:"action"`
		ID        string `json:"id"`
		Summary   string `json:"summary"`
		SessionID string `json:"session_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	// 체크포인트 모니터 생성
	monitor := checkpoint.NewMonitor(s.database)

	// 세션 ID가 없으면 기본값 사용
	sessionID := params.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	switch params.Action {
	case "create":
		// 수동 체크포인트 생성
		cp, err := monitor.CreateManual(sessionID, params.Summary, 0, 0)
		if err != nil {
			return nil, fmt.Errorf("체크포인트 생성 실패: %w", err)
		}
		return map[string]interface{}{
			"id":          cp.ID,
			"summary":     cp.Summary,
			"port_id":     cp.PortID,
			"created_at":  cp.CreatedAt.Format(time.RFC3339),
			"message":     "체크포인트 생성됨",
		}, nil

	case "list":
		// 체크포인트 목록 조회
		checkpoints, err := monitor.List(sessionID, 10)
		if err != nil {
			return nil, err
		}
		
		var items []map[string]interface{}
		for _, cp := range checkpoints {
			items = append(items, map[string]interface{}{
				"id":           cp.ID,
				"summary":      cp.Summary,
				"trigger_type": cp.TriggerType,
				"port_id":      cp.PortID,
				"created_at":   cp.CreatedAt.Format(time.RFC3339),
			})
		}
		return map[string]interface{}{
			"checkpoints": items,
			"count":       len(items),
			"message":     "체크포인트 목록",
		}, nil

	case "restore":
		// 체크포인트 복구 컨텍스트 생성
		if params.ID != "" {
			cp, err := monitor.GetByID(params.ID)
			if err != nil {
				return nil, fmt.Errorf("체크포인트 조회 실패: %w", err)
			}
			return map[string]interface{}{
				"id":            cp.ID,
				"summary":       cp.Summary,
				"port_id":       cp.PortID,
				"active_files":  cp.ActiveFiles,
				"key_points":    cp.KeyPoints,
				"status":        "restored",
				"message":       "체크포인트 복구됨",
			}, nil
		}
		
		// ID가 없으면 최신 체크포인트 복구
		recoveryCtx, err := monitor.GenerateRecoveryContext(sessionID)
		if err != nil {
			return nil, fmt.Errorf("복구 컨텍스트 생성 실패: %w", err)
		}
		recoveryCtx["status"] = "restored"
		recoveryCtx["message"] = "최신 체크포인트 복구됨"
		return recoveryCtx, nil

	default:
		return nil, fmt.Errorf("unknown action: %s", params.Action)
	}
}

// toolPalEscalateHandler handles pal_escalate tool call
func (s *Server) toolPalEscalateHandler(args json.RawMessage) (interface{}, error) {
	var params struct {
		Type       string `json:"type"`
		Issue      string `json:"issue"`
		Context    string `json:"context"`
		Suggestion string `json:"suggestion"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	// 에스컬레이션 이벤트 기록
	escID := fmt.Sprintf("esc-%d", time.Now().Unix())

	// pal hook event escalation 호출
	cmd := exec.Command("pal", "hook", "event", "escalation", params.Issue)
	cmd.Dir = s.projectRoot
	cmd.Run()

	return &EscalateResult{
		EscalationID: escID,
		Status:       "created",
		Message:      fmt.Sprintf("에스컬레이션 생성됨: %s", params.Type),
	}, nil
}

// toolPalContextHandler handles pal_context tool call
func (s *Server) toolPalContextHandler(args json.RawMessage) (interface{}, error) {
	var params struct {
		Type  string `json:"type"`
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if params.Limit == 0 {
		params.Limit = 10000
	}

	result := &ContextResult{
		Items:       []ContextItem{},
		TotalTokens: 0,
	}

	switch params.Type {
	case "convention":
		// pal context conventions 호출
		cmd := exec.Command("pal", "context", "conventions", "--json")
		cmd.Dir = s.projectRoot
		output, _ := cmd.Output()
		
		if len(output) > 0 {
			var items []ContextItem
			json.Unmarshal(output, &items)
			result.Items = items
		}

	case "document":
		// pal context documents 호출
		cmd := exec.Command("pal", "context", "documents", "--json")
		if params.Query != "" {
			cmd.Args = append(cmd.Args, "--query", params.Query)
		}
		cmd.Dir = s.projectRoot
		output, _ := cmd.Output()
		
		if len(output) > 0 {
			var items []ContextItem
			json.Unmarshal(output, &items)
			result.Items = items
		}

	case "port":
		// pal port list 호출
		portSvc := port.NewService(s.database)
		ports, _ := portSvc.List("", 20)
		for _, p := range ports {
			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}
			result.Items = append(result.Items, ContextItem{
				ID:   p.ID,
				Type: "port",
				Path: title,
			})
		}
	}

	// 토큰 합계 계산
	for _, item := range result.Items {
		result.TotalTokens += item.Tokens
	}

	return result, nil
}

// truncateOutput truncates output to maxLen
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// toolPalSessionHandler handles pal_session tool call
func (s *Server) toolPalSessionHandler(args json.RawMessage) (interface{}, error) {
	var params struct {
		SessionID     string `json:"session_id"`
		IncludeEvents bool   `json:"include_events"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	result := map[string]interface{}{}

	// 세션 정보 조회 (pal session list --json 호출)
	cmd := exec.Command("pal", "session", "list", "--json", "--limit", "1")
	cmd.Dir = s.projectRoot
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		var sessions []map[string]interface{}
		if json.Unmarshal(output, &sessions) == nil && len(sessions) > 0 {
			result["session"] = sessions[0]
		}
	}

	// Attention 상태 조회
	cmd = exec.Command("pal", "attention", "status", "--json")
	cmd.Dir = s.projectRoot
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		var attention map[string]interface{}
		if json.Unmarshal(output, &attention) == nil {
			result["attention"] = attention
		}
	}

	// 활성 포트
	portSvc := port.NewService(s.database)
	runningPorts, _ := portSvc.List("running", 5)
	var activePorts []map[string]interface{}
	for _, p := range runningPorts {
		title := p.ID
		if p.Title.Valid {
			title = p.Title.String
		}
		activePorts = append(activePorts, map[string]interface{}{
			"id":     p.ID,
			"title":  title,
			"status": p.Status,
		})
	}
	result["active_ports"] = activePorts

	// 이벤트 포함 (optional)
	if params.IncludeEvents {
		cmd = exec.Command("pal", "hook", "events", "--json", "--limit", "10")
		cmd.Dir = s.projectRoot
		output, err = cmd.Output()
		if err == nil && len(output) > 0 {
			var events map[string]interface{}
			if json.Unmarshal(output, &events) == nil {
				result["events"] = events
			}
		}
	}

	return result, nil
}

// toolPalHierarchyHandler handles pal_hierarchy tool call
func (s *Server) toolPalHierarchyHandler(args json.RawMessage) (interface{}, error) {
	var params struct {
		SessionID string `json:"session_id"`
		Depth     int    `json:"depth"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if params.Depth == 0 {
		params.Depth = 3
	}

	result := map[string]interface{}{
		"hierarchy": []interface{}{},
	}

	// 세션 계층 조회 (pal session hierarchy --json 호출)
	cmd := exec.Command("pal", "session", "list", "--json")
	cmd.Dir = s.projectRoot
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		var sessions []map[string]interface{}
		if json.Unmarshal(output, &sessions) == nil {
			// 간단한 계층 구조 생성
			for _, sess := range sessions {
				sessType, _ := sess["session_type"].(string)
				if sessType == "main" {
					result["hierarchy"] = append(result["hierarchy"].([]interface{}), sess)
				}
			}
		}
	}

	// 포트 계층
	portSvc := port.NewService(s.database)
	allPorts, _ := portSvc.List("", 50)
	
	portsByStatus := map[string][]map[string]interface{}{
		"running": {},
		"pending": {},
		"complete": {},
		"blocked": {},
	}
	
	for _, p := range allPorts {
		title := p.ID
		if p.Title.Valid {
			title = p.Title.String
		}
		portInfo := map[string]interface{}{
			"id":     p.ID,
			"title":  title,
			"status": p.Status,
		}
		if _, ok := portsByStatus[p.Status]; ok {
			portsByStatus[p.Status] = append(portsByStatus[p.Status], portInfo)
		}
	}
	
	result["ports"] = portsByStatus

	// 요약 정보
	result["summary"] = map[string]interface{}{
		"running_ports":  len(portsByStatus["running"]),
		"pending_ports":  len(portsByStatus["pending"]),
		"complete_ports": len(portsByStatus["complete"]),
		"blocked_ports":  len(portsByStatus["blocked"]),
	}

	return result, nil
}
