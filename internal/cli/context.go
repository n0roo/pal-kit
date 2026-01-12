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

var ctxReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "컨텍스트 새로고침",
	Long:  `현재 활성 워커의 컨텍스트를 새로고침합니다.`,
	RunE:  runCtxReload,
}

var ctxClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Claude 통합 컨텍스트",
	Long: `Claude Code와 연동되는 컨텍스트를 표시합니다.

컨텍스트 로딩 순서:
1. CLAUDE.md (프로젝트 기본 정보)
2. 패키지 컨벤션 (architecture.md)
3. 워커 공통 컨벤션 (_common.md)
4. 워커 개별 컨벤션 ({worker}.md)
5. 포트 명세 (ports/{port-id}.md)
6. 워커 프롬프트 (agents/{worker}.yaml → prompt)`,
	RunE: runCtxClaude,
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(ctxShowCmd)
	contextCmd.AddCommand(ctxInjectCmd)
	contextCmd.AddCommand(ctxForPortCmd)
	contextCmd.AddCommand(ctxReloadCmd)
	contextCmd.AddCommand(ctxClaudeCmd)

	ctxInjectCmd.Flags().StringVar(&ctxFile, "file", "", "CLAUDE.md 파일 경로 (자동 탐색)")
	ctxClaudeCmd.Flags().StringVar(&ctxPortID, "port", "", "포트 ID")
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

func runCtxReload(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	claudeSvc := context.NewClaudeService(database, projectRoot)

	result, err := claudeSvc.ReloadContext()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Printf("🔄 컨텍스트 새로고침 완료\n")
	fmt.Printf("   워커: %s\n", result.WorkerID)
	fmt.Printf("   토큰: ~%d\n", result.TokenCount)
	if len(result.Checklist) > 0 {
		fmt.Printf("   체크리스트: %d 항목\n", len(result.Checklist))
	}

	return nil
}

func runCtxClaude(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	claudeSvc := context.NewClaudeService(database, projectRoot)

	ctx, err := claudeSvc.GetCurrentContext(ctxPortID, "")
	if err != nil {
		return err
	}

	if jsonOut {
		output := map[string]interface{}{
			"context": ctx,
			"port_id": ctxPortID,
		}
		return json.NewEncoder(os.Stdout).Encode(output)
	}

	fmt.Println(ctx)
	return nil
}
