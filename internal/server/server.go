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
	"github.com/n0roo/pal-kit/internal/history"
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
	VaultPath   string // Knowledge Base vault path
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

	// History API (detailed event log)
	mux.HandleFunc("/api/history/events", s.withCORS(s.handleHistoryEvents))
	mux.HandleFunc("/api/history/types", s.withCORS(s.handleHistoryTypes))
	mux.HandleFunc("/api/history/projects", s.withCORS(s.handleHistoryProjects))
	mux.HandleFunc("/api/history/stats", s.withCORS(s.handleHistoryStats))
	mux.HandleFunc("/api/history/export", s.withCORS(s.handleHistoryExport))

	// Session visualization API
	mux.HandleFunc("/api/sessions/tree", s.withCORS(s.handleSessionTree))
	mux.HandleFunc("/api/ports/flow", s.withCORS(s.handlePortFlow))
	mux.HandleFunc("/api/ports/progress", s.withCORS(s.handlePortProgress))

	// v2 API routes
	s.RegisterV2Routes(mux)

	// Projects API routes
	s.RegisterProjectRoutes(mux)

	// KB API routes
	s.RegisterKBRoutes(mux)

	// SSE (Server-Sent Events) for real-time updates
	sseHub := NewSSEHub()
	go sseHub.Run()
	s.RegisterSSERoutes(mux, sseHub)

	// v2 Status endpoint
	mux.HandleFunc("/api/v2/status", s.withCORS(s.handleV2Status))

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("static files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	// Wrap entire mux with CORS middleware
	corsHandler := s.corsMiddleware(mux)

	s.srv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      corsHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second, // Increased for SSE
	}

	log.Printf("🚀 PAL Kit Dashboard running at http://localhost:%d", s.config.Port)
	log.Printf("📡 v2 API available at /api/v2/*")
	log.Printf("📁 Projects API available at /api/v2/projects/*")
	log.Printf("📚 KB API available at /api/v2/kb/*")
	log.Printf("🔔 SSE events at /api/v2/events")
	return s.srv.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

// corsMiddleware wraps a handler with CORS headers for all requests
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for all requests
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// withCORS adds CORS headers (kept for compatibility, but corsMiddleware handles it globally)
func (s *Server) withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS headers are now set by global middleware
		// Just handle OPTIONS for safety
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
	return RunWithConfig(Config{
		Port:        port,
		ProjectRoot: projectRoot,
		DBPath:      dbPath,
	})
}

// RunWithConfig starts the server with full configuration
func RunWithConfig(config Config) error {
	// Check if static files exist
	if _, err := staticFiles.ReadFile("static/index.html"); err != nil {
		// Create default files if not embedded
		if err := createDefaultStaticFiles(); err != nil {
			log.Printf("Warning: Could not create static files: %v", err)
		}
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

// handleHistoryEvents returns detailed event log
func (s *Server) handleHistoryEvents(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	// Parse filter parameters
	filter := history.Filter{}

	if v := r.URL.Query().Get("session_id"); v != "" {
		filter.SessionID = v
	}
	if v := r.URL.Query().Get("event_type"); v != "" {
		filter.EventType = v
	}
	if v := r.URL.Query().Get("project"); v != "" {
		filter.ProjectRoot = v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		filter.Search = v
	}
	if v := r.URL.Query().Get("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.StartDate = t
		}
	}
	if v := r.URL.Query().Get("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.EndDate = t.Add(24*time.Hour - time.Second) // End of day
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	svc := history.NewService(database)
	events, total, err := svc.List(filter)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, map[string]interface{}{
		"events": events,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// handleHistoryTypes returns available event types
func (s *Server) handleHistoryTypes(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := history.NewService(database)
	types, err := svc.GetEventTypes()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, types)
}

// handleHistoryProjects returns available projects
func (s *Server) handleHistoryProjects(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := history.NewService(database)
	projects, err := svc.GetProjects()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, projects)
}

// handleHistoryStats returns history statistics
func (s *Server) handleHistoryStats(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := history.NewService(database)
	stats, err := svc.GetStats()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, stats)
}

// handleHistoryExport exports history in JSON or CSV format
func (s *Server) handleHistoryExport(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	// Parse filter (same as handleHistoryEvents)
	filter := history.Filter{}
	if v := r.URL.Query().Get("session_id"); v != "" {
		filter.SessionID = v
	}
	if v := r.URL.Query().Get("event_type"); v != "" {
		filter.EventType = v
	}
	if v := r.URL.Query().Get("project"); v != "" {
		filter.ProjectRoot = v
	}
	if v := r.URL.Query().Get("search"); v != "" {
		filter.Search = v
	}
	if v := r.URL.Query().Get("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.StartDate = t
		}
	}
	if v := r.URL.Query().Get("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.EndDate = t.Add(24*time.Hour - time.Second)
		}
	}
	filter.Limit = 10000 // Max export limit

	svc := history.NewService(database)
	format := r.URL.Query().Get("format")

	switch format {
	case "csv":
		data, err := svc.ExportCSV(filter)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=history.csv")
		w.Write([]byte(data))

	default: // JSON
		data, err := svc.ExportJSON(filter)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=history.json")
		w.Write(data)
	}
}

// handleSessionTree returns hierarchical session tree
func (s *Server) handleSessionTree(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := session.NewService(database)

	// Get root sessions (sessions without parent)
	rootSessions, err := svc.GetRootSessions(50)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	// Build trees for each root session
	var trees []SessionTreeNodeDTO
	for _, root := range rootSessions {
		tree, err := svc.GetTree(root.ID)
		if err != nil {
			continue
		}
		trees = append(trees, toSessionTreeNodeDTO(*tree))
	}

	if trees == nil {
		trees = []SessionTreeNodeDTO{}
	}

	s.jsonResponse(w, map[string]interface{}{
		"sessions": trees,
	})
}

// handlePortFlow returns port dependency flow diagram data
func (s *Server) handlePortFlow(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	sessionID := r.URL.Query().Get("session")

	portSvc := port.NewService(database)
	docsSvc := docs.NewService(s.config.ProjectRoot)

	// Get ports
	var ports []port.Port
	if sessionID != "" {
		// Get ports for specific session
		ports, err = portSvc.ListBySession(sessionID)
	} else {
		ports, err = portSvc.List("", 100)
	}
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	// Convert to nodes
	var portNodes []PortNodeDTO
	for _, p := range ports {
		node := PortNodeDTO{
			ID:     p.ID,
			Status: p.Status,
		}
		if p.Title.Valid {
			node.Title = p.Title.String
		}
		if p.AgentID.Valid {
			node.Agent = p.AgentID.String
		}
		portNodes = append(portNodes, node)
	}

	// Get dependencies from document content
	var deps []PortDependencyDTO
	for _, p := range ports {
		// Try to read port document and parse dependencies
		docPath := fmt.Sprintf("ports/%s.md", p.ID)
		content, err := docsSvc.GetContent(docPath)
		if err != nil {
			continue
		}

		// Parse dependency from markdown metadata table
		// Look for: | 의존성 | value | pattern
		depStr := parseMarkdownDependency(content)
		if depStr != "" {
			depList := strings.Split(depStr, ",")
			for _, dep := range depList {
				dep = strings.TrimSpace(dep)
				if dep != "" && dep != "-" && dep != "없음" {
					deps = append(deps, PortDependencyDTO{
						From: dep,
						To:   p.ID,
					})
				}
			}
		}
	}

	if portNodes == nil {
		portNodes = []PortNodeDTO{}
	}
	if deps == nil {
		deps = []PortDependencyDTO{}
	}

	s.jsonResponse(w, PortFlowDTO{
		Ports:        portNodes,
		Dependencies: deps,
	})
}

// parseMarkdownDependency extracts dependency value from markdown content
func parseMarkdownDependency(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Look for dependency row in markdown table: | 의존성 | value |
		if strings.Contains(line, "의존성") && strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 3 {
				// parts[0] is empty, parts[1] is "의존성", parts[2] is value
				return strings.TrimSpace(parts[2])
			}
		}
	}
	return ""
}

// handlePortProgress returns ports grouped by status
func (s *Server) handlePortProgress(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	portSvc := port.NewService(database)
	ports, err := portSvc.List("", 100)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	result := PortProgressDTO{
		Completed:  []PortNodeDTO{},
		InProgress: []PortNodeDTO{},
		Pending:    []PortNodeDTO{},
	}

	for _, p := range ports {
		node := PortNodeDTO{
			ID:     p.ID,
			Status: p.Status,
		}
		if p.Title.Valid {
			node.Title = p.Title.String
		}
		if p.AgentID.Valid {
			node.Agent = p.AgentID.String
		}

		switch p.Status {
		case "complete", "done":
			result.Completed = append(result.Completed, node)
		case "running", "in_progress":
			result.InProgress = append(result.InProgress, node)
		default: // pending, draft, blocked, etc.
			result.Pending = append(result.Pending, node)
		}
	}

	s.jsonResponse(w, result)
}
