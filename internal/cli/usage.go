package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/usage"
	"github.com/spf13/cobra"
)

var (
	usageSessionID string
	usageToday     bool
	usageSince     string
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "토큰 사용량",
	Long:  `Claude 토큰 사용량을 추적합니다.`,
}

var usageSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "JSONL에서 사용량 동기화",
	RunE:  runUsageSync,
}

var usageSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "사용량 요약",
	RunE:  runUsageSummary,
}

var usageShowCmd = &cobra.Command{
	Use:   "show <session-id>",
	Short: "세션 사용량 상세",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsageShow,
}

func init() {
	rootCmd.AddCommand(usageCmd)
	usageCmd.AddCommand(usageSyncCmd)
	usageCmd.AddCommand(usageSummaryCmd)
	usageCmd.AddCommand(usageShowCmd)

	usageSyncCmd.Flags().StringVar(&usageSessionID, "session", "", "특정 세션만 동기화")

	usageSummaryCmd.Flags().BoolVar(&usageToday, "today", false, "오늘만")
	usageSummaryCmd.Flags().StringVar(&usageSince, "since", "", "시작 날짜 (YYYY-MM-DD)")
	usageSummaryCmd.Flags().StringVar(&usageSessionID, "session", "", "특정 세션")
}

func getUsageService() (*usage.Service, func(), error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, err
	}
	return usage.NewService(database), func() { database.Close() }, nil
}

func runUsageSync(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getUsageService()
	if err != nil {
		return err
	}
	defer cleanup()

	projectsDir := usage.GetProjectsDir()
	if projectsDir == "" {
		return fmt.Errorf("Claude projects 디렉토리를 찾을 수 없습니다")
	}

	synced, err := svc.SyncFromJSONL(projectsDir)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":       "synced",
			"sessions":     synced,
			"projects_dir": projectsDir,
		})
	} else {
		fmt.Printf("✓ %d개 세션 동기화 완료\n", synced)
		fmt.Printf("  소스: %s\n", projectsDir)
	}

	return nil
}

func runUsageSummary(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getUsageService()
	if err != nil {
		return err
	}
	defer cleanup()

	// 특정 세션 요청
	if usageSessionID != "" {
		return showSessionUsage(svc, usageSessionID)
	}

	// 날짜 필터 계산
	var since time.Time
	if usageToday {
		now := time.Now()
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else if usageSince != "" {
		parsed, err := time.Parse("2006-01-02", usageSince)
		if err != nil {
			return fmt.Errorf("날짜 형식 오류 (YYYY-MM-DD): %w", err)
		}
		since = parsed
	}

	summary, err := svc.GetSummary(since)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(summary)
		return nil
	}

	// 기간 표시
	periodLabel := "전체"
	if usageToday {
		periodLabel = "오늘"
	} else if usageSince != "" {
		periodLabel = fmt.Sprintf("%s 이후", usageSince)
	}

	fmt.Printf("사용량 요약 (%s)\n", periodLabel)
	fmt.Println(strings.Repeat("=", 40))
	fmt.Println()
	fmt.Printf("세션 수: %d\n", summary.MessageCount)
	fmt.Println()
	fmt.Println("토큰:")
	fmt.Printf("  입력:       %s\n", formatNumber(summary.InputTokens))
	fmt.Printf("  출력:       %s\n", formatNumber(summary.OutputTokens))
	fmt.Printf("  캐시 읽기:  %s\n", formatNumber(summary.CacheReadTokens))
	fmt.Printf("  캐시 생성:  %s\n", formatNumber(summary.CacheCreateTokens))
	fmt.Println()
	fmt.Printf("총 비용: $%.4f\n", summary.CostUSD)

	return nil
}

func showSessionUsage(svc *usage.Service, sessionID string) error {
	summary, err := svc.GetSessionUsage(sessionID)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(summary)
		return nil
	}

	fmt.Printf("세션 사용량: %s\n", sessionID)
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("입력 토큰:      %s\n", formatNumber(summary.InputTokens))
	fmt.Printf("출력 토큰:      %s\n", formatNumber(summary.OutputTokens))
	fmt.Printf("캐시 읽기:      %s\n", formatNumber(summary.CacheReadTokens))
	fmt.Printf("캐시 생성:      %s\n", formatNumber(summary.CacheCreateTokens))
	fmt.Printf("비용:           $%.4f\n", summary.CostUSD)

	return nil
}

func runUsageShow(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	svc, cleanup, err := getUsageService()
	if err != nil {
		return err
	}
	defer cleanup()

	return showSessionUsage(svc, sessionID)
}

// formatNumber formats a number with comma separators
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	s := fmt.Sprintf("%d", n)
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
