package kb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
)

// IndexService handles document indexing
type IndexService struct {
	vaultPath string
	dbPath    string
	db        *sql.DB
}

// DocumentIndex represents an indexed document
type DocumentIndex struct {
	ID        int64    `json:"id"`
	Path      string   `json:"path"`
	Title     string   `json:"title"`
	Type      string   `json:"type,omitempty"`
	Status    string   `json:"status,omitempty"`
	Domain    string   `json:"domain,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Aliases   []string `json:"aliases,omitempty"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
	IndexedAt string   `json:"indexed_at"`
}

// SearchResult represents a search result
type SearchResult struct {
	Document   *DocumentIndex `json:"document"`
	Score      float64        `json:"score"`
	Highlights []string       `json:"highlights,omitempty"`
}

// SearchOptions represents search options
type SearchOptions struct {
	Type        string   `json:"type,omitempty"`
	Domain      string   `json:"domain,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	TokenBudget int      `json:"token_budget,omitempty"`
}

// IndexStats represents indexing statistics
type IndexStats struct {
	TotalDocs   int            `json:"total_docs"`
	ByType      map[string]int `json:"by_type"`
	ByDomain    map[string]int `json:"by_domain"`
	ByStatus    map[string]int `json:"by_status"`
	LastIndexed string         `json:"last_indexed"`
}

// NewIndexService creates a new index service
func NewIndexService(vaultPath string) *IndexService {
	return &IndexService{
		vaultPath: vaultPath,
		dbPath:    filepath.Join(vaultPath, MetaDir, "index.db"),
	}
}

// Open opens the database connection
func (s *IndexService) Open() error {
	// Ensure directory exists
	dir := filepath.Dir(s.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	db, err := sql.Open("sqlite3", s.dbPath)
	if err != nil {
		return fmt.Errorf("DB 열기 실패: %w", err)
	}
	s.db = db

	// Initialize schema
	return s.initSchema()
}

// Close closes the database connection
func (s *IndexService) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *IndexService) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS documents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		type TEXT,
		status TEXT,
		domain TEXT,
		summary TEXT,
		created_at TEXT,
		updated_at TEXT,
		indexed_at TEXT NOT NULL,
		content_hash TEXT
	);

	CREATE TABLE IF NOT EXISTS document_tags (
		doc_id INTEGER NOT NULL,
		tag TEXT NOT NULL,
		FOREIGN KEY (doc_id) REFERENCES documents(id) ON DELETE CASCADE,
		PRIMARY KEY (doc_id, tag)
	);

	CREATE TABLE IF NOT EXISTS document_aliases (
		doc_id INTEGER NOT NULL,
		alias TEXT NOT NULL,
		FOREIGN KEY (doc_id) REFERENCES documents(id) ON DELETE CASCADE,
		PRIMARY KEY (doc_id, alias)
	);

	CREATE INDEX IF NOT EXISTS idx_documents_type ON documents(type);
	CREATE INDEX IF NOT EXISTS idx_documents_domain ON documents(domain);
	CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
	CREATE INDEX IF NOT EXISTS idx_document_tags_tag ON document_tags(tag);
	CREATE INDEX IF NOT EXISTS idx_document_aliases_alias ON document_aliases(alias);

	CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts4(
		path,
		title,
		summary,
		content='documents'
	);

	CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
		INSERT INTO documents_fts(docid, path, title, summary)
		VALUES (new.id, new.path, new.title, new.summary);
	END;

	CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
		DELETE FROM documents_fts WHERE docid = old.id;
	END;

	CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
		DELETE FROM documents_fts WHERE docid = old.id;
		INSERT INTO documents_fts(docid, path, title, summary)
		VALUES (new.id, new.path, new.title, new.summary);
	END;
	`

	_, err := s.db.Exec(schema)
	return err
}

// BuildIndex builds or rebuilds the entire index
func (s *IndexService) BuildIndex() (*IndexStats, error) {
	stats := &IndexStats{
		ByType:   make(map[string]int),
		ByDomain: make(map[string]int),
		ByStatus: make(map[string]int),
	}

	// Scan all markdown files
	sections := []string{SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}

	for _, section := range sections {
		sectionPath := filepath.Join(s.vaultPath, section)
		if _, err := os.Stat(sectionPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Skip non-markdown and special files
			if filepath.Ext(path) != ".md" {
				return nil
			}
			name := info.Name()
			if name == "_toc.md" || strings.HasPrefix(name, ".") {
				return nil
			}

			// Index the document
			doc, err := s.indexDocument(path)
			if err != nil {
				return nil // Skip errors, continue indexing
			}

			stats.TotalDocs++
			if doc.Type != "" {
				stats.ByType[doc.Type]++
			}
			if doc.Domain != "" {
				stats.ByDomain[doc.Domain]++
			}
			if doc.Status != "" {
				stats.ByStatus[doc.Status]++
			}

			return nil
		})

		if err != nil {
			return stats, fmt.Errorf("%s 색인 실패: %w", section, err)
		}
	}

	stats.LastIndexed = time.Now().Format(time.RFC3339)
	return stats, nil
}

// UpdateIndex updates only changed documents
func (s *IndexService) UpdateIndex() (int, int, error) {
	var added, updated int

	sections := []string{SystemDir, DomainsDir, ProjectsDir, ReferencesDir, ArchiveDir}

	for _, section := range sections {
		sectionPath := filepath.Join(s.vaultPath, section)
		if _, err := os.Stat(sectionPath); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(sectionPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".md" {
				return nil
			}
			name := info.Name()
			if name == "_toc.md" || strings.HasPrefix(name, ".") {
				return nil
			}

			relPath, _ := filepath.Rel(s.vaultPath, path)

			// Check if already indexed and up to date
			var indexedAt string
			err = s.db.QueryRow("SELECT indexed_at FROM documents WHERE path = ?", relPath).Scan(&indexedAt)
			switch err {
			case sql.ErrNoRows:
				// New document
				if _, err := s.indexDocument(path); err == nil {
					added++
				}
			case nil:
				// Check if modified
				indexed, _ := time.Parse(time.RFC3339, indexedAt)
				if info.ModTime().After(indexed) {
					if _, err := s.indexDocument(path); err == nil {
						updated++
					}
				}
			}

			return nil
		})
	}

	// Remove deleted documents
	rows, err := s.db.Query("SELECT id, path FROM documents")
	if err != nil {
		return added, updated, err
	}
	defer rows.Close()

	var toDelete []int64
	for rows.Next() {
		var id int64
		var path string
		rows.Scan(&id, &path)

		fullPath := filepath.Join(s.vaultPath, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		s.db.Exec("DELETE FROM documents WHERE id = ?", id)
	}

	return added, updated, nil
}

func (s *IndexService) indexDocument(path string) (*DocumentIndex, error) {
	relPath, _ := filepath.Rel(s.vaultPath, path)

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	doc := &DocumentIndex{
		Path:      relPath,
		IndexedAt: time.Now().Format(time.RFC3339),
	}

	// Parse frontmatter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			var fm map[string]interface{}
			if yaml.Unmarshal([]byte(parts[1]), &fm) == nil {
				if v, ok := fm["title"].(string); ok {
					doc.Title = v
				}
				if v, ok := fm["type"].(string); ok {
					doc.Type = v
				}
				if v, ok := fm["status"].(string); ok {
					doc.Status = v
				}
				if v, ok := fm["domain"].(string); ok {
					doc.Domain = v
				}
				if v, ok := fm["summary"].(string); ok {
					doc.Summary = v
				}
				if v, ok := fm["created"].(string); ok {
					doc.CreatedAt = v
				}
				if v, ok := fm["updated"].(string); ok {
					doc.UpdatedAt = v
				}

				// Parse tags
				if tags, ok := fm["tags"].([]interface{}); ok {
					for _, t := range tags {
						if tag, ok := t.(string); ok {
							doc.Tags = append(doc.Tags, tag)
						}
					}
				}

				// Parse aliases
				if aliases, ok := fm["aliases"].([]interface{}); ok {
					for _, a := range aliases {
						if alias, ok := a.(string); ok {
							doc.Aliases = append(doc.Aliases, alias)
						}
					}
				}
			}
		}
	}

	// Extract title from first heading if not in frontmatter
	if doc.Title == "" {
		doc.Title = s.extractTitle(content)
	}

	// Extract summary from first paragraph if not in frontmatter
	if doc.Summary == "" {
		doc.Summary = s.extractSummary(content)
	}

	// Upsert document
	result, err := s.db.Exec(`
		INSERT INTO documents (path, title, type, status, domain, summary, created_at, updated_at, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			title = excluded.title,
			type = excluded.type,
			status = excluded.status,
			domain = excluded.domain,
			summary = excluded.summary,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at,
			indexed_at = excluded.indexed_at
	`, doc.Path, doc.Title, doc.Type, doc.Status, doc.Domain, doc.Summary,
		doc.CreatedAt, doc.UpdatedAt, doc.IndexedAt)

	if err != nil {
		return nil, err
	}

	// Get document ID
	docID, _ := result.LastInsertId()
	if docID == 0 {
		s.db.QueryRow("SELECT id FROM documents WHERE path = ?", doc.Path).Scan(&docID)
	}
	doc.ID = docID

	// Update tags
	s.db.Exec("DELETE FROM document_tags WHERE doc_id = ?", docID)
	for _, tag := range doc.Tags {
		s.db.Exec("INSERT OR IGNORE INTO document_tags (doc_id, tag) VALUES (?, ?)", docID, tag)
	}

	// Update aliases
	s.db.Exec("DELETE FROM document_aliases WHERE doc_id = ?", docID)
	for _, alias := range doc.Aliases {
		s.db.Exec("INSERT OR IGNORE INTO document_aliases (doc_id, alias) VALUES (?, ?)", docID, alias)
	}

	return doc, nil
}

func (s *IndexService) extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

func (s *IndexService) extractSummary(content string) string {
	// Skip frontmatter
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			content = parts[2]
		}
	}

	// Find first paragraph (after heading)
	lines := strings.Split(content, "\n")
	var inParagraph bool
	var summary strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip headings
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Skip blockquotes (often used for descriptions)
		if strings.HasPrefix(line, ">") {
			text := strings.TrimPrefix(line, ">")
			text = strings.TrimSpace(text)
			if text != "" && summary.Len() < 200 {
				if summary.Len() > 0 {
					summary.WriteString(" ")
				}
				summary.WriteString(text)
			}
			continue
		}

		// Empty line ends paragraph
		if line == "" {
			if inParagraph && summary.Len() > 0 {
				break
			}
			continue
		}

		// Skip code blocks, lists, etc.
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "- ") ||
			strings.HasPrefix(line, "* ") || strings.HasPrefix(line, "|") {
			continue
		}

		inParagraph = true
		if summary.Len() < 200 {
			if summary.Len() > 0 {
				summary.WriteString(" ")
			}
			summary.WriteString(line)
		}
	}

	result := summary.String()
	if len(result) > 200 {
		result = result[:197] + "..."
	}
	return result
}

// Search searches documents
func (s *IndexService) Search(query string, opts *SearchOptions) ([]*SearchResult, error) {
	if opts == nil {
		opts = &SearchOptions{}
	}
	if opts.Limit == 0 {
		opts.Limit = 20
	}

	var results []*SearchResult
	var args []interface{}

	// Build query
	sqlQuery := `
		SELECT d.id, d.path, d.title, d.type, d.status, d.domain, d.summary,
		       d.created_at, d.updated_at, d.indexed_at
		FROM documents d
		JOIN documents_fts fts ON d.id = fts.docid
		WHERE documents_fts MATCH ?
	`
	args = append(args, s.buildFTSQuery(query))

	// Add filters
	if opts.Type != "" {
		sqlQuery += " AND d.type = ?"
		args = append(args, opts.Type)
	}
	if opts.Domain != "" {
		sqlQuery += " AND d.domain = ?"
		args = append(args, opts.Domain)
	}
	if opts.Status != "" {
		sqlQuery += " AND d.status = ?"
		args = append(args, opts.Status)
	}
	if len(opts.Tags) > 0 {
		sqlQuery += " AND d.id IN (SELECT doc_id FROM document_tags WHERE tag IN (" +
			strings.Repeat("?,", len(opts.Tags)-1) + "?))"
		for _, tag := range opts.Tags {
			args = append(args, tag)
		}
	}

	sqlQuery += " LIMIT ?"
	args = append(args, opts.Limit)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalTokens := 0
	resultIdx := 0
	for rows.Next() {
		var doc DocumentIndex
		var createdAt, updatedAt sql.NullString

		err := rows.Scan(&doc.ID, &doc.Path, &doc.Title, &doc.Type, &doc.Status,
			&doc.Domain, &doc.Summary, &createdAt, &updatedAt, &doc.IndexedAt)
		if err != nil {
			continue
		}

		if createdAt.Valid {
			doc.CreatedAt = createdAt.String
		}
		if updatedAt.Valid {
			doc.UpdatedAt = updatedAt.String
		}

		// Load tags
		tagRows, _ := s.db.Query("SELECT tag FROM document_tags WHERE doc_id = ?", doc.ID)
		if tagRows != nil {
			for tagRows.Next() {
				var tag string
				tagRows.Scan(&tag)
				doc.Tags = append(doc.Tags, tag)
			}
			tagRows.Close()
		}

		// Load aliases
		aliasRows, _ := s.db.Query("SELECT alias FROM document_aliases WHERE doc_id = ?", doc.ID)
		if aliasRows != nil {
			for aliasRows.Next() {
				var alias string
				aliasRows.Scan(&alias)
				doc.Aliases = append(doc.Aliases, alias)
			}
			aliasRows.Close()
		}

		result := &SearchResult{
			Document:   &doc,
			Score:      float64(100 - resultIdx), // Simple ranking by order
			Highlights: s.generateHighlights(query, &doc),
		}
		resultIdx++

		// Check token budget
		if opts.TokenBudget > 0 {
			docTokens := s.estimateTokens(&doc)
			if totalTokens+docTokens > opts.TokenBudget {
				break
			}
			totalTokens += docTokens
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *IndexService) buildFTSQuery(query string) string {
	// Simple query: wrap terms in quotes for phrase search
	// Advanced: support field prefixes like title:foo
	words := strings.Fields(query)
	var parts []string

	for _, word := range words {
		// Check for field prefix
		if strings.Contains(word, ":") {
			parts = append(parts, word)
		} else {
			// Escape special characters and add wildcard
			word = regexp.MustCompile(`[^\w가-힣]`).ReplaceAllString(word, "")
			if word != "" {
				parts = append(parts, word+"*")
			}
		}
	}

	return strings.Join(parts, " ")
}

func (s *IndexService) generateHighlights(query string, doc *DocumentIndex) []string {
	var highlights []string
	queryLower := strings.ToLower(query)
	words := strings.Fields(queryLower)

	// Check title
	titleLower := strings.ToLower(doc.Title)
	for _, word := range words {
		if strings.Contains(titleLower, word) {
			highlights = append(highlights, fmt.Sprintf("제목: %s", doc.Title))
			break
		}
	}

	// Check summary
	if doc.Summary != "" {
		summaryLower := strings.ToLower(doc.Summary)
		for _, word := range words {
			if strings.Contains(summaryLower, word) {
				highlights = append(highlights, fmt.Sprintf("요약: %s", doc.Summary))
				break
			}
		}
	}

	return highlights
}

func (s *IndexService) estimateTokens(doc *DocumentIndex) int {
	// Rough estimate: ~4 chars per token for mixed Korean/English
	total := len(doc.Title) + len(doc.Summary) + len(doc.Path)
	for _, tag := range doc.Tags {
		total += len(tag)
	}
	return total / 4
}

// GetStats returns index statistics
func (s *IndexService) GetStats() (*IndexStats, error) {
	stats := &IndexStats{
		ByType:   make(map[string]int),
		ByDomain: make(map[string]int),
		ByStatus: make(map[string]int),
	}

	// Total count
	s.db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&stats.TotalDocs)

	// By type
	rows, _ := s.db.Query("SELECT type, COUNT(*) FROM documents WHERE type != '' GROUP BY type")
	if rows != nil {
		for rows.Next() {
			var t string
			var count int
			rows.Scan(&t, &count)
			stats.ByType[t] = count
		}
		rows.Close()
	}

	// By domain
	rows, _ = s.db.Query("SELECT domain, COUNT(*) FROM documents WHERE domain != '' GROUP BY domain")
	if rows != nil {
		for rows.Next() {
			var d string
			var count int
			rows.Scan(&d, &count)
			stats.ByDomain[d] = count
		}
		rows.Close()
	}

	// By status
	rows, _ = s.db.Query("SELECT status, COUNT(*) FROM documents WHERE status != '' GROUP BY status")
	if rows != nil {
		for rows.Next() {
			var st string
			var count int
			rows.Scan(&st, &count)
			stats.ByStatus[st] = count
		}
		rows.Close()
	}

	// Last indexed
	var lastIndexed sql.NullString
	s.db.QueryRow("SELECT MAX(indexed_at) FROM documents").Scan(&lastIndexed)
	if lastIndexed.Valid {
		stats.LastIndexed = lastIndexed.String
	}

	return stats, nil
}

// SearchByAlias searches by document alias
func (s *IndexService) SearchByAlias(alias string) (*DocumentIndex, error) {
	var docID int64
	err := s.db.QueryRow("SELECT doc_id FROM document_aliases WHERE alias = ?", alias).Scan(&docID)
	if err != nil {
		return nil, err
	}

	return s.getDocumentByID(docID)
}

// SearchByTag searches by tag
func (s *IndexService) SearchByTag(tag string, limit int) ([]*DocumentIndex, error) {
	if limit == 0 {
		limit = 20
	}

	rows, err := s.db.Query(`
		SELECT d.id FROM documents d
		JOIN document_tags t ON d.id = t.doc_id
		WHERE t.tag = ?
		LIMIT ?
	`, tag, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*DocumentIndex
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		if doc, err := s.getDocumentByID(id); err == nil {
			docs = append(docs, doc)
		}
	}

	return docs, nil
}

func (s *IndexService) getDocumentByID(id int64) (*DocumentIndex, error) {
	doc := &DocumentIndex{}
	var createdAt, updatedAt sql.NullString

	err := s.db.QueryRow(`
		SELECT id, path, title, type, status, domain, summary, created_at, updated_at, indexed_at
		FROM documents WHERE id = ?
	`, id).Scan(&doc.ID, &doc.Path, &doc.Title, &doc.Type, &doc.Status,
		&doc.Domain, &doc.Summary, &createdAt, &updatedAt, &doc.IndexedAt)

	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		doc.CreatedAt = createdAt.String
	}
	if updatedAt.Valid {
		doc.UpdatedAt = updatedAt.String
	}

	// Load tags
	tagRows, _ := s.db.Query("SELECT tag FROM document_tags WHERE doc_id = ?", id)
	if tagRows != nil {
		for tagRows.Next() {
			var tag string
			tagRows.Scan(&tag)
			doc.Tags = append(doc.Tags, tag)
		}
		tagRows.Close()
	}

	// Load aliases
	aliasRows, _ := s.db.Query("SELECT alias FROM document_aliases WHERE doc_id = ?", id)
	if aliasRows != nil {
		for aliasRows.Next() {
			var alias string
			aliasRows.Scan(&alias)
			doc.Aliases = append(doc.Aliases, alias)
		}
		aliasRows.Close()
	}

	return doc, nil
}

// ListTags returns all tags with counts
func (s *IndexService) ListTags() (map[string]int, error) {
	tags := make(map[string]int)

	rows, err := s.db.Query("SELECT tag, COUNT(*) FROM document_tags GROUP BY tag ORDER BY COUNT(*) DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		var count int
		rows.Scan(&tag, &count)
		tags[tag] = count
	}

	return tags, nil
}

// ListDomains returns all domains with counts
func (s *IndexService) ListDomains() (map[string]int, error) {
	domains := make(map[string]int)

	rows, err := s.db.Query("SELECT domain, COUNT(*) FROM documents WHERE domain != '' GROUP BY domain ORDER BY COUNT(*) DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var domain string
		var count int
		rows.Scan(&domain, &count)
		domains[domain] = count
	}

	return domains, nil
}
