package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/escalation"
	"github.com/spf13/cobra"
)

var (
	escIssue     string
	escSessionID string
	escPortID    string
	escStatus    string
	escLimit     int
)

var escalationCmd = &cobra.Command{
	Use:     "escalation",
	Aliases: []string{"esc"},
	Short:   "에스컬레이션 관리",
	Long:    `상위 에스컬레이션을 관리합니다.`,
}

var escCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "에스컬레이션 생성",
	RunE:  runEscCreate,
}

var escListCmd = &cobra.Command{
	Use:   "list",
	Short: "에스컬레이션 목록",
	RunE:  runEscList,
}

var escShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "에스컬레이션 상세",
	Args:  cobra.ExactArgs(1),
	RunE:  runEscShow,
}

var escResolveCmd = &cobra.Command{
	Use:   "resolve <id>",
	Short: "에스컬레이션 해결",
	Args:  cobra.ExactArgs(1),
	RunE:  runEscResolve,
}

var escDismissCmd = &cobra.Command{
	Use:   "dismiss <id>",
	Short: "에스컬레이션 무시",
	Args:  cobra.ExactArgs(1),
	RunE:  runEscDismiss,
}

var escSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "에스컬레이션 요약",
	RunE:  runEscSummary,
}

func init() {
	rootCmd.AddCommand(escalationCmd)
	escalationCmd.AddCommand(escCreateCmd)
	escalationCmd.AddCommand(escListCmd)
	escalationCmd.AddCommand(escShowCmd)
	escalationCmd.AddCommand(escResolveCmd)
	escalationCmd.AddCommand(escDismissCmd)
	escalationCmd.AddCommand(escSummaryCmd)

	escCreateCmd.Flags().StringVar(&escIssue, "issue", "", "이슈 내용 (필수)")
	escCreateCmd.Flags().StringVar(&escSessionID, "session", "", "발생 세션")
	escCreateCmd.Flags().StringVar(&escPortID, "port", "", "발생 포트")
	escCreateCmd.MarkFlagRequired("issue")

	escListCmd.Flags().StringVar(&escStatus, "status", "", "상태 필터 (open|resolved|dismissed)")
	escListCmd.Flags().IntVar(&escLimit, "limit", 20, "결과 수 제한")
}

func getEscalationService() (*escalation.Service, func(), error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, err
	}
	return escalation.NewService(database), func() { database.Close() }, nil
}

func runEscCreate(cmd *cobra.Command, args []string) error {
	if escIssue == "" {
		return fmt.Errorf("--issue 플래그가 필요합니다")
	}

	// 환경변수에서 기본값
	sessionID := escSessionID
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	svc, cleanup, err := getEscalationService()
	if err != nil {
		return err
	}
	defer cleanup()

	id, err := svc.Create(escIssue, sessionID, escPortID)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "created",
			"id":      id,
			"issue":   escIssue,
			"session": sessionID,
			"port":    escPortID,
		})
	} else {
		fmt.Printf("🚨 에스컬레이션 생성: #%d\n", id)
		fmt.Printf("  이슈: %s\n", escIssue)
		if sessionID != "" {
			fmt.Printf("  세션: %s\n", sessionID)
		}
		if escPortID != "" {
			fmt.Printf("  포트: %s\n", escPortID)
		}
	}

	return nil
}

func runEscList(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getEscalationService()
	if err != nil {
		return err
	}
	defer cleanup()

	escalations, err := svc.List(escStatus, escLimit)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"escalations": escalations,
		})
		return nil
	}

	if len(escalations) == 0 {
		fmt.Println("에스컬레이션이 없습니다.")
		return nil
	}

	fmt.Printf("%-5s %-10s %-12s %-12s %s\n", "ID", "STATUS", "SESSION", "PORT", "ISSUE")
	fmt.Println(strings.Repeat("-", 80))
	for _, e := range escalations {
		session := "-"
		if e.FromSession.Valid {
			session = e.FromSession.String
		}
		port := "-"
		if e.FromPort.Valid {
			port = e.FromPort.String
		}
		issue := e.Issue
		if len(issue) > 35 {
			issue = issue[:32] + "..."
		}

		statusIcon := map[string]string{
			"open":      "🔴",
			"resolved":  "✅",
			"dismissed": "⚪",
		}

		fmt.Printf("%-5d %s %-8s %-12s %-12s %s\n",
			e.ID, statusIcon[e.Status], e.Status, session, port, issue)
	}

	return nil
}

func runEscShow(cmd *cobra.Command, args []string) error {
	var id int64
	fmt.Sscanf(args[0], "%d", &id)

	svc, cleanup, err := getEscalationService()
	if err != nil {
		return err
	}
	defer cleanup()

	e, err := svc.Get(id)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(e)
		return nil
	}

	fmt.Printf("에스컬레이션 #%d\n", e.ID)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("상태: %s\n", e.Status)
	fmt.Printf("이슈: %s\n", e.Issue)
	if e.FromSession.Valid {
		fmt.Printf("세션: %s\n", e.FromSession.String)
	}
	if e.FromPort.Valid {
		fmt.Printf("포트: %s\n", e.FromPort.String)
	}
	fmt.Printf("생성: %s\n", e.CreatedAt.Format("2006-01-02 15:04:05"))
	if e.ResolvedAt.Valid {
		fmt.Printf("해결: %s\n", e.ResolvedAt.Time.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runEscResolve(cmd *cobra.Command, args []string) error {
	var id int64
	fmt.Sscanf(args[0], "%d", &id)

	svc, cleanup, err := getEscalationService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.Resolve(id); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "resolved",
			"id":     id,
		})
	} else {
		fmt.Printf("✅ 에스컬레이션 해결: #%d\n", id)
	}

	return nil
}

func runEscDismiss(cmd *cobra.Command, args []string) error {
	var id int64
	fmt.Sscanf(args[0], "%d", &id)

	svc, cleanup, err := getEscalationService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.Dismiss(id); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "dismissed",
			"id":     id,
		})
	} else {
		fmt.Printf("⚪ 에스컬레이션 무시: #%d\n", id)
	}

	return nil
}

func runEscSummary(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getEscalationService()
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

	fmt.Printf("에스컬레이션 요약 (총 %d개)\n", total)
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("🔴 open:      %d\n", summary["open"])
	fmt.Printf("✅ resolved:  %d\n", summary["resolved"])
	fmt.Printf("⚪ dismissed: %d\n", summary["dismissed"])

	return nil
}
