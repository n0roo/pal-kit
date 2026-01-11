package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/docs"
	"github.com/spf13/cobra"
)

var (
	initForce bool
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "프로젝트 초기화",
	Long: `PAL Kit 프로젝트를 초기화합니다.

생성되는 항목:
  - .claude/settings.json (Claude Code Hook 설정)
  - .claude/rules/        (조건부 규칙 디렉토리)
  - agents/               (프로젝트 에이전트 - 선택적)
  - ports/                (포트 명세)
  - conventions/          (프로젝트 컨벤션 - 선택적)
  - CLAUDE.md             (프로젝트 컨텍스트)

데이터베이스는 전역 (~/.pal/pal.db)에서 관리됩니다.
먼저 'pal install'로 전역 설치가 필요합니다.
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initForce, "force", false, "기존 설정 덮어쓰기")
}

func runInit(cmd *cobra.Command, args []string) error {
	// 전역 설치 확인
	if !config.IsInstalled() {
		return fmt.Errorf("PAL Kit이 설치되지 않았습니다.\n먼저 'pal install' 명령을 실행하세요.")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("현재 디렉토리 확인 실패: %w", err)
	}

	projectName := filepath.Base(cwd)
	if len(args) > 0 {
		projectName = args[0]
	}

	// 이미 초기화되었는지 확인
	settingsPath := config.ProjectSettingsPath(cwd)
	if _, err := os.Stat(settingsPath); err == nil && !initForce {
		return fmt.Errorf("이미 초기화된 프로젝트입니다. --force 옵션으로 재초기화 가능")
	}

	var created []string

	// 1. 프로젝트 디렉토리 생성
	if err := config.EnsureProjectDirs(cwd); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}
	created = append(created, "디렉토리 구조")

	// 2. settings.json 생성 (Hook 설정)
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) || initForce {
		if err := createSettingsJSON(settingsPath); err != nil {
			return fmt.Errorf("settings.json 생성 실패: %w", err)
		}
		created = append(created, ".claude/settings.json")
	}

	// 3. CLAUDE.md 생성
	docsSvc := docs.NewService(cwd)
	if files, err := docsSvc.InitProject(projectName); err == nil {
		created = append(created, files...)
	}

	// 4. 전역 DB에 프로젝트 등록
	if err := registerProject(cwd, projectName); err != nil {
		fmt.Fprintf(os.Stderr, "경고: 프로젝트 등록 실패: %v\n", err)
	} else {
		created = append(created, "전역 DB에 프로젝트 등록")
	}

	// 5. .gitignore 업데이트
	if err := updateGitignore(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "경고: .gitignore 업데이트 실패: %v\n", err)
	}

	// 결과 출력
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":       "initialized",
			"project_name": projectName,
			"project_root": cwd,
			"created":      created,
		})
	}

	fmt.Println("🚀 PAL Kit 프로젝트 초기화 완료!")
	fmt.Println()
	fmt.Printf("프로젝트: %s\n", projectName)
	fmt.Printf("경로: %s\n", cwd)
	fmt.Println()
	fmt.Println("생성된 항목:")
	for _, item := range created {
		fmt.Printf("  ✅ %s\n", item)
	}
	fmt.Println()
	fmt.Println("📁 디렉토리 구조:")
	fmt.Println("  .claude/")
	fmt.Println("  ├── settings.json   # Claude Code Hook 설정")
	fmt.Println("  └── rules/          # 조건부 규칙")
	fmt.Println("  agents/             # 프로젝트 에이전트 (선택적)")
	fmt.Println("  ports/              # 포트 명세")
	fmt.Println("  conventions/        # 프로젝트 컨벤션 (선택적)")
	fmt.Println("  CLAUDE.md           # 프로젝트 컨텍스트")
	fmt.Println()
	fmt.Println("💡 전역 에이전트/컨벤션 확인:")
	fmt.Printf("  ls %s\n", config.GlobalAgentsDir())
	fmt.Printf("  ls %s\n", config.GlobalConventionsDir())
	fmt.Println()
	fmt.Println("다음 단계:")
	fmt.Println("  1. CLAUDE.md 편집하여 프로젝트 설명 추가")
	fmt.Println("  2. claude 실행")
	fmt.Println("  3. pal serve 로 대시보드 확인")
	fmt.Println()
	fmt.Println("도움말: pal --help")

	return nil
}

// registerProject registers a project in the global database
func registerProject(projectRoot, projectName string) error {
	database, err := db.Open(config.GlobalDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	_, err = database.Exec(`
		INSERT INTO projects (root, name, last_active, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(root) DO UPDATE SET
			name = excluded.name,
			last_active = CURRENT_TIMESTAMP
	`, projectRoot, projectName)

	return err
}

// createSettingsJSON creates Claude Code settings.json with hooks
func createSettingsJSON(path string) error {
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"SessionStart": []map[string]interface{}{
				{
					"matcher": "",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": "pal hook session-start",
						},
					},
				},
			},
			"SessionEnd": []map[string]interface{}{
				{
					"matcher": "",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": "pal hook session-end",
						},
					},
				},
			},
			"PreToolUse": []map[string]interface{}{
				{
					"matcher": "",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": "pal hook pre-tool-use",
						},
					},
				},
			},
			"PostToolUse": []map[string]interface{}{
				{
					"matcher": "",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": "pal hook post-tool-use",
						},
					},
				},
			},
			"PreCompact": []map[string]interface{}{
				{
					"matcher": "auto",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": "pal hook pre-compact",
						},
					},
				},
			},
			"Stop": []map[string]interface{}{
				{
					"matcher": "",
					"hooks": []map[string]interface{}{
						{
							"type":    "command",
							"command": "pal hook stop",
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// updateGitignore adds PAL Kit entries to .gitignore
func updateGitignore(projectRoot string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	entries := `
# PAL Kit (project-level)
.claude/state/
.claude/rules/*.md
`

	// 파일이 존재하면 추가, 없으면 새로 생성
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(entries)
	return err
}
