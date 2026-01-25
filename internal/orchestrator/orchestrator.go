package orchestrator

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/message"
	"github.com/n0roo/pal-kit/internal/session"
)

// OrchestrationStatus defines the status of an orchestration
type OrchestrationStatus string

const (
	StatusPending    OrchestrationStatus = "pending"
	StatusRunning    OrchestrationStatus = "running"
	StatusPaused     OrchestrationStatus = "paused"
	StatusComplete   OrchestrationStatus = "complete"
	StatusFailed     OrchestrationStatus = "failed"
	StatusCancelled  OrchestrationStatus = "cancelled"
)

// WorkerType defines the type of worker session
type WorkerType string

const (
	WorkerTypeImpl     WorkerType = "impl"
	WorkerTypeTest     WorkerType = "test"
	WorkerTypePair     WorkerType = "impl_test_pair"
	WorkerTypeSingle   WorkerType = "single"
)

// AtomicPort represents a single port in an orchestration
type AtomicPort struct {
	PortID    string   `json:"port_id"`
	Order     int      `json:"order"`
	DependsOn []string `json:"depends_on,omitempty"`
	Status    string   `json:"status,omitempty"`
}

// OrchestrationPort represents an orchestration port definition
type OrchestrationPort struct {
	ID              string        `json:"id"`
	Title           string        `json:"title"`
	Description     string        `json:"description,omitempty"`
	AtomicPorts     []AtomicPort  `json:"atomic_ports"`
	Status          OrchestrationStatus `json:"status"`
	CurrentPortID   string        `json:"current_port_id,omitempty"`
	ProgressPercent int           `json:"progress_percent"`
	CreatedAt       time.Time     `json:"created_at"`
	StartedAt       *time.Time    `json:"started_at,omitempty"`
	CompletedAt     *time.Time    `json:"completed_at,omitempty"`
}

// WorkerSession represents a worker session
type WorkerSession struct {
	ID              string     `json:"id"`
	OrchestrationID string     `json:"orchestration_id,omitempty"`
	PortID          string     `json:"port_id"`
	WorkerType      WorkerType `json:"worker_type"`
	ImplSessionID   string     `json:"impl_session_id,omitempty"`
	TestSessionID   string     `json:"test_session_id,omitempty"`
	Status          string     `json:"status"`
	Substatus       string     `json:"substatus,omitempty"`
	Result          string     `json:"result,omitempty"` // JSON
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// WorkerPairResult holds the result of a worker pair execution
type WorkerPairResult struct {
	ImplResult   interface{} `json:"impl_result,omitempty"`
	TestResult   interface{} `json:"test_result,omitempty"`
	Success      bool        `json:"success"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// Service handles orchestration operations
type Service struct {
	db             *db.DB
	sessionService *session.Service
	messageStore   *message.Store
}

// NewService creates a new orchestrator service
func NewService(database *db.DB, sessionSvc *session.Service, msgStore *message.Store) *Service {
	return &Service{
		db:             database,
		sessionService: sessionSvc,
		messageStore:   msgStore,
	}
}

// CreateOrchestration creates a new orchestration port
func (s *Service) CreateOrchestration(title, description string, atomicPorts []AtomicPort) (*OrchestrationPort, error) {
	id := uuid.New().String()
	now := time.Now()

	portsJSON, err := json.Marshal(atomicPorts)
	if err != nil {
		return nil, fmt.Errorf("포트 직렬화 실패: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO orchestration_ports (
			id, title, description, atomic_ports, status, progress_percent, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, title, description, string(portsJSON), StatusPending, 0, now)

	if err != nil {
		return nil, fmt.Errorf("Orchestration 생성 실패: %w", err)
	}

	return &OrchestrationPort{
		ID:              id,
		Title:           title,
		Description:     description,
		AtomicPorts:     atomicPorts,
		Status:          StatusPending,
		ProgressPercent: 0,
		CreatedAt:       now,
	}, nil
}

// GetOrchestration retrieves an orchestration by ID
func (s *Service) GetOrchestration(id string) (*OrchestrationPort, error) {
	var op OrchestrationPort
	var atomicPortsJSON string
	var description, currentPortID sql.NullString
	var startedAt, completedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, title, description, atomic_ports, status, current_port_id,
		       progress_percent, created_at, started_at, completed_at
		FROM orchestration_ports WHERE id = ?
	`, id).Scan(&op.ID, &op.Title, &description, &atomicPortsJSON, &op.Status,
		&currentPortID, &op.ProgressPercent, &op.CreatedAt, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Orchestration '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	if description.Valid {
		op.Description = description.String
	}
	if currentPortID.Valid {
		op.CurrentPortID = currentPortID.String
	}
	if startedAt.Valid {
		op.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		op.CompletedAt = &completedAt.Time
	}

	if err := json.Unmarshal([]byte(atomicPortsJSON), &op.AtomicPorts); err != nil {
		return nil, fmt.Errorf("포트 역직렬화 실패: %w", err)
	}

	return &op, nil
}

// StartOrchestration starts an orchestration execution
func (s *Service) StartOrchestration(id, operatorSessionID string) error {
	op, err := s.GetOrchestration(id)
	if err != nil {
		return err
	}

	if op.Status != StatusPending && op.Status != StatusPaused {
		return fmt.Errorf("시작할 수 없는 상태입니다: %s", op.Status)
	}

	now := time.Now()
	_, err = s.db.Exec(`
		UPDATE orchestration_ports 
		SET status = ?, started_at = ?
		WHERE id = ?
	`, StatusRunning, now, id)

	return err
}

// GetNextPort returns the next port to execute
func (s *Service) GetNextPort(orchestrationID string) (*AtomicPort, error) {
	op, err := s.GetOrchestration(orchestrationID)
	if err != nil {
		return nil, err
	}

	// Find first pending port with satisfied dependencies
	for i := range op.AtomicPorts {
		port := &op.AtomicPorts[i]
		if port.Status != "" && port.Status != "pending" {
			continue
		}

		// Check dependencies
		allSatisfied := true
		for _, depID := range port.DependsOn {
			for _, p := range op.AtomicPorts {
				if p.PortID == depID && p.Status != "complete" {
					allSatisfied = false
					break
				}
			}
			if !allSatisfied {
				break
			}
		}

		if allSatisfied {
			return port, nil
		}
	}

	return nil, nil // No more ports to execute
}

// UpdatePortStatus updates the status of a port in an orchestration
func (s *Service) UpdatePortStatus(orchestrationID, portID, status string) error {
	op, err := s.GetOrchestration(orchestrationID)
	if err != nil {
		return err
	}

	// Update port status
	found := false
	completedCount := 0
	for i := range op.AtomicPorts {
		if op.AtomicPorts[i].PortID == portID {
			op.AtomicPorts[i].Status = status
			found = true
		}
		if op.AtomicPorts[i].Status == "complete" {
			completedCount++
		}
	}

	if !found {
		return fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", portID)
	}

	// Calculate progress
	progress := 0
	if len(op.AtomicPorts) > 0 {
		progress = completedCount * 100 / len(op.AtomicPorts)
	}

	// Check if all complete
	orchStatus := op.Status
	if completedCount == len(op.AtomicPorts) {
		orchStatus = StatusComplete
	}

	portsJSON, _ := json.Marshal(op.AtomicPorts)

	_, err = s.db.Exec(`
		UPDATE orchestration_ports 
		SET atomic_ports = ?, current_port_id = ?, progress_percent = ?, status = ?,
		    completed_at = CASE WHEN ? = 'complete' THEN CURRENT_TIMESTAMP ELSE completed_at END
		WHERE id = ?
	`, string(portsJSON), portID, progress, orchStatus, orchStatus, orchestrationID)

	return err
}

// SpawnWorkerPair creates a Worker + Test session pair
func (s *Service) SpawnWorkerPair(opts WorkerPairOptions) (*WorkerSession, error) {
	wsID := uuid.New().String()
	now := time.Now()

	// Create Impl Worker session
	implSession, err := s.sessionService.StartHierarchical(session.HierarchyStartOptions{
		Title:       fmt.Sprintf("[Impl] %s", opts.PortTitle),
		Type:        session.TypeWorker,
		ParentID:    opts.OperatorSessionID,
		PortID:      opts.PortID,
		AgentID:     opts.ImplAgentID,
		TokenBudget: opts.TokenBudget,
		ProjectRoot: opts.ProjectRoot,
	})
	if err != nil {
		return nil, fmt.Errorf("Impl 세션 생성 실패: %w", err)
	}

	// Create Test Worker session
	testSession, err := s.sessionService.StartHierarchical(session.HierarchyStartOptions{
		Title:       fmt.Sprintf("[Test] %s", opts.PortTitle),
		Type:        session.TypeTest,
		ParentID:    opts.OperatorSessionID,
		PortID:      opts.PortID,
		AgentID:     opts.TestAgentID,
		TokenBudget: opts.TokenBudget,
		ProjectRoot: opts.ProjectRoot,
	})
	if err != nil {
		// Cleanup impl session on failure
		s.sessionService.End(implSession.ID)
		return nil, fmt.Errorf("Test 세션 생성 실패: %w", err)
	}

	// Create worker session record
	_, err = s.db.Exec(`
		INSERT INTO worker_sessions (
			id, orchestration_id, port_id, worker_type, impl_session_id, test_session_id,
			status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, wsID, opts.OrchestrationID, opts.PortID, WorkerTypePair,
		implSession.ID, testSession.ID, "running", now, now)

	if err != nil {
		return nil, fmt.Errorf("Worker 세션 기록 실패: %w", err)
	}

	ws := &WorkerSession{
		ID:              wsID,
		OrchestrationID: opts.OrchestrationID,
		PortID:          opts.PortID,
		WorkerType:      WorkerTypePair,
		ImplSessionID:   implSession.ID,
		TestSessionID:   testSession.ID,
		Status:          "running",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Send task assignment to impl worker
	if s.messageStore != nil {
		s.messageStore.SendTaskAssign(
			opts.OperatorSessionID,
			implSession.ID,
			opts.PortID,
			message.TaskAssignPayload{
				PortID:      opts.PortID,
				PortSpec:    opts.PortSpec,
				Conventions: opts.Conventions,
			},
		)
	}

	return ws, nil
}

// WorkerPairOptions contains options for spawning a worker pair
type WorkerPairOptions struct {
	OrchestrationID   string
	OperatorSessionID string
	PortID            string
	PortTitle         string
	PortSpec          string
	Conventions       []string
	ImplAgentID       string
	TestAgentID       string
	TokenBudget       int
	ProjectRoot       string
}

// SpawnSingleWorker creates a single worker session
func (s *Service) SpawnSingleWorker(opts SingleWorkerOptions) (*WorkerSession, error) {
	wsID := uuid.New().String()
	now := time.Now()

	// Determine session type
	sessionType := session.TypeWorker
	if opts.WorkerType == WorkerTypeTest {
		sessionType = session.TypeTest
	}

	// Create worker session
	workerSession, err := s.sessionService.StartHierarchical(session.HierarchyStartOptions{
		Title:       opts.Title,
		Type:        sessionType,
		ParentID:    opts.OperatorSessionID,
		PortID:      opts.PortID,
		AgentID:     opts.AgentID,
		TokenBudget: opts.TokenBudget,
		ProjectRoot: opts.ProjectRoot,
	})
	if err != nil {
		return nil, fmt.Errorf("Worker 세션 생성 실패: %w", err)
	}

	// Determine which session ID field to use
	implSessionID := ""
	testSessionID := ""
	if opts.WorkerType == WorkerTypeTest {
		testSessionID = workerSession.ID
	} else {
		implSessionID = workerSession.ID
	}

	// Create worker session record
	_, err = s.db.Exec(`
		INSERT INTO worker_sessions (
			id, orchestration_id, port_id, worker_type, impl_session_id, test_session_id,
			status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, wsID, opts.OrchestrationID, opts.PortID, WorkerTypeSingle,
		nullableString(implSessionID), nullableString(testSessionID), "running", now, now)

	if err != nil {
		return nil, fmt.Errorf("Worker 세션 기록 실패: %w", err)
	}

	ws := &WorkerSession{
		ID:              wsID,
		OrchestrationID: opts.OrchestrationID,
		PortID:          opts.PortID,
		WorkerType:      WorkerTypeSingle,
		ImplSessionID:   implSessionID,
		TestSessionID:   testSessionID,
		Status:          "running",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Send task assignment
	if s.messageStore != nil {
		targetSession := implSessionID
		if testSessionID != "" {
			targetSession = testSessionID
		}
		s.messageStore.SendTaskAssign(
			opts.OperatorSessionID,
			targetSession,
			opts.PortID,
			message.TaskAssignPayload{
				PortID:      opts.PortID,
				PortSpec:    opts.PortSpec,
				Conventions: opts.Conventions,
			},
		)
	}

	return ws, nil
}

// SingleWorkerOptions contains options for spawning a single worker
type SingleWorkerOptions struct {
	OrchestrationID   string
	OperatorSessionID string
	PortID            string
	Title             string
	PortSpec          string
	Conventions       []string
	WorkerType        WorkerType
	AgentID           string
	TokenBudget       int
	ProjectRoot       string
}

// GetWorkerSession retrieves a worker session by ID
func (s *Service) GetWorkerSession(id string) (*WorkerSession, error) {
	var ws WorkerSession
	var orchestrationID, implSessionID, testSessionID, substatus, result sql.NullString

	err := s.db.QueryRow(`
		SELECT id, orchestration_id, port_id, worker_type, impl_session_id, test_session_id,
		       status, substatus, result, created_at, updated_at
		FROM worker_sessions WHERE id = ?
	`, id).Scan(&ws.ID, &orchestrationID, &ws.PortID, &ws.WorkerType,
		&implSessionID, &testSessionID, &ws.Status, &substatus, &result,
		&ws.CreatedAt, &ws.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Worker 세션 '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	if orchestrationID.Valid {
		ws.OrchestrationID = orchestrationID.String
	}
	if implSessionID.Valid {
		ws.ImplSessionID = implSessionID.String
	}
	if testSessionID.Valid {
		ws.TestSessionID = testSessionID.String
	}
	if substatus.Valid {
		ws.Substatus = substatus.String
	}
	if result.Valid {
		ws.Result = result.String
	}

	return &ws, nil
}

// GetWorkerSessionByPort retrieves a worker session by port ID
func (s *Service) GetWorkerSessionByPort(portID string) (*WorkerSession, error) {
	var id string
	err := s.db.QueryRow(`
		SELECT id FROM worker_sessions WHERE port_id = ? ORDER BY created_at DESC LIMIT 1
	`, portID).Scan(&id)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("포트 '%s'의 Worker 세션을 찾을 수 없습니다", portID)
	}
	if err != nil {
		return nil, err
	}

	return s.GetWorkerSession(id)
}

// UpdateWorkerStatus updates the status of a worker session
func (s *Service) UpdateWorkerStatus(id, status, substatus string) error {
	_, err := s.db.Exec(`
		UPDATE worker_sessions SET status = ?, substatus = ?, updated_at = ?
		WHERE id = ?
	`, status, substatus, time.Now(), id)
	return err
}

// CompleteWorkerSession completes a worker session with result
func (s *Service) CompleteWorkerSession(id string, result WorkerPairResult) error {
	resultJSON, _ := json.Marshal(result)
	status := "complete"
	if !result.Success {
		status = "failed"
	}

	_, err := s.db.Exec(`
		UPDATE worker_sessions SET status = ?, result = ?, updated_at = ?
		WHERE id = ?
	`, status, string(resultJSON), time.Now(), id)

	if err != nil {
		return err
	}

	// End underlying sessions
	ws, err := s.GetWorkerSession(id)
	if err != nil {
		return nil // Worker session already updated
	}

	if ws.ImplSessionID != "" {
		s.sessionService.EndWithSummary(ws.ImplSessionID, status, result.ImplResult)
	}
	if ws.TestSessionID != "" {
		s.sessionService.EndWithSummary(ws.TestSessionID, status, result.TestResult)
	}

	return nil
}

// ListWorkerSessions lists worker sessions for an orchestration
func (s *Service) ListWorkerSessions(orchestrationID string) ([]*WorkerSession, error) {
	rows, err := s.db.Query(`
		SELECT id, orchestration_id, port_id, worker_type, impl_session_id, test_session_id,
		       status, substatus, result, created_at, updated_at
		FROM worker_sessions
		WHERE orchestration_id = ?
		ORDER BY created_at
	`, orchestrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*WorkerSession
	for rows.Next() {
		var ws WorkerSession
		var orchestrationID, implSessionID, testSessionID, substatus, result sql.NullString

		err := rows.Scan(&ws.ID, &orchestrationID, &ws.PortID, &ws.WorkerType,
			&implSessionID, &testSessionID, &ws.Status, &substatus, &result,
			&ws.CreatedAt, &ws.UpdatedAt)
		if err != nil {
			continue
		}

		if orchestrationID.Valid {
			ws.OrchestrationID = orchestrationID.String
		}
		if implSessionID.Valid {
			ws.ImplSessionID = implSessionID.String
		}
		if testSessionID.Valid {
			ws.TestSessionID = testSessionID.String
		}
		if substatus.Valid {
			ws.Substatus = substatus.String
		}
		if result.Valid {
			ws.Result = result.String
		}

		sessions = append(sessions, &ws)
	}

	return sessions, nil
}

// ListOrchestrations lists orchestration ports
func (s *Service) ListOrchestrations(status OrchestrationStatus, limit int) ([]*OrchestrationPort, error) {
	query := `
		SELECT id, title, description, atomic_ports, status, current_port_id,
		       progress_percent, created_at, started_at, completed_at
		FROM orchestration_ports
	`
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orchestrations []*OrchestrationPort
	for rows.Next() {
		var op OrchestrationPort
		var atomicPortsJSON string
		var description, currentPortID sql.NullString
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(&op.ID, &op.Title, &description, &atomicPortsJSON, &op.Status,
			&currentPortID, &op.ProgressPercent, &op.CreatedAt, &startedAt, &completedAt)
		if err != nil {
			continue
		}

		if description.Valid {
			op.Description = description.String
		}
		if currentPortID.Valid {
			op.CurrentPortID = currentPortID.String
		}
		if startedAt.Valid {
			op.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			op.CompletedAt = &completedAt.Time
		}

		json.Unmarshal([]byte(atomicPortsJSON), &op.AtomicPorts)
		orchestrations = append(orchestrations, &op)
	}

	return orchestrations, nil
}

// GetOrchestrationStats returns statistics for an orchestration
func (s *Service) GetOrchestrationStats(id string) (*OrchestrationStats, error) {
	op, err := s.GetOrchestration(id)
	if err != nil {
		return nil, err
	}

	stats := &OrchestrationStats{
		TotalPorts: len(op.AtomicPorts),
	}

	for _, port := range op.AtomicPorts {
		switch port.Status {
		case "complete":
			stats.CompletedPorts++
		case "running":
			stats.RunningPorts++
		case "failed":
			stats.FailedPorts++
		default:
			stats.PendingPorts++
		}
	}

	if stats.TotalPorts > 0 {
		stats.ProgressPercent = stats.CompletedPorts * 100 / stats.TotalPorts
	}

	// Get worker session stats
	workers, _ := s.ListWorkerSessions(id)
	for _, w := range workers {
		stats.TotalWorkers++
		if w.Status == "running" {
			stats.ActiveWorkers++
		}
	}

	return stats, nil
}

// OrchestrationStats represents orchestration statistics
type OrchestrationStats struct {
	TotalPorts      int `json:"total_ports"`
	PendingPorts    int `json:"pending_ports"`
	RunningPorts    int `json:"running_ports"`
	CompletedPorts  int `json:"completed_ports"`
	FailedPorts     int `json:"failed_ports"`
	ProgressPercent int `json:"progress_percent"`
	TotalWorkers    int `json:"total_workers"`
	ActiveWorkers   int `json:"active_workers"`
}

func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
