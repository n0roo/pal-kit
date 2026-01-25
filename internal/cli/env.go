package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/env"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "환경 프로파일 관리",
	Long: `다중 접속 환경(집, 회사 등)의 프로파일을 관리합니다.

환경 프로파일을 통해 각 PC에서 동일한 작업 데이터를 공유할 수 있습니다.
경로는 논리 변수($workspace, $claude_data 등)로 추상화되어 저장됩니다.`,
}

var envSetupCmd = &cobra.Command{
	Use:   "setup [name]",
	Short: "현재 환경 등록",
	Long: `현재 PC를 새 환경으로 등록하거나 기존 환경을 업데이트합니다.

이름을 지정하지 않으면 hostname을 기본값으로 사용합니다.
경로는 --workspace, --claude-data 플래그로 지정하거나 기본값을 사용합니다.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEnvSetup,
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "등록된 환경 목록",
	RunE:  runEnvList,
}

var envCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "현재 환경 표시",
	RunE:  runEnvCurrent,
}

var envSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "환경 전환",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvSwitch,
}

var envDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "환경 자동 감지",
	Long:  `hostname 또는 경로 존재 여부를 기반으로 현재 환경을 자동 감지합니다.`,
	RunE:  runEnvDetect,
}

var envDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "환경 삭제",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvDelete,
}

var envShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "환경 상세 정보",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runEnvShow,
}

var envAddProjectCmd = &cobra.Command{
	Use:   "add-project <project-id>",
	Short: "프로젝트 경로 추가",
	Long:  `현재 환경에 프로젝트 ID와 경로 매핑을 추가합니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvAddProject,
}

var envRemoveProjectCmd = &cobra.Command{
	Use:   "remove-project <project-id>",
	Short: "프로젝트 제거",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvRemoveProject,
}

var envAddDocsCmd = &cobra.Command{
	Use:   "add-docs <docs-id>",
	Short: "Docs vault 추가",
	Long:  `현재 환경에 Docs vault ID와 경로 매핑을 추가합니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvAddDocs,
}

var envRemoveDocsCmd = &cobra.Command{
	Use:   "remove-docs <docs-id>",
	Short: "Docs vault 제거",
	Args:  cobra.ExactArgs(1),
	RunE:  runEnvRemoveDocs,
}

var envLinkCmd = &cobra.Command{
	Use:   "link <project-id> <docs-id>",
	Short: "프로젝트와 Docs 연결",
	Long:  `프로젝트를 Docs vault에 연결합니다. 연결 후 apply 시 settings.local.json에 반영됩니다.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runEnvLink,
}

var envApplyCmd = &cobra.Command{
	Use:   "apply [project-id]",
	Short: "settings.local.json 생성",
	Long: `프로젝트의 .claude/settings.local.json을 생성합니다.
project-id를 생략하면 모든 프로젝트에 적용합니다.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEnvApply,
}

var (
	envWorkspace   string
	envClaudeData  string
	envAutoSwitch  bool
	envProjectPath string
	envProjectDocs string
	envDocsPath    string
)

func init() {
	rootCmd.AddCommand(envCmd)

	envCmd.AddCommand(envSetupCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envCurrentCmd)
	envCmd.AddCommand(envSwitchCmd)
	envCmd.AddCommand(envDetectCmd)
	envCmd.AddCommand(envDeleteCmd)
	envCmd.AddCommand(envShowCmd)
	envCmd.AddCommand(envAddProjectCmd)
	envCmd.AddCommand(envRemoveProjectCmd)
	envCmd.AddCommand(envAddDocsCmd)
	envCmd.AddCommand(envRemoveDocsCmd)
	envCmd.AddCommand(envLinkCmd)
	envCmd.AddCommand(envApplyCmd)

	// setup flags
	defaults := env.DefaultPaths()
	envSetupCmd.Flags().StringVar(&envWorkspace, "workspace", defaults.Workspace, "작업 디렉토리 경로")
	envSetupCmd.Flags().StringVar(&envClaudeData, "claude-data", defaults.ClaudeData, "Claude 데이터 경로")

	// detect flags
	envDetectCmd.Flags().BoolVar(&envAutoSwitch, "switch", false, "감지된 환경으로 자동 전환")

	// add-project flags
	envAddProjectCmd.Flags().StringVarP(&envProjectPath, "path", "p", "", "프로젝트 경로 (필수)")
	envAddProjectCmd.Flags().StringVarP(&envProjectDocs, "docs", "d", "", "연결할 Docs ID")
	envAddProjectCmd.MarkFlagRequired("path")

	// add-docs flags
	envAddDocsCmd.Flags().StringVarP(&envDocsPath, "path", "p", "", "Vault 경로 (필수)")
	envAddDocsCmd.MarkFlagRequired("path")
}

func runEnvSetup(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)

	name := env.SuggestName()
	if len(args) > 0 {
		name = args[0]
	}

	// Expand paths
	home, _ := os.UserHomeDir()
	workspace := expandPath(envWorkspace, home)
	claudeData := expandPath(envClaudeData, home)

	paths := env.PathVariables{
		Workspace:  workspace,
		ClaudeData: claudeData,
		Home:       home,
	}

	result, err := svc.Setup(name, paths)
	if err != nil {
		return fmt.Errorf("환경 설정 실패: %w", err)
	}

	if IsJSON() {
		return printJSON(result)
	}

	fmt.Printf("환경 등록 완료: %s\n", result.Name)
	fmt.Printf("  ID: %s\n", result.ID)
	fmt.Printf("  Hostname: %s\n", result.Hostname)
	fmt.Printf("  Workspace: %s\n", result.Paths.Workspace)
	fmt.Printf("  Claude Data: %s\n", result.Paths.ClaudeData)
	if result.IsCurrent {
		fmt.Printf("  상태: 현재 환경 (활성)\n")
	}

	return nil
}

func runEnvList(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)

	envs, err := svc.List()
	if err != nil {
		return fmt.Errorf("환경 목록 조회 실패: %w", err)
	}

	if len(envs) == 0 {
		fmt.Println("등록된 환경이 없습니다. 'pal env setup'으로 환경을 등록하세요.")
		return nil
	}

	if IsJSON() {
		return printJSON(envs)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tID\tHOSTNAME\tWORKSPACE\tCURRENT")
	fmt.Fprintln(w, "----\t--\t--------\t---------\t-------")

	for _, e := range envs {
		current := ""
		if e.IsCurrent {
			current = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.Name, e.ID, e.Hostname, e.Paths.Workspace, current)
	}
	w.Flush()

	return nil
}

func runEnvCurrent(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)

	current, err := svc.Current()
	if err != nil {
		fmt.Println("현재 환경이 설정되지 않았습니다. 'pal env setup'으로 환경을 등록하세요.")
		return nil
	}

	if IsJSON() {
		return printJSON(current)
	}

	fmt.Printf("현재 환경: %s\n", current.Name)
	fmt.Printf("  ID: %s\n", current.ID)
	fmt.Printf("  Hostname: %s\n", current.Hostname)
	fmt.Printf("  경로:\n")
	fmt.Printf("    $workspace  → %s\n", current.Paths.Workspace)
	fmt.Printf("    $claude_data → %s\n", current.Paths.ClaudeData)
	fmt.Printf("    $home       → %s\n", current.Paths.Home)

	return nil
}

func runEnvSwitch(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)
	name := args[0]

	if err := svc.Switch(name); err != nil {
		return fmt.Errorf("환경 전환 실패: %w", err)
	}

	fmt.Printf("환경 전환 완료: %s\n", name)

	// Auto-regenerate settings.local.json for all projects
	paths, err := svc.GenerateAllSettingsLocal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "경고: settings.local.json 생성 실패: %v\n", err)
	} else if len(paths) > 0 {
		fmt.Println("\nsettings.local.json 업데이트:")
		for _, p := range paths {
			fmt.Printf("  ✓ %s\n", p)
		}
	}

	return nil
}

func runEnvDetect(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)

	if envAutoSwitch {
		detected, err := svc.AutoSwitch()
		if err != nil {
			return fmt.Errorf("환경 자동 전환 실패: %w", err)
		}
		fmt.Printf("환경 감지 및 전환 완료: %s\n", detected.Name)
		return nil
	}

	detected, err := svc.Detect()
	if err != nil {
		hostname, _ := os.Hostname()
		fmt.Printf("환경 감지 실패 (현재 hostname: %s)\n", hostname)
		fmt.Println("'pal env setup'으로 새 환경을 등록하거나 'pal env switch'로 수동 전환하세요.")
		return nil
	}

	if IsJSON() {
		return printJSON(detected)
	}

	fmt.Printf("감지된 환경: %s\n", detected.Name)
	if !detected.IsCurrent {
		fmt.Printf("  (현재 활성 환경 아님, 'pal env detect --switch'로 전환 가능)\n")
	}

	return nil
}

func runEnvDelete(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)
	name := args[0]

	if err := svc.Delete(name); err != nil {
		return fmt.Errorf("환경 삭제 실패: %w", err)
	}

	fmt.Printf("환경 삭제 완료: %s\n", name)
	return nil
}

func runEnvShow(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)

	var target *env.Environment
	if len(args) > 0 {
		target, err = svc.Get(args[0])
	} else {
		target, err = svc.Current()
	}
	if err != nil {
		return fmt.Errorf("환경 조회 실패: %w", err)
	}

	if IsJSON() {
		return printJSON(target)
	}

	fmt.Printf("환경: %s", target.Name)
	if target.IsCurrent {
		fmt.Printf(" (현재)")
	}
	fmt.Println()
	fmt.Printf("  ID: %s\n", target.ID)
	fmt.Printf("  Hostname: %s\n", target.Hostname)
	fmt.Printf("\n기본 경로:\n")
	fmt.Printf("  $workspace   → %s\n", target.Paths.Workspace)
	fmt.Printf("  $claude_data → %s\n", target.Paths.ClaudeData)
	fmt.Printf("  $home        → %s\n", target.Paths.Home)

	if len(target.Docs) > 0 {
		fmt.Printf("\nDocs Vaults:\n")
		for id, vault := range target.Docs {
			fmt.Printf("  %s → %s\n", id, vault.Path)
		}
	}

	if len(target.Projects) > 0 {
		fmt.Printf("\n프로젝트:\n")
		for id, proj := range target.Projects {
			docsInfo := ""
			if proj.DocsRef != "" {
				docsInfo = fmt.Sprintf(" (docs: %s)", proj.DocsRef)
			}
			fmt.Printf("  %s → %s%s\n", id, proj.Path, docsInfo)
		}
	}

	return nil
}

func runEnvAddProject(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)
	projectID := args[0]

	if err := svc.AddProject(projectID, envProjectPath, envProjectDocs); err != nil {
		return fmt.Errorf("프로젝트 추가 실패: %w", err)
	}

	fmt.Printf("프로젝트 추가 완료: %s → %s\n", projectID, envProjectPath)
	if envProjectDocs != "" {
		fmt.Printf("  연결된 Docs: %s\n", envProjectDocs)
	}

	return nil
}

func runEnvRemoveProject(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)
	projectID := args[0]

	if err := svc.RemoveProject(projectID); err != nil {
		return fmt.Errorf("프로젝트 제거 실패: %w", err)
	}

	fmt.Printf("프로젝트 제거 완료: %s\n", projectID)
	return nil
}

func runEnvAddDocs(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)
	docsID := args[0]

	if err := svc.AddDocs(docsID, envDocsPath); err != nil {
		return fmt.Errorf("Docs 추가 실패: %w", err)
	}

	fmt.Printf("Docs vault 추가 완료: %s → %s\n", docsID, envDocsPath)
	return nil
}

func runEnvRemoveDocs(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)
	docsID := args[0]

	if err := svc.RemoveDocs(docsID); err != nil {
		return fmt.Errorf("Docs 제거 실패: %w", err)
	}

	fmt.Printf("Docs vault 제거 완료: %s\n", docsID)
	return nil
}

func runEnvLink(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)
	projectID := args[0]
	docsID := args[1]

	if err := svc.LinkProjectToDocs(projectID, docsID); err != nil {
		return fmt.Errorf("연결 실패: %w", err)
	}

	fmt.Printf("연결 완료: %s ↔ %s\n", projectID, docsID)
	fmt.Println("'pal env apply'로 settings.local.json을 생성하세요.")
	return nil
}

func runEnvApply(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	svc := env.NewService(database)

	if len(args) > 0 {
		// Apply to specific project
		projectID := args[0]
		path, err := svc.GenerateSettingsLocal(projectID)
		if err != nil {
			return fmt.Errorf("설정 생성 실패: %w", err)
		}
		fmt.Printf("생성 완료: %s\n", path)
		return nil
	}

	// Apply to all projects
	paths, err := svc.GenerateAllSettingsLocal()
	if err != nil {
		return fmt.Errorf("설정 생성 실패: %w", err)
	}

	if len(paths) == 0 {
		fmt.Println("등록된 프로젝트가 없습니다.")
		return nil
	}

	fmt.Println("생성 완료:")
	for _, p := range paths {
		fmt.Printf("  %s\n", p)
	}
	return nil
}

// Helper functions

func expandPath(path, home string) string {
	if strings.HasPrefix(path, "~") {
		return strings.Replace(path, "~", home, 1)
	}
	return path
}

func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
