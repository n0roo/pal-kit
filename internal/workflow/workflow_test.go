package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0roo/pal-kit/internal/config"
)

func TestNewService(t *testing.T) {
	svc := NewService("/tmp/test-project")
	if svc == nil {
		t.Fatal("expected service, got nil")
		return // unreachable but silences staticcheck
	}
	if svc.projectRoot != "/tmp/test-project" {
		t.Errorf("expected project root /tmp/test-project, got %s", svc.projectRoot)
	}
}

func TestGetContext_NoConfig(t *testing.T) {
	// 임시 디렉토리 (설정 없음)
	tmpDir, err := os.MkdirTemp("", "pal-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)
	ctx, err := svc.GetContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 기본값 확인
	if ctx.WorkflowType != config.WorkflowSimple {
		t.Errorf("expected workflow type simple, got %s", ctx.WorkflowType)
	}

	if len(ctx.Agents.Core) != 1 || ctx.Agents.Core[0] != "collaborator" {
		t.Errorf("expected core agent [collaborator], got %v", ctx.Agents.Core)
	}
}

func TestGetContext_WithConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 설정 생성
	cfg := config.DefaultProjectConfig("test-project")
	cfg.Workflow.Type = config.WorkflowIntegrate
	cfg.Agents = config.DefaultAgentsForWorkflow(config.WorkflowIntegrate)

	if err := config.SaveProjectConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// 컨텍스트 조회
	svc := NewService(tmpDir)
	ctx, err := svc.GetContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ctx.WorkflowType != config.WorkflowIntegrate {
		t.Errorf("expected workflow type integrate, got %s", ctx.WorkflowType)
	}

	if ctx.ProjectName != "test-project" {
		t.Errorf("expected project name test-project, got %s", ctx.ProjectName)
	}
}

func TestGenerateRulesContent_Simple(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)
	ctx := &Context{
		WorkflowType: config.WorkflowSimple,
		ProjectName:  "test-project",
		Agents: config.AgentsConfig{
			Core: []string{"collaborator"},
		},
	}

	content := svc.GenerateRulesContent(ctx)

	// 필수 내용 확인
	if !strings.Contains(content, "PAL Kit 워크플로우 컨텍스트") {
		t.Error("missing header in rules content")
	}

	if !strings.Contains(content, "simple") {
		t.Error("missing workflow type in rules content")
	}

	if !strings.Contains(content, "Collaborator") {
		t.Error("missing Collaborator mention in simple workflow guide")
	}
}

func TestGenerateRulesContent_Integrate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)
	ctx := &Context{
		WorkflowType: config.WorkflowIntegrate,
		ProjectName:  "test-project",
		Agents: config.AgentsConfig{
			Core:    []string{"builder", "planner", "manager"},
			Workers: []string{"worker-go"},
		},
	}

	content := svc.GenerateRulesContent(ctx)

	if !strings.Contains(content, "Integrate") {
		t.Error("missing Integrate workflow guide")
	}

	if !strings.Contains(content, "빌더 세션") {
		t.Error("missing builder session mention")
	}

	if !strings.Contains(content, "워커 세션") {
		t.Error("missing worker session mention")
	}
}

func TestWriteRulesFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)
	ctx := &Context{
		WorkflowType: config.WorkflowSingle,
		ProjectName:  "test-project",
		Agents: config.AgentsConfig{
			Core: []string{"builder", "planner"},
		},
	}

	if err := svc.WriteRulesFile(ctx); err != nil {
		t.Fatalf("failed to write rules file: %v", err)
	}

	// 파일 존재 확인
	rulesPath := svc.GetRulesPath()
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Fatalf("rules file not created: %s", rulesPath)
	}

	// 내용 확인
	content, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("failed to read rules file: %v", err)
	}

	if !strings.Contains(string(content), "single") {
		t.Error("rules file missing workflow type")
	}
}

func TestCleanupRulesFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)

	// 먼저 파일 생성
	ctx := &Context{
		WorkflowType: config.WorkflowSimple,
		ProjectName:  "test",
	}
	if err := svc.WriteRulesFile(ctx); err != nil {
		t.Fatalf("failed to write rules file: %v", err)
	}

	// 파일 존재 확인
	rulesPath := svc.GetRulesPath()
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Fatalf("rules file not created")
	}

	// 정리
	if err := svc.CleanupRulesFile(); err != nil {
		t.Fatalf("failed to cleanup rules file: %v", err)
	}

	// 파일 삭제 확인
	if _, err := os.Stat(rulesPath); !os.IsNotExist(err) {
		t.Error("rules file not deleted after cleanup")
	}
}

func TestGetRulesPath(t *testing.T) {
	svc := NewService("/home/user/project")
	path := svc.GetRulesPath()
	expected := filepath.Join("/home/user/project", ".claude", "rules", "workflow.md")

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestSetActiveAgent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)

	if err := svc.SetActiveAgent("builder"); err != nil {
		t.Fatalf("failed to set active agent: %v", err)
	}

	// rules 파일이 생성되었는지 확인
	rulesPath := svc.GetRulesPath()
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Error("rules file not created after SetActiveAgent")
	}
}

func TestSetActivePort(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-workflow-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)

	if err := svc.SetActivePort("port-001"); err != nil {
		t.Fatalf("failed to set active port: %v", err)
	}

	// rules 파일이 생성되었는지 확인
	rulesPath := svc.GetRulesPath()
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		t.Error("rules file not created after SetActivePort")
	}
}
