package server

import (
	"log"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/server/events"
)

// WebSocket 메시지 타입
const (
	WSTypeSubscribe   = "subscribe"
	WSTypeUnsubscribe = "unsubscribe"
	WSTypeEvent       = "event"
	WSTypePing        = "ping"
	WSTypePong        = "pong"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

// EventType defines event types (legacy compatibility)
type EventType string

const (
	EventSessionStart          EventType = "session.start"
	EventSessionEnd            EventType = "session.end"
	EventSessionUpdate         EventType = "session.update"
	EventOrchestrationStart    EventType = "orchestration.start"
	EventOrchestrationUpdate   EventType = "orchestration.update"
	EventOrchestrationComplete EventType = "orchestration.complete"
	EventWorkerSpawn           EventType = "worker.spawn"
	EventWorkerComplete        EventType = "worker.complete"
	EventWorkerProgress        EventType = "worker.progress"  // 신규: Worker 진행률
	EventWorkerFeedback        EventType = "worker.feedback"  // 신규: Worker 피드백
	EventPortUpdate            EventType = "port.update"
	EventAttentionWarning      EventType = "attention.warning"
	EventEscalation            EventType = "escalation.new"
	EventEscalationResolved    EventType = "escalation.resolved" // 신규: 에스컬레이션 해결
	EventMessageNew            EventType = "message.new"
	EventDirectMessage         EventType = "direct.message" // 신규: 직접 메시지
)

// Event represents a real-time event (legacy compatibility)
type Event struct {
	Type      EventType   `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	PortID    string      `json:"port_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// EventEmitter provides methods to emit events using the new SSE system
type EventEmitter struct {
	publisher *events.Publisher
	db        *db.DB
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter(hub *SSEHub, database *db.DB) *EventEmitter {
	return &EventEmitter{
		publisher: events.GetPublisher(),
		db:        database,
	}
}

// EmitSessionStart emits a session start event
func (e *EventEmitter) EmitSessionStart(sessionID, sessionType, title string) {
	e.publisher.PublishSessionStart(sessionID, sessionType, title, "")
}

// EmitSessionEnd emits a session end event
func (e *EventEmitter) EmitSessionEnd(sessionID, status string) {
	e.publisher.PublishSessionEnd(sessionID, "", status)
}

// EmitOrchestrationStart emits an orchestration start event
func (e *EventEmitter) EmitOrchestrationStart(orchID, title string, portCount int) {
	e.publisher.Publish(events.NewEvent("orchestration:start", map[string]interface{}{
		"id":         orchID,
		"title":      title,
		"port_count": portCount,
	}))
}

// EmitOrchestrationUpdate emits an orchestration update event
func (e *EventEmitter) EmitOrchestrationUpdate(orchID string, progress int, currentPort string) {
	e.publisher.Publish(events.NewEvent("orchestration:update", map[string]interface{}{
		"id":           orchID,
		"progress":     progress,
		"current_port": currentPort,
	}))
}

// EmitOrchestrationComplete emits an orchestration complete event
func (e *EventEmitter) EmitOrchestrationComplete(orchID, status string) {
	e.publisher.Publish(events.NewEvent("orchestration:complete", map[string]interface{}{
		"id":     orchID,
		"status": status,
	}))
}

// EmitWorkerSpawn emits a worker spawn event
func (e *EventEmitter) EmitWorkerSpawn(workerID, portID, workerType string) {
	e.publisher.Publish(events.NewEvent("worker:spawn", map[string]interface{}{
		"worker_id": workerID,
		"port_id":   portID,
		"type":      workerType,
	}))
}

// EmitWorkerComplete emits a worker complete event
func (e *EventEmitter) EmitWorkerComplete(workerID, portID string, success bool, result interface{}) {
	e.publisher.Publish(events.NewEvent("worker:complete", map[string]interface{}{
		"worker_id": workerID,
		"port_id":   portID,
		"success":   success,
		"result":    result,
	}))
}

// EmitPortUpdate emits a port update event
func (e *EventEmitter) EmitPortUpdate(portID, status string) {
	e.publisher.Publish(events.NewEvent("port:update", map[string]interface{}{
		"port_id": portID,
		"status":  status,
	}))
}

// EmitAttentionWarning emits an attention warning event
func (e *EventEmitter) EmitAttentionWarning(sessionID string, tokenPercent float64, tokensUsed int) {
	e.publisher.PublishAttentionWarning(sessionID, tokenPercent, tokensUsed, 0)
}

// EmitEscalation emits an escalation event
func (e *EventEmitter) EmitEscalation(escID, sessionID, escType, issue string, severity string) {
	e.publisher.PublishEscalationCreated(sessionID, "", escID, escType, issue, severity)
}

// EmitMessage emits a new message event
func (e *EventEmitter) EmitMessage(msgID, fromSession, toSession, msgType string) {
	e.publisher.PublishMessageReceived(toSession, map[string]interface{}{
		"id":       msgID,
		"from":     fromSession,
		"msg_type": msgType,
	})
}

// WorkerProgressEvent represents a worker progress event
type WorkerProgressEvent struct {
	SessionID   string  `json:"session_id"`
	PortID      string  `json:"port_id"`
	Progress    float64 `json:"progress"`     // 0.0 ~ 1.0
	CurrentTask string  `json:"current_task"`
	TokensUsed  int     `json:"tokens_used"`
}

// WorkerFeedbackEvent represents a worker feedback event
type WorkerFeedbackEvent struct {
	SessionID    string   `json:"session_id"`
	PortID       string   `json:"port_id"`
	Success      bool     `json:"success"`
	TestsPassed  int      `json:"tests_passed"`
	TestsFailed  int      `json:"tests_failed"`
	FailedTests  []string `json:"failed_tests,omitempty"`
	Suggestions  []string `json:"suggestions,omitempty"`
	RetryCount   int      `json:"retry_count"`
}

// DirectMessageEvent represents a direct message event
type DirectMessageEvent struct {
	ChannelID   string      `json:"channel_id"`
	FromSession string      `json:"from_session"`
	ToSession   string      `json:"to_session"`
	MessageType string      `json:"message_type"`
	Payload     interface{} `json:"payload,omitempty"`
}

// EscalationResolvedEvent represents an escalation resolved event
type EscalationResolvedEvent struct {
	EscalationID string `json:"escalation_id"`
	SessionID    string `json:"session_id"`
	PortID       string `json:"port_id,omitempty"`
	Resolution   string `json:"resolution"`
	ResolvedBy   string `json:"resolved_by,omitempty"`
}

// EmitWorkerProgress emits a worker progress event
func (e *EventEmitter) EmitWorkerProgress(sessionID, portID string, progress float64, currentTask string, tokensUsed int) {
	e.publisher.Publish(events.NewEvent("worker:progress", WorkerProgressEvent{
		SessionID:   sessionID,
		PortID:      portID,
		Progress:    progress,
		CurrentTask: currentTask,
		TokensUsed:  tokensUsed,
	}))
}

// EmitWorkerFeedback emits a worker feedback event
func (e *EventEmitter) EmitWorkerFeedback(event WorkerFeedbackEvent) {
	e.publisher.Publish(events.NewEvent("worker:feedback", event))
}

// EmitDirectMessage emits a direct message event
func (e *EventEmitter) EmitDirectMessage(event DirectMessageEvent) {
	e.publisher.Publish(events.NewEvent("direct:message", event))
}

// EmitEscalationResolved emits an escalation resolved event
func (e *EventEmitter) EmitEscalationResolved(event EscalationResolvedEvent) {
	e.publisher.Publish(events.NewEvent("escalation:resolved", event))
}

// StartPolling starts polling for changes (fallback for systems without real-time updates)
func (e *EventEmitter) StartPolling(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			e.pollForChanges()
		}
	}()
}

func (e *EventEmitter) pollForChanges() {
	// This is a simplified polling implementation
	// In production, you'd track last check time and query for changes
	log.Println("Polling for changes...")
}
