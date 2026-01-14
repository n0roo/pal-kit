package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "버전 정보 출력",
	Long:  `PAL Kit 버전 및 빌드 정보를 출력합니다.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	info := map[string]interface{}{
		"version": Version,
		"commit":  Commit,
		"date":    Date,
		"go":      runtime.Version(),
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
	}

	// 설치 상태 확인
	if config.IsInstalled() {
		info["installed"] = true
		info["global_db"] = config.GlobalDBPath()
	} else {
		info["installed"] = false
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(info)
		return
	}

	fmt.Printf("PAL Kit %s\n", Version)
	fmt.Println()
	fmt.Printf("  Commit:    %s\n", Commit)
	fmt.Printf("  Built:     %s\n", Date)
	fmt.Printf("  Go:        %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:   %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Println()

	if config.IsInstalled() {
		fmt.Printf("  Installed: ✅\n")
		fmt.Printf("  Global DB: %s\n", config.GlobalDBPath())
	} else {
		fmt.Printf("  Installed: ❌ (run 'pal install' first)\n")
	}
}
