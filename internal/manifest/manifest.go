package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
	"gopkg.in/yaml.v3"
)

// FileType represents the type of tracked file
type FileType string

const (
	FileTypeContext    FileType = "context"    // CLAUDE.md
	FileTypeAgent      FileType = "agent"      // agents/*.yaml
	FileTypeConvention FileType = "convention" // conventions/*.yaml, *.md
	FileTypePort       FileType = "port"       // ports/*.md
	FileTypeConfig     FileType = "config"     // .pal/config.yaml
)

// FileStatus represents the status of a tracked file
type FileStatus string

const (
	StatusSynced   FileStatus = "synced"   // 동기화됨
	StatusModified FileStatus = "modified" // 변경됨
	StatusNew      FileStatus = "new"      // 새 파일
	StatusDeleted  FileStatus = "deleted"  // 삭제됨
)

// ManagedBy represents who manages the file
type ManagedBy string

const (
	ManagedByPal    ManagedBy = "pal"    // PAL Kit 생성
	ManagedByUser   ManagedBy = "user"   // 사용자 생성
	ManagedByClaude ManagedBy = "claude" // Claude 생성
)

// TrackedFile represents a file being tracked
type TrackedFile struct {
	Path      string     `yaml:"path" json:"path"`
	Type      FileType   `yaml:"type" json:"type"`
	Hash      string     `yaml:"hash" json:"hash"`
	Size      int64      `yaml:"size" json:"size"`
	MTime     time.Time  `yaml:"mtime" json:"mtime"`
	ManagedBy ManagedBy  `yaml:"managed_by" json:"managed_by"`
	Status    FileStatus `yaml:"status,omitempty" json:"status,omitempty"`
}

// Manifest represents the manifest file structure
type Manifest struct {
	Version   string                  `yaml:"version"`
	UpdatedAt time.Time               `yaml:"updated_at"`
	Files     map[string]*TrackedFile `yaml:"files"`
}

// ChangeRecord represents a file change
type ChangeRecord struct {
	ProjectRoot string     `json:"project_root"`
	FilePath    string     `json:"file_path"`
	ChangeType  string     `json:"change_type"` // created, modified, deleted
	OldHash     string     `json:"old_hash,omitempty"`
	NewHash     string     `json:"new_hash,omitempty"`
	ChangedAt   time.Time  `json:"changed_at"`
	SessionID   string     `json:"session_id,omitempty"`
}

// Service handles manifest operations
type Service struct {
	db          *db.DB
	projectRoot string
	manifestDir string
}

// NewService creates a new manifest service
func NewService(database *db.DB, projectRoot string) *Service {
	return &Service{
		db:          database,
		projectRoot: projectRoot,
		manifestDir: filepath.Join(projectRoot, ".pal"),
	}
}

// GetManifestPath returns the manifest file path
func (s *Service) GetManifestPath() string {
	return filepath.Join(s.manifestDir, "manifest.yaml")
}

// LoadManifest loads manifest from YAML file
func (s *Service) LoadManifest() (*Manifest, error) {
	path := s.GetManifestPath()
	
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 새 manifest 생성
			return &Manifest{
				Version:   "1",
				UpdatedAt: time.Now(),
				Files:     make(map[string]*TrackedFile),
			}, nil
		}
		return nil, fmt.Errorf("manifest 읽기 실패: %w", err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("manifest 파싱 실패: %w", err)
	}

	if manifest.Files == nil {
		manifest.Files = make(map[string]*TrackedFile)
	}

	return &manifest, nil
}

// SaveManifest saves manifest to YAML file
func (s *Service) SaveManifest(manifest *Manifest) error {
	// 디렉토리 생성
	if err := os.MkdirAll(s.manifestDir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	manifest.UpdatedAt = time.Now()

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("manifest 직렬화 실패: %w", err)
	}

	path := s.GetManifestPath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("manifest 저장 실패: %w", err)
	}

	return nil
}

// ComputeHash calculates SHA256 hash of a file
func (s *Service) ComputeHash(filePath string) (string, error) {
	fullPath := filepath.Join(s.projectRoot, filePath)
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

// GetFileInfo gets file information
func (s *Service) GetFileInfo(filePath string) (*TrackedFile, error) {
	fullPath := filepath.Join(s.projectRoot, filePath)

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	hash, err := s.ComputeHash(filePath)
	if err != nil {
		return nil, err
	}

	fileType := s.detectFileType(filePath)

	return &TrackedFile{
		Path:      filePath,
		Type:      fileType,
		Hash:      hash,
		Size:      info.Size(),
		MTime:     info.ModTime(),
		ManagedBy: ManagedByPal,
		Status:    StatusSynced,
	}, nil
}

// detectFileType determines file type from path
func (s *Service) detectFileType(filePath string) FileType {
	if filePath == "CLAUDE.md" {
		return FileTypeContext
	}
	if strings.HasPrefix(filePath, "agents/") {
		return FileTypeAgent
	}
	if strings.HasPrefix(filePath, "conventions/") {
		return FileTypeConvention
	}
	if strings.HasPrefix(filePath, "ports/") {
		return FileTypePort
	}
	if filePath == ".pal/config.yaml" {
		return FileTypeConfig
	}
	return FileTypeContext // default
}

// ScanTrackedFiles scans all trackable files
func (s *Service) ScanTrackedFiles() ([]string, error) {
	var files []string

	// CLAUDE.md
	if _, err := os.Stat(filepath.Join(s.projectRoot, "CLAUDE.md")); err == nil {
		files = append(files, "CLAUDE.md")
	}

	// .pal/config.yaml
	if _, err := os.Stat(filepath.Join(s.projectRoot, ".pal/config.yaml")); err == nil {
		files = append(files, ".pal/config.yaml")
	}

	// agents/
	agentFiles, _ := s.scanDirectory("agents", []string{".yaml", ".yml"})
	files = append(files, agentFiles...)

	// conventions/
	convFiles, _ := s.scanDirectory("conventions", []string{".yaml", ".yml", ".md"})
	files = append(files, convFiles...)

	// ports/
	portFiles, _ := s.scanDirectory("ports", []string{".md"})
	files = append(files, portFiles...)

	return files, nil
}

// scanDirectory scans a directory for files with given extensions
func (s *Service) scanDirectory(dir string, extensions []string) ([]string, error) {
	var files []string
	fullDir := filepath.Join(s.projectRoot, dir)

	if _, err := os.Stat(fullDir); os.IsNotExist(err) {
		return files, nil
	}

	err := filepath.WalkDir(fullDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		// 제외 패턴
		name := d.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".bak") || strings.HasSuffix(name, ".tmp") {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, allowedExt := range extensions {
			if ext == allowedExt {
				relPath, _ := filepath.Rel(s.projectRoot, path)
				files = append(files, relPath)
				break
			}
		}
		return nil
	})

	return files, err
}

// Status checks all tracked files and returns their status
func (s *Service) Status() ([]TrackedFile, error) {
	manifest, err := s.LoadManifest()
	if err != nil {
		return nil, err
	}

	// 현재 존재하는 파일들 스캔
	currentFiles, err := s.ScanTrackedFiles()
	if err != nil {
		return nil, err
	}

	var results []TrackedFile
	checkedPaths := make(map[string]bool)

	// 현재 파일들 체크
	for _, filePath := range currentFiles {
		checkedPaths[filePath] = true

		currentInfo, err := s.GetFileInfo(filePath)
		if err != nil {
			continue
		}

		tracked, exists := manifest.Files[filePath]
		if !exists {
			// 새 파일
			currentInfo.Status = StatusNew
			results = append(results, *currentInfo)
			continue
		}

		// mtime 먼저 비교 (빠른 체크)
		if !tracked.MTime.Equal(currentInfo.MTime) || tracked.Size != currentInfo.Size {
			// 변경 가능성 있음 - 해시 비교
			if tracked.Hash != currentInfo.Hash {
				currentInfo.Status = StatusModified
				currentInfo.ManagedBy = tracked.ManagedBy
			} else {
				currentInfo.Status = StatusSynced
				currentInfo.ManagedBy = tracked.ManagedBy
			}
		} else {
			currentInfo.Status = StatusSynced
			currentInfo.ManagedBy = tracked.ManagedBy
		}

		results = append(results, *currentInfo)
	}

	// 삭제된 파일 체크
	for path, tracked := range manifest.Files {
		if !checkedPaths[path] {
			tracked.Status = StatusDeleted
			results = append(results, *tracked)
		}
	}

	return results, nil
}

// QuickCheck performs a quick mtime-based change detection
func (s *Service) QuickCheck() ([]string, error) {
	manifest, err := s.LoadManifest()
	if err != nil {
		return nil, err
	}

	var changed []string

	currentFiles, err := s.ScanTrackedFiles()
	if err != nil {
		return nil, err
	}

	// 새 파일 체크
	for _, filePath := range currentFiles {
		if _, exists := manifest.Files[filePath]; !exists {
			changed = append(changed, filePath)
			continue
		}
	}

	// 변경된 파일 체크 (mtime 기반)
	for path, tracked := range manifest.Files {
		fullPath := filepath.Join(s.projectRoot, path)
		info, err := os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				changed = append(changed, path)
			}
			continue
		}

		if !tracked.MTime.Equal(info.ModTime()) || tracked.Size != info.Size() {
			changed = append(changed, path)
		}
	}

	return changed, nil
}

// Sync synchronizes manifest with current file states
func (s *Service) Sync() ([]ChangeRecord, error) {
	manifest, err := s.LoadManifest()
	if err != nil {
		return nil, err
	}

	statuses, err := s.Status()
	if err != nil {
		return nil, err
	}

	var changes []ChangeRecord

	for _, file := range statuses {
		switch file.Status {
		case StatusNew:
			manifest.Files[file.Path] = &TrackedFile{
				Path:      file.Path,
				Type:      file.Type,
				Hash:      file.Hash,
				Size:      file.Size,
				MTime:     file.MTime,
				ManagedBy: file.ManagedBy,
			}
			changes = append(changes, ChangeRecord{
				ProjectRoot: s.projectRoot,
				FilePath:    file.Path,
				ChangeType:  "created",
				NewHash:     file.Hash,
				ChangedAt:   time.Now(),
			})

		case StatusModified:
			oldHash := ""
			if old, exists := manifest.Files[file.Path]; exists {
				oldHash = old.Hash
			}
			manifest.Files[file.Path] = &TrackedFile{
				Path:      file.Path,
				Type:      file.Type,
				Hash:      file.Hash,
				Size:      file.Size,
				MTime:     file.MTime,
				ManagedBy: file.ManagedBy,
			}
			changes = append(changes, ChangeRecord{
				ProjectRoot: s.projectRoot,
				FilePath:    file.Path,
				ChangeType:  "modified",
				OldHash:     oldHash,
				NewHash:     file.Hash,
				ChangedAt:   time.Now(),
			})

		case StatusDeleted:
			oldHash := ""
			if old, exists := manifest.Files[file.Path]; exists {
				oldHash = old.Hash
			}
			delete(manifest.Files, file.Path)
			changes = append(changes, ChangeRecord{
				ProjectRoot: s.projectRoot,
				FilePath:    file.Path,
				ChangeType:  "deleted",
				OldHash:     oldHash,
				ChangedAt:   time.Now(),
			})
		}
	}

	// manifest 저장
	if err := s.SaveManifest(manifest); err != nil {
		return nil, err
	}

	// DB에 변경 사항 저장
	if err := s.saveChangesToDB(changes); err != nil {
		return nil, fmt.Errorf("DB 저장 실패: %w", err)
	}

	// DB manifest 테이블 동기화
	if err := s.syncToDB(manifest); err != nil {
		return nil, fmt.Errorf("DB 동기화 실패: %w", err)
	}

	return changes, nil
}

// saveChangesToDB saves change records to database
func (s *Service) saveChangesToDB(changes []ChangeRecord) error {
	for _, change := range changes {
		_, err := s.db.Exec(`
			INSERT INTO file_changes (project_root, file_path, change_type, old_hash, new_hash, changed_at, session_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, change.ProjectRoot, change.FilePath, change.ChangeType, change.OldHash, change.NewHash, change.ChangedAt, change.SessionID)
		if err != nil {
			return err
		}
	}
	return nil
}

// syncToDB synchronizes manifest to DB
func (s *Service) syncToDB(manifest *Manifest) error {
	// 기존 레코드 삭제
	_, err := s.db.Exec(`DELETE FROM file_manifests WHERE project_root = ?`, s.projectRoot)
	if err != nil {
		return err
	}

	// 새 레코드 삽입
	for _, file := range manifest.Files {
		_, err := s.db.Exec(`
			INSERT INTO file_manifests (project_root, file_path, file_type, hash, size, mtime, managed_by, status, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, s.projectRoot, file.Path, file.Type, file.Hash, file.Size, file.MTime, file.ManagedBy, StatusSynced, time.Now())
		if err != nil {
			return err
		}
	}

	return nil
}

// AddFile adds a file to manifest
func (s *Service) AddFile(filePath string, managedBy ManagedBy) error {
	manifest, err := s.LoadManifest()
	if err != nil {
		return err
	}

	info, err := s.GetFileInfo(filePath)
	if err != nil {
		return fmt.Errorf("파일 정보 조회 실패: %w", err)
	}

	info.ManagedBy = managedBy
	manifest.Files[filePath] = info

	if err := s.SaveManifest(manifest); err != nil {
		return err
	}

	// DB 동기화
	return s.syncToDB(manifest)
}

// RemoveFile removes a file from manifest
func (s *Service) RemoveFile(filePath string) error {
	manifest, err := s.LoadManifest()
	if err != nil {
		return err
	}

	delete(manifest.Files, filePath)

	if err := s.SaveManifest(manifest); err != nil {
		return err
	}

	// DB 동기화
	return s.syncToDB(manifest)
}

// GetChanges returns change history from DB
func (s *Service) GetChanges(limit int) ([]ChangeRecord, error) {
	rows, err := s.db.Query(`
		SELECT project_root, file_path, change_type, old_hash, new_hash, changed_at, session_id
		FROM file_changes
		WHERE project_root = ?
		ORDER BY changed_at DESC
		LIMIT ?
	`, s.projectRoot, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []ChangeRecord
	for rows.Next() {
		var c ChangeRecord
		var sessionID *string
		if err := rows.Scan(&c.ProjectRoot, &c.FilePath, &c.ChangeType, &c.OldHash, &c.NewHash, &c.ChangedAt, &sessionID); err != nil {
			continue
		}
		if sessionID != nil {
			c.SessionID = *sessionID
		}
		changes = append(changes, c)
	}

	return changes, nil
}

// Init initializes manifest for a new project
func (s *Service) Init() error {
	manifest := &Manifest{
		Version:   "1",
		UpdatedAt: time.Now(),
		Files:     make(map[string]*TrackedFile),
	}

	// CLAUDE.md 추가
	if info, err := s.GetFileInfo("CLAUDE.md"); err == nil {
		info.ManagedBy = ManagedByPal
		manifest.Files["CLAUDE.md"] = info
	}

	// 기존 파일들 스캔 및 추가
	files, err := s.ScanTrackedFiles()
	if err != nil {
		return err
	}

	for _, filePath := range files {
		if _, exists := manifest.Files[filePath]; exists {
			continue
		}
		if info, err := s.GetFileInfo(filePath); err == nil {
			info.ManagedBy = ManagedByUser
			manifest.Files[filePath] = info
		}
	}

	if err := s.SaveManifest(manifest); err != nil {
		return err
	}

	return s.syncToDB(manifest)
}
