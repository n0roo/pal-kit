package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/port"
)

// ActivatePortRules creates rules file for port
func ActivatePortRules(database *db.DB, projectRoot, portID string) error {
	// Get port info
	portSvc := port.NewService(database)
	p, err := portSvc.Get(portID)
	if err != nil {
		return fmt.Errorf("포트 조회 실패: %w", err)
	}

	portTitle := portID
	if p.Title.Valid {
		portTitle = p.Title.String
	}

	// Load port spec
	portSpec := ""
	if p.FilePath.Valid && p.FilePath.String != "" {
		specPath := p.FilePath.String
		if !filepath.IsAbs(specPath) {
			specPath = filepath.Join(projectRoot, specPath)
		}
		if content, err := os.ReadFile(specPath); err == nil {
			portSpec = string(content)
		}
	}

	// Extract checklist from spec
	checklist := extractChecklistFromSpec(portSpec)

	// Create rules directory
	rulesDir := filepath.Join(projectRoot, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("rules 디렉토리 생성 실패: %w", err)
	}

	// Generate rules content
	content := generatePortRules(portID, portTitle, portSpec, checklist)

	// Write rules file
	rulePath := filepath.Join(rulesDir, portID+".md")
	if err := os.WriteFile(rulePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("rules 파일 생성 실패: %w", err)
	}

	return nil
}

// DeactivatePortRules removes rules file for port
func DeactivatePortRules(projectRoot, portID string) error {
	rulePath := filepath.Join(projectRoot, ".claude", "rules", portID+".md")
	
	// Check if file exists
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to do
	}

	if err := os.Remove(rulePath); err != nil {
		return fmt.Errorf("rules 파일 삭제 실패: %w", err)
	}

	return nil
}

// generatePortRules generates rules content for port
func generatePortRules(portID, portTitle, portSpec string, checklist []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`---
description: Port %s - %s
globs:
  - "**/*"
alwaysApply: true
---

# 포트: %s

%s

`, portID, portTitle, portTitle, getPortObjective(portSpec)))

	// Add checklist if available
	if len(checklist) > 0 {
		sb.WriteString("## 완료 체크리스트\n\n")
		for _, item := range checklist {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
		sb.WriteString("\n")
	}

	// Add conventions reminder
	sb.WriteString(`## 주의사항

- 작업 완료 시 ` + "`pal_port_end`" + ` 호출
- 빌드/테스트 실패 시 자동으로 블록 처리됨
- 문제 발생 시 ` + "`pal_escalate`" + ` 사용
`)

	return sb.String()
}

// getPortObjective extracts objective from port spec
func getPortObjective(portSpec string) string {
	lines := strings.Split(portSpec, "\n")
	inObjective := false
	var objective strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## 목표") || strings.HasPrefix(line, "## 목표") {
			inObjective = true
			continue
		}
		if inObjective {
			if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "---") {
				break
			}
			objective.WriteString(line + "\n")
		}
	}

	result := strings.TrimSpace(objective.String())
	if result == "" {
		// Try to extract from > quote at the top
		for _, line := range lines {
			if strings.HasPrefix(line, "> ") {
				result = strings.TrimPrefix(line, "> ")
				break
			}
		}
	}

	return result
}

// extractChecklistFromSpec extracts checklist from port spec
func extractChecklistFromSpec(spec string) []string {
	var checklist []string
	lines := strings.Split(spec, "\n")
	inChecklist := false

	for _, line := range lines {
		if strings.Contains(line, "체크리스트") || strings.Contains(line, "완료 기준") {
			inChecklist = true
			continue
		}
		if inChecklist {
			if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "---") {
				break
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]") {
				item := strings.TrimPrefix(line, "- [ ]")
				item = strings.TrimPrefix(item, "- [x]")
				item = strings.TrimSpace(item)
				if item != "" {
					checklist = append(checklist, item)
				}
			}
		}
	}

	return checklist
}

// SyncPortRules synchronizes rules files for all running ports
func SyncPortRules(database *db.DB, projectRoot string) error {
	portSvc := port.NewService(database)
	
	// Get running ports
	runningPorts, err := portSvc.List("running", 100)
	if err != nil {
		return fmt.Errorf("포트 목록 조회 실패: %w", err)
	}

	runningIDs := make(map[string]bool)
	for _, p := range runningPorts {
		runningIDs[p.ID] = true
	}

	// Create rules for running ports
	for _, p := range runningPorts {
		rulePath := filepath.Join(projectRoot, ".claude", "rules", p.ID+".md")
		if _, err := os.Stat(rulePath); os.IsNotExist(err) {
			if err := ActivatePortRules(database, projectRoot, p.ID); err != nil {
				return err
			}
		}
	}

	// Remove rules for non-running ports
	rulesDir := filepath.Join(projectRoot, ".claude", "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		portID := strings.TrimSuffix(name, ".md")
		if !runningIDs[portID] {
			// Check if this is a port rules file (not a general rules file)
			_, err := portSvc.Get(portID)
			if err == nil {
				// It's a port, remove if not running
				DeactivatePortRules(projectRoot, portID)
			}
		}
	}

	return nil
}
