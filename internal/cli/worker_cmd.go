package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/worker"
	"github.com/spf13/cobra"
)

var (
	workerPortID string
	workerFilter string
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "워커 관리",
	Long:  `PA-Layered 워커를 관리합니다.`,
}

var workerListCmd = &cobra.Command{
	Use:   "list",
	Short: "워커 목록",
	Long:  `사용 가능한 워커 목록을 표시합니다.`,
	RunE:  runWorkerList,
}

var workerShowCmd = &cobra.Command{
	Use:   "show <worker-id>",
	Short: "워커 상세",
	Long:  `워커의 상세 정보를 표시합니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkerShow,
}

var workerSwitchCmd = &cobra.Command{
	Use:   "switch <worker-id>",
	Short: "워커 전환",
	Long: `지정된 워커로 전환합니다.

전환 시 수행되는 작업:
- CLAUDE.md 활성 워커 섹션 업데이트
- 해당 워커의 컨벤션 로드
- 체크리스트 업데이트`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkerSwitch,
}

var workerMapCmd = &cobra.Command{
	Use:   "map <port-id>",
	Short: "포트에 적합한 워커 찾기",
	Long:  `포트 명세를 분석하여 적합한 워커를 찾습니다.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkerMap,
}

func init() {
	rootCmd.AddCommand(workerCmd)
	workerCmd.AddCommand(workerListCmd)
	workerCmd.AddCommand(workerShowCmd)
	workerCmd.AddCommand(workerSwitchCmd)
	workerCmd.AddCommand(workerMapCmd)

	workerListCmd.Flags().StringVarP(&workerFilter, "filter", "f", "", "필터 (backend, frontend)")
	workerSwitchCmd.Flags().StringVarP(&workerPortID, "port", "p", "", "포트 ID")
}

func runWorkerList(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	mapper := worker.NewMapper(projectRoot)
	workers, err := mapper.ListWorkers()
	if err != nil {
		return err
	}

	// Sort workers
	sort.Slice(workers, func(i, j int) bool {
		return workers[i].Agent.ID < workers[j].Agent.ID
	})

	// Filter
	if workerFilter != "" {
		var filtered []*worker.WorkerSpec
		filter := strings.ToLower(workerFilter)
		for _, w := range workers {
			category := categorizeWorkerID(w.Agent.ID)
			if strings.Contains(category, filter) {
				filtered = append(filtered, w)
			}
		}
		workers = filtered
	}

	if jsonOut {
		output := make([]map[string]interface{}, len(workers))
		for i, w := range workers {
			output[i] = map[string]interface{}{
				"id":          w.Agent.ID,
				"name":        w.Agent.Name,
				"layer":       w.Agent.Layer,
				"tech":        w.Agent.Tech,
				"description": w.Agent.Description,
			}
		}
		return json.NewEncoder(os.Stdout).Encode(output)
	}

	// Group by category
	backendWorkers := []*worker.WorkerSpec{}
	frontendWorkers := []*worker.WorkerSpec{}
	otherWorkers := []*worker.WorkerSpec{}

	for _, w := range workers {
		category := categorizeWorkerID(w.Agent.ID)
		switch category {
		case "backend":
			backendWorkers = append(backendWorkers, w)
		case "frontend":
			frontendWorkers = append(frontendWorkers, w)
		default:
			otherWorkers = append(otherWorkers, w)
		}
	}

	if len(backendWorkers) > 0 {
		fmt.Println("## Backend Workers")
		printWorkerTable(backendWorkers)
		fmt.Println()
	}

	if len(frontendWorkers) > 0 {
		fmt.Println("## Frontend Workers")
		printWorkerTable(frontendWorkers)
		fmt.Println()
	}

	if len(otherWorkers) > 0 {
		fmt.Println("## Other Workers")
		printWorkerTable(otherWorkers)
	}

	return nil
}

func printWorkerTable(workers []*worker.WorkerSpec) {
	fmt.Println("| ID | 이름 | 레이어 | 기술 |")
	fmt.Println("|---|---|---|---|")
	for _, w := range workers {
		tech := w.Agent.Tech.Language
		if len(w.Agent.Tech.Frameworks) > 0 {
			tech += " (" + strings.Join(w.Agent.Tech.Frameworks, ", ") + ")"
		}
		fmt.Printf("| %s | %s | %s | %s |\n",
			w.Agent.ID, w.Agent.Name, w.Agent.Layer, tech)
	}
}

func categorizeWorkerID(id string) string {
	backendIDs := []string{"entity", "cache", "document", "service", "router", "test-worker"}
	frontendIDs := []string{"engineer", "model", "ui", "e2e", "unit-tc"}

	for _, b := range backendIDs {
		if strings.Contains(id, b) {
			return "backend"
		}
	}
	for _, f := range frontendIDs {
		if strings.Contains(id, f) {
			return "frontend"
		}
	}
	return "other"
}

func runWorkerShow(cmd *cobra.Command, args []string) error {
	workerID := args[0]

	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	mapper := worker.NewMapper(projectRoot)
	w, err := mapper.GetWorker(workerID)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(w.Agent)
	}

	fmt.Printf("# %s\n\n", w.Agent.Name)
	fmt.Printf("- **ID**: %s\n", w.Agent.ID)
	fmt.Printf("- **레이어**: %s\n", w.Agent.Layer)
	fmt.Printf("- **타입**: %s\n", w.Agent.Type)
	fmt.Printf("- **언어**: %s\n", w.Agent.Tech.Language)
	fmt.Printf("- **프레임워크**: %s\n", strings.Join(w.Agent.Tech.Frameworks, ", "))
	fmt.Println()

	if w.Agent.Description != "" {
		fmt.Printf("## 설명\n\n%s\n\n", w.Agent.Description)
	}

	if len(w.Agent.Responsibilities) > 0 {
		fmt.Println("## 책임")
		for _, r := range w.Agent.Responsibilities {
			fmt.Printf("- %s\n", r)
		}
		fmt.Println()
	}

	if len(w.Agent.Checklist) > 0 {
		fmt.Println("## 체크리스트")
		for _, c := range w.Agent.Checklist {
			fmt.Printf("- [ ] %s\n", c)
		}
		fmt.Println()
	}

	if len(w.Agent.PortTypes) > 0 {
		fmt.Println("## 포트 타입")
		for _, p := range w.Agent.PortTypes {
			fmt.Printf("- %s\n", p)
		}
		fmt.Println()
	}

	if w.Agent.ConventionsRef != "" {
		fmt.Printf("## 컨벤션\n\n`%s`\n", w.Agent.ConventionsRef)
	}

	return nil
}

func runWorkerSwitch(cmd *cobra.Command, args []string) error {
	workerID := args[0]

	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	claudeSvc := context.NewClaudeService(database, projectRoot)

	if err := claudeSvc.SwitchWorker(workerID, workerPortID); err != nil {
		return err
	}

	// Get worker details
	mapper := worker.NewMapper(projectRoot)
	w, _ := mapper.GetWorker(workerID)

	if jsonOut {
		output := map[string]interface{}{
			"worker_id": workerID,
			"port_id":   workerPortID,
			"status":    "switched",
		}
		if w != nil {
			output["worker_name"] = w.Agent.Name
		}
		return json.NewEncoder(os.Stdout).Encode(output)
	}

	if w != nil {
		fmt.Printf("✅ 워커 전환: %s (%s)\n", w.Agent.Name, workerID)
		fmt.Printf("   레이어: %s\n", w.Agent.Layer)
		if len(w.Agent.Checklist) > 0 {
			fmt.Printf("   체크리스트: %d 항목\n", len(w.Agent.Checklist))
		}
	} else {
		fmt.Printf("✅ 워커 전환: %s\n", workerID)
	}

	return nil
}

func runWorkerMap(cmd *cobra.Command, args []string) error {
	portID := args[0]

	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	// Load port spec
	portSpec, err := context.ReadPortSpec(portID, projectRoot+"/ports")
	if err != nil {
		return err
	}

	// Parse hints
	hints := worker.ParsePortSpecHints(portSpec)

	// Map to worker
	mapper := worker.NewMapper(projectRoot)
	workerID, err := mapper.MapPortToWorker(hints)
	if err != nil {
		return err
	}

	w, _ := mapper.GetWorker(workerID)

	if jsonOut {
		output := map[string]interface{}{
			"port_id":    portID,
			"worker_id":  workerID,
			"port_types": hints.PortTypes,
			"layer":      hints.Layer,
			"tech":       hints.Tech,
		}
		if w != nil {
			output["worker_name"] = w.Agent.Name
		}
		return json.NewEncoder(os.Stdout).Encode(output)
	}

	fmt.Printf("🔍 포트 분석: %s\n", portID)
	fmt.Printf("   포트 타입: %v\n", hints.PortTypes)
	if hints.Layer != "" {
		fmt.Printf("   레이어: %s\n", hints.Layer)
	}
	fmt.Println()

	if w != nil {
		fmt.Printf("✅ 추천 워커: %s (%s)\n", w.Agent.Name, workerID)
		fmt.Printf("   레이어: %s\n", w.Agent.Layer)
		fmt.Printf("   기술: %s (%s)\n", w.Agent.Tech.Language, strings.Join(w.Agent.Tech.Frameworks, ", "))
	} else {
		fmt.Printf("✅ 추천 워커: %s\n", workerID)
	}

	return nil
}
