package sync

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/env"
)

// Exporter handles data export
type Exporter struct {
	db      *db.DB
	envSvc  *env.Service
}

// NewExporter creates a new exporter
func NewExporter(database *db.DB, envSvc *env.Service) *Exporter {
	return &Exporter{
		db:     database,
		envSvc: envSvc,
	}
}

// ExportAll exports all syncable data
func (e *Exporter) ExportAll() (*SyncData, error) {
	data := &SyncData{}

	// Get current environment info
	currentEnv, err := e.envSvc.Current()
	if err != nil {
		// Use defaults if no environment configured
		hostname, _ := os.Hostname()
		data.Manifest = SyncManifest{
			Version:     1,
			ExportedAt:  time.Now(),
			ExportedBy:  hostname,
			ExportedEnv: "unknown",
		}
	} else {
		data.Manifest = SyncManifest{
			Version:     1,
			ExportedAt:  time.Now(),
			ExportedBy:  currentEnv.Name,
			ExportedEnv: currentEnv.ID,
		}
	}

	// Export each data type
	var exportErr error

	data.Ports, exportErr = e.ExportPorts()
	if exportErr != nil {
		return nil, fmt.Errorf("ports export 실패: %w", exportErr)
	}

	data.Sessions, exportErr = e.ExportSessions()
	if exportErr != nil {
		return nil, fmt.Errorf("sessions export 실패: %w", exportErr)
	}

	data.Escalations, exportErr = e.ExportEscalations()
	if exportErr != nil {
		return nil, fmt.Errorf("escalations export 실패: %w", exportErr)
	}

	data.Pipelines, exportErr = e.ExportPipelines()
	if exportErr != nil {
		return nil, fmt.Errorf("pipelines export 실패: %w", exportErr)
	}

	data.Projects, exportErr = e.ExportProjects()
	if exportErr != nil {
		return nil, fmt.Errorf("projects export 실패: %w", exportErr)
	}

	// Update stats
	data.Manifest.Stats = SyncStats{
		PortsCount:       len(data.Ports),
		SessionsCount:    len(data.Sessions),
		EscalationsCount: len(data.Escalations),
		PipelinesCount:   len(data.Pipelines),
		ProjectsCount:    len(data.Projects),
	}

	return data, nil
}

// ExportPorts exports all ports
func (e *Exporter) ExportPorts() ([]PortData, error) {
	rows, err := e.db.Query(`
		SELECT id, title, status, file_path, created_at, started_at, completed_at,
		       COALESCE(input_tokens, 0), COALESCE(output_tokens, 0),
		       COALESCE(cost_usd, 0), COALESCE(duration_secs, 0), agent_id
		FROM ports
		ORDER BY created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []PortData
	for rows.Next() {
		var p PortData
		var title, filePath, agentID sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&p.ID, &title, &p.Status, &filePath, &p.CreatedAt,
			&startedAt, &completedAt, &p.InputTokens, &p.OutputTokens,
			&p.CostUSD, &p.DurationSecs, &agentID,
		); err != nil {
			return nil, err
		}

		if title.Valid {
			p.Title = title.String
		}
		if filePath.Valid {
			p.FilePath = filePath.String
		}
		if agentID.Valid {
			p.AgentID = agentID.String
		}
		if startedAt.Valid {
			p.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			p.CompletedAt = &completedAt.Time
		}

		// Get dependencies
		deps, _ := e.getPortDependencies(p.ID)
		p.Dependencies = deps

		ports = append(ports, p)
	}

	return ports, nil
}

// getPortDependencies returns dependencies for a port
func (e *Exporter) getPortDependencies(portID string) ([]string, error) {
	rows, err := e.db.Query(`
		SELECT depends_on FROM port_dependencies WHERE port_id = ?
	`, portID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []string
	for rows.Next() {
		var dep string
		if err := rows.Scan(&dep); err != nil {
			return nil, err
		}
		deps = append(deps, dep)
	}
	return deps, nil
}

// ExportSessions exports sessions (excluding local-only fields)
func (e *Exporter) ExportSessions() ([]SessionData, error) {
	rows, err := e.db.Query(`
		SELECT id, port_id, title, status,
		       COALESCE(session_type, 'single'), parent_session,
		       started_at, ended_at,
		       COALESCE(input_tokens, 0), COALESCE(output_tokens, 0),
		       COALESCE(cache_read_tokens, 0), COALESCE(cache_create_tokens, 0),
		       COALESCE(cost_usd, 0), COALESCE(compact_count, 0),
		       project_root, project_name, created_env, last_env
		FROM sessions
		ORDER BY started_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionData
	for rows.Next() {
		var s SessionData
		var portID, title, parentSession sql.NullString
		var projectRoot, projectName, createdEnv, lastEnv sql.NullString
		var endedAt sql.NullTime

		if err := rows.Scan(
			&s.ID, &portID, &title, &s.Status,
			&s.SessionType, &parentSession,
			&s.StartedAt, &endedAt,
			&s.InputTokens, &s.OutputTokens,
			&s.CacheReadTokens, &s.CacheCreateTokens,
			&s.CostUSD, &s.CompactCount,
			&projectRoot, &projectName, &createdEnv, &lastEnv,
		); err != nil {
			return nil, err
		}

		if portID.Valid {
			s.PortID = portID.String
		}
		if title.Valid {
			s.Title = title.String
		}
		if parentSession.Valid {
			s.ParentSession = parentSession.String
		}
		if endedAt.Valid {
			s.EndedAt = &endedAt.Time
		}
		if projectRoot.Valid {
			s.ProjectRoot = projectRoot.String
		}
		if projectName.Valid {
			s.ProjectName = projectName.String
		}
		if createdEnv.Valid {
			s.CreatedEnv = createdEnv.String
		}
		if lastEnv.Valid {
			s.LastEnv = lastEnv.String
		}

		sessions = append(sessions, s)
	}

	return sessions, nil
}

// ExportEscalations exports all escalations
func (e *Exporter) ExportEscalations() ([]Escalation, error) {
	rows, err := e.db.Query(`
		SELECT id, from_session, from_port, issue, status, created_at, resolved_at
		FROM escalations
		ORDER BY created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var escalations []Escalation
	for rows.Next() {
		var esc Escalation
		var fromSession, fromPort sql.NullString
		var resolvedAt sql.NullTime

		if err := rows.Scan(
			&esc.ID, &fromSession, &fromPort, &esc.Issue, &esc.Status,
			&esc.CreatedAt, &resolvedAt,
		); err != nil {
			return nil, err
		}

		if fromSession.Valid {
			esc.FromSession = fromSession.String
		}
		if fromPort.Valid {
			esc.FromPort = fromPort.String
		}
		if resolvedAt.Valid {
			esc.ResolvedAt = &resolvedAt.Time
		}

		escalations = append(escalations, esc)
	}

	return escalations, nil
}

// ExportPipelines exports all pipelines with their ports
func (e *Exporter) ExportPipelines() ([]PipelineData, error) {
	rows, err := e.db.Query(`
		SELECT id, name, session_id, status, created_at, started_at, completed_at
		FROM pipelines
		ORDER BY created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pipelines []PipelineData
	for rows.Next() {
		var p PipelineData
		var sessionID sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&p.ID, &p.Name, &sessionID, &p.Status,
			&p.CreatedAt, &startedAt, &completedAt,
		); err != nil {
			return nil, err
		}

		if sessionID.Valid {
			p.SessionID = sessionID.String
		}
		if startedAt.Valid {
			p.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			p.CompletedAt = &completedAt.Time
		}

		// Get pipeline ports
		p.Ports, _ = e.getPipelinePorts(p.ID)

		pipelines = append(pipelines, p)
	}

	return pipelines, nil
}

// getPipelinePorts returns ports for a pipeline
func (e *Exporter) getPipelinePorts(pipelineID string) ([]PipelinePortData, error) {
	rows, err := e.db.Query(`
		SELECT port_id, group_order, status
		FROM pipeline_ports
		WHERE pipeline_id = ?
		ORDER BY group_order
	`, pipelineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ports []PipelinePortData
	for rows.Next() {
		var p PipelinePortData
		if err := rows.Scan(&p.PortID, &p.GroupOrder, &p.Status); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}
	return ports, nil
}

// ExportProjects exports all projects (with logical paths)
func (e *Exporter) ExportProjects() ([]ProjectData, error) {
	rows, err := e.db.Query(`
		SELECT COALESCE(logical_root, root) as root, name, description,
		       last_active, session_count, total_tokens, total_cost, created_at
		FROM projects
		ORDER BY last_active DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []ProjectData
	for rows.Next() {
		var p ProjectData
		var name, description sql.NullString
		var lastActive sql.NullTime

		if err := rows.Scan(
			&p.Root, &name, &description,
			&lastActive, &p.SessionCount, &p.TotalTokens, &p.TotalCost, &p.CreatedAt,
		); err != nil {
			return nil, err
		}

		if name.Valid {
			p.Name = name.String
		}
		if description.Valid {
			p.Description = description.String
		}
		if lastActive.Valid {
			p.LastActive = lastActive.Time
		}

		projects = append(projects, p)
	}

	return projects, nil
}

// ExportToYAML exports data to YAML format
func (e *Exporter) ExportToYAML() ([]byte, error) {
	data, err := e.ExportAll()
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(data)
}

// ExportToFile exports data to a file
func (e *Exporter) ExportToFile(path string) error {
	data, err := e.ExportToYAML()
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
