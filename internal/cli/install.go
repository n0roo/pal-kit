package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var (
	installForce         bool
	installImportExisting bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "PAL Kit 전역 설치",
	Long: `PAL Kit을 전역으로 설치합니다.

생성되는 항목:
  ~/.pal/
  ├── pal.db           # 통합 데이터베이스
  ├── agents/          # 전역 에이전트 템플릿
  ├── conventions/     # 전역 컨벤션
  └── templates/       # CLAUDE.md 템플릿 등

설치 후 프로젝트에서 'pal init' 명령으로 초기화할 수 있습니다.
`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&installForce, "force", false, "기존 설치 덮어쓰기")
	installCmd.Flags().BoolVar(&installImportExisting, "import-existing", false, "기존 프로젝트 DB 마이그레이션")
}

func runInstall(cmd *cobra.Command, args []string) error {
	globalDir := config.GlobalDir()

	// 1. 이미 설치되었는지 확인
	if config.IsInstalled() && !installForce {
		return fmt.Errorf("이미 설치되어 있습니다 (%s)\n--force 옵션으로 재설치 가능", globalDir)
	}

	var created []string

	// 2. 전역 디렉토리 생성
	if err := config.EnsureGlobalDirs(); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}
	created = append(created, "~/.pal/")

	// 3. 전역 DB 초기화
	dbPath := config.GlobalDBPath()
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("데이터베이스 생성 실패: %w", err)
	}
	database.Close()
	created = append(created, "~/.pal/pal.db")

	// 4. 기본 에이전트 템플릿 생성
	if err := createGlobalAgents(); err != nil {
		fmt.Fprintf(os.Stderr, "경고: 기본 에이전트 생성 실패: %v\n", err)
	} else {
		created = append(created, "~/.pal/agents/")
	}

	// 5. 기본 컨벤션 템플릿 생성
	if err := createGlobalConventions(); err != nil {
		fmt.Fprintf(os.Stderr, "경고: 기본 컨벤션 생성 실패: %v\n", err)
	} else {
		created = append(created, "~/.pal/conventions/")
	}

	// 6. CLAUDE.md 템플릿 생성
	if err := createGlobalTemplates(); err != nil {
		fmt.Fprintf(os.Stderr, "경고: 템플릿 생성 실패: %v\n", err)
	} else {
		created = append(created, "~/.pal/templates/")
	}

	// 7. 기존 프로젝트 DB 마이그레이션 (선택)
	var imported int
	if installImportExisting {
		imported = importExistingProjects(dbPath)
	}

	// 결과 출력
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":   "installed",
			"path":     globalDir,
			"created":  created,
			"imported": imported,
		})
	}

	fmt.Println("🎉 PAL Kit 전역 설치 완료!")
	fmt.Println()
	fmt.Printf("설치 경로: %s\n", globalDir)
	fmt.Println()
	fmt.Println("생성된 항목:")
	for _, item := range created {
		fmt.Printf("  ✅ %s\n", item)
	}
	if imported > 0 {
		fmt.Printf("\n📦 %d개 기존 프로젝트 마이그레이션 완료\n", imported)
	}
	fmt.Println()
	fmt.Println("다음 단계:")
	fmt.Println("  1. 프로젝트 디렉토리로 이동")
	fmt.Println("  2. pal init [project-name]")
	fmt.Println("  3. claude 실행")
	fmt.Println()
	fmt.Println("도움말: pal --help")

	return nil
}

// createGlobalAgents creates default agent templates in ~/.pal/agents/
func createGlobalAgents() error {
	agentsDir := config.GlobalAgentsDir()

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

	// Analyzer agent
	analyzerContent := `agent:
  id: analyzer
  name: Project Analyzer
  type: analyzer
  description: |
    프로젝트 구조를 분석하고 컨벤션/에이전트를 제안하는 에이전트입니다.
  prompt: |
    # Project Analyzer Agent

    당신은 프로젝트 분석 에이전트입니다.

    ## 책임
    - 프로젝트 구조 분석
    - 기술 스택 감지
    - 적합한 컨벤션 제안
    - 에이전트 구성 제안

    ## 분석 항목
    1. 언어/프레임워크 감지
       - package.json, go.mod, requirements.txt 등
    2. 프로젝트 구조 파악
       - 디렉토리 구조, 주요 파일
    3. 기존 설정 확인
       - .eslintrc, .prettierrc, tsconfig.json 등
    4. 코딩 스타일 추론
       - 기존 코드에서 패턴 추출

    ## 출력
    - conventions/*.yaml 제안
    - agents/*.yaml 제안
    - CLAUDE.md 개선 제안
  tools:
    - bash
    - read
  context:
    - .
`
	if err := os.WriteFile(filepath.Join(agentsDir, "analyzer.yaml"), []byte(analyzerContent), 0644); err != nil {
		return err
	}

	return nil
}

// createGlobalConventions creates default convention templates
func createGlobalConventions() error {
	conventionsDir := config.GlobalConventionsDir()

	// Common conventions
	commonConv := `id: common
name: Common Conventions
type: coding-style
description: 공통 코딩 컨벤션
enabled: true
priority: 5
rules:
  - id: todo-format
    description: TODO 주석 형식 준수 - TODO(담당자): 설명
    pattern: "TODO\\([\\w-]+\\):"
    severity: info
examples:
  good:
    - code: "// TODO(username): 설명"
      description: 담당자와 설명이 있는 TODO
  bad:
    - code: "// TODO: 나중에 수정"
      description: 담당자 없는 TODO
`
	if err := os.WriteFile(filepath.Join(conventionsDir, "common.yaml"), []byte(commonConv), 0644); err != nil {
		return err
	}

	return nil
}

// createGlobalTemplates creates CLAUDE.md templates
func createGlobalTemplates() error {
	templatesDir := config.GlobalTemplatesDir()

	// Default CLAUDE.md template
	claudeMD := `# {{PROJECT_NAME}}

## 개요

프로젝트 설명을 작성하세요.

## 기술 스택

- Language: 
- Framework: 
- Database: 

## 디렉토리 구조

` + "```" + `
.
├── src/
├── tests/
└── docs/
` + "```" + `

## 개발 가이드

### 빌드

` + "```bash" + `
# 빌드 명령어
` + "```" + `

### 테스트

` + "```bash" + `
# 테스트 명령어
` + "```" + `

## 컨벤션

- conventions/ 디렉토리 참조
- 또는 ` + "`pal convention list`" + ` 실행

## PAL Kit 사용법

` + "```bash" + `
# 상태 확인
pal status

# 세션 시작
pal session start --title "작업명"

# 대시보드
pal serve
` + "```" + `
`
	if err := os.WriteFile(filepath.Join(templatesDir, "CLAUDE.md"), []byte(claudeMD), 0644); err != nil {
		return err
	}

	return nil
}

// importExistingProjects imports data from existing project DBs
func importExistingProjects(dbPath string) int {
	// TODO: 기존 .claude/pal.db 파일들을 찾아서 마이그레이션
	// 현재는 placeholder
	_ = dbPath
	return 0
}
