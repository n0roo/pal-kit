package context

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/escalation"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/session"
)

const (
	contextStartMarker = "<!-- pal:context:start -->"
	contextEndMarker   = "<!-- pal:context:end -->"
)

// Service handles context operations
type Service struct {
	db         *db.DB
	portSvc    *port.Service
	sessionSvc *session.Service
	escSvc     *escalation.Service
}

// NewService creates a new context service
func NewService(database *db.DB) *Service {
	return &Service{
		db:         database,
		portSvc:    port.NewService(database),
		sessionSvc: session.NewService(database),
		escSvc:     escalation.NewService(database),
	}
}

// GenerateContext generates current context as markdown
func (s *Service) GenerateContext() (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("> 마지막 업데이트: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 활성 세션
	sessions, _ := s.sessionSvc.List(true, 5)
	sb.WriteString("### 활성 세션\n")
	if len(sessions) == 0 {
		sb.WriteString("- 없음\n")
	} else {
		for _, sess := range sessions {
			title := "-"
			if sess.Title.Valid {
				title = sess.Title.String
			}
			portInfo := ""
			if sess.PortID.Valid {
				portInfo = fmt.Sprintf(" (포트: %s)", sess.PortID.String)
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s%s\n", sess.ID, title, portInfo))
		}
	}
	sb.WriteString("\n")

	// 포트 현황
	sb.WriteString("### 포트 현황\n")
	portSummary, _ := s.portSvc.Summary()
	if len(portSummary) == 0 {
		sb.WriteString("- 없음\n")
	} else {
		statusEmoji := map[string]string{
			"pending":  "⏳",
			"running":  "🔄",
			"complete": "✅",
			"failed":   "❌",
			"blocked":  "🚫",
		}
		for status, count := range portSummary {
			sb.WriteString(fmt.Sprintf("- %s %s: %d\n", statusEmoji[status], status, count))
		}
	}
	sb.WriteString("\n")

	// 진행 중인 포트
	runningPorts, _ := s.portSvc.List("running", 5)
	if len(runningPorts) > 0 {
		sb.WriteString("### 진행 중인 작업\n")
		for _, p := range runningPorts {
			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", p.ID, title))
		}
		sb.WriteString("\n")
	}

	// 에스컬레이션
	openCount, _ := s.escSvc.OpenCount()
	sb.WriteString("### 에스컬레이션\n")
	if openCount == 0 {
		sb.WriteString("- 없음\n")
	} else {
		sb.WriteString(fmt.Sprintf("- 🔴 **%d개** 미해결\n", openCount))
		escalations, _ := s.escSvc.List("open", 3)
		for _, e := range escalations {
			issue := e.Issue
			if len(issue) > 50 {
				issue = issue[:47] + "..."
			}
			sb.WriteString(fmt.Sprintf("  - #%d: %s\n", e.ID, issue))
		}
	}

	return sb.String(), nil
}

// InjectToFile injects context into CLAUDE.md file
// Creates the file with default template if it doesn't exist
func (s *Service) InjectToFile(filePath string) error {
	var content []byte
	var err error

	// 파일 읽기 시도
	content, err = os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 파일이 없으면 기본 템플릿으로 생성
			content = []byte(defaultClaudeMDTemplate)
		} else {
			return fmt.Errorf("파일 읽기 실패: %w", err)
		}
	}

	// 컨텍스트 생성
	ctx, err := s.GenerateContext()
	if err != nil {
		return fmt.Errorf("컨텍스트 생성 실패: %w", err)
	}

	// 마커가 없으면 추가
	contentStr := string(content)
	if !strings.Contains(contentStr, contextStartMarker) {
		contentStr += "\n\n" + contextStartMarker + "\n" + contextEndMarker + "\n"
		content = []byte(contentStr)
	}

	// 마커 찾기 및 교체
	lines := strings.Split(string(content), "\n")
	var result []string
	inContext := false

	for _, line := range lines {
		if strings.Contains(line, contextStartMarker) {
			result = append(result, line)
			result = append(result, ctx)
			inContext = true
			continue
		}
		if strings.Contains(line, contextEndMarker) {
			result = append(result, line)
			inContext = false
			continue
		}
		if !inContext {
			result = append(result, line)
		}
	}

	// 파일 쓰기
	return os.WriteFile(filePath, []byte(strings.Join(result, "\n")), 0644)
}

// 기본 CLAUDE.md 템플릿
const defaultClaudeMDTemplate = `# Project

## 개요

PAL로 관리되는 프로젝트입니다.

## 컨벤션

(프로젝트 컨벤션을 여기에 작성하세요)

## 현재 작업 컨텍스트

<!-- pal:context:start -->
<!-- pal:context:end -->
`

// GenerateForPort generates context for a specific port
func (s *Service) GenerateForPort(portID string) (string, error) {
	// 포트 정보
	p, err := s.portSvc.Get(portID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 포트 컨텍스트: %s\n\n", portID))

	// 포트 기본 정보
	sb.WriteString("## 포트 정보\n")
	if p.Title.Valid {
		sb.WriteString(fmt.Sprintf("- **제목**: %s\n", p.Title.String))
	}
	sb.WriteString(fmt.Sprintf("- **상태**: %s\n", p.Status))
	if p.FilePath.Valid {
		sb.WriteString(fmt.Sprintf("- **명세**: %s\n", p.FilePath.String))
	}
	sb.WriteString("\n")

	// 포트 명세 파일 읽기
	if p.FilePath.Valid {
		specContent, err := os.ReadFile(p.FilePath.String)
		if err == nil {
			sb.WriteString("## 포트 명세\n\n")
			sb.WriteString("```markdown\n")
			sb.WriteString(string(specContent))
			sb.WriteString("\n```\n\n")
		}
	}

	// 관련 세션
	if p.SessionID.Valid {
		sb.WriteString("## 연결된 세션\n")
		sb.WriteString(fmt.Sprintf("- %s\n", p.SessionID.String))
		sb.WriteString("\n")
	}

	// 실행 명령
	sb.WriteString("## 실행 명령\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("# 작업 시작\n"))
	sb.WriteString(fmt.Sprintf("pal lock acquire %s\n", portID))
	sb.WriteString(fmt.Sprintf("pal port status %s running\n\n", portID))
	sb.WriteString(fmt.Sprintf("# 작업 완료 후\n"))
	sb.WriteString(fmt.Sprintf("pal lock release %s\n", portID))
	sb.WriteString(fmt.Sprintf("pal port status %s complete\n", portID))
	sb.WriteString("```\n")

	return sb.String(), nil
}

// FindClaudeMD finds CLAUDE.md file in pal-initialized project
// It looks for .claude directory to identify project root
func FindClaudeMD(startDir string) string {
	return findClaudeMDWithDepth(startDir, 0, 5) // 최대 5단계까지만 상위 탐색
}

func findClaudeMDWithDepth(dir string, depth, maxDepth int) string {
	// .claude 디렉토리가 있으면 이곳이 프로젝트 루트
	claudeDir := filepath.Join(dir, ".claude")
	if _, err := os.Stat(claudeDir); err == nil {
		// 이 프로젝트의 CLAUDE.md 찾기
		candidates := []string{
			filepath.Join(dir, "CLAUDE.md"),
			filepath.Join(claudeDir, "CLAUDE.md"),
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		// .claude는 있지만 CLAUDE.md가 없으면 기본 경로 반환 (생성 가능)
		return filepath.Join(dir, "CLAUDE.md")
	}

	// 깊이 제한 확인
	if depth >= maxDepth {
		return ""
	}

	// 상위 디렉토리 검색
	parent := filepath.Dir(dir)
	if parent != dir {
		return findClaudeMDWithDepth(parent, depth+1, maxDepth)
	}

	return ""
}

// FindProjectRoot finds the project root directory (where .claude exists)
func FindProjectRoot(startDir string) string {
	return findProjectRootWithDepth(startDir, 0, 5)
}

func findProjectRootWithDepth(dir string, depth, maxDepth int) string {
	claudeDir := filepath.Join(dir, ".claude")
	if _, err := os.Stat(claudeDir); err == nil {
		return dir
	}

	if depth >= maxDepth {
		return ""
	}

	parent := filepath.Dir(dir)
	if parent != dir {
		return findProjectRootWithDepth(parent, depth+1, maxDepth)
	}

	return ""
}

// ReadPortSpec reads a port specification file
func ReadPortSpec(portID, portsDir string) (string, error) {
	// 여러 패턴 시도
	patterns := []string{
		filepath.Join(portsDir, portID+".md"),
		filepath.Join(portsDir, portID+"-*.md"),
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

// ParsePortDependencies extracts dependencies from port spec
func ParsePortDependencies(specContent string) []string {
	var deps []string

	scanner := bufio.NewScanner(strings.NewReader(specContent))
	inDepends := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(strings.ToLower(line), "depends:") ||
			strings.Contains(strings.ToLower(line), "선행 작업") {
			inDepends = true
			continue
		}

		if inDepends {
			if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
				// port-xxx 패턴 찾기
				if idx := strings.Index(line, "port-"); idx >= 0 {
					end := idx + 8 // port-xxx
					for i := idx + 5; i < len(line); i++ {
						if line[i] == ' ' || line[i] == ')' || line[i] == ':' {
							end = i
							break
						}
						end = i + 1
					}
					deps = append(deps, line[idx:end])
				}
			} else if line == "" || strings.HasPrefix(line, "#") {
				inDepends = false
			}
		}
	}

	return deps
}
