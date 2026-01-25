package session

import (
	"os"
	"strconv"
	"strings"
)

// GetTTY returns the current terminal identifier
// Tries multiple methods: SSH_TTY, GPG_TTY, or /dev/tty
func GetTTY() string {
	// 1. SSH_TTY 환경변수 (SSH 세션인 경우)
	if tty := os.Getenv("SSH_TTY"); tty != "" {
		return tty
	}

	// 2. GPG_TTY 환경변수 (GPG 설정된 경우)
	if tty := os.Getenv("GPG_TTY"); tty != "" {
		return tty
	}

	// 3. TTY 환경변수 (일부 시스템)
	if tty := os.Getenv("TTY"); tty != "" {
		return tty
	}

	// 4. /dev/tty 파일 확인 (Unix 계열)
	if _, err := os.Stat("/dev/tty"); err == nil {
		// tty 명령어로 실제 경로 가져오기 시도는 생략
		// (exec.Command 의존성 추가 없이)
		return "/dev/tty"
	}

	// 5. stdin이 터미널인지 확인
	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) != 0 {
		return "stdin-tty"
	}

	return ""
}

// GetParentPID returns the parent process ID
func GetParentPID() int {
	return os.Getppid()
}

// GetProcessInfo returns TTY and ParentPID for the current process
func GetProcessInfo() (tty string, parentPID int) {
	return GetTTY(), GetParentPID()
}

// GetTerminalIdentifier returns a string that can identify the terminal session
// Combines multiple signals for better uniqueness
func GetTerminalIdentifier() string {
	parts := []string{}

	// TTY
	if tty := GetTTY(); tty != "" {
		parts = append(parts, tty)
	}

	// Parent PID
	ppid := GetParentPID()
	if ppid > 0 {
		parts = append(parts, strconv.Itoa(ppid))
	}

	// TERM_SESSION_ID (macOS Terminal.app)
	if sid := os.Getenv("TERM_SESSION_ID"); sid != "" {
		parts = append(parts, sid)
	}

	// WINDOWID (X11)
	if wid := os.Getenv("WINDOWID"); wid != "" {
		parts = append(parts, wid)
	}

	// ITERM_SESSION_ID (iTerm2)
	if isid := os.Getenv("ITERM_SESSION_ID"); isid != "" {
		parts = append(parts, isid)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, ":")
}
