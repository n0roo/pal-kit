package orchestrator

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/message"
)

// FeedbackLoopStatus defines the status of a feedback loop
type FeedbackLoopStatus string

const (
	FeedbackStatusRunning   FeedbackLoopStatus = "running"
	FeedbackStatusSuccess   FeedbackLoopStatus = "success"
	FeedbackStatusFailed    FeedbackLoopStatus = "failed"
	FeedbackStatusEscalated FeedbackLoopStatus = "escalated"
)

// FeedbackLoop represents a feedback loop between Impl and Test workers
type FeedbackLoop struct {
	ID             string             `json:"id"`
	ChannelID      string             `json:"channel_id"`
	ImplSessionID  string             `json:"impl_session_id"`
	TestSessionID  string             `json:"test_session_id"`
	PortID         string             `json:"port_id,omitempty"`
	MaxRetries     int                `json:"max_retries"`
	CurrentRetry   int                `json:"current_retry"`
	Status         FeedbackLoopStatus `json:"status"`
	LastFeedbackAt *time.Time         `json:"last_feedback_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	CompletedAt    *time.Time         `json:"completed_at,omitempty"`
}

// FeedbackResult represents the result of a feedback iteration
type FeedbackResult struct {
	Iteration    int                          `json:"iteration"`
	Success      bool                         `json:"success"`
	TestsPassed  int                          `json:"tests_passed"`
	TestsFailed  int                          `json:"tests_failed"`
	Coverage     float64                      `json:"coverage,omitempty"`
	FailedTests  []message.FailedTest         `json:"failed_tests,omitempty"`
	Suggestions  []string                     `json:"suggestions,omitempty"`
	Duration     time.Duration                `json:"duration"`
}

// FeedbackService handles feedback loop operations
type FeedbackService struct {
	db          *db.DB
	directStore *message.DirectStore
}

// NewFeedbackService creates a new feedback service
func NewFeedbackService(database *db.DB) *FeedbackService {
	return &FeedbackService{
		db:          database,
		directStore: message.NewDirectStore(database.DB),
	}
}

// CreateFeedbackLoop creates a new feedback loop
func (s *FeedbackService) CreateFeedbackLoop(channelID, implSession, testSession, portID string, maxRetries int) (*FeedbackLoop, error) {
	if maxRetries <= 0 {
		maxRetries = 3
	}

	loop := &FeedbackLoop{
		ID:            uuid.New().String(),
		ChannelID:     channelID,
		ImplSessionID: implSession,
		TestSessionID: testSession,
		PortID:        portID,
		MaxRetries:    maxRetries,
		CurrentRetry:  0,
		Status:        FeedbackStatusRunning,
		CreatedAt:     time.Now(),
	}

	_, err := s.db.Exec(`
		INSERT INTO feedback_loops (id, channel_id, impl_session, test_session, port_id, max_retries, current_retry, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, loop.ID, loop.ChannelID, loop.ImplSessionID, loop.TestSessionID, loop.PortID, loop.MaxRetries, loop.CurrentRetry, loop.Status, loop.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("피드백 루프 생성 실패: %w", err)
	}

	return loop, nil
}

// GetFeedbackLoop retrieves a feedback loop by ID
func (s *FeedbackService) GetFeedbackLoop(loopID string) (*FeedbackLoop, error) {
	var loop FeedbackLoop
	var portID sql.NullString
	var lastFeedbackAt, completedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, channel_id, impl_session, test_session, port_id, max_retries, current_retry, status, last_feedback_at, created_at, completed_at
		FROM feedback_loops WHERE id = ?
	`, loopID).Scan(
		&loop.ID, &loop.ChannelID, &loop.ImplSessionID, &loop.TestSessionID,
		&portID, &loop.MaxRetries, &loop.CurrentRetry, &loop.Status,
		&lastFeedbackAt, &loop.CreatedAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("피드백 루프 조회 실패: %w", err)
	}

	if portID.Valid {
		loop.PortID = portID.String
	}
	if lastFeedbackAt.Valid {
		loop.LastFeedbackAt = &lastFeedbackAt.Time
	}
	if completedAt.Valid {
		loop.CompletedAt = &completedAt.Time
	}

	return &loop, nil
}

// GetFeedbackLoopByChannel retrieves a feedback loop by channel ID
func (s *FeedbackService) GetFeedbackLoopByChannel(channelID string) (*FeedbackLoop, error) {
	var loop FeedbackLoop
	var portID sql.NullString
	var lastFeedbackAt, completedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, channel_id, impl_session, test_session, port_id, max_retries, current_retry, status, last_feedback_at, created_at, completed_at
		FROM feedback_loops WHERE channel_id = ?
		ORDER BY created_at DESC LIMIT 1
	`, channelID).Scan(
		&loop.ID, &loop.ChannelID, &loop.ImplSessionID, &loop.TestSessionID,
		&portID, &loop.MaxRetries, &loop.CurrentRetry, &loop.Status,
		&lastFeedbackAt, &loop.CreatedAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("피드백 루프 조회 실패: %w", err)
	}

	if portID.Valid {
		loop.PortID = portID.String
	}
	if lastFeedbackAt.Valid {
		loop.LastFeedbackAt = &lastFeedbackAt.Time
	}
	if completedAt.Valid {
		loop.CompletedAt = &completedAt.Time
	}

	return &loop, nil
}

// GetActiveLoopForPort retrieves the active feedback loop for a port
func (s *FeedbackService) GetActiveLoopForPort(portID string) (*FeedbackLoop, error) {
	var loop FeedbackLoop
	var pID sql.NullString
	var lastFeedbackAt, completedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, channel_id, impl_session, test_session, port_id, max_retries, current_retry, status, last_feedback_at, created_at, completed_at
		FROM feedback_loops WHERE port_id = ? AND status = 'running'
		ORDER BY created_at DESC LIMIT 1
	`, portID).Scan(
		&loop.ID, &loop.ChannelID, &loop.ImplSessionID, &loop.TestSessionID,
		&pID, &loop.MaxRetries, &loop.CurrentRetry, &loop.Status,
		&lastFeedbackAt, &loop.CreatedAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("피드백 루프 조회 실패: %w", err)
	}

	if pID.Valid {
		loop.PortID = pID.String
	}
	if lastFeedbackAt.Valid {
		loop.LastFeedbackAt = &lastFeedbackAt.Time
	}
	if completedAt.Valid {
		loop.CompletedAt = &completedAt.Time
	}

	return &loop, nil
}

// IncrementRetry increments the retry counter
func (s *FeedbackService) IncrementRetry(loopID string) error {
	_, err := s.db.Exec(`
		UPDATE feedback_loops
		SET current_retry = current_retry + 1, last_feedback_at = ?
		WHERE id = ?
	`, time.Now(), loopID)
	return err
}

// MarkSuccess marks the feedback loop as successful
func (s *FeedbackService) MarkSuccess(loopID string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE feedback_loops
		SET status = 'success', completed_at = ?
		WHERE id = ?
	`, now, loopID)
	return err
}

// MarkFailed marks the feedback loop as failed
func (s *FeedbackService) MarkFailed(loopID string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE feedback_loops
		SET status = 'failed', completed_at = ?
		WHERE id = ?
	`, now, loopID)
	return err
}

// MarkEscalated marks the feedback loop as escalated
func (s *FeedbackService) MarkEscalated(loopID string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE feedback_loops
		SET status = 'escalated', completed_at = ?
		WHERE id = ?
	`, now, loopID)
	return err
}

// ShouldEscalate checks if the loop should be escalated
func (s *FeedbackService) ShouldEscalate(loop *FeedbackLoop) bool {
	return loop.CurrentRetry >= loop.MaxRetries
}

// ProcessTestFeedback processes test feedback and updates the loop
func (s *FeedbackService) ProcessTestFeedback(ctx context.Context, loopID string, feedback message.TestFeedbackPayload) (*FeedbackResult, error) {
	loop, err := s.GetFeedbackLoop(loopID)
	if err != nil {
		return nil, err
	}
	if loop == nil {
		return nil, fmt.Errorf("피드백 루프를 찾을 수 없습니다: %s", loopID)
	}

	result := &FeedbackResult{
		Iteration:   loop.CurrentRetry + 1,
		Success:     feedback.Success,
		TestsPassed: feedback.PassedTests,
		TestsFailed: feedback.TotalTests - feedback.PassedTests,
		Coverage:    feedback.Coverage,
		FailedTests: feedback.FailedTests,
		Suggestions: feedback.Suggestions,
	}

	if feedback.Success {
		// 테스트 성공 - 루프 완료
		if err := s.MarkSuccess(loopID); err != nil {
			return nil, err
		}
		return result, nil
	}

	// 테스트 실패 - 재시도 카운트 증가
	if err := s.IncrementRetry(loopID); err != nil {
		return nil, err
	}

	// 최대 재시도 초과 시 에스컬레이션
	loop.CurrentRetry++
	if s.ShouldEscalate(loop) {
		if err := s.MarkEscalated(loopID); err != nil {
			return nil, err
		}
		return result, fmt.Errorf("최대 재시도 횟수 초과 (에스컬레이션 필요)")
	}

	// 피드백 메시지를 Impl Worker에 전송
	s.directStore.SendTestFeedback(loop.ChannelID, loop.TestSessionID, loop.ImplSessionID, feedback)

	return result, nil
}

// ListActiveLoops lists all active feedback loops
func (s *FeedbackService) ListActiveLoops(limit int) ([]*FeedbackLoop, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(`
		SELECT id, channel_id, impl_session, test_session, port_id, max_retries, current_retry, status, last_feedback_at, created_at, completed_at
		FROM feedback_loops WHERE status = 'running'
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("피드백 루프 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var loops []*FeedbackLoop
	for rows.Next() {
		var loop FeedbackLoop
		var portID sql.NullString
		var lastFeedbackAt, completedAt sql.NullTime

		err := rows.Scan(
			&loop.ID, &loop.ChannelID, &loop.ImplSessionID, &loop.TestSessionID,
			&portID, &loop.MaxRetries, &loop.CurrentRetry, &loop.Status,
			&lastFeedbackAt, &loop.CreatedAt, &completedAt,
		)
		if err != nil {
			continue
		}

		if portID.Valid {
			loop.PortID = portID.String
		}
		if lastFeedbackAt.Valid {
			loop.LastFeedbackAt = &lastFeedbackAt.Time
		}
		if completedAt.Valid {
			loop.CompletedAt = &completedAt.Time
		}

		loops = append(loops, &loop)
	}

	return loops, nil
}

// FeedbackStats represents statistics for feedback loops
type FeedbackStats struct {
	TotalLoops    int     `json:"total_loops"`
	SuccessLoops  int     `json:"success_loops"`
	FailedLoops   int     `json:"failed_loops"`
	EscalatedLoops int    `json:"escalated_loops"`
	RunningLoops  int     `json:"running_loops"`
	AvgRetries    float64 `json:"avg_retries"`
	SuccessRate   float64 `json:"success_rate"`
}

// GetStats returns feedback loop statistics
func (s *FeedbackService) GetStats() (*FeedbackStats, error) {
	stats := &FeedbackStats{}

	// Count by status
	rows, err := s.db.Query(`
		SELECT status, COUNT(*) as cnt FROM feedback_loops GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		stats.TotalLoops += count
		switch status {
		case "success":
			stats.SuccessLoops = count
		case "failed":
			stats.FailedLoops = count
		case "escalated":
			stats.EscalatedLoops = count
		case "running":
			stats.RunningLoops = count
		}
	}

	// Average retries
	err = s.db.QueryRow(`
		SELECT COALESCE(AVG(current_retry), 0) FROM feedback_loops WHERE status IN ('success', 'failed', 'escalated')
	`).Scan(&stats.AvgRetries)
	if err != nil {
		stats.AvgRetries = 0
	}

	// Success rate
	completed := stats.SuccessLoops + stats.FailedLoops + stats.EscalatedLoops
	if completed > 0 {
		stats.SuccessRate = float64(stats.SuccessLoops) / float64(completed) * 100
	}

	return stats, nil
}

// SerializeFeedbackPayload converts feedback payload to JSON
func SerializeFeedbackPayload(payload interface{}) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
