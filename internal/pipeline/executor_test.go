package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

func TestNewExecutor(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")

	executor := NewExecutor(svc, "test-pipeline", "/tmp/project")
	if executor == nil {
		t.Fatal("Executor가 nil입니다")
	}
}

func TestExecutor_SetOptions(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")

	executor := NewExecutor(svc, "test-pipeline", "/tmp/project")
	
	// 체이닝 테스트
	result := executor.
		SetDryRun(true).
		SetVerbose(true).
		SetParallel(false)
	
	if result != executor {
		t.Error("체이닝이 올바르게 동작하지 않음")
	}
}

func TestExecutor_DryRun(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")
	svc.AddPort("test-pipeline", "port-001", 0)
	svc.AddPort("test-pipeline", "port-002", 1)
	svc.AddDependency("port-002", "port-001")

	executor := NewExecutor(svc, "test-pipeline", "/tmp")
	executor.SetDryRun(true)
	executor.SetVerbose(false)

	err := executor.Execute()
	if err != nil {
		t.Fatalf("드라이 런 실패: %v", err)
	}

	// 파이프라인 완료 확인
	pl, _ := svc.Get("test-pipeline")
	if pl.Status != StatusComplete {
		t.Errorf("파이프라인 상태 = %s, want %s", pl.Status, StatusComplete)
	}

	// 포트 완료 확인
	ports, _ := svc.GetPorts("test-pipeline")
	for _, p := range ports {
		if p.Status != StatusComplete {
			t.Errorf("포트 %s 상태 = %s, want %s", p.PortID, p.Status, StatusComplete)
		}
	}
}

func TestExecutor_Callback(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")
	svc.AddPort("test-pipeline", "port-001", 0)
	svc.AddPort("test-pipeline", "port-002", 0)

	var results []ExecutionResult
	
	executor := NewExecutor(svc, "test-pipeline", "/tmp")
	executor.SetDryRun(true)
	executor.SetVerbose(false)
	executor.SetCallback(func(result ExecutionResult) {
		results = append(results, result)
	})

	executor.Execute()

	if len(results) != 2 {
		t.Errorf("콜백 호출 횟수 = %d, want 2", len(results))
	}

	for _, r := range results {
		if !r.Success {
			t.Errorf("포트 %s 실패", r.PortID)
		}
		if r.Duration == 0 {
			t.Errorf("포트 %s Duration이 0", r.PortID)
		}
	}
}

func TestExecutor_SequentialExecution(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")
	svc.AddPort("test-pipeline", "port-001", 0)
	svc.AddPort("test-pipeline", "port-002", 0)
	svc.AddPort("test-pipeline", "port-003", 0)

	var completionOrder []string
	
	executor := NewExecutor(svc, "test-pipeline", "/tmp")
	executor.SetDryRun(true)
	executor.SetVerbose(false)
	executor.SetParallel(false) // 순차 실행
	executor.SetCallback(func(result ExecutionResult) {
		completionOrder = append(completionOrder, result.PortID)
	})

	executor.Execute()

	// 순차 실행이므로 순서가 보장되어야 함
	if len(completionOrder) != 3 {
		t.Errorf("완료된 포트 수 = %d, want 3", len(completionOrder))
	}
}

func TestExecutor_ParallelExecution(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")
	// 같은 그룹에 3개 포트 → 병렬 실행
	svc.AddPort("test-pipeline", "port-001", 0)
	svc.AddPort("test-pipeline", "port-002", 0)
	svc.AddPort("test-pipeline", "port-003", 0)

	completedCount := 0
	
	executor := NewExecutor(svc, "test-pipeline", "/tmp")
	executor.SetDryRun(true)
	executor.SetVerbose(false)
	executor.SetParallel(true) // 병렬 실행
	executor.SetCallback(func(result ExecutionResult) {
		completedCount++
	})

	start := time.Now()
	executor.Execute()
	duration := time.Since(start)

	if completedCount != 3 {
		t.Errorf("완료된 포트 수 = %d, want 3", completedCount)
	}

	// 병렬 실행이므로 순차 실행보다 빠를 것으로 예상
	// (드라이 런에서 100ms 지연이 있으므로)
	if duration > 500*time.Millisecond {
		t.Logf("병렬 실행 시간: %v (예상보다 느림)", duration)
	}
}

func TestExecutor_GroupOrder(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")
	svc.AddPort("test-pipeline", "port-001", 0) // 그룹 0
	svc.AddPort("test-pipeline", "port-002", 1) // 그룹 1
	svc.AddPort("test-pipeline", "port-003", 2) // 그룹 2
	svc.AddDependency("port-002", "port-001")
	svc.AddDependency("port-003", "port-002")

	var completionOrder []string
	
	executor := NewExecutor(svc, "test-pipeline", "/tmp")
	executor.SetDryRun(true)
	executor.SetVerbose(false)
	executor.SetCallback(func(result ExecutionResult) {
		completionOrder = append(completionOrder, result.PortID)
	})

	executor.Execute()

	// 그룹 순서대로 실행되어야 함
	if len(completionOrder) != 3 {
		t.Fatalf("완료된 포트 수 = %d, want 3", len(completionOrder))
	}
	if completionOrder[0] != "port-001" {
		t.Errorf("첫 번째 완료 = %s, want port-001", completionOrder[0])
	}
	if completionOrder[1] != "port-002" {
		t.Errorf("두 번째 완료 = %s, want port-002", completionOrder[1])
	}
	if completionOrder[2] != "port-003" {
		t.Errorf("세 번째 완료 = %s, want port-003", completionOrder[2])
	}
}

func TestExecutor_SkipCompleted(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")
	svc.AddPort("test-pipeline", "port-001", 0)
	svc.AddPort("test-pipeline", "port-002", 1)
	
	// port-001을 이미 완료 상태로 설정
	svc.UpdatePortStatus("test-pipeline", "port-001", StatusComplete)

	completedPorts := make(map[string]bool)
	
	executor := NewExecutor(svc, "test-pipeline", "/tmp")
	executor.SetDryRun(true)
	executor.SetVerbose(false)
	executor.SetCallback(func(result ExecutionResult) {
		completedPorts[result.PortID] = true
	})

	executor.Execute()

	// port-001은 이미 완료되어 스킵되어야 함
	if completedPorts["port-001"] {
		t.Error("이미 완료된 port-001이 다시 실행됨")
	}
	if !completedPorts["port-002"] {
		t.Error("port-002가 실행되지 않음")
	}
}

func TestExecutor_Cancel(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")
	svc.AddPort("test-pipeline", "port-001", 0)
	svc.AddPort("test-pipeline", "port-002", 1)

	executor := NewExecutor(svc, "test-pipeline", "/tmp")
	executor.SetDryRun(true)
	executor.SetVerbose(false)

	// 바로 취소
	executor.Cancel()

	err := executor.Execute()
	if err == nil {
		t.Error("취소 후에도 에러가 발생하지 않음")
	}
}

func TestExecutionResult(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Second)
	
	result := ExecutionResult{
		PortID:    "test-port",
		Success:   true,
		Output:    "test output",
		StartedAt: start,
		EndedAt:   end,
	}
	result.Duration = result.EndedAt.Sub(result.StartedAt)

	if result.PortID != "test-port" {
		t.Errorf("PortID = %s, want test-port", result.PortID)
	}
	if !result.Success {
		t.Error("Success = false, want true")
	}
	// 1초 근처인지 확인 (오차 허용)
	if result.Duration < 900*time.Millisecond || result.Duration > 1100*time.Millisecond {
		t.Errorf("Duration = %v, want ~1s", result.Duration)
	}
}

// 실제 명령 실행 테스트 (통합 테스트)
func TestExecutor_RealExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("통합 테스트 스킵")
	}

	// 임시 프로젝트 디렉토리 생성
	tmpDir, err := os.MkdirTemp("", "pal-exec-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("test-pipeline", "Test", "")
	svc.AddPort("test-pipeline", "port-001", 0)

	executor := NewExecutor(svc, "test-pipeline", tmpDir)
	executor.SetDryRun(false) // 실제 실행
	executor.SetVerbose(false)

	err = executor.Execute()
	if err != nil {
		t.Fatalf("실제 실행 실패: %v", err)
	}
}

// setupTestDB는 pipeline_test.go에서 이미 정의됨
// 여기서는 사용만 함
func setupTestDBForExecutor(t *testing.T) (*db.DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("DB 열기 실패: %v", err)
	}

	if err := database.Init(); err != nil {
		database.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("DB 초기화 실패: %v", err)
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	return database, cleanup
}
