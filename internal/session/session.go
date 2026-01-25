package session

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Session type constants
const (
	TypeSingle  = "single"  // 단일 세션 (legacy)
	TypeMain    = "main"    // 메인 세션 (사용자가 직접 시작)
	TypeSub     = "sub"     // 서브 세션 (Builder가 spawn)
	TypeMulti   = "multi"   // 멀티 세션 (병렬 독립)
	TypeBuilder = "builder" // 빌더 세션 (파이프라인 관리)
)

// Event type constants
const (
	// 세션 이벤트
	EventSessionStart = "session_start" // 세션 시작
	EventSessionEnd   = "session_end"   // 세션 종료

	// 포트 이벤트
	EventPortStart = "port_start" // 포트 작업 시작
	EventPortEnd   = "port_end"   // 포트 작업 완료

	// 사용자 이벤트
	EventUserRequest = "user_request" // 사용자 요구사항 입력

	// 파일 이벤트
	EventFileEdit      = "file_edit"      // 파일 수정 (추적됨)
	EventUntrackedEdit = "untracked_edit" // 파일 수정 (추적 안됨)

	// 의사결정 이벤트
	EventDecision   = "decision"   // 주요 결정 사항
	EventEscalation = "escalation" // 에스컬레이션 발생

	// 시스템 이벤트
	EventCompact       = "compact"        // 컨텍스트 컴팩트
	EventZombieCleanup = "zombie_cleanup" // 좀비 세션 정리

	// v11: 컨텍스트 이벤트
	EventContextLoaded   = "context_loaded"   // 컨텍스트 로드 완료
	EventContextOverflow = "context_overflow" // 토큰 예산 초과

	// v11: 에이전트 이벤트
	EventAgentActivated   = "agent_activated"   // 에이전트 활성화
	EventAgentDeactivated = "agent_deactivated" // 에이전트 비활성화

	// v11: 의존성 이벤트
	EventDependencyResolved = "dependency_resolved" // 의존성 해결

	// v11: 품질 이벤트
	EventQualityWarning = "quality_warning" // 코드 품질 문제 감지

	// v11: 체크포인트 이벤트
	EventCheckpointCreated = "checkpoint_created" // 체크포인트 생성
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
	// v3 필드
	ClaudeSessionID sql.NullString
	ProjectRoot     sql.NullString
	ProjectName     sql.NullString
	TranscriptPath  sql.NullString
	Cwd             sql.NullString
	// v11 필드 - 세션 식별 강화
	TTY         sql.NullString
	ParentPID   sql.NullInt64
	Fingerprint sql.NullString
}

// SessionEvent represents a session event for history tracking
type SessionEvent struct {
	ID        int64
	SessionID string
	EventType string
	EventData string
	CreatedAt time.Time
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

// StartOptions contains options for creating a session
type StartOptions struct {
	ID              string
	PortID          string
	Title           string
	SessionType     string
	ParentSession   string
	ClaudeSessionID string
	ProjectRoot     string
	ProjectName     string
	TranscriptPath  string
	Cwd             string
	// v11 필드 - 세션 식별 강화
	TTY       string
	ParentPID int
}

// SessionIdentifier contains fields for unique session identification
type SessionIdentifier struct {
	CWD         string    `json:"cwd"`
	TTY         string    `json:"tty"`
	ParentPID   int       `json:"parent_pid"`
	StartTime   time.Time `json:"start_time"`
	Fingerprint string    `json:"fingerprint"`
}

// GenerateFingerprint creates a unique fingerprint from session identifier fields
func GenerateFingerprint(cwd, tty string, parentPID int, startTime time.Time) string {
	data := fmt.Sprintf("%s:%s:%d:%d", cwd, tty, parentPID, startTime.Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16] // 첫 16자만 사용
}

// NewSessionIdentifier creates a SessionIdentifier with auto-generated fingerprint
func NewSessionIdentifier(cwd, tty string, parentPID int) *SessionIdentifier {
	now := time.Now()
	return &SessionIdentifier{
		CWD:         cwd,
		TTY:         tty,
		ParentPID:   parentPID,
		StartTime:   now,
		Fingerprint: GenerateFingerprint(cwd, tty, parentPID, now),
	}
}

// StartWithOptions creates a new session with type and parent
func (s *Service) StartWithOptions(id, portID, title, sessionType, parentSession string) error {
	return s.StartWithFullOptions(StartOptions{
		ID:            id,
		PortID:        portID,
		Title:         title,
		SessionType:   sessionType,
		ParentSession: parentSession,
	})
}

// StartWithFullOptions creates a new session with all options
func (s *Service) StartWithFullOptions(opts StartOptions) error {
	var portIDNull, titleNull, parentNull sql.NullString
	var claudeIDNull, projRootNull, projNameNull, transcriptNull, cwdNull sql.NullString
	var ttyNull, fingerprintNull sql.NullString
	var parentPIDNull sql.NullInt64

	if opts.PortID != "" {
		// 포트 존재 여부 확인 (없으면 NULL로 처리)
		var exists int
		err := s.db.QueryRow(`SELECT 1 FROM ports WHERE id = ?`, opts.PortID).Scan(&exists)
		if err == nil {
			portIDNull = sql.NullString{String: opts.PortID, Valid: true}
		}
	}
	if opts.Title != "" {
		titleNull = sql.NullString{String: opts.Title, Valid: true}
	}
	if opts.ParentSession != "" {
		var exists int
		err := s.db.QueryRow(`SELECT 1 FROM sessions WHERE id = ?`, opts.ParentSession).Scan(&exists)
		if err != nil {
			return fmt.Errorf("상위 세션 '%s'을(를) 찾을 수 없습니다", opts.ParentSession)
		}
		parentNull = sql.NullString{String: opts.ParentSession, Valid: true}
	}
	if opts.SessionType == "" {
		opts.SessionType = TypeSingle
	}
	if opts.ClaudeSessionID != "" {
		claudeIDNull = sql.NullString{String: opts.ClaudeSessionID, Valid: true}
	}
	if opts.ProjectRoot != "" {
		projRootNull = sql.NullString{String: opts.ProjectRoot, Valid: true}
	}
	if opts.ProjectName != "" {
		projNameNull = sql.NullString{String: opts.ProjectName, Valid: true}
	}
	if opts.TranscriptPath != "" {
		transcriptNull = sql.NullString{String: opts.TranscriptPath, Valid: true}
	}
	if opts.Cwd != "" {
		cwdNull = sql.NullString{String: opts.Cwd, Valid: true}
	}
	// v11 세션 식별 강화 필드
	if opts.TTY != "" {
		ttyNull = sql.NullString{String: opts.TTY, Valid: true}
	}
	if opts.ParentPID > 0 {
		parentPIDNull = sql.NullInt64{Int64: int64(opts.ParentPID), Valid: true}
	}
	// fingerprint 자동 생성
	fingerprint := GenerateFingerprint(opts.Cwd, opts.TTY, opts.ParentPID, time.Now())
	fingerprintNull = sql.NullString{String: fingerprint, Valid: true}

	_, err := s.db.Exec(`
		INSERT INTO sessions (id, port_id, title, status, session_type, parent_session,
			claude_session_id, project_root, project_name, transcript_path, cwd,
			tty, parent_pid, fingerprint)
		VALUES (?, ?, ?, 'running', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, opts.ID, portIDNull, titleNull, opts.SessionType, parentNull,
		claudeIDNull, projRootNull, projNameNull, transcriptNull, cwdNull,
		ttyNull, parentPIDNull, fingerprintNull)

	if err != nil {
		return fmt.Errorf("세션 생성 실패: %w", err)
	}

	// 세션 시작 이벤트 로깅
	s.LogEvent(opts.ID, "session_start", fmt.Sprintf(`{"claude_session_id":"%s","project":"%s","fingerprint":"%s"}`,
		opts.ClaudeSessionID, opts.ProjectName, fingerprint))

	return nil
}

// End marks a session as ended
func (s *Service) End(id string) error {
	return s.EndWithReason(id, "")
}

// EndWithReason marks a session as ended with a reason
func (s *Service) EndWithReason(id, reason string) error {
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

	// 세션 종료 이벤트 로깅
	s.LogEvent(id, "session_end", fmt.Sprintf(`{"reason":"%s"}`, reason))

	return nil
}

// EndAllByClaudeSession closes all running sessions for a Claude session ID
func (s *Service) EndAllByClaudeSession(claudeSessionID, reason string) (int, error) {
	result, err := s.db.Exec(`
		UPDATE sessions
		SET status = 'complete', ended_at = CURRENT_TIMESTAMP
		WHERE claude_session_id = ? AND status = 'running'
	`, claudeSessionID)

	if err != nil {
		return 0, fmt.Errorf("세션 종료 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// CleanupZombieSessions closes sessions that have been running for too long
func (s *Service) CleanupZombieSessions(maxAgeHours int) (int, error) {
	result, err := s.db.Exec(`
		UPDATE sessions
		SET status = 'complete', ended_at = CURRENT_TIMESTAMP
		WHERE status = 'running'
		AND started_at < datetime('now', ? || ' hours')
	`, -maxAgeHours)

	if err != nil {
		return 0, fmt.Errorf("좀비 세션 정리 실패: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// LogEvent logs a session event
func (s *Service) LogEvent(sessionID, eventType, eventData string) error {
	_, err := s.db.Exec(`
		INSERT INTO session_events (session_id, event_type, event_data)
		VALUES (?, ?, ?)
	`, sessionID, eventType, eventData)
	return err
}

// GetEvents returns events for a session with optional type filter
func (s *Service) GetEvents(sessionID string, eventType string, limit int) ([]SessionEvent, error) {
	var args []interface{}
	query := `
		SELECT id, session_id, event_type, COALESCE(event_data, ''), created_at
		FROM session_events
		WHERE session_id = ?`
	args = append(args, sessionID)

	if eventType != "" {
		query += ` AND event_type = ?`
		args = append(args, eventType)
	}

	query += ` ORDER BY created_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SessionEvent
	for rows.Next() {
		var e SessionEvent
		if err := rows.Scan(&e.ID, &e.SessionID, &e.EventType, &e.EventData, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// GetRecentEvents returns recent events across all sessions
func (s *Service) GetRecentEvents(limit int) ([]SessionEvent, error) {
	query := `
		SELECT id, session_id, event_type, COALESCE(event_data, ''), created_at
		FROM session_events
		ORDER BY created_at DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []SessionEvent
	for rows.Next() {
		var e SessionEvent
		if err := rows.Scan(&e.ID, &e.SessionID, &e.EventType, &e.EventData, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// FindByClaudeSessionID finds a session by Claude Code session ID
func (s *Service) FindByClaudeSessionID(claudeSessionID string) (*Session, error) {
	var sess Session
	var sessionType, parentSession sql.NullString

	err := s.db.QueryRow(`
		SELECT id, port_id, title, status, 
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       claude_session_id, project_root, project_name, transcript_path, cwd
		FROM sessions WHERE claude_session_id = ? AND status = 'running'
		ORDER BY started_at DESC LIMIT 1
	`, claudeSessionID).Scan(
		&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
		&sessionType, &parentSession,
		&sess.StartedAt, &sess.EndedAt,
		&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
		&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		&sess.ClaudeSessionID, &sess.ProjectRoot, &sess.ProjectName, &sess.TranscriptPath, &sess.Cwd,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("세션을 찾을 수 없습니다")
	}
	if err != nil {
		return nil, err
	}

	if sessionType.Valid {
		sess.SessionType = sessionType.String
	} else {
		sess.SessionType = TypeSingle
	}
	sess.ParentSession = parentSession

	return &sess, nil
}

// FindActiveSession finds an active session with multiple fallback strategies
// 1. Try by Claude session ID (if provided)
// 2. Try by cwd + project_root (if provided)
// 3. Fall back to most recent running session
func (s *Service) FindActiveSession(claudeSessionID, cwd, projectRoot string) (*Session, error) {
	// Strategy 1: Claude session ID (가장 정확)
	if claudeSessionID != "" {
		sess, err := s.FindByClaudeSessionID(claudeSessionID)
		if err == nil && sess != nil {
			return sess, nil
		}
	}

	// Strategy 2: cwd + project_root 기반
	if cwd != "" || projectRoot != "" {
		sess, err := s.findByLocation(cwd, projectRoot)
		if err == nil && sess != nil {
			return sess, nil
		}
	}

	// Strategy 3: 가장 최근 running 세션
	sess, err := s.findMostRecentRunning()
	if err == nil && sess != nil {
		return sess, nil
	}

	return nil, fmt.Errorf("활성 세션을 찾을 수 없습니다")
}

// FindActiveSessionWithIdentifier finds a session using SessionIdentifier for more accurate matching
func (s *Service) FindActiveSessionWithIdentifier(claudeSessionID string, identifier *SessionIdentifier, projectRoot string) (*Session, error) {
	// Strategy 1: Claude session ID
	if claudeSessionID != "" {
		sess, err := s.FindByClaudeSessionID(claudeSessionID)
		if err == nil && sess != nil {
			return sess, nil
		}
	}

	// Strategy 2: Fingerprint (가장 정확한 로컬 식별)
	if identifier != nil && identifier.Fingerprint != "" {
		sess, err := s.FindByFingerprint(identifier.Fingerprint)
		if err == nil && sess != nil {
			return sess, nil
		}
	}

	// Strategy 3: cwd + project_root 기반
	cwd := ""
	if identifier != nil {
		cwd = identifier.CWD
	}
	if cwd != "" || projectRoot != "" {
		sess, err := s.findByLocation(cwd, projectRoot)
		if err == nil && sess != nil {
			return sess, nil
		}
	}

	// Strategy 4: 가장 최근 running 세션
	sess, err := s.findMostRecentRunning()
	if err == nil && sess != nil {
		return sess, nil
	}

	return nil, fmt.Errorf("활성 세션을 찾을 수 없습니다")
}

// FindByFingerprint finds a running session by fingerprint
func (s *Service) FindByFingerprint(fingerprint string) (*Session, error) {
	var sess Session
	var sessionType, parentSession sql.NullString

	query := `
		SELECT id, port_id, title, status,
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       claude_session_id, project_root, project_name, transcript_path, cwd,
		       tty, parent_pid, fingerprint
		FROM sessions
		WHERE status = 'running' AND fingerprint = ?
		ORDER BY started_at DESC
		LIMIT 1
	`

	err := s.db.QueryRow(query, fingerprint).Scan(
		&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
		&sessionType, &parentSession,
		&sess.StartedAt, &sess.EndedAt,
		&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
		&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		&sess.ClaudeSessionID, &sess.ProjectRoot, &sess.ProjectName, &sess.TranscriptPath, &sess.Cwd,
		&sess.TTY, &sess.ParentPID, &sess.Fingerprint,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if sessionType.Valid {
		sess.SessionType = sessionType.String
	} else {
		sess.SessionType = TypeSingle
	}
	sess.ParentSession = parentSession

	return &sess, nil
}

// findByLocation finds a running session by cwd or project_root
// Strategy: 1) Exact cwd match, 2) project_root match
func (s *Service) findByLocation(cwd, projectRoot string) (*Session, error) {
	// Strategy 1: 정확한 cwd 매칭 (가장 정확)
	if cwd != "" {
		sess, err := s.findByCwd(cwd)
		if err == nil && sess != nil {
			return sess, nil
		}
	}

	// Strategy 2: project_root 매칭 (같은 프로젝트의 다른 디렉토리)
	if projectRoot != "" {
		sess, err := s.findByProjectRoot(projectRoot)
		if err == nil && sess != nil {
			return sess, nil
		}
	}

	return nil, fmt.Errorf("해당 위치의 세션을 찾을 수 없습니다")
}

// findByCwd finds a running session with exact cwd match
func (s *Service) findByCwd(cwd string) (*Session, error) {
	var sess Session
	var sessionType, parentSession sql.NullString

	query := `
		SELECT id, port_id, title, status,
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       claude_session_id, project_root, project_name, transcript_path, cwd
		FROM sessions
		WHERE status = 'running' AND cwd = ?
		ORDER BY started_at DESC
		LIMIT 1
	`

	err := s.db.QueryRow(query, cwd).Scan(
		&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
		&sessionType, &parentSession,
		&sess.StartedAt, &sess.EndedAt,
		&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
		&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		&sess.ClaudeSessionID, &sess.ProjectRoot, &sess.ProjectName, &sess.TranscriptPath, &sess.Cwd,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if sessionType.Valid {
		sess.SessionType = sessionType.String
	} else {
		sess.SessionType = TypeSingle
	}
	sess.ParentSession = parentSession

	return &sess, nil
}

// findByProjectRoot finds a running session in the same project
func (s *Service) findByProjectRoot(projectRoot string) (*Session, error) {
	var sess Session
	var sessionType, parentSession sql.NullString

	query := `
		SELECT id, port_id, title, status,
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       claude_session_id, project_root, project_name, transcript_path, cwd
		FROM sessions
		WHERE status = 'running' AND project_root = ?
		ORDER BY started_at DESC
		LIMIT 1
	`

	err := s.db.QueryRow(query, projectRoot).Scan(
		&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
		&sessionType, &parentSession,
		&sess.StartedAt, &sess.EndedAt,
		&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
		&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		&sess.ClaudeSessionID, &sess.ProjectRoot, &sess.ProjectName, &sess.TranscriptPath, &sess.Cwd,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if sessionType.Valid {
		sess.SessionType = sessionType.String
	} else {
		sess.SessionType = TypeSingle
	}
	sess.ParentSession = parentSession

	return &sess, nil
}

// CountRunningByProject counts running sessions in a project
func (s *Service) CountRunningByProject(projectRoot string) (int, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM sessions
		WHERE status = 'running' AND project_root = ?
	`, projectRoot).Scan(&count)
	return count, err
}

// findMostRecentRunning finds the most recent running session
func (s *Service) findMostRecentRunning() (*Session, error) {
	var sess Session
	var sessionType, parentSession sql.NullString

	err := s.db.QueryRow(`
		SELECT id, port_id, title, status,
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at, jsonl_path,
		       input_tokens, output_tokens, cache_read_tokens, cache_create_tokens,
		       cost_usd, compact_count, last_compact_at,
		       claude_session_id, project_root, project_name, transcript_path, cwd
		FROM sessions
		WHERE status = 'running'
		ORDER BY started_at DESC
		LIMIT 1
	`).Scan(
		&sess.ID, &sess.PortID, &sess.Title, &sess.Status,
		&sessionType, &parentSession,
		&sess.StartedAt, &sess.EndedAt,
		&sess.JSONLPath, &sess.InputTokens, &sess.OutputTokens, &sess.CacheReadTokens,
		&sess.CacheCreateTokens, &sess.CostUSD, &sess.CompactCount, &sess.LastCompactAt,
		&sess.ClaudeSessionID, &sess.ProjectRoot, &sess.ProjectName, &sess.TranscriptPath, &sess.Cwd,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("running 세션이 없습니다")
	}
	if err != nil {
		return nil, err
	}

	if sessionType.Valid {
		sess.SessionType = sessionType.String
	} else {
		sess.SessionType = TypeSingle
	}
	sess.ParentSession = parentSession

	return &sess, nil
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

// UpdateTitle updates the session title
func (s *Service) UpdateTitle(id, title string) error {
	result, err := s.db.Exec(`
		UPDATE sessions SET title = ? WHERE id = ?
	`, title, id)

	if err != nil {
		return fmt.Errorf("타이틀 업데이트 실패: %w", err)
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

// SessionStats represents session statistics
type SessionStats struct {
	TotalSessions     int     `json:"total_sessions"`
	ActiveSessions    int     `json:"active_sessions"`
	CompletedSessions int     `json:"completed_sessions"`
	TotalInputTokens  int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalCacheRead    int64   `json:"total_cache_read_tokens"`
	TotalCacheCreate  int64   `json:"total_cache_create_tokens"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	TotalDurationSecs int64   `json:"total_duration_secs"`
	AvgDurationSecs   float64 `json:"avg_duration_secs"`
}

// GetStats returns overall session statistics
func (s *Service) GetStats() (*SessionStats, error) {
	stats := &SessionStats{}

	// Count sessions by status
	err := s.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as active,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as completed
		FROM sessions
	`).Scan(&stats.TotalSessions, &stats.ActiveSessions, &stats.CompletedSessions)
	if err != nil {
		return nil, err
	}

	// Sum token usage and cost
	err = s.db.QueryRow(`
		SELECT 
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(cache_read_tokens), 0),
			COALESCE(SUM(cache_create_tokens), 0),
			COALESCE(SUM(cost_usd), 0)
		FROM sessions
	`).Scan(&stats.TotalInputTokens, &stats.TotalOutputTokens, 
		&stats.TotalCacheRead, &stats.TotalCacheCreate, &stats.TotalCostUSD)
	if err != nil {
		return nil, err
	}

	// Calculate total and average duration for completed sessions
	err = s.db.QueryRow(`
		SELECT 
			COALESCE(SUM(CAST((julianday(ended_at) - julianday(started_at)) * 86400 AS INTEGER)), 0),
			COALESCE(AVG(CAST((julianday(ended_at) - julianday(started_at)) * 86400 AS REAL)), 0)
		FROM sessions
		WHERE ended_at IS NOT NULL
	`).Scan(&stats.TotalDurationSecs, &stats.AvgDurationSecs)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// SessionDetail represents detailed session info with duration
type SessionDetail struct {
	Session
	DurationSecs  int64  `json:"duration_secs"`
	DurationStr   string `json:"duration_str"`
	ChildrenCount int    `json:"children_count"`
}

// GetDetail returns session with computed fields
func (s *Service) GetDetail(id string) (*SessionDetail, error) {
	sess, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	detail := &SessionDetail{Session: *sess}

	// Calculate duration
	if sess.EndedAt.Valid {
		detail.DurationSecs = int64(sess.EndedAt.Time.Sub(sess.StartedAt).Seconds())
	} else {
		detail.DurationSecs = int64(time.Since(sess.StartedAt).Seconds())
	}
	detail.DurationStr = formatDuration(detail.DurationSecs)

	// Count children
	err = s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE parent_session = ?`, id).Scan(&detail.ChildrenCount)
	if err != nil {
		detail.ChildrenCount = 0
	}

	return detail, nil
}

// ListDetailed returns sessions with computed fields
func (s *Service) ListDetailed(activeOnly bool, limit int) ([]SessionDetail, error) {
	sessions, err := s.List(activeOnly, limit)
	if err != nil {
		return nil, err
	}

	details := make([]SessionDetail, len(sessions))
	for i, sess := range sessions {
		details[i] = SessionDetail{Session: sess}

		// Calculate duration
		if sess.EndedAt.Valid {
			details[i].DurationSecs = int64(sess.EndedAt.Time.Sub(sess.StartedAt).Seconds())
		} else {
			details[i].DurationSecs = int64(time.Since(sess.StartedAt).Seconds())
		}
		details[i].DurationStr = formatDuration(details[i].DurationSecs)

		// Count children
		s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE parent_session = ?`, sess.ID).Scan(&details[i].ChildrenCount)
	}

	return details, nil
}

// GetHistory returns sessions grouped by date
func (s *Service) GetHistory(days int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			DATE(started_at) as date,
			COUNT(*) as count,
			SUM(CASE WHEN status = 'complete' THEN 1 ELSE 0 END) as completed,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cost_usd), 0) as cost_usd,
			COALESCE(SUM(CAST((julianday(COALESCE(ended_at, CURRENT_TIMESTAMP)) - julianday(started_at)) * 86400 AS INTEGER)), 0) as total_duration
		FROM sessions
		WHERE started_at >= DATE('now', '-' || ? || ' days')
		GROUP BY DATE(started_at)
		ORDER BY date DESC
	`

	rows, err := s.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var date string
		var count, completed int
		var inputTokens, outputTokens, totalDuration int64
		var costUSD float64

		if err := rows.Scan(&date, &count, &completed, &inputTokens, &outputTokens, &costUSD, &totalDuration); err != nil {
			return nil, err
		}

		history = append(history, map[string]interface{}{
			"date":           date,
			"count":          count,
			"completed":      completed,
			"input_tokens":   inputTokens,
			"output_tokens":  outputTokens,
			"cost_usd":       costUSD,
			"total_duration": totalDuration,
			"duration_str":   formatDuration(totalDuration),
		})
	}

	return history, nil
}

// formatDuration formats seconds into human readable string
func formatDuration(secs int64) string {
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm %ds", secs/60, secs%60)
	}
	hours := secs / 3600
	mins := (secs % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}
