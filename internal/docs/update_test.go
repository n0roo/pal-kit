package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n0roo/pal-kit/internal/config"
)

func TestUpdateClaudeMDAfterSetup_NewFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-docs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)

	cfg := &config.ProjectConfig{
		Version: "0.4.0",
		Project: config.ProjectInfo{
			Name:        "test-project",
			Description: "A test project",
		},
		Workflow: config.WorkflowConfig{
			Type: config.WorkflowSingle,
		},
		Agents: config.AgentsConfig{
			Core:    []string{"builder", "planner", "tester"},
			Workers: []string{"worker-go"},
		},
	}

	if err := svc.UpdateClaudeMDAfterSetup(cfg); err != nil {
		t.Fatalf("UpdateClaudeMDAfterSetup failed: %v", err)
	}

	// 파일 존재 확인
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMDPath); os.IsNotExist(err) {
		t.Fatal("CLAUDE.md not created")
	}

	// 내용 확인
	content, err := os.ReadFile(claudeMDPath)
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}

	contentStr := string(content)

	// 프로젝트 이름
	if !strings.Contains(contentStr, "test-project") {
		t.Error("missing project name")
	}

	// 워크플로우 타입
	if !strings.Contains(contentStr, "single") {
		t.Error("missing workflow type")
	}

	// 에이전트 목록
	if !strings.Contains(contentStr, "builder") {
		t.Error("missing core agent")
	}

	if !strings.Contains(contentStr, "worker-go") {
		t.Error("missing worker agent")
	}

	// configured 상태
	if !strings.Contains(contentStr, "pal:config:status=configured") {
		t.Error("missing configured status marker")
	}
}

func TestUpdateClaudeMDAfterSetup_ReplacePending(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-docs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// pending 상태의 CLAUDE.md 생성
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	pendingContent := `# Old Project

## 🚀 PAL Kit 초기 설정 필요

<!-- pal:config:status=pending -->
`
	if err := os.WriteFile(claudeMDPath, []byte(pendingContent), 0644); err != nil {
		t.Fatalf("failed to write pending CLAUDE.md: %v", err)
	}

	svc := NewService(tmpDir)

	cfg := &config.ProjectConfig{
		Version: "0.4.0",
		Project: config.ProjectInfo{
			Name: "new-project",
		},
		Workflow: config.WorkflowConfig{
			Type: config.WorkflowIntegrate,
		},
		Agents: config.AgentsConfig{
			Core: []string{"builder", "manager"},
		},
	}

	if err := svc.UpdateClaudeMDAfterSetup(cfg); err != nil {
		t.Fatalf("UpdateClaudeMDAfterSetup failed: %v", err)
	}

	// 업데이트된 내용 확인
	content, err := os.ReadFile(claudeMDPath)
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}

	contentStr := string(content)

	// 새 프로젝트 이름
	if !strings.Contains(contentStr, "new-project") {
		t.Error("missing new project name")
	}

	// pending이 없어야 함
	if strings.Contains(contentStr, "status=pending") {
		t.Error("should not have pending status")
	}

	// configured 상태
	if !strings.Contains(contentStr, "status=configured") {
		t.Error("missing configured status")
	}
}

func TestUpdateClaudeMDAfterSetup_SkipConfigured(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-docs-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 이미 configured 상태의 CLAUDE.md
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	configuredContent := `# Existing Project

Custom content that should not be changed.

<!-- pal:config:status=configured -->
`
	if err := os.WriteFile(claudeMDPath, []byte(configuredContent), 0644); err != nil {
		t.Fatalf("failed to write configured CLAUDE.md: %v", err)
	}

	svc := NewService(tmpDir)

	cfg := &config.ProjectConfig{
		Version: "0.4.0",
		Project: config.ProjectInfo{
			Name: "different-project",
		},
		Workflow: config.WorkflowConfig{
			Type: config.WorkflowMulti,
		},
	}

	if err := svc.UpdateClaudeMDAfterSetup(cfg); err != nil {
		t.Fatalf("UpdateClaudeMDAfterSetup failed: %v", err)
	}

	// 내용이 변경되지 않았는지 확인
	content, err := os.ReadFile(claudeMDPath)
	if err != nil {
		t.Fatalf("failed to read CLAUDE.md: %v", err)
	}

	contentStr := string(content)

	// 원래 내용 유지
	if !strings.Contains(contentStr, "Existing Project") {
		t.Error("original content should be preserved")
	}

	// 새 프로젝트 이름이 없어야 함
	if strings.Contains(contentStr, "different-project") {
		t.Error("should not have new project name")
	}
}

func TestGenerateConfiguredClaudeMD_Simple(t *testing.T) {
	cfg := &config.ProjectConfig{
		Version: "0.4.0",
		Project: config.ProjectInfo{
			Name: "simple-project",
		},
		Workflow: config.WorkflowConfig{
			Type: config.WorkflowSimple,
		},
		Agents: config.AgentsConfig{
			Core: []string{"collaborator"},
		},
	}

	content := generateConfiguredClaudeMD(cfg)

	// Simple 워크플로우 가이드
	if !strings.Contains(content, "Collaborator") {
		t.Error("missing Collaborator mention for simple workflow")
	}

	if !strings.Contains(content, "대화") {
		t.Error("missing conversational aspect for simple workflow")
	}
}

func TestGenerateConfiguredClaudeMD_Integrate(t *testing.T) {
	cfg := &config.ProjectConfig{
		Version: "0.4.0",
		Project: config.ProjectInfo{
			Name: "integrate-project",
		},
		Workflow: config.WorkflowConfig{
			Type: config.WorkflowIntegrate,
		},
		Agents: config.AgentsConfig{
			Core:    []string{"builder", "planner", "manager"},
			Workers: []string{"worker-go", "worker-react"},
		},
	}

	content := generateConfiguredClaudeMD(cfg)

	// Integrate 워크플로우 가이드
	if !strings.Contains(content, "빌더 세션") {
		t.Error("missing builder session mention")
	}

	if !strings.Contains(content, "워커 세션") {
		t.Error("missing worker session mention")
	}

	// 파이프라인 명령어
	if !strings.Contains(content, "pipeline") {
		t.Error("missing pipeline command for integrate workflow")
	}

	// 워커 에이전트 목록
	if !strings.Contains(content, "worker-go") {
		t.Error("missing worker-go in agents list")
	}
}

func TestGetWorkflowGuide(t *testing.T) {
	tests := []struct {
		wt       config.WorkflowType
		contains []string
	}{
		{
			config.WorkflowSimple,
			[]string{"Collaborator", "대화", "협업"},
		},
		{
			config.WorkflowSingle,
			[]string{"역할", "전환", "Builder"},
		},
		{
			config.WorkflowIntegrate,
			[]string{"빌더", "워커", "세션"},
		},
		{
			config.WorkflowMulti,
			[]string{"Integrate", "병렬", "조율"},
		},
	}

	for _, tt := range tests {
		guide := getWorkflowGuide(tt.wt)

		if guide == "" {
			t.Errorf("empty guide for %s", tt.wt)
			continue
		}

		for _, expected := range tt.contains {
			if !strings.Contains(guide, expected) {
				t.Errorf("%s guide missing: %s", tt.wt, expected)
			}
		}
	}
}
