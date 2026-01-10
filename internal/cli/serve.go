package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/server"
	"github.com/spf13/cobra"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "웹 대시보드 실행",
	Long:  `웹 기반 대시보드를 실행합니다.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "서버 포트")
}

func runServe(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	dbPath := GetDBPath()

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()

	return server.Run(servePort, projectRoot, dbPath)
}
