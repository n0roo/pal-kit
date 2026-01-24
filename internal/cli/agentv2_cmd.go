package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0roo/pal-kit/internal/agentv2"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var agentV2Cmd = &cobra.Command{
	Use:     "agent",
	Aliases: []string{"ag"},
	Short:   "에이전트 관리 (v2)",
	Long:    `에이전트 정의와 버전을 관리합니다.`,
}

var agListCmd = &cobra.Command{
	Use:   "list",
	Short: "에이전트 목록",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := agentv2.NewStore(database.DB)
		agentType, _ := cmd.Flags().GetString("type")

		agents, err := store.ListAgents(agentv2.AgentType(agentType))
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(agents, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(agents) == 0 {
			fmt.Println("에이전트가 없습니다.")
			return nil
		}

		fmt.Printf("%-36s %-20s %-10s %s\n", "ID", "Name", "Type", "Version")
		fmt.Println(strings.Repeat("-", 80))
		for _, a := range agents {
			fmt.Printf("%-36s %-20s %-10s v%d\n",
				truncate(a.ID, 36),
				truncate(a.Name, 20),
				a.Type,
				a.CurrentVersion)
		}

		return nil
	},
}

var agShowCmd = &cobra.Command{
	Use:   "show [id-or-name]",
	Short: "에이전트 상세 조회",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := agentv2.NewStore(database.DB)

		// Try by ID first, then by name
		agent, err := store.GetAgent(args[0])
		if err != nil {
			agent, err = store.GetAgentByName(args[0])
			if err != nil {
				return err
			}
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(agent, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("ID: %s\n", agent.ID)
		fmt.Printf("Name: %s\n", agent.Name)
		fmt.Printf("Type: %s\n", agent.Type)
		fmt.Printf("Version: v%d\n", agent.CurrentVersion)
		if agent.Description != "" {
			fmt.Printf("Description: %s\n", agent.Description)
		}
		if len(agent.Capabilities) > 0 {
			fmt.Printf("Capabilities: %s\n", strings.Join(agent.Capabilities, ", "))
		}
		fmt.Printf("Created: %s\n", agent.CreatedAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

var agVersionsCmd = &cobra.Command{
	Use:   "versions [agent-id]",
	Short: "에이전트 버전 목록",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := agentv2.NewStore(database.DB)
		versions, err := store.ListVersions(args[0])
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(versions, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(versions) == 0 {
			fmt.Println("버전이 없습니다.")
			return nil
		}

		fmt.Printf("%-5s %-12s %-10s %-10s %-10s %s\n", "Ver", "Status", "Attention", "Completion", "Usage", "Summary")
		fmt.Println(strings.Repeat("-", 90))
		for _, v := range versions {
			summary := truncate(v.ChangeSummary, 30)
			fmt.Printf("v%-4d %-12s %-10.2f %-10.2f %-10d %s\n",
				v.Version,
				v.Status,
				v.AvgAttentionScore,
				v.AvgCompletionRate,
				v.UsageCount,
				summary)
		}

		return nil
	},
}

var agStatsCmd = &cobra.Command{
	Use:   "stats [agent-id] [version]",
	Short: "에이전트 버전 통계",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := agentv2.NewStore(database.DB)

		var version int
		fmt.Sscanf(args[1], "%d", &version)
		if version == 0 {
			fmt.Sscanf(args[1], "v%d", &version)
		}

		stats, err := store.GetVersionStats(args[0], version)
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Agent: %s v%d\n", args[0], version)
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Usage Count:    %d\n", stats.UsageCount)
		fmt.Printf("Avg Attention:  %.2f\n", stats.AvgAttention)
		fmt.Printf("Avg Quality:    %.2f\n", stats.AvgQuality)
		fmt.Printf("Success Rate:   %.1f%%\n", stats.SuccessRate)
		fmt.Printf("Avg Tokens:     %.0f\n", stats.AvgTokens)
		fmt.Printf("Avg Compacts:   %.1f\n", stats.AvgCompacts)

		return nil
	},
}

var agCompareCmd = &cobra.Command{
	Use:   "compare [agent-id]",
	Short: "버전 비교",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := agentv2.NewStore(database.DB)

		v1, _ := cmd.Flags().GetInt("v1")
		v2, _ := cmd.Flags().GetInt("v2")

		comparison, err := store.CompareVersions(args[0], v1, v2)
		if err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(comparison, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Comparing v%d vs v%d\n", comparison.Version1, comparison.Version2)
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Attention:    %+.2f\n", comparison.AttentionDiff)
		fmt.Printf("Quality:      %+.2f\n", comparison.QualityDiff)
		fmt.Printf("Success Rate: %+.1f%%\n", comparison.SuccessRateDiff)
		fmt.Printf("Tokens:       %+.0f\n", comparison.TokenDiff)
		fmt.Printf("Compacts:     %+.1f\n", comparison.CompactDiff)
		fmt.Println()
		fmt.Printf("Recommendation: %s\n", comparison.RecommendedAction)

		return nil
	},
}

var agCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "에이전트 생성",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := agentv2.NewStore(database.DB)

		agentType, _ := cmd.Flags().GetString("type")
		description, _ := cmd.Flags().GetString("description")
		capabilitiesStr, _ := cmd.Flags().GetString("capabilities")

		var capabilities []string
		if capabilitiesStr != "" {
			capabilities = strings.Split(capabilitiesStr, ",")
		}

		agent := &agentv2.Agent{
			Name:         args[0],
			Type:         agentv2.AgentType(agentType),
			Description:  description,
			Capabilities: capabilities,
		}

		if err := store.CreateAgent(agent); err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(agent, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("✓ 에이전트 생성됨: %s (ID: %s)\n", agent.Name, agent.ID)
		return nil
	},
}

var agNewVersionCmd = &cobra.Command{
	Use:   "new-version [agent-id]",
	Short: "새 버전 생성",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.Open(GetDBPath())
		if err != nil {
			return err
		}
		defer database.Close()

		store := agentv2.NewStore(database.DB)

		specFile, _ := cmd.Flags().GetString("spec")
		summary, _ := cmd.Flags().GetString("summary")
		reason, _ := cmd.Flags().GetString("reason")

		specContent := ""
		if specFile != "" {
			// In real implementation, read from file
			specContent = fmt.Sprintf("Spec from %s", specFile)
		}

		version := &agentv2.AgentVersion{
			AgentID:       args[0],
			SpecContent:   specContent,
			ChangeSummary: summary,
			ChangeReason:  reason,
		}

		if err := store.CreateVersion(version); err != nil {
			return err
		}

		if IsJSON() {
			data, _ := json.MarshalIndent(version, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("✓ 버전 생성됨: %s v%d\n", args[0], version.Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(agentV2Cmd)

	agentV2Cmd.AddCommand(agListCmd)
	agListCmd.Flags().StringP("type", "t", "", "타입 필터 (spec, operator, worker, test)")

	agentV2Cmd.AddCommand(agShowCmd)
	agentV2Cmd.AddCommand(agVersionsCmd)
	agentV2Cmd.AddCommand(agStatsCmd)

	agentV2Cmd.AddCommand(agCompareCmd)
	agCompareCmd.Flags().Int("v1", 0, "비교할 첫 번째 버전")
	agCompareCmd.Flags().Int("v2", 0, "비교할 두 번째 버전")
	agCompareCmd.MarkFlagRequired("v1")
	agCompareCmd.MarkFlagRequired("v2")

	agentV2Cmd.AddCommand(agCreateCmd)
	agCreateCmd.Flags().StringP("type", "t", "worker", "타입 (spec, operator, worker, test)")
	agCreateCmd.Flags().StringP("description", "d", "", "설명")
	agCreateCmd.Flags().StringP("capabilities", "c", "", "능력 목록 (쉼표 구분)")

	agentV2Cmd.AddCommand(agNewVersionCmd)
	agNewVersionCmd.Flags().StringP("spec", "f", "", "명세 파일 경로")
	agNewVersionCmd.Flags().StringP("summary", "s", "", "변경 사항 요약")
	agNewVersionCmd.Flags().StringP("reason", "r", "", "변경 이유")
}
