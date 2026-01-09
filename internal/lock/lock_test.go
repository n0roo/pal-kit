package lock

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

func TestAcquire(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	err := svc.Acquire("entity", "session-123")
	if err != nil {
		t.Fatalf("Lock 획득 실패: %v", err)
	}

	// 동일 리소스 재획득 시도
	err = svc.Acquire("entity", "session-456")
	if err == nil {
		t.Error("이미 잠긴 리소스 획득이 성공함")
	}
}

func TestRelease(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	svc.Acquire("entity", "session-123")
	
	err := svc.Release("entity")
	if err != nil {
		t.Fatalf("Lock 해제 실패: %v", err)
	}

	// 해제 후 다른 세션이 획득 가능
	err = svc.Acquire("entity", "session-456")
	if err != nil {
		t.Error("Lock 해제 후 획득 실패")
	}
}

func TestRelease_NotFound(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	err := svc.Release("nonexistent")
	if err == nil {
		t.Error("존재하지 않는 Lock 해제가 성공함")
	}
}

func TestList(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	svc.Acquire("entity", "session-1")
	svc.Acquire("service", "session-2")
	svc.Acquire("api", "session-1")

	locks, err := svc.List()
	if err != nil {
		t.Fatalf("목록 조회 실패: %v", err)
	}

	if len(locks) != 3 {
		t.Errorf("Lock 수 = %d, want 3", len(locks))
	}
}

func TestReleaseBySession(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	svc.Acquire("entity", "session-1")
	svc.Acquire("service", "session-1")
	svc.Acquire("api", "session-2")

	// session-1의 모든 Lock 해제
	locks, _ := svc.List()
	for _, l := range locks {
		if l.SessionID == "session-1" {
			svc.Release(l.Resource)
		}
	}

	// session-2의 Lock만 남아있어야 함
	locks, _ = svc.List()
	if len(locks) != 1 {
		t.Errorf("남은 Lock 수 = %d, want 1", len(locks))
	}
	if locks[0].Resource != "api" {
		t.Errorf("남은 리소스 = %s, want api", locks[0].Resource)
	}
}

func TestMultipleResources(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	resources := []string{"entity", "service", "api", "domain"}
	
	for _, r := range resources {
		if err := svc.Acquire(r, "session-1"); err != nil {
			t.Errorf("리소스 %s 획득 실패: %v", r, err)
		}
	}

	locks, _ := svc.List()
	if len(locks) != 4 {
		t.Errorf("Lock 수 = %d, want 4", len(locks))
	}
}

func TestLockStruct(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Acquire("entity", "session-123")

	locks, _ := svc.List()
	if len(locks) != 1 {
		t.Fatal("Lock이 없음")
	}

	lock := locks[0]
	if lock.Resource != "entity" {
		t.Errorf("Resource = %s, want entity", lock.Resource)
	}
	if lock.SessionID != "session-123" {
		t.Errorf("SessionID = %s, want session-123", lock.SessionID)
	}
	if lock.AcquiredAt.IsZero() {
		t.Error("AcquiredAt이 설정되지 않음")
	}
}
