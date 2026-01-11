package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n0roo/pal-kit/internal/config"
)

func setupTestProjectForAnalyze(t *testing.T, files map[string]string) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "pal-analyze-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("failed to write file: %v", err)
		}
	}

	return tmpDir, func() { os.RemoveAll(tmpDir) }
}

func TestAnalyzeProject_GoProject(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"go.mod":          "module example.com/test",
		"go.sum":          "",
		"main.go":         "package main",
		"internal/app.go": "package internal",
		"cmd/main.go":     "package main",
	})
	defer cleanup()

	analysis := analyzeProject(projectRoot)

	// 언어 감지 확인
	foundGo := false
	for _, lang := range analysis.TechStack.Languages {
		if lang == "Go" {
			foundGo = true
			break
		}
	}
	if !foundGo {
		t.Error("expected Go language detected")
	}

	// 빌드 도구 확인
	foundGoTool := false
	for _, tool := range analysis.TechStack.BuildTools {
		if tool == "go" {
			foundGoTool = true
			break
		}
	}
	if !foundGoTool {
		t.Error("expected 'go' build tool detected")
	}

	// 디렉토리 구조 확인
	if !analysis.Structure.HasSrc {
		t.Error("expected HasSrc true for internal/cmd dirs")
	}
}

func TestAnalyzeProject_NextJsProject(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"package.json":     `{"name": "test"}`,
		"next.config.js":   "module.exports = {}",
		"tsconfig.json":    "{}",
		"app/page.tsx":     "export default function Page() {}",
		"components/ui.tsx": "",
	})
	defer cleanup()

	analysis := analyzeProject(projectRoot)

	// TypeScript 감지
	foundTS := false
	for _, lang := range analysis.TechStack.Languages {
		if lang == "TypeScript" {
			foundTS = true
			break
		}
	}
	if !foundTS {
		t.Error("expected TypeScript detected")
	}

	// Next.js 감지
	foundNext := false
	for _, fw := range analysis.TechStack.Frameworks {
		if fw == "Next.js" {
			foundNext = true
			break
		}
	}
	if !foundNext {
		t.Error("expected Next.js framework detected")
	}
}

func TestAnalyzeProject_KotlinProject(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"build.gradle.kts": "plugins { kotlin(\"jvm\") }",
		"src/main/kotlin/App.kt": "fun main() {}",
	})
	defer cleanup()

	analysis := analyzeProject(projectRoot)

	foundKotlin := false
	for _, lang := range analysis.TechStack.Languages {
		if lang == "Kotlin" {
			foundKotlin = true
			break
		}
	}
	if !foundKotlin {
		t.Error("expected Kotlin detected")
	}

	foundGradle := false
	for _, tool := range analysis.TechStack.BuildTools {
		if tool == "gradle" {
			foundGradle = true
			break
		}
	}
	if !foundGradle {
		t.Error("expected gradle build tool detected")
	}
}

func TestAnalyzeProject_ProjectSize(t *testing.T) {
	// 작은 프로젝트 (10개 미만 파일)
	smallProject, cleanupSmall := setupTestProjectForAnalyze(t, map[string]string{
		"main.go": "package main",
		"go.mod":  "module test",
	})
	defer cleanupSmall()

	smallAnalysis := analyzeProject(smallProject)
	if smallAnalysis.Structure.EstimatedSize != "small" {
		t.Errorf("expected small project, got %s", smallAnalysis.Structure.EstimatedSize)
	}
}

func TestAnalyzeProject_ExistingConfig(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"go.mod":    "module test",
		"CLAUDE.md": "# Test",
	})
	defer cleanup()

	// PAL 설정 추가
	cfg := config.DefaultProjectConfig("test")
	cfg.Workflow.Type = config.WorkflowIntegrate
	if err := config.SaveProjectConfig(projectRoot, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	analysis := analyzeProject(projectRoot)

	if !analysis.Existing.HasClaudeMD {
		t.Error("expected HasClaudeMD true")
	}

	if !analysis.Existing.HasPalConfig {
		t.Error("expected HasPalConfig true")
	}

	if analysis.Existing.CurrentConfig == nil {
		t.Error("expected CurrentConfig not nil")
	}

	if analysis.Existing.CurrentConfig.Workflow.Type != config.WorkflowIntegrate {
		t.Error("expected workflow type integrate in existing config")
	}
}

func TestAnalyzeProject_Suggestions_Small(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"main.go": "package main",
	})
	defer cleanup()

	analysis := analyzeProject(projectRoot)

	// 작은 프로젝트는 simple 추천
	if analysis.Suggestions.WorkflowType != config.WorkflowSimple {
		t.Errorf("expected simple workflow for small project, got %s", analysis.Suggestions.WorkflowType)
	}
}

func TestAnalyzeProject_Suggestions_GoWorker(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"go.mod":           "module test",
		"main.go":          "package main",
		"internal/app.go":  "package internal",
		"internal/db.go":   "package internal",
		"internal/api.go":  "package internal",
		"cmd/server/main.go": "package main",
	})
	defer cleanup()

	// 더 많은 파일 추가하여 medium 규모로
	for i := 0; i < 50; i++ {
		path := filepath.Join(projectRoot, "pkg", "util", "file"+string(rune('a'+i%26))+".go")
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("package util"), 0644)
	}

	analysis := analyzeProject(projectRoot)

	// Go 워커 추천 확인
	foundGoWorker := false
	for _, agent := range analysis.Suggestions.RecommendedAgents {
		if agent.ID == "worker-go" && agent.Type == "worker" {
			foundGoWorker = true
			break
		}
	}
	if !foundGoWorker {
		t.Error("expected worker-go recommendation for Go project")
	}
}

func TestAnalyzeProject_Suggestions_NextWorker(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"package.json":   `{"name": "test"}`,
		"next.config.js": "module.exports = {}",
		"tsconfig.json":  "{}",
	})
	defer cleanup()

	// 더 많은 파일 추가
	for i := 0; i < 60; i++ {
		path := filepath.Join(projectRoot, "components", "Component"+string(rune('A'+i%26))+".tsx")
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte("export default function() {}"), 0644)
	}

	analysis := analyzeProject(projectRoot)

	// Next.js 워커 추천 확인
	foundNextWorker := false
	for _, agent := range analysis.Suggestions.RecommendedAgents {
		if agent.ID == "worker-next" && agent.Type == "worker" {
			foundNextWorker = true
			break
		}
	}
	if !foundNextWorker {
		t.Error("expected worker-next recommendation for Next.js project")
	}
}

func TestDetectTechStack_Indicators(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"go.mod":        "module test",
		".eslintrc.js":  "module.exports = {}",
		".prettierrc":   "{}",
		"Makefile":      "build:",
	})
	defer cleanup()

	info := detectTechStack(projectRoot)

	// Indicators 확인
	if _, ok := info.Indicators["go.mod"]; !ok {
		t.Error("expected go.mod in indicators")
	}

	if _, ok := info.Indicators[".eslintrc.js"]; !ok {
		t.Error("expected .eslintrc.js in indicators")
	}

	// 빌드 도구 확인
	expectedTools := map[string]bool{"go": true, "eslint": true, "prettier": true, "make": true}
	for _, tool := range info.BuildTools {
		if !expectedTools[tool] {
			t.Errorf("unexpected build tool: %s", tool)
		}
	}
}

func TestAnalyzeStructure_Directories(t *testing.T) {
	projectRoot, cleanup := setupTestProjectForAnalyze(t, map[string]string{
		"src/index.ts":     "",
		"tests/test.ts":    "",
		"docs/README.md":   "",
		"lib/util.ts":      "",
	})
	defer cleanup()

	structure := analyzeStructure(projectRoot)

	if !structure.HasSrc {
		t.Error("expected HasSrc true")
	}

	if !structure.HasTests {
		t.Error("expected HasTests true")
	}

	if !structure.HasDocs {
		t.Error("expected HasDocs true")
	}

	// MainDirs 확인
	expectedDirs := map[string]bool{"src": true, "tests": true, "docs": true, "lib": true}
	for _, dir := range structure.MainDirs {
		if !expectedDirs[dir] {
			t.Errorf("unexpected main dir: %s", dir)
		}
	}
}
