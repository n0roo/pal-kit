package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/docs"
	"github.com/spf13/cobra"
)

var (
	docsMessage   string
	docsOverwrite bool
	docsAll       bool
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
