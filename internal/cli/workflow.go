package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/config"
	palContext "github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/workflow"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "워크플로우 관리",
	Long: `PAL Kit 워크플로우를 관리합니다.

워크플로우 타입:
  simple    - 대화형 협업, 종합 에이전트
  single    - 단일 세션, 역할 전환
  integrate - 빌더 관리, 서브세션
  multi     - 복수 integrate (대규모)

예시:
  pal workflow show      # 현재 워크플로우 확인
  pal workflow set simple
  pal workflow context   # 컨텍스트 미리보기
`,
}

var workflowShowCmd = &cobra.Command{
	Use:   "show",
	Short: "현재 워크플로우 표시",
	RunE:  runWorkflowShow,
}

var workflowSetCmd = &cobra.Command{
	Use:   "set <type>",
	Short: "워크플로우 타입 설정",
	Long: `워크플로우 타입을 설정합니다.

사용 가능한 타입:
  simple    - 대화형 협업, 종합 에이전트 (기본)
  single    - 단일 세션, 역할 전환
  integrate - 빌더 관리, 서브세션
  multi     - 복수 integrate (대규모)
`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkflowSet,
}

var workflowContextCmd = &cobra.Command{
	Use:   "context",
	Short: "워크플로우 컨텍스트 미리보기",
	Long:  `현재 워크플로우에 대한 컨텍스트(rules 파일 내용)를 미리봅니다.`,
	RunE:  runWorkflowContext,
}

var workflowRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "워크플로우 컨텍스트 갱신",
	Long:  `워크플로우 rules 파일을 다시 생성합니다.`,
	RunE:  runWorkflowRefresh,
}

func init() {
	rootCmd.AddCommand(workflowCmd)
	workflowCmd.AddCommand(workflowShowCmd)
	workflowCmd.AddCommand(workflowSetCmd)
	workflowCmd.AddCommand(workflowContextCmd)
	workflowCmd.AddCommand(workflowRefreshCmd)
}

func getWorkflowService() (*workflow.Service, string, error) {
	cwd, _ := os.Getwd()
	projectRoot := palContext.FindProjectRoot(cwd)
	if projectRoot == "" {
		return nil, "", fmt.Errorf("PAL Kit 프로젝트를 찾을 수 없습니다")
	}
	return workflow.NewService(projectRoot), projectRoot, nil
}

func runWorkflowShow(cmd *cobra.Command, args []string) error {
	svc, projectRoot, err := getWorkflowService()
	if err != nil {
		return err
	}

	ctx, err := svc.GetContext()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(ctx)
	}

	fmt.Println("📋 워크플로우 정보")
	fmt.Println()
	fmt.Printf("프로젝트: %s\n", ctx.ProjectName)
	fmt.Printf("타입: %s\n", ctx.WorkflowType)
	fmt.Printf("설명: %s\n", config.WorkflowDescription(ctx.WorkflowType))
	fmt.Println()

	// 에이전트 정보
	if len(ctx.Agents.Core) > 0 {
		fmt.Printf("Core 에이전트: %v\n", ctx.Agents.Core)
	}
	if len(ctx.Agents.Workers) > 0 {
		fmt.Printf("Workers: %v\n", ctx.Agents.Workers)
	}

	// rules 파일 경로
	rulesPath := svc.GetRulesPath()
	if _, err := os.Stat(rulesPath); err == nil {
		fmt.Println()
		fmt.Printf("Rules 파일: %s\n", rulesPath)
	}

	// 설정 파일 경로
	configPath := config.ProjectConfigPath(projectRoot)
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("설정 파일: %s\n", configPath)
	}

	return nil
}

func runWorkflowSet(cmd *cobra.Command, args []string) error {
	workflowType := args[0]

	// 유효성 검사
	wt := config.WorkflowType(workflowType)
	valid := false
	for _, t := range config.GetWorkflowTypes() {
		if t == wt {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("유효하지 않은 워크플로우 타입: %s\n사용 가능: simple, single, integrate, multi", workflowType)
	}

	svc, projectRoot, err := getWorkflowService()
	if err != nil {
		return err
	}

	// 설정 로드 또는 생성
	cfg, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		// 기본 설정 생성
		projectName := projectRoot
		if idx := len(projectRoot) - 1; idx >= 0 {
			for i := idx; i >= 0; i-- {
				if projectRoot[i] == '/' {
					projectName = projectRoot[i+1:]
					break
				}
			}
		}
		cfg = config.DefaultProjectConfig(projectName)
	}

	// 워크플로우 설정
	cfg.Workflow.Type = wt
	cfg.Agents = config.DefaultAgentsForWorkflow(wt)

	// 저장
	if err := config.SaveProjectConfig(projectRoot, cfg); err != nil {
		return err
	}

	// rules 파일 갱신
	ctx, err := svc.GetContext()
	if err == nil {
		svc.WriteRulesFile(ctx)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":   "updated",
			"workflow": wt,
			"agents":   cfg.Agents,
		})
	}

	fmt.Printf("✅ 워크플로우 변경: %s\n", wt)
	fmt.Printf("   설명: %s\n", config.WorkflowDescription(wt))
	fmt.Println()
	fmt.Printf("에이전트 설정: %v\n", cfg.Agents.Core)

	return nil
}

func runWorkflowContext(cmd *cobra.Command, args []string) error {
	svc, _, err := getWorkflowService()
	if err != nil {
		return err
	}

	ctx, err := svc.GetContext()
	if err != nil {
		return err
	}

	content := svc.GenerateRulesContent(ctx)

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"workflow": ctx.WorkflowType,
			"content":  content,
		})
	}

	fmt.Println(content)
	return nil
}

func runWorkflowRefresh(cmd *cobra.Command, args []string) error {
	svc, _, err := getWorkflowService()
	if err != nil {
		return err
	}

	ctx, err := svc.GetContext()
	if err != nil {
		return err
	}

	if err := svc.WriteRulesFile(ctx); err != nil {
		return fmt.Errorf("rules 파일 갱신 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "refreshed",
			"path":   svc.GetRulesPath(),
		})
	}

	fmt.Println("✅ 워크플로우 컨텍스트 갱신 완료")
	fmt.Printf("   파일: %s\n", svc.GetRulesPath())

	return nil
}
