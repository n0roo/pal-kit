package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Service handles .claude/rules/ management
type Service struct {
	rulesDir string
}

// NewService creates a new rules service
func NewService(projectRoot string) *Service {
	return &Service{
		rulesDir: filepath.Join(projectRoot, ".claude", "rules"),
	}
}

// EnsureDir ensures the rules directory exists
func (s *Service) EnsureDir() error {
	return os.MkdirAll(s.rulesDir, 0755)
}

// ActivatePort creates a rule file for a port
func (s *Service) ActivatePort(portID, title, filePath string, filePatterns []string) error {
	if err := s.EnsureDir(); err != nil {
		return fmt.Errorf("rules 디렉토리 생성 실패: %w", err)
	}

	rulePath := filepath.Join(s.rulesDir, fmt.Sprintf("%s.md", portID))

	// paths 패턴 생성
	var paths []string
	if len(filePatterns) > 0 {
		paths = filePatterns
	} else {
		// 기본 패턴: 포트 명세 파일
		if filePath != "" {
			paths = append(paths, filePath)
		}
	}

	content := s.generateRuleContent(portID, title, paths)

	if err := os.WriteFile(rulePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("규칙 파일 생성 실패: %w", err)
	}

	return nil
}

// DeactivatePort removes a rule file for a port
func (s *Service) DeactivatePort(portID string) error {
	rulePath := filepath.Join(s.rulesDir, fmt.Sprintf("%s.md", portID))

	if err := os.Remove(rulePath); err != nil {
		if os.IsNotExist(err) {
			return nil // 이미 없으면 성공
		}
		return fmt.Errorf("규칙 파일 삭제 실패: %w", err)
	}

	return nil
}

// ListActiveRules returns list of active rule files
func (s *Service) ListActiveRules() ([]string, error) {
	entries, err := os.ReadDir(s.rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var rules []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			rules = append(rules, strings.TrimSuffix(entry.Name(), ".md"))
		}
	}

	return rules, nil
}

// GetRulePath returns the path to a rule file
func (s *Service) GetRulePath(portID string) string {
	return filepath.Join(s.rulesDir, fmt.Sprintf("%s.md", portID))
}

// RuleExists checks if a rule file exists
func (s *Service) RuleExists(portID string) bool {
	_, err := os.Stat(s.GetRulePath(portID))
	return err == nil
}

// generateRuleContent generates the rule file content
func (s *Service) generateRuleContent(portID, title string, paths []string) string {
	var sb strings.Builder

	// YAML frontmatter with paths
	sb.WriteString("---\n")
	if len(paths) > 0 {
		sb.WriteString("paths:\n")
		for _, p := range paths {
			sb.WriteString(fmt.Sprintf("  - %s\n", p))
		}
	}
	sb.WriteString("---\n\n")

	// Title
	if title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	} else {
		sb.WriteString(fmt.Sprintf("# Port: %s\n\n", portID))
	}

	// Metadata
	sb.WriteString(fmt.Sprintf("> Port ID: %s\n", portID))
	sb.WriteString(fmt.Sprintf("> Activated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// Instructions
	sb.WriteString("## 작업 지침\n\n")
	sb.WriteString("이 포트에서 작업 시 다음 사항을 준수하세요:\n\n")
	sb.WriteString("1. 포트 명세에 정의된 파일만 수정\n")
	sb.WriteString("2. 작업 시작 전 Lock 획득 확인\n")
	sb.WriteString("3. 완료 시 검증 명령 실행\n\n")

	// Commands
	sb.WriteString("## 실행 명령\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 상태 확인\n")
	sb.WriteString(fmt.Sprintf("pal port show %s\n\n", portID))
	sb.WriteString("# 완료 처리\n")
	sb.WriteString(fmt.Sprintf("pal port status %s complete\n", portID))
	sb.WriteString("```\n")

	return sb.String()
}

// ActivatePortWithSpec creates a rule file with port specification content
func (s *Service) ActivatePortWithSpec(portID, title, specPath string, filePatterns []string) error {
	if err := s.EnsureDir(); err != nil {
		return fmt.Errorf("rules 디렉토리 생성 실패: %w", err)
	}

	rulePath := filepath.Join(s.rulesDir, fmt.Sprintf("%s.md", portID))

	// 포트 명세 파일 읽기
	var specContent string
	if specPath != "" {
		if content, err := os.ReadFile(specPath); err == nil {
			specContent = string(content)
		}
	}

	content := s.generateRuleContentWithSpec(portID, title, filePatterns, specContent)

	if err := os.WriteFile(rulePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("규칙 파일 생성 실패: %w", err)
	}

	return nil
}

// generateRuleContentWithSpec generates rule content including port specification
func (s *Service) generateRuleContentWithSpec(portID, title string, paths []string, specContent string) string {
	var sb strings.Builder

	// YAML frontmatter with paths
	sb.WriteString("---\n")
	if len(paths) > 0 {
		sb.WriteString("paths:\n")
		for _, p := range paths {
			sb.WriteString(fmt.Sprintf("  - %s\n", p))
		}
	}
	sb.WriteString("---\n\n")

	// Title
	if title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	} else {
		sb.WriteString(fmt.Sprintf("# Port: %s\n\n", portID))
	}

	// Metadata
	sb.WriteString(fmt.Sprintf("> Port ID: %s\n", portID))
	sb.WriteString(fmt.Sprintf("> Activated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("> Status: running\n\n")

	// 포트 명세 내용 포함
	if specContent != "" {
		sb.WriteString("---\n\n")
		sb.WriteString(specContent)
		sb.WriteString("\n")
	}

	// 실행 명령
	sb.WriteString("\n---\n\n")
	sb.WriteString("## PAL 명령\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 포트 상태 확인\n")
	sb.WriteString(fmt.Sprintf("pal port show %s\n\n", portID))
	sb.WriteString("# 작업 완료\n")
	sb.WriteString(fmt.Sprintf("pal port status %s complete\n", portID))
	sb.WriteString("```\n")

	return sb.String()
}

// AppendToRule appends content to an existing rule file
func (s *Service) AppendToRule(portID, content string) error {
	rulePath := s.GetRulePath(portID)

	// 기존 파일 읽기
	existingContent, err := os.ReadFile(rulePath)
	if err != nil {
		return fmt.Errorf("규칙 파일 읽기 실패: %w", err)
	}

	// 새 내용 추가
	newContent := string(existingContent) + "\n" + content

	if err := os.WriteFile(rulePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("규칙 파일 업데이트 실패: %w", err)
	}

	return nil
}
