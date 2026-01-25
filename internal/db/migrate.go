package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MigrationResult contains migration statistics
type MigrationResult struct {
	TablesProcessed int
	RowsMigrated    map[string]int
	Errors          []string
}

// MigrateSQLiteToDuckDB migrates data from SQLite to DuckDB
func MigrateSQLiteToDuckDB(sqlitePath, duckdbPath string) (*MigrationResult, error) {
	result := &MigrationResult{
		RowsMigrated: make(map[string]int),
	}

	// SQLite 열기
	sqliteDB, err := Open(sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("SQLite 열기 실패: %w", err)
	}
	defer sqliteDB.Close()

	// 기존 DuckDB 파일 백업
	if _, err := os.Stat(duckdbPath); err == nil {
		backupPath := duckdbPath + ".backup"
		os.Rename(duckdbPath, backupPath)
	}

	// DuckDB 열기 (새로 생성)
	duckDB, err := OpenDuckDB(duckdbPath)
	if err != nil {
		return nil, fmt.Errorf("DuckDB 열기 실패: %w", err)
	}
	defer duckDB.Close()

	// 마이그레이션할 테이블 목록
	tables := []string{
		"metadata",
		"locks",
		"projects",
		"sessions",
		"session_events",
		"session_attention",
		"compact_events",
		"compactions",
		"ports",
		"port_dependencies",
		"port_handoffs",
		"agents",
		"agent_versions",
		"agent_performance",
		"messages",
		"orchestration_ports",
		"pipelines",
		"pipeline_ports",
		"worker_sessions",
		"build_outputs",
		"escalations",
		"documents",
		"document_tags",
		"document_links",
		"code_markers",
		"code_marker_deps",
		"marker_port_links",
		"file_manifests",
		"file_changes",
		"environments",
		"sync_meta",
		"sync_history",
	}

	for _, table := range tables {
		count, err := migrateTable(sqliteDB.DB, duckDB.DB, table)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", table, err))
			continue
		}
		result.RowsMigrated[table] = count
		result.TablesProcessed++
	}

	return result, nil
}

// migrateTable migrates a single table from SQLite to DuckDB
func migrateTable(sqlite, duckdb *sql.DB, tableName string) (int, error) {
	// SQLite에서 테이블 존재 확인
	var exists int
	err := sqlite.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tableName).Scan(&exists)
	if err != nil || exists == 0 {
		return 0, nil // 테이블이 없으면 스킵
	}

	// 컬럼 정보 가져오기
	rows, err := sqlite.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, tableName))
	if err != nil {
		return 0, err
	}

	var columns []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			continue
		}
		columns = append(columns, name)
	}
	rows.Close()

	if len(columns) == 0 {
		return 0, nil
	}

	// DuckDB에서 해당 테이블의 컬럼 확인
	duckColumns := make(map[string]bool)
	duckRows, err := duckdb.Query(fmt.Sprintf(`SELECT column_name FROM information_schema.columns WHERE table_name = '%s'`, tableName))
	if err == nil {
		for duckRows.Next() {
			var col string
			duckRows.Scan(&col)
			duckColumns[col] = true
		}
		duckRows.Close()
	}

	// 공통 컬럼만 사용
	var commonColumns []string
	for _, col := range columns {
		if duckColumns[col] {
			commonColumns = append(commonColumns, col)
		}
	}

	if len(commonColumns) == 0 {
		return 0, nil
	}

	columnList := strings.Join(commonColumns, ", ")
	placeholders := strings.Repeat("?, ", len(commonColumns))
	placeholders = placeholders[:len(placeholders)-2] // 마지막 ", " 제거

	// 데이터 조회
	selectQuery := fmt.Sprintf(`SELECT %s FROM %s`, columnList, tableName)
	dataRows, err := sqlite.Query(selectQuery)
	if err != nil {
		return 0, err
	}
	defer dataRows.Close()

	// DuckDB에 삽입
	insertQuery := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, tableName, columnList, placeholders)

	count := 0
	for dataRows.Next() {
		// 동적으로 값 읽기
		values := make([]interface{}, len(commonColumns))
		valuePtrs := make([]interface{}, len(commonColumns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := dataRows.Scan(valuePtrs...); err != nil {
			continue
		}

		// DuckDB에 삽입 (중복 무시)
		_, err := duckdb.Exec(insertQuery, values...)
		if err != nil {
			// 중복 키 에러는 무시
			if !strings.Contains(err.Error(), "UNIQUE") && !strings.Contains(err.Error(), "duplicate") {
				continue
			}
		}
		count++
	}

	return count, nil
}

// AutoSelectDB opens the appropriate database based on file existence and type
func AutoSelectDB(basePath string) (interface{}, string, error) {
	duckdbPath := strings.TrimSuffix(basePath, ".db") + ".duckdb"
	sqlitePath := basePath

	// DuckDB 파일이 있으면 DuckDB 사용
	if _, err := os.Stat(duckdbPath); err == nil {
		db, err := OpenDuckDB(duckdbPath)
		return db, "duckdb", err
	}

	// SQLite 파일이 있으면 SQLite 사용
	if _, err := os.Stat(sqlitePath); err == nil {
		db, err := Open(sqlitePath)
		return db, "sqlite", err
	}

	// 둘 다 없으면 DuckDB 새로 생성 (기본)
	db, err := OpenDuckDB(duckdbPath)
	return db, "duckdb", err
}

// GetDuckDBPath returns the DuckDB path for a given base path
func GetDuckDBPath(basePath string) string {
	return strings.TrimSuffix(basePath, ".db") + ".duckdb"
}

// BackupAndMigrate creates a backup and migrates to DuckDB
func BackupAndMigrate(sqlitePath string) (*MigrationResult, error) {
	// 백업 생성
	backupPath := sqlitePath + ".backup." + filepath.Base(sqlitePath)
	
	src, err := os.Open(sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("원본 파일 열기 실패: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return nil, fmt.Errorf("백업 파일 생성 실패: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return nil, fmt.Errorf("백업 복사 실패: %w", err)
	}

	// DuckDB 경로
	duckdbPath := GetDuckDBPath(sqlitePath)

	// 마이그레이션 실행
	return MigrateSQLiteToDuckDB(sqlitePath, duckdbPath)
}
