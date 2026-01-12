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

var (
	envWorkspace  string
	envClaudeData string
	envAutoSwitch bool
)

func init() {
	rootCmd.AddCommand(envCmd)

	envCmd.AddCommand(envSetupCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envCurrentCmd)
	envCmd.AddCommand(envSwitchCmd)
	envCmd.AddCommand(envDetectCmd)
	envCmd.AddCommand(envDeleteCmd)

	// setup flags
	defaults := env.DefaultPaths()
	envSetupCmd.Flags().StringVar(&envWorkspace, "workspace", defaults.Workspace, "작업 디렉토리 경로")
	envSetupCmd.Flags().StringVar(&envClaudeData, "claude-data", defaults.ClaudeData, "Claude 데이터 경로")

	// detect flags
	envDetectCmd.Flags().BoolVar(&envAutoSwitch, "switch", false, "감지된 환경으로 자동 전환")
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
