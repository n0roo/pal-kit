package document

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/db"
)

// Document represents an indexed document
type Document struct {
	ID          string
	Path        string
	Type        string
	Domain      string
	Status      string
	Priority    string
	Tokens      int64
	Summary     sql.NullString
	ContentHash string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Tags        []string
}

// Link represents a document link
type Link struct {
	FromID   string
	ToID     string
	LinkType string
}

// Service handles document operations
type Service struct {
	db          *db.DB
	projectRoot string
}

// NewService creates a new document service
func NewService(database *db.DB, projectRoot string) *Service {
	return &Service{
		db:          database,
		projectRoot: projectRoot,
	}
}

// IndexResult contains results from indexing operation
type IndexResult struct {
	Added   int
	Updated int
	Removed int
	Errors  []string
}

// Index scans and indexes documents in the project
func (s *Service) Index() (*IndexResult, error) {
	result := &IndexResult{}

	// 1. 스캔할 디렉토리 목록
	scanPaths := []struct {
		dir      string
		docType  string
		patterns []string
	}{
		{"ports", "port", []string{"*.md"}},
		{"conventions", "convention", []string{"*.md"}},
		{"agents", "agent", []string{"*.yaml", "*.yml"}},
		{"docs", "docs", []string{"*.md"}},
		{".pal/sessions", "session", []string{"*.md"}},
		{".pal/decisions", "adr", []string{"*.md"}},
	}

	existingDocs := make(map[string]bool)

	// 2. 각 경로 스캔
	for _, sp := range scanPaths {
		dir := filepath.Join(s.projectRoot, sp.dir)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		for _, pattern := range sp.patterns {
			files, err := filepath.Glob(filepath.Join(dir, "**", pattern))
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("glob error: %v", err))
				continue
			}

			// 직접 스캔 (Glob이 ** 지원 안 할 수 있음)
			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}

				matched, _ := filepath.Match(pattern, info.Name())
				if !matched {
					return nil
				}

				files = append(files, path)
				return nil
			})

			// 중복 제거
			seen := make(map[string]bool)
			for _, f := range files {
				if seen[f] {
					continue
				}
				seen[f] = true

				relPath, _ := filepath.Rel(s.projectRoot, f)
				existingDocs[relPath] = true

				added, updated, err := s.indexFile(f, sp.docType)
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", relPath, err))
					continue
				}

				if added {
					result.Added++
				}
				if updated {
					result.Updated++
				}
			}
		}
	}

	// 3. 삭제된 문서 정리
	removed, err := s.cleanupDeleted(existingDocs)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("cleanup: %v", err))
	}
	result.Removed = removed

	return result, nil
}

// indexFile indexes a single file
func (s *Service) indexFile(path string, docType string) (added, updated bool, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, false, err
	}

	// 해시 계산
	hash := md5.Sum(content)
	contentHash := hex.EncodeToString(hash[:])

	// 상대 경로
	relPath, _ := filepath.Rel(s.projectRoot, path)

	// ID 생성 (경로 기반)
	id := strings.ReplaceAll(relPath, "/", "-")
	id = strings.TrimSuffix(id, filepath.Ext(id))

	// 기존 문서 확인
	var existingHash string
	err = s.db.QueryRow(`SELECT content_hash FROM documents WHERE path = ?`, relPath).Scan(&existingHash)

	if err == sql.ErrNoRows {
		// 새 문서
		meta := s.parseMetadata(string(content), docType)
		tokens := s.estimateTokens(string(content))

		_, err = s.db.Exec(`
			INSERT INTO documents (id, path, type, domain, status, priority, tokens, content_hash, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, id, relPath, meta.Type, meta.Domain, meta.Status, meta.Priority, tokens, contentHash)

		if err != nil {
			return false, false, err
		}

		// 태그 저장
		for _, tag := range meta.Tags {
			s.db.Exec(`INSERT OR IGNORE INTO document_tags (document_id, tag) VALUES (?, ?)`, id, tag)
		}

		return true, false, nil
	} else if err != nil {
		return false, false, err
	}

	// 해시 비교
	if existingHash == contentHash {
		return false, false, nil
	}

	// 업데이트 필요
	meta := s.parseMetadata(string(content), docType)
	tokens := s.estimateTokens(string(content))

	_, err = s.db.Exec(`
		UPDATE documents
		SET type = ?, domain = ?, status = ?, priority = ?, tokens = ?, content_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE path = ?
	`, meta.Type, meta.Domain, meta.Status, meta.Priority, tokens, contentHash, relPath)

	if err != nil {
		return false, false, err
	}

	// 태그 업데이트
	s.db.Exec(`DELETE FROM document_tags WHERE document_id = ?`, id)
	for _, tag := range meta.Tags {
		s.db.Exec(`INSERT OR IGNORE INTO document_tags (document_id, tag) VALUES (?, ?)`, id, tag)
	}

	return false, true, nil
}

type docMetadata struct {
	Type     string
	Domain   string
	Status   string
	Priority string
	Tags     []string
}

// parseMetadata extracts metadata from document content
func (s *Service) parseMetadata(content, defaultType string) docMetadata {
	meta := docMetadata{
		Type:   defaultType,
		Status: "active",
	}

	// YAML frontmatter 파싱
	if strings.HasPrefix(content, "---") {
		endIdx := strings.Index(content[3:], "---")
		if endIdx > 0 {
			frontmatter := content[3 : 3+endIdx]
			meta = s.parseFrontmatter(frontmatter, meta)
		}
	}

	// 마크다운 메타데이터 테이블 파싱
	tableRe := regexp.MustCompile(`\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|`)
	matches := tableRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		key := strings.TrimSpace(strings.ToLower(m[1]))
		value := strings.TrimSpace(m[2])

		switch key {
		case "상태", "status":
			meta.Status = value
		case "우선순위", "priority":
			meta.Priority = value
		case "도메인", "domain":
			meta.Domain = value
		case "타입", "type":
			meta.Type = value
		}
	}

	// pal 상태 마커 파싱
	statusRe := regexp.MustCompile(`<!-- pal:port:status=(\w+) -->`)
	if match := statusRe.FindStringSubmatch(content); len(match) > 1 {
		meta.Status = match[1]
	}

	// 태그 파싱
	tagRe := regexp.MustCompile(`#(\w+)`)
	tagMatches := tagRe.FindAllStringSubmatch(content, -1)
	for _, m := range tagMatches {
		meta.Tags = append(meta.Tags, m[1])
	}

	return meta
}

func (s *Service) parseFrontmatter(fm string, meta docMetadata) docMetadata {
	lines := strings.Split(fm, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "type":
			meta.Type = value
		case "domain":
			meta.Domain = value
		case "status":
			meta.Status = value
		case "priority":
			meta.Priority = value
		case "tags":
			// YAML array 처리
			value = strings.Trim(value, "[]")
			for _, tag := range strings.Split(value, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					meta.Tags = append(meta.Tags, tag)
				}
			}
		}
	}
	return meta
}

// estimateTokens estimates token count (rough: ~4 chars per token)
func (s *Service) estimateTokens(content string) int64 {
	return int64(len(content) / 4)
}

// cleanupDeleted removes documents that no longer exist
func (s *Service) cleanupDeleted(existing map[string]bool) (int, error) {
	rows, err := s.db.Query(`SELECT id, path FROM documents`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var toDelete []string
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			continue
		}

		if !existing[path] {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		s.db.Exec(`DELETE FROM documents WHERE id = ?`, id)
	}

	return len(toDelete), nil
}

// Search searches documents with filters
func (s *Service) Search(query string, filters SearchFilters) ([]Document, error) {
	sqlQuery := `
		SELECT d.id, d.path, d.type, d.domain, d.status, d.priority, d.tokens, d.summary, d.content_hash, d.created_at, d.updated_at
		FROM documents d
		WHERE 1=1
	`
	args := []interface{}{}

	// 타입 필터
	if filters.Type != "" {
		sqlQuery += ` AND d.type = ?`
		args = append(args, filters.Type)
	}

	// 도메인 필터
	if filters.Domain != "" {
		sqlQuery += ` AND d.domain = ?`
		args = append(args, filters.Domain)
	}

	// 상태 필터
	if filters.Status != "" {
		sqlQuery += ` AND d.status = ?`
		args = append(args, filters.Status)
	}

	// 태그 필터
	if filters.Tag != "" {
		sqlQuery += ` AND d.id IN (SELECT document_id FROM document_tags WHERE tag = ?)`
		args = append(args, filters.Tag)
	}

	// 텍스트 검색 (경로, ID에서)
	if query != "" {
		sqlQuery += ` AND (d.path LIKE ? OR d.id LIKE ?)`
		pattern := "%" + query + "%"
		args = append(args, pattern, pattern)
	}

	// 토큰 제한
	if filters.MaxTokens > 0 {
		sqlQuery += ` AND d.tokens <= ?`
		args = append(args, filters.MaxTokens)
	}

	// 정렬
	sqlQuery += ` ORDER BY d.updated_at DESC`

	// 제한
	if filters.Limit > 0 {
		sqlQuery += fmt.Sprintf(` LIMIT %d`, filters.Limit)
	}

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("검색 실패: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(
			&d.ID, &d.Path, &d.Type, &d.Domain, &d.Status, &d.Priority,
			&d.Tokens, &d.Summary, &d.ContentHash, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}

	return docs, nil
}

// SearchFilters contains search filter options
type SearchFilters struct {
	Type      string
	Domain    string
	Status    string
	Tag       string
	MaxTokens int64
	Limit     int
}

// Get retrieves a document by ID
func (s *Service) Get(id string) (*Document, error) {
	var d Document
	err := s.db.QueryRow(`
		SELECT id, path, type, domain, status, priority, tokens, summary, content_hash, created_at, updated_at
		FROM documents WHERE id = ?
	`, id).Scan(
		&d.ID, &d.Path, &d.Type, &d.Domain, &d.Status, &d.Priority,
		&d.Tokens, &d.Summary, &d.ContentHash, &d.CreatedAt, &d.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("문서 '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	// 태그 로드
	rows, err := s.db.Query(`SELECT tag FROM document_tags WHERE document_id = ?`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tag string
			if rows.Scan(&tag) == nil {
				d.Tags = append(d.Tags, tag)
			}
		}
	}

	return &d, nil
}

// GetByPath retrieves a document by path
func (s *Service) GetByPath(path string) (*Document, error) {
	var d Document
	err := s.db.QueryRow(`
		SELECT id, path, type, domain, status, priority, tokens, summary, content_hash, created_at, updated_at
		FROM documents WHERE path = ?
	`, path).Scan(
		&d.ID, &d.Path, &d.Type, &d.Domain, &d.Status, &d.Priority,
		&d.Tokens, &d.Summary, &d.ContentHash, &d.CreatedAt, &d.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("문서 '%s'을(를) 찾을 수 없습니다", path)
	}
	if err != nil {
		return nil, err
	}

	return &d, nil
}

// GetContent reads and returns the document content
func (s *Service) GetContent(id string) (string, error) {
	doc, err := s.Get(id)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(s.projectRoot, doc.Path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("파일 읽기 실패: %w", err)
	}

	return string(content), nil
}

// FindPort finds a port document by name or alias
func (s *Service) FindPort(name string) (*Document, error) {
	// 정확히 일치
	docs, err := s.Search(name, SearchFilters{Type: "port", Limit: 10})
	if err != nil {
		return nil, err
	}

	for _, d := range docs {
		// ID 일치
		if d.ID == name || d.ID == "ports-"+name {
			return &d, nil
		}

		// 경로 기반 일치
		baseName := strings.TrimSuffix(filepath.Base(d.Path), ".md")
		if baseName == name {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("포트 '%s'을(를) 찾을 수 없습니다", name)
}

// DocStats represents document statistics
type DocStats struct {
	TotalDocs   int            `json:"total_docs"`
	TotalTokens int64          `json:"total_tokens"`
	ByType      map[string]int `json:"by_type"`
	ByStatus    map[string]int `json:"by_status"`
	ByDomain    map[string]int `json:"by_domain"`
}

// GetStats returns document statistics
func (s *Service) GetStats() (*DocStats, error) {
	stats := &DocStats{
		ByType:   make(map[string]int),
		ByStatus: make(map[string]int),
		ByDomain: make(map[string]int),
	}

	// 총 개수 및 토큰
	err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(tokens), 0) FROM documents`).Scan(&stats.TotalDocs, &stats.TotalTokens)
	if err != nil {
		return nil, err
	}

	// 타입별
	rows, err := s.db.Query(`SELECT COALESCE(type, 'unknown'), COUNT(*) FROM documents GROUP BY type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var c int
		if rows.Scan(&t, &c) == nil {
			stats.ByType[t] = c
		}
	}

	// 상태별
	rows, err = s.db.Query(`SELECT COALESCE(status, 'unknown'), COUNT(*) FROM documents GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		var c int
		if rows.Scan(&s, &c) == nil {
			stats.ByStatus[s] = c
		}
	}

	// 도메인별
	rows, err = s.db.Query(`SELECT COALESCE(domain, 'unknown'), COUNT(*) FROM documents WHERE domain IS NOT NULL AND domain != '' GROUP BY domain`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		var c int
		if rows.Scan(&d, &c) == nil {
			stats.ByDomain[d] = c
		}
	}

	return stats, nil
}

// GetRelatedDocs finds related documents for context loading
func (s *Service) GetRelatedDocs(portPath string, tokenBudget int64) ([]Document, error) {
	// 1. 포트 문서 로드
	port, err := s.GetByPath(portPath)
	if err != nil {
		return nil, err
	}

	var docs []Document
	var totalTokens int64

	// 2. 같은 도메인의 L1 문서 검색
	if port.Domain != "" {
		l1Docs, _ := s.Search("", SearchFilters{Type: "l1", Domain: port.Domain, Limit: 5})
		for _, d := range l1Docs {
			if totalTokens+d.Tokens <= tokenBudget {
				docs = append(docs, d)
				totalTokens += d.Tokens
			}
		}
	}

	// 3. 관련 컨벤션 검색
	conventions, _ := s.Search("", SearchFilters{Type: "convention", Limit: 10})
	for _, d := range conventions {
		if totalTokens+d.Tokens <= tokenBudget {
			docs = append(docs, d)
			totalTokens += d.Tokens
		}
	}

	return docs, nil
}

// AddLink adds a link between two documents
func (s *Service) AddLink(fromID, toID, linkType string) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO document_links (from_id, to_id, link_type)
		VALUES (?, ?, ?)
	`, fromID, toID, linkType)
	return err
}

// GetLinksFrom returns documents that the given document links to
func (s *Service) GetLinksFrom(docID string) ([]Document, error) {
	rows, err := s.db.Query(`
		SELECT d.id, d.path, d.type, d.domain, d.status, d.priority, d.tokens, d.summary, d.content_hash, d.created_at, d.updated_at
		FROM documents d
		JOIN document_links l ON d.id = l.to_id
		WHERE l.from_id = ?
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(
			&d.ID, &d.Path, &d.Type, &d.Domain, &d.Status, &d.Priority,
			&d.Tokens, &d.Summary, &d.ContentHash, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}

	return docs, nil
}

// GetLinksTo returns documents that link to the given document
func (s *Service) GetLinksTo(docID string) ([]Document, error) {
	rows, err := s.db.Query(`
		SELECT d.id, d.path, d.type, d.domain, d.status, d.priority, d.tokens, d.summary, d.content_hash, d.created_at, d.updated_at
		FROM documents d
		JOIN document_links l ON d.id = l.from_id
		WHERE l.to_id = ?
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(
			&d.ID, &d.Path, &d.Type, &d.Domain, &d.Status, &d.Priority,
			&d.Tokens, &d.Summary, &d.ContentHash, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}

	return docs, nil
}

// ListByType returns all documents of a specific type
func (s *Service) ListByType(docType string) ([]Document, error) {
	return s.Search("", SearchFilters{Type: docType})
}

// RefreshDocument re-indexes a single document
func (s *Service) RefreshDocument(path string) error {
	fullPath := filepath.Join(s.projectRoot, path)

	// 파일이 존재하는지 확인
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// 삭제된 경우
		_, err := s.db.Exec(`DELETE FROM documents WHERE path = ?`, path)
		return err
	}

	// 타입 추론
	docType := "unknown"
	if strings.HasPrefix(path, "ports/") {
		docType = "port"
	} else if strings.HasPrefix(path, "conventions/") {
		docType = "convention"
	} else if strings.HasPrefix(path, "agents/") {
		docType = "agent"
	} else if strings.HasPrefix(path, "docs/") {
		docType = "docs"
	}

	_, _, err := s.indexFile(fullPath, docType)
	return err
}

// Close performs cleanup (if needed)
func (s *Service) Close() error {
	return nil
}

// ReadDocumentContent reads a document file and returns its content
func ReadDocumentContent(projectRoot, docPath string) (string, error) {
	fullPath := filepath.Join(projectRoot, docPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// EstimateTokens estimates token count for given content
func EstimateTokens(content string) int64 {
	return int64(len(content) / 4)
}

// ParseQueryString parses a query string like "type:l1 AND domain:order"
func ParseQueryString(query string) SearchFilters {
	filters := SearchFilters{}

	// type:xxx
	typeRe := regexp.MustCompile(`type:(\w+)`)
	if match := typeRe.FindStringSubmatch(query); len(match) > 1 {
		filters.Type = match[1]
	}

	// domain:xxx
	domainRe := regexp.MustCompile(`domain:(\w+)`)
	if match := domainRe.FindStringSubmatch(query); len(match) > 1 {
		filters.Domain = match[1]
	}

	// status:xxx
	statusRe := regexp.MustCompile(`status:(\w+)`)
	if match := statusRe.FindStringSubmatch(query); len(match) > 1 {
		filters.Status = match[1]
	}

	// tag:xxx
	tagRe := regexp.MustCompile(`tag:(\w+)`)
	if match := tagRe.FindStringSubmatch(query); len(match) > 1 {
		filters.Tag = match[1]
	}

	return filters
}

// CleanQueryString removes filter patterns from query, returning remaining text for full-text search
func CleanQueryString(query string) string {
	// 필터 패턴 제거
	patterns := []string{
		`type:\w+`,
		`domain:\w+`,
		`status:\w+`,
		`tag:\w+`,
		`\bAND\b`,
		`\bOR\b`,
		`\bNOT\b`,
	}

	result := query
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		result = re.ReplaceAllString(result, "")
	}

	// 공백 정리
	result = strings.TrimSpace(result)
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

	return result
}

// unused but keeping io import for potential future use
var _ = io.EOF
