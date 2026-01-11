package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/agent"
	"github.com/n0roo/pal-kit/internal/config"
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

var agentTemplatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "사용 가능한 템플릿 목록",
	Long: `전역 에이전트 템플릿 목록을 표시합니다.

템플릿 위치: ~/.pal/agents/

템플릿을 프로젝트에 추가:
  pal agent add worker-go
  pal agent add core/builder
`,
	RunE: runAgentTemplates,
}

var agentAddCmd = &cobra.Command{
	Use:   "add <template>",
	Short: "템플릿에서 에이전트 추가",
	Long: `전역 템플릿에서 에이전트를 프로젝트에 추가합니다.

사용 가능한 템플릿 확인:
  pal agent templates

예시:
  pal agent add core/collaborator   # Core 에이전트
  pal agent add workers/backend/go  # Go 워커
  pal agent add workers/frontend/react  # React 워커
`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentAdd,
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentShowCmd)
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentDeleteCmd)
	agentCmd.AddCommand(agentPromptCmd)
	agentCmd.AddCommand(agentTypesCmd)
	agentCmd.AddCommand(agentTemplatesCmd)
	agentCmd.AddCommand(agentAddCmd)

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

func runAgentTemplates(cmd *cobra.Command, args []string) error {
	// 전역 템플릿 디렉토리 확인
	globalAgentsDir := config.GlobalAgentsDir()

	if _, err := os.Stat(globalAgentsDir); os.IsNotExist(err) {
		return fmt.Errorf("전역 에이전트가 설치되지 않았습니다. 'pal install' 실행하세요")
	}

	// 템플릿 스캨
	var templates []map[string]string

	err := filepath.Walk(globalAgentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		relPath, _ := filepath.Rel(globalAgentsDir, path)
		name := strings.TrimSuffix(relPath, ext)

		templates = append(templates, map[string]string{
			"name": name,
			"path": relPath,
		})

		return nil
	})

	if err != nil {
		return fmt.Errorf("템플릿 스캨 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(templates)
	}

	fmt.Println("📋 사용 가능한 에이전트 템플릿")
	fmt.Println()
	fmt.Printf("위치: %s\n", globalAgentsDir)
	fmt.Println()

	// 카테고리별 그룹화
	coreTemplates := []string{}
	backendTemplates := []string{}
	frontendTemplates := []string{}

	for _, t := range templates {
		name := t["name"]
		if strings.HasPrefix(name, "core/") {
			coreTemplates = append(coreTemplates, strings.TrimPrefix(name, "core/"))
		} else if strings.HasPrefix(name, "workers/backend/") {
			backendTemplates = append(backendTemplates, strings.TrimPrefix(name, "workers/backend/"))
		} else if strings.HasPrefix(name, "workers/frontend/") {
			frontendTemplates = append(frontendTemplates, strings.TrimPrefix(name, "workers/frontend/"))
		}
	}

	fmt.Println("🏛️  Core 에이전트:")
	for _, name := range coreTemplates {
		fmt.Printf("   - core/%s\n", name)
	}
	fmt.Println()

	fmt.Println("⚙️  Backend 워커:")
	for _, name := range backendTemplates {
		fmt.Printf("   - workers/backend/%s\n", name)
	}
	fmt.Println()

	fmt.Println("🎨 Frontend 워커:")
	for _, name := range frontendTemplates {
		fmt.Printf("   - workers/frontend/%s\n", name)
	}
	fmt.Println()

	fmt.Println("💡 프로젝트에 추가:")
	fmt.Println("   pal agent add core/collaborator")
	fmt.Println("   pal agent add workers/backend/go")

	return nil
}

func runAgentAdd(cmd *cobra.Command, args []string) error {
	templateName := args[0]

	// 프로젝트 루트 찾기
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL Kit 프로젝트를 찾을 수 없습니다")
	}

	// 전역 템플릿 경로
	globalAgentsDir := config.GlobalAgentsDir()

	// 템플릿 파일 찾기
	var templatePath string
	possiblePaths := []string{
		filepath.Join(globalAgentsDir, templateName+".yaml"),
		filepath.Join(globalAgentsDir, templateName+".yml"),
		filepath.Join(globalAgentsDir, templateName),
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			templatePath = p
			break
		}
	}

	if templatePath == "" {
		return fmt.Errorf("템플릿을 찾을 수 없습니다: %s\n사용 가능한 템플릿: pal agent templates", templateName)
	}

	// 템플릿 내용 읽기
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("템플릿 읽기 실패: %w", err)
	}

	// 대상 파일 경로 결정
	// workers/backend/go.yaml → agents/worker-go.yaml
	// core/builder.yaml → agents/builder.yaml
	baseName := filepath.Base(templatePath)
	ext := filepath.Ext(baseName)
	name := strings.TrimSuffix(baseName, ext)

	dir := filepath.Dir(templatePath)
	relDir, _ := filepath.Rel(globalAgentsDir, dir)

	var targetName string
	if relDir == "core" {
		targetName = name + ext
	} else if strings.Contains(relDir, "backend") || strings.Contains(relDir, "frontend") {
		targetName = "worker-" + name + ext
	} else {
		targetName = name + ext
	}

	// agents 디렉토리 생성
	agentsDir := filepath.Join(projectRoot, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	targetPath := filepath.Join(agentsDir, targetName)

	// 이미 존재하는지 확인
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("에이전트가 이미 존재합니다: %s", targetPath)
	}

	// 파일 쓰기
	if err := os.WriteFile(targetPath, content, 0644); err != nil {
		return fmt.Errorf("파일 작성 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":   "added",
			"template": templateName,
			"path":     targetPath,
		})
	}

	fmt.Printf("✅ 에이전트 추가: %s\n", targetName)
	fmt.Printf("   파일: %s\n", targetPath)
	fmt.Println()
	fmt.Println("💡 에이전트 확인:")
	fmt.Printf("   pal agent show %s\n", strings.TrimSuffix(targetName, ext))

	return nil
}
