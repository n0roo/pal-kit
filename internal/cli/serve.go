package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/server"
	"github.com/spf13/cobra"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "웹 대시보드 실행",
	Long: `웹 기반 대시보드를 실행합니다.

전역 DB (~/.pal/pal.db)를 사용하여 모든 프로젝트의
세션, 포트, 히스토리를 통합 조회합니다.`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "서버 포트")
}

func runServe(cmd *cobra.Command, args []string) error {
	// 전역 설치 확인
	if !config.IsInstalled() {
		return fmt.Errorf("PAL Kit이 설치되지 않았습니다.\n먼저 'pal install' 명령을 실행하세요.")
	}

	// 전역 DB 경로 사용
	dbPath := GetDBPath()

	// 프로젝트 루트 (현재 디렉토리 또는 감지된 프로젝트)
	projectRoot := config.FindProjectRoot()

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()

	fmt.Printf("📊 전역 DB: %s\n", dbPath)
	fmt.Printf("📁 프로젝트: %s\n", projectRoot)

	return server.Run(servePort, projectRoot, dbPath)
}
