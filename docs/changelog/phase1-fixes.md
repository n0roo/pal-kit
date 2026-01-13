# Phase 1 수정 완료

> 날짜: 2026-01-13
> 이슈: 에이전트/컨벤션 신규 프로젝트 문제, YAML 파싱 실패

---

## 수정 내용

### 1. 컨벤션 재귀 로딩 수정

**파일:** `internal/convention/convention.go`

**문제:**
- `os.ReadDir`로 루트 디렉토리만 읽음
- `conventions/agents/core/*.md`, `conventions/agents/workers/**/*.md` 파일들이 로드되지 않음

**수정:**
- `filepath.WalkDir`로 재귀 탐색
- `.md` 파일 지원 추가
- 경로 기반 ID 생성 (`agents-core-builder`, `agents-workers-backend-entity` 등)

**변경 사항:**
```go
// Before: 루트만 스캔
entries, err := os.ReadDir(s.conventionsDir)
for _, entry := range entries {
    if entry.IsDir() {
        continue  // 하위 디렉토리 무시!
    }
}

// After: 재귀 스캔
err := filepath.WalkDir(s.conventionsDir, func(path string, d os.DirEntry, err error) error {
    if d.IsDir() {
        return nil  // 계속 탐색
    }

    // .yaml, .yml, .md 처리
    if strings.HasSuffix(name, ".md") {
        return s.loadMarkdownConvention(...)
    }
    // ...
})
```

**결과:**
- ✅ `conventions/agents/core/builder.md` → 로드됨
- ✅ `conventions/agents/workers/backend/entity.md` → 로드됨
- ✅ 총 24개 에이전트 컨벤션 로드 성공

**테스트:**
```bash
$ pal convention list | grep agents-core
✅ 💻 _common (P5) - ID: agents-core-_common
✅ 💻 builder (P5) - ID: agents-core-builder
✅ 💻 planner (P5) - ID: agents-core-planner
✅ 💻 architect (P5) - ID: agents-core-architect
✅ 💻 manager (P5) - ID: agents-core-manager
✅ 💻 tester (P5) - ID: agents-core-tester
✅ 💻 logger (P5) - ID: agents-core-logger
```

---

### 2. YAML Timestamp 파싱 실패 수정

**파일:** `internal/manifest/manifest.go`

**문제:**
- `time.Time`이 RFC3339Nano로 자동 마샬링: `2026-01-13T00:19:53.743199+09:00`
- 타임존 표기의 콜론(`+09:00`)이 YAML 파서에서 key-value separator로 오인됨
- 원격 PC에서 DB pull 시 에러: `yaml: line 2: mapping values are not allowed in this context`

**수정:**
- `ManifestYAML` 구조체 추가 (YAML 직렬화 전용)
- `time.Time` → RFC3339 문자열로 명시적 변환
- 저장/로드 시 타입 변환 처리

**변경 사항:**
```go
// ManifestYAML 추가
type ManifestYAML struct {
    Version   string                  `yaml:"version"`
    UpdatedAt string                  `yaml:"updated_at"`  // RFC3339 string
    Files     map[string]*TrackedFile `yaml:"files"`
}

// SaveManifest
yamlManifest := ManifestYAML{
    Version:   manifest.Version,
    UpdatedAt: manifest.UpdatedAt.Format(time.RFC3339),  // 명시적 변환
    Files:     manifest.Files,
}
data, err := yaml.Marshal(yamlManifest)

// LoadManifest
var yamlManifest ManifestYAML
yaml.Unmarshal(data, &yamlManifest)
updatedAt, _ := time.Parse(time.RFC3339, yamlManifest.UpdatedAt)
```

**결과:**
```yaml
# Before (파싱 실패)
version: "1"
updated_at: 2026-01-13T00:19:53.743199+09:00
           ↑ 콜론 문제!

# After (파싱 성공)
version: "1"
updated_at: "2026-01-13T14:12:56+09:00"
           ↑ 따옴표로 감싸짐
```

**테스트:**
```bash
$ pal manifest sync
🔄 Manifest 동기화 완료

$ head -3 .pal/manifest.yaml
version: "1"
updated_at: "2026-01-13T14:23:52+09:00"
files:

$ pal manifest status
📋 Manifest 상태
총: 51개 파일 (동기화: 51, ...)
```

---

## 테스트 결과

### 단위 테스트
```bash
$ go test ./internal/convention -v
PASS
ok  	github.com/n0roo/pal-kit/internal/convention	0.401s

$ go test ./internal/manifest -v
PASS
ok  	github.com/n0roo/pal-kit/internal/manifest	0.089s
```

### Lint
```bash
$ golangci-lint run ./internal/convention/... ./internal/manifest/...
0 issues.
```

### 통합 테스트
```bash
✅ pal convention list - 24개 에이전트 컨벤션 로드 성공
✅ pal manifest sync - YAML 저장/로드 성공
✅ pal manifest status - 51개 파일 추적 성공
```

---

## 영향 범위

### 수정된 파일
- `internal/convention/convention.go` - Load(), loadConventionFile(), loadMarkdownConvention() 추가
- `internal/manifest/manifest.go` - ManifestYAML 추가, SaveManifest(), LoadManifest() 수정

### 호환성
- ✅ 기존 YAML 컨벤션 파일과 호환
- ✅ 기존 manifest.yaml 파일 로드 가능 (파싱 실패 시 현재 시간 사용)
- ✅ 모든 기존 테스트 통과

---

## 남은 작업 (Phase 2)

Phase 1은 **Quick Fix**로 즉시 해결 가능한 문제만 수정했습니다.

### Phase 2 계획 (아키텍처 개선)
1. **전역 에이전트 템플릿 시스템**
   - `internal/agent/embed.go`의 템플릿 시스템 활용
   - PAL Kit 바이너리에 기본 에이전트/컨벤션 embed
   - `pal init` 시 프로젝트로 복사

2. **프로젝트 초기화 개선**
   - `pal init` 명령어 강화
   - 기본 에이전트/컨벤션 자동 설치
   - 패키지 선택 기능 (향후)

3. **패키지 시스템 설계**
   - 패키지별 전용 에이전트 제공 방식 검토
   - 워크플로우와 에이전트 연동 방식 정의

---

## 배포

```bash
# 빌드 및 설치
go build ./cmd/pal
go install ./cmd/pal

# 버전 확인
pal --version
```

---

**Phase 1 수정 완료 ✅**
