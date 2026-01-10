package convention

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestProject(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-conv-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestNewService(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	if svc == nil {
		t.Fatal("Service가 nil")
	}
}

func TestEnsureDir(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	if err := svc.EnsureDir(); err != nil {
		t.Fatalf("디렉토리 생성 실패: %v", err)
	}

	convDir := filepath.Join(projectRoot, "conventions")
	if _, err := os.Stat(convDir); os.IsNotExist(err) {
		t.Error("conventions 디렉토리가 생성되지 않음")
	}
}

func TestCreate(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	conv := &Convention{
		ID:          "test-conv",
		Name:        "Test Convention",
		Type:        TypeCustom,
		Description: "Test description",
		Enabled:     true,
		Priority:    5,
	}

	if err := svc.Create(conv); err != nil {
		t.Fatalf("컨벤션 생성 실패: %v", err)
	}

	// 파일 존재 확인
	filePath := filepath.Join(projectRoot, "conventions", "test-conv.yaml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("컨벤션 파일이 생성되지 않음")
	}
}

func TestGet(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 컨벤션 생성
	conv := &Convention{
		ID:      "get-test",
		Name:    "Get Test",
		Type:    TypeNaming,
		Enabled: true,
	}
	svc.Create(conv)

	// 조회
	loaded, err := svc.Get("get-test")
	if err != nil {
		t.Fatalf("컨벤션 조회 실패: %v", err)
	}

	if loaded.Name != "Get Test" {
		t.Errorf("Name = %s, want Get Test", loaded.Name)
	}
}

func TestList(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 여러 컨벤션 생성
	for i := 0; i < 3; i++ {
		conv := &Convention{
			ID:      string(rune('a' + i)),
			Name:    string(rune('A' + i)),
			Type:    TypeCustom,
			Enabled: i%2 == 0,
		}
		svc.Create(conv)
	}

	// 전체 목록
	all, err := svc.List()
	if err != nil {
		t.Fatalf("목록 조회 실패: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("컨벤션 수 = %d, want 3", len(all))
	}

	// 활성 목록
	enabled, err := svc.ListEnabled()
	if err != nil {
		t.Fatalf("활성 목록 조회 실패: %v", err)
	}

	if len(enabled) != 2 {
		t.Errorf("활성 컨벤션 수 = %d, want 2", len(enabled))
	}
}

func TestDelete(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 컨벤션 생성
	conv := &Convention{
		ID:   "delete-test",
		Name: "Delete Test",
		Type: TypeCustom,
	}
	svc.Create(conv)

	// 삭제
	if err := svc.Delete("delete-test"); err != nil {
		t.Fatalf("삭제 실패: %v", err)
	}

	// 조회 시 에러
	_, err := svc.Get("delete-test")
	if err == nil {
		t.Error("삭제된 컨벤션 조회가 성공함")
	}
}

func TestEnableDisable(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 컨벤션 생성 (비활성)
	conv := &Convention{
		ID:      "toggle-test",
		Name:    "Toggle Test",
		Type:    TypeCustom,
		Enabled: false,
	}
	svc.Create(conv)

	// 활성화
	if err := svc.Enable("toggle-test"); err != nil {
		t.Fatalf("활성화 실패: %v", err)
	}

	loaded, _ := svc.Get("toggle-test")
	if !loaded.Enabled {
		t.Error("활성화 안됨")
	}

	// 비활성화
	if err := svc.Disable("toggle-test"); err != nil {
		t.Fatalf("비활성화 실패: %v", err)
	}

	loaded, _ = svc.Get("toggle-test")
	if loaded.Enabled {
		t.Error("비활성화 안됨")
	}
}

func TestCheck(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 컨벤션 생성 (anti-pattern 검사)
	conv := &Convention{
		ID:      "check-test",
		Name:    "Check Test",
		Type:    TypeCodingStyle,
		Enabled: true,
		Rules: []Rule{
			{
				ID:          "no-todo",
				Description: "TODO 주석 금지",
				AntiPattern: `TODO`,
				Severity:    "warning",
			},
		},
	}
	svc.Create(conv)

	// 테스트 파일 생성
	testFile := filepath.Join(projectRoot, "test.go")
	os.WriteFile(testFile, []byte("// TODO: fix this"), 0644)

	// 검사
	results, err := svc.Check([]string{testFile})
	if err != nil {
		t.Fatalf("검사 실패: %v", err)
	}

	if len(results) == 0 {
		t.Error("위반 사항이 감지되지 않음")
	}
}

func TestInitDefaultConventions(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	created, err := svc.InitDefaultConventions()
	if err != nil {
		t.Fatalf("초기화 실패: %v", err)
	}

	if len(created) == 0 {
		t.Error("기본 컨벤션이 생성되지 않음")
	}

	// 목록 확인
	all, _ := svc.List()
	if len(all) != len(created) {
		t.Errorf("컨벤션 수 불일치: %d != %d", len(all), len(created))
	}
}

func TestLearn(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)

	// 테스트 파일 생성
	os.MkdirAll(filepath.Join(projectRoot, "src"), 0755)
	os.WriteFile(filepath.Join(projectRoot, "src", "user_service.go"), []byte("package src"), 0644)
	os.WriteFile(filepath.Join(projectRoot, "src", "order_handler.go"), []byte("package src"), 0644)
	os.WriteFile(filepath.Join(projectRoot, "src", "data_model.go"), []byte("package src"), 0644)

	result, err := svc.Learn([]string{projectRoot}, []string{".go"})
	if err != nil {
		t.Fatalf("학습 실패: %v", err)
	}

	if result.FilesScanned == 0 {
		t.Error("스캔된 파일이 없음")
	}

	if len(result.Patterns) == 0 {
		t.Error("발견된 패턴이 없음")
	}
}

func TestSummary(t *testing.T) {
	projectRoot, cleanup := setupTestProject(t)
	defer cleanup()

	svc := NewService(projectRoot)
	svc.InitDefaultConventions()

	summary, err := svc.Summary()
	if err != nil {
		t.Fatalf("요약 조회 실패: %v", err)
	}

	if summary["total"] == 0 {
		t.Error("총 컨벤션이 0")
	}
	if summary["rules"] == 0 {
		t.Error("총 규칙이 0")
	}
}

func TestGetConventionTypes(t *testing.T) {
	types := GetConventionTypes()
	if len(types) == 0 {
		t.Error("컨벤션 타입이 없음")
	}
}
