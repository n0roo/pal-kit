package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-rules-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	// .claude/rules 디렉토리 생성
	rulesDir := filepath.Join(tmpDir, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("rules 디렉토리 생성 실패: %v", err)
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

func TestActivatePort(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	err := svc.ActivatePort("port-001", "Order Entity", "", nil)
	if err != nil {
		t.Fatalf("포트 활성화 실패: %v", err)
	}

	// 규칙 파일 존재 확인
	rulePath := svc.GetRulePath("port-001")
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		t.Error("규칙 파일이 생성되지 않음")
	}
}

func TestActivatePortWithSpec(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	// 명세 파일 생성
	portsDir := filepath.Join(projectRoot, "ports")
	os.MkdirAll(portsDir, 0755)
	specPath := filepath.Join(portsDir, "port-001.md")
	os.WriteFile(specPath, []byte("# Port Spec\n\nTest content"), 0644)

	svc := NewService(projectRoot)

	paths := []string{"internal/entity/*", "internal/service/*"}
	err := svc.ActivatePortWithSpec("port-001", "Order Entity", specPath, paths)
	if err != nil {
		t.Fatalf("포트 활성화 실패: %v", err)
	}

	// 규칙 파일 내용 확인
	rulePath := svc.GetRulePath("port-001")
	content, _ := os.ReadFile(rulePath)
	
	if len(content) == 0 {
		t.Error("규칙 파일이 비어있음")
	}

	// 경로 패턴 포함 확인
	contentStr := string(content)
	if !strings.Contains(contentStr, "internal/entity/*") {
		t.Error("경로 패턴이 포함되지 않음")
	}
}

func TestDeactivatePort(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 활성화
	svc.ActivatePort("port-001", "Order Entity", "", nil)
	rulePath := svc.GetRulePath("port-001")

	// 파일 존재 확인
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		t.Fatal("규칙 파일이 생성되지 않음")
	}

	// 비활성화
	err := svc.DeactivatePort("port-001")
	if err != nil {
		t.Fatalf("포트 비활성화 실패: %v", err)
	}

	// 파일 삭제 확인
	if _, err := os.Stat(rulePath); !os.IsNotExist(err) {
		t.Error("규칙 파일이 삭제되지 않음")
	}
}

func TestListActiveRules(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 여러 포트 활성화
	svc.ActivatePort("port-001", "Entity", "", nil)
	svc.ActivatePort("port-002", "Service", "", nil)
	svc.ActivatePort("port-003", "API", "", nil)

	rules, err := svc.ListActiveRules()
	if err != nil {
		t.Fatalf("활성 규칙 조회 실패: %v", err)
	}

	if len(rules) != 3 {
		t.Errorf("활성 규칙 수 = %d, want 3", len(rules))
	}
}

func TestGetRulePath(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	path := svc.GetRulePath("port-001")
	expected := filepath.Join(projectRoot, ".claude", "rules", "port-001.md")

	if path != expected {
		t.Errorf("규칙 경로 = %s, want %s", path, expected)
	}
}

func TestIsActive_ByFileCheck(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	rulePath := svc.GetRulePath("port-001")

	// 비활성 상태 (파일 없음)
	if _, err := os.Stat(rulePath); !os.IsNotExist(err) {
		t.Error("비활성 포트에 파일이 존재함")
	}

	// 활성화 후
	svc.ActivatePort("port-001", "Entity", "", nil)
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		t.Error("활성화 후 파일이 없음")
	}

	// 비활성화 후
	svc.DeactivatePort("port-001")
	if _, err := os.Stat(rulePath); !os.IsNotExist(err) {
		t.Error("비활성화 후 파일이 남아있음")
	}
}

func TestMultiplePorts(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	ports := []string{"port-001", "port-002", "port-003", "port-004"}
	
	for _, p := range ports {
		svc.ActivatePort(p, "Test Port", "", nil)
	}

	rules, _ := svc.ListActiveRules()
	if len(rules) != 4 {
		t.Errorf("활성 규칙 수 = %d, want 4", len(rules))
	}

	// 일부 비활성화
	svc.DeactivatePort("port-002")
	svc.DeactivatePort("port-004")

	rules, _ = svc.ListActiveRules()
	if len(rules) != 2 {
		t.Errorf("남은 규칙 수 = %d, want 2", len(rules))
	}
}

func TestDeactivatePort_NotExist(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 존재하지 않는 포트 비활성화 - 에러 없이 성공해야 함
	err := svc.DeactivatePort("nonexistent")
	if err != nil {
		t.Errorf("존재하지 않는 포트 비활성화 실패: %v", err)
	}
}

func TestActivatePort_WithFilePatterns(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	patterns := []string{"src/**/*.go", "internal/**/*.go"}
	err := svc.ActivatePort("port-001", "Test", "", patterns)
	if err != nil {
		t.Fatalf("패턴과 함께 활성화 실패: %v", err)
	}

	// 규칙 파일 내용에 패턴이 포함되어 있는지 확인
	rulePath := svc.GetRulePath("port-001")
	content, _ := os.ReadFile(rulePath)
	contentStr := string(content)

	if !strings.Contains(contentStr, "src/**/*.go") {
		t.Error("파일 패턴이 규칙에 포함되지 않음")
	}
}
