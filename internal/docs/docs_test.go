package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-docs-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestNewService(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	if svc == nil {
		t.Fatal("Service가 nil")
	}
}

func TestEnsureDirectories(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	if err := svc.EnsureDirectories(); err != nil {
		t.Fatalf("디렉토리 생성 실패: %v", err)
	}

	dirs := []string{"agents", "ports", "conventions", "templates", ".claude/rules", ".pal/snapshots"}
	for _, dir := range dirs {
		path := filepath.Join(projectRoot, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("디렉토리 생성 안됨: %s", dir)
		}
	}
}

func TestList_Empty(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	docs, err := svc.List()
	if err != nil {
		t.Fatalf("목록 조회 실패: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("빈 프로젝트인데 문서가 있음: %d", len(docs))
	}
}

func TestList_WithDocuments(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	svc.EnsureDirectories()

	// 문서 생성
	os.WriteFile(filepath.Join(projectRoot, "CLAUDE.md"), []byte("# Project"), 0644)
	os.WriteFile(filepath.Join(projectRoot, "agents", "worker.yaml"), []byte("agent:\n  id: worker"), 0644)
	os.WriteFile(filepath.Join(projectRoot, "ports", "port-001.md"), []byte("# Port"), 0644)

	docs, err := svc.List()
	if err != nil {
		t.Fatalf("목록 조회 실패: %v", err)
	}

	if len(docs) != 3 {
		t.Errorf("문서 수 = %d, want 3", len(docs))
	}
}

func TestGet(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// CLAUDE.md 생성
	claudeMD := filepath.Join(projectRoot, "CLAUDE.md")
	os.WriteFile(claudeMD, []byte("# Project"), 0644)

	doc, err := svc.Get("CLAUDE.md")
	if err != nil {
		t.Fatalf("문서 조회 실패: %v", err)
	}

	if doc.Type != DocTypeClaude {
		t.Errorf("Type = %s, want claude", doc.Type)
	}
}

func TestGetContent(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 파일 생성
	content := "# Test Content"
	os.WriteFile(filepath.Join(projectRoot, "test.md"), []byte(content), 0644)

	result, err := svc.GetContent("test.md")
	if err != nil {
		t.Fatalf("내용 조회 실패: %v", err)
	}

	if result != content {
		t.Errorf("내용 = %s, want %s", result, content)
	}
}

func TestInitProject(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	created, err := svc.InitProject("Test Project")
	if err != nil {
		t.Fatalf("초기화 실패: %v", err)
	}

	// CLAUDE.md 생성 확인
	claudeMD := filepath.Join(projectRoot, "CLAUDE.md")
	if _, err := os.Stat(claudeMD); os.IsNotExist(err) {
		t.Error("CLAUDE.md가 생성되지 않음")
	}

	if len(created) == 0 {
		t.Error("생성된 파일이 없음")
	}
}

func TestCreateSnapshot(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	svc.EnsureDirectories()

	// 문서 생성
	os.WriteFile(filepath.Join(projectRoot, "CLAUDE.md"), []byte("# Project"), 0644)

	snapshot, err := svc.CreateSnapshot("Test snapshot")
	if err != nil {
		t.Fatalf("스냅샷 생성 실패: %v", err)
	}

	if snapshot.ID == "" {
		t.Error("스냅샷 ID가 비어있음")
	}
	if len(snapshot.Documents) != 1 {
		t.Errorf("문서 수 = %d, want 1", len(snapshot.Documents))
	}
}

func TestListSnapshots(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	svc.EnsureDirectories()

	// 문서 생성
	os.WriteFile(filepath.Join(projectRoot, "CLAUDE.md"), []byte("# Project"), 0644)

	// 스냅샷 생성
	svc.CreateSnapshot("Snapshot 1")

	snapshots, err := svc.ListSnapshots()
	if err != nil {
		t.Fatalf("스냅샷 목록 조회 실패: %v", err)
	}

	if len(snapshots) != 1 {
		t.Errorf("스냅샷 수 = %d, want 1", len(snapshots))
	}
}

func TestDiffWithLatest(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	svc.EnsureDirectories()

	// 문서 생성 및 스냅샷
	os.WriteFile(filepath.Join(projectRoot, "CLAUDE.md"), []byte("# Project"), 0644)
	svc.CreateSnapshot("Initial")

	// 문서 수정
	os.WriteFile(filepath.Join(projectRoot, "CLAUDE.md"), []byte("# Modified Project"), 0644)
	// 새 문서 추가
	os.WriteFile(filepath.Join(projectRoot, "agents", "new.yaml"), []byte("agent:"), 0644)

	diff, err := svc.DiffWithLatest()
	if err != nil {
		t.Fatalf("diff 실패: %v", err)
	}

	if len(diff.Modified) != 1 {
		t.Errorf("수정된 문서 = %d, want 1", len(diff.Modified))
	}
	if len(diff.Added) != 1 {
		t.Errorf("추가된 문서 = %d, want 1", len(diff.Added))
	}
}

func TestLint(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	svc.EnsureDirectories()

	// 유효한 CLAUDE.md
	claudeContent := `# Project

## 프로젝트 개요
Test project

## 디렉토리 구조
None

## 개발 규칙
None
`
	os.WriteFile(filepath.Join(projectRoot, "CLAUDE.md"), []byte(claudeContent), 0644)

	results, err := svc.Lint(nil)
	if err != nil {
		t.Fatalf("lint 실패: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("결과 수 = %d, want 1", len(results))
	}
}

func TestLint_InvalidAgent(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	svc.EnsureDirectories()

	// 잘못된 에이전트 파일
	os.WriteFile(filepath.Join(projectRoot, "agents", "bad.yaml"), []byte("invalid: yaml: content"), 0644)

	results, err := svc.Lint(nil)
	if err != nil {
		t.Fatalf("lint 실패: %v", err)
	}

	// 에러가 있어야 함
	hasError := false
	for _, r := range results {
		for _, issue := range r.Issues {
			if issue.Severity == SeverityError {
				hasError = true
				break
			}
		}
	}

	if !hasError {
		t.Error("잘못된 YAML에서 에러가 발견되지 않음")
	}
}

func TestListTemplates(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	templates := svc.ListTemplates()

	if len(templates) == 0 {
		t.Error("템플릿이 없음")
	}

	// claude-md 템플릿 확인
	found := false
	for _, t := range templates {
		if t.Name == "claude-md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("claude-md 템플릿이 없음")
	}
}

func TestApplyTemplate(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	data := TemplateData{
		ProjectName: "Test Project",
	}

	fileName, content, err := svc.ApplyTemplate("claude-md", data)
	if err != nil {
		t.Fatalf("템플릿 적용 실패: %v", err)
	}

	if fileName != "CLAUDE.md" {
		t.Errorf("파일명 = %s, want CLAUDE.md", fileName)
	}
	if content == "" {
		t.Error("내용이 비어있음")
	}
}

func TestCreateFromTemplate(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	data := TemplateData{
		ProjectName: "Test Project",
	}

	fileName, err := svc.CreateFromTemplate("claude-md", data, false)
	if err != nil {
		t.Fatalf("파일 생성 실패: %v", err)
	}

	// 파일 존재 확인
	filePath := filepath.Join(projectRoot, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("파일이 생성되지 않음")
	}

	// 중복 생성 시도 - 에러 발생해야 함
	_, err = svc.CreateFromTemplate("claude-md", data, false)
	if err == nil {
		t.Error("중복 파일 생성이 성공함")
	}
}
