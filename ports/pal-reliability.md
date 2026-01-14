# Port: pal-reliability

> PAL Kit 핵심 기능 안정화 및 사용성 개선

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | pal-reliability |
| 상태 | draft |
| 우선순위 | critical |
| 의존성 | - |
| 예상 복잡도 | high |

---

## 배경

사용자 피드백에 따른 핵심 문제점:

1. **`pal init` 동기화 불완전** - agents, conventions, rules.md가 전달/갱신되지 않음
2. **`pal version` 없음** - 버전 확인 불가
3. **워크플로우 적용 안됨** - `.claude` 워크플로우가 디폴트 상태로 수정되지 않음
4. **세션 식별 불가** - 복수 터미널에서 자기 세션 특정 못함
5. **포트 기록 누락** - 거의 모든 작업에서 포트 명세가 기록되지 않음
6. **docs 구조 미동작** - 문서 구조 설정 동작 안함
7. **사용법 강제 없음** - Claude가 PAL Kit 사용법을 강제받지 않음

**핵심 요구사항:**
- 세션/포트 기록이 **반드시** 동작해야 함
- 지식베이스가 쌓이는 경험 필요
- 작업 전 포트 기록 여부 확인 강제

---

## 목표

1. PAL Kit의 핵심 기능이 안정적으로 동작하도록 개선
2. 세션 및 포트 추적의 정확성과 완전성 보장
3. Claude에게 PAL Kit 사용법을 강제하는 메커니즘 구축

---

## 작업 항목

### P1: 세션 식별 강화 (Critical)

**문제:** 복수 터미널에서 세션 구분 불가

**해결:**
- [ ] 세션 테이블에 `cwd` (실행 위치) 컬럼 활용 강화
- [ ] `session_type` 필드로 메인/서브 세션 명확히 구분
  - `main`: 사용자가 직접 시작한 세션
  - `sub`: Builder가 spawn한 서브 세션
  - `builder`: 빌더 에이전트 세션
- [ ] 세션 시작 시 `cwd` 기반으로 중복 세션 감지
- [ ] `FindActiveSession` 함수 개선: cwd + project_root 조합으로 정확한 세션 식별

```go
// session.go 개선
type Session struct {
    // 기존 필드...
    Cwd           sql.NullString  // 실행 위치 (이미 있음, 활용 강화)
    SessionType   string          // main, sub, builder
    MainSessionID sql.NullString  // 서브세션인 경우 메인 세션 참조
}
```

### P2: 포트 기록 강제화 (Critical)

**문제:** 포트 명세 기록이 안됨

**해결:**
- [ ] SessionStart Hook에서 포트 기록 안내 메시지 출력
- [ ] 이벤트 로깅 강화: 포트 없이도 요구사항 이벤트 기록
- [ ] PreToolUse Hook에서 Edit/Write 시 활성 포트 확인
- [ ] 활성 포트 없으면 경고 메시지로 기록 유도

```go
// hook.go - PreToolUse 개선
func runHookPreToolUse(cmd *cobra.Command, args []string) error {
    // Edit/Write 도구 사용 시
    if input.ToolName == "Edit" || input.ToolName == "Write" {
        // 활성 포트 확인
        runningPorts, _ := portSvc.List("running", 1)
        if len(runningPorts) == 0 {
            // stderr로 Claude에 피드백
            fmt.Fprintln(os.Stderr, "⚠️ 활성 포트가 없습니다. `pal hook port-start <id>` 또는 새 포트를 생성하세요.")
            // 이벤트 로깅
            sessionSvc.LogEvent(sessionID, "untracked_edit", ...)
        }
    }
}
```

### P3: 이벤트 로깅 고도화 (High)

**문제:** 포트 없이는 로그가 남지 않음

**해결:**
- [ ] `user_request` 이벤트 타입 추가 (사용자 요구사항 입력)
- [ ] `untracked_work` 이벤트 타입 추가 (포트 없는 작업)
- [ ] SessionStart에서 첫 번째 사용자 메시지 캡처 (transcript에서)
- [ ] 모든 주요 작업에 이벤트 로깅

```sql
-- 이벤트 타입 확장
-- session_start, session_end, port_start, port_end (기존)
-- user_request: 사용자 요구사항 입력
-- untracked_work: 포트 없는 작업 감지
-- escalation: 에스컬레이션 발생
-- decision: 주요 결정 사항
```

### P4: `pal version` 명령어 (Medium)

**문제:** 버전 확인 불가

**해결:**
- [ ] `cmd/pal/main.go`에 버전 정보 추가
- [ ] `pal version` 명령어 구현
- [ ] 빌드 시 버전/커밋 정보 주입

```go
// cmd/pal/main.go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)

// internal/cli/version.go
var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "버전 정보 출력",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("PAL Kit %s\n", Version)
        fmt.Printf("  Commit: %s\n", Commit)
        fmt.Printf("  Built:  %s\n", BuildDate)
    },
}
```

### P5: `pal init` 템플릿 동기화 (High)

**문제:** agents, conventions, rules.md가 복사/갱신되지 않음

**해결:**
- [ ] 템플릿 구조 정리: `internal/agent/templates/` 재구성
  - `agents/core/*.yaml` + `*.rules.md`
  - `agents/workers/**/*.yaml` + `*.rules.md`
  - `conventions/agents/**/*.md`
- [ ] `InstallTemplates` 함수 개선: 모든 파일 복사
- [ ] `pal init --templates-force` 옵션으로 덮어쓰기
- [ ] 변경 감지 및 갱신 로직 추가

```go
// embed.go 구조 개선
//go:embed templates/agents/* templates/conventions/*
var templateFS embed.FS
```

### P6: 워크플로우 강제 적용 (Medium)

**문제:** `.claude` 워크플로우가 적용되지 않음

**해결:**
- [ ] SessionStart에서 워크플로우 rules 확실히 작성
- [ ] 워크플로우 타입에 따른 기본 지시사항 포함
- [ ] `.claude/rules/workflow.md` 파일로 강제 주입

### P7: Claude 사용법 강제 (High)

**문제:** Claude가 PAL Kit 사용법을 강제받지 않음

**해결:**
- [ ] SessionStart Hook에서 필수 지시사항 출력
- [ ] PreToolUse에서 파일 수정 시 경고/차단
- [ ] 워크플로우 rules에 PAL Kit 사용 지침 포함

```markdown
<!-- .claude/rules/pal-usage.md -->
# PAL Kit 필수 사용 규칙

## 작업 시작 전
1. 포트 명세가 있는지 확인
2. 없으면 `pal port create <id>` 또는 기존 포트 활성화
3. `pal hook port-start <id>` 실행

## 코드 수정 전
⚠️ 활성 포트 없이 코드를 수정하면 추적되지 않습니다.
- 포트를 먼저 활성화하세요

## 작업 완료 후
1. `pal hook port-end <id>` 실행
2. 변경 사항 커밋
```

### P8: docs 구조 설정 동작 (Medium)

**문제:** 문서 구조 설정이 동작하지 않음

**해결:**
- [ ] `pal init`에서 docs 디렉토리 구조 생성
- [ ] Manifest에 docs 인덱싱 트리거
- [ ] 문서 변경 시 자동 인덱싱

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| version 명령어 | internal/cli/version.go | 버전 출력 |
| 개선된 세션 서비스 | internal/session/session.go | 세션 식별 강화 |
| 개선된 Hook | internal/cli/hook.go | 포트 강제, 이벤트 로깅 |
| 개선된 템플릿 | internal/agent/templates/ | 전체 구조 포함 |
| PAL 사용 규칙 | 템플릿 rules | Claude 강제 지침 |

---

## 완료 기준

- [ ] `pal version` 명령어 동작
- [ ] `pal init`에서 모든 agents, conventions, rules.md 복사
- [ ] 복수 터미널에서 각 세션이 정확히 식별됨
- [ ] 메인/서브 세션 구분 가능
- [ ] 포트 없는 코드 수정 시 경고 메시지 출력
- [ ] 모든 사용자 요구사항이 이벤트로 로깅됨
- [ ] 워크플로우 rules가 세션 시작 시 적용됨
- [ ] Claude가 PAL Kit 사용법을 인지하고 따름

---

## 구현 우선순위

```
1. P2: 포트 기록 강제화      ← 가장 긴급
2. P1: 세션 식별 강화        ← 기반 인프라
3. P3: 이벤트 로깅 고도화    ← 추적성
4. P7: Claude 사용법 강제    ← 사용성
5. P5: pal init 동기화       ← 초기 설정
6. P4: pal version           ← 간단
7. P6: 워크플로우 강제       ← 개선
8. P8: docs 구조             ← 개선
```

---

<!-- pal:port:status=draft -->
