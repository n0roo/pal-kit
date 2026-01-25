package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/session"
	"github.com/spf13/cobra"
)

var hierarchyCmd = &cobra.Command{
	Use:     "hierarchy",
	Aliases: []string{"hier", "tree"},
	Short:   "세션 계층 조회",
	Long:    `세션의 계층 구조를 트리 형태로 조회합니다.`,
}

var hierShowCmd = &cobra.Command{
	Use:   "show [root-session-id]",
	Short: "세션 계층 트리 조회",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		svc := session.NewService(database)
		tree, err := svc.GetSessionHierarchy(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(tree, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		printSessionTree(tree, "", true)
		return nil
	},
}

var hierListCmd = &cobra.Command{
	Use:   "list [root-session-id]",
	Short: "세션 계층 목록 조회",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		svc := session.NewService(database)
		sessions, err := svc.ListByRoot(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(sessions, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(sessions) == 0 {
			fmt.Println("세션이 없습니다.")
			return nil
		}

		fmt.Printf("%-36s %-8s %-10s %-10s %s\n", "ID", "Type", "Status", "Attention", "Title")
		fmt.Println(strings.Repeat("-", 100))
		for _, s := range sessions {
			indent := strings.Repeat("  ", s.Depth)
			attention := "-"
			if s.AttentionScore.Valid {
				attention = fmt.Sprintf("%.2f", s.AttentionScore.Float64)
			}
			title := ""
			if s.Title.Valid {
				title = s.Title.String
			}
			fmt.Printf("%s%-*s %-8s %-10s %-10s %s\n",
				indent,
				36-len(indent), truncate(s.ID, 36-len(indent)),
				s.Type,
				s.Status,
				attention,
				truncate(title, 30))
		}

		return nil
	},
}

var hierStatsCmd = &cobra.Command{
	Use:   "stats [root-session-id]",
	Short: "세션 계층 통계",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		svc := session.NewService(database)
		stats, err := svc.GetHierarchyStats(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Session Hierarchy Stats\n")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Total Sessions:  %d\n", stats.TotalSessions)
		fmt.Printf("  Operators:     %d\n", stats.OperatorCount)
		fmt.Printf("  Workers:       %d\n", stats.WorkerCount)
		fmt.Printf("  Tests:         %d\n", stats.TestCount)
		fmt.Println()
		fmt.Printf("Running:         %d\n", stats.RunningCount)
		fmt.Printf("Complete:        %d\n", stats.CompleteCount)
		fmt.Printf("Failed:          %d\n", stats.FailedCount)
		fmt.Println()
		fmt.Printf("Avg Attention:   %.2f\n", stats.AvgAttention)
		fmt.Printf("Total Tokens:    %d\n", stats.TotalTokens)
		fmt.Printf("Total Compacts:  %d\n", stats.TotalCompacts)
		fmt.Printf("Progress:        %.1f%%\n", stats.ProgressPercent)

		return nil
	},
}

var hierBuildsCmd = &cobra.Command{
	Use:   "builds",
	Short: "Build 세션 목록",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		svc := session.NewService(database)
		active, _ := cmd.Flags().GetBool("active")
		limit, _ := cmd.Flags().GetInt("limit")

		sessions, err := svc.GetBuildSessions(active, limit)
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(sessions, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(sessions) == 0 {
			fmt.Println("Build 세션이 없습니다.")
			return nil
		}

		fmt.Printf("%-36s %-10s %-10s %s\n", "ID", "Status", "Attention", "Title")
		fmt.Println(strings.Repeat("-", 80))
		for _, s := range sessions {
			attention := "-"
			if s.AttentionScore.Valid {
				attention = fmt.Sprintf("%.2f", s.AttentionScore.Float64)
			}
			title := ""
			if s.Title.Valid {
				title = s.Title.String
			}
			fmt.Printf("%-36s %-10s %-10s %s\n",
				truncate(s.ID, 36),
				s.Status,
				attention,
				truncate(title, 30))
		}

		return nil
	},
}

func printSessionTree(node *session.SessionHierarchyNode, prefix string, isLast bool) {
	// Determine the connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	// Print current node
	typeIcon := getTypeIcon(node.Session.Type)
	statusIcon := getStatusIcon(node.Session.Status)
	title := ""
	if node.Session.Title.Valid {
		title = node.Session.Title.String
	}
	attention := ""
	if node.Session.AttentionScore.Valid {
		attention = fmt.Sprintf(" [%.2f]", node.Session.AttentionScore.Float64)
	}

	if prefix == "" {
		// Root node
		fmt.Printf("%s %s %s%s\n", typeIcon, statusIcon, title, attention)
	} else {
		fmt.Printf("%s%s%s %s %s%s\n", prefix, connector, typeIcon, statusIcon, title, attention)
	}

	// Calculate new prefix for children
	var newPrefix string
	if prefix == "" {
		newPrefix = ""
	} else if isLast {
		newPrefix = prefix + "    "
	} else {
		newPrefix = prefix + "│   "
	}

	// Print children
	for i, child := range node.Children {
		printSessionTree(child, newPrefix, i == len(node.Children)-1)
	}
}

func getTypeIcon(t string) string {
	switch t {
	case session.TypeBuild:
		return "📋"
	case session.TypeOperator:
		return "🎯"
	case session.TypeWorker:
		return "⚙️"
	case session.TypeTest:
		return "🧪"
	default:
		return "📄"
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "running":
		return "●"
	case "complete":
		return "✓"
	case "failed":
		return "✗"
	case "paused":
		return "⏸"
	default:
		return "○"
	}
}

func init() {
	rootCmd.AddCommand(hierarchyCmd)

	hierarchyCmd.AddCommand(hierShowCmd)
	hierarchyCmd.AddCommand(hierListCmd)
	hierarchyCmd.AddCommand(hierStatsCmd)

	hierarchyCmd.AddCommand(hierBuildsCmd)
	hierBuildsCmd.Flags().BoolP("active", "a", false, "활성 세션만")
	hierBuildsCmd.Flags().IntP("limit", "l", 10, "최대 개수")
}
