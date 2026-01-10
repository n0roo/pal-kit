package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/convention"
	"github.com/spf13/cobra"
)

var (
	convEnabled  bool
	convPriority int
	convFileTypes []string
)

var conventionCmd = &cobra.Command{
	Use:     "convention",
	Aliases: []string{"conv"},
	Short:   "컨벤션 관리",
	Long:    `프로젝트 컨벤션을 관리합니다.`,
}

var convListCmd = &cobra.Command{
	Use:   "list",
	Short: "컨벤션 목록",
	RunE:  runConvList,
}

var convShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "컨벤션 상세",
	Args:  cobra.ExactArgs(1),
	RunE:  runConvShow,
}

var convCreateCmd = &cobra.Command{
	Use:   "create <id> <name>",
	Short: "컨벤션 생성",
	Args:  cobra.ExactArgs(2),
	RunE:  runConvCreate,
}

var convDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "컨벤션 삭제",
	Args:  cobra.ExactArgs(1),
	RunE:  runConvDelete,
}

var convEnableCmd = &cobra.Command{
	Use:   "enable <id>",
	Short: "컨벤션 활성화",
	Args:  cobra.ExactArgs(1),
	RunE:  runConvEnable,
}

var convDisableCmd = &cobra.Command{
	Use:   "disable <id>",
	Short: "컨벤션 비활성화",
	Args:  cobra.ExactArgs(1),
	RunE:  runConvDisable,
}

var convCheckCmd = &cobra.Command{
	Use:   "check [paths...]",
	Short: "컨벤션 준수 검사",
	RunE:  runConvCheck,
}

var convLearnCmd = &cobra.Command{
	Use:   "learn [paths...]",
	Short: "패턴 학습",
	Long:  `프로젝트 파일에서 패턴을 학습합니다.`,
	RunE:  runConvLearn,
}

var convInitCmd = &cobra.Command{
	Use:   "init",
	Short: "기본 컨벤션 초기화",
	RunE:  runConvInit,
}

var convTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "컨벤션 타입 목록",
	RunE:  runConvTypes,
}

var convSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "컨벤션 요약",
	RunE:  runConvSummary,
}

func init() {
	rootCmd.AddCommand(conventionCmd)

	conventionCmd.AddCommand(convListCmd)
	conventionCmd.AddCommand(convShowCmd)
	conventionCmd.AddCommand(convCreateCmd)
	conventionCmd.AddCommand(convDeleteCmd)
	conventionCmd.AddCommand(convEnableCmd)
	conventionCmd.AddCommand(convDisableCmd)
	conventionCmd.AddCommand(convCheckCmd)
	conventionCmd.AddCommand(convLearnCmd)
	conventionCmd.AddCommand(convInitCmd)
	conventionCmd.AddCommand(convTypesCmd)
	conventionCmd.AddCommand(convSummaryCmd)

	convCreateCmd.Flags().BoolVar(&convEnabled, "enabled", true, "활성화 여부")
	convCreateCmd.Flags().IntVar(&convPriority, "priority", 5, "우선순위 (1-10)")

	convLearnCmd.Flags().StringSliceVar(&convFileTypes, "types", []string{".go"}, "파일 타입")
}

func getConventionService() (*convention.Service, error) {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}
	return convention.NewService(projectRoot), nil
}

func runConvList(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	conventions, err := svc.List()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(conventions)
	}

	if len(conventions) == 0 {
		fmt.Println("등록된 컨벤션이 없습니다.")
		fmt.Println("\n컨벤션 초기화:")
		fmt.Println("  pal conv init")
		return nil
	}

	fmt.Println("📋 컨벤션 목록")
	fmt.Println()

	typeEmoji := map[convention.ConventionType]string{
		convention.TypeCodingStyle:   "💻",
		convention.TypeNaming:        "📝",
		convention.TypeCommitMessage: "💬",
		convention.TypeFileStructure: "📁",
		convention.TypeDocumentation: "📚",
		convention.TypeTesting:       "🧪",
		convention.TypeErrorHandling: "⚠️",
		convention.TypeCustom:        "⚙️",
	}

	for _, conv := range conventions {
		emoji := typeEmoji[conv.Type]
		if emoji == "" {
			emoji = "📋"
		}

		status := "✅"
		if !conv.Enabled {
			status = "⚪"
		}

		fmt.Printf("%s %s %s (P%d)\n", status, emoji, conv.Name, conv.Priority)
		fmt.Printf("   ID: %s | 타입: %s | 규칙: %d개\n", conv.ID, conv.Type, len(conv.Rules))
	}

	return nil
}

func runConvShow(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	conv, err := svc.Get(args[0])
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(conv)
	}

	status := "✅ 활성"
	if !conv.Enabled {
		status = "⚪ 비활성"
	}

	fmt.Printf("📋 %s\n", conv.Name)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("ID:       %s\n", conv.ID)
	fmt.Printf("타입:     %s\n", conv.Type)
	fmt.Printf("상태:     %s\n", status)
	fmt.Printf("우선순위: %d\n", conv.Priority)
	fmt.Printf("설명:     %s\n", conv.Description)

	if len(conv.Rules) > 0 {
		fmt.Println()
		fmt.Printf("📏 규칙 (%d개)\n", len(conv.Rules))
		for _, rule := range conv.Rules {
			severityEmoji := map[string]string{
				"error":   "❌",
				"warning": "⚠️",
				"info":    "ℹ️",
			}
			emoji := severityEmoji[rule.Severity]
			if emoji == "" {
				emoji = "•"
			}
			fmt.Printf("   %s %s: %s\n", emoji, rule.ID, rule.Description)
		}
	}

	if len(conv.Examples.Good) > 0 || len(conv.Examples.Bad) > 0 {
		fmt.Println()
		fmt.Println("📝 예시")
		if len(conv.Examples.Good) > 0 {
			fmt.Println("   Good:")
			for _, ex := range conv.Examples.Good {
				fmt.Printf("     ✅ %s\n", ex.Code)
			}
		}
		if len(conv.Examples.Bad) > 0 {
			fmt.Println("   Bad:")
			for _, ex := range conv.Examples.Bad {
				fmt.Printf("     ❌ %s\n", ex.Code)
			}
		}
	}

	return nil
}

func runConvCreate(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	conv := &convention.Convention{
		ID:       args[0],
		Name:     args[1],
		Type:     convention.TypeCustom,
		Enabled:  convEnabled,
		Priority: convPriority,
	}

	if err := svc.Create(conv); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(conv)
	}

	fmt.Printf("✅ 컨벤션 생성: %s\n", conv.Name)
	fmt.Printf("   파일: %s\n", conv.FilePath)

	return nil
}

func runConvDelete(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	if err := svc.Delete(args[0]); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "deleted",
			"id":     args[0],
		})
	}

	fmt.Printf("✅ 컨벤션 삭제: %s\n", args[0])
	return nil
}

func runConvEnable(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	if err := svc.Enable(args[0]); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "enabled",
			"id":     args[0],
		})
	}

	fmt.Printf("✅ 컨벤션 활성화: %s\n", args[0])
	return nil
}

func runConvDisable(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	if err := svc.Disable(args[0]); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "disabled",
			"id":     args[0],
		})
	}

	fmt.Printf("⚪ 컨벤션 비활성화: %s\n", args[0])
	return nil
}

func runConvCheck(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	// 기본 경로
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// 파일 수집
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				// 숨김 파일 스킵
				if strings.HasPrefix(info.Name(), ".") {
					return nil
				}
				files = append(files, p)
				return nil
			})
		} else {
			files = append(files, path)
		}
	}

	results, err := svc.Check(files)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	if len(results) == 0 {
		fmt.Println("✅ 모든 컨벤션을 준수합니다!")
		return nil
	}

	fmt.Println("🔍 컨벤션 검사 결과")
	fmt.Println()

	severityEmoji := map[string]string{
		"error":   "❌",
		"warning": "⚠️",
		"info":    "ℹ️",
	}

	// 파일별로 그룹화
	byFile := make(map[string][]convention.CheckResult)
	for _, r := range results {
		byFile[r.FilePath] = append(byFile[r.FilePath], r)
	}

	for file, fileResults := range byFile {
		fmt.Printf("📄 %s\n", file)
		for _, r := range fileResults {
			emoji := severityEmoji[r.Severity]
			if r.Line > 0 {
				fmt.Printf("   %s L%d [%s] %s\n", emoji, r.Line, r.RuleID, r.Message)
			} else {
				fmt.Printf("   %s [%s] %s\n", emoji, r.RuleID, r.Message)
			}
		}
		fmt.Println()
	}

	// 요약
	summary := map[string]int{}
	for _, r := range results {
		summary[r.Severity]++
	}
	fmt.Printf("요약: ❌ %d errors, ⚠️ %d warnings, ℹ️ %d info\n",
		summary["error"], summary["warning"], summary["info"])

	return nil
}

func runConvLearn(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	// 기본 경로
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	result, err := svc.Learn(paths, convFileTypes)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Println("🎓 패턴 학습 결과")
	fmt.Printf("   스캔된 파일: %d개\n", result.FilesScanned)
	fmt.Println()

	if len(result.Patterns) > 0 {
		fmt.Println("📊 발견된 패턴:")
		for _, p := range result.Patterns {
			fmt.Printf("   • %s (%s): %d회\n", p.Pattern, p.Type, p.Occurrences)
			if len(p.Examples) > 0 {
				fmt.Printf("     예: %s\n", strings.Join(p.Examples, ", "))
			}
		}
		fmt.Println()
	}

	if len(result.Suggestions) > 0 {
		fmt.Println("💡 컨벤션 제안:")
		for _, s := range result.Suggestions {
			fmt.Printf("   • %s (%.0f%% 확신)\n", s.Name, s.Confidence*100)
			fmt.Printf("     %s\n", s.Description)
		}
		fmt.Println()
		fmt.Println("제안을 컨벤션으로 추가하려면:")
		fmt.Println("  pal conv create <id> <name>")
	}

	return nil
}

func runConvInit(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	created, err := svc.InitDefaultConventions()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "initialized",
			"created": created,
		})
	}

	fmt.Println("✅ 기본 컨벤션 초기화 완료")
	fmt.Println()

	if len(created) > 0 {
		fmt.Println("생성된 컨벤션:")
		for _, id := range created {
			fmt.Printf("   📋 %s\n", id)
		}
	} else {
		fmt.Println("  (이미 초기화됨)")
	}

	return nil
}

func runConvTypes(cmd *cobra.Command, args []string) error {
	types := convention.GetConventionTypes()

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(types)
	}

	fmt.Println("📋 컨벤션 타입")
	fmt.Println()

	descriptions := map[convention.ConventionType]string{
		convention.TypeCodingStyle:   "코딩 스타일 규칙",
		convention.TypeNaming:        "네이밍 규칙",
		convention.TypeCommitMessage: "커밋 메시지 규칙",
		convention.TypeFileStructure: "파일/디렉토리 구조",
		convention.TypeDocumentation: "문서화 규칙",
		convention.TypeTesting:       "테스트 규칙",
		convention.TypeErrorHandling: "에러 처리 규칙",
		convention.TypeCustom:        "사용자 정의",
	}

	for _, t := range types {
		fmt.Printf("  %-18s %s\n", t, descriptions[t])
	}

	return nil
}

func runConvSummary(cmd *cobra.Command, args []string) error {
	svc, err := getConventionService()
	if err != nil {
		return err
	}

	summary, err := svc.Summary()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(summary)
	}

	fmt.Println("📊 컨벤션 요약")
	fmt.Println()
	fmt.Printf("총 컨벤션: %d개\n", summary["total"])
	fmt.Printf("  ✅ 활성:   %d개\n", summary["enabled"])
	fmt.Printf("  ⚪ 비활성: %d개\n", summary["disabled"])
	fmt.Printf("  📏 규칙:   %d개\n", summary["rules"])

	return nil
}
