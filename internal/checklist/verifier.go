package checklist

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Verifier handles checklist verification
type Verifier struct {
	ProjectRoot string
}

// NewVerifier creates a new verifier
func NewVerifier(projectRoot string) *Verifier {
	return &Verifier{
		ProjectRoot: projectRoot,
	}
}

// Verify runs verification for all items in the checklist
func (v *Verifier) Verify(checklist Checklist) *VerificationResult {
	startTime := time.Now()

	result := &VerificationResult{
		Passed:    true,
		Results:   []CheckResult{},
		BlockedBy: []string{},
	}

	for _, item := range checklist.Items {
		checkResult := v.verifyItem(item)
		result.Results = append(result.Results, checkResult)

		if checkResult.Passed {
			result.PassedCount++
		} else {
			result.FailedCount++
			if item.Required {
				result.Passed = false
				result.BlockedBy = append(result.BlockedBy, item.ID)
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result
}

// verifyItem verifies a single checklist item
func (v *Verifier) verifyItem(item ChecklistItem) CheckResult {
	startTime := time.Now()

	result := CheckResult{
		ItemID:      item.ID,
		Description: item.Description,
		Required:    item.Required,
	}

	switch item.Type {
	case "auto":
		passed, message, output := v.runCommand(item.Command)
		result.Passed = passed
		result.Message = message
		result.Output = truncate(output, 500)
	case "manual":
		// Manual items are assumed passed (user must verify)
		result.Passed = true
		result.Message = "수동 확인 필요"
	default:
		result.Passed = true
		result.Message = "알 수 없는 타입"
	}

	result.Duration = time.Since(startTime)
	return result
}

// runCommand runs a shell command and returns the result
func (v *Verifier) runCommand(command string) (bool, string, string) {
	// Detect project type and adjust command
	command = v.adjustCommand(command)

	// Split command for exec
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false, "빈 명령어", ""
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = v.ProjectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if err != nil {
		return false, fmt.Sprintf("실패: %s", err.Error()), output
	}

	return true, "성공", output
}

// adjustCommand adjusts command based on project type
func (v *Verifier) adjustCommand(command string) string {
	// Check for package.json (Node.js project)
	if v.fileExists("package.json") {
		command = strings.ReplaceAll(command, "go build ./...", "npm run build")
		command = strings.ReplaceAll(command, "go test ./...", "npm test")
		command = strings.ReplaceAll(command, "golangci-lint run", "npm run lint")
	}

	// Check for Cargo.toml (Rust project)
	if v.fileExists("Cargo.toml") {
		command = strings.ReplaceAll(command, "go build ./...", "cargo build")
		command = strings.ReplaceAll(command, "go test ./...", "cargo test")
		command = strings.ReplaceAll(command, "golangci-lint run", "cargo clippy")
	}

	return command
}

// fileExists checks if a file exists in the project root
func (v *Verifier) fileExists(name string) bool {
	cmd := exec.Command("test", "-f", name)
	cmd.Dir = v.ProjectRoot
	return cmd.Run() == nil
}

// VerifyBuild runs build verification
func (v *Verifier) VerifyBuild() CheckResult {
	return v.verifyItem(ChecklistItem{
		ID:          "build",
		Description: "빌드 성공",
		Type:        "auto",
		Command:     "go build ./...",
		Required:    true,
	})
}

// VerifyTest runs test verification
func (v *Verifier) VerifyTest() CheckResult {
	return v.verifyItem(ChecklistItem{
		ID:          "test",
		Description: "테스트 통과",
		Type:        "auto",
		Command:     "go test ./...",
		Required:    true,
	})
}

// VerifyLint runs lint verification
func (v *Verifier) VerifyLint() CheckResult {
	return v.verifyItem(ChecklistItem{
		ID:          "lint",
		Description: "린트 경고 없음",
		Type:        "auto",
		Command:     "golangci-lint run",
		Required:    false,
	})
}

// QuickVerify runs only essential verifications (build + test)
func (v *Verifier) QuickVerify() *VerificationResult {
	startTime := time.Now()

	buildResult := v.VerifyBuild()
	testResult := v.VerifyTest()

	result := &VerificationResult{
		Passed: buildResult.Passed && testResult.Passed,
		Results: []CheckResult{
			buildResult,
			testResult,
		},
		BlockedBy: []string{},
	}

	if buildResult.Passed {
		result.PassedCount++
	} else {
		result.FailedCount++
		result.BlockedBy = append(result.BlockedBy, "build")
	}

	if testResult.Passed {
		result.PassedCount++
	} else {
		result.FailedCount++
		result.BlockedBy = append(result.BlockedBy, "test")
	}

	result.Duration = time.Since(startTime)
	return result
}

// truncate truncates a string to maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
