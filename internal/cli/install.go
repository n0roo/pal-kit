package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/n0roo/pal-kit/internal/agent"
	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/spf13/cobra"
)

var (
	installForce          bool
	installImportExisting bool
	installBin            bool
	installZsh            bool
	installBinPath        string
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

바이너리 설치:
  --bin              /usr/local/bin에 복사 (sudo 필요)
  --bin-path <path>  지정 경로에 복사
  --zsh              ~/.zshrc에 PATH 추가

설치 후 프로젝트에서 'pal init' 명령으로 초기화할 수 있습니다.
`,
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&installForce, "force", false, "기존 설치 덮어쓰기")
	installCmd.Flags().BoolVar(&installImportExisting, "import-existing", false, "기존 프로젝트 DB 마이그레이션")
	installCmd.Flags().BoolVar(&installBin, "bin", false, "/usr/local/bin에 바이너리 복사")
	installCmd.Flags().BoolVar(&installZsh, "zsh", false, "~/.zshrc에 PATH 추가")
	installCmd.Flags().StringVar(&installBinPath, "bin-path", "", "바이너리 설치 경로 지정")
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

	// 8. 바이너리 설치 (선택)
	var binInstalled string
	if installBin || installBinPath != "" || installZsh {
		var err error
		binInstalled, err = installBinary()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ 바이너리 설치 실패: %v\n", err)
		} else if binInstalled != "" {
			created = append(created, binInstalled)
		}
	}

	// 결과 출력
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":       "installed",
			"path":         globalDir,
			"created":      created,
			"imported":     imported,
			"bin_installed": binInstalled,
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

	// embed된 템플릿 파일들을 복사
	if err := agent.InstallTemplates(agentsDir); err != nil {
		return fmt.Errorf("에이전트 템플릿 설치 실패: %w", err)
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

// installBinary installs the pal binary to system path
func installBinary() (string, error) {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("실행 파일 경로 확인 실패: %w", err)
	}

	// Resolve symlink
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("심볼릭 링크 해석 실패: %w", err)
	}

	// Determine target path
	var targetPath string
	if installBinPath != "" {
		targetPath = filepath.Join(installBinPath, "pal")
	} else if installBin {
		targetPath = "/usr/local/bin/pal"
	} else if installZsh {
		// Install to ~/.local/bin and add to PATH
		homeDir, _ := os.UserHomeDir()
		localBin := filepath.Join(homeDir, ".local", "bin")
		os.MkdirAll(localBin, 0755)
		targetPath = filepath.Join(localBin, "pal")

		// Add to .zshrc
		if err := addToZshrc(localBin); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ .zshrc 수정 실패: %v\n", err)
		}
	}

	if targetPath == "" {
		return "", nil
	}

	// Check if source and target are the same
	if execPath == targetPath {
		return targetPath, nil
	}

	// Copy binary
	if err := copyBinary(execPath, targetPath); err != nil {
		// Try with sudo for /usr/local/bin
		if strings.HasPrefix(targetPath, "/usr/local/bin") && runtime.GOOS != "windows" {
			fmt.Println("관리자 권한이 필요합니다...")
			cmd := exec.Command("sudo", "cp", execPath, targetPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("sudo 복사 실패: %w", err)
			}
			// Set permissions
			exec.Command("sudo", "chmod", "+x", targetPath).Run()
			return targetPath, nil
		}
		return "", fmt.Errorf("복사 실패: %w", err)
	}

	// Set executable permission
	os.Chmod(targetPath, 0755)

	return targetPath, nil
}

// copyBinary copies a file from src to dst
func copyBinary(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(dst), 0755)

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// addToZshrc adds a path to .zshrc if not already present
func addToZshrc(binPath string) error {
	homeDir, _ := os.UserHomeDir()
	zshrcPath := filepath.Join(homeDir, ".zshrc")

	// Read existing content
	content, err := os.ReadFile(zshrcPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if already added
	exportLine := fmt.Sprintf("export PATH=\"%s:$PATH\"", binPath)
	if strings.Contains(string(content), binPath) {
		return nil // Already added
	}

	// Append to .zshrc
	f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	comment := "\n# PAL Kit\n"
	if _, err := f.WriteString(comment + exportLine + "\n"); err != nil {
		return err
	}

	fmt.Println("✅ ~/.zshrc에 PATH 추가됨")
	fmt.Println("   새 터미널을 열거나 'source ~/.zshrc' 실행")

	return nil
}
