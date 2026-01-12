# 에이전트 고도화 제안서

> 작성일: 2026-01-12 | 포트: agent-spec-review

---

## 1. 개요

### 1.1 목적

PAL Kit 에이전트의 완성도를 높여 작업 완료성과 품질을 보장한다.

### 1.2 목표

1. 에이전트 명세 구조 표준화
2. 역할 책임 명확화
3. 완료 체크리스트 표준화
4. 컨벤션 분리 및 체계화

---

## 2. 제안 1: 에이전트 명세 구조 표준화

### 2.1 표준 스키마 (Core)

```yaml
agent:
  # 필수 필드
  id: string                    # 고유 식별자
  name: string                  # 표시 이름
  type: core                    # 에이전트 타입
  workflow: [simple|single|integrate|multi]  # 지원 워크플로우

  # 역할 정의
  description: string           # 에이전트 설명
  responsibilities: string[]    # 책임 목록

  # 입출력
  inputs: string[]             # 필요 입력 (NEW)
  outputs: string[]            # 산출물

  # 도구
  commands: string[]           # PAL CLI 명령어

  # 컨벤션
  conventions:                 # 컨벤션 참조 (NEW)
    common: string             # 공통 컨벤션 경로
    specific: string           # 개별 컨벤션 경로

  # 완료 기준 (NEW)
  completion:
    checklist: string[]        # 완료 체크리스트
    required: boolean          # 체크리스트 필수 여부

  # 에스컬레이션 (NEW)
  escalation:
    criteria: string[]         # 에스컬레이션 기준
    target: string             # 에스컬레이션 대상 (manager|user)

  # 프롬프트
  prompt: string               # 시스템 프롬프트
```

### 2.2 표준 스키마 (Worker)

```yaml
agent:
  # 필수 필드
  id: string                    # worker-{tech}
  name: string                  # {Tech} Worker
  type: worker
  tech: string                  # 기술 스택 식별자
  workflow: [single|integrate|multi]

  # 역할 정의
  description: string
  responsibilities: string[]

  # 입출력
  inputs:                      # (NEW)
    - port-spec               # 포트 명세
    - conventions             # 컨벤션
  outputs:                     # (NEW)
    - source-code
    - tests

  # 도구
  tools: string[]              # 기술 스택 도구
  commands: string[]           # PAL CLI 명령어 (NEW)

  # 컨벤션
  conventions:                 # (NEW - 구조화)
    common: conventions/agents/workers/_common.md
    tech: conventions/agents/workers/{tech}.md

  # 완료 기준 (NEW)
  completion:
    checklist:
      - 빌드 성공
      - 테스트 통과
      - 컨벤션 준수
    required: true

  # 프롬프트
  prompt: string
```

---

## 3. 제안 2: 역할 책임 명확화

### 3.1 테스트 책임 분리

**현재 문제:** Tester와 Worker 모두 테스트 작성 책임 보유

**제안:**

| 역할 | Tester (Core) | Worker |
|------|---------------|--------|
| 단위 테스트 | 검토/보완 | **작성** |
| 통합 테스트 | **작성** | - |
| E2E 테스트 | **작성** | - |
| 테스트 실행 | **담당** | 로컬 확인만 |
| 커버리지 분석 | **담당** | - |

**Worker 프롬프트 수정:**
```
## 테스트 규칙
- 단위 테스트만 작성
- 통합/E2E 테스트는 Tester에게 위임
- 로컬에서 테스트 실행 후 빌드 확인
```

### 3.2 구조 설계 책임 분리

**현재 문제:** Architect와 Worker 모두 디렉토리 구조 정의

**제안:**

| 역할 | Architect | Worker |
|------|-----------|--------|
| 전체 아키텍처 | **결정** | 준수 |
| 레이어 구조 | **결정** | 준수 |
| 패키지/모듈 구조 | 가이드 제공 | **세부 구현** |
| 파일 네이밍 | 컨벤션 정의 | **준수** |

### 3.3 누락 에이전트 추가

#### Reviewer (신규)

```yaml
agent:
  id: reviewer
  name: Reviewer
  type: core
  workflow: [single, integrate, multi]

  description: |
    코드 리뷰를 담당하는 에이전트.
    Worker의 산출물을 검토하고 피드백을 제공합니다.

  responsibilities:
    - 코드 품질 검토
    - 컨벤션 준수 확인
    - 보안 이슈 식별
    - 성능 문제 식별
    - 개선 제안

  inputs:
    - port-output (Worker 산출물)
    - conventions

  outputs:
    - review-report
    - feedback-items

  completion:
    checklist:
      - 모든 변경 파일 검토
      - 컨벤션 준수 확인
      - 보안 이슈 확인
      - 피드백 작성
    required: true
```

#### Docs (신규)

```yaml
agent:
  id: docs
  name: Docs Writer
  type: core
  workflow: [single, integrate, multi]

  description: |
    문서화를 담당하는 에이전트.
    API 문서, 사용자 가이드, README를 작성합니다.

  responsibilities:
    - API 문서 작성
    - 사용자 가이드 작성
    - README 업데이트
    - 코드 주석 검토

  outputs:
    - docs/*.md
    - README.md
    - API 문서
```

---

## 4. 제안 3: 완료 체크리스트 표준화

### 4.1 Core 에이전트 체크리스트

#### Builder
```yaml
completion:
  checklist:
    - 요구사항 명확화 완료
    - 모든 포트 생성
    - 포트 간 의존성 정의
    - 포트 명세 검토 완료
    - 사용자 승인 획득
  required: true
```

#### Planner
```yaml
completion:
  checklist:
    - 모든 포트 파이프라인에 할당
    - 의존성 순서 검증
    - 에이전트 할당 완료
    - 파이프라인 생성
    - 사용자 승인 획득
  required: true
```

#### Architect
```yaml
completion:
  checklist:
    - 아키텍처 결정 문서화
    - 디렉토리 구조 정의
    - 기술 스택 결정
    - ADR 작성 (주요 결정)
    - 사용자 승인 획득
  required: true
```

#### Manager
```yaml
completion:
  checklist:
    - 포트 완료조건 충족 확인
    - 빌드 성공 확인
    - 테스트 통과 확인
    - 컨벤션 준수 확인
    - 에스컬레이션 처리 완료
  required: true
```

#### Tester
```yaml
completion:
  checklist:
    - 테스트 케이스 작성
    - 테스트 실행 완료
    - 커버리지 목표 달성
    - 실패 테스트 없음
    - 테스트 리포트 작성
  required: true
```

#### Logger
```yaml
completion:
  checklist:
    - 변경사항 분석 완료
    - 커밋 메시지 작성
    - CHANGELOG 업데이트
    - 사용자 승인 획득
  required: true
```

### 4.2 Worker 에이전트 체크리스트

```yaml
completion:
  checklist:
    - 포트 명세 요구사항 충족
    - 빌드 성공 (lint/compile)
    - 단위 테스트 작성 및 통과
    - 컨벤션 준수
    - 코드 주석 작성 (필요시)
  required: true
```

### 4.3 체크리스트 강제 메커니즘

```
┌──────────────────────────────────────────────┐
│                포트 완료 플로우               │
├──────────────────────────────────────────────┤
│                                              │
│  Worker 작업 완료                             │
│       ↓                                      │
│  체크리스트 검증 (자동)                        │
│       ↓                                      │
│  ┌─────────────────────────────────────┐     │
│  │ 빌드 성공?      [Y] → 다음          │     │
│  │                 [N] → 수정 요청      │     │
│  └─────────────────────────────────────┘     │
│       ↓                                      │
│  ┌─────────────────────────────────────┐     │
│  │ 테스트 통과?    [Y] → 다음          │     │
│  │                 [N] → 수정 요청      │     │
│  └─────────────────────────────────────┘     │
│       ↓                                      │
│  ┌─────────────────────────────────────┐     │
│  │ 컨벤션 준수?    [Y] → 완료          │     │
│  │                 [N] → 수정 요청      │     │
│  └─────────────────────────────────────┘     │
│       ↓                                      │
│  포트 상태: done                              │
│                                              │
└──────────────────────────────────────────────┘
```

**구현 방안:**
1. `pal hook port-end` 시 체크리스트 검증
2. 실패 시 에스컬레이션 또는 수정 요청
3. 성공 시에만 포트 상태 변경

---

## 5. 제안 4: 컨벤션 분리 구조화

### 5.1 디렉토리 구조

```
conventions/
├── agents/
│   ├── core/
│   │   ├── _common.md        # Core 공통 컨벤션
│   │   ├── builder.md
│   │   ├── planner.md
│   │   ├── architect.md
│   │   ├── manager.md
│   │   ├── tester.md
│   │   ├── logger.md
│   │   └── reviewer.md       # (신규)
│   └── workers/
│       ├── _common.md        # Worker 공통 컨벤션
│       ├── go.md
│       ├── kotlin.md
│       ├── nestjs.md
│       ├── react.md
│       └── next.md
└── project/
    ├── code-style.md         # 프로젝트 코드 스타일
    ├── git.md                # Git 컨벤션
    └── naming.md             # 네이밍 컨벤션
```

### 5.2 Core 공통 컨벤션 (_common.md)

```markdown
# Core 에이전트 공통 컨벤션

## 기본 원칙
1. 사용자 승인 우선: 주요 결정 전 사용자 확인
2. 단계별 진행: 한 번에 하나씩
3. 명확한 출력: 산출물 형식 준수
4. 에스컬레이션: 범위 초과 시 즉시 보고

## 역할 전환 규칙
- 현재 역할 작업 완료 후 전환
- 전환 전 사용자 알림
- 컨텍스트 전달 (이전 작업 요약)

## 에스컬레이션 기준
- 포트 범위 초과 작업 발생
- 기술적 결정 필요
- 의존성 충돌 발견
- 예상치 못한 복잡도

## 완료 체크리스트 규칙
- 모든 항목 체크 후에만 완료 선언
- 체크 불가 항목은 사유 명시
- 에스컬레이션 후 진행
```

### 5.3 Worker 공통 컨벤션 (_common.md)

```markdown
# Worker 에이전트 공통 컨벤션

## 기본 원칙
1. 포트 명세 준수: 범위 내 작업만
2. 컨벤션 우선: 기존 코드 스타일 따르기
3. 테스트 필수: 단위 테스트 작성
4. 빌드 확인: 작업 후 빌드 성공 확인

## 코드 품질 기준
- 린터 경고 없음
- 컴파일 에러 없음
- 테스트 통과
- 주석 작성 (public API)

## 작업 흐름
1. 포트 명세 읽기
2. 기존 코드 분석
3. 구현
4. 빌드 확인
5. 테스트 작성/실행
6. 완료 체크리스트 확인

## 에스컬레이션 기준
- 포트 범위 초과 작업 필요
- 아키텍처 변경 필요
- 다른 포트 의존성 발견
- 기술적 결정 필요
```

### 5.4 기술별 컨벤션 로딩 순서

```
Worker 프롬프트 구성:

1. conventions/agents/workers/_common.md    (Worker 공통)
2. conventions/agents/workers/{tech}.md     (기술 특화)
3. conventions/project/*.md                 (프로젝트 규칙)
4. agent prompt                            (에이전트 프롬프트)
```

---

## 6. 구현 우선순위

### Phase 1: 기반 구축 (agent-convention-split)

| 순서 | 작업 | 설명 |
|------|------|------|
| 1-1 | 컨벤션 디렉토리 구조 생성 | conventions/agents/ 생성 |
| 1-2 | Core 공통 컨벤션 작성 | _common.md |
| 1-3 | Worker 공통 컨벤션 작성 | _common.md |
| 1-4 | 기술별 컨벤션 분리 | go.md, kotlin.md, etc. |

### Phase 2: 에이전트 명세 업데이트

| 순서 | 작업 | 설명 |
|------|------|------|
| 2-1 | 표준 스키마 정의 | agent-schema.yaml |
| 2-2 | Core 에이전트 명세 업데이트 | 7개 |
| 2-3 | Worker 에이전트 명세 업데이트 | 5개 |
| 2-4 | 신규 에이전트 추가 | reviewer, docs |

### Phase 3: 메커니즘 구현

| 순서 | 작업 | 설명 |
|------|------|------|
| 3-1 | 컨벤션 로딩 로직 | agent.go 수정 |
| 3-2 | 체크리스트 검증 로직 | hook.go 수정 |
| 3-3 | 에스컬레이션 자동화 | escalation.go |

---

## 7. 기대 효과

| 영역 | 현재 | 개선 후 |
|------|------|---------|
| 완료 보장 | 체크리스트 없음 | 강제 체크리스트 |
| 역할 명확성 | 중복/누락 | 명확한 분리 |
| 컨벤션 적용 | 프롬프트 내 산재 | 분리/로딩 자동화 |
| 품질 게이트 | integrate만 | 모든 워크플로우 |

---

## 8. 다음 단계

이 제안서의 내용은 `agent-convention-split` 포트에서 구현됩니다.

---

<!-- pal:port:agent-spec-review -->
