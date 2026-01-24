package orchestrator

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/message"
)

// ExecutionState represents the state of an orchestration execution
type ExecutionState struct {
	OrchestrationID   string              `json:"orchestration_id"`
	OperatorSessionID string              `json:"operator_session_id"`
	Status            OrchestrationStatus `json:"status"`
	CurrentPortID     string              `json:"current_port_id,omitempty"`
	CompletedPorts    []string            `json:"completed_ports"`
	FailedPorts       []string            `json:"failed_ports"`
	ActiveWorkers     []string            `json:"active_workers"`
	RetryCount        map[string]int      `json:"retry_count"`
	StartedAt         time.Time           `json:"started_at"`
	LastUpdateAt      time.Time           `json:"last_update_at"`
}

// ExecutorConfig holds executor configuration
type ExecutorConfig struct {
	MaxRetries      int           `json:"max_retries"`
	RetryDelay      time.Duration `json:"retry_delay"`
	WorkerTimeout   time.Duration `json:"worker_timeout"`
	DefaultAgentIDs AgentIDs      `json:"default_agent_ids"`
}

// AgentIDs holds default agent IDs for different worker types
type AgentIDs struct {
	ImplWorker string `json:"impl_worker"`
	TestWorker string `json:"test_worker"`
}

// DefaultExecutorConfig returns default configuration
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		MaxRetries:    3,
		RetryDelay:    time.Second * 5,
		WorkerTimeout: time.Minute * 30,
		DefaultAgentIDs: AgentIDs{
			ImplWorker: "impl-worker-v1",
			TestWorker: "test-worker-v1",
		},
	}
}

// Executor handles orchestration execution
type Executor struct {
	service *Service
	config  ExecutorConfig
	states  map[string]*ExecutionState
}

// NewExecutor creates a new executor
func NewExecutor(service *Service, config ExecutorConfig) *Executor {
	return &Executor{
		service: service,
		config:  config,
		states:  make(map[string]*ExecutionState),
	}
}

// Start starts an orchestration execution
func (e *Executor) Start(orchestrationID, operatorSessionID, projectRoot string) (*ExecutionState, error) {
	op, err := e.service.GetOrchestration(orchestrationID)
	if err != nil {
		return nil, err
	}

	if err := e.service.StartOrchestration(orchestrationID, operatorSessionID); err != nil {
		return nil, err
	}

	state := &ExecutionState{
		OrchestrationID:   orchestrationID,
		OperatorSessionID: operatorSessionID,
		Status:            StatusRunning,
		RetryCount:        make(map[string]int),
		StartedAt:         time.Now(),
		LastUpdateAt:      time.Now(),
	}

	e.states[orchestrationID] = state

	// Start first batch of ports
	if err := e.processNextPorts(state, op, projectRoot); err != nil {
		return state, err
	}

	return state, nil
}

// processNextPorts processes the next available ports
func (e *Executor) processNextPorts(state *ExecutionState, op *OrchestrationPort, projectRoot string) error {
	for {
		nextPort, err := e.service.GetNextPort(state.OrchestrationID)
		if err != nil {
			return err
		}

		if nextPort == nil {
			// No more ports to process
			break
		}

		// Check if we already have an active worker for this port
		alreadyActive := false
		for _, activePort := range state.ActiveWorkers {
			if activePort == nextPort.PortID {
				alreadyActive = true
				break
			}
		}

		if alreadyActive {
			break
		}

		// Spawn worker for this port
		ws, err := e.spawnWorkerForPort(state, nextPort, projectRoot)
		if err != nil {
			// Mark port as failed
			e.service.UpdatePortStatus(state.OrchestrationID, nextPort.PortID, "failed")
			state.FailedPorts = append(state.FailedPorts, nextPort.PortID)
			continue
		}

		state.CurrentPortID = nextPort.PortID
		state.ActiveWorkers = append(state.ActiveWorkers, ws.ID)
		e.service.UpdatePortStatus(state.OrchestrationID, nextPort.PortID, "running")
	}

	state.LastUpdateAt = time.Now()
	return nil
}

// spawnWorkerForPort spawns a worker for a port
func (e *Executor) spawnWorkerForPort(state *ExecutionState, port *AtomicPort, projectRoot string) (*WorkerSession, error) {
	// Get port spec (simplified - in real implementation, load from file)
	portSpec := fmt.Sprintf("Port ID: %s", port.PortID)

	// Spawn worker pair
	return e.service.SpawnWorkerPair(WorkerPairOptions{
		OrchestrationID:   state.OrchestrationID,
		OperatorSessionID: state.OperatorSessionID,
		PortID:            port.PortID,
		PortTitle:         port.PortID,
		PortSpec:          portSpec,
		ImplAgentID:       e.config.DefaultAgentIDs.ImplWorker,
		TestAgentID:       e.config.DefaultAgentIDs.TestWorker,
		TokenBudget:       15000,
		ProjectRoot:       projectRoot,
	})
}

// HandleWorkerComplete handles worker completion
func (e *Executor) HandleWorkerComplete(workerSessionID string, result WorkerPairResult) error {
	ws, err := e.service.GetWorkerSession(workerSessionID)
	if err != nil {
		return err
	}

	state, ok := e.states[ws.OrchestrationID]
	if !ok {
		return fmt.Errorf("실행 상태를 찾을 수 없습니다: %s", ws.OrchestrationID)
	}

	// Complete worker session
	if err := e.service.CompleteWorkerSession(workerSessionID, result); err != nil {
		return err
	}

	// Remove from active workers
	for i, activeID := range state.ActiveWorkers {
		if activeID == workerSessionID {
			state.ActiveWorkers = append(state.ActiveWorkers[:i], state.ActiveWorkers[i+1:]...)
			break
		}
	}

	if result.Success {
		// Mark port as complete
		e.service.UpdatePortStatus(state.OrchestrationID, ws.PortID, "complete")
		state.CompletedPorts = append(state.CompletedPorts, ws.PortID)
	} else {
		// Check retry count
		state.RetryCount[ws.PortID]++
		if state.RetryCount[ws.PortID] >= e.config.MaxRetries {
			e.service.UpdatePortStatus(state.OrchestrationID, ws.PortID, "failed")
			state.FailedPorts = append(state.FailedPorts, ws.PortID)
		} else {
			// Reset to pending for retry
			e.service.UpdatePortStatus(state.OrchestrationID, ws.PortID, "pending")
		}
	}

	// Process next ports
	op, err := e.service.GetOrchestration(state.OrchestrationID)
	if err != nil {
		return err
	}

	return e.processNextPorts(state, op, "")
}

// HandleMessage handles incoming messages for orchestration
func (e *Executor) HandleMessage(msg *message.Message) error {
	switch msg.Subtype {
	case message.SubtypeTaskComplete:
		return e.handleTaskComplete(msg)
	case message.SubtypeTaskFailed:
		return e.handleTaskFailed(msg)
	case message.SubtypeTestPass:
		return e.handleTestPass(msg)
	case message.SubtypeTestFail:
		return e.handleTestFail(msg)
	case message.SubtypeTaskBlocked:
		return e.handleTaskBlocked(msg)
	}
	return nil
}

func (e *Executor) handleTaskComplete(msg *message.Message) error {
	ws, err := e.service.GetWorkerSessionByPort(msg.PortID)
	if err != nil {
		return err
	}

	// If this is impl complete, wait for test
	if msg.FromSession == ws.ImplSessionID {
		e.service.UpdateWorkerStatus(ws.ID, "running", "testing")

		// Notify test worker
		if e.service.messageStore != nil {
			var payload message.ImplReadyPayload
			if payloadData, ok := msg.Payload.(map[string]interface{}); ok {
				if files, ok := payloadData["files"].([]interface{}); ok {
					for _, f := range files {
						if s, ok := f.(string); ok {
							payload.Files = append(payload.Files, s)
						}
					}
				}
				if changes, ok := payloadData["changes"].(string); ok {
					payload.Changes = changes
				}
			}
			e.service.messageStore.SendImplReady(
				msg.FromSession,
				ws.TestSessionID,
				msg.PortID,
				payload,
			)
		}
	}

	return nil
}

func (e *Executor) handleTaskFailed(msg *message.Message) error {
	ws, err := e.service.GetWorkerSessionByPort(msg.PortID)
	if err != nil {
		return err
	}

	result := WorkerPairResult{
		Success:      false,
		ErrorMessage: "Task failed",
	}

	if payloadData, ok := msg.Payload.(map[string]interface{}); ok {
		if errMsg, ok := payloadData["error"].(string); ok {
			result.ErrorMessage = errMsg
		}
	}

	return e.HandleWorkerComplete(ws.ID, result)
}

func (e *Executor) handleTestPass(msg *message.Message) error {
	ws, err := e.service.GetWorkerSessionByPort(msg.PortID)
	if err != nil {
		return err
	}

	result := WorkerPairResult{
		Success: true,
	}

	if payloadData, ok := msg.Payload.(map[string]interface{}); ok {
		result.TestResult = payloadData
	}

	return e.HandleWorkerComplete(ws.ID, result)
}

func (e *Executor) handleTestFail(msg *message.Message) error {
	ws, err := e.service.GetWorkerSessionByPort(msg.PortID)
	if err != nil {
		return err
	}

	// Extract failures
	var failures []string
	if payloadData, ok := msg.Payload.(map[string]interface{}); ok {
		if f, ok := payloadData["failures"].([]interface{}); ok {
			for _, item := range f {
				if s, ok := item.(string); ok {
					failures = append(failures, s)
				}
			}
		}
	}

	// Check retry count for test
	state, ok := e.states[ws.OrchestrationID]
	if !ok {
		return fmt.Errorf("실행 상태를 찾을 수 없습니다")
	}

	retryKey := ws.PortID + ":test"
	state.RetryCount[retryKey]++

	if state.RetryCount[retryKey] >= e.config.MaxRetries {
		// Max retries exceeded - escalate
		result := WorkerPairResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("테스트 실패 %d회 - 사용자 개입 필요", state.RetryCount[retryKey]),
		}
		return e.HandleWorkerComplete(ws.ID, result)
	}

	// Send fix request to impl worker
	if e.service.messageStore != nil {
		e.service.messageStore.SendFixRequest(
			ws.TestSessionID,
			ws.ImplSessionID,
			msg.PortID,
			message.FixRequestPayload{
				Failures: failures,
			},
		)
	}

	e.service.UpdateWorkerStatus(ws.ID, "running", "fixing")
	return nil
}

func (e *Executor) handleTaskBlocked(msg *message.Message) error {
	ws, err := e.service.GetWorkerSessionByPort(msg.PortID)
	if err != nil {
		return err
	}

	e.service.UpdateWorkerStatus(ws.ID, "blocked", "dependency")
	return nil
}

// GetState returns the execution state for an orchestration
func (e *Executor) GetState(orchestrationID string) (*ExecutionState, error) {
	state, ok := e.states[orchestrationID]
	if !ok {
		return nil, fmt.Errorf("실행 상태를 찾을 수 없습니다: %s", orchestrationID)
	}
	return state, nil
}

// Pause pauses an orchestration
func (e *Executor) Pause(orchestrationID string) error {
	state, ok := e.states[orchestrationID]
	if !ok {
		return fmt.Errorf("실행 상태를 찾을 수 없습니다: %s", orchestrationID)
	}

	state.Status = StatusPaused
	_, err := e.service.db.Exec(`UPDATE orchestration_ports SET status = ? WHERE id = ?`,
		StatusPaused, orchestrationID)
	return err
}

// Resume resumes a paused orchestration
func (e *Executor) Resume(orchestrationID, projectRoot string) error {
	state, ok := e.states[orchestrationID]
	if !ok {
		return fmt.Errorf("실행 상태를 찾을 수 없습니다: %s", orchestrationID)
	}

	state.Status = StatusRunning
	_, err := e.service.db.Exec(`UPDATE orchestration_ports SET status = ? WHERE id = ?`,
		StatusRunning, orchestrationID)
	if err != nil {
		return err
	}

	op, err := e.service.GetOrchestration(orchestrationID)
	if err != nil {
		return err
	}

	return e.processNextPorts(state, op, projectRoot)
}

// Cancel cancels an orchestration
func (e *Executor) Cancel(orchestrationID string) error {
	state, ok := e.states[orchestrationID]
	if !ok {
		return fmt.Errorf("실행 상태를 찾을 수 없습니다: %s", orchestrationID)
	}

	// End all active workers
	for _, wsID := range state.ActiveWorkers {
		e.service.CompleteWorkerSession(wsID, WorkerPairResult{
			Success:      false,
			ErrorMessage: "Orchestration cancelled",
		})
	}

	state.Status = StatusCancelled
	state.ActiveWorkers = nil

	_, err := e.service.db.Exec(`
		UPDATE orchestration_ports SET status = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, StatusCancelled, orchestrationID)

	delete(e.states, orchestrationID)
	return err
}

// ExportState exports the execution state as JSON
func (e *Executor) ExportState(orchestrationID string) (string, error) {
	state, err := e.GetState(orchestrationID)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
