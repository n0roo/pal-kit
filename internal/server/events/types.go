package events

import (
	"time"
)

// EventType defines the type of event
type EventType string

const (
	// Session events
	EventSessionStart  EventType = "session:start"
	EventSessionEnd    EventType = "session:end"
	EventSessionUpdate EventType = "session:update"

	// Attention events
	EventAttentionWarning  EventType = "attention:warning"  // 80%
	EventAttentionCritical EventType = "attention:critical" // 90%
	EventCompactTriggered  EventType = "compact:triggered"
	EventCheckpointCreated EventType = "checkpoint:created"

	// Port events
	EventPortStart   EventType = "port:start"
	EventPortEnd     EventType = "port:end"
	EventPortBlocked EventType = "port:blocked"

	// Checklist events
	EventChecklistFailed EventType = "checklist:failed"
	EventChecklistPassed EventType = "checklist:passed"

	// Escalation events
	EventEscalationCreated  EventType = "escalation:created"
	EventEscalationResolved EventType = "escalation:resolved"

	// Message events
	EventMessageReceived EventType = "message:received"

	// Build events
	EventBuildFailed EventType = "build:failed"
	EventTestFailed  EventType = "test:failed"
)

// Event represents a real-time event
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	SessionID string      `json:"session_id,omitempty"`
	PortID    string      `json:"port_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, data interface{}) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// WithSession sets session ID
func (e *Event) WithSession(sessionID string) *Event {
	e.SessionID = sessionID
	return e
}

// WithPort sets port ID
func (e *Event) WithPort(portID string) *Event {
	e.PortID = portID
	return e
}

// SessionStartData represents session start event data
type SessionStartData struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Type        string `json:"type,omitempty"`
	ProjectRoot string `json:"project_root,omitempty"`
}

// SessionEndData represents session end event data
type SessionEndData struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
	Status string `json:"status,omitempty"`
}

// AttentionWarningData represents attention warning event data
type AttentionWarningData struct {
	UsagePercent float64 `json:"usage_percent"`
	TokensUsed   int     `json:"tokens_used"`
	TokenBudget  int     `json:"token_budget"`
	Warning      string  `json:"warning"`
}

// CompactTriggeredData represents compact triggered event data
type CompactTriggeredData struct {
	Trigger      string `json:"trigger"`
	CheckpointID string `json:"checkpoint_id,omitempty"`
	RecoveryHint string `json:"recovery_hint,omitempty"`
}

// CheckpointCreatedData represents checkpoint created event data
type CheckpointCreatedData struct {
	ID          string   `json:"id"`
	Summary     string   `json:"summary"`
	TriggerType string   `json:"trigger_type"`
	ActiveFiles []string `json:"active_files,omitempty"`
}

// PortStartData represents port start event data
type PortStartData struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	Checklist []string `json:"checklist,omitempty"`
}

// PortEndData represents port end event data
type PortEndData struct {
	ID       string `json:"id"`
	Status   string `json:"status"` // complete, blocked
	Duration int64  `json:"duration_secs,omitempty"`
}

// ChecklistResultData represents checklist result event data
type ChecklistResultData struct {
	Passed      bool          `json:"passed"`
	PassedCount int           `json:"passed_count"`
	FailedCount int           `json:"failed_count"`
	Items       []CheckItem   `json:"items"`
	BlockedBy   []string      `json:"blocked_by,omitempty"`
}

// CheckItem represents a checklist item result
type CheckItem struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// EscalationData represents escalation event data
type EscalationData struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Issue      string `json:"issue"`
	Suggestion string `json:"suggestion,omitempty"`
}

// BuildFailedData represents build/test failed event data
type BuildFailedData struct {
	FailType string `json:"fail_type"` // build, test
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}
