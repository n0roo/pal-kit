package transcript

import (
	"testing"
)

func TestParseFile(t *testing.T) {
	// Skip if file doesn't exist
	testFile := "/Users/n0roo/.claude/projects/-Users-n0roo-playground-CodeSpace-pal-kit/fe1cf5ea-1223-45d1-9dd8-49e01e413805.jsonl"

	usage, err := ParseFile(testFile)
	if err != nil {
		t.Skipf("Test file not found: %v", err)
	}

	if usage.InputTokens == 0 {
		t.Error("Expected non-zero input tokens")
	}
	if usage.OutputTokens == 0 {
		t.Error("Expected non-zero output tokens")
	}
	if usage.MessageCount == 0 {
		t.Error("Expected non-zero message count")
	}

	t.Logf("Input tokens: %d", usage.InputTokens)
	t.Logf("Output tokens: %d", usage.OutputTokens)
	t.Logf("Cache read: %d", usage.CacheReadTokens)
	t.Logf("Cache create: %d", usage.CacheCreateTokens)
	t.Logf("Cost: $%.4f", usage.CostUSD)
	t.Logf("Messages: %d", usage.MessageCount)
}
