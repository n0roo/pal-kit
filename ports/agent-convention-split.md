# Port: agent-convention-split

> 코어/워커 에이전트 컨벤션 분리 및 고도화

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | agent-convention-split |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | agent-spec-review |
| 예상 복잡도 | high |

---

## 목표

Core 에이전트와 Worker 에이전트의 특화된 컨벤션을 분리하여 작업 완료성을 확보한다.

---

## 범위

### 포함

- Core 에이전트 공통 컨벤션 정의
- Core 에이전트 개별 컨벤션 분리
- Worker 에이전트 공통 컨벤션 정의
- Worker 에이전트 기술 스택별 컨벤션 분리
- 컨벤션 로딩 메커니즘 구현/개선
- 작업 완료 체크리스트 표준화

### 제외

- 에이전트 실행 로직 변경
- 새 에이전트 타입 추가

---

## 작업 항목

### Core 컨벤션 분리

- [x] 공통 컨벤션 정의 (`conventions/agents/core/_common.md`)
  - [x] 역할 전환 규칙
  - [x] 에스컬레이션 기준
  - [x] 완료 체크리스트 표준
- [x] 개별 컨벤션 분리
  - [x] builder 컨벤션
  - [x] planner 컨벤션
  - [x] architect 컨벤션
  - [x] manager 컨벤션
  - [x] tester 컨벤션
  - [x] logger 컨벤션

### Worker 컨벤션 분리

- [x] 공통 컨벤션 정의 (`conventions/agents/workers/_common.md`)
  - [x] 코드 품질 기준 (CQS 규칙, 에러 처리, 코드 스타일)
  - [x] 테스트 요구사항 (단위 테스트 책임, 커버리지 목표)
  - [x] 완료 체크리스트 표준
- [x] 레이어별 Worker 컨벤션 (PA-Layered 기반)
  - [x] Backend Workers (6종)
    - [x] entity-worker (L1 JPA/ORM)
    - [x] cache-worker (L1 Redis)
    - [x] document-worker (L1 MongoDB)
    - [x] service-worker (LM/L2)
    - [x] router-worker (L3 API)
    - [x] test-worker (테스트 보완)
  - [x] Frontend Workers (5종)
    - [x] frontend-engineer-worker (Orchestration)
    - [x] component-model-worker (Logic)
    - [x] component-ui-worker (View)
    - [x] e2e-worker (E2E 테스트)
    - [x] unit-tc-worker (단위 테스트)

### 메커니즘 구현

- [x] 컨벤션 로딩 순서 정의 (공통 → 개별 → 포트)
- [x] 컨벤션 매칭 알고리즘 설계
- [x] 패키지 오버라이드 메커니즘 정의
- [x] Claude 통합 방안 문서화

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| Core 공통 컨벤션 | conventions/agents/core/_common.md | 코어 에이전트 공통 규칙 |
| Core 개별 컨벤션 (6종) | conventions/agents/core/*.md | builder, planner, architect, manager, tester, logger |
| Worker 공통 컨벤션 | conventions/agents/workers/_common.md | 워커 에이전트 공통 규칙 (CQS, 테스트, 에러처리) |
| Backend Worker 컨벤션 (6종) | conventions/agents/workers/backend/*.md | entity, cache, document, service, router, test |
| Frontend Worker 컨벤션 (5종) | conventions/agents/workers/frontend/*.md | engineer, model, ui, e2e, unit-tc |
| 로딩 메커니즘 문서 | conventions/CONVENTION-LOADING.md | 컨벤션 로딩/적용 규칙 |

---

## 완료 기준

- [x] 모든 Core 에이전트에 개별 컨벤션 적용 (6종)
- [x] PA-Layered 기반 Worker 컨벤션 정의 (Backend 6종 + Frontend 5종)
- [x] 작업 완료 체크리스트가 모든 에이전트에 표준화
- [x] 컨벤션 로딩 메커니즘 문서화

---

<!-- pal:port:status=complete -->
