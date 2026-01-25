package checkpoint

import (
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/attention"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/port"
)

// Monitor monitors token usage and creates auto checkpoints
type Monitor struct {
	store    *Store
	attStore *attention.Store
	portSvc  *port.Service
}

// NewMonitor creates a new checkpoint monitor
func NewMonitor(database *db.DB) *Monitor {
	store := NewStore(database)
	store.EnsureTable() // Ensure table exists

	return &Monitor{
		store:    store,
		attStore: attention.NewStore(database.DB),
		portSvc:  port.NewService(database),
	}
}

// CheckResult represents the result of a checkpoint check
type CheckResult struct {
	Usage      float64     `json:"usage"`       // Token usage ratio (0-1)
	Checkpoint *Checkpoint `json:"checkpoint"`  // Created checkpoint (if any)
	Message    string      `json:"message"`     // Message for Claude
	Warning    bool        `json:"warning"`     // Is this a warning level?
}

// ThresholdConfig defines checkpoint thresholds
type ThresholdConfig struct {
	Threshold80  float64       // 80% threshold
	Threshold90  float64       // 90% threshold
	Cooldown     time.Duration // Minimum time between checkpoints of same type
}

// DefaultThresholdConfig returns default threshold configuration
func DefaultThresholdConfig() ThresholdConfig {
	return ThresholdConfig{
		Threshold80: 0.8,
		Threshold90: 0.9,
		Cooldown:    5 * time.Minute,
	}
}

// CheckAndCreate checks token usage and creates checkpoint if needed
// Called from PreToolUse Hook
func (m *Monitor) CheckAndCreate(sessionID string, tokensUsed, tokenBudget int) (*CheckResult, error) {
	config := DefaultThresholdConfig()

	// Calculate usage ratio
	usage := float64(0)
	if tokenBudget > 0 {
		usage = float64(tokensUsed) / float64(tokenBudget)
	}

	result := &CheckResult{
		Usage: usage,
	}

	// 90% threshold - higher priority
	if usage >= config.Threshold90 {
		if !m.store.HasRecentCheckpoint(sessionID, "auto_90", config.Cooldown) {
			cp := m.createAutoCheckpoint(sessionID, "auto_90", tokensUsed, tokenBudget)
			result.Checkpoint = cp
			result.Warning = true
			result.Message = fmt.Sprintf("⚠️ 토큰 %.0f%% - 작업 마무리 권장", usage*100)
		}
		return result, nil
	}

	// 80% threshold
	if usage >= config.Threshold80 {
		if !m.store.HasRecentCheckpoint(sessionID, "auto_80", config.Cooldown) {
			cp := m.createAutoCheckpoint(sessionID, "auto_80", tokensUsed, tokenBudget)
			result.Checkpoint = cp
			result.Warning = false
			result.Message = fmt.Sprintf("체크포인트 생성됨 (토큰 %.0f%%)", usage*100)
		}
		return result, nil
	}

	return result, nil
}

// createAutoCheckpoint creates an automatic checkpoint
func (m *Monitor) createAutoCheckpoint(sessionID, triggerType string, tokensUsed, tokenBudget int) *Checkpoint {
	// Get current port
	portID := ""
	runningPorts, err := m.portSvc.List("running", 1)
	if err == nil && len(runningPorts) > 0 {
		portID = runningPorts[0].ID
	}

	// Generate summary based on trigger type
	summary := m.generateSummary(sessionID, triggerType, tokensUsed, tokenBudget)

	// Get active files from attention state
	activeFiles := m.getActiveFiles(sessionID)

	// Extract key points
	keyPoints := m.extractKeyPoints(sessionID, portID)

	cp := &Checkpoint{
		SessionID:   sessionID,
		PortID:      portID,
		TriggerType: triggerType,
		TokensUsed:  tokensUsed,
		TokenBudget: tokenBudget,
		Summary:     summary,
		ActiveFiles: activeFiles,
		KeyPoints:   keyPoints,
	}

	if err := m.store.Create(cp); err != nil {
		fmt.Printf("체크포인트 생성 실패: %v\n", err)
		return nil
	}

	return cp
}

// generateSummary generates a summary for the checkpoint
func (m *Monitor) generateSummary(sessionID, triggerType string, tokensUsed, tokenBudget int) string {
	usage := float64(tokensUsed) / float64(tokenBudget) * 100

	switch triggerType {
	case "auto_80":
		return fmt.Sprintf("자동 체크포인트 (토큰 %.0f%% 사용)", usage)
	case "auto_90":
		return fmt.Sprintf("경고 체크포인트 (토큰 %.0f%% 사용) - 작업 마무리 권장", usage)
	case "manual":
		return "수동 체크포인트"
	default:
		return fmt.Sprintf("체크포인트 (토큰 %.0f%%)", usage)
	}
}

// getActiveFiles gets active files from attention state
func (m *Monitor) getActiveFiles(sessionID string) []string {
	state, err := m.attStore.Get(sessionID)
	if err != nil {
		return []string{}
	}

	return state.LoadedFiles
}

// extractKeyPoints extracts key points from current context
func (m *Monitor) extractKeyPoints(sessionID, portID string) []string {
	keyPoints := []string{}

	// Get port info
	if portID != "" {
		p, err := m.portSvc.Get(portID)
		if err == nil {
			if p.Title.Valid {
				keyPoints = append(keyPoints, fmt.Sprintf("작업 중: %s", p.Title.String))
			}
		}
	}

	return keyPoints
}

// CreateManual creates a manual checkpoint
func (m *Monitor) CreateManual(sessionID, summary string, tokensUsed, tokenBudget int) (*Checkpoint, error) {
	// Get current port
	portID := ""
	runningPorts, err := m.portSvc.List("running", 1)
	if err == nil && len(runningPorts) > 0 {
		portID = runningPorts[0].ID
	}

	activeFiles := m.getActiveFiles(sessionID)
	keyPoints := m.extractKeyPoints(sessionID, portID)

	cp := &Checkpoint{
		SessionID:   sessionID,
		PortID:      portID,
		TriggerType: "manual",
		TokensUsed:  tokensUsed,
		TokenBudget: tokenBudget,
		Summary:     summary,
		ActiveFiles: activeFiles,
		KeyPoints:   keyPoints,
	}

	if err := m.store.Create(cp); err != nil {
		return nil, fmt.Errorf("체크포인트 생성 실패: %w", err)
	}

	return cp, nil
}

// GetLatest returns the latest checkpoint for a session
func (m *Monitor) GetLatest(sessionID string) (*Checkpoint, error) {
	return m.store.GetLatest(sessionID)
}

// List returns checkpoints for a session
func (m *Monitor) List(sessionID string, limit int) ([]*Checkpoint, error) {
	return m.store.List(sessionID, limit)
}

// GetByID returns a checkpoint by ID
func (m *Monitor) GetByID(id string) (*Checkpoint, error) {
	return m.store.GetByID(id)
}

// GenerateRecoveryContext generates recovery context from the latest checkpoint
func (m *Monitor) GenerateRecoveryContext(sessionID string) (map[string]interface{}, error) {
	cp, err := m.store.GetLatest(sessionID)
	if err != nil {
		return nil, fmt.Errorf("체크포인트 없음: %w", err)
	}

	return map[string]interface{}{
		"checkpoint_id":   cp.ID,
		"summary":         cp.Summary,
		"active_files":    cp.ActiveFiles,
		"key_points":      cp.KeyPoints,
		"port_id":         cp.PortID,
		"created_at":      cp.CreatedAt,
		"recovery_prompt": m.generateRecoveryPrompt(cp),
	}, nil
}

// generateRecoveryPrompt generates a recovery prompt for Claude
func (m *Monitor) generateRecoveryPrompt(cp *Checkpoint) string {
	prompt := fmt.Sprintf("## 체크포인트 복구\n\n**요약:** %s\n\n", cp.Summary)

	if cp.PortID != "" {
		prompt += fmt.Sprintf("**활성 포트:** %s\n\n", cp.PortID)
	}

	if len(cp.ActiveFiles) > 0 {
		prompt += "**작업 중인 파일:**\n"
		for _, f := range cp.ActiveFiles {
			prompt += fmt.Sprintf("- %s\n", f)
		}
		prompt += "\n"
	}

	if len(cp.KeyPoints) > 0 {
		prompt += "**핵심 포인트:**\n"
		for _, p := range cp.KeyPoints {
			prompt += fmt.Sprintf("- %s\n", p)
		}
		prompt += "\n"
	}

	prompt += "위 컨텍스트를 참고하여 작업을 계속하세요.\n"

	return prompt
}
