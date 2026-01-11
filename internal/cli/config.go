package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "프로젝트 설정 관리",
	Long: `PAL Kit 프로젝트 설정을 관리합니다.

설정 파일: .pal/config.yaml

예시:
  pal config show              # 현재 설정 표시
  pal config init              # 설정 초기화
  pal config set workflow single
  pal config set project.name "My Project"
`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "현재 설정 표시",
	RunE:  runConfigShow,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "설정 초기화",
	Long: `프로젝트 설정을 초기화합니다.

이 명령어는 기본 설정으로 .pal/config.yaml을 생성합니다.
Claude와 대화하며 설정하려면 "PAL Kit 환경을 설정해줘"라고 요청하세요.
`,
	RunE: runConfigInit,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "설정 값 변경",
	Long: `설정 값을 변경합니다.

사용 가능한 키:
  workflow           워크플로우 타입 (simple, single, integrate, multi)
  project.name       프로젝트 이름
  project.description 프로젝트 설명
  settings.auto_port_create      자동 포트 생성 (true/false)
  settings.require_user_review   사용자 리뷰 필수 (true/false)
  settings.auto_test_on_complete 완료 시 자동 테스트 (true/false)

예시:
  pal config set workflow integrate
  pal config set project.name "My Project"
`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "설정 값 조회",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var (
	configForce bool
)

func init() {
	rootCmd.AddCommand(configCmd)
	
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)

	configInitCmd.Flags().BoolVar(&configForce, "force", false, "기존 설정 덮어쓰기")
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	projectRoot := config.FindProjectRoot()
	if projectRoot == "" {
		return fmt.Errorf("PAL Kit 프로젝트를 찾을 수 없습니다")
	}

	cfg, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		// 설정 파일이 없는 경우
		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"configured": false,
				"message":    err.Error(),
			})
		}
		fmt.Println("⚠️  프로젝트 설정이 없습니다.")
		fmt.Println()
		fmt.Println("설정 방법:")
		fmt.Println("  1. Claude에게: \"이 프로젝트의 PAL Kit 환경을 설정해줘\"")
		fmt.Println("  2. 또는: pal config init")
		return nil
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(cfg)
	}

	fmt.Println("📋 PAL Kit 프로젝트 설정")
	fmt.Println()
	fmt.Printf("프로젝트: %s\n", cfg.Project.Name)
	if cfg.Project.Description != "" {
		fmt.Printf("설명: %s\n", cfg.Project.Description)
	}
	fmt.Println()
	fmt.Printf("워크플로우: %s\n", cfg.Workflow.Type)
	fmt.Printf("  └─ %s\n", config.WorkflowDescription(cfg.Workflow.Type))
	fmt.Println()
	fmt.Println("에이전트:")
	if len(cfg.Agents.Core) > 0 {
		fmt.Printf("  Core: %s\n", strings.Join(cfg.Agents.Core, ", "))
	}
	if len(cfg.Agents.Workers) > 0 {
		fmt.Printf("  Workers: %s\n", strings.Join(cfg.Agents.Workers, ", "))
	}
	if len(cfg.Agents.Testers) > 0 {
		fmt.Printf("  Testers: %s\n", strings.Join(cfg.Agents.Testers, ", "))
	}
	fmt.Println()
	fmt.Println("설정:")
	fmt.Printf("  자동 포트 생성: %v\n", cfg.Settings.AutoPortCreate)
	fmt.Printf("  사용자 리뷰 필수: %v\n", cfg.Settings.RequireUserReview)
	fmt.Printf("  완료 시 자동 테스트: %v\n", cfg.Settings.AutoTestOnComplete)
	fmt.Println()
	fmt.Printf("설정 파일: %s\n", config.ProjectConfigPath(projectRoot))

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	projectRoot := config.FindProjectRoot()
	if projectRoot == "" {
		return fmt.Errorf("PAL Kit 프로젝트를 찾을 수 없습니다. 먼저 'pal init' 실행하세요")
	}

	configPath := config.ProjectConfigPath(projectRoot)
	if config.HasProjectConfig(projectRoot) && !configForce {
		return fmt.Errorf("설정 파일이 이미 존재합니다: %s\n--force 옵션으로 덮어쓰기 가능", configPath)
	}

	// 프로젝트 이름 추출
	projectName := projectRoot
	if idx := strings.LastIndex(projectRoot, "/"); idx >= 0 {
		projectName = projectRoot[idx+1:]
	}

	cfg := config.DefaultProjectConfig(projectName)

	if err := config.SaveProjectConfig(projectRoot, cfg); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "created",
			"path":   configPath,
			"config": cfg,
		})
	}

	fmt.Println("✅ 프로젝트 설정 생성 완료!")
	fmt.Printf("   파일: %s\n", configPath)
	fmt.Println()
	fmt.Println("기본 설정:")
	fmt.Printf("  워크플로우: %s\n", cfg.Workflow.Type)
	fmt.Printf("  에이전트: %s\n", strings.Join(cfg.Agents.Core, ", "))
	fmt.Println()
	fmt.Println("💡 워크플로우 변경:")
	fmt.Println("  pal config set workflow single")
	fmt.Println("  pal config set workflow integrate")

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	projectRoot := config.FindProjectRoot()
	if projectRoot == "" {
		return fmt.Errorf("PAL Kit 프로젝트를 찾을 수 없습니다")
	}

	key := args[0]
	value := args[1]

	// 설정 로드 (없으면 기본값)
	cfg, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		cfg = config.DefaultProjectConfig("")
	}

	// 값 설정
	switch key {
	case "workflow":
		wt := config.WorkflowType(value)
		// 유효성 검사
		valid := false
		for _, t := range config.GetWorkflowTypes() {
			if t == wt {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("유효하지 않은 워크플로우 타입: %s\n사용 가능: simple, single, integrate, multi", value)
		}
		cfg.Workflow.Type = wt
		// 워크플로우에 맞는 기본 에이전트 설정
		cfg.Agents = config.DefaultAgentsForWorkflow(wt)

	case "project.name":
		cfg.Project.Name = value

	case "project.description":
		cfg.Project.Description = value

	case "settings.auto_port_create":
		cfg.Settings.AutoPortCreate = value == "true"

	case "settings.require_user_review":
		cfg.Settings.RequireUserReview = value == "true"

	case "settings.auto_test_on_complete":
		cfg.Settings.AutoTestOnComplete = value == "true"

	default:
		return fmt.Errorf("알 수 없는 설정 키: %s", key)
	}

	// 저장
	if err := config.SaveProjectConfig(projectRoot, cfg); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "updated",
			"key":    key,
			"value":  value,
		})
	}

	fmt.Printf("✅ 설정 변경: %s = %s\n", key, value)

	// 워크플로우 변경 시 추가 정보
	if key == "workflow" {
		fmt.Println()
		fmt.Printf("에이전트 자동 설정: %s\n", strings.Join(cfg.Agents.Core, ", "))
		fmt.Println()
		fmt.Println("💡 에이전트 커스터마이징:")
		fmt.Println("  pal agent list")
		fmt.Println("  pal agent add worker-go")
	}

	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	projectRoot := config.FindProjectRoot()
	if projectRoot == "" {
		return fmt.Errorf("PAL Kit 프로젝트를 찾을 수 없습니다")
	}

	cfg, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		return err
	}

	key := args[0]
	var value interface{}

	switch key {
	case "workflow":
		value = cfg.Workflow.Type
	case "project.name":
		value = cfg.Project.Name
	case "project.description":
		value = cfg.Project.Description
	case "agents.core":
		value = cfg.Agents.Core
	case "agents.workers":
		value = cfg.Agents.Workers
	case "settings.auto_port_create":
		value = cfg.Settings.AutoPortCreate
	case "settings.require_user_review":
		value = cfg.Settings.RequireUserReview
	case "settings.auto_test_on_complete":
		value = cfg.Settings.AutoTestOnComplete
	default:
		return fmt.Errorf("알 수 없는 설정 키: %s", key)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"key":   key,
			"value": value,
		})
	}

	fmt.Printf("%v\n", value)
	return nil
}
