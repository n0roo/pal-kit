package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/worker"
)

// ContextOrder defines the context loading order
type ContextOrder int

const (
	OrderClaudeMD        ContextOrder = iota // 1. CLAUDE.md
	OrderPackageConv                         // 2. Package conventions (architecture.md)
	OrderWorkerCommon                        // 3. Worker common conventions (_common.md)
	OrderWorkerSpecific                      // 4. Worker specific conventions ({worker}.md)
	OrderPortSpec                            // 5. Port specification
	OrderWorkerPrompt                        // 6. Worker prompt from YAML
)

// ContextSection represents a section of the context
type ContextSection struct {
	Order    ContextOrder
	Title    string
	Content  string
	FilePath string
	Tokens   int // estimated token count
}

// Builder builds prompts for Claude Code integration
type Builder struct {
	projectRoot  string
	workerMapper *worker.Mapper
}

// NewBuilder creates a new prompt builder
func NewBuilder(projectRoot string) *Builder {
	return &Builder{
		projectRoot:  projectRoot,
		workerMapper: worker.NewMapper(projectRoot),
	}
}

// BuildContext builds the complete context for a port and worker
func (b *Builder) BuildContext(portID, workerID string, portSpec string) (*BuildResult, error) {
	result := &BuildResult{
		Sections: make([]ContextSection, 0),
	}

	// 1. CLAUDE.md
	claudeMD := b.loadClaudeMD()
	if claudeMD != "" {
		result.AddSection(ContextSection{
			Order:    OrderClaudeMD,
			Title:    "프로젝트 컨텍스트",
			Content:  claudeMD,
			FilePath: filepath.Join(b.projectRoot, "CLAUDE.md"),
			Tokens:   estimateTokens(claudeMD),
		})
	}

	// 2. Package conventions (architecture.md)
	archConv := b.loadArchitectureConvention()
	if archConv != "" {
		result.AddSection(ContextSection{
			Order:    OrderPackageConv,
			Title:    "아키텍처 규칙",
			Content:  archConv,
			FilePath: filepath.Join(b.projectRoot, "conventions", "architecture.md"),
			Tokens:   estimateTokens(archConv),
		})
	}

	// 3. Worker common conventions
	workerCommon := b.loadWorkerCommonConvention(workerID)
	if workerCommon != "" {
		result.AddSection(ContextSection{
			Order:    OrderWorkerCommon,
			Title:    "워커 공통 규칙",
			Content:  workerCommon,
			Tokens:   estimateTokens(workerCommon),
		})
	}

	// 4. Worker specific conventions
	workerSpecific := b.loadWorkerSpecificConvention(workerID)
	if workerSpecific != "" {
		result.AddSection(ContextSection{
			Order:    OrderWorkerSpecific,
			Title:    "워커 컨벤션",
			Content:  workerSpecific,
			FilePath: b.workerMapper.GetWorkerConventionPath(workerID),
			Tokens:   estimateTokens(workerSpecific),
		})
	}

	// 5. Port specification
	if portSpec != "" {
		result.AddSection(ContextSection{
			Order:    OrderPortSpec,
			Title:    fmt.Sprintf("현재 포트 명세: %s", portID),
			Content:  portSpec,
			FilePath: filepath.Join(b.projectRoot, "ports", portID+".md"),
			Tokens:   estimateTokens(portSpec),
		})
	}

	// 6. Worker prompt
	workerPrompt := b.workerMapper.GetWorkerPrompt(workerID)
	if workerPrompt != "" {
		result.AddSection(ContextSection{
			Order:   OrderWorkerPrompt,
			Title:   "작업 지침",
			Content: workerPrompt,
			Tokens:  estimateTokens(workerPrompt),
		})
	}

	// Add checklist
	checklist := b.workerMapper.GetWorkerChecklist(workerID)
	if len(checklist) > 0 {
		result.Checklist = checklist
	}

	return result, nil
}

// loadClaudeMD loads the CLAUDE.md file, extracting relevant sections
func (b *Builder) loadClaudeMD() string {
	path := filepath.Join(b.projectRoot, "CLAUDE.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	// Extract content before pal:context markers to avoid duplication
	text := string(content)
	if idx := strings.Index(text, "<!-- pal:context:start -->"); idx >= 0 {
		text = strings.TrimSpace(text[:idx])
	}

	return text
}

// loadArchitectureConvention loads the architecture convention
func (b *Builder) loadArchitectureConvention() string {
	// Try multiple paths
	paths := []string{
		filepath.Join(b.projectRoot, "conventions", "architecture.md"),
		filepath.Join(b.projectRoot, "conventions", "ARCHITECTURE.md"),
		filepath.Join(b.projectRoot, ".pal", "conventions", "architecture.md"),
	}

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content)
		}
	}

	return ""
}

// loadWorkerCommonConvention loads the common convention for a worker category
func (b *Builder) loadWorkerCommonConvention(workerID string) string {
	// Determine worker category from ID
	category := categorizeWorker(workerID)
	if category == "" {
		return ""
	}

	paths := []string{
		filepath.Join(b.projectRoot, "conventions", "agents", "workers", category, "_common.md"),
		filepath.Join(b.projectRoot, "conventions", "workers", category, "_common.md"),
	}

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content)
		}
	}

	return ""
}

// loadWorkerSpecificConvention loads the specific convention for a worker
func (b *Builder) loadWorkerSpecificConvention(workerID string) string {
	convPath := b.workerMapper.GetWorkerConventionPath(workerID)
	if convPath == "" {
		return ""
	}

	content, err := os.ReadFile(convPath)
	if err != nil {
		return ""
	}

	return string(content)
}

// categorizeWorker returns the category (backend/frontend) for a worker
func categorizeWorker(workerID string) string {
	backendWorkers := []string{
		"entity-worker", "cache-worker", "document-worker",
		"service-worker", "router-worker", "test-worker",
	}
	frontendWorkers := []string{
		"frontend-engineer-worker", "component-model-worker",
		"component-ui-worker", "e2e-worker", "unit-tc-worker",
	}

	for _, w := range backendWorkers {
		if workerID == w {
			return "backend"
		}
	}
	for _, w := range frontendWorkers {
		if workerID == w {
			return "frontend"
		}
	}

	return ""
}

// estimateTokens provides a rough token estimate (1 token ≈ 4 characters)
func estimateTokens(content string) int {
	return len(content) / 4
}

// BuildResult contains the built context
type BuildResult struct {
	Sections  []ContextSection
	Checklist []string
	WorkerID  string
}

// AddSection adds a section to the result
func (r *BuildResult) AddSection(section ContextSection) {
	r.Sections = append(r.Sections, section)
}

// TotalTokens returns the total estimated token count
func (r *BuildResult) TotalTokens() int {
	total := 0
	for _, s := range r.Sections {
		total += s.Tokens
	}
	return total
}

// ToMarkdown converts the build result to markdown format
func (r *BuildResult) ToMarkdown() string {
	var sb strings.Builder

	for _, section := range r.Sections {
		sb.WriteString(fmt.Sprintf("## %s\n\n", section.Title))
		if section.FilePath != "" {
			sb.WriteString(fmt.Sprintf("> 소스: `%s`\n\n", section.FilePath))
		}
		sb.WriteString(section.Content)
		sb.WriteString("\n\n---\n\n")
	}

	if len(r.Checklist) > 0 {
		sb.WriteString("## 완료 체크리스트\n\n")
		for _, item := range r.Checklist {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// ToCompact returns a compact version (respecting token budget)
func (r *BuildResult) ToCompact(tokenBudget int) string {
	if tokenBudget <= 0 || r.TotalTokens() <= tokenBudget {
		return r.ToMarkdown()
	}

	var sb strings.Builder
	remainingTokens := tokenBudget

	// Priority order: WorkerPrompt > PortSpec > WorkerSpecific > others
	priority := []ContextOrder{
		OrderWorkerPrompt,
		OrderPortSpec,
		OrderWorkerSpecific,
		OrderWorkerCommon,
		OrderPackageConv,
		OrderClaudeMD,
	}

	included := make(map[ContextOrder]bool)

	for _, order := range priority {
		for _, section := range r.Sections {
			if section.Order != order {
				continue
			}
			if included[order] {
				continue
			}

			if section.Tokens <= remainingTokens {
				sb.WriteString(fmt.Sprintf("## %s\n\n", section.Title))
				sb.WriteString(section.Content)
				sb.WriteString("\n\n---\n\n")
				remainingTokens -= section.Tokens
				included[order] = true
			}
		}
	}

	// Always include checklist
	if len(r.Checklist) > 0 {
		sb.WriteString("## 완료 체크리스트\n\n")
		for _, item := range r.Checklist {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
	}

	return sb.String()
}

// GetActiveWorkerSection returns a section for CLAUDE.md indicating active worker
func (r *BuildResult) GetActiveWorkerSection(workerID, portID string) string {
	var sb strings.Builder

	sb.WriteString("### 현재 활성 워커\n\n")
	sb.WriteString(fmt.Sprintf("- **워커**: `%s`\n", workerID))
	if portID != "" {
		sb.WriteString(fmt.Sprintf("- **포트**: `%s`\n", portID))
	}
	sb.WriteString("\n")

	if len(r.Checklist) > 0 {
		sb.WriteString("### 체크리스트\n\n")
		for _, item := range r.Checklist {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
	}

	return sb.String()
}
