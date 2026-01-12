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

## 워크플로우

**integrate** - 빌더 관리, 서브세션 방식

복잡한 기능 개발과 여러 기술 스택을 다루는 작업에 적합합니다.

---

## 에이전트

| 에이전트 | 타입 | 역할 |
|---------|------|------|
| builder | core | 빌드 관리 |
| planner | core | 작업 계획 |
| architect | core | 설계 |
| manager | core | 세션 관리 |
| tester | core | 테스트 |
| logger | core | 로깅 |
| worker-go | worker | Go 코드 작성 |

---

## PAL Kit 명령어

```bash
# 상태 확인
pal status

# 포트 관리
pal port list
pal port create <id> --title "작업명"

# 작업 시작/종료
pal hook port-start <id>
pal hook port-end <id>

# 파이프라인
pal pipeline list
pal pl plan <n>

# 대시보드
pal serve
```

---

## 디렉토리 구조

```
.
├── CLAUDE.md           # 프로젝트 컨텍스트
├── cmd/                # CLI 진입점
├── internal/           # 내부 패키지
├── docs/               # 문서
├── agents/             # 에이전트 정의
├── ports/              # 포트 명세
├── conventions/        # 컨벤션 문서
├── .claude/
│   ├── settings.json   # Claude Code Hook 설정
│   └── rules/          # 조건부 규칙
└── .pal/
    └── config.yaml     # PAL Kit 설정
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
> 마지막 업데이트: 2026-01-12 23:17:59

### 활성 세션
- **7acea453**: -

### 포트 현황
- ✅ complete: 12

### 에스컬레이션
- 없음

<!-- pal:context:end -->
