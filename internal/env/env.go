package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
)

// PathVariables defines environment-specific path mappings
type PathVariables struct {
	Workspace  string `json:"workspace" yaml:"workspace"`
	ClaudeData string `json:"claude_data" yaml:"claude_data"`
	Home       string `json:"home" yaml:"home"`
}

// MatchCondition defines how to detect an environment
type MatchCondition struct {
	Hostname   string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	PathExists string `json:"path_exists,omitempty" yaml:"path_exists,omitempty"`
}

// Environment represents a single environment profile
type Environment struct {
	ID         string         `json:"id" yaml:"id"`
	Name       string         `json:"name" yaml:"name"`
	Hostname   string         `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Paths      PathVariables  `json:"paths" yaml:"paths"`
	Match      MatchCondition `json:"match,omitempty" yaml:"match,omitempty"`
	IsCurrent  bool           `json:"is_current" yaml:"is_current"`
	CreatedAt  time.Time      `json:"created_at" yaml:"created_at"`
	LastActive time.Time      `json:"last_active,omitempty" yaml:"last_active,omitempty"`
}

// Config represents the environments.yaml file
type Config struct {
	Version      int                    `yaml:"version"`
	Current      string                 `yaml:"current"`
	Environments map[string]Environment `yaml:"environments"`
}

// Service manages environment profiles
type Service struct {
	db         *db.DB
	configPath string
	config     *Config
}

// NewService creates a new environment service
func NewService(database *db.DB) *Service {
	return &Service{
		db:         database,
		configPath: filepath.Join(config.GlobalDir(), "environments.yaml"),
	}
}

// Init initializes the environment config file if not exists
func (s *Service) Init() error {
	if _, err := os.Stat(s.configPath); os.IsNotExist(err) {
		s.config = &Config{
			Version:      1,
			Current:      "",
			Environments: make(map[string]Environment),
		}
		return s.saveConfig()
	}
	return s.loadConfig()
}

// loadConfig loads the environments.yaml file
func (s *Service) loadConfig() error {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return fmt.Errorf("환경 설정 읽기 실패: %w", err)
	}

	s.config = &Config{}
	if err := yaml.Unmarshal(data, s.config); err != nil {
		return fmt.Errorf("환경 설정 파싱 실패: %w", err)
	}

	if s.config.Environments == nil {
		s.config.Environments = make(map[string]Environment)
	}

	return nil
}

// saveConfig saves the environments.yaml file
func (s *Service) saveConfig() error {
	data, err := yaml.Marshal(s.config)
	if err != nil {
		return fmt.Errorf("환경 설정 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("환경 설정 저장 실패: %w", err)
	}

	return nil
}

// Setup creates or updates an environment profile for the current machine
func (s *Service) Setup(name string, paths PathVariables) (*Environment, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()

	// Check if environment with this name exists
	if existing, ok := s.config.Environments[name]; ok {
		// Update existing
		existing.Hostname = hostname
		existing.Paths = paths
		existing.Match = MatchCondition{Hostname: hostname}
		existing.LastActive = time.Now()
		s.config.Environments[name] = existing

		if err := s.updateDB(&existing); err != nil {
			return nil, err
		}
		if err := s.saveConfig(); err != nil {
			return nil, err
		}
		return &existing, nil
	}

	// Create new environment
	env := Environment{
		ID:        uuid.New().String()[:8],
		Name:      name,
		Hostname:  hostname,
		Paths:     paths,
		Match:     MatchCondition{Hostname: hostname},
		IsCurrent: len(s.config.Environments) == 0, // First env is current
		CreatedAt: time.Now(),
	}

	s.config.Environments[name] = env

	if env.IsCurrent {
		s.config.Current = name
	}

	if err := s.insertDB(&env); err != nil {
		return nil, err
	}

	if err := s.saveConfig(); err != nil {
		return nil, err
	}

	return &env, nil
}

// insertDB inserts environment into database
func (s *Service) insertDB(env *Environment) error {
	pathsJSON, _ := json.Marshal(env.Paths)
	isCurrent := 0
	if env.IsCurrent {
		isCurrent = 1
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO environments (id, name, hostname, paths, is_current, created_at, last_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, env.ID, env.Name, env.Hostname, string(pathsJSON), isCurrent, env.CreatedAt, time.Now())

	return err
}

// updateDB updates environment in database
func (s *Service) updateDB(env *Environment) error {
	pathsJSON, _ := json.Marshal(env.Paths)
	isCurrent := 0
	if env.IsCurrent {
		isCurrent = 1
	}

	_, err := s.db.Exec(`
		UPDATE environments
		SET hostname = ?, paths = ?, is_current = ?, last_active = ?
		WHERE name = ?
	`, env.Hostname, string(pathsJSON), isCurrent, time.Now(), env.Name)

	return err
}

// List returns all registered environments
func (s *Service) List() ([]Environment, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}

	var envs []Environment
	for _, env := range s.config.Environments {
		envs = append(envs, env)
	}
	return envs, nil
}

// Current returns the current environment
func (s *Service) Current() (*Environment, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}

	if s.config.Current == "" {
		return nil, fmt.Errorf("현재 환경이 설정되지 않음")
	}

	if env, ok := s.config.Environments[s.config.Current]; ok {
		return &env, nil
	}

	return nil, fmt.Errorf("현재 환경을 찾을 수 없음: %s", s.config.Current)
}

// Switch changes the current environment
func (s *Service) Switch(name string) error {
	if err := s.Init(); err != nil {
		return err
	}

	if _, ok := s.config.Environments[name]; !ok {
		return fmt.Errorf("환경을 찾을 수 없음: %s", name)
	}

	// Update is_current for all environments
	for n, env := range s.config.Environments {
		env.IsCurrent = (n == name)
		if env.IsCurrent {
			env.LastActive = time.Now()
		}
		s.config.Environments[n] = env

		if err := s.updateDB(&env); err != nil {
			return err
		}
	}

	s.config.Current = name
	return s.saveConfig()
}

// Detect attempts to auto-detect the current environment
func (s *Service) Detect() (*Environment, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()

	for _, env := range s.config.Environments {
		// Check hostname match
		if env.Match.Hostname != "" && env.Match.Hostname == hostname {
			return &env, nil
		}

		// Check path exists match
		if env.Match.PathExists != "" {
			if _, err := os.Stat(env.Match.PathExists); err == nil {
				return &env, nil
			}
		}

		// Fallback: check if hostname matches
		if env.Hostname == hostname {
			return &env, nil
		}
	}

	return nil, fmt.Errorf("현재 환경을 감지할 수 없음 (hostname: %s)", hostname)
}

// AutoSwitch detects and switches to the appropriate environment
func (s *Service) AutoSwitch() (*Environment, error) {
	detected, err := s.Detect()
	if err != nil {
		return nil, err
	}

	if err := s.Switch(detected.Name); err != nil {
		return nil, err
	}

	return detected, nil
}

// Delete removes an environment
func (s *Service) Delete(name string) error {
	if err := s.Init(); err != nil {
		return err
	}

	env, ok := s.config.Environments[name]
	if !ok {
		return fmt.Errorf("환경을 찾을 수 없음: %s", name)
	}

	// Cannot delete current environment
	if env.IsCurrent {
		return fmt.Errorf("현재 환경은 삭제할 수 없음")
	}

	delete(s.config.Environments, name)

	// Delete from DB
	_, err := s.db.Exec(`DELETE FROM environments WHERE name = ?`, name)
	if err != nil {
		return err
	}

	return s.saveConfig()
}

// Get returns a specific environment by name
func (s *Service) Get(name string) (*Environment, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}

	if env, ok := s.config.Environments[name]; ok {
		return &env, nil
	}

	return nil, fmt.Errorf("환경을 찾을 수 없음: %s", name)
}

// GetCurrentID returns the current environment ID
func (s *Service) GetCurrentID() (string, error) {
	current, err := s.Current()
	if err != nil {
		return "", err
	}
	return current.ID, nil
}

// DefaultPaths returns suggested default paths for the current system
func DefaultPaths() PathVariables {
	home, _ := os.UserHomeDir()
	return PathVariables{
		Workspace:  filepath.Join(home, "playground"),
		ClaudeData: filepath.Join(home, ".claude"),
		Home:       home,
	}
}

// SuggestName suggests an environment name based on hostname
func SuggestName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "default"
	}
	return hostname
}
