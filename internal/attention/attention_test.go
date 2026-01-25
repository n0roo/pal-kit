package attention

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-attention-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	db, err := sql.Open("sqlite3", tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS session_attention (
			session_id TEXT PRIMARY KEY,
			port_id TEXT,
			current_context_hash TEXT,
			loaded_tokens INTEGER DEFAULT 0,
			available_tokens INTEGER DEFAULT 15000,
			focus_score REAL DEFAULT 1.0,
			drift_count INTEGER DEFAULT 0,
			last_compaction_at DATETIME,
			loaded_files TEXT,
			loaded_conventions TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS compact_events (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			trigger_reason TEXT NOT NULL,
			before_tokens INTEGER,
			after_tokens INTEGER,
			preserved_context TEXT,
			discarded_context TEXT,
			checkpoint_before TEXT,
			recovery_hint TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		db.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to create tables: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}

func TestInitializeAttention(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)

	err := store.Initialize("session-001", "port-001", 10000)
	if err != nil {
		t.Fatalf("Failed to initialize attention: %v", err)
	}

	// Verify
	att, err := store.Get("session-001")
	if err != nil {
		t.Fatalf("Failed to get attention: %v", err)
	}

	if att.SessionID != "session-001" {
		t.Errorf("Expected session ID 'session-001', got '%s'", att.SessionID)
	}

	if att.PortID != "port-001" {
		t.Errorf("Expected port ID 'port-001', got '%s'", att.PortID)
	}

	if att.AvailableTokens != 10000 {
		t.Errorf("Expected available tokens 10000, got %d", att.AvailableTokens)
	}

	if att.FocusScore != 1.0 {
		t.Errorf("Expected focus score 1.0, got %f", att.FocusScore)
	}
}

func TestInitializeWithDefaultBudget(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)

	// Initialize with 0 budget - should use default
	err := store.Initialize("session-001", "", 0)
	if err != nil {
		t.Fatalf("Failed to initialize attention: %v", err)
	}

	att, err := store.Get("session-001")
	if err != nil {
		t.Fatalf("Failed to get attention: %v", err)
	}

	if att.AvailableTokens != DefaultTokenBudget {
		t.Errorf("Expected default budget %d, got %d", DefaultTokenBudget, att.AvailableTokens)
	}
}

func TestGetNonExistent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)

	_, err := store.Get("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestAttentionStatus(t *testing.T) {
	tests := []struct {
		status   AttentionStatus
		expected string
	}{
		{StatusFocused, "focused"},
		{StatusDrifting, "drifting"},
		{StatusWarning, "warning"},
		{StatusCritical, "critical"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Expected status '%s', got '%s'", tt.expected, tt.status)
		}
	}
}

func TestDefaultTokenBudget(t *testing.T) {
	if DefaultTokenBudget != 15000 {
		t.Errorf("Expected DefaultTokenBudget 15000, got %d", DefaultTokenBudget)
	}
}

func TestSessionAttentionStruct(t *testing.T) {
	att := SessionAttention{
		SessionID:       "test-session",
		PortID:          "test-port",
		LoadedTokens:    5000,
		AvailableTokens: 15000,
		FocusScore:      0.85,
		DriftCount:      2,
		LoadedFiles:     []string{"file1.go", "file2.go"},
		LoadedConventions: []string{"conv1", "conv2"},
	}

	if att.SessionID != "test-session" {
		t.Errorf("Expected SessionID 'test-session', got '%s'", att.SessionID)
	}

	if len(att.LoadedFiles) != 2 {
		t.Errorf("Expected 2 loaded files, got %d", len(att.LoadedFiles))
	}

	if len(att.LoadedConventions) != 2 {
		t.Errorf("Expected 2 loaded conventions, got %d", len(att.LoadedConventions))
	}
}

func TestCompactEventStruct(t *testing.T) {
	event := CompactEvent{
		ID:              "event-001",
		SessionID:       "session-001",
		TriggerReason:   "token_limit",
		BeforeTokens:    14000,
		AfterTokens:     8000,
		PreservedContext: []string{"context1"},
		DiscardedContext: []string{"context2"},
		RecoveryHint:    "Check point A",
	}

	if event.ID != "event-001" {
		t.Errorf("Expected ID 'event-001', got '%s'", event.ID)
	}

	if event.TriggerReason != "token_limit" {
		t.Errorf("Expected TriggerReason 'token_limit', got '%s'", event.TriggerReason)
	}

	if event.BeforeTokens != 14000 {
		t.Errorf("Expected BeforeTokens 14000, got %d", event.BeforeTokens)
	}
}

func TestUpdateDuplicateSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)

	// First init
	err := store.Initialize("session-001", "port-001", 10000)
	if err != nil {
		t.Fatalf("First init failed: %v", err)
	}

	// Second init (update)
	err = store.Initialize("session-001", "port-002", 20000)
	if err != nil {
		t.Fatalf("Second init failed: %v", err)
	}

	att, err := store.Get("session-001")
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if att.PortID != "port-002" {
		t.Errorf("Expected port 'port-002', got '%s'", att.PortID)
	}

	if att.AvailableTokens != 20000 {
		t.Errorf("Expected 20000 tokens, got %d", att.AvailableTokens)
	}
}
