# Port: claude-integration

> Claude Code와 PAL Kit 에이전트 연계 시스템

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | claude-integration |
| 상태 | draft |
| 우선순위 | high |
| 의존성 | agent-package-system, worker-backend-package, worker-frontend-package |
| 예상 복잡도 | high |

---

## 목표

PAL Kit 에이전트 시스템과 Claude Code를 연계하여,
포트 명세 기반으로 적절한 워커가 자동 할당되고 컨텍스트가 구성되도록 한다.

---

## 범위

### 포함

- 컨텍스트 로딩 순서 정의
- 프롬프트 구성 템플릿
- 워커 전환 로직
- CLAUDE.md 자동 업데이트
- Hook 연동 (port-start, port-end)

### 제외

- Claude Code 내부 수정
- API 직접 호출 (CLI 통해서만)

---

## 컨텍스트 로딩 순서

```
┌─────────────────────────────────────────────────────────────────┐
│                    Claude Context Loading                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. CLAUDE.md (프로젝트 기본 정보)                               │
│       ↓                                                         │
│  2. 패키지 컨벤션 (architecture.md)                              │
│       ↓                                                         │
│  3. 워커 공통 컨벤션 (_common.md)                                │
│       ↓                                                         │
│  4. 워커 개별 컨벤션 ({worker}.md)                               │
│       ↓                                                         │
│  5. 포트 명세 (ports/{port-id}.md)                               │
│       ↓                                                         │
│  6. 워커 프롬프트 (agents/{worker}.yaml → prompt)                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 작업 항목

### 컨텍스트 로딩

- [ ] 로딩 순서 정의 및 문서화
- [ ] 패키지별 컨벤션 경로 매핑
- [ ] 조건부 로딩 (포트 타입 → 워커 컨벤션)
- [ ] 토큰 예산 관리

### 프롬프트 구성

- [ ] 프롬프트 템플릿 정의
  ```
  # {Worker Name}

  ## 패키지 정보
  {package.description}

  ## 아키텍처 규칙
  {architecture.conventions}

  ## 워커 컨벤션
  {worker.conventions}

  ## 현재 포트 명세
  {port.spec}

  ## 완료 체크리스트
  {worker.completion.checklist}

  ## 작업 지침
  포트 명세의 요구사항을 구현하세요.
  완료 체크리스트의 모든 항목을 만족해야 합니다.
  ```
- [ ] 동적 프롬프트 생성 로직
- [ ] 프롬프트 캐싱

### 워커 전환

- [ ] 포트 타입 → 워커 매핑 규칙
  ```yaml
  mapping:
    L1: [entity-worker, cache-worker, document-worker]
    LM: [service-worker]
    L2: [service-worker, router-worker]
    L3: [router-worker]
    client-api: [component-model-worker]
    client-feature: [frontend-engineer-worker]
    client-component: [component-ui-worker]
  ```
- [ ] 포트 내 tech 힌트로 워커 선택
- [ ] 전환 시 컨텍스트 유지/갱신

### CLAUDE.md 업데이트

- [ ] 현재 활성 워커 표시
- [ ] 포트 작업 상태 표시
- [ ] 완료 체크리스트 진행률 표시

### Hook 연동

- [ ] `port-start` hook
  - [ ] 포트 타입 분석
  - [ ] 워커 결정
  - [ ] 컨텍스트 로딩
  - [ ] CLAUDE.md 업데이트
- [ ] `port-end` hook
  - [ ] 완료 체크리스트 검증
  - [ ] 다음 워커로 전환 (필요시)
  - [ ] 상태 업데이트

### CLI 명령어

- [ ] `pal context show` - 현재 로드된 컨텍스트 표시
- [ ] `pal context reload` - 컨텍스트 새로고침
- [ ] `pal worker switch <id>` - 수동 워커 전환

---

## 워커 전환 규칙 상세

### Backend 패키지

| 포트 타입 | 기본 워커 | 선택 기준 |
|----------|----------|-----------|
| L1 | entity-worker | tech=jpa/jooq |
| L1 | cache-worker | tech=redis/valkey |
| L1 | document-worker | tech=mongodb |
| LM | service-worker | - |
| L2 | service-worker | 비즈니스 로직 |
| L2 | router-worker | API Endpoint |
| TC | test-worker | - |

### Frontend 패키지

| 포트 타입 | 기본 워커 | 선택 기준 |
|----------|----------|-----------|
| client-feature | frontend-engineer-worker | 오케스트레이션 |
| client-api | component-model-worker | API 연동 |
| client-query | component-model-worker | Data Fetching |
| client-component | component-ui-worker | UI 구현 |
| e2e | e2e-worker | - |
| unit-tc | unit-tc-worker | - |

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| 컨텍스트 서비스 | internal/context/service.go | 로딩 로직 |
| 프롬프트 빌더 | internal/prompt/builder.go | 프롬프트 구성 |
| 워커 매퍼 | internal/worker/mapper.go | 워커 결정 |
| Hook 핸들러 | internal/hook/claude.go | Hook 연동 |
| 연계 가이드 | docs/claude-integration.md | 사용 가이드 |

---

## 완료 기준

- [ ] 컨텍스트 로딩 순서 구현 및 동작 확인
- [ ] 포트 타입 → 워커 자동 매핑 동작
- [ ] Hook 연동으로 자동 컨텍스트 갱신
- [ ] CLAUDE.md 자동 업데이트 동작
- [ ] `pal context show` 명령어 동작

---

<!-- pal:port:status=draft -->
