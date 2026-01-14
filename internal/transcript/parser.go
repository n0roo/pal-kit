package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

// Usage represents aggregated token usage from a transcript
type Usage struct {
	InputTokens       int64   `json:"input_tokens"`
	OutputTokens      int64   `json:"output_tokens"`
	CacheReadTokens   int64   `json:"cache_read_tokens"`
	CacheCreateTokens int64   `json:"cache_create_tokens"`
	CostUSD           float64 `json:"cost_usd"`
	MessageCount      int     `json:"message_count"`
}

// TranscriptEntry represents a single entry in the JSONL transcript
type TranscriptEntry struct {
	Type      string         `json:"type"`
	SessionID string         `json:"sessionId"`
	Message   *MessageDetail `json:"message,omitempty"`
}

// MessageDetail contains the message details including usage
type MessageDetail struct {
	Model string       `json:"model"`
	Usage *UsageDetail `json:"usage,omitempty"`
}

// UsageDetail contains token usage information
type UsageDetail struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// ModelPricing contains pricing per 1M tokens for a model
type ModelPricing struct {
	InputPer1M       float64
	OutputPer1M      float64
	CacheReadPer1M   float64
	CacheCreatePer1M float64
}

// Pricing for different models (per 1M tokens)
var modelPricing = map[string]ModelPricing{
	"claude-opus-4-5-20251101": {
		InputPer1M:       15.0,
		OutputPer1M:      75.0,
		CacheReadPer1M:   1.5,
		CacheCreatePer1M: 18.75,
	},
	"claude-sonnet-4-20250514": {
		InputPer1M:       3.0,
		OutputPer1M:      15.0,
		CacheReadPer1M:   0.30,
		CacheCreatePer1M: 3.75,
	},
	"claude-3-5-sonnet-20241022": {
		InputPer1M:       3.0,
		OutputPer1M:      15.0,
		CacheReadPer1M:   0.30,
		CacheCreatePer1M: 3.75,
	},
	"claude-3-5-haiku-20241022": {
		InputPer1M:       0.80,
		OutputPer1M:      4.0,
		CacheReadPer1M:   0.08,
		CacheCreatePer1M: 1.0,
	},
}

// defaultPricing is used for unknown models (assumes sonnet-like pricing)
var defaultPricing = ModelPricing{
	InputPer1M:       3.0,
	OutputPer1M:      15.0,
	CacheReadPer1M:   0.30,
	CacheCreatePer1M: 3.75,
}

// ParseFile parses a JSONL transcript file and returns aggregated usage
func ParseFile(path string) (*Usage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("파일 열기 실패: %w", err)
	}
	defer file.Close()

	usage := &Usage{}
	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line size

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry TranscriptEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines
			continue
		}

		// Only process assistant messages with usage data
		if entry.Type != "assistant" {
			continue
		}
		if entry.Message == nil || entry.Message.Usage == nil {
			continue
		}

		u := entry.Message.Usage
		usage.InputTokens += u.InputTokens
		usage.OutputTokens += u.OutputTokens
		usage.CacheReadTokens += u.CacheReadInputTokens
		usage.CacheCreateTokens += u.CacheCreationInputTokens
		usage.MessageCount++

		// Calculate cost for this message
		pricing := getPricing(entry.Message.Model)
		usage.CostUSD += calculateCost(u, pricing)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}

	return usage, nil
}

// getPricing returns pricing for a model
func getPricing(model string) ModelPricing {
	if p, ok := modelPricing[model]; ok {
		return p
	}
	return defaultPricing
}

// calculateCost calculates cost for a single usage entry
func calculateCost(u *UsageDetail, p ModelPricing) float64 {
	cost := 0.0
	cost += float64(u.InputTokens) / 1_000_000 * p.InputPer1M
	cost += float64(u.OutputTokens) / 1_000_000 * p.OutputPer1M
	cost += float64(u.CacheReadInputTokens) / 1_000_000 * p.CacheReadPer1M
	cost += float64(u.CacheCreationInputTokens) / 1_000_000 * p.CacheCreatePer1M
	return cost
}

// UserMessage represents a user message from the transcript
type UserMessage struct {
	Content   string `json:"content"`
	Timestamp string `json:"timestamp,omitempty"`
	Index     int    `json:"index"`
}

// HumanEntry represents a human message entry in the JSONL
type HumanEntry struct {
	Type    string `json:"type"`
	Message struct {
		Content []ContentBlock `json:"content,omitempty"`
		Text    string         `json:"text,omitempty"`
	} `json:"message,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// ContentBlock represents a content block in message
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// GetFirstUserMessage extracts the first user message from transcript
func GetFirstUserMessage(path string) (*UserMessage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("파일 열기 실패: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	index := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry HumanEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Type != "human" {
			continue
		}

		// 첫 번째 사용자 메시지 추출
		var content string
		if entry.Message.Text != "" {
			content = entry.Message.Text
		} else if len(entry.Message.Content) > 0 {
			for _, block := range entry.Message.Content {
				if block.Type == "text" && block.Text != "" {
					content = block.Text
					break
				}
			}
		}

		if content != "" {
			return &UserMessage{
				Content:   content,
				Timestamp: entry.Timestamp,
				Index:     index,
			}, nil
		}
		index++
	}

	return nil, fmt.Errorf("사용자 메시지를 찾을 수 없습니다")
}

// GetUserMessages extracts all user messages from transcript
func GetUserMessages(path string, limit int) ([]UserMessage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("파일 열기 실패: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var messages []UserMessage
	index := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry HumanEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Type != "human" {
			continue
		}

		var content string
		if entry.Message.Text != "" {
			content = entry.Message.Text
		} else if len(entry.Message.Content) > 0 {
			for _, block := range entry.Message.Content {
				if block.Type == "text" && block.Text != "" {
					content = block.Text
					break
				}
			}
		}

		if content != "" {
			messages = append(messages, UserMessage{
				Content:   content,
				Timestamp: entry.Timestamp,
				Index:     index,
			})

			if limit > 0 && len(messages) >= limit {
				break
			}
		}
		index++
	}

	return messages, nil
}
