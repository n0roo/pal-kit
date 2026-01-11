package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/config"
)

// UpdateClaudeMDAfterSetup updates CLAUDE.md after setup completion
func (s *Service) UpdateClaudeMDAfterSetup(cfg *config.ProjectConfig) error {
	claudeMDPath := filepath.Join(s.projectRoot, "CLAUDE.md")

	// 기존 파일이 없으면 새로 생성
	if _, err := os.Stat(claudeMDPath); os.IsNotExist(err) {
		return s.createConfiguredClaudeMD(cfg)
	}

	// 기존 파일 읽기
	content, err := os.ReadFile(claudeMDPath)
	if err != nil {
		return fmt.Errorf("CLAUDE.md 읽기 실패: %w", err)
	}

	// 설정 상태 확인
	if strings.Contains(string(content), "pal:config:status=pending") {
		// 설정 필요 상태 → 설정 완료 상태로 업데이트
		return s.createConfiguredClaudeMD(cfg)
	}

	// 이미 설정된 상태면 스킵
	return nil
}

// createConfiguredClaudeMD creates a configured CLAUDE.md
func (s *Service) createConfiguredClaudeMD(cfg *config.ProjectConfig) error {
	claudeMDPath := filepath.Join(s.projectRoot, "CLAUDE.md")

	content := generateConfiguredClaudeMD(cfg)

	if err := os.WriteFile(claudeMDPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("CLAUDE.md 저장 실패: %w", err)
	}

	return nil
}

// generateConfiguredClaudeMD generates CLAUDE.md content for configured project
func generateConfiguredClaudeMD(cfg *config.ProjectConfig) string {
	var sb strings.Builder

	// 헤더
	sb.WriteString(fmt.Sprintf("# %s\n\n", cfg.Project.Name))
	if cfg.Project.Description != "" {
		sb.WriteString(fmt.Sprintf("> %s\n\n", cfg.Project.Description))
	}
	sb.WriteString(fmt.Sprintf("> PAL Kit %s | 워크플로우: %s\n\n", cfg.Version, cfg.Workflow.Type))
	sb.WriteString("---\n\n")

	// 프로젝트 개요 (사용자가 채울 섹션)
	sb.WriteString("## 개요\n\n")
	sb.WriteString("<!-- 프로젝트 설명을 작성하세요 -->\n\n")
	sb.WriteString("---\n\n")

	// 워크플로우 가이드
	sb.WriteString("## 워크플로우 가이드\n\n")
	sb.WriteString(fmt.Sprintf("이 프로젝트는 **%s** 워크플로우를 사용합니다.\n\n", cfg.Workflow.Type))
	sb.WriteString(getWorkflowGuide(cfg.Workflow.Type))
	sb.WriteString("\n---\n\n")

	// 에이전트 구성
	sb.WriteString("## 에이전트 구성\n\n")
	if len(cfg.Agents.Core) > 0 {
		sb.WriteString("### Core 에이전트\n")
		for _, agent := range cfg.Agents.Core {
			sb.WriteString(fmt.Sprintf("- `%s`\n", agent))
		}
		sb.WriteString("\n")
	}
	if len(cfg.Agents.Workers) > 0 {
		sb.WriteString("### Worker 에이전트\n")
		for _, agent := range cfg.Agents.Workers {
			sb.WriteString(fmt.Sprintf("- `%s`\n", agent))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("---\n\n")

	// 명령어 참조
	sb.WriteString("## PAL Kit 명령어\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 상태 확인\n")
	sb.WriteString("pal status\n\n")
	sb.WriteString("# 포트 관리\n")
	sb.WriteString("pal port list\n")
	sb.WriteString("pal port create <id> --title \"작업명\"\n\n")
	sb.WriteString("# 작업 시작/종료\n")
	sb.WriteString("pal hook port-start <id>\n")
	sb.WriteString("pal hook port-end <id>\n\n")

	if cfg.Workflow.Type == config.WorkflowIntegrate || cfg.Workflow.Type == config.WorkflowMulti {
		sb.WriteString("# 파이프라인\n")
		sb.WriteString("pal pipeline list\n")
		sb.WriteString("pal pl plan <n>\n\n")
	}

	sb.WriteString("# 대시보드\n")
	sb.WriteString("pal serve\n")
	sb.WriteString("```\n\n")
	sb.WriteString("---\n\n")

	// 디렉토리 구조
	sb.WriteString("## 디렉토리 구조\n\n")
	sb.WriteString("```\n")
	sb.WriteString(".\n")
	sb.WriteString("├── CLAUDE.md           # 이 파일\n")
	sb.WriteString("├── agents/             # 에이전트 정의\n")
	sb.WriteString("├── ports/              # 포트 명세\n")
	sb.WriteString("├── conventions/        # 컨벤션 문서\n")
	sb.WriteString("├── .claude/\n")
	sb.WriteString("│   ├── settings.json   # Claude Code Hook 설정\n")
	sb.WriteString("│   └── rules/          # 조건부 규칙\n")
	sb.WriteString("└── .pal/\n")
	sb.WriteString("    └── config.yaml     # PAL Kit 설정\n")
	sb.WriteString("```\n\n")
	sb.WriteString("---\n\n")

	// 메타데이터
	sb.WriteString(fmt.Sprintf("<!-- pal:config:status=configured -->\n"))
	sb.WriteString(fmt.Sprintf("<!-- pal:workflow:%s -->\n", cfg.Workflow.Type))
	sb.WriteString(fmt.Sprintf("<!-- pal:updated:%s -->\n", time.Now().Format("2006-01-02")))

	return sb.String()
}

// getWorkflowGuide returns workflow-specific guide
func getWorkflowGuide(wt config.WorkflowType) string {
	switch wt {
	case config.WorkflowSimple:
		return `### Simple 워크플로우

**Collaborator**로서 모든 역할을 종합 수행합니다.

**작업 방식:**
1. 요청을 이해하고 명확화
2. 작업 범위 확인
3. 단계별 진행 (사용자 피드백 반영)
4. 완료 후 결과 요약

**포인트:**
- 사용자가 Git/코드 관리
- 대화하며 협업
- 필요시 포트 생성하여 작업 추적
`

	case config.WorkflowSingle:
		return `### Single 워크플로우

하나의 세션에서 **역할을 전환**하며 작업합니다.

**역할 순서:**
1. **Builder** → 요구사항 분석, 포트 분해
2. **Planner** → 실행 순서 계획
3. **Architect** → 기술 결정 (필요시)
4. **Worker** → 실제 구현
5. **Tester** → 테스트 작성
6. **Logger** → 커밋/문서화

**포인트:**
- 포트 기반 작업 추적
- 완료 조건 체크리스트 활용
- 역할 전환 시 명시적 선언
`

	case config.WorkflowIntegrate:
		return `### Integrate 워크플로우

**빌더 세션**이 전체를 관리하고, **워커 세션**이 개별 포트를 작업합니다.

**빌더 역할:**
1. 요구사항 분석 및 포트 분해
2. 파이프라인 구성
3. 워커 세션 spawn
4. 품질 게이트 운영

**워커 세션:**
- Claude 새 창에서 특정 포트 작업
- ` + "`pal session start --type sub --port <id>`" + `

**포인트:**
- 포트 간 의존성 관리
- 에스컬레이션 처리
- 빌드/테스트 통과 필수
`

	case config.WorkflowMulti:
		return `### Multi 워크플로우

복수의 **Integrate 워크플로우**를 병렬 운영합니다.

**구조:**
- 전체 조율 세션
- 서브 프로젝트별 빌더 세션
- 각 빌더 아래 워커 세션들

**조율 작업:**
1. 서브 프로젝트 간 의존성 관리
2. 통합 지점 조율
3. 전체 진행 상황 모니터링
`

	default:
		return ""
	}
}
