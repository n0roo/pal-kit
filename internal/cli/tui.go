package cli

import (
	"os"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "대시보드 TUI 실행",
	Long:  `터미널 기반 대시보드를 실행합니다.`,
	RunE:  runTui,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTui(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	dbPath := GetDBPath()

	return tui.Run(projectRoot, dbPath)
}
