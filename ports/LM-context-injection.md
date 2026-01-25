# Port: LM-context-injection

> 자동 컨텍스트 주입 - Claude에게 적시에 필요한 정보 제공

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | LM-context-injection |
| 타입 | atomic |
| 레이어 | LM (Service) |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | L1-hook-system, LM-mcp-tools |
| 예상 토큰 | 8,000 |

---

## 목표

Claude가 작업에 필요한 컨텍스트(컨벤션, 문서, 포트 명세)를 **자동으로 받을 수 있도록** 시스템을 구성한다.

---

## 컨텍스트 주입 시점

| 시점 | 주입 내용 | 트리거 |
|------|----------|--------|
| 세션 시작 | 프로젝트 개요, 활성 작업 | Hook: SessionStart |
| 포트 시작 | 포트 명세, 에이전트 컨벤션 | MCP: pal_port_start |
| 파일 수정 | 관련 문서, 컨벤션 | Hook: PreToolUse |
| Compact 후 | 체크포인트 요약, 핵심 컨텍스트 | Hook: Notification |

---

## 작업 항목

### 1. 세션 시작 컨텍스트

- [ ] `internal/context/session_context.go`
  ```go
  type SessionContext struct {
      ProjectOverview string     `json:"project_overview"`
      ActivePorts     []PortInfo `json:"active_ports"`
      PendingPorts    []PortInfo `json:"pending_ports"`
      Conventions     []string   `json:"conventions"`
      WorkflowType    string     `json:"workflow_type"`
      RecentDecisions []string   `json:"recent_decisions"`
  }
  
  func GenerateSessionContext(projectRoot string) (*SessionContext, error) {
      ctx := &SessionContext{}
      
      // CLAUDE.md에서 프로젝트 개요 추출
      ctx.ProjectOverview = extractProjectOverview(projectRoot)
      
      // 활성/대기 포트
      ctx.ActivePorts = getPorts("running")
      ctx.PendingPorts = getPorts("pending")
      
      // 워크플로우 타입
      ctx.WorkflowType = getWorkflowType()
      
      // 최근 결정 사항 (이전 세션에서)
      ctx.RecentDecisions = getRecentDecisions(5)
      
      return ctx, nil
  }
  ```

### 2. 포트 시작 컨텍스트

- [ ] `internal/context/port_context.go`
  ```go
  type PortContext struct {
      PortSpec      string   `json:"port_spec"`       // 포트 명세 전체
      AgentPrompt   string   `json:"agent_prompt"`    // 에이전트 프롬프트
      Conventions   []string `json:"conventions"`     // 관련 컨벤션
      RelatedDocs   []string `json:"related_docs"`    // 관련 문서
      Checklist     []string `json:"checklist"`       // 완료 체크리스트
      Dependencies  []string `json:"dependencies"`    // 의존 포트 결과
      TokenEstimate int      `json:"token_estimate"`  // 예상 토큰
  }
  
  func GeneratePortContext(portID string) (*PortContext, error) {
      port := getPort(portID)
      agent := mapAgent(port)
      
      ctx := &PortContext{}
      
      // 포트 명세
      if port.FilePath != "" {
          ctx.PortSpec = readFile(port.FilePath)
      }
      
      // 에이전트 프롬프트 + 컨벤션
      ctx.AgentPrompt = agent.Prompt
      ctx.Conventions = loadConventions(agent.ConventionsRef)
      
      // 관련 문서 (토큰 예산 내에서)
      ctx.RelatedDocs = findRelatedDocs(port, 10000)
      
      // 의존 포트 결과 (Handoff)
      ctx.Dependencies = getDependencyHandoffs(portID)
      
      // 체크리스트
      ctx.Checklist = agent.Checklist
      
      return ctx, nil
  }
  ```

### 3. 파일 수정 시 컨텍스트

- [ ] `internal/context/file_context.go`
  ```go
  type FileContext struct {
      RelatedConventions []string `json:"related_conventions"`
      RelatedDocs        []string `json:"related_docs"`
      LayerInfo          string   `json:"layer_info"` // L1/L2/LM
  }
  
  func GenerateFileContext(filePath string) (*FileContext, error) {
      ctx := &FileContext{}
      
      // 파일 경로/이름에서 레이어 추론
      ctx.LayerInfo = inferLayer(filePath)
      
      // 레이어에 맞는 컨벤션
      ctx.RelatedConventions = getLayerConventions(ctx.LayerInfo)
      
      // 관련 문서
      ctx.RelatedDocs = findDocsByFile(filePath, 5000)
      
      return ctx, nil
  }
  ```

### 4. Compact 복구 컨텍스트

- [ ] `internal/context/recovery_context.go`
  ```go
  type RecoveryContext struct {
      CheckpointSummary string   `json:"checkpoint_summary"`
      ActivePort        string   `json:"active_port"`
      PortProgress      string   `json:"port_progress"`
      PendingTasks      []string `json:"pending_tasks"`
      RecentFiles       []string `json:"recent_files"`
      KeyDecisions      []string `json:"key_decisions"`
  }
  
  func GenerateRecoveryContext(sessionID string) (*RecoveryContext, error) {
      // 마지막 체크포인트에서 복구
      checkpoint := getLatestCheckpoint(sessionID)
      
      ctx := &RecoveryContext{
          CheckpointSummary: checkpoint.Summary,
          ActivePort:        checkpoint.PortID,
          RecentFiles:       checkpoint.ActiveFiles,
      }
      
      // 포트 진행 상태
      if checkpoint.PortID != "" {
          port := getPort(checkpoint.PortID)
          ctx.PortProgress = generatePortProgress(port)
          ctx.PendingTasks = getRemainingChecklist(port)
      }
      
      // 주요 결정 사항
      ctx.KeyDecisions = getDecisionEvents(sessionID, 3)
      
      return ctx, nil
  }
  ```

### 5. CLAUDE.md 동적 업데이트

- [ ] `internal/context/claudemd.go`
  ```go
  func UpdateClaudeMD(projectRoot string) error {
      claudeMD := filepath.Join(projectRoot, "CLAUDE.md")
      
      // PAL Kit 섹션 생성
      section := generatePALSection()
      
      // CLAUDE.md에 주입 (기존 내용 보존)
      return injectSection(claudeMD, "<!-- PAL-KIT-START -->", "<!-- PAL-KIT-END -->", section)
  }
  
  func generatePALSection() string {
      return `
  ## PAL Kit 연동
  
  이 프로젝트는 PAL Kit으로 관리됩니다.
  
  ### 사용 가능한 MCP 도구
  - \`pal_status\` - 현재 상태 확인
  - \`pal_port_start\` - 작업 시작
  - \`pal_port_end\` - 작업 완료 (자동 검증)
  - \`pal_checkpoint\` - 체크포인트 관리
  - \`pal_escalate\` - 에스컬레이션
  
  ### 워크플로우
  1. 작업 시작 시 \`pal_status\`로 상태 확인
  2. \`pal_port_start\`로 포트 시작
  3. 작업 완료 시 \`pal_port_end\` 호출
  4. 자동 체크리스트 검증 후 완료/블록 처리
  `
  }
  ```

### 6. Rules 파일 자동 관리

- [ ] `internal/rules/auto_rules.go`
  ```go
  // 포트 시작 시 자동 rules 파일 생성
  func ActivatePortRules(portID string, context *PortContext) error {
      rulePath := filepath.Join(".claude", "rules", portID+".md")
      
      content := fmt.Sprintf(`---
  description: Port %s - %s
  globs:
    - "**/*"
  alwaysApply: true
  ---
  
  # 포트: %s
  
  %s
  
  ## 체크리스트
  %s
  
  ## 컨벤션
  %s
  `, portID, context.PortTitle, portID, 
     context.PortSpec,
     formatChecklist(context.Checklist),
     formatConventions(context.Conventions))
      
      return writeFile(rulePath, content)
  }
  
  // 포트 완료 시 rules 파일 제거
  func DeactivatePortRules(portID string) error {
      rulePath := filepath.Join(".claude", "rules", portID+".md")
      return os.Remove(rulePath)
  }
  ```

---

## 컨텍스트 흐름

```
[세션 시작]
    ↓
Hook: SessionStart
    ↓
GenerateSessionContext() → CLAUDE.md 업데이트, Operator Briefing
    ↓
Claude: "pal_status"
    ↓
Claude: 상태 확인, 작업 계획

[포트 시작]
    ↓
Claude: "pal_port_start {id, title}"
    ↓
GeneratePortContext() → Rules 파일 생성
    ↓
Claude: 포트 명세, 컨벤션, 체크리스트 수신
    ↓
Claude: 작업 시작

[Compact 발생]
    ↓
Hook: Notification (compact)
    ↓
GenerateRecoveryContext()
    ↓
Claude: 체크포인트 요약, 진행 상태 수신
    ↓
Claude: 작업 계속
```

---

## 완료 기준

- [ ] 세션 시작 시 컨텍스트 자동 주입
- [ ] 포트 시작 시 Rules 파일 자동 생성
- [ ] Compact 발생 시 복구 컨텍스트 제공
- [ ] CLAUDE.md PAL Kit 섹션 자동 관리

---

## 참조

- `internal/context/` - 기존 컨텍스트 관리
- `internal/rules/` - Rules 파일 관리

---

<!-- pal:port:LM-context-injection -->
