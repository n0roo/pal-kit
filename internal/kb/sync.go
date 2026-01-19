package kb

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SyncService handles project synchronization
type SyncService struct {
	vaultPath   string
	projectPath string
	projectName string
}

// SyncState represents the synchronization state
type SyncState struct {
	Version      string                  `yaml:"version"`
	LastSync     string                  `yaml:"last_sync"`
	Projects     map[string]ProjectState `yaml:"projects"`
	SyncMode     string                  `yaml:"sync_mode"` // "one-way" or "two-way"
	LastModified string                  `yaml:"last_modified"`
}

// ProjectState represents a project's sync state
type ProjectState struct {
	Name       string            `yaml:"name"`
	SourcePath string            `yaml:"source_path"`
	LastSync   string            `yaml:"last_sync"`
	Files      map[string]string `yaml:"files"` // path -> hash
}

// SyncResult represents sync operation result
type SyncResult struct {
	ProjectName string        `json:"project_name"`
	Added       []string      `json:"added,omitempty"`
	Updated     []string      `json:"updated,omitempty"`
	Deleted     []string      `json:"deleted,omitempty"`
	Skipped     []string      `json:"skipped,omitempty"`
	Conflicts   []SyncConflict `json:"conflicts,omitempty"`
	SyncTime    string        `json:"sync_time"`
}

// SyncConflict represents a sync conflict
type SyncConflict struct {
	Path       string `json:"path"`
	SourceTime string `json:"source_time"`
	TargetTime string `json:"target_time"`
	Resolution string `json:"resolution,omitempty"`
}

// SyncOptions represents sync options
type SyncOptions struct {
	DryRun    bool   `json:"dry_run"`
	Force     bool   `json:"force"`
	Direction string `json:"direction"` // "to-vault", "from-vault", "both"
}

// NewSyncService creates a new sync service
func NewSyncService(vaultPath, projectPath string) *SyncService {
	projectName := filepath.Base(projectPath)
	return &SyncService{
		vaultPath:   vaultPath,
		projectPath: projectPath,
		projectName: projectName,
	}
}

// SyncMappings defines what to sync from project to vault
var SyncMappings = []struct {
	Source string
	Target string
	Desc   string
}{
	{"ports", "ports", "포트 명세"},
	{".pal/decisions", "decisions", "결정 기록"},
	{".pal/sessions", "sessions", "세션 기록"},
	{"docs", "docs", "문서"},
}

// Sync synchronizes project to vault
func (s *SyncService) Sync(opts *SyncOptions) (*SyncResult, error) {
	if opts == nil {
		opts = &SyncOptions{Direction: "to-vault"}
	}

	result := &SyncResult{
		ProjectName: s.projectName,
		SyncTime:    time.Now().Format(time.RFC3339),
	}

	// Load or create sync state
	state, err := s.loadSyncState()
	if err != nil {
		state = &SyncState{
			Version:  "1",
			Projects: make(map[string]ProjectState),
			SyncMode: "one-way",
		}
	}

	// Get or create project state
	projectState, ok := state.Projects[s.projectName]
	if !ok {
		projectState = ProjectState{
			Name:       s.projectName,
			SourcePath: s.projectPath,
			Files:      make(map[string]string),
		}
	}

	// Ensure target directory exists
	targetDir := filepath.Join(s.vaultPath, ProjectsDir, s.projectName)
	if !opts.DryRun {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return nil, fmt.Errorf("타겟 디렉토리 생성 실패: %w", err)
		}
	}

	// Process each mapping
	for _, mapping := range SyncMappings {
		sourcePath := filepath.Join(s.projectPath, mapping.Source)
		targetPath := filepath.Join(targetDir, mapping.Target)

		// Check if source exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			continue
		}

		// Sync directory (use mapping.Target as prefix for unique file keys)
		syncResult, err := s.syncDirectory(sourcePath, targetPath, mapping.Target, &projectState, opts)
		if err != nil {
			return result, fmt.Errorf("%s 동기화 실패: %w", mapping.Desc, err)
		}

		result.Added = append(result.Added, syncResult.Added...)
		result.Updated = append(result.Updated, syncResult.Updated...)
		result.Deleted = append(result.Deleted, syncResult.Deleted...)
		result.Skipped = append(result.Skipped, syncResult.Skipped...)
		result.Conflicts = append(result.Conflicts, syncResult.Conflicts...)
	}

	// Create project index if new
	if !opts.DryRun {
		indexPath := filepath.Join(targetDir, "_index.md")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			s.createProjectIndex(indexPath)
		}
	}

	// Save sync state
	if !opts.DryRun {
		projectState.LastSync = result.SyncTime
		state.Projects[s.projectName] = projectState
		state.LastSync = result.SyncTime
		state.LastModified = result.SyncTime
		if err := s.saveSyncState(state); err != nil {
			return result, fmt.Errorf("상태 저장 실패: %w", err)
		}
	}

	return result, nil
}

func (s *SyncService) syncDirectory(sourcePath, targetPath, keyPrefix string, projectState *ProjectState, opts *SyncOptions) (*SyncResult, error) {
	result := &SyncResult{}

	// Ensure target directory exists
	if !opts.DryRun {
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return nil, err
		}
	}

	// Track files in source (using prefixed keys)
	sourceFiles := make(map[string]bool)

	// Walk source directory
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Only sync markdown files
		if filepath.Ext(path) != ".md" && filepath.Ext(path) != ".yaml" {
			return nil
		}

		// Get relative path and create prefixed key for unique identification
		relPath, _ := filepath.Rel(sourcePath, path)
		fileKey := filepath.Join(keyPrefix, relPath)
		sourceFiles[fileKey] = true

		// Calculate hash
		hash, err := s.fileHash(path)
		if err != nil {
			return nil
		}

		// Check if file changed
		targetFile := filepath.Join(targetPath, relPath)
		prevHash := projectState.Files[fileKey]

		if prevHash == hash {
			// No change
			result.Skipped = append(result.Skipped, relPath)
			return nil
		}

		// Check for conflicts (target modified independently)
		if prevHash != "" && !opts.Force {
			if targetInfo, err := os.Stat(targetFile); err == nil {
				targetHash, _ := s.fileHash(targetFile)
				if targetHash != prevHash {
					// Both source and target changed
					result.Conflicts = append(result.Conflicts, SyncConflict{
						Path:       relPath,
						SourceTime: info.ModTime().Format(time.RFC3339),
						TargetTime: targetInfo.ModTime().Format(time.RFC3339),
					})
					return nil
				}
			}
		}

		// Copy file
		if !opts.DryRun {
			// Ensure parent directory exists
			parentDir := filepath.Dir(targetFile)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return err
			}

			if err := s.copyFile(path, targetFile); err != nil {
				return err
			}
			projectState.Files[fileKey] = hash
		}

		if prevHash == "" {
			result.Added = append(result.Added, relPath)
		} else {
			result.Updated = append(result.Updated, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Check for deleted files (only within this keyPrefix scope)
	for fileKey := range projectState.Files {
		// Only check files belonging to this mapping
		if !strings.HasPrefix(fileKey, keyPrefix+"/") && fileKey != keyPrefix {
			continue
		}
		if !sourceFiles[fileKey] {
			// Extract relative path from prefixed key
			relPath := strings.TrimPrefix(fileKey, keyPrefix+"/")
			targetFile := filepath.Join(targetPath, relPath)
			if _, err := os.Stat(targetFile); err == nil {
				if !opts.DryRun {
					os.Remove(targetFile)
					delete(projectState.Files, fileKey)
				}
				result.Deleted = append(result.Deleted, relPath)
			}
		}
	}

	return result, nil
}

func (s *SyncService) fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (s *SyncService) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (s *SyncService) createProjectIndex(path string) error {
	content := fmt.Sprintf(`---
type: project
title: %s
created: %s
---

# %s

> 프로젝트 문서

## 섹션

- [[ports/_toc|포트 명세]]
- [[decisions/_toc|결정 기록]]
- [[sessions/_toc|세션 기록]]

---

tags: #project #%s
`, s.projectName, time.Now().Format("2006-01-02"), s.projectName, s.projectName)

	return os.WriteFile(path, []byte(content), 0644)
}

func (s *SyncService) loadSyncState() (*SyncState, error) {
	statePath := filepath.Join(s.vaultPath, MetaDir, "sync-state.yaml")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state SyncState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (s *SyncService) saveSyncState(state *SyncState) error {
	statePath := filepath.Join(s.vaultPath, MetaDir, "sync-state.yaml")

	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}

// GetSyncStatus returns the sync status for a project
func (s *SyncService) GetSyncStatus() (*ProjectState, error) {
	state, err := s.loadSyncState()
	if err != nil {
		return nil, err
	}

	projectState, ok := state.Projects[s.projectName]
	if !ok {
		return nil, fmt.Errorf("프로젝트 '%s'가 동기화된 적 없음", s.projectName)
	}

	return &projectState, nil
}

// ListSyncedProjects returns all synced projects
func (s *SyncService) ListSyncedProjects() ([]ProjectState, error) {
	state, err := s.loadSyncState()
	if err != nil {
		return nil, err
	}

	var projects []ProjectState
	for _, p := range state.Projects {
		projects = append(projects, p)
	}

	return projects, nil
}

// CheckChanges checks for changes without syncing
func (s *SyncService) CheckChanges() (*SyncResult, error) {
	return s.Sync(&SyncOptions{DryRun: true})
}

// ResolvePath resolves a project document path to vault path
func (s *SyncService) ResolvePath(projectPath string) string {
	// Check which mapping this path belongs to
	for _, mapping := range SyncMappings {
		if strings.HasPrefix(projectPath, mapping.Source) {
			relPath := strings.TrimPrefix(projectPath, mapping.Source)
			relPath = strings.TrimPrefix(relPath, "/")
			return filepath.Join(ProjectsDir, s.projectName, mapping.Target, relPath)
		}
	}
	return ""
}

// GetBacklink returns the project path for a vault document
func (s *SyncService) GetBacklink(vaultPath string) string {
	// Check if this is a synced document
	prefix := filepath.Join(ProjectsDir, s.projectName)
	if !strings.HasPrefix(vaultPath, prefix) {
		return ""
	}

	relPath := strings.TrimPrefix(vaultPath, prefix+"/")

	// Find which mapping this belongs to
	for _, mapping := range SyncMappings {
		if strings.HasPrefix(relPath, mapping.Target) {
			subPath := strings.TrimPrefix(relPath, mapping.Target)
			subPath = strings.TrimPrefix(subPath, "/")
			return filepath.Join(mapping.Source, subPath)
		}
	}

	return ""
}
