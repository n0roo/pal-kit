package sync

import (
	"time"
)

// SyncManifest represents the sync state file
type SyncManifest struct {
	Version     int       `yaml:"version" json:"version"`
	ExportedAt  time.Time `yaml:"exported_at" json:"exported_at"`
	ExportedBy  string    `yaml:"exported_by" json:"exported_by"`   // environment name
	ExportedEnv string    `yaml:"exported_env" json:"exported_env"` // environment ID
	Checksum    string    `yaml:"checksum,omitempty" json:"checksum,omitempty"`
	Stats       SyncStats `yaml:"stats" json:"stats"`
}

// SyncStats contains export statistics
type SyncStats struct {
	PortsCount       int `yaml:"ports" json:"ports"`
	SessionsCount    int `yaml:"sessions" json:"sessions"`
	EscalationsCount int `yaml:"escalations" json:"escalations"`
	PipelinesCount   int `yaml:"pipelines" json:"pipelines"`
	ProjectsCount    int `yaml:"projects" json:"projects"`
}

// SyncData represents the complete sync payload
type SyncData struct {
	Manifest    SyncManifest   `yaml:"manifest" json:"manifest"`
	Ports       []PortData     `yaml:"ports" json:"ports"`
	Sessions    []SessionData  `yaml:"sessions" json:"sessions"`
	Escalations []Escalation   `yaml:"escalations" json:"escalations"`
	Pipelines   []PipelineData `yaml:"pipelines" json:"pipelines"`
	Projects    []ProjectData  `yaml:"projects" json:"projects"`
}

// PortData represents a port for sync
type PortData struct {
	ID          string     `yaml:"id" json:"id"`
	Title       string     `yaml:"title,omitempty" json:"title,omitempty"`
	Status      string     `yaml:"status" json:"status"`
	FilePath    string     `yaml:"file_path,omitempty" json:"file_path,omitempty"`
	CreatedAt   time.Time  `yaml:"created_at" json:"created_at"`
	StartedAt   *time.Time `yaml:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt *time.Time `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
	InputTokens int64      `yaml:"input_tokens" json:"input_tokens"`
	OutputTokens int64     `yaml:"output_tokens" json:"output_tokens"`
	CostUSD     float64    `yaml:"cost_usd" json:"cost_usd"`
	DurationSecs int64     `yaml:"duration_secs" json:"duration_secs"`
	AgentID     string     `yaml:"agent_id,omitempty" json:"agent_id,omitempty"`
	// Dependencies
	Dependencies []string `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

// SessionData represents a session for sync (excludes local-only fields)
type SessionData struct {
	ID                string     `yaml:"id" json:"id"`
	PortID            string     `yaml:"port_id,omitempty" json:"port_id,omitempty"`
	Title             string     `yaml:"title,omitempty" json:"title,omitempty"`
	Status            string     `yaml:"status" json:"status"`
	SessionType       string     `yaml:"session_type" json:"session_type"`
	ParentSession     string     `yaml:"parent_session,omitempty" json:"parent_session,omitempty"`
	StartedAt         time.Time  `yaml:"started_at" json:"started_at"`
	EndedAt           *time.Time `yaml:"ended_at,omitempty" json:"ended_at,omitempty"`
	InputTokens       int64      `yaml:"input_tokens" json:"input_tokens"`
	OutputTokens      int64      `yaml:"output_tokens" json:"output_tokens"`
	CacheReadTokens   int64      `yaml:"cache_read_tokens" json:"cache_read_tokens"`
	CacheCreateTokens int64      `yaml:"cache_create_tokens" json:"cache_create_tokens"`
	CostUSD           float64    `yaml:"cost_usd" json:"cost_usd"`
	CompactCount      int        `yaml:"compact_count" json:"compact_count"`
	// Logical paths (environment-independent)
	ProjectRoot string `yaml:"project_root,omitempty" json:"project_root,omitempty"`
	ProjectName string `yaml:"project_name,omitempty" json:"project_name,omitempty"`
	// Environment tracking
	CreatedEnv string `yaml:"created_env,omitempty" json:"created_env,omitempty"`
	LastEnv    string `yaml:"last_env,omitempty" json:"last_env,omitempty"`
}

// Escalation represents an escalation for sync
type Escalation struct {
	ID          int64      `yaml:"id" json:"id"`
	FromSession string     `yaml:"from_session,omitempty" json:"from_session,omitempty"`
	FromPort    string     `yaml:"from_port,omitempty" json:"from_port,omitempty"`
	Issue       string     `yaml:"issue" json:"issue"`
	Status      string     `yaml:"status" json:"status"`
	CreatedAt   time.Time  `yaml:"created_at" json:"created_at"`
	ResolvedAt  *time.Time `yaml:"resolved_at,omitempty" json:"resolved_at,omitempty"`
}

// PipelineData represents a pipeline for sync
type PipelineData struct {
	ID          string     `yaml:"id" json:"id"`
	Name        string     `yaml:"name" json:"name"`
	SessionID   string     `yaml:"session_id,omitempty" json:"session_id,omitempty"`
	Status      string     `yaml:"status" json:"status"`
	CreatedAt   time.Time  `yaml:"created_at" json:"created_at"`
	StartedAt   *time.Time `yaml:"started_at,omitempty" json:"started_at,omitempty"`
	CompletedAt *time.Time `yaml:"completed_at,omitempty" json:"completed_at,omitempty"`
	// Pipeline ports
	Ports []PipelinePortData `yaml:"ports,omitempty" json:"ports,omitempty"`
}

// PipelinePortData represents a port in a pipeline
type PipelinePortData struct {
	PortID     string `yaml:"port_id" json:"port_id"`
	GroupOrder int    `yaml:"group_order" json:"group_order"`
	Status     string `yaml:"status" json:"status"`
}

// ProjectData represents a project for sync
type ProjectData struct {
	Root         string    `yaml:"root" json:"root"`           // Logical path
	Name         string    `yaml:"name,omitempty" json:"name,omitempty"`
	Description  string    `yaml:"description,omitempty" json:"description,omitempty"`
	LastActive   time.Time `yaml:"last_active,omitempty" json:"last_active,omitempty"`
	SessionCount int       `yaml:"session_count" json:"session_count"`
	TotalTokens  int64     `yaml:"total_tokens" json:"total_tokens"`
	TotalCost    float64   `yaml:"total_cost" json:"total_cost"`
	CreatedAt    time.Time `yaml:"created_at" json:"created_at"`
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	Success     bool            `json:"success"`
	Imported    ImportStats     `json:"imported"`
	Skipped     ImportStats     `json:"skipped"`
	Conflicts   []ConflictItem  `json:"conflicts,omitempty"`
	Errors      []string        `json:"errors,omitempty"`
}

// ImportStats contains import statistics
type ImportStats struct {
	Ports       int `json:"ports"`
	Sessions    int `json:"sessions"`
	Escalations int `json:"escalations"`
	Pipelines   int `json:"pipelines"`
	Projects    int `json:"projects"`
}

// ConflictItem represents a sync conflict
type ConflictItem struct {
	Type       string      `json:"type"` // port, session, etc.
	ID         string      `json:"id"`
	LocalData  interface{} `json:"local"`
	RemoteData interface{} `json:"remote"`
	Resolution string      `json:"resolution,omitempty"` // keep_local, keep_remote, merged
}

// MergeStrategy defines how to handle conflicts
type MergeStrategy string

const (
	MergeStrategyLastWriteWins MergeStrategy = "last_write_wins"
	MergeStrategyKeepLocal     MergeStrategy = "keep_local"
	MergeStrategyKeepRemote    MergeStrategy = "keep_remote"
	MergeStrategyManual        MergeStrategy = "manual"
)

// ImportOptions configures import behavior
type ImportOptions struct {
	Strategy      MergeStrategy `json:"strategy"`
	DryRun        bool          `json:"dry_run"`
	SkipConflicts bool          `json:"skip_conflicts"`
}
