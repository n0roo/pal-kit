package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

func init() {
	rootCmd.AddCommand(statusCmd)
}

// StatusSummary holds all status information
type StatusSummary struct {
	Sessions     SessionStatus     `json:"sessions"`
	Ports        PortStatus        `json:"ports"`
	Pipelines    PipelineStatus    `json:"pipelines"`
	Locks        []lock.Lock       `json:"locks"`
	Escalations  EscalationStatus  `json:"escalations"`
}

type SessionStatus struct {
	Active int              `json:"active"`
	Total  int              `json:"total"`
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

	summary := StatusSummary{}

	// 세션 현황
	activeSessions, _ := sessionSvc.List(true, 10)
	allSessions, _ := sessionSvc.List(false, 100)
	summary.Sessions = SessionStatus{
		Active: len(activeSessions),
		Total:  len(allSessions),
		List:   activeSessions,
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
			fmt.Printf("   %s %s%s\n", emoji, title, portInfo)
		}
	}
	fmt.Println()

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
	fmt.Printf("💡 Tip: pal session tree, pal port list, pal pl show <id>\n")

	return nil
}
