package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/config"
	palContext "github.com/n0roo/pal-kit/internal/context"
	"github.com/spf13/cobra"
)

// ProjectAnalysis holds project analysis results
type ProjectAnalysis struct {
	ProjectRoot string              `json:"project_root"`
	ProjectName string              `json:"project_name"`
	TechStack   TechStackInfo       `json:"tech_stack"`
	Structure   ProjectStructure    `json:"structure"`
	Existing    ExistingConfig      `json:"existing"`
	Suggestions SetupSuggestions    `json:"suggestions"`
}

// TechStackInfo holds detected technology stack
type TechStackInfo struct {
	Languages   []string          `json:"languages"`
	Frameworks  []string          `json:"frameworks"`
	BuildTools  []string          `json:"build_tools"`
	Indicators  map[string]string `json:"indicators"` // file -> detected tech
}

// ProjectStructure holds directory structure info
type ProjectStructure struct {
	HasSrc         bool     `json:"has_src"`
	HasTests       bool     `json:"has_tests"`
	HasDocs        bool     `json:"has_docs"`
	MainDirs       []string `json:"main_dirs"`
	ConfigFiles    []string `json:"config_files"`
	EstimatedSize  string   `json:"estimated_size"` // small, medium, large
	EstimatedFiles int      `json:"estimated_files"`
}

// ExistingConfig holds existing PAL Kit configuration
type ExistingConfig struct {
	HasClaudeMD    bool                `json:"has_claude_md"`
	HasPalConfig   bool                `json:"has_pal_config"`
	HasAgents      bool                `json:"has_agents"`
	HasConventions bool                `json:"has_conventions"`
	HasPorts       bool                `json:"has_ports"`
	CurrentConfig  *config.ProjectConfig `json:"current_config,omitempty"`
}

// SetupSuggestions holds recommended setup
type SetupSuggestions struct {
	WorkflowType     config.WorkflowType `json:"workflow_type"`
	WorkflowReason   string              `json:"workflow_reason"`
	RecommendedAgents []AgentSuggestion  `json:"recommended_agents"`
	ConventionHints  []string            `json:"convention_hints"`
}

// AgentSuggestion holds agent recommendation
type AgentSuggestion struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // core, worker
	Reason   string `json:"reason"`
	Template string `json:"template"` // template path
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "프로젝트 분석",
	Long: `프로젝트 구조를 분석하고 PAL Kit 설정을 제안합니다.

분석 항목:
  - 기술 스택 (언어, 프레임워크)
  - 프로젝트 구조
  - 기존 설정 확인
  - 워크플로우/에이전트 추천

Claude가 이 명령어를 실행하여 설정을 도와줍니다.
`,
	RunE: runAnalyze,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	projectRoot := palContext.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	analysis := analyzeProject(projectRoot)

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(analysis)
	}

	printAnalysis(analysis)
	return nil
}

func analyzeProject(projectRoot string) *ProjectAnalysis {
	projectName := filepath.Base(projectRoot)

	analysis := &ProjectAnalysis{
		ProjectRoot: projectRoot,
		ProjectName: projectName,
		TechStack:   detectTechStack(projectRoot),
		Structure:   analyzeStructure(projectRoot),
		Existing:    checkExistingConfig(projectRoot),
	}

	// 분석 결과 기반 제안
	analysis.Suggestions = generateSuggestions(analysis)

	return analysis
}

func detectTechStack(projectRoot string) TechStackInfo {
	info := TechStackInfo{
		Languages:  []string{},
		Frameworks: []string{},
		BuildTools: []string{},
		Indicators: make(map[string]string),
	}

	// 언어/프레임워크 감지 규칙
	detectionRules := map[string]struct {
		language  string
		framework string
		buildTool string
	}{
		"go.mod":           {"Go", "", "go"},
		"go.sum":           {"Go", "", "go"},
		"package.json":     {"JavaScript/TypeScript", "", "npm"},
		"pnpm-lock.yaml":   {"JavaScript/TypeScript", "", "pnpm"},
		"yarn.lock":        {"JavaScript/TypeScript", "", "yarn"},
		"tsconfig.json":    {"TypeScript", "", ""},
		"next.config.js":   {"", "Next.js", ""},
		"next.config.mjs":  {"", "Next.js", ""},
		"next.config.ts":   {"", "Next.js", ""},
		"vite.config.ts":   {"", "Vite", ""},
		"vite.config.js":   {"", "Vite", ""},
		"nuxt.config.ts":   {"", "Nuxt", ""},
		"angular.json":     {"", "Angular", ""},
		"build.gradle":     {"Java/Kotlin", "", "gradle"},
		"build.gradle.kts": {"Kotlin", "", "gradle"},
		"pom.xml":          {"Java", "", "maven"},
		"requirements.txt": {"Python", "", "pip"},
		"pyproject.toml":   {"Python", "", "poetry/pip"},
		"Cargo.toml":       {"Rust", "", "cargo"},
		"Gemfile":          {"Ruby", "", "bundler"},
		"composer.json":    {"PHP", "", "composer"},
		"Makefile":         {"", "", "make"},
		"CMakeLists.txt":   {"C/C++", "", "cmake"},
		".eslintrc.js":     {"", "", "eslint"},
		".prettierrc":      {"", "", "prettier"},
		"nest-cli.json":    {"", "NestJS", ""},
		"tailwind.config.js": {"", "Tailwind CSS", ""},
	}

	langSet := make(map[string]bool)
	frameworkSet := make(map[string]bool)
	buildToolSet := make(map[string]bool)

	for file, detection := range detectionRules {
		path := filepath.Join(projectRoot, file)
		if _, err := os.Stat(path); err == nil {
			info.Indicators[file] = fmt.Sprintf("%s/%s/%s", detection.language, detection.framework, detection.buildTool)
			
			if detection.language != "" {
				langSet[detection.language] = true
			}
			if detection.framework != "" {
				frameworkSet[detection.framework] = true
			}
			if detection.buildTool != "" {
				buildToolSet[detection.buildTool] = true
			}
		}
	}

	for lang := range langSet {
		info.Languages = append(info.Languages, lang)
	}
	for fw := range frameworkSet {
		info.Frameworks = append(info.Frameworks, fw)
	}
	for bt := range buildToolSet {
		info.BuildTools = append(info.BuildTools, bt)
	}

	return info
}

func analyzeStructure(projectRoot string) ProjectStructure {
	structure := ProjectStructure{
		MainDirs:    []string{},
		ConfigFiles: []string{},
	}

	// 주요 디렉토리 확인
	commonDirs := []string{"src", "lib", "pkg", "internal", "cmd", "app", "components", "pages", "api", "tests", "test", "__tests__", "spec", "docs", "doc"}
	for _, dir := range commonDirs {
		path := filepath.Join(projectRoot, dir)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			structure.MainDirs = append(structure.MainDirs, dir)
			
			switch dir {
			case "src", "lib", "pkg", "internal", "cmd", "app":
				structure.HasSrc = true
			case "tests", "test", "__tests__", "spec":
				structure.HasTests = true
			case "docs", "doc":
				structure.HasDocs = true
			}
		}
	}

	// 설정 파일 확인
	configPatterns := []string{".env*", "*.config.*", "tsconfig*.json", ".eslintrc*", ".prettierrc*"}
	for _, pattern := range configPatterns {
		matches, _ := filepath.Glob(filepath.Join(projectRoot, pattern))
		for _, match := range matches {
			structure.ConfigFiles = append(structure.ConfigFiles, filepath.Base(match))
		}
	}

	// 파일 수 추정
	fileCount := 0
	filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// node_modules, .git 등 제외
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
		}
		if !info.IsDir() {
			fileCount++
		}
		return nil
	})

	structure.EstimatedFiles = fileCount
	if fileCount < 50 {
		structure.EstimatedSize = "small"
	} else if fileCount < 500 {
		structure.EstimatedSize = "medium"
	} else {
		structure.EstimatedSize = "large"
	}

	return structure
}

func checkExistingConfig(projectRoot string) ExistingConfig {
	existing := ExistingConfig{}

	// CLAUDE.md
	if _, err := os.Stat(filepath.Join(projectRoot, "CLAUDE.md")); err == nil {
		existing.HasClaudeMD = true
	}

	// .pal/config.yaml
	if cfg, err := config.LoadProjectConfig(projectRoot); err == nil {
		existing.HasPalConfig = true
		existing.CurrentConfig = cfg
	}

	// agents/
	if entries, err := os.ReadDir(filepath.Join(projectRoot, "agents")); err == nil && len(entries) > 0 {
		existing.HasAgents = true
	}

	// conventions/
	if entries, err := os.ReadDir(filepath.Join(projectRoot, "conventions")); err == nil && len(entries) > 0 {
		existing.HasConventions = true
	}

	// ports/
	if entries, err := os.ReadDir(filepath.Join(projectRoot, "ports")); err == nil && len(entries) > 0 {
		existing.HasPorts = true
	}

	return existing
}

func generateSuggestions(analysis *ProjectAnalysis) SetupSuggestions {
	suggestions := SetupSuggestions{
		RecommendedAgents: []AgentSuggestion{},
		ConventionHints:   []string{},
	}

	// 워크플로우 타입 결정
	size := analysis.Structure.EstimatedSize
	techCount := len(analysis.TechStack.Languages) + len(analysis.TechStack.Frameworks)

	switch {
	case size == "small" && techCount <= 2:
		suggestions.WorkflowType = config.WorkflowSimple
		suggestions.WorkflowReason = "작은 프로젝트이며 단일 기술 스택으로 simple 워크플로우가 적합합니다"
	case size == "medium" || techCount <= 3:
		suggestions.WorkflowType = config.WorkflowSingle
		suggestions.WorkflowReason = "중간 규모 프로젝트로 역할 전환이 가능한 single 워크플로우를 권장합니다"
	case size == "large" || techCount > 3:
		suggestions.WorkflowType = config.WorkflowIntegrate
		suggestions.WorkflowReason = "큰 프로젝트이거나 복수 기술 스택으로 integrate 워크플로우가 효율적입니다"
	default:
		suggestions.WorkflowType = config.WorkflowSimple
		suggestions.WorkflowReason = "기본 워크플로우로 시작하고 필요시 변경하세요"
	}

	// 에이전트 추천
	// Core 에이전트
	coreAgents := config.DefaultAgentsForWorkflow(suggestions.WorkflowType)
	for _, agentID := range coreAgents.Core {
		suggestions.RecommendedAgents = append(suggestions.RecommendedAgents, AgentSuggestion{
			ID:       agentID,
			Name:     agentID,
			Type:     "core",
			Template: "core/" + agentID,
			Reason:   fmt.Sprintf("%s 워크플로우의 기본 에이전트", suggestions.WorkflowType),
		})
	}

	// Worker 에이전트 (기술 스택 기반)
	for _, lang := range analysis.TechStack.Languages {
		switch {
		case strings.Contains(lang, "Go"):
			suggestions.RecommendedAgents = append(suggestions.RecommendedAgents, AgentSuggestion{
				ID:       "worker-go",
				Name:     "Go Worker",
				Type:     "worker",
				Template: "workers/backend/go",
				Reason:   "Go 프로젝트 감지됨",
			})
		case strings.Contains(lang, "TypeScript") || strings.Contains(lang, "JavaScript"):
			// 프레임워크 확인
			for _, fw := range analysis.TechStack.Frameworks {
				switch fw {
				case "Next.js":
					suggestions.RecommendedAgents = append(suggestions.RecommendedAgents, AgentSuggestion{
						ID:       "worker-next",
						Name:     "Next.js Worker",
						Type:     "worker",
						Template: "workers/frontend/next",
						Reason:   "Next.js 프로젝트 감지됨",
					})
				case "NestJS":
					suggestions.RecommendedAgents = append(suggestions.RecommendedAgents, AgentSuggestion{
						ID:       "worker-nestjs",
						Name:     "NestJS Worker",
						Type:     "worker",
						Template: "workers/backend/nestjs",
						Reason:   "NestJS 프로젝트 감지됨",
					})
				}
			}
			// 기본 React
			if !containsFramework(analysis.TechStack.Frameworks, "Next.js") {
				hasReact := containsFramework(analysis.TechStack.Frameworks, "Vite") || 
				            analysis.Structure.HasSrc
				if hasReact {
					suggestions.RecommendedAgents = append(suggestions.RecommendedAgents, AgentSuggestion{
						ID:       "worker-react",
						Name:     "React Worker",
						Type:     "worker",
						Template: "workers/frontend/react",
						Reason:   "React/TypeScript 프로젝트로 추정",
					})
				}
			}
		case strings.Contains(lang, "Kotlin") || strings.Contains(lang, "Java"):
			suggestions.RecommendedAgents = append(suggestions.RecommendedAgents, AgentSuggestion{
				ID:       "worker-kotlin",
				Name:     "Kotlin Worker",
				Type:     "worker",
				Template: "workers/backend/kotlin",
				Reason:   "Kotlin/Java 프로젝트 감지됨",
			})
		}
	}

	// 컨벤션 힌트
	for _, bt := range analysis.TechStack.BuildTools {
		switch bt {
		case "eslint":
			suggestions.ConventionHints = append(suggestions.ConventionHints, "ESLint 설정이 있습니다. 린트 규칙을 컨벤션에 반영하세요.")
		case "prettier":
			suggestions.ConventionHints = append(suggestions.ConventionHints, "Prettier 설정이 있습니다. 코드 포맷 규칙을 컨벤션에 반영하세요.")
		}
	}

	if analysis.Structure.HasTests {
		suggestions.ConventionHints = append(suggestions.ConventionHints, "테스트 디렉토리가 있습니다. 테스트 컨벤션을 정의하세요.")
	}

	return suggestions
}

func containsFramework(frameworks []string, target string) bool {
	for _, fw := range frameworks {
		if fw == target {
			return true
		}
	}
	return false
}

func printAnalysis(analysis *ProjectAnalysis) {
	fmt.Println("🔍 프로젝트 분석 결과")
	fmt.Println()
	fmt.Printf("프로젝트: %s\n", analysis.ProjectName)
	fmt.Printf("경로: %s\n", analysis.ProjectRoot)
	fmt.Println()

	// 기술 스택
	fmt.Println("📚 기술 스택:")
	if len(analysis.TechStack.Languages) > 0 {
		fmt.Printf("   언어: %s\n", strings.Join(analysis.TechStack.Languages, ", "))
	}
	if len(analysis.TechStack.Frameworks) > 0 {
		fmt.Printf("   프레임워크: %s\n", strings.Join(analysis.TechStack.Frameworks, ", "))
	}
	if len(analysis.TechStack.BuildTools) > 0 {
		fmt.Printf("   빌드 도구: %s\n", strings.Join(analysis.TechStack.BuildTools, ", "))
	}
	fmt.Println()

	// 프로젝트 구조
	fmt.Println("📁 프로젝트 구조:")
	fmt.Printf("   규모: %s (%d개 파일)\n", analysis.Structure.EstimatedSize, analysis.Structure.EstimatedFiles)
	if len(analysis.Structure.MainDirs) > 0 {
		fmt.Printf("   주요 디렉토리: %s\n", strings.Join(analysis.Structure.MainDirs, ", "))
	}
	fmt.Println()

	// 기존 설정
	fmt.Println("⚙️  기존 PAL Kit 설정:")
	fmt.Printf("   CLAUDE.md: %v\n", boolToEmoji(analysis.Existing.HasClaudeMD))
	fmt.Printf("   config.yaml: %v\n", boolToEmoji(analysis.Existing.HasPalConfig))
	fmt.Printf("   agents/: %v\n", boolToEmoji(analysis.Existing.HasAgents))
	fmt.Printf("   conventions/: %v\n", boolToEmoji(analysis.Existing.HasConventions))
	fmt.Println()

	// 추천
	fmt.Println("💡 추천 설정:")
	fmt.Printf("   워크플로우: %s\n", analysis.Suggestions.WorkflowType)
	fmt.Printf("   이유: %s\n", analysis.Suggestions.WorkflowReason)
	fmt.Println()

	if len(analysis.Suggestions.RecommendedAgents) > 0 {
		fmt.Println("   추천 에이전트:")
		for _, agent := range analysis.Suggestions.RecommendedAgents {
			fmt.Printf("   - %s (%s): %s\n", agent.Name, agent.Type, agent.Reason)
		}
	}

	if len(analysis.Suggestions.ConventionHints) > 0 {
		fmt.Println()
		fmt.Println("   컨벤션 힌트:")
		for _, hint := range analysis.Suggestions.ConventionHints {
			fmt.Printf("   - %s\n", hint)
		}
	}

	fmt.Println()
	fmt.Println("📋 다음 단계:")
	fmt.Println("   1. pal config set workflow " + string(analysis.Suggestions.WorkflowType))
	for _, agent := range analysis.Suggestions.RecommendedAgents {
		if agent.Type == "worker" {
			fmt.Printf("   2. pal agent add %s\n", agent.Template)
		}
	}
}

func boolToEmoji(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}
