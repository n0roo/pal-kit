package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

const (
	CurrentDBVersion = 7
)

var (
	doctorFix     bool
	doctorMigrate bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "설치 상태 확인",
	Long: `PAL Kit 설치 상태와 구성을 확인합니다.

버전 확인:
  - 앱 버전과 설치된 바이너리 버전 비교
  - DB 스키마 버전 확인
  - 마이그레이션 필요 여부 확인

옵션:
  --fix       자동으로 문제 해결 시도
  --migrate   DB 스키마 마이그레이션 실행`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "자동으로 문제 해결 시도")
	doctorCmd.Flags().BoolVar(&doctorMigrate, "migrate", false, "DB 스키마 마이그레이션 실행")
}

// CheckResult represents a single check result
type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok, warning, error
	Message string `json:"message"`
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var checks []CheckResult
	needsMigration := false
	var dbVersion int

	// 1. 시스템 정보
	checks = append(checks, CheckResult{
		Name:    "System",
		Status:  "ok",
		Message: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	})

	// 2. 앱 버전
	checks = append(checks, CheckResult{
		Name:    "App Version",
		Status:  "ok",
		Message: Version,
	})

	// 3. 설치된 바이너리 버전 확인
	installedVersion := checkInstalledVersion()
	if installedVersion != "" {
		if installedVersion == Version {
			checks = append(checks, CheckResult{
				Name:    "Installed Binary",
				Status:  "ok",
				Message: installedVersion,
			})
		} else {
			checks = append(checks, CheckResult{
				Name:    "Installed Binary",
				Status:  "warning",
				Message: fmt.Sprintf("%s (현재: %s) - 'pal install --bin' 또는 '--zsh'로 업데이트", installedVersion, Version),
			})
		}
	} else {
		checks = append(checks, CheckResult{
			Name:    "Installed Binary",
			Status:  "warning",
			Message: "PATH에서 찾을 수 없음 - 'pal install --bin' 또는 '--zsh'로 설치",
		})
	}

	// 4. 전역 설치 확인
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

	// 5. 전역 DB 확인 및 버전 비교
	dbPath := config.GlobalDBPath()
	if _, err := os.Stat(dbPath); err == nil {
		database, err := db.Open(dbPath)
		if err == nil {
			dbVersion, _ = database.GetVersion()
			database.Close()

			if dbVersion < CurrentDBVersion {
				needsMigration = true
				checks = append(checks, CheckResult{
					Name:    "Database",
					Status:  "warning",
					Message: fmt.Sprintf("v%d → v%d 마이그레이션 필요 ('pal doctor --migrate')", dbVersion, CurrentDBVersion),
				})
			} else {
				checks = append(checks, CheckResult{
					Name:    "Database",
					Status:  "ok",
					Message: fmt.Sprintf("v%d (%s)", dbVersion, dbPath),
				})
			}
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

	// Handle migration if requested
	if doctorMigrate && needsMigration {
		fmt.Println("📦 DB 마이그레이션 실행 중...")
		if err := runMigration(dbPath, dbVersion); err != nil {
			fmt.Printf("❌ 마이그레이션 실패: %v\n", err)
			return err
		}
		fmt.Printf("✅ 마이그레이션 완료: v%d → v%d\n", dbVersion, CurrentDBVersion)
		return nil
	}

	// Handle auto-fix if requested
	if doctorFix {
		fmt.Println("\n🔧 자동 수정 시도 중...")
		fixIssues(checks)
	}

	if hasError {
		fmt.Println("❌ 문제가 발견되었습니다. 위 메시지를 확인하세요.")
		return fmt.Errorf("check failed")
	}

	if needsMigration {
		fmt.Println("⚠️ DB 마이그레이션이 필요합니다.")
		fmt.Println("   'pal doctor --migrate' 실행")
		return nil
	}

	fmt.Println("✨ 모든 검사를 통과했습니다.")

	return nil
}

// checkInstalledVersion checks the version of pal binary in PATH
func checkInstalledVersion() string {
	// Check common paths
	paths := []string{
		"/usr/local/bin/pal",
		filepath.Join(os.Getenv("HOME"), ".local", "bin", "pal"),
	}

	// Also check PATH
	if palPath, err := exec.LookPath("pal"); err == nil {
		// Get version from installed binary
		out, err := exec.Command(palPath, "--version").Output()
		if err == nil {
			// Parse version from output like "pal version 0.2.0"
			version := strings.TrimSpace(string(out))
			version = strings.TrimPrefix(version, "pal version ")
			return version
		}
	}

	// Try specific paths
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			out, err := exec.Command(p, "--version").Output()
			if err == nil {
				version := strings.TrimSpace(string(out))
				version = strings.TrimPrefix(version, "pal version ")
				return version
			}
		}
	}

	return ""
}

// runMigration runs database migration
func runMigration(dbPath string, currentVersion int) error {
	// Open database - this will automatically run migrations
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	// Verify migration
	newVersion, err := database.GetVersion()
	if err != nil {
		return fmt.Errorf("버전 확인 실패: %w", err)
	}

	if newVersion < CurrentDBVersion {
		return fmt.Errorf("마이그레이션 불완전: v%d (예상: v%d)", newVersion, CurrentDBVersion)
	}

	return nil
}

// fixIssues attempts to fix common issues
func fixIssues(checks []CheckResult) {
	for _, c := range checks {
		if c.Status == "error" || c.Status == "warning" {
			switch c.Name {
			case "Global Install":
				fmt.Println("  → 'pal install' 실행 중...")
				cmd := exec.Command(os.Args[0], "install")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Printf("  ❌ 설치 실패: %v\n", err)
				}

			case "Installed Binary":
				if strings.Contains(c.Message, "PATH에서 찾을 수 없음") {
					fmt.Println("  → 바이너리 설치 중...")
					cmd := exec.Command(os.Args[0], "install", "--zsh")
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						fmt.Printf("  ❌ 바이너리 설치 실패: %v\n", err)
					}
				}

			case "Database":
				if strings.Contains(c.Message, "마이그레이션 필요") {
					fmt.Println("  → DB 마이그레이션은 '--migrate' 옵션으로 별도 실행하세요")
				}
			}
		}
	}
}
