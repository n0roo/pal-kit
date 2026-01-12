package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/env"
	"github.com/n0roo/pal-kit/internal/sync"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "데이터 동기화 관리",
	Long: `다중 환경 간 PAL Kit 데이터를 동기화합니다.

Export/Import를 통해 세션, 포트, 에스컬레이션 등의 데이터를
다른 환경과 공유할 수 있습니다.`,
}

var syncExportCmd = &cobra.Command{
	Use:   "export [output-file]",
	Short: "데이터 내보내기",
	Long: `현재 환경의 데이터를 YAML 파일로 내보냅니다.

내보내는 데이터:
- 포트 (ports)
- 세션 (sessions) - 로컬 전용 필드 제외
- 에스컬레이션 (escalations)
- 파이프라인 (pipelines)
- 프로젝트 (projects)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSyncExport,
}

var syncImportCmd = &cobra.Command{
	Use:   "import <input-file>",
	Short: "데이터 가져오기",
	Long: `YAML 파일에서 데이터를 가져옵니다.

Merge 전략:
- last_write_wins: 최신 데이터 우선 (기본)
- keep_local: 로컬 데이터 유지
- keep_remote: 원격 데이터 우선
- manual: 충돌 시 건너뛰기`,
	Args: cobra.ExactArgs(1),
	RunE: runSyncImport,
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "동기화 상태 확인",
	RunE:  runSyncStatus,
}

var syncInitCmd = &cobra.Command{
	Use:   "init [remote-url]",
	Short: "Git 동기화 초기화",
	Long: `동기화 디렉토리를 Git 저장소로 초기화합니다.

원격 저장소 URL을 지정하면 자동으로 연결됩니다.
예: pal sync init git@github.com:user/pal-sync.git`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSyncInit,
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "변경사항 푸시",
	Long: `현재 데이터를 내보내고 Git 원격 저장소에 푸시합니다.

1. 로컬 DB → YAML 내보내기
2. Git add, commit
3. Git push (원격 설정 시)`,
	RunE: runSyncPush,
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "변경사항 풀",
	Long: `Git 원격 저장소에서 풀하고 데이터를 가져옵니다.

1. Git pull
2. YAML → 로컬 DB 가져오기
3. 충돌 감지 및 알림`,
	RunE: runSyncPull,
}

var syncConflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "충돌 목록 확인",
	Long: `현재 대기 중인 동기화 충돌 목록을 표시합니다.

충돌 해결:
  pal sync resolve --all --keep-local    # 모든 충돌을 로컬 데이터로 해결
  pal sync resolve --all --keep-remote   # 모든 충돌을 원격 데이터로 해결
  pal sync resolve <id> --keep-local     # 특정 충돌 해결`,
	RunE: runSyncConflicts,
}

var syncResolveCmd = &cobra.Command{
	Use:   "resolve [id]",
	Short: "충돌 해결",
	Long: `동기화 충돌을 해결합니다.

해결 방법:
  --keep-local   로컬 데이터 유지
  --keep-remote  원격 데이터로 덮어쓰기
  --skip         충돌 건너뛰기

예시:
  pal sync resolve --all --keep-local
  pal sync resolve abc123 --keep-remote`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSyncResolve,
}

var syncDiffCmd = &cobra.Command{
	Use:   "diff [file]",
	Short: "로컬과 원격 데이터 비교",
	Long: `로컬 데이터와 동기화 파일의 차이점을 비교합니다.

파일을 지정하지 않으면 가장 최근 동기화 파일과 비교합니다.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSyncDiff,
}

var (
	syncOutputFile    string
	syncStrategy      string
	syncDryRun        bool
	syncSkipConflicts bool
	syncMessage       string
	syncResolveAll    bool
	syncKeepLocal     bool
	syncKeepRemote    bool
	syncSkipResolve   bool
	syncConflictType  string
)

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.AddCommand(syncExportCmd)
	syncCmd.AddCommand(syncImportCmd)
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncInitCmd)
	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncConflictsCmd)
	syncCmd.AddCommand(syncResolveCmd)
	syncCmd.AddCommand(syncDiffCmd)

	// Export flags
	syncExportCmd.Flags().StringVarP(&syncOutputFile, "output", "o", "", "출력 파일 경로")

	// Import flags
	syncImportCmd.Flags().StringVar(&syncStrategy, "strategy", "last_write_wins",
		"Merge 전략 (last_write_wins, keep_local, keep_remote, manual)")
	syncImportCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "실제 변경 없이 결과만 표시")
	syncImportCmd.Flags().BoolVar(&syncSkipConflicts, "skip-conflicts", false, "충돌 시 건너뛰기")

	// Push flags
	syncPushCmd.Flags().StringVarP(&syncMessage, "message", "m", "", "커밋 메시지")

	// Pull flags
	syncPullCmd.Flags().StringVar(&syncStrategy, "strategy", "last_write_wins",
		"Merge 전략 (last_write_wins, keep_local, keep_remote, manual)")
	syncPullCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "실제 변경 없이 결과만 표시")
	syncPullCmd.Flags().BoolVar(&syncSkipConflicts, "skip-conflicts", false, "충돌 시 건너뛰기")

	// Resolve flags
	syncResolveCmd.Flags().BoolVar(&syncResolveAll, "all", false, "모든 충돌 해결")
	syncResolveCmd.Flags().BoolVar(&syncKeepLocal, "keep-local", false, "로컬 데이터 유지")
	syncResolveCmd.Flags().BoolVar(&syncKeepRemote, "keep-remote", false, "원격 데이터 사용")
	syncResolveCmd.Flags().BoolVar(&syncSkipResolve, "skip", false, "충돌 건너뛰기")
	syncResolveCmd.Flags().StringVar(&syncConflictType, "type", "", "충돌 유형 (port, session, escalation, pipeline, project)")
}

func runSyncExport(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)
	exporter := sync.NewExporter(database, envSvc)

	// Determine output file
	outputPath := syncOutputFile
	if len(args) > 0 {
		outputPath = args[0]
	}
	if outputPath == "" {
		// Default: ~/.pal/sync/pal-sync-{timestamp}.yaml
		syncDir := filepath.Join(config.GlobalDir(), "sync")
		os.MkdirAll(syncDir, 0755)
		outputPath = filepath.Join(syncDir, fmt.Sprintf("pal-sync-%s.yaml", time.Now().Format("20060102-150405")))
	}

	// Export
	data, err := exporter.ExportAll()
	if err != nil {
		return fmt.Errorf("export 실패: %w", err)
	}

	if IsJSON() {
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	// Write to file
	if err := exporter.ExportToFile(outputPath); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	fmt.Printf("Export 완료: %s\n", outputPath)
	fmt.Printf("\n통계:\n")
	fmt.Printf("  포트: %d\n", data.Manifest.Stats.PortsCount)
	fmt.Printf("  세션: %d\n", data.Manifest.Stats.SessionsCount)
	fmt.Printf("  에스컬레이션: %d\n", data.Manifest.Stats.EscalationsCount)
	fmt.Printf("  파이프라인: %d\n", data.Manifest.Stats.PipelinesCount)
	fmt.Printf("  프로젝트: %d\n", data.Manifest.Stats.ProjectsCount)
	fmt.Printf("\n내보낸 환경: %s (%s)\n", data.Manifest.ExportedBy, data.Manifest.ExportedEnv)

	return nil
}

func runSyncImport(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)

	// Parse strategy
	var strategy sync.MergeStrategy
	switch syncStrategy {
	case "last_write_wins":
		strategy = sync.MergeStrategyLastWriteWins
	case "keep_local":
		strategy = sync.MergeStrategyKeepLocal
	case "keep_remote":
		strategy = sync.MergeStrategyKeepRemote
	case "manual":
		strategy = sync.MergeStrategyManual
	default:
		return fmt.Errorf("알 수 없는 전략: %s", syncStrategy)
	}

	options := sync.ImportOptions{
		Strategy:      strategy,
		DryRun:        syncDryRun,
		SkipConflicts: syncSkipConflicts,
	}

	importer := sync.NewImporter(database, envSvc, options)

	inputPath := args[0]
	result, err := importer.ImportFromFile(inputPath)
	if err != nil {
		return fmt.Errorf("import 실패: %w", err)
	}

	if IsJSON() {
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	// Display result
	if syncDryRun {
		fmt.Println("[ Dry Run 모드 - 실제 변경 없음 ]")
		fmt.Println()
	}

	fmt.Println("Import 결과:")
	fmt.Printf("\n가져온 항목:\n")
	fmt.Printf("  포트: %d\n", result.Imported.Ports)
	fmt.Printf("  세션: %d\n", result.Imported.Sessions)
	fmt.Printf("  에스컬레이션: %d\n", result.Imported.Escalations)
	fmt.Printf("  파이프라인: %d\n", result.Imported.Pipelines)
	fmt.Printf("  프로젝트: %d\n", result.Imported.Projects)

	if result.Skipped.Ports > 0 || result.Skipped.Sessions > 0 ||
		result.Skipped.Escalations > 0 || result.Skipped.Pipelines > 0 ||
		result.Skipped.Projects > 0 {
		fmt.Printf("\n건너뛴 항목:\n")
		fmt.Printf("  포트: %d\n", result.Skipped.Ports)
		fmt.Printf("  세션: %d\n", result.Skipped.Sessions)
		fmt.Printf("  에스컬레이션: %d\n", result.Skipped.Escalations)
		fmt.Printf("  파이프라인: %d\n", result.Skipped.Pipelines)
		fmt.Printf("  프로젝트: %d\n", result.Skipped.Projects)
	}

	if len(result.Conflicts) > 0 {
		fmt.Printf("\n충돌 항목: %d개\n", len(result.Conflicts))
		for _, c := range result.Conflicts {
			fmt.Printf("  - [%s] %s\n", c.Type, c.ID)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n오류:\n")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	if result.Success {
		fmt.Println("\nImport 완료!")
	} else {
		fmt.Println("\nImport 완료 (일부 오류 발생)")
	}

	return nil
}

func runSyncStatus(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)

	// Get current environment
	currentEnv, err := envSvc.Current()
	if err != nil {
		fmt.Println("현재 환경이 설정되지 않았습니다.")
		fmt.Println("'pal env setup'으로 환경을 등록하세요.")
		return nil
	}

	fmt.Printf("현재 환경: %s (%s)\n", currentEnv.Name, currentEnv.ID)
	fmt.Println()

	// Check Git status
	gitSync := sync.NewGitSync(database, envSvc)
	gitStatus, err := gitSync.GetStatus()
	if err == nil && gitStatus.Initialized {
		fmt.Println("Git 저장소:")
		fmt.Printf("  디렉토리: %s\n", gitStatus.SyncDir)
		if gitStatus.Remote != "" {
			fmt.Printf("  원격: %s\n", gitStatus.Remote)
		} else {
			fmt.Println("  원격: 미설정")
		}
		if gitStatus.Branch != "" {
			fmt.Printf("  브랜치: %s\n", gitStatus.Branch)
		}
		if gitStatus.HasLocalChanges {
			fmt.Println("  상태: 로컬 변경사항 있음")
		}
		if gitStatus.Ahead > 0 || gitStatus.Behind > 0 {
			fmt.Printf("  동기화: %d ahead, %d behind\n", gitStatus.Ahead, gitStatus.Behind)
		}
		fmt.Println()
	}

	// Check sync directory
	syncDir := filepath.Join(config.GlobalDir(), "sync")
	if _, err := os.Stat(syncDir); os.IsNotExist(err) {
		fmt.Println("동기화 디렉토리가 없습니다.")
		fmt.Println("'pal sync init'으로 초기화하세요.")
		return nil
	}

	// List sync files
	files, err := filepath.Glob(filepath.Join(syncDir, "*.yaml"))
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("내보낸 파일이 없습니다.")
		return nil
	}

	fmt.Printf("내보낸 파일 (%d개):\n", len(files))
	for _, f := range files {
		info, _ := os.Stat(f)
		fmt.Printf("  - %s (%s)\n", filepath.Base(f), info.ModTime().Format("2006-01-02 15:04:05"))
	}

	// Get sync history from DB
	rows, err := database.Query(`
		SELECT direction, env_id, items_count, conflicts_count, synced_at
		FROM sync_history
		ORDER BY synced_at DESC
		LIMIT 5
	`)
	if err == nil {
		defer rows.Close()

		fmt.Println("\n최근 동기화 기록:")
		hasHistory := false
		for rows.Next() {
			hasHistory = true
			var direction, envID string
			var itemsCount, conflictsCount int
			var syncedAt time.Time

			rows.Scan(&direction, &envID, &itemsCount, &conflictsCount, &syncedAt)
			fmt.Printf("  - [%s] %s: %d 항목, %d 충돌 (%s)\n",
				direction, envID, itemsCount, conflictsCount,
				syncedAt.Format("2006-01-02 15:04:05"))
		}

		if !hasHistory {
			fmt.Println("  기록 없음")
		}
	}

	return nil
}

func runSyncInit(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)
	gitSync := sync.NewGitSync(database, envSvc)

	// Check if already initialized
	if gitSync.IsInitialized() {
		if gitSync.HasRemote() {
			remote, _ := gitSync.GetRemote()
			fmt.Printf("이미 초기화됨: %s\n", gitSync.SyncDir())
			fmt.Printf("원격 저장소: %s\n", remote)
			return nil
		}
		fmt.Printf("Git 저장소 존재: %s\n", gitSync.SyncDir())
	}

	// Get remote URL from args
	remoteURL := ""
	if len(args) > 0 {
		remoteURL = args[0]
	}

	// Initialize
	if err := gitSync.Init(remoteURL); err != nil {
		return fmt.Errorf("초기화 실패: %w", err)
	}

	fmt.Printf("동기화 저장소 초기화 완료: %s\n", gitSync.SyncDir())
	if remoteURL != "" {
		fmt.Printf("원격 저장소 연결: %s\n", remoteURL)
	} else {
		fmt.Println("\n원격 저장소 연결:")
		fmt.Println("  pal sync init <remote-url>")
	}

	return nil
}

func runSyncPush(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)
	gitSync := sync.NewGitSync(database, envSvc)

	result, err := gitSync.Push(syncMessage)
	if err != nil {
		return err
	}

	if IsJSON() {
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	// Display result
	fmt.Println("Push 결과:")
	fmt.Printf("\n내보낸 데이터:\n")
	fmt.Printf("  포트: %d\n", result.ExportedStats.PortsCount)
	fmt.Printf("  세션: %d\n", result.ExportedStats.SessionsCount)
	fmt.Printf("  에스컬레이션: %d\n", result.ExportedStats.EscalationsCount)
	fmt.Printf("  파이프라인: %d\n", result.ExportedStats.PipelinesCount)
	fmt.Printf("  프로젝트: %d\n", result.ExportedStats.ProjectsCount)

	if result.NothingToCommit {
		fmt.Println("\n변경사항 없음 - 커밋 건너뜀")
		return nil
	}

	if result.Committed {
		fmt.Println("\n✓ 커밋 완료")
	}

	if result.Pushed {
		fmt.Println("✓ 푸시 완료")
	} else if result.NoRemote {
		fmt.Println("\n원격 저장소 미설정")
		fmt.Println("  'pal sync init <remote-url>'로 원격 저장소를 연결하세요")
	} else if result.PushError != "" {
		fmt.Printf("\n⚠ %s\n", result.PushError)
	}

	return nil
}

func runSyncPull(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)
	gitSync := sync.NewGitSync(database, envSvc)

	// Parse strategy
	var strategy sync.MergeStrategy
	switch syncStrategy {
	case "last_write_wins":
		strategy = sync.MergeStrategyLastWriteWins
	case "keep_local":
		strategy = sync.MergeStrategyKeepLocal
	case "keep_remote":
		strategy = sync.MergeStrategyKeepRemote
	case "manual":
		strategy = sync.MergeStrategyManual
	default:
		return fmt.Errorf("알 수 없는 전략: %s", syncStrategy)
	}

	options := sync.ImportOptions{
		Strategy:      strategy,
		DryRun:        syncDryRun,
		SkipConflicts: syncSkipConflicts,
	}

	result, err := gitSync.Pull(options)
	if err != nil {
		return err
	}

	if IsJSON() {
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	// Display result
	fmt.Println("Pull 결과:")

	if result.NoRemote {
		fmt.Println("\n원격 저장소 미설정")
		fmt.Println("  'pal sync init <remote-url>'로 원격 저장소를 연결하세요")
		return nil
	}

	if result.HasConflict {
		fmt.Println("\n⚠ Git 충돌 발생!")
		fmt.Println(result.ConflictMessage)
		fmt.Println("\n수동으로 충돌을 해결한 후 다시 시도하세요.")
		return nil
	}

	if result.PullError != "" {
		fmt.Printf("\n⚠ %s\n", result.PullError)
	} else if result.Pulled {
		fmt.Println("\n✓ Git pull 완료")
	}

	if result.NoData {
		fmt.Println("\n동기화 데이터 없음")
		return nil
	}

	if result.ImportResult != nil {
		fmt.Printf("\n가져온 항목:\n")
		fmt.Printf("  포트: %d\n", result.ImportResult.Imported.Ports)
		fmt.Printf("  세션: %d\n", result.ImportResult.Imported.Sessions)
		fmt.Printf("  에스컬레이션: %d\n", result.ImportResult.Imported.Escalations)
		fmt.Printf("  파이프라인: %d\n", result.ImportResult.Imported.Pipelines)
		fmt.Printf("  프로젝트: %d\n", result.ImportResult.Imported.Projects)

		if len(result.ImportResult.Conflicts) > 0 {
			fmt.Printf("\n충돌 항목: %d개\n", len(result.ImportResult.Conflicts))
		}
	}

	fmt.Println("\n✓ Pull 완료")
	return nil
}

func runSyncConflicts(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)
	resolver := sync.NewConflictResolver(database, envSvc)

	conflicts, err := resolver.GetPendingConflicts()
	if err != nil {
		return fmt.Errorf("충돌 목록 조회 실패: %w", err)
	}

	if IsJSON() {
		jsonData, _ := json.MarshalIndent(conflicts, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	if len(conflicts) == 0 {
		fmt.Println("대기 중인 충돌이 없습니다.")
		return nil
	}

	fmt.Printf("대기 중인 충돌: %d개\n\n", len(conflicts))

	for i, c := range conflicts {
		fmt.Printf("%d. [%s] %s\n", i+1, c.Type, c.ID)

		if !c.LocalModified.IsZero() {
			fmt.Printf("   로컬 수정: %s", c.LocalModified.Format("2006-01-02 15:04:05"))
			if c.LocalEnv != "" {
				fmt.Printf(" (%s)", c.LocalEnv)
			}
			fmt.Println()
		}

		if !c.RemoteModified.IsZero() {
			fmt.Printf("   원격 수정: %s", c.RemoteModified.Format("2006-01-02 15:04:05"))
			if c.RemoteEnv != "" {
				fmt.Printf(" (%s)", c.RemoteEnv)
			}
			fmt.Println()
		}

		if len(c.Differences) > 0 {
			fmt.Println("   차이점:")
			for _, diff := range c.Differences {
				fmt.Printf("     - %s: %v → %v\n", diff.Field, diff.LocalValue, diff.RemoteValue)
			}
		}
		fmt.Println()
	}

	fmt.Println("충돌 해결:")
	fmt.Println("  pal sync resolve --all --keep-local   # 모든 충돌을 로컬로 해결")
	fmt.Println("  pal sync resolve --all --keep-remote  # 모든 충돌을 원격으로 해결")
	fmt.Println("  pal sync resolve <id> --keep-local    # 특정 충돌 해결")

	return nil
}

func runSyncResolve(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)
	resolver := sync.NewConflictResolver(database, envSvc)

	// Determine resolution
	var resolution string
	if syncKeepLocal {
		resolution = "keep_local"
	} else if syncKeepRemote {
		resolution = "keep_remote"
	} else if syncSkipResolve {
		resolution = "skip"
	} else {
		return fmt.Errorf("해결 방법을 지정하세요: --keep-local, --keep-remote, --skip")
	}

	if syncResolveAll {
		// Resolve all conflicts
		if err := resolver.ResolveAll(resolution); err != nil {
			return fmt.Errorf("충돌 해결 실패: %w", err)
		}
		fmt.Printf("모든 충돌을 '%s'로 해결했습니다.\n", resolution)
	} else if len(args) > 0 {
		// Resolve specific conflict
		id := args[0]
		conflictType := sync.ConflictType(syncConflictType)

		if syncConflictType == "" {
			// Try to find the conflict type automatically
			conflicts, err := resolver.GetPendingConflicts()
			if err != nil {
				return err
			}
			for _, c := range conflicts {
				if c.ID == id {
					conflictType = c.Type
					break
				}
			}
			if conflictType == "" {
				return fmt.Errorf("충돌을 찾을 수 없음: %s", id)
			}
		}

		if err := resolver.ResolveConflict(id, conflictType, resolution); err != nil {
			return fmt.Errorf("충돌 해결 실패: %w", err)
		}
		fmt.Printf("충돌 '%s'를 '%s'로 해결했습니다.\n", id, resolution)
	} else {
		return fmt.Errorf("충돌 ID를 지정하거나 --all 옵션을 사용하세요")
	}

	return nil
}

func runSyncDiff(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	envSvc := env.NewService(database)
	resolver := sync.NewConflictResolver(database, envSvc)

	// Determine file to compare
	var filePath string
	if len(args) > 0 {
		filePath = args[0]
	} else {
		// Find the most recent sync file
		syncDir := filepath.Join(config.GlobalDir(), "sync")
		files, err := filepath.Glob(filepath.Join(syncDir, "pal-data.yaml"))
		if err != nil || len(files) == 0 {
			// Try to find any YAML file
			files, err = filepath.Glob(filepath.Join(syncDir, "*.yaml"))
			if err != nil || len(files) == 0 {
				return fmt.Errorf("동기화 파일을 찾을 수 없습니다")
			}
		}
		filePath = files[0]
	}

	// Read the sync file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("파일 읽기 실패: %w", err)
	}

	var syncData sync.SyncData
	if err := yaml.Unmarshal(data, &syncData); err != nil {
		return fmt.Errorf("YAML 파싱 실패: %w", err)
	}

	// Detect conflicts
	store, err := resolver.DetectConflicts(&syncData)
	if err != nil {
		return fmt.Errorf("충돌 감지 실패: %w", err)
	}

	if IsJSON() {
		jsonData, _ := json.MarshalIndent(store, "", "  ")
		fmt.Println(string(jsonData))
		return nil
	}

	fmt.Printf("비교 파일: %s\n", filePath)
	fmt.Printf("내보낸 환경: %s (%s)\n", syncData.Manifest.ExportedBy, syncData.Manifest.ExportedEnv)
	fmt.Printf("내보낸 시간: %s\n\n", syncData.Manifest.ExportedAt.Format("2006-01-02 15:04:05"))

	if len(store.Conflicts) == 0 {
		fmt.Println("차이점 없음 - 로컬과 원격 데이터가 동일합니다.")
		return nil
	}

	fmt.Printf("발견된 차이점: %d개\n\n", len(store.Conflicts))

	// Group by type
	byType := make(map[sync.ConflictType][]sync.ConflictDetail)
	for _, c := range store.Conflicts {
		byType[c.Type] = append(byType[c.Type], c)
	}

	typeNames := map[sync.ConflictType]string{
		sync.ConflictTypePort:       "포트",
		sync.ConflictTypeSession:    "세션",
		sync.ConflictTypeEscalation: "에스컬레이션",
		sync.ConflictTypePipeline:   "파이프라인",
		sync.ConflictTypeProject:    "프로젝트",
	}

	for cType, conflicts := range byType {
		fmt.Printf("## %s (%d개)\n\n", typeNames[cType], len(conflicts))

		for _, c := range conflicts {
			fmt.Printf("  %s\n", c.ID)

			newer := "동일"
			if c.LocalModified.After(c.RemoteModified) {
				newer = "로컬이 최신"
			} else if c.RemoteModified.After(c.LocalModified) {
				newer = "원격이 최신"
			}
			fmt.Printf("    %s\n", newer)

			for _, diff := range c.Differences {
				fmt.Printf("    - %s:\n", diff.Field)
				fmt.Printf("        로컬:  %v\n", diff.LocalValue)
				fmt.Printf("        원격:  %v\n", diff.RemoteValue)
			}
			fmt.Println()
		}
	}

	// Save conflicts for resolution
	if err := resolver.SaveConflicts(store); err != nil {
		fmt.Printf("⚠ 충돌 저장 실패: %v\n", err)
	} else {
		fmt.Println("충돌을 저장했습니다. 'pal sync resolve'로 해결할 수 있습니다.")
	}

	return nil
}
