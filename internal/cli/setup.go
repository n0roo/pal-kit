package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/config"
	palContext "github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/docs"
	"github.com/n0roo/pal-kit/internal/workflow"
	"github.com/spf13/cobra"
)

var (
	setupAuto   bool
	setupYes    bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "대화형 프로젝트 설정",
	Long: `프로젝트를 분석하고 PAL Kit 설정을 대화형으로 진행합니다.

이 명령어는 Claude가 실행하거나 사용자가 직접 실행할 수 있습니다.

플로우:
  1. 프로젝트 구조 분석
  2. 기술 스택 감지
  3. 워크플로우 타입 추천
  4. 에이전트 추천
  5. 사용자 확인 후 적용

옵션:
  --auto  추천 설정 자동 적용
  --yes   모든 확인에 yes 응답
`,
	RunE: runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolVar(&setupAuto, "auto", false, "추천 설정 자동 적용")
	setupCmd.Flags().BoolVarP(&setupYes, "yes", "y", false, "모든 확인에 yes 응답")
}

func runSetup(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	projectRoot := palContext.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	// 1. 프로젝트 분석
	analysis := analyzeProject(projectRoot)

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"analysis": analysis,
			"status":   "analyzed",
		})
	}

	fmt.Println("🔧 PAL Kit 프로젝트 설정")
	fmt.Println("=" + strings.Repeat("=", 40))
	fmt.Println()

	// 2. 분석 결과 출력
	fmt.Printf("프로젝트: %s\n", analysis.ProjectName)
	fmt.Println()

	if len(analysis.TechStack.Languages) > 0 {
		fmt.Printf("감지된 기술 스택:\n")
		fmt.Printf("  언어: %s\n", strings.Join(analysis.TechStack.Languages, ", "))
		if len(analysis.TechStack.Frameworks) > 0 {
			fmt.Printf("  프레임워크: %s\n", strings.Join(analysis.TechStack.Frameworks, ", "))
		}
		fmt.Println()
	}

	fmt.Printf("프로젝트 규모: %s (%d개 파일)\n", analysis.Structure.EstimatedSize, analysis.Structure.EstimatedFiles)
	fmt.Println()

	// 3. 기존 설정 확인
	if analysis.Existing.HasPalConfig && analysis.Existing.CurrentConfig != nil {
		fmt.Println("⚠️  기존 설정이 있습니다:")
		fmt.Printf("   워크플로우: %s\n", analysis.Existing.CurrentConfig.Workflow.Type)
		fmt.Printf("   에이전트: %v\n", analysis.Existing.CurrentConfig.Agents.Core)
		fmt.Println()

		if !setupAuto && !setupYes {
			if !confirm("기존 설정을 덮어쓰시겠습니까?") {
				fmt.Println("설정을 유지합니다.")
				return nil
			}
		}
		fmt.Println()
	}

	// 4. 워크플로우 설정
	fmt.Println("📋 워크플로우 추천:")
	fmt.Printf("   %s\n", analysis.Suggestions.WorkflowType)
	fmt.Printf("   → %s\n", analysis.Suggestions.WorkflowReason)
	fmt.Println()

	selectedWorkflow := analysis.Suggestions.WorkflowType
	if !setupAuto {
		fmt.Println("사용 가능한 워크플로우:")
		for _, wt := range config.GetWorkflowTypes() {
			marker := "  "
			if wt == selectedWorkflow {
				marker = "→ "
			}
			fmt.Printf("  %s%s: %s\n", marker, wt, config.WorkflowDescription(wt))
		}
		fmt.Println()

		if !setupYes {
			input := prompt(fmt.Sprintf("워크플로우 선택 [%s]: ", selectedWorkflow))
			if input != "" {
				wt := config.WorkflowType(input)
				valid := false
				for _, t := range config.GetWorkflowTypes() {
					if t == wt {
						valid = true
						selectedWorkflow = wt
						break
					}
				}
				if !valid {
					fmt.Printf("⚠️  유효하지 않은 워크플로우입니다. 기본값(%s) 사용\n", selectedWorkflow)
				}
			}
		}
	}

	// 5. 에이전트 설정
	fmt.Println()
	fmt.Println("🤖 추천 에이전트:")
	
	selectedAgents := []AgentSuggestion{}
	for _, agent := range analysis.Suggestions.RecommendedAgents {
		fmt.Printf("  - %s (%s)\n", agent.Name, agent.Type)
		fmt.Printf("    이유: %s\n", agent.Reason)
		selectedAgents = append(selectedAgents, agent)
	}
	fmt.Println()

	if !setupAuto && !setupYes {
		if !confirm("추천 에이전트를 설치하시겠습니까?") {
			selectedAgents = []AgentSuggestion{}
		}
	}

	// 6. 설정 적용
	fmt.Println()
	fmt.Println("⚙️  설정 적용 중...")
	fmt.Println()

	// 워크플로우 설정 저장
	cfg, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		cfg = config.DefaultProjectConfig(analysis.ProjectName)
	}

	cfg.Workflow.Type = selectedWorkflow
	cfg.Agents = config.DefaultAgentsForWorkflow(selectedWorkflow)

	// 워커 에이전트 추가
	for _, agent := range selectedAgents {
		if agent.Type == "worker" {
			cfg.Agents.Workers = append(cfg.Agents.Workers, agent.ID)
		}
	}

	if err := config.SaveProjectConfig(projectRoot, cfg); err != nil {
		return fmt.Errorf("설정 저장 실패: %w", err)
	}
	fmt.Printf("  ✅ 워크플로우 설정: %s\n", selectedWorkflow)

	// 에이전트 템플릿 복사
	for _, agent := range selectedAgents {
		if agent.Type == "worker" && agent.Template != "" {
			if err := copyAgentTemplate(projectRoot, agent.Template); err != nil {
				fmt.Printf("  ⚠️  에이전트 복사 실패: %s (%v)\n", agent.Name, err)
			} else {
				fmt.Printf("  ✅ 에이전트 추가: %s\n", agent.Name)
			}
		}
	}

	// 워크플로우 rules 갱신
	workflowSvc := workflow.NewService(projectRoot)
	ctx, err := workflowSvc.GetContext()
	if err == nil {
		if err := workflowSvc.WriteRulesFile(ctx); err == nil {
			fmt.Println("  ✅ 워크플로우 컨텍스트 생성")
		}
	}

	// CLAUDE.md 업데이트
	docsSvc := docs.NewService(projectRoot)
	if err := docsSvc.UpdateClaudeMDAfterSetup(cfg); err != nil {
		fmt.Printf("  ⚠️  CLAUDE.md 업데이트 실패: %v\n", err)
	} else {
		fmt.Println("  ✅ CLAUDE.md 업데이트")
	}

	fmt.Println()
	fmt.Println("🎉 설정 완료!")
	fmt.Println()
	fmt.Println("다음 단계:")
	fmt.Println("  1. CLAUDE.md의 '개요' 섹션에 프로젝트 설명 추가")
	fmt.Println("  2. claude 실행하여 작업 시작")
	fmt.Println()
	fmt.Println("설정 확인: pal config show")
	fmt.Println("워크플로우 확인: pal workflow show")

	return nil
}

func copyAgentTemplate(projectRoot, templatePath string) error {
	globalAgentsDir := config.GlobalAgentsDir()
	
	// 템플릿 파일 경로
	srcPath := globalAgentsDir + "/" + templatePath + ".yaml"
	
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// 대상 파일명 결정
	baseName := templatePath
	if idx := strings.LastIndex(templatePath, "/"); idx >= 0 {
		baseName = templatePath[idx+1:]
	}
	
	// workers/backend/go -> worker-go
	if strings.Contains(templatePath, "workers/") {
		baseName = "worker-" + baseName
	}

	// agents 디렉토리 생성
	agentsDir := projectRoot + "/agents"
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return err
	}

	dstPath := agentsDir + "/" + baseName + ".yaml"
	
	// 이미 존재하면 스킵
	if _, err := os.Stat(dstPath); err == nil {
		return nil
	}

	return os.WriteFile(dstPath, content, 0644)
}

func confirm(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", message)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func prompt(message string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(message)
	response, _ := reader.ReadString('\n')
	return strings.TrimSpace(response)
}
