package docs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DocType represents the type of document
type DocType string

const (
	DocTypeClaude    DocType = "claude"    // CLAUDE.md
	DocTypeAgent     DocType = "agent"     // agents/*.yaml, agents/*.md
	DocTypePort      DocType = "port"      // ports/*.md
	DocTypeRule      DocType = "rule"      // .claude/rules/*.md
	DocTypeTemplate  DocType = "template"  // templates/*.md
	DocTypeConvention DocType = "convention" // conventions/*.md
	DocTypeOther     DocType = "other"
)

// DocStatus represents the status of a document
type DocStatus string

const (
	StatusValid    DocStatus = "valid"    // 검증 통과
	StatusModified DocStatus = "modified" // 수정됨 (스냅샷 이후)
	StatusOutdated DocStatus = "outdated" // 오래됨
	StatusInvalid  DocStatus = "invalid"  // 검증 실패
	StatusNew      DocStatus = "new"      // 새 문서
)

// Document represents a managed document
type Document struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	Type         DocType   `json:"type"`
	Status       DocStatus `json:"status"`
	Hash         string    `json:"hash"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
	Issues       []string  `json:"issues,omitempty"`
}

// Service handles document operations
type Service struct {
	projectRoot string
	snapshotDir string
}

// NewService creates a new docs service
func NewService(projectRoot string) *Service {
	return &Service{
		projectRoot: projectRoot,
		snapshotDir: filepath.Join(projectRoot, ".pal", "snapshots"),
	}
}

// GetProjectRoot returns the project root
func (s *Service) GetProjectRoot() string {
	return s.projectRoot
}

// List returns all managed documents
func (s *Service) List() ([]Document, error) {
	var docs []Document

	// CLAUDE.md
	claudeMD := filepath.Join(s.projectRoot, "CLAUDE.md")
	if doc, err := s.getDocument(claudeMD, DocTypeClaude); err == nil {
		docs = append(docs, *doc)
	}

	// agents/
	agentsDir := filepath.Join(s.projectRoot, "agents")
	agentDocs, _ := s.scanDirectory(agentsDir, DocTypeAgent, []string{".yaml", ".yml", ".md"})
	docs = append(docs, agentDocs...)

	// ports/
	portsDir := filepath.Join(s.projectRoot, "ports")
	portDocs, _ := s.scanDirectory(portsDir, DocTypePort, []string{".md"})
	docs = append(docs, portDocs...)

	// .claude/rules/
	rulesDir := filepath.Join(s.projectRoot, ".claude", "rules")
	ruleDocs, _ := s.scanDirectory(rulesDir, DocTypeRule, []string{".md"})
	docs = append(docs, ruleDocs...)

	// templates/
	templatesDir := filepath.Join(s.projectRoot, "templates")
	templateDocs, _ := s.scanDirectory(templatesDir, DocTypeTemplate, []string{".md", ".yaml"})
	docs = append(docs, templateDocs...)

	// conventions/
	conventionsDir := filepath.Join(s.projectRoot, "conventions")
	convDocs, _ := s.scanDirectory(conventionsDir, DocTypeConvention, []string{".md", ".yaml"})
	docs = append(docs, convDocs...)

	return docs, nil
}

// Get returns a specific document
func (s *Service) Get(path string) (*Document, error) {
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(s.projectRoot, path)
	}

	docType := s.detectType(fullPath)
	return s.getDocument(fullPath, docType)
}

// GetContent returns the content of a document
func (s *Service) GetContent(path string) (string, error) {
	fullPath := path
	if !filepath.IsAbs(path) {
		fullPath = filepath.Join(s.projectRoot, path)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("문서 읽기 실패: %w", err)
	}

	return string(content), nil
}

// Status returns the overall status of documents
func (s *Service) Status() (map[DocStatus]int, error) {
	docs, err := s.List()
	if err != nil {
		return nil, err
	}

	status := make(map[DocStatus]int)
	for _, doc := range docs {
		status[doc.Status]++
	}

	return status, nil
}

// scanDirectory scans a directory for documents
func (s *Service) scanDirectory(dir string, docType DocType, extensions []string) ([]Document, error) {
	var docs []Document

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return docs, nil
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, allowedExt := range extensions {
			if ext == allowedExt {
				if doc, err := s.getDocument(path, docType); err == nil {
					docs = append(docs, *doc)
				}
				break
			}
		}

		return nil
	})

	return docs, err
}

// getDocument creates a Document from a file path
func (s *Service) getDocument(path string, docType DocType) (*Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	hash := s.computeHash(content)
	relPath, _ := filepath.Rel(s.projectRoot, path)

	doc := &Document{
		Path:         path,
		RelativePath: relPath,
		Type:         docType,
		Hash:         hash,
		Size:         info.Size(),
		ModifiedAt:   info.ModTime(),
	}

	// 상태 결정
	doc.Status = s.determineStatus(doc)

	return doc, nil
}

// detectType detects document type from path
func (s *Service) detectType(path string) DocType {
	relPath, _ := filepath.Rel(s.projectRoot, path)

	if strings.HasSuffix(relPath, "CLAUDE.md") {
		return DocTypeClaude
	}
	if strings.HasPrefix(relPath, "agents") {
		return DocTypeAgent
	}
	if strings.HasPrefix(relPath, "ports") {
		return DocTypePort
	}
	if strings.HasPrefix(relPath, ".claude/rules") || strings.HasPrefix(relPath, ".claude\\rules") {
		return DocTypeRule
	}
	if strings.HasPrefix(relPath, "templates") {
		return DocTypeTemplate
	}
	if strings.HasPrefix(relPath, "conventions") {
		return DocTypeConvention
	}

	return DocTypeOther
}

// determineStatus determines the status of a document
func (s *Service) determineStatus(doc *Document) DocStatus {
	// 스냅샷과 비교
	snapshotHash, err := s.getSnapshotHash(doc.RelativePath)
	if err != nil {
		return StatusNew
	}

	if snapshotHash != doc.Hash {
		return StatusModified
	}

	// 7일 이상 수정 안됨
	if time.Since(doc.ModifiedAt) > 7*24*time.Hour {
		return StatusOutdated
	}

	return StatusValid
}

// computeHash computes SHA256 hash of content
func (s *Service) computeHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// getSnapshotHash returns the hash from the latest snapshot
func (s *Service) getSnapshotHash(relPath string) (string, error) {
	snapshotFile := filepath.Join(s.snapshotDir, "latest", relPath+".hash")
	content, err := os.ReadFile(snapshotFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// EnsureDirectories creates necessary directories
func (s *Service) EnsureDirectories() error {
	dirs := []string{
		filepath.Join(s.projectRoot, "agents"),
		filepath.Join(s.projectRoot, "ports"),
		filepath.Join(s.projectRoot, "templates"),
		filepath.Join(s.projectRoot, "conventions"),
		filepath.Join(s.projectRoot, ".claude", "rules"),
		filepath.Join(s.projectRoot, ".pal", "snapshots"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("디렉토리 생성 실패 %s: %w", dir, err)
		}
	}

	return nil
}
