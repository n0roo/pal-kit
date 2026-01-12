package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackageLoad(t *testing.T) {
	// 임시 디렉토리 생성
	tmpDir, err := os.MkdirTemp("", "pal-pkg-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 테스트 패키지 파일 생성
	pkgDir := filepath.Join(tmpDir, "packages")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	pkgContent := `package:
  id: test-pkg
  name: Test Package
  version: "1.0.0"
  tech:
    language: go
  architecture:
    name: Clean
    layers:
      - domain
      - usecase
      - interface
  methodology:
    port_driven: true
  workers:
    - go-worker
`

	if err := os.WriteFile(filepath.Join(pkgDir, "test-pkg.yaml"), []byte(pkgContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Service 생성 및 로드
	svc := NewService(tmpDir, "")
	if err := svc.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 패키지 가져오기
	pkg, err := svc.Get("test-pkg")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// 검증
	if pkg.ID != "test-pkg" {
		t.Errorf("expected ID 'test-pkg', got '%s'", pkg.ID)
	}
	if pkg.Tech.Language != "go" {
		t.Errorf("expected language 'go', got '%s'", pkg.Tech.Language)
	}
	if len(pkg.Architecture.Layers) != 3 {
		t.Errorf("expected 3 layers, got %d", len(pkg.Architecture.Layers))
	}
	if !pkg.Methodology.PortDriven {
		t.Error("expected port_driven to be true")
	}
}

func TestPackageInheritance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-pkg-inherit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pkgDir := filepath.Join(tmpDir, "packages")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 부모 패키지
	parentContent := `package:
  id: parent-pkg
  name: Parent Package
  version: "1.0.0"
  tech:
    language: kotlin
    frameworks:
      - spring
  architecture:
    name: Layered
    layers:
      - L1
      - L2
  methodology:
    port_driven: true
    cqs: true
  workers:
    - entity-worker
    - service-worker
`

	// 자식 패키지
	childContent := `package:
  id: child-pkg
  name: Child Package
  version: "1.0.0"
  extends: parent-pkg
  tech:
    frameworks:
      - spring
      - jooq
  workers:
    - cache-worker
`

	if err := os.WriteFile(filepath.Join(pkgDir, "parent-pkg.yaml"), []byte(parentContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "child-pkg.yaml"), []byte(childContent), 0644); err != nil {
		t.Fatal(err)
	}

	svc := NewService(tmpDir, "")
	if err := svc.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	child, err := svc.Get("child-pkg")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// 상속 검증
	if child.Tech.Language != "kotlin" {
		t.Errorf("expected inherited language 'kotlin', got '%s'", child.Tech.Language)
	}

	// Workers는 부모 + 자식 병합
	if len(child.Workers) != 3 {
		t.Errorf("expected 3 workers (merged), got %d: %v", len(child.Workers), child.Workers)
	}

	// Methodology는 부모에서 상속
	if !child.Methodology.CQS {
		t.Error("expected inherited cqs to be true")
	}
}

func TestPackageValidation(t *testing.T) {
	svc := NewService("", "")

	// 유효한 패키지
	validPkg := &Package{
		ID:   "valid",
		Name: "Valid Package",
		Tech: TechConfig{
			Language: "go",
		},
		Architecture: ArchConfig{
			Name:   "Clean",
			Layers: []string{"domain", "usecase"},
		},
	}

	errors := svc.Validate(validPkg)
	if len(errors) != 0 {
		t.Errorf("expected no errors, got %v", errors)
	}

	// 유효하지 않은 패키지
	invalidPkg := &Package{
		ID: "invalid",
		// Name 누락
		Tech: TechConfig{
			// Language 누락
		},
		Architecture: ArchConfig{
			// Name 누락
			// Layers 누락
		},
	}

	errors = svc.Validate(invalidPkg)
	if len(errors) != 4 {
		t.Errorf("expected 4 errors, got %d: %v", len(errors), errors)
	}
}

func TestCircularInheritance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pal-pkg-circular-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	pkgDir := filepath.Join(tmpDir, "packages")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 순환 참조 패키지들
	pkg1 := `package:
  id: pkg1
  name: Package 1
  version: "1.0.0"
  extends: pkg2
  tech:
    language: go
  architecture:
    name: Test
    layers: [L1]
`

	pkg2 := `package:
  id: pkg2
  name: Package 2
  version: "1.0.0"
  extends: pkg1
  tech:
    language: go
  architecture:
    name: Test
    layers: [L1]
`

	if err := os.WriteFile(filepath.Join(pkgDir, "pkg1.yaml"), []byte(pkg1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "pkg2.yaml"), []byte(pkg2), 0644); err != nil {
		t.Fatal(err)
	}

	svc := NewService(tmpDir, "")
	if err := svc.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 순환 참조 감지
	_, err = svc.Get("pkg1")
	if err == nil {
		t.Error("expected circular inheritance error")
	}
}
