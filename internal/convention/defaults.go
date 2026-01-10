package convention

// DefaultConventions returns built-in convention templates
func DefaultConventions() []*Convention {
	return []*Convention{
		{
			ID:          "go-coding-style",
			Name:        "Go 코딩 스타일",
			Type:        TypeCodingStyle,
			Description: "Go 언어 코딩 스타일 컨벤션",
			Enabled:     true,
			Priority:    8,
			Rules: []Rule{
				{
					ID:          "error-handling",
					Description: "에러는 반드시 처리해야 합니다",
					AntiPattern: `\b\w+,\s*_\s*:?=.*\(`,
					FileTypes:   []string{".go"},
					Severity:    "warning",
				},
				{
					ID:          "package-comment",
					Description: "패키지는 주석이 필요합니다",
					Pattern:     `(?m)^// Package \w+`,
					FileTypes:   []string{".go"},
					Severity:    "info",
				},
			},
			Examples: Examples{
				Good: []Example{
					{Code: "if err != nil { return err }", Description: "에러 처리"},
				},
				Bad: []Example{
					{Code: "result, _ := doSomething()", Description: "에러 무시"},
				},
			},
		},
		{
			ID:          "commit-message",
			Name:        "커밋 메시지",
			Type:        TypeCommitMessage,
			Description: "Conventional Commits 스타일 커밋 메시지",
			Enabled:     true,
			Priority:    7,
			Rules: []Rule{
				{
					ID:          "conventional-format",
					Description: "커밋 메시지는 type(scope): description 형식을 사용합니다",
					Pattern:     `^(feat|fix|docs|style|refactor|test|chore|build|ci|perf)(\([a-z0-9-]+\))?:\s.+`,
					Severity:    "warning",
				},
				{
					ID:          "subject-length",
					Description: "제목은 50자 이내로 작성합니다",
					AntiPattern: `^.{51,}$`,
					Severity:    "info",
				},
			},
			Examples: Examples{
				Good: []Example{
					{Code: "feat(auth): add JWT authentication", Description: "기능 추가"},
					{Code: "fix(api): resolve null pointer exception", Description: "버그 수정"},
					{Code: "docs: update README with installation guide", Description: "문서 수정"},
				},
				Bad: []Example{
					{Code: "fixed bug", Description: "타입 누락"},
					{Code: "feat: Add new feature that does many things and has a very long description", Description: "너무 긴 제목"},
				},
			},
		},
		{
			ID:          "naming-convention",
			Name:        "네이밍 컨벤션",
			Type:        TypeNaming,
			Description: "파일 및 변수 네이밍 규칙",
			Enabled:     true,
			Priority:    6,
			Rules: []Rule{
				{
					ID:          "go-file-naming",
					Description: "Go 파일은 snake_case를 사용합니다",
					Pattern:     `^[a-z][a-z0-9_]*\.go$`,
					FileTypes:   []string{".go"},
					Severity:    "warning",
				},
				{
					ID:          "test-file-naming",
					Description: "테스트 파일은 _test.go로 끝납니다",
					Pattern:     `_test\.go$`,
					FileTypes:   []string{".go"},
					Severity:    "info",
				},
			},
		},
		{
			ID:          "documentation",
			Name:        "문서화",
			Type:        TypeDocumentation,
			Description: "코드 문서화 규칙",
			Enabled:     true,
			Priority:    5,
			Rules: []Rule{
				{
					ID:          "exported-docs",
					Description: "exported 함수는 주석이 필요합니다",
					Pattern:     `(?m)^// \w+ `,
					FileTypes:   []string{".go"},
					Severity:    "info",
				},
				{
					ID:          "readme-exists",
					Description: "README.md 파일이 필요합니다",
					Pattern:     `README`,
					Severity:    "warning",
				},
			},
		},
		{
			ID:          "testing",
			Name:        "테스팅",
			Type:        TypeTesting,
			Description: "테스트 코드 규칙",
			Enabled:     true,
			Priority:    7,
			Rules: []Rule{
				{
					ID:          "test-function-naming",
					Description: "테스트 함수는 Test로 시작합니다",
					Pattern:     `func Test\w+\(`,
					FileTypes:   []string{".go"},
					Severity:    "error",
				},
				{
					ID:          "table-driven-tests",
					Description: "테이블 기반 테스트를 권장합니다",
					Pattern:     `tests\s*:?=\s*\[\]struct`,
					FileTypes:   []string{".go"},
					Severity:    "info",
				},
			},
		},
		{
			ID:          "error-handling",
			Name:        "에러 처리",
			Type:        TypeErrorHandling,
			Description: "에러 처리 규칙",
			Enabled:     true,
			Priority:    9,
			Rules: []Rule{
				{
					ID:          "wrap-errors",
					Description: "에러는 컨텍스트와 함께 wrap합니다",
					Pattern:     `fmt\.Errorf\(.+%w`,
					FileTypes:   []string{".go"},
					Severity:    "info",
				},
				{
					ID:          "no-panic",
					Description: "라이브러리에서 panic 사용을 피합니다",
					AntiPattern: `\bpanic\(`,
					FileTypes:   []string{".go"},
					Severity:    "warning",
				},
			},
			Examples: Examples{
				Good: []Example{
					{Code: `return fmt.Errorf("failed to open file: %w", err)`, Description: "에러 래핑"},
				},
				Bad: []Example{
					{Code: `panic("something went wrong")`, Description: "panic 사용"},
				},
			},
		},
	}
}

// InitDefaultConventions creates default convention files
func (s *Service) InitDefaultConventions() ([]string, error) {
	if err := s.EnsureDir(); err != nil {
		return nil, err
	}

	var created []string
	defaults := DefaultConventions()

	for _, conv := range defaults {
		// 이미 존재하는지 확인
		if _, err := s.Get(conv.ID); err == nil {
			continue
		}

		if err := s.Create(conv); err != nil {
			continue
		}

		created = append(created, conv.ID)
	}

	return created, nil
}
