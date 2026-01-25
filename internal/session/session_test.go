package session

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

func TestStart(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	err := svc.Start("test-session", "", "")
	if err != nil {
		t.Fatalf("세션 시작 실패: %v", err)
	}

	// 세션 조회
	sess, err := svc.Get("test-session")
	if err != nil {
		t.Fatalf("세션 조회 실패: %v", err)
	}

	if sess.ID != "test-session" {
		t.Errorf("ID = %s, want test-session", sess.ID)
	}
	if sess.Status != "running" {
		t.Errorf("Status = %s, want running", sess.Status)
	}
}

func TestStartWithOptions(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// builder 세션 생성
	err := svc.StartWithOptions("builder-1", "", "Builder Session", TypeBuilder, "")
	if err != nil {
		t.Fatalf("빌더 세션 생성 실패: %v", err)
	}

	// sub 세션 생성
	err = svc.StartWithOptions("sub-1", "", "Sub Session", TypeSub, "builder-1")
	if err != nil {
		t.Fatalf("서브 세션 생성 실패: %v", err)
	}

	// 세션 조회
	builder, _ := svc.Get("builder-1")
	if builder.SessionType != TypeBuilder {
		t.Errorf("SessionType = %s, want %s", builder.SessionType, TypeBuilder)
	}

	sub, _ := svc.Get("sub-1")
	if sub.SessionType != TypeSub {
		t.Errorf("SessionType = %s, want %s", sub.SessionType, TypeSub)
	}
	if !sub.ParentSession.Valid || sub.ParentSession.String != "builder-1" {
		t.Errorf("ParentSession = %v, want builder-1", sub.ParentSession)
	}
}

func TestStartWithOptions_InvalidParent(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// 존재하지 않는 parent로 sub 세션 생성 시도
	err := svc.StartWithOptions("sub-1", "", "Sub Session", TypeSub, "nonexistent")
	if err == nil {
		t.Error("존재하지 않는 parent로 생성이 성공함")
	}
}

func TestEnd(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Start("test-session", "", "")

	err := svc.End("test-session")
	if err != nil {
		t.Fatalf("세션 종료 실패: %v", err)
	}

	sess, _ := svc.Get("test-session")
	if sess.Status != "complete" {
		t.Errorf("Status = %s, want complete", sess.Status)
	}
	if !sess.EndedAt.Valid {
		t.Error("EndedAt이 설정되지 않음")
	}
}

func TestUpdateStatus(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Start("test-session", "", "")

	err := svc.UpdateStatus("test-session", "failed")
	if err != nil {
		t.Fatalf("상태 업데이트 실패: %v", err)
	}

	sess, _ := svc.Get("test-session")
	if sess.Status != "failed" {
		t.Errorf("Status = %s, want failed", sess.Status)
	}
}

func TestList(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Start("session-1", "", "First")
	svc.Start("session-2", "", "Second")
	svc.End("session-1")

	// 전체 목록
	all, err := svc.List(false, 10)
	if err != nil {
		t.Fatalf("목록 조회 실패: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("전체 수 = %d, want 2", len(all))
	}

	// 활성 세션만
	active, err := svc.List(true, 10)
	if err != nil {
		t.Fatalf("활성 목록 조회 실패: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("활성 수 = %d, want 1", len(active))
	}
}

func TestGetChildren(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// 계층 구조 생성
	svc.StartWithOptions("parent", "", "Parent", TypeBuilder, "")
	svc.StartWithOptions("child-1", "", "Child 1", TypeSub, "parent")
	svc.StartWithOptions("child-2", "", "Child 2", TypeSub, "parent")

	children, err := svc.GetChildren("parent")
	if err != nil {
		t.Fatalf("하위 세션 조회 실패: %v", err)
	}

	if len(children) != 2 {
		t.Errorf("하위 세션 수 = %d, want 2", len(children))
	}
}

func TestGetTree(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// 3단계 계층 구조 생성
	svc.StartWithOptions("root", "", "Root", TypeBuilder, "")
	svc.StartWithOptions("child-1", "", "Child 1", TypeSub, "root")
	svc.StartWithOptions("child-2", "", "Child 2", TypeSub, "root")
	svc.StartWithOptions("grandchild", "", "Grandchild", TypeSub, "child-1")

	tree, err := svc.GetTree("root")
	if err != nil {
		t.Fatalf("트리 조회 실패: %v", err)
	}

	if tree.Session.ID != "root" {
		t.Errorf("루트 ID = %s, want root", tree.Session.ID)
	}
	if len(tree.Children) != 2 {
		t.Errorf("자식 수 = %d, want 2", len(tree.Children))
	}

	// child-1의 자식 확인
	var child1 *SessionNode
	for i := range tree.Children {
		if tree.Children[i].Session.ID == "child-1" {
			child1 = &tree.Children[i]
			break
		}
	}
	if child1 == nil {
		t.Fatal("child-1을 찾을 수 없음")
		return // unreachable but silences staticcheck
	}
	if len(child1.Children) != 1 {
		t.Errorf("child-1의 자식 수 = %d, want 1", len(child1.Children))
	}
}

func TestGetRootSessions(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// 루트 세션과 자식 세션 생성
	svc.StartWithOptions("root-1", "", "Root 1", TypeBuilder, "")
	svc.StartWithOptions("root-2", "", "Root 2", TypeSingle, "")
	svc.StartWithOptions("child", "", "Child", TypeSub, "root-1")

	roots, err := svc.GetRootSessions(10)
	if err != nil {
		t.Fatalf("루트 세션 조회 실패: %v", err)
	}

	if len(roots) != 2 {
		t.Errorf("루트 세션 수 = %d, want 2", len(roots))
	}
}

func TestIncrementCompact(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Start("test-session", "", "")

	// 컴팩션 증가
	svc.IncrementCompact("test-session")
	svc.IncrementCompact("test-session")

	sess, _ := svc.Get("test-session")
	if sess.CompactCount != 2 {
		t.Errorf("CompactCount = %d, want 2", sess.CompactCount)
	}
}
