# Port: L1-auto-checkpoint

> 자동 체크포인트 - Hook 기반 토큰 모니터링 및 자동 저장

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | L1-auto-checkpoint |
| 타입 | atomic |
| 레이어 | L1 (Core) |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | L1-hook-system |
| 예상 토큰 | 6,000 |

---

## 목표

**사용자 개입 없이** 토큰 사용량을 모니터링하고, 임계치 도달 시 자동으로 체크포인트를 생성한다.

---

## 트리거 조건

| 조건 | 트리거 | 동작 |
|------|--------|------|
| 토큰 80% | Hook: PreToolUse | 자동 체크포인트 + Claude 알림 |
| 토큰 90% | Hook: PreToolUse | 자동 체크포인트 + 강한 경고 |
| 주요 결정 | MCP: pal_checkpoint | Claude가 명시적 호출 |
| Compact 전 | Hook: Notification | 마지막 체크포인트 정보 제공 |

---

## 작업 항목

### 1. 체크포인트 저장소

- [ ] `internal/checkpoint/store.go`
  ```go
  type Checkpoint struct {
      ID          string    `json:"id"`
      SessionID   string    `json:"session_id"`
      PortID      string    `json:"port_id,omitempty"`
      TriggerType string    `json:"trigger_type"`  // auto_80, auto_90, manual
      TokensUsed  int       `json:"tokens_used"`
      TokenBudget int       `json:"token_budget"`
      Summary     string    `json:"summary"`       // 현재 작업 요약
      ActiveFiles []string  `json:"active_files"`  // 수정 중인 파일
      KeyPoints   []string  `json:"key_points"`    // 핵심 포인트
      CreatedAt   time.Time `json:"created_at"`
  }
  
  type Store struct {
      db *db.DB
  }
  
  func (s *Store) Create(cp *Checkpoint) error
  func (s *Store) GetLatest(sessionID string) (*Checkpoint, error)
  func (s *Store) List(sessionID string, limit int) ([]*Checkpoint, error)
  ```

### 2. 토큰 모니터링 (PreToolUse에서)

- [ ] `internal/checkpoint/monitor.go`
  ```go
  type Monitor struct {
      store     *Store
      attention *attention.Store
  }
  
  // PreToolUse Hook에서 호출
  func (m *Monitor) CheckAndCreate(sessionID string) (*CheckResult, error) {
      state := m.attention.GetState(sessionID)
      usage := float64(state.TokensUsed) / float64(state.TokenBudget)
      
      result := &CheckResult{Usage: usage}
      
      // 80% 체크포인트
      if usage >= 0.8 && usage < 0.9 {
          if !m.hasRecentCheckpoint(sessionID, "auto_80", 5*time.Minute) {
              cp := m.createAuto(sessionID, "auto_80")
              result.Checkpoint = cp
              result.Message = fmt.Sprintf("체크포인트 생성됨 (토큰 %.0f%%)", usage*100)
          }
      }
      
      // 90% 체크포인트 + 경고
      if usage >= 0.9 {
          if !m.hasRecentCheckpoint(sessionID, "auto_90", 5*time.Minute) {
              cp := m.createAuto(sessionID, "auto_90")
              result.Checkpoint = cp
              result.Warning = true
              result.Message = fmt.Sprintf("⚠️ 토큰 %.0f%% - 작업 마무리 권장", usage*100)
          }
      }
      
      return result, nil
  }
  
  func (m *Monitor) createAuto(sessionID, triggerType string) *Checkpoint {
      // 현재 컨텍스트에서 요약 생성
      summary := m.generateSummary(sessionID)
      activeFiles := m.getActiveFiles(sessionID)
      keyPoints := m.extractKeyPoints(sessionID)
      
      cp := &Checkpoint{
          ID:          generateID(),
          SessionID:   sessionID,
          PortID:      getCurrentPortID(sessionID),
          TriggerType: triggerType,
          Summary:     summary,
          ActiveFiles: activeFiles,
          KeyPoints:   keyPoints,
      }
      
      m.store.Create(cp)
      return cp
  }
  ```

### 3. Hook 연동

- [ ] `internal/cli/hook.go` 수정
  ```go
  func runHookPreToolUse(cmd *cobra.Command, args []string) error {
      input, _ := readHookInput()
      
      // 체크포인트 모니터 실행
      monitor := checkpoint.NewMonitor(database)
      result, _ := monitor.CheckAndCreate(input.SessionID)
      
      if result.Checkpoint != nil {
          // Claude에 알림
          output := HookOutput{
              Continue: true,
              HookOutput: map[string]interface{}{
                  "checkpoint": result.Checkpoint.ID,
                  "message":    result.Message,
                  "warning":    result.Warning,
              },
          }
          json.NewEncoder(os.Stdout).Encode(output)
      }
      
      // 기존 로직 (활성 포트 확인 등)
      // ...
  }
  ```

### 4. DB 스키마

- [ ] v11 마이그레이션
  ```sql
  CREATE TABLE IF NOT EXISTS checkpoints (
      id TEXT PRIMARY KEY,
      session_id TEXT NOT NULL,
      port_id TEXT,
      trigger_type TEXT NOT NULL,
      tokens_used INTEGER,
      token_budget INTEGER,
      summary TEXT,
      active_files TEXT,  -- JSON
      key_points TEXT,    -- JSON
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );
  
  CREATE INDEX idx_checkpoints_session ON checkpoints(session_id, created_at DESC);
  CREATE INDEX idx_checkpoints_trigger ON checkpoints(trigger_type);
  ```

---

## Claude가 수신하는 정보

**80% 도달 시:**
```json
{
  "continue": true,
  "hookSpecificOutput": {
    "checkpoint": "cp-abc123",
    "message": "체크포인트 생성됨 (토큰 82%)",
    "warning": false
  }
}
```

**90% 도달 시:**
```json
{
  "continue": true,
  "hookSpecificOutput": {
    "checkpoint": "cp-def456",
    "message": "⚠️ 토큰 91% - 작업 마무리 권장",
    "warning": true
  }
}
```

---

## 완료 기준

- [ ] PreToolUse에서 토큰 모니터링 동작
- [ ] 80%/90% 도달 시 자동 체크포인트 생성
- [ ] Claude에 JSON 형식으로 알림
- [ ] 중복 체크포인트 방지 (5분 이내)

---

<!-- pal:port:L1-auto-checkpoint -->
