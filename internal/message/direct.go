package message

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DirectMessageType defines types of direct messages
type DirectMessageType string

const (
	DirectTypeResult   DirectMessageType = "result"   // 구현 결과
	DirectTypeFeedback DirectMessageType = "feedback" // 테스트 피드백
	DirectTypeQuery    DirectMessageType = "query"    // 질의
	DirectTypeAck      DirectMessageType = "ack"      // 확인
)

// DirectChannel represents a direct communication channel between two workers
type DirectChannel struct {
	ID              string     `json:"id"`
	SessionA        string     `json:"session_a"`        // Impl Worker
	SessionB        string     `json:"session_b"`        // Test Worker
	PortID          string     `json:"port_id,omitempty"`
	OrchestrationID string     `json:"orchestration_id,omitempty"`
	Status          string     `json:"status"` // active, closed
	CreatedAt       time.Time  `json:"created_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
}

// DirectMessage represents a message in a direct channel
type DirectMessage struct {
	ID          string            `json:"id"`
	ChannelID   string            `json:"channel_id"`
	FromSession string            `json:"from_session"`
	ToSession   string            `json:"to_session"`
	Type        DirectMessageType `json:"type"`
	Payload     interface{}       `json:"payload"`
	DeliveredAt *time.Time        `json:"delivered_at,omitempty"`
	ProcessedAt *time.Time        `json:"processed_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// TestFeedbackPayload represents feedback from Test Worker to Impl Worker
type TestFeedbackPayload struct {
	Success      bool         `json:"success"`
	FailedTests  []FailedTest `json:"failed_tests,omitempty"`
	Suggestions  []string     `json:"suggestions,omitempty"`
	RetryCount   int          `json:"retry_count"`
	TotalTests   int          `json:"total_tests"`
	PassedTests  int          `json:"passed_tests"`
	Coverage     float64      `json:"coverage,omitempty"`
}

// FailedTest represents a single failed test
type FailedTest struct {
	Name         string `json:"name"`
	Expected     string `json:"expected,omitempty"`
	Actual       string `json:"actual,omitempty"`
	StackTrace   string `json:"stack_trace,omitempty"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
	File         string `json:"file,omitempty"`
	Line         int    `json:"line,omitempty"`
}

// ImplResultPayload represents result from Impl Worker to Test Worker
type ImplResultPayload struct {
	Status       string   `json:"status"` // ready, partial, failed
	ChangedFiles []string `json:"changed_files"`
	Summary      string   `json:"summary"`
	BuildPassed  bool     `json:"build_passed"`
	Notes        string   `json:"notes,omitempty"`
}

// DirectStore handles direct channel and message persistence
type DirectStore struct {
	db *sql.DB
}

// NewDirectStore creates a new direct store
func NewDirectStore(db *sql.DB) *DirectStore {
	return &DirectStore{db: db}
}

// CreateChannel creates a new direct channel between two workers
func (s *DirectStore) CreateChannel(sessionA, sessionB, portID, orchestrationID string) (*DirectChannel, error) {
	channel := &DirectChannel{
		ID:              uuid.New().String(),
		SessionA:        sessionA,
		SessionB:        sessionB,
		PortID:          portID,
		OrchestrationID: orchestrationID,
		Status:          "active",
		CreatedAt:       time.Now(),
	}

	_, err := s.db.Exec(`
		INSERT INTO direct_channels (id, session_a, session_b, port_id, orchestration_id, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, channel.ID, channel.SessionA, channel.SessionB, channel.PortID, channel.OrchestrationID, channel.Status, channel.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("채널 생성 실패: %w", err)
	}

	return channel, nil
}

// GetChannel retrieves a channel by ID
func (s *DirectStore) GetChannel(channelID string) (*DirectChannel, error) {
	var channel DirectChannel
	var portID, orchestrationID sql.NullString
	var closedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, session_a, session_b, port_id, orchestration_id, status, created_at, closed_at
		FROM direct_channels WHERE id = ?
	`, channelID).Scan(
		&channel.ID, &channel.SessionA, &channel.SessionB,
		&portID, &orchestrationID, &channel.Status,
		&channel.CreatedAt, &closedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("채널 조회 실패: %w", err)
	}

	if portID.Valid {
		channel.PortID = portID.String
	}
	if orchestrationID.Valid {
		channel.OrchestrationID = orchestrationID.String
	}
	if closedAt.Valid {
		channel.ClosedAt = &closedAt.Time
	}

	return &channel, nil
}

// GetChannelByPort retrieves an active channel for a port
func (s *DirectStore) GetChannelByPort(portID string) (*DirectChannel, error) {
	var channel DirectChannel
	var pID, orchestrationID sql.NullString
	var closedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, session_a, session_b, port_id, orchestration_id, status, created_at, closed_at
		FROM direct_channels WHERE port_id = ? AND status = 'active'
		ORDER BY created_at DESC LIMIT 1
	`, portID).Scan(
		&channel.ID, &channel.SessionA, &channel.SessionB,
		&pID, &orchestrationID, &channel.Status,
		&channel.CreatedAt, &closedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("채널 조회 실패: %w", err)
	}

	if pID.Valid {
		channel.PortID = pID.String
	}
	if orchestrationID.Valid {
		channel.OrchestrationID = orchestrationID.String
	}
	if closedAt.Valid {
		channel.ClosedAt = &closedAt.Time
	}

	return &channel, nil
}

// GetChannelForSession retrieves active channel where session is a participant
func (s *DirectStore) GetChannelForSession(sessionID string) (*DirectChannel, error) {
	var channel DirectChannel
	var portID, orchestrationID sql.NullString
	var closedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, session_a, session_b, port_id, orchestration_id, status, created_at, closed_at
		FROM direct_channels
		WHERE (session_a = ? OR session_b = ?) AND status = 'active'
		ORDER BY created_at DESC LIMIT 1
	`, sessionID, sessionID).Scan(
		&channel.ID, &channel.SessionA, &channel.SessionB,
		&portID, &orchestrationID, &channel.Status,
		&channel.CreatedAt, &closedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("채널 조회 실패: %w", err)
	}

	if portID.Valid {
		channel.PortID = portID.String
	}
	if orchestrationID.Valid {
		channel.OrchestrationID = orchestrationID.String
	}
	if closedAt.Valid {
		channel.ClosedAt = &closedAt.Time
	}

	return &channel, nil
}

// CloseChannel closes a direct channel
func (s *DirectStore) CloseChannel(channelID string) error {
	_, err := s.db.Exec(`
		UPDATE direct_channels SET status = 'closed', closed_at = ? WHERE id = ?
	`, time.Now(), channelID)
	if err != nil {
		return fmt.Errorf("채널 닫기 실패: %w", err)
	}
	return nil
}

// SendDirect sends a message through a direct channel
func (s *DirectStore) SendDirect(channelID, fromSession, toSession string, msgType DirectMessageType, payload interface{}) (*DirectMessage, error) {
	msg := &DirectMessage{
		ID:          uuid.New().String(),
		ChannelID:   channelID,
		FromSession: fromSession,
		ToSession:   toSession,
		Type:        msgType,
		Payload:     payload,
		CreatedAt:   time.Now(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("payload 직렬화 실패: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO direct_messages (id, channel_id, from_session, to_session, message_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.ChannelID, msg.FromSession, msg.ToSession, msg.Type, string(payloadJSON), msg.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("메시지 전송 실패: %w", err)
	}

	return msg, nil
}

// ReceiveDirect retrieves undelivered messages for a session
func (s *DirectStore) ReceiveDirect(channelID, recipientSession string, limit int) ([]*DirectMessage, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT id, channel_id, from_session, to_session, message_type, payload, delivered_at, processed_at, created_at
		FROM direct_messages
		WHERE channel_id = ? AND to_session = ? AND delivered_at IS NULL
		ORDER BY created_at ASC
		LIMIT ?
	`, channelID, recipientSession, limit)
	if err != nil {
		return nil, fmt.Errorf("메시지 조회 실패: %w", err)
	}
	defer rows.Close()

	return s.scanDirectMessages(rows)
}

// ReceiveAllPending retrieves all pending messages for a session across all channels
func (s *DirectStore) ReceiveAllPending(sessionID string, limit int) ([]*DirectMessage, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT dm.id, dm.channel_id, dm.from_session, dm.to_session, dm.message_type, dm.payload, dm.delivered_at, dm.processed_at, dm.created_at
		FROM direct_messages dm
		JOIN direct_channels dc ON dm.channel_id = dc.id
		WHERE dm.to_session = ? AND dm.delivered_at IS NULL AND dc.status = 'active'
		ORDER BY dm.created_at ASC
		LIMIT ?
	`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("메시지 조회 실패: %w", err)
	}
	defer rows.Close()

	return s.scanDirectMessages(rows)
}

// MarkDirectDelivered marks a direct message as delivered
func (s *DirectStore) MarkDirectDelivered(messageID string) error {
	_, err := s.db.Exec(`
		UPDATE direct_messages SET delivered_at = ? WHERE id = ?
	`, time.Now(), messageID)
	return err
}

// MarkDirectProcessed marks a direct message as processed
func (s *DirectStore) MarkDirectProcessed(messageID string) error {
	_, err := s.db.Exec(`
		UPDATE direct_messages SET processed_at = ? WHERE id = ?
	`, time.Now(), messageID)
	return err
}

// GetChannelHistory retrieves all messages in a channel
func (s *DirectStore) GetChannelHistory(channelID string, limit int) ([]*DirectMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT id, channel_id, from_session, to_session, message_type, payload, delivered_at, processed_at, created_at
		FROM direct_messages
		WHERE channel_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("메시지 히스토리 조회 실패: %w", err)
	}
	defer rows.Close()

	return s.scanDirectMessages(rows)
}

// scanDirectMessages scans rows into DirectMessage slice
func (s *DirectStore) scanDirectMessages(rows *sql.Rows) ([]*DirectMessage, error) {
	var messages []*DirectMessage
	for rows.Next() {
		var msg DirectMessage
		var payloadJSON string
		var deliveredAt, processedAt sql.NullTime

		err := rows.Scan(
			&msg.ID, &msg.ChannelID, &msg.FromSession, &msg.ToSession,
			&msg.Type, &payloadJSON, &deliveredAt, &processedAt, &msg.CreatedAt,
		)
		if err != nil {
			continue
		}

		if deliveredAt.Valid {
			msg.DeliveredAt = &deliveredAt.Time
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

// SendTestFeedback sends test feedback from Test Worker to Impl Worker
func (s *DirectStore) SendTestFeedback(channelID, testSession, implSession string, feedback TestFeedbackPayload) (*DirectMessage, error) {
	return s.SendDirect(channelID, testSession, implSession, DirectTypeFeedback, feedback)
}

// SendImplResult sends implementation result from Impl Worker to Test Worker
func (s *DirectStore) SendImplResult(channelID, implSession, testSession string, result ImplResultPayload) (*DirectMessage, error) {
	return s.SendDirect(channelID, implSession, testSession, DirectTypeResult, result)
}

// GetPendingFeedbackCount returns the number of pending feedback messages for a session
func (s *DirectStore) GetPendingFeedbackCount(sessionID string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM direct_messages dm
		JOIN direct_channels dc ON dm.channel_id = dc.id
		WHERE dm.to_session = ? AND dm.message_type = 'feedback' AND dm.delivered_at IS NULL AND dc.status = 'active'
	`, sessionID).Scan(&count)
	return count, err
}

// ListActiveChannels lists all active channels for an orchestration
func (s *DirectStore) ListActiveChannels(orchestrationID string) ([]*DirectChannel, error) {
	rows, err := s.db.Query(`
		SELECT id, session_a, session_b, port_id, orchestration_id, status, created_at, closed_at
		FROM direct_channels
		WHERE orchestration_id = ? AND status = 'active'
		ORDER BY created_at ASC
	`, orchestrationID)
	if err != nil {
		return nil, fmt.Errorf("채널 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var channels []*DirectChannel
	for rows.Next() {
		var channel DirectChannel
		var portID, orchID sql.NullString
		var closedAt sql.NullTime

		err := rows.Scan(
			&channel.ID, &channel.SessionA, &channel.SessionB,
			&portID, &orchID, &channel.Status, &channel.CreatedAt, &closedAt,
		)
		if err != nil {
			continue
		}

		if portID.Valid {
			channel.PortID = portID.String
		}
		if orchID.Valid {
			channel.OrchestrationID = orchID.String
		}
		if closedAt.Valid {
			channel.ClosedAt = &closedAt.Time
		}

		channels = append(channels, &channel)
	}

	return channels, nil
}
