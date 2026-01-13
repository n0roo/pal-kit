package cli

import (
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/agent"
)

// installTemplates installs agent and convention templates to the project
func installTemplates(projectRoot string, forceOverwrite bool) error {
	// 템플릿 개수 확인
	count, err := agent.CountTemplates()
	if err != nil {
		return fmt.Errorf("템플릿 목록 조회 실패: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("사용 가능한 템플릿이 없습니다")
	}

	// 템플릿 설치
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
		fmt.Fprintf(os.Stderr, "✅ %d개 템플릿 파일 설치 완료\n", count)
	}

	return nil
}
