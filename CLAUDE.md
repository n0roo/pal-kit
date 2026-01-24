# PAL Kit v1.0

> Personal Agentic Layer - Claude Code 에이전트 오케스트레이션 도구

## 현재 버전

**v1.0-redesign** (feat/redesign-pakit-v1 브랜치)

## 아키텍처

```
User (Claude Desktop) ─── MCP Server ──── Spec Agent
         │                    │
         ▼                    ▼
    Build Session ──── HTTP API ──── Electron GUI
         │                    │
         │              SSE Events
    ┌────┴────┐              │
    ▼         ▼              ▼
  Operator   Operator    WebSocket
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

## 핵심 패키지

### Phase 1 (완료)

| 패키지 | 기능 |
|--------|------|
| `internal/db/` | 스키마 v10 (세션 계층, 에이전트, 메시지) |
| `internal/analytics/` | DuckDB 연동, 문서 색인, 통계 쿼리 |
| `internal/message/` | 메시지 타입, 저장/조회, 헬퍼 함수 |
| `internal/agentv2/` | 에이전트 CRUD, 버전 관리, 성능 추적 |
| `internal/attention/` | Attention 상태, Compact 이벤트, 리포트 |
| `internal/session/` | 계층적 세션 지원 (hierarchy.go) |

### Phase 2 (완료)

| 패키지 | 기능 |
|--------|------|
| `internal/orchestrator/` | Orchestration 포트, Worker Pair 스폰, 실행기 |
| `internal/handoff/` | 포트 간 컨텍스트 전달, 토큰 예산 검증 |
| `internal/escalation/` | 확장된 에스컬레이션 (타입, 심각도) |

### Phase 3 (완료)

| 패키지 | 기능 |
|--------|------|
| `internal/mcp/` | MCP Server (Claude Desktop 연동) |
| `internal/server/api_v2.go` | v2 REST API |
| `internal/server/websocket.go` | SSE 실시간 이벤트 |

## CLI 명령어

### 기본 명령어

```bash
pal init                    # 프로젝트 초기화
pal serve                   # HTTP 서버 시작
pal mcp                     # MCP 서버 시작
pal mcp config              # Claude Desktop 설정 출력
```

### Orchestration 관리

```bash
pal orchestration create "user-service" -p "port-001,port-002,port-003"
pal orchestration list --status running
pal orchestration show <orch-id>
pal orchestration stats <orch-id>
```

### 세션 계층 조회

```bash
pal hierarchy show <root-session-id>    # 트리 뷰
pal hierarchy list <root-session-id>    # 목록 뷰
pal hierarchy stats <root-session-id>   # 통계
pal hierarchy builds --active           # Build 세션 목록
```

### Attention 관리

```bash
pal attention show <session-id>
pal attention report <session-id>
pal attention history <session-id>
pal attention init <session-id> --budget 15000
```

### Handoff 관리

```bash
pal handoff list <port-id> --direction from
pal handoff create <from-port> <to-port> -t api_contract -c '{"entity":"User"}'
pal handoff estimate -c '{"fields":[...]}'
pal handoff total <port-id>
```

### 에이전트 관리

```bash
pal agent list --type worker
pal agent show <agent-id>
pal agent versions <agent-id>
pal agent stats <agent-id> 2
pal agent compare <agent-id> --v1 1 --v2 2
pal agent create "impl-worker" -t worker -d "구현 워커"
pal agent new-version <agent-id> --summary "Compact 빈도 개선"
```

## MCP Server

### 설정 (Claude Desktop)

`claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "pal-kit": {
      "command": "pal",
      "args": ["mcp", "--db", "~/.pal/pal.db", "--project", "/path/to/project"]
    }
  }
}
```

### MCP Tools

| Tool | 설명 |
|------|------|
| `session_start` | 새 세션 시작 |
| `session_end` | 세션 종료 |
| `session_hierarchy` | 세션 계층 조회 |
| `attention_status` | Attention 상태 조회 |
| `attention_update` | Attention 업데이트 |
| `orchestration_create` | Orchestration 생성 |
| `orchestration_status` | Orchestration 상태 |
| `message_send` | 메시지 전송 |
| `message_receive` | 메시지 수신 |
| `handoff_create` | Handoff 생성 |
| `handoff_get` | Handoff 조회 |
| `agent_list` | 에이전트 목록 |
| `agent_version` | 에이전트 버전 |
| `compact_record` | Compact 기록 |

### MCP Prompts

| Prompt | 설명 |
|--------|------|
| `start_build` | 새 빌드 세션 시작 |
| `worker_context` | Worker 컨텍스트 로드 |
| `test_feedback` | 테스트 결과 피드백 |

### MCP Resources

| Resource | 설명 |
|----------|------|
| `pal://sessions/active` | 활성 세션 목록 |
| `pal://orchestrations/running` | 실행 중 Orchestration |
| `pal://agents` | 에이전트 목록 |

## HTTP API v2

### Orchestration

```
GET  /api/v2/orchestrations
POST /api/v2/orchestrations
GET  /api/v2/orchestrations/:id
GET  /api/v2/orchestrations/:id/stats
GET  /api/v2/orchestrations/:id/workers
POST /api/v2/orchestrations/:id/start
```

### Session Hierarchy

```
GET  /api/v2/sessions/hierarchy
GET  /api/v2/sessions/hierarchy/:id
GET  /api/v2/sessions/hierarchy/:id/tree
GET  /api/v2/sessions/hierarchy/:id/list
GET  /api/v2/sessions/hierarchy/:id/stats
GET  /api/v2/sessions/builds
```

### Attention

```
GET  /api/v2/attention/:session_id
GET  /api/v2/attention/:session_id/report
GET  /api/v2/attention/:session_id/history
POST /api/v2/attention/:session_id/init
```

### Handoff

```
GET  /api/v2/handoffs?port=xxx
POST /api/v2/handoffs
GET  /api/v2/handoffs/:id
POST /api/v2/handoffs/estimate
```

### Agent v2

```
GET  /api/v2/agents
POST /api/v2/agents
GET  /api/v2/agents/:id
GET  /api/v2/agents/:id/versions
POST /api/v2/agents/:id/versions
GET  /api/v2/agents/:id/stats
GET  /api/v2/agents/:id/compare?v1=1&v2=2
```

### Messages

```
GET  /api/v2/messages?conversation=xxx
GET  /api/v2/messages?session=xxx
POST /api/v2/messages
POST /api/v2/messages/:id/delivered
POST /api/v2/messages/:id/processed
```

### Workers

```
GET  /api/v2/workers?orchestration=xxx
GET  /api/v2/workers/:id
```

### SSE Events

```
GET  /api/v2/events?channel=xxx
POST /api/v2/events/emit
```

**이벤트 타입:**
- `session.start`, `session.end`, `session.update`
- `orchestration.start`, `orchestration.update`, `orchestration.complete`
- `worker.spawn`, `worker.complete`
- `port.update`
- `attention.warning`
- `escalation.new`
- `message.new`

## Storage

```
SQLite (OLTP)                  DuckDB (OLAP)
├── sessions (계층 확장)        ├── docs-index.json
├── messages                    ├── conventions.json
├── agents                      └── token-history.parquet
├── agent_versions
├── agent_performance
├── session_attention
├── compact_events
├── worker_sessions
├── orchestration_ports
├── port_handoffs
└── escalations (확장)
```

## Phase 완료 현황

### Phase 1 ✅
- [x] DB 스키마 v10
- [x] DuckDB analytics 패키지
- [x] 메시지 패싱 패키지
- [x] 에이전트 버전 관리 패키지
- [x] Attention 추적 패키지
- [x] 세션 계층 확장

### Phase 2 ✅
- [x] Orchestrator 패키지 (Worker Pair 스폰)
- [x] Executor (의존성 기반 실행)
- [x] Handoff 패키지 (컨텍스트 전달)
- [x] Escalation 확장
- [x] CLI 명령어 추가

### Phase 3 ✅
- [x] MCP Server 구현
- [x] MCP Tools/Prompts/Resources
- [x] HTTP API v2 구현
- [x] SSE 실시간 이벤트
- [x] CLI mcp 명령어

### Phase 4 ✅
- [x] Electron GUI (`electron-gui/`)
- [x] 세션 계층 시각화
- [x] Compact Alert UI
- [x] 에이전트 진화 뷰

### Phase 5 ✅
- [x] 통합 테스트 (Go: orchestrator, handoff, attention, session)
- [x] Unit 테스트 (Vitest: hooks)
- [x] E2E 테스트 (Playwright)
- [x] 패키징 (electron-builder)
- [x] CI/CD (GitHub Actions)

## 에이전트 템플릿

`agents/v1/` 디렉토리:
- `spec-agent.md` - 명세 설계 에이전트
- `operator-agent.md` - Operator 에이전트
- `impl-worker.md` - 구현 Worker
- `test-worker.md` - 테스트 Worker

## 개발 가이드

### MCP 사용 예시

```
# Claude Desktop에서
> session_start로 build 세션 시작해줘

# 결과:
{
  "session": {
    "id": "xxx",
    "type": "build",
    "status": "running"
  }
}

> 요구사항을 분석해서 orchestration_create로 실행 계획을 만들어줘
```

### HTTP API 사용 예시

```bash
# Orchestration 생성
curl -X POST http://localhost:8080/api/v2/orchestrations \
  -H "Content-Type: application/json" \
  -d '{
    "title": "user-service",
    "ports": [
      {"port_id": "port-001", "order": 1},
      {"port_id": "port-002", "order": 2, "depends_on": ["port-001"]}
    ]
  }'

# SSE 이벤트 수신
curl -N http://localhost:8080/api/v2/events
```

## Electron GUI

### 실행 방법

```bash
cd electron-gui
npm install
npm run electron:dev
```

### 구조

```
electron-gui/
├── electron/
│   ├── main.ts          # Electron 메인 프로세스
│   └── preload.ts       # 프리로드 스크립트
└── src/
    ├── hooks/           # API 및 SSE 훅
    ├── components/      # 재사용 컴포넌트
    └── pages/           # 페이지 컴포넌트
```

### 주요 페이지

| 페이지 | 설명 |
|--------|------|
| Dashboard | 개요, 실행 중 Orchestration, 최근 이벤트 |
| Sessions | 세션 계층 트리 뷰, 상세 정보 |
| Orchestrations | 필터링, 통계, 상태 관리 |
| Agents | 버전 히스토리, 버전 비교 |
| Attention | 토큰 사용률, Compact 이력, 경고 |

### 주요 컴포넌트

| 컴포넌트 | 설명 |
|----------|------|
| `SessionTree` | 세션 계층 트리 뷰 |
| `AttentionGauge` | 토큰 사용률 게이지 |
| `OrchestrationProgress` | Orchestration 진행률 카드 |
| `AgentCard` | 에이전트 정보 카드 |
| `CompactAlert` | Compact 경고 오버레이 |

## 관련 문서

- **온보딩 가이드**: `docs/ONBOARDING.md`
- 설계: `mcp-docs/10-Personal/Projects/pal-kit/specs/`
- 기존 문서: `docs/`


<!-- pal:context:start -->
> 마지막 업데이트: 2026-01-24 15:35:00

### 포트 현황
- ✅ complete: 15
- ⏳ pending: 0
- 🔄 running: 0

### 에스컬레이션
- 없음

<!-- pal:context:end -->
