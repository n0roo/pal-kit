package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/session"
	"github.com/spf13/cobra"
)

var (
	sessionPortID string
	sessionTitle  string
	sessionStatus string
	sessionActive bool
	sessionLimit  int
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "세션 관리",
	Long:  `작업 세션을 관리합니다.`,
}

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "세션 시작",
	RunE:  runSessionStart,
}

var sessionEndCmd = &cobra.Command{
	Use:   "end [id]",
	Short: "세션 종료",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSessionEnd,
}

var sessionUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "세션 업데이트",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionUpdate,
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "세션 목록",
	RunE:  runSessionList,
}

var sessionShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "세션 상세",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionShow,
}

func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	sessionCmd.AddCommand(sessionUpdateCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)

	sessionStartCmd.Flags().StringVar(&sessionPortID, "port", "", "포트 ID")
	sessionStartCmd.Flags().StringVar(&sessionTitle, "title", "", "세션 제목")

	sessionUpdateCmd.Flags().StringVar(&sessionStatus, "status", "", "상태 (running|complete|failed|cancelled)")

	sessionListCmd.Flags().BoolVar(&sessionActive, "active", false, "활성 세션만")
	sessionListCmd.Flags().IntVar(&sessionLimit, "limit", 20, "결과 수 제한")
}

func getSessionService() (*session.Service, func(), error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, err
	}
	return session.NewService(database), func() { database.Close() }, nil
}

func runSessionStart(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getSessionService()
	if err != nil {
		return err
	}
	defer cleanup()

	// 세션 ID 생성 또는 환경변수에서 가져오기
	sessionID := os.Getenv("CLAUDE_SESSION_ID")
	if sessionID == "" {
		sessionID = uuid.New().String()[:8]
	}

	if err := svc.Start(sessionID, sessionPortID, sessionTitle); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":  "started",
			"id":      sessionID,
			"port_id": sessionPortID,
			"title":   sessionTitle,
		})
	} else {
		fmt.Printf("✓ 세션 시작: %s\n", sessionID)
		if sessionPortID != "" {
			fmt.Printf("  포트: %s\n", sessionPortID)
		}
		if sessionTitle != "" {
			fmt.Printf("  제목: %s\n", sessionTitle)
		}
	}

	return nil
}

func runSessionEnd(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getSessionService()
	if err != nil {
		return err
	}
	defer cleanup()

	sessionID := ""
	if len(args) > 0 {
		sessionID = args[0]
	} else {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	if sessionID == "" {
		return fmt.Errorf("세션 ID가 필요합니다")
	}

	if err := svc.End(sessionID); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "ended",
			"id":     sessionID,
		})
	} else {
		fmt.Printf("✓ 세션 종료: %s\n", sessionID)
	}

	return nil
}

func runSessionUpdate(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	if sessionStatus == "" {
		return fmt.Errorf("--status 플래그가 필요합니다")
	}

	svc, cleanup, err := getSessionService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.UpdateStatus(sessionID, sessionStatus); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":     "updated",
			"id":         sessionID,
			"new_status": sessionStatus,
		})
	} else {
		fmt.Printf("✓ 세션 업데이트: %s → %s\n", sessionID, sessionStatus)
	}

	return nil
}

func runSessionList(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getSessionService()
	if err != nil {
		return err
	}
	defer cleanup()

	sessions, err := svc.List(sessionActive, sessionLimit)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"sessions": sessions,
		})
		return nil
	}

	if len(sessions) == 0 {
		fmt.Println("세션이 없습니다.")
		return nil
	}

	fmt.Printf("%-12s %-12s %-20s %-10s %s\n", "ID", "PORT", "TITLE", "STATUS", "STARTED")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, s := range sessions {
		portID := "-"
		if s.PortID.Valid {
			portID = s.PortID.String
		}
		title := "-"
		if s.Title.Valid {
			title = s.Title.String
			if len(title) > 20 {
				title = title[:17] + "..."
			}
		}
		fmt.Printf("%-12s %-12s %-20s %-10s %s\n",
			s.ID, portID, title, s.Status, s.StartedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

func runSessionShow(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	svc, cleanup, err := getSessionService()
	if err != nil {
		return err
	}
	defer cleanup()

	sess, err := svc.Get(sessionID)
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(sess)
		return nil
	}

	fmt.Printf("세션: %s\n", sess.ID)
	fmt.Println("----------------------------------------")
	fmt.Printf("상태: %s\n", sess.Status)
	if sess.PortID.Valid {
		fmt.Printf("포트: %s\n", sess.PortID.String)
	}
	if sess.Title.Valid {
		fmt.Printf("제목: %s\n", sess.Title.String)
	}
	fmt.Printf("시작: %s\n", sess.StartedAt.Format("2006-01-02 15:04:05"))
	if sess.EndedAt.Valid {
		fmt.Printf("종료: %s\n", sess.EndedAt.Time.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()
	fmt.Printf("토큰 사용량:\n")
	fmt.Printf("  입력: %d\n", sess.InputTokens)
	fmt.Printf("  출력: %d\n", sess.OutputTokens)
	fmt.Printf("  캐시 읽기: %d\n", sess.CacheReadTokens)
	fmt.Printf("  캐시 생성: %d\n", sess.CacheCreateTokens)
	fmt.Printf("  비용: $%.4f\n", sess.CostUSD)
	fmt.Println()
	fmt.Printf("컴팩션: %d회\n", sess.CompactCount)

	return nil
}
