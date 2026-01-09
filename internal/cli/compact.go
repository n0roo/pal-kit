package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/compact"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var (
	compactSessionID   string
	compactSummary     string
	compactTriggerType string
	compactTokens      int64
	compactLimit       int
)

var compactCmd = &cobra.Command{
	Use:   "compact",
	Short: "컴팩션 기록",
	Long:  `컨텍스트 컴팩션 이벤트를 기록합니다.`,
}

var compactRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "컴팩션 기록",
	RunE:  runCompactRecord,
}

var compactListCmd = &cobra.Command{
	Use:   "list",
	Short: "컴팩션 히스토리",
	RunE:  runCompactList,
}

var compactSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "컴팩션 요약",
	RunE:  runCompactSummary,
}

func init() {
	rootCmd.AddCommand(compactCmd)
	compactCmd.AddCommand(compactRecordCmd)
	compactCmd.AddCommand(compactListCmd)
	compactCmd.AddCommand(compactSummaryCmd)

	compactRecordCmd.Flags().StringVar(&compactSessionID, "session", "", "세션 ID")
	compactRecordCmd.Flags().StringVar(&compactSummary, "summary", "", "컨텍스트 요약")
	compactRecordCmd.Flags().StringVar(&compactTriggerType, "trigger", "auto", "트리거 타입 (auto|manual)")
	compactRecordCmd.Flags().Int64Var(&compactTokens, "tokens", 0, "압축 전 토큰 수")

	compactListCmd.Flags().StringVar(&compactSessionID, "session", "", "세션 ID 필터")
	compactListCmd.Flags().IntVar(&compactLimit, "limit", 20, "결과 수 제한")
}

func getCompactService() (*compact.Service, func(), error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, err
	}
	return compact.NewService(database), func() { database.Close() }, nil
}

func runCompactRecord(cmd *cobra.Command, args []string) error {
	sessionID := compactSessionID
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}
	if sessionID == "" {
		return fmt.Errorf("세션 ID가 필요합니다 (--session 또는 CLAUDE_SESSION_ID)")
	}

	svc, cleanup, err := getCompactService()
	if err != nil {
		return err
	}
	defer cleanup()

	id, err := svc.Record(sessionID, compactTriggerType, compactSummary, compactTokens)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":     "recorded",
			"id":         id,
			"session_id": sessionID,
			"trigger":    compactTriggerType,
		})
	} else {
		fmt.Printf("📦 컴팩션 기록: #%d\n", id)
		fmt.Printf("  세션: %s\n", sessionID)
		fmt.Printf("  트리거: %s\n", compactTriggerType)
		if compactSummary != "" {
			fmt.Printf("  요약: %s\n", compactSummary)
		}
	}

	return nil
}

func runCompactList(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getCompactService()
	if err != nil {
		return err
	}
	defer cleanup()

	compactions, err := svc.List(compactSessionID, compactLimit)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"compactions": compactions,
		})
		return nil
	}

	if len(compactions) == 0 {
		fmt.Println("컴팩션 기록이 없습니다.")
		return nil
	}

	fmt.Printf("%-6s %-12s %-8s %-20s %s\n", "ID", "SESSION", "TRIGGER", "TIME", "SUMMARY")
	fmt.Println(strings.Repeat("-", 80))
	for _, c := range compactions {
		summary := c.ContextSummary
		if len(summary) > 30 {
			summary = summary[:27] + "..."
		}
		if summary == "" {
			summary = "-"
		}
		fmt.Printf("%-6d %-12s %-8s %-20s %s\n",
			c.ID, c.SessionID, c.TriggerType,
			c.TriggeredAt.Format("2006-01-02 15:04"),
			summary)
	}

	return nil
}

func runCompactSummary(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getCompactService()
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

	fmt.Println("컴팩션 요약")
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("총 컴팩션:  %d\n", summary["total"])
	fmt.Printf("  자동:     %d\n", summary["auto"])
	fmt.Printf("  수동:     %d\n", summary["manual"])

	return nil
}
