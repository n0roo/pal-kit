package agentv2

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AgentType defines the type of agent
type AgentType string

const (
	TypeSpec     AgentType = "spec"
	TypeOperator AgentType = "operator"
	TypeWorker   AgentType = "worker"
	TypeTest     AgentType = "test"
)

// VersionStatus defines the status of an agent version
type VersionStatus string

const (
	StatusActive       VersionStatus = "active"
	StatusDeprecated   VersionStatus = "deprecated"
	StatusExperimental VersionStatus = "experimental"
)

// Agent represents an agent definition
type Agent struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           AgentType `json:"type"`
	Description    string    `json:"description,omitempty"`
	Capabilities   []string  `json:"capabilities,omitempty"`
	CurrentVersion int       `json:"current_version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AgentVersion represents a specific version of an agent
type AgentVersion struct {
	ID                 string        `json:"id"`
	AgentID            string        `json:"agent_id"`
	Version            int           `json:"version"`
	SpecContent        string        `json:"spec_content"`
	SpecHash           string        `json:"spec_hash"`
	ChangeSummary      string        `json:"change_summary,omitempty"`
	ChangeReason       string        `json:"change_reason,omitempty"`
	AvgAttentionScore  float64       `json:"avg_attention_score,omitempty"`
	AvgCompletionRate  float64       `json:"avg_completion_rate,omitempty"`
	AvgTokenEfficiency float64       `json:"avg_token_efficiency,omitempty"`
	UsageCount         int           `json:"usage_count"`
	Status             VersionStatus `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
}

// AgentPerformance represents performance metrics for an agent session
type AgentPerformance struct {
	ID                     string    `json:"id"`
	AgentID                string    `json:"agent_id"`
	AgentVersion           int       `json:"agent_version"`
	SessionID              string    `json:"session_id"`
	AttentionAvg           float64   `json:"attention_avg"`
	AttentionMin           float64   `json:"attention_min"`
	TokenUsed              int       `json:"token_used"`
	CompactCount           int       `json:"compact_count"`
	CompletionTimeSeconds  int       `json:"completion_time_seconds"`
	Outcome                string    `json:"outcome"` // success, partial, failed
	QualityScore           float64   `json:"quality_score"`
	Feedback               string    `json:"feedback,omitempty"`
	ImprovementSuggestions []string  `json:"improvement_suggestions,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
}

// Store handles agent persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new agent store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateAgent creates a new agent
func (s *Store) CreateAgent(agent *Agent) error {
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	agent.CurrentVersion = 1
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()

	capabilitiesJSON, _ := json.Marshal(agent.Capabilities)

	_, err := s.db.Exec(`
		INSERT INTO agents (id, name, type, description, capabilities, current_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, agent.ID, agent.Name, agent.Type, agent.Description, string(capabilitiesJSON),
		agent.CurrentVersion, agent.CreatedAt, agent.UpdatedAt)
	if err != nil {
		return fmt.Errorf("에이전트 생성 실패: %w", err)
	}

	return nil
}

// GetAgent gets an agent by ID
func (s *Store) GetAgent(id string) (*Agent, error) {
	var agent Agent
	var capabilitiesJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, type, description, capabilities, current_version, created_at, updated_at
		FROM agents WHERE id = ?
	`, id).Scan(&agent.ID, &agent.Name, &agent.Type, &agent.Description,
		&capabilitiesJSON, &agent.CurrentVersion, &agent.CreatedAt, &agent.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if capabilitiesJSON.Valid {
		json.Unmarshal([]byte(capabilitiesJSON.String), &agent.Capabilities)
	}

	return &agent, nil
}

// GetAgentByName gets an agent by name
func (s *Store) GetAgentByName(name string) (*Agent, error) {
	var agent Agent
	var capabilitiesJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, type, description, capabilities, current_version, created_at, updated_at
		FROM agents WHERE name = ?
	`, name).Scan(&agent.ID, &agent.Name, &agent.Type, &agent.Description,
		&capabilitiesJSON, &agent.CurrentVersion, &agent.CreatedAt, &agent.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if capabilitiesJSON.Valid {
		json.Unmarshal([]byte(capabilitiesJSON.String), &agent.Capabilities)
	}

	return &agent, nil
}

// ListAgents lists all agents
func (s *Store) ListAgents(agentType AgentType) ([]*Agent, error) {
	query := `SELECT id, name, type, description, capabilities, current_version, created_at, updated_at FROM agents`
	args := []interface{}{}

	if agentType != "" {
		query += " WHERE type = ?"
		args = append(args, agentType)
	}
	query += " ORDER BY name"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		var agent Agent
		var capabilitiesJSON sql.NullString
		err := rows.Scan(&agent.ID, &agent.Name, &agent.Type, &agent.Description,
			&capabilitiesJSON, &agent.CurrentVersion, &agent.CreatedAt, &agent.UpdatedAt)
		if err != nil {
			continue
		}
		if capabilitiesJSON.Valid {
			json.Unmarshal([]byte(capabilitiesJSON.String), &agent.Capabilities)
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// CreateVersion creates a new version of an agent
func (s *Store) CreateVersion(version *AgentVersion) error {
	if version.ID == "" {
		version.ID = uuid.New().String()
	}
	version.SpecHash = hashContent(version.SpecContent)
	version.Status = StatusActive
	version.CreatedAt = time.Now()

	// Get next version number
	var maxVersion int
	s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM agent_versions WHERE agent_id = ?`,
		version.AgentID).Scan(&maxVersion)
	version.Version = maxVersion + 1

	_, err := s.db.Exec(`
		INSERT INTO agent_versions (
			id, agent_id, version, spec_content, spec_hash, change_summary, change_reason,
			avg_attention_score, avg_completion_rate, avg_token_efficiency, usage_count, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, version.ID, version.AgentID, version.Version, version.SpecContent, version.SpecHash,
		version.ChangeSummary, version.ChangeReason, version.AvgAttentionScore,
		version.AvgCompletionRate, version.AvgTokenEfficiency, version.UsageCount,
		version.Status, version.CreatedAt)
	if err != nil {
		return fmt.Errorf("버전 생성 실패: %w", err)
	}

	// Update agent's current version
	_, err = s.db.Exec(`
		UPDATE agents SET current_version = ?, updated_at = ? WHERE id = ?
	`, version.Version, time.Now(), version.AgentID)
	if err != nil {
		return fmt.Errorf("에이전트 버전 업데이트 실패: %w", err)
	}

	// Deprecate old versions (optional: keep last N active)
	_, _ = s.db.Exec(`
		UPDATE agent_versions SET status = ?
		WHERE agent_id = ? AND version < ? AND status = ?
	`, StatusDeprecated, version.AgentID, version.Version-2, StatusActive)

	return nil
}

// GetVersion gets a specific version of an agent
func (s *Store) GetVersion(agentID string, version int) (*AgentVersion, error) {
	var av AgentVersion
	err := s.db.QueryRow(`
		SELECT id, agent_id, version, spec_content, spec_hash, change_summary, change_reason,
		       avg_attention_score, avg_completion_rate, avg_token_efficiency, usage_count, status, created_at
		FROM agent_versions
		WHERE agent_id = ? AND version = ?
	`, agentID, version).Scan(&av.ID, &av.AgentID, &av.Version, &av.SpecContent, &av.SpecHash,
		&av.ChangeSummary, &av.ChangeReason, &av.AvgAttentionScore, &av.AvgCompletionRate,
		&av.AvgTokenEfficiency, &av.UsageCount, &av.Status, &av.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &av, nil
}

// GetCurrentVersion gets the current version of an agent
func (s *Store) GetCurrentVersion(agentID string) (*AgentVersion, error) {
	agent, err := s.GetAgent(agentID)
	if err != nil {
		return nil, err
	}
	return s.GetVersion(agentID, agent.CurrentVersion)
}

// ListVersions lists all versions of an agent
func (s *Store) ListVersions(agentID string) ([]*AgentVersion, error) {
	rows, err := s.db.Query(`
		SELECT id, agent_id, version, spec_content, spec_hash, change_summary, change_reason,
		       avg_attention_score, avg_completion_rate, avg_token_efficiency, usage_count, status, created_at
		FROM agent_versions
		WHERE agent_id = ?
		ORDER BY version DESC
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*AgentVersion
	for rows.Next() {
		var av AgentVersion
		err := rows.Scan(&av.ID, &av.AgentID, &av.Version, &av.SpecContent, &av.SpecHash,
			&av.ChangeSummary, &av.ChangeReason, &av.AvgAttentionScore, &av.AvgCompletionRate,
			&av.AvgTokenEfficiency, &av.UsageCount, &av.Status, &av.CreatedAt)
		if err != nil {
			continue
		}
		versions = append(versions, &av)
	}

	return versions, nil
}

// RecordPerformance records performance metrics for a session
func (s *Store) RecordPerformance(perf *AgentPerformance) error {
	if perf.ID == "" {
		perf.ID = uuid.New().String()
	}
	perf.CreatedAt = time.Now()

	suggestionsJSON, _ := json.Marshal(perf.ImprovementSuggestions)

	_, err := s.db.Exec(`
		INSERT INTO agent_performance (
			id, agent_id, agent_version, session_id, attention_avg, attention_min,
			token_used, compact_count, completion_time_seconds, outcome, quality_score,
			feedback, improvement_suggestions, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, perf.ID, perf.AgentID, perf.AgentVersion, perf.SessionID, perf.AttentionAvg,
		perf.AttentionMin, perf.TokenUsed, perf.CompactCount, perf.CompletionTimeSeconds,
		perf.Outcome, perf.QualityScore, perf.Feedback, string(suggestionsJSON), perf.CreatedAt)
	if err != nil {
		return fmt.Errorf("성능 기록 실패: %w", err)
	}

	// Update usage count
	_, _ = s.db.Exec(`
		UPDATE agent_versions SET usage_count = usage_count + 1
		WHERE agent_id = ? AND version = ?
	`, perf.AgentID, perf.AgentVersion)

	// Update aggregated stats
	s.updateVersionStats(perf.AgentID, perf.AgentVersion)

	return nil
}

// updateVersionStats updates the aggregated stats for a version
func (s *Store) updateVersionStats(agentID string, version int) {
	_, _ = s.db.Exec(`
		UPDATE agent_versions SET
			avg_attention_score = (
				SELECT AVG(attention_avg) FROM agent_performance
				WHERE agent_id = ? AND agent_version = ?
			),
			avg_completion_rate = (
				SELECT AVG(CASE WHEN outcome = 'success' THEN 1.0 ELSE 0.0 END)
				FROM agent_performance WHERE agent_id = ? AND agent_version = ?
			),
			avg_token_efficiency = (
				SELECT AVG(CASE WHEN token_used > 0 THEN quality_score / token_used * 10000 ELSE 0 END)
				FROM agent_performance WHERE agent_id = ? AND agent_version = ?
			)
		WHERE agent_id = ? AND version = ?
	`, agentID, version, agentID, version, agentID, version, agentID, version)
}

// GetVersionStats gets performance statistics for a version
func (s *Store) GetVersionStats(agentID string, version int) (*VersionStats, error) {
	var stats VersionStats
	err := s.db.QueryRow(`
		SELECT 
			COUNT(*) as usage_count,
			COALESCE(AVG(attention_avg), 0) as avg_attention,
			COALESCE(AVG(quality_score), 0) as avg_quality,
			COALESCE(SUM(CASE WHEN outcome = 'success' THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(*), 0), 0) as success_rate,
			COALESCE(AVG(token_used), 0) as avg_tokens,
			COALESCE(AVG(compact_count), 0) as avg_compacts
		FROM agent_performance
		WHERE agent_id = ? AND agent_version = ?
	`, agentID, version).Scan(&stats.UsageCount, &stats.AvgAttention, &stats.AvgQuality,
		&stats.SuccessRate, &stats.AvgTokens, &stats.AvgCompacts)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// VersionStats represents aggregated statistics for a version
type VersionStats struct {
	UsageCount   int     `json:"usage_count"`
	AvgAttention float64 `json:"avg_attention"`
	AvgQuality   float64 `json:"avg_quality"`
	SuccessRate  float64 `json:"success_rate"`
	AvgTokens    float64 `json:"avg_tokens"`
	AvgCompacts  float64 `json:"avg_compacts"`
}

// CompareVersions compares two versions
func (s *Store) CompareVersions(agentID string, v1, v2 int) (*VersionComparison, error) {
	stats1, err := s.GetVersionStats(agentID, v1)
	if err != nil {
		return nil, err
	}
	stats2, err := s.GetVersionStats(agentID, v2)
	if err != nil {
		return nil, err
	}

	return &VersionComparison{
		Version1:          v1,
		Version2:          v2,
		AttentionDiff:     stats2.AvgAttention - stats1.AvgAttention,
		QualityDiff:       stats2.AvgQuality - stats1.AvgQuality,
		SuccessRateDiff:   stats2.SuccessRate - stats1.SuccessRate,
		TokenDiff:         stats2.AvgTokens - stats1.AvgTokens,
		CompactDiff:       stats2.AvgCompacts - stats1.AvgCompacts,
		RecommendedAction: determineRecommendation(stats1, stats2),
	}, nil
}

// VersionComparison represents comparison between two versions
type VersionComparison struct {
	Version1          int     `json:"version1"`
	Version2          int     `json:"version2"`
	AttentionDiff     float64 `json:"attention_diff"`
	QualityDiff       float64 `json:"quality_diff"`
	SuccessRateDiff   float64 `json:"success_rate_diff"`
	TokenDiff         float64 `json:"token_diff"`
	CompactDiff       float64 `json:"compact_diff"`
	RecommendedAction string  `json:"recommended_action"`
}

func determineRecommendation(stats1, stats2 *VersionStats) string {
	if stats2.SuccessRate > stats1.SuccessRate && stats2.AvgAttention >= stats1.AvgAttention {
		return "adopt_v2"
	}
	if stats2.SuccessRate < stats1.SuccessRate-5 {
		return "rollback_to_v1"
	}
	if stats2.UsageCount < 5 {
		return "needs_more_data"
	}
	return "maintain_current"
}

func hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:8]) // First 8 bytes
}
