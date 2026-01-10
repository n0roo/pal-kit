package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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
  - .claude/pal.db        (SQLite 데이터베이스)
  - .claude/settings.json (Claude Code Hook 설정)
  - .claude/rules/        (조건부 규칙 디렉토리)
  - agents/               (에이전트 정의)
  - ports/                (포트 명세)
  - conventions/          (컨벤션)
  - CLAUDE.md             (프로젝트 컨텍스트)
`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initForce, "force", false, "기존 설정 덮어쓰기")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("현재 디렉토리 확인 실패: %w", err)
	}

	projectName := filepath.Base(cwd)
	if len(args) > 0 {
		projectName = args[0]
	}

	// 이미 초기화되었는지 확인
	palDB := filepath.Join(cwd, ".claude", "pal.db")
	if _, err := os.Stat(palDB); err == nil && !initForce {
		return fmt.Errorf("이미 초기화된 프로젝트입니다. --force 옵션으로 재초기화 가능")
	}

	var created []string

	// 1. 디렉토리 생성
	dirs := []string{
		".claude",
		".claude/rules",
		".claude/hooks",
		".claude/state",
		"agents",
		"ports",
		"conventions",
		"docs",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(cwd, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("디렉토리 생성 실패 %s: %w", dir, err)
		}
	}
	created = append(created, "디렉토리 구조")

	// 2. 데이터베이스 초기화
	database, err := db.Open(palDB)
	if err != nil {
		return fmt.Errorf("데이터베이스 생성 실패: %w", err)
	}
	database.Close()
	created = append(created, ".claude/pal.db")

	// 3. settings.json 생성
	settingsPath := filepath.Join(cwd, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) || initForce {
		if err := createSettingsJSON(settingsPath); err != nil {
			return fmt.Errorf("settings.json 생성 실패: %w", err)
		}
		created = append(created, ".claude/settings.json")
	}

	// 4. CLAUDE.md 생성
	docsSvc := docs.NewService(cwd)
	if files, err := docsSvc.InitProject(projectName); err == nil {
		created = append(created, files...)
	}

	// 5. 기본 에이전트 생성
	if err := createDefaultAgents(cwd); err != nil {
		// 실패해도 계속 진행
		fmt.Fprintf(os.Stderr, "경고: 기본 에이전트 생성 실패: %v\n", err)
	} else {
		created = append(created, "agents/builder.yaml", "agents/worker.yaml")
	}

	// 6. .gitignore 업데이트
	if err := updateGitignore(cwd); err != nil {
		// 실패해도 계속 진행
		fmt.Fprintf(os.Stderr, "경고: .gitignore 업데이트 실패: %v\n", err)
	}

	// 결과 출력
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":       "initialized",
			"project_name": projectName,
			"created":      created,
		})
	}

	fmt.Println("🚀 PAL Kit 프로젝트 초기화 완료!")
	fmt.Println()
	fmt.Printf("프로젝트: %s\n", projectName)
	fmt.Println()
	fmt.Println("생성된 항목:")
	for _, item := range created {
		fmt.Printf("  ✅ %s\n", item)
	}
	fmt.Println()
	fmt.Println("📁 디렉토리 구조:")
	fmt.Println("  .claude/")
	fmt.Println("  ├── pal.db          # 데이터베이스")
	fmt.Println("  ├── settings.json   # Claude Code Hook 설정")
	fmt.Println("  ├── rules/          # 조건부 규칙")
	fmt.Println("  └── hooks/          # Hook 스크립트")
	fmt.Println("  agents/             # 에이전트 정의")
	fmt.Println("  ports/              # 포트 명세")
	fmt.Println("  conventions/        # 컨벤션")
	fmt.Println("  CLAUDE.md           # 프로젝트 컨텍스트")
	fmt.Println()
	fmt.Println("다음 단계:")
	fmt.Println("  1. CLAUDE.md 편집하여 프로젝트 설명 추가")
	fmt.Println("  2. pal port create <id> --title \"작업명\"")
	fmt.Println("  3. pal session start --type builder")
	fmt.Println()
	fmt.Println("도움말: pal --help")

	return nil
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

// createDefaultAgents creates default agent files
func createDefaultAgents(projectRoot string) error {
	agentsDir := filepath.Join(projectRoot, "agents")

	// Builder agent
	builderContent := `agent:
  id: builder
  name: Builder Agent
  type: builder
  description: |
    요건을 분석하고, 태스크를 분해하고, 세션을 기획하는 최상위 에이전트입니다.
  prompt: |
    # Builder Agent

    당신은 빌더 에이전트입니다.

    ## 책임
    - 요건 분석 및 태스크 분해
    - 에이전트 구성 계획
    - 하위 세션 spawn 및 관리
    - 파이프라인 진행 상황 모니터링

    ## 사용 가능한 도구
    - pal session start --type sub --port <id>
    - pal pipeline create/add/plan/exec
    - pal port create/status
    - pal status

    ## 워크플로우
    1. 요건 분석 - 사용자 요구사항 이해
    2. 작업 분해 - 포트 단위로 분할
    3. 에이전트 할당 - 적합한 에이전트 선택
    4. 파이프라인 구성 - 의존성 설정
    5. 실행 및 모니터링 - 진행 상황 추적

    ## 포트 명세 작성 원칙
    - 자기완결적: 포트 문서만으로 작업 가능
    - 명확한 범위: 포함/제외 명시
    - 완료 조건: 체크리스트 형태
  tools:
    - bash
    - pal
  context:
    - CLAUDE.md
    - agents/*.yaml
    - ports/
`
	if err := os.WriteFile(filepath.Join(agentsDir, "builder.yaml"), []byte(builderContent), 0644); err != nil {
		return err
	}

	// Worker agent
	workerContent := `agent:
  id: worker
  name: Generic Worker
  type: worker
  description: |
    포트 명세에 따라 실제 작업을 수행하는 범용 워커 에이전트입니다.
  prompt: |
    # Generic Worker Agent

    당신은 워커 에이전트입니다.

    ## 책임
    - 할당된 포트 명세에 따라 작업 수행
    - 컨벤션 준수
    - 완료 조건 충족
    - 문제 발생 시 에스컬레이션

    ## 작업 시작 전
    1. 포트 명세 확인
    2. 완료 조건 체크리스트 확인
    3. 관련 컨벤션 확인

    ## 작업 중
    - 포트 범위 내에서만 작업
    - 단계별 진행 상황 기록
    - 블로커 발생 시 즉시 보고

    ## 작업 완료
    - 모든 완료 조건 체크
    - pal hook port-end <port-id>
    - 산출물 정리
  tools:
    - bash
    - editor
  context:
    - ports/{port-id}.md
    - conventions/
`
	if err := os.WriteFile(filepath.Join(agentsDir, "worker.yaml"), []byte(workerContent), 0644); err != nil {
		return err
	}

	return nil
}

// updateGitignore adds PAL Kit entries to .gitignore
func updateGitignore(projectRoot string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	entries := `
# PAL Kit
.claude/pal.db
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
