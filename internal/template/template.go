package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// TemplateType represents the type of template
type TemplateType string

const (
	TypePort    TemplateType = "port"
	TypeAgent   TemplateType = "agent"
	TypeSession TemplateType = "session"
	TypeHook    TemplateType = "hook"
)

// ValidTypes lists all valid template types
var ValidTypes = []TemplateType{TypePort, TypeAgent, TypeSession, TypeHook}

// TemplateData holds data for template rendering
type TemplateData struct {
	ID        string
	Title     string
	Date      string
	Timestamp string
}

// Service handles template operations
type Service struct {
	templatesDir string
}

// NewService creates a new template service
func NewService(baseDir string) *Service {
	return &Service{
		templatesDir: filepath.Join(baseDir, ".claude", "templates"),
	}
}

// List returns available templates
func (s *Service) List() []TemplateType {
	return ValidTypes
}

// Create creates a file from a template
func (s *Service) Create(templateType TemplateType, outputPath string, data TemplateData) error {
	// 기본값 설정
	if data.Date == "" {
		data.Date = time.Now().Format("2006-01-02")
	}
	if data.Timestamp == "" {
		data.Timestamp = time.Now().Format("2006-01-02 15:04:05")
	}

	content, err := s.GetTemplateContent(templateType)
	if err != nil {
		return err
	}

	// 템플릿 파싱 및 렌더링
	tmpl, err := template.New(string(templateType)).Parse(content)
	if err != nil {
		return fmt.Errorf("템플릿 파싱 실패: %w", err)
	}

	// 출력 디렉토리 생성
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	// 파일 생성
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("파일 생성 실패: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("템플릿 렌더링 실패: %w", err)
	}

	return nil
}

// GetTemplateContent returns the template content
func (s *Service) GetTemplateContent(templateType TemplateType) (string, error) {
	switch templateType {
	case TypePort:
		return portTemplate, nil
	case TypeAgent:
		return agentTemplate, nil
	case TypeSession:
		return sessionTemplate, nil
	case TypeHook:
		return hookTemplate, nil
	default:
		return "", fmt.Errorf("알 수 없는 템플릿 타입: %s", templateType)
	}
}

// IsValidType checks if a template type is valid
func IsValidType(t string) bool {
	for _, valid := range ValidTypes {
		if string(valid) == t {
			return true
		}
	}
	return false
}

// GetTypeDescription returns description for a template type
func GetTypeDescription(t TemplateType) string {
	switch t {
	case TypePort:
		return "작업 단위 명세서"
	case TypeAgent:
		return "에이전트 프롬프트"
	case TypeSession:
		return "세션 기록"
	case TypeHook:
		return "Hook 스크립트"
	default:
		return ""
	}
}

// Templates

var portTemplate = `# {{.Title}}

> Port ID: {{.ID}}
> Created: {{.Date}}

---

## 컨텍스트

- **상위 요구사항**: 
- **작업 목적**: 

---

## 입력

### 선행 작업 산출물
- 

### 참조할 기존 코드
- 

---

## 작업 범위 (배타적 소유권)

### 생성/수정할 파일
` + "```" + `
# 이 포트에서 수정 가능한 파일 목록
` + "```" + `

### 구현할 기능
- [ ] 

---

## 컨벤션

### 적용할 규칙
- 

### 코드 패턴 예시
` + "```kotlin" + `
// 예시 코드
` + "```" + `

---

## 검증

### 컴파일/테스트 명령
` + "```bash" + `
# 빌드 확인
./gradlew compileKotlin

# 테스트 실행
./gradlew test
` + "```" + `

### 완료 체크리스트
- [ ] 컴파일 성공
- [ ] 테스트 통과
- [ ] 컨벤션 준수
- [ ] 코드 리뷰 준비

---

## 출력

### 완료 조건
- 

### 후속 작업에 전달할 정보
- 
`

var agentTemplate = `---
name: {{.ID}}
description: {{.Title}}
model: sonnet
color: blue
---

# {{.Title}}

## 역할
당신은 {{.Title}} 전문가입니다.

## 핵심 책임
- 

## 작업 지침

### 시작 시
1. 포트 문서를 먼저 읽습니다
2. 작업 범위를 확인합니다
3. Lock을 획득합니다

### 작업 중
1. 포트에 명시된 파일만 수정합니다
2. 컨벤션을 준수합니다
3. 불확실한 경우 에스컬레이션합니다

### 완료 시
1. 검증 명령을 실행합니다
2. Lock을 해제합니다
3. 포트 상태를 업데이트합니다

## 에스컬레이션 기준
- 포트 범위를 벗어나는 변경이 필요한 경우
- 상위 레이어 스펙 변경이 필요한 경우
- 의존성 충돌이 발생한 경우

## 사용 도구
- pal lock acquire/release
- pal port status
- pal escalation create
`

var sessionTemplate = `# 세션 기록: {{.Title}}

> Session ID: {{.ID}}
> Started: {{.Timestamp}}

---

## 목표


## 진행 상황

### 완료
- 

### 진행 중
- 

### 대기
- 

---

## 결정 사항

| 결정 | 이유 | 영향 |
|------|------|------|
| | | |

---

## 이슈 / 에스컬레이션

- 

---

## 다음 단계

1. 
`

var hookTemplate = `#!/bin/bash
# Hook: {{.Title}}
# Created: {{.Date}}
# 
# Exit codes:
#   0 - 성공
#   2 - 차단 (stderr → Claude 피드백)
#   기타 - 비차단 에러

set -e

# 입력 읽기 (stdin JSON)
INPUT=$(cat)

# 환경변수
PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"
SESSION_ID="${CLAUDE_SESSION_ID:-unknown}"

# TODO: Hook 로직 구현

exit 0
`

// GetDefaultOutputPath returns the default output path for a template type
func GetDefaultOutputPath(templateType TemplateType, id string) string {
	switch templateType {
	case TypePort:
		return fmt.Sprintf("ports/%s.md", id)
	case TypeAgent:
		return fmt.Sprintf(".claude/agents/%s.md", id)
	case TypeSession:
		return fmt.Sprintf(".claude/sessions/%s.md", id)
	case TypeHook:
		return fmt.Sprintf(".claude/hooks/%s.sh", strings.ReplaceAll(id, "-", "/"))
	default:
		return fmt.Sprintf("%s.md", id)
	}
}
