package agentmgr

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
	VersionActive       VersionStatus = "active"
	VersionDeprecated   VersionStatus = "deprecated"
	VersionExperimental VersionStatus = "experimental"
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

// AgentPerformance represents performance metrics for an agent usage
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

// Manager handles agent management operations
type Manager struct {
	db *sql.DB
}

// NewManager creates a new agent manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// CreateAgent creates a new agent
func (m *Manager) CreateAgent(name string, agentType AgentType, description string, capabilities []string) (*Agent, error) {
	id := uuid.New().String()
	now := time.Now()

	capsJSON, _ := json.Marshal(capabilities)

	_, err := m.db.Exec(`
		INSERT INTO agents (id, name, type, description, capabilities, current_version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)
	`, id, name, agentType, description, string(capsJSON), now, now)
	if err != nil {
		return nil, fmt.Errorf("에이전트 생성 실패: %w", err)
	}

	return &Agent{
		ID:             id,
		Name:           name,
		Type:           agentType,
		Description:    description,
		Capabilities:   capabilities,
		CurrentVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// GetAgent retrieves an agent by ID
func (m *Manager) GetAgent(id string) (*Agent, error) {
	var agent Agent
	var capsJSON sql.NullString

	err := m.db.QueryRow(`
		SELECT id, name, type, description, capabilities, current_version, created_at, updated_at
		FROM agents WHERE id = ?
	`, id).Scan(
		&agent.ID, &agent.Name, &agent.Type, &agent.Description,
		&capsJSON, &agent.CurrentVersion, &agent.CreatedAt, &agent.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("에이전트 조회 실패: %w", err)
	}

	if capsJSON.Valid {
		json.Unmarshal([]byte(capsJSON.String), &agent.Capabilities)
	}

	return &agent, nil
}

// GetAgentByName retrieves an agent by name
func (m *Manager) GetAgentByName(name string) (*Agent, error) {
	var agent Agent
	var capsJSON sql.NullString

	err := m.db.QueryRow(`
		SELECT id, name, type, description, capabilities, current_version, created_at, updated_at
		FROM agents WHERE name = ?
	`, name).Scan(
		&agent.ID, &agent.Name, &agent.Type, &agent.Description,
		&capsJSON, &agent.CurrentVersion, &agent.CreatedAt, &agent.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("에이전트 조회 실패: %w", err)
	}

	if capsJSON.Valid {
		json.Unmarshal([]byte(capsJSON.String), &agent.Capabilities)
	}

	return &agent, nil
}

// ListAgents lists all agents
func (m *Manager) ListAgents(agentType AgentType) ([]*Agent, error) {
	query := `SELECT id, name, type, description, capabilities, current_version, created_at, updated_at FROM agents`
	args := []interface{}{}
	
	if agentType != "" {
		query += " WHERE type = ?"
		args = append(args, agentType)
	}
	query += " ORDER BY name"

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("에이전트 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var agents []*Agent
	for rows.Next() {
		var agent Agent
		var capsJSON sql.NullString

		err := rows.Scan(
			&agent.ID, &agent.Name, &agent.Type, &agent.Description,
			&capsJSON, &agent.CurrentVersion, &agent.CreatedAt, &agent.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if capsJSON.Valid {
			json.Unmarshal([]byte(capsJSON.String), &agent.Capabilities)
		}

		agents = append(agents, &agent)
	}

	return agents, nil
}

// CreateVersion creates a new version of an agent
func (m *Manager) CreateVersion(agentID string, specContent string, changeSummary, changeReason string) (*AgentVersion, error) {
	// 현재 버전 조회
	var currentVersion int
	err := m.db.QueryRow(`SELECT current_version FROM agents WHERE id = ?`, agentID).Scan(&currentVersion)
	if err != nil {
		return nil, fmt.Errorf("에이전트 조회 실패: %w", err)
	}

	newVersion := currentVersion + 1
	id := uuid.New().String()
	specHash := hashContent(specContent)
	now := time.Now()

	// 트랜잭션 시작
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// 새 버전 생성
	_, err = tx.Exec(`
		INSERT INTO agent_versions (id, agent_id, version, spec_content, spec_hash, change_summary, change_reason, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?)
	`, id, agentID, newVersion, specContent, specHash, changeSummary, changeReason, now)
	if err != nil {
		return nil, fmt.Errorf("버전 생성 실패: %w", err)
	}

	// 에이전트 현재 버전 업데이트
	_, err = tx.Exec(`UPDATE agents SET current_version = ?, updated_at = ? WHERE id = ?`, newVersion, now, agentID)
	if err != nil {
		return nil, fmt.Errorf("에이전트 버전 업데이트 실패: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &AgentVersion{
		ID:            id,
		AgentID:       agentID,
		Version:       newVersion,
		SpecContent:   specContent,
		SpecHash:      specHash,
		ChangeSummary: changeSummary,
		ChangeReason:  changeReason,
		Status:        VersionActive,
		CreatedAt:     now,
	}, nil
}

// GetVersion retrieves a specific version
func (m *Manager) GetVersion(agentID string, version int) (*AgentVersion, error) {
	var v AgentVersion
	var changeSummary, changeReason sql.NullString

	err := m.db.QueryRow(`
		SELECT id, agent_id, version, spec_content, spec_hash, change_summary, change_reason,
		       avg_attention_score, avg_completion_rate, avg_token_efficiency, usage_count, status, created_at
		FROM agent_versions
		WHERE agent_id = ? AND version = ?
	`, agentID, version).Scan(
		&v.ID, &v.AgentID, &v.Version, &v.SpecContent, &v.SpecHash,
		&changeSummary, &changeReason, &v.AvgAttentionScore, &v.AvgCompletionRate,
		&v.AvgTokenEfficiency, &v.UsageCount, &v.Status, &v.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("버전 조회 실패: %w", err)
	}

	if changeSummary.Valid {
		v.ChangeSummary = changeSummary.String
	}
	if changeReason.Valid {
		v.ChangeReason = changeReason.String
	}

	return &v, nil
}

// GetCurrentVersion retrieves the current version of an agent
func (m *Manager) GetCurrentVersion(agentID string) (*AgentVersion, error) {
	var currentVersion int
	err := m.db.QueryRow(`SELECT current_version FROM agents WHERE id = ?`, agentID).Scan(&currentVersion)
	if err != nil {
		return nil, fmt.Errorf("에이전트 조회 실패: %w", err)
	}
	return m.GetVersion(agentID, currentVersion)
}

// ListVersions lists all versions of an agent
func (m *Manager) ListVersions(agentID string) ([]*AgentVersion, error) {
	rows, err := m.db.Query(`
		SELECT id, agent_id, version, spec_content, spec_hash, change_summary, change_reason,
		       avg_attention_score, avg_completion_rate, avg_token_efficiency, usage_count, status, created_at
		FROM agent_versions
		WHERE agent_id = ?
		ORDER BY version DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("버전 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var versions []*AgentVersion
	for rows.Next() {
		var v AgentVersion
		var changeSummary, changeReason sql.NullString

		err := rows.Scan(
			&v.ID, &v.AgentID, &v.Version, &v.SpecContent, &v.SpecHash,
			&changeSummary, &changeReason, &v.AvgAttentionScore, &v.AvgCompletionRate,
			&v.AvgTokenEfficiency, &v.UsageCount, &v.Status, &v.CreatedAt,
		)
		if err != nil {
			continue
		}

		if changeSummary.Valid {
			v.ChangeSummary = changeSummary.String
		}
		if changeReason.Valid {
			v.ChangeReason = changeReason.String
		}

		versions = append(versions, &v)
	}

	return versions, nil
}

// DeprecateVersion marks a version as deprecated
func (m *Manager) DeprecateVersion(agentID string, version int) error {
	_, err := m.db.Exec(`
		UPDATE agent_versions SET status = 'deprecated' WHERE agent_id = ? AND version = ?
	`, agentID, version)
	return err
}

// RecordPerformance records performance metrics for an agent usage
func (m *Manager) RecordPerformance(perf *AgentPerformance) error {
	if perf.ID == "" {
		perf.ID = uuid.New().String()
	}
	if perf.CreatedAt.IsZero() {
		perf.CreatedAt = time.Now()
	}

	suggestionsJSON, _ := json.Marshal(perf.ImprovementSuggestions)

	_, err := m.db.Exec(`
		INSERT INTO agent_performance (
			id, agent_id, agent_version, session_id, attention_avg, attention_min,
			token_used, compact_count, completion_time_seconds, outcome, quality_score,
			feedback, improvement_suggestions, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		perf.ID, perf.AgentID, perf.AgentVersion, perf.SessionID,
		perf.AttentionAvg, perf.AttentionMin, perf.TokenUsed, perf.CompactCount,
		perf.CompletionTimeSeconds, perf.Outcome, perf.QualityScore,
		perf.Feedback, string(suggestionsJSON), perf.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("성능 기록 실패: %w", err)
	}

	// 버전 사용 횟수 및 통계 업데이트
	_, err = m.db.Exec(`
		UPDATE agent_versions
		SET usage_count = usage_count + 1,
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
	`, perf.AgentID, perf.AgentVersion, perf.AgentID, perf.AgentVersion,
		perf.AgentID, perf.AgentVersion, perf.AgentID, perf.AgentVersion)
	
	return err
}

// GetPerformanceHistory retrieves performance history for an agent
func (m *Manager) GetPerformanceHistory(agentID string, limit int) ([]*AgentPerformance, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := m.db.Query(`
		SELECT id, agent_id, agent_version, session_id, attention_avg, attention_min,
		       token_used, compact_count, completion_time_seconds, outcome, quality_score,
		       feedback, improvement_suggestions, created_at
		FROM agent_performance
		WHERE agent_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("성능 히스토리 조회 실패: %w", err)
	}
	defer rows.Close()

	var perfs []*AgentPerformance
	for rows.Next() {
		var p AgentPerformance
		var feedback, suggestionsJSON sql.NullString

		err := rows.Scan(
			&p.ID, &p.AgentID, &p.AgentVersion, &p.SessionID,
			&p.AttentionAvg, &p.AttentionMin, &p.TokenUsed, &p.CompactCount,
			&p.CompletionTimeSeconds, &p.Outcome, &p.QualityScore,
			&feedback, &suggestionsJSON, &p.CreatedAt,
		)
		if err != nil {
			continue
		}

		if feedback.Valid {
			p.Feedback = feedback.String
		}
		if suggestionsJSON.Valid {
			json.Unmarshal([]byte(suggestionsJSON.String), &p.ImprovementSuggestions)
		}

		perfs = append(perfs, &p)
	}

	return perfs, nil
}

// CompareVersions returns comparison statistics between two versions
func (m *Manager) CompareVersions(agentID string, v1, v2 int) (map[string]interface{}, error) {
	ver1, err := m.GetVersion(agentID, v1)
	if err != nil || ver1 == nil {
		return nil, fmt.Errorf("버전 %d 조회 실패", v1)
	}
	ver2, err := m.GetVersion(agentID, v2)
	if err != nil || ver2 == nil {
		return nil, fmt.Errorf("버전 %d 조회 실패", v2)
	}

	return map[string]interface{}{
		"version_1": map[string]interface{}{
			"version":          ver1.Version,
			"usage_count":      ver1.UsageCount,
			"avg_attention":    ver1.AvgAttentionScore,
			"avg_completion":   ver1.AvgCompletionRate,
			"avg_efficiency":   ver1.AvgTokenEfficiency,
			"status":           ver1.Status,
		},
		"version_2": map[string]interface{}{
			"version":          ver2.Version,
			"usage_count":      ver2.UsageCount,
			"avg_attention":    ver2.AvgAttentionScore,
			"avg_completion":   ver2.AvgCompletionRate,
			"avg_efficiency":   ver2.AvgTokenEfficiency,
			"status":           ver2.Status,
		},
		"comparison": map[string]interface{}{
			"attention_diff":   ver2.AvgAttentionScore - ver1.AvgAttentionScore,
			"completion_diff":  ver2.AvgCompletionRate - ver1.AvgCompletionRate,
			"efficiency_diff":  ver2.AvgTokenEfficiency - ver1.AvgTokenEfficiency,
			"recommended":      recommendVersion(ver1, ver2),
		},
	}, nil
}

// recommendVersion recommends which version is better based on metrics
func recommendVersion(v1, v2 *AgentVersion) int {
	score1 := v1.AvgAttentionScore*0.3 + v1.AvgCompletionRate*0.5 + v1.AvgTokenEfficiency*0.2
	score2 := v2.AvgAttentionScore*0.3 + v2.AvgCompletionRate*0.5 + v2.AvgTokenEfficiency*0.2
	
	if score2 > score1 {
		return v2.Version
	}
	return v1.Version
}

// hashContent creates a SHA256 hash of content
func hashContent(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
