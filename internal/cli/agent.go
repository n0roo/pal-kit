package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/agent"
	"github.com/n0roo/pal-kit/internal/context"
	"github.com/spf13/cobra"
)

var (
	agentType   string
	agentPrompt string
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "에이전트 관리",
	Long:  `에이전트 프롬프트를 관리합니다.`,
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "에이전트 목록",
	RunE:  runAgentList,
}

var agentShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "에이전트 상세",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentShow,
}

var agentCreateCmd = &cobra.Command{
	Use:   "create <id> <name>",
	Short: "에이전트 생성",
	Args:  cobra.ExactArgs(2),
	RunE:  runAgentCreate,
}

var agentDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "에이전트 삭제",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentDelete,
}

var agentPromptCmd = &cobra.Command{
	Use:   "prompt <id>",
	Short: "에이전트 프롬프트 출력",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentPrompt,
}

var agentTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "에이전트 타입 목록",
	RunE:  runAgentTypes,
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentShowCmd)
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentDeleteCmd)
	agentCmd.AddCommand(agentPromptCmd)
	agentCmd.AddCommand(agentTypesCmd)

	agentCreateCmd.Flags().StringVar(&agentType, "type", "worker", "에이전트 타입")
	agentCreateCmd.Flags().StringVar(&agentPrompt, "prompt", "", "프롬프트 (또는 file:경로)")
}

func getAgentService() (*agent.Service, error) {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}
	return agent.NewService(projectRoot), nil
}

func runAgentList(cmd *cobra.Command, args []string) error {
	svc, err := getAgentService()
	if err != nil {
		return err
	}

	agents, err := svc.List()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(agents)
	}

	if len(agents) == 0 {
		fmt.Println("등록된 에이전트가 없습니다.")
		fmt.Println("\n에이전트 생성:")
		fmt.Println("  pal agent create <id> <name> --type worker")
		return nil
	}

	fmt.Println("📋 에이전트 목록")
	fmt.Println()

	typeEmoji := map[string]string{
		"builder":  "🏗️",
		"worker":   "👷",
		"reviewer": "🔍",
		"planner":  "📝",
		"tester":   "🧪",
		"docs":     "📚",
		"custom":   "⚙️",
	}

	for _, a := range agents {
		emoji := typeEmoji[a.Type]
		if emoji == "" {
			emoji = "🤖"
		}
		desc := a.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Printf("%s %s (%s)\n", emoji, a.Name, a.ID)
		fmt.Printf("   타입: %s\n", a.Type)
		fmt.Printf("   설명: %s\n", desc)
		fmt.Println()
	}

	return nil
}

func runAgentShow(cmd *cobra.Command, args []string) error {
	id := args[0]

	svc, err := getAgentService()
	if err != nil {
		return err
	}

	a, err := svc.Get(id)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(a)
	}

	fmt.Printf("🤖 에이전트: %s\n", a.Name)
	fmt.Println()
	fmt.Printf("ID:   %s\n", a.ID)
	fmt.Printf("타입: %s\n", a.Type)
	fmt.Printf("설명: %s\n", a.Description)
	fmt.Printf("파일: %s\n", a.FilePath)

	if len(a.Tools) > 0 {
		fmt.Printf("도구: %s\n", strings.Join(a.Tools, ", "))
	}

	if len(a.Config) > 0 {
		fmt.Println("설정:")
		for k, v := range a.Config {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	// 프롬프트 미리보기
	if a.Prompt != "" {
		fmt.Println()
		fmt.Println("📝 프롬프트 (앞부분):")
		preview := a.Prompt
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		fmt.Printf("   %s\n", strings.ReplaceAll(preview, "\n", "\n   "))
	}

	return nil
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
	id := args[0]
	name := args[1]

	svc, err := getAgentService()
	if err != nil {
		return err
	}

	a := &agent.Agent{
		ID:     id,
		Name:   name,
		Type:   agentType,
		Prompt: agentPrompt,
	}

	if err := svc.Create(a); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(a)
	}

	fmt.Printf("✅ 에이전트 생성: %s\n", name)
	fmt.Printf("   파일: %s\n", a.FilePath)

	return nil
}

func runAgentDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	svc, err := getAgentService()
	if err != nil {
		return err
	}

	if err := svc.Delete(id); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "deleted",
			"id":     id,
		})
	} else {
		fmt.Printf("✅ 에이전트 삭제: %s\n", id)
	}

	return nil
}

func runAgentPrompt(cmd *cobra.Command, args []string) error {
	id := args[0]

	svc, err := getAgentService()
	if err != nil {
		return err
	}

	prompt, err := svc.GetPrompt(id)
	if err != nil {
		return err
	}

	fmt.Println(prompt)
	return nil
}

func runAgentTypes(cmd *cobra.Command, args []string) error {
	types := agent.GetAgentTypes()

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(types)
	}

	fmt.Println("📋 에이전트 타입")
	fmt.Println()

	descriptions := map[string]string{
		"builder":  "파이프라인/포트 관리, 작업 분배",
		"worker":   "실제 코드 작성 및 수정",
		"reviewer": "코드 리뷰, 품질 검토",
		"planner":  "작업 계획 수립",
		"tester":   "테스트 코드 작성",
		"docs":     "문서화 작업",
		"custom":   "사용자 정의",
	}

	for _, t := range types {
		fmt.Printf("  %-10s  %s\n", t, descriptions[t])
	}

	return nil
}
