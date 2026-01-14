package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/kb"
	"github.com/spf13/cobra"
)

var kbCmd = &cobra.Command{
	Use:   "kb",
	Short: "Knowledge Base 관리",
	Long:  `Knowledge Base 구조 관리 및 검색`,
}

var kbInitCmd = &cobra.Command{
	Use:   "init [vault-path]",
	Short: "KB 초기화",
	Long: `Knowledge Base 구조를 초기화합니다.

생성되는 구조:
  _taxonomy/          분류체계 정의
  00-System/          시스템 문서
  10-Domains/         도메인 지식
  20-Projects/        프로젝트 문서
  30-References/      참조 문서
  40-Archive/         아카이브
  .pal-kb/            메타데이터`,
	Args: cobra.MaximumNArgs(1),
	RunE: runKBInit,
}

var kbStatusCmd = &cobra.Command{
	Use:   "status [vault-path]",
	Short: "KB 상태 확인",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runKBStatus,
}

func init() {
	rootCmd.AddCommand(kbCmd)
	kbCmd.AddCommand(kbInitCmd)
	kbCmd.AddCommand(kbStatusCmd)
}

func getVaultPath(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	cwd, _ := os.Getwd()
	return cwd
}

func runKBInit(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	svc := kb.NewService(vaultPath)
	if err := svc.Init(); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":     "initialized",
			"vault_path": vaultPath,
		})
	}

	fmt.Println("✅ Knowledge Base 초기화 완료")
	fmt.Printf("   경로: %s\n", vaultPath)
	fmt.Println()
	fmt.Println("📁 생성된 구조:")
	fmt.Println("   _taxonomy/      분류체계")
	fmt.Println("   00-System/      시스템")
	fmt.Println("   10-Domains/     도메인")
	fmt.Println("   20-Projects/    프로젝트")
	fmt.Println("   30-References/  참조")
	fmt.Println("   40-Archive/     아카이브")
	fmt.Println()
	fmt.Println("다음 단계:")
	fmt.Println("  1. _taxonomy/domains.yaml 편집")
	fmt.Println("  2. pal kb toc generate")

	return nil
}

func runKBStatus(cmd *cobra.Command, args []string) error {
	vaultPath := getVaultPath(args)

	svc := kb.NewService(vaultPath)
	status, err := svc.Status()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(status)
	}

	fmt.Println("📚 Knowledge Base 상태")
	fmt.Printf("   경로: %s\n", status.VaultPath)

	if !status.Initialized {
		fmt.Println("   상태: ❌ 초기화되지 않음")
		fmt.Println()
		fmt.Println("초기화: pal kb init")
		return nil
	}

	fmt.Println("   상태: ✅ 초기화됨")
	fmt.Printf("   버전: %s\n", status.Version)
	fmt.Printf("   생성: %s\n", status.CreatedAt)
	fmt.Println()
	fmt.Println("📊 문서 수:")
	for section, count := range status.Sections {
		fmt.Printf("   %-15s %d\n", section, count)
	}

	return nil
}
