package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultProjectConfig(t *testing.T) {
	cfg := DefaultProjectConfig("test-project")

	if cfg.Version != "0.4.0" {
		t.Errorf("expected version 0.4.0, got %s", cfg.Version)
	}

	if cfg.Project.Name != "test-project" {
		t.Errorf("expected project name test-project, got %s", cfg.Project.Name)
	}

	if cfg.Workflow.Type != WorkflowSimple {
		t.Errorf("expected workflow type simple, got %s", cfg.Workflow.Type)
	}

	if len(cfg.Agents.Core) != 1 || cfg.Agents.Core[0] != "collaborator" {
		t.Errorf("expected core agent [collaborator], got %v", cfg.Agents.Core)
	}
}

func TestSaveAndLoadProjectConfig(t *testing.T) {
	// 임시 디렉토리 생성
	tmpDir, err := os.MkdirTemp("", "pal-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 설정 생성
	cfg := DefaultProjectConfig("test-project")
	cfg.Workflow.Type = WorkflowIntegrate
	cfg.Agents.Workers = []string{"worker-go", "worker-react"}

	// 저장
	if err := SaveProjectConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// 파일 존재 확인
	configPath := ProjectConfigPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("config file not created: %s", configPath)
	}

	// 로드
	loaded, err := LoadProjectConfig(tmpDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// 검증
	if loaded.Version != cfg.Version {
		t.Errorf("version mismatch: expected %s, got %s", cfg.Version, loaded.Version)
	}

	if loaded.Project.Name != cfg.Project.Name {
		t.Errorf("project name mismatch: expected %s, got %s", cfg.Project.Name, loaded.Project.Name)
	}

	if loaded.Workflow.Type != cfg.Workflow.Type {
		t.Errorf("workflow type mismatch: expected %s, got %s", cfg.Workflow.Type, loaded.Workflow.Type)
	}

	if len(loaded.Agents.Workers) != 2 {
		t.Errorf("workers count mismatch: expected 2, got %d", len(loaded.Agents.Workers))
	}
}

func TestLoadProjectConfigNotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = LoadProjectConfig(tmpDir)
	if err == nil {
		t.Error("expected error for non-existent config, got nil")
	}
}

func TestHasProjectConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-config-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 설정 없음
	if HasProjectConfig(tmpDir) {
		t.Error("expected no config, but HasProjectConfig returned true")
	}

	// 설정 생성
	cfg := DefaultProjectConfig("test")
	if err := SaveProjectConfig(tmpDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// 설정 있음
	if !HasProjectConfig(tmpDir) {
		t.Error("expected config exists, but HasProjectConfig returned false")
	}
}

func TestGetWorkflowTypes(t *testing.T) {
	types := GetWorkflowTypes()

	if len(types) != 4 {
		t.Errorf("expected 4 workflow types, got %d", len(types))
	}

	expected := []WorkflowType{WorkflowSimple, WorkflowSingle, WorkflowIntegrate, WorkflowMulti}
	for i, wt := range expected {
		if types[i] != wt {
			t.Errorf("workflow type %d mismatch: expected %s, got %s", i, wt, types[i])
		}
	}
}

func TestWorkflowDescription(t *testing.T) {
	tests := []struct {
		wt       WorkflowType
		notEmpty bool
	}{
		{WorkflowSimple, true},
		{WorkflowSingle, true},
		{WorkflowIntegrate, true},
		{WorkflowMulti, true},
		{WorkflowType("unknown"), false},
	}

	for _, tt := range tests {
		desc := WorkflowDescription(tt.wt)
		if tt.notEmpty && desc == "" {
			t.Errorf("expected description for %s, got empty", tt.wt)
		}
		if !tt.notEmpty && desc != "" {
			t.Errorf("expected empty description for %s, got %s", tt.wt, desc)
		}
	}
}

func TestDefaultAgentsForWorkflow(t *testing.T) {
	tests := []struct {
		wt            WorkflowType
		expectedCore  []string
		minCoreCount  int
	}{
		{WorkflowSimple, []string{"collaborator"}, 1},
		{WorkflowSingle, nil, 5},
		{WorkflowIntegrate, nil, 6},
		{WorkflowMulti, nil, 6},
	}

	for _, tt := range tests {
		agents := DefaultAgentsForWorkflow(tt.wt)

		if tt.expectedCore != nil {
			if len(agents.Core) != len(tt.expectedCore) {
				t.Errorf("%s: expected %d core agents, got %d", tt.wt, len(tt.expectedCore), len(agents.Core))
			}
		}

		if len(agents.Core) < tt.minCoreCount {
			t.Errorf("%s: expected at least %d core agents, got %d", tt.wt, tt.minCoreCount, len(agents.Core))
		}
	}
}

func TestProjectConfigPath(t *testing.T) {
	path := ProjectConfigPath("/home/user/project")
	expected := filepath.Join("/home/user/project", ".pal", "config.yaml")

	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
