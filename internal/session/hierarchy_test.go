package session

import (
	"testing"
)

// 참고: setupTestDB는 session_test.go에 정의되어 있음

func TestSessionTypes(t *testing.T) {
	tests := []struct {
		sessionType string
		expected    string
	}{
		{TypeBuild, "build"},
		{TypeOperator, "operator"},
		{TypeWorker, "worker"},
		{TypeTest, "test"},
	}

	for _, tt := range tests {
		if tt.sessionType != tt.expected {
			t.Errorf("Expected type '%s', got '%s'", tt.expected, tt.sessionType)
		}
	}
}

func TestStartHierarchicalBuild(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// Create build session (root)
	opts := HierarchyStartOptions{
		Title:       "Test Build",
		Type:        TypeBuild,
		TokenBudget: 15000,
		ProjectRoot: "/test/project",
	}

	session, err := svc.StartHierarchical(opts)
	if err != nil {
		t.Fatalf("Failed to start build session: %v", err)
	}

	if session.Session.ID == "" {
		t.Error("Session ID should not be empty")
	}

	if session.Type != TypeBuild {
		t.Errorf("Expected type '%s', got '%s'", TypeBuild, session.Type)
	}

	if session.Depth != 0 {
		t.Errorf("Expected depth 0, got %d", session.Depth)
	}

	if session.Path != session.Session.ID {
		t.Errorf("Expected path '%s', got '%s'", session.Session.ID, session.Path)
	}
}

func TestStartHierarchicalOperator(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// Create build session first
	build, err := svc.StartHierarchical(HierarchyStartOptions{
		Title: "Build",
		Type:  TypeBuild,
	})
	if err != nil {
		t.Fatalf("Failed to create build: %v", err)
	}

	// Create operator session under build
	operator, err := svc.StartHierarchical(HierarchyStartOptions{
		Title:    "Operator",
		Type:     TypeOperator,
		ParentID: build.Session.ID,
	})
	if err != nil {
		t.Fatalf("Failed to create operator: %v", err)
	}

	if operator.Depth != 1 {
		t.Errorf("Expected depth 1, got %d", operator.Depth)
	}

	if !operator.ParentID.Valid || operator.ParentID.String != build.Session.ID {
		t.Errorf("Expected parent ID '%s'", build.Session.ID)
	}

	if !operator.RootID.Valid || operator.RootID.String != build.Session.ID {
		t.Errorf("Expected root ID '%s'", build.Session.ID)
	}
}

func TestStartHierarchicalWorker(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// Build -> Operator -> Worker
	build, _ := svc.StartHierarchical(HierarchyStartOptions{Title: "Build", Type: TypeBuild})
	operator, _ := svc.StartHierarchical(HierarchyStartOptions{Title: "Operator", Type: TypeOperator, ParentID: build.Session.ID})

	worker, err := svc.StartHierarchical(HierarchyStartOptions{
		Title:    "Worker",
		Type:     TypeWorker,
		ParentID: operator.Session.ID,
		PortID:   "port-001",
	})
	if err != nil {
		t.Fatalf("Failed to create worker: %v", err)
	}

	if worker.Depth != 2 {
		t.Errorf("Expected depth 2, got %d", worker.Depth)
	}

	// Root should still be build
	if !worker.RootID.Valid || worker.RootID.String != build.Session.ID {
		t.Errorf("Expected root ID '%s'", build.Session.ID)
	}
}

func TestGetHierarchical(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	created, err := svc.StartHierarchical(HierarchyStartOptions{
		Title: "Test Session",
		Type:  TypeBuild,
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get by ID
	session, err := svc.GetHierarchical(created.Session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session.Session.ID != created.Session.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.Session.ID, session.Session.ID)
	}

	// Get non-existent
	_, err = svc.GetHierarchical("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestHierarchyStartOptionsDefaults(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// Empty options
	session, err := svc.StartHierarchical(HierarchyStartOptions{})
	if err != nil {
		t.Fatalf("Failed with empty options: %v", err)
	}

	// Should have defaults
	if session.Session.ID == "" {
		t.Error("ID should be auto-generated")
	}

	if session.Type != TypeWorker {
		t.Errorf("Expected default type '%s', got '%s'", TypeWorker, session.Type)
	}
}

func TestInvalidParentID(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// Try to create with invalid parent
	_, err := svc.StartHierarchical(HierarchyStartOptions{
		Title:    "Orphan",
		ParentID: "invalid-parent-id",
	})

	if err == nil {
		t.Error("Expected error for invalid parent ID")
	}
}

func TestHierarchicalSessionDepthPath(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	// Create deep hierarchy: build -> op -> worker -> test
	build, _ := svc.StartHierarchical(HierarchyStartOptions{Title: "Build", Type: TypeBuild})
	op, _ := svc.StartHierarchical(HierarchyStartOptions{Title: "Op", Type: TypeOperator, ParentID: build.Session.ID})
	worker, _ := svc.StartHierarchical(HierarchyStartOptions{Title: "Worker", Type: TypeWorker, ParentID: op.Session.ID})
	test, _ := svc.StartHierarchical(HierarchyStartOptions{Title: "Test", Type: TypeTest, ParentID: worker.Session.ID})

	// Check depths
	if build.Depth != 0 {
		t.Errorf("Build depth = %d, want 0", build.Depth)
	}
	if op.Depth != 1 {
		t.Errorf("Operator depth = %d, want 1", op.Depth)
	}
	if worker.Depth != 2 {
		t.Errorf("Worker depth = %d, want 2", worker.Depth)
	}
	if test.Depth != 3 {
		t.Errorf("Test depth = %d, want 3", test.Depth)
	}

	// All should have same root
	if test.RootID.String != build.Session.ID {
		t.Errorf("Test root = %s, want %s", test.RootID.String, build.Session.ID)
	}
}
