package events

import (
	"sync"
)

// Publisher publishes events to SSE clients
type Publisher struct {
	sse *SSEServer
}

var (
	globalPublisher *Publisher
	publisherOnce   sync.Once
)

// GetPublisher returns the global publisher instance
func GetPublisher() *Publisher {
	publisherOnce.Do(func() {
		globalPublisher = &Publisher{
			sse: NewSSEServer(),
		}
		globalPublisher.sse.Start()
	})
	return globalPublisher
}

// SetSSEServer sets the SSE server (for testing or custom setup)
func (p *Publisher) SetSSEServer(sse *SSEServer) {
	p.sse = sse
}

// GetSSEServer returns the SSE server
func (p *Publisher) GetSSEServer() *SSEServer {
	return p.sse
}

// Publish publishes an event
func (p *Publisher) Publish(event *Event) {
	if p.sse != nil {
		p.sse.Broadcast(event)
	}
}

// PublishSessionStart publishes session start event
func (p *Publisher) PublishSessionStart(sessionID, title, sessionType, projectRoot string) {
	event := NewEvent(EventSessionStart, SessionStartData{
		ID:          sessionID,
		Title:       title,
		Type:        sessionType,
		ProjectRoot: projectRoot,
	}).WithSession(sessionID)

	p.Publish(event)
}

// PublishSessionEnd publishes session end event
func (p *Publisher) PublishSessionEnd(sessionID, reason, status string) {
	event := NewEvent(EventSessionEnd, SessionEndData{
		ID:     sessionID,
		Reason: reason,
		Status: status,
	}).WithSession(sessionID)

	p.Publish(event)
}

// PublishSessionUpdate publishes session update event
func (p *Publisher) PublishSessionUpdate(sessionID string, data interface{}) {
	event := NewEvent(EventSessionUpdate, data).WithSession(sessionID)
	p.Publish(event)
}

// PublishAttentionWarning publishes attention warning event (80%)
func (p *Publisher) PublishAttentionWarning(sessionID string, usagePercent float64, tokensUsed, tokenBudget int) {
	event := NewEvent(EventAttentionWarning, AttentionWarningData{
		UsagePercent: usagePercent,
		TokensUsed:   tokensUsed,
		TokenBudget:  tokenBudget,
		Warning:      "80% 토큰 사용 - 체크포인트 생성 권장",
	}).WithSession(sessionID)

	p.Publish(event)
}

// PublishAttentionCritical publishes attention critical event (90%)
func (p *Publisher) PublishAttentionCritical(sessionID string, usagePercent float64, tokensUsed, tokenBudget int) {
	event := NewEvent(EventAttentionCritical, AttentionWarningData{
		UsagePercent: usagePercent,
		TokensUsed:   tokensUsed,
		TokenBudget:  tokenBudget,
		Warning:      "90% 토큰 사용 - Compact 임박",
	}).WithSession(sessionID)

	p.Publish(event)
}

// PublishCompactTriggered publishes compact triggered event
func (p *Publisher) PublishCompactTriggered(sessionID, trigger, checkpointID, recoveryHint string) {
	event := NewEvent(EventCompactTriggered, CompactTriggeredData{
		Trigger:      trigger,
		CheckpointID: checkpointID,
		RecoveryHint: recoveryHint,
	}).WithSession(sessionID)

	p.Publish(event)
}

// PublishCheckpointCreated publishes checkpoint created event
func (p *Publisher) PublishCheckpointCreated(sessionID, checkpointID, summary, triggerType string, activeFiles []string) {
	event := NewEvent(EventCheckpointCreated, CheckpointCreatedData{
		ID:          checkpointID,
		Summary:     summary,
		TriggerType: triggerType,
		ActiveFiles: activeFiles,
	}).WithSession(sessionID)

	p.Publish(event)
}

// PublishPortStart publishes port start event
func (p *Publisher) PublishPortStart(sessionID, portID, title string, checklist []string) {
	event := NewEvent(EventPortStart, PortStartData{
		ID:        portID,
		Title:     title,
		Checklist: checklist,
	}).WithSession(sessionID).WithPort(portID)

	p.Publish(event)
}

// PublishPortEnd publishes port end event
func (p *Publisher) PublishPortEnd(sessionID, portID, status string, durationSecs int64) {
	event := NewEvent(EventPortEnd, PortEndData{
		ID:       portID,
		Status:   status,
		Duration: durationSecs,
	}).WithSession(sessionID).WithPort(portID)

	p.Publish(event)
}

// PublishPortBlocked publishes port blocked event
func (p *Publisher) PublishPortBlocked(sessionID, portID, reason string) {
	event := NewEvent(EventPortBlocked, map[string]interface{}{
		"id":     portID,
		"reason": reason,
	}).WithSession(sessionID).WithPort(portID)

	p.Publish(event)
}

// PublishChecklistResult publishes checklist result event
func (p *Publisher) PublishChecklistResult(sessionID, portID string, passed bool, passedCount, failedCount int, items []CheckItem, blockedBy []string) {
	eventType := EventChecklistPassed
	if !passed {
		eventType = EventChecklistFailed
	}

	event := NewEvent(eventType, ChecklistResultData{
		Passed:      passed,
		PassedCount: passedCount,
		FailedCount: failedCount,
		Items:       items,
		BlockedBy:   blockedBy,
	}).WithSession(sessionID).WithPort(portID)

	p.Publish(event)
}

// PublishEscalationCreated publishes escalation created event
func (p *Publisher) PublishEscalationCreated(sessionID, portID, escalationID, escalationType, issue, suggestion string) {
	event := NewEvent(EventEscalationCreated, EscalationData{
		ID:         escalationID,
		Type:       escalationType,
		Issue:      issue,
		Suggestion: suggestion,
	}).WithSession(sessionID).WithPort(portID)

	p.Publish(event)
}

// PublishBuildFailed publishes build failed event
func (p *Publisher) PublishBuildFailed(sessionID, portID string, exitCode int, errorMsg string) {
	event := NewEvent(EventBuildFailed, BuildFailedData{
		FailType: "build",
		ExitCode: exitCode,
		Error:    errorMsg,
	}).WithSession(sessionID).WithPort(portID)

	p.Publish(event)
}

// PublishTestFailed publishes test failed event
func (p *Publisher) PublishTestFailed(sessionID, portID string, exitCode int, errorMsg string) {
	event := NewEvent(EventTestFailed, BuildFailedData{
		FailType: "test",
		ExitCode: exitCode,
		Error:    errorMsg,
	}).WithSession(sessionID).WithPort(portID)

	p.Publish(event)
}

// PublishMessageReceived publishes message received event
func (p *Publisher) PublishMessageReceived(sessionID string, data interface{}) {
	event := NewEvent(EventMessageReceived, data).WithSession(sessionID)
	p.Publish(event)
}
