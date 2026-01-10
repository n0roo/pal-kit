package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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
	sessionType   string
	sessionParent string
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "세션 관리",
	Long:  `작업 세션을 관리합니다.`,
}

var sessionStartCmd = &cobra.Command{
	Use:   "start",
	Short: "세션 시작",
	Long: `새 세션을 시작합니다.

세션 유형:
  single  - 단일 세션 (기본)
  multi   - 멀티 세션 (병렬 독립)
  sub     - 서브 세션 (상위에서 spawn)
  builder - 빌더 세션 (파이프라인 관리)`,
	RunE: runSessionStart,
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

var sessionTreeCmd = &cobra.Command{
	Use:   "tree [id]",
	Short: "세션 트리 조회",
	Long: `세션의 계층 구조를 트리 형태로 조회합니다.
ID를 지정하지 않으면 모든 루트 세션의 트리를 출력합니다.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSessionTree,
}

func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(sessionStartCmd)
	sessionCmd.AddCommand(sessionEndCmd)
	sessionCmd.AddCommand(sessionUpdateCmd)
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)
	sessionCmd.AddCommand(sessionTreeCmd)

	sessionStartCmd.Flags().StringVar(&sessionPortID, "port", "", "포트 ID")
	sessionStartCmd.Flags().StringVar(&sessionTitle, "title", "", "세션 제목")
	sessionStartCmd.Flags().StringVar(&sessionType, "type", "single", "세션 유형 (single|multi|sub|builder)")
	sessionStartCmd.Flags().StringVar(&sessionParent, "parent", "", "상위 세션 ID")

	sessionUpdateCmd.Flags().StringVar(&sessionStatus, "status", "", "상태 (running|complete|failed|cancelled)")

	sessionListCmd.Flags().BoolVar(&sessionActive, "active", false, "활성 세션만")
	sessionListCmd.Flags().IntVar(&sessionLimit, "limit", 20, "결과 수 제한")

	sessionTreeCmd.Flags().IntVar(&sessionLimit, "limit", 10, "루트 세션 수 제한")
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

	// 유효한 타입인지 확인
	validTypes := map[string]bool{
		session.TypeSingle:  true,
		session.TypeMulti:   true,
		session.TypeSub:     true,
		session.TypeBuilder: true,
	}
	if !validTypes[sessionType] {
		return fmt.Errorf("유효하지 않은 세션 유형: %s", sessionType)
	}

	// sub 타입은 parent가 필요
	if sessionType == session.TypeSub && sessionParent == "" {
		return fmt.Errorf("sub 세션은 --parent가 필요합니다")
	}

	// 포트 존재 여부 미리 확인 (경고용)
	portWarning := ""
	if sessionPortID != "" {
		database, _ := db.Open(GetDBPath())
		var exists int
		err := database.QueryRow(`SELECT 1 FROM ports WHERE id = ?`, sessionPortID).Scan(&exists)
		database.Close()
		if err != nil {
			portWarning = fmt.Sprintf("⚠️  포트 '%s'가 존재하지 않습니다 (pal port create %s)", sessionPortID, sessionPortID)
		}
	}

	if err := svc.StartWithOptions(sessionID, sessionPortID, sessionTitle, sessionType, sessionParent); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":  "started",
			"id":      sessionID,
			"port_id": sessionPortID,
			"title":   sessionTitle,
			"type":    sessionType,
			"parent":  sessionParent,
		})
	} else {
		typeEmoji := map[string]string{
			"single":  "📍",
			"multi":   "🔀",
			"sub":     "📎",
			"builder": "🏗️",
		}
		fmt.Printf("%s 세션 시작: %s\n", typeEmoji[sessionType], sessionID)
		fmt.Printf("  유형: %s\n", sessionType)
		if sessionParent != "" {
			fmt.Printf("  상위: %s\n", sessionParent)
		}
		if sessionPortID != "" {
			fmt.Printf("  포트: %s\n", sessionPortID)
		}
		if sessionTitle != "" {
			fmt.Printf("  제목: %s\n", sessionTitle)
		}
		if portWarning != "" {
			fmt.Println(portWarning)
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

	fmt.Printf("%-10s %-8s %-10s %-18s %-10s %s\n", "ID", "TYPE", "PORT", "TITLE", "STATUS", "STARTED")
	fmt.Println(strings.Repeat("-", 85))

	typeEmoji := map[string]string{
		"single":  "📍",
		"multi":   "🔀",
		"sub":     "📎",
		"builder": "🏗️",
	}

	for _, s := range sessions {
		portID := "-"
		if s.PortID.Valid {
			portID = s.PortID.String
		}
		title := "-"
		if s.Title.Valid {
			title = s.Title.String
			if len(title) > 18 {
				title = title[:15] + "..."
			}
		}
		emoji := typeEmoji[s.SessionType]
		if emoji == "" {
			emoji = "📍"
		}
		fmt.Printf("%-10s %s %-6s %-10s %-18s %-10s %s\n",
			s.ID, emoji, s.SessionType, portID, title, s.Status, s.StartedAt.Format("01-02 15:04"))
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

	typeEmoji := map[string]string{
		"single":  "📍",
		"multi":   "🔀",
		"sub":     "📎",
		"builder": "🏗️",
	}

	emoji := typeEmoji[sess.SessionType]
	if emoji == "" {
		emoji = "📍"
	}

	fmt.Printf("%s 세션: %s\n", emoji, sess.ID)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("유형: %s\n", sess.SessionType)
	fmt.Printf("상태: %s\n", sess.Status)
	if sess.ParentSession.Valid {
		fmt.Printf("상위: %s\n", sess.ParentSession.String)
	}
	if sess.PortID.Valid {
		fmt.Printf("포트: %s\n", sess.PortID.String)
	}
	if sess.Title.Valid {
		fmt.Printf("제목: %s\n", sess.Title.String)
	}

	fmt.Println()
	fmt.Println("⏱️  시간 정보:")
	fmt.Printf("  시작: %s\n", sess.StartedAt.Format("2006-01-02 15:04:05"))
	if sess.EndedAt.Valid {
		fmt.Printf("  종료: %s\n", sess.EndedAt.Time.Format("2006-01-02 15:04:05"))
		duration := sess.EndedAt.Time.Sub(sess.StartedAt)
		fmt.Printf("  체류: %s\n", formatDuration(duration))
	} else if sess.Status == "running" {
		duration := time.Since(sess.StartedAt)
		fmt.Printf("  체류: %s (진행 중)\n", formatDuration(duration))
	}

	fmt.Println()
	fmt.Println("📊 토큰 사용량:")
	totalTokens := sess.InputTokens + sess.OutputTokens
	if totalTokens > 0 {
		fmt.Printf("  입력:      %s\n", formatTokens(sess.InputTokens))
		fmt.Printf("  출력:      %s\n", formatTokens(sess.OutputTokens))
		fmt.Printf("  합계:      %s\n", formatTokens(totalTokens))
		if sess.CacheReadTokens > 0 || sess.CacheCreateTokens > 0 {
			fmt.Printf("  캐시 읽기: %s\n", formatTokens(sess.CacheReadTokens))
			fmt.Printf("  캐시 생성: %s\n", formatTokens(sess.CacheCreateTokens))
		}
		fmt.Printf("  비용:      $%.4f\n", sess.CostUSD)
	} else {
		fmt.Println("  (사용량 없음)")
	}

	if sess.CompactCount > 0 {
		fmt.Println()
		fmt.Printf("📦 컴팩션: %d회\n", sess.CompactCount)
		if sess.LastCompactAt.Valid {
			fmt.Printf("  마지막: %s\n", sess.LastCompactAt.Time.Format("2006-01-02 15:04:05"))
		}
	}

	// 하위 세션 조회
	children, _ := svc.GetChildren(sess.ID)
	if len(children) > 0 {
		fmt.Println()
		fmt.Printf("👥 하위 세션: %d개\n", len(children))
		for _, child := range children {
			childEmoji := typeEmoji[child.SessionType]
			if childEmoji == "" {
				childEmoji = "📍"
			}
			title := "-"
			if child.Title.Valid {
				title = child.Title.String
			}
			statusEmoji := map[string]string{
				"running":  "🔄",
				"complete": "✅",
				"failed":   "❌",
			}
			sEmoji := statusEmoji[child.Status]
			if sEmoji == "" {
				sEmoji = "⏳"
			}
			fmt.Printf("  %s %s %s: %s\n", childEmoji, sEmoji, child.ID, title)
		}
	}

	return nil
}

func runSessionTree(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getSessionService()
	if err != nil {
		return err
	}
	defer cleanup()

	if len(args) > 0 {
		// 특정 세션의 트리 출력
		tree, err := svc.GetTree(args[0])
		if err != nil {
			return err
		}

		if jsonOut {
			json.NewEncoder(os.Stdout).Encode(tree)
			return nil
		}

		printSessionTree(tree, "", true, true)
	} else {
		// 모든 루트 세션의 트리 출력
		roots, err := svc.GetRootSessions(sessionLimit)
		if err != nil {
			return err
		}

		if jsonOut {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"roots": roots,
			})
			return nil
		}

		if len(roots) == 0 {
			fmt.Println("세션이 없습니다.")
			return nil
		}

		fmt.Printf("세션 트리 (루트 %d개)\n", len(roots))
		fmt.Println(strings.Repeat("=", 50))

		for i, root := range roots {
			tree, err := svc.GetTree(root.ID)
			if err != nil {
				continue
			}
			printSessionTree(tree, "", i == len(roots)-1, true)
			if i < len(roots)-1 {
				fmt.Println()
			}
		}
	}

	return nil
}

func printSessionTree(node *session.SessionNode, prefix string, isLast bool, isRoot bool) {
	typeEmoji := map[string]string{
		"single":  "📍",
		"multi":   "🔀",
		"sub":     "📎",
		"builder": "🏗️",
	}
	statusEmoji := map[string]string{
		"running":   "🔄",
		"complete":  "✅",
		"failed":    "❌",
		"cancelled": "⚪",
	}

	emoji := typeEmoji[node.Session.SessionType]
	if emoji == "" {
		emoji = "📍"
	}

	status := statusEmoji[node.Session.Status]
	if status == "" {
		status = "⏳"
	}

	title := node.Session.ID
	if node.Session.Title.Valid && node.Session.Title.String != "" {
		title = node.Session.Title.String
	}

	portInfo := ""
	if node.Session.PortID.Valid {
		portInfo = fmt.Sprintf(" [%s]", node.Session.PortID.String)
	}

	// 현재 노드 출력
	if isRoot {
		fmt.Printf("%s %s %s%s\n", emoji, status, title, portInfo)
	} else {
		connector := "├─"
		if isLast {
			connector = "└─"
		}
		fmt.Printf("%s%s %s %s %s%s\n", prefix, connector, emoji, status, title, portInfo)
	}

	// 하위 노드를 위한 prefix 계산
	var childPrefix string
	if isRoot {
		childPrefix = ""
	} else if isLast {
		childPrefix = prefix + "   "
	} else {
		childPrefix = prefix + "│  "
	}

	// 하위 노드 출력
	for i, child := range node.Children {
		printSessionTree(&child, childPrefix, i == len(node.Children)-1, false)
	}
}
