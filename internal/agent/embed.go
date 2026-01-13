package agent

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/*
var templateFS embed.FS

// InstallTemplates copies embedded templates to the target directory
// targetDir should be the project root (e.g., /path/to/project)
func InstallTemplates(targetDir string) error {
	return fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 상대 경로 계산 (templates/ 제거)
		relPath, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		// 루트는 스킵
		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(targetDir, relPath)

		if d.IsDir() {
			// 디렉토리 생성
			return os.MkdirAll(targetPath, 0755)
		}

		// 파일 복사
		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("템플릿 읽기 실패 %s: %w", path, err)
		}

		// 디렉토리 확인
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		// 파일이 이미 존재하면 스킵
		if _, err := os.Stat(targetPath); err == nil {
			return nil
		}

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("템플릿 쓰기 실패 %s: %w", targetPath, err)
		}

		return nil
	})
}

// InstallTemplatesWithOverwrite copies embedded templates, overwriting existing files
func InstallTemplatesWithOverwrite(targetDir string) error {
	return fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(targetDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		content, err := templateFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("템플릿 읽기 실패 %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(targetPath, content, 0644); err != nil {
			return fmt.Errorf("템플릿 쓰기 실패 %s: %w", targetPath, err)
		}

		return nil
	})
}

// ListTemplates returns a list of all template files
func ListTemplates() ([]string, error) {
	var templates []string

	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			relPath, _ := filepath.Rel("templates", path)
			templates = append(templates, relPath)
		}
		return nil
	})

	return templates, err
}

// CountTemplates returns the number of template files
func CountTemplates() (int, error) {
	templates, err := ListTemplates()
	return len(templates), err
}

// GetTemplate returns the content of a specific template
func GetTemplate(name string) ([]byte, error) {
	path := filepath.Join("templates", name)
	return templateFS.ReadFile(path)
}

// CopyTemplateToProject copies a specific template to a project
func CopyTemplateToProject(templatePath, projectRoot string) error {
	content, err := GetTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("템플릿 읽기 실패: %w", err)
	}

	// templates/core/builder.yaml → agents/builder.yaml
	// templates/workers/backend/go.yaml → agents/worker-go.yaml
	baseName := filepath.Base(templatePath)
	dir := filepath.Dir(templatePath)

	var targetName string
	if dir == "core" {
		targetName = baseName
	} else if filepath.Base(dir) == "backend" || filepath.Base(dir) == "frontend" {
		// worker-go.yaml, worker-react.yaml 형식
		ext := filepath.Ext(baseName)
		name := baseName[:len(baseName)-len(ext)]
		targetName = "worker-" + name + ext
	} else {
		targetName = baseName
	}

	targetPath := filepath.Join(projectRoot, "agents", targetName)

	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(targetPath, content, 0644)
}
