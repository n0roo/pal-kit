# Phase 3: 에이전트 워크플로우 개선 명세

> Port ID: agent-workflow-enhancement
> 상태: draft
> 우선순위: medium
> 의존성: hook-enhancement, context-management

---

## 개요

PAL Kit의 다중 에이전트 패턴을 개선하여 더 효율적인 작업 분배와 협업을 구현합니다.

---

## 현재 상태 분석

### 세션 계층 구조

```
internal/session/hierarchy.go

Build Session (depth=0)
    │
    ├─ Operator Session (depth=1)
    │       │
    │       ├─ Worker Session (depth=2)
    │       │       └─ Impl Worker
    │       │
    │       └─ Worker Session (depth=2)
    │               └─ Test Worker
    │
    └─ Operator Session (depth=1)
            └─ ...
```

### Orchestration 흐름

```go
// internal/orchestrator/orchestrator.go

type Orchestrator struct {
    sessionSvc  *session.Service
    portSvc     *port.Service
    handoffSvc  *handoff.Service
}

func (o *Orchestrator) Execute(ctx context.Context, spec OrchSpec) error {
    // 1. Build 세션 생성
    // 2. 포트별 Operator 세션 스폰
    // 3. Worker Pair (Impl + Test) 스폰
    // 4. 의존성 순서대로 실행
}
```

### 현재 Worker Pair 구조

```
Operator
    │
    ├─ Impl Worker  ─────────────┐
    │                            │
    └─ Test Worker  ◄────────────┘
          (검증)           (구현 결과)
```

### 식별된 문제점

1. **Worker 간 통신 제한적**: 직접 메시지 교환 불가, Operator 경유 필요
2. **에스컬레이션 처리 불완전**: Worker→Operator 에스컬레이션만 지원
3. **병렬 실행 제한**: 의존성 없는 포트도 순차 실행
4. **피드백 루프 부재**: Test Worker 실패 시 Impl Worker에 직접 피드백 없음

---

## 개선 사항

### 3.1 Worker 간 직접 통신

**현재**:
```
Impl Worker ─(escalation)─▶ Operator ─(message)─▶ Test Worker
```

**개선**:
```
Impl Worker ◄────(direct channel)────▶ Test Worker
      │                                      │
      └───────────(escalation)───────────────┘
                       │
                       ▼
                   Operator
```

**구현**:
```go
// internal/message/direct.go (신규)

type DirectChannel struct {
    ID        string
    SessionA  string  // Impl Worker
    SessionB  string  // Test Worker
    CreatedAt time.Time
}

type DirectMessage struct {
    ID        string
    ChannelID string
    From      string
    To        string
    Type      string    // result, feedback, query
    Payload   any
    CreatedAt time.Time
}

// 채널 생성
func (s *Service) CreateDirectChannel(sessionA, sessionB string) (*DirectChannel, error)

// 직접 메시지 전송
func (s *Service) SendDirect(channelID, from, to string, msg DirectMessage) error

// 메시지 수신 (polling)
func (s *Service) ReceiveDirect(channelID, recipient string) ([]DirectMessage, error)
```

**데이터베이스 스키마**:
```sql
CREATE TABLE direct_channels (
    id TEXT PRIMARY KEY,
    session_a TEXT NOT NULL,
    session_b TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_a) REFERENCES sessions(id),
    FOREIGN KEY (session_b) REFERENCES sessions(id)
);

CREATE TABLE direct_messages (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    from_session TEXT NOT NULL,
    to_session TEXT NOT NULL,
    message_type TEXT NOT NULL,
    payload TEXT,
    delivered_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (channel_id) REFERENCES direct_channels(id)
);
```

**변경 파일**:
- `internal/db/schema.go`: 테이블 추가
- `internal/message/direct.go`: 직접 통신 서비스
- `internal/orchestrator/worker.go`: Worker Pair 생성 시 채널 설정

---

### 3.2 피드백 루프 구현

**Test → Impl 피드백 흐름**:
```
┌──────────────┐     ┌──────────────┐
│ Impl Worker  │     │ Test Worker  │
│              │     │              │
│  구현 완료   │─────▶│  테스트 실행  │
│              │     │              │
│  수정 적용   │◀─────│  실패 피드백  │
│              │     │              │
│  재구현      │─────▶│  재검증      │
│              │     │              │
└──────────────┘     └──────────────┘
        │                   │
        └─────── 성공 ──────┘
                  │
                  ▼
              Operator
            (완료 보고)
```

**구현**:
```go
// internal/orchestrator/feedback.go (신규)

type FeedbackLoop struct {
    ID          string
    ImplSession string
    TestSession string
    ChannelID   string
    MaxRetries  int
    CurrentTry  int
    Status      string  // running, success, failed, escalated
}

type TestFeedback struct {
    Success     bool
    FailedTests []FailedTest
    Suggestions []string
    Metrics     TestMetrics
}

type FailedTest struct {
    Name        string
    Expected    string
    Actual      string
    StackTrace  string
    SuggestedFix string
}

func (o *Orchestrator) RunWithFeedback(ctx context.Context, implSession, testSession string) error {
    loop := &FeedbackLoop{
        ID:          generateID(),
        ImplSession: implSession,
        TestSession: testSession,
        MaxRetries:  3,
    }

    for loop.CurrentTry < loop.MaxRetries {
        // 1. Impl 작업 대기
        implResult := o.waitForImpl(ctx, implSession)

        // 2. Test 실행
        testResult := o.runTests(ctx, testSession, implResult)

        if testResult.Success {
            loop.Status = "success"
            return nil
        }

        // 3. 피드백 전송
        feedback := o.buildFeedback(testResult)
        o.sendDirect(loop.ChannelID, testSession, implSession, feedback)

        loop.CurrentTry++
    }

    // 최대 재시도 초과 → 에스컬레이션
    loop.Status = "escalated"
    return o.escalateToOperator(loop)
}
```

**Hook 연동**:
```go
// hook.go에 추가

case "test-feedback":
    // Test Worker가 피드백 전송 시 호출
    return runHookTestFeedback(input)

func runHookTestFeedback(input HookInput) HookOutput {
    feedback := input.Payload.(TestFeedback)

    // 이벤트 로깅
    sessionSvc.LogEvent(input.SessionID, "test_feedback", feedback)

    // 피드백 메시지 포맷팅
    var message strings.Builder
    if feedback.Success {
        message.WriteString("✅ 모든 테스트 통과\n")
    } else {
        message.WriteString("❌ 테스트 실패\n\n")
        for _, ft := range feedback.FailedTests {
            message.WriteString(fmt.Sprintf("- %s: %s\n", ft.Name, ft.SuggestedFix))
        }
    }

    return HookOutput{
        Decision: "approve",
        Notifications: []Notification{{
            Level:   "info",
            Title:   "Test Feedback",
            Message: message.String(),
        }},
    }
}
```

**변경 파일**:
- `internal/orchestrator/feedback.go`: 피드백 루프 로직
- `internal/cli/hook.go`: test-feedback hook 추가
- `internal/session/events.go`: test_feedback 이벤트 타입

---

### 3.3 병렬 실행 개선

**현재**:
```
Port A ────▶ Port B ────▶ Port C ────▶ Port D
  (순차 실행, 의존성 무관)
```

**개선**:
```
Port A ─────────────────────┐
                            │
Port B ─────┬───────────────┤
            │               │
Port C ◄────┘               ▼
            (A 의존)      Port D
                        (A,B,C 의존)
```

**구현**:
```go
// internal/orchestrator/executor.go (개선)

type ParallelExecutor struct {
    maxConcurrency int
    depGraph       *DependencyGraph
    semaphore      chan struct{}
}

type DependencyGraph struct {
    nodes map[string]*PortNode
    edges map[string][]string  // port -> depends_on
}

func (e *ParallelExecutor) Execute(ctx context.Context, ports []PortSpec) error {
    // 1. 의존성 그래프 구축
    graph := e.buildGraph(ports)

    // 2. 토폴로지 정렬로 실행 순서 결정
    levels := graph.TopologicalLevels()

    // 3. 레벨별 병렬 실행
    for _, level := range levels {
        var wg sync.WaitGroup
        errCh := make(chan error, len(level))

        for _, portID := range level {
            wg.Add(1)
            go func(pid string) {
                defer wg.Done()
                e.semaphore <- struct{}{} // 동시성 제한
                defer func() { <-e.semaphore }()

                if err := e.executePort(ctx, pid); err != nil {
                    errCh <- err
                }
            }(portID)
        }

        wg.Wait()
        close(errCh)

        // 에러 수집
        for err := range errCh {
            if err != nil {
                return err // 또는 에러 수집 후 계속
            }
        }
    }

    return nil
}

// 토폴로지 레벨 계산
func (g *DependencyGraph) TopologicalLevels() [][]string {
    // Kahn's algorithm으로 레벨별 노드 분류
    levels := [][]string{}
    inDegree := make(map[string]int)

    // 초기 in-degree 계산
    for node := range g.nodes {
        inDegree[node] = 0
    }
    for _, deps := range g.edges {
        for _, dep := range deps {
            inDegree[dep]++
        }
    }

    // 레벨별 처리
    for len(inDegree) > 0 {
        level := []string{}
        for node, degree := range inDegree {
            if degree == 0 {
                level = append(level, node)
            }
        }

        if len(level) == 0 {
            return nil // 순환 의존성
        }

        levels = append(levels, level)

        for _, node := range level {
            delete(inDegree, node)
            for _, dep := range g.edges[node] {
                inDegree[dep]--
            }
        }
    }

    return levels
}
```

**설정**:
```yaml
# .pal/config.yaml
orchestration:
  max_concurrency: 3      # 최대 병렬 Worker 수
  parallel_enabled: true  # 병렬 실행 활성화
  timeout_per_port: 30m   # 포트당 타임아웃
```

**변경 파일**:
- `internal/orchestrator/executor.go`: 병렬 실행 로직
- `internal/config/config.go`: orchestration 설정 추가
- `internal/orchestrator/graph.go`: 의존성 그래프 (신규)

---

### 3.4 에스컬레이션 체계 확장

**현재 에스컬레이션 타입**:
```go
const (
    EscalationTypeGeneral = "general"
    EscalationTypeBlocked = "blocked"
    EscalationTypeDecision = "decision"
)
```

**확장 에스컬레이션 타입**:
```go
// internal/escalation/types.go (확장)

const (
    // 기존
    EscalationTypeGeneral  = "general"
    EscalationTypeBlocked  = "blocked"
    EscalationTypeDecision = "decision"

    // 신규
    EscalationTypeTestFailure    = "test_failure"     // 테스트 반복 실패
    EscalationTypeTokenExhausted = "token_exhausted"  // 토큰 소진
    EscalationTypeTimeout        = "timeout"          // 타임아웃
    EscalationTypeConflict       = "conflict"         // Worker 간 충돌
    EscalationTypeDependency     = "dependency"       // 의존성 해결 불가
    EscalationTypeQuality        = "quality"          // 품질 기준 미달
)

// 심각도
const (
    SeverityLow      = "low"
    SeverityMedium   = "medium"
    SeverityHigh     = "high"
    SeverityCritical = "critical"
)

type Escalation struct {
    ID          string
    SessionID   string
    PortID      string
    Type        string
    Severity    string
    Title       string
    Description string
    Context     map[string]any
    SuggestedActions []string
    ResolvedBy  string
    ResolvedAt  *time.Time
    CreatedAt   time.Time
}
```

**자동 에스컬레이션 트리거**:
```go
// internal/orchestrator/escalation_trigger.go (신규)

type EscalationTrigger struct {
    Type       string
    Condition  func(ctx WorkerContext) bool
    Severity   string
    AutoResolve bool
}

var defaultTriggers = []EscalationTrigger{
    {
        Type: EscalationTypeTestFailure,
        Condition: func(ctx WorkerContext) bool {
            return ctx.TestRetries >= ctx.MaxRetries
        },
        Severity: SeverityHigh,
    },
    {
        Type: EscalationTypeTokenExhausted,
        Condition: func(ctx WorkerContext) bool {
            return ctx.TokensUsed >= ctx.TokenBudget * 0.95
        },
        Severity: SeverityMedium,
    },
    {
        Type: EscalationTypeTimeout,
        Condition: func(ctx WorkerContext) bool {
            return time.Since(ctx.StartTime) > ctx.Timeout
        },
        Severity: SeverityHigh,
    },
}

func (o *Orchestrator) CheckEscalationTriggers(ctx WorkerContext) *Escalation {
    for _, trigger := range defaultTriggers {
        if trigger.Condition(ctx) {
            return &Escalation{
                Type:     trigger.Type,
                Severity: trigger.Severity,
                // ...
            }
        }
    }
    return nil
}
```

**변경 파일**:
- `internal/escalation/types.go`: 타입 확장
- `internal/escalation/service.go`: 트리거 로직
- `internal/db/schema.go`: escalations 테이블 확장

---

### 3.5 세션 상태 동기화

**SSE 이벤트 확장**:
```go
// internal/server/websocket.go (확장)

const (
    // 기존 이벤트
    EventSessionStart    = "session.start"
    EventSessionEnd      = "session.end"

    // Worker 이벤트
    EventWorkerSpawn     = "worker.spawn"
    EventWorkerComplete  = "worker.complete"
    EventWorkerProgress  = "worker.progress"  // 신규
    EventWorkerFeedback  = "worker.feedback"  // 신규

    // 채널 이벤트
    EventDirectMessage   = "direct.message"   // 신규

    // 에스컬레이션 이벤트
    EventEscalationNew      = "escalation.new"
    EventEscalationResolved = "escalation.resolved"  // 신규
)

type WorkerProgressEvent struct {
    SessionID   string  `json:"session_id"`
    PortID      string  `json:"port_id"`
    Progress    float64 `json:"progress"`      // 0.0 ~ 1.0
    CurrentTask string  `json:"current_task"`
    TokensUsed  int     `json:"tokens_used"`
}
```

**GUI 연동**:
```typescript
// electron-gui/src/hooks/useWorkerProgress.ts (신규)

export function useWorkerProgress(sessionId: string) {
  const [progress, setProgress] = useState<WorkerProgress>({
    progress: 0,
    currentTask: '',
    tokensUsed: 0,
  })

  useSSE(`worker.progress.${sessionId}`, (event) => {
    setProgress(event.data)
  })

  return progress
}
```

---

## 구현 순서

```
3.1 Worker 간 직접 통신  (기반)
  ↓
3.2 피드백 루프 구현
  ↓
3.3 병렬 실행 개선
  ↓
3.4 에스컬레이션 확장
  ↓
3.5 세션 상태 동기화
```

---

## 테스트 계획

### 단위 테스트

```go
// internal/orchestrator/executor_test.go

func TestParallelExecution(t *testing.T) {
    // 의존성 없는 포트들이 병렬 실행되는지 확인
}

func TestDependencyOrder(t *testing.T) {
    // 의존성 순서가 지켜지는지 확인
}

func TestFeedbackLoop(t *testing.T) {
    // 피드백 루프가 정상 동작하는지 확인
    // 최대 재시도 후 에스컬레이션 발생하는지 확인
}

func TestDirectChannel(t *testing.T) {
    // Worker 간 직접 통신 확인
}
```

### 통합 테스트

```bash
# test/integration/workflow_test.sh

# 1. 병렬 실행 테스트
./test_parallel_execution.sh

# 2. 피드백 루프 테스트
./test_feedback_loop.sh

# 3. 에스컬레이션 테스트
./test_escalation_triggers.sh
```

---

## 완료 기준

- [ ] Worker 간 직접 채널로 메시지 교환 가능
- [ ] Test→Impl 피드백 루프 동작 (최대 3회 재시도)
- [ ] 의존성 없는 포트 병렬 실행
- [ ] 확장된 에스컬레이션 타입 7개 추가
- [ ] GUI에서 Worker 진행률 실시간 표시
- [ ] 모든 테스트 통과

---

## 관련 문서

- [ROADMAP-CLAUDE-INTEGRATION.md](../ROADMAP-CLAUDE-INTEGRATION.md)
- [internal/orchestrator/](../../internal/orchestrator/)
- [internal/session/hierarchy.go](../../internal/session/hierarchy.go)
