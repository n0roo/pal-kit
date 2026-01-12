package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/context"
	pkg "github.com/n0roo/pal-kit/internal/package"
	"github.com/spf13/cobra"
)

var packageCmd = &cobra.Command{
	Use:     "package",
	Aliases: []string{"pkg"},
	Short:   "패키지 관리",
	Long: `에이전트 패키지를 관리합니다.

패키지는 기술 스택, 아키텍처, 컨벤션, 워커를 묶는 상위 구조입니다.

예시:
  pal package list              # 사용 가능한 패키지 목록
  pal package show backend      # 패키지 상세 정보
  pal package use backend       # 프로젝트에 패키지 적용
  pal package create my-pkg     # 새 패키지 생성
`,
}

var packageListCmd = &cobra.Command{
	Use:   "list",
	Short: "사용 가능한 패키지 목록",
	RunE:  runPackageList,
}

var packageShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "패키지 상세 정보",
	Args:  cobra.ExactArgs(1),
	RunE:  runPackageShow,
}

var packageUseCmd = &cobra.Command{
	Use:   "use <id>",
	Short: "프로젝트에 패키지 적용",
	Args:  cobra.ExactArgs(1),
	RunE:  runPackageUse,
}

var packageCreateCmd = &cobra.Command{
	Use:   "create <id>",
	Short: "새 패키지 생성",
	Long: `새로운 패키지를 생성합니다.

예시:
  pal package create my-backend --extends pa-layered-backend
  pal package create my-frontend --lang typescript
`,
	Args: cobra.ExactArgs(1),
	RunE: runPackageCreate,
}

var packageValidateCmd = &cobra.Command{
	Use:   "validate [id]",
	Short: "패키지 검증",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPackageValidate,
}

var packageWorkersCmd = &cobra.Command{
	Use:   "workers <id>",
	Short: "패키지의 워커 목록",
	Args:  cobra.ExactArgs(1),
	RunE:  runPackageWorkers,
}

var (
	pkgExtends  string
	pkgLang     string
	pkgArch     string
)

func init() {
	rootCmd.AddCommand(packageCmd)
	packageCmd.AddCommand(packageListCmd)
	packageCmd.AddCommand(packageShowCmd)
	packageCmd.AddCommand(packageUseCmd)
	packageCmd.AddCommand(packageCreateCmd)
	packageCmd.AddCommand(packageValidateCmd)
	packageCmd.AddCommand(packageWorkersCmd)

	packageCreateCmd.Flags().StringVar(&pkgExtends, "extends", "", "상속할 부모 패키지 ID")
	packageCreateCmd.Flags().StringVar(&pkgLang, "lang", "", "주 언어 (kotlin, typescript, go 등)")
	packageCreateCmd.Flags().StringVar(&pkgArch, "arch", "PA-Layered", "아키텍처 이름")
}

func getPackageService() (*pkg.Service, error) {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	globalDir := filepath.Join(config.GlobalPalDir(), "packages")

	return pkg.NewService(projectRoot, globalDir), nil
}

func runPackageList(cmd *cobra.Command, args []string) error {
	svc, err := getPackageService()
	if err != nil {
		return err
	}

	packages, err := svc.List()
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(packages)
	}

	if len(packages) == 0 {
		fmt.Println("등록된 패키지가 없습니다.")
		fmt.Println()
		fmt.Println("패키지 생성:")
		fmt.Println("  pal package create my-package --extends pa-layered-backend")
		return nil
	}

	fmt.Println("📦 패키지 목록")
	fmt.Println()

	// 카테고리별 그룹화
	backendPkgs := []*pkg.Package{}
	frontendPkgs := []*pkg.Package{}
	basePkgs := []*pkg.Package{}

	for _, p := range packages {
		if strings.Contains(p.ID, "backend") {
			backendPkgs = append(backendPkgs, p)
		} else if strings.Contains(p.ID, "frontend") {
			frontendPkgs = append(frontendPkgs, p)
		} else {
			basePkgs = append(basePkgs, p)
		}
	}

	if len(basePkgs) > 0 {
		fmt.Println("🏛️  Base 패키지:")
		for _, p := range basePkgs {
			printPackageSummary(p)
		}
		fmt.Println()
	}

	if len(backendPkgs) > 0 {
		fmt.Println("⚙️  Backend 패키지:")
		for _, p := range backendPkgs {
			printPackageSummary(p)
		}
		fmt.Println()
	}

	if len(frontendPkgs) > 0 {
		fmt.Println("🎨 Frontend 패키지:")
		for _, p := range frontendPkgs {
			printPackageSummary(p)
		}
		fmt.Println()
	}

	fmt.Println("💡 패키지 상세:")
	fmt.Println("   pal package show <id>")

	return nil
}

func printPackageSummary(p *pkg.Package) {
	extends := ""
	if p.Extends != "" {
		extends = fmt.Sprintf(" (extends: %s)", p.Extends)
	}
	fmt.Printf("   %-25s %s%s\n", p.ID, p.Name, extends)
	if p.Tech.Language != "" {
		fmt.Printf("      언어: %s", p.Tech.Language)
		if len(p.Tech.Frameworks) > 0 {
			fmt.Printf(", 프레임워크: %s", strings.Join(p.Tech.Frameworks, ", "))
		}
		fmt.Println()
	}
}

func runPackageShow(cmd *cobra.Command, args []string) error {
	id := args[0]

	svc, err := getPackageService()
	if err != nil {
		return err
	}

	p, err := svc.Get(id)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(p)
	}

	fmt.Printf("📦 패키지: %s\n", p.Name)
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("ID:      %s\n", p.ID)
	fmt.Printf("버전:    %s\n", p.Version)
	if p.Extends != "" {
		fmt.Printf("상속:    %s\n", p.Extends)
	}
	if p.Description != "" {
		fmt.Printf("설명:\n%s\n", indentText(p.Description, "   "))
	}
	fmt.Println()

	fmt.Println("🔧 기술 스택")
	fmt.Printf("   언어:       %s\n", p.Tech.Language)
	if len(p.Tech.Frameworks) > 0 {
		fmt.Printf("   프레임워크: %s\n", strings.Join(p.Tech.Frameworks, ", "))
	}
	if p.Tech.BuildTool != "" {
		fmt.Printf("   빌드 도구:  %s\n", p.Tech.BuildTool)
	}
	if p.Tech.Runtime != "" {
		fmt.Printf("   런타임:     %s\n", p.Tech.Runtime)
	}
	fmt.Println()

	fmt.Println("🏗️  아키텍처")
	fmt.Printf("   이름:       %s\n", p.Architecture.Name)
	fmt.Printf("   레이어:     %s\n", strings.Join(p.Architecture.Layers, " → "))
	if p.Architecture.ConventionsRef != "" {
		fmt.Printf("   컨벤션:     %s\n", p.Architecture.ConventionsRef)
	}
	fmt.Println()

	fmt.Println("📐 방법론")
	fmt.Printf("   Port Driven: %v\n", p.Methodology.PortDriven)
	fmt.Printf("   CQS:         %v\n", p.Methodology.CQS)
	fmt.Printf("   Event Driven: %v\n", p.Methodology.EventDriven)
	fmt.Println()

	if len(p.Workers) > 0 {
		fmt.Println("👷 워커")
		for _, w := range p.Workers {
			fmt.Printf("   - %s\n", w)
		}
		fmt.Println()
	}

	if len(p.CoreOverrides) > 0 {
		fmt.Println("⚙️  Core 오버라이드")
		for name, override := range p.CoreOverrides {
			fmt.Printf("   %s:\n", name)
			if override.ConventionsRef != "" {
				fmt.Printf("      컨벤션: %s\n", override.ConventionsRef)
			}
			if len(override.PortTemplates) > 0 {
				fmt.Printf("      템플릿: %d개\n", len(override.PortTemplates))
			}
			if len(override.ValidationRules) > 0 {
				fmt.Printf("      검증: %s\n", strings.Join(override.ValidationRules, ", "))
			}
		}
		fmt.Println()
	}

	fmt.Printf("📁 파일: %s\n", p.FilePath)

	return nil
}

func runPackageUse(cmd *cobra.Command, args []string) error {
	id := args[0]

	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL Kit 프로젝트를 찾을 수 없습니다")
	}

	svc, err := getPackageService()
	if err != nil {
		return err
	}

	// 패키지 존재 확인
	p, err := svc.Get(id)
	if err != nil {
		return err
	}

	// 프로젝트 설정 로드
	cfg, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		return err
	}

	// 패키지 설정 추가 (config에 Package 필드 추가 필요)
	// 현재는 Workers만 설정
	cfg.Agents.Workers = p.Workers

	// 저장
	if err := config.SaveProjectConfig(projectRoot, cfg); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "applied",
			"package": id,
			"workers": p.Workers,
		})
	}

	fmt.Printf("✅ 패키지 적용: %s\n", p.Name)
	fmt.Println()
	fmt.Println("적용된 워커:")
	for _, w := range p.Workers {
		fmt.Printf("   - %s\n", w)
	}
	fmt.Println()
	fmt.Println("💡 워커 에이전트 추가:")
	fmt.Println("   pal agent add workers/backend/<type>")

	return nil
}

func runPackageCreate(cmd *cobra.Command, args []string) error {
	id := args[0]

	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	svc, err := getPackageService()
	if err != nil {
		return err
	}

	// 기본 패키지 구조
	newPkg := &pkg.Package{
		ID:      id,
		Name:    id,
		Version: "1.0.0",
		Extends: pkgExtends,
		Tech: pkg.TechConfig{
			Language: pkgLang,
		},
		Architecture: pkg.ArchConfig{
			Name:   pkgArch,
			Layers: []string{"L1", "L2"},
		},
		Methodology: pkg.MethodConfig{
			PortDriven: true,
		},
		Workers: []string{},
	}

	// 상속이 있으면 부모에서 기본값 가져오기
	if pkgExtends != "" {
		parent, err := svc.Get(pkgExtends)
		if err != nil {
			return fmt.Errorf("부모 패키지 로드 실패: %w", err)
		}
		if pkgLang == "" {
			newPkg.Tech.Language = parent.Tech.Language
		}
		newPkg.Architecture = parent.Architecture
		newPkg.Methodology = parent.Methodology
	}

	// packages 디렉토리에 저장
	targetDir := filepath.Join(projectRoot, "packages")
	if err := svc.Create(newPkg, targetDir); err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(newPkg)
	}

	fmt.Printf("✅ 패키지 생성: %s\n", newPkg.Name)
	fmt.Printf("   파일: %s\n", newPkg.FilePath)
	fmt.Println()
	fmt.Println("💡 패키지 편집 후 적용:")
	fmt.Printf("   pal package use %s\n", id)

	return nil
}

func runPackageValidate(cmd *cobra.Command, args []string) error {
	svc, err := getPackageService()
	if err != nil {
		return err
	}

	var packages []*pkg.Package

	if len(args) > 0 {
		// 특정 패키지만 검증
		p, err := svc.Get(args[0])
		if err != nil {
			return err
		}
		packages = append(packages, p)
	} else {
		// 모든 패키지 검증
		packages, err = svc.List()
		if err != nil {
			return err
		}
	}

	allValid := true
	results := make([]map[string]interface{}, 0)

	for _, p := range packages {
		errors := svc.Validate(p)
		result := map[string]interface{}{
			"id":     p.ID,
			"valid":  len(errors) == 0,
			"errors": errors,
		}
		results = append(results, result)

		if len(errors) > 0 {
			allValid = false
		}
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	fmt.Println("🔍 패키지 검증 결과")
	fmt.Println()

	for _, result := range results {
		id := result["id"].(string)
		valid := result["valid"].(bool)
		errors := result["errors"].([]string)

		if valid {
			fmt.Printf("✅ %s: 유효\n", id)
		} else {
			fmt.Printf("❌ %s: 오류\n", id)
			for _, e := range errors {
				fmt.Printf("   - %s\n", e)
			}
		}
	}

	if !allValid {
		return fmt.Errorf("일부 패키지에 오류가 있습니다")
	}

	return nil
}

func runPackageWorkers(cmd *cobra.Command, args []string) error {
	id := args[0]

	svc, err := getPackageService()
	if err != nil {
		return err
	}

	workers, err := svc.GetWorkers(id)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(workers)
	}

	fmt.Printf("👷 %s 패키지 워커\n", id)
	fmt.Println()

	for _, w := range workers {
		fmt.Printf("   - %s\n", w)
	}

	return nil
}

func indentText(text, prefix string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
