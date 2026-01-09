package session

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Session type constants
const (
	TypeSingle  = "single"  // 단일 세션
	TypeMulti   = "multi"   // 멀티 세션 (병렬 독립)
	TypeSub     = "sub"     // 서브 세션 (상위에서 spawn)
	TypeBuilder = "builder" // 빌더 세션 (파이프라인 관리)
)

// Session represents a work session
type Session struct {
	ID                string
	PortID            sql.NullString
	Title             sql.NullString
	Status            string
	SessionType       string
	ParentSession     sql.NullString
	StartedAt         time.Time
	EndedAt           sql.NullTime
	JSONLPath         sql.NullString
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheCreateTokens int64
	CostUSD           float64
	CompactCount      int
	LastCompactAt     sql.NullTime
}

// SessionNode represents a session in a tree structure
type SessionNode struct {
	Session  Session
	Children []SessionNode
}

// Service handles session operations
type Service struct {
	db *db.DB
}

// NewService creates a new session service
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Start creates a new session
func (s *Service) Start(id, portID, title string) error {
	return s.StartWithOptions(id, portID, title, TypeSingle, "")
}

// StartWithOptions creates a new session with type and parent
func (s *Service) StartWithOptions(id, portID, title, sessionType, parentSession string) error {
	var portIDNull, titleNull, parentNull sql.NullString

	if portID != "" {
		// 포트 존재 여부 확인 (없으면 NULL로 처리)
		var exists int
		err := s.db.QueryRow(`SELECT 1 FROM ports WHERE id = ?`, portID).Scan(&exists)
		if err == nil {
			portIDNull = sql.NullString{String: portID, Valid: true}
		}
		// 포트가 없어도 세션은 생성 (경고만)
	}
	if title != "" {
		titleNull = sql.NullString{String: title, Valid: true}
	}
	if parentSession != "" {
		// 상위 세션 존재 여부 확인
		var exists int
		err := s.db.QueryRow(`SELECT 1 FROM sessions WHERE id = ?`, parentSession).Scan(&exists)
		if err != nil {
			return fmt.Errorf("상위 세션 '%s'을(를) 찾을 수 없습니다", parentSession)
		}
		parentNull = sql.NullString{String: parentSession, Valid: true}
	}
	if sessionType == "" {
		sessionType = TypeSingle
	}

	_, err := s.db.Exec(`
		INSERT INTO sessions (id, port_id, title, status, session_type, parent_session)
		VALUES (?, ?, ?, 'running', ?, ?)
	`, id, portIDNull, titleNull, sessionType, parentNull)

	if err != nil {
		return fmt.Errorf("세션 생성 실패: %w", err)
	}
	return nil
}

// End marks a session as ended
func (s *Service) End(id string) error {
	result, err := s.db.Exec(`
		UPDATE sessions 
		SET status = 'complete', ended_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'running'
	`, id)

	if err != nil {
		return fmt.Errorf("세션 종료 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("세션 '%s'을(를) 찾을 수 없거나 이미 종료됨", id)
	}

	return nil
}

// UpdateStatus updates session status
func (s *Service) UpdateStatus(id, status string) error {
	result, err := s.db.Exec(`
		UPDATE sessions SET status = ? WHERE id = ?
	`, status, id)

	if err != nil {
		return fmt.Errorf("상태 업데이트 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("세션 '%s'을(를) 찾을 수 없습니다", id)
	}

	return nil
}

// Get retrieves a session by ID
func (s *Service) Get(id string) (*Session, error) {
	var sess Session
	var sessionType, parentSession sql.NullString

	err := s.db.QueryRow(`
		SELECT id, port_id, title, status, 
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at
		FROM sessions WHERE id = ?
	`, id).Scan(
		&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
		&sessionType, &parentSession,
		&sess.StartedAt, &sess.EndedAt,
		&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
		&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("세션 '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	sess.SessionType = sessionType.String
	if sessionType.Valid {
		sess.SessionType = sessionType.String
	} else {
		sess.SessionType = TypeSingle
	}
	sess.ParentSession = parentSession

	return &sess, nil
}

// List returns sessions with optional filters
func (s *Service) List(activeOnly bool, limit int) ([]Session, error) {
	query := `
		SELECT id, port_id, title, status, 
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at
		FROM sessions
	`

	if activeOnly {
		query += ` WHERE status = 'running'`
	}

	query += ` ORDER BY started_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("세션 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var sessionType, parentSession sql.NullString
		if err := rows.Scan(
			&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
			&sessionType, &parentSession,
			&sess.StartedAt, &sess.EndedAt,
			&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
			&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		); err != nil {
			return nil, err
		}
		if sessionType.Valid {
			sess.SessionType = sessionType.String
		} else {
			sess.SessionType = TypeSingle
		}
		sess.ParentSession = parentSession
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// GetChildren returns child sessions of a parent
func (s *Service) GetChildren(parentID string) ([]Session, error) {
	rows, err := s.db.Query(`
		SELECT id, port_id, title, status, 
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at
		FROM sessions
		WHERE parent_session = ?
		ORDER BY started_at
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var sessionType, parentSession sql.NullString
		if err := rows.Scan(
			&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
			&sessionType, &parentSession,
			&sess.StartedAt, &sess.EndedAt,
			&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
			&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		); err != nil {
			return nil, err
		}
		if sessionType.Valid {
			sess.SessionType = sessionType.String
		} else {
			sess.SessionType = TypeSingle
		}
		sess.ParentSession = parentSession
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// GetTree returns a session tree starting from the given session
func (s *Service) GetTree(rootID string) (*SessionNode, error) {
	sess, err := s.Get(rootID)
	if err != nil {
		return nil, err
	}

	node := &SessionNode{Session: *sess}
	if err := s.buildTree(node); err != nil {
		return nil, err
	}

	return node, nil
}

// buildTree recursively builds the session tree
func (s *Service) buildTree(node *SessionNode) error {
	children, err := s.GetChildren(node.Session.ID)
	if err != nil {
		return err
	}

	for _, child := range children {
		childNode := SessionNode{Session: child}
		if err := s.buildTree(&childNode); err != nil {
			return err
		}
		node.Children = append(node.Children, childNode)
	}

	return nil
}

// GetRootSessions returns sessions without a parent
func (s *Service) GetRootSessions(limit int) ([]Session, error) {
	query := `
		SELECT id, port_id, title, status, 
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at
		FROM sessions
		WHERE parent_session IS NULL
		ORDER BY started_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(` LIMIT %d`, limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		var sessionType, parentSession sql.NullString
		if err := rows.Scan(
			&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
			&sessionType, &parentSession,
			&sess.StartedAt, &sess.EndedAt,
			&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
			&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		); err != nil {
			return nil, err
		}
		if sessionType.Valid {
			sess.SessionType = sessionType.String
		} else {
			sess.SessionType = TypeSingle
		}
		sess.ParentSession = parentSession
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

// IncrementCompact increments compact count for a session
func (s *Service) IncrementCompact(id string) error {
	_, err := s.db.Exec(`
		UPDATE sessions 
		SET compact_count = compact_count + 1, last_compact_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

// UpdateUsage updates token usage for a session
func (s *Service) UpdateUsage(id string, input, output, cacheRead, cacheCreate int64, cost float64) error {
	_, err := s.db.Exec(`
		UPDATE sessions 
		SET input_tokens = ?, output_tokens = ?, cache_read_tokens = ?,
		    cache_create_tokens = ?, cost_usd = ?
		WHERE id = ?
	`, input, output, cacheRead, cacheCreate, cost, id)
	return err
}
