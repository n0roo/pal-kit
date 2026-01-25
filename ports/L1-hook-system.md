# Port: L1-hook-system

> Claude Code Hook 완전 연동 - 모든 이벤트에서 PAL Kit 자동 호출

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | L1-hook-system |
| 타입 | atomic |
| 레이어 | L1 (Core) |
| 상태 | complete |
| 우선순위 | critical |
| 의존성 | - |
| 예상 토큰 | 8,000 |

---

## 목표

Claude Code의 모든 Hook 이벤트에서 PAL Kit이 자동으로 연동되어, **사용자 개입 없이** 상태 추적, 컨텍스트 관리가 이루어지도록 한다.

---

## Claude Code Hook 명세

### 사용 가능한 Hook 이벤트

| Hook | 트리거 시점 | PAL Kit 활용 |
|------|------------|--------------|
| `PreToolUse` | 도구 실행 전 | 토큰 체크, 체크포인트 |
| `PostToolUse` | 도구 실행 후 | 결과 추적, 파일 변경 기록 |
| `Notification` | 알림 발생 시 | Compact 감지, 에러 처리 |
| `Stop` | 중단 요청 시 | 상태 저장 |
| `SubagentSpawn` | Task tool 실행 시 | 세션 계층 연결, Handoff |

---

## 작업 항목

### 1. Hook 설정 자동 생성

- [ ] `pal init` 시 `.claude/settings.json` 자동 구성
  ```json
  {
    "hooks": {
      "PreToolUse": [
        {
          "matcher": "Edit|Write|Bash",
          "hooks": [
            {
              "type": "command",
              "command": "pal hook pre-tool"
            }
          ]
        }
      ],
      "PostToolUse": [
        {
          "matcher": "Edit|Write|Bash|Task",
          "hooks": [
            {
              "type": "command", 
              "command": "pal hook post-tool"
            }
          ]
        }
      ],
      "Notification": [
        {
          "matcher": ".*",
          "hooks": [
            {
              "type": "command",
              "command": "pal hook notification"
            }
          ]
        }
      ],
      "Stop": [
        {
          "matcher": ".*",
          "hooks": [
            {
              "type": "command",
              "command": "pal hook stop"
            }
          ]
        }
      ]
    }
  }
  ```

### 2. PreToolUse Hook 강화

- [ ] `internal/cli/hook.go` - `runHookPreToolUse` 수정
  ```go
  func runHookPreToolUse(cmd *cobra.Command, args []string) error {
      input, _ := readHookInput()
      
      // 1. 토큰 사용량 체크
      attentionState := getAttentionState(input.SessionID)
      usage := attentionState.UsageRatio()
      
      if usage >= 0.8 && usage < 0.9 {
          // 자동 체크포인트 생성
          cp := createCheckpoint(input.SessionID, "auto_80")
          
          // Claude에 알림 (stdout으로 - Claude가 읽음)
          output := HookOutput{
              Continue: true,
              HookOutput: map[string]interface{}{
                  "checkpoint": cp.ID,
                  "message":    fmt.Sprintf("체크포인트 생성됨 (토큰 %.0f%%)", usage*100),
              },
          }
          json.NewEncoder(os.Stdout).Encode(output)
      }
      
      if usage >= 0.9 {
          // 강한 경고
          output := HookOutput{
              Continue: true,
              HookOutput: map[string]interface{}{
                  "warning": fmt.Sprintf("토큰 %.0f%% - 작업 마무리 권장", usage*100),
                  "suggestion": "현재 작업을 완료하고 새 포트로 분리하세요",
              },
          }
          json.NewEncoder(os.Stdout).Encode(output)
      }
      
      // 2. 활성 포트 없이 코드 수정 시 경고
      if (input.ToolName == "Edit" || input.ToolName == "Write") && !hasActivePort() {
          output := HookOutput{
              Continue: true,
              HookOutput: map[string]interface{}{
                  "warning": "활성 포트 없음 - 작업이 추적되지 않습니다",
                  "suggestion": "pal_port_start 도구로 포트를 시작하세요",
              },
          }
          json.NewEncoder(os.Stdout).Encode(output)
      }
      
      return nil
  }
  ```

### 3. PostToolUse Hook 추가

- [ ] `internal/cli/hook.go` - `runHookPostToolUse` 강화
  ```go
  func runHookPostToolUse(cmd *cobra.Command, args []string) error {
      input, _ := readHookInput()
      
      // 1. 파일 변경 기록
      if input.ToolName == "Edit" || input.ToolName == "Write" {
          filePath := input.ToolInput["file_path"].(string)
          recordFileChange(input.SessionID, filePath)
      }
      
      // 2. Bash 결과 분석 (빌드/테스트 실패 감지)
      if input.ToolName == "Bash" {
          result := input.ToolResponse
          if isBuildFailure(result) || isTestFailure(result) {
              output := HookOutput{
                  HookOutput: map[string]interface{}{
                      "build_status": "failed",
                      "suggestion": "에러를 수정한 후 다시 시도하세요",
                  },
              }
              json.NewEncoder(os.Stdout).Encode(output)
          }
      }
      
      // 3. Task tool (서브에이전트) 결과 처리
      if input.ToolName == "Task" {
          taskResult := input.ToolResponse
          handleSubagentComplete(input.SessionID, taskResult)
      }
      
      return nil
  }
  ```

### 4. Notification Hook (Compact 감지)

- [ ] `internal/cli/hook.go` - `runHookNotification` 신규
  ```go
  func runHookNotification(cmd *cobra.Command, args []string) error {
      input, _ := readHookInput()
      
      // Compact 감지
      if input.NotificationType == "compact" || 
         strings.Contains(input.Message, "compact") {
          // 마지막 체크포인트 조회
          lastCP := getLatestCheckpoint(input.SessionID)
          
          output := HookOutput{
              HookOutput: map[string]interface{}{
                  "event":      "compact_detected",
                  "checkpoint": lastCP.ID,
                  "summary":    lastCP.Summary,
                  "recovery":   "마지막 체크포인트에서 컨텍스트를 복구할 수 있습니다",
              },
          }
          json.NewEncoder(os.Stdout).Encode(output)
          
          // 컴팩트 이벤트 기록
          recordCompactEvent(input.SessionID)
      }
      
      return nil
  }
  ```

### 5. SubagentSpawn Hook (Task tool 연동)

- [ ] `internal/cli/hook.go` - `runHookSubagentSpawn` 신규
  ```go
  func runHookSubagentSpawn(cmd *cobra.Command, args []string) error {
      input, _ := readHookInput()
      
      // 부모 세션 확인
      parentSessionID := input.SessionID
      
      // 자식 세션 생성 (계층 연결)
      childSession := createChildSession(parentSessionID)
      
      // Handoff 컨텍스트 생성
      handoff := generateHandoff(parentSessionID, childSession.ID)
      
      // 자식 에이전트에게 전달할 컨텍스트
      output := HookOutput{
          HookOutput: map[string]interface{}{
              "child_session":  childSession.ID,
              "parent_session": parentSessionID,
              "handoff":        handoff,
              "port_context":   getActivePortContext(),
          },
      }
      json.NewEncoder(os.Stdout).Encode(output)
      
      return nil
  }
  ```

### 6. Hook 출력 형식 표준화

```go
// HookOutput - Claude가 읽는 JSON 형식
type HookOutput struct {
    Continue   bool                   `json:"continue,omitempty"`
    Decision   string                 `json:"decision,omitempty"` // approve, block
    Reason     string                 `json:"reason,omitempty"`
    HookOutput map[string]interface{} `json:"hookSpecificOutput,omitempty"`
}
```

---

## Claude가 수신하는 정보

**PreToolUse 응답 예시:**
```json
{
  "continue": true,
  "hookSpecificOutput": {
    "checkpoint": "cp-abc123",
    "message": "체크포인트 생성됨 (토큰 82%)"
  }
}
```

**Notification 응답 예시 (Compact):**
```json
{
  "hookSpecificOutput": {
    "event": "compact_detected",
    "checkpoint": "cp-abc123",
    "summary": "User 엔티티 구현 중...",
    "recovery": "마지막 체크포인트에서 컨텍스트를 복구할 수 있습니다"
  }
}
```

---

## 완료 기준

- [ ] `pal init` 시 `.claude/settings.json`에 Hook 설정 자동 생성
- [ ] PreToolUse에서 토큰 체크 및 자동 체크포인트
- [ ] PostToolUse에서 파일 변경/빌드 결과 추적
- [ ] Notification에서 Compact 감지 및 복구 힌트
- [ ] SubagentSpawn에서 세션 계층 자동 연결

---

## 참조

- Claude Code Hook 문서: https://docs.anthropic.com/claude-code/hooks
- `internal/cli/hook.go` - 현재 Hook 구현

---

<!-- pal:port:L1-hook-system -->
