# Port: hook-auto-checkpoint

> Hook 자동 체크포인트 - pre-tool-use에서 80% 토큰 도달 시 자동 생성

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | hook-auto-checkpoint |
| 타입 | atomic |
| 레이어 | L1 (Hook) |
| 상태 | pending |
| 우선순위 | high |
| 의존성 | - |
| 예상 토큰 | 8,000 |

---

## 설계 원칙

**사용자는 체크포인트를 의식하지 않는다**

```
[Claude Code Hook: pre-tool-use]
    ↓
[PAL Kit: 토큰 사용량 체크]
    ↓ 80% 초과?
[자동 체크포인트 생성]
    ↓
[Claude에 stderr로 알림] → Claude가 인지하고 작업 계속
```

---

## 범위

### 포함

- `pre-tool-use` Hook에서 토큰 체크 로직
- 80% 도달 시 자동 체크포인트 생성
- Claude에 stderr 알림 (Claude가 읽음)
- 90% 도달 시 경고 강화

### 제외

- 체크포인트 복구 (MCP 도구로 분리)
- GUI 알림 (SSE 포트에서 처리)

---

## 작업 항목

### 1. 체크포인트 저장소

- [ ] `internal/checkpoint/store.go` 생성
  ```go
  type Checkpoint struct {
      ID            string    `json:"id"`
      SessionID     string    `json:"session_id"`
      PortID        string    `json:"port_id,omitempty"`
      TokensUsed    int       `json:"tokens_used"`
      TokenBudget   int       `json:"token_budget"`
      TriggerType   string    `json:"trigger_type"` // auto_80, auto_90, pre_heavy
      ContextHash   string    `json:"context_hash"`
      Summary       string    `json:"summary"`      // 현재 작업 요약
      ActiveFiles   []string  `json:"active_files"` // 작업 중인 파일
      CreatedAt     time.Time `json:"created_at"`
  }
  
  type Store struct {
      db *db.DB
  }
  
  func (s *Store) Create(cp *Checkpoint) error
  func (s *Store) Get(id string) (*Checkpoint, error)
  func (s *Store) ListBySession(sessionID string) ([]*Checkpoint, error)
  func (s *Store) GetLatest(sessionID string) (*Checkpoint, error)
  ```

### 2. pre-tool-use Hook 수정

- [ ] `internal/cli/hook.go` - `runHookPreToolUse` 수정
  ```go
  func runHookPreToolUse(cmd *cobra.Command, args []string) error {
      input, err := readHookInput()
      // ...
      
      // ★ 토큰 사용량 체크 (새로운 로직)
      attentionSvc := attention.NewService(database)
      state, err := attentionSvc.GetSessionState(palSessionID)
      if err == nil && state != nil {
          usage := float64(state.TokensUsed) / float64(state.TokenBudget)
          
          // 80% 도달: 자동 체크포인트
          if usage >= 0.8 && usage < 0.9 {
              cp, err := checkpointSvc.CreateAuto(palSessionID, "auto_80")
              if err == nil {
                  // Claude에 알림 (stderr로 출력 - Claude가 읽음)
                  fmt.Fprintf(os.Stderr, "\n")
                  fmt.Fprintf(os.Stderr, "💾 [PAL Kit] 자동 체크포인트 생성됨 (토큰 사용량 %.0f%%)\n", usage*100)
                  fmt.Fprintf(os.Stderr, "   체크포인트: %s\n", cp.ID)
                  fmt.Fprintf(os.Stderr, "   Compact 발생 시 이 시점으로 복구 가능\n")
                  fmt.Fprintf(os.Stderr, "\n")
              }
          }
          
          // 90% 도달: 강한 경고
          if usage >= 0.9 {
              fmt.Fprintf(os.Stderr, "\n")
              fmt.Fprintf(os.Stderr, "⚠️  [PAL Kit] 토큰 사용량 위험 수준 (%.0f%%)\n", usage*100)
              fmt.Fprintf(os.Stderr, "   💡 현재 작업을 마무리하고 새 포트로 분리하는 것을 권장합니다.\n")
              fmt.Fprintf(os.Stderr, "   📋 pal_checkpoint_list로 체크포인트 확인 가능\n")
              fmt.Fprintf(os.Stderr, "\n")
              
              // 90% 체크포인트도 생성
              checkpointSvc.CreateAuto(palSessionID, "auto_90")
          }
      }
      
      // 기존 로직 (Edit/Write 도구 확인 등)
      // ...
  }
  ```

### 3. DB 스키마

- [ ] `checkpoints` 테이블 (v11 마이그레이션)
  ```sql
  CREATE TABLE IF NOT EXISTS checkpoints (
      id TEXT PRIMARY KEY,
      session_id TEXT NOT NULL,
      port_id TEXT,
      tokens_used INTEGER,
      token_budget INTEGER,
      trigger_type TEXT NOT NULL,  -- auto_80, auto_90, pre_heavy, manual
      context_hash TEXT,
      summary TEXT,
      active_files TEXT,           -- JSON array
      created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (session_id) REFERENCES sessions(id)
  );
  
  CREATE INDEX idx_checkpoints_session ON checkpoints(session_id, created_at DESC);
  ```

### 4. Attention 서비스 연동

- [ ] `internal/attention/attention.go` 확장
  ```go
  // 토큰 예산 업데이트 (세션 시작 시 설정)
  func (s *Store) SetTokenBudget(sessionID string, budget int) error
  
  // 현재 토큰 사용량 업데이트 (transcript 파싱 결과)
  func (s *Store) UpdateTokensUsed(sessionID string, tokens int) error
  
  // 사용률 계산
  func (s *Store) GetUsageRatio(sessionID string) (float64, error)
  ```

### 5. Claude 알림 형식

```
💾 [PAL Kit] 자동 체크포인트 생성됨 (토큰 사용량 82%)
   체크포인트: cp-abc123
   Compact 발생 시 이 시점으로 복구 가능

⚠️  [PAL Kit] 토큰 사용량 위험 수준 (91%)
   💡 현재 작업을 마무리하고 새 포트로 분리하는 것을 권장합니다.
   📋 pal_checkpoint_list로 체크포인트 확인 가능
```

---

## 테스트 시나리오

**자동 테스트 (사용자 개입 없음)**

1. Claude Code에서 작업 시작
2. 여러 파일 수정으로 토큰 사용량 증가
3. 80% 도달 시:
   - PAL이 자동으로 체크포인트 생성
   - Claude에 알림 표시
   - 사용자는 아무것도 할 필요 없음
4. Claude가 작업 계속

---

## 참조

- `internal/cli/hook.go` - 현재 Hook 구현
- `internal/attention/attention.go` - Attention 추적
- `specs/SESSION-AGENT-DESIGN.md` - Compact 관리 설계

---

<!-- pal:port:hook-auto-checkpoint -->
