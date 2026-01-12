package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Package represents an agent package configuration
type Package struct {
	ID          string       `yaml:"id"`
	Name        string       `yaml:"name"`
	Version     string       `yaml:"version"`
	Description string       `yaml:"description,omitempty"`
	Extends     string       `yaml:"extends,omitempty"`
	Tech        TechConfig   `yaml:"tech"`
	Architecture ArchConfig  `yaml:"architecture"`
	Methodology  MethodConfig `yaml:"methodology"`
	Workers     []string     `yaml:"workers"`
	CoreOverrides map[string]CoreOverride `yaml:"core_overrides,omitempty"`
	FilePath    string       `yaml:"-"`
}

// TechConfig holds technology stack configuration
type TechConfig struct {
	Language   string   `yaml:"language"`
	Frameworks []string `yaml:"frameworks,omitempty"`
	BuildTool  string   `yaml:"build_tool,omitempty"`
	Runtime    string   `yaml:"runtime,omitempty"`
}

// ArchConfig holds architecture configuration
type ArchConfig struct {
	Name           string   `yaml:"name"`
	Layers         []string `yaml:"layers"`
	ConventionsRef string   `yaml:"conventions_ref,omitempty"`
	DependencyRule string   `yaml:"dependency_rule,omitempty"`
}

// MethodConfig holds methodology configuration
type MethodConfig struct {
	PortDriven   bool `yaml:"port_driven"`
	CQS          bool `yaml:"cqs"`
	EventDriven  bool `yaml:"event_driven"`
}

// CoreOverride holds override settings for core agents
type CoreOverride struct {
	ConventionsRef string   `yaml:"conventions_ref,omitempty"`
	PortTemplates  []string `yaml:"port_templates,omitempty"`
	ValidationRules []string `yaml:"validation_rules,omitempty"`
}

// PackageSpec is the YAML structure for package files
type PackageSpec struct {
	Package Package `yaml:"package"`
}

// Service handles package operations
type Service struct {
	projectRoot    string
	globalDir      string
	packages       map[string]*Package
	resolvedCache  map[string]*Package // 상속 해결된 캐시
}

// NewService creates a new package service
func NewService(projectRoot string, globalDir string) *Service {
	return &Service{
		projectRoot:   projectRoot,
		globalDir:     globalDir,
		packages:      make(map[string]*Package),
		resolvedCache: make(map[string]*Package),
	}
}

// Load loads all packages from project and global directories
func (s *Service) Load() error {
	s.packages = make(map[string]*Package)
	s.resolvedCache = make(map[string]*Package)

	// 1. 전역 패키지 로드
	if s.globalDir != "" {
		if err := s.loadFromDir(s.globalDir); err != nil {
			// 전역 디렉토리 없으면 무시
			if !os.IsNotExist(err) {
				return fmt.Errorf("전역 패키지 로드 실패: %w", err)
			}
		}
	}

	// 2. 프로젝트 패키지 로드 (오버라이드)
	projectPkgDir := filepath.Join(s.projectRoot, "packages")
	if err := s.loadFromDir(projectPkgDir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("프로젝트 패키지 로드 실패: %w", err)
		}
	}

	// 3. .pal/packages 디렉토리도 확인
	palPkgDir := filepath.Join(s.projectRoot, ".pal", "packages")
	if err := s.loadFromDir(palPkgDir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf(".pal 패키지 로드 실패: %w", err)
		}
	}

	return nil
}

// loadFromDir loads packages from a directory
func (s *Service) loadFromDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		filePath := filepath.Join(dir, name)
		pkg, err := s.loadPackageFile(filePath)
		if err != nil {
			// 로딩 실패한 파일은 스킵하고 경고
			fmt.Fprintf(os.Stderr, "⚠️  패키지 로드 실패 (%s): %v\n", name, err)
			continue
		}

		s.packages[pkg.ID] = pkg
	}

	return nil
}

// loadPackageFile loads a package from a file
func (s *Service) loadPackageFile(filePath string) (*Package, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var spec PackageSpec
	if err := yaml.Unmarshal(content, &spec); err != nil {
		return nil, fmt.Errorf("YAML 파싱 실패: %w", err)
	}

	pkg := &spec.Package
	pkg.FilePath = filePath

	// ID가 없으면 파일명에서 추출
	if pkg.ID == "" {
		ext := filepath.Ext(filePath)
		pkg.ID = strings.TrimSuffix(filepath.Base(filePath), ext)
	}

	return pkg, nil
}

// Get returns a package by ID with inheritance resolved
func (s *Service) Get(id string) (*Package, error) {
	if len(s.packages) == 0 {
		if err := s.Load(); err != nil {
			return nil, err
		}
	}

	// 캐시 확인
	if resolved, ok := s.resolvedCache[id]; ok {
		return resolved, nil
	}

	pkg, ok := s.packages[id]
	if !ok {
		return nil, fmt.Errorf("패키지 '%s'을(를) 찾을 수 없습니다", id)
	}

	// 상속 해결
	resolved, err := s.resolveInheritance(pkg, make(map[string]bool))
	if err != nil {
		return nil, err
	}

	s.resolvedCache[id] = resolved
	return resolved, nil
}

// resolveInheritance resolves package inheritance
func (s *Service) resolveInheritance(pkg *Package, visited map[string]bool) (*Package, error) {
	// 순환 참조 검사
	if visited[pkg.ID] {
		return nil, fmt.Errorf("순환 상속 감지: %s", pkg.ID)
	}
	visited[pkg.ID] = true

	// 상속이 없으면 그대로 반환
	if pkg.Extends == "" {
		return pkg, nil
	}

	// 부모 패키지 가져오기
	parent, ok := s.packages[pkg.Extends]
	if !ok {
		return nil, fmt.Errorf("부모 패키지 '%s'을(를) 찾을 수 없습니다", pkg.Extends)
	}

	// 부모의 상속 먼저 해결
	resolvedParent, err := s.resolveInheritance(parent, visited)
	if err != nil {
		return nil, err
	}

	// 병합
	return s.mergePackages(resolvedParent, pkg), nil
}

// mergePackages merges child into parent (child overrides parent)
func (s *Service) mergePackages(parent, child *Package) *Package {
	merged := &Package{
		ID:          child.ID,
		Name:        child.Name,
		Version:     child.Version,
		Description: child.Description,
		FilePath:    child.FilePath,
	}

	// Name이 비어있으면 부모에서
	if merged.Name == "" {
		merged.Name = parent.Name
	}

	// Tech 병합
	merged.Tech = parent.Tech
	if child.Tech.Language != "" {
		merged.Tech.Language = child.Tech.Language
	}
	if len(child.Tech.Frameworks) > 0 {
		merged.Tech.Frameworks = child.Tech.Frameworks
	}
	if child.Tech.BuildTool != "" {
		merged.Tech.BuildTool = child.Tech.BuildTool
	}
	if child.Tech.Runtime != "" {
		merged.Tech.Runtime = child.Tech.Runtime
	}

	// Architecture 병합
	merged.Architecture = parent.Architecture
	if child.Architecture.Name != "" {
		merged.Architecture.Name = child.Architecture.Name
	}
	if len(child.Architecture.Layers) > 0 {
		merged.Architecture.Layers = child.Architecture.Layers
	}
	if child.Architecture.ConventionsRef != "" {
		merged.Architecture.ConventionsRef = child.Architecture.ConventionsRef
	}

	// Methodology 병합 (child 우선)
	merged.Methodology = child.Methodology
	if !child.Methodology.PortDriven && parent.Methodology.PortDriven {
		merged.Methodology.PortDriven = parent.Methodology.PortDriven
	}
	if !child.Methodology.CQS && parent.Methodology.CQS {
		merged.Methodology.CQS = parent.Methodology.CQS
	}
	if !child.Methodology.EventDriven && parent.Methodology.EventDriven {
		merged.Methodology.EventDriven = parent.Methodology.EventDriven
	}

	// Workers 병합 (부모 + 자식)
	workerSet := make(map[string]bool)
	for _, w := range parent.Workers {
		workerSet[w] = true
	}
	for _, w := range child.Workers {
		workerSet[w] = true
	}
	merged.Workers = make([]string, 0, len(workerSet))
	for w := range workerSet {
		merged.Workers = append(merged.Workers, w)
	}

	// CoreOverrides 병합
	merged.CoreOverrides = make(map[string]CoreOverride)
	for k, v := range parent.CoreOverrides {
		merged.CoreOverrides[k] = v
	}
	for k, v := range child.CoreOverrides {
		merged.CoreOverrides[k] = v
	}

	return merged
}

// List returns all loaded packages
func (s *Service) List() ([]*Package, error) {
	if len(s.packages) == 0 {
		if err := s.Load(); err != nil {
			return nil, err
		}
	}

	var packages []*Package
	for _, pkg := range s.packages {
		packages = append(packages, pkg)
	}

	return packages, nil
}

// Validate validates a package
func (s *Service) Validate(pkg *Package) []string {
	var errors []string

	if pkg.ID == "" {
		errors = append(errors, "ID가 필요합니다")
	}
	if pkg.Name == "" {
		errors = append(errors, "Name이 필요합니다")
	}
	if pkg.Tech.Language == "" {
		errors = append(errors, "Tech.Language가 필요합니다")
	}
	if pkg.Architecture.Name == "" {
		errors = append(errors, "Architecture.Name이 필요합니다")
	}
	if len(pkg.Architecture.Layers) == 0 {
		errors = append(errors, "Architecture.Layers가 필요합니다")
	}

	return errors
}

// Create creates a new package file
func (s *Service) Create(pkg *Package, targetDir string) error {
	// 검증
	if errs := s.Validate(pkg); len(errs) > 0 {
		return fmt.Errorf("패키지 검증 실패: %s", strings.Join(errs, ", "))
	}

	// 디렉토리 생성
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	spec := PackageSpec{Package: *pkg}

	content, err := yaml.Marshal(spec)
	if err != nil {
		return fmt.Errorf("YAML 생성 실패: %w", err)
	}

	filePath := filepath.Join(targetDir, pkg.ID+".yaml")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	pkg.FilePath = filePath
	s.packages[pkg.ID] = pkg

	return nil
}

// GetWorkers returns the worker list for a package
func (s *Service) GetWorkers(id string) ([]string, error) {
	pkg, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return pkg.Workers, nil
}

// GetConventionsPath returns the conventions path for a package
func (s *Service) GetConventionsPath(id string) (string, error) {
	pkg, err := s.Get(id)
	if err != nil {
		return "", err
	}
	return pkg.Architecture.ConventionsRef, nil
}
