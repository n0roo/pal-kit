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
	EventPortUpdate            EventType = "port.update"
	EventAttentionWarning      EventType = "attention.warning"
	EventEscalation            EventType = "escalation.new"
	EventMessageNew            EventType = "message.new"
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
