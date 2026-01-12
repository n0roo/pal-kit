package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/context"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/lock"
	"github.com/n0roo/pal-kit/internal/manifest"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/rules"
	"github.com/n0roo/pal-kit/internal/session"
	"github.com/n0roo/pal-kit/internal/transcript"
	"github.com/n0roo/pal-kit/internal/workflow"
	"github.com/spf13/cobra"
)

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

	hookSessionStartCmd.Flags().StringVar(&hookPortID, "port", "", "시작할 포트 ID")
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

	// Claude 세션 ID로 기존 세션 확인
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

		// 세션 시작 (프로젝트 정보 포함)
		opts := session.StartOptions{
			ID:              palSessionID,
			PortID:          hookPortID,
			SessionType:     session.TypeSingle,
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

	if claudeSessionID != "" {
		// 종료 전에 PAL 세션 찾기 (usage 업데이트용)
		palSession, _ := sessionSvc.FindByClaudeSessionID(claudeSessionID)

		// transcript 파싱으로 usage 수집
		transcriptPath := input.TranscriptPath
		if transcriptPath != "" && palSession != nil {
			usage, err := transcript.ParseFile(transcriptPath)
			if err == nil && usage != nil {
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

		// 종료 사유
		reason := input.Reason
		if reason == "" {
			reason = "exit"
		}

		// Claude 세션 ID에 해당하는 모든 세션 종료
		closedCount, err := sessionSvc.EndAllByClaudeSession(claudeSessionID, reason)
		if err == nil && closedCount > 0 {
			// 해당 세션들의 Lock 해제
			locks, _ := lockSvc.List()
			releasedCount := 0
			for _, l := range locks {
				// Claude 세션에 속한 Lock들 해제
				lockSvc.Release(l.Resource)
				releasedCount++
			}

			if verbose {
				fmt.Printf("✓ Sessions ended: %d (reason: %s)\n", closedCount, reason)
				if releasedCount > 0 {
					fmt.Printf("  Released %d locks\n", releasedCount)
				}
			}
		} else if verbose {
			fmt.Printf("⚠️  No PAL session found for Claude session: %s\n", claudeSessionID)
		}
	}

	return nil
}

func runHookPreToolUse(cmd *cobra.Command, args []string) error {
	input, err := readHookInput()
	if err != nil {
		return nil
	}

	// Edit/Write 도구인 경우 Lock 확인
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

		lockSvc := lock.NewService(database)
		_ = lockSvc
		_ = filePath
		
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

	// 포트 상태 변경
	if err := portSvc.UpdateStatus(portID, "running"); err != nil {
		return err
	}

	// 현재 세션에 포트 연결
	claudeSessionID := input.SessionID
	if claudeSessionID == "" {
		claudeSessionID = os.Getenv("CLAUDE_SESSION_ID")
	}
	if claudeSessionID != "" {
		palSession, err := sessionSvc.FindByClaudeSessionID(claudeSessionID)
		if err == nil && palSession != nil {
			// 포트에 세션 ID 할당
			portSvc.AssignSession(portID, palSession.ID)

			// 포트 시작 이벤트 로깅
			sessionSvc.LogEvent(palSession.ID, "port_start", fmt.Sprintf(`{"port_id":"%s","title":"%s"}`, portID, title))

			if verbose {
				fmt.Printf("🔗 Port linked to session: %s\n", palSession.ID)
			}
		}
	}

	// Claude 통합 서비스로 컨텍스트 처리
	claudeSvc := context.NewClaudeService(database, projectRoot)
	result, err := claudeSvc.ProcessPortStart(portID)
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

	// 포트 상태 변경
	if err := portSvc.UpdateStatus(portID, "complete"); err != nil {
		return err
	}

	// 포트 duration 계산 (시작 시간부터 현재까지)
	var durationSecs int64
	if p.StartedAt.Valid {
		durationSecs = int64(time.Since(p.StartedAt.Time).Seconds())
	}

	// 세션에서 이 포트 관련 Lock 해제
	claudeSessionID := input.SessionID
	if claudeSessionID == "" {
		claudeSessionID = os.Getenv("CLAUDE_SESSION_ID")
	}
	if claudeSessionID != "" {
		locks, _ := lockSvc.List()
		for _, l := range locks {
			// 포트 관련 Lock이면 해제 (간단히 전체 해제)
			lockSvc.Release(l.Resource)
		}

		// 포트 완료 이벤트 로깅
		palSession, err := sessionSvc.FindByClaudeSessionID(claudeSessionID)
		if err == nil && palSession != nil {
			sessionSvc.LogEvent(palSession.ID, "port_end", fmt.Sprintf(`{"port_id":"%s","duration_secs":%d}`, portID, durationSecs))
		}
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
			"status": "completed",
			"port":   portID,
		}
		if result != nil {
			output["message"] = result.Message
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("✅ 포트 완료: %s\n", portID)
	}

	return nil
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
