# Port: LM-mcp-tools

> PAL Kit MCP 도구 - Claude가 직접 호출하는 PAL 기능

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | LM-mcp-tools |
| 타입 | atomic |
| 레이어 | LM (Service) |
| 상태 | complete |
| 우선순위 | critical |
| 의존성 | L1-hook-system |
| 예상 토큰 | 10,000 |

---

## 목표

Claude가 자연스럽게 호출할 수 있는 MCP 도구를 제공하여, **Claude가 PAL Kit 기능을 직접 사용**할 수 있게 한다.

---

## MCP 도구 목록

| 도구 | 설명 | Claude 사용 시나리오 |
|------|------|---------------------|
| `pal_status` | 현재 상태 조회 | 작업 시작 시 상태 확인 |
| `pal_port_start` | 포트 시작 | 새 작업 시작 시 |
| `pal_port_end` | 포트 완료 | 작업 완료 시 (자동 검증) |
| `pal_checkpoint` | 체크포인트 관리 | 주요 결정 후 저장 |
| `pal_escalate` | 에스컬레이션 | 문제 발생 시 |
| `pal_handoff` | Handoff 생성 | 서브에이전트 전환 시 |
| `pal_context` | 컨텍스트 조회 | 컨벤션/문서 로드 |

---

## 작업 항목

### 1. MCP 서버 구현

- [ ] `internal/mcp/server.go` 확장
  ```go
  type MCPServer struct {
      db          *db.DB
      projectRoot string
  }
  
  func (s *MCPServer) HandleToolCall(tool string, params map[string]interface{}) (interface{}, error) {
      switch tool {
      case "pal_status":
          return s.handleStatus(params)
      case "pal_port_start":
          return s.handlePortStart(params)
      case "pal_port_end":
          return s.handlePortEnd(params)
      case "pal_checkpoint":
          return s.handleCheckpoint(params)
      case "pal_escalate":
          return s.handleEscalate(params)
      case "pal_handoff":
          return s.handleHandoff(params)
      case "pal_context":
          return s.handleContext(params)
      default:
          return nil, fmt.Errorf("unknown tool: %s", tool)
      }
  }
  ```

### 2. pal_status 도구

- [ ] Claude가 작업 시작 시 현재 상태 파악
  ```go
  // 입력: 없음
  // 출력:
  type StatusResult struct {
      Session       *SessionInfo     `json:"session"`
      ActivePorts   []PortInfo       `json:"active_ports"`
      PendingPorts  []PortInfo       `json:"pending_ports"`
      Attention     *AttentionState  `json:"attention"`
      LastCheckpoint *CheckpointInfo `json:"last_checkpoint"`
      Suggestions   []string         `json:"suggestions"`
  }
  
  func (s *MCPServer) handleStatus(params map[string]interface{}) (*StatusResult, error) {
      result := &StatusResult{}
      
      // 현재 세션
      result.Session = getCurrentSession()
      
      // 활성/대기 포트
      result.ActivePorts = getPorts("running")
      result.PendingPorts = getPorts("pending")
      
      // Attention 상태
      result.Attention = getAttentionState()
      
      // 마지막 체크포인트
      result.LastCheckpoint = getLatestCheckpoint()
      
      // 상황별 제안
      if len(result.ActivePorts) == 0 {
          result.Suggestions = append(result.Suggestions, 
              "활성 포트가 없습니다. pal_port_start로 포트를 시작하세요.")
      }
      if result.Attention.UsageRatio > 0.7 {
          result.Suggestions = append(result.Suggestions,
              fmt.Sprintf("토큰 사용량 %.0f%%. 작업을 마무리하는 것을 권장합니다.", 
                  result.Attention.UsageRatio*100))
      }
      
      return result, nil
  }
  ```

### 3. pal_port_start 도구

- [ ] Claude가 새 작업을 시작할 때 호출
  ```go
  // 입력:
  type PortStartInput struct {
      ID          string   `json:"id"`           // 포트 ID
      Title       string   `json:"title"`        // 포트 제목
      Description string   `json:"description"`  // 설명 (optional)
      Agent       string   `json:"agent"`        // 에이전트 타입 (optional)
  }
  
  // 출력:
  type PortStartResult struct {
      PortID      string   `json:"port_id"`
      Status      string   `json:"status"`
      Context     string   `json:"context"`      // 로드된 컨텍스트
      Checklist   []string `json:"checklist"`    // 완료 체크리스트
      Conventions []string `json:"conventions"`  // 적용 컨벤션
  }
  
  func (s *MCPServer) handlePortStart(params map[string]interface{}) (*PortStartResult, error) {
      // 포트 생성/시작
      port := createOrGetPort(input.ID, input.Title)
      port.Status = "running"
      
      // 에이전트 매핑
      agent := mapAgent(input.Agent, port)
      
      // 컨텍스트 로드
      context := loadContext(agent, port)
      
      // 체크리스트 로드
      checklist := getChecklist(agent)
      
      return &PortStartResult{
          PortID:      port.ID,
          Status:      "running",
          Context:     context,
          Checklist:   checklist,
          Conventions: agent.Conventions,
      }, nil
  }
  ```

### 4. pal_port_end 도구

- [ ] Claude가 작업 완료 시 호출 (자동 체크리스트 검증 포함)
  ```go
  // 입력:
  type PortEndInput struct {
      ID      string `json:"id"`       // 포트 ID
      Summary string `json:"summary"`  // 작업 요약
  }
  
  // 출력:
  type PortEndResult struct {
      Status     string          `json:"status"`     // complete, blocked
      Checklist  *ChecklistResult `json:"checklist"`
      Message    string          `json:"message"`
      NextAction string          `json:"next_action,omitempty"`
  }
  
  func (s *MCPServer) handlePortEnd(params map[string]interface{}) (*PortEndResult, error) {
      // 체크리스트 자동 검증
      verifier := checklist.NewVerifier(s.projectRoot)
      checkResult := verifier.Verify()
      
      if !checkResult.Passed {
          // 실패: 포트 blocked, Claude에 피드백
          updatePortStatus(input.ID, "blocked")
          
          return &PortEndResult{
              Status:    "blocked",
              Checklist: checkResult,
              Message:   "체크리스트 검증 실패",
              NextAction: generateFixSuggestion(checkResult),
          }, nil
      }
      
      // 성공: 포트 complete
      updatePortStatus(input.ID, "complete")
      
      return &PortEndResult{
          Status:    "complete",
          Checklist: checkResult,
          Message:   "포트 완료",
      }, nil
  }
  ```

### 5. pal_checkpoint 도구

- [ ] Claude가 주요 결정 후 체크포인트 저장/복구
  ```go
  // 입력:
  type CheckpointInput struct {
      Action  string `json:"action"`  // create, restore, list
      ID      string `json:"id"`      // restore 시 필요
      Summary string `json:"summary"` // create 시 요약
  }
  
  // 출력 (create):
  type CheckpointCreateResult struct {
      ID        string `json:"id"`
      Summary   string `json:"summary"`
      TokensUsed int   `json:"tokens_used"`
  }
  
  // 출력 (list):
  type CheckpointListResult struct {
      Checkpoints []CheckpointInfo `json:"checkpoints"`
  }
  ```

### 6. pal_escalate 도구

- [ ] Claude가 문제 발생 시 에스컬레이션
  ```go
  // 입력:
  type EscalateInput struct {
      Type       string `json:"type"`        // user, architect, blocked
      Issue      string `json:"issue"`       // 문제 설명
      Context    string `json:"context"`     // 상황 컨텍스트
      Suggestion string `json:"suggestion"`  // 제안 (optional)
  }
  
  // 출력:
  type EscalateResult struct {
      EscalationID string `json:"escalation_id"`
      Status       string `json:"status"`
      Message      string `json:"message"`
  }
  ```

### 7. pal_context 도구

- [ ] Claude가 컨벤션/문서를 조회
  ```go
  // 입력:
  type ContextInput struct {
      Type   string   `json:"type"`   // convention, document, port
      Query  string   `json:"query"`  // 검색어
      Limit  int      `json:"limit"`  // 토큰 제한
  }
  
  // 출력:
  type ContextResult struct {
      Items      []ContextItem `json:"items"`
      TotalTokens int          `json:"total_tokens"`
  }
  ```

### 8. MCP 도구 스키마 정의

- [ ] `internal/mcp/schema.go`
  ```go
  var ToolSchemas = []MCPToolSchema{
      {
          Name:        "pal_status",
          Description: "PAL Kit 현재 상태 조회. 세션, 활성 포트, Attention 상태 확인.",
          InputSchema: map[string]interface{}{
              "type": "object",
              "properties": map[string]interface{}{},
          },
      },
      {
          Name:        "pal_port_start",
          Description: "새 작업 포트 시작. 컨텍스트와 체크리스트가 자동 로드됨.",
          InputSchema: map[string]interface{}{
              "type": "object",
              "properties": map[string]interface{}{
                  "id":    {"type": "string", "description": "포트 ID"},
                  "title": {"type": "string", "description": "포트 제목"},
              },
              "required": []string{"id", "title"},
          },
      },
      {
          Name:        "pal_port_end",
          Description: "작업 포트 완료. 자동으로 빌드/테스트 검증 실행.",
          InputSchema: map[string]interface{}{
              "type": "object",
              "properties": map[string]interface{}{
                  "id":      {"type": "string", "description": "포트 ID"},
                  "summary": {"type": "string", "description": "작업 요약"},
              },
              "required": []string{"id"},
          },
      },
      // ... 나머지 도구들
  }
  ```

---

## Claude 사용 예시

```
사용자: "User 엔티티 구현해줘"

Claude: [pal_status 호출]
→ 결과: 활성 포트 없음, 토큰 20%

Claude: [pal_port_start 호출]
→ 입력: {id: "user-entity", title: "User 엔티티 구현"}
→ 결과: 컨텍스트 로드, 체크리스트 수신

Claude: "User 엔티티를 구현하겠습니다."
[코드 작성...]

Claude: [pal_port_end 호출]
→ 입력: {id: "user-entity", summary: "User 엔티티 및 테스트 구현 완료"}
→ 결과: 체크리스트 통과, 포트 완료
```

---

## 완료 기준

- [ ] 모든 MCP 도구 구현 및 테스트
- [ ] Claude Code MCP 설정으로 연동 확인
- [ ] Claude가 자연스럽게 도구 호출

---

## 참조

- MCP 프로토콜: https://modelcontextprotocol.io
- `internal/mcp/server.go` - 현재 MCP 서버

---

<!-- pal:port:LM-mcp-tools -->
