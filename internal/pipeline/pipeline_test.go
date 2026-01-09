package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n0roo/pal-kit/internal/db"
)

func setupTestDB(t *testing.T) (*db.DB, func()) {
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

func TestCreate(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)

	err := svc.Create("pipeline-1", "Test Pipeline", "session-123")
	if err != nil {
		t.Fatalf("파이프라인 생성 실패: %v", err)
	}

	pl, err := svc.Get("pipeline-1")
	if err != nil {
		t.Fatalf("파이프라인 조회 실패: %v", err)
	}

	if pl.ID != "pipeline-1" {
		t.Errorf("ID = %s, want pipeline-1", pl.ID)
	}
	if pl.Name != "Test Pipeline" {
		t.Errorf("Name = %s, want Test Pipeline", pl.Name)
	}
	if pl.Status != "pending" {
		t.Errorf("Status = %s, want pending", pl.Status)
	}
}

func TestAddPort(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")

	err := svc.AddPort("pipeline-1", "port-001", 0)
	if err != nil {
		t.Fatalf("포트 추가 실패: %v", err)
	}

	err = svc.AddPort("pipeline-1", "port-002", 1)
	if err != nil {
		t.Fatalf("포트 추가 실패: %v", err)
	}

	ports, err := svc.GetPorts("pipeline-1")
	if err != nil {
		t.Fatalf("포트 조회 실패: %v", err)
	}

	if len(ports) != 2 {
		t.Errorf("포트 수 = %d, want 2", len(ports))
	}
}

func TestAddDependency(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)
	svc.AddPort("pipeline-1", "port-002", 1)

	err := svc.AddDependency("port-002", "port-001")
	if err != nil {
		t.Fatalf("의존성 추가 실패: %v", err)
	}

	deps, err := svc.GetDependencies("port-002")
	if err != nil {
		t.Fatalf("의존성 조회 실패: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("의존성 수 = %d, want 1", len(deps))
	}
	if deps[0] != "port-001" {
		t.Errorf("의존성 = %s, want port-001", deps[0])
	}
}

func TestGetGroups(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)
	svc.AddPort("pipeline-1", "port-002", 1)
	svc.AddPort("pipeline-1", "port-003", 1)
	svc.AddPort("pipeline-1", "port-004", 2)

	groups, err := svc.GetGroups("pipeline-1")
	if err != nil {
		t.Fatalf("그룹 조회 실패: %v", err)
	}

	if len(groups) != 3 {
		t.Errorf("그룹 수 = %d, want 3", len(groups))
	}
	if len(groups[0]) != 1 {
		t.Errorf("그룹 0 포트 수 = %d, want 1", len(groups[0]))
	}
	if len(groups[1]) != 2 {
		t.Errorf("그룹 1 포트 수 = %d, want 2", len(groups[1]))
	}
	if len(groups[2]) != 1 {
		t.Errorf("그룹 2 포트 수 = %d, want 1", len(groups[2]))
	}
}

func TestUpdateStatus(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")

	err := svc.UpdateStatus("pipeline-1", StatusRunning)
	if err != nil {
		t.Fatalf("상태 업데이트 실패: %v", err)
	}

	pl, _ := svc.Get("pipeline-1")
	if pl.Status != StatusRunning {
		t.Errorf("Status = %s, want %s", pl.Status, StatusRunning)
	}
	if !pl.StartedAt.Valid {
		t.Error("StartedAt이 설정되지 않음")
	}
}

func TestUpdatePortStatus(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)

	err := svc.UpdatePortStatus("pipeline-1", "port-001", StatusComplete)
	if err != nil {
		t.Fatalf("포트 상태 업데이트 실패: %v", err)
	}

	ports, _ := svc.GetPorts("pipeline-1")
	if ports[0].Status != StatusComplete {
		t.Errorf("포트 Status = %s, want %s", ports[0].Status, StatusComplete)
	}
}

func TestGetProgress(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)
	svc.AddPort("pipeline-1", "port-002", 1)
	svc.AddPort("pipeline-1", "port-003", 1)

	// 초기 상태
	completed, total, _ := svc.GetProgress("pipeline-1")
	if completed != 0 || total != 3 {
		t.Errorf("Progress = %d/%d, want 0/3", completed, total)
	}

	// 하나 완료
	svc.UpdatePortStatus("pipeline-1", "port-001", StatusComplete)
	completed, total, _ = svc.GetProgress("pipeline-1")
	if completed != 1 || total != 3 {
		t.Errorf("Progress = %d/%d, want 1/3", completed, total)
	}
}

func TestCanExecutePort(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)
	svc.AddPort("pipeline-1", "port-002", 1)
	svc.AddDependency("port-002", "port-001")

	// port-001은 의존성 없음 → 실행 가능
	canExec, pending, _ := svc.CanExecutePort("pipeline-1", "port-001")
	if !canExec {
		t.Error("port-001이 실행 불가능으로 판단됨")
	}
	if len(pending) != 0 {
		t.Errorf("pending = %v, want []", pending)
	}

	// port-002는 port-001 의존 → 실행 불가
	canExec, pending, _ = svc.CanExecutePort("pipeline-1", "port-002")
	if canExec {
		t.Error("port-002가 실행 가능으로 판단됨")
	}
	if len(pending) != 1 || pending[0] != "port-001" {
		t.Errorf("pending = %v, want [port-001]", pending)
	}

	// port-001 완료 후 port-002 실행 가능
	svc.UpdatePortStatus("pipeline-1", "port-001", StatusComplete)
	canExec, pending, _ = svc.CanExecutePort("pipeline-1", "port-002")
	if !canExec {
		t.Error("port-002가 실행 불가능으로 판단됨")
	}
}

func TestGetNextPorts(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)
	svc.AddPort("pipeline-1", "port-002", 1)
	svc.AddPort("pipeline-1", "port-003", 1)
	svc.AddDependency("port-002", "port-001")
	svc.AddDependency("port-003", "port-001")

	// 초기: port-001만 실행 가능
	next, _ := svc.GetNextPorts("pipeline-1")
	if len(next) != 1 || next[0] != "port-001" {
		t.Errorf("next = %v, want [port-001]", next)
	}

	// port-001 완료 후: port-002, port-003 실행 가능
	svc.UpdatePortStatus("pipeline-1", "port-001", StatusComplete)
	next, _ = svc.GetNextPorts("pipeline-1")
	if len(next) != 2 {
		t.Errorf("next 수 = %d, want 2", len(next))
	}
}

func TestBuildExecutionPlan(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)
	svc.AddPort("pipeline-1", "port-002", 1)
	svc.AddPort("pipeline-1", "port-003", 1)
	svc.AddDependency("port-002", "port-001")

	plan, err := svc.BuildExecutionPlan("pipeline-1")
	if err != nil {
		t.Fatalf("실행 계획 생성 실패: %v", err)
	}

	if plan.TotalPorts != 3 {
		t.Errorf("TotalPorts = %d, want 3", plan.TotalPorts)
	}
	if len(plan.Groups) != 2 {
		t.Errorf("Groups 수 = %d, want 2", len(plan.Groups))
	}
}

func TestIsComplete(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)
	svc.AddPort("pipeline-1", "port-002", 1)

	// 초기: 미완료
	complete, _ := svc.IsComplete("pipeline-1")
	if complete {
		t.Error("초기 상태에서 완료로 판단됨")
	}

	// 일부 완료: 미완료
	svc.UpdatePortStatus("pipeline-1", "port-001", StatusComplete)
	complete, _ = svc.IsComplete("pipeline-1")
	if complete {
		t.Error("일부 완료 상태에서 완료로 판단됨")
	}

	// 전체 완료
	svc.UpdatePortStatus("pipeline-1", "port-002", StatusComplete)
	complete, _ = svc.IsComplete("pipeline-1")
	if !complete {
		t.Error("전체 완료 상태에서 미완료로 판단됨")
	}
}

func TestDelete(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)

	err := svc.Delete("pipeline-1")
	if err != nil {
		t.Fatalf("삭제 실패: %v", err)
	}

	_, err = svc.Get("pipeline-1")
	if err == nil {
		t.Error("삭제된 파이프라인 조회가 성공함")
	}

	// 연관 포트도 삭제 확인
	ports, _ := svc.GetPorts("pipeline-1")
	if len(ports) != 0 {
		t.Error("연관 포트가 삭제되지 않음")
	}
}

func TestGenerateRunScript(t *testing.T) {
	database, cleanup := setupTestDB(t)
	defer cleanup()

	svc := NewService(database)
	svc.Create("pipeline-1", "Test", "")
	svc.AddPort("pipeline-1", "port-001", 0)
	svc.AddPort("pipeline-1", "port-002", 1)

	script, err := svc.GenerateRunScript("pipeline-1", "/tmp/project")
	if err != nil {
		t.Fatalf("스크립트 생성 실패: %v", err)
	}

	// 기본 내용 확인
	if len(script) == 0 {
		t.Error("스크립트가 비어있음")
	}

	// 포트 ID가 포함되어 있는지 확인
	if !contains(script, "port-001") || !contains(script, "port-002") {
		t.Error("스크립트에 포트 ID가 포함되지 않음")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
