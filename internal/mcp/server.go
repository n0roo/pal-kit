package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/agentv2"
	"github.com/n0roo/pal-kit/internal/attention"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/handoff"
	"github.com/n0roo/pal-kit/internal/message"
	"github.com/n0roo/pal-kit/internal/orchestrator"
	"github.com/n0roo/pal-kit/internal/session"
)

// MCP JSON-RPC 2.0 types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Protocol types
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

// Server represents the MCP server
type Server struct {
	database    *db.DB
	projectRoot string

	// Services
	sessionSvc *session.Service
	orchSvc    *orchestrator.Service
	msgStore   *message.Store
	agentStore *agentv2.Store
	attStore   *attention.Store
	hoStore    *handoff.Store

	// I/O
	reader *bufio.Reader
	writer io.Writer
}

// NewServer creates a new MCP server
func NewServer(dbPath, projectRoot string) (*Server, error) {
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("DB 열기 실패: %w", err)
	}

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)

	return &Server{
		database:    database,
		projectRoot: projectRoot,
		sessionSvc:  sessionSvc,
		orchSvc:     orchestrator.NewService(database, sessionSvc, msgStore),
		msgStore:    msgStore,
		agentStore:  agentv2.NewStore(database.DB),
		attStore:    attention.NewStore(database.DB),
		hoStore:     handoff.NewStore(database),
		reader:      bufio.NewReader(os.Stdin),
		writer:      os.Stdout,
	}, nil
}

// Close closes the server
func (s *Server) Close() error {
	return s.database.Close()
}

// Run starts the MCP server loop
func (s *Server) Run() error {
	log.Println("PAL Kit MCP Server started")

	for {
		line, err := s.reader.ReadString('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("읽기 오류: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(&req)
	}
}

func (s *Server) handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "notifications/initialized":
		// Client is ready
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "prompts/list":
		s.handlePromptsList(req)
	case "prompts/get":
		s.handlePromptsGet(req)
	case "resources/list":
		s.handleResourcesList(req)
	case "resources/read":
		s.handleResourcesRead(req)
	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *Server) handleInitialize(req *JSONRPCRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "pal-kit",
			"version": "1.0.0",
		},
		"capabilities": ServerCapabilities{
			Tools:     &ToolsCapability{ListChanged: true},
			Prompts:   &PromptsCapability{ListChanged: true},
			Resources: &ResourcesCapability{Subscribe: true, ListChanged: true},
		},
	}
	s.sendResult(req.ID, result)
}

func (s *Server) handleToolsList(req *JSONRPCRequest) {
	// Claude 친화적 도구들 먼저 추가
	tools := GetClaudeTools()

	// 기존 도구들 추가
	tools = append(tools, []Tool{
		{
			Name:        "session_start",
			Description: "새 세션을 시작합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "세션 제목"},
					"type": {"type": "string", "enum": ["build", "operator", "worker", "test"], "description": "세션 타입"},
					"parent_id": {"type": "string", "description": "부모 세션 ID (optional)"},
					"port_id": {"type": "string", "description": "포트 ID (optional)"},
					"agent_id": {"type": "string", "description": "에이전트 ID (optional)"},
					"token_budget": {"type": "integer", "description": "토큰 예산 (default: 15000)"}
				},
				"required": ["title", "type"]
			}`),
		},
		{
			Name:        "session_end",
			Description: "세션을 종료합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"session_id": {"type": "string", "description": "세션 ID"},
					"status": {"type": "string", "enum": ["complete", "failed", "cancelled"], "description": "종료 상태"},
					"summary": {"type": "object", "description": "세션 요약 (optional)"}
				},
				"required": ["session_id", "status"]
			}`),
		},
		{
			Name:        "session_hierarchy",
			Description: "세션 계층 구조를 조회합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"root_id": {"type": "string", "description": "루트 세션 ID"}
				},
				"required": ["root_id"]
			}`),
		},
		{
			Name:        "attention_status",
			Description: "세션의 Attention 상태를 조회합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"session_id": {"type": "string", "description": "세션 ID"}
				},
				"required": ["session_id"]
			}`),
		},
		{
			Name:        "attention_update",
			Description: "세션의 Attention 상태를 업데이트합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"session_id": {"type": "string", "description": "세션 ID"},
					"loaded_tokens": {"type": "integer", "description": "현재 로드된 토큰"},
					"focus_score": {"type": "number", "description": "Focus Score (0-1)"},
					"loaded_files": {"type": "array", "items": {"type": "string"}, "description": "로드된 파일 목록"},
					"loaded_conventions": {"type": "array", "items": {"type": "string"}, "description": "로드된 컨벤션 목록"}
				},
				"required": ["session_id"]
			}`),
		},
		{
			Name:        "orchestration_create",
			Description: "Orchestration 포트를 생성합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"title": {"type": "string", "description": "Orchestration 제목"},
					"description": {"type": "string", "description": "설명"},
					"ports": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"port_id": {"type": "string"},
								"order": {"type": "integer"},
								"depends_on": {"type": "array", "items": {"type": "string"}}
							},
							"required": ["port_id", "order"]
						}
					}
				},
				"required": ["title", "ports"]
			}`),
		},
		{
			Name:        "orchestration_status",
			Description: "Orchestration 상태를 조회합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"orchestration_id": {"type": "string", "description": "Orchestration ID"}
				},
				"required": ["orchestration_id"]
			}`),
		},
		{
			Name:        "message_send",
			Description: "다른 세션에 메시지를 전송합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from_session": {"type": "string", "description": "발신 세션 ID"},
					"to_session": {"type": "string", "description": "수신 세션 ID"},
					"type": {"type": "string", "enum": ["request", "response", "report", "escalation"], "description": "메시지 타입"},
					"subtype": {"type": "string", "description": "서브타입 (task_assign, impl_ready, test_pass, etc)"},
					"port_id": {"type": "string", "description": "포트 ID"},
					"payload": {"type": "object", "description": "메시지 페이로드"}
				},
				"required": ["from_session", "to_session", "type", "subtype"]
			}`),
		},
		{
			Name:        "message_receive",
			Description: "세션의 대기 중인 메시지를 수신합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"session_id": {"type": "string", "description": "세션 ID"},
					"limit": {"type": "integer", "description": "최대 개수 (default: 10)"}
				},
				"required": ["session_id"]
			}`),
		},
		{
			Name:        "handoff_create",
			Description: "포트 간 컨텍스트 핸드오프를 생성합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from_port": {"type": "string", "description": "발신 포트 ID"},
					"to_port": {"type": "string", "description": "수신 포트 ID"},
					"type": {"type": "string", "enum": ["api_contract", "file_list", "type_def", "schema", "custom"], "description": "핸드오프 타입"},
					"content": {"type": "object", "description": "핸드오프 내용"}
				},
				"required": ["from_port", "to_port", "type", "content"]
			}`),
		},
		{
			Name:        "handoff_get",
			Description: "포트의 핸드오프를 조회합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"port_id": {"type": "string", "description": "포트 ID"},
					"direction": {"type": "string", "enum": ["from", "to", "both"], "description": "방향 (default: to)"}
				},
				"required": ["port_id"]
			}`),
		},
		{
			Name:        "agent_list",
			Description: "에이전트 목록을 조회합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"type": {"type": "string", "enum": ["spec", "operator", "worker", "test"], "description": "에이전트 타입 필터"}
				}
			}`),
		},
		{
			Name:        "agent_version",
			Description: "에이전트 버전 정보를 조회합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"agent_id": {"type": "string", "description": "에이전트 ID"},
					"version": {"type": "integer", "description": "버전 번호 (optional, 없으면 현재 버전)"}
				},
				"required": ["agent_id"]
			}`),
		},
		{
			Name:        "compact_record",
			Description: "Compact 이벤트를 기록합니다",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"session_id": {"type": "string", "description": "세션 ID"},
					"trigger_reason": {"type": "string", "description": "트리거 이유 (token_limit, focus_drift, manual)"},
					"before_tokens": {"type": "integer", "description": "Compact 전 토큰"},
					"after_tokens": {"type": "integer", "description": "Compact 후 토큰"},
					"preserved_context": {"type": "array", "items": {"type": "string"}, "description": "보존된 컨텍스트"},
					"discarded_context": {"type": "array", "items": {"type": "string"}, "description": "폐기된 컨텍스트"},
					"recovery_hint": {"type": "string", "description": "복구 힌트"}
				},
				"required": ["session_id", "trigger_reason", "before_tokens", "after_tokens"]
			}`),
		},
	}...)

	s.sendResult(req.ID, map[string]interface{}{"tools": tools})
}

func (s *Server) handleToolsCall(req *JSONRPCRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	var result interface{}
	var err error

	switch params.Name {
	// Claude 친화적 도구들
	case "pal_status":
		result, err = s.toolPalStatusHandler(params.Arguments)
	case "pal_port_start":
		result, err = s.toolPalPortStartHandler(params.Arguments)
	case "pal_port_end":
		result, err = s.toolPalPortEndHandler(params.Arguments)
	case "pal_checkpoint":
		result, err = s.toolPalCheckpointHandler(params.Arguments)
	case "pal_escalate":
		result, err = s.toolPalEscalateHandler(params.Arguments)
	case "pal_context":
		result, err = s.toolPalContextHandler(params.Arguments)
	case "pal_session":
		result, err = s.toolPalSessionHandler(params.Arguments)
	case "pal_hierarchy":
		result, err = s.toolPalHierarchyHandler(params.Arguments)
	// 기존 도구들
	case "session_start":
		result, err = s.toolSessionStart(params.Arguments)
	case "session_end":
		result, err = s.toolSessionEnd(params.Arguments)
	case "session_hierarchy":
		result, err = s.toolSessionHierarchy(params.Arguments)
	case "attention_status":
		result, err = s.toolAttentionStatus(params.Arguments)
	case "attention_update":
		result, err = s.toolAttentionUpdate(params.Arguments)
	case "orchestration_create":
		result, err = s.toolOrchestrationCreate(params.Arguments)
	case "orchestration_status":
		result, err = s.toolOrchestrationStatus(params.Arguments)
	case "message_send":
		result, err = s.toolMessageSend(params.Arguments)
	case "message_receive":
		result, err = s.toolMessageReceive(params.Arguments)
	case "handoff_create":
		result, err = s.toolHandoffCreate(params.Arguments)
	case "handoff_get":
		result, err = s.toolHandoffGet(params.Arguments)
	case "agent_list":
		result, err = s.toolAgentList(params.Arguments)
	case "agent_version":
		result, err = s.toolAgentVersion(params.Arguments)
	case "compact_record":
		result, err = s.toolCompactRecord(params.Arguments)
	default:
		s.sendError(req.ID, -32601, "Unknown tool", params.Name)
		return
	}

	if err != nil {
		s.sendResult(req.ID, map[string]interface{}{
			"content": []ContentItem{{Type: "text", Text: fmt.Sprintf("오류: %s", err.Error())}},
			"isError": true,
		})
		return
	}

	// Format result as text
	resultText, _ := json.MarshalIndent(result, "", "  ")
	s.sendResult(req.ID, map[string]interface{}{
		"content": []ContentItem{{Type: "text", Text: string(resultText)}},
	})
}

func (s *Server) handlePromptsList(req *JSONRPCRequest) {
	prompts := []Prompt{
		{
			Name:        "start_build",
			Description: "새 빌드 세션을 시작하는 프롬프트",
			Arguments: []PromptArgument{
				{Name: "title", Description: "프로젝트/기능 제목", Required: true},
				{Name: "requirements", Description: "요구사항 목록", Required: true},
			},
		},
		{
			Name:        "worker_context",
			Description: "Worker 세션의 컨텍스트를 로드하는 프롬프트",
			Arguments: []PromptArgument{
				{Name: "port_id", Description: "포트 ID", Required: true},
				{Name: "session_id", Description: "세션 ID", Required: true},
			},
		},
		{
			Name:        "test_feedback",
			Description: "테스트 결과 피드백을 생성하는 프롬프트",
			Arguments: []PromptArgument{
				{Name: "port_id", Description: "포트 ID", Required: true},
				{Name: "test_results", Description: "테스트 결과", Required: true},
			},
		},
	}

	s.sendResult(req.ID, map[string]interface{}{"prompts": prompts})
}

func (s *Server) handlePromptsGet(req *JSONRPCRequest) {
	var params struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	var messages []map[string]interface{}

	switch params.Name {
	case "start_build":
		messages = []map[string]interface{}{
			{
				"role": "user",
				"content": ContentItem{
					Type: "text",
					Text: fmt.Sprintf(`새 빌드 세션을 시작합니다.

제목: %s
요구사항:
%s

다음 단계:
1. session_start 도구로 build 세션을 시작하세요
2. 요구사항을 분석하여 Atomic Port로 분해하세요
3. orchestration_create로 실행 계획을 생성하세요`, params.Arguments["title"], params.Arguments["requirements"]),
				},
			},
		}

	case "worker_context":
		// Load handoffs for the port
		handoffs, _ := s.hoStore.GetForPort(params.Arguments["port_id"])
		handoffsJSON, _ := json.MarshalIndent(handoffs, "", "  ")

		messages = []map[string]interface{}{
			{
				"role": "user",
				"content": ContentItem{
					Type: "text",
					Text: fmt.Sprintf(`Worker 컨텍스트를 로드합니다.

포트 ID: %s
세션 ID: %s

핸드오프 정보:
%s

다음 단계:
1. attention_status로 현재 Attention 상태를 확인하세요
2. 핸드오프 정보를 기반으로 구현을 시작하세요
3. 구현 완료 시 message_send로 impl_ready를 전송하세요`, params.Arguments["port_id"], params.Arguments["session_id"], string(handoffsJSON)),
				},
			},
		}

	case "test_feedback":
		messages = []map[string]interface{}{
			{
				"role": "user",
				"content": ContentItem{
					Type: "text",
					Text: fmt.Sprintf(`테스트 결과를 분석합니다.

포트 ID: %s
테스트 결과:
%s

결과에 따라:
- 성공: message_send로 test_pass 전송
- 실패: message_send로 fix_request 전송 (구체적인 실패 사유 포함)`, params.Arguments["port_id"], params.Arguments["test_results"]),
				},
			},
		}

	default:
		s.sendError(req.ID, -32602, "Unknown prompt", params.Name)
		return
	}

	s.sendResult(req.ID, map[string]interface{}{"messages": messages})
}

func (s *Server) handleResourcesList(req *JSONRPCRequest) {
	resources := []Resource{
		{
			URI:         "pal://sessions/active",
			Name:        "Active Sessions",
			Description: "현재 활성화된 세션 목록",
			MimeType:    "application/json",
		},
		{
			URI:         "pal://orchestrations/running",
			Name:        "Running Orchestrations",
			Description: "실행 중인 Orchestration 목록",
			MimeType:    "application/json",
		},
		{
			URI:         "pal://agents",
			Name:        "Agents",
			Description: "등록된 에이전트 목록",
			MimeType:    "application/json",
		},
	}

	s.sendResult(req.ID, map[string]interface{}{"resources": resources})
}

func (s *Server) handleResourcesRead(req *JSONRPCRequest) {
	var params struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	var content interface{}

	switch params.URI {
	case "pal://sessions/active":
		sessions, _ := s.sessionSvc.GetBuildSessions(true, 20)
		content = sessions

	case "pal://orchestrations/running":
		orchestrations, _ := s.orchSvc.ListOrchestrations(orchestrator.StatusRunning, 20)
		content = orchestrations

	case "pal://agents":
		agents, _ := s.agentStore.ListAgents("")
		content = agents

	default:
		s.sendError(req.ID, -32602, "Unknown resource", params.URI)
		return
	}

	contentJSON, _ := json.MarshalIndent(content, "", "  ")
	s.sendResult(req.ID, map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      params.URI,
				"mimeType": "application/json",
				"text":     string(contentJSON),
			},
		},
	})
}

// Tool implementations
func (s *Server) toolSessionStart(args json.RawMessage) (interface{}, error) {
	var params struct {
		Title       string `json:"title"`
		Type        string `json:"type"`
		ParentID    string `json:"parent_id"`
		PortID      string `json:"port_id"`
		AgentID     string `json:"agent_id"`
		TokenBudget int    `json:"token_budget"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if params.TokenBudget == 0 {
		params.TokenBudget = 15000
	}

	sess, err := s.sessionSvc.StartHierarchical(session.HierarchyStartOptions{
		Title:       params.Title,
		Type:        params.Type,
		ParentID:    params.ParentID,
		PortID:      params.PortID,
		AgentID:     params.AgentID,
		TokenBudget: params.TokenBudget,
		ProjectRoot: s.projectRoot,
	})

	if err != nil {
		return nil, err
	}

	// Initialize attention tracking
	s.attStore.Initialize(sess.ID, params.PortID, params.TokenBudget)

	return sess, nil
}

func (s *Server) toolSessionEnd(args json.RawMessage) (interface{}, error) {
	var params struct {
		SessionID string      `json:"session_id"`
		Status    string      `json:"status"`
		Summary   interface{} `json:"summary"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if err := s.sessionSvc.EndWithSummary(params.SessionID, params.Status, params.Summary); err != nil {
		return nil, err
	}

	return map[string]string{"status": "ended", "session_id": params.SessionID}, nil
}

func (s *Server) toolSessionHierarchy(args json.RawMessage) (interface{}, error) {
	var params struct {
		RootID string `json:"root_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	return s.sessionSvc.GetSessionHierarchy(params.RootID)
}

func (s *Server) toolAttentionStatus(args json.RawMessage) (interface{}, error) {
	var params struct {
		SessionID string `json:"session_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	return s.attStore.GenerateReport(params.SessionID)
}

func (s *Server) toolAttentionUpdate(args json.RawMessage) (interface{}, error) {
	var params struct {
		SessionID         string   `json:"session_id"`
		LoadedTokens      int      `json:"loaded_tokens"`
		FocusScore        float64  `json:"focus_score"`
		LoadedFiles       []string `json:"loaded_files"`
		LoadedConventions []string `json:"loaded_conventions"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if params.LoadedTokens > 0 {
		if err := s.attStore.UpdateTokens(params.SessionID, params.LoadedTokens); err != nil {
			return nil, err
		}
	}

	if params.FocusScore > 0 {
		if err := s.attStore.UpdateFocusScore(params.SessionID, params.FocusScore); err != nil {
			return nil, err
		}
	}

	if len(params.LoadedFiles) > 0 || len(params.LoadedConventions) > 0 {
		if err := s.attStore.UpdateLoadedContext(params.SessionID, params.LoadedFiles, params.LoadedConventions, ""); err != nil {
			return nil, err
		}
	}

	return s.attStore.Get(params.SessionID)
}

func (s *Server) toolOrchestrationCreate(args json.RawMessage) (interface{}, error) {
	var params struct {
		Title       string                    `json:"title"`
		Description string                    `json:"description"`
		Ports       []orchestrator.AtomicPort `json:"ports"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	return s.orchSvc.CreateOrchestration(params.Title, params.Description, params.Ports)
}

func (s *Server) toolOrchestrationStatus(args json.RawMessage) (interface{}, error) {
	var params struct {
		OrchestrationID string `json:"orchestration_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	orch, err := s.orchSvc.GetOrchestration(params.OrchestrationID)
	if err != nil {
		return nil, err
	}

	stats, _ := s.orchSvc.GetOrchestrationStats(params.OrchestrationID)

	return map[string]interface{}{
		"orchestration": orch,
		"stats":         stats,
	}, nil
}

func (s *Server) toolMessageSend(args json.RawMessage) (interface{}, error) {
	var params struct {
		FromSession string      `json:"from_session"`
		ToSession   string      `json:"to_session"`
		Type        string      `json:"type"`
		Subtype     string      `json:"subtype"`
		PortID      string      `json:"port_id"`
		Payload     interface{} `json:"payload"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	msg := &message.Message{
		FromSession: params.FromSession,
		ToSession:   params.ToSession,
		Type:        message.MessageType(params.Type),
		Subtype:     message.MessageSubtype(params.Subtype),
		PortID:      params.PortID,
		Payload:     params.Payload,
	}

	if err := s.msgStore.Send(msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *Server) toolMessageReceive(args json.RawMessage) (interface{}, error) {
	var params struct {
		SessionID string `json:"session_id"`
		Limit     int    `json:"limit"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	return s.msgStore.Receive(params.SessionID, params.Limit)
}

func (s *Server) toolHandoffCreate(args json.RawMessage) (interface{}, error) {
	var params struct {
		FromPort string      `json:"from_port"`
		ToPort   string      `json:"to_port"`
		Type     string      `json:"type"`
		Content  interface{} `json:"content"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	return s.hoStore.Create(params.FromPort, params.ToPort, handoff.HandoffType(params.Type), params.Content)
}

func (s *Server) toolHandoffGet(args json.RawMessage) (interface{}, error) {
	var params struct {
		PortID    string `json:"port_id"`
		Direction string `json:"direction"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	switch params.Direction {
	case "from":
		return s.hoStore.GetFromPort(params.PortID)
	case "both":
		from, _ := s.hoStore.GetFromPort(params.PortID)
		to, _ := s.hoStore.GetForPort(params.PortID)
		return map[string]interface{}{"from": from, "to": to}, nil
	default: // "to" or empty
		return s.hoStore.GetForPort(params.PortID)
	}
}

func (s *Server) toolAgentList(args json.RawMessage) (interface{}, error) {
	var params struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	return s.agentStore.ListAgents(agentv2.AgentType(params.Type))
}

func (s *Server) toolAgentVersion(args json.RawMessage) (interface{}, error) {
	var params struct {
		AgentID string `json:"agent_id"`
		Version int    `json:"version"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	if params.Version == 0 {
		return s.agentStore.GetCurrentVersion(params.AgentID)
	}

	return s.agentStore.GetVersion(params.AgentID, params.Version)
}

func (s *Server) toolCompactRecord(args json.RawMessage) (interface{}, error) {
	var params struct {
		SessionID        string   `json:"session_id"`
		TriggerReason    string   `json:"trigger_reason"`
		BeforeTokens     int      `json:"before_tokens"`
		AfterTokens      int      `json:"after_tokens"`
		PreservedContext []string `json:"preserved_context"`
		DiscardedContext []string `json:"discarded_context"`
		RecoveryHint     string   `json:"recovery_hint"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, err
	}

	event := &attention.CompactEvent{
		SessionID:        params.SessionID,
		TriggerReason:    params.TriggerReason,
		BeforeTokens:     params.BeforeTokens,
		AfterTokens:      params.AfterTokens,
		PreservedContext: params.PreservedContext,
		DiscardedContext: params.DiscardedContext,
		RecoveryHint:     params.RecoveryHint,
	}

	if err := s.attStore.RecordCompact(event); err != nil {
		return nil, err
	}

	return event, nil
}

// Helper methods
func (s *Server) sendResult(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.send(resp)
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.send(resp)
}

func (s *Server) send(resp JSONRPCResponse) {
	data, _ := json.Marshal(resp)
	fmt.Fprintf(s.writer, "%s\n", data)
}
