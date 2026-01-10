package convention

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConventionType represents the type of convention
type ConventionType string

const (
	TypeCodingStyle   ConventionType = "coding-style"
	TypeNaming        ConventionType = "naming"
	TypeCommitMessage ConventionType = "commit-message"
	TypeFileStructure ConventionType = "file-structure"
	TypeDocumentation ConventionType = "documentation"
	TypeTesting       ConventionType = "testing"
	TypeErrorHandling ConventionType = "error-handling"
	TypeCustom        ConventionType = "custom"
)

// Convention represents a project convention
type Convention struct {
	ID          string         `yaml:"id" json:"id"`
	Name        string         `yaml:"name" json:"name"`
	Type        ConventionType `yaml:"type" json:"type"`
	Description string         `yaml:"description" json:"description"`
	Rules       []Rule         `yaml:"rules" json:"rules"`
	Examples    Examples       `yaml:"examples,omitempty" json:"examples,omitempty"`
	Enabled     bool           `yaml:"enabled" json:"enabled"`
	Priority    int            `yaml:"priority" json:"priority"` // 1-10, 높을수록 중요
	FilePath    string         `yaml:"-" json:"file_path,omitempty"`
	CreatedAt   time.Time      `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt   time.Time      `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// Rule represents a single rule within a convention
type Rule struct {
	ID          string   `yaml:"id" json:"id"`
	Description string   `yaml:"description" json:"description"`
	Pattern     string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`           // regex pattern
	AntiPattern string   `yaml:"anti_pattern,omitempty" json:"anti_pattern,omitempty"` // what to avoid
	FileTypes   []string `yaml:"file_types,omitempty" json:"file_types,omitempty"`     // applicable file types
	Severity    string   `yaml:"severity" json:"severity"`                             // error, warning, info
	AutoFix     bool     `yaml:"auto_fix,omitempty" json:"auto_fix,omitempty"`
}

// Examples contains good and bad examples
type Examples struct {
	Good []Example `yaml:"good,omitempty" json:"good,omitempty"`
	Bad  []Example `yaml:"bad,omitempty" json:"bad,omitempty"`
}

// Example represents a code example
type Example struct {
	Code        string `yaml:"code" json:"code"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// CheckResult represents the result of a convention check
type CheckResult struct {
	ConventionID string       `json:"convention_id"`
	RuleID       string       `json:"rule_id"`
	FilePath     string       `json:"file_path"`
	Line         int          `json:"line,omitempty"`
	Severity     string       `json:"severity"`
	Message      string       `json:"message"`
	Suggestion   string       `json:"suggestion,omitempty"`
}

// Service handles convention operations
type Service struct {
	conventionsDir string
	conventions    map[string]*Convention
}

// NewService creates a new convention service
func NewService(projectRoot string) *Service {
	return &Service{
		conventionsDir: filepath.Join(projectRoot, "conventions"),
		conventions:    make(map[string]*Convention),
	}
}

// EnsureDir ensures the conventions directory exists
func (s *Service) EnsureDir() error {
	return os.MkdirAll(s.conventionsDir, 0755)
}

// Load loads all conventions from the conventions directory
func (s *Service) Load() error {
	s.conventions = make(map[string]*Convention)

	if _, err := os.Stat(s.conventionsDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(s.conventionsDir)
	if err != nil {
		return fmt.Errorf("conventions 디렉토리 읽기 실패: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		filePath := filepath.Join(s.conventionsDir, name)
		conv, err := s.loadConventionFile(filePath)
		if err != nil {
			continue
		}

		s.conventions[conv.ID] = conv
	}

	return nil
}

// loadConventionFile loads a convention from a file
func (s *Service) loadConventionFile(filePath string) (*Convention, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var conv Convention
	if err := yaml.Unmarshal(content, &conv); err != nil {
		return nil, fmt.Errorf("YAML 파싱 실패: %w", err)
	}

	conv.FilePath = filePath

	if conv.ID == "" {
		baseName := filepath.Base(filePath)
		conv.ID = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}

	return &conv, nil
}

// Get returns a convention by ID
func (s *Service) Get(id string) (*Convention, error) {
	if len(s.conventions) == 0 {
		if err := s.Load(); err != nil {
			return nil, err
		}
	}

	conv, ok := s.conventions[id]
	if !ok {
		return nil, fmt.Errorf("컨벤션 '%s'을(를) 찾을 수 없습니다", id)
	}

	return conv, nil
}

// List returns all loaded conventions
func (s *Service) List() ([]*Convention, error) {
	if len(s.conventions) == 0 {
		if err := s.Load(); err != nil {
			return nil, err
		}
	}

	var conventions []*Convention
	for _, conv := range s.conventions {
		conventions = append(conventions, conv)
	}

	return conventions, nil
}

// ListEnabled returns only enabled conventions
func (s *Service) ListEnabled() ([]*Convention, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}

	var enabled []*Convention
	for _, conv := range all {
		if conv.Enabled {
			enabled = append(enabled, conv)
		}
	}

	return enabled, nil
}

// Create creates a new convention file
func (s *Service) Create(conv *Convention) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	if conv.ID == "" {
		return fmt.Errorf("컨벤션 ID가 필요합니다")
	}

	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()
	if conv.Priority == 0 {
		conv.Priority = 5
	}

	content, err := yaml.Marshal(conv)
	if err != nil {
		return fmt.Errorf("YAML 생성 실패: %w", err)
	}

	filePath := filepath.Join(s.conventionsDir, conv.ID+".yaml")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	conv.FilePath = filePath
	s.conventions[conv.ID] = conv

	return nil
}

// Update updates an existing convention
func (s *Service) Update(conv *Convention) error {
	existing, err := s.Get(conv.ID)
	if err != nil {
		return err
	}

	conv.FilePath = existing.FilePath
	conv.UpdatedAt = time.Now()
	if conv.CreatedAt.IsZero() {
		conv.CreatedAt = existing.CreatedAt
	}

	content, err := yaml.Marshal(conv)
	if err != nil {
		return fmt.Errorf("YAML 생성 실패: %w", err)
	}

	if err := os.WriteFile(conv.FilePath, content, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	s.conventions[conv.ID] = conv

	return nil
}

// Delete removes a convention file
func (s *Service) Delete(id string) error {
	conv, err := s.Get(id)
	if err != nil {
		return err
	}

	if conv.FilePath != "" {
		if err := os.Remove(conv.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("파일 삭제 실패: %w", err)
		}
	}

	delete(s.conventions, id)
	return nil
}

// Enable enables a convention
func (s *Service) Enable(id string) error {
	conv, err := s.Get(id)
	if err != nil {
		return err
	}

	conv.Enabled = true
	return s.Update(conv)
}

// Disable disables a convention
func (s *Service) Disable(id string) error {
	conv, err := s.Get(id)
	if err != nil {
		return err
	}

	conv.Enabled = false
	return s.Update(conv)
}

// Check checks files against enabled conventions
func (s *Service) Check(paths []string) ([]CheckResult, error) {
	conventions, err := s.ListEnabled()
	if err != nil {
		return nil, err
	}

	var results []CheckResult

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		for _, conv := range conventions {
			for _, rule := range conv.Rules {
				if !s.isApplicable(path, rule.FileTypes) {
					continue
				}

				violations := s.checkRule(string(content), path, conv, &rule)
				results = append(results, violations...)
			}
		}
	}

	return results, nil
}

// isApplicable checks if a rule applies to a file
func (s *Service) isApplicable(filePath string, fileTypes []string) bool {
	if len(fileTypes) == 0 {
		return true
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	for _, ft := range fileTypes {
		if ext == ft || ext == "."+ft {
			return true
		}
	}

	return false
}

// checkRule checks a single rule against content
func (s *Service) checkRule(content, filePath string, conv *Convention, rule *Rule) []CheckResult {
	var results []CheckResult

	// Pattern 검사 (있어야 하는 패턴)
	if rule.Pattern != "" {
		re, err := regexp.Compile(rule.Pattern)
		if err == nil {
			if !re.MatchString(content) {
				results = append(results, CheckResult{
					ConventionID: conv.ID,
					RuleID:       rule.ID,
					FilePath:     filePath,
					Severity:     rule.Severity,
					Message:      rule.Description,
				})
			}
		}
	}

	// AntiPattern 검사 (없어야 하는 패턴)
	if rule.AntiPattern != "" {
		re, err := regexp.Compile(rule.AntiPattern)
		if err == nil {
			matches := re.FindAllStringIndex(content, -1)
			for _, match := range matches {
				line := strings.Count(content[:match[0]], "\n") + 1
				results = append(results, CheckResult{
					ConventionID: conv.ID,
					RuleID:       rule.ID,
					FilePath:     filePath,
					Line:         line,
					Severity:     rule.Severity,
					Message:      rule.Description,
				})
			}
		}
	}

	return results
}

// GetConventionTypes returns available convention types
func GetConventionTypes() []ConventionType {
	return []ConventionType{
		TypeCodingStyle,
		TypeNaming,
		TypeCommitMessage,
		TypeFileStructure,
		TypeDocumentation,
		TypeTesting,
		TypeErrorHandling,
		TypeCustom,
	}
}

// Summary returns a summary of conventions
func (s *Service) Summary() (map[string]int, error) {
	conventions, err := s.List()
	if err != nil {
		return nil, err
	}

	summary := map[string]int{
		"total":    len(conventions),
		"enabled":  0,
		"disabled": 0,
		"rules":    0,
	}

	for _, conv := range conventions {
		if conv.Enabled {
			summary["enabled"]++
		} else {
			summary["disabled"]++
		}
		summary["rules"] += len(conv.Rules)
	}

	return summary, nil
}
