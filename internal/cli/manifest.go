package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/manifest"
	"github.com/spf13/cobra"
)

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "설정 파일 변경 추적",
	Long: `PAL Kit 설정 파일들의 변경 사항을 추적합니다.

추적 대상:
  - CLAUDE.md
  - agents/*.yaml
  - conventions/*.yaml, *.md
  - ports/*.md
  - .pal/config.yaml

예시:
  pal manifest status     # 변경 상태 확인
  pal manifest sync       # 변경 사항 동기화
  pal manifest add <file> # 파일 추가
`,
}

var manifestStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "파일 변경 상태 확인",
	RunE:  runManifestStatus,
}

var manifestSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "변경 사항 동기화",
	RunE:  runManifestSync,
}

var manifestAddCmd = &cobra.Command{
	Use:   "add <file>",
	Short: "파일 추적 추가",
	Args:  cobra.ExactArgs(1),
	RunE:  runManifestAdd,
}

var manifestRemoveCmd = &cobra.Command{
	Use:   "remove <file>",
	Short: "파일 추적 제거",
	Args:  cobra.ExactArgs(1),
	RunE:  runManifestRemove,
}

var manifestHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "변경 히스토리 조회",
	RunE:  runManifestHistory,
}

var (
	manifestManagedBy string
	manifestLimit     int
)

func init() {
	rootCmd.AddCommand(manifestCmd)
	
	manifestCmd.AddCommand(manifestStatusCmd)
	manifestCmd.AddCommand(manifestSyncCmd)
	manifestCmd.AddCommand(manifestAddCmd)
	manifestCmd.AddCommand(manifestRemoveCmd)
	manifestCmd.AddCommand(manifestHistoryCmd)

	manifestAddCmd.Flags().StringVar(&manifestManagedBy, "managed-by", "user", "관리 주체 (pal, user, claude)")
	manifestHistoryCmd.Flags().IntVar(&manifestLimit, "limit", 20, "조회할 개수")
}

func getManifestService() (*manifest.Service, error) {
	if !config.IsInstalled() {
		return nil, fmt.Errorf("PAL Kit이 설치되지 않았습니다. 'pal install' 실행하세요.")
	}

	projectRoot := config.FindProjectRoot()
	if projectRoot == "" {
		return nil, fmt.Errorf("프로젝트 디렉토리가 아닙니다. 'pal init' 실행하세요.")
	}

	// .claude 디렉토리 존재 여부 확인
	if _, err := os.Stat(config.ProjectDir(projectRoot)); os.IsNotExist(err) {
		return nil, fmt.Errorf("프로젝트가 초기화되지 않았습니다. 'pal init' 실행하세요.")
	}

	database, err := db.Open(config.GlobalDBPath())
	if err != nil {
		return nil, fmt.Errorf("DB 열기 실패: %w", err)
	}

	return manifest.NewService(database, projectRoot), nil
}

func runManifestStatus(cmd *cobra.Command, args []string) error {
	svc, err := getManifestService()
	if err != nil {
		return err
	}

	statuses, err := svc.Status()
	if err != nil {
		return fmt.Errorf("상태 확인 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(statuses)
	}

	// 상태별 분류
	var synced, modified, newFiles, deleted []manifest.TrackedFile
	for _, f := range statuses {
		switch f.Status {
		case manifest.StatusSynced:
			synced = append(synced, f)
		case manifest.StatusModified:
			modified = append(modified, f)
		case manifest.StatusNew:
			newFiles = append(newFiles, f)
		case manifest.StatusDeleted:
			deleted = append(deleted, f)
		}
	}

	fmt.Println("📋 Manifest 상태")
	fmt.Println()

	// 동기화된 파일
	for _, f := range synced {
		fmt.Printf("  ✅ %-40s %s\n", f.Path, f.Type)
	}

	// 변경된 파일
	for _, f := range modified {
		fmt.Printf("  📝 %-40s %s (변경됨)\n", f.Path, f.Type)
	}

	// 새 파일
	for _, f := range newFiles {
		fmt.Printf("  ✨ %-40s %s (새 파일)\n", f.Path, f.Type)
	}

	// 삭제된 파일
	for _, f := range deleted {
		fmt.Printf("  ❌ %-40s %s (삭제됨)\n", f.Path, f.Type)
	}

	fmt.Println()
	fmt.Printf("총: %d개 파일 (동기화: %d, 변경: %d, 새 파일: %d, 삭제: %d)\n",
		len(statuses), len(synced), len(modified), len(newFiles), len(deleted))

	if len(modified)+len(newFiles)+len(deleted) > 0 {
		fmt.Println()
		fmt.Println("💡 변경 사항을 동기화하려면: pal manifest sync")
	}

	return nil
}

func runManifestSync(cmd *cobra.Command, args []string) error {
	svc, err := getManifestService()
	if err != nil {
		return err
	}

	changes, err := svc.Sync()
	if err != nil {
		return fmt.Errorf("동기화 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(changes)
	}

	if len(changes) == 0 {
		fmt.Println("✅ 모든 파일이 이미 동기화되어 있습니다.")
		return nil
	}

	fmt.Println("🔄 Manifest 동기화 완료")
	fmt.Println()

	for _, c := range changes {
		switch c.ChangeType {
		case "created":
			fmt.Printf("  ✨ %s (추가됨)\n", c.FilePath)
		case "modified":
			fmt.Printf("  📝 %s (업데이트됨)\n", c.FilePath)
		case "deleted":
			fmt.Printf("  ❌ %s (제거됨)\n", c.FilePath)
		}
	}

	fmt.Println()
	fmt.Printf("총 %d개 파일 동기화됨\n", len(changes))

	return nil
}

func runManifestAdd(cmd *cobra.Command, args []string) error {
	svc, err := getManifestService()
	if err != nil {
		return err
	}

	filePath := args[0]

	managedBy := manifest.ManagedByUser
	switch manifestManagedBy {
	case "pal":
		managedBy = manifest.ManagedByPal
	case "claude":
		managedBy = manifest.ManagedByClaude
	}

	if err := svc.AddFile(filePath, managedBy); err != nil {
		return fmt.Errorf("파일 추가 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "added",
			"path":   filePath,
		})
	}

	fmt.Printf("✅ %s 추가됨 (managed_by: %s)\n", filePath, managedBy)
	return nil
}

func runManifestRemove(cmd *cobra.Command, args []string) error {
	svc, err := getManifestService()
	if err != nil {
		return err
	}

	filePath := args[0]

	if err := svc.RemoveFile(filePath); err != nil {
		return fmt.Errorf("파일 제거 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "removed",
			"path":   filePath,
		})
	}

	fmt.Printf("✅ %s 추적 제거됨\n", filePath)
	return nil
}

func runManifestHistory(cmd *cobra.Command, args []string) error {
	svc, err := getManifestService()
	if err != nil {
		return err
	}

	changes, err := svc.GetChanges(manifestLimit)
	if err != nil {
		return fmt.Errorf("히스토리 조회 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(changes)
	}

	if len(changes) == 0 {
		fmt.Println("변경 히스토리가 없습니다.")
		return nil
	}

	fmt.Println("📜 변경 히스토리")
	fmt.Println()

	for _, c := range changes {
		icon := "📝"
		switch c.ChangeType {
		case "created":
			icon = "✨"
		case "deleted":
			icon = "❌"
		}
		fmt.Printf("  %s %-40s %s  %s\n", icon, c.FilePath, c.ChangeType, c.ChangedAt.Format("2006-01-02 15:04"))
	}

	return nil
}
