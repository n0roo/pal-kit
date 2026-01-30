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

// RegisterResult represents the result of a document registration
type RegisterResult struct {
	Status       string `json:"status"`       // "registered", "duplicate", "error"
	KBPath       string `json:"kb_path"`       // Path in KB vault
	ExistingPath string `json:"existing_path"` // If duplicate, the existing path
	Message      string `json:"message"`
}

// RegisterService handles document registration to KB
type RegisterService struct {
	vaultPath   string
	projectPath string
	indexSvc    *IndexService
}

// NewRegisterService creates a new register service
func NewRegisterService(vaultPath, projectPath string) *RegisterService {
	return &RegisterService{
		vaultPath:   vaultPath,
		projectPath: projectPath,
		indexSvc:    NewIndexService(vaultPath),
	}
}

// RegisterFromProject registers a project document to KB vault
func (s *RegisterService) RegisterFromProject(sourcePath, targetSection, targetPath string) (*RegisterResult, error) {
	// Resolve source file
	fullSourcePath := sourcePath
	if !filepath.IsAbs(sourcePath) {
		fullSourcePath = filepath.Join(s.projectPath, sourcePath)
	}

	// Check source exists
	if _, err := os.Stat(fullSourcePath); os.IsNotExist(err) {
		return &RegisterResult{
			Status:  "error",
			Message: fmt.Sprintf("소스 파일을 찾을 수 없습니다: %s", sourcePath),
		}, nil
	}

	// Calculate content hash
	hash, err := fileContentHash(fullSourcePath)
	if err != nil {
		return nil, fmt.Errorf("해시 계산 실패: %w", err)
	}

	// Determine target path in vault
	if targetPath == "" {
		fileName := filepath.Base(sourcePath)
		targetPath = filepath.Join(targetSection, fileName)
	}
	fullTargetPath := filepath.Join(s.vaultPath, targetPath)

	// Check for duplicate by path
	if _, err := os.Stat(fullTargetPath); err == nil {
		existingHash, _ := fileContentHash(fullTargetPath)
		if existingHash == hash {
			return &RegisterResult{
				Status:       "duplicate",
				KBPath:       targetPath,
				ExistingPath: targetPath,
				Message:      "이미 등록된 문서입니다 (동일한 내용)",
			}, nil
		}
		// Same path but different content
		return &RegisterResult{
			Status:       "duplicate",
			KBPath:       targetPath,
			ExistingPath: targetPath,
			Message:      "같은 경로에 다른 내용의 문서가 존재합니다",
		}, nil
	}

	// Check for duplicate by content hash in index DB
	if err := s.indexSvc.Open(); err == nil {
		defer s.indexSvc.Close()

		var existingPath string
		err := s.indexSvc.db.QueryRow(
			"SELECT path FROM documents WHERE content_hash = ?", hash,
		).Scan(&existingPath)
		if err == nil && existingPath != "" {
			return &RegisterResult{
				Status:       "duplicate",
				KBPath:       targetPath,
				ExistingPath: existingPath,
				Message:      fmt.Sprintf("동일한 내용의 문서가 이미 존재합니다: %s", existingPath),
			}, nil
		}
	}

	// Copy file to vault
	if err := os.MkdirAll(filepath.Dir(fullTargetPath), 0755); err != nil {
		return nil, fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	if err := copyFileContent(fullSourcePath, fullTargetPath); err != nil {
		return nil, fmt.Errorf("파일 복사 실패: %w", err)
	}

	// Re-index the new document
	if s.indexSvc.db != nil {
		s.indexSvc.indexDocument(fullTargetPath)
	}

	return &RegisterResult{
		Status:  "registered",
		KBPath:  targetPath,
		Message: fmt.Sprintf("KB에 등록되었습니다: %s", targetPath),
	}, nil
}

// RegisterExternal registers an external document (new content) to KB
func (s *RegisterService) RegisterExternal(title, content, targetSection, docType string, tags []string) (*RegisterResult, error) {
	// Generate file name from title
	fileName := strings.ToLower(title)
	fileName = strings.ReplaceAll(fileName, " ", "-")
	// Remove special characters but keep Korean
	var cleaned []rune
	for _, r := range fileName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || (r >= 0xAC00 && r <= 0xD7AF) {
			cleaned = append(cleaned, r)
		}
	}
	fileName = string(cleaned)
	if fileName == "" {
		fileName = "untitled"
	}
	fileName += ".md"

	targetPath := filepath.Join(targetSection, fileName)
	fullTargetPath := filepath.Join(s.vaultPath, targetPath)

	// Check for duplicate path
	if _, err := os.Stat(fullTargetPath); err == nil {
		return &RegisterResult{
			Status:       "duplicate",
			KBPath:       targetPath,
			ExistingPath: targetPath,
			Message:      "같은 경로에 문서가 이미 존재합니다",
		}, nil
	}

	// Build content with frontmatter
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: \"%s\"\n", title))
	if docType != "" {
		sb.WriteString(fmt.Sprintf("type: %s\n", docType))
	}
	sb.WriteString("status: draft\n")
	if len(tags) > 0 {
		sb.WriteString(fmt.Sprintf("tags: [%s]\n", strings.Join(tags, ", ")))
	}
	sb.WriteString(fmt.Sprintf("created: \"%s\"\n", time.Now().Format("2006-01-02")))
	sb.WriteString("---\n\n")

	// If content doesn't start with a heading, add one
	if !strings.HasPrefix(strings.TrimSpace(content), "#") {
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	}
	sb.WriteString(content)

	// Write file
	if err := os.MkdirAll(filepath.Dir(fullTargetPath), 0755); err != nil {
		return nil, fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	if err := os.WriteFile(fullTargetPath, []byte(sb.String()), 0644); err != nil {
		return nil, fmt.Errorf("파일 작성 실패: %w", err)
	}

	// Index the new document
	if err := s.indexSvc.Open(); err == nil {
		defer s.indexSvc.Close()
		s.indexSvc.indexDocument(fullTargetPath)
	}

	return &RegisterResult{
		Status:  "registered",
		KBPath:  targetPath,
		Message: fmt.Sprintf("KB에 등록되었습니다: %s", targetPath),
	}, nil
}

func fileContentHash(path string) (string, error) {
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

func copyFileContent(src, dst string) error {
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

// GetContentHash returns the content_hash for a given file path in the vault
// This uses the yaml frontmatter-based approach (not DB)
func GetContentHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	content := string(data)
	// Try extracting from frontmatter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			var fm map[string]interface{}
			if yaml.Unmarshal([]byte(parts[1]), &fm) == nil {
				if hash, ok := fm["content_hash"].(string); ok {
					return hash, nil
				}
			}
		}
	}

	// Calculate hash
	return fileContentHash(filePath)
}
