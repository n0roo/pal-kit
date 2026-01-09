package usage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// JSONLMessage represents a message from Claude Code JSONL file
type JSONLMessage struct {
	Type      string    `json:"type"`
	UUID      string    `json:"uuid"`
	SessionID string    `json:"sessionId"`
	Timestamp time.Time `json:"timestamp"`
	Message   struct {
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content interface{} `json:"content"`
		Usage   *TokenUsage `json:"usage"`
	} `json:"message"`
	CostUSD    float64 `json:"costUSD"`
	DurationMs int64   `json:"durationMs"`
}

// TokenUsage represents token usage from API response
type TokenUsage struct {
	InputTokens             int64 `json:"input_tokens"`
	OutputTokens            int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens    int64 `json:"cache_read_input_tokens"`
}

// SessionUsage represents aggregated usage for a session
type SessionUsage struct {
	SessionID         string
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheCreateTokens int64
	CostUSD           float64
	MessageCount      int
	FirstMessage      time.Time
	LastMessage       time.Time
}

// Service handles usage tracking
type Service struct {
	db *db.DB
}

// NewService creates a new usage service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ParseJSONLFile parses a Claude Code JSONL file and returns usage
func ParseJSONLFile(path string) (*SessionUsage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("파일 열기 실패: %w", err)
	}
	defer file.Close()

	usage := &SessionUsage{}
	scanner := bufio.NewScanner(file)
	
	// 버퍼 크기 증가 (큰 메시지 처리)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg JSONLMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue // 파싱 실패한 라인 무시
		}

		// 세션 ID 설정
		if usage.SessionID == "" && msg.SessionID != "" {
			usage.SessionID = msg.SessionID
		}

		// assistant 메시지에서 usage 추출
		if msg.Type == "assistant" && msg.Message.Usage != nil {
			u := msg.Message.Usage
			usage.InputTokens += u.InputTokens
			usage.OutputTokens += u.OutputTokens
			usage.CacheReadTokens += u.CacheReadInputTokens
			usage.CacheCreateTokens += u.CacheCreationInputTokens
			usage.CostUSD += msg.CostUSD
			usage.MessageCount++

			if usage.FirstMessage.IsZero() || msg.Timestamp.Before(usage.FirstMessage) {
				usage.FirstMessage = msg.Timestamp
			}
			if msg.Timestamp.After(usage.LastMessage) {
				usage.LastMessage = msg.Timestamp
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("파일 읽기 오류: %w", err)
	}

	return usage, nil
}

// SyncFromJSONL syncs usage data from JSONL files to database
func (s *Service) SyncFromJSONL(projectsDir string) (int, error) {
	synced := 0

	// ~/.claude/projects/ 스캔
	err := filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 에러 무시하고 계속
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".jsonl") {
			usage, err := ParseJSONLFile(path)
			if err != nil {
				return nil // 파싱 실패 무시
			}

			if usage.SessionID != "" {
				// DB 업데이트
				_, err = s.db.Exec(`
					UPDATE sessions 
					SET input_tokens = ?, output_tokens = ?, 
					    cache_read_tokens = ?, cache_create_tokens = ?,
					    cost_usd = ?, jsonl_path = ?
					WHERE id = ?
				`, usage.InputTokens, usage.OutputTokens,
					usage.CacheReadTokens, usage.CacheCreateTokens,
					usage.CostUSD, path, usage.SessionID)

				if err == nil {
					synced++
				}
			}
		}

		return nil
	})

	return synced, err
}

// SyncSession syncs a specific session's usage
func (s *Service) SyncSession(sessionID, jsonlPath string) error {
	usage, err := ParseJSONLFile(jsonlPath)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		UPDATE sessions 
		SET input_tokens = ?, output_tokens = ?, 
		    cache_read_tokens = ?, cache_create_tokens = ?,
		    cost_usd = ?, jsonl_path = ?
		WHERE id = ?
	`, usage.InputTokens, usage.OutputTokens,
		usage.CacheReadTokens, usage.CacheCreateTokens,
		usage.CostUSD, jsonlPath, sessionID)

	return err
}

// GetSummary returns usage summary
func (s *Service) GetSummary(since time.Time) (*SessionUsage, error) {
	var summary SessionUsage

	query := `
		SELECT COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
		       COALESCE(SUM(cache_read_tokens), 0), COALESCE(SUM(cache_create_tokens), 0),
		       COALESCE(SUM(cost_usd), 0), COUNT(*)
		FROM sessions
	`

	var args []interface{}
	if !since.IsZero() {
		query += ` WHERE started_at >= ?`
		args = append(args, since)
	}

	err := s.db.QueryRow(query, args...).Scan(
		&summary.InputTokens, &summary.OutputTokens,
		&summary.CacheReadTokens, &summary.CacheCreateTokens,
		&summary.CostUSD, &summary.MessageCount,
	)

	return &summary, err
}

// GetSessionUsage returns usage for a specific session
func (s *Service) GetSessionUsage(sessionID string) (*SessionUsage, error) {
	var summary SessionUsage
	summary.SessionID = sessionID

	err := s.db.QueryRow(`
		SELECT input_tokens, output_tokens, cache_read_tokens, cache_create_tokens, cost_usd
		FROM sessions WHERE id = ?
	`, sessionID).Scan(
		&summary.InputTokens, &summary.OutputTokens,
		&summary.CacheReadTokens, &summary.CacheCreateTokens,
		&summary.CostUSD,
	)

	return &summary, err
}

// GetProjectsDir returns the Claude projects directory
func GetProjectsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}
