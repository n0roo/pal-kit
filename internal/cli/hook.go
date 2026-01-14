package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/document"
	"github.com/n0roo/pal-kit/internal/lock"
	"github.com/n0roo/pal-kit/internal/manifest"
	"github.com/n0roo/pal-kit/internal/operator"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/rules"
	"github.com/n0roo/pal-kit/internal/session"
	"github.com/n0roo/pal-kit/internal/transcript"
	"github.com/n0roo/pal-kit/internal/workflow"
	"github.com/spf13/cobra"
)

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// escapeJSON escapes special characters for JSON string
func escapeJSON(s string) string {
	// 간단한 JSON 이스케이프 (줄바꿈, 탭, 큰따옴표)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

// HookInput represents the JSON input from Claude Code hooks
// Based on Claude Code Hook specification
type HookInput struct {
	// Common fields
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`

	// SessionStart specific
	Source string `json:"source,omitempty"` // "startup"

	// SessionEnd specific
	Reason string `json:"reason,omitempty"` // "exit", "clear", "logout", "prompt_input_exit", "other"

	// Stop/SubagentStop specific
	StopHookActive bool `json:"stop_hook_active,omitempty"`

	// PreToolUse/PostToolUse specific
	ToolName     string                 `json:"tool_name,omitempty"`
	ToolInput    map[string]interface{} `json:"tool_input,omitempty"`
	ToolResponse map[string]interface{} `json:"tool_response,omitempty"`
	ToolUseID    string                 `json:"tool_use_id,omitempty"`

	// PreCompact specific
	Trigger            string `json:"trigger,omitempty"` // "manual" or "auto"
	CustomInstructions string `json:"custom_instructions,omitempty"`

	// Notification specific
	Message          string `json:"message,omitempty"`
	NotificationType string `json:"notification_type,omitempty"`
}

// HookOutput represents JSON output for hook responses
type HookOutput struct {
	Decision   string                 `json:"decision,omitempty"` // "approve", "block", "allow", "deny", "ask"
	Reason     string                 `json:"reason,omitempty"`
	Continue   bool                   `json:"continue,omitempty"`
	StopReason string                 `json:"stopReason,omitempty"`
	HookOutput map[string]interface{} `json:"hookSpecificOutput,omitempty"`
}

var (
	hookPortID string
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook 지원",
	Long:  `Claude Code Hook에서 호출되는 커맨드입니다.`,
}

var hookSessionStartCmd = &cobra.Command{
	Use:   "session-start",
	Short: "SessionStart Hook",
	Long: `세션 시작 시 호출됩니다.

수행 작업:
- 세션 등록
- CLAUDE.md 컨텍스트 주입
- 활성 포트 rules 확인`,
	RunE: runHookSessionStart,
}

var hookSessionEndCmd = &cobra.Command{
	Use:   "session-end",
	Short: "SessionEnd Hook",
	Long: `세션 종료 시 호출됩니다.

수행 작업:
- 세션 종료 처리
- Lock 자동 해제
- running 포트 정리`,
	RunE: runHookSessionEnd,
}

var hookPreToolUseCmd = &cobra.Command{
	Use:   "pre-tool-use",
	Short: "PreToolUse Hook",
	RunE:  runHookPreToolUse,
}

var hookPostToolUseCmd = &cobra.Command{
	Use:   "post-tool-use",
	Short: "PostToolUse Hook",
	RunE:  runHookPostToolUse,
}

var hookStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Hook",
	RunE:  runHookStop,
}

var hookPreCompactCmd = &cobra.Command{
	Use:   "pre-compact",
	Short: "PreCompact Hook",
	RunE:  runHookPreCompact,
}

var hookPortStartCmd = &cobra.Command{
	Use:   "port-start <port-id>",
	Short: "포트 작업 시작 Hook",
	Long: `포트 작업 시작 시 호출됩니다.

수행 작업:
- 포트 상태를 running으로 변경
- rules 파일 생성
- Lock 획득 (리소스 지정 시)`,
	Args: cobra.ExactArgs(1),
	RunE: runHookPortStart,
}

var hookPortEndCmd = &cobra.Command{
	Use:   "port-end <port-id>",
	Short: "포트 작업 완료 Hook",
	Long: `포트 작업 완료 시 호출됩니다.

수행 작업:
- 포트 상태를 complete로 변경
- rules 파일 삭제
- Lock 해제`,
	Args: cobra.ExactArgs(1),
	RunE: runHookPortEnd,
}

var hookSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "상태 동기화",
	Long: `running 포트의 rules를 동기화합니다.

수행 작업:
- running 포트 조회
- 누락된 rules 파일 생성
- 불필요한 rules 파일 정리`,
	RunE: runHookSync,
}

var hookEventCmd = &cobra.Command{
	Use:   "event <type> <message>",
	Short: "이벤트 로깅",
	Long: `세션 이벤트를 기록합니다.

이벤트 타입:
- decision: 주요 결정 사항
- escalation: 에스컬레이션 (사용자 개입 필요)
- user_request: 사용자 요구사항

예시:
  pal hook event decision "JWT 인증 방식 채택"
  pal hook event escalation "DB 스키마 변경 승인 필요"`,
	Args: cobra.ExactArgs(2),
	RunE: runHookEvent,
}

var hookEventType string
var hookEventContext string

var hookEventsCmd = &cobra.Command{
	Use:   "events [session-id]",
	Short: "이벤트 조회",
	Long: `세션 이벤트를 조회합니다.

session-id를 지정하지 않으면 현재 활성 세션의 이벤트를 조회합니다.

예시:
  pal hook events            # 현재 세션 이벤트
  pal hook events abc123     # 특정 세션 이벤트`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHookEvents,
}

var hookEventsLimit int
var hookEventsTypeFilter string

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookSessionStartCmd)
	hookCmd.AddCommand(hookSessionEndCmd)
	hookCmd.AddCommand(hookPreToolUseCmd)
	hookCmd.AddCommand(hookPostToolUseCmd)
	hookCmd.AddCommand(hookStopCmd)
	hookCmd.AddCommand(hookPreCompactCmd)
	hookCmd.AddCommand(hookPortStartCmd)
	hookCmd.AddCommand(hookPortEndCmd)
	hookCmd.AddCommand(hookSyncCmd)
	hookCmd.AddCommand(hookEventCmd)
	hookCmd.AddCommand(hookEventsCmd)

	hookSessionStartCmd.Flags().StringVar(&hookPortID, "port", "", "시작할 포트 ID")
	hookEventCmd.Flags().StringVar(&hookEventContext, "context", "", "추가 컨텍스트 (JSON)")
	hookEventsCmd.Flags().IntVar(&hookEventsLimit, "limit", 20, "조회할 이벤트 수")
	hookEventsCmd.Flags().StringVar(&hookEventsTypeFilter, "type", "", "이벤트 타입 필터")
}

func readHookInput() (*HookInput, error) {
	// stdin이 터미널이면 (파이프가 아니면) 빈 입력 반환
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return &HookInput{}, nil
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &HookInput{}, nil
	}

	var input HookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}

	return &input, nil
}

func runHookSessionStart(cmd *cobra.Command, args []string) error {
	input, err := readHookInput()
	if err != nil {
		input = &HookInput{}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	sessionSvc := session.NewService(database)
	portSvc := port.NewService(database)

	// 좀비 세션 정리 (24시간 이상 running 상태인 세션)
	cleanedCount, err := sessionSvc.CleanupZombieSessions(24)
	if err == nil && cleanedCount > 0 {
		if verbose {
			fmt.Printf("🧹 Cleaned %d zombie session(s)\n", cleanedCount)
		}
		// 좀비 정리 이벤트 로깅 (전역)
		sessionSvc.LogEvent("system", "zombie_cleanup", fmt.Sprintf(`{"cleaned":%d}`, cleanedCount))
	}

	// 프로젝트 루트 찾기
	cwd := input.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	projectRoot := context.FindProjectRoot(cwd)

	// 프로젝트 이름 추출 (디렉토리 이름)
	projectName := ""
	if projectRoot != "" {
		projectName = filepath.Base(projectRoot)
	}

	// Claude 세션 ID로 기존 세션 확인 (FindActiveSession 사용)
	var palSessionID string
	if input.SessionID != "" {
		existingSession, err := sessionSvc.FindByClaudeSessionID(input.SessionID)
		if err == nil && existingSession != nil {
			// 기존 세션 재사용
			palSessionID = existingSession.ID
			if verbose {
				fmt.Printf("♻️  Reusing existing session: %s\n", palSessionID)
			}
		}
	}

	// 기존 세션이 없으면 새로 생성
	if palSessionID == "" {
		palSessionID = uuid.New().String()[:8]

		// 동일 프로젝트에서 실행 중인 세션 수 확인
		var runningCount int
		var sessionType string
		if projectRoot != "" {
			runningCount, _ = sessionSvc.CountRunningByProject(projectRoot)
		}

		// 세션 타입 결정:
		// - 다른 실행 중인 세션이 없으면 main
		// - 있으면 sub (Builder가 spawn한 것으로 추정)
		if runningCount == 0 {
			sessionType = session.TypeMain
		} else {
			sessionType = session.TypeSub
			// 멀티 세션 경고
			fmt.Fprintf(os.Stderr, "⚠️  [PAL Kit] 이 프로젝트에 %d개의 다른 세션이 실행 중입니다.\n", runningCount)
			fmt.Fprintf(os.Stderr, "   현재 세션: %s (cwd: %s)\n", palSessionID, cwd)
		}

		// 세션 시작 (프로젝트 정보 포함)
		opts := session.StartOptions{
			ID:              palSessionID,
			PortID:          hookPortID,
			SessionType:     sessionType,
			ClaudeSessionID: input.SessionID, // Claude Code의 session_id
			ProjectRoot:     projectRoot,
			ProjectName:     projectName,
			TranscriptPath:  input.TranscriptPath,
			Cwd:             cwd,
		}

		if err := sessionSvc.StartWithFullOptions(opts); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "세션 시작: %v\n", err)
			}
		}

		// 메인 세션인 경우 명시적으로 표시
		if sessionType == session.TypeMain && verbose {
			fmt.Printf("🏠 Main session started: %s\n", palSessionID)
		}

		// 첫 번째 사용자 메시지 캡처 (user_request 이벤트)
		if input.TranscriptPath != "" {
			go func(sessID, transcriptPath string) {
				// transcript 파일이 아직 쓰이는 중일 수 있으므로 잠시 대기
				time.Sleep(500 * time.Millisecond)

				msg, err := transcript.GetFirstUserMessage(transcriptPath)
				if err == nil && msg != nil && msg.Content != "" {
					// 메시지를 500자로 제한
					content := truncateString(msg.Content, 500)
					eventData := fmt.Sprintf(`{"message":"%s","full_length":%d}`,
						escapeJSON(content), len(msg.Content))
					sessionSvc.LogEvent(sessID, session.EventUserRequest, eventData)
				}
			}(palSessionID, input.TranscriptPath)
		}
	}

	// CLAUDE.md에 컨텍스트 주입
	ctxSvc := context.NewService(database)
	claudeMD := context.FindClaudeMD(cwd)
	if claudeMD != "" {
		ctxSvc.InjectToFile(claudeMD)
		if verbose {
			fmt.Printf("📝 Context injected: %s\n", claudeMD)
		}
	}

	// 포트가 지정되었으면 활성화
	if hookPortID != "" && projectRoot != "" {
		rulesSvc := rules.NewService(projectRoot)
		
		p, err := portSvc.Get(hookPortID)
		if err == nil {
			title := hookPortID
			if p.Title.Valid {
				title = p.Title.String
			}
			specPath := ""
			if p.FilePath.Valid {
				specPath = p.FilePath.String
			}
			
			rulesSvc.ActivatePortWithSpec(hookPortID, title, specPath, nil)
			portSvc.UpdateStatus(hookPortID, "running")
			
			// 포트 시작 이벤트 로깅
			sessionSvc.LogEvent(palSessionID, "port_start", fmt.Sprintf(`{"port_id":"%s"}`, hookPortID))
			
			if verbose {
				fmt.Printf("✅ Port activated: %s\n", hookPortID)
			}
		}
	}

	// 현재 상태 요약
	if verbose {
		fmt.Printf("🚀 Session started: %s (claude: %s)\n", palSessionID, input.SessionID)
		fmt.Printf("   Project: %s\n", projectName)
		
		runningPorts, _ := portSvc.List("running", 10)
		if len(runningPorts) > 0 {
			fmt.Printf("🔄 Running ports: %d\n", len(runningPorts))
			for _, p := range runningPorts {
				fmt.Printf("   - %s\n", p.ID)
			}
		}
	}

	// Manifest 변경 감지 (가벼운 알림)
	if projectRoot != "" && config.IsInstalled() {
		manifestSvc := manifest.NewService(database, projectRoot)
		changedFiles, err := manifestSvc.QuickCheck()
		if err == nil && len(changedFiles) > 0 {
			fmt.Printf("💡 설정 파일이 변경되었습니다. `pal manifest status`로 확인해보세요.\n")
		}
	}

	// 빌더 에이전트 자동 활성화
	if projectRoot != "" {
		claudeSvc := context.NewClaudeService(database, projectRoot)
		builderResult, err := claudeSvc.ProcessSessionStart()
		if err == nil && builderResult.BuilderActive {
			if verbose {
				fmt.Printf("🏗️  Builder agent activated: %s\n", builderResult.BuilderName)
				fmt.Printf("   Rules: %s\n", builderResult.RulesFile)
				fmt.Printf("   Tokens: ~%d\n", builderResult.TokenCount)
			}
		}
	}

	// 워크플로우 컨텍스트 주입 (rules 파일로)
	if projectRoot != "" {
		workflowSvc := workflow.NewService(projectRoot)
		ctx, err := workflowSvc.GetContext()
		if err == nil {
			if err := workflowSvc.WriteRulesFile(ctx); err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "워크플로우 rules 작성 실패: %v\n", err)
				}
			} else if verbose {
				fmt.Printf("📝 Workflow context: %s (%s)\n", ctx.WorkflowType, workflowSvc.GetRulesPath())
			}
		}
	}

	// 세션명 제안 마커 출력 (Web UI에서 시작된 세션인 경우)
	// Builder 에이전트가 이 마커를 인식하고 세션명을 제안함
	if palSessionID != "" {
		palSession, err := sessionSvc.Get(palSessionID)
		if err == nil && palSession != nil {
			// 타이틀이 비어있거나 "-"이면 세션명 제안 필요
			needsName := !palSession.Title.Valid || palSession.Title.String == "" || palSession.Title.String == "-"
			if needsName {
				fmt.Println("<!-- pal:session:needs-name -->")
			}
		}
	}

	// Operator 브리핑 생성
	if projectRoot != "" {
		operatorSvc := operator.NewService(database, projectRoot)
		briefing, err := operatorSvc.GenerateBriefing()
		if err == nil {
			// .pal/context/session-briefing.md 저장
			if err := operatorSvc.WriteBriefing(briefing); err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "⚠️  브리핑 저장 실패: %v\n", err)
				}
			}

			// stdout으로 요약 출력 (Claude가 읽음)
			if briefing.Summary != "" && briefing.Summary != "No active work items." {
				fmt.Printf("📋 %s\n", briefing.Summary)
			}

			// 권장 사항 출력
			if len(briefing.Recommendations) > 0 && verbose {
				fmt.Printf("💡 추천: %s\n", briefing.Recommendations[0])
			}

			if verbose {
				fmt.Printf("📄 Briefing: %s\n", operatorSvc.GetBriefingPath())
			}
		} else if verbose {
			fmt.Fprintf(os.Stderr, "⚠️  브리핑 생성 실패: %v\n", err)
		}
	}

	// 포트 사용 안내 (Claude가 읽는 지침)
	// 활성 포트가 없을 때만 안내
	runningPorts, _ := portSvc.List("running", 1)
	if len(runningPorts) == 0 {
		fmt.Println("")
		fmt.Println("<!-- pal:port-guidance")
		fmt.Println("[PAL Kit 포트 사용 안내]")
		fmt.Println("")
		fmt.Println("⚠️ 현재 활성 포트가 없습니다.")
		fmt.Println("코드 변경 작업을 시작하기 전에 포트를 활성화해야 작업이 추적됩니다.")
		fmt.Println("")
		fmt.Println("포트 활성화 방법:")
		fmt.Println("1. 기존 포트 활성화: pal hook port-start <port-id>")
		fmt.Println("2. 새 포트 생성: pal port create <id> --title \"작업명\" && pal hook port-start <id>")
		fmt.Println("")
		fmt.Println("포트 목록 확인: pal port list")
		fmt.Println("-->")
	}

	return nil
}

func runHookSessionEnd(cmd *cobra.Command, args []string) error {
	input, err := readHookInput()
	if err != nil {
		input = &HookInput{}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	sessionSvc := session.NewService(database)
	lockSvc := lock.NewService(database)

	// Claude 세션 ID로 PAL 세션 찾기
	claudeSessionID := input.SessionID
	if claudeSessionID == "" {
		claudeSessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	// 프로젝트 루트 찾기
	cwd := input.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	projectRoot := context.FindProjectRoot(cwd)

	// 워크플로우 rules 파일 정리
	if projectRoot != "" {
		workflowSvc := workflow.NewService(projectRoot)
		workflowSvc.CleanupRulesFile()
	}

	// 빌더 에이전트 정리
	if projectRoot != "" {
		claudeSvc := context.NewClaudeService(database, projectRoot)
		claudeSvc.ProcessSessionEnd()
	}

	// FindActiveSession으로 다중 fallback 전략 사용
	palSession, err := sessionSvc.FindActiveSession(claudeSessionID, cwd, projectRoot)
	if err != nil {
		if verbose {
			fmt.Printf("⚠️  활성 세션을 찾을 수 없습니다: %v\n", err)
		}
		return nil // 세션이 없으면 조용히 종료
	}

	// transcript 파싱으로 usage 수집
	transcriptPath := input.TranscriptPath
	// fallback: 세션에 저장된 transcript 경로 사용
	if transcriptPath == "" && palSession.TranscriptPath.Valid {
		transcriptPath = palSession.TranscriptPath.String
	}

	if transcriptPath != "" {
		// 파일이 아직 쓰는 중일 수 있음 → 재시도 로직
		var usage *transcript.Usage
		for retry := 0; retry < 3; retry++ {
			usage, err = transcript.ParseFile(transcriptPath)
			if err == nil && usage != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if usage != nil {
			// 세션 usage 업데이트
			sessionSvc.UpdateUsage(
				palSession.ID,
				usage.InputTokens,
				usage.OutputTokens,
				usage.CacheReadTokens,
				usage.CacheCreateTokens,
				usage.CostUSD,
			)

			if verbose {
				fmt.Printf("📊 Usage collected:\n")
				fmt.Printf("   Input tokens: %d\n", usage.InputTokens)
				fmt.Printf("   Output tokens: %d\n", usage.OutputTokens)
				fmt.Printf("   Cache read: %d\n", usage.CacheReadTokens)
				fmt.Printf("   Cache create: %d\n", usage.CacheCreateTokens)
				fmt.Printf("   Cost: $%.4f\n", usage.CostUSD)
			}
		} else if verbose && err != nil {
			fmt.Printf("⚠️  Usage 수집 실패: %v\n", err)
		}
	}

	// Operator 세션 요약 생성 (세션 종료 전에)
	if projectRoot != "" {
		operatorSvc := operator.NewService(database, projectRoot)
		summary, err := operatorSvc.GenerateSummary(palSession.ID)
		if err == nil {
			// .pal/sessions/{date}-{id}.md 저장
			if err := operatorSvc.WriteSummary(summary); err != nil {
				if verbose {
					fmt.Fprintf(os.Stderr, "⚠️  세션 요약 저장 실패: %v\n", err)
				}
			} else if verbose {
				fmt.Printf("📝 Session summary saved: %s-%s.md\n",
					summary.GeneratedAt.Format("2006-01-02"), palSession.ID)
			}

			// 완료된 포트 출력
			if len(summary.PortsCompleted) > 0 {
				fmt.Printf("✅ Completed ports: %d\n", len(summary.PortsCompleted))
			}

			// ADR 후보 알림
			if len(summary.ADRCandidates) > 0 && verbose {
				fmt.Printf("📋 ADR candidates detected: %d\n", len(summary.ADRCandidates))
			}
		} else if verbose {
			fmt.Fprintf(os.Stderr, "⚠️  세션 요약 생성 실패: %v\n", err)
		}
	}

	// 종료 사유
	reason := input.Reason
	if reason == "" {
		reason = "exit"
	}

	// 개별 세션 종료 이벤트 로깅
	sessionSvc.LogEvent(palSession.ID, "session_end", fmt.Sprintf(`{"reason":"%s","claude_session_id":"%s"}`, reason, claudeSessionID))

	// 세션 상태를 complete로 확실히 변경
	if err := sessionSvc.EndWithReason(palSession.ID, reason); err != nil {
		if verbose {
			fmt.Printf("⚠️  세션 종료 실패: %v\n", err)
		}
		// fallback: Claude 세션 ID로 종료 시도
		if claudeSessionID != "" {
			sessionSvc.EndAllByClaudeSession(claudeSessionID, reason)
		}
	}

	// Lock 해제
	locks, _ := lockSvc.List()
	releasedCount := 0
	for _, l := range locks {
		lockSvc.Release(l.Resource)
		releasedCount++
	}

	if verbose {
		fmt.Printf("✓ Session ended: %s (reason: %s)\n", palSession.ID, reason)
		if releasedCount > 0 {
			fmt.Printf("  Released %d locks\n", releasedCount)
		}
	}

	return nil
}

func runHookPreToolUse(cmd *cobra.Command, args []string) error {
	input, err := readHookInput()
	if err != nil {
		return nil
	}

	// Edit/Write 도구인 경우 활성 포트 확인
	if input.ToolName == "Edit" || input.ToolName == "Write" {
		filePath, ok := input.ToolInput["file_path"].(string)
		if !ok {
			return nil
		}

		database, err := db.Open(GetDBPath())
		if err != nil {
			return nil
		}
		defer database.Close()

		portSvc := port.NewService(database)
		sessionSvc := session.NewService(database)

		// 프로젝트 루트 찾기
		cwd := input.Cwd
		if cwd == "" {
			cwd, _ = os.Getwd()
		}
		projectRoot := context.FindProjectRoot(cwd)

		// 활성 포트 확인
		runningPorts, _ := portSvc.List("running", 10)

		// 현재 세션 찾기
		claudeSessionID := input.SessionID
		if claudeSessionID == "" {
			claudeSessionID = os.Getenv("CLAUDE_SESSION_ID")
		}

		var palSessionID string
		palSession, err := sessionSvc.FindActiveSession(claudeSessionID, cwd, projectRoot)
		if err == nil && palSession != nil {
			palSessionID = palSession.ID
		}

		// 활성 포트가 없으면 경고
		if len(runningPorts) == 0 {
			// stderr로 Claude에 피드백 (Claude가 읽음)
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "⚠️  [PAL Kit] 활성 포트가 없습니다!")
			fmt.Fprintln(os.Stderr, "   코드 변경이 추적되지 않습니다.")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "   포트를 활성화하려면:")
			fmt.Fprintln(os.Stderr, "   1. pal port create <id> --title \"작업명\"")
			fmt.Fprintln(os.Stderr, "   2. pal hook port-start <id>")
			fmt.Fprintln(os.Stderr, "")

			// untracked_edit 이벤트 로깅
			if palSessionID != "" {
				eventData := fmt.Sprintf(`{"tool":"%s","file":"%s","warning":"no_active_port"}`,
					input.ToolName, filePath)
				sessionSvc.LogEvent(palSessionID, "untracked_edit", eventData)
			}
		} else {
			// 활성 포트가 있으면 포트 ID 표시
			if verbose {
				portIDs := make([]string, len(runningPorts))
				for i, p := range runningPorts {
					portIDs[i] = p.ID
				}
				fmt.Fprintf(os.Stderr, "📝 [PAL Kit] 활성 포트: %s\n", strings.Join(portIDs, ", "))
			}

			// 파일 수정 이벤트 로깅 (추적됨)
			if palSessionID != "" && len(runningPorts) > 0 {
				eventData := fmt.Sprintf(`{"tool":"%s","file":"%s","port":"%s"}`,
					input.ToolName, filePath, runningPorts[0].ID)
				sessionSvc.LogEvent(palSessionID, "file_edit", eventData)
			}
		}

		// Lock 확인 (기존 로직 유지)
		lockSvc := lock.NewService(database)
		_ = lockSvc
		// TODO: 파일 경로 기반 Lock 확인 로직
	}

	return nil
}

func runHookPostToolUse(cmd *cobra.Command, args []string) error {
	return nil
}

func runHookStop(cmd *cobra.Command, args []string) error {
	input, err := readHookInput()
	if err != nil {
		input = &HookInput{}
	}

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	if verbose && sessionID != "" {
		fmt.Printf("🛑 Stop: session=%s\n", sessionID)
	}

	return nil
}

func runHookPreCompact(cmd *cobra.Command, args []string) error {
	input, err := readHookInput()
	if err != nil {
		input = &HookInput{}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil
	}
	defer database.Close()

	sessionSvc := session.NewService(database)

	// Claude 세션 ID로 PAL 세션 찾기
	claudeSessionID := input.SessionID
	if claudeSessionID == "" {
		claudeSessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	if claudeSessionID != "" {
		palSession, err := sessionSvc.FindByClaudeSessionID(claudeSessionID)
		if err == nil && palSession != nil {
			sessionSvc.IncrementCompact(palSession.ID)
			
			// 컴팩트 이벤트 로깅
			trigger := input.Trigger
			if trigger == "" {
				trigger = "auto"
			}
			sessionSvc.LogEvent(palSession.ID, "compact", fmt.Sprintf(`{"trigger":"%s"}`, trigger))

			if verbose {
				fmt.Printf("📦 PreCompact: session=%s, trigger=%s\n", palSession.ID, trigger)
			}
		}
	}

	return nil
}

func runHookPortStart(cmd *cobra.Command, args []string) error {
	portID := args[0]

	// stdin에서 hook 입력 읽기
	input, err := readHookInput()
	if err != nil {
		input = &HookInput{}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	portSvc := port.NewService(database)
	sessionSvc := session.NewService(database)

	// 포트 정보 조회
	p, err := portSvc.Get(portID)
	if err != nil {
		return err
	}

	// 프로젝트 루트 찾기
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	// Rules 활성화
	rulesSvc := rules.NewService(projectRoot)
	title := portID
	if p.Title.Valid {
		title = p.Title.String
	}
	specPath := ""
	if p.FilePath.Valid {
		specPath = p.FilePath.String
	}

	if err := rulesSvc.ActivatePortWithSpec(portID, title, specPath, nil); err != nil {
		return err
	}

	// 문서 컨텍스트 로딩 (P3: docs-management)
	docSvc := document.NewService(database, projectRoot)
	if specPath != "" {
		// 포트 명세에서 관련 문서 검색
		relatedDocs, err := docSvc.GetRelatedDocs(specPath, 50000) // 50K 토큰 예산
		if err == nil && len(relatedDocs) > 0 {
			// .claude/rules/<port-id>.md 파일에 문서 참조 추가
			docContext := generateDocContext(relatedDocs, projectRoot)
			if docContext != "" {
				rulesSvc.AppendToRule(portID, docContext)
			}
			if verbose {
				fmt.Printf("📚 관련 문서 %d건 로드됨\n", len(relatedDocs))
			}
		}
	}

	// 현재 세션 찾기 (FindActiveSession 사용)
	claudeSessionID := input.SessionID
	if claudeSessionID == "" {
		claudeSessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	var palSessionID string
	palSession, err := sessionSvc.FindActiveSession(claudeSessionID, cwd, projectRoot)
	if err == nil && palSession != nil {
		palSessionID = palSession.ID
	}

	// Claude 통합 서비스로 컨텍스트 처리 (먼저 워커 정보 얻기)
	claudeSvc := context.NewClaudeService(database, projectRoot)
	result, err := claudeSvc.ProcessPortStart(portID)

	// 워커 ID 추출
	var agentID string
	if result != nil && result.WorkerID != "" {
		agentID = result.WorkerID
	}

	// RecordStart로 포트 시작 기록 (상태, 시간, 세션, 에이전트 한 번에)
	if err := portSvc.RecordStart(portID, palSessionID, agentID); err != nil {
		// fallback: 기존 방식으로 상태만 변경
		portSvc.UpdateStatus(portID, "running")
	}

	// 포트 시작 이벤트 로깅
	if palSessionID != "" {
		sessionSvc.LogEvent(palSessionID, "port_start", fmt.Sprintf(`{"port_id":"%s","title":"%s","agent_id":"%s"}`, portID, title, agentID))
		if verbose {
			fmt.Printf("🔗 Port linked to session: %s\n", palSessionID)
		}
	}
	if err != nil {
		// 실패해도 기본 동작은 수행
		if verbose {
			fmt.Fprintf(os.Stderr, "⚠️  워커 매핑 실패: %v\n", err)
		}
		// 기본 컨텍스트 주입
		ctxSvc := context.NewService(database)
		claudeMD := context.FindClaudeMD(cwd)
		if claudeMD != "" {
			ctxSvc.InjectToFile(claudeMD)
		}
	}

	if jsonOut {
		output := map[string]interface{}{
			"status": "started",
			"port":   portID,
		}
		if result != nil {
			output["worker_id"] = result.WorkerID
			output["worker_name"] = result.WorkerName
			output["token_count"] = result.TokenCount
			output["checklist"] = result.Checklist
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("▶️  포트 시작: %s\n", portID)
		fmt.Printf("   Rules: %s\n", rulesSvc.GetRulePath(portID))
		if result != nil {
			fmt.Printf("   워커: %s (%s)\n", result.WorkerName, result.WorkerID)
			fmt.Printf("   토큰: ~%d\n", result.TokenCount)
			if len(result.Checklist) > 0 {
				fmt.Printf("   체크리스트: %d 항목\n", len(result.Checklist))
			}
		}
	}

	return nil
}

func runHookPortEnd(cmd *cobra.Command, args []string) error {
	portID := args[0]

	// stdin에서 hook 입력 읽기
	input, err := readHookInput()
	if err != nil {
		input = &HookInput{}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	portSvc := port.NewService(database)
	lockSvc := lock.NewService(database)
	sessionSvc := session.NewService(database)

	// 포트 정보 조회 (시작 시간 확인용)
	p, err := portSvc.Get(portID)
	if err != nil {
		return err
	}

	// 프로젝트 루트 찾기
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)

	// Rules 비활성화
	if projectRoot != "" {
		rulesSvc := rules.NewService(projectRoot)
		rulesSvc.DeactivatePort(portID)
	}

	// 세션 찾기 (FindActiveSession 사용)
	claudeSessionID := input.SessionID
	if claudeSessionID == "" {
		claudeSessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	var palSessionID string
	palSession, err := sessionSvc.FindActiveSession(claudeSessionID, cwd, projectRoot)
	if err == nil && palSession != nil {
		palSessionID = palSession.ID
	}

	// 포트 duration 계산 (시작 시간부터 현재까지)
	var durationSecs int64
	if p.StartedAt.Valid {
		durationSecs = int64(time.Since(p.StartedAt.Time).Seconds())
	}

	// transcript에서 포트 작업 중 사용량 추정 (시작~종료 사이)
	// 현재는 세션 전체 usage에서 추정하기 어려우므로 0으로 설정
	// 향후 transcript 타임스탬프 기반 분할 수집 가능
	var inputTokens, outputTokens int64
	var costUSD float64

	// 세션에 usage가 있으면 이 포트에 할당 (단순화: 포트 작업 중 발생한 것으로 간주)
	// 더 정확한 추적은 port-start/port-end 사이의 transcript 분석 필요
	if palSession != nil {
		// 이미 세션에 수집된 usage 중 이 포트에 배분 (현재는 기록만)
		// 실제 구현에서는 transcript 시간 범위 기반 분석 필요
	}

	// RecordCompletion으로 포트 완료 기록 (상태, 시간, duration, usage)
	if err := portSvc.RecordCompletion(portID, inputTokens, outputTokens, costUSD); err != nil {
		// fallback: 기존 방식
		portSvc.UpdateStatus(portID, "complete")
		portSvc.SetDuration(portID, durationSecs)
	}

	// Lock 해제
	locks, _ := lockSvc.List()
	for _, l := range locks {
		lockSvc.Release(l.Resource)
	}

	// 포트 완료 이벤트 로깅
	if palSessionID != "" {
		sessionSvc.LogEvent(palSessionID, "port_end", fmt.Sprintf(
			`{"port_id":"%s","duration_secs":%d,"input_tokens":%d,"output_tokens":%d,"cost_usd":%.4f}`,
			portID, durationSecs, inputTokens, outputTokens, costUSD))
	}

	// Claude 통합 서비스로 컨텍스트 정리
	var result *context.PortEndResult
	if projectRoot != "" {
		claudeSvc := context.NewClaudeService(database, projectRoot)
		result, _ = claudeSvc.ProcessPortEnd(portID, "")
	}

	// 기본 컨텍스트 업데이트
	ctxSvc := context.NewService(database)
	claudeMD := context.FindClaudeMD(cwd)
	if claudeMD != "" {
		ctxSvc.InjectToFile(claudeMD)
	}

	if jsonOut {
		output := map[string]interface{}{
			"status":        "completed",
			"port":          portID,
			"duration_secs": durationSecs,
		}
		if result != nil {
			output["message"] = result.Message
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("✅ 포트 완료: %s\n", portID)
		if durationSecs > 0 {
			fmt.Printf("   소요 시간: %s\n", formatPortDuration(durationSecs))
		}
	}

	return nil
}

// formatPortDuration formats seconds into human readable string
func formatPortDuration(secs int64) string {
	if secs < 60 {
		return fmt.Sprintf("%d초", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%d분 %d초", secs/60, secs%60)
	}
	hours := secs / 3600
	mins := (secs % 3600) / 60
	return fmt.Sprintf("%d시간 %d분", hours, mins)
}

func runHookSync(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	portSvc := port.NewService(database)

	// 프로젝트 루트 찾기
	cwd, _ := os.Getwd()
	projectRoot := context.FindProjectRoot(cwd)
	if projectRoot == "" {
		return fmt.Errorf("PAL 프로젝트를 찾을 수 없습니다")
	}

	rulesSvc := rules.NewService(projectRoot)

	// 워크플로우 rules 갱신
	workflowSvc := workflow.NewService(projectRoot)
	ctx, err := workflowSvc.GetContext()
	if err == nil {
		if err := workflowSvc.WriteRulesFile(ctx); err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "⚠️  워크플로우 rules 갱신 실패: %v\n", err)
			}
		} else if verbose {
			fmt.Printf("📝 Workflow rules: %s\n", workflowSvc.GetRulesPath())
		}
	}

	// running 포트 조회
	runningPorts, err := portSvc.List("running", 100)
	if err != nil {
		return err
	}

	// 현재 활성 rules 조회
	activeRules, _ := rulesSvc.ListActiveRules()
	activeRulesMap := make(map[string]bool)
	for _, r := range activeRules {
		activeRulesMap[r] = true
	}

	// running 포트 ID 맵
	runningPortsMap := make(map[string]bool)
	for _, p := range runningPorts {
		runningPortsMap[p.ID] = true
	}

	activated := 0
	deactivated := 0

	// running 포트에 rules가 없으면 생성
	for _, p := range runningPorts {
		if !activeRulesMap[p.ID] {
			title := p.ID
			if p.Title.Valid {
				title = p.Title.String
			}
			specPath := ""
			if p.FilePath.Valid {
				specPath = p.FilePath.String
			}
			rulesSvc.ActivatePortWithSpec(p.ID, title, specPath, nil)
			activated++
		}
	}

	// running이 아닌데 rules가 있으면 삭제
	for _, ruleID := range activeRules {
		if !runningPortsMap[ruleID] {
			rulesSvc.DeactivatePort(ruleID)
			deactivated++
		}
	}

	// 문서 인덱싱 실행
	docSvc := document.NewService(database, projectRoot)
	indexResult, err := docSvc.Index()
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "⚠️  문서 인덱싱 실패: %v\n", err)
		}
	} else if verbose && (indexResult.Added > 0 || indexResult.Updated > 0 || indexResult.Removed > 0) {
		fmt.Printf("📚 문서 인덱싱: +%d /%d -%d\n", indexResult.Added, indexResult.Updated, indexResult.Removed)
	}

	// 컨텍스트 업데이트
	ctxSvc := context.NewService(database)
	claudeMD := context.FindClaudeMD(cwd)
	if claudeMD != "" {
		ctxSvc.InjectToFile(claudeMD)
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"activated":   activated,
			"deactivated": deactivated,
			"running":     len(runningPorts),
		})
	} else {
		fmt.Printf("🔄 Sync 완료\n")
		fmt.Printf("   Running ports: %d\n", len(runningPorts))
		if activated > 0 {
			fmt.Printf("   Activated: %d\n", activated)
		}
		if deactivated > 0 {
			fmt.Printf("   Deactivated: %d\n", deactivated)
		}
	}

	return nil
}

func runHookEvent(cmd *cobra.Command, args []string) error {
	eventType := args[0]
	message := args[1]

	// 허용된 이벤트 타입 확인
	allowedTypes := map[string]string{
		"decision":     session.EventDecision,
		"escalation":   session.EventEscalation,
		"user_request": session.EventUserRequest,
	}

	mappedType, ok := allowedTypes[eventType]
	if !ok {
		return fmt.Errorf("지원하지 않는 이벤트 타입: %s (허용: decision, escalation, user_request)", eventType)
	}

	// stdin에서 hook 입력 읽기
	input, err := readHookInput()
	if err != nil {
		input = &HookInput{}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	sessionSvc := session.NewService(database)
	portSvc := port.NewService(database)

	// 프로젝트 루트 찾기
	cwd := input.Cwd
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	projectRoot := context.FindProjectRoot(cwd)

	// 현재 세션 찾기
	claudeSessionID := input.SessionID
	if claudeSessionID == "" {
		claudeSessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	var palSessionID string
	palSession, err := sessionSvc.FindActiveSession(claudeSessionID, cwd, projectRoot)
	if err == nil && palSession != nil {
		palSessionID = palSession.ID
	}

	if palSessionID == "" {
		return fmt.Errorf("활성 세션을 찾을 수 없습니다")
	}

	// 활성 포트 ID 가져오기
	var activePortID string
	runningPorts, _ := portSvc.List("running", 1)
	if len(runningPorts) > 0 {
		activePortID = runningPorts[0].ID
	}

	// 이벤트 데이터 구성
	eventData := fmt.Sprintf(`{"message":"%s","port_id":"%s"`,
		escapeJSON(truncateString(message, 500)), activePortID)

	if hookEventContext != "" {
		eventData += fmt.Sprintf(`,"context":%s`, hookEventContext)
	}
	eventData += "}"

	// 이벤트 로깅
	if err := sessionSvc.LogEvent(palSessionID, mappedType, eventData); err != nil {
		return fmt.Errorf("이벤트 로깅 실패: %w", err)
	}

	// 출력
	if jsonOut {
		output := map[string]interface{}{
			"status":     "logged",
			"event_type": mappedType,
			"session_id": palSessionID,
		}
		if activePortID != "" {
			output["port_id"] = activePortID
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		icon := "📝"
		switch mappedType {
		case session.EventDecision:
			icon = "🎯"
		case session.EventEscalation:
			icon = "🚨"
		case session.EventUserRequest:
			icon = "💬"
		}
		fmt.Printf("%s 이벤트 기록: %s\n", icon, mappedType)
		fmt.Printf("   세션: %s\n", palSessionID)
		if activePortID != "" {
			fmt.Printf("   포트: %s\n", activePortID)
		}
		fmt.Printf("   메시지: %s\n", truncateString(message, 100))
	}

	return nil
}

func runHookEvents(cmd *cobra.Command, args []string) error {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	sessionSvc := session.NewService(database)

	var targetSessionID string

	if len(args) > 0 {
		// 특정 세션 ID 지정
		targetSessionID = args[0]
	} else {
		// 현재 활성 세션 찾기
		cwd, _ := os.Getwd()
		projectRoot := context.FindProjectRoot(cwd)
		claudeSessionID := os.Getenv("CLAUDE_SESSION_ID")

		palSession, err := sessionSvc.FindActiveSession(claudeSessionID, cwd, projectRoot)
		if err == nil && palSession != nil {
			targetSessionID = palSession.ID
		}
	}

	if targetSessionID == "" {
		return fmt.Errorf("세션을 찾을 수 없습니다")
	}

	// 이벤트 조회
	events, err := sessionSvc.GetEvents(targetSessionID, hookEventsTypeFilter, hookEventsLimit)
	if err != nil {
		return fmt.Errorf("이벤트 조회 실패: %w", err)
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"session_id": targetSessionID,
			"count":      len(events),
			"events":     events,
		})
		return nil
	}

	fmt.Printf("📋 세션 이벤트: %s\n", targetSessionID)
	fmt.Printf("   총 %d건\n\n", len(events))

	if len(events) == 0 {
		fmt.Println("   이벤트가 없습니다.")
		return nil
	}

	// 이벤트 타입별 아이콘
	typeIcons := map[string]string{
		session.EventSessionStart:  "🚀",
		session.EventSessionEnd:    "🏁",
		session.EventPortStart:     "▶️",
		session.EventPortEnd:       "✅",
		session.EventUserRequest:   "💬",
		session.EventDecision:      "🎯",
		session.EventEscalation:    "🚨",
		session.EventFileEdit:      "📝",
		session.EventUntrackedEdit: "⚠️",
		session.EventCompact:       "📦",
	}

	for _, e := range events {
		icon := typeIcons[e.EventType]
		if icon == "" {
			icon = "📌"
		}

		timeStr := e.CreatedAt.Format("15:04:05")
		fmt.Printf("%s %s [%s]\n", icon, e.EventType, timeStr)

		// event_data에서 주요 정보 추출
		if e.EventData != "" {
			var data map[string]interface{}
			if json.Unmarshal([]byte(e.EventData), &data) == nil {
				if msg, ok := data["message"].(string); ok {
					fmt.Printf("   %s\n", truncateString(msg, 80))
				}
				if portID, ok := data["port_id"].(string); ok && portID != "" {
					fmt.Printf("   포트: %s\n", portID)
				}
				if file, ok := data["file"].(string); ok && file != "" {
					fmt.Printf("   파일: %s\n", filepath.Base(file))
				}
			}
		}
		fmt.Println()
	}

	return nil
}

// generateDocContext creates document context markdown from related documents
func generateDocContext(docs []document.Document, projectRoot string) string {
	if len(docs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n---\n\n")
	sb.WriteString("## 관련 문서\n\n")
	sb.WriteString("> 이 포트와 관련된 문서입니다. 작업 시 참고하세요.\n\n")

	// 타입별로 그룹화
	byType := make(map[string][]document.Document)
	for _, d := range docs {
		byType[d.Type] = append(byType[d.Type], d)
	}

	typeOrder := []string{"l1", "l2", "lm", "convention", "template"}
	typeNames := map[string]string{
		"l1":         "L1 Domain",
		"l2":         "L2 Feature",
		"lm":         "LM Coordinator",
		"convention": "컨벤션",
		"template":   "템플릿",
	}

	for _, t := range typeOrder {
		typeDocs, ok := byType[t]
		if !ok || len(typeDocs) == 0 {
			continue
		}

		name := typeNames[t]
		if name == "" {
			name = t
		}

		sb.WriteString(fmt.Sprintf("### %s\n\n", name))
		for _, d := range typeDocs {
			sb.WriteString(fmt.Sprintf("- **%s** (`%s`)\n", d.ID, d.Path))
			if d.Domain != "" {
				sb.WriteString(fmt.Sprintf("  - 도메인: %s\n", d.Domain))
			}
			sb.WriteString(fmt.Sprintf("  - 토큰: ~%d\n", d.Tokens))
		}
		sb.WriteString("\n")
	}

	// 기타 타입
	for t, typeDocs := range byType {
		found := false
		for _, order := range typeOrder {
			if t == order {
				found = true
				break
			}
		}
		if found || len(typeDocs) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s\n\n", t))
		for _, d := range typeDocs {
			sb.WriteString(fmt.Sprintf("- **%s** (`%s`)\n", d.ID, d.Path))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
