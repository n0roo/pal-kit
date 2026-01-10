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

> 생성일: {{.Date}}

## 프로젝트 개요

이 프로젝트는 ...

## 디렉토리 구조

` + "```" + `
.
├── agents/          # 에이전트 정의
├── ports/           # 포트 명세
├── conventions/     # 컨벤션 문서
├── .claude/         # Claude 설정
│   └── rules/       # 활성 규칙
└── .pal/            # PAL Kit 데이터
` + "```" + `

## 개발 규칙

1. 모든 변경은 포트 단위로 진행
2. 에이전트는 역할에 맞는 작업만 수행
3. 컨벤션을 준수

## 컨벤션

- [코딩 스타일](conventions/coding-style.md)
- [커밋 메시지](conventions/commit-message.md)

<!-- pal:context:start -->
<!-- PAL Kit이 자동으로 업데이트합니다 -->
<!-- pal:context:end -->
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
