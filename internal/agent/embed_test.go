package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListTemplates(t *testing.T) {
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	if len(templates) == 0 {
		t.Error("expected some templates, got none")
	}

	// 필수 템플릿 확인
	expectedTemplates := []string{
		"core/collaborator.yaml",
		"core/builder.yaml",
		"core/planner.yaml",
		"workers/backend/go.yaml",
		"workers/frontend/react.yaml",
	}

	templateSet := make(map[string]bool)
	for _, tmpl := range templates {
		templateSet[tmpl] = true
	}

	for _, expected := range expectedTemplates {
		if !templateSet[expected] {
			t.Errorf("expected template %s not found", expected)
		}
	}
}

func TestGetTemplate(t *testing.T) {
	// 존재하는 템플릿
	content, err := GetTemplate("core/collaborator.yaml")
	if err != nil {
		t.Fatalf("GetTemplate failed: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty content")
	}

	// YAML 구조 확인
	if !strings.Contains(string(content), "agent:") {
		t.Error("template missing 'agent:' key")
	}

	if !strings.Contains(string(content), "id:") {
		t.Error("template missing 'id:' key")
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	_, err := GetTemplate("nonexistent/template.yaml")
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestGetTemplate_AllTemplates(t *testing.T) {
	templates, err := ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	for _, tmpl := range templates {
		content, err := GetTemplate(tmpl)
		if err != nil {
			t.Errorf("GetTemplate(%s) failed: %v", tmpl, err)
			continue
		}

		if len(content) == 0 {
			t.Errorf("GetTemplate(%s) returned empty content", tmpl)
		}

		// 기본 YAML 구조 확인
		if !strings.Contains(string(content), "agent:") {
			t.Errorf("template %s missing 'agent:' key", tmpl)
		}
	}
}

func TestInstallTemplates(t *testing.T) {
	// 임시 디렉토리 생성
	tmpDir, err := os.MkdirTemp("", "pal-agent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 템플릿 설치
	if err := InstallTemplates(tmpDir); err != nil {
		t.Fatalf("InstallTemplates failed: %v", err)
	}

	// 설치된 파일 확인
	expectedFiles := []string{
		"core/collaborator.yaml",
		"core/builder.yaml",
		"core/planner.yaml",
		"core/architect.yaml",
		"core/manager.yaml",
		"core/tester.yaml",
		"core/logger.yaml",
		"workers/backend/go.yaml",
		"workers/backend/kotlin.yaml",
		"workers/backend/nestjs.yaml",
		"workers/frontend/react.yaml",
		"workers/frontend/next.yaml",
	}

	for _, relPath := range expectedFiles {
		fullPath := filepath.Join(tmpDir, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file not installed: %s", relPath)
		}
	}
}

func TestInstallTemplates_Content(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-agent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := InstallTemplates(tmpDir); err != nil {
		t.Fatalf("InstallTemplates failed: %v", err)
	}

	// 설치된 파일 내용 확인
	testFiles := []struct {
		path     string
		contains []string
	}{
		{
			"core/collaborator.yaml",
			[]string{"collaborator", "simple", "prompt:"},
		},
		{
			"core/builder.yaml",
			[]string{"builder", "포트", "분해"},
		},
		{
			"workers/backend/go.yaml",
			[]string{"Go", "worker", "go build"},
		},
		{
			"workers/frontend/react.yaml",
			[]string{"React", "component", "useState"},
		},
	}

	for _, tf := range testFiles {
		fullPath := filepath.Join(tmpDir, tf.path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("failed to read %s: %v", tf.path, err)
			continue
		}

		for _, expected := range tf.contains {
			if !strings.Contains(string(content), expected) {
				t.Errorf("%s missing expected content: %s", tf.path, expected)
			}
		}
	}
}

func TestCopyTemplateToProject(t *testing.T) {
	// 임시 프로젝트 디렉토리
	projectDir, err := os.MkdirTemp("", "pal-project-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(projectDir)

	// agents 디렉토리 생성
	agentsDir := filepath.Join(projectDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// core 템플릿 복사
	if err := CopyTemplateToProject("core/builder.yaml", projectDir); err != nil {
		t.Fatalf("CopyTemplateToProject failed: %v", err)
	}

	// 복사된 파일 확인 (core/builder.yaml → agents/builder.yaml)
	copiedPath := filepath.Join(agentsDir, "builder.yaml")
	if _, err := os.Stat(copiedPath); os.IsNotExist(err) {
		t.Error("template not copied to project")
	}

	// 내용 확인
	content, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}

	if !strings.Contains(string(content), "builder") {
		t.Error("copied file missing expected content")
	}
}

func TestCopyTemplateToProject_Worker(t *testing.T) {
	projectDir, err := os.MkdirTemp("", "pal-project-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(projectDir)

	agentsDir := filepath.Join(projectDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// worker 템플릿 복사
	if err := CopyTemplateToProject("workers/backend/go.yaml", projectDir); err != nil {
		t.Fatalf("CopyTemplateToProject failed: %v", err)
	}

	// 복사된 파일 확인 (workers/backend/go.yaml → agents/worker-go.yaml)
	copiedPath := filepath.Join(agentsDir, "worker-go.yaml")
	if _, err := os.Stat(copiedPath); os.IsNotExist(err) {
		t.Error("worker template not copied with correct name")
	}
}

func TestInstallTemplates_Idempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-agent-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 첫 번째 설치
	if err := InstallTemplates(tmpDir); err != nil {
		t.Fatalf("first InstallTemplates failed: %v", err)
	}

	// 두 번째 설치 (덮어쓰기)
	if err := InstallTemplates(tmpDir); err != nil {
		t.Fatalf("second InstallTemplates failed: %v", err)
	}

	// 파일이 여전히 존재하는지 확인
	collaboratorPath := filepath.Join(tmpDir, "core/collaborator.yaml")
	if _, err := os.Stat(collaboratorPath); os.IsNotExist(err) {
		t.Error("template file missing after second install")
	}
}
