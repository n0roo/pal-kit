package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/session"
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

var ctxStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "컨텍스트 예산 상태",
	Long: `현재 컨텍스트의 토큰 예산 사용량을 표시합니다.

출력 정보:
- 총 토큰 예산 및 사용량
- 카테고리별 할당/사용량
- 로드된 문서 목록`,
	RunE: runCtxStatus,
}

var ctxCheckpointsCmd = &cobra.Command{
	Use:   "checkpoints",
	Short: "체크포인트 목록",
	Long:  `저장된 체크포인트 목록을 표시합니다.`,
	RunE:  runCtxCheckpoints,
}

var ctxRestoreCmd = &cobra.Command{
	Use:   "restore <checkpoint-id>",
	Short: "체크포인트에서 복구",
	Long:  `지정한 체크포인트에서 컨텍스트를 복구합니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCtxRestore,
}

var ctxCreateCheckpointCmd = &cobra.Command{
	Use:   "checkpoint",
	Short: "체크포인트 생성",
	Long:  `현재 컨텍스트의 체크포인트를 생성합니다.`,
	RunE:  runCtxCreateCheckpoint,
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(ctxShowCmd)
	contextCmd.AddCommand(ctxInjectCmd)
	contextCmd.AddCommand(ctxForPortCmd)
	contextCmd.AddCommand(ctxReloadCmd)
	contextCmd.AddCommand(ctxClaudeCmd)
	contextCmd.AddCommand(ctxStatusCmd)
	contextCmd.AddCommand(ctxCheckpointsCmd)
	contextCmd.AddCommand(ctxRestoreCmd)
	contextCmd.AddCommand(ctxCreateCheckpointCmd)

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

func runCtxStatus(cmd *cobra.Command, args []string) error {
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

	// BudgetService를 통해 현재 상태 가져오기
	budgetSvc := context.NewBudgetService(database, projectRoot)
	report, err := budgetSvc.GetCurrentStatus()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(report)
	}

	// 헤더
	fmt.Printf("Context Budget: %s / %s tokens (%d%%)\n",
		formatTokenCount(report.Used),
		formatTokenCount(report.Total),
		report.UsagePercent)
	fmt.Println()

	// 로드된 문서
	fmt.Println("Loaded Documents:")
	for _, item := range report.Items {
		icon := getCategoryIconCLI(item.Category)
		status := "✓"
		if !item.Loaded {
			status = "(pending)"
		}
		fmt.Printf("  %s %s (%s)  %s %s\n",
			icon, item.Name, item.Category,
			formatTokenCount(item.Tokens), status)
	}
	fmt.Println()

	// 카테고리별 상세
	fmt.Println("Category Allocation:")
	for _, cat := range report.CategoryDetail {
		percent := 0.0
		if cat.Allocated > 0 {
			percent = float64(cat.Used) / float64(cat.Allocated) * 100
		}
		bar := renderProgressBar(percent, 10)
		fmt.Printf("  %-15s %s %s / %s\n",
			cat.Category, bar,
			formatTokenCount(cat.Used),
			formatTokenCount(cat.Allocated))
	}

	return nil
}

// formatTokenCount formats token count with K suffix
func formatTokenCount(tokens int) string {
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}

// getCategoryIconCLI returns an emoji icon for a category
func getCategoryIconCLI(category string) string {
	switch category {
	case context.CategoryPortSpec:
		return "📄"
	case context.CategoryConventions:
		return "📘"
	case context.CategoryRecentChanges:
		return "📝"
	case context.CategoryRelatedDocs:
		return "📚"
	case context.CategorySessionInfo:
		return "ℹ️"
	default:
		return "📁"
	}
}

func runCtxCheckpoints(cmd *cobra.Command, args []string) error {
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

	cpSvc := context.NewCheckpointService(database, projectRoot)
	checkpoints, err := cpSvc.ListCheckpoints("", 10)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(checkpoints)
	}

	if len(checkpoints) == 0 {
		fmt.Println("저장된 체크포인트가 없습니다.")
		return nil
	}

	fmt.Println("체크포인트 목록:")
	fmt.Println()
	for _, cp := range checkpoints {
		portInfo := "-"
		if cp.ActivePort != nil {
			portInfo = cp.ActivePort.ID
			if cp.ActivePort.Title != "" {
				portInfo += " (" + cp.ActivePort.Title + ")"
			}
		}
		ago := formatTimeAgoCLI(cp.CreatedAt)
		fmt.Printf("  %s  %s\n", cp.ID, ago)
		fmt.Printf("    세션: %s\n", cp.SessionID[:8])
		fmt.Printf("    포트: %s\n", portInfo)
		fmt.Printf("    토큰: %s\n", formatTokenCount(cp.TokensUsed))
		fmt.Println()
	}

	return nil
}

func runCtxRestore(cmd *cobra.Command, args []string) error {
	checkpointID := args[0]

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

	cpSvc := context.NewCheckpointService(database, projectRoot)
	cp, err := cpSvc.RestoreCheckpoint(checkpointID)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":     "restored",
			"checkpoint": cp,
		})
	}

	fmt.Printf("✓ 체크포인트 복구 완료: %s\n", cp.ID)
	fmt.Println()
	fmt.Println(cp.RecoveryPrompt)

	return nil
}

func runCtxCreateCheckpoint(cmd *cobra.Command, args []string) error {
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

	// 활성 세션 찾기
	sessionSvc := session.NewService(database)
	activeSession, err := sessionSvc.FindActiveSession("", cwd, projectRoot)
	if err != nil {
		return fmt.Errorf("활성 세션을 찾을 수 없습니다: %w", err)
	}
	if activeSession == nil {
		return fmt.Errorf("활성 세션이 없습니다")
	}

	cpSvc := context.NewCheckpointService(database, projectRoot)
	cp, err := cpSvc.CreateCheckpoint(activeSession.ID)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":     "created",
			"checkpoint": cp,
		})
	}

	fmt.Printf("✓ 체크포인트 생성 완료: %s\n", cp.ID)
	if cp.ActivePort != nil {
		fmt.Printf("  포트: %s\n", cp.ActivePort.ID)
	}
	fmt.Printf("  토큰: %s\n", formatTokenCount(cp.TokensUsed))

	return nil
}

// formatTimeAgoCLI formats time as "X minutes ago" etc
func formatTimeAgoCLI(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "방금 전"
	} else if d < time.Hour {
		return fmt.Sprintf("%d분 전", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d시간 전", int(d.Hours()))
	} else {
		return fmt.Sprintf("%d일 전", int(d.Hours()/24))
	}
}
