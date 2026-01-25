package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// TOCConfig represents _toc.md frontmatter
type TOCConfig struct {
	Type         string `yaml:"type"`
	AutoGenerate bool   `yaml:"auto_generate"`
	Depth        int    `yaml:"depth"`
	Sort         string `yaml:"sort"` // alphabetical, date, custom
}

// TOCEntry represents a TOC entry
type TOCEntry struct {
	Title    string
	Path     string
	IsDir    bool
	ModTime  time.Time
	Children []*TOCEntry
	Depth    int
}

// TOCStats represents TOC statistics
type TOCStats struct {
	TotalDocs    int       `json:"total_docs"`
	LastModified time.Time `json:"last_modified"`
	Sections     int       `json:"sections"`
}

// TOCCheckResult represents TOC integrity check result
type TOCCheckResult struct {
	Section      string   `json:"section"`
	Valid        bool     `json:"valid"`
	MissingDocs  []string `json:"missing_docs,omitempty"`
	OrphanLinks  []string `json:"orphan_links,omitempty"`
	LastUpdated  string   `json:"last_updated,omitempty"`
	NeedsRefresh bool     `json:"needs_refresh"`
}

// GenerateTOC generates TOC for a section
func (s *Service) GenerateTOC(section string, depth int, sortBy string) (*TOCStats, error) {
	sectionPath := filepath.Join(s.vaultPath, section)

	// Check if section exists
	if _, err := os.Stat(sectionPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("섹션이 존재하지 않습니다: %s", section)
	}

	// Scan documents
	entries, stats, err := s.scanForTOC(sectionPath, depth, sortBy)
	if err != nil {
		return nil, err
	}

	// Generate TOC content
	content := s.generateTOCContent(section, entries, stats, depth, sortBy, sectionPath)

	// Write _toc.md
	tocPath := filepath.Join(sectionPath, "_toc.md")
	if err := os.WriteFile(tocPath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("TOC 파일 쓰기 실패: %w", err)
	}

	return stats, nil
}

// GenerateAllTOC generates TOC for all sections
func (s *Service) GenerateAllTOC(depth int, sortBy string) (map[string]*TOCStats, error) {
	sections := []string{SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}
	results := make(map[string]*TOCStats)

	for _, sec := range sections {
		stats, err := s.GenerateTOC(sec, depth, sortBy)
		if err != nil {
			return results, fmt.Errorf("%s TOC 생성 실패: %w", sec, err)
		}
		results[sec] = stats
	}

	// Update root index
	if err := s.updateRootIndex(results); err != nil {
		return results, fmt.Errorf("루트 인덱스 업데이트 실패: %w", err)
	}

	return results, nil
}

func (s *Service) scanForTOC(basePath string, maxDepth int, sortBy string) ([]*TOCEntry, *TOCStats, error) {
	var entries []*TOCEntry
	stats := &TOCStats{}

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden files, _toc.md, and _index.md (handled by parent dir)
		name := info.Name()
		if strings.HasPrefix(name, ".") || name == "_toc.md" || name == "_index.md" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate relative path and depth
		relPath, _ := filepath.Rel(basePath, path)
		if relPath == "." {
			return nil
		}

		depth := strings.Count(relPath, string(filepath.Separator)) + 1
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process markdown files and directories
		if !info.IsDir() && filepath.Ext(path) != ".md" {
			return nil
		}

		// Get title from file
		title := s.getTitleFromFile(path, info)

		entry := &TOCEntry{
			Title:   title,
			Path:    relPath,
			IsDir:   info.IsDir(),
			ModTime: info.ModTime(),
			Depth:   depth,
		}

		entries = append(entries, entry)

		if !info.IsDir() {
			stats.TotalDocs++
			if info.ModTime().After(stats.LastModified) {
				stats.LastModified = info.ModTime()
			}
		} else {
			stats.Sections++
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Sort entries
	s.sortEntries(entries, sortBy)

	// Build tree structure
	tree := s.buildTree(entries)

	return tree, stats, nil
}

func (s *Service) getTitleFromFile(path string, info os.FileInfo) string {
	if info.IsDir() {
		// Check for _index.md
		indexPath := filepath.Join(path, "_index.md")
		if title := s.extractTitle(indexPath); title != "" {
			return title
		}
		return info.Name()
	}

	// Extract title from frontmatter or first heading
	if title := s.extractTitle(path); title != "" {
		return title
	}

	// Fallback to filename without extension
	return strings.TrimSuffix(info.Name(), ".md")
}

func (s *Service) extractTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	content := string(data)

	// Try frontmatter title
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			var fm map[string]interface{}
			if yaml.Unmarshal([]byte(parts[1]), &fm) == nil {
				if title, ok := fm["title"].(string); ok {
					return title
				}
			}
		}
	}

	// Try first heading
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}

	return ""
}

func (s *Service) sortEntries(entries []*TOCEntry, sortBy string) {
	switch sortBy {
	case "date":
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].ModTime.After(entries[j].ModTime)
		})
	case "alphabetical":
		fallthrough
	default:
		sort.Slice(entries, func(i, j int) bool {
			// Directories first
			if entries[i].IsDir != entries[j].IsDir {
				return entries[i].IsDir
			}
			return entries[i].Title < entries[j].Title
		})
	}
}

func (s *Service) buildTree(entries []*TOCEntry) []*TOCEntry {
	// Map for quick lookup
	pathMap := make(map[string]*TOCEntry)
	var roots []*TOCEntry

	for _, entry := range entries {
		pathMap[entry.Path] = entry

		parentPath := filepath.Dir(entry.Path)
		if parentPath == "." {
			roots = append(roots, entry)
		} else if parent, ok := pathMap[parentPath]; ok {
			parent.Children = append(parent.Children, entry)
		} else {
			roots = append(roots, entry)
		}
	}

	return roots
}

func (s *Service) generateTOCContent(section string, entries []*TOCEntry, stats *TOCStats, depth int, sortBy string, sectionPath string) string {
	sectionTitles := map[string]string{
		SystemDir:     "시스템",
		DomainsDir:    "도메인",
		ProjectsDir:   "프로젝트",
		ReferencesDir: "참조",
		ArchiveDir:    "아카이브",
	}

	sectionDescs := map[string]string{
		SystemDir:     "메타 문서 및 시스템 설정",
		DomainsDir:    "도메인별 지식 문서",
		ProjectsDir:   "프로젝트별 문서",
		ReferencesDir: "참조 문서",
		ArchiveDir:    "아카이브된 문서",
	}

	title := sectionTitles[section]
	if title == "" {
		title = section
	}
	desc := sectionDescs[section]

	var sb strings.Builder

	// Frontmatter
	sb.WriteString("---\n")
	sb.WriteString("type: toc\n")
	sb.WriteString("auto_generate: true\n")
	sb.WriteString(fmt.Sprintf("depth: %d\n", depth))
	sb.WriteString(fmt.Sprintf("sort: %s\n", sortBy))
	sb.WriteString(fmt.Sprintf("generated: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("---\n\n")

	// Header
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	if desc != "" {
		sb.WriteString(fmt.Sprintf("> %s\n\n", desc))
	}

	// Statistics
	sb.WriteString("## 통계\n\n")
	sb.WriteString(fmt.Sprintf("- 문서 수: %d\n", stats.TotalDocs))
	sb.WriteString(fmt.Sprintf("- 섹션 수: %d\n", stats.Sections))
	if !stats.LastModified.IsZero() {
		sb.WriteString(fmt.Sprintf("- 최근 수정: %s\n", stats.LastModified.Format("2006-01-02 15:04")))
	}
	sb.WriteString("\n")

	// Table of Contents
	sb.WriteString("## 목차\n\n")

	if len(entries) == 0 {
		sb.WriteString("(문서 없음)\n")
	} else {
		s.renderEntries(&sb, entries, 0, sectionPath)
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString(fmt.Sprintf("tags: #toc #%s\n", section))

	return sb.String()
}

func (s *Service) renderEntries(sb *strings.Builder, entries []*TOCEntry, indent int, basePath string) {
	prefix := strings.Repeat("  ", indent)

	for _, entry := range entries {
		// Create wikilink
		link := strings.TrimSuffix(entry.Path, ".md")
		if entry.IsDir {
			// Check if _index.md exists
			indexPath := filepath.Join(basePath, entry.Path, "_index.md")
			if _, err := os.Stat(indexPath); err == nil {
				fmt.Fprintf(sb, "%s- 📁 [[%s/_index|%s]]\n", prefix, link, entry.Title)
			} else {
				// Just render as text without link
				fmt.Fprintf(sb, "%s- 📁 %s\n", prefix, entry.Title)
			}
		} else {
			fmt.Fprintf(sb, "%s- [[%s|%s]]\n", prefix, link, entry.Title)
		}

		// Render children
		if len(entry.Children) > 0 {
			s.renderEntries(sb, entry.Children, indent+1, basePath)
		}
	}
}

func (s *Service) updateRootIndex(stats map[string]*TOCStats) error {
	content := fmt.Sprintf(`---
type: index
title: Knowledge Base
updated: %s
---

# Knowledge Base

> PAL Kit Knowledge Base

## 섹션

`, time.Now().Format("2006-01-02"))

	sections := []struct {
		dir  string
		name string
		desc string
	}{
		{SystemDir, "시스템", "메타 문서, 설정"},
		{DomainsDir, "도메인", "도메인별 지식"},
		{ProjectsDir, "프로젝트", "프로젝트별 문서"},
		{ReferencesDir, "참조", "참조 문서"},
		{ArchiveDir, "아카이브", "아카이브"},
	}

	for _, sec := range sections {
		stat := stats[sec.dir]
		docCount := 0
		if stat != nil {
			docCount = stat.TotalDocs
		}
		content += fmt.Sprintf("- [[%s/_toc|%s]] - %s (%d문서)\n", sec.dir, sec.name, sec.desc, docCount)
	}

	content += `
## 분류체계

- [[_taxonomy/domains|도메인 정의]]
- [[_taxonomy/doc-types|문서 타입]]
- [[_taxonomy/tags|태그 계층]]

---

tags: #index #root
`

	return os.WriteFile(filepath.Join(s.vaultPath, "_index.md"), []byte(content), 0644)
}

// UpdateTOC updates TOC if documents changed
func (s *Service) UpdateTOC(section string) (*TOCStats, bool, error) {
	tocPath := filepath.Join(s.vaultPath, section, "_toc.md")

	// Read existing TOC config
	config, lastGenerated, err := s.readTOCConfig(tocPath)
	if err != nil {
		// No existing TOC, generate new one
		stats, err := s.GenerateTOC(section, 2, "alphabetical")
		return stats, true, err
	}

	// Check if any file is newer than TOC
	needsUpdate := false
	sectionPath := filepath.Join(s.vaultPath, section)

	filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" && path != tocPath {
			if info.ModTime().After(lastGenerated) {
				needsUpdate = true
				return filepath.SkipAll
			}
		}
		return nil
	})

	if !needsUpdate {
		return nil, false, nil
	}

	depth := config.Depth
	if depth == 0 {
		depth = 2
	}
	sortBy := config.Sort
	if sortBy == "" {
		sortBy = "alphabetical"
	}

	stats, err := s.GenerateTOC(section, depth, sortBy)
	return stats, true, err
}

func (s *Service) readTOCConfig(tocPath string) (*TOCConfig, time.Time, error) {
	data, err := os.ReadFile(tocPath)
	if err != nil {
		return nil, time.Time{}, err
	}

	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return &TOCConfig{Depth: 2, Sort: "alphabetical"}, time.Time{}, nil
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return &TOCConfig{Depth: 2, Sort: "alphabetical"}, time.Time{}, nil
	}

	var config TOCConfig
	if err := yaml.Unmarshal([]byte(parts[1]), &config); err != nil {
		return &TOCConfig{Depth: 2, Sort: "alphabetical"}, time.Time{}, nil
	}

	// Extract generated time
	var generated time.Time
	re := regexp.MustCompile(`generated:\s*(.+)`)
	if matches := re.FindStringSubmatch(parts[1]); len(matches) > 1 {
		generated, _ = time.Parse(time.RFC3339, strings.TrimSpace(matches[1]))
	}

	return &config, generated, nil
}

// CheckTOC checks TOC integrity
func (s *Service) CheckTOC(section string) (*TOCCheckResult, error) {
	result := &TOCCheckResult{
		Section: section,
		Valid:   true,
	}

	tocPath := filepath.Join(s.vaultPath, section, "_toc.md")
	sectionPath := filepath.Join(s.vaultPath, section)

	// Check if TOC exists
	tocInfo, err := os.Stat(tocPath)
	if os.IsNotExist(err) {
		result.Valid = false
		result.NeedsRefresh = true
		return result, nil
	}

	result.LastUpdated = tocInfo.ModTime().Format("2006-01-02 15:04")

	// Read TOC content
	data, err := os.ReadFile(tocPath)
	if err != nil {
		return nil, err
	}

	content := string(data)

	// Extract wikilinks from TOC
	re := regexp.MustCompile(`\[\[([^\]|]+)`)
	matches := re.FindAllStringSubmatch(content, -1)

	var tocLinks []string
	for _, match := range matches {
		if len(match) > 1 {
			tocLinks = append(tocLinks, match[1])
		}
	}

	// Check each link - resolve relative to section
	for _, link := range tocLinks {
		// Links in TOC are relative to section
		linkPath := filepath.Join(sectionPath, link)
		if !strings.HasSuffix(linkPath, ".md") {
			linkPath += ".md"
		}

		if _, err := os.Stat(linkPath); os.IsNotExist(err) {
			result.OrphanLinks = append(result.OrphanLinks, link)
			result.Valid = false
		}
	}

	// Find documents not in TOC
	filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" || info.Name() == "_toc.md" {
			return nil
		}

		// Get path relative to section
		relPath, _ := filepath.Rel(sectionPath, path)
		relPath = strings.TrimSuffix(relPath, ".md")

		found := false
		for _, link := range tocLinks {
			// Normalize for comparison
			normalizedLink := strings.TrimSuffix(link, "/_index")
			if link == relPath || normalizedLink == relPath || strings.HasPrefix(relPath, link+"/") {
				found = true
				break
			}
		}

		if !found {
			result.MissingDocs = append(result.MissingDocs, relPath)
			result.NeedsRefresh = true
		}

		return nil
	})

	// Check if any file is newer than TOC
	filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || path == tocPath {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			if info.ModTime().After(tocInfo.ModTime()) {
				result.NeedsRefresh = true
				return filepath.SkipAll
			}
		}
		return nil
	})

	return result, nil
}

// CheckAllTOC checks all section TOCs
func (s *Service) CheckAllTOC() ([]*TOCCheckResult, error) {
	sections := []string{SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}
	var results []*TOCCheckResult

	for _, sec := range sections {
		result, err := s.CheckTOC(sec)
		if err != nil {
			return results, fmt.Errorf("%s TOC 검사 실패: %w", sec, err)
		}
		results = append(results, result)
	}

	return results, nil
}
