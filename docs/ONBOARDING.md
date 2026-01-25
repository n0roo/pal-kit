# PAL Kit 온보딩 가이드

> Personal Agentic Layer - Claude Code 에이전트 오케스트레이션 도구
> Version: 1.0.0

---

## 1. PAL Kit이란?

PAL Kit은 **Claude Code 에이전트의 작업 품질과 일관성을 보장**하기 위한 도구입니다.

### 1.1 해결하는 문제

| 문제 | PAL Kit 해결책 |
|------|---------------|
| **컨텍스트 손실** | 세션/포트 단위 작업 추적, Attention 모니터링 |
| **작업 추적 불가** | 포트 명세 기반 진행 상태 관리 |
| **컨벤션 미준수** | 규칙 자동 주입, Hook 기반 강제 |
| **복잡한 작업 관리** | Orchestration으로 다단계 작업 조율 |
| **멀티세션 혼란** | 세션 계층화, 메시지 패싱 |

### 1.2 핵심 가치

```
작업 단위(Port) + 추적(Session) + 품질(Convention) = 일관된 결과물
```

---

## 2. v1.0 리디자인 방향성

### 2.1 설계 원칙

1. **집중** - Claude Hook 연계 + 작업 명세 관리에 집중
2. **단순화** - 16종 에이전트 → 3계층 (Spec, Operator, Worker Pair)
3. **자기완결성** - 포트명세가 ~10K 토큰의 독립 실행 단위
4. **Attention 보존** - 컴팩션 최소화, 토큰 예산 관리

### 2.2 아키텍처 개요

```
┌─────────────────────────────────────────────────────────┐
│  Spec Agent (Claude Desktop / MCP)                      │
│  - 사용자와 협업하여 명세 작성                            │
│  - 포트명세를 ~10K 토큰 단위로 분해                       │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│  Operator (PAL Kit Core)                                │
│  - Orchestration 실행                                    │
│  - Worker Pair 스폰/관리                                 │
│  - 메시지 패싱 중계                                      │
└─────────────────────┬───────────────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        ▼                           ▼
┌───────────────────┐       ┌───────────────────┐
│   Impl Worker     │◄─────►│   Test Worker     │
│   (구현)           │ Msg   │   (테스트)         │
└───────────────────┘       └───────────────────┘
```

### 2.3 세션 계층

| Depth | Type | 역할 |
|-------|------|------|
| 0 | **build** | 명세 설계, 포트 분해 |
| 1 | **operator** | 워커 관리, 진행 조율 |
| 2 | **worker** | 코드 구현 |
| 3 | **test** | 테스트 작성/실행 |

---

## 3. 핵심 개념

### 3.1 Port (포트)

작업의 **기본 단위**. 하나의 완결된 구현 범위를 정의합니다.

```yaml
# ports/user-entity.md
---
type: port
title: User Entity 구현
status: running
priority: high
dependencies: []
---

## 목표
User 도메인 엔티티 및 Repository 구현

## 작업 범위
- internal/user/entity.go
- internal/user/repository.go

## 완료 기준
- [ ] 엔티티 정의
- [ ] CRUD 구현
- [ ] 테스트 통과
```

**포트 상태 흐름:**
```
pending → running → complete
                  → blocked (에스컬레이션)
```

### 3.2 Session (세션)

Claude Code 실행 **컨텍스트 단위**. Hook을 통해 자동 추적됩니다.

```
세션 = 하나의 Claude Code 실행 인스턴스
     + 활성 포트 연결
     + Attention 상태
     + 이벤트 로그
```

### 3.3 Orchestration (오케스트레이션)

여러 포트를 **의존성 순서대로 실행**하는 관리 단위.

```
Orchestration: "user-service 구현"
├── port-001: user-entity (순서: 1)
├── port-002: user-repository (순서: 2, 의존: port-001)
└── port-003: user-service (순서: 3, 의존: port-002)
```

### 3.4 Attention (어텐션)

세션의 **토큰 사용량과 집중도** 추적.

| 지표 | 설명 |
|------|------|
| `loaded_tokens` | 현재 로드된 토큰 수 |
| `available_tokens` | 남은 토큰 예산 |
| `focus_score` | 집중도 (0.0~1.0) |
| `compact_count` | 컴팩션 횟수 |

### 3.5 Handoff (핸드오프)

포트 간 **컨텍스트 전달**. 다음 작업에 필요한 정보를 구조화하여 전달.

```go
Handoff{
    FromPort: "port-001",
    ToPort:   "port-002",
    Type:     "api_contract",
    Context:  { "entity": "User", "fields": [...] },
    TokenEstimate: 850,
}
```

---

## 4. 기능 연계도

```
┌─────────────────────────────────────────────────────────────────┐
│                         PAL Kit v1.0                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────┐    Hook     ┌─────────┐    Event    ┌─────────┐  │
│   │ Claude  │◄───────────►│   CLI   │◄───────────►│   DB    │  │
│   │  Code   │             │  (pal)  │             │ SQLite  │  │
│   └─────────┘             └────┬────┘             └─────────┘  │
│                                │                               │
│        ┌───────────────────────┼───────────────────────┐       │
│        │                       │                       │       │
│        ▼                       ▼                       ▼       │
│   ┌─────────┐            ┌─────────┐            ┌─────────┐   │
│   │ Session │            │  Port   │            │  Agent  │   │
│   │ Tracking│            │ Manager │            │ Manager │   │
│   └────┬────┘            └────┬────┘            └────┬────┘   │
│        │                      │                      │        │
│        ▼                      ▼                      ▼        │
│   ┌─────────┐            ┌─────────┐            ┌─────────┐   │
│   │Attention│            │Handoff  │            │Convention│  │
│   │ Tracker │            │ Manager │            │ Loader  │   │
│   └─────────┘            └─────────┘            └─────────┘   │
│                                                                │
│   ┌─────────────────────────────────────────────────────────┐ │
│   │                    Orchestrator                          │ │
│   │  - 의존성 기반 포트 실행                                   │ │
│   │  - Worker Pair 스폰                                       │ │
│   │  - 메시지 패싱                                            │ │
│   └─────────────────────────────────────────────────────────┘ │
│                                                                │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│   │ MCP Server  │  │ HTTP API    │  │ Electron    │          │
│   │ (Desktop)   │  │ (pal serve) │  │ GUI         │          │
│   └─────────────┘  └─────────────┘  └─────────────┘          │
│                                                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. 주요 기능 상세

### 5.1 Claude Code Hook 연동

PAL Kit은 Claude Code의 **모든 Hook 이벤트**를 처리합니다.

| Hook | 트리거 | PAL Kit 처리 |
|------|--------|-------------|
| `SessionStart` | 세션 시작 | 세션 생성, 워크플로우 규칙 주입 |
| `SessionEnd` | 세션 종료 | 세션 종료 기록, 통계 저장 |
| `PreToolUse` | 도구 사용 전 | 포트 체크, 수정 경고 |
| `PostToolUse` | 도구 사용 후 | 변경 사항 기록 |
| `PreCompact` | 컴팩션 전 | Attention 스냅샷 저장 |
| `Stop` | 중단 | 상태 저장 |

**설정 위치:** `.claude/settings.json`

```json
{
  "hooks": {
    "SessionStart": [{ "type": "command", "command": "pal hook session-start" }],
    "PreToolUse": [{ "type": "command", "command": "pal hook pre-tool-use" }]
  }
}
```

### 5.2 Knowledge Base (KB)

Vault 스타일의 **문서 관리 시스템**.

```bash
# KB 초기화
pal kb init ~/my-vault

# 목차 생성
pal kb toc generate

# 색인 구축 및 검색
pal kb index --rebuild
pal kb search "authentication"

# 링크 검사
pal kb link check

# 프로젝트 동기화
pal kb sync /path/to/project
```

**디렉토리 구조:**
```
vault/
├── _taxonomy/          # 분류체계
├── 00-System/          # 시스템 문서
├── 10-Domains/         # 도메인 지식
├── 20-Projects/        # 프로젝트 문서
├── 30-References/      # 참조 문서
└── .pal-kb/            # 메타데이터 (색인 DB)
```

### 5.3 에이전트 시스템

**템플릿 기반 에이전트 관리**.

```bash
# 에이전트 목록
pal agent list

# 에이전트 상세
pal agent show builder

# 버전 관리
pal agent versions builder
pal agent compare builder --v1 1 --v2 2
```

**에이전트 구조:**
```
agents/
├── core/
│   ├── builder.yaml       # 빌더 에이전트 정의
│   ├── builder.rules.md   # 빌더 규칙
│   ├── operator.yaml
│   └── support.yaml
└── workers/
    ├── backend/
    │   ├── entity.yaml
    │   └── service.yaml
    └── frontend/
        ├── ui.yaml
        └── e2e.yaml
```

### 5.4 MCP Server

**Claude Desktop 연동**을 위한 MCP 서버.

```bash
# MCP 서버 실행
pal mcp --project /path/to/project

# 설정 출력 (claude_desktop_config.json용)
pal mcp config
```

**제공 도구:**
- `session_start/end` - 세션 관리
- `port_create/update` - 포트 관리
- `orchestration_create` - 오케스트레이션
- `attention_status` - Attention 조회
- `message_send/receive` - 메시지 패싱

### 5.5 HTTP API & GUI

```bash
# HTTP 서버 시작
pal serve --port 8080

# Electron GUI (개발)
cd electron-gui && npm run electron:dev
```

**API 엔드포인트:**
- `GET /api/v2/sessions` - 세션 목록
- `GET /api/v2/orchestrations` - 오케스트레이션
- `GET /api/v2/attention/:id` - Attention 상태
- `GET /api/v2/events` - SSE 이벤트 스트림

---

## 6. 빠른 시작

### 6.1 설치

```bash
# Go 설치 (1.21+)
go install github.com/n0roo/pal-kit/cmd/pal@latest

# 또는 소스에서 빌드
git clone https://github.com/n0roo/pal-kit
cd pal-kit
go install ./cmd/pal
```

### 6.2 전역 설정

```bash
# PAL Kit 전역 설치
pal install

# 설치 확인
pal doctor
```

### 6.3 프로젝트 초기화

```bash
cd /your/project
pal init

# 생성되는 구조:
# .claude/
# ├── settings.json    # Hook 설정
# └── rules/           # 워크플로우 규칙
# agents/              # 에이전트 템플릿
# conventions/         # 컨벤션
# ports/               # 포트 명세
# CLAUDE.md            # 프로젝트 컨텍스트
```

### 6.4 기본 워크플로우

```bash
# 1. 포트 생성
pal port create user-auth --title "사용자 인증 구현"

# 2. Claude Code 실행 (Hook 자동 연동)
claude

# 3. 포트 활성화 (Claude 내에서)
# > pal hook port-start user-auth

# 4. 작업 수행...

# 5. 포트 완료
pal port status user-auth complete

# 6. 상태 확인
pal status
```

---

## 7. 패키지 구조

```
internal/
├── cli/            # CLI 명령어
├── db/             # SQLite 스키마 및 연결
├── session/        # 세션 관리, 계층 구조
├── port/           # 포트 CRUD
├── agent/          # 에이전트 템플릿
├── convention/     # 컨벤션 로딩
├── attention/      # Attention 추적
├── handoff/        # 포트 간 컨텍스트 전달
├── orchestrator/   # 오케스트레이션 실행
├── message/        # 메시지 패싱
├── escalation/     # 에스컬레이션 처리
├── workflow/       # 워크플로우 규칙 생성
├── kb/             # Knowledge Base
├── mcp/            # MCP Server
├── server/         # HTTP API
└── analytics/      # DuckDB 분석
```

---

## 8. 데이터 모델

### 8.1 핵심 테이블

```sql
-- 세션 (계층 구조)
sessions (
    id, project_root, status, session_type,
    parent_session_id, depth, cwd, ...
)

-- 포트
ports (
    id, project_root, title, status, priority,
    file_path, session_id, ...
)

-- Attention 상태
session_attention (
    session_id, port_id, loaded_tokens,
    available_tokens, focus_score, ...
)

-- 메시지
messages (
    id, conversation_id, from_session, to_session,
    type, payload, attention_score, token_count, ...
)

-- 오케스트레이션
orchestration_ports (
    id, title, atomic_ports, status,
    current_port_id, progress_percent, ...
)
```

---

## 9. 개발 가이드

### 9.1 빌드 및 테스트

```bash
# 빌드
go build ./cmd/pal

# 테스트
go test ./...

# 린트
golangci-lint run
```

### 9.2 Electron GUI 개발

```bash
cd electron-gui
npm install
npm run electron:dev
```

### 9.3 CI/CD

- **ci.yml**: Go 테스트, Electron 테스트, E2E (Playwright), Lint
- **release.yml**: 멀티플랫폼 빌드, GitHub Release

---

## 10. 로드맵

### 완료 (v1.0)

- [x] 세션/포트 추적
- [x] Claude Code Hook 연동
- [x] Attention 모니터링
- [x] 오케스트레이션
- [x] 메시지 패싱
- [x] Knowledge Base
- [x] MCP Server
- [x] Electron GUI
- [x] CI/CD 파이프라인

### 계획 (v1.1+)

- [ ] 자동 토큰 측정/경고
- [ ] Spec Agent 고도화
- [ ] 실시간 대시보드 개선
- [ ] 플러그인 시스템

---

## 부록: 용어집

| 용어 | 설명 |
|------|------|
| **Port** | 작업의 기본 단위, 완결된 구현 범위 |
| **Session** | Claude Code 실행 컨텍스트 |
| **Orchestration** | 여러 포트의 순차 실행 관리 |
| **Attention** | 토큰 사용량 및 집중도 지표 |
| **Handoff** | 포트 간 컨텍스트 전달 |
| **Hook** | Claude Code 이벤트 핸들러 |
| **Worker Pair** | Impl + Test 에이전트 쌍 |
| **Escalation** | 문제 상위 전달 메커니즘 |

---

> 문서 버전: 1.0.0
> 마지막 업데이트: 2026-01-24
