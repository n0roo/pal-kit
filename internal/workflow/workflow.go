package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/config"
	"gopkg.in/yaml.v3"
)

// Context holds workflow context for a session
type Context struct {
	WorkflowType config.WorkflowType `json:"workflow_type"`
	ProjectName  string              `json:"project_name"`
	Agents       config.AgentsConfig `json:"agents"`
	ActiveAgent  string              `json:"active_agent,omitempty"`
	ActivePort   string              `json:"active_port,omitempty"`
}

// Service handles workflow operations
type Service struct {
	projectRoot string
}

// NewService creates a new workflow service
func NewService(projectRoot string) *Service {
	return &Service{
		projectRoot: projectRoot,
	}
}

// GetContext returns the current workflow context
func (s *Service) GetContext() (*Context, error) {
	cfg, err := config.LoadProjectConfig(s.projectRoot)
	if err != nil {
		// 설정 파일이 없으면 기본값 반환
		projectName := filepath.Base(s.projectRoot)
		return &Context{
			WorkflowType: config.WorkflowSimple,
			ProjectName:  projectName,
			Agents: config.AgentsConfig{
				Core: []string{"collaborator"},
			},
		}, nil
	}

	return &Context{
		WorkflowType: cfg.Workflow.Type,
		ProjectName:  cfg.Project.Name,
		Agents:       cfg.Agents,
	}, nil
}

// GenerateRulesContent generates rules content for the workflow
func (s *Service) GenerateRulesContent(ctx *Context) string {
	var sb strings.Builder

	sb.WriteString("# PAL Kit 워크플로우 컨텍스트\n\n")
	sb.WriteString(fmt.Sprintf("프로젝트: %s\n", ctx.ProjectName))
	sb.WriteString(fmt.Sprintf("워크플로우: %s\n", ctx.WorkflowType))
	sb.WriteString(fmt.Sprintf("설명: %s\n\n", config.WorkflowDescription(ctx.WorkflowType)))

	// PAL Kit 필수 사용 규칙 (모든 워크플로우 공통)
	sb.WriteString(s.generatePALKitRules())

	// 워크플로우별 가이드
	switch ctx.WorkflowType {
	case config.WorkflowSimple:
		sb.WriteString(s.generateSimpleGuide(ctx))
	case config.WorkflowSingle:
		sb.WriteString(s.generateSingleGuide(ctx))
	case config.WorkflowIntegrate:
		sb.WriteString(s.generateIntegrateGuide(ctx))
	case config.WorkflowMulti:
		sb.WriteString(s.generateMultiGuide(ctx))
	}

	// 활성 에이전트가 있으면 해당 프롬프트 로드
	if ctx.ActiveAgent != "" {
		agentPrompt := s.loadAgentPrompt(ctx.ActiveAgent)
		if agentPrompt != "" {
			sb.WriteString("\n---\n\n")
			sb.WriteString(agentPrompt)
		}
	}

	return sb.String()
}

// generatePALKitRules generates mandatory PAL Kit usage rules
func (s *Service) generatePALKitRules() string {
	return `---

## ⚠️ PAL Kit 필수 사용 규칙

> 이 프로젝트는 PAL Kit으로 관리됩니다. 아래 규칙을 **반드시** 준수하세요.

### 🔴 코드 변경 전 필수 확인

1. **활성 포트 확인**: 코드 수정 전에 활성 포트가 있는지 확인
   - 포트가 없으면 작업이 추적되지 않습니다
   - 포트 생성: ` + "`pal port create <id> --title \"작업명\"`" + `
   - 포트 활성화: ` + "`pal hook port-start <id>`" + `

2. **포트 없이 코드 수정 금지**
   - Edit/Write 도구 사용 시 포트 필수
   - 추적되지 않은 변경은 지식베이스에 기록되지 않음

### 📋 작업 흐름

` + "```" + `
1. 요청 분석 → 포트 필요 여부 판단
2. 포트 생성/활성화 → pal hook port-start <id>
3. 작업 수행 → 코드 변경, 테스트
4. 완료 → pal hook port-end <id>
5. 커밋 → 변경사항 기록
` + "```" + `

### 🎯 이벤트 기록

중요한 결정이나 에스컬레이션은 반드시 기록:
- 결정 사항: ` + "`pal hook event decision \"결정 내용\"`" + `
- 에스컬레이션: ` + "`pal hook event escalation \"에스컬레이션 내용\"`" + `

### 📊 상태 확인 명령어

- ` + "`pal status`" + ` - 전체 상태
- ` + "`pal port list`" + ` - 포트 목록
- ` + "`pal hook events`" + ` - 세션 이벤트

---

`
}

func (s *Service) generateSimpleGuide(ctx *Context) string {
	return `## Simple 워크플로우

당신은 **Collaborator** 역할입니다.

### 작업 방식
- 사용자와 대화하며 협업
- 모든 역할(코딩, 리뷰, 테스트)을 종합 수행
- 사용자가 코드와 Git을 직접 관리

### 권장 행동
1. 요청을 이해하고 명확화
2. 작업 범위 확인 후 진행
3. 변경 전 사용자 확인
4. 완료 후 결과 요약

### 사용 가능한 명령어
- ` + "`pal status`" + ` - 현재 상태 확인
- ` + "`pal port create <id>`" + ` - 작업 추적용 포트 생성 (선택)
`
}

func (s *Service) generateSingleGuide(ctx *Context) string {
	agents := strings.Join(ctx.Agents.Core, ", ")
	return fmt.Sprintf(`## Single 워크플로우

### 역할 전환
하나의 세션에서 여러 역할을 순차적으로 수행합니다.

사용 가능한 역할: %s

### 작업 흐름
1. **Builder**: 요구사항 분석 → 포트 분해
2. **Planner**: 실행 순서 계획
3. **Architect**: 기술 결정 (필요시)
4. **Worker**: 실제 구현
5. **Tester**: 테스트 작성
6. **Logger**: 커밋/문서화

### 포트 기반 작업
- 모든 작업은 포트 단위로 추적
- 포트 명세: ports/<id>.md
- 완료 조건 체크리스트 필수

### 명령어
- `+"`pal port create <id> --title \"제목\"`"+` - 포트 생성
- `+"`pal port status <id>`"+` - 포트 상태
- `+"`pal hook port-start <id>`"+` - 포트 작업 시작
- `+"`pal hook port-end <id>`"+` - 포트 작업 완료
`, agents)
}

func (s *Service) generateIntegrateGuide(ctx *Context) string {
	agents := strings.Join(ctx.Agents.Core, ", ")
	workers := "없음"
	if len(ctx.Agents.Workers) > 0 {
		workers = strings.Join(ctx.Agents.Workers, ", ")
	}

	return fmt.Sprintf(`## Integrate 워크플로우

### 역할 구조
- **빌더 세션**: 전체 관리 (현재 세션)
- **워커 세션**: 개별 포트 작업 (서브세션)

Core 역할: %s
Worker 역할: %s

### 빌더 역할
1. 요구사항 분석 및 포트 분해
2. 파이프라인 구성
3. 워커 세션 spawn 및 관리
4. 품질 게이트 운영
5. 에스컬레이션 처리

### 파이프라인 관리
- `+"`pal pipeline create <n>`"+` - 파이프라인 생성
- `+"`pal pipeline add <n> <port>`"+` - 포트 추가
- `+"`pal pl plan <n>`"+` - 실행 계획 확인

### 워커 세션 생성
- `+"`pal session start --type sub --port <id>`"+`
- 또는 Claude 새 창에서 해당 포트 작업

### 품질 체크
- 포트 완료조건 충족
- 빌드/테스트 통과
- 컨벤션 준수
`, agents, workers)
}

func (s *Service) generateMultiGuide(ctx *Context) string {
	return `## Multi 워크플로우

### 대규모 프로젝트 구조
- 복수의 Integrate 워크플로우 병렬 운영
- 각 서브 프로젝트별 빌더 세션
- 전체 조율 세션

### 이 세션의 역할
전체 프로젝트 조율 또는 특정 서브 프로젝트 관리

### 조율 작업
1. 서브 프로젝트 간 의존성 관리
2. 통합 지점 조율
3. 전체 진행 상황 모니터링

### 명령어
- ` + "`pal status`" + ` - 전체 상태
- ` + "`pal session list`" + ` - 활성 세션
- ` + "`pal pipeline list`" + ` - 파이프라인 현황
`
}

// loadAgentPrompt loads an agent's prompt from the agents directory
func (s *Service) loadAgentPrompt(agentID string) string {
	// 프로젝트 에이전트 먼저 확인
	projectAgentPath := filepath.Join(s.projectRoot, "agents", agentID+".yaml")
	if content := s.readAgentPrompt(projectAgentPath); content != "" {
		return content
	}

	// 전역 에이전트 확인
	globalAgentPath := filepath.Join(config.GlobalAgentsDir(), "core", agentID+".yaml")
	if content := s.readAgentPrompt(globalAgentPath); content != "" {
		return content
	}

	return ""
}

func (s *Service) readAgentPrompt(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var spec struct {
		Agent struct {
			Prompt string `yaml:"prompt"`
		} `yaml:"agent"`
	}

	if err := yaml.Unmarshal(data, &spec); err != nil {
		return ""
	}

	return spec.Agent.Prompt
}

// WriteRulesFile writes workflow rules to .claude/rules/
func (s *Service) WriteRulesFile(ctx *Context) error {
	if s.projectRoot == "" {
		return fmt.Errorf("projectRoot가 비어있습니다")
	}

	rulesDir := filepath.Join(s.projectRoot, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("rules 디렉토리 생성 실패: %w", err)
	}

	content := s.GenerateRulesContent(ctx)
	rulesPath := filepath.Join(rulesDir, "workflow.md")

	if err := os.WriteFile(rulesPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("rules 파일 작성 실패: %w", err)
	}

	return nil
}

// GetRulesPath returns the workflow rules file path
func (s *Service) GetRulesPath() string {
	return filepath.Join(s.projectRoot, ".claude", "rules", "workflow.md")
}

// CleanupRulesFile removes the workflow rules file
func (s *Service) CleanupRulesFile() error {
	rulesPath := s.GetRulesPath()
	if err := os.Remove(rulesPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SetActiveAgent sets the active agent and regenerates rules
func (s *Service) SetActiveAgent(agentID string) error {
	ctx, err := s.GetContext()
	if err != nil {
		return err
	}

	ctx.ActiveAgent = agentID
	return s.WriteRulesFile(ctx)
}

// SetActivePort sets the active port
func (s *Service) SetActivePort(portID string) error {
	ctx, err := s.GetContext()
	if err != nil {
		return err
	}

	ctx.ActivePort = portID
	return s.WriteRulesFile(ctx)
}
