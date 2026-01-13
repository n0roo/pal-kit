# 에이전트 시스템 문제 분석 및 수정 계획

> 작성일: 2026-01-13
> 작성자: Builder + Architect 분석

---

## 1. 문제 요약

### 1.1 제기된 문제

1. **에이전트 컨벤션이 신규 프로젝트에서 제안되지 않음**
   - pal-kit 프로젝트에서 생성한 에이전트/컨벤션이 다른 프로젝트에서 작동하지 않음

2. **core/workers 에이전트의 역할 혼란**
   - core, workers 컨벤션이 PAL Kit의 기본 에이전트인지, 프로젝트별 에이전트인지 불명확
   - 패키지 시스템과의 관계 모호

3. **YAML 파싱 실패**
   - 원격 PC에서 DB pull 시 `yaml: line 2: mapping values are not allowed in this context` 에러

---

## 2. 빌더 분석: 현재 시스템 구조

### 2.1 에이전트 로딩 메커니즘

```go
// internal/agent/agent.go:49
func (s *Service) Load() error {
    s.agents = make(map[string]*Agent)

    // 로컬 agents/ 디렉토리만 스캔
    err := filepath.WalkDir(s.agentsDir, ...)
}
```

**문제점:**
- ❌ 프로젝트 로컬 `agents/` 폴더만 스캔
- ❌ PAL Kit 자체에 내장된 "전역 에이전트 템플릿"과 구분 없음
- ❌ 새 프로젝트에서는 에이전트가 비어있음

### 2.2 컨벤션 로딩 메커니즘

```go
// internal/convention/convention.go:97
func (s *Service) Load() error {
    // 로컬 conventions/ 디렉토리만 스캔
    entries, err := os.ReadDir(s.conventionsDir)
}
```

**문제점:**
- ❌ 프로젝트 로컬 `conventions/` 폴더만 스캔
- ❌ 컨벤션 파일이 재귀 탐색되지 않음 (하위 디렉토리 무시)
- ❌ `conventions/agents/core/*.md`, `conventions/agents/workers/**/*.md` 파일들이 로드되지 않음

### 2.3 현재 디렉토리 구조

```
pal-kit/
├── agents/                   # 프로젝트별 에이전트 (pal-kit용)
│   ├── worker-go.yaml
│   └── workers/
│       ├── backend/*.yaml
│       └── frontend/*.yaml
├── conventions/              # 프로젝트별 컨벤션
│   ├── agents/               # ❌ Load()에서 스캔되지 않음!
│   │   ├── core/*.md
│   │   └── workers/**/*.md
│   ├── go-style.yaml         # ✅ 스캔됨
│   └── pal-kit.yaml          # ✅ 스캔됨
└── internal/agent/
    └── templates/            # ❌ embed된 템플릿이지만 사용되지 않음
```

**핵심 문제:**
```
conventions/agents/core/builder.md  ← 이 파일들은 Load()에서 무시됨
conventions/agents/workers/backend/entity.md  ← 재귀 탐색 안 됨
```

### 2.4 YAML 파싱 문제

manifest.yaml 구조:
```yaml
version: "1"
updated_at: 2026-01-13T00:19:53.743199+09:00  # ← line 2
files:
    .pal/config.yaml:
        path: .pal/config.yaml
        ...
```

**잠재적 원인:**
1. time.Time 직렬화 형식 불일치
2. DB→YAML 변환 시 특수문자 이스케이프 누락
3. 원격/로컬 간 YAML 라이브러리 버전 차이

---

## 3. 아키텍처 분석: 설계 문제

### 3.1 전역 vs 프로젝트 에이전트 구분 부재

**현재 상황:**
```
pal-kit 프로젝트의 agents/workers/backend/entity.yaml
  → 이것은 "PAL Kit 기본 제공 에이전트"인가?
  → 아니면 "pal-kit 프로젝트 전용 에이전트"인가?
```

**사용자 기대:**
```
core, workers = PAL Kit이 기본 제공하는 에이전트
packages = 특정 프로젝트에서 이 에이전트들을 조합한 패키지
```

**실제 구현:**
```
core, workers = pal-kit 프로젝트 로컬에만 존재
다른 프로젝트 = 빈 agents/ 폴더로 시작
```

### 3.2 컨벤션 계층 구조 불일치

**문서상 계층 (CONVENTION-LOADING.md):**
```
Package → Agent Common → Agent Specific → Port Spec
```

**실제 구현:**
```
로컬 conventions/*.yaml 파일만 로드
conventions/agents/ 하위는 무시됨
```

### 3.3 아키텍처 갭

| 계층 | 기대 | 실제 |
|------|------|------|
| 전역 에이전트 | PAL Kit 내장 | 없음 |
| 전역 컨벤션 | PAL Kit 내장 | 없음 |
| 프로젝트 에이전트 | 전역 + 프로젝트 | 프로젝트만 |
| 프로젝트 컨벤션 | 전역 + 프로젝트 | 루트만 |
| 패키지 | 에이전트 조합 | 개념만 존재 |

---

## 4. 원인 분석

### 4.1 에이전트/컨벤션이 신규 프로젝트에서 작동하지 않는 이유

**근본 원인:**
1. **전역 템플릿 부재**: PAL Kit에 내장된 전역 에이전트/컨벤션이 없음
2. **초기화 누락**: 새 프로젝트 생성 시 기본 에이전트/컨벤션 복사 안 됨
3. **로딩 범위 제한**: 로컬 폴더만 스캔, 하위 디렉토리 무시

**시나리오:**
```bash
# pal-kit 프로젝트
agents/workers/backend/entity.yaml  ← 존재
conventions/agents/workers/backend/entity.md  ← 존재 (but 로드 안 됨)

# 새 프로젝트 (my-app)
cd ~/my-app
pal init
ls agents/  ← 비어있음!
ls conventions/  ← 비어있음!

# entity-worker 실행 시
pal agent get entity-worker  ← Error: 에이전트를 찾을 수 없습니다
```

### 4.2 컨벤션 재귀 로딩 문제

`internal/convention/convention.go:104-126`:
```go
entries, err := os.ReadDir(s.conventionsDir)  // 루트만 읽음
for _, entry := range entries {
    if entry.IsDir() {
        continue  // ← 디렉토리는 스킵!
    }
}
```

**결과:**
- `conventions/go-style.yaml` ✅ 로드됨
- `conventions/agents/core/builder.md` ❌ 로드 안 됨

### 4.3 YAML 파싱 실패

**추정 원인:**
```yaml
version: "1"
updated_at: 2026-01-13T00:19:53.743199+09:00
```

Go time.Time → YAML 변환 시:
- RFC3339Nano 형식: `2026-01-13T00:19:53.743199+09:00`
- 일부 YAML 파서에서 `:`이 value separator로 인식될 수 있음

**재현 조건:**
- DB에서 pull한 manifest 파일
- 다른 PC에서 생성된 timestamp

---

## 5. 협의 필요 사항

### 5.1 에이전트 배포 모델 확정

**Option A: 전역 내장 + 프로젝트 오버라이드**
```
~/.pal/
  agents/           ← PAL Kit 설치 시 전역으로 설치
  conventions/

~/my-project/
  agents/           ← 프로젝트별 추가/오버라이드
  conventions/
```

**Option B: 템플릿 복사 방식**
```
pal init
  → PAL Kit 내장 템플릿을 프로젝트로 복사
  → 이후 프로젝트 독립적으로 관리
```

**Option C: 하이브리드**
```
전역: 읽기 전용 기본 에이전트
프로젝트: 수정 가능 복사본
로딩: 전역 → 프로젝트 순으로 오버라이드
```

**질문:**
> core, workers 에이전트를 어떻게 배포하고 싶으신가요?
> 1. PAL Kit 전역 설치 (모든 프로젝트 공유)
> 2. 프로젝트 초기화 시 복사 (각 프로젝트 독립)
> 3. 전역 + 프로젝트 하이브리드

### 5.2 패키지 시스템 역할

**현재 이해:**
- 패키지 = 에이전트들의 조합

**불명확한 점:**
- 패키지는 어디에 정의되나?
- 패키지 선택이 에이전트 선택에 어떻게 영향?
- `kotlin-spring` 패키지 예시 구조는?

**질문:**
> 패키지 시스템을 어떻게 설계하고 싶으신가요?
>
> 예: "kotlin-spring" 패키지 선택 시
> - [ ] 자동으로 entity-worker, service-worker 등 활성화?
> - [ ] 컨벤션만 오버라이드하고 에이전트는 동일?
> - [ ] 패키지별 전용 에이전트 제공?

### 5.3 컨벤션 재귀 로딩 범위

**질문:**
> conventions/ 하위를 어떻게 탐색하나요?
> - [ ] 재귀적으로 모든 .md, .yaml 파일 로드
> - [ ] agents/ 하위만 특별 처리
> - [ ] 명시적으로 등록된 경로만

---

## 6. 수정 계획 (Draft)

### Phase 1: 컨벤션 재귀 로딩 수정
```go
// internal/convention/convention.go
func (s *Service) Load() error {
    // filepath.WalkDir로 재귀 탐색
    err := filepath.WalkDir(s.conventionsDir, func(path string, d fs.DirEntry, err error) error {
        if !d.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".md")) {
            // 로드
        }
    })
}
```

### Phase 2: 전역 에이전트 템플릿 시스템
```go
// internal/agent/global.go
type GlobalService struct {
    globalDir   string // ~/.pal/agents/
    projectDir  string // ./agents/
}

func (s *GlobalService) Load() {
    // 1. 전역 에이전트 로드
    // 2. 프로젝트 에이전트 로드 (오버라이드)
}
```

### Phase 3: 프로젝트 초기화 개선
```bash
pal init
  → ~/.pal/templates/agents/ 복사
  → ~/.pal/templates/conventions/ 복사
  → manifest 생성
```

### Phase 4: YAML 파싱 안정화
```go
// manifest.yaml 저장 시 Time 형식 명시
data, err := yaml.Marshal(manifest)
// updated_at를 string으로 변환하여 저장
```

---

## 7. 다음 단계

1. **협의 필요 항목 응답 대기**
   - 에이전트 배포 모델 (Option A/B/C)
   - 패키지 시스템 설계
   - 컨벤션 로딩 범위

2. **상세 설계 작성**
   - 선택된 옵션 기반 상세 설계
   - API 변경 사항 정리
   - 마이그레이션 계획

3. **구현 착수**
   - Phase 1부터 순차 구현
   - 각 Phase별 테스트

---

## 8. 추가 발견 사항

### 8.1 embed.go의 템플릿 시스템 미사용

`internal/agent/embed.go`:
```go
//go:embed templates/*
var templateFS embed.FS
```

- 템플릿 파일 embed 준비는 되어있음
- 하지만 `templates/` 폴더가 실제로 존재하지 않음
- `InstallTemplates()` 함수는 구현되었으나 호출되지 않음

**제안:**
이 구조를 활용하여 전역 에이전트 템플릿 시스템 구축 가능

### 8.2 YAML 파싱 문제 상세 분석

**에러 메시지:**
```
yaml: line 2: mapping values are not allowed in this context
```

**manifest.yaml line 2:**
```yaml
updated_at: 2026-01-13T00:19:53.743199+09:00
```

**문제 원인 추정:**

1. **Time 형식의 타임존 표현**
   - RFC3339Nano: `2026-01-13T00:19:53.743199+09:00`
   - `+09:00`에서 `:`가 YAML 파서에 의해 key-value separator로 오인될 수 있음
   - 특정 YAML 파서 버전이나 설정에서 발생

2. **Go yaml.v3의 Time 직렬화**
   ```go
   // internal/manifest/manifest.go:134
   manifest.UpdatedAt = time.Now()
   data, err := yaml.Marshal(manifest)
   ```
   - Time은 자동으로 RFC3339Nano 형식으로 마샬링됨
   - 따옴표 없이 직렬화될 수 있음

3. **재현 조건**
   - DB에서 pull한 데이터 (다른 PC의 timestamp)
   - 로컬에서 생성한 manifest는 정상 작동
   - 원격→로컬 동기화 시에만 발생

**해결 방안:**

```go
// Option A: Time을 문자열로 변환하여 저장
type ManifestForYAML struct {
    Version   string                  `yaml:"version"`
    UpdatedAt string                  `yaml:"updated_at"`  // RFC3339 string
    Files     map[string]*TrackedFile `yaml:"files"`
}

func (s *Service) SaveManifest(manifest *Manifest) error {
    yamlManifest := ManifestForYAML{
        Version:   manifest.Version,
        UpdatedAt: manifest.UpdatedAt.Format(time.RFC3339),  // 명시적 형식화
        Files:     manifest.Files,
    }
    data, err := yaml.Marshal(yamlManifest)
    // ...
}
```

```go
// Option B: Flow style로 강제
data, err := yaml.Marshal(manifest)
if err != nil {
    return err
}

// YAML 후처리: timestamp를 따옴표로 감싸기
content := string(data)
content = regexp.MustCompile(`updated_at: (.+\+\d{2}:\d{2})`).
    ReplaceAllString(content, `updated_at: "$1"`)
data = []byte(content)
```

```go
// Option C: MarshalYAML 커스텀 구현
func (m *Manifest) MarshalYAML() (interface{}, error) {
    return &struct {
        Version   string                  `yaml:"version"`
        UpdatedAt string                  `yaml:"updated_at"`
        Files     map[string]*TrackedFile `yaml:"files"`
    }{
        Version:   m.Version,
        UpdatedAt: fmt.Sprintf("%q", m.UpdatedAt.Format(time.RFC3339)),  // 따옴표 포함
        Files:     m.Files,
    }, nil
}
```

**권장 해결책: Option A**
- 가장 단순하고 명확
- YAML spec과 완전 호환
- 디버깅 용이

---

**작성 완료 - 협의 대기 중**
