# pal-kit

> PAL Kit CLI 도구 프로젝트 | Go 기반

---

## 프로젝트 개요

PAL Kit은 Claude Code와 함께 사용하는 프로젝트 관리 CLI 도구입니다.
포트 기반 작업 관리, 에이전트 시스템, 파이프라인 워크플로우를 제공합니다.

---

## 기술 스택

- **언어**: Go
- **구조**: cmd/, internal/ 기반 표준 Go 레이아웃

---

## PAL Kit 통합 가이드

### 세션 시작 시 필수 체크

**매 세션 시작 시 아래 순서로 상태를 확인합니다:**

```bash
# 1. 전체 상태 확인
pal status

# 2. 활성/대기 포트 확인
pal port list

# 3. 이전 세션 기록 확인 (있는 경우)
ls -la .pal/sessions/
```

**확인 후 행동:**
- 활성 포트가 있으면 → 해당 포트 작업 우선
- `.claude/rules/{port-id}.md` 있으면 → 해당 rule 참조
- 블로커가 있으면 → 해결 또는 에스컬레이션

---

### 서브에이전트(Task tool) 활용 패턴

**복잡한 작업은 Task tool로 서브에이전트에 위임합니다.**

| 작업 유형 | 서브에이전트 | Task tool 사용 |
|----------|-------------|---------------|
| 새 기능 구현 | Explore → Plan → 구현 | ✅ 적극 사용 |
| 코드 탐색 | Explore agent | ✅ 사용 |
| 아키텍처 분석 | Plan agent | ✅ 사용 |
| 단순 수정 | 직접 수행 | ❌ 불필요 |

**서브에이전트 활성화 예시:**

```
# 1. 복잡한 기능 구현 시
User: "주문 기능 구현해줘"
Claude:
  1. Task(Explore) → 기존 코드 구조 파악
  2. Task(Plan) → 구현 계획 수립
  3. 직접 구현 또는 추가 Task 분할

# 2. 코드 분석 시
User: "에러 핸들링 어떻게 되어있어?"
Claude:
  1. Task(Explore) → 에러 핸들링 패턴 탐색
  2. 결과 요약 제공
```

---

### 포트 기반 작업 흐름

**포트(Port)는 작업의 기본 단위입니다.**

```
1. 포트 확인
   pal port list

2. 포트 시작 (Rule 자동 생성)
   pal hook port-start <port-id>
   → .claude/rules/{port-id}.md 생성됨

3. 작업 진행
   - Rule 파일 참조하며 작업
   - 완료 체크리스트 확인

4. 포트 완료
   pal hook port-end <port-id>
   → Rule 파일 정리됨
```

**포트 명세 구조 (ports/*.md):**
```markdown
# {포트 ID}
## 목표: {달성할 목표}
## 범위: {포함/제외 항목}
## 완료 조건: {체크리스트}
## 의존성: {선행 포트}
```

---

### 동적 Rule 활용

**포트 시작 시 `.claude/rules/` 에 Rule이 자동 생성됩니다.**

```
.claude/rules/
└── {port-id}.md    # 포트별 작업 지침
```

**Rule 파일 포함 내용:**
- 포트 목표 및 범위
- 적용할 워커/컨벤션
- 완료 체크리스트
- 에스컬레이션 기준

**Claude는 활성 Rule을 우선 참조하여 작업합니다.**

---

### Core vs Worker 에이전트

| 구분 | Core 에이전트 | Worker 에이전트 |
|------|-------------|----------------|
| 역할 | 조율/관리 | 실제 구현 |
| 기술 종속 | 없음 | 있음 (Kotlin, Go 등) |
| 예시 | builder, planner, operator | entity-worker, router-worker |

**에이전트 선택 기준:**
- 계획/설계 → Core (planner, architect)
- 코드 구현 → Worker (entity, service, router)
- 상태 관리 → Core (operator)
- 테스트 → Core (tester) + Worker (test-worker)

---

### 작업 완료 시 기록

**중요한 작업 완료 시 기록을 남깁니다:**

```bash
# 세션 요약 저장 위치
.pal/sessions/{date}-{session-id}.md

# 아키텍처 결정 기록 (ADR)
.pal/decisions/ADR-{번호}-{제목}.md
```

**ADR 생성 기준:**
- 아키텍처 변경
- 기술 스택 선택
- 중요한 설계 결정
- 트레이드오프 선택

---

## 워크플로우

**integrate** - 빌더 관리, 서브세션 방식

복잡한 기능 개발과 여러 기술 스택을 다루는 작업에 적합합니다.

---

## 에이전트

### Core 에이전트

| 에이전트 | 역할 | 활용 시점 |
|---------|------|----------|
| builder | 요구사항 분석, 포트 분해 | 새 기능 시작 |
| planner | 작업 계획, 우선순위 | 복잡한 작업 계획 |
| architect | 설계 검토, 의존성 검증 | 아키텍처 결정 |
| operator | 운영/연속성 관리 | 세션 시작/종료 |
| tester | 품질 검증, TC 관리 | 테스트 단계 |

### Worker 에이전트

| 에이전트 | 기술 스택 | 담당 영역 |
|---------|----------|----------|
| entity-worker | Kotlin/JPA | L1 Entity, Repository |
| cache-worker | Redis | L1 Cache |
| document-worker | MongoDB | L1 Document |
| service-worker | Kotlin/Spring | LM/L2 Service |
| router-worker | Kotlin/Spring MVC | L3 Controller |
| test-worker | JUnit/MockK | 테스트 보완 |
| worker-go | Go | Go 코드 작성 |

---

## PAL Kit 명령어

```bash
# 상태 확인
pal status

# 포트 관리
pal port list
pal port create <id> --title "작업명"
pal port status <id>

# 작업 시작/종료 (Rule 자동 생성/정리)
pal hook port-start <id>
pal hook port-end <id>

# 세션 관리
pal session list
pal session summary

# 파이프라인
pal pipeline list
pal pl plan <n>

# Manifest 관리
pal manifest status
pal manifest sync

# 대시보드
pal serve
```

---

## 디렉토리 구조

```
.
├── CLAUDE.md           # 프로젝트 컨텍스트 (이 파일)
├── cmd/                # CLI 진입점
├── internal/           # 내부 패키지
├── docs/               # 문서
│   ├── ARCHITECTURE.md # 아키텍처 설명
│   └── PACKAGE-GUIDE.md# 패키지 가이드
├── agents/             # 에이전트 정의
│   ├── core/           # Core 에이전트 YAML
│   └── workers/        # Worker 에이전트 YAML
├── ports/              # 포트 명세
├── conventions/        # 컨벤션 문서
│   └── agents/         # 에이전트별 컨벤션
├── packages/           # 패키지 정의
├── .claude/
│   ├── settings.json   # Claude Code Hook 설정
│   └── rules/          # 동적 Rule (포트별)
└── .pal/
    ├── config.yaml     # PAL Kit 설정
    ├── manifest.yaml   # 파일 추적
    ├── sessions/       # 세션 기록
    ├── decisions/      # ADR
    └── context/        # 현재 상태
```

---

<!-- pal:config:status=configured -->
<!--
  PAL Kit 설정 상태: 완료
  워크플로우: integrate
  설정일: 2026-01-12
-->


<!-- pal:active-worker:start -->
<!-- pal:active-worker:end -->

<!-- pal:context:start -->
> 마지막 업데이트: 2026-01-14 03:11:13

### 활성 세션
- **0e7b2795**: impl-p4-test-0114
- **a4236f8c**: -
- **c777173a**: -
- **f5c32ae6**: -
- **f974a6ea**: -

### 포트 현황
- ✅ complete: 12

### 에스컬레이션
- 없음

<!-- pal:context:end -->
