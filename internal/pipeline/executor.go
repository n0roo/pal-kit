package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ExecutionResult represents the result of a port execution
type ExecutionResult struct {
	PortID    string
	Success   bool
	Output    string
	Error     error
	StartedAt time.Time
	EndedAt   time.Time
	Duration  time.Duration
}

// ExecutionCallback is called when a port execution completes
type ExecutionCallback func(result ExecutionResult)

// Executor handles pipeline execution
type Executor struct {
	service     *Service
	pipelineID  string
	projectRoot string
	dryRun      bool
	verbose     bool
	parallel    bool
	onComplete  ExecutionCallback
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewExecutor creates a new pipeline executor
func NewExecutor(svc *Service, pipelineID, projectRoot string) *Executor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Executor{
		service:     svc,
		pipelineID:  pipelineID,
		projectRoot: projectRoot,
		parallel:    true,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetDryRun enables dry run mode (no actual execution)
func (e *Executor) SetDryRun(dryRun bool) *Executor {
	e.dryRun = dryRun
	return e
}

// SetVerbose enables verbose output
func (e *Executor) SetVerbose(verbose bool) *Executor {
	e.verbose = verbose
	return e
}

// SetParallel enables/disables parallel execution within groups
func (e *Executor) SetParallel(parallel bool) *Executor {
	e.parallel = parallel
	return e
}

// SetCallback sets the completion callback
func (e *Executor) SetCallback(cb ExecutionCallback) *Executor {
	e.onComplete = cb
	return e
}

// Cancel cancels the execution
func (e *Executor) Cancel() {
	e.cancel()
}

// Execute runs the pipeline
func (e *Executor) Execute() error {
	// 파이프라인 상태 업데이트
	if err := e.service.UpdateStatus(e.pipelineID, StatusRunning); err != nil {
		return fmt.Errorf("파이프라인 상태 업데이트 실패: %w", err)
	}

	// 실행 계획 조회
	plan, err := e.service.BuildExecutionPlan(e.pipelineID)
	if err != nil {
		return fmt.Errorf("실행 계획 조회 실패: %w", err)
	}

	if e.verbose {
		fmt.Printf("🚀 Pipeline: %s (%d ports in %d groups)\n",
			e.pipelineID, plan.TotalPorts, len(plan.Groups))
	}

	// 그룹별 실행
	for _, group := range plan.Groups {
		if e.verbose {
			fmt.Printf("\n═══ Group %d (%d ports) ═══\n", group.Order, len(group.Ports))
		}

		// 이미 완료된 포트 필터링
		var pendingPorts []PortExecution
		for _, port := range group.Ports {
			if port.Status == StatusPending {
				pendingPorts = append(pendingPorts, port)
			} else if e.verbose {
				fmt.Printf("⏭️  %s: already %s\n", port.PortID, port.Status)
			}
		}

		if len(pendingPorts) == 0 {
			continue
		}

		// 실행
		var results []ExecutionResult
		if e.parallel && len(pendingPorts) > 1 {
			results = e.executeParallel(pendingPorts)
		} else {
			results = e.executeSequential(pendingPorts)
		}

		// 결과 확인
		for _, result := range results {
			if !result.Success {
				e.service.UpdateStatus(e.pipelineID, StatusFailed)
				return fmt.Errorf("포트 %s 실행 실패: %v", result.PortID, result.Error)
			}
		}

		// 컨텍스트 취소 확인
		select {
		case <-e.ctx.Done():
			e.service.UpdateStatus(e.pipelineID, StatusCancelled)
			return fmt.Errorf("실행 취소됨")
		default:
		}
	}

	// 완료 상태 업데이트
	if err := e.service.UpdateStatus(e.pipelineID, StatusComplete); err != nil {
		return fmt.Errorf("완료 상태 업데이트 실패: %w", err)
	}

	if e.verbose {
		fmt.Printf("\n🎉 Pipeline complete: %s\n", e.pipelineID)
	}

	return nil
}

func (e *Executor) executeSequential(ports []PortExecution) []ExecutionResult {
	var results []ExecutionResult
	for _, port := range ports {
		result := e.executePort(port.PortID)
		results = append(results, result)
		if e.onComplete != nil {
			e.onComplete(result)
		}
		if !result.Success {
			break
		}
	}
	return results
}

func (e *Executor) executeParallel(ports []PortExecution) []ExecutionResult {
	var wg sync.WaitGroup
	results := make([]ExecutionResult, len(ports))
	
	for i, port := range ports {
		wg.Add(1)
		go func(idx int, portID string) {
			defer wg.Done()
			result := e.executePort(portID)
			results[idx] = result
			if e.onComplete != nil {
				e.onComplete(result)
			}
		}(i, port.PortID)
	}
	
	wg.Wait()
	return results
}

func (e *Executor) executePort(portID string) ExecutionResult {
	result := ExecutionResult{
		PortID:    portID,
		StartedAt: time.Now(),
	}

	if e.verbose {
		fmt.Printf("▶️  %s: starting\n", portID)
	}

	// 포트 상태 업데이트
	e.service.UpdatePortStatus(e.pipelineID, portID, StatusRunning)

	if e.dryRun {
		// Dry run: 실제 실행 없이 시뮬레이션
		time.Sleep(100 * time.Millisecond)
		result.Success = true
		result.Output = "[dry-run] simulated execution"
	} else {
		// 실제 실행
		output, err := e.runPortCommand(portID)
		result.Output = output
		result.Error = err
		result.Success = (err == nil)
	}

	result.EndedAt = time.Now()
	result.Duration = result.EndedAt.Sub(result.StartedAt)

	// 포트 상태 업데이트
	if result.Success {
		e.service.UpdatePortStatus(e.pipelineID, portID, StatusComplete)
		if e.verbose {
			fmt.Printf("✅ %s: complete (%.2fs)\n", portID, result.Duration.Seconds())
		}
	} else {
		e.service.UpdatePortStatus(e.pipelineID, portID, StatusFailed)
		if e.verbose {
			fmt.Printf("❌ %s: failed - %v\n", portID, result.Error)
		}
	}

	return result
}

func (e *Executor) runPortCommand(portID string) (string, error) {
	// 포트 명세 파일에서 실행할 명령 찾기
	// 기본: 포트 활성화 → 완료 시뮬레이션
	
	// ports/<portID>.md 파일의 ## Command 섹션 또는 기본 명령 실행
	cmdStr := fmt.Sprintf("cd %s && echo 'Executing port: %s'", e.projectRoot, portID)
	
	cmd := exec.CommandContext(e.ctx, "bash", "-c", cmdStr)
	cmd.Dir = e.projectRoot
	
	// 환경 변수 설정
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PAL_PORT_ID=%s", portID),
		fmt.Sprintf("PAL_PIPELINE_ID=%s", e.pipelineID),
		fmt.Sprintf("PAL_PROJECT_ROOT=%s", e.projectRoot),
	)

	// stdout/stderr 캡처
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	// 출력 수집
	var output string
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			line := scanner.Text()
			output += line + "\n"
			if e.verbose {
				fmt.Printf("   [%s] %s\n", portID, line)
			}
		}
	}()

	err = cmd.Wait()
	return output, err
}

// ExecuteWithScript runs a custom script for each port
func (e *Executor) ExecuteWithScript(scriptTemplate string) error {
	// 파이프라인 상태 업데이트
	if err := e.service.UpdateStatus(e.pipelineID, StatusRunning); err != nil {
		return fmt.Errorf("파이프라인 상태 업데이트 실패: %w", err)
	}

	// 실행 계획 조회
	plan, err := e.service.BuildExecutionPlan(e.pipelineID)
	if err != nil {
		return fmt.Errorf("실행 계획 조회 실패: %w", err)
	}

	// 그룹별 실행
	for _, group := range plan.Groups {
		var pendingPorts []PortExecution
		for _, port := range group.Ports {
			if port.Status == StatusPending {
				pendingPorts = append(pendingPorts, port)
			}
		}

		if len(pendingPorts) == 0 {
			continue
		}

		// 스크립트 기반 실행
		for _, port := range pendingPorts {
			result := e.executePortWithScript(port.PortID, scriptTemplate)
			if e.onComplete != nil {
				e.onComplete(result)
			}
			if !result.Success {
				e.service.UpdateStatus(e.pipelineID, StatusFailed)
				return fmt.Errorf("포트 %s 실행 실패", port.PortID)
			}
		}
	}

	e.service.UpdateStatus(e.pipelineID, StatusComplete)
	return nil
}

func (e *Executor) executePortWithScript(portID, scriptTemplate string) ExecutionResult {
	result := ExecutionResult{
		PortID:    portID,
		StartedAt: time.Now(),
	}

	e.service.UpdatePortStatus(e.pipelineID, portID, StatusRunning)

	// 스크립트 템플릿에서 변수 치환
	script := scriptTemplate
	// TODO: 변수 치환 구현

	cmd := exec.CommandContext(e.ctx, "bash", "-c", script)
	cmd.Dir = e.projectRoot
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PAL_PORT_ID=%s", portID),
		fmt.Sprintf("PAL_PIPELINE_ID=%s", e.pipelineID),
	)

	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.Error = err
	result.Success = (err == nil)
	result.EndedAt = time.Now()
	result.Duration = result.EndedAt.Sub(result.StartedAt)

	if result.Success {
		e.service.UpdatePortStatus(e.pipelineID, portID, StatusComplete)
	} else {
		e.service.UpdatePortStatus(e.pipelineID, portID, StatusFailed)
	}

	return result
}
