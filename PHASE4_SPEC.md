# PAL Kit v1.0 Phase 4 작업 명세

> 작성일: 2026-01-22
> 목적: 컨텍스트 압축 시 작업 유실 방지를 위한 백업

## 프로젝트 정보

- **경로**: `/Users/n0roo/playground/CodeSpace/pal-kit`
- **브랜치**: `feat/redesign-pakit-v1`
- **현재 Phase**: 4 (Electron GUI) ✅ 완료

## Phase 1-3 완료 요약

### Phase 1 ✅
- `internal/db/schema.go` - v10 스키마
- `internal/analytics/` - DuckDB 연동
- `internal/message/` - 메시지 패싱
- `internal/agentv2/` - 에이전트 버전 관리
- `internal/attention/` - Attention 추적
- `internal/session/hierarchy.go` - 세션 계층

### Phase 2 ✅
- `internal/orchestrator/orchestrator.go` - Orchestration, Worker Pair
- `internal/orchestrator/executor.go` - 의존성 기반 실행
- `internal/handoff/handoff.go` - 컨텍스트 전달 (2000 토큰)
- `internal/escalation/escalation.go` - 확장 (타입/심각도)
- CLI: orchestration, hierarchy, attention, handoff, agent

### Phase 3 ✅
- `internal/mcp/server.go` - MCP Server (14 Tools)
- `internal/server/api_v2.go` - HTTP API v2
- `internal/server/websocket.go` - SSE 실시간 이벤트
- CLI: `pal mcp`, `pal mcp config`

## Phase 4 완료 ✅

### 1. Electron 프로젝트 구조 ✅
```
electron-gui/
├── package.json            ✅
├── vite.config.ts          ✅
├── tsconfig.json           ✅
├── tsconfig.node.json      ✅
├── tailwind.config.js      ✅
├── postcss.config.js       ✅
├── index.html              ✅
├── .gitignore              ✅
├── electron/
│   ├── main.ts             ✅ Electron 메인 프로세스
│   └── preload.ts          ✅ 프리로드 스크립트
└── src/
    ├── main.tsx            ✅ React 진입점
    ├── index.css           ✅ Tailwind CSS
    ├── App.tsx             ✅ 메인 앱 (라우팅, 레이아웃)
    ├── hooks/
    │   ├── index.ts        ✅
    │   ├── useApi.ts       ✅ API 클라이언트 훅
    │   └── useSSE.ts       ✅ SSE 실시간 이벤트 훅
    ├── components/
    │   ├── index.ts        ✅
    │   ├── StatusBar.tsx   ✅ 하단 상태 바
    │   ├── CompactAlert.tsx ✅ Compact 경고 오버레이
    │   ├── SessionTree.tsx ✅ 세션 트리 뷰
    │   ├── AttentionGauge.tsx ✅ 토큰 사용률 게이지
    │   ├── OrchestrationProgress.tsx ✅ Orchestration 진행률
    │   └── AgentCard.tsx   ✅ 에이전트 카드
    └── pages/
        ├── index.ts        ✅
        ├── Dashboard.tsx   ✅ 대시보드 (개요, 최근 이벤트)
        ├── Sessions.tsx    ✅ 세션 계층 시각화
        ├── Orchestrations.tsx ✅ Orchestration 관리
        ├── Agents.tsx      ✅ 에이전트 진화 뷰
        └── Attention.tsx   ✅ Attention 모니터
```

### 2. 핵심 컴포넌트 ✅

#### 2.1 세션 계층 시각화 ✅
- `SessionTree.tsx` - 트리 뷰 (Build → Operator → Worker → Test)
- `Sessions.tsx` - 세션 목록, 검색, 상세 뷰

#### 2.2 Orchestration 대시보드 ✅
- `OrchestrationProgress.tsx` - 진행률 바, 상태 표시
- `Orchestrations.tsx` - 필터, 통계 사이드바

#### 2.3 Attention 모니터 ✅
- `AttentionGauge.tsx` - 토큰 사용률, Focus/Drift 점수
- `CompactAlert.tsx` - Compact 경고 알림 오버레이
- `Attention.tsx` - 세션 선택, 이력, 권장사항

#### 2.4 에이전트 진화 뷰 ✅
- `AgentCard.tsx` - 에이전트 카드 (타입별 색상)
- `Agents.tsx` - 버전 히스토리, 버전 비교

### 3. API 연동 ✅

**useApi.ts 훅:**
- `useApi()` - 상태 폴링
- `useSessions()` - 세션 목록, 계층 조회
- `useOrchestrations()` - Orchestration 목록, 통계
- `useAgents()` - 에이전트 목록, 버전, 비교
- `useAttention()` - Attention 상태, 리포트, 이력

**useSSE.ts 훅:**
- SSE 연결 관리 (자동 재연결)
- 이벤트 타입별 필터링
- 이벤트 히스토리 (최대 100개)

### 4. UI 디자인 ✅

**컬러 스킴:**
- Primary: Blue (#3B82F6)
- Success: Green (#22C55E)
- Warning: Yellow (#EAB308)
- Error: Red (#EF4444)
- Background: Slate (#0F172A)

**상태 표시:**
- status-running: 초록 발광
- status-complete: 회색
- status-failed: 빨강 발광
- status-pending: 노랑

**애니메이션:**
- alert-pulse: 경고 깜빡임
- progress-indeterminate: 진행 중 바
- card-hover: 카드 호버 효과

## 통합 아키텍처

```
┌─────────────────────────────────────────┐
│           Electron App (Single Process)     │
│  ┌─────────────────────────────────────┐  │
│  │          Main Process                   │  │
│  │  ┌───────────────┐ ┌───────────────┐ │  │
│  │  │  PAL Server   │ │  IPC Handlers │ │  │
│  │  │  (spawned)    │ │  (api-request)│ │  │
│  │  └───────────────┘ └───────────────┘ │  │
│  │         │              │               │  │
│  │         └──── http ────┘               │  │
│  └─────────────────────────────────────┘  │
│              ↑ IPC                            │
│  ┌─────────────────────────────────────┐  │
│  │       Renderer Process (React)          │  │
│  │  - window.palAPI.getStatus()           │  │
│  │  - window.palAPI.getOrchestrations()   │  │
│  │  - window.palAPI.getSessions()         │  │
│  │  - NO direct HTTP calls (no CORS)      │  │
│  └─────────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

**핵심 특징:**
- Renderer는 HTTP 직접 호출 없음 (CORS 문제 없음)
- 모든 API 호출은 IPC를 통해 Main Process로 전달
- Main Process가 내부적으로 PAL 서버에 HTTP 요청
- 사용자에게는 단일 앱으로 보임

## 실행 방법

```bash
# 1. PAL Kit 서버 실행 (필수)
cd /Users/n0roo/playground/CodeSpace/pal-kit
pal serve

# 2. Electron GUI 실행
cd electron-gui
npm install
npm run electron:dev
```

**주의**: Electron GUI는 http://localhost:8080 에서 실행되는 PAL Kit 서버와 통신합니다. 서버가 실행 중이어야 합니다.

## 버그 수정 이력

### 2026-01-22
- `electron-squirrel-startup` 모듈 제거 (개발 시 불필요)
- API 에러 처리 개선 (null 반환, 빈 배열 처리)
- SSE 연결 재시도 로직 추가 (지수 백오프)
- `HierarchicalSession` 타입 추가 및 Sessions 페이지 수정
- Dashboard에 서버 연결 경고 추가
- Attention 페이지 null 체크 개선
- CSP 헤더 업데이트 (localhost:8080 허용)

## 복구 시 참고

컨텍스트 압축 후 이 파일 읽기:
```bash
cat /Users/n0roo/playground/CodeSpace/pal-kit/PHASE4_SPEC.md
```

CLAUDE.md 확인:
```bash
cat /Users/n0roo/playground/CodeSpace/pal-kit/CLAUDE.md
```

## 다음 작업 (Phase 5 예정)

- [ ] 통합 테스트
- [ ] E2E 테스트 (Playwright)
- [ ] 패키징 (electron-builder)
- [ ] 배포 준비
