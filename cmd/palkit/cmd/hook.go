package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/lock"
	"github.com/n0roo/pal-kit/internal/session"
	"github.com/spf13/cobra"
)

// HookInput represents the JSON input from Claude Code hooks
type HookInput struct {
	SessionID     string                 `json:"session_id"`
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	HookEventName string                 `json:"hook_event_name"`
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook 지원",
	Long:  `Claude Code Hook에서 호출되는 커맨드입니다.`,
}

var hookSessionStartCmd = &cobra.Command{
	Use:   "session-start",
	Short: "SessionStart Hook",
	RunE:  runHookSessionStart,
}

var hookSessionEndCmd = &cobra.Command{
	Use:   "session-end",
	Short: "SessionEnd Hook",
	RunE:  runHookSessionEnd,
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

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookSessionStartCmd)
	hookCmd.AddCommand(hookSessionEndCmd)
	hookCmd.AddCommand(hookPreToolUseCmd)
	hookCmd.AddCommand(hookPostToolUseCmd)
	hookCmd.AddCommand(hookStopCmd)
	hookCmd.AddCommand(hookPreCompactCmd)
}

func readHookInput() (*HookInput, error) {
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
		// stdin이 없어도 계속 진행
		input = &HookInput{}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return err
	}
	defer database.Close()

	svc := session.NewService(database)

	// 세션 ID 결정
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}
	if sessionID == "" {
		sessionID = uuid.New().String()[:8]
	}

	// 세션 시작
	if err := svc.Start(sessionID, "", ""); err != nil {
		// 이미 존재하면 무시
		if verbose {
			fmt.Fprintf(os.Stderr, "세션 시작: %v\n", err)
		}
	}

	// 프로젝트 정보 출력
	projectDir := os.Getenv("CLAUDE_PROJECT_DIR")
	if projectDir != "" && verbose {
		fmt.Printf("📁 Project: %s\n", projectDir)
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

	svc := session.NewService(database)
	lockSvc := lock.NewService(database)

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	if sessionID != "" {
		// 세션 종료
		svc.End(sessionID)

		// 해당 세션의 Lock 해제
		locks, _ := lockSvc.List()
		for _, l := range locks {
			if l.SessionID == sessionID {
				lockSvc.Release(l.Resource)
			}
		}
	}

	return nil
}

func runHookPreToolUse(cmd *cobra.Command, args []string) error {
	input, err := readHookInput()
	if err != nil {
		return nil // stdin 없으면 통과
	}

	// Edit/Write 도구인 경우 Lock 확인
	if input.ToolName == "Edit" || input.ToolName == "Write" {
		filePath, ok := input.ToolInput["file_path"].(string)
		if !ok {
			return nil
		}

		database, err := db.Open(GetDBPath())
		if err != nil {
			return nil // DB 오류면 통과
		}
		defer database.Close()

		lockSvc := lock.NewService(database)

		// 파일 경로에서 리소스 추출 (예: entity, service 등)
		// 현재는 단순히 통과
		_ = lockSvc
		_ = filePath
	}

	return nil
}

func runHookPostToolUse(cmd *cobra.Command, args []string) error {
	// 현재는 단순히 통과
	return nil
}

func runHookStop(cmd *cobra.Command, args []string) error {
	input, err := readHookInput()
	if err != nil {
		input = &HookInput{}
	}

	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil
	}
	defer database.Close()

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	// 현재는 단순히 로그
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

	svc := session.NewService(database)

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}

	if sessionID != "" {
		// 컴팩션 카운트 증가
		svc.IncrementCompact(sessionID)

		if verbose {
			fmt.Printf("📦 PreCompact: session=%s\n", sessionID)
		}
	}

	return nil
}
