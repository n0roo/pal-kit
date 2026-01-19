package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// QualityService handles document quality checking
type QualityService struct {
	vaultPath  string
	classifier *ClassifierService
	linkSvc    *LinkService
}

// QualityResult represents the quality check result
type QualityResult struct {
	FilePath   string         `json:"file_path"`
	Score      int            `json:"score"`      // 0-100
	Grade      string         `json:"grade"`      // A, B, C, D, F
	Issues     []QualityIssue `json:"issues"`
	Warnings   []QualityIssue `json:"warnings"`
	Passed     bool           `json:"passed"`
	Summary    string         `json:"summary"`
}

// QualityIssue represents a quality issue
type QualityIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"` // error, warning, info
	Message     string `json:"message"`
	Line        int    `json:"line,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// QualityOptions represents quality check options
type QualityOptions struct {
	CheckLinks     bool   `json:"check_links"`
	CheckTags      bool   `json:"check_tags"`
	StrictMode     bool   `json:"strict_mode"`
	RequiredFields []string `json:"required_fields,omitempty"`
}

// NewQualityService creates a new quality service
func NewQualityService(vaultPath string) *QualityService {
	return &QualityService{
		vaultPath:  vaultPath,
		classifier: NewClassifierService(vaultPath),
		linkSvc:    NewLinkService(vaultPath),
	}
}

// Check performs quality check on a document
func (q *QualityService) Check(filePath string, opts *QualityOptions) (*QualityResult, error) {
	if opts == nil {
		opts = &QualityOptions{
			CheckLinks: true,
			CheckTags:  true,
		}
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}

	result := &QualityResult{
		FilePath: filePath,
		Score:    100,
		Issues:   []QualityIssue{},
		Warnings: []QualityIssue{},
	}

	// Parse content
	meta, body := q.parseFrontmatter(string(content))
	lines := strings.Split(string(content), "\n")

	// 1. Check frontmatter existence
	if !strings.HasPrefix(string(content), "---") {
		result.addIssue("frontmatter", "error", "Frontmatter가 없습니다", 1, "---로 시작하는 YAML frontmatter를 추가하세요")
	} else {
		// 2. Check required fields
		q.checkRequiredFields(result, meta, opts)
	}

	// 3. Check title
	q.checkTitle(result, meta, body)

	// 4. Check content structure
	q.checkStructure(result, body, lines)

	// 5. Check links if enabled
	if opts.CheckLinks {
		q.checkLinks(result, filePath, body)
	}

	// 6. Check tags if enabled
	if opts.CheckTags {
		q.checkTags(result, meta, body)
	}

	// 7. Check aliases
	q.checkAliases(result, meta)

	// Calculate final score and grade
	q.calculateScore(result, opts.StrictMode)

	return result, nil
}

// CheckDirectory checks all markdown files in a directory
func (q *QualityService) CheckDirectory(dirPath string, opts *QualityOptions) ([]*QualityResult, error) {
	results := []*QualityResult{}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Skip TOC files
		if strings.HasPrefix(filepath.Base(path), "_toc") {
			return nil
		}

		result, err := q.Check(path, opts)
		if err != nil {
			// Add error as a result
			results = append(results, &QualityResult{
				FilePath: path,
				Score:    0,
				Grade:    "F",
				Passed:   false,
				Issues: []QualityIssue{{
					Type:     "file",
					Severity: "error",
					Message:  err.Error(),
				}},
			})
			return nil
		}

		results = append(results, result)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (q *QualityService) parseFrontmatter(content string) (map[string]any, string) {
	meta := make(map[string]any)
	body := content

	if !strings.HasPrefix(content, "---") {
		return meta, body
	}

	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		return meta, body
	}

	yaml.Unmarshal([]byte(parts[0]), &meta)
	body = strings.TrimSpace(parts[1])

	return meta, body
}

func (q *QualityService) checkRequiredFields(result *QualityResult, meta map[string]any, opts *QualityOptions) {
	// Default required fields
	baseRequired := []string{"title"}

	// Check document type and add type-specific required fields
	docType, hasType := meta["type"].(string)
	if !hasType {
		result.addWarning("frontmatter", "warning", "type 필드가 없습니다", "문서 타입을 지정하세요 (예: type: concept)")
	}

	// Type-specific required fields
	typeRequired := map[string][]string{
		"port":      {"status", "priority"},
		"adr":       {"status", "decision_date"},
		"concept":   {"domain"},
		"guide":     {},
		"session":   {"date"},
	}

	requiredFields := append(baseRequired, opts.RequiredFields...)
	if typeReq, ok := typeRequired[docType]; ok {
		requiredFields = append(requiredFields, typeReq...)
	}

	for _, field := range requiredFields {
		if _, ok := meta[field]; !ok {
			result.addIssue("frontmatter", "error",
				fmt.Sprintf("필수 필드 '%s'가 없습니다", field),
				0,
				fmt.Sprintf("%s: <값> 을 frontmatter에 추가하세요", field))
		}
	}
}

func (q *QualityService) checkTitle(result *QualityResult, meta map[string]any, body string) {
	// Check frontmatter title
	title, hasTitle := meta["title"].(string)
	if hasTitle && title == "" {
		result.addIssue("title", "error", "title이 비어있습니다", 0, "의미있는 제목을 입력하세요")
	}

	// Check H1 heading
	h1Regex := regexp.MustCompile(`(?m)^# (.+)$`)
	h1Matches := h1Regex.FindStringSubmatch(body)

	if len(h1Matches) == 0 {
		result.addWarning("structure", "warning", "H1 제목이 없습니다", "# 제목 형식의 헤더를 추가하세요")
	} else if hasTitle && h1Matches[1] != title {
		result.addWarning("consistency", "warning",
			fmt.Sprintf("frontmatter title('%s')과 H1('%s')이 다릅니다", title, h1Matches[1]),
			"일관성을 위해 제목을 통일하세요")
	}
}

func (q *QualityService) checkStructure(result *QualityResult, body string, lines []string) {
	// Check for empty document
	trimmedBody := strings.TrimSpace(body)
	if trimmedBody == "" {
		result.addIssue("content", "error", "문서 내용이 비어있습니다", 0, "문서 내용을 작성하세요")
		return
	}

	// Check minimum content length
	if len(trimmedBody) < 50 {
		result.addWarning("content", "warning", "문서 내용이 너무 짧습니다", "더 상세한 내용을 추가하세요")
	}

	// Check heading hierarchy
	prevLevel := 0
	headingRegex := regexp.MustCompile(`^(#{1,6}) `)

	for i, line := range lines {
		matches := headingRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			level := len(matches[1])
			if prevLevel > 0 && level > prevLevel+1 {
				result.addWarning("structure", "warning",
					fmt.Sprintf("%d번 줄: 헤딩 레벨이 건너뛰어졌습니다 (H%d → H%d)", i+1, prevLevel, level),
					"순차적인 헤딩 레벨을 사용하세요")
			}
			prevLevel = level
		}
	}
}

func (q *QualityService) checkLinks(result *QualityResult, filePath, body string) {
	// Extract wikilinks
	wikilinkRegex := regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)
	matches := wikilinkRegex.FindAllStringSubmatch(body, -1)

	brokenLinks := []string{}
	for _, m := range matches {
		target := m[1]

		// Check if link target exists
		if !q.linkExists(filePath, target) {
			brokenLinks = append(brokenLinks, target)
		}
	}

	if len(brokenLinks) > 0 {
		result.addWarning("links", "warning",
			fmt.Sprintf("깨진 링크 %d개: %s", len(brokenLinks), strings.Join(brokenLinks, ", ")),
			"링크 대상 문서를 생성하거나 링크를 수정하세요")
	}
}

func (q *QualityService) linkExists(fromPath, target string) bool {
	// Simple check - look for file with the target name
	baseDir := filepath.Dir(fromPath)

	// Check various possible paths
	possiblePaths := []string{
		filepath.Join(baseDir, target+".md"),
		filepath.Join(baseDir, target, "_index.md"),
		filepath.Join(q.vaultPath, target+".md"),
		filepath.Join(q.vaultPath, target, "_index.md"),
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}

	return false
}

func (q *QualityService) checkTags(result *QualityResult, meta map[string]any, body string) {
	// Check frontmatter tags
	tags, hasTags := meta["tags"]
	if !hasTags {
		result.addWarning("tags", "warning", "tags 필드가 없습니다", "관련 태그를 추가하세요")
		return
	}

	// Check if tags is a list
	tagList, ok := tags.([]any)
	if !ok {
		result.addWarning("tags", "warning", "tags가 배열 형식이 아닙니다", "tags: [tag1, tag2] 형식을 사용하세요")
		return
	}

	if len(tagList) == 0 {
		result.addWarning("tags", "warning", "태그가 비어있습니다", "관련 태그를 추가하세요")
	}

	// Check tag format
	for _, tag := range tagList {
		tagStr, ok := tag.(string)
		if !ok {
			continue
		}

		// Check for spaces in tags
		if strings.Contains(tagStr, " ") {
			result.addWarning("tags", "warning",
				fmt.Sprintf("태그 '%s'에 공백이 포함되어 있습니다", tagStr),
				"공백 대신 하이픈(-)을 사용하세요")
		}

		// Check for uppercase
		if tagStr != strings.ToLower(tagStr) {
			result.addWarning("tags", "info",
				fmt.Sprintf("태그 '%s'에 대문자가 포함되어 있습니다", tagStr),
				"소문자 태그를 권장합니다")
		}
	}
}

func (q *QualityService) checkAliases(result *QualityResult, meta map[string]any) {
	aliases, hasAliases := meta["aliases"]
	if !hasAliases {
		// Aliases are optional, just a suggestion
		result.addWarning("aliases", "info", "aliases가 없습니다", "검색을 위한 별칭 추가를 고려하세요")
		return
	}

	aliasList, ok := aliases.([]any)
	if !ok {
		result.addWarning("aliases", "warning", "aliases가 배열 형식이 아닙니다", "aliases: [별칭1, 별칭2] 형식을 사용하세요")
	} else if len(aliasList) == 0 {
		result.addWarning("aliases", "info", "aliases가 비어있습니다", "검색을 위한 별칭 추가를 고려하세요")
	}
}

func (q *QualityService) calculateScore(result *QualityResult, strictMode bool) {
	// Start with 100
	score := 100

	// Deduct for issues
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "error":
			score -= 15
		case "warning":
			score -= 5
		case "info":
			score -= 1
		}
	}

	// Deduct for warnings
	for _, warn := range result.Warnings {
		switch warn.Severity {
		case "error":
			score -= 10
		case "warning":
			score -= 3
		case "info":
			score -= 1
		}
	}

	// Ensure score is in range
	if score < 0 {
		score = 0
	}

	result.Score = score

	// Determine grade
	switch {
	case score >= 90:
		result.Grade = "A"
	case score >= 80:
		result.Grade = "B"
	case score >= 70:
		result.Grade = "C"
	case score >= 60:
		result.Grade = "D"
	default:
		result.Grade = "F"
	}

	// Determine pass/fail
	if strictMode {
		result.Passed = len(result.Issues) == 0
	} else {
		result.Passed = score >= 60
	}

	// Generate summary
	errorCount := len(result.Issues)
	warningCount := len(result.Warnings)
	result.Summary = fmt.Sprintf("점수: %d/100 (등급: %s) - 오류: %d, 경고: %d",
		score, result.Grade, errorCount, warningCount)
}

func (r *QualityResult) addIssue(issueType, severity, message string, line int, suggestion string) {
	r.Issues = append(r.Issues, QualityIssue{
		Type:       issueType,
		Severity:   severity,
		Message:    message,
		Line:       line,
		Suggestion: suggestion,
	})
}

func (r *QualityResult) addWarning(issueType, severity, message, suggestion string) {
	r.Warnings = append(r.Warnings, QualityIssue{
		Type:       issueType,
		Severity:   severity,
		Message:    message,
		Suggestion: suggestion,
	})
}

// GetQualitySummary returns a summary of multiple quality results
func GetQualitySummary(results []*QualityResult) map[string]any {
	summary := map[string]any{
		"total":        len(results),
		"passed":       0,
		"failed":       0,
		"avg_score":    0.0,
		"grade_counts": map[string]int{"A": 0, "B": 0, "C": 0, "D": 0, "F": 0},
		"common_issues": map[string]int{},
	}

	totalScore := 0
	issueCounts := make(map[string]int)

	for _, r := range results {
		totalScore += r.Score
		if r.Passed {
			summary["passed"] = summary["passed"].(int) + 1
		} else {
			summary["failed"] = summary["failed"].(int) + 1
		}

		gradeCounts := summary["grade_counts"].(map[string]int)
		gradeCounts[r.Grade]++

		for _, issue := range r.Issues {
			issueCounts[issue.Type]++
		}
	}

	if len(results) > 0 {
		summary["avg_score"] = float64(totalScore) / float64(len(results))
	}
	summary["common_issues"] = issueCounts

	return summary
}
