package path

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/env"
)

func setupTestResolver(t *testing.T) (*Resolver, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "pal-path-test-*")
	if err != nil {
		t.Fatalf("임시 디렉토리 생성 실패: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("DB 생성 실패: %v", err)
	}

	// For simplicity, use direct resolver with preset paths

	cleanup := func() {
		database.Close()
		os.RemoveAll(tmpDir)
	}

	// Create resolver with preset paths
	resolver := &Resolver{
		paths: &env.PathVariables{
			Workspace:  "/Users/test/playground",
			ClaudeData: "/Users/test/.claude",
			Home:       "/Users/test",
		},
	}

	return resolver, cleanup
}

func TestToLogical(t *testing.T) {
	resolver, cleanup := setupTestResolver(t)
	defer cleanup()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "workspace path",
			input:    "/Users/test/playground/CodeSpace/project",
			expected: "$workspace/CodeSpace/project",
		},
		{
			name:     "workspace exact",
			input:    "/Users/test/playground",
			expected: "$workspace",
		},
		{
			name:     "claude_data path",
			input:    "/Users/test/.claude/projects/xxx",
			expected: "$claude_data/projects/xxx",
		},
		{
			name:     "home path",
			input:    "/Users/test/Documents",
			expected: "$home/Documents",
		},
		{
			name:     "unmatched path",
			input:    "/var/log/system.log",
			expected: "/var/log/system.log",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "already logical",
			input:    "$workspace/project",
			expected: "$workspace/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ToLogical(tt.input)
			if err != nil {
				t.Fatalf("ToLogical 실패: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ToLogical(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToAbsolute(t *testing.T) {
	resolver, cleanup := setupTestResolver(t)
	defer cleanup()

	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "workspace path",
			input:    "$workspace/CodeSpace/project",
			expected: "/Users/test/playground/CodeSpace/project",
		},
		{
			name:     "workspace exact",
			input:    "$workspace",
			expected: "/Users/test/playground",
		},
		{
			name:     "claude_data path",
			input:    "$claude_data/projects/xxx",
			expected: "/Users/test/.claude/projects/xxx",
		},
		{
			name:     "home path",
			input:    "$home/Documents",
			expected: "/Users/test/Documents",
		},
		{
			name:     "already absolute",
			input:    "/var/log/system.log",
			expected: "/var/log/system.log",
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "unknown variable",
			input:    "$unknown/path",
			expected: "$unknown/path",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ToAbsolute(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("ToAbsolute(%q) expected error but got none", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ToAbsolute 실패: %v", err)
			}
			if result != tt.expected {
				t.Errorf("ToAbsolute(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	resolver, cleanup := setupTestResolver(t)
	defer cleanup()

	paths := []string{
		"/Users/test/playground/project",
		"/Users/test/.claude/sessions",
		"/Users/test/Documents/notes",
	}

	for _, original := range paths {
		logical, err := resolver.ToLogical(original)
		if err != nil {
			t.Fatalf("ToLogical 실패: %v", err)
		}

		absolute, err := resolver.ToAbsolute(logical)
		if err != nil {
			t.Fatalf("ToAbsolute 실패: %v", err)
		}

		if absolute != original {
			t.Errorf("왕복 변환 실패: %q -> %q -> %q", original, logical, absolute)
		}
	}
}

func TestIsLogical(t *testing.T) {
	resolver, cleanup := setupTestResolver(t)
	defer cleanup()

	tests := []struct {
		input    string
		expected bool
	}{
		{"$workspace/project", true},
		{"$home/Documents", true},
		{"/Users/test/playground", false},
		{"/var/log", false},
		{"", false},
	}

	for _, tt := range tests {
		result := resolver.IsLogical(tt.input)
		if result != tt.expected {
			t.Errorf("IsLogical(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestIsResolvable(t *testing.T) {
	resolver, cleanup := setupTestResolver(t)
	defer cleanup()

	tests := []struct {
		input    string
		expected bool
	}{
		{"$workspace/project", true},
		{"$home/Documents", true},
		{"$unknown/path", false},
		{"/var/log", true}, // Absolute paths are always resolvable
	}

	for _, tt := range tests {
		result := resolver.IsResolvable(tt.input)
		if result != tt.expected {
			t.Errorf("IsResolvable(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestAnalyze(t *testing.T) {
	resolver, cleanup := setupTestResolver(t)
	defer cleanup()

	// Test with absolute path
	info, err := resolver.Analyze("/Users/test/playground/project")
	if err != nil {
		t.Fatalf("Analyze 실패: %v", err)
	}

	if info.Original != "/Users/test/playground/project" {
		t.Errorf("Original 불일치: %q", info.Original)
	}
	if info.Logical != "$workspace/project" {
		t.Errorf("Logical 불일치: %q", info.Logical)
	}
	if info.Variable != "$workspace" {
		t.Errorf("Variable 불일치: %q", info.Variable)
	}
	if !info.Resolvable {
		t.Error("Resolvable should be true")
	}

	// Test with logical path
	info, err = resolver.Analyze("$workspace/project")
	if err != nil {
		t.Fatalf("Analyze 실패: %v", err)
	}

	if info.Absolute != "/Users/test/playground/project" {
		t.Errorf("Absolute 불일치: %q", info.Absolute)
	}
}

func TestGetVariable(t *testing.T) {
	resolver, cleanup := setupTestResolver(t)
	defer cleanup()

	tests := []struct {
		input    string
		expected string
	}{
		{"$workspace/project", "$workspace"},
		{"$home/Documents", "$home"},
		{"$claude_data", "$claude_data"},
		{"/absolute/path", ""},
	}

	for _, tt := range tests {
		result := resolver.GetVariable(tt.input)
		if result != tt.expected {
			t.Errorf("GetVariable(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestBatchConvert(t *testing.T) {
	resolver, cleanup := setupTestResolver(t)
	defer cleanup()

	paths := []string{
		"/Users/test/playground/project1",
		"/Users/test/playground/project2",
		"/Users/test/.claude/data",
	}

	results := resolver.BatchToLogical(paths)

	expected := []string{
		"$workspace/project1",
		"$workspace/project2",
		"$claude_data/data",
	}

	for i, r := range results {
		if r.Converted != expected[i] {
			t.Errorf("BatchToLogical[%d] = %q, want %q", i, r.Converted, expected[i])
		}
		if r.Error != "" {
			t.Errorf("BatchToLogical[%d] unexpected error: %s", i, r.Error)
		}
	}
}

func TestPathSpecificity(t *testing.T) {
	// Test that more specific paths are matched first
	resolver := &Resolver{
		paths: &env.PathVariables{
			Workspace:  "/Users/test/playground",
			ClaudeData: "/Users/test/playground/claude-data", // More specific
			Home:       "/Users/test",
		},
	}

	// Should match $claude_data (more specific) not $workspace
	result, err := resolver.ToLogical("/Users/test/playground/claude-data/sessions")
	if err != nil {
		t.Fatalf("ToLogical 실패: %v", err)
	}

	if result != "$claude_data/sessions" {
		t.Errorf("더 구체적인 경로가 매칭되어야 함: got %q, want $claude_data/sessions", result)
	}
}
