package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/env"
)

const (
	ConflictFileName = "conflicts.yaml"
)

// ConflictType represents the type of data in conflict
type ConflictType string

const (
	ConflictTypePort       ConflictType = "port"
	ConflictTypeSession    ConflictType = "session"
	ConflictTypeEscalation ConflictType = "escalation"
	ConflictTypePipeline   ConflictType = "pipeline"
	ConflictTypeProject    ConflictType = "project"
)

// ConflictDetail provides detailed conflict information
type ConflictDetail struct {
	ID            string       `yaml:"id" json:"id"`
	Type          ConflictType `yaml:"type" json:"type"`
	LocalVersion  interface{}  `yaml:"local" json:"local"`
	RemoteVersion interface{}  `yaml:"remote" json:"remote"`
	LocalEnv      string       `yaml:"local_env,omitempty" json:"local_env,omitempty"`
	RemoteEnv     string       `yaml:"remote_env,omitempty" json:"remote_env,omitempty"`
	LocalModified time.Time    `yaml:"local_modified,omitempty" json:"local_modified,omitempty"`
	RemoteModified time.Time   `yaml:"remote_modified,omitempty" json:"remote_modified,omitempty"`
	Differences   []FieldDiff  `yaml:"differences" json:"differences"`
	Resolution    string       `yaml:"resolution,omitempty" json:"resolution,omitempty"`
	ResolvedAt    *time.Time   `yaml:"resolved_at,omitempty" json:"resolved_at,omitempty"`
}

// FieldDiff represents a single field difference
type FieldDiff struct {
	Field      string      `yaml:"field" json:"field"`
	LocalValue interface{} `yaml:"local" json:"local"`
	RemoteValue interface{} `yaml:"remote" json:"remote"`
}

// ConflictStore stores pending conflicts
type ConflictStore struct {
	Version   int              `yaml:"version" json:"version"`
	CreatedAt time.Time        `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time        `yaml:"updated_at" json:"updated_at"`
	Conflicts []ConflictDetail `yaml:"conflicts" json:"conflicts"`
}

// ConflictResolver handles conflict detection and resolution
type ConflictResolver struct {
	db       *db.DB
	envSvc   *env.Service
	syncDir  string
	storePath string
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(database *db.DB, envSvc *env.Service) *ConflictResolver {
	syncDir := filepath.Join(config.GlobalDir(), SyncDirName)
	return &ConflictResolver{
		db:        database,
		envSvc:    envSvc,
		syncDir:   syncDir,
		storePath: filepath.Join(syncDir, ConflictFileName),
	}
}

// DetectConflicts detects conflicts between local and remote data
func (r *ConflictResolver) DetectConflicts(remoteData *SyncData) (*ConflictStore, error) {
	store := &ConflictStore{
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Conflicts: []ConflictDetail{},
	}

	// Get current environment
	currentEnv, _ := r.envSvc.Current()
	localEnvID := ""
	if currentEnv != nil {
		localEnvID = currentEnv.ID
	}

	remoteEnvID := remoteData.Manifest.ExportedEnv

	// Detect port conflicts
	for _, remotePort := range remoteData.Ports {
		localPort := r.getLocalPort(remotePort.ID)
		if localPort != nil {
			conflict := r.comparePort(localPort, &remotePort, localEnvID, remoteEnvID)
			if conflict != nil {
				store.Conflicts = append(store.Conflicts, *conflict)
			}
		}
	}

	// Detect session conflicts
	for _, remoteSession := range remoteData.Sessions {
		localSession := r.getLocalSession(remoteSession.ID)
		if localSession != nil {
			conflict := r.compareSession(localSession, &remoteSession, localEnvID, remoteEnvID)
			if conflict != nil {
				store.Conflicts = append(store.Conflicts, *conflict)
			}
		}
	}

	// Detect escalation conflicts
	for _, remoteEsc := range remoteData.Escalations {
		localEsc := r.getLocalEscalation(remoteEsc.ID)
		if localEsc != nil {
			conflict := r.compareEscalation(localEsc, &remoteEsc, localEnvID, remoteEnvID)
			if conflict != nil {
				store.Conflicts = append(store.Conflicts, *conflict)
			}
		}
	}

	// Detect pipeline conflicts
	for _, remotePipeline := range remoteData.Pipelines {
		localPipeline := r.getLocalPipeline(remotePipeline.ID)
		if localPipeline != nil {
			conflict := r.comparePipeline(localPipeline, &remotePipeline, localEnvID, remoteEnvID)
			if conflict != nil {
				store.Conflicts = append(store.Conflicts, *conflict)
			}
		}
	}

	// Detect project conflicts
	for _, remoteProject := range remoteData.Projects {
		localProject := r.getLocalProject(remoteProject.Root)
		if localProject != nil {
			conflict := r.compareProject(localProject, &remoteProject, localEnvID, remoteEnvID)
			if conflict != nil {
				store.Conflicts = append(store.Conflicts, *conflict)
			}
		}
	}

	return store, nil
}

// SaveConflicts saves pending conflicts to file
func (r *ConflictResolver) SaveConflicts(store *ConflictStore) error {
	os.MkdirAll(r.syncDir, 0755)

	store.UpdatedAt = time.Now()

	data, err := yaml.Marshal(store)
	if err != nil {
		return fmt.Errorf("충돌 데이터 직렬화 실패: %w", err)
	}

	return os.WriteFile(r.storePath, data, 0644)
}

// LoadConflicts loads pending conflicts from file
func (r *ConflictResolver) LoadConflicts() (*ConflictStore, error) {
	data, err := os.ReadFile(r.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ConflictStore{
				Version:   1,
				Conflicts: []ConflictDetail{},
			}, nil
		}
		return nil, fmt.Errorf("충돌 파일 읽기 실패: %w", err)
	}

	var store ConflictStore
	if err := yaml.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("충돌 파일 파싱 실패: %w", err)
	}

	return &store, nil
}

// ClearConflicts removes the conflict file
func (r *ConflictResolver) ClearConflicts() error {
	if _, err := os.Stat(r.storePath); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(r.storePath)
}

// HasPendingConflicts checks if there are unresolved conflicts
func (r *ConflictResolver) HasPendingConflicts() bool {
	store, err := r.LoadConflicts()
	if err != nil {
		return false
	}

	for _, c := range store.Conflicts {
		if c.Resolution == "" {
			return true
		}
	}
	return false
}

// GetPendingConflicts returns unresolved conflicts
func (r *ConflictResolver) GetPendingConflicts() ([]ConflictDetail, error) {
	store, err := r.LoadConflicts()
	if err != nil {
		return nil, err
	}

	pending := []ConflictDetail{}
	for _, c := range store.Conflicts {
		if c.Resolution == "" {
			pending = append(pending, c)
		}
	}
	return pending, nil
}

// ResolveConflict resolves a single conflict
func (r *ConflictResolver) ResolveConflict(id string, conflictType ConflictType, resolution string) error {
	store, err := r.LoadConflicts()
	if err != nil {
		return err
	}

	found := false
	for i, c := range store.Conflicts {
		if c.ID == id && c.Type == conflictType {
			now := time.Now()
			store.Conflicts[i].Resolution = resolution
			store.Conflicts[i].ResolvedAt = &now
			found = true

			// Apply resolution
			if err := r.applyResolution(&store.Conflicts[i]); err != nil {
				return fmt.Errorf("해결 적용 실패: %w", err)
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("충돌을 찾을 수 없음: %s (%s)", id, conflictType)
	}

	return r.SaveConflicts(store)
}

// ResolveAll resolves all conflicts with the given strategy
func (r *ConflictResolver) ResolveAll(resolution string) error {
	store, err := r.LoadConflicts()
	if err != nil {
		return err
	}

	now := time.Now()
	for i := range store.Conflicts {
		if store.Conflicts[i].Resolution == "" {
			store.Conflicts[i].Resolution = resolution
			store.Conflicts[i].ResolvedAt = &now

			if err := r.applyResolution(&store.Conflicts[i]); err != nil {
				return fmt.Errorf("해결 적용 실패 (%s): %w", store.Conflicts[i].ID, err)
			}
		}
	}

	return r.SaveConflicts(store)
}

// applyResolution applies the resolution to the database
func (r *ConflictResolver) applyResolution(conflict *ConflictDetail) error {
	switch conflict.Resolution {
	case "keep_local":
		// Nothing to do - local data is already in DB
		return nil

	case "keep_remote":
		// Update local DB with remote data
		return r.applyRemoteData(conflict)

	case "skip":
		// Do nothing
		return nil

	default:
		return fmt.Errorf("알 수 없는 해결 방법: %s", conflict.Resolution)
	}
}

// applyRemoteData updates local DB with remote data
func (r *ConflictResolver) applyRemoteData(conflict *ConflictDetail) error {
	switch conflict.Type {
	case ConflictTypePort:
		port, ok := conflict.RemoteVersion.(*PortData)
		if !ok {
			// Try to unmarshal from map
			data, _ := json.Marshal(conflict.RemoteVersion)
			port = &PortData{}
			json.Unmarshal(data, port)
		}
		return r.updatePort(port)

	case ConflictTypeSession:
		session, ok := conflict.RemoteVersion.(*SessionData)
		if !ok {
			data, _ := json.Marshal(conflict.RemoteVersion)
			session = &SessionData{}
			json.Unmarshal(data, session)
		}
		return r.updateSession(session)

	case ConflictTypeEscalation:
		esc, ok := conflict.RemoteVersion.(*Escalation)
		if !ok {
			data, _ := json.Marshal(conflict.RemoteVersion)
			esc = &Escalation{}
			json.Unmarshal(data, esc)
		}
		return r.updateEscalation(esc)

	case ConflictTypePipeline:
		pipeline, ok := conflict.RemoteVersion.(*PipelineData)
		if !ok {
			data, _ := json.Marshal(conflict.RemoteVersion)
			pipeline = &PipelineData{}
			json.Unmarshal(data, pipeline)
		}
		return r.updatePipeline(pipeline)

	case ConflictTypeProject:
		project, ok := conflict.RemoteVersion.(*ProjectData)
		if !ok {
			data, _ := json.Marshal(conflict.RemoteVersion)
			project = &ProjectData{}
			json.Unmarshal(data, project)
		}
		return r.updateProject(project)
	}

	return nil
}

// Comparison methods

func (r *ConflictResolver) comparePort(local, remote *PortData, localEnv, remoteEnv string) *ConflictDetail {
	diffs := []FieldDiff{}

	if local.Status != remote.Status {
		diffs = append(diffs, FieldDiff{"status", local.Status, remote.Status})
	}
	if local.Title != remote.Title {
		diffs = append(diffs, FieldDiff{"title", local.Title, remote.Title})
	}
	if local.InputTokens != remote.InputTokens {
		diffs = append(diffs, FieldDiff{"input_tokens", local.InputTokens, remote.InputTokens})
	}
	if local.OutputTokens != remote.OutputTokens {
		diffs = append(diffs, FieldDiff{"output_tokens", local.OutputTokens, remote.OutputTokens})
	}
	if local.CostUSD != remote.CostUSD {
		diffs = append(diffs, FieldDiff{"cost_usd", local.CostUSD, remote.CostUSD})
	}

	if len(diffs) == 0 {
		return nil
	}

	// Determine modification times
	localMod := local.CreatedAt
	if local.CompletedAt != nil {
		localMod = *local.CompletedAt
	} else if local.StartedAt != nil {
		localMod = *local.StartedAt
	}

	remoteMod := remote.CreatedAt
	if remote.CompletedAt != nil {
		remoteMod = *remote.CompletedAt
	} else if remote.StartedAt != nil {
		remoteMod = *remote.StartedAt
	}

	return &ConflictDetail{
		ID:             local.ID,
		Type:           ConflictTypePort,
		LocalVersion:   local,
		RemoteVersion:  remote,
		LocalEnv:       localEnv,
		RemoteEnv:      remoteEnv,
		LocalModified:  localMod,
		RemoteModified: remoteMod,
		Differences:    diffs,
	}
}

func (r *ConflictResolver) compareSession(local, remote *SessionData, localEnv, remoteEnv string) *ConflictDetail {
	diffs := []FieldDiff{}

	if local.Status != remote.Status {
		diffs = append(diffs, FieldDiff{"status", local.Status, remote.Status})
	}
	if local.InputTokens != remote.InputTokens {
		diffs = append(diffs, FieldDiff{"input_tokens", local.InputTokens, remote.InputTokens})
	}
	if local.OutputTokens != remote.OutputTokens {
		diffs = append(diffs, FieldDiff{"output_tokens", local.OutputTokens, remote.OutputTokens})
	}
	if local.CostUSD != remote.CostUSD {
		diffs = append(diffs, FieldDiff{"cost_usd", local.CostUSD, remote.CostUSD})
	}
	if local.CompactCount != remote.CompactCount {
		diffs = append(diffs, FieldDiff{"compact_count", local.CompactCount, remote.CompactCount})
	}

	if len(diffs) == 0 {
		return nil
	}

	localMod := local.StartedAt
	if local.EndedAt != nil {
		localMod = *local.EndedAt
	}

	remoteMod := remote.StartedAt
	if remote.EndedAt != nil {
		remoteMod = *remote.EndedAt
	}

	return &ConflictDetail{
		ID:             local.ID,
		Type:           ConflictTypeSession,
		LocalVersion:   local,
		RemoteVersion:  remote,
		LocalEnv:       localEnv,
		RemoteEnv:      remoteEnv,
		LocalModified:  localMod,
		RemoteModified: remoteMod,
		Differences:    diffs,
	}
}

func (r *ConflictResolver) compareEscalation(local, remote *Escalation, localEnv, remoteEnv string) *ConflictDetail {
	diffs := []FieldDiff{}

	if local.Status != remote.Status {
		diffs = append(diffs, FieldDiff{"status", local.Status, remote.Status})
	}
	if local.Issue != remote.Issue {
		diffs = append(diffs, FieldDiff{"issue", local.Issue, remote.Issue})
	}

	if len(diffs) == 0 {
		return nil
	}

	localMod := local.CreatedAt
	if local.ResolvedAt != nil {
		localMod = *local.ResolvedAt
	}

	remoteMod := remote.CreatedAt
	if remote.ResolvedAt != nil {
		remoteMod = *remote.ResolvedAt
	}

	return &ConflictDetail{
		ID:             fmt.Sprintf("%d", local.ID),
		Type:           ConflictTypeEscalation,
		LocalVersion:   local,
		RemoteVersion:  remote,
		LocalEnv:       localEnv,
		RemoteEnv:      remoteEnv,
		LocalModified:  localMod,
		RemoteModified: remoteMod,
		Differences:    diffs,
	}
}

func (r *ConflictResolver) comparePipeline(local, remote *PipelineData, localEnv, remoteEnv string) *ConflictDetail {
	diffs := []FieldDiff{}

	if local.Status != remote.Status {
		diffs = append(diffs, FieldDiff{"status", local.Status, remote.Status})
	}
	if local.Name != remote.Name {
		diffs = append(diffs, FieldDiff{"name", local.Name, remote.Name})
	}

	if len(diffs) == 0 {
		return nil
	}

	localMod := local.CreatedAt
	if local.CompletedAt != nil {
		localMod = *local.CompletedAt
	} else if local.StartedAt != nil {
		localMod = *local.StartedAt
	}

	remoteMod := remote.CreatedAt
	if remote.CompletedAt != nil {
		remoteMod = *remote.CompletedAt
	} else if remote.StartedAt != nil {
		remoteMod = *remote.StartedAt
	}

	return &ConflictDetail{
		ID:             local.ID,
		Type:           ConflictTypePipeline,
		LocalVersion:   local,
		RemoteVersion:  remote,
		LocalEnv:       localEnv,
		RemoteEnv:      remoteEnv,
		LocalModified:  localMod,
		RemoteModified: remoteMod,
		Differences:    diffs,
	}
}

func (r *ConflictResolver) compareProject(local, remote *ProjectData, localEnv, remoteEnv string) *ConflictDetail {
	diffs := []FieldDiff{}

	if local.SessionCount != remote.SessionCount {
		diffs = append(diffs, FieldDiff{"session_count", local.SessionCount, remote.SessionCount})
	}
	if local.TotalTokens != remote.TotalTokens {
		diffs = append(diffs, FieldDiff{"total_tokens", local.TotalTokens, remote.TotalTokens})
	}
	if local.TotalCost != remote.TotalCost {
		diffs = append(diffs, FieldDiff{"total_cost", local.TotalCost, remote.TotalCost})
	}

	if len(diffs) == 0 {
		return nil
	}

	return &ConflictDetail{
		ID:             local.Root,
		Type:           ConflictTypeProject,
		LocalVersion:   local,
		RemoteVersion:  remote,
		LocalEnv:       localEnv,
		RemoteEnv:      remoteEnv,
		LocalModified:  local.LastActive,
		RemoteModified: remote.LastActive,
		Differences:    diffs,
	}
}

// Database query methods

func (r *ConflictResolver) getLocalPort(id string) *PortData {
	var p PortData
	var title, filePath, agentID *string
	var startedAt, completedAt *time.Time

	err := r.db.QueryRow(`
		SELECT id, title, status, file_path, created_at, started_at, completed_at,
		       COALESCE(input_tokens, 0), COALESCE(output_tokens, 0),
		       COALESCE(cost_usd, 0), COALESCE(duration_secs, 0), agent_id
		FROM ports WHERE id = ?
	`, id).Scan(
		&p.ID, &title, &p.Status, &filePath, &p.CreatedAt,
		&startedAt, &completedAt, &p.InputTokens, &p.OutputTokens,
		&p.CostUSD, &p.DurationSecs, &agentID,
	)

	if err != nil {
		return nil
	}

	if title != nil {
		p.Title = *title
	}
	if filePath != nil {
		p.FilePath = *filePath
	}
	if agentID != nil {
		p.AgentID = *agentID
	}
	if startedAt != nil {
		p.StartedAt = startedAt
	}
	if completedAt != nil {
		p.CompletedAt = completedAt
	}

	return &p
}

func (r *ConflictResolver) getLocalSession(id string) *SessionData {
	var s SessionData
	var portID, title, parentSession, projectRoot, projectName, createdEnv, lastEnv *string
	var endedAt *time.Time

	err := r.db.QueryRow(`
		SELECT id, port_id, title, status, session_type, parent_session,
		       started_at, ended_at, input_tokens, output_tokens,
		       cache_read_tokens, cache_create_tokens, cost_usd, compact_count,
		       project_root, project_name, created_env, last_env
		FROM sessions WHERE id = ?
	`, id).Scan(
		&s.ID, &portID, &title, &s.Status, &s.SessionType, &parentSession,
		&s.StartedAt, &endedAt, &s.InputTokens, &s.OutputTokens,
		&s.CacheReadTokens, &s.CacheCreateTokens, &s.CostUSD, &s.CompactCount,
		&projectRoot, &projectName, &createdEnv, &lastEnv,
	)

	if err != nil {
		return nil
	}

	if portID != nil {
		s.PortID = *portID
	}
	if title != nil {
		s.Title = *title
	}
	if parentSession != nil {
		s.ParentSession = *parentSession
	}
	if projectRoot != nil {
		s.ProjectRoot = *projectRoot
	}
	if projectName != nil {
		s.ProjectName = *projectName
	}
	if createdEnv != nil {
		s.CreatedEnv = *createdEnv
	}
	if lastEnv != nil {
		s.LastEnv = *lastEnv
	}
	if endedAt != nil {
		s.EndedAt = endedAt
	}

	return &s
}

func (r *ConflictResolver) getLocalEscalation(id int64) *Escalation {
	var e Escalation
	var fromSession, fromPort *string
	var resolvedAt *time.Time

	err := r.db.QueryRow(`
		SELECT id, from_session, from_port, issue, status, created_at, resolved_at
		FROM escalations WHERE id = ?
	`, id).Scan(
		&e.ID, &fromSession, &fromPort, &e.Issue, &e.Status, &e.CreatedAt, &resolvedAt,
	)

	if err != nil {
		return nil
	}

	if fromSession != nil {
		e.FromSession = *fromSession
	}
	if fromPort != nil {
		e.FromPort = *fromPort
	}
	if resolvedAt != nil {
		e.ResolvedAt = resolvedAt
	}

	return &e
}

func (r *ConflictResolver) getLocalPipeline(id string) *PipelineData {
	var p PipelineData
	var sessionID *string
	var startedAt, completedAt *time.Time

	err := r.db.QueryRow(`
		SELECT id, name, session_id, status, created_at, started_at, completed_at
		FROM pipelines WHERE id = ?
	`, id).Scan(
		&p.ID, &p.Name, &sessionID, &p.Status, &p.CreatedAt, &startedAt, &completedAt,
	)

	if err != nil {
		return nil
	}

	if sessionID != nil {
		p.SessionID = *sessionID
	}
	if startedAt != nil {
		p.StartedAt = startedAt
	}
	if completedAt != nil {
		p.CompletedAt = completedAt
	}

	return &p
}

func (r *ConflictResolver) getLocalProject(root string) *ProjectData {
	var p ProjectData
	var name, description *string
	var lastActive *time.Time

	err := r.db.QueryRow(`
		SELECT root, name, description, last_active, session_count, total_tokens, total_cost, created_at
		FROM projects WHERE root = ? OR logical_root = ?
	`, root, root).Scan(
		&p.Root, &name, &description, &lastActive,
		&p.SessionCount, &p.TotalTokens, &p.TotalCost, &p.CreatedAt,
	)

	if err != nil {
		return nil
	}

	if name != nil {
		p.Name = *name
	}
	if description != nil {
		p.Description = *description
	}
	if lastActive != nil {
		p.LastActive = *lastActive
	}

	return &p
}

// Update methods for applying remote data

func (r *ConflictResolver) updatePort(p *PortData) error {
	_, err := r.db.Exec(`
		UPDATE ports SET title = ?, status = ?, file_path = ?, started_at = ?, completed_at = ?,
		                 input_tokens = ?, output_tokens = ?, cost_usd = ?, duration_secs = ?, agent_id = ?
		WHERE id = ?
	`, nullString(p.Title), p.Status, nullString(p.FilePath),
		nullTime(p.StartedAt), nullTime(p.CompletedAt),
		p.InputTokens, p.OutputTokens, p.CostUSD, p.DurationSecs, nullString(p.AgentID), p.ID)
	return err
}

func (r *ConflictResolver) updateSession(s *SessionData) error {
	currentEnv, _ := r.envSvc.Current()
	currentEnvID := ""
	if currentEnv != nil {
		currentEnvID = currentEnv.ID
	}

	_, err := r.db.Exec(`
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

func (r *ConflictResolver) updateEscalation(e *Escalation) error {
	_, err := r.db.Exec(`
		UPDATE escalations SET issue = ?, status = ?, resolved_at = ?
		WHERE id = ?
	`, e.Issue, e.Status, nullTime(e.ResolvedAt), e.ID)
	return err
}

func (r *ConflictResolver) updatePipeline(p *PipelineData) error {
	_, err := r.db.Exec(`
		UPDATE pipelines SET name = ?, status = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`, p.Name, p.Status, nullTime(p.StartedAt), nullTime(p.CompletedAt), p.ID)
	return err
}

func (r *ConflictResolver) updateProject(p *ProjectData) error {
	_, err := r.db.Exec(`
		UPDATE projects SET name = ?, description = ?, last_active = ?,
		                    session_count = ?, total_tokens = ?, total_cost = ?
		WHERE root = ? OR logical_root = ?
	`, nullString(p.Name), nullString(p.Description), nullTime(&p.LastActive),
		p.SessionCount, p.TotalTokens, p.TotalCost, p.Root, p.Root)
	return err
}

// DiffSummary provides a summary of differences
type DiffSummary struct {
	TotalConflicts   int            `json:"total_conflicts"`
	ByType           map[string]int `json:"by_type"`
	NewerLocal       int            `json:"newer_local"`
	NewerRemote      int            `json:"newer_remote"`
	PendingCount     int            `json:"pending_count"`
	ResolvedCount    int            `json:"resolved_count"`
}

// GetDiffSummary returns a summary of conflicts
func (r *ConflictResolver) GetDiffSummary() (*DiffSummary, error) {
	store, err := r.LoadConflicts()
	if err != nil {
		return nil, err
	}

	summary := &DiffSummary{
		TotalConflicts: len(store.Conflicts),
		ByType:         make(map[string]int),
	}

	for _, c := range store.Conflicts {
		summary.ByType[string(c.Type)]++

		if c.LocalModified.After(c.RemoteModified) {
			summary.NewerLocal++
		} else if c.RemoteModified.After(c.LocalModified) {
			summary.NewerRemote++
		}

		if c.Resolution == "" {
			summary.PendingCount++
		} else {
			summary.ResolvedCount++
		}
	}

	return summary, nil
}
