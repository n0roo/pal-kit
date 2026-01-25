# Phase 1: Hook 시스템 개선 명세

> Port ID: hook-enhancement
> 상태: draft
> 우선순위: high
> 의존성: -

---

## 개요

Claude Code의 Hook 시스템과 PAL Kit의 통합을 강화하여, 더 정확한 컨텍스트 전달과 작업 추적을 달성합니다.

---

## 현재 상태 분석

### 구현된 Hook (12개)

```
internal/cli/hook.go (1,900+ lines)

├─ session-start    # 세션 시작, 컨텍스트 주입
├─ session-end      # 세션 종료, usage 수집
├─ pre-tool-use     # 도구 사용 전 검증
├─ post-tool-use    # 도구 사용 후 처리
├─ stop             # 중지 처리
├─ pre-compact      # Compact 전 체크포인트
├─ notification     # 알림 (Compact 복구)
├─ subagent         # 서브에이전트 연결
├─ port-start       # 포트 작업 시작
├─ port-end         # 포트 작업 종료
├─ sync             # 상태 동기화
└─ event            # 이벤트 로깅
```

### 현재 통신 구조

```
Claude Code
    │
    ▼ (stdin: JSON)
┌─────────────────┐
│   PAL Hook      │
│                 │
│ HookInput {     │
│   session_id    │
│   hook_type     │
│   tool_name     │
│   tool_input    │
│   cwd           │
│   ...           │
│ }               │
└────────┬────────┘
         │
         ▼ (stdout: JSON)
┌─────────────────┐
│ HookOutput {    │
│   decision      │ ← approve/block/allow/deny
│   reason        │
│   hookSpecific  │
│ }               │
└─────────────────┘
         │
         ▼ (stderr: text)
    Claude가 읽는
    경고/안내 메시지
```

### 식별된 문제점

1. **응답 구조 제한적**: 추가 컨텍스트 전달 어려움
2. **세션 식별 불완전**: cwd만으로 복수 터미널 구분 안 됨
3. **포트 추적 선택적**: 경고만 하고 강제하지 않음
4. **이벤트 타입 제한적**: 필요한 이벤트 부족

---

## 개선 사항

### 1.1 Hook 응답 구조 확장

**현재**:
```go
type HookOutput struct {
    Decision string `json:"decision"`
    Reason   string `json:"reason,omitempty"`
    // hook별 개별 필드들...
}
```

**개선**:
```go
type HookOutput struct {
    // 기본 필드 (호환성)
    Decision string `json:"decision"`
    Reason   string `json:"reason,omitempty"`

    // 확장 필드
    Context       *ContextInfo    `json:"context,omitempty"`
    Notifications []Notification  `json:"notifications,omitempty"`
    Suggestions   []string        `json:"suggestions,omitempty"`
    Metadata      map[string]any  `json:"metadata,omitempty"`
}

type ContextInfo struct {
    ActivePort     *PortSummary `json:"active_port,omitempty"`
    LoadedDocs     []DocRef     `json:"loaded_docs,omitempty"`
    SessionState   string       `json:"session_state"`
    TokensUsed     int          `json:"tokens_used"`
    TokenBudget    int          `json:"token_budget"`
}

type Notification struct {
    Level   string `json:"level"`   // info/warn/error
    Title   string `json:"title"`
    Message string `json:"message"`
    Action  string `json:"action,omitempty"` // 제안 액션
}
```

**변경 파일**:
- `internal/cli/hook.go`: HookOutput 구조체 확장
- `internal/cli/hook_*.go`: 각 hook 응답 개선

**검증 방법**:
```bash
# Hook 응답 테스트
echo '{"hook_type":"session-start","cwd":"/test"}' | pal hook | jq '.context'
```

---

### 1.2 세션 식별 강화

**현재 식별 방식**:
```go
// FindActiveSession - cwd + project_root로 검색
func (s *Service) FindActiveSession(projectRoot, cwd string) (*Session, error)
```

**개선된 식별 방식**:
```go
type SessionIdentifier struct {
    CWD         string    `json:"cwd"`
    TTY         string    `json:"tty"`          // 터미널 식별자
    ParentPID   int       `json:"parent_pid"`   // 부모 프로세스 ID
    StartTime   time.Time `json:"start_time"`
    Fingerprint string    `json:"fingerprint"`  // 복합 해시
}

// TTY 수집 방법
func getTTY() string {
    // 1. 환경변수: $SSH_TTY, $GPG_TTY
    // 2. /dev/tty 확인
    // 3. os.Stdin이 터미널인지 확인
}

// Fingerprint 생성
func (si *SessionIdentifier) GenerateFingerprint() string {
    data := fmt.Sprintf("%s:%s:%d:%d",
        si.CWD, si.TTY, si.ParentPID, si.StartTime.Unix())
    return sha256Hash(data)[:16]
}
```

**데이터베이스 스키마 변경**:
```sql
ALTER TABLE sessions ADD COLUMN tty TEXT;
ALTER TABLE sessions ADD COLUMN parent_pid INTEGER;
ALTER TABLE sessions ADD COLUMN fingerprint TEXT;

CREATE INDEX idx_sessions_fingerprint ON sessions(fingerprint);
```

**변경 파일**:
- `internal/session/session.go`: SessionIdentifier 추가
- `internal/db/schema.go`: 스키마 업데이트
- `internal/cli/hook.go`: session-start에서 fingerprint 사용

**검증 방법**:
```bash
# 터미널 1
pal hook session-start < input1.json  # fingerprint: abc123

# 터미널 2 (같은 디렉토리)
pal hook session-start < input2.json  # fingerprint: def456 (다름)
```

---

### 1.3 포트 추적 강제화

**현재 동작**:
```go
// pre-tool-use hook
if toolName == "Edit" || toolName == "Write" {
    if len(runningPorts) == 0 {
        // stderr로 경고만
        fmt.Fprintln(os.Stderr, "⚠️ 활성 포트가 없습니다...")
    }
}
```

**개선된 동작**:
```go
// 설정 기반 모드
type PortTrackingMode string

const (
    TrackingModeStrict PortTrackingMode = "strict" // 포트 없으면 block
    TrackingModeWarn   PortTrackingMode = "warn"   // 경고만
    TrackingModeOff    PortTrackingMode = "off"    // 추적 안 함
)

// config.yaml
// tracking:
//   mode: strict  # strict | warn | off
//   auto_create: true  # 자동 포트 생성 제안

func runHookPreToolUse(input HookInput) HookOutput {
    if isEditOrWrite(input.ToolName) {
        runningPorts := portSvc.ListRunning()

        if len(runningPorts) == 0 {
            switch config.Tracking.Mode {
            case TrackingModeStrict:
                return HookOutput{
                    Decision: "block",
                    Reason:   "활성 포트가 없습니다",
                    Suggestions: []string{
                        "pal port create <id> --title \"작업명\"",
                        "pal hook port-start <existing-port-id>",
                    },
                }
            case TrackingModeWarn:
                // 기존 동작 + 이벤트 로깅
                logEvent("untracked_edit", input)
                return HookOutput{Decision: "approve"}
            }
        }
    }
}
```

**자동 포트 생성 제안**:
```go
// auto_create: true 일 때
if config.Tracking.AutoCreate {
    suggestedID := generatePortID(input.ToolInput)
    suggestions = append(suggestions,
        fmt.Sprintf("pal port create %s --title \"%s\" && pal hook port-start %s",
            suggestedID, inferTitle(input), suggestedID))
}
```

**변경 파일**:
- `internal/config/config.go`: TrackingMode 설정 추가
- `internal/cli/hook.go`: pre-tool-use 로직 개선
- `.pal/config.yaml` 템플릿 업데이트

**검증 방법**:
```bash
# strict 모드
echo '{"hook_type":"pre-tool-use","tool_name":"Edit"}' | pal hook
# → {"decision":"block","suggestions":["pal port create..."]}

# warn 모드 (기존 동작)
# → {"decision":"approve"} + stderr 경고
```

---

### 1.4 Hook 이벤트 확장

**현재 이벤트 타입**:
```
session_start, session_end, port_start, port_end,
user_request, untracked_edit, escalation, decision,
file_edit, build_failed, test_failed, compact
```

**추가할 이벤트 타입**:

| 이벤트 | 트리거 | 용도 |
|--------|--------|------|
| `context_loaded` | 컨텍스트 로드 완료 | 로드된 문서 추적 |
| `context_overflow` | 토큰 예산 초과 | 컨텍스트 최적화 |
| `agent_activated` | 에이전트 활성화 | 워크플로우 추적 |
| `agent_deactivated` | 에이전트 비활성화 | 워크플로우 추적 |
| `dependency_resolved` | 의존성 해결 | 포트 의존성 추적 |
| `quality_warning` | 코드 품질 문제 | 자동 리뷰 |
| `checkpoint_created` | 체크포인트 생성 | Compact 복구 |

**이벤트 스키마**:
```go
type SessionEvent struct {
    ID        string         `json:"id"`
    SessionID string         `json:"session_id"`
    PortID    string         `json:"port_id,omitempty"`
    EventType string         `json:"event_type"`
    Summary   string         `json:"summary"`
    Details   map[string]any `json:"details,omitempty"`
    Severity  string         `json:"severity"` // info/warn/error
    CreatedAt time.Time      `json:"created_at"`
}
```

**SSE 연동**:
```go
// 이벤트 발생 시 SSE 발행
func (s *Service) LogEvent(event SessionEvent) error {
    // DB 저장
    if err := s.store.SaveEvent(event); err != nil {
        return err
    }

    // SSE 발행 (GUI 실시간 업데이트)
    s.sseHub.Broadcast(SSEEvent{
        Type: "session.event",
        Data: event,
    })

    return nil
}
```

**변경 파일**:
- `internal/session/events.go`: 이벤트 타입 정의
- `internal/session/service.go`: LogEvent 확장
- `internal/server/websocket.go`: SSE 이벤트 연동

---

## 구현 순서

```
1.2 세션 식별 강화 (기반 작업)
  ↓
1.1 Hook 응답 구조 확장
  ↓
1.3 포트 추적 강제화
  ↓
1.4 Hook 이벤트 확장
```

---

## 테스트 계획

### 단위 테스트

```go
// internal/cli/hook_test.go

func TestSessionIdentifier(t *testing.T) {
    // 같은 cwd, 다른 TTY → 다른 fingerprint
    // 같은 cwd, 같은 TTY → 같은 fingerprint
}

func TestPortTrackingStrict(t *testing.T) {
    // strict 모드에서 포트 없이 Edit → block
}

func TestHookOutputContext(t *testing.T) {
    // Context 필드가 올바르게 채워지는지 확인
}
```

### 통합 테스트

```bash
# test/integration/hook_test.sh

# 1. 세션 식별 테스트
./test_session_identification.sh

# 2. 포트 추적 테스트
./test_port_tracking.sh

# 3. 이벤트 발행 테스트
./test_event_emission.sh
```

---

## 완료 기준

- [ ] 세션 fingerprint로 복수 터미널 구분 100% 가능
- [ ] strict 모드에서 포트 없는 Edit/Write 100% 차단
- [ ] Hook 응답에 Context 정보 포함
- [ ] 새 이벤트 타입 7개 추가 및 SSE 연동
- [ ] 모든 테스트 통과

---

## 관련 문서

- [ROADMAP-CLAUDE-INTEGRATION.md](../ROADMAP-CLAUDE-INTEGRATION.md)
- [internal/cli/hook.go](../../internal/cli/hook.go)
- [internal/session/session.go](../../internal/session/session.go)
