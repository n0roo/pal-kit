package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/handoff"
	"github.com/spf13/cobra"
)

var handoffCmd = &cobra.Command{
	Use:     "handoff",
	Aliases: []string{"ho"},
	Short:   "Handoff 관리",
	Long:    `포트 간 컨텍스트 전달(Handoff)을 관리합니다.`,
}

var hoListCmd = &cobra.Command{
	Use:   "list [port-id]",
	Short: "포트의 Handoff 목록",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := handoff.NewStore(database)
		direction, _ := cmd.Flags().GetString("direction")

		var handoffs []*handoff.Handoff
		switch direction {
		case "from":
			handoffs, err = store.GetFromPort(args[0])
		case "to":
			handoffs, err = store.GetForPort(args[0])
		default:
			// Get both
			from, _ := store.GetFromPort(args[0])
			to, _ := store.GetForPort(args[0])
			handoffs = append(from, to...)
		}

		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(handoffs, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(handoffs) == 0 {
			fmt.Println("Handoff가 없습니다.")
			return nil
		}

		fmt.Printf("%-36s %-15s %-20s %-20s %s\n", "ID", "Type", "From", "To", "Tokens")
		fmt.Println(strings.Repeat("-", 110))
		for _, h := range handoffs {
			fmt.Printf("%-36s %-15s %-20s %-20s %d/%d\n",
				truncate(h.ID, 36),
				h.Type,
				truncate(h.FromPortID, 20),
				truncate(h.ToPortID, 20),
				h.TokenCount,
				h.MaxTokenBudget)
		}

		return nil
	},
}

var hoShowCmd = &cobra.Command{
	Use:   "show [handoff-id]",
	Short: "Handoff 상세 조회",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := handoff.NewStore(database)
		h, err := store.Get(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(h, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println(handoff.Summarize(h))
		return nil
	},
}

var hoCreateCmd = &cobra.Command{
	Use:   "create [from-port] [to-port]",
	Short: "Handoff 생성",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := handoff.NewStore(database)

		hoType, _ := cmd.Flags().GetString("type")
		contentStr, _ := cmd.Flags().GetString("content")

		// Parse content as JSON
		var content interface{}
		if contentStr != "" {
			if err := json.Unmarshal([]byte(contentStr), &content); err != nil {
				content = contentStr // Use as plain string
			}
		}

		h, err := store.Create(args[0], args[1], handoff.HandoffType(hoType), content)
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(h, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("✓ Handoff 생성됨: %s\n", h.ID)
		fmt.Printf("  Type: %s\n", h.Type)
		fmt.Printf("  From: %s → To: %s\n", h.FromPortID, h.ToPortID)
		fmt.Printf("  Tokens: %d/%d\n", h.TokenCount, h.MaxTokenBudget)
		return nil
	},
}

var hoEstimateCmd = &cobra.Command{
	Use:   "estimate",
	Short: "컨텐츠 토큰 추정",
	RunE: func(cmd *cobra.Command, args []string) error {
		contentStr, _ := cmd.Flags().GetString("content")
		filePath, _ := cmd.Flags().GetString("file")

		var content interface{}
		if filePath != "" {
			// Read from file - simplified
			content = map[string]string{"file": filePath}
		} else if contentStr != "" {
			if err := json.Unmarshal([]byte(contentStr), &content); err != nil {
				content = contentStr
			}
		} else {
			return fmt.Errorf("--content 또는 --file을 지정하세요")
		}

		tokens, err := handoff.EstimateTokens(content)
		if err != nil {
			return err
		}

		budget := handoff.MaxTokenBudget
		percent := float64(tokens) / float64(budget) * 100

		if IsJSON() {
			data, _ := json.Marshal(map[string]interface{}{
				"tokens":  tokens,
				"budget":  budget,
				"percent": percent,
				"valid":   tokens <= budget,
			})
			fmt.Println(string(data))
			return nil
		}

		status := "✓"
		if tokens > budget {
			status = "✗ 예산 초과"
		}

		fmt.Printf("Token Estimate: %d / %d (%.1f%%) %s\n", tokens, budget, percent, status)
		return nil
	},
}

var hoTotalCmd = &cobra.Command{
	Use:   "total [port-id]",
	Short: "포트의 총 Handoff 토큰",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := handoff.NewStore(database)
		total, err := store.GetTotalTokens(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.Marshal(map[string]interface{}{
				"port_id":      args[0],
				"total_tokens": total,
			})
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Port %s: Total Handoff Tokens = %d\n", args[0], total)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(handoffCmd)

	handoffCmd.AddCommand(hoListCmd)
	hoListCmd.Flags().StringP("direction", "d", "", "방향 필터 (from, to)")

	handoffCmd.AddCommand(hoShowCmd)

	handoffCmd.AddCommand(hoCreateCmd)
	hoCreateCmd.Flags().StringP("type", "t", "custom", "타입 (api_contract, file_list, type_def, schema, custom)")
	hoCreateCmd.Flags().StringP("content", "c", "", "콘텐츠 (JSON)")

	handoffCmd.AddCommand(hoEstimateCmd)
	hoEstimateCmd.Flags().StringP("content", "c", "", "콘텐츠 (JSON)")
	hoEstimateCmd.Flags().StringP("file", "f", "", "파일 경로")

	handoffCmd.AddCommand(hoTotalCmd)
}
