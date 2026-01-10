package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-agent-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	// agents 디렉토리 생성
	agentsDir := filepath.Join(tmpDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("agents 디렉토리 생성 실패: %v", err)
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

func TestLoad_Empty(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	err := svc.Load()
	if err != nil {
		t.Fatalf("빈 디렉토리 로드 실패: %v", err)
	}

	agents, _ := svc.List()
	if len(agents) != 0 {
		t.Errorf("에이전트 수 = %d, want 0", len(agents))
	}
}

func TestLoad_YAMLFile(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	// YAML 에이전트 파일 생성
	agentContent := `agent:
  id: worker-1
  name: Worker Agent
  type: worker
  description: Test worker agent
  prompt: |
    You are a helpful worker agent.
  tools:
    - bash
    - editor
  config:
    max_tokens: "8000"
`
	agentPath := filepath.Join(projectRoot, "agents", "worker-1.yaml")
	os.WriteFile(agentPath, []byte(agentContent), 0644)

	svc := NewService(projectRoot)
	err := svc.Load()
	if err != nil {
		t.Fatalf("로드 실패: %v", err)
	}

	agent, err := svc.Get("worker-1")
	if err != nil {
		t.Fatalf("에이전트 조회 실패: %v", err)
	}

	if agent.Name != "Worker Agent" {
		t.Errorf("Name = %s, want Worker Agent", agent.Name)
	}
	if agent.Type != "worker" {
		t.Errorf("Type = %s, want worker", agent.Type)
	}
	if len(agent.Tools) != 2 {
		t.Errorf("Tools 수 = %d, want 2", len(agent.Tools))
	}
}

func TestLoad_MarkdownFile(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	// Markdown 에이전트 파일 생성
	promptContent := `# Builder Agent

You are a builder agent responsible for managing pipelines.

## Responsibilities
- Create and manage ports
- Coordinate work distribution
`
	agentPath := filepath.Join(projectRoot, "agents", "builder.md")
	os.WriteFile(agentPath, []byte(promptContent), 0644)

	svc := NewService(projectRoot)
	err := svc.Load()
	if err != nil {
		t.Fatalf("로드 실패: %v", err)
	}

	agent, err := svc.Get("builder")
	if err != nil {
		t.Fatalf("에이전트 조회 실패: %v", err)
	}

	if agent.ID != "builder" {
		t.Errorf("ID = %s, want builder", agent.ID)
	}
	if agent.Prompt == "" {
		t.Error("Prompt가 비어있음")
	}
}

func TestCreate(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	agent := &Agent{
		ID:          "test-agent",
		Name:        "Test Agent",
		Type:        "worker",
		Description: "A test agent",
		Prompt:      "You are a test agent.",
	}

	err := svc.Create(agent)
	if err != nil {
		t.Fatalf("에이전트 생성 실패: %v", err)
	}

	// 파일 존재 확인
	agentPath := filepath.Join(projectRoot, "agents", "test-agent.yaml")
	if _, err := os.Stat(agentPath); os.IsNotExist(err) {
		t.Error("에이전트 파일이 생성되지 않음")
	}

	// 다시 로드해서 확인
	svc2 := NewService(projectRoot)
	loaded, err := svc2.Get("test-agent")
	if err != nil {
		t.Fatalf("생성된 에이전트 조회 실패: %v", err)
	}
	if loaded.Name != "Test Agent" {
		t.Errorf("Name = %s, want Test Agent", loaded.Name)
	}
}

func TestDelete(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 에이전트 생성
	agent := &Agent{
		ID:   "delete-test",
		Name: "Delete Test",
		Type: "worker",
	}
	svc.Create(agent)

	// 삭제
	err := svc.Delete("delete-test")
	if err != nil {
		t.Fatalf("삭제 실패: %v", err)
	}

	// 조회 시 에러
	_, err = svc.Get("delete-test")
	if err == nil {
		t.Error("삭제된 에이전트 조회가 성공함")
	}
}

func TestGetPrompt(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	agent := &Agent{
		ID:     "prompt-test",
		Name:   "Prompt Test",
		Type:   "worker",
		Prompt: "This is the prompt content.",
	}
	svc.Create(agent)

	prompt, err := svc.GetPrompt("prompt-test")
	if err != nil {
		t.Fatalf("프롬프트 조회 실패: %v", err)
	}

	if prompt != "This is the prompt content." {
		t.Errorf("Prompt = %s, want This is the prompt content.", prompt)
	}
}

func TestList(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 여러 에이전트 생성
	for i := 0; i < 3; i++ {
		agent := &Agent{
			ID:   string(rune('a' + i)),
			Name: string(rune('A' + i)),
			Type: "worker",
		}
		svc.Create(agent)
	}

	agents, err := svc.List()
	if err != nil {
		t.Fatalf("목록 조회 실패: %v", err)
	}

	if len(agents) != 3 {
		t.Errorf("에이전트 수 = %d, want 3", len(agents))
	}
}

func TestGet_NotFound(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	_, err := svc.Get("nonexistent")
	if err == nil {
		t.Error("존재하지 않는 에이전트 조회가 성공함")
	}
}

func TestGetAgentTypes(t *testing.T) {
	types := GetAgentTypes()
	if len(types) == 0 {
		t.Error("에이전트 타입이 비어있음")
	}

	// builder, worker 포함 확인
	hasBuilder := false
	hasWorker := false
	for _, t := range types {
		if t == "builder" {
			hasBuilder = true
		}
		if t == "worker" {
			hasWorker = true
		}
	}
	if !hasBuilder {
		t.Error("builder 타입이 없음")
	}
	if !hasWorker {
		t.Error("worker 타입이 없음")
	}
}
