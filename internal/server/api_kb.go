package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/n0roo/pal-kit/internal/kb"
)

// RegisterKBRoutes registers Knowledge Base API routes
func (s *Server) RegisterKBRoutes(mux *http.ServeMux) {
	// KB Status & Init
	mux.HandleFunc("/api/v2/kb/status", s.withCORS(s.handleKBStatus))
	mux.HandleFunc("/api/v2/kb/init", s.withCORS(s.handleKBInit))

	// TOC Management
	mux.HandleFunc("/api/v2/kb/toc", s.withCORS(s.handleKBTocList))
	mux.HandleFunc("/api/v2/kb/toc/", s.withCORS(s.handleKBTocDetail))

	// Document Operations
	mux.HandleFunc("/api/v2/kb/documents", s.withCORS(s.handleKBDocuments))
	mux.HandleFunc("/api/v2/kb/documents/", s.withCORS(s.handleKBDocumentDetail))

	// Index Operations
	mux.HandleFunc("/api/v2/kb/index", s.withCORS(s.handleKBIndex))

	// Tags
	mux.HandleFunc("/api/v2/kb/tags", s.withCORS(s.handleKBTags))

	// Sections
	mux.HandleFunc("/api/v2/kb/sections", s.withCORS(s.handleKBSections))
}

// getVaultPath returns the vault path from config or default
func (s *Server) getVaultPath() string {
	if s.config.VaultPath != "" {
		return s.config.VaultPath
	}
	// Default to ~/mcp-docs
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "mcp-docs")
}

// ========================================
// KB Status & Init Handlers
// ========================================

func (s *Server) handleKBStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	vaultPath := s.getVaultPath()
	svc := kb.NewService(vaultPath)

	status, err := svc.Status()
	if err != nil {
		// Not initialized
		s.jsonResponse(w, map[string]interface{}{
			"initialized": false,
			"vault_path":  vaultPath,
			"error":       err.Error(),
		})
		return
	}

	s.jsonResponse(w, map[string]interface{}{
		"initialized": status.Initialized,
		"vault_path":  status.VaultPath,
		"version":     status.Version,
		"created_at":  status.CreatedAt,
		"sections":    status.Sections,
	})
}

func (s *Server) handleKBInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	vaultPath := s.getVaultPath()

	// Allow custom vault path from request body
	var req struct {
		VaultPath string `json:"vault_path"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.VaultPath != "" {
		vaultPath = req.VaultPath
	}

	svc := kb.NewService(vaultPath)
	if err := svc.Init(); err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, map[string]string{
		"status":     "initialized",
		"vault_path": vaultPath,
	})
}

// ========================================
// TOC Handlers
// ========================================

func (s *Server) handleKBTocList(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	vaultPath := s.getVaultPath()
	svc := kb.NewService(vaultPath)

	// Get all section TOCs
	sections := []string{kb.SystemDir, kb.DomainsDir, kb.ProjectsDir, kb.ReferencesDir, kb.ArchiveDir}
	result := make([]map[string]interface{}, 0)

	for _, section := range sections {
		tocPath := filepath.Join(vaultPath, section, "_toc.md")
		if _, err := os.Stat(tocPath); err == nil {
			// TOC exists, get check result for stats
			checkResult, _ := svc.CheckTOC(section)
			item := map[string]interface{}{
				"section": section,
				"exists":  true,
			}
			if checkResult != nil {
				item["valid"] = checkResult.Valid
				item["needs_refresh"] = checkResult.NeedsRefresh
				item["missing_count"] = len(checkResult.MissingDocs)
				item["orphan_count"] = len(checkResult.OrphanLinks)
			}
			result = append(result, item)
		} else {
			result = append(result, map[string]interface{}{
				"section": section,
				"exists":  false,
			})
		}
	}

	s.jsonResponse(w, result)
}

func (s *Server) handleKBTocDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/kb/toc/")
	parts := strings.Split(path, "/")
	section := parts[0]

	if section == "" {
		s.errorResponse(w, 400, "Section required")
		return
	}

	vaultPath := s.getVaultPath()
	svc := kb.NewService(vaultPath)

	// Check for sub-resources
	if len(parts) > 1 {
		action := parts[1]
		switch action {
		case "generate":
			if r.Method != "POST" {
				s.errorResponse(w, 405, "Method not allowed")
				return
			}

			var req struct {
				Depth  int    `json:"depth"`
				SortBy string `json:"sort_by"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.Depth == 0 {
				req.Depth = 3
			}
			if req.SortBy == "" {
				req.SortBy = "alphabetical"
			}

			stats, err := svc.GenerateTOC(section, req.Depth, req.SortBy)
			if err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, stats)
			return

		case "check":
			result, err := svc.CheckTOC(section)
			if err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, result)
			return
		}
	}

	// GET section TOC content
	if r.Method != "GET" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	tocPath := filepath.Join(vaultPath, section, "_toc.md")
	content, err := os.ReadFile(tocPath)
	if err != nil {
		s.errorResponse(w, 404, "TOC not found")
		return
	}

	// Parse TOC content to structured format
	entries := parseTocEntries(string(content), vaultPath, section)

	s.jsonResponse(w, map[string]interface{}{
		"section": section,
		"content": string(content),
		"entries": entries,
	})
}

// parseTocEntries extracts TOC entries from markdown content
func parseTocEntries(content string, vaultPath string, section string) []map[string]interface{} {
	entries := make([]map[string]interface{}, 0)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		// Match [[path|title]] pattern
		if strings.Contains(line, "[[") && strings.Contains(line, "]]") {
			start := strings.Index(line, "[[")
			end := strings.Index(line, "]]")
			if start < end {
				link := line[start+2 : end]
				parts := strings.SplitN(link, "|", 2)

				path := parts[0]
				title := path
				if len(parts) > 1 {
					title = parts[1]
				}

				// Count leading spaces/tabs for depth
				indent := 0
				for _, c := range line {
					switch c {
					case ' ':
						indent++
					case '\t':
						indent += 2
					default:
						goto countDone
					}
				}
			countDone:
				depth := indent / 2

				// Check if directory
				fullPath := filepath.Join(vaultPath, section, path)
				isDir := false
				if info, err := os.Stat(fullPath); err == nil {
					isDir = info.IsDir()
				}

				entries = append(entries, map[string]interface{}{
					"path":   path,
					"title":  title,
					"depth":  depth,
					"is_dir": isDir,
				})
			}
		}
	}

	return entries
}

// ========================================
// Document Handlers
// ========================================

func (s *Server) handleKBDocuments(w http.ResponseWriter, r *http.Request) {
	vaultPath := s.getVaultPath()

	switch r.Method {
	case "GET":
		// Search documents
		query := r.URL.Query().Get("q")
		docType := r.URL.Query().Get("type")
		domain := r.URL.Query().Get("domain")
		status := r.URL.Query().Get("status")
		tag := r.URL.Query().Get("tag")
		limitStr := r.URL.Query().Get("limit")
		tokenBudgetStr := r.URL.Query().Get("token_budget")

		limit := 50
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}

		tokenBudget := 0
		if tb, err := strconv.Atoi(tokenBudgetStr); err == nil && tb > 0 {
			tokenBudget = tb
		}

		indexSvc := kb.NewIndexService(vaultPath)
		if err := indexSvc.Open(); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		defer indexSvc.Close()

		opts := &kb.SearchOptions{
			Type:        docType,
			Domain:      domain,
			Status:      status,
			Limit:       limit,
			TokenBudget: tokenBudget,
		}
		if tag != "" {
			opts.Tags = []string{tag}
		}

		results, err := indexSvc.Search(query, opts)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}

		s.jsonResponse(w, results)

	case "POST":
		// Create document
		var req struct {
			Path    string   `json:"path"`
			Title   string   `json:"title"`
			Type    string   `json:"type"`
			Domain  string   `json:"domain"`
			Status  string   `json:"status"`
			Tags    []string `json:"tags"`
			Content string   `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.errorResponse(w, 400, "Invalid request body")
			return
		}

		if req.Path == "" || req.Title == "" {
			s.errorResponse(w, 400, "path and title required")
			return
		}

		// Build frontmatter
		var sb strings.Builder
		sb.WriteString("---\n")
		sb.WriteString("title: " + req.Title + "\n")
		if req.Type != "" {
			sb.WriteString("type: " + req.Type + "\n")
		}
		if req.Domain != "" {
			sb.WriteString("domain: " + req.Domain + "\n")
		}
		if req.Status != "" {
			sb.WriteString("status: " + req.Status + "\n")
		} else {
			sb.WriteString("status: draft\n")
		}
		if len(req.Tags) > 0 {
			sb.WriteString("tags: [" + strings.Join(req.Tags, ", ") + "]\n")
		}
		sb.WriteString("---\n\n")
		sb.WriteString("# " + req.Title + "\n\n")
		sb.WriteString(req.Content)

		// Write file
		fullPath := filepath.Join(vaultPath, req.Path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		if err := os.WriteFile(fullPath, []byte(sb.String()), 0644); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}

		s.jsonResponse(w, map[string]string{
			"status": "created",
			"path":   req.Path,
		})

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) handleKBDocumentDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/kb/documents/")
	if path == "" {
		s.errorResponse(w, 400, "Document path required")
		return
	}

	vaultPath := s.getVaultPath()

	// Check for move action
	if strings.HasSuffix(path, "/move") {
		if r.Method != "POST" {
			s.errorResponse(w, 405, "Method not allowed")
			return
		}
		docPath := strings.TrimSuffix(path, "/move")
		s.handleKBDocumentMove(w, r, vaultPath, docPath)
		return
	}

	// Check for content action
	if strings.HasSuffix(path, "/content") {
		docPath := strings.TrimSuffix(path, "/content")
		s.handleKBDocumentContent(w, r, vaultPath, docPath)
		return
	}

	switch r.Method {
	case "GET":
		// Get document metadata and content
		fullPath := filepath.Join(vaultPath, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			s.errorResponse(w, 404, "Document not found")
			return
		}

		// Parse frontmatter
		metadata := parseDocumentFrontmatter(string(content))
		metadata["path"] = path
		metadata["content"] = string(content)

		s.jsonResponse(w, metadata)

	case "PUT":
		// Update document
		fullPath := filepath.Join(vaultPath, path)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.errorResponse(w, 400, "Failed to read body")
			return
		}

		var req struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			// If not JSON, treat as raw content
			req.Content = string(body)
		}

		if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}

		s.jsonResponse(w, map[string]string{"status": "updated"})

	case "DELETE":
		// Delete document
		fullPath := filepath.Join(vaultPath, path)
		if err := os.Remove(fullPath); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}

		s.jsonResponse(w, map[string]string{"status": "deleted"})

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) handleKBDocumentContent(w http.ResponseWriter, r *http.Request, vaultPath, docPath string) {
	fullPath := filepath.Join(vaultPath, docPath)

	if r.Method != "GET" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		s.errorResponse(w, 404, "Document not found")
		return
	}

	s.jsonResponse(w, map[string]string{
		"path":    docPath,
		"content": string(content),
	})
}

func (s *Server) handleKBDocumentMove(w http.ResponseWriter, r *http.Request, vaultPath, docPath string) {
	var req struct {
		NewPath string `json:"new_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, 400, "Invalid request body")
		return
	}

	if req.NewPath == "" {
		s.errorResponse(w, 400, "new_path required")
		return
	}

	oldPath := filepath.Join(vaultPath, docPath)
	newPath := filepath.Join(vaultPath, req.NewPath)

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(newPath), 0755); err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	// Move file
	if err := os.Rename(oldPath, newPath); err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, map[string]string{
		"status":   "moved",
		"old_path": docPath,
		"new_path": req.NewPath,
	})
}

// parseDocumentFrontmatter extracts frontmatter from markdown content
func parseDocumentFrontmatter(content string) map[string]interface{} {
	result := make(map[string]interface{})

	if !strings.HasPrefix(content, "---") {
		return result
	}

	endIdx := strings.Index(content[3:], "---")
	if endIdx == -1 {
		return result
	}

	frontmatter := content[3 : endIdx+3]
	lines := strings.Split(frontmatter, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Handle arrays [a, b, c]
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			inner := value[1 : len(value)-1]
			items := strings.Split(inner, ",")
			arr := make([]string, 0, len(items))
			for _, item := range items {
				arr = append(arr, strings.TrimSpace(item))
			}
			result[key] = arr
		} else {
			result[key] = value
		}
	}

	return result
}

// ========================================
// Index Handlers
// ========================================

func (s *Server) handleKBIndex(w http.ResponseWriter, r *http.Request) {
	vaultPath := s.getVaultPath()
	indexSvc := kb.NewIndexService(vaultPath)

	switch r.Method {
	case "GET":
		// Get index stats
		if err := indexSvc.Open(); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		defer indexSvc.Close()

		stats, err := indexSvc.GetStats()
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, stats)

	case "POST":
		// Rebuild index
		action := r.URL.Query().Get("action")

		if err := indexSvc.Open(); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		defer indexSvc.Close()

		if action == "update" {
			added, updated, err := indexSvc.UpdateIndex()
			if err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, map[string]interface{}{
				"status":  "updated",
				"added":   added,
				"updated": updated,
			})
		} else {
			stats, err := indexSvc.BuildIndex()
			if err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, map[string]interface{}{
				"status": "rebuilt",
				"stats":  stats,
			})
		}

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

// ========================================
// Tags Handlers
// ========================================

func (s *Server) handleKBTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	vaultPath := s.getVaultPath()
	indexSvc := kb.NewIndexService(vaultPath)

	if err := indexSvc.Open(); err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer indexSvc.Close()

	tags, err := indexSvc.ListTags()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, tags)
}

// ========================================
// Sections Handler
// ========================================

func (s *Server) handleKBSections(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	// Return predefined KB sections
	sections := []map[string]interface{}{
		{
			"id":          kb.SystemDir,
			"name":        "시스템",
			"description": "메타 문서 및 시스템 설정",
			"icon":        "settings",
		},
		{
			"id":          kb.DomainsDir,
			"name":        "도메인",
			"description": "도메인별 지식 문서",
			"icon":        "folder",
		},
		{
			"id":          kb.ProjectsDir,
			"name":        "프로젝트",
			"description": "프로젝트별 문서",
			"icon":        "project",
		},
		{
			"id":          kb.ReferencesDir,
			"name":        "참조",
			"description": "참조 문서",
			"icon":        "book",
		},
		{
			"id":          kb.ArchiveDir,
			"name":        "아카이브",
			"description": "아카이브된 문서",
			"icon":        "archive",
		},
	}

	s.jsonResponse(w, sections)
}
