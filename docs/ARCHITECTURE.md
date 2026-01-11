# PAL Kit 아키텍처 원칙

> 버전: 0.3.0
> 최종 수정: 2026-01-10

## 1. 핵심 철학

PAL Kit은 **멀티 세션 에이전트 오케스트레이션**을 목표로 합니다.

### 1.1 세션 독립성 (Session Independence)

```
각 에이전트 = 독립된 Claude 세션
```

- 빌더 에이전트가 하위 에이전트를 **별도 세션**으로 spawn
- 각 세션은 자신만의 컨텍스트 윈도우를 가짐
- 부모 세션의 전체 컨텍스트를 상속하지 **않음**

### 1.2 자기완결적 포트 명세 (Self-Contained Port Specification)

```
포트 문서 = 작업에 필요한 모든 것
```

- 에이전트 간 소통은 **구조화된 포트 명세**를 통해
- 하위 에이전트는 포트 문서만으로 작업 수행 가능해야 함
- 컨텍스트 로드 오버헤드 최소화

### 1.3 계층적 책임 분리 (Hierarchical Responsibility)

```
Builder → Engineer → Worker
(기획)    (관리)     (실행)
```

- **Builder**: 요건 분석, 태스크 분해, 세션 기획
- **Engineer**: 포트 명세 배포, 하위 Worker 관리
- **Worker**: 컨벤션 준수, 실제 코드 작성

---

## 2. 에이전트 아키텍처

### 2.1 계층 구조

```
Level 0: Master Builder (멀티 빌더 관리, 선택적)
         │
Level 1: Builder (세션별 1개)
         │
         ├─── Architecture (기술 검토)
         │
         └─── Port Engineer (API 계약 정의)
                   │
Level 2:          ├─── Backend Engineer
                  │         │
Level 3:          │         ├─── JPA Worker
                  │         ├─── Redis Worker
                  │         ├─── Mongo Worker
                  │         ├─── Context Worker (L0)
                  │         └─── API Service Worker (L2/LM)
                  │
                  └─── Frontend Engineer
                            │
                            ├─── FE Service Worker
                            ├─── FE Component Worker
                            └─── E2E Worker
```

### 2.2 에이전트 역할 정의

| 에이전트 | 역할 | 입력 | 출력 |
|----------|------|------|------|
| **Builder** | 요건→태스크 분해, 세션 기획 | 사용자 요구사항 | 작업 계획, 에이전트 구성 |
| **Architecture** | 기술 검토, 리뷰 | 작업 계획 | 기술 피드백, 승인/반려 |
| **Port Engineer** | API 계약 정의, 병렬작업 조율 | 승인된 계획 | 자기완결적 포트 명세 |
| **Backend Engineer** | BE Worker 관리, 포트 배포 | BE 포트 명세 | Worker 할당, 진행 관리 |
| **Frontend Engineer** | FE Worker 관리, 포트 배포 | FE 포트 명세 | Worker 할당, 진행 관리 |
| **JPA Worker** | L1 JPA 도메인 구현 | JPA 포트 명세 | Entity, Repository |
| **Redis Worker** | L1 Redis 도메인 구현 | Redis 포트 명세 | Cache 모듈 |
| **Mongo Worker** | L1 Mongo 도메인 구현 | Mongo 포트 명세 | Document 모듈 |
| **Context Worker** | L0 공통 컴포넌트 | Context 포트 명세 | 재사용 모듈 |
| **API Service Worker** | L2/LM 서비스 레이어 | Service 포트 명세 | API 구현 |
| **FE Service Worker** | API 연동, Hook 설계 | FE Service 포트 | Fetch, Hook |
| **FE Component Worker** | UI 컴포넌트 구현 | Component 포트 | React 컴포넌트 |
| **E2E Worker** | 테스트 작성 | Test 포트 명세 | Unit, E2E 테스트 |

### 2.3 단계별 도입 계획

#### Phase 1: 코어 에이전트 (4개)
```yaml
- builder
- architect  
- port-engineer
- generic-worker
```

#### Phase 2: 역할 분리 (6개)
```yaml
- backend-engineer
- frontend-engineer
```

#### Phase 3: 전문화 Worker (12개+)
```yaml
- jpa-worker
- redis-worker
- mongo-worker
- context-worker
- api-service-worker
- fe-service-worker
- fe-component-worker
- e2e-worker
```

---

## 3. 포트 명세 표준

### 3.1 자기완결성 요구사항

포트 명세는 다음을 **모두 포함**해야 함:

```markdown
# Port: {port-id}

## 1. 개요
- 목적, 범위

## 2. 입력 (Input)
- 의존하는 포트/모듈
- API 계약 (있는 경우)

## 3. 작업 범위 (Scope)
- 포함: 명시적 목록
- 제외: 명시적 목록

## 4. 기술 명세 (Technical Spec)
- 사용 기술, 컨벤션 참조
- 파일 패턴

## 5. 완료 조건 (Acceptance Criteria)
- 체크리스트
- 테스트 요구사항

## 6. 산출물 (Output)
- 생성할 파일 목록
- 다음 포트로 전달할 정보
```

### 3.2 API 계약 (BE/FE 병렬 작업)

Port Engineer가 정의하는 API 계약:

```yaml
# api-contract.yaml
endpoint: /api/orders
method: POST
request:
  body:
    type: CreateOrderRequest
    fields:
      - name: customerId
        type: string
        required: true
response:
  success:
    type: OrderResponse
  error:
    codes: [400, 401, 404]
```

---

## 4. 세션 관리

### 4.1 세션 생성 흐름

```
1. Builder 세션 시작
   ├─ pal session start --type builder --title "Feature X"
   └─ 파이프라인 계획 수립

2. 하위 세션 spawn
   ├─ pal session start --type sub --parent {builder-id} --port {port-id}
   └─ 포트 명세가 세션 컨텍스트로 로드됨

3. Worker 작업 수행
   ├─ 포트 명세 기반 작업
   └─ 완료 시 pal hook port-end {port-id}

4. 결과 집계
   └─ Builder가 하위 세션 상태 모니터링
```

### 4.2 컨텍스트 로드 전략

| 세션 유형 | 로드되는 컨텍스트 |
|----------|------------------|
| Builder | CLAUDE.md, 에이전트 목록, 파이프라인 상태 |
| Engineer | 담당 영역 포트 목록, 하위 Worker 상태 |
| Worker | 할당된 포트 명세, 관련 컨벤션 |

---

## 5. PAL Kit 연동

### 5.1 에이전트 정의 (`agents/`)

```yaml
# agents/builder.yaml
agent:
  id: builder
  name: Builder Agent
  type: builder
  prompt: |
    당신은 빌더 에이전트입니다.
    
    ## 책임
    - 요건 분석 및 태스크 분해
    - 에이전트 구성 계획
    - 하위 세션 spawn 및 관리
    
    ## 사용 가능한 도구
    - pal session start --type sub
    - pal pipeline create/add
    - pal port create
    
    ## 워크플로우
    1. 요건 분석
    2. 작업 분해 (포트 단위)
    3. 에이전트 할당
    4. 파이프라인 구성
    5. 실행 및 모니터링
  tools:
    - bash
    - pal
  context:
    - CLAUDE.md
    - agents/*.yaml
    - pipelines/
```

### 5.2 세션 뷰어 요구사항

세션별 다음 정보 조회:

- 로드된 CLAUDE.md 버전/해시
- 컨텍스트에 포함된 문서 목록
- 토큰 사용량 (입력/출력/캐시)
- 세션 실행 시간
- 할당된 에이전트
- 적용된 컨벤션
- 활성 rules 목록

### 5.3 알림 시스템

| 이벤트 | 알림 대상 | 내용 |
|--------|----------|------|
| 포트 완료 | 상위 Engineer/Builder | 완료 상태, 산출물 |
| 포트 실패 | 상위 Engineer/Builder | 실패 원인, 로그 |
| 에스컬레이션 | Builder | 이슈 내용, 블로커 |
| 리뷰 필요 | Architecture | 검토 요청 |
| 빌더 완료 | Master Builder | 전체 진행 상황 |

---

## 6. 구현 로드맵

### Phase A: 핵심 기능 완성
- [ ] `pal init` 구현
- [ ] port activate/deactivate 확인
- [ ] hook 실제 연동 테스트
- [ ] Web GUI 버그 수정

### Phase B: GUI 개선
- [ ] Web GUI undefined 수정
- [ ] 세션 뷰어 (컨텍스트, 토큰, 문서 목록)
- [ ] TUI 액션 추가
- [ ] 알림 시스템 기반

### Phase C: 에이전트 시스템 v1
- [ ] 포트 명세 표준 정의
- [ ] 코어 4 에이전트 템플릿
- [ ] 에이전트 간 메시지 패싱
- [ ] 파이프라인 자동 구성

### Phase D: 에이전트 시스템 v2
- [ ] BE/FE Engineer 분리
- [ ] 전문 Worker 추가
- [ ] 멀티 빌더 관리
- [ ] 알림/대시보드 통합

### Phase E: 실제 프로젝트 적용
- [ ] 파일럿 프로젝트 선정
- [ ] 워크플로우 테스트
- [ ] 피드백 기반 개선

---

## 7. 참고

### 7.1 관련 문서

- [README.md](../README.md) - 사용법
- [ports/](../ports/) - 포트 명세 예시
- [agents/](../agents/) - 에이전트 정의

### 7.2 설계 결정 기록

| 날짜 | 결정 | 근거 |
|------|------|------|
| 2026-01-10 | 멀티 세션 구조 채택 | 컨텍스트 격리, 토큰 효율화 |
| 2026-01-10 | 자기완결적 포트 명세 | 핸드오프 오버헤드 최소화 |
| 2026-01-10 | 단계적 에이전트 도입 | 복잡도 관리, 점진적 검증 |
