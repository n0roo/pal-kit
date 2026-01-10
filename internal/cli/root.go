package cli

import (
	"os"
	"path/filepath"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/spf13/cobra"
)

var (
	dbPath  string
	verbose bool
	jsonOut bool
)

var rootCmd = &cobra.Command{
	Use:   "pal",
	Short: "Claude Code 보조 도구",
	Long: `PAL - Claude Code 보조 도구

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
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "SQLite DB 경로 (기본: .claude/pal.db)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "상세 출력")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON 출력")
}

// GetDBPath returns the database path
func GetDBPath() string {
	if dbPath != "" {
		return dbPath
	}
	
	// 프로젝트 루트에서 .claude/pal.db 찾기
	cwd, err := os.Getwd()
	if err != nil {
		return ".claude/pal.db"
	}
	
	// 프로젝트 루트 찾기 (.claude 디렉토리가 있는 곳)
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot != "" {
		return filepath.Join(projectRoot, ".claude", "pal.db")
	}
	
	return filepath.Join(cwd, ".claude", "pal.db")
}

// IsVerbose returns verbose flag
func IsVerbose() bool {
	return verbose
}

// IsJSON returns json output flag
func IsJSON() bool {
	return jsonOut
}
