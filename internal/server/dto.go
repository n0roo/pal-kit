package server

import (
	"fmt"
	"time"

	"github.com/n0roo/pal-kit/internal/escalation"
	"github.com/n0roo/pal-kit/internal/lock"
	"github.com/n0roo/pal-kit/internal/pipeline"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/session"
)

// Session DTO for JSON response
type SessionDTO struct {
	ID          string  `json:"id"`
	PortID      string  `json:"port_id,omitempty"`
	Title       string  `json:"title,omitempty"`
	Status      string  `json:"status"`
	SessionType string  `json:"session_type,omitempty"`
	Parent      string  `json:"parent,omitempty"`
	StartedAt   string  `json:"started_at,omitempty"`
	EndedAt     string  `json:"ended_at,omitempty"`
}

func toSessionDTO(s session.Session) SessionDTO {
	dto := SessionDTO{
		ID:     s.ID,
		Status: s.Status,
	}
	if s.PortID.Valid {
		dto.PortID = s.PortID.String
	}
	if s.Title.Valid {
		dto.Title = s.Title.String
	}
	if s.SessionType != "" {
		dto.SessionType = s.SessionType
	}
	if s.ParentSession.Valid {
		dto.Parent = s.ParentSession.String
	}
	dto.StartedAt = s.StartedAt.Format(time.RFC3339)
	if s.EndedAt.Valid {
		dto.EndedAt = s.EndedAt.Time.Format(time.RFC3339)
	}
	return dto
}

func toSessionDTOs(sessions []session.Session) []SessionDTO {
	result := make([]SessionDTO, len(sessions))
	for i, s := range sessions {
		result[i] = toSessionDTO(s)
	}
	return result
}

// SessionDetailDTO includes duration and children count
type SessionDetailDTO struct {
	ID            string  `json:"id"`
	PortID        string  `json:"port_id,omitempty"`
	Title         string  `json:"title,omitempty"`
	Status        string  `json:"status"`
	SessionType   string  `json:"session_type,omitempty"`
	Parent        string  `json:"parent,omitempty"`
	StartedAt     string  `json:"started_at,omitempty"`
	EndedAt       string  `json:"ended_at,omitempty"`
	DurationSecs  int64   `json:"duration_secs"`
	DurationStr   string  `json:"duration_str"`
	ChildrenCount int     `json:"children_count"`
	InputTokens   int64   `json:"input_tokens"`
	OutputTokens  int64   `json:"output_tokens"`
	CacheRead     int64   `json:"cache_read_tokens"`
	CacheCreate   int64   `json:"cache_create_tokens"`
	CostUSD       float64 `json:"cost_usd"`
	CompactCount  int     `json:"compact_count"`
}

func toSessionDetailDTO(d session.SessionDetail) SessionDetailDTO {
	dto := SessionDetailDTO{
		ID:            d.ID,
		Status:        d.Status,
		SessionType:   d.SessionType,
		DurationSecs:  d.DurationSecs,
		DurationStr:   d.DurationStr,
		ChildrenCount: d.ChildrenCount,
		InputTokens:   d.InputTokens,
		OutputTokens:  d.OutputTokens,
		CacheRead:     d.CacheReadTokens,
		CacheCreate:   d.CacheCreateTokens,
		CostUSD:       d.CostUSD,
		CompactCount:  d.CompactCount,
	}
	if d.PortID.Valid {
		dto.PortID = d.PortID.String
	}
	if d.Title.Valid {
		dto.Title = d.Title.String
	}
	if d.ParentSession.Valid {
		dto.Parent = d.ParentSession.String
	}
	dto.StartedAt = d.StartedAt.Format(time.RFC3339)
	if d.EndedAt.Valid {
		dto.EndedAt = d.EndedAt.Time.Format(time.RFC3339)
	}
	return dto
}

func toSessionDetailDTOs(details []session.SessionDetail) []SessionDetailDTO {
	result := make([]SessionDetailDTO, len(details))
	for i, d := range details {
		result[i] = toSessionDetailDTO(d)
	}
	return result
}

// SessionEventDTO for timeline view
type SessionEventDTO struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	EventType string `json:"event_type"`
	EventData string `json:"event_data,omitempty"`
	CreatedAt string `json:"created_at"`
}

func toSessionEventDTO(e session.SessionEvent) SessionEventDTO {
	return SessionEventDTO{
		ID:        e.ID,
		SessionID: e.SessionID,
		EventType: e.EventType,
		EventData: e.EventData,
		CreatedAt: e.CreatedAt.Format(time.RFC3339),
	}
}

func toSessionEventDTOs(events []session.SessionEvent) []SessionEventDTO {
	if events == nil {
		return []SessionEventDTO{}
	}
	result := make([]SessionEventDTO, len(events))
	for i, e := range events {
		result[i] = toSessionEventDTO(e)
	}
	return result
}

// Port DTO for JSON response
type PortDTO struct {
	ID           string  `json:"id"`
	Title        string  `json:"title,omitempty"`
	Status       string  `json:"status"`
	SessionID    string  `json:"session_id,omitempty"`
	FilePath     string  `json:"file_path,omitempty"`
	CreatedAt    string  `json:"created_at,omitempty"`
	StartedAt    string  `json:"started_at,omitempty"`
	CompletedAt  string  `json:"completed_at,omitempty"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	DurationSecs int64   `json:"duration_secs"`
	DurationStr  string  `json:"duration_str,omitempty"`
	AgentID      string  `json:"agent_id,omitempty"`
}

func toPortDTO(p port.Port) PortDTO {
	dto := PortDTO{
		ID:           p.ID,
		Status:       p.Status,
		InputTokens:  p.InputTokens,
		OutputTokens: p.OutputTokens,
		CostUSD:      p.CostUSD,
		DurationSecs: p.DurationSecs,
	}
	if p.Title.Valid {
		dto.Title = p.Title.String
	}
	if p.SessionID.Valid {
		dto.SessionID = p.SessionID.String
	}
	if p.FilePath.Valid {
		dto.FilePath = p.FilePath.String
	}
	if p.AgentID.Valid {
		dto.AgentID = p.AgentID.String
	}
	dto.CreatedAt = p.CreatedAt.Format(time.RFC3339)
	if p.StartedAt.Valid {
		dto.StartedAt = p.StartedAt.Time.Format(time.RFC3339)
	}
	if p.CompletedAt.Valid {
		dto.CompletedAt = p.CompletedAt.Time.Format(time.RFC3339)
	}
	// Calculate duration string
	if dto.DurationSecs > 0 {
		dto.DurationStr = formatDuration(dto.DurationSecs)
	} else if p.StartedAt.Valid && p.CompletedAt.Valid {
		dto.DurationSecs = int64(p.CompletedAt.Time.Sub(p.StartedAt.Time).Seconds())
		dto.DurationStr = formatDuration(dto.DurationSecs)
	}
	return dto
}

func toPortDTOs(ports []port.Port) []PortDTO {
	result := make([]PortDTO, len(ports))
	for i, p := range ports {
		result[i] = toPortDTO(p)
	}
	return result
}

// Pipeline DTO for JSON response
type PipelineDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SessionID   string `json:"session_id,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

func toPipelineDTO(p pipeline.Pipeline) PipelineDTO {
	dto := PipelineDTO{
		ID:     p.ID,
		Name:   p.Name,
		Status: p.Status,
	}
	if p.SessionID.Valid {
		dto.SessionID = p.SessionID.String
	}
	dto.CreatedAt = p.CreatedAt.Format(time.RFC3339)
	if p.StartedAt.Valid {
		dto.StartedAt = p.StartedAt.Time.Format(time.RFC3339)
	}
	if p.CompletedAt.Valid {
		dto.CompletedAt = p.CompletedAt.Time.Format(time.RFC3339)
	}
	return dto
}

func toPipelineDTOs(pipelines []pipeline.Pipeline) []PipelineDTO {
	result := make([]PipelineDTO, len(pipelines))
	for i, p := range pipelines {
		result[i] = toPipelineDTO(p)
	}
	return result
}

// Escalation DTO for JSON response
type EscalationDTO struct {
	ID          int64  `json:"id"`
	FromSession string `json:"from_session,omitempty"`
	FromPort    string `json:"from_port,omitempty"`
	Issue       string `json:"issue"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at,omitempty"`
	ResolvedAt  string `json:"resolved_at,omitempty"`
}

func toEscalationDTO(e escalation.Escalation) EscalationDTO {
	dto := EscalationDTO{
		ID:     e.ID,
		Issue:  e.Issue,
		Status: e.Status,
	}
	if e.FromSession.Valid {
		dto.FromSession = e.FromSession.String
	}
	if e.FromPort.Valid {
		dto.FromPort = e.FromPort.String
	}
	dto.CreatedAt = e.CreatedAt.Format(time.RFC3339)
	if e.ResolvedAt.Valid {
		dto.ResolvedAt = e.ResolvedAt.Time.Format(time.RFC3339)
	}
	return dto
}

func toEscalationDTOs(escalations []escalation.Escalation) []EscalationDTO {
	result := make([]EscalationDTO, len(escalations))
	for i, e := range escalations {
		result[i] = toEscalationDTO(e)
	}
	return result
}

// Lock DTO for JSON response
type LockDTO struct {
	Resource   string `json:"resource"`
	SessionID  string `json:"session_id"`
	AcquiredAt string `json:"acquired_at,omitempty"`
}

func toLockDTO(l lock.Lock) LockDTO {
	return LockDTO{
		Resource:   l.Resource,
		SessionID:  l.SessionID,
		AcquiredAt: l.AcquiredAt.Format(time.RFC3339),
	}
}

func toLockDTOs(locks []lock.Lock) []LockDTO {
	result := make([]LockDTO, len(locks))
	for i, l := range locks {
		result[i] = toLockDTO(l)
	}
	return result
}

// formatDuration formats seconds into human readable string
func formatDuration(secs int64) string {
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm %ds", secs/60, secs%60)
	}
	hours := secs / 3600
	mins := (secs % 3600) / 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// SessionTreeNodeDTO for hierarchical session tree
type SessionTreeNodeDTO struct {
	ID          string               `json:"id"`
	Name        string               `json:"name,omitempty"`
	Agent       string               `json:"agent,omitempty"`
	Status      string               `json:"status"`
	SessionType string               `json:"session_type,omitempty"`
	StartedAt   string               `json:"started_at,omitempty"`
	EndedAt     string               `json:"ended_at,omitempty"`
	Children    []SessionTreeNodeDTO `json:"children,omitempty"`
}

func toSessionTreeNodeDTO(node session.SessionNode) SessionTreeNodeDTO {
	dto := SessionTreeNodeDTO{
		ID:          node.Session.ID,
		Status:      node.Session.Status,
		SessionType: node.Session.SessionType,
		Agent:       node.Session.SessionType, // Use session type as agent identifier
	}
	if node.Session.Title.Valid {
		dto.Name = node.Session.Title.String
	}
	dto.StartedAt = node.Session.StartedAt.Format(time.RFC3339)
	if node.Session.EndedAt.Valid {
		dto.EndedAt = node.Session.EndedAt.Time.Format(time.RFC3339)
	}
	if len(node.Children) > 0 {
		dto.Children = make([]SessionTreeNodeDTO, len(node.Children))
		for i, child := range node.Children {
			dto.Children[i] = toSessionTreeNodeDTO(child)
		}
	}
	return dto
}

// PortFlowDTO for port dependency visualization
type PortFlowDTO struct {
	Ports        []PortNodeDTO        `json:"ports"`
	Dependencies []PortDependencyDTO  `json:"dependencies"`
}

// PortNodeDTO represents a port in the flow diagram
type PortNodeDTO struct {
	ID       string `json:"id"`
	Title    string `json:"title,omitempty"`
	Status   string `json:"status"`
	Agent    string `json:"agent,omitempty"`
	Progress int    `json:"progress,omitempty"`
}

// PortDependencyDTO represents a dependency between ports
type PortDependencyDTO struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PortProgressDTO for port progress dashboard
type PortProgressDTO struct {
	Completed  []PortNodeDTO `json:"completed"`
	InProgress []PortNodeDTO `json:"in_progress"`
	Pending    []PortNodeDTO `json:"pending"`
}

// WorkflowTimelineDTO for session workflow visualization
type WorkflowTimelineDTO struct {
	SessionID string              `json:"session_id"`
	Phases    []WorkflowPhaseDTO  `json:"phases"`
}

// WorkflowPhaseDTO represents a phase in the workflow
type WorkflowPhaseDTO struct {
	Name      string              `json:"name"`
	Status    string              `json:"status"` // complete, active, pending
	Agent     string              `json:"agent,omitempty"`
	SubPhases []WorkflowPhaseDTO  `json:"sub_phases,omitempty"`
	StartedAt string              `json:"started_at,omitempty"`
	EndedAt   string              `json:"ended_at,omitempty"`
	Detail    string              `json:"detail,omitempty"`
}
