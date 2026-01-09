package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/pipeline"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/spf13/cobra"
)

var (
	pipelineStatus  string
	pipelineLimit   int
	pipelineGroup   int
	pipelineAfter   string
	pipelineTmux    bool
	pipelineOutFile string
)

var pipelineCmd = &cobra.Command{
	Use:     "pipeline",
	Aliases: []string{"pl"},
	Short:   "파이프라인 관리",
	Long:    `포트 실행 파이프라인을 관리합니다.`,
}

var plCreateCmd = &cobra.Command{
	Use:   "create <id> [name]",
	Short: "파이프라인 생성",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPlCreate,
}

var plAddCmd = &cobra.Command{
	Use:   "add <pipeline-id> <port-id>",
	Short: "파이프라인에 포트 추가",
	Long: `파이프라인에 포트를 추가합니다.

--group: 실행 그룹 번호 (낮을수록 먼저 실행, 같은 그룹은 병렬 가능)
--after: 의존성 추가 (이 포트 완료 후 실행)`,
	Args: cobra.ExactArgs(2),
	RunE: runPlAdd,
}

var plListCmd = &cobra.Command{
	Use:   "list",
	Short: "파이프라인 목록",
	RunE:  runPlList,
}

var plShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "파이프라인 상세 (트리뷰)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlShow,
}

var plStatusCmd = &cobra.Command{
	Use:   "status <id> [status]",
	Short: "파이프라인 상태 조회/변경",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPlStatus,
}

var plDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "파이프라인 삭제",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlDelete,
}

var plPlanCmd = &cobra.Command{
	Use:   "plan <id>",
	Short: "실행 계획 조회",
	Long:  `파이프라인의 실행 계획을 조회합니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPlPlan,
}

var plNextCmd = &cobra.Command{
	Use:   "next <id>",
	Short: "다음 실행 가능한 포트",
	Long:  `의존성이 충족되어 바로 실행 가능한 포트 목록을 반환합니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPlNext,
}

var plRunCmd = &cobra.Command{
	Use:   "run <id>",
	Short: "실행 스크립트 생성",
	Long: `파이프라인 실행을 위한 쉘 스크립트를 생성합니다.

--tmux: tmux 병렬 실행 스크립트 생성
--out: 파일로 저장 (기본: stdout)`,
	Args: cobra.ExactArgs(1),
	RunE: runPlRun,
}

var plPortStatusCmd = &cobra.Command{
	Use:   "port-status <pipeline-id> <port-id> <status>",
	Short: "파이프라인 내 포트 상태 변경",
	Args:  cobra.ExactArgs(3),
	RunE:  runPlPortStatus,
}

func init() {
	rootCmd.AddCommand(pipelineCmd)
	pipelineCmd.AddCommand(plCreateCmd)
	pipelineCmd.AddCommand(plAddCmd)
	pipelineCmd.AddCommand(plListCmd)
	pipelineCmd.AddCommand(plShowCmd)
	pipelineCmd.AddCommand(plStatusCmd)
	pipelineCmd.AddCommand(plDeleteCmd)
	pipelineCmd.AddCommand(plPlanCmd)
	pipelineCmd.AddCommand(plNextCmd)
	pipelineCmd.AddCommand(plRunCmd)
	pipelineCmd.AddCommand(plPortStatusCmd)

	plListCmd.Flags().StringVar(&pipelineStatus, "status", "", "상태 필터")
	plListCmd.Flags().IntVar(&pipelineLimit, "limit", 20, "결과 수 제한")

	plAddCmd.Flags().IntVar(&pipelineGroup, "group", 0, "실행 그룹 (기본: 0)")
	plAddCmd.Flags().StringVar(&pipelineAfter, "after", "", "의존 포트 ID")

	plRunCmd.Flags().BoolVar(&pipelineTmux, "tmux", false, "tmux 병렬 실행 스크립트")
	plRunCmd.Flags().StringVarP(&pipelineOutFile, "out", "o", "", "출력 파일 경로")
}

func getPipelineService() (*pipeline.Service, func(), error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, err
	}
	return pipeline.NewService(database), func() { database.Close() }, nil
}

func runPlCreate(cmd *cobra.Command, args []string) error {
	id := args[0]
	name := id
	if len(args) > 1 {
		name = args[1]
	}

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	sessionID := os.Getenv("CLAUDE_SESSION_ID")
	if err := svc.Create(id, name, sessionID); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "created",
			"id":     id,
			"name":   name,
		})
	} else {
		fmt.Printf("✓ 파이프라인 생성: %s\n", id)
		if name != id {
			fmt.Printf("  이름: %s\n", name)
		}
	}

	return nil
}

func runPlAdd(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]
	portID := args[1]

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	// 파이프라인 존재 확인
	if _, err := svc.Get(pipelineID); err != nil {
		return err
	}

	// 포트 추가
	if err := svc.AddPort(pipelineID, portID, pipelineGroup); err != nil {
		return err
	}

	// 의존성 추가
	if pipelineAfter != "" {
		if err := svc.AddDependency(portID, pipelineAfter); err != nil {
			return err
		}
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":     "added",
			"pipeline":   pipelineID,
			"port":       portID,
			"group":      pipelineGroup,
			"depends_on": pipelineAfter,
		})
	} else {
		fmt.Printf("✓ 포트 추가: %s → %s (그룹: %d)\n", portID, pipelineID, pipelineGroup)
		if pipelineAfter != "" {
			fmt.Printf("  의존성: %s 완료 후 실행\n", pipelineAfter)
		}
	}

	return nil
}

func runPlList(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	pipelines, err := svc.List(pipelineStatus, pipelineLimit)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"pipelines": pipelines,
		})
		return nil
	}

	if len(pipelines) == 0 {
		fmt.Println("파이프라인이 없습니다.")
		return nil
	}

	fmt.Printf("%-15s %-20s %-10s %s\n", "ID", "NAME", "STATUS", "CREATED")
	fmt.Println(strings.Repeat("-", 70))
	for _, p := range pipelines {
		statusEmoji := map[string]string{
			"pending":   "⏳",
			"running":   "🔄",
			"complete":  "✅",
			"failed":    "❌",
			"cancelled": "⚪",
		}
		name := p.Name
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		fmt.Printf("%-15s %-20s %s %-8s %s\n",
			p.ID, name, statusEmoji[p.Status], p.Status, p.CreatedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

func runPlShow(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]

	plSvc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	// 파이프라인 정보
	pl, err := plSvc.Get(pipelineID)
	if err != nil {
		return err
	}

	// 포트 정보를 위한 포트 서비스
	database, _ := db.Open(GetDBPath())
	defer database.Close()
	portSvc := port.NewService(database)

	// 그룹별 포트
	groups, err := plSvc.GetGroups(pipelineID)
	if err != nil {
		return err
	}

	// 진행률
	completed, total, _ := plSvc.GetProgress(pipelineID)

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"pipeline":  pl,
			"groups":    groups,
			"completed": completed,
			"total":     total,
		})
		return nil
	}

	// 트리뷰 출력
	statusEmoji := map[string]string{
		"pending":   "⏳",
		"running":   "🔄",
		"complete":  "✅",
		"failed":    "❌",
		"cancelled": "⚪",
		"skipped":   "⏭️",
	}

	fmt.Printf("📦 Pipeline: %s\n", pl.Name)
	fmt.Printf("├─ %s Status: %s\n", statusEmoji[pl.Status], pl.Status)
	fmt.Printf("├─ 📊 Progress: %d/%d complete\n", completed, total)
	fmt.Println("│")

	// 그룹 순서대로 정렬
	var groupOrders []int
	for g := range groups {
		groupOrders = append(groupOrders, g)
	}
	sort.Ints(groupOrders)

	for i, groupOrder := range groupOrders {
		ports := groups[groupOrder]
		isLast := i == len(groupOrders)-1
		prefix := "├─"
		childPrefix := "│  "
		if isLast {
			prefix = "└─"
			childPrefix = "   "
		}

		// 그룹 상태 계산
		groupStatus := "pending"
		allComplete := true
		anyRunning := false
		anyFailed := false
		for _, pp := range ports {
			if pp.Status != "complete" {
				allComplete = false
			}
			if pp.Status == "running" {
				anyRunning = true
			}
			if pp.Status == "failed" {
				anyFailed = true
			}
		}
		if allComplete {
			groupStatus = "complete"
		} else if anyFailed {
			groupStatus = "failed"
		} else if anyRunning {
			groupStatus = "running"
		}

		fmt.Printf("%s Group %d %s\n", prefix, groupOrder, statusEmoji[groupStatus])

		for j, pp := range ports {
			isLastPort := j == len(ports)-1
			portPrefix := childPrefix + "├─"
			if isLastPort {
				portPrefix = childPrefix + "└─"
			}

			// 포트 제목 가져오기
			portTitle := pp.PortID
			if p, err := portSvc.Get(pp.PortID); err == nil && p.Title.Valid {
				portTitle = p.Title.String
			}

			fmt.Printf("%s %s %s (%s)\n", portPrefix, statusEmoji[pp.Status], pp.PortID, portTitle)

			// 의존성 표시
			deps, _ := plSvc.GetDependencies(pp.PortID)
			if len(deps) > 0 {
				depPrefix := childPrefix
				if isLastPort {
					depPrefix += "   "
				} else {
					depPrefix += "│  "
				}
				fmt.Printf("%s└─ Depends: %s\n", depPrefix, strings.Join(deps, ", "))
			}
		}

		if !isLast {
			fmt.Println("│")
		}
	}

	return nil
}

func runPlStatus(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	// 상태 변경
	if len(args) > 1 {
		newStatus := args[1]
		if err := svc.UpdateStatus(pipelineID, newStatus); err != nil {
			return err
		}

		if jsonOut {
			json.NewEncoder(os.Stdout).Encode(map[string]string{
				"status":     "updated",
				"id":         pipelineID,
				"new_status": newStatus,
			})
		} else {
			fmt.Printf("✓ 파이프라인 상태 변경: %s → %s\n", pipelineID, newStatus)
		}
		return nil
	}

	// 상태 조회
	pl, err := svc.Get(pipelineID)
	if err != nil {
		return err
	}

	completed, total, _ := svc.GetProgress(pipelineID)

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"id":        pl.ID,
			"status":    pl.Status,
			"completed": completed,
			"total":     total,
		})
	} else {
		fmt.Printf("파이프라인: %s\n", pl.ID)
		fmt.Printf("상태: %s\n", pl.Status)
		fmt.Printf("진행: %d/%d\n", completed, total)
	}

	return nil
}

func runPlDelete(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.Delete(pipelineID); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "deleted",
			"id":     pipelineID,
		})
	} else {
		fmt.Printf("✓ 파이프라인 삭제: %s\n", pipelineID)
	}

	return nil
}

func runPlPlan(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	plan, err := svc.BuildExecutionPlan(pipelineID)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(plan)
		return nil
	}

	fmt.Printf("📋 Execution Plan: %s\n", pipelineID)
	fmt.Printf("   Total ports: %d\n", plan.TotalPorts)
	fmt.Println()

	statusEmoji := map[string]string{
		"pending":  "⏳",
		"running":  "🔄",
		"complete": "✅",
		"failed":   "❌",
	}

	for _, group := range plan.Groups {
		parallel := ""
		if len(group.Ports) > 1 {
			parallel = " (병렬 가능)"
		}
		fmt.Printf("Group %d%s:\n", group.Order, parallel)

		for _, port := range group.Ports {
			emoji := statusEmoji[port.Status]
			if emoji == "" {
				emoji = "⏳"
			}
			deps := ""
			if len(port.Dependencies) > 0 {
				deps = fmt.Sprintf(" ← %s", strings.Join(port.Dependencies, ", "))
			}
			fmt.Printf("  %s %s%s\n", emoji, port.PortID, deps)
		}
		fmt.Println()
	}

	return nil
}

func runPlNext(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	nextPorts, err := svc.GetNextPorts(pipelineID)
	if err != nil {
		return err
	}

	runningPorts, _ := svc.GetRunningPorts(pipelineID)

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"ready":   nextPorts,
			"running": runningPorts,
		})
		return nil
	}

	if len(runningPorts) > 0 {
		fmt.Printf("🔄 Running: %s\n", strings.Join(runningPorts, ", "))
	}

	if len(nextPorts) == 0 {
		isComplete, _ := svc.IsComplete(pipelineID)
		if isComplete {
			fmt.Println("✅ 모든 포트 완료")
		} else if len(runningPorts) > 0 {
			fmt.Println("⏳ 실행 중인 포트 완료 대기")
		} else {
			fmt.Println("❌ 실행 가능한 포트 없음 (의존성 확인)")
		}
		return nil
	}

	fmt.Printf("▶️  Ready: %s\n", strings.Join(nextPorts, ", "))
	fmt.Println()
	fmt.Println("실행 명령:")
	for _, portID := range nextPorts {
		fmt.Printf("  pal port activate %s && pal pl port-status %s %s running\n",
			portID, pipelineID, portID)
	}

	return nil
}

func runPlRun(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	// 프로젝트 루트 찾기
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}

	var script string
	if pipelineTmux {
		script, err = svc.GenerateTmuxScript(pipelineID, projectRoot, "")
	} else {
		script, err = svc.GenerateRunScript(pipelineID, projectRoot)
	}

	if err != nil {
		return err
	}

	// 출력
	if pipelineOutFile != "" {
		outPath := pipelineOutFile
		if !filepath.IsAbs(outPath) {
			outPath = filepath.Join(projectRoot, outPath)
		}

		if err := os.WriteFile(outPath, []byte(script), 0755); err != nil {
			return fmt.Errorf("파일 저장 실패: %w", err)
		}

		if !jsonOut {
			fmt.Printf("✓ 스크립트 생성: %s\n", outPath)
			fmt.Printf("  실행: bash %s\n", outPath)
		} else {
			json.NewEncoder(os.Stdout).Encode(map[string]string{
				"status": "generated",
				"file":   outPath,
			})
		}
	} else {
		fmt.Println(script)
	}

	return nil
}

func runPlPortStatus(cmd *cobra.Command, args []string) error {
	pipelineID := args[0]
	portID := args[1]
	newStatus := args[2]

	svc, cleanup, err := getPipelineService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.UpdatePortStatus(pipelineID, portID, newStatus); err != nil {
		return err
	}

	// 파이프라인 완료 체크
	isComplete, _ := svc.IsComplete(pipelineID)
	hasFailed, _ := svc.HasFailure(pipelineID)

	if isComplete && !hasFailed {
		svc.UpdateStatus(pipelineID, pipeline.StatusComplete)
	} else if hasFailed {
		svc.UpdateStatus(pipelineID, pipeline.StatusFailed)
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":     "updated",
			"pipeline":   pipelineID,
			"port":       portID,
			"new_status": newStatus,
		})
	} else {
		statusEmoji := map[string]string{
			"pending":  "⏳",
			"running":  "🔄",
			"complete": "✅",
			"failed":   "❌",
		}
		fmt.Printf("%s %s: %s → %s\n", statusEmoji[newStatus], pipelineID, portID, newStatus)

		if isComplete && !hasFailed {
			fmt.Println("🎉 파이프라인 완료!")
		}
	}

	return nil
}
