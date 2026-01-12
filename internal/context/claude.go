package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/prompt"
	"github.com/n0roo/pal-kit/internal/worker"
)

const (
	activeWorkerStartMarker = "<!-- pal:active-worker:start -->"
	activeWorkerEndMarker   = "<!-- pal:active-worker:end -->"
)

// ClaudeService handles Claude Code integration
type ClaudeService struct {
	db            *db.DB
	projectRoot   string
	workerMapper  *worker.Mapper
	promptBuilder *prompt.Builder
}

// NewClaudeService creates a new Claude integration service
func NewClaudeService(database *db.DB, projectRoot string) *ClaudeService {
	return &ClaudeService{
		db:            database,
		projectRoot:   projectRoot,
		workerMapper:  worker.NewMapper(projectRoot),
		promptBuilder: prompt.NewBuilder(projectRoot),
	}
}

// PortStartResult contains the result of port-start operation
type PortStartResult struct {
	PortID       string
	WorkerID     string
	WorkerName   string
	Context      string
	Checklist    []string
	TokenCount   int
	ConvFiles    []string
	UpdatedFiles []string
}

// ProcessPortStart handles the port-start hook
func (s *ClaudeService) ProcessPortStart(portID string) (*PortStartResult, error) {
	result := &PortStartResult{
		PortID: portID,
	}

	// 1. Load port specification
	portSpec, err := s.loadPortSpec(portID)
	if err != nil {
		return nil, fmt.Errorf("포트 명세 로드 실패: %w", err)
	}

	// 2. Parse hints from port spec
	hints := worker.ParsePortSpecHints(portSpec)

	// 3. Determine appropriate worker
	workerID, err := s.workerMapper.MapPortToWorker(hints)
	if err != nil {
		return nil, fmt.Errorf("워커 매핑 실패: %w", err)
	}
	result.WorkerID = workerID

	// Get worker details
	workerSpec, err := s.workerMapper.GetWorker(workerID)
	if err == nil {
		result.WorkerName = workerSpec.Agent.Name
	}

	// 4. Build context
	buildResult, err := s.promptBuilder.BuildContext(portID, workerID, portSpec)
	if err != nil {
		return nil, fmt.Errorf("컨텍스트 빌드 실패: %w", err)
	}

	result.Context = buildResult.ToMarkdown()
	result.Checklist = buildResult.Checklist
	result.TokenCount = buildResult.TotalTokens()

	// Collect convention files
	for _, section := range buildResult.Sections {
		if section.FilePath != "" {
			result.ConvFiles = append(result.ConvFiles, section.FilePath)
		}
	}

	// 5. Update CLAUDE.md with active worker info
	claudeMD := FindClaudeMD(s.projectRoot)
	if claudeMD != "" {
		if err := s.updateActiveWorkerSection(claudeMD, workerID, portID, buildResult); err == nil {
			result.UpdatedFiles = append(result.UpdatedFiles, claudeMD)
		}
	}

	return result, nil
}

// PortEndResult contains the result of port-end operation
type PortEndResult struct {
	PortID       string
	WorkerID     string
	Completed    bool
	NextWorkerID string
	Message      string
}

// ProcessPortEnd handles the port-end hook
func (s *ClaudeService) ProcessPortEnd(portID, workerID string) (*PortEndResult, error) {
	result := &PortEndResult{
		PortID:   portID,
		WorkerID: workerID,
	}

	// Clear active worker section in CLAUDE.md
	claudeMD := FindClaudeMD(s.projectRoot)
	if claudeMD != "" {
		s.clearActiveWorkerSection(claudeMD)
	}

	result.Completed = true
	result.Message = fmt.Sprintf("포트 %s 작업 완료 (워커: %s)", portID, workerID)

	return result, nil
}

// GetCurrentContext returns the current context for display
func (s *ClaudeService) GetCurrentContext(portID, workerID string) (string, error) {
	// Load port spec if port ID is provided
	var portSpec string
	if portID != "" {
		spec, err := s.loadPortSpec(portID)
		if err == nil {
			portSpec = spec
		}
	}

	// If no worker ID, try to determine from port
	if workerID == "" && portID != "" {
		hints := worker.ParsePortSpecHints(portSpec)
		workerID, _ = s.workerMapper.MapPortToWorker(hints)
	}

	if workerID == "" {
		return "", fmt.Errorf("워커가 지정되지 않았습니다")
	}

	buildResult, err := s.promptBuilder.BuildContext(portID, workerID, portSpec)
	if err != nil {
		return "", err
	}

	return buildResult.ToMarkdown(), nil
}

// SwitchWorker switches to a different worker
func (s *ClaudeService) SwitchWorker(workerID string, portID string) error {
	// Verify worker exists
	_, err := s.workerMapper.GetWorker(workerID)
	if err != nil {
		return fmt.Errorf("워커를 찾을 수 없습니다: %s", workerID)
	}

	// Load port spec if provided
	var portSpec string
	if portID != "" {
		spec, err := s.loadPortSpec(portID)
		if err == nil {
			portSpec = spec
		}
	}

	// Build context for new worker
	buildResult, err := s.promptBuilder.BuildContext(portID, workerID, portSpec)
	if err != nil {
		return fmt.Errorf("컨텍스트 빌드 실패: %w", err)
	}

	// Update CLAUDE.md
	claudeMD := FindClaudeMD(s.projectRoot)
	if claudeMD != "" {
		return s.updateActiveWorkerSection(claudeMD, workerID, portID, buildResult)
	}

	return nil
}

// ListAvailableWorkers returns all available workers
func (s *ClaudeService) ListAvailableWorkers() ([]*worker.WorkerSpec, error) {
	return s.workerMapper.ListWorkers()
}

// loadPortSpec loads a port specification file
func (s *ClaudeService) loadPortSpec(portID string) (string, error) {
	portsDir := filepath.Join(s.projectRoot, "ports")

	// Try exact match first
	path := filepath.Join(portsDir, portID+".md")
	content, err := os.ReadFile(path)
	if err == nil {
		return string(content), nil
	}

	// Try with patterns
	patterns := []string{
		filepath.Join(portsDir, portID+".md"),
		filepath.Join(portsDir, portID+"-*.md"),
		filepath.Join(portsDir, "*-"+portID+".md"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			content, err := os.ReadFile(matches[0])
			if err == nil {
				return string(content), nil
			}
		}
	}

	return "", fmt.Errorf("포트 명세를 찾을 수 없습니다: %s", portID)
}

// updateActiveWorkerSection updates the active worker section in CLAUDE.md
func (s *ClaudeService) updateActiveWorkerSection(filePath, workerID, portID string, buildResult *prompt.BuildResult) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Generate active worker section
	activeSection := s.generateActiveWorkerSection(workerID, portID, buildResult)

	contentStr := string(content)

	// Check if markers exist
	if !strings.Contains(contentStr, activeWorkerStartMarker) {
		// Add markers before context markers if they exist
		if idx := strings.Index(contentStr, contextStartMarker); idx >= 0 {
			contentStr = contentStr[:idx] + activeWorkerStartMarker + "\n" + activeWorkerEndMarker + "\n\n" + contentStr[idx:]
		} else {
			// Add at the end
			contentStr += "\n\n" + activeWorkerStartMarker + "\n" + activeWorkerEndMarker + "\n"
		}
	}

	// Replace content between markers
	lines := strings.Split(contentStr, "\n")
	var result []string
	inSection := false

	for _, line := range lines {
		if strings.Contains(line, activeWorkerStartMarker) {
			result = append(result, line)
			result = append(result, activeSection)
			inSection = true
			continue
		}
		if strings.Contains(line, activeWorkerEndMarker) {
			result = append(result, line)
			inSection = false
			continue
		}
		if !inSection {
			result = append(result, line)
		}
	}

	return os.WriteFile(filePath, []byte(strings.Join(result, "\n")), 0644)
}

// clearActiveWorkerSection removes the active worker section content
func (s *ClaudeService) clearActiveWorkerSection(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, activeWorkerStartMarker) {
		return nil
	}

	lines := strings.Split(contentStr, "\n")
	var result []string
	inSection := false

	for _, line := range lines {
		if strings.Contains(line, activeWorkerStartMarker) {
			result = append(result, line)
			inSection = true
			continue
		}
		if strings.Contains(line, activeWorkerEndMarker) {
			result = append(result, line)
			inSection = false
			continue
		}
		if !inSection {
			result = append(result, line)
		}
	}

	return os.WriteFile(filePath, []byte(strings.Join(result, "\n")), 0644)
}

// generateActiveWorkerSection generates the active worker section content
func (s *ClaudeService) generateActiveWorkerSection(workerID, portID string, buildResult *prompt.BuildResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("> 업데이트: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("### 현재 활성 워커\n\n")

	// Worker info
	workerSpec, err := s.workerMapper.GetWorker(workerID)
	if err == nil {
		sb.WriteString(fmt.Sprintf("- **워커**: %s (`%s`)\n", workerSpec.Agent.Name, workerID))
		sb.WriteString(fmt.Sprintf("- **레이어**: %s\n", workerSpec.Agent.Layer))
		sb.WriteString(fmt.Sprintf("- **기술**: %s (%s)\n",
			workerSpec.Agent.Tech.Language,
			strings.Join(workerSpec.Agent.Tech.Frameworks, ", ")))
	} else {
		sb.WriteString(fmt.Sprintf("- **워커**: `%s`\n", workerID))
	}

	if portID != "" {
		sb.WriteString(fmt.Sprintf("- **포트**: `%s`\n", portID))
	}

	sb.WriteString("\n")

	// Checklist
	if len(buildResult.Checklist) > 0 {
		sb.WriteString("### 체크리스트\n\n")
		for _, item := range buildResult.Checklist {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
		sb.WriteString("\n")
	}

	// Token info
	sb.WriteString(fmt.Sprintf("*컨텍스트 토큰: ~%d*\n", buildResult.TotalTokens()))

	return sb.String()
}

// ReloadContext reloads the context for the current worker
func (s *ClaudeService) ReloadContext() (*PortStartResult, error) {
	// Read current state from CLAUDE.md
	claudeMD := FindClaudeMD(s.projectRoot)
	if claudeMD == "" {
		return nil, fmt.Errorf("CLAUDE.md를 찾을 수 없습니다")
	}

	content, err := os.ReadFile(claudeMD)
	if err != nil {
		return nil, err
	}

	// Parse current worker and port from active section
	workerID, portID := s.parseActiveWorkerFromContent(string(content))
	if workerID == "" {
		return nil, fmt.Errorf("활성 워커가 없습니다")
	}

	// Reload by processing port start again
	if portID != "" {
		return s.ProcessPortStart(portID)
	}

	// Just reload worker context
	result := &PortStartResult{
		WorkerID: workerID,
	}

	buildResult, err := s.promptBuilder.BuildContext("", workerID, "")
	if err != nil {
		return nil, err
	}

	result.Context = buildResult.ToMarkdown()
	result.Checklist = buildResult.Checklist
	result.TokenCount = buildResult.TotalTokens()

	return result, nil
}

// parseActiveWorkerFromContent extracts worker and port ID from CLAUDE.md content
func (s *ClaudeService) parseActiveWorkerFromContent(content string) (workerID, portID string) {
	// Find active worker section
	startIdx := strings.Index(content, activeWorkerStartMarker)
	endIdx := strings.Index(content, activeWorkerEndMarker)

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return "", ""
	}

	section := content[startIdx:endIdx]

	// Parse worker ID
	if idx := strings.Index(section, "**워커**:"); idx >= 0 {
		line := section[idx:]
		if endLine := strings.Index(line, "\n"); endLine > 0 {
			line = line[:endLine]
		}
		// Extract ID from backticks
		if start := strings.Index(line, "`"); start >= 0 {
			rest := line[start+1:]
			if end := strings.Index(rest, "`"); end >= 0 {
				workerID = rest[:end]
			}
		}
	}

	// Parse port ID
	if idx := strings.Index(section, "**포트**:"); idx >= 0 {
		line := section[idx:]
		if endLine := strings.Index(line, "\n"); endLine > 0 {
			line = line[:endLine]
		}
		if start := strings.Index(line, "`"); start >= 0 {
			rest := line[start+1:]
			if end := strings.Index(rest, "`"); end >= 0 {
				portID = rest[:end]
			}
		}
	}

	return workerID, portID
}
