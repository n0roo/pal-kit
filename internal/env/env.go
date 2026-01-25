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

// ProjectMapping defines a project's path in an environment
type ProjectMapping struct {
	Path    string `json:"path" yaml:"path"`       // Absolute path to project
	DocsRef string `json:"docs" yaml:"docs"`       // Reference to docs vault ID
}

// DocsVault defines a docs vault path in an environment
type DocsVault struct {
	Path string `json:"path" yaml:"path"` // Absolute path to vault
}

// MatchCondition defines how to detect an environment
type MatchCondition struct {
	Hostname   string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	PathExists string `json:"path_exists,omitempty" yaml:"path_exists,omitempty"`
}

// Environment represents a single environment profile
type Environment struct {
	ID         string                    `json:"id" yaml:"id"`
	Name       string                    `json:"name" yaml:"name"`
	Hostname   string                    `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Paths      PathVariables             `json:"paths" yaml:"paths"`
	Projects   map[string]ProjectMapping `json:"projects,omitempty" yaml:"projects,omitempty"`
	Docs       map[string]DocsVault      `json:"docs,omitempty" yaml:"docs,omitempty"`
	Match      MatchCondition            `json:"match,omitempty" yaml:"match,omitempty"`
	IsCurrent  bool                      `json:"is_current" yaml:"is_current"`
	CreatedAt  time.Time                 `json:"created_at" yaml:"created_at"`
	LastActive time.Time                 `json:"last_active,omitempty" yaml:"last_active,omitempty"`
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
		Projects:  make(map[string]ProjectMapping),
		Docs:      make(map[string]DocsVault),
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

// ============================================
// Project Management
// ============================================

// AddProject adds a project mapping to the current environment
func (s *Service) AddProject(projectID, projectPath, docsRef string) error {
	if err := s.Init(); err != nil {
		return err
	}

	current, err := s.Current()
	if err != nil {
		return err
	}

	// Expand path
	expandedPath := expandTildePath(projectPath)

	if current.Projects == nil {
		current.Projects = make(map[string]ProjectMapping)
	}

	current.Projects[projectID] = ProjectMapping{
		Path:    expandedPath,
		DocsRef: docsRef,
	}

	s.config.Environments[current.Name] = *current
	return s.saveConfig()
}

// RemoveProject removes a project from the current environment
func (s *Service) RemoveProject(projectID string) error {
	if err := s.Init(); err != nil {
		return err
	}

	current, err := s.Current()
	if err != nil {
		return err
	}

	if current.Projects == nil {
		return fmt.Errorf("프로젝트를 찾을 수 없음: %s", projectID)
	}

	if _, exists := current.Projects[projectID]; !exists {
		return fmt.Errorf("프로젝트를 찾을 수 없음: %s", projectID)
	}

	delete(current.Projects, projectID)
	s.config.Environments[current.Name] = *current
	return s.saveConfig()
}

// GetProjectPath returns the project path for the current environment
func (s *Service) GetProjectPath(projectID string) (string, error) {
	if err := s.Init(); err != nil {
		return "", err
	}

	current, err := s.Current()
	if err != nil {
		return "", err
	}

	if current.Projects == nil {
		return "", fmt.Errorf("프로젝트를 찾을 수 없음: %s", projectID)
	}

	proj, exists := current.Projects[projectID]
	if !exists {
		return "", fmt.Errorf("프로젝트를 찾을 수 없음: %s", projectID)
	}

	return proj.Path, nil
}

// ListProjects returns all projects for the current environment
func (s *Service) ListProjects() (map[string]ProjectMapping, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}

	current, err := s.Current()
	if err != nil {
		return nil, err
	}

	if current.Projects == nil {
		return make(map[string]ProjectMapping), nil
	}

	return current.Projects, nil
}

// ============================================
// Docs Vault Management
// ============================================

// AddDocs adds a docs vault to the current environment
func (s *Service) AddDocs(docsID, vaultPath string) error {
	if err := s.Init(); err != nil {
		return err
	}

	current, err := s.Current()
	if err != nil {
		return err
	}

	// Expand path
	expandedPath := expandTildePath(vaultPath)

	if current.Docs == nil {
		current.Docs = make(map[string]DocsVault)
	}

	current.Docs[docsID] = DocsVault{
		Path: expandedPath,
	}

	s.config.Environments[current.Name] = *current
	return s.saveConfig()
}

// RemoveDocs removes a docs vault from the current environment
func (s *Service) RemoveDocs(docsID string) error {
	if err := s.Init(); err != nil {
		return err
	}

	current, err := s.Current()
	if err != nil {
		return err
	}

	if current.Docs == nil {
		return fmt.Errorf("Docs를 찾을 수 없음: %s", docsID)
	}

	if _, exists := current.Docs[docsID]; !exists {
		return fmt.Errorf("Docs를 찾을 수 없음: %s", docsID)
	}

	delete(current.Docs, docsID)
	s.config.Environments[current.Name] = *current
	return s.saveConfig()
}

// GetDocsPath returns the docs vault path for the current environment
func (s *Service) GetDocsPath(docsID string) (string, error) {
	if err := s.Init(); err != nil {
		return "", err
	}

	current, err := s.Current()
	if err != nil {
		return "", err
	}

	if current.Docs == nil {
		return "", fmt.Errorf("Docs를 찾을 수 없음: %s", docsID)
	}

	vault, exists := current.Docs[docsID]
	if !exists {
		return "", fmt.Errorf("Docs를 찾을 수 없음: %s", docsID)
	}

	return vault.Path, nil
}

// ListDocs returns all docs vaults for the current environment
func (s *Service) ListDocs() (map[string]DocsVault, error) {
	if err := s.Init(); err != nil {
		return nil, err
	}

	current, err := s.Current()
	if err != nil {
		return nil, err
	}

	if current.Docs == nil {
		return make(map[string]DocsVault), nil
	}

	return current.Docs, nil
}

// LinkProjectToDocs links a project to a docs vault
func (s *Service) LinkProjectToDocs(projectID, docsID string) error {
	if err := s.Init(); err != nil {
		return err
	}

	current, err := s.Current()
	if err != nil {
		return err
	}

	// Check project exists
	if current.Projects == nil {
		return fmt.Errorf("프로젝트를 찾을 수 없음: %s", projectID)
	}
	proj, exists := current.Projects[projectID]
	if !exists {
		return fmt.Errorf("프로젝트를 찾을 수 없음: %s", projectID)
	}

	// Check docs exists
	if current.Docs == nil {
		return fmt.Errorf("Docs를 찾을 수 없음: %s", docsID)
	}
	if _, exists := current.Docs[docsID]; !exists {
		return fmt.Errorf("Docs를 찾을 수 없음: %s", docsID)
	}

	// Link
	proj.DocsRef = docsID
	current.Projects[projectID] = proj
	s.config.Environments[current.Name] = *current
	return s.saveConfig()
}

// GetProjectWithDocs returns project info with resolved docs path
func (s *Service) GetProjectWithDocs(projectID string) (projectPath string, docsPath string, err error) {
	if err := s.Init(); err != nil {
		return "", "", err
	}

	current, err := s.Current()
	if err != nil {
		return "", "", err
	}

	if current.Projects == nil {
		return "", "", fmt.Errorf("프로젝트를 찾을 수 없음: %s", projectID)
	}

	proj, exists := current.Projects[projectID]
	if !exists {
		return "", "", fmt.Errorf("프로젝트를 찾을 수 없음: %s", projectID)
	}

	projectPath = proj.Path

	// Resolve docs path if linked
	if proj.DocsRef != "" && current.Docs != nil {
		if vault, ok := current.Docs[proj.DocsRef]; ok {
			docsPath = vault.Path
		}
	}

	return projectPath, docsPath, nil
}

// ============================================
// Settings Local Generation
// ============================================

// SettingsLocal represents .claude/settings.local.json
type SettingsLocal struct {
	MCPServers            map[string]MCPServerConfig `json:"mcpServers,omitempty"`
	AdditionalDirectories []string                   `json:"additionalDirectories,omitempty"`
}

// MCPServerConfig represents MCP server configuration
type MCPServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// GenerateSettingsLocal generates .claude/settings.local.json for a project
func (s *Service) GenerateSettingsLocal(projectID string) (string, error) {
	projectPath, docsPath, err := s.GetProjectWithDocs(projectID)
	if err != nil {
		return "", err
	}

	settings := SettingsLocal{
		MCPServers:            make(map[string]MCPServerConfig),
		AdditionalDirectories: []string{},
	}

	// Add pa-retriever MCP server if docs path exists
	if docsPath != "" {
		settings.MCPServers["pa-retriever"] = MCPServerConfig{
			Command: "npx",
			Args:    []string{"-y", "@anthropics/mcp-pa-retriever", docsPath},
		}
		settings.AdditionalDirectories = append(settings.AdditionalDirectories, docsPath)
	}

	// Generate JSON
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("settings.local.json 생성 실패: %w", err)
	}

	// Write to project's .claude directory
	settingsPath := filepath.Join(projectPath, ".claude", "settings.local.json")

	// Ensure .claude directory exists
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return "", fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return "", fmt.Errorf("settings.local.json 저장 실패: %w", err)
	}

	return settingsPath, nil
}

// GenerateAllSettingsLocal generates settings.local.json for all projects in current env
func (s *Service) GenerateAllSettingsLocal() ([]string, error) {
	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}

	var generated []string
	for projectID := range projects {
		path, err := s.GenerateSettingsLocal(projectID)
		if err != nil {
			// Log but continue
			fmt.Fprintf(os.Stderr, "경고: %s 설정 생성 실패: %v\n", projectID, err)
			continue
		}
		generated = append(generated, path)
	}

	return generated, nil
}

// ============================================
// Helpers
// ============================================

// expandTildePath expands ~ to home directory
func expandTildePath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
