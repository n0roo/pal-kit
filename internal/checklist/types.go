package checklist

import "time"

// ChecklistItem represents a single checklist item
type ChecklistItem struct {
	ID          string `json:"id" yaml:"id"`
	Description string `json:"description" yaml:"description"`
	Type        string `json:"type" yaml:"type"`       // auto, manual
	Command     string `json:"command" yaml:"command"` // Command to run for auto verification
	Required    bool   `json:"required" yaml:"required"`
}

// Checklist represents a complete checklist for an agent type
type Checklist struct {
	AgentType string          `json:"agent_type" yaml:"agent_type"`
	PortType  string          `json:"port_type" yaml:"port_type"`
	Items     []ChecklistItem `json:"items" yaml:"items"`
}

// CheckResult represents the result of checking a single item
type CheckResult struct {
	ItemID      string        `json:"item_id"`
	Description string        `json:"description"`
	Passed      bool          `json:"passed"`
	Message     string        `json:"message"`
	Output      string        `json:"output,omitempty"`
	Duration    time.Duration `json:"duration"`
	Required    bool          `json:"required"`
}

// VerificationResult represents the overall verification result
type VerificationResult struct {
	PortID      string        `json:"port_id"`
	Passed      bool          `json:"passed"`
	Results     []CheckResult `json:"results"`
	PassedCount int           `json:"passed_count"`
	FailedCount int           `json:"failed_count"`
	BlockedBy   []string      `json:"blocked_by,omitempty"` // Failed required items
	Duration    time.Duration `json:"duration"`
}

// DefaultWorkerChecklist returns the default checklist for worker agents
func DefaultWorkerChecklist() Checklist {
	return Checklist{
		AgentType: "worker",
		Items: []ChecklistItem{
			{
				ID:          "build",
				Description: "빌드 성공",
				Type:        "auto",
				Command:     "go build ./...",
				Required:    true,
			},
			{
				ID:          "test",
				Description: "테스트 통과",
				Type:        "auto",
				Command:     "go test ./...",
				Required:    true,
			},
			{
				ID:          "lint",
				Description: "린트 경고 없음",
				Type:        "auto",
				Command:     "golangci-lint run",
				Required:    false,
			},
		},
	}
}

// DefaultReviewerChecklist returns the default checklist for reviewer agents
func DefaultReviewerChecklist() Checklist {
	return Checklist{
		AgentType: "reviewer",
		Items: []ChecklistItem{
			{
				ID:          "build",
				Description: "빌드 성공",
				Type:        "auto",
				Command:     "go build ./...",
				Required:    true,
			},
			{
				ID:          "test",
				Description: "테스트 통과",
				Type:        "auto",
				Command:     "go test ./...",
				Required:    true,
			},
		},
	}
}

// GetDefaultChecklist returns the default checklist for an agent type
func GetDefaultChecklist(agentType string) Checklist {
	switch agentType {
	case "worker":
		return DefaultWorkerChecklist()
	case "reviewer":
		return DefaultReviewerChecklist()
	default:
		return DefaultWorkerChecklist()
	}
}
