package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/n0roo/pal-kit/internal/agent"
	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/convention"
	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/docs"
	"github.com/n0roo/pal-kit/internal/escalation"
	"github.com/n0roo/pal-kit/internal/lock"
	"github.com/n0roo/pal-kit/internal/manifest"
	"github.com/n0roo/pal-kit/internal/pipeline"
	"github.com/n0roo/pal-kit/internal/port"
	"github.com/n0roo/pal-kit/internal/session"
)

//go:embed static/*
var staticFiles embed.FS

// Config holds server configuration
type Config struct {
	Port        int
	ProjectRoot string
	DBPath      string
}

// Server represents the web server
type Server struct {
	config Config
	srv    *http.Server
}

// NewServer creates a new server
func NewServer(config Config) *Server {
	return &Server{
		config: config,
	}
}

// Start starts the server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/status", s.withCORS(s.handleStatus))
	mux.HandleFunc("/api/projects", s.withCORS(s.handleProjects))
	mux.HandleFunc("/api/projects/detail", s.withCORS(s.handleProjectDetail))
	mux.HandleFunc("/api/sessions", s.withCORS(s.handleSessions))
	mux.HandleFunc("/api/sessions/stats", s.withCORS(s.handleSessionStats))
	mux.HandleFunc("/api/sessions/history", s.withCORS(s.handleSessionHistory))
	mux.HandleFunc("/api/sessions/", s.withCORS(s.handleSessionDetail))
	mux.HandleFunc("/api/ports", s.withCORS(s.handlePorts))
	mux.HandleFunc("/api/pipelines", s.withCORS(s.handlePipelines))
	mux.HandleFunc("/api/agents", s.withCORS(s.handleAgents))
	mux.HandleFunc("/api/docs", s.withCORS(s.handleDocs))
	mux.HandleFunc("/api/docs/content", s.withCORS(s.handleDocContent))
	mux.HandleFunc("/api/conventions", s.withCORS(s.handleConventions))
	mux.HandleFunc("/api/locks", s.withCORS(s.handleLocks))
	mux.HandleFunc("/api/escalations", s.withCORS(s.handleEscalations))
	mux.HandleFunc("/api/manifest", s.withCORS(s.handleManifest))
	mux.HandleFunc("/api/manifest/changes", s.withCORS(s.handleManifestChanges))

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("static files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("🚀 PAL Kit Dashboard running at http://localhost:%d", s.config.Port)
	return s.srv.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

// withCORS adds CORS headers
func (s *Server) withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

// JSON response helper
func (s *Server) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// Error response helper
func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Database helper
func (s *Server) getDB() (*db.DB, error) {
	return db.Open(s.config.DBPath)
}

// handleStatus returns overall status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	status := map[string]interface{}{
		"project_root": s.config.ProjectRoot,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	// Sessions
	sessionSvc := session.NewService(database)
	if sessions, err := sessionSvc.List(false, 100); err == nil {
		active := 0
		for _, s := range sessions {
			if s.Status == "active" {
				active++
			}
		}
		status["sessions"] = map[string]int{
			"active": active,
			"total":  len(sessions),
		}
	}

	// Ports
	portSvc := port.NewService(database)
	if ports, err := portSvc.List("", 100); err == nil {
		status["ports"] = map[string]int{
			"total": len(ports),
		}
	}

	// Pipelines
	plSvc := pipeline.NewService(database)
	if pipelines, err := plSvc.List("", 100); err == nil {
		running := 0
		for _, p := range pipelines {
			if p.Status == "running" {
				running++
			}
		}
		status["pipelines"] = map[string]int{
			"running": running,
			"total":   len(pipelines),
		}
	}

	// Docs
	docsSvc := docs.NewService(s.config.ProjectRoot)
	if documents, err := docsSvc.List(); err == nil {
		status["docs"] = map[string]int{
			"total": len(documents),
		}
	}

	// Conventions
	convSvc := convention.NewService(s.config.ProjectRoot)
	if conventions, err := convSvc.List(); err == nil {
		enabled := 0
		for _, c := range conventions {
			if c.Enabled {
				enabled++
			}
		}
		status["conventions"] = map[string]int{
			"enabled": enabled,
			"total":   len(conventions),
		}
	}

	// Locks
	lockSvc := lock.NewService(database)
	if locks, err := lockSvc.List(); err == nil {
		status["locks"] = map[string]int{
			"active": len(locks),
		}
	}

	// Escalations
	escSvc := escalation.NewService(database)
	if escalations, err := escSvc.List("open", 100); err == nil {
		status["escalations"] = map[string]int{
			"open": len(escalations),
		}
	}

	s.jsonResponse(w, status)
}

// handleSessions returns session list
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := session.NewService(database)
	
	// Use detailed list for richer info
	details, err := svc.ListDetailed(false, 50)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, toSessionDetailDTOs(details))
}

// handleSessionStats returns session statistics
func (s *Server) handleSessionStats(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := session.NewService(database)
	stats, err := svc.GetStats()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, stats)
}

// handleSessionHistory returns session history by date
func (s *Server) handleSessionHistory(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	// Default to 30 days
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	svc := session.NewService(database)
	history, err := svc.GetHistory(days)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, history)
}

// handleSessionDetail returns single session detail or events
func (s *Server) handleSessionDetail(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from path: /api/sessions/{id} or /api/sessions/{id}/events
	path := r.URL.Path
	trimmed := strings.TrimPrefix(path, "/api/sessions/")
	if trimmed == "" || trimmed == "stats" || trimmed == "history" {
		s.errorResponse(w, 400, "session ID required")
		return
	}

	// Check if requesting events
	parts := strings.Split(trimmed, "/")
	id := parts[0]
	isEventsRequest := len(parts) > 1 && parts[1] == "events"

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := session.NewService(database)

	if isEventsRequest {
		// Handle events request
		limit := 50
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		// Optional type filter
		eventType := r.URL.Query().Get("type")

		events, err := svc.GetEvents(id, eventType, limit)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}

		s.jsonResponse(w, toSessionEventDTOs(events))
		return
	}

	// Handle session detail request
	detail, err := svc.GetDetail(id)
	if err != nil {
		s.errorResponse(w, 404, err.Error())
		return
	}

	// Also get children
	children, _ := svc.GetChildren(id)

	response := map[string]interface{}{
		"session":  toSessionDetailDTO(*detail),
		"children": toSessionDTOs(children),
	}

	s.jsonResponse(w, response)
}

// handlePorts returns port list
func (s *Server) handlePorts(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := port.NewService(database)
	ports, err := svc.List("", 50)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, toPortDTOs(ports))
}

// handlePipelines returns pipeline list
func (s *Server) handlePipelines(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := pipeline.NewService(database)
	pipelines, err := svc.List("", 50)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, toPipelineDTOs(pipelines))
}

// handleAgents returns agent list
func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	svc := agent.NewService(s.config.ProjectRoot)

	agents, err := svc.List()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, agents)
}

// handleDocs returns document list
func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	svc := docs.NewService(s.config.ProjectRoot)
	documents, err := svc.List()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, documents)
}

// handleDocContent returns the content of a specific document
func (s *Server) handleDocContent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		s.errorResponse(w, 400, "path parameter required")
		return
	}

	svc := docs.NewService(s.config.ProjectRoot)
	content, err := svc.GetContent(path)
	if err != nil {
		s.errorResponse(w, 404, err.Error())
		return
	}

	doc, _ := svc.Get(path)
	response := map[string]interface{}{
		"path":    path,
		"content": content,
	}
	if doc != nil {
		response["type"] = doc.Type
		response["size"] = doc.Size
		response["modified_at"] = doc.ModifiedAt
	}

	s.jsonResponse(w, response)
}

// handleConventions returns convention list
func (s *Server) handleConventions(w http.ResponseWriter, r *http.Request) {
	svc := convention.NewService(s.config.ProjectRoot)
	conventions, err := svc.List()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, conventions)
}

// handleLocks returns lock list
func (s *Server) handleLocks(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := lock.NewService(database)
	locks, err := svc.List()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, toLockDTOs(locks))
}

// handleProjects returns registered projects list
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	rows, err := database.Query(`
		SELECT root, name, description, last_active, session_count, total_tokens, total_cost, created_at
		FROM projects
		ORDER BY last_active DESC
	`)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var projects []map[string]interface{}
	for rows.Next() {
		var root, name string
		var description, lastActive, createdAt *string
		var sessionCount, totalTokens int64
		var totalCost float64

		if err := rows.Scan(&root, &name, &description, &lastActive, &sessionCount, &totalTokens, &totalCost, &createdAt); err != nil {
			continue
		}

		project := map[string]interface{}{
			"root":          root,
			"name":          name,
			"description":   description,
			"last_active":   lastActive,
			"session_count": sessionCount,
			"total_tokens":  totalTokens,
			"total_cost":    totalCost,
			"created_at":    createdAt,
		}
		projects = append(projects, project)
	}

	if projects == nil {
		projects = []map[string]interface{}{}
	}

	s.jsonResponse(w, projects)
}

// handleProjectDetail returns detailed project information
func (s *Server) handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	root := r.URL.Query().Get("root")
	if root == "" {
		s.errorResponse(w, 400, "root parameter required")
		return
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	// Get project from DB
	var name string
	var description, lastActive, createdAt *string
	var sessionCount, totalTokens int64
	var totalCost float64

	err = database.QueryRow(`
		SELECT name, description, last_active, session_count, total_tokens, total_cost, created_at
		FROM projects WHERE root = ?
	`, root).Scan(&name, &description, &lastActive, &sessionCount, &totalTokens, &totalCost, &createdAt)
	if err != nil {
		s.errorResponse(w, 404, "project not found")
		return
	}

	// Get project config if available
	var configData map[string]interface{}
	if cfg, err := config.LoadProjectConfig(root); err == nil {
		configData = map[string]interface{}{
			"version":  cfg.Version,
			"workflow": cfg.Workflow.Type,
			"agents": map[string]interface{}{
				"core":    cfg.Agents.Core,
				"workers": cfg.Agents.Workers,
				"testers": cfg.Agents.Testers,
			},
			"settings": map[string]interface{}{
				"auto_port_create":     cfg.Settings.AutoPortCreate,
				"require_user_review":  cfg.Settings.RequireUserReview,
				"auto_test_on_complete": cfg.Settings.AutoTestOnComplete,
			},
		}
	}

	// Get recent sessions for this project
	rows, err := database.Query(`
		SELECT id, session_type, title, status, input_tokens, output_tokens,
		       cost_usd, started_at, ended_at
		FROM sessions
		WHERE project_root = ?
		ORDER BY started_at DESC
		LIMIT 10
	`, root)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var id, sessionType, title, status string
		var inputTokens, outputTokens int64
		var cost float64
		var startTime, endTime *string

		if err := rows.Scan(&id, &sessionType, &title, &status, &inputTokens, &outputTokens, &cost, &startTime, &endTime); err != nil {
			continue
		}

		sessions = append(sessions, map[string]interface{}{
			"id":            id,
			"type":          sessionType,
			"title":         title,
			"status":        status,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"cost":          cost,
			"start_time":    startTime,
			"end_time":      endTime,
		})
	}

	// Get ports for this project
	portSvc := port.NewService(database)
	ports, _ := portSvc.List("", 100)
	var portList []map[string]interface{}
	for _, p := range ports {
		var title string
		if p.Title.Valid {
			title = p.Title.String
		}
		portList = append(portList, map[string]interface{}{
			"id":     p.ID,
			"title":  title,
			"status": p.Status,
		})
	}

	result := map[string]interface{}{
		"root":          root,
		"name":          name,
		"description":   description,
		"last_active":   lastActive,
		"session_count": sessionCount,
		"total_tokens":  totalTokens,
		"total_cost":    totalCost,
		"created_at":    createdAt,
		"config":        configData,
		"sessions":      sessions,
		"ports":         portList,
	}

	s.jsonResponse(w, result)
}

// handleEscalations returns escalation list
func (s *Server) handleEscalations(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := escalation.NewService(database)
	escalations, err := svc.List("", 50)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, toEscalationDTOs(escalations))
}

// Run starts the server (convenience function)
func Run(port int, projectRoot, dbPath string) error {
	// Check if static files exist
	if _, err := staticFiles.ReadFile("static/index.html"); err != nil {
		// Create default files if not embedded
		if err := createDefaultStaticFiles(); err != nil {
			log.Printf("Warning: Could not create static files: %v", err)
		}
	}

	config := Config{
		Port:        port,
		ProjectRoot: projectRoot,
		DBPath:      dbPath,
	}

	server := NewServer(config)
	return server.Start()
}

func createDefaultStaticFiles() error {
	// This is a fallback - normally files should be embedded
	return nil
}

// handleManifest returns manifest status
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	manifestSvc := manifest.NewService(database, s.config.ProjectRoot)
	statuses, err := manifestSvc.Status()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	// 상태별 분류
	result := map[string]interface{}{
		"project_root": s.config.ProjectRoot,
		"files":        statuses,
		"summary": map[string]int{
			"total":    len(statuses),
			"synced":   countByStatus(statuses, "synced"),
			"modified": countByStatus(statuses, "modified"),
			"new":      countByStatus(statuses, "new"),
			"deleted":  countByStatus(statuses, "deleted"),
		},
	}

	s.jsonResponse(w, result)
}

// handleManifestChanges returns manifest change history
func (s *Server) handleManifestChanges(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	manifestSvc := manifest.NewService(database, s.config.ProjectRoot)
	changes, err := manifestSvc.GetChanges(limit)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, changes)
}

// countByStatus counts files by status
func countByStatus(files []manifest.TrackedFile, status string) int {
	count := 0
	for _, f := range files {
		if string(f.Status) == status {
			count++
		}
	}
	return count
}
