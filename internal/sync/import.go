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

// Importer handles data import
type Importer struct {
	db      *db.DB
	envSvc  *env.Service
	options ImportOptions
}

// NewImporter creates a new importer
func NewImporter(database *db.DB, envSvc *env.Service, options ImportOptions) *Importer {
	return &Importer{
		db:      database,
		envSvc:  envSvc,
		options: options,
	}
}

// ImportFromFile imports data from a YAML file
func (i *Importer) ImportFromFile(path string) (*ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}

	return i.ImportFromYAML(data)
}

// ImportFromYAML imports data from YAML bytes
func (i *Importer) ImportFromYAML(data []byte) (*ImportResult, error) {
	var syncData SyncData
	if err := yaml.Unmarshal(data, &syncData); err != nil {
		return nil, fmt.Errorf("YAML 파싱 실패: %w", err)
	}

	return i.Import(&syncData)
}

// Import imports sync data
func (i *Importer) Import(data *SyncData) (*ImportResult, error) {
	result := &ImportResult{
		Success: true,
	}

	// Import in order: ports -> sessions -> escalations -> pipelines -> projects
	if err := i.importPorts(data.Ports, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("ports: %v", err))
	}

	if err := i.importSessions(data.Sessions, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("sessions: %v", err))
	}

	if err := i.importEscalations(data.Escalations, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("escalations: %v", err))
	}

	if err := i.importPipelines(data.Pipelines, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("pipelines: %v", err))
	}

	if err := i.importProjects(data.Projects, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("projects: %v", err))
	}

	if len(result.Errors) > 0 {
		result.Success = false
	}

	return result, nil
}

// importPorts imports ports
func (i *Importer) importPorts(ports []PortData, result *ImportResult) error {
	for _, p := range ports {
		exists, existingData := i.portExists(p.ID)

		if exists {
			if i.options.SkipConflicts {
				result.Skipped.Ports++
				continue
			}

			// Check for conflict
			conflict := i.detectPortConflict(p, existingData)
			if conflict != nil {
				result.Conflicts = append(result.Conflicts, *conflict)
				if i.options.Strategy == MergeStrategyManual {
					result.Skipped.Ports++
					continue
				}
			}

			// Apply merge strategy
			if i.options.Strategy == MergeStrategyKeepLocal {
				result.Skipped.Ports++
				continue
			}

			// Update existing (last_write_wins or keep_remote)
			if !i.options.DryRun {
				if err := i.updatePort(p); err != nil {
					return err
				}
			}
			result.Imported.Ports++
		} else {
			// Insert new
			if !i.options.DryRun {
				if err := i.insertPort(p); err != nil {
					return err
				}
			}
			result.Imported.Ports++
		}
	}
	return nil
}

// portExists checks if a port exists and returns its data
func (i *Importer) portExists(id string) (bool, *PortData) {
	var p PortData
	var title, filePath, agentID sql.NullString
	var startedAt, completedAt sql.NullTime

	err := i.db.QueryRow(`
		SELECT id, title, status, file_path, created_at, started_at, completed_at,
		       COALESCE(input_tokens, 0), COALESCE(output_tokens, 0),
		       COALESCE(cost_usd, 0), COALESCE(duration_secs, 0), agent_id
		FROM ports WHERE id = ?
	`, id).Scan(
		&p.ID, &title, &p.Status, &filePath, &p.CreatedAt,
		&startedAt, &completedAt, &p.InputTokens, &p.OutputTokens,
		&p.CostUSD, &p.DurationSecs, &agentID,
	)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, nil
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

	return true, &p
}

// detectPortConflict detects if there's a meaningful conflict
func (i *Importer) detectPortConflict(remote PortData, local *PortData) *ConflictItem {
	if local == nil {
		return nil
	}

	// Check if data differs
	if remote.Status != local.Status ||
		remote.Title != local.Title ||
		remote.InputTokens != local.InputTokens {
		return &ConflictItem{
			Type:       "port",
			ID:         remote.ID,
			LocalData:  local,
			RemoteData: remote,
		}
	}

	return nil
}

// insertPort inserts a new port
func (i *Importer) insertPort(p PortData) error {
	_, err := i.db.Exec(`
		INSERT INTO ports (id, title, status, file_path, created_at, started_at, completed_at,
		                   input_tokens, output_tokens, cost_usd, duration_secs, agent_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, nullString(p.Title), p.Status, nullString(p.FilePath), p.CreatedAt,
		nullTime(p.StartedAt), nullTime(p.CompletedAt),
		p.InputTokens, p.OutputTokens, p.CostUSD, p.DurationSecs, nullString(p.AgentID))

	if err != nil {
		return err
	}

	// Insert dependencies
	for _, dep := range p.Dependencies {
		i.db.Exec(`
			INSERT OR IGNORE INTO port_dependencies (port_id, depends_on) VALUES (?, ?)
		`, p.ID, dep)
	}

	return nil
}

// updatePort updates an existing port
func (i *Importer) updatePort(p PortData) error {
	_, err := i.db.Exec(`
		UPDATE ports SET title = ?, status = ?, file_path = ?, started_at = ?, completed_at = ?,
		                 input_tokens = ?, output_tokens = ?, cost_usd = ?, duration_secs = ?, agent_id = ?
		WHERE id = ?
	`, nullString(p.Title), p.Status, nullString(p.FilePath),
		nullTime(p.StartedAt), nullTime(p.CompletedAt),
		p.InputTokens, p.OutputTokens, p.CostUSD, p.DurationSecs, nullString(p.AgentID), p.ID)

	return err
}

// importSessions imports sessions
func (i *Importer) importSessions(sessions []SessionData, result *ImportResult) error {
	// Get current environment
	currentEnv, _ := i.envSvc.Current()
	currentEnvID := ""
	if currentEnv != nil {
		currentEnvID = currentEnv.ID
	}

	for _, s := range sessions {
		exists := i.sessionExists(s.ID)

		if exists {
			if i.options.SkipConflicts {
				result.Skipped.Sessions++
				continue
			}

			if i.options.Strategy == MergeStrategyKeepLocal {
				result.Skipped.Sessions++
				continue
			}

			// Update existing
			if !i.options.DryRun {
				if err := i.updateSession(s, currentEnvID); err != nil {
					return err
				}
			}
			result.Imported.Sessions++
		} else {
			// Insert new
			if !i.options.DryRun {
				if err := i.insertSession(s, currentEnvID); err != nil {
					return err
				}
			}
			result.Imported.Sessions++
		}
	}
	return nil
}

// sessionExists checks if a session exists
func (i *Importer) sessionExists(id string) bool {
	var exists int
	err := i.db.QueryRow(`SELECT 1 FROM sessions WHERE id = ?`, id).Scan(&exists)
	return err == nil
}

// insertSession inserts a new session
func (i *Importer) insertSession(s SessionData, currentEnvID string) error {
	// Set created_env if not set
	createdEnv := s.CreatedEnv
	if createdEnv == "" {
		createdEnv = currentEnvID
	}

	_, err := i.db.Exec(`
		INSERT INTO sessions (id, port_id, title, status, session_type, parent_session,
		                      started_at, ended_at, input_tokens, output_tokens,
		                      cache_read_tokens, cache_create_tokens, cost_usd, compact_count,
		                      project_root, project_name, created_env, last_env)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, nullString(s.PortID), nullString(s.Title), s.Status, s.SessionType,
		nullString(s.ParentSession), s.StartedAt, nullTime(s.EndedAt),
		s.InputTokens, s.OutputTokens, s.CacheReadTokens, s.CacheCreateTokens,
		s.CostUSD, s.CompactCount, nullString(s.ProjectRoot), nullString(s.ProjectName),
		nullString(createdEnv), nullString(currentEnvID))

	return err
}

// updateSession updates an existing session
func (i *Importer) updateSession(s SessionData, currentEnvID string) error {
	_, err := i.db.Exec(`
		UPDATE sessions SET title = ?, status = ?, ended_at = ?,
		                    input_tokens = ?, output_tokens = ?,
		                    cache_read_tokens = ?, cache_create_tokens = ?,
		                    cost_usd = ?, compact_count = ?, last_env = ?
		WHERE id = ?
	`, nullString(s.Title), s.Status, nullTime(s.EndedAt),
		s.InputTokens, s.OutputTokens, s.CacheReadTokens, s.CacheCreateTokens,
		s.CostUSD, s.CompactCount, nullString(currentEnvID), s.ID)

	return err
}

// importEscalations imports escalations
func (i *Importer) importEscalations(escalations []Escalation, result *ImportResult) error {
	for _, e := range escalations {
		exists := i.escalationExists(e.ID)

		if exists {
			if i.options.SkipConflicts || i.options.Strategy == MergeStrategyKeepLocal {
				result.Skipped.Escalations++
				continue
			}

			if !i.options.DryRun {
				if err := i.updateEscalation(e); err != nil {
					return err
				}
			}
			result.Imported.Escalations++
		} else {
			if !i.options.DryRun {
				if err := i.insertEscalation(e); err != nil {
					return err
				}
			}
			result.Imported.Escalations++
		}
	}
	return nil
}

// escalationExists checks if an escalation exists
func (i *Importer) escalationExists(id int64) bool {
	var exists int
	err := i.db.QueryRow(`SELECT 1 FROM escalations WHERE id = ?`, id).Scan(&exists)
	return err == nil
}

// insertEscalation inserts a new escalation
func (i *Importer) insertEscalation(e Escalation) error {
	_, err := i.db.Exec(`
		INSERT INTO escalations (id, from_session, from_port, issue, status, created_at, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, e.ID, nullString(e.FromSession), nullString(e.FromPort),
		e.Issue, e.Status, e.CreatedAt, nullTime(e.ResolvedAt))
	return err
}

// updateEscalation updates an existing escalation
func (i *Importer) updateEscalation(e Escalation) error {
	_, err := i.db.Exec(`
		UPDATE escalations SET issue = ?, status = ?, resolved_at = ?
		WHERE id = ?
	`, e.Issue, e.Status, nullTime(e.ResolvedAt), e.ID)
	return err
}

// importPipelines imports pipelines
func (i *Importer) importPipelines(pipelines []PipelineData, result *ImportResult) error {
	for _, p := range pipelines {
		exists := i.pipelineExists(p.ID)

		if exists {
			if i.options.SkipConflicts || i.options.Strategy == MergeStrategyKeepLocal {
				result.Skipped.Pipelines++
				continue
			}

			if !i.options.DryRun {
				if err := i.updatePipeline(p); err != nil {
					return err
				}
			}
			result.Imported.Pipelines++
		} else {
			if !i.options.DryRun {
				if err := i.insertPipeline(p); err != nil {
					return err
				}
			}
			result.Imported.Pipelines++
		}
	}
	return nil
}

// pipelineExists checks if a pipeline exists
func (i *Importer) pipelineExists(id string) bool {
	var exists int
	err := i.db.QueryRow(`SELECT 1 FROM pipelines WHERE id = ?`, id).Scan(&exists)
	return err == nil
}

// insertPipeline inserts a new pipeline
func (i *Importer) insertPipeline(p PipelineData) error {
	_, err := i.db.Exec(`
		INSERT INTO pipelines (id, name, session_id, status, created_at, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, nullString(p.SessionID), p.Status,
		p.CreatedAt, nullTime(p.StartedAt), nullTime(p.CompletedAt))

	if err != nil {
		return err
	}

	// Insert pipeline ports
	for _, pp := range p.Ports {
		i.db.Exec(`
			INSERT OR REPLACE INTO pipeline_ports (pipeline_id, port_id, group_order, status)
			VALUES (?, ?, ?, ?)
		`, p.ID, pp.PortID, pp.GroupOrder, pp.Status)
	}

	return nil
}

// updatePipeline updates an existing pipeline
func (i *Importer) updatePipeline(p PipelineData) error {
	_, err := i.db.Exec(`
		UPDATE pipelines SET name = ?, status = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`, p.Name, p.Status, nullTime(p.StartedAt), nullTime(p.CompletedAt), p.ID)

	if err != nil {
		return err
	}

	// Update pipeline ports
	i.db.Exec(`DELETE FROM pipeline_ports WHERE pipeline_id = ?`, p.ID)
	for _, pp := range p.Ports {
		i.db.Exec(`
			INSERT INTO pipeline_ports (pipeline_id, port_id, group_order, status)
			VALUES (?, ?, ?, ?)
		`, p.ID, pp.PortID, pp.GroupOrder, pp.Status)
	}

	return nil
}

// importProjects imports projects
func (i *Importer) importProjects(projects []ProjectData, result *ImportResult) error {
	for _, p := range projects {
		exists := i.projectExists(p.Root)

		if exists {
			if i.options.SkipConflicts || i.options.Strategy == MergeStrategyKeepLocal {
				result.Skipped.Projects++
				continue
			}

			if !i.options.DryRun {
				if err := i.updateProject(p); err != nil {
					return err
				}
			}
			result.Imported.Projects++
		} else {
			if !i.options.DryRun {
				if err := i.insertProject(p); err != nil {
					return err
				}
			}
			result.Imported.Projects++
		}
	}
	return nil
}

// projectExists checks if a project exists
func (i *Importer) projectExists(root string) bool {
	var exists int
	// Check both root and logical_root
	err := i.db.QueryRow(`SELECT 1 FROM projects WHERE root = ? OR logical_root = ?`, root, root).Scan(&exists)
	return err == nil
}

// insertProject inserts a new project
func (i *Importer) insertProject(p ProjectData) error {
	_, err := i.db.Exec(`
		INSERT INTO projects (root, logical_root, name, description, last_active,
		                      session_count, total_tokens, total_cost, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.Root, p.Root, nullString(p.Name), nullString(p.Description),
		nullTime(&p.LastActive), p.SessionCount, p.TotalTokens, p.TotalCost, p.CreatedAt)
	return err
}

// updateProject updates an existing project
func (i *Importer) updateProject(p ProjectData) error {
	_, err := i.db.Exec(`
		UPDATE projects SET name = ?, description = ?, last_active = ?,
		                    session_count = ?, total_tokens = ?, total_cost = ?
		WHERE root = ? OR logical_root = ?
	`, nullString(p.Name), nullString(p.Description), nullTime(&p.LastActive),
		p.SessionCount, p.TotalTokens, p.TotalCost, p.Root, p.Root)
	return err
}

// Helper functions

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil || t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
