# PAL Kit v1.0

> Personal Agentic Layer - Claude Code 에이전트 오케스트레이션 도구

## 현재 버전

**v1.0-redesign** (개발 중)

## 아키텍처

```
User (Claude Desktop) ─── Spec Agent (MCP)
         │
         ▼
    Build Session ──── HTTP API ──── Electron GUI
         │
    ┌────┴────┐
    ▼         ▼
  Operator   Operator
    │           │
  Workers    Workers
```

## 세션 계층

| Type | 역할 | Depth |
|------|------|-------|
| **build** | 명세 설계, 포트 분해 | 0 |
| **operator** | 워커 관리, 진행 조율 | 1 |
| **worker** | 코드 구현 | 2 |
| **test** | 테스트 작성/실행 | 3 |

## 핵심 패키지 (v1.0)

### 신규 패키지

- `internal/analytics/` - DuckDB 연동 (문서 색인, 통계)
- `internal/message/` - 세션 간 메시지 패싱
- `internal/agentv2/` - 에이전트 버전 관리
- `internal/attention/` - Attention 추적, Compact 관리

### 확장된 패키지

- `internal/db/` - 스키마 v10 (세션 계층, 에이전트, 메시지)
- `internal/session/` - 계층적 세션 지원 (hierarchy.go)

## Storage

```
SQLite (OLTP)           DuckDB (OLAP)
├── sessions            ├── docs-index.json
├── messages            ├── conventions.json
├── agents              └── token-history.parquet
├── agent_versions
├── compact_events
└── port_handoffs
```

## 주요 명령어

```bash
# 초기화
pal init

# 세션 관리
pal session start --type build --title "명세 설계"
pal session list --type operator
pal session hierarchy <root-id>

# 포트 관리
pal port create --type atomic --title "UserEntity"
pal port list --type orchestration
pal port analyze <port-id>  # 토큰 분석

# 에이전트 관리
pal agent list
pal agent version <agent-id>
pal agent compare <agent-id> --v1 1 --v2 2

# 상태 확인
pal status
pal attention <session-id>
```

## 개발 가이드

### 세션 생성 (계층적)

```go
import "github.com/n0roo/pal-kit/internal/session"

svc := session.NewService(db)

// Build 세션 생성
build, _ := svc.StartHierarchical(session.HierarchyStartOptions{
    Title: "user-service 명세 설계",
    Type:  session.TypeBuild,
    TokenBudget: 50000,
})

// Operator 세션 생성
operator, _ := svc.StartHierarchical(session.HierarchyStartOptions{
    Title:    "user-entity-group",
    Type:     session.TypeOperator,
    ParentID: build.Session.ID,
})

// Worker 세션 생성
worker, _ := svc.StartHierarchical(session.HierarchyStartOptions{
    Title:    "UserEntity 구현",
    Type:     session.TypeWorker,
    ParentID: operator.Session.ID,
    PortID:   "port-001",
    AgentID:  "impl-worker",
})
```

### 메시지 전송

```go
import "github.com/n0roo/pal-kit/internal/message"

store := message.NewStore(db.DB)

// 작업 할당
store.SendTaskAssign(operatorID, workerID, portID, message.TaskAssignPayload{
    PortID:   "port-001",
    PortSpec: portContent,
})

// 구현 완료 알림
store.SendImplReady(workerID, testWorkerID, portID, message.ImplReadyPayload{
    Files:       []string{"user_entity.go"},
    BuildStatus: "success",
})
```

### Attention 추적

```go
import "github.com/n0roo/pal-kit/internal/attention"

store := attention.NewStore(db.DB)

// 초기화
store.Initialize(sessionID, portID, 15000)

// 토큰 업데이트
store.UpdateTokens(sessionID, 12000)

// Compact 기록
store.RecordCompact(&attention.CompactEvent{
    SessionID:     sessionID,
    TriggerReason: "token_limit",
    BeforeTokens:  45000,
    AfterTokens:   12000,
    PreservedContext: []string{"current_task", "decisions"},
})

// 리포트 생성
report, _ := store.GenerateReport(sessionID)
```

## Phase 1 완료 항목

- [x] DB 스키마 v10
- [x] DuckDB analytics 패키지
- [x] 메시지 패싱 패키지
- [x] 에이전트 버전 관리 패키지
- [x] Attention 추적 패키지
- [x] 세션 계층 확장

## Phase 2 예정

- [ ] Worker Pair 스폰/관리
- [ ] 포트 의존성 기반 실행
- [ ] Handoff 프로토콜
- [ ] Escalation 처리

## 관련 문서

- 설계: `mcp-docs/10-Personal/Projects/pal-kit/specs/`
- 기존 문서: `docs/`
