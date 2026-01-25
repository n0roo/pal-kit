package orchestrator

import (
	"os"
	"testing"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/message"
	"github.com/n0roo/pal-kit/internal/session"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
	t.Helper()

	// Create temp file for test DB
	tmpFile, err := os.CreateTemp("", "test-pal-*.db")
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

func TestCreateOrchestration(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	svc := NewService(database, sessionSvc, msgStore)

	ports := []AtomicPort{
		{PortID: "port-001", Order: 1},
		{PortID: "port-002", Order: 2, DependsOn: []string{"port-001"}},
		{PortID: "port-003", Order: 3, DependsOn: []string{"port-002"}},
	}

	orch, err := svc.CreateOrchestration("Test Orchestration", "A test orchestration", ports)
	if err != nil {
		t.Fatalf("Failed to create orchestration: %v", err)
	}

	if orch.ID == "" {
		t.Error("Orchestration ID should not be empty")
	}

	if orch.Title != "Test Orchestration" {
		t.Errorf("Expected title 'Test Orchestration', got '%s'", orch.Title)
	}

	if orch.Status != StatusPending {
		t.Errorf("Expected status '%s', got '%s'", StatusPending, orch.Status)
	}

	if len(orch.AtomicPorts) != 3 {
		t.Errorf("Expected 3 atomic ports, got %d", len(orch.AtomicPorts))
	}
}

func TestListOrchestrations(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	svc := NewService(database, sessionSvc, msgStore)

	// Create multiple orchestrations
	for i := 0; i < 3; i++ {
		ports := []AtomicPort{{PortID: "port-" + string(rune('a'+i)), Order: 1}}
		_, err := svc.CreateOrchestration("Orch-"+string(rune('0'+i)), "", ports)
		if err != nil {
			t.Fatalf("Failed to create orchestration %d: %v", i, err)
		}
	}

	// List all
	list, err := svc.ListOrchestrations("", 10)
	if err != nil {
		t.Fatalf("Failed to list orchestrations: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 orchestrations, got %d", len(list))
	}

	// List by status
	list, err = svc.ListOrchestrations(StatusPending, 10)
	if err != nil {
		t.Fatalf("Failed to list pending orchestrations: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 pending orchestrations, got %d", len(list))
	}
}

func TestGetOrchestration(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	svc := NewService(database, sessionSvc, msgStore)

	ports := []AtomicPort{{PortID: "port-001", Order: 1}}
	created, err := svc.CreateOrchestration("Test", "", ports)
	if err != nil {
		t.Fatalf("Failed to create orchestration: %v", err)
	}

	// Get by ID
	orch, err := svc.GetOrchestration(created.ID)
	if err != nil {
		t.Fatalf("Failed to get orchestration: %v", err)
	}

	if orch.ID != created.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.ID, orch.ID)
	}

	// Get non-existent
	_, err = svc.GetOrchestration("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent orchestration")
	}
}

func TestOrchestrationStats(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	svc := NewService(database, sessionSvc, msgStore)

	ports := []AtomicPort{
		{PortID: "port-001", Order: 1},
		{PortID: "port-002", Order: 2},
	}

	orch, err := svc.CreateOrchestration("Test", "", ports)
	if err != nil {
		t.Fatalf("Failed to create orchestration: %v", err)
	}

	stats, err := svc.GetOrchestrationStats(orch.ID)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalPorts != 2 {
		t.Errorf("Expected 2 total ports, got %d", stats.TotalPorts)
	}

	if stats.PendingPorts != 2 {
		t.Errorf("Expected 2 pending ports, got %d", stats.PendingPorts)
	}
}

func TestWorkerType(t *testing.T) {
	tests := []struct {
		wt       WorkerType
		expected string
	}{
		{WorkerTypeImpl, "impl"},
		{WorkerTypeTest, "test"},
		{WorkerTypePair, "impl_test_pair"},
		{WorkerTypeSingle, "single"},
	}

	for _, tt := range tests {
		if string(tt.wt) != tt.expected {
			t.Errorf("Expected WorkerType '%s', got '%s'", tt.expected, tt.wt)
		}
	}
}

func TestOrchestrationStatus(t *testing.T) {
	tests := []struct {
		status   OrchestrationStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusPaused, "paused"},
		{StatusComplete, "complete"},
		{StatusFailed, "failed"},
		{StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected status '%s', got '%s'", tt.expected, tt.status)
		}
	}
}
