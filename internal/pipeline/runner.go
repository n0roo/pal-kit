package pipeline

import (
	"fmt"
	"sort"
)

// ExecutionPlan represents a pipeline execution plan
type ExecutionPlan struct {
	PipelineID string
	Groups     []ExecutionGroup
	TotalPorts int
}

// ExecutionGroup represents a group of ports that can run in parallel
type ExecutionGroup struct {
	Order int
	Ports []PortExecution
}

// PortExecution represents a port to be executed
type PortExecution struct {
	PortID       string
	Dependencies []string
	Status       string
}

// BuildExecutionPlan creates an execution plan for a pipeline
func (s *Service) BuildExecutionPlan(pipelineID string) (*ExecutionPlan, error) {
	// 파이프라인 존재 확인
	if _, err := s.Get(pipelineID); err != nil {
		return nil, err
	}

	// 그룹별 포트 조회
	groups, err := s.GetGroups(pipelineID)
	if err != nil {
		return nil, err
	}

	// 그룹 순서 정렬
	var groupOrders []int
	for order := range groups {
		groupOrders = append(groupOrders, order)
	}
	sort.Ints(groupOrders)

	plan := &ExecutionPlan{
		PipelineID: pipelineID,
	}

	for _, order := range groupOrders {
		ports := groups[order]
		execGroup := ExecutionGroup{Order: order}

		for _, pp := range ports {
			deps, _ := s.GetDependencies(pp.PortID)
			execGroup.Ports = append(execGroup.Ports, PortExecution{
				PortID:       pp.PortID,
				Dependencies: deps,
				Status:       pp.Status,
			})
			plan.TotalPorts++
		}

		plan.Groups = append(plan.Groups, execGroup)
	}

	return plan, nil
}

// GetNextPorts returns ports ready to execute (dependencies met, not started)
func (s *Service) GetNextPorts(pipelineID string) ([]string, error) {
	ports, err := s.GetPorts(pipelineID)
	if err != nil {
		return nil, err
	}

	var ready []string
	for _, pp := range ports {
		// 이미 시작되었거나 완료된 포트는 스킵
		if pp.Status != StatusPending {
			continue
		}

		// 의존성 확인
		canExecute, _, err := s.CanExecutePort(pipelineID, pp.PortID)
		if err != nil {
			continue
		}

		if canExecute {
			ready = append(ready, pp.PortID)
		}
	}

	return ready, nil
}

// GetRunningPorts returns currently running ports
func (s *Service) GetRunningPorts(pipelineID string) ([]string, error) {
	rows, err := s.db.Query(`
		SELECT port_id FROM pipeline_ports 
		WHERE pipeline_id = ? AND status = 'running'
	`, pipelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var running []string
	for rows.Next() {
		var portID string
		if err := rows.Scan(&portID); err != nil {
			continue
		}
		running = append(running, portID)
	}
	return running, nil
}

// IsComplete checks if pipeline is complete
func (s *Service) IsComplete(pipelineID string) (bool, error) {
	var pending int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM pipeline_ports 
		WHERE pipeline_id = ? AND status NOT IN ('complete', 'skipped', 'failed')
	`, pipelineID).Scan(&pending)
	
	if err != nil {
		return false, err
	}
	return pending == 0, nil
}

// HasFailure checks if pipeline has any failed ports
func (s *Service) HasFailure(pipelineID string) (bool, error) {
	var failed int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM pipeline_ports 
		WHERE pipeline_id = ? AND status = 'failed'
	`, pipelineID).Scan(&failed)
	
	if err != nil {
		return false, err
	}
	return failed > 0, nil
}

// GenerateRunScript generates a shell script for running the pipeline
func (s *Service) GenerateRunScript(pipelineID, projectRoot string) (string, error) {
	plan, err := s.BuildExecutionPlan(pipelineID)
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`#!/bin/bash
# PAL Pipeline Runner: %s
# Generated automatically - do not edit

set -e

PIPELINE_ID="%s"
PROJECT_ROOT="%s"

cd "$PROJECT_ROOT"

echo "🚀 Starting pipeline: $PIPELINE_ID"
pal pipeline status "$PIPELINE_ID" running

`, pipelineID, pipelineID, projectRoot)

	for _, group := range plan.Groups {
		script += fmt.Sprintf("\n# ═══════════════════════════════════════\n")
		script += fmt.Sprintf("# Group %d (%d ports)\n", group.Order, len(group.Ports))
		script += fmt.Sprintf("# ═══════════════════════════════════════\n\n")

		if len(group.Ports) == 1 {
			// 단일 포트: 순차 실행
			port := group.Ports[0]
			script += s.generatePortScript(pipelineID, port.PortID)
		} else {
			// 복수 포트: 병렬 실행 (wait 사용)
			script += "# Parallel execution\n"
			script += "pids=()\n\n"
			
			for _, port := range group.Ports {
				script += fmt.Sprintf("(\n%s) &\npids+=($!)\n\n", 
					s.generatePortScript(pipelineID, port.PortID))
			}
			
			script += `# Wait for all ports in this group
for pid in "${pids[@]}"; do
    wait $pid || {
        echo "❌ A port in group failed"
        pal pipeline status "$PIPELINE_ID" failed
        exit 1
    }
done
echo "✅ Group complete"

`
		}
	}

	script += fmt.Sprintf(`
# ═══════════════════════════════════════
# Pipeline Complete
# ═══════════════════════════════════════
pal pipeline status "$PIPELINE_ID" complete
echo "🎉 Pipeline complete: $PIPELINE_ID"
`)

	return script, nil
}

func (s *Service) generatePortScript(pipelineID, portID string) string {
	return fmt.Sprintf(`echo "▶️  Starting port: %s"
pal port activate %s
pal port status %s running

# TODO: 실제 작업 실행 (claude 호출 등)
# claude --port %s

pal port status %s complete
pal port deactivate %s
echo "✅ Port complete: %s"
`, portID, portID, portID, portID, portID, portID, portID)
}

// GenerateTmuxScript generates a tmux-based parallel execution script
func (s *Service) GenerateTmuxScript(pipelineID, projectRoot, sessionName string) (string, error) {
	plan, err := s.BuildExecutionPlan(pipelineID)
	if err != nil {
		return "", err
	}

	if sessionName == "" {
		sessionName = fmt.Sprintf("pal-%s", pipelineID)
	}

	script := fmt.Sprintf(`#!/bin/bash
# PAL Pipeline Runner (tmux): %s
# Generated automatically

set -e

PIPELINE_ID="%s"
PROJECT_ROOT="%s"
TMUX_SESSION="%s"

cd "$PROJECT_ROOT"

# Kill existing session if any
tmux kill-session -t "$TMUX_SESSION" 2>/dev/null || true

# Create new session
tmux new-session -d -s "$TMUX_SESSION" -n "control"

echo "🚀 Starting pipeline: $PIPELINE_ID"
pal pipeline status "$PIPELINE_ID" running

`, pipelineID, pipelineID, projectRoot, sessionName)

	windowIndex := 1
	for _, group := range plan.Groups {
		script += fmt.Sprintf("\n# Group %d\n", group.Order)
		
		for _, port := range group.Ports {
			script += fmt.Sprintf(`
tmux new-window -t "$TMUX_SESSION:%d" -n "%s"
tmux send-keys -t "$TMUX_SESSION:%d" 'cd "%s" && pal port activate %s && pal port status %s running && echo "Ready: %s"' Enter
`, windowIndex, port.PortID, windowIndex, projectRoot, port.PortID, port.PortID, port.PortID)
			windowIndex++
		}

		// 그룹 완료 대기 (수동 또는 자동화 가능)
		script += fmt.Sprintf("\necho \"Group %d: %d windows created\"\n", group.Order, len(group.Ports))
	}

	script += fmt.Sprintf(`
echo "📺 tmux session created: $TMUX_SESSION"
echo "   Attach: tmux attach -t $TMUX_SESSION"
echo "   Windows: %d"
`, windowIndex-1)

	return script, nil
}
