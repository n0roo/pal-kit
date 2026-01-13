package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Template represents a document template
type Template struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        DocType `json:"type"`
	FileName    string `json:"file_name"`
	Content     string `json:"content,omitempty"`
}

// TemplateData holds data for template rendering
type TemplateData struct {
	ProjectName string
	Date        string
	Author      string
	PortID      string
	PortTitle   string
	AgentID     string
	AgentName   string
	AgentType   string
	// Support Agent 템플릿용
	DomainName  string
	Name        string
	ADRID       string
	ADRSlug     string
	ADRTitle    string
	Custom      map[string]string
}

// DefaultTemplates returns built-in templates
func DefaultTemplates() []Template {
	return []Template{
		{
			Name:        "claude-md",
			Description: "프로젝트 CLAUDE.md 템플릿",
			Type:        DocTypeClaude,
			FileName:    "CLAUDE.md",
			Content:     claudeMDTemplate,
		},
		{
			Name:        "agent-builder",
			Description: "빌더 에이전트 템플릿",
			Type:        DocTypeAgent,
			FileName:    "agents/{{.AgentID}}.yaml",
			Content:     agentBuilderTemplate,
		},
		{
			Name:        "agent-worker",
			Description: "워커 에이전트 템플릿",
			Type:        DocTypeAgent,
			FileName:    "agents/{{.AgentID}}.yaml",
			Content:     agentWorkerTemplate,
		},
		{
			Name:        "port-spec",
			Description: "포트 명세 템플릿",
			Type:        DocTypePort,
			FileName:    "ports/{{.PortID}}.md",
			Content:     portSpecTemplate,
		},
		{
			Name:        "convention",
			Description: "컨벤션 문서 템플릿",
			Type:        DocTypeConvention,
			FileName:    "conventions/{{.Name}}.md",
			Content:     conventionTemplate,
		},
		// Support Agent용 템플릿
		{
			Name:        "domain-spec",
			Description: "도메인 명세 템플릿",
			Type:        DocTypeTemplate,
			FileName:    "domains/{{.DomainName}}/overview.md",
			Content:     domainSpecTemplate,
		},
		{
			Name:        "business-rule",
			Description: "비즈니스 규칙 템플릿",
			Type:        DocTypeTemplate,
			FileName:    "domains/{{.DomainName}}/rules.md",
			Content:     businessRuleTemplate,
		},
		{
			Name:        "adr",
			Description: "아키텍처 결정 기록 템플릿",
			Type:        DocTypeTemplate,
			FileName:    ".pal/decisions/{{.ADRID}}-{{.ADRSlug}}.md",
			Content:     adrTemplate,
		},
	}
}

// ListTemplates returns available templates
func (s *Service) ListTemplates() []Template {
	templates := DefaultTemplates()

	// 프로젝트 커스텀 템플릿 추가
	customDir := filepath.Join(s.projectRoot, "templates")
	if entries, err := os.ReadDir(customDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			templates = append(templates, Template{
				Name:        "custom:" + name,
				Description: "사용자 정의 템플릿",
				Type:        DocTypeTemplate,
				FileName:    entry.Name(),
			})
		}
	}

	return templates
}

// GetTemplate returns a template by name
func (s *Service) GetTemplate(name string) (*Template, error) {
	// 커스텀 템플릿 먼저 확인
	if strings.HasPrefix(name, "custom:") {
		customName := strings.TrimPrefix(name, "custom:")
		customPath := filepath.Join(s.projectRoot, "templates", customName+".md")
		content, err := os.ReadFile(customPath)
		if err == nil {
			return &Template{
				Name:     name,
				Content:  string(content),
				FileName: customName + ".md",
			}, nil
		}
	}

	// 기본 템플릿
	for _, t := range DefaultTemplates() {
		if t.Name == name {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("템플릿 '%s'을(를) 찾을 수 없습니다", name)
}

// ApplyTemplate applies a template with given data
func (s *Service) ApplyTemplate(templateName string, data TemplateData) (string, string, error) {
	tmpl, err := s.GetTemplate(templateName)
	if err != nil {
		return "", "", err
	}

	// 기본 데이터 설정
	if data.Date == "" {
		data.Date = time.Now().Format("2006-01-02")
	}
	if data.Custom == nil {
		data.Custom = make(map[string]string)
	}

	// 파일명 렌더링
	fileNameTmpl, err := template.New("filename").Parse(tmpl.FileName)
	if err != nil {
		return "", "", fmt.Errorf("파일명 템플릿 파싱 실패: %w", err)
	}
	var fileNameBuf strings.Builder
	if err := fileNameTmpl.Execute(&fileNameBuf, data); err != nil {
		return "", "", fmt.Errorf("파일명 렌더링 실패: %w", err)
	}

	// 내용 렌더링
	contentTmpl, err := template.New("content").Parse(tmpl.Content)
	if err != nil {
		return "", "", fmt.Errorf("내용 템플릿 파싱 실패: %w", err)
	}
	var contentBuf strings.Builder
	if err := contentTmpl.Execute(&contentBuf, data); err != nil {
		return "", "", fmt.Errorf("내용 렌더링 실패: %w", err)
	}

	return fileNameBuf.String(), contentBuf.String(), nil
}

// CreateFromTemplate creates a file from template
func (s *Service) CreateFromTemplate(templateName string, data TemplateData, overwrite bool) (string, error) {
	fileName, content, err := s.ApplyTemplate(templateName, data)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(s.projectRoot, fileName)

	// 파일 존재 확인
	if !overwrite {
		if _, err := os.Stat(fullPath); err == nil {
			return "", fmt.Errorf("파일이 이미 존재합니다: %s", fileName)
		}
	}

	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	// 파일 생성
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("파일 생성 실패: %w", err)
	}

	return fileName, nil
}

// InitProject initializes project with default documents
func (s *Service) InitProject(projectName string) ([]string, error) {
	if err := s.EnsureDirectories(); err != nil {
		return nil, err
	}

	var created []string
	data := TemplateData{
		ProjectName: projectName,
		Date:        time.Now().Format("2006-01-02"),
	}

	// CLAUDE.md 생성
	if _, err := os.Stat(filepath.Join(s.projectRoot, "CLAUDE.md")); os.IsNotExist(err) {
		if file, err := s.CreateFromTemplate("claude-md", data, false); err == nil {
			created = append(created, file)
		}
	}

	return created, nil
}

// Template contents
const claudeMDTemplate = `# {{.ProjectName}}

> PAL Kit 관리 프로젝트 | 생성일: {{.Date}}

---

## 🚀 PAL Kit 초기 설정 필요

이 프로젝트는 **PAL Kit 환경이 구성되지 않았습니다.**

### Claude에게 요청하세요:

` + "```" + `
이 프로젝트의 PAL Kit 환경을 설정해줘
` + "```" + `

또는 직접 실행:

` + "```bash" + `
pal setup
` + "```" + `

---

## 설정 플로우 안내 (Claude용)

### 1단계: 프로젝트 분석

` + "```bash" + `
pal analyze --json
` + "```" + `

실행 후 분석 결과를 확인하고 사용자에게 설명합니다.

### 2단계: 워크플로우 선택

| 타입 | 설명 | 적합한 경우 |
|------|------|------------|
| **simple** | 대화형 협업, 종합 에이전트 | 간단한 작업, 학습 |
| **single** | 단일 세션, 역할 전환 | 중간 규모 기능 |
| **integrate** | 빌더 관리, 서브세션 | 복잡한 기능, 여러 기술 |
| **multi** | 복수 integrate | 대규모 프로젝트 |

사용자에게 추천 워크플로우를 설명하고 확인합니다.

### 3단계: 설정 적용

` + "```bash" + `
# 워크플로우 설정
pal config set workflow <type>

# 필요한 워커 에이전트 추가 (기술 스택에 따라)
pal agent add workers/backend/go
pal agent add workers/frontend/react

# 설정 확인
pal config show
` + "```" + `

### 4단계: CLAUDE.md 업데이트

설정 완료 후 이 파일의 "설정 필요" 섹션을 삭제하고,
프로젝트 설명과 작업 가이드로 대체합니다.

---

## PAL Kit 연계 가이드

### 세션 시작 시 필수 체크

` + "```bash" + `
# 프로젝트 상태 확인
pal status

# 포트 목록 확인 (진행 중인 작업)
pal port list

# 최근 세션 기록 확인
pal session list
` + "```" + `

### 서브에이전트(Task tool) 활용 패턴

` + "```" + `
# Worker 에이전트 호출
Task tool로 "worker-{tech}" 서브에이전트 실행
예: worker-go, worker-react, worker-python

# 작업 완료 후 결과 수집
서브에이전트 결과를 바탕으로 상태 업데이트
` + "```" + `

### 포트 기반 작업 흐름

1. 포트 시작: ` + "`pal hook port-start <id>`" + `
2. .claude/rules/ 에 동적 규칙 생성됨
3. 작업 수행 (규칙 참조)
4. 포트 종료: ` + "`pal hook port-end <id>`" + `

### Core vs Worker 에이전트

| 타입 | 역할 | 예시 |
|------|------|------|
| Core | 오케스트레이션 | builder, planner, architect, operator |
| Worker | 실행 | worker-go, worker-react |

### 작업 완료 시 기록

**ADR 생성 기준:**
- 아키텍처 변경
- 기술 스택 선택
- 설계 패턴 결정
- 중요한 트레이드오프

---

## PAL Kit 명령어

` + "```bash" + `
# 상태 확인
pal status
pal status --dashboard

# 포트 관리
pal port list
pal port create <id> --title "작업명"
pal port status <id>

# 작업 시작/종료
pal hook port-start <id>
pal hook port-end <id>

# 세션 관리
pal session list
pal session show <id>
pal session summary

# 파이프라인
pal pipeline list
pal pl plan <n>

# 대시보드
pal serve
` + "```" + `

---

## 디렉토리 구조

` + "```" + `
.
├── CLAUDE.md           # 이 파일 (프로젝트 컨텍스트)
├── agents/             # 에이전트 정의
│   ├── core/           # 코어 에이전트
│   └── workers/        # 워커 에이전트
├── ports/              # 포트 명세
├── conventions/        # 컨벤션 문서
├── .claude/
│   ├── settings.json   # Claude Code Hook 설정
│   └── rules/          # 조건부 규칙 (동적 생성)
└── .pal/
    ├── config.yaml     # PAL Kit 설정
    ├── sessions/       # 세션 기록
    ├── decisions/      # ADR
    └── context/        # 컨텍스트 파일
` + "```" + `

---

<!-- pal:config:status=pending -->
<!--
  PAL Kit 설정 상태: 미완료
  설정 완료 후 이 섹션이 업데이트됩니다.
-->
`

const agentBuilderTemplate = `agent:
  id: {{.AgentID}}
  name: {{.AgentName}}
  type: builder
  description: 파이프라인과 포트를 관리하는 빌더 에이전트
  prompt: |
    # {{.AgentName}}

    당신은 빌더 에이전트입니다.

    ## 역할
    - 파이프라인 생성 및 관리
    - 포트 분배 및 의존성 설정
    - 작업 진행 상황 모니터링

    ## 규칙
    1. 작업을 적절한 크기의 포트로 분할
    2. 의존성을 명확히 설정
    3. 완료된 작업은 즉시 상태 업데이트
  tools:
    - bash
    - pal
  config:
    max_tokens: "8000"
`

const agentWorkerTemplate = `agent:
  id: {{.AgentID}}
  name: {{.AgentName}}
  type: worker
  description: 실제 작업을 수행하는 워커 에이전트
  prompt: |
    # {{.AgentName}}

    당신은 워커 에이전트입니다.

    ## 역할
    - 포트 명세에 따라 작업 수행
    - 코드 작성 및 테스트
    - 문서화

    ## 규칙
    1. 포트 범위 내에서만 작업
    2. 컨벤션 준수
    3. 문제 발생 시 에스컬레이션
  tools:
    - bash
    - editor
  config:
    max_tokens: "16000"
`

const portSpecTemplate = `# {{.PortTitle}}

> Port ID: {{.PortID}}
> 생성일: {{.Date}}

## 개요

이 포트는 ...

## 작업 범위

### 포함
- 

### 제외
- 

## 파일 패턴

` + "```" + `
internal/{{.PortID}}/**/*
` + "```" + `

## 완료 조건

- [ ] 기능 구현
- [ ] 테스트 작성
- [ ] 문서화

## 참고 자료

- 
`

const conventionTemplate = `# {{.Name}}

> 생성일: {{.Date}}

## 개요

이 컨벤션은 ...

## 규칙

### 1.

### 2.

## 예시

### Good

` + "```" + `
` + "```" + `

### Bad

` + "```" + `
` + "```" + `

## 예외

-
`

// Support Agent 템플릿

const domainSpecTemplate = `# {{.DomainName}} 도메인 명세

> 생성일: {{.Date}}

---

## 개요

{{.DomainName}} 도메인은 ...

---

## 핵심 개념

### 엔티티

| 이름 | 설명 | 속성 |
|------|------|------|
| - | - | - |

### 값 객체

| 이름 | 설명 | 속성 |
|------|------|------|
| - | - | - |

---

## 경계 컨텍스트

### 포함

-

### 제외

-

---

## 도메인 이벤트

| 이벤트 | 트리거 | 페이로드 |
|--------|--------|----------|
| - | - | - |

---

## 관련 포트

| 포트 ID | 레이어 | 설명 |
|---------|--------|------|
| - | - | - |

---

## 비즈니스 규칙

자세한 비즈니스 규칙은 [rules.md](./rules.md) 참조

---

## 관련 문서

- [[관련 문서 1]]
- [[관련 문서 2]]

---

tags: #domain/{{.DomainName}} #type/domain-spec
`

const businessRuleTemplate = `# {{.DomainName}} 비즈니스 규칙

> 생성일: {{.Date}}
> 도메인: {{.DomainName}}

---

## 규칙 목록

### BR-001: 규칙 이름

**설명**: 규칙 설명

**조건**:
- 조건 1
- 조건 2

**결과**:
- 결과 1

**예외**:
- 예외 상황

` + "```" + `pseudo
IF 조건1 AND 조건2
THEN 결과
` + "```" + `

---

### BR-002: 규칙 이름

**설명**: ...

---

## 규칙 간 의존성

` + "```" + `
BR-001 → BR-002 (선행)
BR-003 ← BR-001 (후행)
` + "```" + `

---

## 검증 체크리스트

- [ ] 모든 규칙에 예외 상황 정의됨
- [ ] 규칙 간 충돌 없음
- [ ] 도메인 전문가 검토 완료

---

tags: #domain/{{.DomainName}} #type/business-rule
`

const adrTemplate = `# ADR-{{.ADRID}}: {{.ADRTitle}}

> 생성일: {{.Date}}
> 상태: proposed

---

## 컨텍스트

어떤 문제를 해결하려고 하는가?

---

## 결정

무엇을 결정했는가?

---

## 대안

### 대안 1: ...

**장점**:
-

**단점**:
-

### 대안 2: ...

**장점**:
-

**단점**:
-

---

## 결과

### 긍정적

-

### 부정적

-

---

## 참고 자료

-

---

<!-- adr:status=proposed -->
`
