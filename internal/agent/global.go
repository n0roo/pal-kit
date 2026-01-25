package agent

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GlobalAgentStore manages agents in the global directory (~/.pal/agents)
type GlobalAgentStore struct {
	basePath string
}

// AgentInfo represents agent metadata
type AgentInfo struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // core, worker, skill
	Category    string    `json:"category"`
	Description string    `json:"description,omitempty"`
	HasRules    bool      `json:"has_rules"`
	ModifiedAt  time.Time `json:"modified_at"`
	Size        int64     `json:"size"`
}

// GlobalManifest tracks global agent versions
type GlobalManifest struct {
	Version       string            `yaml:"version"`
	InitializedAt time.Time         `yaml:"initialized_at"`
	LastUpdated   time.Time         `yaml:"last_updated"`
	EmbeddedHash  string            `yaml:"embedded_hash,omitempty"`
	CustomAgents  []string          `yaml:"custom_agents,omitempty"`
	Overrides     map[string]string `yaml:"overrides,omitempty"`
}

// NewGlobalAgentStore creates a new global agent store
func NewGlobalAgentStore(basePath string) *GlobalAgentStore {
	return &GlobalAgentStore{basePath: basePath}
}

// IsInitialized checks if global agents are initialized
func (s *GlobalAgentStore) IsInitialized() bool {
	manifestPath := filepath.Join(s.basePath, "manifest.yaml")
	_, err := os.Stat(manifestPath)
	return err == nil
}

// Initialize copies embedded templates to global directory
func (s *GlobalAgentStore) Initialize(force bool) error {
	if s.IsInitialized() && !force {
		return nil // Already initialized
	}

	// Create base directories
	dirs := []string{
		s.basePath,
		filepath.Join(s.basePath, "core"),
		filepath.Join(s.basePath, "workers"),
		filepath.Join(s.basePath, "skills"),
		filepath.Join(s.basePath, "conventions"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("디렉토리 생성 실패 %s: %w", dir, err)
		}
	}

	// Copy embedded templates
	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(s.basePath, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Skip if exists and not forcing
		if _, err := os.Stat(targetPath); err == nil && !force {
			return nil
		}

		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("템플릿 읽기 실패 %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("템플릿 쓰기 실패 %s: %w", targetPath, err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Create manifest
	manifest := GlobalManifest{
		Version:       "1.0.0",
		InitializedAt: time.Now(),
		LastUpdated:   time.Now(),
		CustomAgents:  []string{},
		Overrides:     map[string]string{},
	}

	return s.saveManifest(&manifest)
}

// List returns all agents in the global store
func (s *GlobalAgentStore) List() ([]AgentInfo, error) {
	var agents []AgentInfo

	// Walk agents directory
	agentsPath := filepath.Join(s.basePath, "agents")
	if _, err := os.Stat(agentsPath); err == nil {
		err := filepath.WalkDir(agentsPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			if d.IsDir() {
				return nil
			}

			// Only process .yaml files
			if !strings.HasSuffix(path, ".yaml") {
				return nil
			}

			relPath, _ := filepath.Rel(s.basePath, path)
			info, err := s.getAgentInfo(path, relPath)
			if err == nil {
				agents = append(agents, *info)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return agents, nil
}

// ListSkills returns all skills
func (s *GlobalAgentStore) ListSkills() ([]AgentInfo, error) {
	var skills []AgentInfo

	skillsPath := filepath.Join(s.basePath, "agents", "skills")
	if _, err := os.Stat(skillsPath); err != nil {
		return skills, nil
	}

	err := filepath.WalkDir(skillsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		relPath, _ := filepath.Rel(s.basePath, path)
		info, err := d.Info()
		if err != nil {
			return nil
		}

		skills = append(skills, AgentInfo{
			Path:       relPath,
			Name:       strings.TrimSuffix(d.Name(), ".md"),
			Type:       "skill",
			Category:   filepath.Base(filepath.Dir(path)),
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
		})

		return nil
	})

	return skills, err
}

// ListConventions returns all conventions
func (s *GlobalAgentStore) ListConventions() ([]AgentInfo, error) {
	var conventions []AgentInfo

	convPath := filepath.Join(s.basePath, "conventions")
	if _, err := os.Stat(convPath); err != nil {
		return conventions, nil
	}

	err := filepath.WalkDir(convPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		relPath, _ := filepath.Rel(s.basePath, path)
		info, err := d.Info()
		if err != nil {
			return nil
		}

		conventions = append(conventions, AgentInfo{
			Path:       relPath,
			Name:       strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
			Type:       "convention",
			Category:   filepath.Base(filepath.Dir(path)),
			ModifiedAt: info.ModTime(),
			Size:       info.Size(),
		})

		return nil
	})

	return conventions, err
}

// Read reads an agent file
func (s *GlobalAgentStore) Read(relPath string) ([]byte, error) {
	fullPath := filepath.Join(s.basePath, relPath)

	// Security: ensure path is within basePath
	if !strings.HasPrefix(fullPath, s.basePath) {
		return nil, fmt.Errorf("잘못된 경로: %s", relPath)
	}

	return os.ReadFile(fullPath)
}

// Write writes an agent file
func (s *GlobalAgentStore) Write(relPath string, content []byte) error {
	fullPath := filepath.Join(s.basePath, relPath)

	// Security: ensure path is within basePath
	if !strings.HasPrefix(fullPath, s.basePath) {
		return fmt.Errorf("잘못된 경로: %s", relPath)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	// Write file
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return err
	}

	// Update manifest
	manifest, err := s.loadManifest()
	if err == nil {
		manifest.LastUpdated = time.Now()
		s.saveManifest(manifest)
	}

	return nil
}

// Delete deletes an agent file
func (s *GlobalAgentStore) Delete(relPath string) error {
	fullPath := filepath.Join(s.basePath, relPath)

	// Security: ensure path is within basePath
	if !strings.HasPrefix(fullPath, s.basePath) {
		return fmt.Errorf("잘못된 경로: %s", relPath)
	}

	return os.Remove(fullPath)
}

// SyncToProject copies global agents to a project
func (s *GlobalAgentStore) SyncToProject(projectRoot string, forceOverwrite bool) (int, error) {
	copied := 0

	// Sync agents
	agentsPath := filepath.Join(s.basePath, "agents")
	if _, err := os.Stat(agentsPath); err == nil {
		err := filepath.WalkDir(agentsPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			relPath, _ := filepath.Rel(s.basePath, path)
			targetPath := filepath.Join(projectRoot, relPath)

			// Skip if exists and not forcing
			if _, err := os.Stat(targetPath); err == nil && !forceOverwrite {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return nil
			}

			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return nil
			}

			copied++
			return nil
		})
		if err != nil {
			return copied, err
		}
	}

	// Sync conventions
	convPath := filepath.Join(s.basePath, "conventions")
	if _, err := os.Stat(convPath); err == nil {
		err := filepath.WalkDir(convPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}

			relPath, _ := filepath.Rel(s.basePath, path)
			targetPath := filepath.Join(projectRoot, relPath)

			if _, err := os.Stat(targetPath); err == nil && !forceOverwrite {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return nil
			}

			if err := os.WriteFile(targetPath, content, 0644); err != nil {
				return nil
			}

			copied++
			return nil
		})
		if err != nil {
			return copied, err
		}
	}

	return copied, nil
}

// GetManifest returns the global manifest
func (s *GlobalAgentStore) GetManifest() (*GlobalManifest, error) {
	return s.loadManifest()
}

// private helpers

func (s *GlobalAgentStore) getAgentInfo(fullPath, relPath string) (*AgentInfo, error) {
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	// Parse YAML to get metadata
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	var agentYAML struct {
		Agent struct {
			ID          string `yaml:"id"`
			Name        string `yaml:"name"`
			Type        string `yaml:"type"`
			Description string `yaml:"description"`
		} `yaml:"agent"`
	}

	yaml.Unmarshal(content, &agentYAML)

	// Check for rules file
	rulesPath := strings.TrimSuffix(fullPath, ".yaml") + ".rules.md"
	_, hasRules := os.Stat(rulesPath)

	// Determine category from path
	parts := strings.Split(relPath, string(os.PathSeparator))
	category := ""
	if len(parts) >= 3 {
		category = parts[1] // agents/core/... -> core
	}

	return &AgentInfo{
		Path:        relPath,
		Name:        agentYAML.Agent.Name,
		Type:        agentYAML.Agent.Type,
		Category:    category,
		Description: agentYAML.Agent.Description,
		HasRules:    hasRules == nil,
		ModifiedAt:  info.ModTime(),
		Size:        info.Size(),
	}, nil
}

func (s *GlobalAgentStore) loadManifest() (*GlobalManifest, error) {
	manifestPath := filepath.Join(s.basePath, "manifest.yaml")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest GlobalManifest
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func (s *GlobalAgentStore) saveManifest(manifest *GlobalManifest) error {
	manifestPath := filepath.Join(s.basePath, "manifest.yaml")
	content, err := yaml.Marshal(manifest)
	if err != nil {
		return err
	}

	return os.WriteFile(manifestPath, content, 0644)
}

// InstallFromGlobal installs agents from global store to project
// This replaces the embedded template installation
func InstallFromGlobal(globalPath, projectRoot string, forceOverwrite bool) (int, error) {
	store := NewGlobalAgentStore(globalPath)

	// Initialize if not already
	if !store.IsInitialized() {
		if err := store.Initialize(false); err != nil {
			return 0, fmt.Errorf("전역 에이전트 초기화 실패: %w", err)
		}
	}

	return store.SyncToProject(projectRoot, forceOverwrite)
}
