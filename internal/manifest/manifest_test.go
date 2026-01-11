package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n0roo/pal-kit/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	tmpFile, err := os.CreateTemp("", "pal-manifest-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp db file: %v", err)
	}
	tmpFile.Close()

	database, err := db.Open(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("failed to open db: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.Remove(tmpFile.Name())
	}

	return database, cleanup
}

func setupTestProject(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "pal-manifest-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// 테스트 파일 생성
	files := map[string]string{
		"CLAUDE.md":             "# Test Project",
		".claude/settings.json": `{"hooks": {}}`,
		"go.mod":                "module test",
		"main.go":               "package main",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write file: %v", err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestNewService(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	svc := NewService(database, "/tmp/test")
	if svc == nil {
		t.Fatal("expected service, got nil")
	}
}

func TestInit(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Manifest 파일 생성 확인
	manifestPath := filepath.Join(projectRoot, ".pal", "manifest.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("manifest.yaml not created")
	}
}

func TestComputeHash(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// CLAUDE.md 해시 계산
	hash, err := svc.ComputeHash("CLAUDE.md")
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// sha256: 접두사 확인
	if len(hash) < 7 || hash[:7] != "sha256:" {
		t.Error("hash should start with sha256:")
	}

	// 동일 파일은 동일 해시
	hash2, err := svc.ComputeHash("CLAUDE.md")
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	if hash != hash2 {
		t.Error("same file should have same hash")
	}

	// 다른 파일은 다른 해시
	hash3, err := svc.ComputeHash("go.mod")
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	if hash == hash3 {
		t.Error("different files should have different hashes")
	}
}

func TestAddFile(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// 파일 추가
	if err := svc.AddFile("CLAUDE.md", ManagedByPal); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	// manifest 로드 확인
	manifest, err := svc.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if _, exists := manifest.Files["CLAUDE.md"]; !exists {
		t.Error("added file not found in manifest")
	}

	if manifest.Files["CLAUDE.md"].ManagedBy != ManagedByPal {
		t.Errorf("expected ManagedBy pal, got %s", manifest.Files["CLAUDE.md"].ManagedBy)
	}
}

func TestRemoveFile(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// 파일 추가
	if err := svc.AddFile("CLAUDE.md", ManagedByPal); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	// 파일 제거
	if err := svc.RemoveFile("CLAUDE.md"); err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}

	// manifest 확인
	manifest, err := svc.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if _, exists := manifest.Files["CLAUDE.md"]; exists {
		t.Error("removed file still in manifest")
	}
}

func TestStatus(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// 파일 추가
	if err := svc.AddFile("CLAUDE.md", ManagedByPal); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	// 초기 상태에서는 synced
	statuses, err := svc.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Path == "CLAUDE.md" {
			found = true
			if s.Status != StatusSynced {
				t.Errorf("expected status synced, got %s", s.Status)
			}
			break
		}
	}

	if !found {
		t.Error("CLAUDE.md not found in status")
	}

	// 파일 수정
	claudePath := filepath.Join(projectRoot, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Modified"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// 변경 감지
	statuses, err = svc.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	found = false
	for _, s := range statuses {
		if s.Path == "CLAUDE.md" {
			found = true
			if s.Status != StatusModified {
				t.Errorf("expected status modified, got %s", s.Status)
			}
			break
		}
	}

	if !found {
		t.Error("CLAUDE.md not found in status after modification")
	}
}

func TestQuickCheck(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// 파일 추가
	if err := svc.AddFile("CLAUDE.md", ManagedByPal); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	// 초기 상태
	changed, err := svc.QuickCheck()
	if err != nil {
		t.Fatalf("QuickCheck failed: %v", err)
	}

	// CLAUDE.md가 변경되지 않았으므로 빈 결과 (또는 다른 새 파일들)
	hasClaudeChanged := false
	for _, path := range changed {
		if path == "CLAUDE.md" {
			hasClaudeChanged = true
			break
		}
	}

	if hasClaudeChanged {
		t.Error("CLAUDE.md should not be in changed list initially")
	}

	// 파일 수정
	claudePath := filepath.Join(projectRoot, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Modified Content"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// 빠른 체크
	changed, err = svc.QuickCheck()
	if err != nil {
		t.Fatalf("QuickCheck failed: %v", err)
	}

	hasClaudeChanged = false
	for _, path := range changed {
		if path == "CLAUDE.md" {
			hasClaudeChanged = true
			break
		}
	}

	if !hasClaudeChanged {
		t.Error("CLAUDE.md should be in changed list after modification")
	}
}

func TestSync(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// 파일 추가
	if err := svc.AddFile("CLAUDE.md", ManagedByPal); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	// 파일 수정
	claudePath := filepath.Join(projectRoot, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Synced Content"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// 동기화
	changes, err := svc.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// 변경 기록 확인
	hasModified := false
	for _, c := range changes {
		if c.FilePath == "CLAUDE.md" && c.ChangeType == "modified" {
			hasModified = true
			break
		}
	}

	if !hasModified {
		t.Error("expected modified change record for CLAUDE.md")
	}

	// 동기화 후 상태 확인
	statuses, err := svc.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	for _, s := range statuses {
		if s.Path == "CLAUDE.md" {
			if s.Status != StatusSynced {
				t.Errorf("expected status synced after sync, got %s", s.Status)
			}
			break
		}
	}
}

func TestGetChanges(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// 파일 추가
	if err := svc.AddFile("CLAUDE.md", ManagedByPal); err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	// 파일 수정 후 동기화
	claudePath := filepath.Join(projectRoot, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Version 2"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	if _, err := svc.Sync(); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// 히스토리 조회
	changes, err := svc.GetChanges(10)
	if err != nil {
		t.Fatalf("GetChanges failed: %v", err)
	}

	if len(changes) < 1 {
		t.Error("expected at least 1 change record")
	}
}

func TestLoadManifest_Empty(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// manifest 파일이 없을 때 새로 생성
	manifest, err := svc.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if manifest.Version != "1" {
		t.Errorf("expected version 1, got %s", manifest.Version)
	}

	if manifest.Files == nil {
		t.Error("expected Files map to be initialized")
	}
}

func TestSaveManifest(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	manifest := &Manifest{
		Version: "1",
		Files:   make(map[string]*TrackedFile),
	}

	manifest.Files["test.md"] = &TrackedFile{
		Path:      "test.md",
		Type:      FileTypeContext,
		Hash:      "sha256:abc123",
		Size:      100,
		ManagedBy: ManagedByPal,
	}

	if err := svc.SaveManifest(manifest); err != nil {
		t.Fatalf("SaveManifest failed: %v", err)
	}

	// 다시 로드
	loaded, err := svc.LoadManifest()
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if _, exists := loaded.Files["test.md"]; !exists {
		t.Error("saved file not found after reload")
	}
}

func TestScanTrackedFiles(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	// agents 디렉토리 생성
	agentsDir := filepath.Join(projectRoot, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "builder.yaml"), []byte("agent: test"), 0644); err != nil {
		t.Fatalf("failed to write agent file: %v", err)
	}

	svc := NewService(database, projectRoot)

	files, err := svc.ScanTrackedFiles()
	if err != nil {
		t.Fatalf("ScanTrackedFiles failed: %v", err)
	}

	// CLAUDE.md 포함 확인
	hasClaude := false
	hasAgent := false
	for _, f := range files {
		if f == "CLAUDE.md" {
			hasClaude = true
		}
		if f == "agents/builder.yaml" {
			hasAgent = true
		}
	}

	if !hasClaude {
		t.Error("CLAUDE.md not found in scanned files")
	}

	if !hasAgent {
		t.Error("agents/builder.yaml not found in scanned files")
	}
}

func TestGetFileInfo(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	info, err := svc.GetFileInfo("CLAUDE.md")
	if err != nil {
		t.Fatalf("GetFileInfo failed: %v", err)
	}

	if info.Path != "CLAUDE.md" {
		t.Errorf("expected path CLAUDE.md, got %s", info.Path)
	}

	if info.Type != FileTypeContext {
		t.Errorf("expected type context, got %s", info.Type)
	}

	if info.Hash == "" {
		t.Error("expected non-empty hash")
	}

	if info.Size == 0 {
		t.Error("expected non-zero size")
	}
}

func TestDetectFileType(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	svc := NewService(database, "/tmp/test")

	tests := []struct {
		path     string
		expected FileType
	}{
		{"CLAUDE.md", FileTypeContext},
		{"agents/builder.yaml", FileTypeAgent},
		{"conventions/code.yaml", FileTypeConvention},
		{"ports/feature.md", FileTypePort},
		{".pal/config.yaml", FileTypeConfig},
	}

	for _, tt := range tests {
		result := svc.detectFileType(tt.path)
		if result != tt.expected {
			t.Errorf("detectFileType(%s): expected %s, got %s", tt.path, tt.expected, result)
		}
	}
}

func TestNewFileDetection(t *testing.T) {
	database, cleanupDB := setupTestDB(t)
	defer cleanupDB()

	projectRoot, cleanupProject := setupTestProject(t)
	defer cleanupProject()

	svc := NewService(database, projectRoot)

	// 초기화 (빈 manifest)
	if _, err := svc.LoadManifest(); err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	// CLAUDE.md가 새 파일로 감지되어야 함
	statuses, err := svc.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	foundNew := false
	for _, s := range statuses {
		if s.Path == "CLAUDE.md" && s.Status == StatusNew {
			foundNew = true
			break
		}
	}

	if !foundNew {
		t.Error("CLAUDE.md should be detected as new file")
	}
}
