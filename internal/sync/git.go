package sync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/env"
)

const (
	SyncDirName   = "sync"
	SyncFileName  = "pal-data.yaml"
	GitIgnoreFile = ".gitignore"
)

// GitSync handles Git-based synchronization
type GitSync struct {
	db       *db.DB
	envSvc   *env.Service
	syncDir  string
	dataFile string
}

// NewGitSync creates a new GitSync instance
func NewGitSync(database *db.DB, envSvc *env.Service) *GitSync {
	syncDir := filepath.Join(config.GlobalDir(), SyncDirName)
	return &GitSync{
		db:       database,
		envSvc:   envSvc,
		syncDir:  syncDir,
		dataFile: filepath.Join(syncDir, SyncFileName),
	}
}

// SyncDir returns the sync directory path
func (g *GitSync) SyncDir() string {
	return g.syncDir
}

// IsInitialized checks if the sync directory is a git repository
func (g *GitSync) IsInitialized() bool {
	gitDir := filepath.Join(g.syncDir, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// HasRemote checks if a remote is configured
func (g *GitSync) HasRemote() bool {
	if !g.IsInitialized() {
		return false
	}
	output, err := g.runGit("remote", "-v")
	return err == nil && strings.TrimSpace(output) != ""
}

// GetRemote returns the configured remote URL
func (g *GitSync) GetRemote() (string, error) {
	if !g.IsInitialized() {
		return "", fmt.Errorf("Git 저장소가 초기화되지 않음")
	}
	output, err := g.runGit("remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("원격 저장소 없음")
	}
	return strings.TrimSpace(output), nil
}

// Init initializes the sync directory as a git repository
func (g *GitSync) Init(remoteURL string) error {
	// Create sync directory
	if err := os.MkdirAll(g.syncDir, 0755); err != nil {
		return fmt.Errorf("sync 디렉토리 생성 실패: %w", err)
	}

	// Initialize git if not already
	if !g.IsInitialized() {
		if _, err := g.runGit("init"); err != nil {
			return fmt.Errorf("Git init 실패: %w", err)
		}
	}

	// Create .gitignore
	gitignoreContent := `# Local files
*.local
*.tmp
.DS_Store
`
	gitignorePath := filepath.Join(g.syncDir, GitIgnoreFile)
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf(".gitignore 생성 실패: %w", err)
	}

	// Set remote if provided
	if remoteURL != "" {
		// Remove existing origin if any
		g.runGit("remote", "remove", "origin")

		if _, err := g.runGit("remote", "add", "origin", remoteURL); err != nil {
			return fmt.Errorf("원격 저장소 설정 실패: %w", err)
		}
	}

	// Create initial commit if no commits exist
	if !g.hasCommits() {
		// Create empty data file
		if err := os.WriteFile(g.dataFile, []byte("# PAL Kit Sync Data\n"), 0644); err != nil {
			return err
		}

		if _, err := g.runGit("add", "."); err != nil {
			return err
		}

		if _, err := g.runGit("commit", "-m", "Initial sync setup"); err != nil {
			return err
		}
	}

	return nil
}

// hasCommits checks if the repository has any commits
func (g *GitSync) hasCommits() bool {
	_, err := g.runGit("rev-parse", "HEAD")
	return err == nil
}

// Push exports data and pushes to remote
func (g *GitSync) Push(message string) (*PushResult, error) {
	if !g.IsInitialized() {
		return nil, fmt.Errorf("Git 저장소가 초기화되지 않음. 'pal sync init' 실행 필요")
	}

	result := &PushResult{}

	// Export data to YAML
	exporter := NewExporter(g.db, g.envSvc)
	data, err := exporter.ExportAll()
	if err != nil {
		return nil, fmt.Errorf("Export 실패: %w", err)
	}

	if err := exporter.ExportToFile(g.dataFile); err != nil {
		return nil, fmt.Errorf("파일 저장 실패: %w", err)
	}

	result.ExportedStats = data.Manifest.Stats

	// Check for changes
	status, err := g.runGit("status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("Git status 실패: %w", err)
	}

	if strings.TrimSpace(status) == "" {
		result.NothingToCommit = true
		return result, nil
	}

	// Get current environment for commit message
	currentEnv, _ := g.envSvc.Current()
	envName := "unknown"
	if currentEnv != nil {
		envName = currentEnv.Name
	}

	// Generate commit message
	if message == "" {
		message = fmt.Sprintf("Sync from %s at %s", envName, time.Now().Format("2006-01-02 15:04:05"))
	}

	// Add and commit
	if _, err := g.runGit("add", "."); err != nil {
		return nil, fmt.Errorf("Git add 실패: %w", err)
	}

	if _, err := g.runGit("commit", "-m", message); err != nil {
		return nil, fmt.Errorf("Git commit 실패: %w", err)
	}

	result.Committed = true

	// Push if remote exists
	if g.HasRemote() {
		// Try to get current branch
		branch, err := g.getCurrentBranch()
		if err != nil {
			branch = "main"
		}

		// Push with set-upstream if needed
		output, err := g.runGit("push", "-u", "origin", branch)
		if err != nil {
			result.PushError = fmt.Sprintf("Push 실패: %v - %s", err, output)
		} else {
			result.Pushed = true
		}
	} else {
		result.NoRemote = true
	}

	// Record sync history
	g.recordSyncHistory("push", data.Manifest.Stats)

	return result, nil
}

// Pull fetches from remote and imports data
func (g *GitSync) Pull(options ImportOptions) (*PullResult, error) {
	if !g.IsInitialized() {
		return nil, fmt.Errorf("Git 저장소가 초기화되지 않음. 'pal sync init' 실행 필요")
	}

	result := &PullResult{}

	// Pull from remote if exists
	if g.HasRemote() {
		branch, err := g.getCurrentBranch()
		if err != nil {
			branch = "main"
		}

		output, err := g.runGit("pull", "origin", branch, "--rebase")
		if err != nil {
			// Check if it's a conflict
			if strings.Contains(output, "CONFLICT") || strings.Contains(err.Error(), "conflict") {
				result.HasConflict = true
				result.ConflictMessage = output
				return result, nil
			}
			result.PullError = fmt.Sprintf("Pull 실패: %v - %s", err, output)
		} else {
			result.Pulled = true
		}
	} else {
		result.NoRemote = true
	}

	// Check if data file exists
	if _, err := os.Stat(g.dataFile); os.IsNotExist(err) {
		result.NoData = true
		return result, nil
	}

	// Import data
	importer := NewImporter(g.db, g.envSvc, options)
	importResult, err := importer.ImportFromFile(g.dataFile)
	if err != nil {
		return nil, fmt.Errorf("Import 실패: %w", err)
	}

	result.ImportResult = importResult

	// Record sync history
	stats := SyncStats{
		PortsCount:       importResult.Imported.Ports,
		SessionsCount:    importResult.Imported.Sessions,
		EscalationsCount: importResult.Imported.Escalations,
		PipelinesCount:   importResult.Imported.Pipelines,
		ProjectsCount:    importResult.Imported.Projects,
	}
	g.recordSyncHistory("pull", stats)

	return result, nil
}

// GetStatus returns the current sync status
func (g *GitSync) GetStatus() (*SyncStatus, error) {
	status := &SyncStatus{
		Initialized: g.IsInitialized(),
		SyncDir:     g.syncDir,
	}

	if !status.Initialized {
		return status, nil
	}

	// Get remote
	remote, err := g.GetRemote()
	if err == nil {
		status.Remote = remote
	}

	// Get current branch
	branch, err := g.getCurrentBranch()
	if err == nil {
		status.Branch = branch
	}

	// Check local changes
	localStatus, err := g.runGit("status", "--porcelain")
	if err == nil {
		status.HasLocalChanges = strings.TrimSpace(localStatus) != ""
	}

	// Check if behind/ahead of remote
	if g.HasRemote() {
		g.runGit("fetch", "origin")

		aheadBehind, err := g.runGit("rev-list", "--left-right", "--count", fmt.Sprintf("origin/%s...HEAD", status.Branch))
		if err == nil {
			parts := strings.Fields(aheadBehind)
			if len(parts) == 2 {
				fmt.Sscanf(parts[0], "%d", &status.Behind)
				fmt.Sscanf(parts[1], "%d", &status.Ahead)
			}
		}
	}

	// Get last sync info from data file
	if _, err := os.Stat(g.dataFile); err == nil {
		info, _ := os.Stat(g.dataFile)
		status.LastSyncFile = g.dataFile
		status.LastSyncTime = info.ModTime()
	}

	return status, nil
}

// getCurrentBranch returns the current git branch name
func (g *GitSync) getCurrentBranch() (string, error) {
	output, err := g.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// runGit executes a git command in the sync directory
func (g *GitSync) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.syncDir

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// recordSyncHistory records sync operation in database
func (g *GitSync) recordSyncHistory(direction string, stats SyncStats) {
	currentEnv, _ := g.envSvc.Current()
	envID := ""
	if currentEnv != nil {
		envID = currentEnv.ID
	}

	totalItems := stats.PortsCount + stats.SessionsCount + stats.EscalationsCount +
		stats.PipelinesCount + stats.ProjectsCount

	g.db.Exec(`
		INSERT INTO sync_history (direction, env_id, items_count, conflicts_count, synced_at)
		VALUES (?, ?, ?, 0, CURRENT_TIMESTAMP)
	`, direction, envID, totalItems)
}

// PushResult represents the result of a push operation
type PushResult struct {
	ExportedStats   SyncStats `json:"exported_stats"`
	NothingToCommit bool      `json:"nothing_to_commit"`
	Committed       bool      `json:"committed"`
	Pushed          bool      `json:"pushed"`
	NoRemote        bool      `json:"no_remote"`
	PushError       string    `json:"push_error,omitempty"`
}

// PullResult represents the result of a pull operation
type PullResult struct {
	Pulled          bool          `json:"pulled"`
	NoRemote        bool          `json:"no_remote"`
	NoData          bool          `json:"no_data"`
	HasConflict     bool          `json:"has_conflict"`
	ConflictMessage string        `json:"conflict_message,omitempty"`
	PullError       string        `json:"pull_error,omitempty"`
	ImportResult    *ImportResult `json:"import_result,omitempty"`
}

// SyncStatus represents the current sync status
type SyncStatus struct {
	Initialized     bool      `json:"initialized"`
	SyncDir         string    `json:"sync_dir"`
	Remote          string    `json:"remote,omitempty"`
	Branch          string    `json:"branch,omitempty"`
	HasLocalChanges bool      `json:"has_local_changes"`
	Ahead           int       `json:"ahead"`
	Behind          int       `json:"behind"`
	LastSyncFile    string    `json:"last_sync_file,omitempty"`
	LastSyncTime    time.Time `json:"last_sync_time,omitempty"`
}
