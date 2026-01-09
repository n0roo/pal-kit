package port

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n0roo/pal-kit/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("DB 열기 실패: %v", err)
	}

	if err := database.Init(); err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("DB 초기화 실패: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

func TestCreate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	err := svc.Create("port-001", "Order Entity", "ports/port-001.md")
	if err != nil {
		t.Fatalf("포트 생성 실패: %v", err)
	}

	port, err := svc.Get("port-001")
	if err != nil {
		t.Fatalf("포트 조회 실패: %v", err)
	}

	if port.ID != "port-001" {
		t.Errorf("ID = %s, want port-001", port.ID)
	}
	if !port.Title.Valid || port.Title.String != "Order Entity" {
		t.Errorf("Title = %v, want Order Entity", port.Title)
	}
	if port.Status != "pending" {
		t.Errorf("Status = %s, want pending", port.Status)
	}
}

func TestCreate_Duplicate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	svc.Create("port-001", "First", "")
	err := svc.Create("port-001", "Second", "")
	if err == nil {
		t.Error("중복 포트 생성이 성공함")
	}
}

func TestGet_NotFound(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	_, err := svc.Get("nonexistent")
	if err == nil {
		t.Error("존재하지 않는 포트 조회가 성공함")
	}
}

func TestUpdateStatus(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("port-001", "Test", "")

	// pending → running
	err := svc.UpdateStatus("port-001", "running")
	if err != nil {
		t.Fatalf("상태 업데이트 실패: %v", err)
	}

	port, _ := svc.Get("port-001")
	if port.Status != "running" {
		t.Errorf("Status = %s, want running", port.Status)
	}

	// running → complete
	svc.UpdateStatus("port-001", "complete")
	port, _ = svc.Get("port-001")
	if port.Status != "complete" {
		t.Errorf("Status = %s, want complete", port.Status)
	}
}

func TestDelete(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("port-001", "Test", "")

	err := svc.Delete("port-001")
	if err != nil {
		t.Fatalf("포트 삭제 실패: %v", err)
	}

	_, err = svc.Get("port-001")
	if err == nil {
		t.Error("삭제된 포트 조회가 성공함")
	}
}

func TestList(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("port-001", "First", "")
	svc.Create("port-002", "Second", "")
	svc.Create("port-003", "Third", "")
	svc.UpdateStatus("port-002", "running")

	// 전체 목록
	all, err := svc.List("", 10)
	if err != nil {
		t.Fatalf("목록 조회 실패: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("전체 수 = %d, want 3", len(all))
	}

	// running만
	running, err := svc.List("running", 10)
	if err != nil {
		t.Fatalf("running 목록 조회 실패: %v", err)
	}
	if len(running) != 1 {
		t.Errorf("running 수 = %d, want 1", len(running))
	}

	// pending만
	pending, err := svc.List("pending", 10)
	if err != nil {
		t.Fatalf("pending 목록 조회 실패: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("pending 수 = %d, want 2", len(pending))
	}
}

func TestSummary(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("port-001", "First", "")
	svc.Create("port-002", "Second", "")
	svc.Create("port-003", "Third", "")
	svc.UpdateStatus("port-001", "running")
	svc.UpdateStatus("port-002", "complete")

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("요약 조회 실패: %v", err)
	}

	if summary["pending"] != 1 {
		t.Errorf("pending = %d, want 1", summary["pending"])
	}
	if summary["running"] != 1 {
		t.Errorf("running = %d, want 1", summary["running"])
	}
	if summary["complete"] != 1 {
		t.Errorf("complete = %d, want 1", summary["complete"])
	}
}

func TestList_WithLimit(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	for i := 0; i < 10; i++ {
		svc.Create(string(rune('a'+i)), "Test", "")
	}

	limited, err := svc.List("", 5)
	if err != nil {
		t.Fatalf("제한 목록 조회 실패: %v", err)
	}
	if len(limited) != 5 {
		t.Errorf("제한 수 = %d, want 5", len(limited))
	}
}
