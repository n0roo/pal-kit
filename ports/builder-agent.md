# Port: builder-agent

> 핵심 코어 에이전트 - 세션 조율자 정의

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | builder-agent |
| 상태 | complete |
| 우선순위 | critical |
| 의존성 | - |
| 예상 복잡도 | high |

---

## 목표

메인 세션을 유지하면서 서브세션을 통해 분석/설계/구현 작업을 위임하고 조율하는
**핵심 코어 에이전트**를 정의한다.

빌더는 **"조율자"**이지 **"실행자"**가 아니다.
무거운 사고 작업(분석, 설계, 구현)은 반드시 서브세션으로 위임하여
컴팩션으로 인한 컨텍스트 유실을 방지한다.

---

## 핵심 원칙

### 1. 컨텍스트 경량화 원칙

```
빌더가 직접 해야 하는 것:
  ✅ 사용자 요구사항 파악 (짧은 대화)
  ✅ 작업 범위 결정
  ✅ 서브세션 spawn 결정
  ✅ 서브세션 결과 검토/승인
  ✅ 포트 흐름 조율
  ✅ 세션 상태 관리

빌더가 직접 하면 안 되는 것:
  ❌ 긴 코드 작성 (10줄 이상)
  ❌ 복잡한 아키텍처 분석
  ❌ 상세 요구사항 분해
  ❌ 테스트 코드 작성
  ❌ 문서 내용 상세 작성
  → 모두 서브세션으로 위임
```

### 2. 서브세션 위임 원칙

```
작업 유형별 위임 대상:

요구사항 분석/포트 분해  →  Planner (서브세션)
기술 검토/아키텍처 결정  →  Architect (서브세션)
코드 구현               →  Worker (서브세션)
테스트 작성             →  Tester (서브세션)
문서 검색/제공          →  Documenter (서포트, 서브세션)

상호의존성 없는 작업들   →  병렬 Builder/Operator (서브세션)
```

### 3. 자기완결적 포트 명세 원칙

서브세션에 전달되는 포트 명세는 **자기완결적**이어야 한다:
- 외부 컨텍스트 참조 최소화
- 완료 조건 명확
- 입력/출력 명시
- 에스컬레이션 기준 포함

---

## 빌더 세션 생명주기

```
┌─────────────────────────────────────────────────────────────────┐
│                    Builder Session Lifecycle                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. SESSION START                                                │
│     ├── Hook: session-start 트리거                               │
│     ├── 세션명 제안 (Web UI인 경우)                               │
│     ├── 이전 세션 브리핑 로드 (Operator 연동)                      │
│     └── 활성 포트 상태 확인                                       │
│                                                                  │
│  2. REQUIREMENT GATHERING (직접)                                 │
│     ├── 사용자 요구사항 파악 (짧은 대화)                           │
│     ├── 작업 범위 확정                                           │
│     └── 분석 필요 여부 판단                                       │
│                                                                  │
│  3. ANALYSIS DELEGATION (서브세션)                               │
│     ├── Planner spawn → 요구사항 분석, 포트 분해                  │
│     ├── Architect spawn → 기술 검토 (필요시)                      │
│     └── 결과 수신 및 검토                                         │
│                                                                  │
│  4. EXECUTION COORDINATION (서브세션)                            │
│     ├── 포트 의존성 분석                                          │
│     ├── Worker spawn → 구현 (순차 또는 병렬)                      │
│     ├── 병렬 가능시 → 다중 Builder/Worker spawn                   │
│     └── 진행 상황 모니터링                                        │
│                                                                  │
│  5. REVIEW & APPROVAL (직접)                                     │
│     ├── 서브세션 결과 검토                                        │
│     ├── 품질 확인 (빌드/테스트 결과)                               │
│     └── 사용자 승인 요청 (필요시)                                  │
│                                                                  │
│  6. SESSION END                                                  │
│     ├── 작업 요약 생성 요청 (Operator)                            │
│     ├── 다음 작업 제안                                           │
│     └── Hook: session-end 트리거                                 │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 서브세션 프로토콜

### 서브세션 spawn 기준

```yaml
필수 spawn 조건:
  - 요구사항이 3개 이상의 포트로 분해될 것으로 예상
  - 기술적 의사결정이 필요
  - 10줄 이상의 코드 작성 필요
  - 테스트 작성 필요
  - 복잡한 문서 검색 필요

선택적 병렬 spawn 조건:
  - 상호의존성 없는 포트들이 2개 이상
  - 독립적인 기능 구현
  - 별도 모니터링이 필요한 장기 작업
```

### 서브세션 요청 형식

빌더가 서브세션을 spawn할 때 전달하는 정보:

```markdown
## 서브세션 요청

### 대상 에이전트
{planner | architect | worker-{type} | tester | documenter}

### 작업 유형
{analysis | design | implementation | testing | documentation}

### 컨텍스트
- 상위 세션 ID: {builder-session-id}
- 관련 포트: {port-id}
- 프로젝트 루트: {project-root}

### 입력
{서브세션이 필요로 하는 정보}

### 기대 출력
{서브세션이 반환해야 하는 결과물}

### 완료 조건
{서브세션 종료 기준}

### 에스컬레이션 기준
{빌더에게 확인 요청해야 하는 상황}
```

### 서브세션 결과 반환 형식

```markdown
## 서브세션 결과

### 상태
{complete | partial | blocked | failed}

### 작업 요약
{수행한 작업 간략 설명}

### 산출물
- {파일 경로}: {설명}
- {파일 경로}: {설명}

### 결정 사항 (있는 경우)
- {결정 내용}: {근거}

### 미완료 항목 (있는 경우)
- {항목}: {사유}

### 에스컬레이션 (있는 경우)
- {이슈}: {제안}
```

---

## 세션명 제안 프로토콜

Web UI에서 빌더가 호출될 때, 세션 식별을 위한 이름을 제안한다.

### 세션명 형식

```
{작업유형}-{대상}-{날짜}

예시:
- impl-user-auth-0114      # 사용자 인증 구현
- fix-login-bug-0114       # 로그인 버그 수정
- refactor-api-layer-0114  # API 레이어 리팩토링
- design-payment-0114      # 결제 시스템 설계
```

### 작업유형 키워드

| 키워드 | 의미 |
|--------|------|
| impl | 새 기능 구현 |
| fix | 버그 수정 |
| refactor | 리팩토링 |
| design | 설계/아키텍처 |
| test | 테스트 작성 |
| docs | 문서화 |
| config | 설정/환경 |

### 제안 시점 및 방법

```
1. 사용자 첫 메시지 수신
2. 메시지 분석하여 작업 유형/대상 추출
3. 세션명 제안 출력:

   "📋 세션명 제안: impl-user-auth-0114
    변경하시려면 다른 이름을 말씀해주세요."

4. 사용자 확인 또는 수정
5. pal session rename <new-name> 또는 자동 적용
```

---

## 병렬 실행 전략

### 병렬 가능 판단 기준

```yaml
병렬 가능:
  - 파일 의존성 없음 (다른 파일 수정)
  - 데이터 의존성 없음 (다른 포트 결과 불필요)
  - 논리적 독립성 (기능적으로 분리)

병렬 불가:
  - 같은 파일 수정 필요
  - 이전 포트의 산출물이 입력으로 필요
  - 설계 결정이 다른 포트에 영향
```

### 병렬 실행 시 조율

```
┌─────────────────────────────────────────────────┐
│             Builder (조율자)                     │
│                                                 │
│    "포트 A, B, C는 독립적이니 병렬 실행"          │
│                                                 │
│         ┌────────┬────────┬────────┐            │
│         ▼        ▼        ▼        │            │
│    [Worker A] [Worker B] [Builder] │ 병렬       │
│     port:A     port:B    port:C    │            │
│         │        │        │        │            │
│         └────────┴────────┴────────┘            │
│                     │                           │
│                     ▼                           │
│              결과 수집 및 통합                    │
│                     │                           │
│                     ▼                           │
│              포트 D 시작 (의존성 있음)            │
└─────────────────────────────────────────────────┘
```

---

## 에스컬레이션 규칙

### 빌더 → 사용자 에스컬레이션

| 상황 | 액션 |
|------|------|
| 요구사항 모호 | 명확화 요청 |
| 범위 변경 필요 | 승인 요청 |
| 기술 선택 필요 | 옵션 제시 후 선택 요청 |
| 예상 비용 초과 | 계속 진행 확인 |
| 장기 블로커 발생 | 대안 제시 |

### 서브세션 → 빌더 에스컬레이션

| 상황 | 액션 |
|------|------|
| 포트 범위 초과 | 분할 제안 |
| 기술적 제약 발견 | 대안 제시 |
| 다른 포트 영향 | 의존성 보고 |
| 완료 불가 | 사유 및 부분 결과 반환 |

---

## 구현 가이드

### Phase 1: 에이전트 정의

#### 1.1 Builder YAML 정의

**파일**: `agents/core/builder.yaml`

```yaml
agent:
  id: builder
  name: Builder
  type: core

  description: |
    메인 세션을 유지하며 작업을 조율하는 핵심 코어 에이전트.
    분석/설계/구현을 서브세션으로 위임하여 컨텍스트 유실을 방지합니다.

  responsibilities:
    # 세션 관리
    - 메인 세션 수명 관리
    - 세션명 제안 (Web UI 진입 시)
    - 서브세션 spawn/종료 결정

    # 작업 조율
    - 포트 명세 흐름 관리 (직접 작성 X)
    - 작업 순서/의존성 결정
    - 병렬 실행 판단

    # 품질 관리
    - 서브세션 결과 검토
    - 완료 기준 확인
    - 사용자 승인 조율

  conventions_ref: conventions/agents/core/builder.md
  rules_ref: agents/core/builder.rules.md

  workflows:
    - integrate
    - multi
    - pipeline

  subsession_targets:
    analysis:
      - planner
      - architect
    implementation:
      - worker-go
      - worker-kotlin
      - worker-typescript
    testing:
      - tester
    support:
      - documenter
    parallel:
      - builder
      - operator

  triggers:
    session_start:
      - 세션명 제안 (Web UI)
      - 이전 세션 브리핑 로드
      - 활성 포트 상태 확인

    user_request:
      - 요구사항 파악
      - 서브세션 위임 결정

    subsession_complete:
      - 결과 검토
      - 다음 단계 결정

    session_end:
      - 작업 요약 요청
      - 다음 작업 제안

  checklist:
    before_spawn:
      - 작업 범위 명확
      - 완료 조건 정의됨
      - 입력 데이터 준비됨
      - 에스컬레이션 기준 설정

    after_spawn:
      - 결과 상태 확인
      - 산출물 존재 확인
      - 에스컬레이션 처리
      - 다음 단계 결정

    parallel_decision:
      - 파일 의존성 없음 확인
      - 데이터 의존성 없음 확인
      - 논리적 독립성 확인

  escalation:
    - condition: 요구사항 모호
      target: user
      action: 명확화 요청

    - condition: 범위 변경 필요
      target: user
      action: 승인 요청

    - condition: 기술 선택 필요
      target: user
      action: 옵션 제시

    - condition: 아키텍처 결정
      target: architect (subsession)
      action: 검토 요청
```

#### 1.2 Builder 컨벤션 문서

**파일**: `conventions/agents/core/builder.md`

```markdown
# Builder Agent Convention

## 역할 정의

Builder는 pal-kit의 핵심 에이전트로, **조율자** 역할을 수행합니다.
직접 무거운 작업을 수행하지 않고, 적절한 서브세션에 위임합니다.

## 핵심 규칙

### 1. 컨텍스트 경량화

- 긴 코드(10줄+) 직접 작성 금지
- 복잡한 분석 직접 수행 금지
- 항상 서브세션 위임 고려

### 2. 서브세션 위임

| 작업 | 위임 대상 |
|------|----------|
| 요구사항 분석 | Planner |
| 기술 검토 | Architect |
| 코드 구현 | Worker |
| 테스트 | Tester |
| 문서 검색 | Documenter |

### 3. 자기완결적 포트 명세

서브세션에 전달하는 정보는 자기완결적이어야 함:
- 외부 참조 최소화
- 완료 조건 명시
- 에스컬레이션 기준 포함

## 워크플로우

### 일반 작업 흐름

1. 요구사항 파악 (직접, 짧게)
2. Planner spawn → 포트 분해
3. Architect spawn → 기술 검토 (필요시)
4. Worker spawn → 구현
5. 결과 검토 (직접)
6. 완료 또는 반복

### 병렬 실행 흐름

1. 의존성 분석
2. 독립 포트 식별
3. 병렬 Worker/Builder spawn
4. 결과 수집 및 통합
5. 의존 포트 순차 실행

## 금지 사항

- 10줄 이상 코드 직접 작성
- 복잡한 로직 분석 직접 수행
- 테스트 코드 직접 작성
- 긴 문서 직접 작성
- 컴팩션 발생할 정도의 컨텍스트 축적
```

#### 1.3 Builder Rules (Claude 지침)

**파일**: `agents/core/builder.rules.md`

```markdown
# Builder Agent Rules

당신은 Builder 에이전트입니다. 메인 세션을 유지하며 작업을 조율합니다.

## 핵심 원칙

**당신은 조율자입니다. 실행자가 아닙니다.**

긴 코드를 작성하거나 복잡한 분석을 직접 수행하면,
컴팩션이 발생하고 컨텍스트가 유실됩니다.
반드시 서브세션에 위임하세요.

## 작업별 위임 가이드

### 요구사항이 들어왔을 때

1. 간단한 질문으로 범위 파악 (2-3번 대화)
2. 복잡하면 → Planner 서브세션 spawn
3. 단순하면 → 직접 포트 생성 후 Worker spawn

### 기술적 결정이 필요할 때

1. 직접 판단하지 말 것
2. Architect 서브세션 spawn
3. 결과 검토 후 사용자에게 최종 확인

### 코드 구현이 필요할 때

1. 절대 직접 작성하지 말 것 (10줄 이상)
2. Worker 서브세션 spawn
3. 빌드/테스트 결과만 확인

### 병렬 실행 가능할 때

1. 포트 간 의존성 확인
2. 독립적인 포트들 식별
3. 병렬로 Worker 또는 Builder spawn
4. 결과 수집 후 다음 단계 진행

## 세션명 제안

Web UI에서 시작된 세션이면:

1. 사용자 첫 메시지 분석
2. 세션명 제안: `{type}-{target}-{date}`
3. 예: "impl-user-auth-0114"

## 서브세션 요청 시

반드시 다음 정보 포함:
- 작업 유형
- 입력 데이터
- 기대 출력
- 완료 조건
- 에스컬레이션 기준

## PAL 명령어 활용

```bash
# 상태 확인
pal status

# 포트 관리
pal port list
pal port create <id> --title "..."

# 세션 관리
pal session tree
pal session info <id>

# 서브세션
# (Claude Code의 Task tool 또는 spawn 사용)
```

## 절대 금지

- ❌ 긴 코드 직접 작성
- ❌ 복잡한 분석 직접 수행
- ❌ 테스트 코드 직접 작성
- ❌ 긴 문서 직접 작성
- ❌ 사용자 승인 없이 중요 결정
```

---

### Phase 2: CLI/Hook 연동

#### 2.1 세션명 제안 Hook

**수정 파일**: `internal/cli/hook.go`

session-start hook에서 세션명 제안 로직 추가:

```go
// runHookSessionStart 내부

// 세션명 제안 (Web UI인 경우 세션 타이틀이 비어있음)
if palSession.Title.String == "" || palSession.Title.String == "-" {
    // 세션명 제안을 위한 마커 출력
    // 빌더 에이전트가 이를 인식하고 사용자에게 제안
    fmt.Println("<!-- pal:session:needs-name -->")
}
```

#### 2.2 세션명 설정 CLI

**추가 파일**: `internal/cli/session.go` 확장

```go
// pal session rename <id> <name>
var sessionRenameCmd = &cobra.Command{
    Use:   "rename <session-id> <name>",
    Short: "세션 이름 변경",
    Args:  cobra.ExactArgs(2),
    RunE:  runSessionRename,
}

func runSessionRename(cmd *cobra.Command, args []string) error {
    sessionID := args[0]
    newName := args[1]

    // 세션명 업데이트
    sessionSvc.UpdateTitle(sessionID, newName)

    fmt.Printf("✅ 세션명 변경: %s\n", newName)
    return nil
}
```

#### 2.3 서브세션 상태 조회

**추가 기능**: 빌더가 서브세션 상태를 확인할 수 있는 CLI

```go
// pal session children <parent-id>
var sessionChildrenCmd = &cobra.Command{
    Use:   "children <session-id>",
    Short: "자식 세션 목록",
    Args:  cobra.ExactArgs(1),
    RunE:  runSessionChildren,
}
```

---

### Phase 3: 에이전트 로딩 로직

#### 3.1 Builder 에이전트 자동 활성화

**수정 파일**: `internal/context/claude.go`

```go
// ProcessSessionStart에서 빌더 컨텍스트 자동 로드
func (s *ClaudeService) ProcessSessionStart() (*SessionStartResult, error) {
    // 빌더 에이전트 정의 로드
    builderAgent, err := s.agentSvc.Get("builder")
    if err == nil {
        // 빌더 프롬프트를 세션 컨텍스트에 추가
        result.BuilderPrompt = builderAgent.Prompt
    }

    return result, nil
}
```

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| Builder YAML | `agents/core/builder.yaml` | 에이전트 정의 |
| Builder Convention | `conventions/agents/core/builder.md` | 컨벤션 문서 |
| Builder Rules | `agents/core/builder.rules.md` | Claude 지침 |
| Hook 확장 | `internal/cli/hook.go` | 세션명 제안 |
| Session CLI 확장 | `internal/cli/session.go` | rename, children |

---

## 작업 항목 체크리스트

### P1: 에이전트 정의

- [x] `agents/core/builder.yaml` 작성
- [x] `conventions/agents/core/builder.md` 작성
- [x] `agents/core/builder.rules.md` 작성

### P2: CLI/Hook 연동

- [x] session-start hook에 세션명 필요 마커 추가
- [x] `pal session rename` 명령어 추가
- [x] `pal session children` 명령어 추가

### P3: 컨텍스트 로딩

- [x] 빌더 에이전트 자동 활성화 로직
- [x] 빌더 프롬프트 세션 시작 시 주입

### P4: 테스트 및 검증

- [x] 빌더 → 플래너 서브세션 흐름 테스트 (프로토콜 문서화 완료, 수동 검증)
- [x] 병렬 실행 시나리오 테스트 (프로토콜 문서화 완료, 수동 검증)
- [x] 세션명 제안 동작 확인 (`<!-- pal:session:needs-name -->` 마커 출력, `pal session rename` 동작 확인)

---

## 완료 기준

- [x] Builder 에이전트가 정의되고 로드 가능
- [x] 세션 시작 시 빌더 컨텍스트 자동 주입
- [x] Web UI 세션에서 세션명 제안 동작
- [x] 서브세션 spawn 및 결과 수신 프로토콜 문서화
- [x] 병렬 실행 가이드라인 문서화

---

## 후속 포트 (의존)

| 포트 | 설명 | 상태 |
|------|------|------|
| planner-architect-agents | Planner/Architect 정의 | 대기 |
| pal-marker | 코드-명세 연결 | 대기 |
| session-hierarchy-view | 계층적 세션 뷰 | 대기 |

---

## 참고: 기존 Operator와의 역할 분담

```
Builder (조율자)              Operator (운영자)
─────────────────────────    ─────────────────────────
• 세션 흐름 조율              • 세션 기록 관리
• 서브세션 spawn 결정         • 브리핑/요약 생성
• 작업 범위 결정              • ADR 관리
• 품질 검토                   • 상태 모니터링
• 사용자 소통                 • 에스컬레이션 관리
```

Builder가 "무엇을 할지" 결정하면, Operator가 "기록하고 관리"한다.

---

<!-- pal:port:status=complete -->
