package attention

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/server/events"
)

// SessionAttention represents the attention state of a session
type SessionAttention struct {
	SessionID           string    `json:"session_id"`
	PortID              string    `json:"port_id,omitempty"`
	CurrentContextHash  string    `json:"current_context_hash,omitempty"`
	LoadedTokens        int       `json:"loaded_tokens"`
	AvailableTokens     int       `json:"available_tokens"`
	FocusScore          float64   `json:"focus_score"`
	DriftCount          int       `json:"drift_count"`
	LastCompactionAt    *time.Time `json:"last_compaction_at,omitempty"`
	LoadedFiles         []string  `json:"loaded_files,omitempty"`
	LoadedConventions   []string  `json:"loaded_conventions,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CompactEvent represents a compaction event
type CompactEvent struct {
	ID                string    `json:"id"`
	SessionID         string    `json:"session_id"`
	TriggerReason     string    `json:"trigger_reason"` // token_limit, user_request, auto
	BeforeTokens      int       `json:"before_tokens"`
	AfterTokens       int       `json:"after_tokens"`
	PreservedContext  []string  `json:"preserved_context,omitempty"`
	DiscardedContext  []string  `json:"discarded_context,omitempty"`
	CheckpointBefore  string    `json:"checkpoint_before,omitempty"`
	RecoveryHint      string    `json:"recovery_hint,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// AttentionStatus represents the overall attention status
type AttentionStatus string

const (
	StatusFocused  AttentionStatus = "focused"
	StatusDrifting AttentionStatus = "drifting"
	StatusWarning  AttentionStatus = "warning"
	StatusCritical AttentionStatus = "critical"
)

// DefaultTokenBudget is the default token budget
const DefaultTokenBudget = 15000

// Store handles attention tracking persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new attention store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Initialize creates or updates attention tracking for a session
func (s *Store) Initialize(sessionID, portID string, tokenBudget int) error {
	if tokenBudget <= 0 {
		tokenBudget = DefaultTokenBudget
	}

	_, err := s.db.Exec(`
		INSERT INTO session_attention (
			session_id, port_id, loaded_tokens, available_tokens, focus_score, drift_count, updated_at
		) VALUES (?, ?, 0, ?, 1.0, 0, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			port_id = excluded.port_id,
			available_tokens = excluded.available_tokens,
			updated_at = excluded.updated_at
	`, sessionID, portID, tokenBudget, time.Now())
	return err
}

// Get retrieves the attention state for a session
func (s *Store) Get(sessionID string) (*SessionAttention, error) {
	var att SessionAttention
	var portID, contextHash, filesJSON, convsJSON sql.NullString
	var lastCompact sql.NullTime

	err := s.db.QueryRow(`
		SELECT session_id, port_id, current_context_hash, loaded_tokens, available_tokens,
		       focus_score, drift_count, last_compaction_at, loaded_files, loaded_conventions, updated_at
		FROM session_attention WHERE session_id = ?
	`, sessionID).Scan(&att.SessionID, &portID, &contextHash, &att.LoadedTokens, &att.AvailableTokens,
		&att.FocusScore, &att.DriftCount, &lastCompact, &filesJSON, &convsJSON, &att.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if portID.Valid {
		att.PortID = portID.String
	}
	if contextHash.Valid {
		att.CurrentContextHash = contextHash.String
	}
	if lastCompact.Valid {
		att.LastCompactionAt = &lastCompact.Time
	}
	if filesJSON.Valid {
		json.Unmarshal([]byte(filesJSON.String), &att.LoadedFiles)
	}
	if convsJSON.Valid {
		json.Unmarshal([]byte(convsJSON.String), &att.LoadedConventions)
	}

	return &att, nil
}

// UpdateTokens updates the token usage for a session
func (s *Store) UpdateTokens(sessionID string, loadedTokens int) error {
	_, err := s.db.Exec(`
		UPDATE session_attention SET
			loaded_tokens = ?,
			updated_at = ?
		WHERE session_id = ?
	`, loadedTokens, time.Now(), sessionID)
	if err != nil {
		return err
	}

	// 토큰 사용량 기반 SSE 이벤트 발행 (LM-sse-stream)
	att, err := s.Get(sessionID)
	if err == nil && att != nil {
		usagePercent := att.GetTokenUsagePercent()
		publisher := events.GetPublisher()

		if usagePercent >= 90 {
			// Critical: 90% 이상
			publisher.PublishAttentionCritical(sessionID, usagePercent, loadedTokens, att.AvailableTokens)
		} else if usagePercent >= 80 {
			// Warning: 80% 이상
			publisher.PublishAttentionWarning(sessionID, usagePercent, loadedTokens, att.AvailableTokens)
		}
	}

	return nil
}

// UpdateFocusScore updates the focus score for a session
func (s *Store) UpdateFocusScore(sessionID string, score float64) error {
	_, err := s.db.Exec(`
		UPDATE session_attention SET
			focus_score = ?,
			updated_at = ?
		WHERE session_id = ?
	`, score, time.Now(), sessionID)
	return err
}

// IncrementDrift increments the drift count for a session
func (s *Store) IncrementDrift(sessionID string) error {
	_, err := s.db.Exec(`
		UPDATE session_attention SET
			drift_count = drift_count + 1,
			focus_score = MAX(0, focus_score - 0.1),
			updated_at = ?
		WHERE session_id = ?
	`, time.Now(), sessionID)
	return err
}

// UpdateLoadedContext updates the loaded context for a session
func (s *Store) UpdateLoadedContext(sessionID string, files, conventions []string, contextHash string) error {
	filesJSON, _ := json.Marshal(files)
	convsJSON, _ := json.Marshal(conventions)

	_, err := s.db.Exec(`
		UPDATE session_attention SET
			loaded_files = ?,
			loaded_conventions = ?,
			current_context_hash = ?,
			updated_at = ?
		WHERE session_id = ?
	`, string(filesJSON), string(convsJSON), contextHash, time.Now(), sessionID)
	return err
}

// RecordCompact records a compaction event
func (s *Store) RecordCompact(event *CompactEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	event.CreatedAt = time.Now()

	preservedJSON, _ := json.Marshal(event.PreservedContext)
	discardedJSON, _ := json.Marshal(event.DiscardedContext)

	_, err := s.db.Exec(`
		INSERT INTO compact_events (
			id, session_id, trigger_reason, before_tokens, after_tokens,
			preserved_context, discarded_context, checkpoint_before, recovery_hint, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.SessionID, event.TriggerReason, event.BeforeTokens, event.AfterTokens,
		string(preservedJSON), string(discardedJSON), event.CheckpointBefore, event.RecoveryHint, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("Compact 이벤트 기록 실패: %w", err)
	}

	// Update session attention
	now := time.Now()
	_, _ = s.db.Exec(`
		UPDATE session_attention SET
			loaded_tokens = ?,
			last_compaction_at = ?,
			updated_at = ?
		WHERE session_id = ?
	`, event.AfterTokens, now, now, event.SessionID)

	// SSE 이벤트 발행 (LM-sse-stream)
	publisher := events.GetPublisher()
	publisher.PublishCompactTriggered(event.SessionID, event.TriggerReason, event.CheckpointBefore, event.RecoveryHint)

	return nil
}

// GetCompactHistory gets the compact history for a session
func (s *Store) GetCompactHistory(sessionID string, limit int) ([]*CompactEvent, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT id, session_id, trigger_reason, before_tokens, after_tokens,
		       preserved_context, discarded_context, checkpoint_before, recovery_hint, created_at
		FROM compact_events
		WHERE session_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*CompactEvent
	for rows.Next() {
		var event CompactEvent
		var preservedJSON, discardedJSON, checkpoint, hint sql.NullString

		err := rows.Scan(&event.ID, &event.SessionID, &event.TriggerReason,
			&event.BeforeTokens, &event.AfterTokens, &preservedJSON, &discardedJSON,
			&checkpoint, &hint, &event.CreatedAt)
		if err != nil {
			continue
		}

		if preservedJSON.Valid {
			json.Unmarshal([]byte(preservedJSON.String), &event.PreservedContext)
		}
		if discardedJSON.Valid {
			json.Unmarshal([]byte(discardedJSON.String), &event.DiscardedContext)
		}
		if checkpoint.Valid {
			event.CheckpointBefore = checkpoint.String
		}
		if hint.Valid {
			event.RecoveryHint = hint.String
		}

		events = append(events, &event)
	}

	return events, nil
}

// GetStatus calculates the current attention status
func (s *Store) GetStatus(sessionID string) (AttentionStatus, error) {
	att, err := s.Get(sessionID)
	if err != nil {
		return "", err
	}

	return CalculateStatus(att), nil
}

// CalculateStatus calculates the attention status from attention state
func CalculateStatus(att *SessionAttention) AttentionStatus {
	usagePercent := float64(att.LoadedTokens) / float64(att.AvailableTokens)

	// Critical: >95% token usage or very low focus
	if usagePercent >= 0.95 || att.FocusScore < 0.3 {
		return StatusCritical
	}

	// Warning: >80% token usage or low focus
	if usagePercent >= 0.80 || att.FocusScore < 0.5 {
		return StatusWarning
	}

	// Drifting: focus score dropping or high drift count
	if att.FocusScore < 0.7 || att.DriftCount > 3 {
		return StatusDrifting
	}

	return StatusFocused
}

// GetTokenUsagePercent returns the token usage percentage
func (att *SessionAttention) GetTokenUsagePercent() float64 {
	if att.AvailableTokens == 0 {
		return 0
	}
	return float64(att.LoadedTokens) / float64(att.AvailableTokens) * 100
}

// ShouldCheckpoint determines if a checkpoint should be created
func (att *SessionAttention) ShouldCheckpoint() bool {
	return att.GetTokenUsagePercent() >= 80 || att.DriftCount > 2
}

// ShouldWarn determines if a warning should be shown
func (att *SessionAttention) ShouldWarn() bool {
	return att.GetTokenUsagePercent() >= 80 || att.FocusScore < 0.6
}

// AttentionReport provides a summary of attention state
type AttentionReport struct {
	SessionID        string          `json:"session_id"`
	Status           AttentionStatus `json:"status"`
	TokenUsage       string          `json:"token_usage"`
	TokenPercent     float64         `json:"token_percent"`
	FocusScore       float64         `json:"focus_score"`
	DriftCount       int             `json:"drift_count"`
	CompactCount     int             `json:"compact_count"`
	Recommendations  []string        `json:"recommendations,omitempty"`
}

// GenerateReport generates an attention report for a session
func (s *Store) GenerateReport(sessionID string) (*AttentionReport, error) {
	att, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}

	compacts, err := s.GetCompactHistory(sessionID, 100)
	if err != nil {
		return nil, err
	}

	report := &AttentionReport{
		SessionID:    sessionID,
		Status:       CalculateStatus(att),
		TokenUsage:   fmt.Sprintf("%d / %d", att.LoadedTokens, att.AvailableTokens),
		TokenPercent: att.GetTokenUsagePercent(),
		FocusScore:   att.FocusScore,
		DriftCount:   att.DriftCount,
		CompactCount: len(compacts),
	}

	// Add recommendations
	if report.TokenPercent >= 80 {
		report.Recommendations = append(report.Recommendations,
			"토큰 사용량이 높습니다. 체크포인트 생성을 권장합니다.")
	}
	if report.FocusScore < 0.6 {
		report.Recommendations = append(report.Recommendations,
			"Focus Score가 낮습니다. 작업 범위를 좁히거나 세션을 분리하세요.")
	}
	if report.DriftCount > 3 {
		report.Recommendations = append(report.Recommendations,
			"컨텍스트 드리프트가 자주 발생합니다. 명세를 더 명확히 정의하세요.")
	}
	if len(compacts) > 2 {
		report.Recommendations = append(report.Recommendations,
			"Compact가 빈번합니다. 작업을 더 작은 단위로 분리하는 것을 검토하세요.")
	}

	return report, nil
}
