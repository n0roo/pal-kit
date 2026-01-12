# Port: worker-backend-package

> PA-Layered Backend 워커 명세 구현

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | worker-backend-package |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | agent-package-system |
| 예상 복잡도 | high |

---

## 목표

PA-Layered Backend 패키지에 포함되는 6종의 워커 명세를 구현한다.
각 워커는 특정 레이어에 특화되어 포트 명세 기반으로 동작한다.

---

## 범위

### 포함

- Entity Worker (L1 - JPA/JOOQ)
- Cache Worker (L1 - Redis/Valkey)
- Document Worker (L1 - MongoDB)
- Service Worker (L2/LM)
- Router Worker (L3)
- Test Worker (Test Layer)

### 제외

- 워커 실행 로직 (별도 구현)
- Claude 프롬프트 통합 (별도 포트)

---

## 워커 목록 및 레이어 매핑

| 워커 ID | 이름 | 레이어 | 포트 타입 | 기술 |
|---------|------|--------|-----------|------|
| entity-worker | Entity Worker | L1 | tpl-server-l1-port | JPA, JOOQ |
| cache-worker | Cache Worker | L1 | tpl-server-l1-cache-port | Redis, Valkey |
| document-worker | Document Worker | L1 | tpl-server-l1-document-port | MongoDB |
| service-worker | Service Worker | L2, LM | tpl-server-l2-port, tpl-server-lm-port | - |
| router-worker | Router Worker | L3 | API Endpoint 부분 | Spring MVC |
| test-worker | Test Worker | Test | TC 체크리스트 | JUnit5, RestDoc |

---

## 작업 항목

### Entity Worker

- [x] 워커 명세 작성 (agents/workers/backend/entity.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/backend/entity.md)
  - [x] JPA Entity 규칙
  - [x] JOOQ Repository 규칙
  - [x] CommandService 규칙 (CQS)
  - [x] QueryService 규칙 (CQS)
  - [x] DTO 네이밍 규칙
- [x] 완료 체크리스트 정의
- [x] 에스컬레이션 기준 정의

### Cache Worker

- [x] 워커 명세 작성 (agents/workers/backend/cache.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/backend/cache.md)
  - [x] Redis Template 사용법
  - [x] Key 네이밍 전략
  - [x] TTL 정책
  - [x] Cache-Aside 패턴
- [x] 완료 체크리스트 정의

### Document Worker

- [x] 워커 명세 작성 (agents/workers/backend/document.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/backend/document.md)
  - [x] MongoDB Document 규칙
  - [x] Index 설계 가이드
  - [x] Aggregation Pipeline
- [x] 완료 체크리스트 정의

### Service Worker

- [x] 워커 명세 작성 (agents/workers/backend/service.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/backend/service.md)
  - [x] L2 CompositeService 규칙
  - [x] LM Coordinator 규칙
  - [x] 의존성 규칙 (L2 → LM → L1)
  - [x] Event 발행 규칙
  - [x] 트랜잭션 경계
- [x] 완료 체크리스트 정의
- [x] 에스컬레이션 기준 정의 (의존성 위반)

### Router Worker

- [x] 워커 명세 작성 (agents/workers/backend/router.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/backend/router.md)
  - [x] Controller 네이밍
  - [x] REST API 규칙
  - [x] 인증/인가 설정
  - [x] Validation 규칙
  - [x] Exception Handler
- [x] 완료 체크리스트 정의

### Test Worker

- [x] 워커 명세 작성 (agents/workers/backend/test.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/backend/test.md)
  - [x] JUnit5 규칙
  - [x] Given-When-Then 패턴
  - [x] Spring RestDoc 가이드
  - [x] Mock 설정 가이드
  - [x] 커버리지 목표
- [x] 완료 체크리스트 정의

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| Entity Worker | agents/workers/backend/entity.yaml | 명세 |
| Cache Worker | agents/workers/backend/cache.yaml | 명세 |
| Document Worker | agents/workers/backend/document.yaml | 명세 |
| Service Worker | agents/workers/backend/service.yaml | 명세 |
| Router Worker | agents/workers/backend/router.yaml | 명세 |
| Test Worker | agents/workers/backend/test.yaml | 명세 |
| 컨벤션 문서 | conventions/workers/backend/*.md | 6종 |

---

## 완료 기준

- [x] 6종 워커 명세 파일 작성 완료
- [x] 6종 컨벤션 문서 작성 완료
- [x] 모든 워커에 완료 체크리스트 정의
- [x] PA-Layered 의존성 규칙 반영 확인
- [x] 워커 YAML에 prompt 포함 (Claude 연계용)

---

## 완료 요약

### 생성된 워커 명세 (YAML)

| 워커 | 파일 | 레이어 | 기술 |
|------|------|--------|------|
| Entity Worker | `agents/workers/backend/entity.yaml` | L1 | JPA, Hibernate, JOOQ |
| Cache Worker | `agents/workers/backend/cache.yaml` | L1 | Redis, Valkey |
| Document Worker | `agents/workers/backend/document.yaml` | L1 | MongoDB |
| Service Worker | `agents/workers/backend/service.yaml` | L2, LM | Spring TX |
| Router Worker | `agents/workers/backend/router.yaml` | L3 | Spring MVC |
| Test Worker | `agents/workers/backend/test.yaml` | Test | JUnit5, RestDoc |

### 컨벤션 문서 (MD)

| 워커 | 파일 | 내용 |
|------|------|------|
| Entity Worker | `conventions/agents/workers/backend/entity.md` | Entity, Repository, CQS |
| Cache Worker | `conventions/agents/workers/backend/cache.md` | Cache-Aside, TTL, 분산 락 |
| Document Worker | `conventions/agents/workers/backend/document.md` | Document, Index, Aggregation |
| Service Worker | `conventions/agents/workers/backend/service.md` | Coordinator, CompositeService |
| Router Worker | `conventions/agents/workers/backend/router.md` | Controller, Validation, Exception |
| Test Worker | `conventions/agents/workers/backend/test.md` | 통합 테스트, E2E, RestDoc |

---

<!-- pal:port:status=complete -->
