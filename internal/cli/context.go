package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var (
	ctxFile   string
	ctxPortID string
)

var contextCmd = &cobra.Command{
	Use:     "context",
	Aliases: []string{"ctx"},
	Short:   "컨텍스트 관리",
	Long:    `CLAUDE.md 및 에이전트 컨텍스트를 관리합니다.`,
}

var ctxShowCmd = &cobra.Command{
	Use:   "show",
	Short: "현재 컨텍스트 출력",
	RunE:  runCtxShow,
}

var ctxInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "CLAUDE.md에 컨텍스트 주입",
	Long: `CLAUDE.md 파일의 pal:context 섹션에 현재 상태를 주입합니다.

CLAUDE.md에 다음 마커가 필요합니다:
<!-- pal:context:start -->
<!-- pal:context:end -->`,
	RunE: runCtxInject,
}

var ctxForPortCmd = &cobra.Command{
	Use:   "for-port <port-id>",
	Short: "포트 기반 컨텍스트 생성",
	Args:  cobra.ExactArgs(1),
	RunE:  runCtxForPort,
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(ctxShowCmd)
	contextCmd.AddCommand(ctxInjectCmd)
	contextCmd.AddCommand(ctxForPortCmd)

	ctxInjectCmd.Flags().StringVar(&ctxFile, "file", "", "CLAUDE.md 파일 경로 (자동 탐색)")
}

func getContextService() (*context.Service, func(), error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, err
	}
	return context.NewService(database), func() { database.Close() }, nil
}

func runCtxShow(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getContextService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, err := svc.GenerateContext()
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"context": ctx,
		})
		return nil
	}

	fmt.Println("=== 현재 컨텍스트 ===")
	fmt.Println()
	fmt.Println(ctx)

	return nil
}

func runCtxInject(cmd *cobra.Command, args []string) error {
	// CLAUDE.md 파일 찾기
	filePath := ctxFile
	if filePath == "" {
		cwd, _ := os.Getwd()
		filePath = context.FindClaudeMD(cwd)
	}

	if filePath == "" {
		return fmt.Errorf("CLAUDE.md 파일을 찾을 수 없습니다. --file로 지정하세요")
	}

	svc, cleanup, err := getContextService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.InjectToFile(filePath); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "injected",
			"file":   filePath,
		})
	} else {
		fmt.Printf("✓ 컨텍스트 주입 완료: %s\n", filePath)
	}

	return nil
}

func runCtxForPort(cmd *cobra.Command, args []string) error {
	portID := args[0]

	svc, cleanup, err := getContextService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, err := svc.GenerateForPort(portID)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"port_id": portID,
			"context": ctx,
		})
		return nil
	}

	fmt.Println(ctx)

	return nil
}
