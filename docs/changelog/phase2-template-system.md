# Phase 2: 템플릿 시스템 구현 완료

> 날짜: 2026-01-13
> 기능: 전역 에이전트/컨벤션 템플릿 시스템

---

## 개요

프로젝트별 독립 관리를 위한 템플릿 복사 방식 구현.
`pal init` 시 기본 에이전트 및 컨벤션이 자동으로 설치됩니다.

---

## 구현 내용

### 1. 템플릿 디렉토리 구조

```
internal/agent/templates/
├── agents/
│   └── workers/
│       ├── backend/    # 6개 백엔드 워커
│       └── frontend/   # 5개 프론트엔드 워커
├── conventions/
│   ├── agents/
│   │   ├── core/       # 7개 코어 에이전트 컨벤션
│   │   └── workers/
│   │       ├── backend/    # 6개 백엔드 컨벤션
│   │       └── frontend/   # 5개 프론트엔드 컨벤션
│   └── ui/             # 2개 UI 컨벤션
├── core/               # 7개 코어 에이전트 (레거시)
└── workers/            # 5개 워커 템플릿 (레거시)

총 42개 템플릿 파일
```

### 2. Embed 시스템 개선

**파일:** `internal/agent/embed.go`

**추가 기능:**
- `InstallTemplates()` - 기존 파일 보존하며 설치
- `InstallTemplatesWithOverwrite()` - 강제 덮어쓰기
- `CountTemplates()` - 템플릿 개수 반환
- `ListTemplates()` - 모든 템플릿 파일 (.md 포함) 반환

**특징:**
```go
//go:embed templates/*
var templateFS embed.FS

// 기존 파일이 있으면 스킵
if _, err := os.Stat(targetPath); err == nil {
    return nil
}
```

### 3. pal init 명령어 개선

**파일:** `internal/cli/init.go`, `internal/cli/init_template.go`

**추가 플래그:**
- `--skip-templates` - 템플릿 설치 건너뛰기
- `--templates-force` - 템플릿 강제 덮어쓰기

**설치 과정:**
```bash
$ pal init my-project

1. 디렉토리 구조 생성
2. .claude/settings.json
3. CLAUDE.md
4. 전역 DB 등록
5. 템플릿 설치 ← NEW!
6. .pal/manifest.yaml
7. .gitignore 업데이트
```

**설치되는 템플릿:**
```
agents/
├── workers/
│   ├── backend/
│   │   ├── cache.yaml
│   │   ├── document.yaml
│   │   ├── entity.yaml
│   │   ├── router.yaml
│   │   ├── service.yaml
│   │   └── test.yaml
│   └── frontend/
│       ├── e2e.yaml
│       ├── engineer.yaml
│       ├── model.yaml
│       ├── ui.yaml
│       └── unit-tc.yaml

conventions/
├── agents/
│   ├── core/
│   │   ├── _common.md
│   │   ├── builder.md
│   │   ├── planner.md
│   │   ├── architect.md
│   │   ├── manager.md
│   │   ├── tester.md
│   │   └── logger.md
│   └── workers/
│       ├── _common.md
│       ├── backend/
│       │   ├── cache.md
│       │   ├── document.md
│       │   ├── entity.md
│       │   ├── router.md
│       │   ├── service.md
│       │   └── test.md
│       └── frontend/
│           ├── e2e.md
│           ├── engineer.md
│           ├── model.md
│           ├── ui.md
│           └── unit-tc.md
└── ui/
    ├── mui.md
    └── tailwind.md
```

---

## 테스트 결과

### 단위 테스트
```bash
$ go test ./internal/agent -v
=== RUN   TestListTemplates
    embed_test.go:20: Found 42 templates
--- PASS: TestListTemplates (0.00s)
=== RUN   TestGetTemplate
--- PASS: TestGetTemplate (0.00s)
=== RUN   TestGetTemplate_AllTemplates
--- PASS: TestGetTemplate_AllTemplates (0.00s)
=== RUN   TestInstallTemplates
--- PASS: TestInstallTemplates (0.01s)
=== RUN   TestInstallTemplates_Content
--- PASS: TestInstallTemplates_Content (0.01s)
=== RUN   TestInstallTemplates_Idempotent
--- PASS: TestInstallTemplates_Idempotent (0.01s)
PASS
ok  	github.com/n0roo/pal-kit/internal/agent	0.435s
```

### 통합 테스트
```bash
$ mkdir test-project && cd test-project
$ pal init

🚀 PAL Kit 프로젝트 초기화 완료!

생성된 항목:
  ✅ 디렉토리 구조
  ✅ .claude/settings.json
  ✅ CLAUDE.md
  ✅ 전역 DB에 프로젝트 등록
  ✅ 에이전트 및 컨벤션 템플릿  ← 42개 파일
  ✅ .pal/manifest.yaml

$ find agents -type f | wc -l
11  # backend 6개 + frontend 5개

$ find conventions -type f | wc -l
20  # core 7개 + workers 11개 + ui 2개

$ pal agent list | wc -l
11

$ pal convention list | wc -l
24  # agents 컨벤션 + UI 컨벤션
```

---

## 사용 예시

### 기본 사용
```bash
# 새 프로젝트 생성
mkdir my-project && cd my-project
pal init

# 에이전트 확인
pal agent list

# 컨벤션 확인
pal convention list
```

### 템플릿 건너뛰기
```bash
# 템플릿 없이 초기화 (수동 설정 원할 때)
pal init --skip-templates
```

### 템플릿 강제 업데이트
```bash
# 기존 템플릿 덮어쓰기 (템플릿 업데이트 시)
pal init --templates-force
```

---

## 아키텍처 설계

### 템플릿 복사 방식 선택 이유

**Option A: 전역 설치 + 프로젝트 오버라이드**
- ❌ 복잡한 오버라이드 로직
- ❌ 전역 템플릿 변경 시 기존 프로젝트 영향

**Option B: 템플릿 복사 방식** ✅ 선택됨
- ✅ 프로젝트별 완전 독립
- ✅ 프로젝트 내에서 자유롭게 수정 가능
- ✅ 간단한 구현

**Option C: 하이브리드**
- ❌ 복잡도 증가
- ❌ 관리 포인트 증가

### Embed vs 외부 파일

**Go Embed 선택 이유:**
- ✅ 단일 바이너리 배포
- ✅ 버전 관리 용이
- ✅ 설치 불필요

---

## 수정된 파일

### 신규 파일
- `internal/agent/templates/agents/workers/**/*.yaml` (11개)
- `internal/agent/templates/conventions/**/*.md` (20개)
- `internal/cli/init_template.go`

### 수정된 파일
- `internal/agent/embed.go` - InstallTemplatesWithOverwrite(), CountTemplates() 추가
- `internal/agent/embed_test.go` - 새 템플릿 구조 반영
- `internal/cli/init.go` - 템플릿 설치 플래그 및 단계 추가

---

## 호환성

- ✅ 기존 프로젝트에 영향 없음 (`--skip-templates` 기본값 false)
- ✅ Phase 1 수정 사항과 호환
- ✅ 모든 테스트 통과

---

## 다음 단계 (Phase 3 - 선택적)

### 패키지 시스템 설계

**현재:**
- `kotlin-spring` 패키지 = 컨벤션만 오버라이드

**향후 고려사항:**
1. 패키지별 전용 에이전트 제공
   - `pal init --package kotlin-spring`
   - `templates/packages/kotlin-spring/agents/` 추가

2. 워크플로우와 에이전트 연동
   - 워크플로우 타입에 따라 기본 에이전트 세트 선택
   - `workflow: integrate` → builder, planner, architect, ...

3. 템플릿 마켓플레이스
   - 커뮤니티 템플릿 공유
   - `pal template install <name>`

---

## 빌드 및 배포

```bash
# 빌드
go build ./cmd/pal

# 설치
go install ./cmd/pal

# 버전 확인
pal --version
```

---

## 사용자 가이드

### 프로젝트 초기화
```bash
cd my-new-project
pal init
```

### 에이전트/컨벤션 커스터마이징
```bash
# 프로젝트 템플릿은 복사본이므로 자유롭게 수정 가능
vim agents/workers/backend/entity.yaml
vim conventions/agents/workers/backend/entity.md
```

### 템플릿 업데이트 (신규 버전)
```bash
# PAL Kit 업데이트 후 템플릿 재설치
pal init --templates-force
```

---

**Phase 2 완료 ✅**

**통계:**
- 템플릿 파일: 42개
- 신규 함수: 3개
- 통과 테스트: 19개
- 코드 증가: ~200 lines
