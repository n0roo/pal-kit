package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/db"
)

// Project represents a PAL Kit project
type Project struct {
	Root         string    `json:"root"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	LastActive   time.Time `json:"last_active,omitempty"`
	SessionCount int       `json:"session_count"`
	PortCount    int       `json:"port_count"`
	ActivePorts  int       `json:"active_ports"`
	TotalTokens  int64     `json:"total_tokens"`
	CreatedAt    time.Time `json:"created_at"`
	Initialized  bool      `json:"initialized"`
}

// RegisterProjectRoutes registers project management routes
func (s *Server) RegisterProjectRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v2/projects", s.withCORS(s.handleProjectsV2))
	mux.HandleFunc("/api/v2/projects/import", s.withCORS(s.handleProjectImport))
	mux.HandleFunc("/api/v2/projects/init", s.withCORS(s.handleProjectInit))
	mux.HandleFunc("/api/v2/projects/", s.withCORS(s.handleProjectDetailV2))
}

func (s *Server) handleProjectsV2(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.listProjects(w, r)
	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	// Query projects from database
	rows, err := database.Query(`
		SELECT
			root, name, description, last_active,
			session_count, total_tokens, created_at
		FROM projects
		ORDER BY last_active DESC
	`)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer rows.Close()

	projects := make([]Project, 0)
	for rows.Next() {
		var p Project
		var lastActive, createdAt *string
		var description *string

		err := rows.Scan(
			&p.Root, &p.Name, &description, &lastActive,
			&p.SessionCount, &p.TotalTokens, &createdAt,
		)
		if err != nil {
			continue
		}

		if description != nil {
			p.Description = *description
		}
		if lastActive != nil {
			p.LastActive, _ = time.Parse(time.RFC3339, *lastActive)
		}
		if createdAt != nil {
			p.CreatedAt, _ = time.Parse(time.RFC3339, *createdAt)
		}

		// Check if project is initialized (has .pal folder)
		p.Initialized = isProjectInitialized(p.Root)

		// Get port counts
		pc := s.getProjectPortCounts(database, p.Root)
		p.PortCount = pc.total
		p.ActivePorts = pc.active

		projects = append(projects, p)
	}

	s.jsonResponse(w, projects)
}

type portCounts struct {
	total  int
	active int
}

func (s *Server) getProjectPortCounts(database *db.DB, root string) portCounts {
	counts := portCounts{}

	// Count total ports for this project (via sessions)
	row := database.QueryRow(`
		SELECT COUNT(DISTINCT p.id)
		FROM ports p
		JOIN sessions s ON p.session_id = s.id
		WHERE s.project_root = ?
	`, root)
	row.Scan(&counts.total)

	// Count active (running) ports
	row = database.QueryRow(`
		SELECT COUNT(DISTINCT p.id)
		FROM ports p
		JOIN sessions s ON p.session_id = s.id
		WHERE s.project_root = ? AND p.status = 'running'
	`, root)
	row.Scan(&counts.active)

	return counts
}

func isProjectInitialized(root string) bool {
	palPath := filepath.Join(root, ".pal")
	_, err := os.Stat(palPath)
	return err == nil
}

func (s *Server) handleProjectDetailV2(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/projects/")
	if path == "" || path == "import" || path == "init" {
		s.errorResponse(w, 400, "Project root required")
		return
	}

	// URL decode the path
	root := path

	switch r.Method {
	case "GET":
		s.getProject(w, r, root)
	case "DELETE":
		s.removeProject(w, r, root)
	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request, root string) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	var p Project
	var lastActive, createdAt *string
	var description *string

	err = database.DB.QueryRow(`
		SELECT root, name, description, last_active, session_count, total_tokens, created_at
		FROM projects WHERE root = ?
	`, root).Scan(
		&p.Root, &p.Name, &description, &lastActive,
		&p.SessionCount, &p.TotalTokens, &createdAt,
	)
	if err != nil {
		s.errorResponse(w, 404, "Project not found")
		return
	}

	if description != nil {
		p.Description = *description
	}
	if lastActive != nil {
		p.LastActive, _ = time.Parse(time.RFC3339, *lastActive)
	}
	if createdAt != nil {
		p.CreatedAt, _ = time.Parse(time.RFC3339, *createdAt)
	}
	p.Initialized = isProjectInitialized(p.Root)

	s.jsonResponse(w, p)
}

func (s *Server) removeProject(w http.ResponseWriter, r *http.Request, root string) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	_, err = database.Exec("DELETE FROM projects WHERE root = ?", root)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, map[string]string{"status": "removed"})
}

func (s *Server) handleProjectImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, 400, "Invalid request body")
		return
	}

	// Validate path exists
	info, err := os.Stat(req.Path)
	if err != nil || !info.IsDir() {
		s.errorResponse(w, 400, "Invalid directory path")
		return
	}

	// Check if .pal exists
	initialized := isProjectInitialized(req.Path)

	// Determine name
	name := req.Name
	if name == "" {
		name = filepath.Base(req.Path)
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	// Insert or update project
	_, err = database.Exec(`
		INSERT INTO projects (root, name, last_active, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(root) DO UPDATE SET
			name = excluded.name,
			last_active = CURRENT_TIMESTAMP
	`, req.Path, name)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, map[string]interface{}{
		"root":        req.Path,
		"name":        name,
		"initialized": initialized,
		"status":      "imported",
	})
}

func (s *Server) handleProjectInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, 400, "Invalid request body")
		return
	}

	// Validate path exists
	info, err := os.Stat(req.Path)
	if err != nil || !info.IsDir() {
		s.errorResponse(w, 400, "Invalid directory path")
		return
	}

	// Determine name
	name := req.Name
	if name == "" {
		name = filepath.Base(req.Path)
	}

	// Initialize project using config package
	defaultConfig := config.DefaultProjectConfig(name)
	if err := config.SaveProjectConfig(req.Path, defaultConfig); err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	// Insert or update project
	_, err = database.Exec(`
		INSERT INTO projects (root, name, last_active, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(root) DO UPDATE SET
			name = excluded.name,
			last_active = CURRENT_TIMESTAMP
	`, req.Path, name)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, map[string]interface{}{
		"root":        req.Path,
		"name":        name,
		"initialized": true,
		"status":      "initialized",
	})
}
