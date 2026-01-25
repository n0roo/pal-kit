package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "데이터베이스 마이그레이션",
	Long:  `SQLite에서 DuckDB로 데이터베이스를 마이그레이션합니다.`,
}

var migrateToDuckDBCmd = &cobra.Command{
	Use:   "to-duckdb",
	Short: "SQLite → DuckDB 마이그레이션",
	Long: `SQLite 데이터베이스를 DuckDB로 마이그레이션합니다.

마이그레이션 과정:
1. 기존 SQLite 파일 백업
2. DuckDB 스키마 생성
3. 데이터 복사
4. 검증

예시:
  pal migrate to-duckdb              # 기본 DB 마이그레이션
  pal migrate to-duckdb --source ~/.pal/pal.db  # 특정 파일 지정`,
	RunE: runMigrateToDuckDB,
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "현재 DB 상태 확인",
	Long:  `현재 사용 중인 데이터베이스 타입과 상태를 확인합니다.`,
	RunE:  runMigrateStatus,
}

var (
	migrateSource string
	migrateForce  bool
)

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateToDuckDBCmd)
	migrateCmd.AddCommand(migrateStatusCmd)

	migrateToDuckDBCmd.Flags().StringVar(&migrateSource, "source", "", "SQLite DB 파일 경로 (기본: ~/.pal/pal.db)")
	migrateToDuckDBCmd.Flags().BoolVar(&migrateForce, "force", false, "기존 DuckDB 파일 덮어쓰기")
}

func runMigrateToDuckDB(cmd *cobra.Command, args []string) error {
	// 소스 경로 결정
	sqlitePath := migrateSource
	if sqlitePath == "" {
		sqlitePath = GetDBPath()
	}

	// 파일 존재 확인
	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		return fmt.Errorf("SQLite 파일이 없습니다: %s", sqlitePath)
	}

	// DuckDB 경로
	duckdbPath := db.GetDuckDBPath(sqlitePath)

	// 기존 DuckDB 파일 확인
	if _, err := os.Stat(duckdbPath); err == nil && !migrateForce {
		return fmt.Errorf("DuckDB 파일이 이미 존재합니다: %s\n--force 옵션으로 덮어쓸 수 있습니다", duckdbPath)
	}

	fmt.Printf("🔄 마이그레이션 시작...\n")
	fmt.Printf("   소스: %s\n", sqlitePath)
	fmt.Printf("   대상: %s\n", duckdbPath)
	fmt.Println()

	// 마이그레이션 실행
	result, err := db.MigrateSQLiteToDuckDB(sqlitePath, duckdbPath)
	if err != nil {
		return fmt.Errorf("마이그레이션 실패: %w", err)
	}

	// 결과 출력
	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(result)
		return nil
	}

	fmt.Printf("✅ 마이그레이션 완료!\n\n")
	fmt.Printf("📊 결과:\n")
	fmt.Printf("   처리된 테이블: %d개\n", result.TablesProcessed)
	fmt.Println()

	// 테이블별 행 수
	totalRows := 0
	for table, count := range result.RowsMigrated {
		if count > 0 {
			fmt.Printf("   - %s: %d행\n", table, count)
			totalRows += count
		}
	}
	fmt.Printf("\n   총 %d행 마이그레이션됨\n", totalRows)

	// 에러 표시
	if len(result.Errors) > 0 {
		fmt.Printf("\n⚠️  경고:\n")
		for _, e := range result.Errors {
			fmt.Printf("   - %s\n", e)
		}
	}

	fmt.Printf("\n💡 DuckDB를 사용하려면 환경변수를 설정하세요:\n")
	fmt.Printf("   export PAL_DB_TYPE=duckdb\n")
	fmt.Printf("\n   또는 PAL Kit 설정에서:\n")
	fmt.Printf("   pal config set db.type duckdb\n")

	return nil
}

func runMigrateStatus(cmd *cobra.Command, args []string) error {
	basePath := GetDBPath()
	sqlitePath := basePath
	duckdbPath := db.GetDuckDBPath(basePath)

	sqliteExists := false
	duckdbExists := false
	var sqliteSize, duckdbSize int64

	// SQLite 파일 확인
	if info, err := os.Stat(sqlitePath); err == nil {
		sqliteExists = true
		sqliteSize = info.Size()
	}

	// DuckDB 파일 확인
	if info, err := os.Stat(duckdbPath); err == nil {
		duckdbExists = true
		duckdbSize = info.Size()
	}

	// 현재 사용 중인 DB 타입 확인
	currentType := os.Getenv("PAL_DB_TYPE")
	if currentType == "" {
		currentType = "sqlite" // 기본값
	}

	if jsonOut {
		output := map[string]interface{}{
			"current_type":  currentType,
			"sqlite_exists": sqliteExists,
			"sqlite_path":   sqlitePath,
			"sqlite_size":   sqliteSize,
			"duckdb_exists": duckdbExists,
			"duckdb_path":   duckdbPath,
			"duckdb_size":   duckdbSize,
		}
		json.NewEncoder(os.Stdout).Encode(output)
		return nil
	}

	fmt.Printf("📊 데이터베이스 상태\n\n")
	fmt.Printf("현재 사용: %s\n\n", currentType)

	fmt.Printf("SQLite:\n")
	if sqliteExists {
		fmt.Printf("   ✅ 존재: %s\n", sqlitePath)
		fmt.Printf("   📦 크기: %s\n", formatSize(sqliteSize))

		// 버전 확인
		if sqliteDB, err := db.Open(sqlitePath); err == nil {
			if ver, err := sqliteDB.GetVersion(); err == nil {
				fmt.Printf("   🔢 스키마 버전: v%d\n", ver)
			}
			sqliteDB.Close()
		}
	} else {
		fmt.Printf("   ❌ 없음: %s\n", sqlitePath)
	}

	fmt.Printf("\nDuckDB:\n")
	if duckdbExists {
		fmt.Printf("   ✅ 존재: %s\n", duckdbPath)
		fmt.Printf("   📦 크기: %s\n", formatSize(duckdbSize))

		// 버전 확인
		if duckDB, err := db.OpenDuckDB(duckdbPath); err == nil {
			if ver, err := duckDB.GetVersion(); err == nil {
				fmt.Printf("   🔢 스키마 버전: v%d\n", ver)
			}
			duckDB.Close()
		}
	} else {
		fmt.Printf("   ❌ 없음: %s\n", duckdbPath)
	}

	// 권장사항
	fmt.Printf("\n💡 권장사항:\n")
	if !duckdbExists && sqliteExists {
		fmt.Printf("   DuckDB로 마이그레이션하려면: pal migrate to-duckdb\n")
	} else if duckdbExists && currentType != "duckdb" {
		fmt.Printf("   DuckDB 사용하려면: export PAL_DB_TYPE=duckdb\n")
	} else if duckdbExists && currentType == "duckdb" {
		fmt.Printf("   ✅ DuckDB를 사용 중입니다.\n")
	}

	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GetDBPathWithType returns the appropriate DB path based on type setting
func GetDBPathWithType() (string, string) {
	basePath := GetDBPath()

	dbType := os.Getenv("PAL_DB_TYPE")
	if dbType == "" {
		dbType = "sqlite" // 기본값
	}

	if dbType == "duckdb" {
		return db.GetDuckDBPath(basePath), "duckdb"
	}
	return basePath, "sqlite"
}

// OpenDBWithType opens the appropriate database based on type setting
func OpenDBWithType() (*db.DB, error) {
	path, dbType := GetDBPathWithType()

	if dbType == "duckdb" {
		// DuckDB는 현재 *db.DuckDB를 반환하므로 호환성 레이어 필요
		// 임시로 SQLite 폴백
		return db.Open(GetDBPath())
	}

	return db.Open(path)
}
