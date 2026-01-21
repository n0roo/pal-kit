package message

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageType defines the type of message
type MessageType string

const (
	TypeRequest    MessageType = "request"
	TypeResponse   MessageType = "response"
	TypeReport     MessageType = "report"
	TypeEscalation MessageType = "escalation"
)

// MessageSubtype defines specific message subtypes
type MessageSubtype string

const (
	SubtypeTaskAssign   MessageSubtype = "task_assign"
	SubtypeTaskComplete MessageSubtype = "task_complete"
	SubtypeTaskFailed   MessageSubtype = "task_failed"
	SubtypeTaskBlocked  MessageSubtype = "task_blocked"
	SubtypeImplReady    MessageSubtype = "impl_ready"
	SubtypeTestPass     MessageSubtype = "test_pass"
	SubtypeTestFail     MessageSubtype = "test_fail"
	SubtypeFixRequest   MessageSubtype = "fix_request"
	SubtypeProgress     MessageSubtype = "progress"
)

// MessageStatus defines the status of a message
type MessageStatus string

const (
	StatusPending   MessageStatus = "pending"
	StatusDelivered MessageStatus = "delivered"
	StatusProcessed MessageStatus = "processed"
	StatusExpired   MessageStatus = "expired"
)

// Message represents a message between sessions
type Message struct {
	ID               string         `json:"id"`
	ConversationID   string         `json:"conversation_id"`
	FromSession      string         `json:"from_session"`
	ToSession        string         `json:"to_session,omitempty"`
	Type             MessageType    `json:"type"`
	Subtype          MessageSubtype `json:"subtype,omitempty"`
	Payload          interface{}    `json:"payload"`
	AttentionScore   float64        `json:"attention_score,omitempty"`
	ContextSnapshot  string         `json:"context_snapshot,omitempty"`
	TokenCount       int            `json:"token_count,omitempty"`
	CumulativeTokens int            `json:"cumulative_tokens,omitempty"`
	Status           MessageStatus  `json:"status"`
	CreatedAt        time.Time      `json:"created_at"`
	ProcessedAt      *time.Time     `json:"processed_at,omitempty"`
	PortID           string         `json:"port_id,omitempty"`
	Priority         int            `json:"priority"`
}

// TaskAssignPayload is the payload for task assignment
type TaskAssignPayload struct {
	PortID      string   `json:"port_id"`
	PortSpec    string   `json:"port_spec"`
	Conventions []string `json:"conventions,omitempty"`
	Context     string   `json:"context,omitempty"`
}

// TaskReportPayload is the payload for task reports
type TaskReportPayload struct {
	Status  string                 `json:"status"`
	Output  map[string]interface{} `json:"output,omitempty"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// ImplReadyPayload is the payload when implementation is ready
type ImplReadyPayload struct {
	Files       []string `json:"files"`
	Changes     string   `json:"changes"`
	BuildStatus string   `json:"build_status"`
}

// TestResultPayload is the payload for test results
type TestResultPayload struct {
	Passed   int      `json:"passed"`
	Failed   int      `json:"failed"`
	Coverage float64  `json:"coverage,omitempty"`
	Feedback string   `json:"feedback,omitempty"`
	Failures []string `json:"failures,omitempty"`
}

// FixRequestPayload is the payload for fix requests
type FixRequestPayload struct {
	Failures    []string `json:"failures"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// Store handles message persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new message store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Send creates and stores a new message
func (s *Store) Send(msg *Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Status == "" {
		msg.Status = StatusPending
	}
	if msg.Priority == 0 {
		msg.Priority = 5
	}
	msg.CreatedAt = time.Now()

	payloadJSON, err := json.Marshal(msg.Payload)
	if err != nil {
		return fmt.Errorf("payload 직렬화 실패: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO messages (
			id, conversation_id, from_session, to_session, type, subtype,
			payload, attention_score, context_snapshot, token_count,
			cumulative_tokens, status, created_at, port_id, priority
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		msg.ID, msg.ConversationID, msg.FromSession, msg.ToSession,
		msg.Type, msg.Subtype, string(payloadJSON), msg.AttentionScore,
		msg.ContextSnapshot, msg.TokenCount, msg.CumulativeTokens,
		msg.Status, msg.CreatedAt, msg.PortID, msg.Priority,
	)
	if err != nil {
		return fmt.Errorf("메시지 저장 실패: %w", err)
	}

	return nil
}

// Receive gets pending messages for a session
func (s *Store) Receive(sessionID string, limit int) ([]*Message, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT id, conversation_id, from_session, to_session, type, subtype,
		       payload, attention_score, context_snapshot, token_count,
		       cumulative_tokens, status, created_at, processed_at, port_id, priority
		FROM messages
		WHERE (to_session = ? OR to_session IS NULL)
		  AND status = ?
		ORDER BY priority ASC, created_at ASC
		LIMIT ?
	`, sessionID, StatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("메시지 조회 실패: %w", err)
	}
	defer rows.Close()

	return s.scanMessages(rows)
}

// GetByConversation gets all messages in a conversation
func (s *Store) GetByConversation(conversationID string) ([]*Message, error) {
	rows, err := s.db.Query(`
		SELECT id, conversation_id, from_session, to_session, type, subtype,
		       payload, attention_score, context_snapshot, token_count,
		       cumulative_tokens, status, created_at, processed_at, port_id, priority
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("대화 메시지 조회 실패: %w", err)
	}
	defer rows.Close()

	return s.scanMessages(rows)
}

// MarkDelivered marks a message as delivered
func (s *Store) MarkDelivered(messageID string) error {
	_, err := s.db.Exec(`
		UPDATE messages SET status = ? WHERE id = ?
	`, StatusDelivered, messageID)
	return err
}

// MarkProcessed marks a message as processed
func (s *Store) MarkProcessed(messageID string) error {
	_, err := s.db.Exec(`
		UPDATE messages SET status = ?, processed_at = ? WHERE id = ?
	`, StatusProcessed, time.Now(), messageID)
	return err
}

// GetPendingCount returns the number of pending messages for a session
func (s *Store) GetPendingCount(sessionID string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM messages
		WHERE (to_session = ? OR to_session IS NULL)
		  AND status = ?
	`, sessionID, StatusPending).Scan(&count)
	return count, err
}

// GetConversationTokens returns the cumulative token count for a conversation
func (s *Store) GetConversationTokens(conversationID string) (int, error) {
	var total int
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(token_count), 0) FROM messages
		WHERE conversation_id = ?
	`, conversationID).Scan(&total)
	return total, err
}

// scanMessages scans rows into Message slice
func (s *Store) scanMessages(rows *sql.Rows) ([]*Message, error) {
	var messages []*Message
	for rows.Next() {
		var msg Message
		var payloadJSON string
		var toSession, subtype, contextSnapshot, portID sql.NullString
		var attentionScore sql.NullFloat64
		var tokenCount, cumulativeTokens sql.NullInt64
		var processedAt sql.NullTime

		err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.FromSession, &toSession,
			&msg.Type, &subtype, &payloadJSON, &attentionScore,
			&contextSnapshot, &tokenCount, &cumulativeTokens,
			&msg.Status, &msg.CreatedAt, &processedAt, &portID, &msg.Priority,
		)
		if err != nil {
			continue
		}

		if toSession.Valid {
			msg.ToSession = toSession.String
		}
		if subtype.Valid {
			msg.Subtype = MessageSubtype(subtype.String)
		}
		if contextSnapshot.Valid {
			msg.ContextSnapshot = contextSnapshot.String
		}
		if portID.Valid {
			msg.PortID = portID.String
		}
		if attentionScore.Valid {
			msg.AttentionScore = attentionScore.Float64
		}
		if tokenCount.Valid {
			msg.TokenCount = int(tokenCount.Int64)
		}
		if cumulativeTokens.Valid {
			msg.CumulativeTokens = int(cumulativeTokens.Int64)
		}
		if processedAt.Valid {
			msg.ProcessedAt = &processedAt.Time
		}

		// Unmarshal payload
		if payloadJSON != "" {
			json.Unmarshal([]byte(payloadJSON), &msg.Payload)
		}

		messages = append(messages, &msg)
	}

	return messages, nil
}

// SendTaskAssign sends a task assignment message
func (s *Store) SendTaskAssign(fromSession, toSession, portID string, payload TaskAssignPayload) error {
	msg := &Message{
		ConversationID: portID,
		FromSession:    fromSession,
		ToSession:      toSession,
		Type:           TypeRequest,
		Subtype:        SubtypeTaskAssign,
		Payload:        payload,
		PortID:         portID,
		Priority:       1,
	}
	return s.Send(msg)
}

// SendTaskReport sends a task report message
func (s *Store) SendTaskReport(fromSession, toSession, portID string, subtype MessageSubtype, payload TaskReportPayload) error {
	msg := &Message{
		ConversationID: portID,
		FromSession:    fromSession,
		ToSession:      toSession,
		Type:           TypeReport,
		Subtype:        subtype,
		Payload:        payload,
		PortID:         portID,
		Priority:       2,
	}
	return s.Send(msg)
}

// SendImplReady sends an implementation ready message
func (s *Store) SendImplReady(fromSession, toSession, portID string, payload ImplReadyPayload) error {
	msg := &Message{
		ConversationID: portID,
		FromSession:    fromSession,
		ToSession:      toSession,
		Type:           TypeResponse,
		Subtype:        SubtypeImplReady,
		Payload:        payload,
		PortID:         portID,
		Priority:       3,
	}
	return s.Send(msg)
}

// SendTestResult sends a test result message
func (s *Store) SendTestResult(fromSession, toSession, portID string, passed bool, payload TestResultPayload) error {
	subtype := SubtypeTestPass
	if !passed {
		subtype = SubtypeTestFail
	}
	msg := &Message{
		ConversationID: portID,
		FromSession:    fromSession,
		ToSession:      toSession,
		Type:           TypeResponse,
		Subtype:        subtype,
		Payload:        payload,
		PortID:         portID,
		Priority:       3,
	}
	return s.Send(msg)
}

// SendFixRequest sends a fix request message
func (s *Store) SendFixRequest(fromSession, toSession, portID string, payload FixRequestPayload) error {
	msg := &Message{
		ConversationID: portID,
		FromSession:    fromSession,
		ToSession:      toSession,
		Type:           TypeRequest,
		Subtype:        SubtypeFixRequest,
		Payload:        payload,
		PortID:         portID,
		Priority:       2,
	}
	return s.Send(msg)
}
