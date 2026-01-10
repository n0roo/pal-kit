package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Agent represents an agent configuration
type Agent struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"` // builder, worker, reviewer, etc.
	Prompt      string            `yaml:"prompt"`
	Tools       []string          `yaml:"tools"`
	Config      map[string]string `yaml:"config"`
	FilePath    string            `yaml:"-"` // 파일 경로 (로드 시 설정)
}

// AgentSpec is the YAML structure for agent files
type AgentSpec struct {
	Agent Agent `yaml:"agent"`
}

// Service handles agent operations
type Service struct {
	agentsDir string
	agents    map[string]*Agent
}

// NewService creates a new agent service
func NewService(projectRoot string) *Service {
	return &Service{
		agentsDir: filepath.Join(projectRoot, "agents"),
		agents:    make(map[string]*Agent),
	}
}

// EnsureDir ensures the agents directory exists
func (s *Service) EnsureDir() error {
	return os.MkdirAll(s.agentsDir, 0755)
}

// Load loads all agents from the agents directory
func (s *Service) Load() error {
	s.agents = make(map[string]*Agent)

	// 디렉토리가 없으면 빈 상태로 반환
	if _, err := os.Stat(s.agentsDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(s.agentsDir)
	if err != nil {
		return fmt.Errorf("agents 디렉토리 읽기 실패: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".md") {
			continue
		}

		filePath := filepath.Join(s.agentsDir, name)
		agent, err := s.loadAgentFile(filePath)
		if err != nil {
			// 로딩 실패한 파일은 스킵
			continue
		}

		s.agents[agent.ID] = agent
	}

	return nil
}

// loadAgentFile loads an agent from a file
func (s *Service) loadAgentFile(filePath string) (*Agent, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(filePath)
	baseName := strings.TrimSuffix(filepath.Base(filePath), ext)

	var agent Agent

	if ext == ".md" {
		// Markdown 파일: 전체를 프롬프트로 사용
		agent = Agent{
			ID:       baseName,
			Name:     baseName,
			Prompt:   string(content),
			FilePath: filePath,
		}
	} else {
		// YAML 파일
		var spec AgentSpec
		if err := yaml.Unmarshal(content, &spec); err != nil {
			return nil, fmt.Errorf("YAML 파싱 실패: %w", err)
		}
		agent = spec.Agent
		agent.FilePath = filePath

		if agent.ID == "" {
			agent.ID = baseName
		}
		if agent.Name == "" {
			agent.Name = baseName
		}
	}

	return &agent, nil
}

// Get returns an agent by ID
func (s *Service) Get(id string) (*Agent, error) {
	if len(s.agents) == 0 {
		if err := s.Load(); err != nil {
			return nil, err
		}
	}

	agent, ok := s.agents[id]
	if !ok {
		return nil, fmt.Errorf("에이전트 '%s'을(를) 찾을 수 없습니다", id)
	}

	return agent, nil
}

// List returns all loaded agents
func (s *Service) List() ([]*Agent, error) {
	if len(s.agents) == 0 {
		if err := s.Load(); err != nil {
			return nil, err
		}
	}

	var agents []*Agent
	for _, agent := range s.agents {
		agents = append(agents, agent)
	}

	return agents, nil
}

// GetPrompt returns the prompt for an agent
func (s *Service) GetPrompt(id string) (string, error) {
	agent, err := s.Get(id)
	if err != nil {
		return "", err
	}

	// 프롬프트가 파일 참조인 경우 로드
	if strings.HasPrefix(agent.Prompt, "file:") {
		promptPath := strings.TrimPrefix(agent.Prompt, "file:")
		if !filepath.IsAbs(promptPath) {
			promptPath = filepath.Join(s.agentsDir, promptPath)
		}
		content, err := os.ReadFile(promptPath)
		if err != nil {
			return "", fmt.Errorf("프롬프트 파일 로드 실패: %w", err)
		}
		return string(content), nil
	}

	return agent.Prompt, nil
}

// Create creates a new agent file
func (s *Service) Create(agent *Agent) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	spec := AgentSpec{Agent: *agent}

	content, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("YAML 생성 실패: %w", err)
	}

	filePath := filepath.Join(s.agentsDir, agent.ID+".yaml")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	agent.FilePath = filePath
	s.agents[agent.ID] = agent

	return nil
}

// Delete removes an agent file
func (s *Service) Delete(id string) error {
	agent, err := s.Get(id)
	if err != nil {
		return err
	}

	if agent.FilePath != "" {
		if err := os.Remove(agent.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("파일 삭제 실패: %w", err)
		}
	}

	delete(s.agents, id)
	return nil
}

// GetAgentTypes returns predefined agent types
func GetAgentTypes() []string {
	return []string{
		"builder",  // 파이프라인/포트 관리
		"worker",   // 실제 작업 수행
		"reviewer", // 코드 리뷰
		"planner",  // 작업 계획
		"tester",   // 테스트 작성
		"docs",     // 문서화
		"custom",   // 사용자 정의
	}
}
