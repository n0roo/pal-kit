package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/env"
	"github.com/n0roo/pal-kit/internal/path"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "경로 변환 및 분석",
	Long: `논리 경로($workspace, $claude_data 등)와 절대 경로 간 변환을 수행합니다.

논리 경로를 사용하면 다른 환경에서도 동일한 프로젝트를 참조할 수 있습니다.`,
}

var pathToLogicalCmd = &cobra.Command{
	Use:   "to-logical <absolute-path>",
	Short: "절대 경로를 논리 경로로 변환",
	Args:  cobra.ExactArgs(1),
	RunE:  runPathToLogical,
}

var pathToAbsoluteCmd = &cobra.Command{
	Use:   "to-absolute <logical-path>",
	Short: "논리 경로를 절대 경로로 변환",
	Args:  cobra.ExactArgs(1),
	RunE:  runPathToAbsolute,
}

var pathAnalyzeCmd = &cobra.Command{
	Use:   "analyze <path>",
	Short: "경로 분석",
	Long:  `경로를 분석하여 논리 경로, 절대 경로, 변수, 해석 가능 여부를 표시합니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPathAnalyze,
}

var pathMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "기존 데이터를 논리 경로로 마이그레이션",
	Long: `DB에 저장된 기존 절대 경로를 논리 경로로 변환합니다.
이 작업은 다중 환경 동기화를 위한 준비 단계입니다.`,
	RunE: runPathMigrate,
}

var (
	pathMigrateDryRun bool
)

func init() {
	rootCmd.AddCommand(pathCmd)

	pathCmd.AddCommand(pathToLogicalCmd)
	pathCmd.AddCommand(pathToAbsoluteCmd)
	pathCmd.AddCommand(pathAnalyzeCmd)
	pathCmd.AddCommand(pathMigrateCmd)

	pathMigrateCmd.Flags().BoolVar(&pathMigrateDryRun, "dry-run", false, "실제 변경 없이 변환 결과만 표시")
}

func getResolver() (*path.Resolver, *db.DB, error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, fmt.Errorf("DB 열기 실패: %w", err)
	}

	envSvc := env.NewService(database)
	resolver := path.NewResolver(envSvc)

	return resolver, database, nil
}

func runPathToLogical(cmd *cobra.Command, args []string) error {
	resolver, database, err := getResolver()
	if err != nil {
		return err
	}
	defer database.Close()

	absPath := args[0]
	logical, err := resolver.ToLogical(absPath)
	if err != nil {
		return fmt.Errorf("변환 실패: %w", err)
	}

	if IsJSON() {
		return printJSON(map[string]string{
			"input":  absPath,
			"output": logical,
		})
	}

	fmt.Println(logical)
	return nil
}

func runPathToAbsolute(cmd *cobra.Command, args []string) error {
	resolver, database, err := getResolver()
	if err != nil {
		return err
	}
	defer database.Close()

	logicalPath := args[0]
	absolute, err := resolver.ToAbsolute(logicalPath)
	if err != nil {
		return fmt.Errorf("변환 실패: %w", err)
	}

	if IsJSON() {
		return printJSON(map[string]string{
			"input":  logicalPath,
			"output": absolute,
		})
	}

	fmt.Println(absolute)
	return nil
}

func runPathAnalyze(cmd *cobra.Command, args []string) error {
	resolver, database, err := getResolver()
	if err != nil {
		return err
	}
	defer database.Close()

	inputPath := args[0]
	info, err := resolver.Analyze(inputPath)
	if err != nil {
		return fmt.Errorf("분석 실패: %w", err)
	}

	if IsJSON() {
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("경로 분석: %s\n", info.Original)
	fmt.Printf("  논리 경로: %s\n", info.Logical)
	fmt.Printf("  절대 경로: %s\n", info.Absolute)
	if info.Variable != "" {
		fmt.Printf("  변수: %s\n", info.Variable)
	}
	fmt.Printf("  해석 가능: %v\n", info.Resolvable)

	return nil
}

func runPathMigrate(cmd *cobra.Command, args []string) error {
	resolver, database, err := getResolver()
	if err != nil {
		return err
	}
	defer database.Close()

	// Collect paths to migrate
	migrations := []struct {
		table  string
		column string
		query  string
	}{
		{"sessions", "project_root", "SELECT id, project_root FROM sessions WHERE project_root IS NOT NULL AND project_root != '' AND project_root NOT LIKE '$%'"},
		{"sessions", "cwd", "SELECT id, cwd FROM sessions WHERE cwd IS NOT NULL AND cwd != '' AND cwd NOT LIKE '$%'"},
		{"sessions", "jsonl_path", "SELECT id, jsonl_path FROM sessions WHERE jsonl_path IS NOT NULL AND jsonl_path != '' AND jsonl_path NOT LIKE '$%'"},
		{"sessions", "transcript_path", "SELECT id, transcript_path FROM sessions WHERE transcript_path IS NOT NULL AND transcript_path != '' AND transcript_path NOT LIKE '$%'"},
		{"projects", "root", "SELECT root, name FROM projects WHERE root NOT LIKE '$%'"},
	}

	type migrationItem struct {
		Table    string
		Column   string
		ID       string
		Original string
		Logical  string
	}

	var items []migrationItem
	var errors []string

	for _, m := range migrations {
		rows, err := database.Query(m.query)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s.%s: %v", m.table, m.column, err))
			continue
		}

		for rows.Next() {
			var id, value string
			if err := rows.Scan(&id, &value); err != nil {
				continue
			}

			logical, err := resolver.ToLogical(value)
			if err != nil {
				continue
			}

			// Only add if conversion resulted in a change
			if logical != value {
				items = append(items, migrationItem{
					Table:    m.table,
					Column:   m.column,
					ID:       id,
					Original: value,
					Logical:  logical,
				})
			}
		}
		rows.Close()
	}

	if len(items) == 0 {
		fmt.Println("마이그레이션할 경로가 없습니다.")
		return nil
	}

	if IsJSON() {
		return printJSON(map[string]interface{}{
			"dry_run": pathMigrateDryRun,
			"count":   len(items),
			"items":   items,
			"errors":  errors,
		})
	}

	// Display migration plan
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TABLE\tCOLUMN\tID\tORIGINAL\tLOGICAL")
	fmt.Fprintln(w, "-----\t------\t--\t--------\t-------")

	for _, item := range items {
		// Truncate long paths
		original := truncatePath(item.Original, 30)
		logical := truncatePath(item.Logical, 30)
		id := item.ID
		if len(id) > 8 {
			id = id[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			item.Table, item.Column, id, original, logical)
	}
	w.Flush()

	fmt.Printf("\n총 %d개 경로 마이그레이션 예정\n", len(items))

	if pathMigrateDryRun {
		fmt.Println("\n--dry-run 모드: 실제 변경 없음")
		return nil
	}

	// Execute migration
	fmt.Println("\n마이그레이션 실행 중...")

	successCount := 0
	for _, item := range items {
		var query string
		var args []interface{}

		if item.Table == "projects" && item.Column == "root" {
			// projects table uses root as primary key, need special handling
			query = "UPDATE projects SET logical_root = ? WHERE root = ?"
			args = []interface{}{item.Logical, item.ID}
		} else {
			query = fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", item.Table, item.Column)
			args = []interface{}{item.Logical, item.ID}
		}

		_, err := database.Exec(query, args...)
		if err != nil {
			fmt.Printf("  실패: %s.%s (ID: %s): %v\n", item.Table, item.Column, item.ID, err)
			continue
		}
		successCount++
	}

	fmt.Printf("\n마이그레이션 완료: %d/%d 성공\n", successCount, len(items))

	return nil
}

func truncatePath(p string, maxLen int) string {
	if len(p) <= maxLen {
		return p
	}
	return "..." + p[len(p)-maxLen+3:]
}
