package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// WorkflowType represents the workflow type
type WorkflowType string

const (
	WorkflowSimple    WorkflowType = "simple"
	WorkflowSingle    WorkflowType = "single"
	WorkflowIntegrate WorkflowType = "integrate"
	WorkflowMulti     WorkflowType = "multi"
)

// ProjectConfig represents .pal/config.yaml
type ProjectConfig struct {
	Version  string          `yaml:"version"`
	Project  ProjectInfo     `yaml:"project"`
	Workflow WorkflowConfig  `yaml:"workflow"`
	Agents   AgentsConfig    `yaml:"agents"`
	Settings ProjectSettings `yaml:"settings"`
}

// ProjectInfo holds project metadata
type ProjectInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Root        string `yaml:"root,omitempty"`
}

// WorkflowConfig holds workflow settings
type WorkflowConfig struct {
	Type WorkflowType `yaml:"type"`
}

// AgentsConfig holds agent configuration
type AgentsConfig struct {
	Core    []string `yaml:"core,omitempty"`
	Workers []string `yaml:"workers,omitempty"`
	Testers []string `yaml:"testers,omitempty"`
}

// ProjectSettings holds project-level settings
type ProjectSettings struct {
	AutoPortCreate    bool `yaml:"auto_port_create"`
	RequireUserReview bool `yaml:"require_user_review"`
	AutoTestOnComplete bool `yaml:"auto_test_on_complete"`
}

// DefaultProjectConfig returns a default config
func DefaultProjectConfig(projectName string) *ProjectConfig {
	return &ProjectConfig{
		Version: "0.4.0",
		Project: ProjectInfo{
			Name: projectName,
		},
		Workflow: WorkflowConfig{
			Type: WorkflowSimple,
		},
		Agents: AgentsConfig{
			Core:    []string{"collaborator"},
			Workers: []string{},
			Testers: []string{},
		},
		Settings: ProjectSettings{
			AutoPortCreate:    true,
			RequireUserReview: true,
			AutoTestOnComplete: true,
		},
	}
}

// ProjectConfigPath returns the config file path for a project
func ProjectConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".pal", "config.yaml")
}

// LoadProjectConfig loads config from .pal/config.yaml
func LoadProjectConfig(projectRoot string) (*ProjectConfig, error) {
	configPath := ProjectConfigPath(projectRoot)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("프로젝트 설정이 없습니다. Claude에게 'PAL Kit 환경을 설정해줘'라고 요청하세요")
		}
		return nil, fmt.Errorf("설정 파일 읽기 실패: %w", err)
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("설정 파일 파싱 실패: %w", err)
	}

	return &config, nil
}

// SaveProjectConfig saves config to .pal/config.yaml
func SaveProjectConfig(projectRoot string, config *ProjectConfig) error {
	configPath := ProjectConfigPath(projectRoot)

	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("설정 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("설정 파일 저장 실패: %w", err)
	}

	return nil
}

// HasProjectConfig checks if project config exists
func HasProjectConfig(projectRoot string) bool {
	_, err := os.Stat(ProjectConfigPath(projectRoot))
	return err == nil
}

// GetWorkflowTypes returns all workflow types
func GetWorkflowTypes() []WorkflowType {
	return []WorkflowType{
		WorkflowSimple,
		WorkflowSingle,
		WorkflowIntegrate,
		WorkflowMulti,
	}
}

// WorkflowDescription returns a description for the workflow type
func WorkflowDescription(wt WorkflowType) string {
	switch wt {
	case WorkflowSimple:
		return "대화형 협업, 종합 에이전트 (간단한 작업, 학습)"
	case WorkflowSingle:
		return "단일 세션, 역할 전환 (중간 규모 기능)"
	case WorkflowIntegrate:
		return "빌더 관리, 서브세션 (복잡한 기능, 여러 기술)"
	case WorkflowMulti:
		return "복수 integrate (대규모 프로젝트)"
	default:
		return ""
	}
}

// DefaultAgentsForWorkflow returns default agents for a workflow type
func DefaultAgentsForWorkflow(wt WorkflowType) AgentsConfig {
	switch wt {
	case WorkflowSimple:
		return AgentsConfig{
			Core: []string{"collaborator"},
		}
	case WorkflowSingle:
		return AgentsConfig{
			Core: []string{"builder", "planner", "architect", "tester", "logger"},
		}
	case WorkflowIntegrate:
		return AgentsConfig{
			Core: []string{"builder", "planner", "architect", "manager", "tester", "logger"},
		}
	case WorkflowMulti:
		return AgentsConfig{
			Core: []string{"builder", "planner", "architect", "manager", "tester", "logger"},
		}
	default:
		return AgentsConfig{
			Core: []string{"collaborator"},
		}
	}
}
