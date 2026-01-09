package context

import (
	"os"
	"path/filepath"
	"strings"
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

func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-ctx-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	// .claude 디렉토리 생성
	claudeDir := filepath.Join(tmpDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestFindProjectRoot(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	// 하위 디렉토리 생성
	subDir := filepath.Join(projectRoot, "src", "internal")
	os.MkdirAll(subDir, 0755)

	// 하위 디렉토리에서 프로젝트 루트 찾기
	found := FindProjectRoot(subDir)
	if found != projectRoot {
		t.Errorf("프로젝트 루트 = %s, want %s", found, projectRoot)
	}
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	// 임시 디렉토리 (PAL 프로젝트 아님)
	tmpDir, err := os.MkdirTemp("", "not-pal-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	found := FindProjectRoot(tmpDir)
	if found != "" {
		t.Errorf("프로젝트 루트가 발견됨: %s", found)
	}
}

func TestFindClaudeMD(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	// CLAUDE.md 생성
	claudeMD := filepath.Join(projectRoot, "CLAUDE.md")
	os.WriteFile(claudeMD, []byte("# Project"), 0644)

	// 하위 디렉토리에서 찾기
	subDir := filepath.Join(projectRoot, "src")
	os.MkdirAll(subDir, 0755)

	found := FindClaudeMD(subDir)
	if found != claudeMD {
		t.Errorf("CLAUDE.md = %s, want %s", found, claudeMD)
	}
}

func TestFindClaudeMD_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "no-claude-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	found := FindClaudeMD(tmpDir)
	if found != "" {
		t.Errorf("CLAUDE.md가 발견됨: %s", found)
	}
}

func TestGenerateContext(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// 세션 생성
	database.Exec(`INSERT INTO sessions (id, status) VALUES (?, ?)`, "session-1", "running")

	// 포트 생성
	database.Exec(`INSERT INTO ports (id, title, status) VALUES (?, ?, ?)`, "port-001", "Entity", "running")

	ctx, err := svc.GenerateContext()
	if err != nil {
		t.Fatalf("컨텍스트 생성 실패: %v", err)
	}

	if ctx == "" {
		t.Error("컨텍스트가 비어있음")
	}

	if !strings.Contains(ctx, "session-1") {
		t.Error("세션 정보가 포함되지 않음")
	}
}

func TestInjectToFile(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	database, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	// CLAUDE.md 생성
	claudeMD := filepath.Join(projectRoot, "CLAUDE.md")
	initialContent := `# Project

Some content here.

<!-- pal:context:start -->
Old context
<!-- pal:context:end -->

More content.
`
	os.WriteFile(claudeMD, []byte(initialContent), 0644)

	// 세션 추가
	database.Exec(`INSERT INTO sessions (id, status) VALUES (?, ?)`, "session-1", "running")

	svc := NewService(database)
	err := svc.InjectToFile(claudeMD)
	if err != nil {
		t.Fatalf("주입 실패: %v", err)
	}

	// 파일 내용 확인
	content, _ := os.ReadFile(claudeMD)
	contentStr := string(content)

	if !strings.Contains(contentStr, "pal:context:start") {
		t.Error("시작 마커가 없음")
	}
	if !strings.Contains(contentStr, "pal:context:end") {
		t.Error("종료 마커가 없음")
	}
	if strings.Contains(contentStr, "Old context") {
		t.Error("이전 컨텍스트가 남아있음")
	}
}

func TestInjectToFile_NoMarker(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	database, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	// 마커 없는 CLAUDE.md 생성
	claudeMD := filepath.Join(projectRoot, "CLAUDE.md")
	initialContent := `# Project

Some content here.
`
	os.WriteFile(claudeMD, []byte(initialContent), 0644)

	svc := NewService(database)
	err := svc.InjectToFile(claudeMD)
	if err != nil {
		t.Fatalf("주입 실패: %v", err)
	}

	// 파일 내용 확인 - 마커가 추가되어야 함
	content, _ := os.ReadFile(claudeMD)
	contentStr := string(content)

	if !strings.Contains(contentStr, "pal:context:start") {
		t.Error("시작 마커가 추가되지 않음")
	}
}

func TestNewService(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	if svc == nil {
		t.Fatal("Service가 nil")
	}
}

func TestGenerateContext_Empty(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// 데이터 없이 컨텍스트 생성
	ctx, err := svc.GenerateContext()
	if err != nil {
		t.Fatalf("빈 컨텍스트 생성 실패: %v", err)
	}

	// 빈 상태에서도 기본 구조는 있어야 함
	if ctx == "" {
		t.Error("컨텍스트가 완전히 비어있음")
	}
}

func TestFindProjectRoot_DeepNested(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	// 깊은 하위 디렉토리 생성
	deepDir := filepath.Join(projectRoot, "src", "internal", "domain", "entity")
	os.MkdirAll(deepDir, 0755)

	// 깊은 디렉토리에서도 프로젝트 루트 찾기
	found := FindProjectRoot(deepDir)
	if found != projectRoot {
		t.Errorf("프로젝트 루트 = %s, want %s", found, projectRoot)
	}
}
