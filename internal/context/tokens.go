package context

import (
	"os"
	"unicode"
	"unicode/utf8"
)

// TokenCounter defines the interface for token counting
type TokenCounter interface {
	Count(text string) int
	CountFile(path string) (int, error)
}

// ApproximateCounter provides fast approximate token counting
// Uses heuristics based on character type:
// - English: ~4 chars/token
// - Korean: ~2 chars/token
// - Code: ~3 chars/token
type ApproximateCounter struct{}

// NewApproximateCounter creates a new approximate counter
func NewApproximateCounter() *ApproximateCounter {
	return &ApproximateCounter{}
}

// Count estimates token count for the given text
func (c *ApproximateCounter) Count(text string) int {
	if len(text) == 0 {
		return 0
	}

	var tokens float64
	var englishChars, koreanChars, codeChars, otherChars int

	for _, r := range text {
		switch {
		case unicode.Is(unicode.Hangul, r):
			koreanChars++
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			englishChars++
		case isCodeChar(r):
			codeChars++
		default:
			otherChars++
		}
	}

	// Apply different ratios for different character types
	tokens += float64(englishChars) / 4.0  // ~4 chars per token for English
	tokens += float64(koreanChars) / 2.0   // ~2 chars per token for Korean
	tokens += float64(codeChars) / 3.0     // ~3 chars per token for code
	tokens += float64(otherChars) / 4.0    // ~4 chars per token for other

	// Add some buffer for special tokens (newlines, etc.)
	lineCount := countLines(text)
	tokens += float64(lineCount) * 0.5

	return int(tokens)
}

// CountFile counts tokens in a file
func (c *ApproximateCounter) CountFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return c.Count(string(data)), nil
}

// isCodeChar checks if a rune is typically found in code
func isCodeChar(r rune) bool {
	codeChars := []rune{'{', '}', '[', ']', '(', ')', '<', '>', '=', '+', '-', '*', '/', '%', '&', '|', '^', '~', '!', '?', ':', ';', ',', '.', '@', '#', '$', '_', '\\', '`', '"', '\''}
	for _, c := range codeChars {
		if r == c {
			return true
		}
	}
	return false
}

// countLines counts the number of lines in text
func countLines(text string) int {
	if len(text) == 0 {
		return 0
	}
	count := 1
	for _, r := range text {
		if r == '\n' {
			count++
		}
	}
	return count
}

// EstimateTokens is a simple helper function for quick estimation
func EstimateTokens(text string) int {
	counter := NewApproximateCounter()
	return counter.Count(text)
}

// EstimateTokensForBytes estimates tokens for a byte length
// Useful for quick budget calculations
func EstimateTokensForBytes(byteLen int) int {
	// Average: ~3.5 chars per token
	return byteLen / 4
}

// TokenBudget represents a token budget with tracking
type TokenBudget struct {
	Total     int
	Used      int
	Reserved  int
}

// NewTokenBudget creates a new token budget
func NewTokenBudget(total int) *TokenBudget {
	return &TokenBudget{
		Total:    total,
		Used:     0,
		Reserved: 0,
	}
}

// Available returns the available tokens
func (tb *TokenBudget) Available() int {
	return tb.Total - tb.Used - tb.Reserved
}

// CanFit checks if the given tokens can fit in the budget
func (tb *TokenBudget) CanFit(tokens int) bool {
	return tokens <= tb.Available()
}

// Use consumes tokens from the budget
func (tb *TokenBudget) Use(tokens int) bool {
	if !tb.CanFit(tokens) {
		return false
	}
	tb.Used += tokens
	return true
}

// Reserve reserves tokens for later use
func (tb *TokenBudget) Reserve(tokens int) bool {
	if tokens > tb.Available() {
		return false
	}
	tb.Reserved += tokens
	return true
}

// Release releases reserved tokens
func (tb *TokenBudget) Release(tokens int) {
	tb.Reserved -= tokens
	if tb.Reserved < 0 {
		tb.Reserved = 0
	}
}

// UsagePercent returns the usage percentage
func (tb *TokenBudget) UsagePercent() int {
	if tb.Total == 0 {
		return 0
	}
	return (tb.Used * 100) / tb.Total
}

// FormatUsage returns a formatted usage string
func (tb *TokenBudget) FormatUsage() string {
	return formatTokens(tb.Used) + " / " + formatTokens(tb.Total) + " tokens"
}

// formatTokens formats token count with K suffix
func formatTokens(tokens int) string {
	if tokens >= 1000 {
		return formatFloat(float64(tokens)/1000, 1) + "K"
	}
	return formatInt(tokens)
}

// formatFloat formats a float with given decimal places
func formatFloat(f float64, decimals int) string {
	format := "%." + formatInt(decimals) + "f"
	return sprintf(format, f)
}

// formatInt converts int to string
func formatInt(i int) string {
	return sprintf("%d", i)
}

// sprintf is a simple fmt.Sprintf wrapper
func sprintf(format string, args ...interface{}) string {
	// Simple implementation to avoid importing fmt
	if len(args) == 0 {
		return format
	}
	// For our simple cases
	result := format
	for _, arg := range args {
		switch v := arg.(type) {
		case int:
			result = replaceFirst(result, "%d", intToString(v))
		case float64:
			result = replaceFirst(result, "%.1f", floatToString(v, 1))
		}
	}
	return result
}

func replaceFirst(s, old, new string) string {
	for i := 0; i <= len(s)-len(old); i++ {
		if s[i:i+len(old)] == old {
			return s[:i] + new + s[i+len(old):]
		}
	}
	return s
}

func intToString(i int) string {
	if i == 0 {
		return "0"
	}
	negative := i < 0
	if negative {
		i = -i
	}
	var digits []byte
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func floatToString(f float64, decimals int) string {
	intPart := int(f)
	fracPart := int((f - float64(intPart)) * 10)
	if fracPart < 0 {
		fracPart = -fracPart
	}
	return intToString(intPart) + "." + intToString(fracPart)
}

// CharCount returns the character count (runes) in text
func CharCount(text string) int {
	return utf8.RuneCountInString(text)
}
