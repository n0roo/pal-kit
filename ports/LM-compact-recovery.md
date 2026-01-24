# Port: LM-compact-recovery

> Compact 자동 복구 - Compact 발생 시 컨텍스트 자동 복구

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | LM-compact-recovery |
| 타입 | atomic |
| 레이어 | LM (Service) |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | L1-hook-system, L1-auto-checkpoint |
| 예상 토큰 | 6,000 |

---

## 목표

Compact 발생 시 **자동으로** 마지막 체크포인트에서 컨텍스트를 복구하여 Claude가 작업을 계속할 수 있도록 한다.

---

## 동작 흐름

```
[Compact 발생]
    ↓
Hook: Notification (type: compact)
    ↓
pal hook notification 실행
    ↓
[자동 복구 컨텍스트 생성]
├── 마지막 체크포인트 조회
├── 활성 포트 상태 확인
├── 핵심 컨텍스트 추출
└── 복구 힌트 생성
    ↓
Claude에 복구 컨텍스트 전달
    ↓
Claude: 컨텍스트 인지 후 작업 계속
```

---

## 작업 항목

### 1. Notification Hook 구현

- [ ] `internal/cli/hook.go` - `runHookNotification` 신규
  ```go
  var hookNotificationCmd = &cobra.Command{
      Use:   "notification",
      Short: "Notification Hook",
      RunE:  runHookNotification,
  }
  
  func init() {
      hookCmd.AddCommand(hookNotificationCmd)
  }
  
  func runHookNotification(cmd *cobra.Command, args []string) error {
      input, _ := readHookInput()
      
      // Compact 감지
      isCompact := input.NotificationType == "compact" ||
                   strings.Contains(strings.ToLower(input.Message), "compact") ||
                   strings.Contains(strings.ToLower(input.Message), "context")
      
      if !isCompact {
          // Compact가 아니면 기본 처리
          return nil
      }
      
      database, _ := db.Open(GetDBPath())
      defer database.Close()
      
      // 복구 컨텍스트 생성
      recoverySvc := recovery.NewService(database)
      recoveryCtx, err := recoverySvc.GenerateRecoveryContext(input.SessionID)
      if err != nil {
          return nil
      }
      
      // Compact 이벤트 기록
      sessionSvc := session.NewService(database)
      sessionSvc.IncrementCompact(input.SessionID)
      sessionSvc.LogEvent(input.SessionID, "compact", 
          fmt.Sprintf(`{"checkpoint":"%s"}`, recoveryCtx.CheckpointID))
      
      // Claude에 복구 컨텍스트 전달
      output := HookOutput{
          HookOutput: map[string]interface{}{
              "event":           "compact_recovery",
              "checkpoint_id":   recoveryCtx.CheckpointID,
              "summary":         recoveryCtx.Summary,
              "active_port":     recoveryCtx.ActivePort,
              "port_progress":   recoveryCtx.PortProgress,
              "pending_tasks":   recoveryCtx.PendingTasks,
              "recent_files":    recoveryCtx.RecentFiles,
              "key_decisions":   recoveryCtx.KeyDecisions,
              "recovery_prompt": recoveryCtx.RecoveryPrompt,
          },
      }
      
      json.NewEncoder(os.Stdout).Encode(output)
      return nil
  }
  ```

### 2. 복구 컨텍스트 서비스

- [ ] `internal/recovery/service.go`
  ```go
  type Service struct {
      db          *db.DB
      checkpoint  *checkpoint.Store
      port        *port.Service
      session     *session.Service
  }
  
  type RecoveryContext struct {
      CheckpointID   string   `json:"checkpoint_id"`
      Summary        string   `json:"summary"`
      ActivePort     string   `json:"active_port"`
      PortProgress   string   `json:"port_progress"`
      PendingTasks   []string `json:"pending_tasks"`
      RecentFiles    []string `json:"recent_files"`
      KeyDecisions   []string `json:"key_decisions"`
      RecoveryPrompt string   `json:"recovery_prompt"`
  }
  
  func (s *Service) GenerateRecoveryContext(sessionID string) (*RecoveryContext, error) {
      ctx := &RecoveryContext{}
      
      // 마지막 체크포인트
      cp, err := s.checkpoint.GetLatest(sessionID)
      if err == nil && cp != nil {
          ctx.CheckpointID = cp.ID
          ctx.Summary = cp.Summary
          ctx.RecentFiles = cp.ActiveFiles
          ctx.KeyDecisions = cp.KeyPoints
      }
      
      // 활성 포트
      ports, _ := s.port.List("running", 1)
      if len(ports) > 0 {
          ctx.ActivePort = ports[0].ID
          ctx.PortProgress = s.generatePortProgress(ports[0])
          ctx.PendingTasks = s.getRemainingTasks(ports[0])
      }
      
      // 복구 프롬프트 생성
      ctx.RecoveryPrompt = s.generateRecoveryPrompt(ctx)
      
      return ctx, nil
  }
  
  func (s *Service) generateRecoveryPrompt(ctx *RecoveryContext) string {
      var sb strings.Builder
      
      sb.WriteString("## Compact 복구\n\n")
      
      if ctx.Summary != "" {
          sb.WriteString(fmt.Sprintf("**마지막 상태**: %s\n\n", ctx.Summary))
      }
      
      if ctx.ActivePort != "" {
          sb.WriteString(fmt.Sprintf("**활성 포트**: %s\n", ctx.ActivePort))
          sb.WriteString(fmt.Sprintf("**진행 상황**: %s\n\n", ctx.PortProgress))
      }
      
      if len(ctx.PendingTasks) > 0 {
          sb.WriteString("**남은 작업**:\n")
          for _, task := range ctx.PendingTasks {
              sb.WriteString(fmt.Sprintf("- [ ] %s\n", task))
          }
          sb.WriteString("\n")
      }
      
      if len(ctx.KeyDecisions) > 0 {
          sb.WriteString("**주요 결정**:\n")
          for _, dec := range ctx.KeyDecisions {
              sb.WriteString(fmt.Sprintf("- %s\n", dec))
          }
      }
      
      return sb.String()
  }
  
  func (s *Service) generatePortProgress(p *port.Port) string {
      // 체크리스트 진행률 계산
      checklist := getPortChecklist(p.ID)
      completed := countCompleted(checklist)
      total := len(checklist)
      
      return fmt.Sprintf("%d/%d 완료 (%.0f%%)", 
          completed, total, float64(completed)/float64(total)*100)
  }
  
  func (s *Service) getRemainingTasks(p *port.Port) []string {
      checklist := getPortChecklist(p.ID)
      var remaining []string
      for _, item := range checklist {
          if !item.Completed {
              remaining = append(remaining, item.Description)
          }
      }
      return remaining
  }
  ```

### 3. Claude에 전달되는 형식

```json
{
  "hookSpecificOutput": {
    "event": "compact_recovery",
    "checkpoint_id": "cp-abc123",
    "summary": "User 엔티티 구현 중 - Create, Read 완료",
    "active_port": "user-entity",
    "port_progress": "3/5 완료 (60%)",
    "pending_tasks": [
      "Update 메서드 구현",
      "Delete 메서드 구현"
    ],
    "recent_files": [
      "internal/entity/user.go",
      "internal/entity/user_test.go"
    ],
    "key_decisions": [
      "UUID v7 사용",
      "soft delete 적용"
    ],
    "recovery_prompt": "## Compact 복구\n\n**마지막 상태**: User 엔티티 구현 중...\n\n**활성 포트**: user-entity\n**진행 상황**: 3/5 완료 (60%)\n\n**남은 작업**:\n- [ ] Update 메서드 구현\n- [ ] Delete 메서드 구현\n\n**주요 결정**:\n- UUID v7 사용\n- soft delete 적용"
  }
}
```

---

## Claude 동작 예시

```
[Compact 발생]
    ↓
Claude: Hook 출력 수신
    ↓
Claude: "Compact가 발생했습니다. 이전 작업을 계속하겠습니다.
         
현재 상태:
- 포트: user-entity (60% 완료)
- 남은 작업: Update, Delete 구현

Update 메서드부터 구현하겠습니다."
    ↓
Claude: [작업 계속]
```

---

## 완료 기준

- [ ] Notification Hook에서 Compact 감지
- [ ] 마지막 체크포인트에서 복구 컨텍스트 생성
- [ ] Claude에 구조화된 복구 정보 전달
- [ ] Claude가 자연스럽게 작업 계속 가능

---

<!-- pal:port:LM-compact-recovery -->
