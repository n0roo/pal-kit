package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var (
	initGlobal  bool
	initForce   bool
	initMinimal bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "프로젝트 초기화",
	Long: `프로젝트에 PAL 구조를 초기화합니다.

.claude/ 디렉토리를 생성하고 필요한 구조를 설정합니다.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initGlobal, "global", false, "전역 설정 초기화 (~/.claude/)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "기존 설정 덮어쓰기")
	initCmd.Flags().BoolVar(&initMinimal, "minimal", false, "최소 구조만 생성")
}

func runInit(cmd *cobra.Command, args []string) error {
	var baseDir string

	if initGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("홈 디렉토리 확인 실패: %w", err)
		}
		baseDir = filepath.Join(home, ".claude")
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("현재 디렉토리 확인 실패: %w", err)
		}
		baseDir = filepath.Join(cwd, ".claude")
	}

	// 이미 존재하는지 확인
	if _, err := os.Stat(baseDir); err == nil && !initForce {
		return fmt.Errorf("%s 이미 존재합니다. --force로 덮어쓰기", baseDir)
	}

	// 디렉토리 구조 생성
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "hooks"),
		filepath.Join(baseDir, "hooks", "pre-tool-use"),
		filepath.Join(baseDir, "hooks", "post-tool-use"),
		filepath.Join(baseDir, "hooks", "stop"),
		filepath.Join(baseDir, "hooks", "pre-compact"),
		filepath.Join(baseDir, "hooks", "session-start"),
		filepath.Join(baseDir, "hooks", "session-end"),
		filepath.Join(baseDir, "agents"),
		filepath.Join(baseDir, "state"),
		filepath.Join(baseDir, "state", "locks"),
		filepath.Join(baseDir, "state", "ports"),
	}

	if initMinimal {
		dirs = []string{baseDir}
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("디렉토리 생성 실패 %s: %w", dir, err)
		}
		if verbose {
			fmt.Printf("✓ %s\n", dir)
		}
	}

	// DB 초기화
	dbPath := filepath.Join(baseDir, "pal.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	defer database.Close()

	if err := database.Init(); err != nil {
		return fmt.Errorf("DB 초기화 실패: %w", err)
	}

	if verbose {
		fmt.Printf("✓ %s (SQLite)\n", dbPath)
	}

	// settings.json 생성 (최소가 아닌 경우)
	if !initMinimal {
		settingsPath := filepath.Join(baseDir, "settings.json")
		if err := createSettingsFile(settingsPath); err != nil {
			return fmt.Errorf("settings.json 생성 실패: %w", err)
		}
		if verbose {
			fmt.Printf("✓ %s\n", settingsPath)
		}
	}

	fmt.Printf("\n✅ PAL 초기화 완료: %s\n", baseDir)
	return nil
}

func createSettingsFile(path string) error {
	content := `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "pal hook session-start"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "pal hook pre-tool-use"
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "pal hook stop"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": "auto",
        "hooks": [
          {
            "type": "command",
            "command": "pal hook pre-compact"
          }
        ]
      }
    ]
  }
}
`
	return os.WriteFile(path, []byte(content), 0644)
}
