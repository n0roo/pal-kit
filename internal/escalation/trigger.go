package escalation

import (
	"time"
)

// WorkerContext represents the context for trigger evaluation
type WorkerContext struct {
	SessionID    string
	PortID       string
	TestRetries  int
	MaxRetries   int
	TokensUsed   int
	TokenBudget  int
	StartTime    time.Time
	Timeout      time.Duration
	TestsPassed  int
	TestsFailed  int
	BuildFailed  bool
	IsBlocked    bool
	BlockReason  string
}

// TriggerCondition is a function that evaluates whether to trigger an escalation
type TriggerCondition func(ctx WorkerContext) bool

// EscalationTrigger defines an automatic escalation trigger
type EscalationTrigger struct {
	Type        EscalationType
	Condition   TriggerCondition
	Severity    Severity
	AutoResolve bool
	BuildIssue  func(ctx WorkerContext) string
	BuildSuggestion func(ctx WorkerContext) string
}

// DefaultTriggers returns the default set of escalation triggers
func DefaultTriggers() []EscalationTrigger {
	return []EscalationTrigger{
		// Test failure - high severity
		{
			Type: TypeTestFail,
			Condition: func(ctx WorkerContext) bool {
				return ctx.TestRetries >= ctx.MaxRetries
			},
			Severity: SeverityHigh,
			BuildIssue: func(ctx WorkerContext) string {
				return "테스트 반복 실패로 최대 재시도 횟수 초과"
			},
			BuildSuggestion: func(ctx WorkerContext) string {
				return "실패한 테스트 케이스를 검토하고 근본 원인을 파악하세요"
			},
		},
		// Token exhausted - medium severity
		{
			Type: TypeTokenExceeded,
			Condition: func(ctx WorkerContext) bool {
				if ctx.TokenBudget == 0 {
					return false
				}
				return float64(ctx.TokensUsed) >= float64(ctx.TokenBudget)*0.95
			},
			Severity: SeverityMedium,
			BuildIssue: func(ctx WorkerContext) string {
				return "토큰 예산의 95% 이상 소진"
			},
			BuildSuggestion: func(ctx WorkerContext) string {
				return "Compact를 실행하거나 토큰 예산을 늘리세요"
			},
		},
		// Timeout - high severity
		{
			Type:     TypeBlocked,
			Condition: func(ctx WorkerContext) bool {
				if ctx.Timeout == 0 {
					return false
				}
				return time.Since(ctx.StartTime) > ctx.Timeout
			},
			Severity: SeverityHigh,
			BuildIssue: func(ctx WorkerContext) string {
				return "작업 타임아웃 초과"
			},
			BuildSuggestion: func(ctx WorkerContext) string {
				return "작업 범위를 줄이거나 더 작은 단위로 분할하세요"
			},
		},
		// Build failure - high severity
		{
			Type: TypeBuildFail,
			Condition: func(ctx WorkerContext) bool {
				return ctx.BuildFailed
			},
			Severity: SeverityHigh,
			BuildIssue: func(ctx WorkerContext) string {
				return "빌드 실패"
			},
			BuildSuggestion: func(ctx WorkerContext) string {
				return "컴파일 오류를 수정하세요"
			},
		},
		// Blocked - medium severity
		{
			Type: TypeBlocked,
			Condition: func(ctx WorkerContext) bool {
				return ctx.IsBlocked
			},
			Severity: SeverityMedium,
			BuildIssue: func(ctx WorkerContext) string {
				if ctx.BlockReason != "" {
					return "작업 블록됨: " + ctx.BlockReason
				}
				return "작업이 블록됨"
			},
			BuildSuggestion: func(ctx WorkerContext) string {
				return "블로킹 의존성을 해결하세요"
			},
		},
		// Compact warning - low severity
		{
			Type: TypeCompactWarning,
			Condition: func(ctx WorkerContext) bool {
				if ctx.TokenBudget == 0 {
					return false
				}
				usage := float64(ctx.TokensUsed) / float64(ctx.TokenBudget)
				return usage >= 0.8 && usage < 0.95
			},
			Severity:    SeverityLow,
			AutoResolve: true,
			BuildIssue: func(ctx WorkerContext) string {
				return "토큰 사용량 80% 초과 - Compact 권장"
			},
			BuildSuggestion: func(ctx WorkerContext) string {
				return "곧 Compact가 필요합니다"
			},
		},
	}
}

// TriggerService handles escalation trigger checking
type TriggerService struct {
	escalationSvc *Service
	triggers      []EscalationTrigger
}

// NewTriggerService creates a new trigger service
func NewTriggerService(escalationSvc *Service) *TriggerService {
	return &TriggerService{
		escalationSvc: escalationSvc,
		triggers:      DefaultTriggers(),
	}
}

// AddTrigger adds a custom trigger
func (ts *TriggerService) AddTrigger(trigger EscalationTrigger) {
	ts.triggers = append(ts.triggers, trigger)
}

// CheckTriggers evaluates all triggers and creates escalations if needed
func (ts *TriggerService) CheckTriggers(ctx WorkerContext) ([]*EnhancedEscalation, error) {
	var escalations []*EnhancedEscalation

	for _, trigger := range ts.triggers {
		if trigger.Condition(ctx) {
			issue := "자동 감지된 문제"
			if trigger.BuildIssue != nil {
				issue = trigger.BuildIssue(ctx)
			}

			suggestion := ""
			if trigger.BuildSuggestion != nil {
				suggestion = trigger.BuildSuggestion(ctx)
			}

			esc, err := ts.escalationSvc.CreateEnhanced(EnhancedEscalationOptions{
				FromSession: ctx.SessionID,
				FromPort:    ctx.PortID,
				Type:        trigger.Type,
				Severity:    trigger.Severity,
				Issue:       issue,
				Suggestion:  suggestion,
				Context: map[string]interface{}{
					"tokens_used":   ctx.TokensUsed,
					"token_budget":  ctx.TokenBudget,
					"test_retries":  ctx.TestRetries,
					"tests_passed":  ctx.TestsPassed,
					"tests_failed":  ctx.TestsFailed,
					"auto_resolved": trigger.AutoResolve,
				},
			})
			if err != nil {
				return escalations, err
			}

			escalations = append(escalations, esc)
		}
	}

	return escalations, nil
}

// CheckSingleTrigger checks a specific trigger type
func (ts *TriggerService) CheckSingleTrigger(triggerType EscalationType, ctx WorkerContext) *EscalationTrigger {
	for _, trigger := range ts.triggers {
		if trigger.Type == triggerType && trigger.Condition(ctx) {
			return &trigger
		}
	}
	return nil
}

// EvaluateSeverity returns the highest severity from triggered escalations
func (ts *TriggerService) EvaluateSeverity(ctx WorkerContext) Severity {
	highestSeverity := SeverityLow
	severityOrder := map[Severity]int{
		SeverityLow:      1,
		SeverityMedium:   2,
		SeverityHigh:     3,
		SeverityCritical: 4,
	}

	for _, trigger := range ts.triggers {
		if trigger.Condition(ctx) {
			if severityOrder[trigger.Severity] > severityOrder[highestSeverity] {
				highestSeverity = trigger.Severity
			}
		}
	}

	return highestSeverity
}

// HasCriticalTriggers checks if any critical triggers are active
func (ts *TriggerService) HasCriticalTriggers(ctx WorkerContext) bool {
	for _, trigger := range ts.triggers {
		if trigger.Severity == SeverityCritical && trigger.Condition(ctx) {
			return true
		}
	}
	return false
}

// GetActiveTriggersCount returns the count of active triggers
func (ts *TriggerService) GetActiveTriggersCount(ctx WorkerContext) int {
	count := 0
	for _, trigger := range ts.triggers {
		if trigger.Condition(ctx) {
			count++
		}
	}
	return count
}
