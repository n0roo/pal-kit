package cli

import (
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/agent"
	"github.com/n0roo/pal-kit/internal/config"
)

// installTemplates installs agent and convention templates to the project
// Uses global agent store (~/.pal/agents) as the source
func installTemplates(projectRoot string, forceOverwrite bool) error {
	globalPath := config.GlobalDir()

	// 전역 에이전트 스토어에서 설치
	count, err := agent.InstallFromGlobal(globalPath, projectRoot, forceOverwrite)
	if err != nil {
		return fmt.Errorf("에이전트 설치 실패: %w", err)
	}

	if count == 0 {
		// 새 파일이 없으면 (이미 존재) - 정상
		if verbose {
			fmt.Fprintf(os.Stderr, "ℹ️ 에이전트 파일이 이미 존재합니다 (덮어쓰기: %v)\n", forceOverwrite)
		}
		return nil
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "✅ %d개 에이전트/컨벤션 파일 설치 완료\n", count)
	}

	return nil
}

// installTemplatesFromEmbedded is a fallback using embedded templates directly
func installTemplatesFromEmbedded(projectRoot string, forceOverwrite bool) error {
	count, err := agent.CountTemplates()
	if err != nil {
		return fmt.Errorf("템플릿 목록 조회 실패: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("사용 가능한 템플릿이 없습니다")
	}

	var installErr error
	if forceOverwrite {
		installErr = agent.InstallTemplatesWithOverwrite(projectRoot)
	} else {
		installErr = agent.InstallTemplates(projectRoot)
	}

	if installErr != nil {
		return fmt.Errorf("템플릿 복사 실패: %w", installErr)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "✅ %d개 템플릿 파일 설치 완료 (embedded)\n", count)
	}

	return nil
}
