package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "설치 상태 확인",
	Long:  `PAL Kit 설치 상태와 구성을 확인합니다.`,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// CheckResult represents a single check result
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok, warning, error
	Message string `json:"message"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var checks []CheckResult

	// 1. 시스템 정보
	checks = append(checks, CheckResult{
		Name:    "System",
		Status:  "ok",
		Message: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})

	// 2. 전역 설치 확인
	globalDir := config.GlobalDir()
	if config.IsInstalled() {
		checks = append(checks, CheckResult{
			Name:    "Global Install",
			Status:  "ok",
			Message: globalDir,
		})
	} else {
		checks = append(checks, CheckResult{
			Name:    "Global Install",
			Status:  "error",
			Message: fmt.Sprintf("설치되지 않음 - 'pal install' 실행 필요"),
		})
	}

	// 3. 전역 DB 확인
	dbPath := config.GlobalDBPath()
	if _, err := os.Stat(dbPath); err == nil {
		database, err := db.Open(dbPath)
		if err == nil {
			version, _ := database.GetVersion()
			database.Close()
			checks = append(checks, CheckResult{
				Name:    "Database",
				Status:  "ok",
				Message: fmt.Sprintf("v%d (%s)", version, dbPath),
			})
		} else {
			checks = append(checks, CheckResult{
				Name:    "Database",
				Status:  "error",
				Message: fmt.Sprintf("열기 실패: %v", err),
			})
		}
	} else {
		checks = append(checks, CheckResult{
			Name:    "Database",
			Status:  "warning",
			Message: "DB 파일 없음",
		})
	}

	// 4. 전역 에이전트 확인
	agentsDir := config.GlobalAgentsDir()
	if entries, err := os.ReadDir(agentsDir); err == nil && len(entries) > 0 {
		checks = append(checks, CheckResult{
			Name:    "Global Agents",
			Status:  "ok",
			Message: fmt.Sprintf("%d개 발견", len(entries)),
		})
	} else {
		checks = append(checks, CheckResult{
			Name:    "Global Agents",
			Status:  "warning",
			Message: "에이전트 템플릿 없음",
		})
	}

	// 5. 프로젝트 확인 (현재 디렉토리)
	projectRoot := config.FindProjectRoot()
	cwd, _ := os.Getwd()
	if projectRoot != cwd {
		settingsPath := config.ProjectSettingsPath(projectRoot)
		if _, err := os.Stat(settingsPath); err == nil {
			checks = append(checks, CheckResult{
				Name:    "Project",
				Status:  "ok",
				Message: projectRoot,
			})
		} else {
			checks = append(checks, CheckResult{
				Name:    "Project",
				Status:  "warning",
				Message: fmt.Sprintf("%s (.claude/settings.json 없음)", projectRoot),
			})
		}
	} else {
		checks = append(checks, CheckResult{
			Name:    "Project",
			Status:  "warning",
			Message: "프로젝트 디렉토리가 아님 - 'pal init' 실행 필요",
		})
	}

	// 6. Claude Code 설치 확인
	claudePaths := []string{
		"/usr/local/bin/claude",
		os.ExpandEnv("$HOME/.local/bin/claude"),
		os.ExpandEnv("$HOME/.claude/local/claude"),
	}
	claudeFound := false
	for _, p := range claudePaths {
		if _, err := os.Stat(p); err == nil {
			claudeFound = true
			checks = append(checks, CheckResult{
				Name:    "Claude Code",
				Status:  "ok",
				Message: p,
			})
			break
		}
	}
	if !claudeFound {
		checks = append(checks, CheckResult{
			Name:    "Claude Code",
			Status:  "warning",
			Message: "경로에서 찾을 수 없음 (직접 설치 확인 필요)",
		})
	}

	// 출력
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(checks)
	}

	fmt.Println("🩺 PAL Kit Doctor")
	fmt.Println()

	hasError := false
	for _, c := range checks {
		var icon string
		switch c.Status {
		case "ok":
			icon = "✅"
		case "warning":
			icon = "⚠️"
		case "error":
			icon = "❌"
			hasError = true
		}
		fmt.Printf("%s %s: %s\n", icon, c.Name, c.Message)
	}

	fmt.Println()
	if hasError {
		fmt.Println("❌ 문제가 발견되었습니다. 위 메시지를 확인하세요.")
		return fmt.Errorf("check failed")
	}
	fmt.Println("✨ 모든 검사를 통과했습니다.")

	return nil
}
