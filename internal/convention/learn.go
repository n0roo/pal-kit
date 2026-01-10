package convention

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Pattern represents a detected pattern
type Pattern struct {
	Type        string `json:"type"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	Occurrences int    `json:"occurrences"`
	Examples    []string `json:"examples,omitempty"`
}

// LearnResult represents the result of pattern learning
type LearnResult struct {
	Patterns   []Pattern `json:"patterns"`
	FilesScanned int     `json:"files_scanned"`
	Suggestions []Suggestion `json:"suggestions,omitempty"`
}

// Suggestion represents a convention suggestion
type Suggestion struct {
	Type        ConventionType `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Rules       []Rule         `json:"rules"`
	Confidence  float64        `json:"confidence"` // 0.0 - 1.0
}

// Learn analyzes files and learns patterns
func (s *Service) Learn(paths []string, fileTypes []string) (*LearnResult, error) {
	result := &LearnResult{
		Patterns: []Pattern{},
	}

	// 파일 수집
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			collected, _ := s.collectFiles(path, fileTypes)
			files = append(files, collected...)
		} else {
			files = append(files, path)
		}
	}

	result.FilesScanned = len(files)

	// 패턴 분석
	namingPatterns := s.analyzeNamingPatterns(files)
	result.Patterns = append(result.Patterns, namingPatterns...)

	importPatterns := s.analyzeImportPatterns(files)
	result.Patterns = append(result.Patterns, importPatterns...)

	commentPatterns := s.analyzeCommentPatterns(files)
	result.Patterns = append(result.Patterns, commentPatterns...)

	// 제안 생성
	result.Suggestions = s.generateSuggestions(result.Patterns)

	return result, nil
}

// collectFiles collects files from a directory
func (s *Service) collectFiles(dir string, fileTypes []string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// 숨김 디렉토리 스킵
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// 파일 타입 필터
		if len(fileTypes) > 0 {
			ext := strings.ToLower(filepath.Ext(path))
			matched := false
			for _, ft := range fileTypes {
				if ext == ft || ext == "."+ft {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// analyzeNamingPatterns analyzes naming conventions
func (s *Service) analyzeNamingPatterns(files []string) []Pattern {
	patterns := make(map[string]int)
	examples := make(map[string][]string)

	snakeCase := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	camelCase := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	pascalCase := regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*$`)
	kebabCase := regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

	for _, file := range files {
		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

		var patternType string
		switch {
		case snakeCase.MatchString(baseName):
			patternType = "snake_case"
		case camelCase.MatchString(baseName):
			patternType = "camelCase"
		case pascalCase.MatchString(baseName):
			patternType = "PascalCase"
		case kebabCase.MatchString(baseName):
			patternType = "kebab-case"
		default:
			patternType = "mixed"
		}

		patterns[patternType]++
		if len(examples[patternType]) < 3 {
			examples[patternType] = append(examples[patternType], baseName)
		}
	}

	var result []Pattern
	for patternType, count := range patterns {
		result = append(result, Pattern{
			Type:        "naming",
			Pattern:     patternType,
			Description: "파일명 네이밍 패턴",
			Occurrences: count,
			Examples:    examples[patternType],
		})
	}

	// 많이 사용된 순으로 정렬
	sort.Slice(result, func(i, j int) bool {
		return result[i].Occurrences > result[j].Occurrences
	})

	return result
}

// analyzeImportPatterns analyzes import/require patterns
func (s *Service) analyzeImportPatterns(files []string) []Pattern {
	patterns := make(map[string]int)

	groupedImports := regexp.MustCompile(`(?m)^import \([\s\S]*?\)`)
	singleImports := regexp.MustCompile(`(?m)^import "[^"]+"|^import '[^']+'`)
	requirePattern := regexp.MustCompile(`(?m)^(const|let|var)\s+\w+\s*=\s*require\(`)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		contentStr := string(content)

		if groupedImports.MatchString(contentStr) {
			patterns["grouped_imports"]++
		}
		if singleImports.MatchString(contentStr) {
			patterns["single_imports"]++
		}
		if requirePattern.MatchString(contentStr) {
			patterns["commonjs_require"]++
		}
	}

	var result []Pattern
	for patternType, count := range patterns {
		result = append(result, Pattern{
			Type:        "import",
			Pattern:     patternType,
			Description: "임포트 스타일",
			Occurrences: count,
		})
	}

	return result
}

// analyzeCommentPatterns analyzes comment patterns
func (s *Service) analyzeCommentPatterns(files []string) []Pattern {
	patterns := make(map[string]int)

	todoPattern := regexp.MustCompile(`(?i)//\s*TODO:|#\s*TODO:`)
	fixmePattern := regexp.MustCompile(`(?i)//\s*FIXME:|#\s*FIXME:`)
	docComment := regexp.MustCompile(`(?m)^/\*\*[\s\S]*?\*/|^///`)
	hashComment := regexp.MustCompile(`(?m)^#[^!]`)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		contentStr := string(content)

		if todoPattern.MatchString(contentStr) {
			patterns["todo_comments"]++
		}
		if fixmePattern.MatchString(contentStr) {
			patterns["fixme_comments"]++
		}
		if docComment.MatchString(contentStr) {
			patterns["doc_comments"]++
		}
		if hashComment.MatchString(contentStr) {
			patterns["hash_comments"]++
		}
	}

	var result []Pattern
	for patternType, count := range patterns {
		result = append(result, Pattern{
			Type:        "comment",
			Pattern:     patternType,
			Description: "주석 스타일",
			Occurrences: count,
		})
	}

	return result
}

// generateSuggestions generates convention suggestions from patterns
func (s *Service) generateSuggestions(patterns []Pattern) []Suggestion {
	var suggestions []Suggestion

	// 네이밍 컨벤션 제안
	var dominantNaming string
	var maxCount int
	for _, p := range patterns {
		if p.Type == "naming" && p.Occurrences > maxCount {
			dominantNaming = p.Pattern
			maxCount = p.Occurrences
		}
	}

	if dominantNaming != "" && maxCount > 3 {
		suggestions = append(suggestions, Suggestion{
			Type:        TypeNaming,
			Name:        "file-naming",
			Description: "파일 네이밍 컨벤션",
			Rules: []Rule{
				{
					ID:          "file-name-style",
					Description: "파일명은 " + dominantNaming + " 스타일을 사용합니다",
					Severity:    "warning",
				},
			},
			Confidence: float64(maxCount) / float64(len(patterns)),
		})
	}

	return suggestions
}

// ProposeFromSuggestion creates a convention from a suggestion
func (s *Service) ProposeFromSuggestion(suggestion *Suggestion) (*Convention, error) {
	conv := &Convention{
		ID:          suggestion.Name,
		Name:        suggestion.Name,
		Type:        suggestion.Type,
		Description: suggestion.Description,
		Rules:       suggestion.Rules,
		Enabled:     false, // 제안은 기본적으로 비활성화
		Priority:    5,
	}

	return conv, nil
}

// AnalyzeCommitMessages analyzes git commit messages
func (s *Service) AnalyzeCommitMessages(messages []string) []Pattern {
	patterns := make(map[string]int)
	examples := make(map[string][]string)

	// 커밋 메시지 패턴
	conventional := regexp.MustCompile(`^(feat|fix|docs|style|refactor|test|chore)(\(.+\))?:`)
	gitmoji := regexp.MustCompile(`^:[a-z_]+:`)
	ticketRef := regexp.MustCompile(`^[A-Z]+-\d+`)
	imperative := regexp.MustCompile(`^(Add|Fix|Update|Remove|Refactor|Implement)`)

	for _, msg := range messages {
		// 첫 줄만 분석
		firstLine := msg
		if idx := strings.Index(msg, "\n"); idx > 0 {
			firstLine = msg[:idx]
		}

		var patternType string
		switch {
		case conventional.MatchString(firstLine):
			patternType = "conventional"
		case gitmoji.MatchString(firstLine):
			patternType = "gitmoji"
		case ticketRef.MatchString(firstLine):
			patternType = "ticket-reference"
		case imperative.MatchString(firstLine):
			patternType = "imperative"
		default:
			patternType = "freeform"
		}

		patterns[patternType]++
		if len(examples[patternType]) < 3 {
			examples[patternType] = append(examples[patternType], firstLine)
		}
	}

	var result []Pattern
	for patternType, count := range patterns {
		result = append(result, Pattern{
			Type:        "commit-message",
			Pattern:     patternType,
			Description: "커밋 메시지 스타일",
			Occurrences: count,
			Examples:    examples[patternType],
		})
	}

	return result
}

// ReadGitLog reads recent git commit messages
func ReadGitLog(repoPath string, limit int) ([]string, error) {
	logPath := filepath.Join(repoPath, ".git", "logs", "HEAD")
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Git log format: <old-sha> <new-sha> <author> <timestamp> <message>
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 2 {
			messages = append(messages, parts[1])
		}
	}

	// 최신 메시지만 반환
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}
