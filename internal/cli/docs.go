package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/docs"
	"github.com/n0roo/pal-kit/internal/document"
	"github.com/spf13/cobra"
)

var (
	docsMessage    string
	docsOverwrite  bool
	docsAll        bool
	docsType       string
	docsDomain     string
	docsStatus     string
	docsTag        string
	docsMaxTokens  int64
	docsLimit      int
	docsIncludeDeps bool
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "문서 관리",
	Long:  `프로젝트 문서를 관리합니다.`,
}

var docsListCmd = &cobra.Command{
	Use:   "list",
	Short: "관리 문서 목록",
	RunE:  runDocsList,
}

var docsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "문서 상태",
	RunE:  runDocsStatus,
}

var docsShowCmd = &cobra.Command{
	Use:   "show <path>",
	Short: "문서 내용 표시",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsShow,
}

var docsInitCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "프로젝트 문서 초기화",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDocsInit,
}

// 템플릿 관련
var docsTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "템플릿 관리",
}

var docsTemplateListCmd = &cobra.Command{
	Use:   "list",
	Short: "템플릿 목록",
	RunE:  runDocsTemplateList,
}

var docsTemplateApplyCmd = &cobra.Command{
	Use:   "apply <template-name>",
	Short: "템플릿 적용",
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsTemplateApply,
}

// 스냅샷 관련
var docsSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "스냅샷 생성",
	RunE:  runDocsSnapshot,
}

var docsHistoryCmd = &cobra.Command{
	Use:   "history [path]",
	Short: "변경 이력",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDocsHistory,
}

var docsDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "현재 변경사항",
	RunE:  runDocsDiff,
}

var docsRestoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id> [paths...]",
	Short: "이전 버전 복원",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runDocsRestore,
}

// 검증 관련
var docsLintCmd = &cobra.Command{
	Use:   "lint [path]",
	Short: "문서 검증",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDocsLint,
}

// 인덱싱 관련 (document 패키지 사용)
var docsIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "문서 인덱싱",
	Long:  `프로젝트 문서를 스캔하여 인덱스를 갱신합니다.`,
	RunE:  runDocsIndex,
}

var docsSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "문서 검색",
	Long: `문서를 검색합니다.

쿼리 형식:
  type:l1 AND domain:order
  tag:important
  status:draft

플래그:
  --type     문서 타입 (port, convention, agent, l1, l2, lm)
  --domain   도메인 필터
  --status   상태 필터
  --tag      태그 필터`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDocsSearch,
}

var docsPortCmd = &cobra.Command{
	Use:   "port <name>",
	Short: "포트 조회",
	Long:  `포트 이름 또는 별칭으로 포트 문서를 찾습니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDocsPort,
}

var docsStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "문서 통계",
	Long:  `인덱싱된 문서의 통계를 표시합니다.`,
	RunE:  runDocsStats,
}

var docsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "문서 조회",
	Long: `문서 ID로 내용을 조회합니다.

예시:
  pal docs get ports-auth-service
  pal docs get ports-auth-service --summary
  pal docs get ports-auth-service --tokens 2000`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsGet,
}

var docsContextCmd = &cobra.Command{
	Use:   "context <query>",
	Short: "Support Agent용 컨텍스트 조회",
	Long: `Support Agent가 사용하는 형식으로 문서를 검색하고 제공합니다.

예시:
  pal docs context "domain:auth"
  pal docs context "type:convention" --budget 5000`,
	Args: cobra.ExactArgs(1),
	RunE: runDocsContext,
}

var (
	docsGetSummary   bool
	docsGetTokens    int64
	docsTokenBudget  int64
	docsIncludeContent bool
)

func init() {
	rootCmd.AddCommand(docsCmd)

	docsCmd.AddCommand(docsListCmd)
	docsCmd.AddCommand(docsStatusCmd)
	docsCmd.AddCommand(docsShowCmd)
	docsCmd.AddCommand(docsInitCmd)
	docsCmd.AddCommand(docsTemplateCmd)
	docsCmd.AddCommand(docsSnapshotCmd)
	docsCmd.AddCommand(docsHistoryCmd)
	docsCmd.AddCommand(docsDiffCmd)
	docsCmd.AddCommand(docsRestoreCmd)
	docsCmd.AddCommand(docsLintCmd)

	docsTemplateCmd.AddCommand(docsTemplateListCmd)
	docsTemplateCmd.AddCommand(docsTemplateApplyCmd)

	docsSnapshotCmd.Flags().StringVarP(&docsMessage, "message", "m", "", "스냅샷 메시지")
	docsTemplateApplyCmd.Flags().BoolVar(&docsOverwrite, "overwrite", false, "기존 파일 덮어쓰기")
	docsLintCmd.Flags().BoolVar(&docsAll, "all", false, "모든 이슈 표시 (info 포함)")

	// 인덱싱 관련 명령어
	docsCmd.AddCommand(docsIndexCmd)
	docsCmd.AddCommand(docsSearchCmd)
	docsCmd.AddCommand(docsPortCmd)
	docsCmd.AddCommand(docsStatsCmd)
	docsCmd.AddCommand(docsGetCmd)
	docsCmd.AddCommand(docsContextCmd)

	// 검색 플래그
	docsSearchCmd.Flags().StringVar(&docsType, "type", "", "문서 타입 (port, convention, agent)")
	docsSearchCmd.Flags().StringVar(&docsDomain, "domain", "", "도메인 필터")
	docsSearchCmd.Flags().StringVar(&docsStatus, "status", "", "상태 필터")
	docsSearchCmd.Flags().StringVar(&docsTag, "tag", "", "태그 필터")
	docsSearchCmd.Flags().Int64Var(&docsMaxTokens, "max-tokens", 0, "최대 토큰 수 제한")
	docsSearchCmd.Flags().IntVar(&docsLimit, "limit", 20, "결과 수 제한")
	docsSearchCmd.Flags().BoolVar(&docsIncludeContent, "content", false, "내용 포함")

	// 포트 조회 플래그
	docsPortCmd.Flags().BoolVar(&docsIncludeDeps, "deps", false, "의존성 포함")

	// 문서 조회 플래그
	docsGetCmd.Flags().BoolVar(&docsGetSummary, "summary", false, "요약만 표시")
	docsGetCmd.Flags().Int64Var(&docsGetTokens, "tokens", 0, "최대 토큰 수 제한")

	// 컨텍스트 조회 플래그 (Support Agent용)
	docsContextCmd.Flags().Int64Var(&docsTokenBudget, "budget", 5000, "토큰 예산")
}

func getDocsService() (*docs.Service, error) {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}
	return docs.NewService(projectRoot), nil
}

func runDocsList(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	docList, err := svc.List()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(docList)
	}

	if len(docList) == 0 {
		fmt.Println("관리 중인 문서가 없습니다.")
		fmt.Println("\n문서 초기화:")
		fmt.Println("  pal docs init")
		return nil
	}

	fmt.Println("📚 관리 문서 목록")
	fmt.Println()

	typeEmoji := map[docs.DocType]string{
		docs.DocTypeClaude:     "📄",
		docs.DocTypeAgent:      "🤖",
		docs.DocTypePort:       "📦",
		docs.DocTypeRule:       "📏",
		docs.DocTypeTemplate:   "📝",
		docs.DocTypeConvention: "📋",
	}

	statusEmoji := map[docs.DocStatus]string{
		docs.StatusValid:    "✅",
		docs.StatusModified: "📝",
		docs.StatusOutdated: "⚠️",
		docs.StatusInvalid:  "❌",
		docs.StatusNew:      "🆕",
	}

	// 타입별로 그룹화
	byType := make(map[docs.DocType][]docs.Document)
	for _, doc := range docList {
		byType[doc.Type] = append(byType[doc.Type], doc)
	}

	typeOrder := []docs.DocType{
		docs.DocTypeClaude,
		docs.DocTypeAgent,
		docs.DocTypePort,
		docs.DocTypeConvention,
		docs.DocTypeRule,
		docs.DocTypeTemplate,
	}

	for _, docType := range typeOrder {
		docsByType := byType[docType]
		if len(docsByType) == 0 {
			continue
		}

		emoji := typeEmoji[docType]
		fmt.Printf("%s %s (%d)\n", emoji, docType, len(docsByType))
		for _, doc := range docsByType {
			status := statusEmoji[doc.Status]
			fmt.Printf("   %s %s\n", status, doc.RelativePath)
		}
		fmt.Println()
	}

	return nil
}

func runDocsStatus(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	status, err := svc.Status()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(status)
	}

	fmt.Println("📊 문서 상태")
	fmt.Println()

	total := 0
	for _, count := range status {
		total += count
	}

	fmt.Printf("총 문서: %d개\n", total)
	fmt.Println()

	statusEmoji := map[docs.DocStatus]string{
		docs.StatusValid:    "✅",
		docs.StatusModified: "📝",
		docs.StatusOutdated: "⚠️",
		docs.StatusInvalid:  "❌",
		docs.StatusNew:      "🆕",
	}

	statusOrder := []docs.DocStatus{
		docs.StatusValid,
		docs.StatusModified,
		docs.StatusNew,
		docs.StatusOutdated,
		docs.StatusInvalid,
	}

	for _, s := range statusOrder {
		count := status[s]
		if count > 0 {
			fmt.Printf("%s %-10s: %d\n", statusEmoji[s], s, count)
		}
	}

	return nil
}

func runDocsShow(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	content, err := svc.GetContent(args[0])
	if err != nil {
		return err
	}

	fmt.Println(content)
	return nil
}

func runDocsInit(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	projectName := "My Project"
	if len(args) > 0 {
		projectName = args[0]
	}

	created, err := svc.InitProject(projectName)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "initialized",
			"created": created,
		})
	}

	fmt.Println("✅ 프로젝트 문서 초기화 완료")
	fmt.Println()
	fmt.Println("생성된 파일:")
	for _, file := range created {
		fmt.Printf("  📄 %s\n", file)
	}

	if len(created) == 0 {
		fmt.Println("  (이미 초기화됨)")
	}

	fmt.Println()
	fmt.Println("생성된 디렉토리:")
	fmt.Println("  📁 agents/")
	fmt.Println("  📁 ports/")
	fmt.Println("  📁 conventions/")
	fmt.Println("  📁 templates/")

	return nil
}

func runDocsTemplateList(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	templates := svc.ListTemplates()

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(templates)
	}

	fmt.Println("📝 사용 가능한 템플릿")
	fmt.Println()

	for _, t := range templates {
		fmt.Printf("  %-20s %s\n", t.Name, t.Description)
	}

	fmt.Println()
	fmt.Println("사용법:")
	fmt.Println("  pal docs template apply <template-name>")

	return nil
}

func runDocsTemplateApply(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	templateName := args[0]

	// 템플릿 데이터 설정
	data := docs.TemplateData{
		ProjectName: "My Project",
	}

	// 템플릿별 추가 데이터 요청
	switch {
	case strings.Contains(templateName, "agent"):
		fmt.Print("에이전트 ID: ")
		fmt.Scanln(&data.AgentID)
		fmt.Print("에이전트 이름: ")
		fmt.Scanln(&data.AgentName)
	case strings.Contains(templateName, "port"):
		fmt.Print("포트 ID: ")
		fmt.Scanln(&data.PortID)
		fmt.Print("포트 제목: ")
		fmt.Scanln(&data.PortTitle)
	}

	fileName, err := svc.CreateFromTemplate(templateName, data, docsOverwrite)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "created",
			"file":   fileName,
		})
	}

	fmt.Printf("✅ 파일 생성: %s\n", fileName)

	return nil
}

func runDocsSnapshot(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	message := docsMessage
	if message == "" {
		message = "Manual snapshot"
	}

	snapshot, err := svc.CreateSnapshot(message)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(snapshot)
	}

	fmt.Printf("✅ 스냅샷 생성: %s\n", snapshot.ID)
	fmt.Printf("   문서: %d개\n", len(snapshot.Documents))
	fmt.Printf("   메시지: %s\n", snapshot.Message)

	return nil
}

func runDocsHistory(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	if len(args) > 0 {
		// 특정 파일의 히스토리
		history, err := svc.GetDocumentHistory(args[0])
		if err != nil {
			return err
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(history)
		}

		fmt.Printf("📜 %s 변경 이력\n", args[0])
		fmt.Println()

		if len(history) == 0 {
			fmt.Println("변경 이력이 없습니다.")
			return nil
		}

		for i, h := range history {
			fmt.Printf("%d. %s (%.0f bytes)\n", i+1, h.Hash[:8], float64(h.Size))
		}
	} else {
		// 모든 스냅샷 목록
		snapshots, err := svc.ListSnapshots()
		if err != nil {
			return err
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(snapshots)
		}

		fmt.Println("📜 스냅샷 목록")
		fmt.Println()

		if len(snapshots) == 0 {
			fmt.Println("스냅샷이 없습니다.")
			fmt.Println("\n스냅샷 생성:")
			fmt.Println("  pal docs snapshot -m \"message\"")
			return nil
		}

		for _, s := range snapshots {
			fmt.Printf("  %s  %s (%d docs)\n",
				s.ID, s.Message, len(s.Documents))
		}
	}

	return nil
}

func runDocsDiff(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	diff, err := svc.DiffWithLatest()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(diff)
	}

	hasChanges := len(diff.Added) > 0 || len(diff.Modified) > 0 || len(diff.Deleted) > 0

	if !hasChanges {
		fmt.Println("✅ 변경사항 없음")
		return nil
	}

	fmt.Println("📝 변경사항")
	fmt.Println()

	if len(diff.Added) > 0 {
		fmt.Println("추가됨:")
		for _, path := range diff.Added {
			fmt.Printf("  🆕 %s\n", path)
		}
		fmt.Println()
	}

	if len(diff.Modified) > 0 {
		fmt.Println("수정됨:")
		for _, path := range diff.Modified {
			fmt.Printf("  📝 %s\n", path)
		}
		fmt.Println()
	}

	if len(diff.Deleted) > 0 {
		fmt.Println("삭제됨:")
		for _, path := range diff.Deleted {
			fmt.Printf("  ❌ %s\n", path)
		}
	}

	return nil
}

func runDocsRestore(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	snapshotID := args[0]
	paths := args[1:]

	if err := svc.RestoreSnapshot(snapshotID, paths); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":   "restored",
			"snapshot": snapshotID,
			"paths":    paths,
		})
	}

	fmt.Printf("✅ 스냅샷 %s에서 복원 완료\n", snapshotID)
	if len(paths) > 0 {
		fmt.Println("복원된 파일:")
		for _, p := range paths {
			fmt.Printf("  📄 %s\n", p)
		}
	} else {
		fmt.Println("  (모든 파일)")
	}

	return nil
}

func runDocsLint(cmd *cobra.Command, args []string) error {
	svc, err := getDocsService()
	if err != nil {
		return err
	}

	opts := &docs.LintOptions{
		IgnoreInfo: !docsAll,
	}

	var results []docs.LintResult

	if len(args) > 0 {
		result, err := svc.LintFile(args[0], opts)
		if err != nil {
			return err
		}
		results = []docs.LintResult{*result}
	} else {
		var err error
		results, err = svc.Lint(opts)
		if err != nil {
			return err
		}
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	summary := docs.LintSummary(results)
	hasIssues := summary[docs.SeverityError] > 0 || summary[docs.SeverityWarning] > 0

	fmt.Println("🔍 문서 검증 결과")
	fmt.Println()

	severityEmoji := map[docs.LintSeverity]string{
		docs.SeverityError:   "❌",
		docs.SeverityWarning: "⚠️",
		docs.SeverityInfo:    "ℹ️",
	}

	for _, result := range results {
		if len(result.Issues) == 0 {
			continue
		}

		fmt.Printf("📄 %s\n", result.Path)
		for _, issue := range result.Issues {
			emoji := severityEmoji[issue.Severity]
			fmt.Printf("   %s [%s] %s\n", emoji, issue.Rule, issue.Message)
		}
		fmt.Println()
	}

	fmt.Printf("요약: ❌ %d errors, ⚠️ %d warnings",
		summary[docs.SeverityError], summary[docs.SeverityWarning])
	if docsAll {
		fmt.Printf(", ℹ️ %d info", summary[docs.SeverityInfo])
	}
	fmt.Println()

	if !hasIssues {
		fmt.Println("\n✅ 모든 문서가 유효합니다!")
	}

	return nil
}

// =====================================
// Document Indexing Commands
// =====================================

func getDocumentService() (*document.Service, error) {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	database, err := db.Open(config.GlobalDBPath())
	if err != nil {
		return nil, fmt.Errorf("DB 연결 실패: %w", err)
	}

	return document.NewService(database, projectRoot), nil
}

func runDocsIndex(cmd *cobra.Command, args []string) error {
	svc, err := getDocumentService()
	if err != nil {
		return err
	}

	fmt.Println("📚 문서 인덱싱 중...")

	result, err := svc.Index()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	fmt.Println()
	fmt.Printf("✅ 인덱싱 완료\n")
	fmt.Printf("   추가: %d\n", result.Added)
	fmt.Printf("   갱신: %d\n", result.Updated)
	fmt.Printf("   제거: %d\n", result.Removed)

	if len(result.Errors) > 0 {
		fmt.Println("\n⚠️ 오류:")
		for _, e := range result.Errors {
			fmt.Printf("   %s\n", e)
		}
	}

	return nil
}

func runDocsSearch(cmd *cobra.Command, args []string) error {
	svc, err := getDocumentService()
	if err != nil {
		return err
	}

	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	// 쿼리 문자열에서 필터 파싱
	filters := document.ParseQueryString(query)

	// 플래그로 지정된 필터 적용 (우선)
	if docsType != "" {
		filters.Type = docsType
	}
	if docsDomain != "" {
		filters.Domain = docsDomain
	}
	if docsStatus != "" {
		filters.Status = docsStatus
	}
	if docsTag != "" {
		filters.Tag = docsTag
	}
	if docsMaxTokens > 0 {
		filters.MaxTokens = docsMaxTokens
	}
	if docsLimit > 0 {
		filters.Limit = docsLimit
	}

	// 필터 패턴을 제거한 순수 검색어 추출
	cleanQuery := document.CleanQueryString(query)

	docs, err := svc.Search(cleanQuery, filters)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(docs)
	}

	if len(docs) == 0 {
		fmt.Println("검색 결과가 없습니다.")
		fmt.Println("\n먼저 인덱싱을 실행해보세요:")
		fmt.Println("  pal docs index")
		return nil
	}

	fmt.Printf("📚 검색 결과 (%d건)\n\n", len(docs))

	typeEmoji := map[string]string{
		"port":       "📦",
		"convention": "📋",
		"agent":      "🤖",
		"l1":         "1️⃣",
		"l2":         "2️⃣",
		"lm":         "🔗",
		"template":   "📝",
		"session":    "💬",
		"adr":        "📄",
	}

	statusEmoji := map[string]string{
		"active":   "✅",
		"draft":    "📝",
		"running":  "🔄",
		"complete": "✔️",
		"archived": "📦",
	}

	for _, d := range docs {
		emoji := typeEmoji[d.Type]
		if emoji == "" {
			emoji = "📄"
		}

		status := statusEmoji[d.Status]
		if status == "" {
			status = "⚪"
		}

		fmt.Printf("%s %s %s\n", emoji, status, d.Path)
		if d.Domain != "" {
			fmt.Printf("   도메인: %s | 토큰: %d\n", d.Domain, d.Tokens)
		} else {
			fmt.Printf("   토큰: %d\n", d.Tokens)
		}
	}

	return nil
}

func runDocsPort(cmd *cobra.Command, args []string) error {
	svc, err := getDocumentService()
	if err != nil {
		return err
	}

	portName := args[0]

	port, err := svc.FindPort(portName)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(port)
	}

	fmt.Printf("📦 포트: %s\n\n", port.ID)
	fmt.Printf("경로: %s\n", port.Path)
	if port.Domain != "" {
		fmt.Printf("도메인: %s\n", port.Domain)
	}
	fmt.Printf("상태: %s\n", port.Status)
	fmt.Printf("토큰: %d\n", port.Tokens)

	if len(port.Tags) > 0 {
		fmt.Printf("태그: %s\n", strings.Join(port.Tags, ", "))
	}

	// 의존성 포함
	if docsIncludeDeps {
		deps, err := svc.GetLinksFrom(port.ID)
		if err == nil && len(deps) > 0 {
			fmt.Println("\n의존성:")
			for _, d := range deps {
				fmt.Printf("  → %s (%s)\n", d.ID, d.Type)
			}
		}

		linked, err := svc.GetLinksTo(port.ID)
		if err == nil && len(linked) > 0 {
			fmt.Println("\n참조됨:")
			for _, d := range linked {
				fmt.Printf("  ← %s (%s)\n", d.ID, d.Type)
			}
		}
	}

	// 내용 미리보기
	content, err := svc.GetContent(port.ID)
	if err == nil && !jsonOut {
		fmt.Println("\n--- 내용 미리보기 ---")
		lines := strings.Split(content, "\n")
		maxLines := 20
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		for i := 0; i < maxLines; i++ {
			fmt.Println(lines[i])
		}
		if len(lines) > maxLines {
			fmt.Printf("\n... (%d줄 더 있음)\n", len(lines)-maxLines)
		}
	}

	return nil
}

func runDocsStats(cmd *cobra.Command, args []string) error {
	svc, err := getDocumentService()
	if err != nil {
		return err
	}

	stats, err := svc.GetStats()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(stats)
	}

	fmt.Println("📊 문서 통계")
	fmt.Println()
	fmt.Printf("총 문서: %d개\n", stats.TotalDocs)
	fmt.Printf("총 토큰: %d (약 %.1fK)\n", stats.TotalTokens, float64(stats.TotalTokens)/1000)
	fmt.Println()

	if len(stats.ByType) > 0 {
		fmt.Println("타입별:")
		typeEmoji := map[string]string{
			"port":       "📦",
			"convention": "📋",
			"agent":      "🤖",
			"session":    "💬",
			"adr":        "📄",
		}
		for t, c := range stats.ByType {
			emoji := typeEmoji[t]
			if emoji == "" {
				emoji = "📄"
			}
			fmt.Printf("  %s %-12s: %d\n", emoji, t, c)
		}
		fmt.Println()
	}

	if len(stats.ByStatus) > 0 {
		fmt.Println("상태별:")
		statusEmoji := map[string]string{
			"active":   "✅",
			"draft":    "📝",
			"running":  "🔄",
			"complete": "✔️",
		}
		for s, c := range stats.ByStatus {
			emoji := statusEmoji[s]
			if emoji == "" {
				emoji = "⚪"
			}
			fmt.Printf("  %s %-12s: %d\n", emoji, s, c)
		}
		fmt.Println()
	}

	if len(stats.ByDomain) > 0 {
		fmt.Println("도메인별:")
		for d, c := range stats.ByDomain {
			fmt.Printf("  🏷️ %-12s: %d\n", d, c)
		}
	}

	return nil
}

func runDocsGet(cmd *cobra.Command, args []string) error {
	svc, err := getDocumentService()
	if err != nil {
		return err
	}

	docID := args[0]

	doc, err := svc.Get(docID)
	if err != nil {
		return err
	}

	content, err := svc.GetContent(docID)
	if err != nil {
		return err
	}

	if jsonOut {
		result := map[string]interface{}{
			"document": doc,
			"content":  content,
			"tokens":   doc.Tokens,
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	}

	// 토큰 제한 처리
	if docsGetTokens > 0 && doc.Tokens > docsGetTokens {
		// 토큰 제한 내로 내용 자르기
		maxChars := docsGetTokens * 4 // 대략적인 문자 수
		if int64(len(content)) > maxChars {
			content = content[:maxChars] + "\n\n... (토큰 제한으로 잘림)"
		}
	}

	// 요약 모드
	if docsGetSummary {
		fmt.Printf("## 문서: %s\n\n", doc.ID)
		fmt.Printf("**경로**: %s\n", doc.Path)
		fmt.Printf("**타입**: %s\n", doc.Type)
		if doc.Domain != "" {
			fmt.Printf("**도메인**: %s\n", doc.Domain)
		}
		fmt.Printf("**상태**: %s\n", doc.Status)
		fmt.Printf("**토큰**: %d\n\n", doc.Tokens)

		// 내용의 첫 부분만 요약으로 제공
		lines := strings.Split(content, "\n")
		maxLines := 30
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		fmt.Println("### 요약")
		for i := 0; i < maxLines; i++ {
			fmt.Println(lines[i])
		}
		if len(lines) > maxLines {
			fmt.Printf("\n... (%d줄 더 있음)\n", len(lines)-maxLines)
		}
		return nil
	}

	// 전체 내용 출력
	fmt.Println(content)

	return nil
}

func runDocsContext(cmd *cobra.Command, args []string) error {
	svc, err := getDocumentService()
	if err != nil {
		return err
	}

	query := args[0]

	// 쿼리 문자열에서 필터 파싱
	filters := document.ParseQueryString(query)
	cleanQuery := document.CleanQueryString(query)

	// 검색 실행
	docs, err := svc.Search(cleanQuery, filters)
	if err != nil {
		return err
	}

	if len(docs) == 0 {
		fmt.Println("## 검색 결과\n\n검색 결과가 없습니다.")
		return nil
	}

	// Support Agent 형식으로 출력
	fmt.Println("## 검색 결과")
	fmt.Printf("\n### 관련 문서 (%d건)\n", len(docs))

	// 토큰 예산 내 문서 선택
	var selectedDocs []document.Document
	var totalTokens int64

	for _, d := range docs {
		if totalTokens+d.Tokens <= docsTokenBudget {
			selectedDocs = append(selectedDocs, d)
			totalTokens += d.Tokens
		}
	}

	// 문서 목록 출력
	for _, d := range selectedDocs {
		summary := ""
		if d.Summary.Valid {
			summary = d.Summary.String
		}
		if summary == "" {
			summary = fmt.Sprintf("토큰: %d", d.Tokens)
		}
		fmt.Printf("- **%s**: %s\n", d.Path, summary)
	}

	fmt.Println("\n### 핵심 내용")

	// 선택된 문서들의 내용 제공
	for _, d := range selectedDocs {
		content, err := svc.GetContent(d.ID)
		if err != nil {
			continue
		}

		fmt.Printf("\n#### %s\n", d.Path)

		// 문서 크기에 따라 요약 또는 전체 제공
		if d.Tokens > 2000 {
			// 요약 제공 (첫 50줄)
			lines := strings.Split(content, "\n")
			maxLines := 50
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				fmt.Println(lines[i])
			}
			if len(lines) > maxLines {
				fmt.Printf("\n... (요약됨, 전체: %d줄)\n", len(lines))
			}
		} else {
			fmt.Println(content)
		}
	}

	// 참고 정보
	fmt.Println("\n### 참고")
	fmt.Printf("- 토큰 사용: ~%d / %d\n", totalTokens, docsTokenBudget)
	if len(docs) > len(selectedDocs) {
		fmt.Printf("- 예산 초과로 제외된 문서: %d건\n", len(docs)-len(selectedDocs))
	}
	fmt.Println("- 추가 문서가 필요하면 요청해주세요")

	return nil
}
