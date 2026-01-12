package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n0roo/pal-kit/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-env-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("DB 생성 실패: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}

func TestSetupAndList(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := &Service{
		db:         database,
		configPath: filepath.Join(os.TempDir(), "test-environments.yaml"),
	}
	defer os.Remove(svc.configPath)

	// Setup first environment
	paths := PathVariables{
		Workspace:  "/Users/test/workspace",
		ClaudeData: "/Users/test/.claude",
		Home:       "/Users/test",
	}

	env1, err := svc.Setup("test-env", paths)
	if err != nil {
		t.Fatalf("환경 설정 실패: %v", err)
	}

	if env1.Name != "test-env" {
		t.Errorf("이름 불일치: got %s, want test-env", env1.Name)
	}

	if !env1.IsCurrent {
		t.Error("첫 번째 환경은 현재 환경이어야 함")
	}

	// List environments
	envs, err := svc.List()
	if err != nil {
		t.Fatalf("환경 목록 조회 실패: %v", err)
	}

	if len(envs) != 1 {
		t.Errorf("환경 개수 불일치: got %d, want 1", len(envs))
	}
}

func TestSwitchEnvironment(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := &Service{
		db:         database,
		configPath: filepath.Join(os.TempDir(), "test-environments-switch.yaml"),
	}
	defer os.Remove(svc.configPath)

	// Setup two environments
	paths1 := PathVariables{
		Workspace:  "/home/user1/workspace",
		ClaudeData: "/home/user1/.claude",
		Home:       "/home/user1",
	}
	paths2 := PathVariables{
		Workspace:  "/home/user2/workspace",
		ClaudeData: "/home/user2/.claude",
		Home:       "/home/user2",
	}

	_, err := svc.Setup("env1", paths1)
	if err != nil {
		t.Fatalf("env1 설정 실패: %v", err)
	}

	_, err = svc.Setup("env2", paths2)
	if err != nil {
		t.Fatalf("env2 설정 실패: %v", err)
	}

	// Verify env1 is current (first registered)
	current, err := svc.Current()
	if err != nil {
		t.Fatalf("현재 환경 조회 실패: %v", err)
	}
	if current.Name != "env1" {
		t.Errorf("현재 환경 불일치: got %s, want env1", current.Name)
	}

	// Switch to env2
	if err := svc.Switch("env2"); err != nil {
		t.Fatalf("환경 전환 실패: %v", err)
	}

	// Verify env2 is now current
	current, err = svc.Current()
	if err != nil {
		t.Fatalf("현재 환경 조회 실패: %v", err)
	}
	if current.Name != "env2" {
		t.Errorf("현재 환경 불일치: got %s, want env2", current.Name)
	}
}

func TestDeleteEnvironment(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := &Service{
		db:         database,
		configPath: filepath.Join(os.TempDir(), "test-environments-delete.yaml"),
	}
	defer os.Remove(svc.configPath)

	// Setup two environments
	paths := PathVariables{
		Workspace:  "/test/workspace",
		ClaudeData: "/test/.claude",
		Home:       "/test",
	}

	_, err := svc.Setup("env1", paths)
	if err != nil {
		t.Fatalf("env1 설정 실패: %v", err)
	}

	_, err = svc.Setup("env2", paths)
	if err != nil {
		t.Fatalf("env2 설정 실패: %v", err)
	}

	// Cannot delete current environment
	err = svc.Delete("env1")
	if err == nil {
		t.Error("현재 환경 삭제가 허용되면 안됨")
	}

	// Can delete non-current environment
	err = svc.Delete("env2")
	if err != nil {
		t.Fatalf("env2 삭제 실패: %v", err)
	}

	// Verify only env1 remains
	envs, err := svc.List()
	if err != nil {
		t.Fatalf("환경 목록 조회 실패: %v", err)
	}
	if len(envs) != 1 {
		t.Errorf("환경 개수 불일치: got %d, want 1", len(envs))
	}
}

func TestDefaultPaths(t *testing.T) {
	paths := DefaultPaths()

	if paths.Workspace == "" {
		t.Error("Workspace 경로가 비어있음")
	}

	if paths.ClaudeData == "" {
		t.Error("ClaudeData 경로가 비어있음")
	}

	if paths.Home == "" {
		t.Error("Home 경로가 비어있음")
	}
}

func TestSuggestName(t *testing.T) {
	name := SuggestName()
	if name == "" {
		t.Error("제안된 이름이 비어있음")
	}
}
