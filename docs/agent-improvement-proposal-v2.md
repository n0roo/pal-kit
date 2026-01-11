# 에이전트 고도화 제안서 v2

> 작성일: 2026-01-12 | 포트: agent-spec-review (보완)

---

## 1. 개요

### 1.1 v1 대비 변경사항

| 영역 | v1 | v2 |
|------|-----|-----|
| 에이전트 구조 | 단일 레벨 | **패키지 → 워커** 계층 구조 |
| 워커 정의 | 기술 스택 기반 | **레이어 + 역할 기반** |
| 포트 연동 | 없음 | **포트 명세 기반 동작** |
| 아키텍처 | 범용 | **PA-Layered 특화** |

### 1.2 핵심 개념

```
┌─────────────────────────────────────────────────────────────────┐
│                         Package                                 │
│  (기술스택 + 아키텍처 + 컨벤션 + 개발방법론의 묶음)               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│   │ Core Agent  │  │ Core Agent  │  │ Core Agent  │            │
│   │  (Builder)  │  │ (Architect) │  │  (Manager)  │            │
│   └─────────────┘  └─────────────┘  └─────────────┘            │
│           │                │                │                   │
│           └────────────────┼────────────────┘                   │
│                            │                                    │
│                     Port 명세 기반                               │
│                            │                                    │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│   │   Worker    │  │   Worker    │  │   Worker    │            │
│   │  (Layer)    │  │  (Layer)    │  │  (Layer)    │            │
│   └─────────────┘  └─────────────┘  └─────────────┘            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 패키지(Package) 개념 정의

### 2.1 패키지란?

패키지는 **특정 기술 스택과 아키텍처에 특화된 에이전트 그룹**을 정의합니다.

```yaml
package:
  id: string                    # 패키지 고유 ID
  name: string                  # 패키지 이름
  version: string               # 버전

  # 기술 스택
  tech:
    language: string            # 주 언어 (kotlin, typescript)
    frameworks: string[]        # 프레임워크 목록
    build_tool: string          # 빌드 도구

  # 아키텍처
  architecture:
    name: string                # PA-Layered, Clean Architecture 등
    layers: string[]            # 레이어 정의
    conventions_ref: string     # 컨벤션 문서 경로

  # 개발 방법론
  methodology:
    port_driven: boolean        # 포트 명세 기반 개발
    cqs: boolean                # Command/Query 분리
    event_driven: boolean       # 이벤트 기반 통신

  # 포함 워커
  workers: string[]             # 워커 ID 목록

  # Core 에이전트 커스터마이징
  core_overrides:
    architect:
      conventions_ref: string   # 아키텍처 특화 컨벤션
    builder:
      port_templates: string[]  # 포트 템플릿 목록
```

### 2.2 기본 제공 패키지: PA-Layered Backend

```yaml
package:
  id: pa-layered-backend
  name: PA-Layered Backend Package
  version: 1.0.0

  tech:
    language: kotlin
    frameworks:
      - spring-boot
      - spring-cloud
      - jpa
      - jooq
    build_tool: gradle-multimodule

  architecture:
    name: PA-Layered
    layers:
      - L1 (Domain)
      - LM (Shared Composition)
      - L2 (Feature Composition)
      - L3 (Router/API Gateway)
    conventions_ref: conventions/pa-layered/backend/

  methodology:
    port_driven: true
    cqs: true
    event_driven: true

  workers:
    - entity-worker
    - cache-worker
    - document-worker
    - service-worker
    - router-worker
    - test-worker

  core_overrides:
    architect:
      conventions_ref: conventions/pa-layered/architecture.md
    builder:
      port_templates:
        - tpl-server-l1-port
        - tpl-server-lm-port
        - tpl-server-l2-port
```

### 2.3 기본 제공 패키지: PA-Layered Frontend

```yaml
package:
  id: pa-layered-frontend
  name: PA-Layered Frontend Package
  version: 1.0.0

  tech:
    language: typescript
    frameworks:
      - react
      - next.js
      - tanstack-query
      - mui
      - tailwind
    build_tool: pnpm

  architecture:
    name: PA-Layered (Client)
    layers:
      - API (Service Layer)
      - Query (Data Fetching)
      - Feature (Business Logic)
      - Component (UI)
    conventions_ref: conventions/pa-layered/frontend/

  methodology:
    port_driven: true
    component_driven: true

  workers:
    - frontend-engineer-worker
    - component-model-worker
    - component-ui-worker
    - e2e-worker
    - unit-tc-worker
```

---

## 3. Backend 패키지 워커 상세

### 3.1 레이어-워커 매핑

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 3: Router / API Gateway                                   │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ router-worker                                               │ │
│ │ - Controller 작성                                           │ │
│ │ - 인증/인가 설정                                             │ │
│ │ - API Gateway 라우팅                                        │ │
│ └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ Layer 2: Feature Composition                                    │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ service-worker                                              │ │
│ │ - CompositeService 구현                                     │ │
│ │ - 비즈니스 플로우 조합                                       │ │
│ │ - Event 발행                                                │ │
│ └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ Layer M: Shared Composition                                     │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ service-worker                                              │ │
│ │ - Coordinator 구현                                          │ │
│ │ - 횡단 관심사 조합                                           │ │
│ └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ Layer 1: Domain                                                 │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐                │
│ │entity-worker│ │cache-worker │ │document-wkr │                │
│ │ JPA/JOOQ    │ │ Redis/Valkey│ │ MongoDB     │                │
│ │ Command/Qry │ │ Cache Layer │ │ NoSQL       │                │
│ └─────────────┘ └─────────────┘ └─────────────┘                │
├─────────────────────────────────────────────────────────────────┤
│ Test Layer                                                      │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ test-worker                                                 │ │
│ │ - Unit Test                                                 │ │
│ │ - Spring RestDoc                                            │ │
│ │ - Integration Test                                          │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Entity Worker

```yaml
worker:
  id: entity-worker
  name: Entity Worker
  type: worker
  layer: L1
  category: data

  description: |
    JPA/JOOQ 기반 엔티티 및 Repository를 작성하는 L1 레이어 워커.
    PA-Layered의 Command/Query 분리 원칙을 준수합니다.

  port_types:
    - L1 (tpl-server-l1-port)

  responsibilities:
    - Entity 클래스 작성
    - Repository 인터페이스 정의
    - CommandService 구현 (Create, Update, Delete)
    - QueryService 구현 (Read, Search)
    - DTO 정의 (Request, Response, Filter)

  conventions:
    tech:
      - JPA Entity 네이밍: {DomainName}.kt
      - JOOQ 사용 시: {Domain}JooqRepository
      - Repository: {Entity}Repository
    cqs:
      - CommandService: 단일 엔티티 쓰기 전용
      - QueryService: 읽기 전용, 도메인 내 조합 허용
      - Master DB Query: CommandService 내 Private 허용
    naming:
      - Create Request: Create{Entity}Request
      - Update Request: Update{Entity}Request
      - Filter: {Entity}Filter

  tools:
    - gradle
    - ktlint
    - detekt

  completion:
    checklist:
      - Entity 정의 완료
      - Repository 구현 완료
      - CommandService 구현 완료 (CUD)
      - QueryService 구현 완료 (R)
      - DTO 정의 완료
      - 컴파일 성공
      - TC 작성 (test-worker 연계)
    required: true

  escalation:
    criteria:
      - 다른 도메인 참조 필요
      - 복잡한 조인 쿼리 필요
      - 아키텍처 변경 필요
    target: architect
```

### 3.3 Cache Worker

```yaml
worker:
  id: cache-worker
  name: Cache Worker
  type: worker
  layer: L1
  category: data

  description: |
    Redis/Valkey 기반 캐시 레이어를 작성하는 L1 워커.
    캐시 전략과 TTL 정책을 포트 명세에 따라 구현합니다.

  port_types:
    - L1-Cache (tpl-server-l1-cache-port)

  responsibilities:
    - Cache Repository 구현
    - Cache Key 전략 정의
    - TTL 정책 구현
    - 캐시 무효화 로직
    - 직렬화/역직렬화 처리

  conventions:
    tech:
      - Redis Template 사용
      - Key 네이밍: {domain}:{entity}:{id}
      - Value: JSON 직렬화
    patterns:
      - Cache-Aside 패턴 기본
      - Write-Through 선택적
    naming:
      - CacheRepository: {Entity}CacheRepository
      - CacheKey: {Entity}CacheKey enum

  tools:
    - redis-cli
    - gradle

  completion:
    checklist:
      - CacheRepository 구현 완료
      - Key 전략 정의 완료
      - TTL 설정 완료
      - 무효화 로직 구현
      - 직렬화 테스트 통과
    required: true
```

### 3.4 Document Worker

```yaml
worker:
  id: document-worker
  name: Document Worker
  type: worker
  layer: L1
  category: data

  description: |
    MongoDB 기반 Document 저장소를 작성하는 L1 워커.
    스키마리스 데이터와 복잡한 쿼리를 처리합니다.

  port_types:
    - L1-Document (tpl-server-l1-document-port)

  responsibilities:
    - Document 클래스 정의
    - MongoRepository 구현
    - Index 설계
    - Aggregation Pipeline 작성
    - 마이그레이션 스크립트

  conventions:
    tech:
      - Spring Data MongoDB 사용
      - Document 네이밍: {Entity}Document
      - Collection 네이밍: snake_case
    patterns:
      - Embedded Document vs Reference
      - Index 전략 문서화
    naming:
      - Document: {Entity}Document
      - Repository: {Entity}DocumentRepository

  completion:
    checklist:
      - Document 클래스 정의 완료
      - Repository 구현 완료
      - Index 생성 완료
      - Aggregation 쿼리 구현 (필요시)
      - 컴파일 성공
    required: true
```

### 3.5 Service Worker

```yaml
worker:
  id: service-worker
  name: Service Worker
  type: worker
  layer: L2/LM
  category: business

  description: |
    L2(Feature Composition)와 LM(Shared Composition) 레이어의
    비즈니스 로직을 작성하는 워커.

  port_types:
    - L2 (tpl-server-l2-port)
    - LM (tpl-server-lm-port)

  responsibilities:
    # L2 Feature
    - CompositeService 구현
    - API Endpoint 대응 로직
    - L1, LM 서비스 조합
    - Event 발행
    # LM Shared
    - Coordinator 구현
    - 횡단 관심사 로직
    - 재사용 가능한 조합

  conventions:
    l2:
      - 단일 API Endpoint 대응
      - L1, LM 자유 참조
      - 다른 L2 참조 금지 (Event로 통신)
      - 네이밍: {Feature}CompositeService
    lm:
      - 재사용 목적
      - L1만 참조 (다른 LM 단방향)
      - 네이밍: {Feature}Coordinator
    events:
      - 발행: {Feature}{Action}Event
      - Listener: {Feature}EventListener

  completion:
    checklist:
      - 포트 명세 요구사항 충족
      - 의존성 규칙 준수
      - 트랜잭션 경계 설정
      - Event 발행 구현 (필요시)
      - 컴파일 성공
      - 단위 테스트 요청 (test-worker)
    required: true

  escalation:
    criteria:
      - 다른 L2 직접 참조 필요
      - 순환 참조 발생
      - 의존성 규칙 위반 불가피
    target: architect
```

### 3.6 Router Worker

```yaml
worker:
  id: router-worker
  name: Router Worker
  type: worker
  layer: L3
  category: api

  description: |
    API Gateway, Controller, 인증/인가를 담당하는 L3 레이어 워커.
    포트 명세의 API Endpoint를 실제 라우팅으로 구현합니다.

  port_types:
    - L2 (API Endpoint 부분)
    - API Gateway 설정

  responsibilities:
    - Controller 클래스 작성
    - Request/Response 매핑
    - 인증/인가 설정 (@PreAuthorize)
    - API Versioning
    - Exception Handler
    - API Gateway 라우팅 (필요시)

  conventions:
    controller:
      - REST 규칙 준수
      - 네이밍: {Feature}Controller
      - 경로: /api/v{version}/{resource}
    security:
      - Method Security 우선
      - Role 기반 접근 제어
    validation:
      - @Valid 사용
      - DTO 레벨 검증

  completion:
    checklist:
      - Controller 구현 완료
      - API 경로 매핑 완료
      - 인증/인가 설정 완료
      - Validation 적용 완료
      - Exception Handler 연동
      - 컴파일 성공
    required: true
```

### 3.7 Test Worker

```yaml
worker:
  id: test-worker
  name: Test Worker
  type: worker
  layer: test
  category: quality

  description: |
    단위 테스트와 Spring RestDoc을 담당하는 테스트 워커.
    다른 워커의 산출물에 대한 테스트 케이스를 작성합니다.

  port_types:
    - L1 TC 체크리스트
    - L2 TC 체크리스트

  responsibilities:
    - Unit Test 작성
    - Integration Test 작성
    - Spring RestDoc 명세
    - Mock 설정
    - Test Fixture 관리

  conventions:
    naming:
      - Test Class: {Target}Test
      - Integration: {Target}IntegrationTest
      - RestDoc: {Feature}ApiDocTest
    structure:
      - Given-When-Then 패턴
      - 테이블 드리븐 테스트 권장
    restdoc:
      - 모든 API Endpoint 문서화
      - Request/Response 필드 설명

  inputs:
    - port_spec: TC 체크리스트
    - target_code: 테스트 대상 코드

  completion:
    checklist:
      - 포트 TC 체크리스트 전체 구현
      - 테스트 전체 통과
      - 커버리지 목표 달성 (80%+)
      - RestDoc 생성 확인
    required: true
```

---

## 4. Frontend 패키지 워커 상세

### 4.1 레이어-워커 매핑

```
┌─────────────────────────────────────────────────────────────────┐
│ Page/Feature Layer                                              │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ frontend-engineer-worker                                    │ │
│ │ - 페이지 단위 작업 관리                                      │ │
│ │ - 태스크 세분화                                              │ │
│ │ - 하위 워커 조율                                             │ │
│ └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ Component Model Layer (Logic)                                   │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ component-model-worker                                      │ │
│ │ - API Service 연동                                          │ │
│ │ - Custom Hooks                                              │ │
│ │ - State Management                                          │ │
│ └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ Component UI Layer (View)                                       │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ component-ui-worker                                         │ │
│ │ - UI 컴포넌트 작성                                           │ │
│ │ - MUI / Tailwind 적용                                       │ │
│ │ - 스타일링                                                   │ │
│ └─────────────────────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│ Test Layer                                                      │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ e2e-worker          │ unit-tc-worker                        │ │
│ │ Playwright E2E      │ Mock 기반 Unit Test                   │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Frontend Engineer Worker

```yaml
worker:
  id: frontend-engineer-worker
  name: Frontend Engineer Worker
  type: worker
  layer: feature
  category: orchestration

  description: |
    프론트엔드 페이지 단위 작업을 관리하는 오케스트레이션 워커.
    하나의 기능 흐름을 태스크로 세분화하고 하위 워커를 조율합니다.

  port_types:
    - client-feature (tpl-client-feature-composition)

  responsibilities:
    - 페이지/기능 단위 작업 분해
    - 하위 워커 태스크 할당
    - 작업 순서 조율
    - 통합 검증

  workflow:
    1. 기능 명세 분석
    2. 컴포넌트 목록 도출
    3. Model/UI 워커 태스크 생성
    4. 구현 순서 결정
    5. 통합 테스트 조율

  conventions:
    task_decomposition:
      - API 연동 → component-model-worker
      - UI 구현 → component-ui-worker
      - E2E 테스트 → e2e-worker
      - 단위 테스트 → unit-tc-worker
    directory:
      - features/{feature}/
      - components/{feature}/

  completion:
    checklist:
      - 기능 명세 분석 완료
      - 하위 태스크 생성 완료
      - 모든 하위 워커 작업 완료
      - 통합 동작 확인
    required: true
```

### 4.3 Component Model Worker

```yaml
worker:
  id: component-model-worker
  name: Component Model Worker
  type: worker
  layer: model
  category: logic

  description: |
    API Service, Custom Hook, State 관리를 담당하는 워커.
    백엔드 API와의 연동 로직을 구현합니다.

  port_types:
    - client-api (tpl-client-api-port)
    - client-query (tpl-client-query-adapter)

  responsibilities:
    - API Service 함수 작성
    - TanStack Query Hook 구현
    - Custom Hook 작성
    - Type 정의 (Request/Response)
    - Error Handling

  conventions:
    api_service:
      - 파일: services/{feature}Api.ts
      - 함수: {action}{Resource}
      - Axios/Fetch 래핑
    query:
      - Hook: use{Feature}
      - Query Key: {feature}Keys
      - Mutation: use{Action}{Feature}
    types:
      - Request: {Feature}Request
      - Response: {Feature}Response

  tools:
    - typescript
    - eslint
    - prettier

  completion:
    checklist:
      - API Service 구현 완료
      - Query/Mutation Hook 구현 완료
      - Type 정의 완료
      - Error Handling 구현
      - API 연동 테스트 통과
    required: true
```

### 4.4 Component UI Worker

```yaml
worker:
  id: component-ui-worker
  name: Component UI Worker
  type: worker
  layer: ui
  category: presentation

  description: |
    UI 컴포넌트를 작성하는 워커.
    MUI/Tailwind 가이드라인을 따르며 재사용 가능한 컴포넌트를 구현합니다.

  port_types:
    - client-component (tpl-client-component-port)

  responsibilities:
    - React 컴포넌트 작성
    - Props Interface 정의
    - 스타일링 (MUI/Tailwind)
    - 접근성 (a11y) 적용
    - Storybook 스토리 작성 (선택)

  conventions:
    structure:
      - 파일: components/{Category}/{ComponentName}.tsx
      - 인덱스: components/{Category}/index.ts
    styling:
      mui:
        - sx prop 사용
        - theme 커스터마이징
        - styled() 최소화
      tailwind:
        - className 직접 사용
        - 커스텀 클래스 최소화
        - dark mode 지원
    naming:
      - 컴포넌트: PascalCase
      - Props: {ComponentName}Props

  completion:
    checklist:
      - 컴포넌트 구현 완료
      - Props 타입 정의 완료
      - 스타일 적용 완료
      - 반응형 처리 완료
      - 접근성 기본 적용
    required: true

  ui_guidelines:
    - MUI 컴포넌트 우선 사용
    - Tailwind로 세부 조정
    - 디자인 시스템 일관성 유지
```

### 4.5 E2E Worker

```yaml
worker:
  id: e2e-worker
  name: E2E Test Worker
  type: worker
  layer: test
  category: quality

  description: |
    Playwright 기반 E2E 테스트를 작성하는 워커.
    사용자 시나리오 기반 테스트를 구현합니다.

  responsibilities:
    - E2E 테스트 시나리오 작성
    - Page Object 패턴 적용
    - 테스트 데이터 설정
    - CI/CD 연동

  conventions:
    structure:
      - 테스트: e2e/{feature}.spec.ts
      - Page Object: e2e/pages/{Feature}Page.ts
      - Fixtures: e2e/fixtures/
    patterns:
      - Page Object Model
      - AAA (Arrange-Act-Assert)
    naming:
      - 테스트: should {expected behavior} when {condition}

  tools:
    - playwright
    - @playwright/test

  completion:
    checklist:
      - 주요 시나리오 테스트 작성
      - Page Object 구현
      - 테스트 전체 통과
      - CI 연동 확인
    required: true
```

### 4.6 Unit TC Worker

```yaml
worker:
  id: unit-tc-worker
  name: Unit Test Worker
  type: worker
  layer: test
  category: quality

  description: |
    Mock 기반 단위 테스트를 작성하는 워커.
    Component Model Worker의 비즈니스 로직을 테스트합니다.

  responsibilities:
    - Hook 단위 테스트
    - Service 함수 테스트
    - 컴포넌트 렌더링 테스트
    - Mock 설정

  conventions:
    structure:
      - 테스트: __tests__/{target}.test.ts
      - Mock: __mocks__/
    patterns:
      - Testing Library 사용
      - Mock Service Worker (msw)
      - renderHook for Hooks
    coverage:
      - Hook: 100%
      - Service: 100%
      - 컴포넌트: 80%+

  tools:
    - vitest / jest
    - @testing-library/react
    - msw

  completion:
    checklist:
      - Hook 테스트 작성 완료
      - Service 테스트 작성 완료
      - 테스트 전체 통과
      - 커버리지 목표 달성
    required: true
```

---

## 5. Core 에이전트 역할 재정의

### 5.1 Architect 역할 확장

```yaml
agent:
  id: architect
  name: Architect
  type: core

  description: |
    프로젝트의 아키텍처(PA-Layered 등)와 컨벤션을 기반으로
    포트 명세의 유효성을 검증하고, 워커 할당을 결정합니다.

  responsibilities:
    - 아키텍처 결정 및 문서화
    - 포트 명세 검증 (레이어 규칙, 의존성)
    - 워커 유형 결정 (포트 타입 → 워커 매핑)
    - 레이어 간 의존성 검증
    - Builder 에이전트와 협업

  collaboration:
    builder:
      - Builder가 포트 초안 작성
      - Architect가 아키텍처 준수 검증
      - 검증 실패 시 Builder에게 수정 요청
    workers:
      - 포트 타입에 따른 워커 할당 결정
      - 레이어별 적합한 워커 지정

  validation_rules:
    pa_layered:
      - L1은 다른 도메인 참조 금지
      - LM은 L2 참조 금지
      - L2는 다른 L2 참조 금지 (Event만)
      - 순환 참조 감지

  completion:
    checklist:
      - 포트 명세 아키텍처 준수 확인
      - 의존성 규칙 검증 완료
      - 워커 할당 결정 완료
      - Builder 승인 완료
    required: true
```

### 5.2 Builder 역할 확장

```yaml
agent:
  id: builder
  name: Builder
  type: core

  description: |
    요구사항을 분석하고 적절한 포트 타입으로 분해합니다.
    Architect와 협업하여 아키텍처 준수를 보장합니다.

  responsibilities:
    - 요구사항 분석
    - 포트 타입 결정 (L1/LM/L2/API 등)
    - 포트 명세 초안 작성
    - Architect 검증 요청
    - 의존성 관계 정의

  port_templates:
    backend:
      - tpl-server-l1-port (Entity Worker)
      - tpl-server-lm-port (Service Worker)
      - tpl-server-l2-port (Service Worker, Router Worker)
    frontend:
      - tpl-client-api-port (Component Model Worker)
      - tpl-client-feature-composition (Frontend Engineer Worker)
      - tpl-client-component-port (Component UI Worker)

  workflow:
    1. 요구사항 분석
    2. 레이어 결정 (L1/LM/L2 등)
    3. 포트 템플릿 선택
    4. 포트 명세 작성
    5. Architect 검증 요청
    6. 수정 또는 승인
    7. 워커 태스크 생성
```

---

## 6. 패키지 커스터마이징 가이드

### 6.1 사용자 정의 패키지 생성

```yaml
# .pal/packages/my-custom-package.yaml

package:
  id: my-custom-backend
  name: My Custom Backend Package
  version: 1.0.0
  extends: pa-layered-backend  # 기본 패키지 상속

  tech:
    language: kotlin
    frameworks:
      - spring-boot
      - spring-cloud
      - jpa
      - kafka  # 추가
    build_tool: gradle-multimodule

  # 추가 워커
  workers:
    - entity-worker
    - cache-worker
    - service-worker
    - router-worker
    - test-worker
    - kafka-worker  # 추가

  # 추가 컨벤션
  conventions:
    kafka:
      ref: conventions/kafka/
      producer_naming: "{Domain}Producer"
      consumer_naming: "{Domain}Consumer"
```

### 6.2 Claude 연계 지침

```yaml
# .pal/claude-integration.yaml

claude:
  # 패키지 컨텍스트 로딩
  context_loading:
    order:
      1. architecture conventions
      2. package conventions
      3. worker conventions
      4. port spec

  # 워커 전환 규칙
  worker_switching:
    trigger: port_type
    rules:
      - port_type: L1
        workers: [entity-worker, cache-worker, document-worker]
        select_by: tech_in_port
      - port_type: L2
        workers: [service-worker]
      - port_type: LM
        workers: [service-worker]
      - port_type: API
        workers: [router-worker]

  # 프롬프트 구성
  prompt_composition:
    template: |
      # {worker_name}

      ## 패키지 정보
      {package_description}

      ## 아키텍처 규칙
      {architecture_conventions}

      ## 워커 컨벤션
      {worker_conventions}

      ## 현재 포트 명세
      {port_spec}

      ## 작업 지침
      {task_instructions}
```

---

## 7. 구현 우선순위

### Phase 1: 패키지 시스템 구축

| 순서 | 작업 | 담당 |
|------|------|------|
| 1-1 | 패키지 스키마 정의 | architect |
| 1-2 | PA-Layered Backend 패키지 작성 | architect |
| 1-3 | PA-Layered Frontend 패키지 작성 | architect |
| 1-4 | 패키지 로딩 로직 구현 | developer |

### Phase 2: 워커 명세 작성

| 순서 | 작업 | 패키지 |
|------|------|--------|
| 2-1 | Backend 워커 6종 명세 | pa-layered-backend |
| 2-2 | Frontend 워커 5종 명세 | pa-layered-frontend |
| 2-3 | 워커 컨벤션 문서 작성 | both |

### Phase 3: Core 에이전트 업데이트

| 순서 | 작업 |
|------|------|
| 3-1 | Architect 역할 확장 (포트 검증) |
| 3-2 | Builder 역할 확장 (포트 템플릿) |
| 3-3 | Manager 역할 확장 (워커 조율) |

### Phase 4: Claude 연계

| 순서 | 작업 |
|------|------|
| 4-1 | 컨텍스트 로딩 순서 정의 |
| 4-2 | 프롬프트 구성 템플릿 |
| 4-3 | 워커 전환 로직 |

---

## 8. 기대 효과

| 영역 | AS-IS | TO-BE |
|------|-------|-------|
| 워커 정의 | 기술 스택 기반 (go, kotlin) | 레이어+역할 기반 (entity-worker, service-worker) |
| 포트 연동 | 없음 | 포트 명세 기반 자동 워커 할당 |
| 아키텍처 준수 | 수동 검증 | Architect 에이전트 자동 검증 |
| 컨벤션 관리 | 개별 프롬프트 | 패키지로 통합 관리 |
| 확장성 | 새 기술마다 워커 추가 | 패키지 상속으로 확장 |

---

<!-- pal:port:agent-spec-review:v2 -->
