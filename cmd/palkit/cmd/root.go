package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath  string
	verbose bool
	jsonOut bool
)

var rootCmd = &cobra.Command{
	Use:   "palkit",
	Short: "Claude Code 보조 도구",
	Long: `PAL Kit - Claude Code 보조 도구

에이전트가 컨벤션을 준수하고 작업 품질을 유지하도록 지원합니다.

주요 기능:
  - Lock 관리: 리소스 동시 접근 제어
  - 세션 관리: 작업 세션 상태 추적
  - 포트 관리: 작업 단위 관리
  - 토큰 사용량: Claude 사용량 추적
  - Hook 지원: Claude Code Hook 연동`,
	Version: "0.1.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "SQLite DB 경로 (기본: .claude/palkit.db)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "상세 출력")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON 출력")
}

// GetDBPath returns the database path
func GetDBPath() string {
	if dbPath != "" {
		return dbPath
	}
	
	// 기본 경로: .claude/palkit.db
	cwd, err := os.Getwd()
	if err != nil {
		return ".claude/palkit.db"
	}
	return fmt.Sprintf("%s/.claude/palkit.db", cwd)
}

// IsVerbose returns verbose flag
func IsVerbose() bool {
	return verbose
}

// IsJSON returns json output flag
func IsJSON() bool {
	return jsonOut
}
