package handoff

import (
	"os"
	"testing"

	"github.com/n0roo/pal-kit/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-handoff-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	database, err := db.Open(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to open database: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.Remove(tmpFile.Name())
	}

	return database, cleanup
}

func TestCreateHandoff(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(database)

	content := APIContractContent{
		Entity: "User",
		Fields: []FieldDef{
			{Name: "id", Type: "string"},
			{Name: "name", Type: "string"},
		},
	}

	handoff, err := store.Create("port-001", "port-002", TypeAPIContract, content)
	if err != nil {
		t.Fatalf("Failed to create handoff: %v", err)
	}

	if handoff.ID == "" {
		t.Error("Handoff ID should not be empty")
	}

	if handoff.FromPortID != "port-001" {
		t.Errorf("Expected FromPortID 'port-001', got '%s'", handoff.FromPortID)
	}

	if handoff.ToPortID != "port-002" {
		t.Errorf("Expected ToPortID 'port-002', got '%s'", handoff.ToPortID)
	}

	if handoff.Type != TypeAPIContract {
		t.Errorf("Expected Type '%s', got '%s'", TypeAPIContract, handoff.Type)
	}

	if handoff.TokenCount <= 0 {
		t.Error("TokenCount should be greater than 0")
	}
}

func TestGetHandoff(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(database)

	content := map[string]string{"key": "value"}
	created, err := store.Create("port-001", "port-002", TypeCustom, content)
	if err != nil {
		t.Fatalf("Failed to create handoff: %v", err)
	}

	// Get by ID
	handoff, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Failed to get handoff: %v", err)
	}

	if handoff.ID != created.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.ID, handoff.ID)
	}

	// Get non-existent
	_, err = store.Get("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent handoff")
	}
}

func TestGetHandoffsForPort(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(database)

	// Create handoffs TO port-002
	_, err := store.Create("port-001", "port-002", TypeAPIContract, "content1")
	if err != nil {
		t.Fatalf("Failed to create handoff 1: %v", err)
	}

	_, err = store.Create("port-003", "port-002", TypeFileList, "content2")
	if err != nil {
		t.Fatalf("Failed to create handoff 2: %v", err)
	}

	// Get handoffs for port-002
	handoffs, err := store.GetForPort("port-002")
	if err != nil {
		t.Fatalf("Failed to get handoffs: %v", err)
	}

	if len(handoffs) != 2 {
		t.Errorf("Expected 2 handoffs, got %d", len(handoffs))
	}
}

func TestGetHandoffsFromPort(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(database)

	// Create handoffs FROM port-001
	_, err := store.Create("port-001", "port-002", TypeAPIContract, "content1")
	if err != nil {
		t.Fatalf("Failed to create handoff 1: %v", err)
	}

	_, err = store.Create("port-001", "port-003", TypeFileList, "content2")
	if err != nil {
		t.Fatalf("Failed to create handoff 2: %v", err)
	}

	// Get handoffs from port-001
	handoffs, err := store.GetFromPort("port-001")
	if err != nil {
		t.Fatalf("Failed to get handoffs: %v", err)
	}

	if len(handoffs) != 2 {
		t.Errorf("Expected 2 handoffs, got %d", len(handoffs))
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		minTok  int
		maxTok  int
	}{
		{
			name:    "simple string",
			content: "hello world",
			minTok:  1,
			maxTok:  50,
		},
		{
			name: "api contract",
			content: APIContractContent{
				Entity: "User",
				Fields: []FieldDef{
					{Name: "id", Type: "string"},
					{Name: "email", Type: "string"},
				},
			},
			minTok: 10,
			maxTok: 200,
		},
		{
			name: "file list",
			content: FileListContent{
				Files: []FileInfo{
					{Path: "/src/main.go", Purpose: "main entry"},
					{Path: "/src/handler.go", Purpose: "handlers"},
				},
			},
			minTok: 10,
			maxTok: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := EstimateTokens(tt.content)
			if err != nil {
				t.Fatalf("Failed to estimate tokens: %v", err)
			}

			if tokens < tt.minTok || tokens > tt.maxTok {
				t.Errorf("Expected tokens between %d and %d, got %d", tt.minTok, tt.maxTok, tokens)
			}
		})
	}
}

func TestHandoffTypes(t *testing.T) {
	types := []struct {
		ht       HandoffType
		expected string
	}{
		{TypeAPIContract, "api_contract"},
		{TypeFileList, "file_list"},
		{TypeTypeDef, "type_def"},
		{TypeSchema, "schema"},
		{TypeConfig, "config"},
		{TypeCustom, "custom"},
	}

	for _, tt := range types {
		if string(tt.ht) != tt.expected {
			t.Errorf("Expected HandoffType '%s', got '%s'", tt.expected, tt.ht)
		}
	}
}

func TestMaxTokenBudget(t *testing.T) {
	if MaxTokenBudget != 2000 {
		t.Errorf("Expected MaxTokenBudget 2000, got %d", MaxTokenBudget)
	}
}

func TestTotalTokensForPort(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(database)

	// Create multiple handoffs to same port
	_, _ = store.Create("port-001", "port-002", TypeAPIContract, map[string]string{"a": "b"})
	_, _ = store.Create("port-003", "port-002", TypeFileList, map[string]string{"c": "d"})

	total, err := store.GetTotalTokens("port-002")
	if err != nil {
		t.Fatalf("Failed to get total tokens: %v", err)
	}

	if total <= 0 {
		t.Error("Total tokens should be greater than 0")
	}
}
