package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents KB configuration
type Config struct {
	VaultPath string `yaml:"vault_path"`
	Version   string `yaml:"version"`
	CreatedAt string `yaml:"created_at"`
}

// Service handles Knowledge Base operations
type Service struct {
	vaultPath string
}

// NewService creates a new KB service
func NewService(vaultPath string) *Service {
	return &Service{
		vaultPath: vaultPath,
	}
}

// Directory structure constants
const (
	TaxonomyDir   = "_taxonomy"
	SystemDir     = "00-System"
	DomainsDir    = "10-Domains"
	ProjectsDir   = "20-Projects"
	ReferencesDir = "30-References"
	ArchiveDir    = "40-Archive"
	MetaDir       = ".pal-kb"
)

// Init initializes KB structure
func (s *Service) Init() error {
	// Create directories
	dirs := []string{
		TaxonomyDir,
		filepath.Join(TaxonomyDir, "templates"),
		filepath.Join(SystemDir, "taxonomy"),
		filepath.Join(SystemDir, "templates"),
		filepath.Join(SystemDir, "guides"),
		filepath.Join(SystemDir, "conventions"),
		DomainsDir,
		ProjectsDir,
		ReferencesDir,
		ArchiveDir,
		MetaDir,
	}

	for _, dir := range dirs {
		path := filepath.Join(s.vaultPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("디렉토리 생성 실패 %s: %w", dir, err)
		}
	}

	// Create taxonomy files
	if err := s.createTaxonomyFiles(); err != nil {
		return err
	}

	// Create TOC files
	if err := s.createTocFiles(); err != nil {
		return err
	}

	// Create root index
	if err := s.createRootIndex(); err != nil {
		return err
	}

	// Save config
	return s.saveConfig()
}

func (s *Service) createTaxonomyFiles() error {
	// domains.yaml
	domains := DomainsConfig{
		Version: "1",
		Domains: map[string]Domain{
			"example": {
				Name:        "예시 도메인",
				Description: "도메인 설명",
				Tags:        []string{"example"},
			},
		},
	}
	if err := s.writeYAML(filepath.Join(TaxonomyDir, "domains.yaml"), domains); err != nil {
		return err
	}

	// doc-types.yaml
	docTypes := DocTypesConfig{
		Version: "1",
		Types: map[string]DocType{
			"port": {
				Name:           "포트 명세",
				Template:       "tpl-port.md",
				RequiredFields: []string{"id", "title", "status"},
			},
			"adr": {
				Name:           "아키텍처 결정",
				Template:       "tpl-adr.md",
				RequiredFields: []string{"id", "title", "status", "date"},
			},
			"concept": {
				Name:           "개념 정의",
				Template:       "tpl-concept.md",
				RequiredFields: []string{"title", "domain"},
			},
			"guide": {
				Name:           "가이드",
				Template:       "tpl-guide.md",
				RequiredFields: []string{"title"},
			},
		},
	}
	if err := s.writeYAML(filepath.Join(TaxonomyDir, "doc-types.yaml"), docTypes); err != nil {
		return err
	}

	// tags.yaml
	tags := TagsConfig{
		Version: "1",
		Hierarchy: map[string]TagGroup{
			"domain": {
				Description: "도메인 태그",
				Tags:        []string{"example"},
			},
			"type": {
				Description: "문서 타입",
				Tags:        []string{"port", "adr", "concept", "guide"},
			},
			"status": {
				Description: "상태",
				Tags:        []string{"draft", "active", "archived"},
			},
		},
	}
	return s.writeYAML(filepath.Join(TaxonomyDir, "tags.yaml"), tags)
}

func (s *Service) createTocFiles() error {
	sections := []struct {
		dir   string
		title string
		desc  string
	}{
		{SystemDir, "시스템", "메타 문서 및 시스템 설정"},
		{DomainsDir, "도메인", "도메인별 지식 문서"},
		{ProjectsDir, "프로젝트", "프로젝트별 문서"},
		{ReferencesDir, "참조", "참조 문서"},
		{ArchiveDir, "아카이브", "아카이브된 문서"},
	}

	for _, sec := range sections {
		content := fmt.Sprintf(`---
type: toc
auto_generate: true
depth: 2
sort: alphabetical
---

# %s

> %s

## 목차

(자동 생성 예정)

---

tags: #toc #%s
`, sec.title, sec.desc, sec.dir)

		path := filepath.Join(s.vaultPath, sec.dir, "_toc.md")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) createRootIndex() error {
	content := fmt.Sprintf(`---
type: index
title: Knowledge Base
created: %s
---

# Knowledge Base

> PAL Kit Knowledge Base

## 섹션

- [[00-System/_toc|시스템]] - 메타 문서, 설정
- [[10-Domains/_toc|도메인]] - 도메인별 지식
- [[20-Projects/_toc|프로젝트]] - 프로젝트별 문서
- [[30-References/_toc|참조]] - 참조 문서
- [[40-Archive/_toc|아카이브]] - 아카이브

## 분류체계

- [[_taxonomy/domains|도메인 정의]]
- [[_taxonomy/doc-types|문서 타입]]
- [[_taxonomy/tags|태그 계층]]

---

tags: #index #root
`, time.Now().Format("2006-01-02"))

	return os.WriteFile(filepath.Join(s.vaultPath, "_index.md"), []byte(content), 0644)
}

func (s *Service) saveConfig() error {
	config := Config{
		VaultPath: s.vaultPath,
		Version:   "1",
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	return s.writeYAML(filepath.Join(MetaDir, "config.yaml"), config)
}

func (s *Service) writeYAML(relPath string, data interface{}) error {
	path := filepath.Join(s.vaultPath, relPath)
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

// Status returns KB status
func (s *Service) Status() (*KBStatus, error) {
	status := &KBStatus{
		VaultPath:   s.vaultPath,
		Initialized: false,
	}

	// Check if initialized
	configPath := filepath.Join(s.vaultPath, MetaDir, "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		status.Initialized = true

		// Load config
		data, err := os.ReadFile(configPath)
		if err == nil {
			var cfg Config
			if yaml.Unmarshal(data, &cfg) == nil {
				status.Version = cfg.Version
				status.CreatedAt = cfg.CreatedAt
			}
		}
	}

	// Count documents
	if status.Initialized {
		status.Sections = s.countSections()
	}

	return status, nil
}

func (s *Service) countSections() map[string]int {
	counts := make(map[string]int)
	sections := []string{SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}

	for _, sec := range sections {
		count := 0
		filepath.Walk(filepath.Join(s.vaultPath, sec), func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && filepath.Ext(path) == ".md" {
				count++
			}
			return nil
		})
		counts[sec] = count
	}

	return counts
}
