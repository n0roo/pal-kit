package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/n0roo/pal-kit/internal/template"
	"github.com/spf13/cobra"
)

var (
	templateOutput string
	templateTitle  string
	templateID     string
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "템플릿 관리",
	Long:  `포트, 에이전트, 세션 등의 템플릿을 관리합니다.`,
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "템플릿 목록",
	RunE:  runTemplateList,
}

var templateCreateCmd = &cobra.Command{
	Use:   "create <type>",
	Short: "템플릿에서 파일 생성",
	Long: `템플릿에서 새 파일을 생성합니다.

타입:
  port     - 작업 단위 명세서
  agent    - 에이전트 프롬프트
  session  - 세션 기록
  hook     - Hook 스크립트`,
	Args: cobra.ExactArgs(1),
	RunE: runTemplateCreate,
}

var templateShowCmd = &cobra.Command{
	Use:   "show <type>",
	Short: "템플릿 내용 보기",
	Args:  cobra.ExactArgs(1),
	RunE:  runTemplateShow,
}

func init() {
	rootCmd.AddCommand(templateCmd)
	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateShowCmd)

	templateCreateCmd.Flags().StringVarP(&templateOutput, "output", "o", "", "출력 파일 경로")
	templateCreateCmd.Flags().StringVar(&templateTitle, "title", "", "제목")
	templateCreateCmd.Flags().StringVar(&templateID, "id", "", "ID")
}

func runTemplateList(cmd *cobra.Command, args []string) error {
	types := template.ValidTypes

	if jsonOut {
		var list []map[string]string
		for _, t := range types {
			list = append(list, map[string]string{
				"type":        string(t),
				"description": template.GetTypeDescription(t),
			})
		}
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"templates": list,
		})
		return nil
	}

	fmt.Println("사용 가능한 템플릿:")
	fmt.Println(strings.Repeat("-", 40))
	for _, t := range types {
		fmt.Printf("  %-10s  %s\n", t, template.GetTypeDescription(t))
	}
	fmt.Println()
	fmt.Println("사용법: pal template create <type> --id <id> --title <title>")

	return nil
}

func runTemplateCreate(cmd *cobra.Command, args []string) error {
	templateType := template.TemplateType(args[0])

	// 타입 유효성 검사
	if !template.IsValidType(args[0]) {
		return fmt.Errorf("알 수 없는 템플릿 타입: %s (가능: %v)", args[0], template.ValidTypes)
	}

	// ID 필수
	if templateID == "" {
		return fmt.Errorf("--id 플래그가 필요합니다")
	}

	// 제목 기본값
	if templateTitle == "" {
		templateTitle = templateID
	}

	// 출력 경로 기본값
	outputPath := templateOutput
	if outputPath == "" {
		outputPath = template.GetDefaultOutputPath(templateType, templateID)
	}

	// 서비스 생성
	cwd, _ := os.Getwd()
	svc := template.NewService(cwd)

	// 템플릿 데이터
	data := template.TemplateData{
		ID:    templateID,
		Title: templateTitle,
	}

	// 파일 생성
	if err := svc.Create(templateType, outputPath, data); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "created",
			"type":   string(templateType),
			"id":     templateID,
			"path":   outputPath,
		})
	} else {
		fmt.Printf("✓ 템플릿 생성: %s\n", outputPath)
		fmt.Printf("  타입: %s\n", templateType)
		fmt.Printf("  ID: %s\n", templateID)
	}

	return nil
}

func runTemplateShow(cmd *cobra.Command, args []string) error {
	templateType := template.TemplateType(args[0])

	if !template.IsValidType(args[0]) {
		return fmt.Errorf("알 수 없는 템플릿 타입: %s", args[0])
	}

	cwd, _ := os.Getwd()
	svc := template.NewService(cwd)

	content, err := svc.GetTemplateContent(templateType)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"type":    string(templateType),
			"content": content,
		})
		return nil
	}

	fmt.Printf("=== 템플릿: %s ===\n\n", templateType)
	fmt.Println(content)

	return nil
}
