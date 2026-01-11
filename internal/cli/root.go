package cli

import (
	"github.com/n0roo/pal-kit/internal/config"
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
  - 전역 대시보드: 모든 프로젝트 세션 통합 조회
  - 세션 관리: 작업 세션 상태 추적
  - 포트 관리: 작업 단위 관리
  - 토큰 사용량: Claude 사용량 추적
  - Hook 지원: Claude Code Hook 연동

전역 설치: pal install
프로젝트 초기화: pal init`,
	Version: "0.2.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "SQLite DB 경로 (기본: ~/.pal/pal.db)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "상세 출력")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "JSON 출력")
}

// GetDBPath returns the database path (global by default)
func GetDBPath() string {
	if dbPath != "" {
		return dbPath
	}
	return config.GlobalDBPath()
}

// GetProjectRoot returns the current project root
func GetProjectRoot() string {
	return config.FindProjectRoot()
}

// IsVerbose returns verbose flag
func IsVerbose() bool {
	return verbose
}

// IsJSON returns json output flag
func IsJSON() bool {
	return jsonOut
}
