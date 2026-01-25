package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// v10 Session Types (계층적 구조)
const (
	TypeBuild    = "build"    // 명세 설계 세션 (최상위)
	TypeOperator = "operator" // 워커 관리 세션
	TypeWorker   = "worker"   // 코드 구현 세션
	TypeTest     = "test"     // 테스트 세션
)

// HierarchicalSession extends Session with v10 hierarchy fields
type HierarchicalSession struct {
	Session

	// 계층 관계 (v10)
	ParentID        sql.NullString `json:"parent_id,omitempty"`
	RootID          sql.NullString `json:"root_id,omitempty"`
	Depth           int            `json:"depth"`
	Path            string         `json:"path"`
	Type            string         `json:"type"`
	AgentID         sql.NullString `json:"agent_id,omitempty"`
	AgentVersion    sql.NullInt64  `json:"agent_version,omitempty"`
	Substatus       sql.NullString `json:"substatus,omitempty"`
	AttentionScore  sql.NullFloat64 `json:"attention_score,omitempty"`
	TokenBudget     sql.NullInt64  `json:"token_budget,omitempty"`
	ContextSnapshot sql.NullString `json:"context_snapshot,omitempty"`
	CheckpointID    sql.NullString `json:"checkpoint_id,omitempty"`
	OutputSummary   sql.NullString `json:"output_summary,omitempty"`
}

// SessionHierarchyNode represents a node in session hierarchy tree
type SessionHierarchyNode struct {
	Session  HierarchicalSession   `json:"session"`
	Children []*SessionHierarchyNode `json:"children,omitempty"`
}

// HierarchyStartOptions contains options for creating hierarchical sessions
type HierarchyStartOptions struct {
	ID              string
	Title           string
	Type            string  // build, operator, worker, test
	ParentID        string  // 부모 세션 ID
	PortID          string
	AgentID         string
	AgentVersion    int
	TokenBudget     int
	ProjectRoot     string
	ProjectName     string
	ClaudeSessionID string
	Cwd             string
}

// StartHierarchical creates a new hierarchical session
func (s *Service) StartHierarchical(opts HierarchyStartOptions) (*HierarchicalSession, error) {
	if opts.ID == "" {
		opts.ID = uuid.New().String()
	}
	if opts.Type == "" {
		opts.Type = TypeWorker
	}
	if opts.TokenBudget == 0 {
		opts.TokenBudget = 15000
	}

	// 계층 정보 계산
	var depth int
	var rootID, path string

	if opts.ParentID != "" {
		parent, err := s.GetHierarchical(opts.ParentID)
		if err != nil {
			return nil, fmt.Errorf("부모 세션 '%s' 조회 실패: %w", opts.ParentID, err)
		}
		depth = parent.Depth + 1
		if parent.RootID.Valid {
			rootID = parent.RootID.String
		} else {
			rootID = parent.Session.ID
		}
		if parent.Path != "" {
			path = parent.Path + "/" + opts.ID
		} else {
			path = parent.Session.ID + "/" + opts.ID
		}
	} else {
		// 루트 세션 (build)
		depth = 0
		rootID = opts.ID
		path = opts.ID
	}

	// 세션 생성
	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO sessions (
			id, port_id, title, status, started_at,
			parent_id, root_id, depth, path, type,
			agent_id, agent_version, token_budget,
			project_root, project_name, claude_session_id, cwd
		) VALUES (?, ?, ?, 'running', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, opts.ID, nullString(opts.PortID), nullString(opts.Title), now,
		nullString(opts.ParentID), nullString(rootID), depth, path, opts.Type,
		nullString(opts.AgentID), nullInt(opts.AgentVersion), opts.TokenBudget,
		nullString(opts.ProjectRoot), nullString(opts.ProjectName),
		nullString(opts.ClaudeSessionID), nullString(opts.Cwd))

	if err != nil {
		return nil, fmt.Errorf("세션 생성 실패: %w", err)
	}

	// 이벤트 로깅
	s.LogEvent(opts.ID, EventSessionStart, fmt.Sprintf(
		`{"type":"%s","parent":"%s","depth":%d}`, opts.Type, opts.ParentID, depth))

	return s.GetHierarchical(opts.ID)
}

// GetHierarchical retrieves a hierarchical session by ID
func (s *Service) GetHierarchical(id string) (*HierarchicalSession, error) {
	var sess HierarchicalSession

	err := s.db.QueryRow(`
		SELECT id, port_id, title, status, started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       parent_id, root_id, COALESCE(depth, 0), COALESCE(path, ''), COALESCE(type, 'single'),
		       agent_id, agent_version, substatus, attention_score,
		       token_budget, context_snapshot, checkpoint_id, output_summary,
		       claude_session_id, project_root, project_name, transcript_path, cwd
		FROM sessions WHERE id = ?
	`, id).Scan(
		&sess.Session.ID, &sess.Session.PortID, &sess.Session.Title, &sess.Session.Status,
		&sess.Session.StartedAt, &sess.Session.EndedAt, &sess.Session.JSONLPath,
		&sess.Session.InputTokens, &sess.Session.OutputTokens, &sess.Session.CacheReadTokens,
		&sess.Session.CacheCreateTokens, &sess.Session.CostUSD, &sess.Session.CompactCount,
		&sess.Session.LastCompactAt,
		&sess.ParentID, &sess.RootID, &sess.Depth, &sess.Path, &sess.Type,
		&sess.AgentID, &sess.AgentVersion, &sess.Substatus, &sess.AttentionScore,
		&sess.TokenBudget, &sess.ContextSnapshot, &sess.CheckpointID, &sess.OutputSummary,
		&sess.Session.ClaudeSessionID, &sess.Session.ProjectRoot, &sess.Session.ProjectName,
		&sess.Session.TranscriptPath, &sess.Session.Cwd,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("세션 '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	return &sess, nil
}

// GetSessionHierarchy returns the full hierarchy tree for a root session
func (s *Service) GetSessionHierarchy(rootID string) (*SessionHierarchyNode, error) {
	root, err := s.GetHierarchical(rootID)
	if err != nil {
		return nil, err
	}

	node := &SessionHierarchyNode{Session: *root}
	if err := s.buildHierarchyTree(node); err != nil {
		return nil, err
	}

	return node, nil
}

// buildHierarchyTree recursively builds the hierarchy tree
func (s *Service) buildHierarchyTree(node *SessionHierarchyNode) error {
	rows, err := s.db.Query(`
		SELECT id FROM sessions WHERE parent_id = ? ORDER BY started_at
	`, node.Session.Session.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var childID string
		if err := rows.Scan(&childID); err != nil {
			continue
		}

		child, err := s.GetHierarchical(childID)
		if err != nil {
			continue
		}

		childNode := &SessionHierarchyNode{Session: *child}
		if err := s.buildHierarchyTree(childNode); err != nil {
			continue
		}
		node.Children = append(node.Children, childNode)
	}

	return nil
}

// ListByRoot returns all sessions under a root
func (s *Service) ListByRoot(rootID string) ([]*HierarchicalSession, error) {
	rows, err := s.db.Query(`
		SELECT id, port_id, title, status, started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       parent_id, root_id, COALESCE(depth, 0), COALESCE(path, ''), COALESCE(type, 'single'),
		       agent_id, agent_version, substatus, attention_score,
		       token_budget, context_snapshot, checkpoint_id, output_summary,
		       claude_session_id, project_root, project_name, transcript_path, cwd
		FROM sessions
		WHERE root_id = ? OR id = ?
		ORDER BY depth, started_at
	`, rootID, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanHierarchicalSessions(rows)
}

// ListByType returns sessions of a specific type
func (s *Service) ListByType(sessionType string, activeOnly bool, limit int) ([]*HierarchicalSession, error) {
	query := `
		SELECT id, port_id, title, status, started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       parent_id, root_id, COALESCE(depth, 0), COALESCE(path, ''), COALESCE(type, 'single'),
		       agent_id, agent_version, substatus, attention_score,
		       token_budget, context_snapshot, checkpoint_id, output_summary,
		       claude_session_id, project_root, project_name, transcript_path, cwd
		FROM sessions WHERE type = ?
	`
	args := []interface{}{sessionType}

	if activeOnly {
		query += " AND status = 'running'"
	}
	query += " ORDER BY started_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanHierarchicalSessions(rows)
}

// scanHierarchicalSessions scans rows into HierarchicalSession slice
func (s *Service) scanHierarchicalSessions(rows *sql.Rows) ([]*HierarchicalSession, error) {
	var sessions []*HierarchicalSession
	for rows.Next() {
		var sess HierarchicalSession
		err := rows.Scan(
			&sess.Session.ID, &sess.Session.PortID, &sess.Session.Title, &sess.Session.Status,
			&sess.Session.StartedAt, &sess.Session.EndedAt, &sess.Session.JSONLPath,
			&sess.Session.InputTokens, &sess.Session.OutputTokens, &sess.Session.CacheReadTokens,
			&sess.Session.CacheCreateTokens, &sess.Session.CostUSD, &sess.Session.CompactCount,
			&sess.Session.LastCompactAt,
			&sess.ParentID, &sess.RootID, &sess.Depth, &sess.Path, &sess.Type,
			&sess.AgentID, &sess.AgentVersion, &sess.Substatus, &sess.AttentionScore,
			&sess.TokenBudget, &sess.ContextSnapshot, &sess.CheckpointID, &sess.OutputSummary,
			&sess.Session.ClaudeSessionID, &sess.Session.ProjectRoot, &sess.Session.ProjectName,
			&sess.Session.TranscriptPath, &sess.Session.Cwd,
		)
		if err != nil {
			continue
		}
		sessions = append(sessions, &sess)
	}
	return sessions, nil
}

// UpdateSubstatus updates the substatus of a session
func (s *Service) UpdateSubstatus(id, substatus string) error {
	_, err := s.db.Exec(`
		UPDATE sessions SET substatus = ? WHERE id = ?
	`, substatus, id)
	return err
}

// UpdateAttention updates attention metrics for a session
func (s *Service) UpdateAttention(id string, score float64) error {
	_, err := s.db.Exec(`
		UPDATE sessions SET attention_score = ? WHERE id = ?
	`, score, id)
	return err
}

// UpdateContextSnapshot saves a context snapshot for recovery
func (s *Service) UpdateContextSnapshot(id, snapshot, checkpointID string) error {
	_, err := s.db.Exec(`
		UPDATE sessions SET context_snapshot = ?, checkpoint_id = ? WHERE id = ?
	`, snapshot, checkpointID, id)
	return err
}

// UpdateOutputSummary updates the output summary for a session
func (s *Service) UpdateOutputSummary(id string, summary interface{}) error {
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		UPDATE sessions SET output_summary = ? WHERE id = ?
	`, string(summaryJSON), id)
	return err
}

// GetBuildSessions returns all build-type sessions
func (s *Service) GetBuildSessions(activeOnly bool, limit int) ([]*HierarchicalSession, error) {
	return s.ListByType(TypeBuild, activeOnly, limit)
}

// GetRootHierarchicalSessions returns all root-level sessions (build type OR no parent) with hierarchy info
func (s *Service) GetRootHierarchicalSessions(activeOnly bool, limit int) ([]*HierarchicalSession, error) {
	query := `
		SELECT id, port_id, title, status, started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       parent_id, root_id, COALESCE(depth, 0), COALESCE(path, ''), COALESCE(type, 'single'),
		       agent_id, agent_version, substatus, attention_score,
		       token_budget, context_snapshot, checkpoint_id, output_summary,
		       claude_session_id, project_root, project_name, transcript_path, cwd
		FROM sessions 
		WHERE (type = 'build' OR parent_id IS NULL OR parent_id = '')
	`

	if activeOnly {
		query += " AND status = 'running'"
	}
	query += " ORDER BY started_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanHierarchicalSessions(rows)
}

// GetHierarchyStats returns statistics for a session hierarchy
func (s *Service) GetHierarchyStats(rootID string) (*HierarchyStats, error) {
	var stats HierarchyStats

	err := s.db.QueryRow(`
		SELECT 
			COUNT(*) as total_sessions,
			SUM(CASE WHEN type = 'operator' THEN 1 ELSE 0 END) as operator_count,
			SUM(CASE WHEN type = 'worker' THEN 1 ELSE 0 END) as worker_count,
			SUM(CASE WHEN type = 'test' THEN 1 ELSE 0 END) as test_count,
			SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as running_count,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as complete_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(AVG(attention_score), 0) as avg_attention,
			COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens,
			COALESCE(SUM(compact_count), 0) as total_compacts
		FROM sessions
		WHERE root_id = ? OR id = ?
	`, rootID, rootID).Scan(
		&stats.TotalSessions, &stats.OperatorCount, &stats.WorkerCount, &stats.TestCount,
		&stats.RunningCount, &stats.CompleteCount, &stats.FailedCount,
		&stats.AvgAttention, &stats.TotalTokens, &stats.TotalCompacts,
	)

	if err != nil {
		return nil, err
	}

	// Calculate progress
	if stats.TotalSessions > 0 {
		stats.ProgressPercent = float64(stats.CompleteCount) / float64(stats.TotalSessions) * 100
	}

	return &stats, nil
}

// HierarchyStats represents statistics for a session hierarchy
type HierarchyStats struct {
	TotalSessions   int     `json:"total_sessions"`
	OperatorCount   int     `json:"operator_count"`
	WorkerCount     int     `json:"worker_count"`
	TestCount       int     `json:"test_count"`
	RunningCount    int     `json:"running_count"`
	CompleteCount   int     `json:"complete_count"`
	FailedCount     int     `json:"failed_count"`
	AvgAttention    float64 `json:"avg_attention"`
	TotalTokens     int64   `json:"total_tokens"`
	TotalCompacts   int     `json:"total_compacts"`
	ProgressPercent float64 `json:"progress_percent"`
}

// EndWithSummary ends a session and saves output summary
func (s *Service) EndWithSummary(id string, status string, summary interface{}) error {
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		summaryJSON = []byte("{}")
	}

	_, err = s.db.Exec(`
		UPDATE sessions 
		SET status = ?, ended_at = CURRENT_TIMESTAMP, output_summary = ?
		WHERE id = ?
	`, status, string(summaryJSON), id)

	if err != nil {
		return fmt.Errorf("세션 종료 실패: %w", err)
	}

	// 이벤트 로깅
	s.LogEvent(id, EventSessionEnd, fmt.Sprintf(`{"status":"%s"}`, status))

	return nil
}

// Helper functions
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}
