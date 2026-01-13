package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/marker"
)

var (
	markerPort    string
	markerDomain  string
	markerLayer   string
	markerStrict  bool
)

var markerCmd = &cobra.Command{
	Use:   "marker",
	Short: "PAL 코드 마커 관리",
	Long:  `코드의 @pal-port, @pal-layer, @pal-domain 마커를 검색하고 검증합니다.`,
}

var markerListCmd = &cobra.Command{
	Use:   "list",
	Short: "마커 목록",
	Long: `프로젝트 내 PAL 마커 목록을 출력합니다.

예시:
  pal marker list                    # 전체 마커
  pal marker list --port L1-*        # L1 레이어 포트
  pal marker list --domain units     # 특정 도메인
  pal marker list --layer l2         # L2 레이어`,
	RunE: runMarkerList,
}

var markerCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "마커 검증",
	Long: `마커의 유효성을 검증합니다.

예시:
  pal marker check                   # 기본 검증
  pal marker check --strict          # 엄격 모드 (layer, domain 필수)`,
	RunE: runMarkerCheck,
}

var markerFilesCmd = &cobra.Command{
	Use:   "files <port>",
	Short: "포트별 파일 목록",
	Long: `특정 포트가 마킹된 파일 목록을 출력합니다.

예시:
  pal marker files L1-InventoryCommandService`,
	Args: cobra.ExactArgs(1),
	RunE: runMarkerFiles,
}

var markerDepsCmd = &cobra.Command{
	Use:   "deps <port>",
	Short: "의존성 트리",
	Long: `포트의 의존성 트리를 출력합니다.

예시:
  pal marker deps L2-AdminStructureQueryCompositeService`,
	Args: cobra.ExactArgs(1),
	RunE: runMarkerDeps,
}

var markerGeneratedCmd = &cobra.Command{
	Use:   "generated",
	Short: "Claude 생성 코드 목록",
	Long:  `@pal-generated 마커가 있는 파일 목록을 출력합니다.`,
	RunE:  runMarkerGenerated,
}

var markerIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "마커 인덱싱",
	Long: `코드 마커를 데이터베이스에 인덱싱합니다.

예시:
  pal marker index              # 마커 인덱싱
  pal marker index --link       # 문서와 연결까지`,
	RunE: runMarkerIndex,
}

var markerStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "마커 통계",
	Long:  `인덱싱된 마커 통계를 출력합니다.`,
	RunE:  runMarkerStats,
}

var markerGraphCmd = &cobra.Command{
	Use:   "graph",
	Short: "의존성 그래프",
	Long:  `포트 간 의존성 그래프를 출력합니다.`,
	RunE:  runMarkerGraph,
}

var markerLinkDocs bool

func init() {
	rootCmd.AddCommand(markerCmd)
	markerCmd.AddCommand(markerListCmd)
	markerCmd.AddCommand(markerCheckCmd)
	markerCmd.AddCommand(markerFilesCmd)
	markerCmd.AddCommand(markerDepsCmd)
	markerCmd.AddCommand(markerGeneratedCmd)
	markerCmd.AddCommand(markerIndexCmd)
	markerCmd.AddCommand(markerStatsCmd)
	markerCmd.AddCommand(markerGraphCmd)

	markerListCmd.Flags().StringVar(&markerPort, "port", "", "포트 패턴 (예: L1-*)")
	markerListCmd.Flags().StringVar(&markerDomain, "domain", "", "도메인 필터")
	markerListCmd.Flags().StringVar(&markerLayer, "layer", "", "레이어 필터 (l1, lm, l2)")

	markerCheckCmd.Flags().BoolVar(&markerStrict, "strict", false, "엄격 모드 (layer, domain 필수)")

	markerIndexCmd.Flags().BoolVar(&markerLinkDocs, "link", false, "문서와 자동 연결")
}

func getMarkerService() (*marker.Service, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	return marker.NewService(projectRoot), nil
}

func runMarkerList(cmd *cobra.Command, args []string) error {
	svc, err := getMarkerService()
	if err != nil {
		return err
	}

	var markers []marker.Marker

	if markerPort != "" {
		markers, err = svc.ListByPort(markerPort)
	} else if markerDomain != "" {
		markers, err = svc.ListByDomain(markerDomain)
	} else if markerLayer != "" {
		markers, err = svc.ListByLayer(markerLayer)
	} else {
		markers, err = svc.Scan()
	}

	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(markers)
	}

	if len(markers) == 0 {
		fmt.Println("마커를 찾을 수 없습니다.")
		return nil
	}

	fmt.Printf("📍 PAL 마커 목록 (%d건)\n\n", len(markers))

	layerEmoji := map[string]string{
		"l1": "1️⃣",
		"lm": "🔗",
		"l2": "2️⃣",
	}

	for _, m := range markers {
		emoji := layerEmoji[m.Layer]
		if emoji == "" {
			emoji = "📄"
		}

		relPath := m.FilePath
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, m.FilePath); err == nil {
				relPath = rel
			}
		}

		fmt.Printf("%s %s\n", emoji, m.Port)
		fmt.Printf("   파일: %s:%d\n", relPath, m.Line)
		if m.Domain != "" {
			fmt.Printf("   도메인: %s\n", m.Domain)
		}
		if len(m.Depends) > 0 {
			fmt.Printf("   의존성: %s\n", strings.Join(m.Depends, ", "))
		}
		if m.Generated {
			fmt.Printf("   🤖 Claude 생성\n")
		}
		fmt.Println()
	}

	return nil
}

func runMarkerCheck(cmd *cobra.Command, args []string) error {
	svc, err := getMarkerService()
	if err != nil {
		return err
	}

	issues, err := svc.CheckValidity(markerStrict)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(issues)
	}

	if len(issues) == 0 {
		fmt.Println("✅ 모든 마커가 유효합니다.")
		return nil
	}

	fmt.Printf("⚠️  마커 이슈 (%d건)\n\n", len(issues))

	for _, issue := range issues {
		relPath := issue.Marker.FilePath
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, issue.Marker.FilePath); err == nil {
				relPath = rel
			}
		}

		fmt.Printf("❌ %s\n", issue.Marker.Port)
		fmt.Printf("   파일: %s:%d\n", relPath, issue.Marker.Line)
		fmt.Printf("   이슈: %s\n\n", issue.Message)
	}

	return nil
}

func runMarkerFiles(cmd *cobra.Command, args []string) error {
	portID := args[0]

	svc, err := getMarkerService()
	if err != nil {
		return err
	}

	markers, err := svc.ListByPort(portID)
	if err != nil {
		return err
	}

	if jsonOut {
		files := make([]string, 0, len(markers))
		for _, m := range markers {
			files = append(files, m.FilePath)
		}
		return json.NewEncoder(os.Stdout).Encode(files)
	}

	if len(markers) == 0 {
		fmt.Printf("포트 '%s'를 찾을 수 없습니다.\n", portID)
		return nil
	}

	fmt.Printf("📁 포트 '%s' 파일 목록 (%d건)\n\n", portID, len(markers))

	for _, m := range markers {
		relPath := m.FilePath
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, m.FilePath); err == nil {
				relPath = rel
			}
		}
		fmt.Printf("  %s:%d\n", relPath, m.Line)
	}

	return nil
}

func runMarkerDeps(cmd *cobra.Command, args []string) error {
	portID := args[0]

	svc, err := getMarkerService()
	if err != nil {
		return err
	}

	tree, err := svc.GetDependencyTree(portID)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(tree)
	}

	if len(tree) == 0 {
		fmt.Printf("포트 '%s'의 의존성을 찾을 수 없습니다.\n", portID)
		return nil
	}

	fmt.Printf("🌳 포트 '%s' 의존성 트리\n\n", portID)

	// Print tree
	var printTree func(port string, indent string, visited map[string]bool)
	printTree = func(port string, indent string, visited map[string]bool) {
		if visited[port] {
			fmt.Printf("%s%s (순환 참조)\n", indent, port)
			return
		}
		visited[port] = true

		fmt.Printf("%s%s\n", indent, port)
		deps := tree[port]
		for i, dep := range deps {
			prefix := "├── "
			if i == len(deps)-1 {
				prefix = "└── "
			}
			printTree(dep, indent+prefix, visited)
		}
	}

	printTree(portID, "", make(map[string]bool))

	return nil
}

func runMarkerGenerated(cmd *cobra.Command, args []string) error {
	svc, err := getMarkerService()
	if err != nil {
		return err
	}

	markers, err := svc.ListGenerated()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(markers)
	}

	if len(markers) == 0 {
		fmt.Println("Claude 생성 코드를 찾을 수 없습니다.")
		return nil
	}

	fmt.Printf("🤖 Claude 생성 코드 목록 (%d건)\n\n", len(markers))

	for _, m := range markers {
		relPath := m.FilePath
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, m.FilePath); err == nil {
				relPath = rel
			}
		}

		fmt.Printf("  %s\n", m.Port)
		fmt.Printf("    %s:%d\n", relPath, m.Line)
	}

	return nil
}

func getMarkerIndexer() (*marker.Indexer, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	database, err := db.Open(filepath.Join(projectRoot, ".pal", "pal.db"))
	if err != nil {
		return nil, fmt.Errorf("DB 열기 실패: %w", err)
	}

	return marker.NewIndexer(database, projectRoot), nil
}

func runMarkerIndex(cmd *cobra.Command, args []string) error {
	indexer, err := getMarkerIndexer()
	if err != nil {
		return err
	}

	fmt.Println("📍 코드 마커 인덱싱 중...")

	result, err := indexer.Index()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Printf("\n✅ 인덱싱 완료\n")
	fmt.Printf("   추가: %d건\n", result.Added)
	fmt.Printf("   업데이트: %d건\n", result.Updated)
	fmt.Printf("   삭제: %d건\n", result.Removed)

	if len(result.Errors) > 0 {
		fmt.Printf("   오류: %d건\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Printf("     - %s\n", e)
		}
	}

	// 문서 연결
	if markerLinkDocs {
		fmt.Println("\n📎 문서 연결 중...")
		linked, err := indexer.LinkToDocuments()
		if err != nil {
			fmt.Printf("   연결 실패: %v\n", err)
		} else {
			fmt.Printf("   연결됨: %d건\n", linked)
		}
	}

	return nil
}

func runMarkerStats(cmd *cobra.Command, args []string) error {
	indexer, err := getMarkerIndexer()
	if err != nil {
		return err
	}

	stats, err := indexer.GetStats()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(stats)
	}

	fmt.Printf("📊 마커 통계\n\n")
	fmt.Printf("총 마커: %d개\n", stats.TotalMarkers)
	fmt.Printf("Claude 생성: %d개\n", stats.Generated)
	fmt.Printf("의존성 있음: %d개\n\n", stats.WithDeps)

	if len(stats.ByLayer) > 0 {
		fmt.Println("레이어별:")
		for layer, count := range stats.ByLayer {
			emoji := "📄"
			switch layer {
			case "l1":
				emoji = "1️⃣"
			case "lm":
				emoji = "🔗"
			case "l2":
				emoji = "2️⃣"
			}
			fmt.Printf("  %s %s: %d\n", emoji, layer, count)
		}
		fmt.Println()
	}

	if len(stats.ByDomain) > 0 {
		fmt.Println("도메인별:")
		for domain, count := range stats.ByDomain {
			fmt.Printf("  📁 %s: %d\n", domain, count)
		}
	}

	return nil
}

func runMarkerGraph(cmd *cobra.Command, args []string) error {
	indexer, err := getMarkerIndexer()
	if err != nil {
		return err
	}

	graph, err := indexer.GetDependencyGraph()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(graph)
	}

	if len(graph) == 0 {
		fmt.Println("의존성 그래프가 비어있습니다.")
		fmt.Println("먼저 'pal marker index'를 실행해주세요.")
		return nil
	}

	fmt.Printf("🌳 의존성 그래프 (%d개 노드)\n\n", len(graph))

	for port, deps := range graph {
		fmt.Printf("%s\n", port)
		for i, dep := range deps {
			prefix := "├──"
			if i == len(deps)-1 {
				prefix = "└──"
			}
			fmt.Printf("  %s %s\n", prefix, dep)
		}
		fmt.Println()
	}

	return nil
}
