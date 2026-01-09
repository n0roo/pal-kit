package escalation

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

	id, err := svc.Create("Something is wrong", "session-1", "port-001")
	if err != nil {
		t.Fatalf("에스컬레이션 생성 실패: %v", err)
	}

	if id == 0 {
		t.Error("ID가 0임")
	}

	// 조회
	esc, err := svc.Get(id)
	if err != nil {
		t.Fatalf("조회 실패: %v", err)
	}

	if esc.Issue != "Something is wrong" {
		t.Errorf("Issue = %s, want Something is wrong", esc.Issue)
	}
	if esc.Status != "open" {
		t.Errorf("Status = %s, want open", esc.Status)
	}
}

func TestList(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	svc.Create("Issue 1", "session-1", "port-001")
	svc.Create("Issue 2", "session-1", "port-002")
	svc.Create("Issue 3", "session-2", "port-003")

	// 전체 목록
	all, err := svc.List("", 10)
	if err != nil {
		t.Fatalf("목록 조회 실패: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("전체 수 = %d, want 3", len(all))
	}

	// 상태 필터
	open, _ := svc.List("open", 10)
	if len(open) != 3 {
		t.Errorf("open 수 = %d, want 3", len(open))
	}
}

func TestResolve(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	id, _ := svc.Create("Issue", "session-1", "port-001")

	err := svc.Resolve(id)
	if err != nil {
		t.Fatalf("해결 실패: %v", err)
	}

	esc, _ := svc.Get(id)
	if esc.Status != "resolved" {
		t.Errorf("Status = %s, want resolved", esc.Status)
	}
}

func TestDismiss(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	id, _ := svc.Create("Issue", "session-1", "port-001")

	err := svc.Dismiss(id)
	if err != nil {
		t.Fatalf("무시 실패: %v", err)
	}

	esc, _ := svc.Get(id)
	if esc.Status != "dismissed" {
		t.Errorf("Status = %s, want dismissed", esc.Status)
	}
}

func TestSummary(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	id1, _ := svc.Create("Issue 1", "session-1", "port-001")
	svc.Create("Issue 2", "session-1", "port-002")
	id3, _ := svc.Create("Issue 3", "session-2", "port-003")

	svc.Resolve(id1)
	svc.Dismiss(id3)

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("요약 조회 실패: %v", err)
	}

	if summary["open"] != 1 {
		t.Errorf("open = %d, want 1", summary["open"])
	}
	if summary["resolved"] != 1 {
		t.Errorf("resolved = %d, want 1", summary["resolved"])
	}
	if summary["dismissed"] != 1 {
		t.Errorf("dismissed = %d, want 1", summary["dismissed"])
	}
}

func TestGet_NotFound(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	_, err := svc.Get(9999)
	if err == nil {
		t.Error("존재하지 않는 에스컬레이션 조회가 성공함")
	}
}

func TestListBySession(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	svc.Create("Issue 1", "session-1", "port-001")
	svc.Create("Issue 2", "session-1", "port-002")
	svc.Create("Issue 3", "session-2", "port-003")

	// 전체 에스컬레이션에서 session-1 것만 카운트
	all, _ := svc.List("", 10)
	
	session1Count := 0
	for _, e := range all {
		if e.FromSession.Valid && e.FromSession.String == "session-1" {
			session1Count++
		}
	}

	if session1Count != 2 {
		t.Errorf("session-1 에스컬레이션 수 = %d, want 2", session1Count)
	}
}

func TestResolve_AlreadyResolved(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	id, _ := svc.Create("Issue", "session-1", "port-001")
	svc.Resolve(id)

	// 이미 해결된 에스컬레이션 재해결 시도
	err := svc.Resolve(id)
	if err == nil {
		t.Error("이미 해결된 에스컬레이션 재해결이 성공함")
	}
}

func TestDismiss_AlreadyDismissed(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	id, _ := svc.Create("Issue", "session-1", "port-001")
	svc.Dismiss(id)

	// 이미 무시된 에스컬레이션 재무시 시도
	err := svc.Dismiss(id)
	if err == nil {
		t.Error("이미 무시된 에스컬레이션 재무시가 성공함")
	}
}
