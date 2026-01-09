package db

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDB를 위한 임시 DB 생성 헬퍼
func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()
	
	// 임시 디렉토리 생성
	tmpDir, err := os.MkdirTemp("", "pal-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}
	
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("DB 열기 실패: %v", err)
	}
	
	if err := db.Init(); err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("DB 초기화 실패: %v", err)
	}
	
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
	
	return db, cleanup
}

func TestOpen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("DB 열기 실패: %v", err)
	}
	defer db.Close()
	
	// 파일이 생성되었는지 확인
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("DB 파일이 생성되지 않음")
	}
}

func TestInit(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// 테이블 존재 확인
	tables := []string{"sessions", "ports", "locks", "escalations", "pipelines", "pipeline_ports", "port_dependencies"}
	
	for _, table := range tables {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("테이블 %s가 존재하지 않음: %v", table, err)
		}
	}
}

func TestGetVersion(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	version, err := db.GetVersion()
	if err != nil {
		t.Fatalf("버전 조회 실패: %v", err)
	}
	
	if version < 1 {
		t.Errorf("버전이 1 이상이어야 함, got: %d", version)
	}
}

func TestExec(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// INSERT 테스트
	result, err := db.Exec(`INSERT INTO ports (id, status) VALUES (?, ?)`, "test-port", "pending")
	if err != nil {
		t.Fatalf("INSERT 실패: %v", err)
	}
	
	affected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("RowsAffected 실패: %v", err)
	}
	
	if affected != 1 {
		t.Errorf("RowsAffected = %d, want 1", affected)
	}
}

func TestQuery(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// 데이터 삽입
	db.Exec(`INSERT INTO ports (id, title, status) VALUES (?, ?, ?)`, "port-1", "First", "pending")
	db.Exec(`INSERT INTO ports (id, title, status) VALUES (?, ?, ?)`, "port-2", "Second", "running")
	
	// Query 테스트
	rows, err := db.Query(`SELECT id, status FROM ports ORDER BY id`)
	if err != nil {
		t.Fatalf("Query 실패: %v", err)
	}
	defer rows.Close()
	
	var results []struct {
		ID     string
		Status string
	}
	
	for rows.Next() {
		var r struct {
			ID     string
			Status string
		}
		if err := rows.Scan(&r.ID, &r.Status); err != nil {
			t.Fatalf("Scan 실패: %v", err)
		}
		results = append(results, r)
	}
	
	if len(results) != 2 {
		t.Errorf("결과 수 = %d, want 2", len(results))
	}
	
	if results[0].ID != "port-1" || results[0].Status != "pending" {
		t.Errorf("첫 번째 결과 불일치: %+v", results[0])
	}
}

func TestQueryRow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// 데이터 삽입
	db.Exec(`INSERT INTO ports (id, title, status) VALUES (?, ?, ?)`, "test-port", "Test Port", "running")
	
	// QueryRow 테스트
	var id, title, status string
	err := db.QueryRow(`SELECT id, title, status FROM ports WHERE id = ?`, "test-port").Scan(&id, &title, &status)
	if err != nil {
		t.Fatalf("QueryRow 실패: %v", err)
	}
	
	if id != "test-port" {
		t.Errorf("id = %s, want test-port", id)
	}
	if title != "Test Port" {
		t.Errorf("title = %s, want Test Port", title)
	}
	if status != "running" {
		t.Errorf("status = %s, want running", status)
	}
}

func TestMigration(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// v2 컬럼 존재 확인 (session_type, parent_session)
	var sessionType string
	err := db.QueryRow(`SELECT session_type FROM sessions LIMIT 0`).Scan(&sessionType)
	// 행이 없어도 컬럼이 존재하면 에러가 아님
	if err != nil && err.Error() != "sql: no rows in result set" {
		// 컬럼이 없으면 "no such column" 에러 발생
		if err.Error() == "no such column: session_type" {
			t.Error("session_type 컬럼이 마이그레이션되지 않음")
		}
	}
}

func TestClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("DB 열기 실패: %v", err)
	}
	
	db.Init()
	
	// Close 후 쿼리 실행 시 에러 확인
	db.Close()
	
	_, err = db.Exec(`SELECT 1`)
	if err == nil {
		t.Error("Close 후에도 쿼리가 실행됨")
	}
}
