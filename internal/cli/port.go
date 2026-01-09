package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/rules"
	"github.com/spf13/cobra"
)

var (
	portTitle    string
	portFile     string
	portStatus   string
	portLimit    int
	portPatterns []string
)

var portCmd = &cobra.Command{
	Use:   "port",
	Short: "포트 관리",
	Long:  `작업 단위(포트)를 관리합니다.`,
}

var portCreateCmd = &cobra.Command{
	Use:   "create <id>",
	Short: "포트 생성",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortCreate,
}

var portStatusCmd = &cobra.Command{
	Use:   "status <id> <status>",
	Short: "포트 상태 업데이트",
	Long: `포트 상태를 업데이트합니다.

상태 값:
  pending   - 대기 중
  running   - 진행 중
  complete  - 완료
  failed    - 실패
  blocked   - 차단됨`,
	Args: cobra.ExactArgs(2),
	RunE: runPortStatus,
}

var portListCmd = &cobra.Command{
	Use:   "list",
	Short: "포트 목록",
	RunE:  runPortList,
}

var portShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "포트 상세",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortShow,
}

var portDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "포트 삭제",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortDelete,
}

var portSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "포트 요약",
	RunE:  runPortSummary,
}

var portActivateCmd = &cobra.Command{
	Use:   "activate <id>",
	Short: "포트 활성화 (rules 파일 생성)",
	Long: `포트를 활성화하고 .claude/rules/에 조건부 규칙 파일을 생성합니다.

Claude Code가 해당 포트 관련 파일 작업 시 자동으로 규칙을 로드합니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPortActivate,
}

var portDeactivateCmd = &cobra.Command{
	Use:   "deactivate <id>",
	Short: "포트 비활성화 (rules 파일 삭제)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPortDeactivate,
}

var portRulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "활성 규칙 목록",
	RunE:  runPortRules,
}

func init() {
	rootCmd.AddCommand(portCmd)
	portCmd.AddCommand(portCreateCmd)
	portCmd.AddCommand(portStatusCmd)
	portCmd.AddCommand(portListCmd)
	portCmd.AddCommand(portShowCmd)
	portCmd.AddCommand(portDeleteCmd)
	portCmd.AddCommand(portSummaryCmd)
	portCmd.AddCommand(portActivateCmd)
	portCmd.AddCommand(portDeactivateCmd)
	portCmd.AddCommand(portRulesCmd)

	portCreateCmd.Flags().StringVar(&portTitle, "title", "", "포트 제목")
	portCreateCmd.Flags().StringVar(&portFile, "file", "", "포트 문서 경로")

	portListCmd.Flags().StringVar(&portStatus, "status", "", "상태 필터 (pending|running|complete|failed|blocked)")
	portListCmd.Flags().IntVar(&portLimit, "limit", 20, "결과 수 제한")

	portActivateCmd.Flags().StringArrayVar(&portPatterns, "path", nil, "적용할 파일 패턴 (여러 개 가능)")
}

func getPortService() (*port.Service, func(), error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, err
	}
	return port.NewService(database), func() { database.Close() }, nil
}

func runPortCreate(cmd *cobra.Command, args []string) error {
	portID := args[0]

	svc, cleanup, err := getPortService()
	if err != nil {
		return err
	}
	defer cleanup()

	// 파일 경로 자동 생성 (지정 안 된 경우)
	filePath := portFile
	if filePath == "" {
		filePath = fmt.Sprintf("ports/%s.md", portID)
	}

	if err := svc.Create(portID, portTitle, filePath); err != nil {
		return err
	}

	// 포트 문서 파일 생성 (디렉토리 확인)
	if portFile == "" {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err == nil {
			createPortDocument(filePath, portID, portTitle)
		}
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":    "created",
			"id":        portID,
			"title":     portTitle,
			"file_path": filePath,
		})
	} else {
		fmt.Printf("✓ 포트 생성: %s\n", portID)
		if portTitle != "" {
			fmt.Printf("  제목: %s\n", portTitle)
		}
		fmt.Printf("  문서: %s\n", filePath)
	}

	return nil
}

func createPortDocument(path, id, title string) error {
	if title == "" {
		title = id
	}

	content := fmt.Sprintf(`# %s

## 컨텍스트

- 상위 요구사항: 
- 작업 목적: 

## 입력

- 선행 작업 산출물: 
- 참조할 기존 코드: 

## 작업 범위 (배타적 소유권)

### 생성/수정할 파일
- 

### 구현할 기능
- 

## 컨벤션

### 적용할 규칙
- 

### 코드 패턴 예시
` + "```" + `
// 예시 코드
` + "```" + `

## 검증

### 컴파일/테스트 명령
` + "```bash" + `
# 빌드 확인
# 테스트 실행
` + "```" + `

### 완료 체크리스트
- [ ] 컴파일 성공
- [ ] 테스트 통과
- [ ] 컨벤션 준수

## 출력

### 완료 조건
- 

### 후속 작업에 전달할 정보
- 
`, title)

	return os.WriteFile(path, []byte(content), 0644)
}

func runPortStatus(cmd *cobra.Command, args []string) error {
	portID := args[0]
	newStatus := args[1]

	svc, cleanup, err := getPortService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.UpdateStatus(portID, newStatus); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":     "updated",
			"id":         portID,
			"new_status": newStatus,
		})
	} else {
		statusEmoji := map[string]string{
			"pending":  "⏳",
			"running":  "🔄",
			"complete": "✅",
			"failed":   "❌",
			"blocked":  "🚫",
		}
		emoji := statusEmoji[newStatus]
		fmt.Printf("%s 포트 상태 변경: %s → %s\n", emoji, portID, newStatus)
	}

	return nil
}

func runPortList(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getPortService()
	if err != nil {
		return err
	}
	defer cleanup()

	ports, err := svc.List(portStatus, portLimit)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"ports": ports,
		})
		return nil
	}

	if len(ports) == 0 {
		fmt.Println("포트가 없습니다.")
		return nil
	}

	fmt.Printf("%-12s %-25s %-10s %-12s %s\n", "ID", "TITLE", "STATUS", "SESSION", "CREATED")
	fmt.Println(strings.Repeat("-", 80))
	for _, p := range ports {
		title := "-"
		if p.Title.Valid {
			title = p.Title.String
			if len(title) > 25 {
				title = title[:22] + "..."
			}
		}
		sessionID := "-"
		if p.SessionID.Valid {
			sessionID = p.SessionID.String
		}
		fmt.Printf("%-12s %-25s %-10s %-12s %s\n",
			p.ID, title, p.Status, sessionID, p.CreatedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

func runPortShow(cmd *cobra.Command, args []string) error {
	portID := args[0]

	svc, cleanup, err := getPortService()
	if err != nil {
		return err
	}
	defer cleanup()

	p, err := svc.Get(portID)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(p)
		return nil
	}

	fmt.Printf("포트: %s\n", p.ID)
	fmt.Println(strings.Repeat("-", 40))
	if p.Title.Valid {
		fmt.Printf("제목: %s\n", p.Title.String)
	}
	fmt.Printf("상태: %s\n", p.Status)
	if p.SessionID.Valid {
		fmt.Printf("세션: %s\n", p.SessionID.String)
	}
	if p.FilePath.Valid {
		fmt.Printf("문서: %s\n", p.FilePath.String)
	}
	fmt.Printf("생성: %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))
	if p.StartedAt.Valid {
		fmt.Printf("시작: %s\n", p.StartedAt.Time.Format("2006-01-02 15:04:05"))
	}
	if p.CompletedAt.Valid {
		fmt.Printf("완료: %s\n", p.CompletedAt.Time.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runPortDelete(cmd *cobra.Command, args []string) error {
	portID := args[0]

	svc, cleanup, err := getPortService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.Delete(portID); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "deleted",
			"id":     portID,
		})
	} else {
		fmt.Printf("✓ 포트 삭제: %s\n", portID)
	}

	return nil
}

func runPortSummary(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getPortService()
	if err != nil {
		return err
	}
	defer cleanup()

	summary, err := svc.Summary()
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(summary)
		return nil
	}

	total := 0
	for _, count := range summary {
		total += count
	}

	fmt.Printf("포트 요약 (총 %d개)\n", total)
	fmt.Println(strings.Repeat("-", 30))

	statusOrder := []string{"pending", "running", "complete", "failed", "blocked"}
	statusEmoji := map[string]string{
		"pending":  "⏳",
		"running":  "🔄",
		"complete": "✅",
		"failed":   "❌",
		"blocked":  "🚫",
	}

	for _, s := range statusOrder {
		count := summary[s]
		if count > 0 {
			fmt.Printf("%s %-10s: %d\n", statusEmoji[s], s, count)
		}
	}

	return nil
}

func runPortActivate(cmd *cobra.Command, args []string) error {
	portID := args[0]

	// 포트 정보 조회
	svc, cleanup, err := getPortService()
	if err != nil {
		return err
	}
	defer cleanup()

	p, err := svc.Get(portID)
	if err != nil {
		return err
	}

	// 프로젝트 루트 찾기
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다 (pal init 실행 필요)")
	}

	// rules 서비스 생성
	rulesSvc := rules.NewService(projectRoot)

	// 포트 명세 경로
	specPath := ""
	if p.FilePath.Valid {
		specPath = p.FilePath.String
	}

	// 제목
	title := portID
	if p.Title.Valid {
		title = p.Title.String
	}

	// 파일 패턴
	patterns := portPatterns
	if len(patterns) == 0 && specPath != "" {
		patterns = []string{specPath}
	}

	// 규칙 파일 생성 (포트 명세 포함)
	if err := rulesSvc.ActivatePortWithSpec(portID, title, specPath, patterns); err != nil {
		return err
	}

	// 포트 상태를 running으로 변경
	if p.Status == "pending" {
		svc.UpdateStatus(portID, "running")
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":    "activated",
			"id":        portID,
			"rule_file": rulesSvc.GetRulePath(portID),
			"patterns":  patterns,
		})
	} else {
		fmt.Printf("✅ 포트 활성화: %s\n", portID)
		fmt.Printf("  규칙 파일: %s\n", rulesSvc.GetRulePath(portID))
		if len(patterns) > 0 {
			fmt.Printf("  적용 패턴: %v\n", patterns)
		}
	}

	return nil
}

func runPortDeactivate(cmd *cobra.Command, args []string) error {
	portID := args[0]

	// 프로젝트 루트 찾기
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	// rules 서비스 생성
	rulesSvc := rules.NewService(projectRoot)

	// 규칙 파일 삭제
	if err := rulesSvc.DeactivatePort(portID); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "deactivated",
			"id":     portID,
		})
	} else {
		fmt.Printf("⚪ 포트 비활성화: %s\n", portID)
	}

	return nil
}

func runPortRules(cmd *cobra.Command, args []string) error {
	// 프로젝트 루트 찾기
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	rulesSvc := rules.NewService(projectRoot)
	rulesList, err := rulesSvc.ListActiveRules()
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"rules": rulesList,
		})
		return nil
	}

	if len(rulesList) == 0 {
		fmt.Println("활성 규칙이 없습니다.")
		return nil
	}

	fmt.Printf("활성 규칙 (%d개)\n", len(rulesList))
	fmt.Println(strings.Repeat("-", 30))
	for _, rule := range rulesList {
		fmt.Printf("📝 %s\n", rule)
	}

	return nil
}
