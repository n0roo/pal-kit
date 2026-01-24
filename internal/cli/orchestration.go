package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/orchestrator"
	"github.com/spf13/cobra"
)

var orchestrationCmd = &cobra.Command{
	Use:     "orchestration",
	Aliases: []string{"orch", "o"},
	Short:   "Orchestration 관리",
	Long:    `Orchestration 포트를 생성하고 관리합니다.`,
}

var orchListCmd = &cobra.Command{
	Use:   "list",
	Short: "Orchestration 목록",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		svc := orchestrator.NewService(database, nil, nil)

		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")

		orchestrations, err := svc.ListOrchestrations(orchestrator.OrchestrationStatus(status), limit)
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(orchestrations, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(orchestrations) == 0 {
			fmt.Println("Orchestration이 없습니다.")
			return nil
		}

		fmt.Printf("%-36s %-30s %-10s %5s %s\n", "ID", "Title", "Status", "Prog", "Ports")
		fmt.Println(strings.Repeat("-", 100))
		for _, o := range orchestrations {
			fmt.Printf("%-36s %-30s %-10s %4d%% %d\n",
				truncate(o.ID, 36),
				truncate(o.Title, 30),
				o.Status,
				o.ProgressPercent,
				len(o.AtomicPorts))
		}

		return nil
	},
}

var orchCreateCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Orchestration 생성",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		svc := orchestrator.NewService(database, nil, nil)

		title := strings.Join(args, " ")
		desc, _ := cmd.Flags().GetString("description")
		portsStr, _ := cmd.Flags().GetString("ports")

		// Parse ports (comma-separated)
		var atomicPorts []orchestrator.AtomicPort
		if portsStr != "" {
			portIDs := strings.Split(portsStr, ",")
			for i, pid := range portIDs {
				atomicPorts = append(atomicPorts, orchestrator.AtomicPort{
					PortID: strings.TrimSpace(pid),
					Order:  i + 1,
				})
			}
		}

		orch, err := svc.CreateOrchestration(title, desc, atomicPorts)
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(orch, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("✓ Orchestration 생성됨: %s\n", orch.ID)
		fmt.Printf("  Title: %s\n", orch.Title)
		fmt.Printf("  Ports: %d\n", len(orch.AtomicPorts))
		return nil
	},
}

var orchShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Orchestration 상세 조회",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		svc := orchestrator.NewService(database, nil, nil)
		orch, err := svc.GetOrchestration(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(orch, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID: %s\n", orch.ID)
		fmt.Printf("Title: %s\n", orch.Title)
		if orch.Description != "" {
			fmt.Printf("Description: %s\n", orch.Description)
		}
		fmt.Printf("Status: %s\n", orch.Status)
		fmt.Printf("Progress: %d%%\n", orch.ProgressPercent)
		fmt.Printf("Created: %s\n", orch.CreatedAt.Format("2006-01-02 15:04:05"))

		if len(orch.AtomicPorts) > 0 {
			fmt.Println("\nPorts:")
			for _, p := range orch.AtomicPorts {
				status := p.Status
				if status == "" {
					status = "pending"
				}
				deps := ""
				if len(p.DependsOn) > 0 {
					deps = fmt.Sprintf(" (depends: %s)", strings.Join(p.DependsOn, ", "))
				}
				fmt.Printf("  %d. %s [%s]%s\n", p.Order, p.PortID, status, deps)
			}
		}

		return nil
	},
}

var orchStatsCmd = &cobra.Command{
	Use:   "stats [id]",
	Short: "Orchestration 통계",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		svc := orchestrator.NewService(database, nil, nil)
		stats, err := svc.GetOrchestrationStats(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Total Ports: %d\n", stats.TotalPorts)
		fmt.Printf("  Pending:   %d\n", stats.PendingPorts)
		fmt.Printf("  Running:   %d\n", stats.RunningPorts)
		fmt.Printf("  Completed: %d\n", stats.CompletedPorts)
		fmt.Printf("  Failed:    %d\n", stats.FailedPorts)
		fmt.Printf("Progress: %d%%\n", stats.ProgressPercent)
		fmt.Printf("\nWorkers: %d total, %d active\n", stats.TotalWorkers, stats.ActiveWorkers)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(orchestrationCmd)

	orchestrationCmd.AddCommand(orchListCmd)
	orchListCmd.Flags().StringP("status", "s", "", "상태 필터 (pending, running, complete, failed)")
	orchListCmd.Flags().IntP("limit", "l", 20, "최대 개수")

	orchestrationCmd.AddCommand(orchCreateCmd)
	orchCreateCmd.Flags().StringP("description", "d", "", "설명")
	orchCreateCmd.Flags().StringP("ports", "p", "", "포트 ID 목록 (쉼표 구분)")

	orchestrationCmd.AddCommand(orchShowCmd)
	orchestrationCmd.AddCommand(orchStatsCmd)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
