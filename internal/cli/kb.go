package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/kb"
	"github.com/spf13/cobra"
)

var kbCmd = &cobra.Command{
	Use:   "kb",
	Short: "Knowledge Base 관리",
	Long:  `Knowledge Base 구조 관리 및 검색`,
}

var kbInitCmd = &cobra.Command{
	Use:   "init [vault-path]",
	Short: "KB 초기화",
	Long: `Knowledge Base 구조를 초기화합니다.

생성되는 구조:
  _taxonomy/          분류체계 정의
  00-System/          시스템 문서
  10-Domains/         도메인 지식
  20-Projects/        프로젝트 문서
  30-References/      참조 문서
  40-Archive/         아카이브
  .pal-kb/            메타데이터`,
	Args: cobra.MaximumNArgs(1),
	RunE: runKBInit,
}

var kbStatusCmd = &cobra.Command{
	Use:   "status [vault-path]",
	Short: "KB 상태 확인",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBStatus,
}

var kbTocCmd = &cobra.Command{
	Use:   "toc",
	Short: "목차(TOC) 관리",
	Long:  `Knowledge Base 목차 생성, 갱신, 검사`,
}

var kbTocGenerateCmd = &cobra.Command{
	Use:   "generate [vault-path]",
	Short: "목차 생성",
	Long: `모든 섹션의 목차를 생성합니다.

옵션:
  --depth     목차 깊이 (기본: 2)
  --sort      정렬 방식: alphabetical, date (기본: alphabetical)
  --section   특정 섹션만 생성`,
	Args: cobra.MaximumNArgs(1),
	RunE: runKBTocGenerate,
}

var kbTocUpdateCmd = &cobra.Command{
	Use:   "update [vault-path]",
	Short: "목차 갱신",
	Long:  `변경된 문서가 있는 섹션의 목차만 갱신합니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBTocUpdate,
}

var kbTocCheckCmd = &cobra.Command{
	Use:   "check [vault-path]",
	Short: "목차 무결성 검사",
	Long:  `목차의 링크 유효성과 누락된 문서를 검사합니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBTocCheck,
}

var tocDepth int
var tocSort string
var tocSection string

var kbIndexCmd = &cobra.Command{
	Use:   "index [vault-path]",
	Short: "색인 구축/갱신",
	Long: `Knowledge Base 문서를 색인합니다.

--rebuild 옵션으로 전체 재색인을 수행할 수 있습니다.
기본적으로 변경된 문서만 업데이트합니다.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runKBIndex,
}

var kbSearchCmd = &cobra.Command{
	Use:   "search <query> [vault-path]",
	Short: "문서 검색",
	Long: `Knowledge Base에서 문서를 검색합니다.

필터 옵션:
  --type      문서 타입 (port, adr, concept, guide)
  --domain    도메인
  --status    상태 (draft, active, archived)
  --tag       태그 (복수 지정 가능)
  --limit     결과 수 제한
  --budget    토큰 예산`,
	Args: cobra.MinimumNArgs(1),
	RunE: runKBSearch,
}

var kbStatsCmd = &cobra.Command{
	Use:   "stats [vault-path]",
	Short: "색인 통계",
	Long:  `색인된 문서의 통계를 표시합니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBStats,
}

var kbLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "링크 관리",
	Long:  `문서 간 링크 검사 및 그래프 생성`,
}

var kbLinkCheckCmd = &cobra.Command{
	Use:   "check [vault-path]",
	Short: "깨진 링크 검사",
	Long:  `모든 문서의 [[wikilink]]를 검사하여 깨진 링크를 찾습니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBLinkCheck,
}

var kbLinkGraphCmd = &cobra.Command{
	Use:   "graph [vault-path]",
	Short: "링크 그래프 생성",
	Long:  `문서 간 링크 그래프를 생성하여 .pal-kb/link-graph.json에 저장합니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBLinkGraph,
}

var kbTagCmd = &cobra.Command{
	Use:   "tag",
	Short: "태그 관리",
	Long:  `태그 목록 및 사용 현황 관리`,
}

var kbTagListCmd = &cobra.Command{
	Use:   "list [vault-path]",
	Short: "태그 목록",
	Long:  `사용 중인 모든 태그와 사용 횟수를 표시합니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBTagList,
}

var kbTagOrphanCmd = &cobra.Command{
	Use:   "orphan [vault-path]",
	Short: "미사용 태그 검출",
	Long:  `_taxonomy/tags.yaml에 정의되었지만 사용되지 않는 태그를 찾습니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBTagOrphan,
}

var kbSyncCmd = &cobra.Command{
	Use:   "sync <project-path> [vault-path]",
	Short: "프로젝트 동기화",
	Long: `프로젝트 문서를 Knowledge Base로 동기화합니다.

동기화 대상:
  - ports/           → 20-Projects/{project}/ports/
  - .pal/decisions/  → 20-Projects/{project}/decisions/
  - .pal/sessions/   → 20-Projects/{project}/sessions/
  - docs/            → 20-Projects/{project}/docs/

옵션:
  --dry-run   실제 동기화 없이 변경 내용만 표시
  --force     충돌 무시하고 강제 동기화`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runKBSync,
}

var kbSyncStatusCmd = &cobra.Command{
	Use:   "sync-status [vault-path]",
	Short: "동기화 상태 확인",
	Long:  `동기화된 프로젝트 목록과 마지막 동기화 시간을 표시합니다.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBSyncStatus,
}

var kbClassifyCmd = &cobra.Command{
	Use:   "classify <file>",
	Short: "문서 분류 추천",
	Long: `문서 내용을 분석하여 적절한 분류를 추천합니다.

추천 항목:
  - 문서 타입 (port, adr, concept, guide 등)
  - 도메인 (auth, api, database 등)
  - 태그

Taxonomy 정의가 있으면 해당 정의를 기반으로 추천합니다.`,
	Args: cobra.ExactArgs(1),
	RunE: runKBClassify,
}

var kbLintCmd = &cobra.Command{
	Use:   "lint <file-or-dir>",
	Short: "문서 품질 검사",
	Long: `문서 품질을 검사하고 문제점을 리포트합니다.

검사 항목:
  - Frontmatter 필수 필드 존재 여부
  - 제목 일관성
  - 문서 구조 (헤딩 레벨 등)
  - 링크 유효성
  - 태그 형식

등급 기준:
  A: 90-100점  B: 80-89점  C: 70-79점
  D: 60-69점   F: 60점 미만`,
	Args: cobra.ExactArgs(1),
	RunE: runKBLint,
}

var kbSyncDryRun bool
var kbSyncForce bool
var lintStrict bool
var lintCheckLinks bool

var indexRebuild bool
var searchType string
var searchDomain string
var searchStatus string
var searchTags []string
var searchLimit int
var searchBudget int

func init() {
	rootCmd.AddCommand(kbCmd)
	kbCmd.AddCommand(kbInitCmd)
	kbCmd.AddCommand(kbStatusCmd)
	kbCmd.AddCommand(kbTocCmd)
	kbCmd.AddCommand(kbIndexCmd)
	kbCmd.AddCommand(kbSearchCmd)
	kbCmd.AddCommand(kbStatsCmd)
	kbCmd.AddCommand(kbLinkCmd)
	kbCmd.AddCommand(kbTagCmd)
	kbCmd.AddCommand(kbSyncCmd)
	kbCmd.AddCommand(kbSyncStatusCmd)
	kbCmd.AddCommand(kbClassifyCmd)
	kbCmd.AddCommand(kbLintCmd)

	kbTocCmd.AddCommand(kbTocGenerateCmd)
	kbTocCmd.AddCommand(kbTocUpdateCmd)
	kbTocCmd.AddCommand(kbTocCheckCmd)

	kbLinkCmd.AddCommand(kbLinkCheckCmd)
	kbLinkCmd.AddCommand(kbLinkGraphCmd)

	kbTagCmd.AddCommand(kbTagListCmd)
	kbTagCmd.AddCommand(kbTagOrphanCmd)

	kbTocGenerateCmd.Flags().IntVar(&tocDepth, "depth", 2, "목차 깊이")
	kbTocGenerateCmd.Flags().StringVar(&tocSort, "sort", "alphabetical", "정렬 방식 (alphabetical, date)")
	kbTocGenerateCmd.Flags().StringVar(&tocSection, "section", "", "특정 섹션만 생성")

	kbIndexCmd.Flags().BoolVar(&indexRebuild, "rebuild", false, "전체 재색인")

	kbSearchCmd.Flags().StringVar(&searchType, "type", "", "문서 타입 필터")
	kbSearchCmd.Flags().StringVar(&searchDomain, "domain", "", "도메인 필터")
	kbSearchCmd.Flags().StringVar(&searchStatus, "status", "", "상태 필터")
	kbSearchCmd.Flags().StringSliceVar(&searchTags, "tag", nil, "태그 필터")
	kbSearchCmd.Flags().IntVar(&searchLimit, "limit", 10, "결과 수 제한")
	kbSearchCmd.Flags().IntVar(&searchBudget, "budget", 0, "토큰 예산")

	kbSyncCmd.Flags().BoolVar(&kbSyncDryRun, "dry-run", false, "실제 동기화 없이 변경 내용만 표시")
	kbSyncCmd.Flags().BoolVar(&kbSyncForce, "force", false, "충돌 무시하고 강제 동기화")

	kbLintCmd.Flags().BoolVar(&lintStrict, "strict", false, "엄격 모드 (오류 시 실패)")
	kbLintCmd.Flags().BoolVar(&lintCheckLinks, "check-links", true, "링크 유효성 검사")
}

func getVaultPath(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	cwd, _ := os.Getwd()
	return cwd
}

func runKBInit(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	svc := kb.NewService(vaultPath)
	if err := svc.Init(); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":     "initialized",
			"vault_path": vaultPath,
		})
	}

	fmt.Println("✅ Knowledge Base 초기화 완료")
	fmt.Printf("   경로: %s\n", vaultPath)
	fmt.Println()
	fmt.Println("📁 생성된 구조:")
	fmt.Println("   _taxonomy/      분류체계")
	fmt.Println("   00-System/      시스템")
	fmt.Println("   10-Domains/     도메인")
	fmt.Println("   20-Projects/    프로젝트")
	fmt.Println("   30-References/  참조")
	fmt.Println("   40-Archive/     아카이브")
	fmt.Println()
	fmt.Println("다음 단계:")
	fmt.Println("  1. _taxonomy/domains.yaml 편집")
	fmt.Println("  2. pal kb toc generate")

	return nil
}

func runKBStatus(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(status)
	}

	fmt.Println("📚 Knowledge Base 상태")
	fmt.Printf("   경로: %s\n", status.VaultPath)

	if !status.Initialized {
		fmt.Println("   상태: ❌ 초기화되지 않음")
		fmt.Println()
		fmt.Println("초기화: pal kb init")
		return nil
	}

	fmt.Println("   상태: ✅ 초기화됨")
	fmt.Printf("   버전: %s\n", status.Version)
	fmt.Printf("   생성: %s\n", status.CreatedAt)
	fmt.Println()
	fmt.Println("📊 문서 수:")
	for section, count := range status.Sections {
		fmt.Printf("   %-15s %d\n", section, count)
	}

	return nil
}

func runKBTocGenerate(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	svc := kb.NewService(vaultPath)

	// Check if initialized
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	if tocSection != "" {
		// Generate single section
		stats, err := svc.GenerateTOC(tocSection, tocDepth, tocSort)
		if err != nil {
			return err
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"section": tocSection,
				"stats":   stats,
			})
		}

		fmt.Printf("✅ %s 목차 생성 완료\n", tocSection)
		fmt.Printf("   문서: %d개, 섹션: %d개\n", stats.TotalDocs, stats.Sections)
		return nil
	}

	// Generate all sections
	results, err := svc.GenerateAllTOC(tocDepth, tocSort)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	fmt.Println("✅ 전체 목차 생성 완료")
	fmt.Println()
	fmt.Println("📊 섹션별 통계:")
	for section, stats := range results {
		fmt.Printf("   %-15s 문서 %d개, 섹션 %d개\n", section, stats.TotalDocs, stats.Sections)
	}

	return nil
}

func runKBTocUpdate(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	svc := kb.NewService(vaultPath)

	// Check if initialized
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	sections := []string{kb.SystemDir, kb.DomainsDir, kb.ProjectsDir, kb.ReferencesDir, kb.ArchiveDir}
	updatedSections := []string{}

	for _, sec := range sections {
		stats, updated, err := svc.UpdateTOC(sec)
		if err != nil {
			return fmt.Errorf("%s 업데이트 실패: %w", sec, err)
		}
		if updated {
			updatedSections = append(updatedSections, sec)
			if !jsonOut {
				fmt.Printf("📝 %s 갱신됨 (문서 %d개)\n", sec, stats.TotalDocs)
			}
		}
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"updated_sections": updatedSections,
		})
	}

	if len(updatedSections) == 0 {
		fmt.Println("✅ 모든 목차가 최신 상태입니다")
	} else {
		fmt.Printf("\n✅ %d개 섹션 갱신 완료\n", len(updatedSections))
	}

	return nil
}

func runKBTocCheck(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	svc := kb.NewService(vaultPath)

	// Check if initialized
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	results, err := svc.CheckAllTOC()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	fmt.Println("📋 목차 무결성 검사")
	fmt.Println()

	allValid := true
	for _, result := range results {
		statusIcon := "✅"
		if !result.Valid {
			statusIcon = "❌"
			allValid = false
		} else if result.NeedsRefresh {
			statusIcon = "⚠️"
		}

		fmt.Printf("%s %s\n", statusIcon, result.Section)

		if result.LastUpdated != "" {
			fmt.Printf("   마지막 갱신: %s\n", result.LastUpdated)
		}

		if len(result.OrphanLinks) > 0 {
			fmt.Printf("   깨진 링크: %d개\n", len(result.OrphanLinks))
			for _, link := range result.OrphanLinks {
				fmt.Printf("     - %s\n", link)
			}
		}

		if len(result.MissingDocs) > 0 {
			fmt.Printf("   누락된 문서: %d개\n", len(result.MissingDocs))
			for _, doc := range result.MissingDocs {
				fmt.Printf("     - %s\n", doc)
			}
		}

		if result.NeedsRefresh && result.Valid {
			fmt.Printf("   💡 갱신 필요 (pal kb toc update)\n")
		}
	}

	fmt.Println()
	if allValid {
		fmt.Println("✅ 모든 목차 유효")
	} else {
		fmt.Println("❌ 문제가 발견됨. 'pal kb toc generate'로 재생성하세요")
	}

	return nil
}

func runKBIndex(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	// Check if initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	indexSvc := kb.NewIndexService(vaultPath)
	if err := indexSvc.Open(); err != nil {
		return err
	}
	defer indexSvc.Close()

	if indexRebuild {
		// Full rebuild
		stats, err := indexSvc.BuildIndex()
		if err != nil {
			return err
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(stats)
		}

		fmt.Println("✅ 색인 구축 완료")
		fmt.Printf("   문서 수: %d\n", stats.TotalDocs)
		if len(stats.ByType) > 0 {
			fmt.Println("   타입별:")
			for t, c := range stats.ByType {
				fmt.Printf("     %s: %d\n", t, c)
			}
		}
		if len(stats.ByDomain) > 0 {
			fmt.Println("   도메인별:")
			for d, c := range stats.ByDomain {
				fmt.Printf("     %s: %d\n", d, c)
			}
		}
	} else {
		// Incremental update
		added, updated, err := indexSvc.UpdateIndex()
		if err != nil {
			return err
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"added":   added,
				"updated": updated,
			})
		}

		if added == 0 && updated == 0 {
			fmt.Println("✅ 색인이 최신 상태입니다")
		} else {
			fmt.Println("✅ 색인 갱신 완료")
			if added > 0 {
				fmt.Printf("   추가: %d\n", added)
			}
			if updated > 0 {
				fmt.Printf("   갱신: %d\n", updated)
			}
		}
	}

	return nil
}

func runKBSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	vaultPath := "."
	if len(args) > 1 {
		vaultPath = args[1]
	}

	// Check if initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	indexSvc := kb.NewIndexService(vaultPath)
	if err := indexSvc.Open(); err != nil {
		return err
	}
	defer indexSvc.Close()

	opts := &kb.SearchOptions{
		Type:        searchType,
		Domain:      searchDomain,
		Status:      searchStatus,
		Tags:        searchTags,
		Limit:       searchLimit,
		TokenBudget: searchBudget,
	}

	results, err := indexSvc.Search(query, opts)
	if err != nil {
		return fmt.Errorf("검색 실패: %w", err)
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	if len(results) == 0 {
		fmt.Println("검색 결과 없음")
		return nil
	}

	fmt.Printf("🔍 '%s' 검색 결과 (%d건)\n\n", query, len(results))

	for i, r := range results {
		doc := r.Document
		fmt.Printf("%d. %s\n", i+1, doc.Title)
		fmt.Printf("   📄 %s\n", doc.Path)

		meta := []string{}
		if doc.Type != "" {
			meta = append(meta, fmt.Sprintf("타입:%s", doc.Type))
		}
		if doc.Domain != "" {
			meta = append(meta, fmt.Sprintf("도메인:%s", doc.Domain))
		}
		if doc.Status != "" {
			meta = append(meta, fmt.Sprintf("상태:%s", doc.Status))
		}
		if len(meta) > 0 {
			fmt.Printf("   %s\n", strings.Join(meta, " | "))
		}

		if doc.Summary != "" {
			fmt.Printf("   %s\n", doc.Summary)
		}

		if len(doc.Tags) > 0 {
			fmt.Printf("   🏷️  %s\n", strings.Join(doc.Tags, ", "))
		}

		fmt.Println()
	}

	return nil
}

func runKBStats(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	// Check if initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	indexSvc := kb.NewIndexService(vaultPath)
	if err := indexSvc.Open(); err != nil {
		return err
	}
	defer indexSvc.Close()

	stats, err := indexSvc.GetStats()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(stats)
	}

	fmt.Println("📊 색인 통계")
	fmt.Printf("   총 문서: %d\n", stats.TotalDocs)

	if stats.LastIndexed != "" {
		t, _ := time.Parse(time.RFC3339, stats.LastIndexed)
		fmt.Printf("   마지막 색인: %s\n", t.Format("2006-01-02 15:04"))
	}

	if len(stats.ByType) > 0 {
		fmt.Println("\n📁 타입별:")
		for t, c := range stats.ByType {
			fmt.Printf("   %-15s %d\n", t, c)
		}
	}

	if len(stats.ByDomain) > 0 {
		fmt.Println("\n🌐 도메인별:")
		for d, c := range stats.ByDomain {
			fmt.Printf("   %-15s %d\n", d, c)
		}
	}

	if len(stats.ByStatus) > 0 {
		fmt.Println("\n📋 상태별:")
		for s, c := range stats.ByStatus {
			fmt.Printf("   %-15s %d\n", s, c)
		}
	}

	return nil
}

func runKBLinkCheck(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	// Check if initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	linkSvc := kb.NewLinkService(vaultPath)
	result, err := linkSvc.CheckLinks()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Println("🔗 링크 검사 결과")
	fmt.Printf("   총 링크: %d\n", result.TotalLinks)
	fmt.Printf("   유효 링크: %d\n", result.ValidLinks)
	fmt.Printf("   깨진 링크: %d\n", len(result.BrokenLinks))

	if len(result.BrokenLinks) > 0 {
		fmt.Println("\n❌ 깨진 링크:")
		for _, broken := range result.BrokenLinks {
			fmt.Printf("   %s:%d\n", broken.Source, broken.Line)
			fmt.Printf("     → [[%s]]\n", broken.Target)
			if broken.Suggestion != "" {
				fmt.Printf("     💡 추천: [[%s]]\n", broken.Suggestion)
			}
		}
	} else {
		fmt.Println("\n✅ 모든 링크 유효")
	}

	return nil
}

func runKBLinkGraph(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	// Check if initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	linkSvc := kb.NewLinkService(vaultPath)
	graph, err := linkSvc.BuildLinkGraph()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(graph)
	}

	// Save graph
	if err := linkSvc.SaveLinkGraph(graph); err != nil {
		return fmt.Errorf("그래프 저장 실패: %w", err)
	}

	fmt.Println("📊 링크 그래프 생성 완료")
	fmt.Printf("   노드 수: %d\n", len(graph.Nodes))
	fmt.Printf("   엣지 수: %d\n", len(graph.Edges))
	fmt.Println("   저장 위치: .pal-kb/link-graph.json")

	// Show top connected nodes
	if len(graph.Nodes) > 0 {
		// Sort by total connections
		nodes := make([]kb.GraphNode, len(graph.Nodes))
		copy(nodes, graph.Nodes)
		sort.Slice(nodes, func(i, j int) bool {
			return (nodes[i].InLinks + nodes[i].OutLinks) > (nodes[j].InLinks + nodes[j].OutLinks)
		})

		fmt.Println("\n🔝 연결이 많은 문서:")
		limit := 5
		if len(nodes) < limit {
			limit = len(nodes)
		}
		for i := 0; i < limit; i++ {
			n := nodes[i]
			fmt.Printf("   %s (in:%d, out:%d)\n", n.Label, n.InLinks, n.OutLinks)
		}
	}

	return nil
}

func runKBTagList(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	// Check if initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	linkSvc := kb.NewLinkService(vaultPath)
	tags, err := linkSvc.ListTags()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(tags)
	}

	if len(tags) == 0 {
		fmt.Println("태그 없음")
		return nil
	}

	// Sort by count descending
	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Count > tags[j].Count
	})

	fmt.Printf("🏷️  태그 목록 (%d개)\n\n", len(tags))
	for _, tag := range tags {
		fmt.Printf("   %-20s %d\n", "#"+tag.Name, tag.Count)
	}

	return nil
}

func runKBTagOrphan(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	// Check if initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	linkSvc := kb.NewLinkService(vaultPath)
	result, err := linkSvc.CheckOrphanTags()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Println("🏷️  태그 사용 현황")
	fmt.Printf("   정의된 태그: %d\n", result.TotalTags)
	fmt.Printf("   사용 중: %d\n", result.UsedTags)

	if len(result.OrphanTags) > 0 {
		fmt.Println("\n⚠️  미사용 태그 (정의되었지만 사용 안 함):")
		for _, tag := range result.OrphanTags {
			fmt.Printf("   #%s\n", tag)
		}
	}

	if len(result.UnknownTags) > 0 {
		fmt.Println("\n❓ 미정의 태그 (사용 중이지만 정의 안 됨):")
		for _, tag := range result.UnknownTags {
			fmt.Printf("   #%-20s (%d회 사용)\n", tag.Name, tag.Count)
		}
	}

	if len(result.OrphanTags) == 0 && len(result.UnknownTags) == 0 {
		fmt.Println("\n✅ 모든 태그가 정상적으로 사용 중")
	}

	return nil
}

func runKBSync(cmd *cobra.Command, args []string) error {
	projectPath := args[0]
	vaultPath := "."
	if len(args) > 1 {
		vaultPath = args[1]
	}

	// Check if vault is initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	// Check if project path exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("프로젝트 경로가 존재하지 않습니다: %s", projectPath)
	}

	// Make project path absolute
	projectPath, err = filepath.Abs(projectPath)
	if err != nil {
		return err
	}

	syncSvc := kb.NewSyncService(vaultPath, projectPath)

	opts := &kb.SyncOptions{
		DryRun: kbSyncDryRun,
		Force:  kbSyncForce,
	}

	result, err := syncSvc.Sync(opts)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	projectName := filepath.Base(projectPath)

	if kbSyncDryRun {
		fmt.Printf("🔄 '%s' 동기화 미리보기\n\n", projectName)
	} else {
		fmt.Printf("🔄 '%s' 동기화 완료\n\n", projectName)
	}

	if len(result.Added) > 0 {
		fmt.Printf("➕ 추가: %d\n", len(result.Added))
		for _, f := range result.Added {
			fmt.Printf("   %s\n", f)
		}
	}

	if len(result.Updated) > 0 {
		fmt.Printf("📝 갱신: %d\n", len(result.Updated))
		for _, f := range result.Updated {
			fmt.Printf("   %s\n", f)
		}
	}

	if len(result.Deleted) > 0 {
		fmt.Printf("🗑️  삭제: %d\n", len(result.Deleted))
		for _, f := range result.Deleted {
			fmt.Printf("   %s\n", f)
		}
	}

	if len(result.Conflicts) > 0 {
		fmt.Printf("\n⚠️  충돌: %d\n", len(result.Conflicts))
		for _, c := range result.Conflicts {
			fmt.Printf("   %s\n", c.Path)
			fmt.Printf("     소스: %s\n", c.SourceTime)
			fmt.Printf("     대상: %s\n", c.TargetTime)
		}
		fmt.Println("\n--force 옵션으로 강제 동기화할 수 있습니다.")
	}

	totalChanges := len(result.Added) + len(result.Updated) + len(result.Deleted)
	if totalChanges == 0 && len(result.Conflicts) == 0 {
		fmt.Println("변경 사항 없음")
	} else if !kbSyncDryRun {
		fmt.Printf("\n✅ 동기화 위치: 20-Projects/%s/\n", projectName)
	}

	return nil
}

func runKBSyncStatus(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	// Check if vault is initialized
	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}
	if !status.Initialized {
		return fmt.Errorf("KB가 초기화되지 않았습니다. 'pal kb init' 실행하세요")
	}

	// Create a dummy sync service to list projects
	syncSvc := kb.NewSyncService(vaultPath, ".")
	projects, err := syncSvc.ListSyncedProjects()
	if err != nil {
		// No sync state yet
		fmt.Println("📋 동기화된 프로젝트 없음")
		fmt.Println("\n사용법: pal kb sync <project-path>")
		return nil
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(projects)
	}

	if len(projects) == 0 {
		fmt.Println("📋 동기화된 프로젝트 없음")
		fmt.Println("\n사용법: pal kb sync <project-path>")
		return nil
	}

	fmt.Printf("📋 동기화된 프로젝트 (%d개)\n\n", len(projects))

	for _, p := range projects {
		lastSync := "알 수 없음"
		if p.LastSync != "" {
			if t, err := time.Parse(time.RFC3339, p.LastSync); err == nil {
				lastSync = t.Format("2006-01-02 15:04")
			}
		}

		fmt.Printf("📁 %s\n", p.Name)
		fmt.Printf("   경로: %s\n", p.SourcePath)
		fmt.Printf("   마지막 동기화: %s\n", lastSync)
		fmt.Printf("   동기화된 파일: %d개\n", len(p.Files))
		fmt.Println()
	}

	return nil
}

func runKBClassify(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("파일이 존재하지 않습니다: %s", filePath)
	}

	// Get vault path from file location or current directory
	vaultPath := "."
	if absPath, err := filepath.Abs(filePath); err == nil {
		// Try to find vault root by looking for .pal-kb
		dir := filepath.Dir(absPath)
		for dir != "/" {
			if _, err := os.Stat(filepath.Join(dir, ".pal-kb")); err == nil {
				vaultPath = dir
				break
			}
			dir = filepath.Dir(dir)
		}
	}

	classifier := kb.NewClassifierService(vaultPath)
	result, err := classifier.Classify(filePath)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Printf("📋 분류 추천: %s\n\n", filePath)

	// Current metadata
	if len(result.CurrentMeta) > 0 {
		fmt.Println("📌 현재 메타데이터:")
		if t, ok := result.CurrentMeta["type"]; ok {
			fmt.Printf("   type: %v\n", t)
		}
		if d, ok := result.CurrentMeta["domain"]; ok {
			fmt.Printf("   domain: %v\n", d)
		}
		if tags, ok := result.CurrentMeta["tags"]; ok {
			fmt.Printf("   tags: %v\n", tags)
		}
		fmt.Println()
	}

	// Suggested type
	if len(result.SuggestedType) > 0 {
		fmt.Println("📝 추천 타입:")
		for i, s := range result.SuggestedType {
			confidence := ""
			if s.Score >= 0.7 {
				confidence = "⭐⭐⭐"
			} else if s.Score >= 0.4 {
				confidence = "⭐⭐"
			} else {
				confidence = "⭐"
			}
			marker := "  "
			if i == 0 {
				marker = "→ "
			}
			fmt.Printf("   %s%s %s (%.0f%%)\n", marker, s.Type, confidence, s.Score*100)
			if s.Reason != "" {
				fmt.Printf("      └ %s\n", s.Reason)
			}
		}
		fmt.Println()
	}

	// Suggested domain
	if len(result.SuggestedDomain) > 0 {
		fmt.Println("🏷️  추천 도메인:")
		for i, s := range result.SuggestedDomain {
			confidence := ""
			if s.Score >= 0.7 {
				confidence = "⭐⭐⭐"
			} else if s.Score >= 0.4 {
				confidence = "⭐⭐"
			} else {
				confidence = "⭐"
			}
			marker := "  "
			if i == 0 {
				marker = "→ "
			}
			fmt.Printf("   %s%s %s (%.0f%%)\n", marker, s.Domain, confidence, s.Score*100)
			if s.Reason != "" {
				fmt.Printf("      └ %s\n", s.Reason)
			}
		}
		fmt.Println()
	}

	// Suggested tags
	if len(result.SuggestedTags) > 0 {
		fmt.Println("🔖 추천 태그:")
		for _, s := range result.SuggestedTags {
			fmt.Printf("   #%s (%.0f%%) - %s\n", s.Tag, s.Score*100, s.Reason)
		}
		fmt.Println()
	}

	// Overall confidence
	confidenceLabel := "낮음"
	if result.Confidence >= 0.7 {
		confidenceLabel = "높음"
	} else if result.Confidence >= 0.4 {
		confidenceLabel = "보통"
	}
	fmt.Printf("📊 전체 신뢰도: %.0f%% (%s)\n", result.Confidence*100, confidenceLabel)

	return nil
}

func runKBLint(cmd *cobra.Command, args []string) error {
	targetPath := args[0]

	// Check if path exists
	info, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("경로가 존재하지 않습니다: %s", targetPath)
	}

	// Get vault path
	vaultPath := "."
	if absPath, err := filepath.Abs(targetPath); err == nil {
		dir := absPath
		if !info.IsDir() {
			dir = filepath.Dir(absPath)
		}
		for dir != "/" {
			if _, err := os.Stat(filepath.Join(dir, ".pal-kb")); err == nil {
				vaultPath = dir
				break
			}
			dir = filepath.Dir(dir)
		}
	}

	qualitySvc := kb.NewQualityService(vaultPath)
	opts := &kb.QualityOptions{
		CheckLinks: lintCheckLinks,
		CheckTags:  true,
		StrictMode: lintStrict,
	}

	var results []*kb.QualityResult

	if info.IsDir() {
		// Check directory
		results, err = qualitySvc.CheckDirectory(targetPath, opts)
		if err != nil {
			return err
		}
	} else {
		// Check single file
		result, err := qualitySvc.Check(targetPath, opts)
		if err != nil {
			return err
		}
		results = []*kb.QualityResult{result}
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	// Print results
	passedCount := 0
	failedCount := 0

	for _, r := range results {
		// Grade emoji
		gradeEmoji := "🔴"
		switch r.Grade {
		case "A":
			gradeEmoji = "🟢"
		case "B":
			gradeEmoji = "🟢"
		case "C":
			gradeEmoji = "🟡"
		case "D":
			gradeEmoji = "🟠"
		}

		// Relative path for display
		relPath := r.FilePath
		if absPath, err := filepath.Abs(targetPath); err == nil {
			if rel, err := filepath.Rel(filepath.Dir(absPath), r.FilePath); err == nil {
				relPath = rel
			}
		}

		fmt.Printf("%s %s [%s] %d점\n", gradeEmoji, relPath, r.Grade, r.Score)

		// Print issues
		if len(r.Issues) > 0 {
			for _, issue := range r.Issues {
				fmt.Printf("   ❌ %s\n", issue.Message)
				if issue.Suggestion != "" {
					fmt.Printf("      → %s\n", issue.Suggestion)
				}
			}
		}

		// Print warnings (only important ones)
		warningCount := 0
		for _, warn := range r.Warnings {
			if warn.Severity == "warning" {
				warningCount++
				if warningCount <= 3 {
					fmt.Printf("   ⚠️  %s\n", warn.Message)
				}
			}
		}
		if warningCount > 3 {
			fmt.Printf("   ... 그 외 %d개의 경고\n", warningCount-3)
		}

		if r.Passed {
			passedCount++
		} else {
			failedCount++
		}

		fmt.Println()
	}

	// Summary
	if len(results) > 1 {
		summary := kb.GetQualitySummary(results)
		fmt.Println("═══════════════════════════════════════")
		fmt.Printf("📊 검사 결과 요약\n")
		fmt.Printf("   총 파일: %d\n", summary["total"])
		fmt.Printf("   통과: %d, 실패: %d\n", summary["passed"], summary["failed"])
		fmt.Printf("   평균 점수: %.1f점\n", summary["avg_score"])

		gradeCounts := summary["grade_counts"].(map[string]int)
		fmt.Printf("   등급 분포: A:%d B:%d C:%d D:%d F:%d\n",
			gradeCounts["A"], gradeCounts["B"], gradeCounts["C"],
			gradeCounts["D"], gradeCounts["F"])
	}

	// Exit with error if strict mode and failed
	if lintStrict && failedCount > 0 {
		return fmt.Errorf("%d개 파일이 품질 검사를 통과하지 못했습니다", failedCount)
	}

	return nil
}
