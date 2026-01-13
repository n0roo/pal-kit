# Port: session-tracking

> 세션 및 포트 추적 시스템 개선

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | session-tracking |
| 상태 | running |
| 우선순위 | critical |
| 의존성 | - |
| 예상 복잡도 | high |

---

## 목표

세션과 포트의 상태 관리, 토큰/비용 추적을 정확하게 동작하도록 개선하고,
Operator 에이전트가 세션 연속성을 관리할 수 있는 기반을 구축한다.

---

## 현재 문제점

### 1. 세션 관련
- 세션 title이 비어있음
- 세션 status가 running으로 고착 (종료 감지 실패)
- 세션 tokens/cost가 항상 0 (수집 미구현)
- 세션 기록이 .pal/sessions/ 에 저장되지 않음

### 2. 포트 관련
- 포트에 session_id 연결 안됨
- 포트에 tokens/cost/duration 컬럼 없음
- 포트별 에이전트 정보 없음

### 3. Operator 연동
- 세션 시작 시 브리핑 생성 안됨
- 세션 종료 시 요약 저장 안됨
- ADR 자동 감지 없음

---

## 구현 가이드

### Phase 1: 세션 종료 로직 수정

#### 1.1 Claude session_id → PAL session 매핑 개선

**현재 코드** (`internal/cli/hook.go:345-441`)
```go
// session-end 시 Claude session ID로 PAL 세션 찾기
claudeSessionID := input.SessionID
palSession, _ := sessionSvc.FindByClaudeSessionID(claudeSessionID)
```

**문제**: Claude가 session_id를 전달 안 하는 경우 세션 종료 실패

**해결**:
```go
// 1. Claude session_id 시도
// 2. 실패 시 cwd + project_root 기반 최근 세션 찾기
// 3. 그래도 실패 시 가장 최근 running 세션 종료
func (s *Service) FindActiveSession(claudeSessionID, cwd, projectRoot string) (*Session, error)
```

#### 1.2 좀비 세션 정리 로직

**추가 필요**: `internal/session/session.go`
```go
// CleanupZombieSessions 이미 있음 (209-218줄)
// 호출 시점 추가 필요:
// - session-start 시 이전 세션 자동 정리
// - 주기적 정리 (cron 또는 CLI)
```

**구현 위치**: `internal/cli/hook.go:runHookSessionStart`
```go
// 세션 시작 전 좀비 세션 정리
cleaned, _ := sessionSvc.CleanupZombieSessions(24) // 24시간 이상
if cleaned > 0 && verbose {
    fmt.Printf("🧹 Cleaned %d zombie sessions\n", cleaned)
}
```

#### 1.3 세션 종료 시 상태 확실히 complete로 변경

**현재 코드** (`internal/session/session.go:166-186`)
- `EndWithReason` 이미 구현됨
- 문제는 호출되지 않는 것

**해결**: session-end 훅에서 확실히 호출
```go
// internal/cli/hook.go:runHookSessionEnd
// 현재 EndAllByClaudeSession 호출하고 있음 (418줄)
// 추가: 개별 세션 종료 이벤트 로깅
for _, sess := range sessions {
    sessionSvc.LogEvent(sess.ID, "session_end", ...)
}
```

---

### Phase 2: Usage 수집 구현

#### 2.1 JSONL 파싱

**현재 코드** (`internal/transcript/parser.go`)
- 이미 구현됨: `ParseFile(transcriptPath) → Usage`

**문제**: 호출 위치에서 에러 무시

**해결** (`internal/cli/hook.go:385-409`):
```go
// 현재 에러 시 warning만 출력
// 변경: 재시도 또는 부분 수집
usage, err := transcript.ParseFile(transcriptPath)
if err != nil {
    // 파일이 아직 쓰는 중일 수 있음 → 재시도
    time.Sleep(100 * time.Millisecond)
    usage, err = transcript.ParseFile(transcriptPath)
}
```

#### 2.2 세션 종료 시 Usage 업데이트

**현재 코드**: 이미 구현됨 (`hook.go:389-406`)

**문제**: transcript 경로가 없거나 잘못됨

**해결**:
```go
// transcript 경로 우선순위:
// 1. input.TranscriptPath (Claude에서 전달)
// 2. palSession.TranscriptPath (세션 시작 시 저장)
// 3. 기본 경로 추론 (~/.claude/projects/.../transcript.jsonl)
```

---

### Phase 3: 포트 스키마 확장

#### 3.1 ports 테이블 컬럼 추가

**DB 마이그레이션 v5**:
```sql
ALTER TABLE ports ADD COLUMN session_id TEXT REFERENCES sessions(id);
ALTER TABLE ports ADD COLUMN input_tokens INTEGER DEFAULT 0;
ALTER TABLE ports ADD COLUMN output_tokens INTEGER DEFAULT 0;
ALTER TABLE ports ADD COLUMN cost_usd REAL DEFAULT 0;
ALTER TABLE ports ADD COLUMN duration_secs INTEGER DEFAULT 0;
ALTER TABLE ports ADD COLUMN worker_id TEXT;
ALTER TABLE ports ADD COLUMN started_at DATETIME;
ALTER TABLE ports ADD COLUMN completed_at DATETIME;
```

#### 3.2 Port Service 확장

**internal/port/port.go 추가**:
```go
// AssignSession: 포트에 세션 연결
func (s *Service) AssignSession(portID, sessionID string) error

// UpdateUsage: 포트 사용량 업데이트
func (s *Service) UpdateUsage(portID string, input, output int64, cost float64) error

// UpdateDuration: 작업 시간 업데이트
func (s *Service) UpdateDuration(portID string, durationSecs int64) error
```

---

### Phase 4: 세션-포트 연계

#### 4.1 port-start hook에서 현재 세션 연결

**수정 위치**: `internal/cli/hook.go:runHookPortStart` (535-647)

```go
// 현재 (591-604줄):
if claudeSessionID != "" {
    palSession, err := sessionSvc.FindByClaudeSessionID(claudeSessionID)
    if err == nil && palSession != nil {
        portSvc.AssignSession(portID, palSession.ID)
        // ...
    }
}

// 추가:
// 포트 시작 시간 기록
portSvc.SetStartedAt(portID, time.Now())

// 워커 ID 기록 (이미 result.WorkerID 있음)
if result != nil {
    portSvc.SetWorkerID(portID, result.WorkerID)
}
```

#### 4.2 port-end hook에서 포트 통계 업데이트

**수정 위치**: `internal/cli/hook.go:runHookPortEnd` (649-742)

```go
// duration 계산 (현재 689-693줄에 있음)
var durationSecs int64
if p.StartedAt.Valid {
    durationSecs = int64(time.Since(p.StartedAt.Time).Seconds())
}

// 추가: 포트에 저장
portSvc.UpdateDuration(portID, durationSecs)

// 추가: 세션의 transcript에서 이 포트 관련 usage 추출
// (시작~종료 시간 범위의 usage)
```

---

### Phase 5: Operator 연동

#### 5.1 세션 시작 시 브리핑 생성

**추가 위치**: `internal/cli/hook.go:runHookSessionStart`

```go
// Operator 브리핑 생성
if projectRoot != "" {
    operatorSvc := operator.NewService(database, projectRoot)
    briefing, err := operatorSvc.GenerateBriefing()
    if err == nil {
        // .pal/context/session-briefing.md 저장
        operatorSvc.WriteBriefing(briefing)

        // stdout으로 출력 (Claude가 읽음)
        fmt.Println(briefing.Summary)
    }
}
```

#### 5.2 세션 종료 시 요약 저장

**추가 위치**: `internal/cli/hook.go:runHookSessionEnd`

```go
// Operator 세션 요약 생성
if projectRoot != "" && palSession != nil {
    operatorSvc := operator.NewService(database, projectRoot)
    summary, err := operatorSvc.GenerateSummary(palSession.ID)
    if err == nil {
        // .pal/sessions/{date}-{id}.md 저장
        operatorSvc.WriteSummary(summary)
    }
}
```

#### 5.3 ADR 자동 감지

**새 패키지**: `internal/operator/operator.go`

```go
type Service struct {
    db          *db.DB
    projectRoot string
}

// GenerateBriefing: 세션 시작 브리핑
func (s *Service) GenerateBriefing() (*Briefing, error)

// GenerateSummary: 세션 종료 요약
func (s *Service) GenerateSummary(sessionID string) (*Summary, error)

// DetectADR: 세션 이벤트에서 ADR 후보 감지
func (s *Service) DetectADR(sessionID string) ([]ADRCandidate, error)

// WriteBriefing: .pal/context/session-briefing.md
func (s *Service) WriteBriefing(b *Briefing) error

// WriteSummary: .pal/sessions/{date}-{id}.md
func (s *Service) WriteSummary(s *Summary) error
```

---

## 작업 항목 체크리스트

### P1: 세션 종료 로직 수정

- [x] `session.FindActiveSession` 구현 (다중 fallback)
- [x] session-start에서 좀비 세션 자동 정리 (24시간)
- [x] session-end에서 확실한 종료 처리 (EndWithReason)

### P1: Usage 수집 구현

- [x] transcript 경로 fallback 로직 (세션 저장 경로 사용)
- [x] 파싱 재시도 로직 (최대 3회, 100ms 간격)
- [ ] 부분 수집 지원

### P2: 포트 스키마 확장

- [x] DB 마이그레이션 v6 (기존 포함) - ports 테이블 컬럼 이미 존재
- [x] Port Service 확장 메서드
  - SetDuration, RecordStart, RecordCompletion
  - GetBySession, GetStats, GetRecentCompleted

### P2: 세션-포트 연계

- [x] port-start에서 시작 시간/워커 기록 (RecordStart)
- [x] port-end에서 duration/usage 기록 (RecordCompletion)

### P3: Operator 연동

- [x] `internal/operator/` 패키지 생성
- [x] 브리핑/요약 생성 로직
- [x] ADR 후보 감지 로직
- [x] hook.go 연동 (session-start, session-end)

---

## 완료 기준

- [x] 세션 종료 시 status가 complete로 변경
- [x] 세션에 토큰/비용 데이터 기록
- [x] 포트에 세션 연결 및 통계 기록
- [x] .pal/sessions/ 에 세션 기록 저장
- [x] 세션 시작 시 이전 작업 브리핑 출력

---

## 파일 변경 목록

| 파일 | 변경 내용 | 상태 |
|------|----------|------|
| `internal/db/db.go` | 스키마 v6에 ports 컬럼 이미 포함 | ✅ 완료 |
| `internal/session/session.go` | FindActiveSession, findByLocation, findMostRecentRunning | ✅ 완료 |
| `internal/port/port.go` | RecordStart, RecordCompletion, GetStats 등 | ✅ 완료 |
| `internal/cli/hook.go` | 브리핑/요약/좀비 정리/포트 연계 | ✅ 완료 |
| `internal/operator/operator.go` | 새 패키지 | ✅ 완료 |

---

<!-- pal:port:status=complete -->
