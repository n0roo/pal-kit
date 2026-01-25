package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/session"
)

// Checkpoint represents a context checkpoint for compact recovery
type Checkpoint struct {
	ID            string         `json:"id"`
	SessionID     string         `json:"session_id"`
	PortID        string         `json:"port_id,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	TokensUsed    int            `json:"tokens_used"`

	// State snapshots
	ActivePort    *PortSnapshot  `json:"active_port,omitempty"`
	LoadedDocs    []DocSnapshot  `json:"loaded_docs"`
	RecentChanges []FileChange   `json:"recent_changes"`
	PendingTasks  []string       `json:"pending_tasks"`

	// Recovery information
	RecoveryPrompt  string `json:"recovery_prompt"`
	RecoveryContext string `json:"recovery_context"`
}

// PortSnapshot represents a snapshot of port state
type PortSnapshot struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Status         string   `json:"status"`
	Progress       int      `json:"progress"`
	CurrentTask    string   `json:"current_task"`
	CompletedTasks []string `json:"completed_tasks"`
	FilePath       string   `json:"file_path,omitempty"`
}

// DocSnapshot represents a snapshot of a loaded document
type DocSnapshot struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Path     string `json:"path"`
	Tokens   int    `json:"tokens"`
}

// FileChange represents a recent file change
type FileChange struct {
	Path      string    `json:"path"`
	ChangedAt time.Time `json:"changed_at"`
	Summary   string    `json:"summary,omitempty"`
}

// CheckpointService manages checkpoint operations
type CheckpointService struct {
	db             *db.DB
	projectRoot    string
	checkpointsDir string
}

// NewCheckpointService creates a new checkpoint service
func NewCheckpointService(database *db.DB, projectRoot string) *CheckpointService {
	return &CheckpointService{
		db:             database,
		projectRoot:    projectRoot,
		checkpointsDir: filepath.Join(projectRoot, ".pal", "checkpoints"),
	}
}

// CreateCheckpoint creates a new checkpoint for the current session
func (s *CheckpointService) CreateCheckpoint(sessionID string) (*Checkpoint, error) {
	// Ensure checkpoints directory exists
	if err := os.MkdirAll(s.checkpointsDir, 0755); err != nil {
		return nil, fmt.Errorf("체크포인트 디렉토리 생성 실패: %w", err)
	}

	// Generate checkpoint ID
	id := fmt.Sprintf("cp-%s-%d", sessionID[:8], time.Now().Unix())

	cp := &Checkpoint{
		ID:        id,
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}

	// Collect active port state (non-critical, errors ignored)
	_ = s.collectPortState(cp)

	// Collect loaded documents (non-critical, errors ignored)
	_ = s.collectLoadedDocs(cp)

	// Collect recent changes (non-critical, errors ignored)
	_ = s.collectRecentChanges(cp)

	// Generate recovery prompt
	cp.RecoveryPrompt = s.generateRecoveryPrompt(cp)
	cp.RecoveryContext = s.generateRecoveryContext(cp)

	// Save checkpoint
	if err := s.saveCheckpoint(cp); err != nil {
		return nil, err
	}

	return cp, nil
}

// collectPortState collects active port state
func (s *CheckpointService) collectPortState(cp *Checkpoint) error {
	portSvc := port.NewService(s.db)
	runningPorts, err := portSvc.List("running", 1)
	if err != nil || len(runningPorts) == 0 {
		return err
	}

	p := runningPorts[0]
	cp.PortID = p.ID

	snapshot := &PortSnapshot{
		ID:     p.ID,
		Status: p.Status,
	}

	if p.Title.Valid {
		snapshot.Title = p.Title.String
	}
	if p.FilePath.Valid {
		snapshot.FilePath = p.FilePath.String
	}

	// TODO: Extract progress and tasks from port spec if available

	cp.ActivePort = snapshot
	return nil
}

// collectLoadedDocs collects currently loaded documents
func (s *CheckpointService) collectLoadedDocs(cp *Checkpoint) error {
	budgetSvc := NewBudgetService(s.db, s.projectRoot)
	status, err := budgetSvc.GetCurrentStatus()
	if err != nil {
		return err
	}

	cp.TokensUsed = status.Used

	for _, item := range status.Items {
		if item.Loaded {
			cp.LoadedDocs = append(cp.LoadedDocs, DocSnapshot{
				ID:       item.ID,
				Name:     item.Name,
				Category: item.Category,
				Tokens:   item.Tokens,
			})
		}
	}

	return nil
}

// collectRecentChanges collects recent file changes
func (s *CheckpointService) collectRecentChanges(cp *Checkpoint) error {
	sessionSvc := session.NewService(s.db)
	// Get file_edit events
	editEvents, err := sessionSvc.GetEvents(cp.SessionID, "file_edit", 10)
	if err != nil {
		return err
	}

	for _, evt := range editEvents {
		path := ""
		if evt.EventData != "" {
			// Try to extract path from JSON data
			var data map[string]interface{}
			if json.Unmarshal([]byte(evt.EventData), &data) == nil {
				if p, ok := data["path"].(string); ok {
					path = p
				}
			}
		}
		if path != "" {
			cp.RecentChanges = append(cp.RecentChanges, FileChange{
				Path:      path,
				ChangedAt: evt.CreatedAt,
			})
		}
	}

	// Get file_write events
	writeEvents, err := sessionSvc.GetEvents(cp.SessionID, "file_write", 10)
	if err != nil {
		return err
	}

	for _, evt := range writeEvents {
		path := ""
		if evt.EventData != "" {
			var data map[string]interface{}
			if json.Unmarshal([]byte(evt.EventData), &data) == nil {
				if p, ok := data["path"].(string); ok {
					path = p
				}
			}
		}
		if path != "" {
			cp.RecentChanges = append(cp.RecentChanges, FileChange{
				Path:      path,
				ChangedAt: evt.CreatedAt,
			})
		}
	}

	return nil
}

// generateRecoveryPrompt generates the recovery prompt
func (s *CheckpointService) generateRecoveryPrompt(cp *Checkpoint) string {
	var sb strings.Builder

	sb.WriteString("## 컨텍스트 복구\n\n")

	// Active port info
	if cp.ActivePort != nil {
		sb.WriteString("### 이전 작업 상태\n")
		sb.WriteString(fmt.Sprintf("- **포트**: %s", cp.ActivePort.ID))
		if cp.ActivePort.Title != "" {
			sb.WriteString(fmt.Sprintf(" - %s", cp.ActivePort.Title))
		}
		sb.WriteString("\n")

		if cp.ActivePort.Progress > 0 {
			sb.WriteString(fmt.Sprintf("- **진행률**: %d%%\n", cp.ActivePort.Progress))
		}
		if cp.ActivePort.CurrentTask != "" {
			sb.WriteString(fmt.Sprintf("- **현재 작업**: %s\n", cp.ActivePort.CurrentTask))
		}
		sb.WriteString("\n")
	}

	// Recent changes
	if len(cp.RecentChanges) > 0 {
		sb.WriteString("### 최근 변경 파일\n")
		for _, change := range cp.RecentChanges {
			ago := formatTimeAgo(change.ChangedAt)
			sb.WriteString(fmt.Sprintf("- `%s` (%s)\n", change.Path, ago))
		}
		sb.WriteString("\n")
	}

	// Loaded docs summary
	if len(cp.LoadedDocs) > 0 {
		sb.WriteString("### 로드된 컨텍스트\n")
		sb.WriteString(fmt.Sprintf("- 문서 %d개, 총 %s 토큰\n", len(cp.LoadedDocs), formatTokens(cp.TokensUsed)))
		sb.WriteString("\n")
	}

	// Pending tasks
	if len(cp.PendingTasks) > 0 {
		sb.WriteString("### 다음 단계\n")
		for _, task := range cp.PendingTasks {
			sb.WriteString(fmt.Sprintf("- %s\n", task))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("포트 명세와 관련 문서가 다시 로드되었습니다.\n")
	sb.WriteString("`pal context status`로 확인할 수 있습니다.\n")

	return sb.String()
}

// generateRecoveryContext generates minimal recovery context
func (s *CheckpointService) generateRecoveryContext(cp *Checkpoint) string {
	var sb strings.Builder

	if cp.ActivePort != nil {
		sb.WriteString(fmt.Sprintf("활성 포트: %s\n", cp.ActivePort.ID))
	}

	if len(cp.RecentChanges) > 0 {
		sb.WriteString("최근 수정 파일:\n")
		for i, change := range cp.RecentChanges {
			if i >= 3 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s\n", change.Path))
		}
	}

	return sb.String()
}

// saveCheckpoint saves checkpoint to file
func (s *CheckpointService) saveCheckpoint(cp *Checkpoint) error {
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("체크포인트 직렬화 실패: %w", err)
	}

	filePath := filepath.Join(s.checkpointsDir, cp.ID+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("체크포인트 저장 실패: %w", err)
	}

	return nil
}

// GetLatestCheckpoint returns the most recent checkpoint for a session
func (s *CheckpointService) GetLatestCheckpoint(sessionID string) (*Checkpoint, error) {
	entries, err := os.ReadDir(s.checkpointsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	prefix := fmt.Sprintf("cp-%s-", sessionID[:8])
	var latest *Checkpoint
	var latestTime time.Time

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".json") {
			cp, err := s.loadCheckpoint(filepath.Join(s.checkpointsDir, entry.Name()))
			if err != nil {
				continue
			}
			if cp.CreatedAt.After(latestTime) {
				latest = cp
				latestTime = cp.CreatedAt
			}
		}
	}

	return latest, nil
}

// GetCheckpoint loads a specific checkpoint by ID
func (s *CheckpointService) GetCheckpoint(checkpointID string) (*Checkpoint, error) {
	filePath := filepath.Join(s.checkpointsDir, checkpointID+".json")
	return s.loadCheckpoint(filePath)
}

// loadCheckpoint loads a checkpoint from file
func (s *CheckpointService) loadCheckpoint(filePath string) (*Checkpoint, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}

	return &cp, nil
}

// ListCheckpoints returns all checkpoints, optionally filtered by session
func (s *CheckpointService) ListCheckpoints(sessionID string, limit int) ([]*Checkpoint, error) {
	entries, err := os.ReadDir(s.checkpointsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var checkpoints []*Checkpoint

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			cp, err := s.loadCheckpoint(filepath.Join(s.checkpointsDir, entry.Name()))
			if err != nil {
				continue
			}

			// Filter by session if specified
			if sessionID != "" && !strings.HasPrefix(cp.SessionID, sessionID) {
				continue
			}

			checkpoints = append(checkpoints, cp)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].CreatedAt.After(checkpoints[j].CreatedAt)
	})

	// Apply limit
	if limit > 0 && len(checkpoints) > limit {
		checkpoints = checkpoints[:limit]
	}

	return checkpoints, nil
}

// CleanOldCheckpoints removes checkpoints older than the specified duration
func (s *CheckpointService) CleanOldCheckpoints(maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(s.checkpointsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			filePath := filepath.Join(s.checkpointsDir, entry.Name())
			cp, err := s.loadCheckpoint(filePath)
			if err != nil {
				continue
			}

			if cp.CreatedAt.Before(cutoff) {
				if err := os.Remove(filePath); err == nil {
					removed++
				}
			}
		}
	}

	return removed, nil
}

// RestoreCheckpoint restores context from a checkpoint
func (s *CheckpointService) RestoreCheckpoint(checkpointID string) (*Checkpoint, error) {
	cp, err := s.GetCheckpoint(checkpointID)
	if err != nil {
		return nil, fmt.Errorf("체크포인트 로드 실패: %w", err)
	}

	// TODO: Regenerate rules files if port is active
	// TODO: Update CLAUDE.md with restored context

	return cp, nil
}

// formatTimeAgo formats time as "X minutes ago" etc
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "방금 전"
	} else if d < time.Hour {
		return fmt.Sprintf("%d분 전", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d시간 전", int(d.Hours()))
	} else {
		return fmt.Sprintf("%d일 전", int(d.Hours()/24))
	}
}

// Note: formatTokens and intToString are defined in tokens.go
