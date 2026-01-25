package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0roo/pal-kit/internal/attention"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var attentionCmd = &cobra.Command{
	Use:     "attention",
	Aliases: []string{"att"},
	Short:   "Attention 상태 관리",
	Long:    `세션의 Attention 상태와 Compact 이력을 조회합니다.`,
}

var attShowCmd = &cobra.Command{
	Use:   "show [session-id]",
	Short: "Attention 상태 조회",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := attention.NewStore(database.DB)
		att, err := store.Get(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(att, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		status := attention.CalculateStatus(att)
		statusIcon := getAttentionStatusIcon(status)

		fmt.Printf("Session: %s\n", att.SessionID)
		fmt.Printf("Status: %s %s\n", statusIcon, status)
		fmt.Println()

		// Token usage bar
		usagePercent := att.GetTokenUsagePercent()
		bar := renderProgressBar(usagePercent, 30)
		fmt.Printf("Tokens: %s %d / %d (%.1f%%)\n", bar, att.LoadedTokens, att.AvailableTokens, usagePercent)
		fmt.Printf("Focus Score: %.2f\n", att.FocusScore)
		fmt.Printf("Drift Count: %d\n", att.DriftCount)

		if att.LastCompactionAt != nil {
			fmt.Printf("Last Compact: %s\n", att.LastCompactionAt.Format("2006-01-02 15:04:05"))
		}

		if len(att.LoadedFiles) > 0 {
			fmt.Println("\nLoaded Files:")
			for _, f := range att.LoadedFiles {
				fmt.Printf("  - %s\n", f)
			}
		}

		if len(att.LoadedConventions) > 0 {
			fmt.Println("\nLoaded Conventions:")
			for _, c := range att.LoadedConventions {
				fmt.Printf("  - %s\n", c)
			}
		}

		return nil
	},
}

var attReportCmd = &cobra.Command{
	Use:   "report [session-id]",
	Short: "Attention 리포트 생성",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := attention.NewStore(database.DB)
		report, err := store.GenerateReport(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		statusIcon := getAttentionStatusIcon(report.Status)
		fmt.Printf("Attention Report: %s\n", report.SessionID)
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Status: %s %s\n", statusIcon, report.Status)
		fmt.Printf("Token Usage: %s (%.1f%%)\n", report.TokenUsage, report.TokenPercent)
		fmt.Printf("Focus Score: %.2f\n", report.FocusScore)
		fmt.Printf("Drift Count: %d\n", report.DriftCount)
		fmt.Printf("Compact Count: %d\n", report.CompactCount)

		if len(report.Recommendations) > 0 {
			fmt.Println("\n⚠️ Recommendations:")
			for _, r := range report.Recommendations {
				fmt.Printf("  • %s\n", r)
			}
		}

		return nil
	},
}

var attHistoryCmd = &cobra.Command{
	Use:   "history [session-id]",
	Short: "Compact 이력 조회",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := attention.NewStore(database.DB)
		limit, _ := cmd.Flags().GetInt("limit")

		events, err := store.GetCompactHistory(args[0], limit)
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(events, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(events) == 0 {
			fmt.Println("Compact 이력이 없습니다.")
			return nil
		}

		fmt.Printf("Compact History for %s\n", args[0])
		fmt.Println(strings.Repeat("-", 70))
		fmt.Printf("%-20s %-15s %-10s %-10s %s\n", "Time", "Reason", "Before", "After", "Preserved")
		fmt.Println(strings.Repeat("-", 70))

		for _, e := range events {
			preserved := len(e.PreservedContext)
			fmt.Printf("%-20s %-15s %-10d %-10d %d items\n",
				e.CreatedAt.Format("01-02 15:04:05"),
				e.TriggerReason,
				e.BeforeTokens,
				e.AfterTokens,
				preserved)
		}

		return nil
	},
}

var attInitCmd = &cobra.Command{
	Use:   "init [session-id]",
	Short: "Attention 추적 초기화",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := attention.NewStore(database.DB)
		portID, _ := cmd.Flags().GetString("port")
		budget, _ := cmd.Flags().GetInt("budget")

		if err := store.Initialize(args[0], portID, budget); err != nil {
			return err
		}

		fmt.Printf("✓ Attention 추적 초기화됨: %s (budget: %d)\n", args[0], budget)
		return nil
	},
}

func getAttentionStatusIcon(status attention.AttentionStatus) string {
	switch status {
	case attention.StatusFocused:
		return "🟢"
	case attention.StatusDrifting:
		return "🟡"
	case attention.StatusWarning:
		return "🟠"
	case attention.StatusCritical:
		return "🔴"
	default:
		return "⚪"
	}
}

func renderProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	// Color based on percentage
	if percent >= 95 {
		return fmt.Sprintf("\033[31m[%s]\033[0m", bar) // Red
	} else if percent >= 80 {
		return fmt.Sprintf("\033[33m[%s]\033[0m", bar) // Yellow
	}
	return fmt.Sprintf("\033[32m[%s]\033[0m", bar) // Green
}

func init() {
	rootCmd.AddCommand(attentionCmd)

	attentionCmd.AddCommand(attShowCmd)
	attentionCmd.AddCommand(attReportCmd)

	attentionCmd.AddCommand(attHistoryCmd)
	attHistoryCmd.Flags().IntP("limit", "l", 10, "최대 개수")

	attentionCmd.AddCommand(attInitCmd)
	attInitCmd.Flags().StringP("port", "p", "", "포트 ID")
	attInitCmd.Flags().IntP("budget", "b", 15000, "토큰 예산")
}
