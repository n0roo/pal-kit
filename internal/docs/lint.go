package docs

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// LintSeverity represents the severity of a lint issue
type LintSeverity string

const (
	SeverityError   LintSeverity = "error"
	SeverityWarning LintSeverity = "warning"
	SeverityInfo    LintSeverity = "info"
)

// LintIssue represents a lint issue
type LintIssue struct {
	Path     string       `json:"path"`
	Line     int          `json:"line,omitempty"`
	Severity LintSeverity `json:"severity"`
	Rule     string       `json:"rule"`
	Message  string       `json:"message"`
}

// LintResult represents lint results for a document
type LintResult struct {
	Path   string      `json:"path"`
	Valid  bool        `json:"valid"`
	Issues []LintIssue `json:"issues"`
}

// LintOptions configures lint behavior
type LintOptions struct {
	IgnoreWarnings bool
	IgnoreInfo     bool
	Rules          []string // 특정 규칙만 실행
}

// Lint validates all documents
func (s *Service) Lint(opts *LintOptions) ([]LintResult, error) {
	if opts == nil {
		opts = &LintOptions{}
	}

	docs, err := s.List()
	if err != nil {
		return nil, err
	}

	var results []LintResult
	for _, doc := range docs {
		result := s.lintDocument(&doc, opts)
		results = append(results, result)
	}

	return results, nil
}

// LintFile validates a single file
func (s *Service) LintFile(path string, opts *LintOptions) (*LintResult, error) {
	if opts == nil {
		opts = &LintOptions{}
	}

	doc, err := s.Get(path)
	if err != nil {
		return nil, err
	}

	result := s.lintDocument(doc, opts)
	return &result, nil
}

// lintDocument validates a document
func (s *Service) lintDocument(doc *Document, opts *LintOptions) LintResult {
	result := LintResult{
		Path:   doc.RelativePath,
		Valid:  true,
		Issues: []LintIssue{},
	}

	content, err := os.ReadFile(doc.Path)
	if err != nil {
		result.Valid = false
		result.Issues = append(result.Issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityError,
			Rule:     "file-readable",
			Message:  fmt.Sprintf("파일을 읽을 수 없습니다: %v", err),
		})
		return result
	}

	contentStr := string(content)

	// 공통 검사
	result.Issues = append(result.Issues, s.lintCommon(doc, contentStr)...)

	// 타입별 검사
	switch doc.Type {
	case DocTypeClaude:
		result.Issues = append(result.Issues, s.lintClaudeMD(doc, contentStr)...)
	case DocTypeAgent:
		result.Issues = append(result.Issues, s.lintAgent(doc, contentStr)...)
	case DocTypePort:
		result.Issues = append(result.Issues, s.lintPort(doc, contentStr)...)
	case DocTypeConvention:
		result.Issues = append(result.Issues, s.lintConvention(doc, contentStr)...)
	}

	// 필터링
	var filteredIssues []LintIssue
	for _, issue := range result.Issues {
		if opts.IgnoreWarnings && issue.Severity == SeverityWarning {
			continue
		}
		if opts.IgnoreInfo && issue.Severity == SeverityInfo {
			continue
		}
		if issue.Severity == SeverityError {
			result.Valid = false
		}
		filteredIssues = append(filteredIssues, issue)
	}
	result.Issues = filteredIssues

	return result
}

// lintCommon performs common lint checks
func (s *Service) lintCommon(doc *Document, content string) []LintIssue {
	var issues []LintIssue

	// 빈 파일 검사
	if strings.TrimSpace(content) == "" {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityError,
			Rule:     "non-empty",
			Message:  "파일이 비어있습니다",
		})
	}

	// 파일 크기 검사
	if doc.Size > 100*1024 { // 100KB
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityWarning,
			Rule:     "file-size",
			Message:  fmt.Sprintf("파일이 너무 큽니다 (%d KB)", doc.Size/1024),
		})
	}

	// TODO 검사
	todoCount := strings.Count(strings.ToUpper(content), "TODO")
	if todoCount > 0 {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityInfo,
			Rule:     "todo-items",
			Message:  fmt.Sprintf("TODO 항목 %d개", todoCount),
		})
	}

	// 트레일링 공백 검사
	lines := strings.Split(content, "\n")
	trailingCount := 0
	for _, line := range lines {
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			trailingCount++
		}
	}
	if trailingCount > 0 {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityInfo,
			Rule:     "trailing-whitespace",
			Message:  fmt.Sprintf("트레일링 공백 %d줄", trailingCount),
		})
	}

	return issues
}

// lintClaudeMD validates CLAUDE.md
func (s *Service) lintClaudeMD(doc *Document, content string) []LintIssue {
	var issues []LintIssue

	// 필수 섹션 검사
	requiredSections := []string{"프로젝트 개요", "개발 규칙", "디렉토리 구조"}
	for _, section := range requiredSections {
		if !strings.Contains(content, section) && !strings.Contains(content, strings.ToLower(section)) {
			issues = append(issues, LintIssue{
				Path:     doc.RelativePath,
				Severity: SeverityWarning,
				Rule:     "claude-sections",
				Message:  fmt.Sprintf("권장 섹션 누락: %s", section),
			})
		}
	}

	// PAL 컨텍스트 마커 검사
	if !strings.Contains(content, "pal:context:start") {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityInfo,
			Rule:     "pal-context",
			Message:  "PAL 컨텍스트 마커가 없습니다 (pal context inject로 추가)",
		})
	}

	return issues
}

// lintAgent validates agent files
func (s *Service) lintAgent(doc *Document, content string) []LintIssue {
	var issues []LintIssue

	// YAML 파일만 검사
	if !strings.HasSuffix(doc.Path, ".yaml") && !strings.HasSuffix(doc.Path, ".yml") {
		return issues
	}

	// YAML 파싱
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityError,
			Rule:     "yaml-valid",
			Message:  fmt.Sprintf("YAML 파싱 실패: %v", err),
		})
		return issues
	}

	// agent 키 확인
	agent, ok := data["agent"].(map[string]interface{})
	if !ok {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityError,
			Rule:     "agent-structure",
			Message:  "'agent' 키가 없거나 형식이 잘못됨",
		})
		return issues
	}

	// 필수 필드
	requiredFields := []string{"id", "name", "type", "prompt"}
	for _, field := range requiredFields {
		if _, exists := agent[field]; !exists {
			issues = append(issues, LintIssue{
				Path:     doc.RelativePath,
				Severity: SeverityError,
				Rule:     "agent-required-fields",
				Message:  fmt.Sprintf("필수 필드 누락: %s", field),
			})
		}
	}

	// 유효한 타입 검사
	if agentType, ok := agent["type"].(string); ok {
		validTypes := map[string]bool{
			"builder": true, "worker": true, "reviewer": true,
			"planner": true, "tester": true, "docs": true, "custom": true,
		}
		if !validTypes[agentType] {
			issues = append(issues, LintIssue{
				Path:     doc.RelativePath,
				Severity: SeverityWarning,
				Rule:     "agent-type",
				Message:  fmt.Sprintf("알 수 없는 에이전트 타입: %s", agentType),
			})
		}
	}

	// 프롬프트 길이 검사
	if prompt, ok := agent["prompt"].(string); ok {
		if len(prompt) < 50 {
			issues = append(issues, LintIssue{
				Path:     doc.RelativePath,
				Severity: SeverityWarning,
				Rule:     "agent-prompt-length",
				Message:  "프롬프트가 너무 짧습니다 (50자 미만)",
			})
		}
	}

	return issues
}

// lintPort validates port spec files
func (s *Service) lintPort(doc *Document, content string) []LintIssue {
	var issues []LintIssue

	// 제목 검사
	if !strings.HasPrefix(strings.TrimSpace(content), "#") {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityWarning,
			Rule:     "port-title",
			Message:  "포트 명세는 제목(#)으로 시작해야 합니다",
		})
	}

	// Port ID 검사
	portIDPattern := regexp.MustCompile(`Port ID:\s*(\S+)`)
	if !portIDPattern.MatchString(content) {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityWarning,
			Rule:     "port-id",
			Message:  "Port ID 필드가 없습니다",
		})
	}

	// 완료 조건 검사
	if !strings.Contains(content, "완료 조건") && !strings.Contains(content, "Acceptance") {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityInfo,
			Rule:     "port-acceptance",
			Message:  "완료 조건 섹션을 추가하세요",
		})
	}

	// 체크박스 검사
	checkboxPattern := regexp.MustCompile(`- \[[ x]\]`)
	if !checkboxPattern.MatchString(content) {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityInfo,
			Rule:     "port-checklist",
			Message:  "체크리스트(- [ ])를 추가하면 진행 상황을 추적할 수 있습니다",
		})
	}

	return issues
}

// lintConvention validates convention files
func (s *Service) lintConvention(doc *Document, content string) []LintIssue {
	var issues []LintIssue

	// 제목 검사
	if !strings.HasPrefix(strings.TrimSpace(content), "#") {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityWarning,
			Rule:     "convention-title",
			Message:  "컨벤션 문서는 제목(#)으로 시작해야 합니다",
		})
	}

	// 예시 섹션 검사
	if !strings.Contains(content, "예시") && !strings.Contains(content, "Example") {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityInfo,
			Rule:     "convention-examples",
			Message:  "예시 섹션을 추가하면 이해하기 쉽습니다",
		})
	}

	// Good/Bad 예시 검사
	hasGood := strings.Contains(content, "Good") || strings.Contains(content, "좋은")
	hasBad := strings.Contains(content, "Bad") || strings.Contains(content, "나쁜")
	if !hasGood || !hasBad {
		issues = append(issues, LintIssue{
			Path:     doc.RelativePath,
			Severity: SeverityInfo,
			Rule:     "convention-good-bad",
			Message:  "Good/Bad 예시를 모두 포함하면 명확합니다",
		})
	}

	return issues
}

// LintSummary returns a summary of lint results
func LintSummary(results []LintResult) map[LintSeverity]int {
	summary := map[LintSeverity]int{
		SeverityError:   0,
		SeverityWarning: 0,
		SeverityInfo:    0,
	}

	for _, result := range results {
		for _, issue := range result.Issues {
			summary[issue.Severity]++
		}
	}

	return summary
}
