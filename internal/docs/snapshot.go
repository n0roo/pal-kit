package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Snapshot represents a document snapshot
type Snapshot struct {
	ID        string            `json:"id"`
	CreatedAt time.Time         `json:"created_at"`
	Message   string            `json:"message"`
	Documents []SnapshotDoc     `json:"documents"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SnapshotDoc represents a document in a snapshot
type SnapshotDoc struct {
	RelativePath string `json:"relative_path"`
	Hash         string `json:"hash"`
	Size         int64  `json:"size"`
}

// SnapshotDiff represents differences between snapshots
type SnapshotDiff struct {
	Added    []string `json:"added"`
	Modified []string `json:"modified"`
	Deleted  []string `json:"deleted"`
}

// CreateSnapshot creates a new snapshot of all documents
func (s *Service) CreateSnapshot(message string) (*Snapshot, error) {
	docs, err := s.List()
	if err != nil {
		return nil, err
	}

	snapshot := &Snapshot{
		ID:        time.Now().Format("20060102-150405"),
		CreatedAt: time.Now(),
		Message:   message,
		Documents: make([]SnapshotDoc, 0, len(docs)),
		Metadata:  make(map[string]string),
	}

	for _, doc := range docs {
		snapshot.Documents = append(snapshot.Documents, SnapshotDoc{
			RelativePath: doc.RelativePath,
			Hash:         doc.Hash,
			Size:         doc.Size,
		})
	}

	// 스냅샷 저장
	if err := s.saveSnapshot(snapshot); err != nil {
		return nil, err
	}

	// latest 링크 업데이트
	if err := s.updateLatestSnapshot(snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

// ListSnapshots returns all snapshots
func (s *Service) ListSnapshots() ([]Snapshot, error) {
	snapshotsDir := s.snapshotDir
	if _, err := os.Stat(snapshotsDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return nil, fmt.Errorf("스냅샷 디렉토리 읽기 실패: %w", err)
	}

	var snapshots []Snapshot
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "latest" {
			continue
		}

		snapshotFile := filepath.Join(snapshotsDir, entry.Name(), "snapshot.json")
		data, err := os.ReadFile(snapshotFile)
		if err != nil {
			continue
		}

		var snapshot Snapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	// 최신순 정렬
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

// GetSnapshot returns a specific snapshot
func (s *Service) GetSnapshot(id string) (*Snapshot, error) {
	snapshotFile := filepath.Join(s.snapshotDir, id, "snapshot.json")
	data, err := os.ReadFile(snapshotFile)
	if err != nil {
		return nil, fmt.Errorf("스냅샷 '%s'을(를) 찾을 수 없습니다", id)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("스냅샷 파싱 실패: %w", err)
	}

	return &snapshot, nil
}

// GetLatestSnapshot returns the latest snapshot
func (s *Service) GetLatestSnapshot() (*Snapshot, error) {
	latestFile := filepath.Join(s.snapshotDir, "latest", "snapshot.json")
	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("최신 스냅샷이 없습니다")
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("스냅샷 파싱 실패: %w", err)
	}

	return &snapshot, nil
}

// DiffWithLatest returns differences from the latest snapshot
func (s *Service) DiffWithLatest() (*SnapshotDiff, error) {
	latest, err := s.GetLatestSnapshot()
	if err != nil {
		// 스냅샷이 없으면 모든 문서가 새 문서
		docs, err := s.List()
		if err != nil {
			return nil, err
		}

		diff := &SnapshotDiff{
			Added: make([]string, 0, len(docs)),
		}
		for _, doc := range docs {
			diff.Added = append(diff.Added, doc.RelativePath)
		}
		return diff, nil
	}

	return s.DiffWithSnapshot(latest.ID)
}

// DiffWithSnapshot returns differences from a specific snapshot
func (s *Service) DiffWithSnapshot(snapshotID string) (*SnapshotDiff, error) {
	snapshot, err := s.GetSnapshot(snapshotID)
	if err != nil {
		return nil, err
	}

	currentDocs, err := s.List()
	if err != nil {
		return nil, err
	}

	// 스냅샷 문서를 맵으로
	snapshotMap := make(map[string]string)
	for _, doc := range snapshot.Documents {
		snapshotMap[doc.RelativePath] = doc.Hash
	}

	// 현재 문서를 맵으로
	currentMap := make(map[string]string)
	for _, doc := range currentDocs {
		currentMap[doc.RelativePath] = doc.Hash
	}

	diff := &SnapshotDiff{
		Added:    []string{},
		Modified: []string{},
		Deleted:  []string{},
	}

	// 추가/수정된 문서
	for path, hash := range currentMap {
		if oldHash, exists := snapshotMap[path]; !exists {
			diff.Added = append(diff.Added, path)
		} else if oldHash != hash {
			diff.Modified = append(diff.Modified, path)
		}
	}

	// 삭제된 문서
	for path := range snapshotMap {
		if _, exists := currentMap[path]; !exists {
			diff.Deleted = append(diff.Deleted, path)
		}
	}

	return diff, nil
}

// RestoreSnapshot restores documents from a snapshot
func (s *Service) RestoreSnapshot(snapshotID string, paths []string) error {
	snapshotDir := filepath.Join(s.snapshotDir, snapshotID, "files")

	// paths가 비어있으면 모든 파일 복원
	if len(paths) == 0 {
		snapshot, err := s.GetSnapshot(snapshotID)
		if err != nil {
			return err
		}
		for _, doc := range snapshot.Documents {
			paths = append(paths, doc.RelativePath)
		}
	}

	for _, relPath := range paths {
		srcPath := filepath.Join(snapshotDir, relPath)
		dstPath := filepath.Join(s.projectRoot, relPath)

		// 소스 파일 읽기
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("스냅샷 파일 읽기 실패 %s: %w", relPath, err)
		}

		// 디렉토리 생성
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("디렉토리 생성 실패: %w", err)
		}

		// 파일 복원
		if err := os.WriteFile(dstPath, content, 0644); err != nil {
			return fmt.Errorf("파일 복원 실패 %s: %w", relPath, err)
		}
	}

	return nil
}

// saveSnapshot saves a snapshot to disk
func (s *Service) saveSnapshot(snapshot *Snapshot) error {
	snapshotDir := filepath.Join(s.snapshotDir, snapshot.ID)

	// 디렉토리 생성
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("스냅샷 디렉토리 생성 실패: %w", err)
	}

	// 문서 파일 복사
	filesDir := filepath.Join(snapshotDir, "files")
	for _, doc := range snapshot.Documents {
		srcPath := filepath.Join(s.projectRoot, doc.RelativePath)
		dstPath := filepath.Join(filesDir, doc.RelativePath)

		// 디렉토리 생성
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		// 파일 복사
		content, err := os.ReadFile(srcPath)
		if err != nil {
			continue // 파일이 없으면 스킵
		}
		if err := os.WriteFile(dstPath, content, 0644); err != nil {
			return err
		}

		// 해시 파일 저장
		hashPath := filepath.Join(snapshotDir, doc.RelativePath+".hash")
		if err := os.MkdirAll(filepath.Dir(hashPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(hashPath, []byte(doc.Hash), 0644); err != nil {
			return err
		}
	}

	// 메타데이터 저장
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("스냅샷 JSON 생성 실패: %w", err)
	}

	metaPath := filepath.Join(snapshotDir, "snapshot.json")
	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("스냅샷 메타데이터 저장 실패: %w", err)
	}

	return nil
}

// updateLatestSnapshot updates the latest symlink/copy
func (s *Service) updateLatestSnapshot(snapshot *Snapshot) error {
	latestDir := filepath.Join(s.snapshotDir, "latest")
	snapshotDir := filepath.Join(s.snapshotDir, snapshot.ID)

	// latest 디렉토리 삭제 후 재생성
	os.RemoveAll(latestDir)

	// 파일 복사 (심볼릭 링크 대신)
	return copyDir(snapshotDir, latestDir)
}

// copyDir copies a directory recursively
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, content, info.Mode())
	})
}

// GetDocumentHistory returns the history of a document across snapshots
func (s *Service) GetDocumentHistory(relPath string) ([]SnapshotDoc, error) {
	snapshots, err := s.ListSnapshots()
	if err != nil {
		return nil, err
	}

	var history []SnapshotDoc
	var lastHash string

	// 오래된 순으로 처리
	for i := len(snapshots) - 1; i >= 0; i-- {
		snapshot := snapshots[i]
		for _, doc := range snapshot.Documents {
			if doc.RelativePath == relPath || strings.HasSuffix(doc.RelativePath, relPath) {
				if doc.Hash != lastHash {
					history = append(history, doc)
					lastHash = doc.Hash
				}
				break
			}
		}
	}

	return history, nil
}
