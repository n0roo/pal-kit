package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/agent"
	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/escalation"
	"github.com/n0roo/pal-kit/internal/lock"
	"github.com/n0roo/pal-kit/internal/pipeline"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/session"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "통합 상태 조회",
	Long: `프로젝트의 현재 상태를 한눈에 조회합니다.

세션, 포트, 파이프라인, Lock, 에스컬레이션 현황을 보여줍니다.`,
	RunE: runStatus,
}

var statusDetailedFlag bool

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&statusDetailedFlag, "detailed", "d", false, "상세 정보 표시 (토큰, 시간)")
}

// StatusSummary holds all status information
type StatusSummary struct {
	Sessions    SessionStatus    `json:"sessions"`
	Ports       PortStatus       `json:"ports"`
	Pipelines   PipelineStatus   `json:"pipelines"`
	Locks       []lock.Lock      `json:"locks"`
	Escalations EscalationStatus `json:"escalations"`
	Agents      AgentStatus      `json:"agents"`
	TotalUsage  UsageSummary     `json:"total_usage"`
}

type SessionStatus struct {
	Active int               `json:"active"`
	Total  int               `json:"total"`
	List   []session.Session `json:"list,omitempty"`
}

type PortStatus struct {
	Summary map[string]int `json:"summary"`
	Running []port.Port    `json:"running,omitempty"`
}

type PipelineStatus struct {
	Active int                 `json:"active"`
	Total  int                 `json:"total"`
	List   []pipeline.Pipeline `json:"list,omitempty"`
}

type EscalationStatus struct {
	Open     int `json:"open"`
	Resolved int `json:"resolved"`
	Total    int `json:"total"`
}

type AgentStatus struct {
	Count int      `json:"count"`
	Types []string `json:"types,omitempty"`
}

type UsageSummary struct {
	TotalInputTokens  int64   `json:"total_input_tokens"`
	TotalOutputTokens int64   `json:"total_output_tokens"`
	TotalCacheRead    int64   `json:"total_cache_read"`
	TotalCacheCreate  int64   `json:"total_cache_create"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	TotalSessions     int     `json:"total_sessions"`
	TotalDuration     string  `json:"total_duration"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	// 서비스 생성
	sessionSvc := session.NewService(database)
	portSvc := port.NewService(database)
	pipelineSvc := pipeline.NewService(database)
	lockSvc := lock.NewService(database)
	escSvc := escalation.NewService(database)

	// 에이전트 서비스
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		projectRoot = cwd
	}
	agentSvc := agent.NewService(projectRoot)

	summary := StatusSummary{}

	// 세션 현황
	activeSessions, _ := sessionSvc.List(true, 10)
	allSessions, _ := sessionSvc.List(false, 100)
	summary.Sessions = SessionStatus{
		Active: len(activeSessions),
		Total:  len(allSessions),
		List:   activeSessions,
	}

	// 토큰 사용량 집계
	var totalInput, totalOutput, totalCacheRead, totalCacheCreate int64
	var totalCost float64
	var totalDuration time.Duration
	for _, s := range allSessions {
		totalInput += s.InputTokens
		totalOutput += s.OutputTokens
		totalCacheRead += s.CacheReadTokens
		totalCacheCreate += s.CacheCreateTokens
		totalCost += s.CostUSD
		if s.EndedAt.Valid {
			totalDuration += s.EndedAt.Time.Sub(s.StartedAt)
		} else if s.Status == "running" {
			totalDuration += time.Since(s.StartedAt)
		}
	}
	summary.TotalUsage = UsageSummary{
		TotalInputTokens:  totalInput,
		TotalOutputTokens: totalOutput,
		TotalCacheRead:    totalCacheRead,
		TotalCacheCreate:  totalCacheCreate,
		TotalCostUSD:      totalCost,
		TotalSessions:     len(allSessions),
		TotalDuration:     formatDuration(totalDuration),
	}

	// 포트 현황
	portSummary, _ := portSvc.Summary()
	runningPorts, _ := portSvc.List("running", 10)
	summary.Ports = PortStatus{
		Summary: portSummary,
		Running: runningPorts,
	}

	// 파이프라인 현황
	activePipelines, _ := pipelineSvc.List("running", 10)
	allPipelines, _ := pipelineSvc.List("", 100)
	summary.Pipelines = PipelineStatus{
		Active: len(activePipelines),
		Total:  len(allPipelines),
		List:   activePipelines,
	}

	// Lock 현황
	locks, _ := lockSvc.List()
	summary.Locks = locks

	// 에스컬레이션 현황
	escSummary, _ := escSvc.Summary()
	summary.Escalations = EscalationStatus{
		Open:     escSummary["open"],
		Resolved: escSummary["resolved"],
		Total:    escSummary["open"] + escSummary["resolved"] + escSummary["dismissed"],
	}

	// 에이전트 현황
	agents, _ := agentSvc.List()
	agentTypes := make(map[string]bool)
	for _, a := range agents {
		agentTypes[a.Type] = true
	}
	var types []string
	for t := range agentTypes {
		types = append(types, t)
	}
	summary.Agents = AgentStatus{
		Count: len(agents),
		Types: types,
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(summary)
		return nil
	}

	// 헤더
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      PAL Status Dashboard                     ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 세션 섹션
	fmt.Printf("📍 Sessions: %d active / %d total\n", summary.Sessions.Active, summary.Sessions.Total)
	if len(summary.Sessions.List) > 0 {
		for _, s := range summary.Sessions.List {
			typeEmoji := map[string]string{
				"single": "📍", "multi": "🔀", "sub": "📎", "builder": "🏗️",
			}
			emoji := typeEmoji[s.SessionType]
			if emoji == "" {
				emoji = "📍"
			}
			title := s.ID
			if s.Title.Valid {
				title = s.Title.String
			}
			portInfo := ""
			if s.PortID.Valid {
				portInfo = fmt.Sprintf(" [%s]", s.PortID.String)
			}

			// 기본 정보
			duration := time.Since(s.StartedAt)
			fmt.Printf("   %s %s%s (%s)\n", emoji, title, portInfo, formatDuration(duration))

			// 상세 모드: 토큰 정보
			if statusDetailedFlag {
				fmt.Printf("      ├─ 시작: %s\n", s.StartedAt.Format("2006-01-02 15:04:05"))
				if s.InputTokens > 0 || s.OutputTokens > 0 {
					fmt.Printf("      ├─ 토큰: in=%s, out=%s", formatTokens(s.InputTokens), formatTokens(s.OutputTokens))
					if s.CacheReadTokens > 0 {
						fmt.Printf(", cache=%s", formatTokens(s.CacheReadTokens))
					}
					fmt.Println()
				}
				if s.CostUSD > 0 {
					fmt.Printf("      ├─ 비용: $%.4f\n", s.CostUSD)
				}
				if s.CompactCount > 0 {
					fmt.Printf("      └─ 컴팩션: %d회\n", s.CompactCount)
				}
			}
		}
	}
	fmt.Println()

	// 총 사용량 (상세 모드)
	if statusDetailedFlag && summary.TotalUsage.TotalInputTokens > 0 {
		fmt.Println("📊 Total Usage:")
		fmt.Printf("   토큰: in=%s, out=%s\n",
			formatTokens(summary.TotalUsage.TotalInputTokens),
			formatTokens(summary.TotalUsage.TotalOutputTokens))
		if summary.TotalUsage.TotalCacheRead > 0 {
			fmt.Printf("   캐시: read=%s, create=%s\n",
				formatTokens(summary.TotalUsage.TotalCacheRead),
				formatTokens(summary.TotalUsage.TotalCacheCreate))
		}
		fmt.Printf("   비용: $%.4f\n", summary.TotalUsage.TotalCostUSD)
		fmt.Printf("   시간: %s\n", summary.TotalUsage.TotalDuration)
		fmt.Println()
	}

	// 포트 섹션
	fmt.Println("📦 Ports:")
	if len(summary.Ports.Summary) == 0 {
		fmt.Println("   (없음)")
	} else {
		statusEmoji := map[string]string{
			"pending": "⏳", "running": "🔄", "complete": "✅", "failed": "❌", "blocked": "🚫",
		}
		statusOrder := []string{"running", "pending", "complete", "failed", "blocked"}
		for _, status := range statusOrder {
			count := summary.Ports.Summary[status]
			if count > 0 {
				fmt.Printf("   %s %s: %d\n", statusEmoji[status], status, count)
			}
		}
	}
	if len(summary.Ports.Running) > 0 {
		fmt.Println("   ─────────────────────")
		for _, p := range summary.Ports.Running {
			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}
			fmt.Printf("   🔄 %s: %s\n", p.ID, title)
		}
	}
	fmt.Println()

	// 파이프라인 섹션
	fmt.Printf("🔀 Pipelines: %d active / %d total\n", summary.Pipelines.Active, summary.Pipelines.Total)
	if len(summary.Pipelines.List) > 0 {
		for _, p := range summary.Pipelines.List {
			completed, total, _ := pipelineSvc.GetProgress(p.ID)
			fmt.Printf("   🔄 %s (%d/%d)\n", p.Name, completed, total)
		}
	}
	fmt.Println()

	// 에이전트 섹션
	if summary.Agents.Count > 0 {
		fmt.Printf("🤖 Agents: %d 등록됨\n", summary.Agents.Count)
		if len(summary.Agents.Types) > 0 {
			fmt.Printf("   타입: %s\n", strings.Join(summary.Agents.Types, ", "))
		}
		fmt.Println()
	}

	// Lock 섹션
	fmt.Printf("🔒 Locks: %d active\n", len(summary.Locks))
	if len(summary.Locks) > 0 {
		for _, l := range summary.Locks {
			fmt.Printf("   🔐 %s (by %s)\n", l.Resource, l.SessionID)
		}
	}
	fmt.Println()

	// 에스컬레이션 섹션
	if summary.Escalations.Open > 0 {
		fmt.Printf("🚨 Escalations: %d open\n", summary.Escalations.Open)
		openEsc, _ := escSvc.List("open", 5)
		for _, e := range openEsc {
			issue := e.Issue
			if len(issue) > 50 {
				issue = issue[:47] + "..."
			}
			fmt.Printf("   ⚠️  #%d: %s\n", e.ID, issue)
		}
	} else {
		fmt.Println("✅ Escalations: 없음")
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 64))
	fmt.Printf("💡 Tip: pal status -d (상세), pal session show <id>\n")

	return nil
}

// formatDuration formats duration in human readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}

// formatTokens formats token count with K/M suffix
func formatTokens(tokens int64) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	if tokens < 1000000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%.2fM", float64(tokens)/1000000)
}
