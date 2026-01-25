package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/n0roo/pal-kit/internal/agent"
	"github.com/n0roo/pal-kit/internal/agentv2"
	"github.com/n0roo/pal-kit/internal/attention"
	"github.com/n0roo/pal-kit/internal/config"
	"github.com/n0roo/pal-kit/internal/document"
	"github.com/n0roo/pal-kit/internal/handoff"
	"github.com/n0roo/pal-kit/internal/message"
	"github.com/n0roo/pal-kit/internal/orchestrator"
	"github.com/n0roo/pal-kit/internal/session"
)

// RegisterV2Routes registers v2 API routes
func (s *Server) RegisterV2Routes(mux *http.ServeMux) {
	// Orchestration API
	mux.HandleFunc("/api/v2/orchestrations", s.withCORS(s.handleOrchestrations))
	mux.HandleFunc("/api/v2/orchestrations/", s.withCORS(s.handleOrchestrationDetail))

	// Session Hierarchy API
	mux.HandleFunc("/api/v2/sessions/hierarchy", s.withCORS(s.handleSessionHierarchy))
	mux.HandleFunc("/api/v2/sessions/hierarchy/", s.withCORS(s.handleSessionHierarchyDetail))
	mux.HandleFunc("/api/v2/sessions/builds", s.withCORS(s.handleBuildSessions))

	// Attention API
	mux.HandleFunc("/api/v2/attention/", s.withCORS(s.handleAttention))
	mux.HandleFunc("/api/v2/attention", s.withCORS(s.handleAttentionList))

	// Handoff API
	mux.HandleFunc("/api/v2/handoffs", s.withCORS(s.handleHandoffs))
	mux.HandleFunc("/api/v2/handoffs/", s.withCORS(s.handleHandoffDetail))

	// Agent v2 API
	mux.HandleFunc("/api/v2/agents/global", s.withCORS(s.handleGlobalAgents))
	mux.HandleFunc("/api/v2/agents/global/", s.withCORS(s.handleGlobalAgentDetail))
	mux.HandleFunc("/api/v2/agents", s.withCORS(s.handleAgentsV2))
	mux.HandleFunc("/api/v2/agents/", s.withCORS(s.handleAgentV2Detail))

	// Message API
	mux.HandleFunc("/api/v2/messages", s.withCORS(s.handleMessages))
	mux.HandleFunc("/api/v2/messages/", s.withCORS(s.handleMessageDetail))

	// Worker Sessions API
	mux.HandleFunc("/api/v2/workers", s.withCORS(s.handleWorkerSessions))
	mux.HandleFunc("/api/v2/workers/", s.withCORS(s.handleWorkerSessionDetail))

	// Document API (order matters: specific routes before wildcard)
	mux.HandleFunc("/api/v2/documents/stats", s.withCORS(s.handleDocumentStats))
	mux.HandleFunc("/api/v2/documents/index", s.withCORS(s.handleDocumentIndex))
	mux.HandleFunc("/api/v2/documents/tree", s.withCORS(s.handleDocumentTree))
	mux.HandleFunc("/api/v2/documents/", s.withCORS(s.handleDocumentDetail))
	mux.HandleFunc("/api/v2/documents", s.withCORS(s.handleDocumentsV2))
}

// ========================================
// Orchestration Handlers
// ========================================

func (s *Server) handleOrchestrations(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	orchSvc := orchestrator.NewService(database, sessionSvc, msgStore)

	switch r.Method {
	case "GET":
		status := r.URL.Query().Get("status")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 20
		}

		orchestrations, err := orchSvc.ListOrchestrations(orchestrator.OrchestrationStatus(status), limit)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, orchestrations)

	case "POST":
		var req struct {
			Title       string                       `json:"title"`
			Description string                       `json:"description"`
			Ports       []orchestrator.AtomicPort    `json:"ports"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.errorResponse(w, 400, "Invalid request body")
			return
		}

		orch, err := orchSvc.CreateOrchestration(req.Title, req.Description, req.Ports)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, orch)

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) handleOrchestrationDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v2/orchestrations/")
	if id == "" {
		s.errorResponse(w, 400, "Orchestration ID required")
		return
	}

	// Check for sub-resources
	parts := strings.Split(id, "/")
	id = parts[0]
	subResource := ""
	if len(parts) > 1 {
		subResource = parts[1]
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	orchSvc := orchestrator.NewService(database, sessionSvc, msgStore)

	switch subResource {
	case "stats":
		stats, err := orchSvc.GetOrchestrationStats(id)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, stats)

	case "workers":
		workers, err := orchSvc.ListWorkerSessions(id)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, workers)

	case "start":
		if r.Method != "POST" {
			s.errorResponse(w, 405, "Method not allowed")
			return
		}
		var req struct {
			OperatorSessionID string `json:"operator_session_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if err := orchSvc.StartOrchestration(id, req.OperatorSessionID); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, map[string]string{"status": "started"})

	default:
		orch, err := orchSvc.GetOrchestration(id)
		if err != nil {
			s.errorResponse(w, 404, err.Error())
			return
		}
		s.jsonResponse(w, orch)
	}
}

// ========================================
// Session Hierarchy Handlers
// ========================================

func (s *Server) handleSessionHierarchy(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := session.NewService(database)

	// Get root sessions (build type OR no parent)
	builds, err := svc.GetRootHierarchicalSessions(false, 20)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, builds)
}

func (s *Server) handleSessionHierarchyDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v2/sessions/hierarchy/")
	if id == "" {
		s.errorResponse(w, 400, "Session ID required")
		return
	}

	parts := strings.Split(id, "/")
	id = parts[0]
	subResource := ""
	if len(parts) > 1 {
		subResource = parts[1]
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := session.NewService(database)

	switch subResource {
	case "tree":
		tree, err := svc.GetSessionHierarchy(id)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, tree)

	case "list":
		sessions, err := svc.ListByRoot(id)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, sessions)

	case "stats":
		stats, err := svc.GetHierarchyStats(id)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, stats)

	default:
		sess, err := svc.GetHierarchical(id)
		if err != nil {
			s.errorResponse(w, 404, err.Error())
			return
		}
		s.jsonResponse(w, sess)
	}
}

func (s *Server) handleBuildSessions(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	svc := session.NewService(database)
	activeOnly := r.URL.Query().Get("active") == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	builds, err := svc.GetBuildSessions(activeOnly, limit)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, builds)
}

// ========================================
// Attention Handlers
// ========================================

func (s *Server) handleAttentionList(w http.ResponseWriter, r *http.Request) {
	// This would need a list method - for now return error
	s.errorResponse(w, 400, "Session ID required in path")
}

func (s *Server) handleAttention(w http.ResponseWriter, r *http.Request) {
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v2/attention/")
	if sessionID == "" {
		s.errorResponse(w, 400, "Session ID required")
		return
	}

	parts := strings.Split(sessionID, "/")
	sessionID = parts[0]
	subResource := ""
	if len(parts) > 1 {
		subResource = parts[1]
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	store := attention.NewStore(database.DB)

	switch subResource {
	case "report":
		report, err := store.GenerateReport(sessionID)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, report)

	case "history":
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 10
		}
		events, err := store.GetCompactHistory(sessionID, limit)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, events)

	case "init":
		if r.Method != "POST" {
			s.errorResponse(w, 405, "Method not allowed")
			return
		}
		var req struct {
			PortID      string `json:"port_id"`
			TokenBudget int    `json:"token_budget"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if err := store.Initialize(sessionID, req.PortID, req.TokenBudget); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, map[string]string{"status": "initialized"})

	default:
		att, err := store.Get(sessionID)
		if err != nil {
			s.errorResponse(w, 404, err.Error())
			return
		}

		// Add status
		status := attention.CalculateStatus(att)
		response := map[string]interface{}{
			"attention": att,
			"status":    status,
		}
		s.jsonResponse(w, response)
	}
}

// ========================================
// Handoff Handlers
// ========================================

func (s *Server) handleHandoffs(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	store := handoff.NewStore(database)

	switch r.Method {
	case "GET":
		portID := r.URL.Query().Get("port")
		direction := r.URL.Query().Get("direction")

		if portID == "" {
			s.errorResponse(w, 400, "port parameter required")
			return
		}

		var handoffs []*handoff.Handoff
		switch direction {
		case "from":
			handoffs, err = store.GetFromPort(portID)
		case "to":
			handoffs, err = store.GetForPort(portID)
		default:
			from, _ := store.GetFromPort(portID)
			to, _ := store.GetForPort(portID)
			handoffs = append(from, to...)
		}

		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, handoffs)

	case "POST":
		var req struct {
			FromPortID string      `json:"from_port_id"`
			ToPortID   string      `json:"to_port_id"`
			Type       string      `json:"type"`
			Content    interface{} `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.errorResponse(w, 400, "Invalid request body")
			return
		}

		ho, err := store.Create(req.FromPortID, req.ToPortID, handoff.HandoffType(req.Type), req.Content)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, ho)

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) handleHandoffDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v2/handoffs/")
	if id == "" {
		s.errorResponse(w, 400, "Handoff ID required")
		return
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	store := handoff.NewStore(database)

	// Check for estimate endpoint
	if id == "estimate" {
		var req struct {
			Content interface{} `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.errorResponse(w, 400, "Invalid request body")
			return
		}

		tokens, err := handoff.EstimateTokens(req.Content)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}

		s.jsonResponse(w, map[string]interface{}{
			"tokens":  tokens,
			"budget":  handoff.MaxTokenBudget,
			"percent": float64(tokens) / float64(handoff.MaxTokenBudget) * 100,
			"valid":   tokens <= handoff.MaxTokenBudget,
		})
		return
	}

	ho, err := store.Get(id)
	if err != nil {
		s.errorResponse(w, 404, err.Error())
		return
	}
	s.jsonResponse(w, ho)
}

// ========================================
// Agent v2 Handlers
// ========================================

func (s *Server) handleAgentsV2(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	store := agentv2.NewStore(database.DB)

	switch r.Method {
	case "GET":
		agentType := r.URL.Query().Get("type")
		includeSystem := r.URL.Query().Get("include_system") == "true"

		// Get project agents from DB
		agents, err := store.ListAgents(agentv2.AgentType(agentType))
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}

		// Add system agents if requested
		if includeSystem {
			systemAgents := s.getSystemAgents(agentType)
			for _, sa := range systemAgents {
				agents = append(agents, sa)
			}
		}

		s.jsonResponse(w, agents)

	case "POST":
		var req struct {
			Name         string   `json:"name"`
			Type         string   `json:"type"`
			Description  string   `json:"description"`
			Capabilities []string `json:"capabilities"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.errorResponse(w, 400, "Invalid request body")
			return
		}

		agent := &agentv2.Agent{
			Name:         req.Name,
			Type:         agentv2.AgentType(req.Type),
			Description:  req.Description,
			Capabilities: req.Capabilities,
		}

		if err := store.CreateAgent(agent); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, agent)

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

// getSystemAgents returns embedded system agent templates
func (s *Server) getSystemAgents(filterType string) []*agentv2.Agent {
	var agents []*agentv2.Agent

	// Core agents
	coreAgents := []struct {
		name     string
		agentType string
		desc     string
	}{
		{"architect", "spec", "시스템 아키텍처 설계 및 기술 결정"},
		{"builder", "operator", "파이프라인 실행 및 빌드 관리"},
		{"docs", "worker", "문서화 및 API 문서 생성"},
		{"operator", "operator", "작업 조율 및 태스크 배분"},
		{"planner", "spec", "작업 계획 수립 및 분해"},
		{"reviewer", "worker", "코드 리뷰 및 품질 검증"},
		{"support", "worker", "이슈 해결 및 기술 지원"},
	}

	for _, ca := range coreAgents {
		if filterType != "" && ca.agentType != filterType {
			continue
		}
		agents = append(agents, &agentv2.Agent{
			ID:          "system-" + ca.name,
			Name:        ca.name,
			Type:        agentv2.AgentType(ca.agentType),
			Description: ca.desc,
			Capabilities: []string{"system"},
		})
	}

	// Backend workers
	backendWorkers := []string{"cache", "document", "entity", "router", "service", "test"}
	for _, name := range backendWorkers {
		if filterType != "" && filterType != "worker" {
			continue
		}
		agents = append(agents, &agentv2.Agent{
			ID:          "system-backend-" + name,
			Name:        "backend/" + name,
			Type:        agentv2.TypeWorker,
			Description: "Backend " + name + " worker",
			Capabilities: []string{"system", "backend"},
		})
	}

	// Frontend workers
	frontendWorkers := []string{"e2e", "engineer", "model", "ui", "unit-tc"}
	for _, name := range frontendWorkers {
		if filterType != "" && filterType != "worker" {
			continue
		}
		agents = append(agents, &agentv2.Agent{
			ID:          "system-frontend-" + name,
			Name:        "frontend/" + name,
			Type:        agentv2.TypeWorker,
			Description: "Frontend " + name + " worker",
			Capabilities: []string{"system", "frontend"},
		})
	}

	return agents
}

func (s *Server) handleAgentV2Detail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v2/agents/")
	if id == "" {
		s.errorResponse(w, 400, "Agent ID required")
		return
	}

	parts := strings.Split(id, "/")
	id = parts[0]
	subResource := ""
	if len(parts) > 1 {
		subResource = parts[1]
	}

	// Handle system agents
	if strings.HasPrefix(id, "system-") {
		s.handleSystemAgentDetail(w, r, id, subResource)
		return
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	store := agentv2.NewStore(database.DB)

	switch subResource {
	case "spec":
		// Get agent spec content from current version
		version, err := store.GetCurrentVersion(id)
		if err != nil {
			s.errorResponse(w, 404, err.Error())
			return
		}
		s.jsonResponse(w, map[string]interface{}{
			"id":      id,
			"version": version.Version,
			"content": version.SpecContent,
		})

	case "versions":
		if r.Method == "POST" {
			// Create new version
			var req struct {
				SpecContent   string `json:"spec_content"`
				ChangeSummary string `json:"change_summary"`
				ChangeReason  string `json:"change_reason"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				s.errorResponse(w, 400, "Invalid request body")
				return
			}

			version := &agentv2.AgentVersion{
				AgentID:       id,
				SpecContent:   req.SpecContent,
				ChangeSummary: req.ChangeSummary,
				ChangeReason:  req.ChangeReason,
			}

			if err := store.CreateVersion(version); err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, version)
			return
		}

		versions, err := store.ListVersions(id)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, versions)

	case "stats":
		versionStr := r.URL.Query().Get("version")
		version, _ := strconv.Atoi(versionStr)
		if version == 0 {
			// Get current version
			agent, err := store.GetAgent(id)
			if err != nil {
				s.errorResponse(w, 404, err.Error())
				return
			}
			version = agent.CurrentVersion
		}

		stats, err := store.GetVersionStats(id, version)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, stats)

	case "compare":
		v1, _ := strconv.Atoi(r.URL.Query().Get("v1"))
		v2, _ := strconv.Atoi(r.URL.Query().Get("v2"))
		if v1 == 0 || v2 == 0 {
			s.errorResponse(w, 400, "v1 and v2 parameters required")
			return
		}

		comparison, err := store.CompareVersions(id, v1, v2)
		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, comparison)

	default:
		agent, err := store.GetAgent(id)
		if err != nil {
			// Try by name
			agent, err = store.GetAgentByName(id)
			if err != nil {
				s.errorResponse(w, 404, err.Error())
				return
			}
		}
		s.jsonResponse(w, agent)
	}
}

// ========================================
// Message Handlers
// ========================================

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	store := message.NewStore(database.DB)

	switch r.Method {
	case "GET":
		conversationID := r.URL.Query().Get("conversation")
		sessionID := r.URL.Query().Get("session")

		if conversationID != "" {
			messages, err := store.GetByConversation(conversationID)
			if err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, messages)
			return
		}

		if sessionID != "" {
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit <= 0 {
				limit = 10
			}
			messages, err := store.Receive(sessionID, limit)
			if err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, messages)
			return
		}

		s.errorResponse(w, 400, "conversation or session parameter required")

	case "POST":
		var msg message.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			s.errorResponse(w, 400, "Invalid request body")
			return
		}

		if err := store.Send(&msg); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, msg)

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) handleMessageDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v2/messages/")
	if id == "" {
		s.errorResponse(w, 400, "Message ID required")
		return
	}

	parts := strings.Split(id, "/")
	messageID := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	store := message.NewStore(database.DB)

	switch action {
	case "delivered":
		if r.Method != "POST" {
			s.errorResponse(w, 405, "Method not allowed")
			return
		}
		if err := store.MarkDelivered(messageID); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, map[string]string{"status": "delivered"})

	case "processed":
		if r.Method != "POST" {
			s.errorResponse(w, 405, "Method not allowed")
			return
		}
		if err := store.MarkProcessed(messageID); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, map[string]string{"status": "processed"})

	default:
		s.errorResponse(w, 404, "Message not found or action not supported")
	}
}

// ========================================
// Worker Session Handlers
// ========================================

func (s *Server) handleWorkerSessions(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	orchSvc := orchestrator.NewService(database, sessionSvc, msgStore)

	orchestrationID := r.URL.Query().Get("orchestration")
	if orchestrationID == "" {
		s.errorResponse(w, 400, "orchestration parameter required")
		return
	}

	workers, err := orchSvc.ListWorkerSessions(orchestrationID)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, workers)
}

func (s *Server) handleWorkerSessionDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v2/workers/")
	if id == "" {
		s.errorResponse(w, 400, "Worker session ID required")
		return
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	orchSvc := orchestrator.NewService(database, sessionSvc, msgStore)

	ws, err := orchSvc.GetWorkerSession(id)
	if err != nil {
		s.errorResponse(w, 404, err.Error())
		return
	}

	s.jsonResponse(w, ws)
}

// ========================================
// Summary Endpoint
// ========================================

func (s *Server) handleV2Status(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	status := map[string]interface{}{}

	// Orchestrations
	sessionSvc := session.NewService(database)
	msgStore := message.NewStore(database.DB)
	orchSvc := orchestrator.NewService(database, sessionSvc, msgStore)

	if orchestrations, err := orchSvc.ListOrchestrations("", 100); err == nil {
		running := 0
		for _, o := range orchestrations {
			if o.Status == orchestrator.StatusRunning {
				running++
			}
		}
		status["orchestrations"] = map[string]int{
			"total":   len(orchestrations),
			"running": running,
		}
	}

	// Build sessions
	if builds, err := sessionSvc.GetBuildSessions(false, 100); err == nil {
		active := 0
		for _, b := range builds {
			if b.Session.Status == "running" {
				active++
			}
		}
		status["builds"] = map[string]int{
			"total":  len(builds),
			"active": active,
		}
	}

	// Agents
	agentStore := agentv2.NewStore(database.DB)
	if agents, err := agentStore.ListAgents(""); err == nil {
		status["agents"] = map[string]int{
			"total": len(agents),
		}
	}

	s.jsonResponse(w, status)
}

// ========================================
// Document Handlers
// ========================================

func (s *Server) handleDocumentsV2(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	docSvc := document.NewService(database, s.config.ProjectRoot)

	// Parse query parameters
	query := r.URL.Query().Get("q")
	docType := r.URL.Query().Get("type")
	domain := r.URL.Query().Get("domain")
	status := r.URL.Query().Get("status")
	tag := r.URL.Query().Get("tag")
	limitStr := r.URL.Query().Get("limit")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	filters := document.SearchFilters{
		Type:   docType,
		Domain: domain,
		Status: status,
		Tag:    tag,
		Limit:  limit,
	}

	docs, err := docSvc.Search(query, filters)
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	// Convert to DTO
	result := make([]map[string]interface{}, 0, len(docs))
	for _, d := range docs {
		item := map[string]interface{}{
			"id":         d.ID,
			"path":       d.Path,
			"type":       d.Type,
			"domain":     d.Domain,
			"status":     d.Status,
			"priority":   d.Priority,
			"tokens":     d.Tokens,
			"tags":       d.Tags,
			"created_at": d.CreatedAt,
			"updated_at": d.UpdatedAt,
		}
		if d.Summary.Valid {
			item["summary"] = d.Summary.String
		}
		result = append(result, item)
	}

	s.jsonResponse(w, result)
}

func (s *Server) handleDocumentStats(w http.ResponseWriter, r *http.Request) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	docSvc := document.NewService(database, s.config.ProjectRoot)
	stats, err := docSvc.GetStats()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, stats)
}

func (s *Server) handleDocumentIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.errorResponse(w, 405, "Method not allowed")
		return
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	docSvc := document.NewService(database, s.config.ProjectRoot)
	result, err := docSvc.Index()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}

	s.jsonResponse(w, map[string]interface{}{
		"added":   result.Added,
		"updated": result.Updated,
		"removed": result.Removed,
		"errors":  result.Errors,
	})
}

func (s *Server) handleDocumentDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v2/documents/")
	if id == "" {
		s.errorResponse(w, 400, "Document ID required")
		return
	}

	// Handle special routes that ServeMux might not match correctly
	if id == "tree" {
		s.handleDocumentTree(w, r)
		return
	}

	// Check for content sub-resource
	if strings.HasSuffix(id, "/content") {
		id = strings.TrimSuffix(id, "/content")
		s.handleDocumentContent(w, r, id)
		return
	}

	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	docSvc := document.NewService(database, s.config.ProjectRoot)
	doc, err := docSvc.Get(id)
	if err != nil {
		s.errorResponse(w, 404, err.Error())
		return
	}

	result := map[string]interface{}{
		"id":         doc.ID,
		"path":       doc.Path,
		"type":       doc.Type,
		"domain":     doc.Domain,
		"status":     doc.Status,
		"priority":   doc.Priority,
		"tokens":     doc.Tokens,
		"tags":       doc.Tags,
		"created_at": doc.CreatedAt,
		"updated_at": doc.UpdatedAt,
	}
	if doc.Summary.Valid {
		result["summary"] = doc.Summary.String
	}

	// Get links
	linksFrom, _ := docSvc.GetLinksFrom(id)
	linksTo, _ := docSvc.GetLinksTo(id)
	result["links_from"] = linksFrom
	result["links_to"] = linksTo

	s.jsonResponse(w, result)
}

func (s *Server) handleDocumentContent(w http.ResponseWriter, r *http.Request, id string) {
	database, err := s.getDB()
	if err != nil {
		s.errorResponse(w, 500, err.Error())
		return
	}
	defer database.Close()

	docSvc := document.NewService(database, s.config.ProjectRoot)
	content, err := docSvc.GetContent(id)
	if err != nil {
		s.errorResponse(w, 404, err.Error())
		return
	}

	s.jsonResponse(w, map[string]string{
		"id":      id,
		"content": content,
	})
}

// DocumentTreeNode represents a node in the document tree
type DocumentTreeNode struct {
	Name     string              `json:"name"`
	Path     string              `json:"path"`
	Type     string              `json:"type"` // "file" or "directory"
	DocType  string              `json:"doc_type,omitempty"`
	Children []*DocumentTreeNode `json:"children,omitempty"`
}

func (s *Server) handleDocumentTree(w http.ResponseWriter, r *http.Request) {
	root := r.URL.Query().Get("root")
	if root == "" {
		root = "."
	}

	depthStr := r.URL.Query().Get("depth")
	maxDepth := 3
	if depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 {
			maxDepth = d
		}
	}

	basePath := filepath.Join(s.config.ProjectRoot, root)

	// Check if path exists
	info, err := os.Stat(basePath)
	if err != nil {
		s.errorResponse(w, 404, "Path not found")
		return
	}
	if !info.IsDir() {
		s.errorResponse(w, 400, "Path is not a directory")
		return
	}

	// Build tree
	tree := s.buildDocumentTree(basePath, root, 0, maxDepth)

	s.jsonResponse(w, tree)
}

func (s *Server) buildDocumentTree(absPath, relPath string, depth, maxDepth int) *DocumentTreeNode {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil
	}

	node := &DocumentTreeNode{
		Name: info.Name(),
		Path: relPath,
	}

	if info.IsDir() {
		node.Type = "directory"

		if depth < maxDepth {
			entries, err := os.ReadDir(absPath)
			if err == nil {
				for _, entry := range entries {
					// Skip hidden files/dirs
					if strings.HasPrefix(entry.Name(), ".") && entry.Name() != ".pal" {
						continue
					}

					childPath := filepath.Join(absPath, entry.Name())
					childRelPath := filepath.Join(relPath, entry.Name())

					child := s.buildDocumentTree(childPath, childRelPath, depth+1, maxDepth)
					if child != nil {
						node.Children = append(node.Children, child)
					}
				}
			}
		}
	} else {
		node.Type = "file"

		// Determine doc type based on extension and path
		ext := strings.ToLower(filepath.Ext(absPath))
		switch ext {
		case ".md":
			if strings.Contains(relPath, "ports") {
				node.DocType = "port"
			} else if strings.Contains(relPath, "conventions") {
				node.DocType = "convention"
			} else if strings.Contains(relPath, "docs") {
				node.DocType = "docs"
			} else if strings.Contains(relPath, "decisions") {
				node.DocType = "adr"
			} else if strings.Contains(relPath, "sessions") {
				node.DocType = "session"
			} else {
				node.DocType = "markdown"
			}
		case ".yaml", ".yml":
			if strings.Contains(relPath, "agents") {
				node.DocType = "agent"
			} else {
				node.DocType = "yaml"
			}
		default:
			node.DocType = "other"
		}
	}

	return node
}

// ========================================
// Global Agent Handlers
// ========================================

func (s *Server) handleGlobalAgents(w http.ResponseWriter, r *http.Request) {
	globalPath := config.GlobalDir()
	store := agent.NewGlobalAgentStore(globalPath)

	switch r.Method {
	case "GET":
		// List global agents
		agentType := r.URL.Query().Get("type") // agents, skills, conventions

		var result interface{}
		var err error

		switch agentType {
		case "skills":
			result, err = store.ListSkills()
		case "conventions":
			result, err = store.ListConventions()
		default:
			result, err = store.List()
		}

		if err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, result)

	case "POST":
		// Handle actions
		action := r.URL.Query().Get("action")

		switch action {
		case "init":
			force := r.URL.Query().Get("force") == "true"
			if err := store.Initialize(force); err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, map[string]string{"status": "initialized"})

		case "sync":
			var req struct {
				ProjectRoot    string `json:"project_root"`
				ForceOverwrite bool   `json:"force_overwrite"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				s.errorResponse(w, 400, "Invalid request body")
				return
			}

			if req.ProjectRoot == "" {
				req.ProjectRoot = s.config.ProjectRoot
			}

			count, err := store.SyncToProject(req.ProjectRoot, req.ForceOverwrite)
			if err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, map[string]interface{}{
				"status": "synced",
				"count":  count,
			})

		case "reset":
			if err := store.Initialize(true); err != nil {
				s.errorResponse(w, 500, err.Error())
				return
			}
			s.jsonResponse(w, map[string]string{"status": "reset"})

		default:
			s.errorResponse(w, 400, "Unknown action. Use: init, sync, reset")
		}

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) handleGlobalAgentDetail(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/agents/global/")
	if path == "" {
		s.errorResponse(w, 400, "Path required")
		return
	}

	// Check for special endpoints
	if path == "manifest" {
		s.handleGlobalManifest(w, r)
		return
	}
	if path == "path" {
		s.jsonResponse(w, map[string]string{"path": config.GlobalDir()})
		return
	}

	globalPath := config.GlobalDir()
	store := agent.NewGlobalAgentStore(globalPath)

	switch r.Method {
	case "GET":
		content, err := store.Read(path)
		if err != nil {
			s.errorResponse(w, 404, err.Error())
			return
		}

		// Return with metadata
		s.jsonResponse(w, map[string]interface{}{
			"path":    path,
			"content": string(content),
		})

	case "PUT":
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.errorResponse(w, 400, "Invalid request body")
			return
		}

		if err := store.Write(path, []byte(req.Content)); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, map[string]string{"status": "updated"})

	case "DELETE":
		if err := store.Delete(path); err != nil {
			s.errorResponse(w, 500, err.Error())
			return
		}
		s.jsonResponse(w, map[string]string{"status": "deleted"})

	default:
		s.errorResponse(w, 405, "Method not allowed")
	}
}

func (s *Server) handleGlobalManifest(w http.ResponseWriter, r *http.Request) {
	globalPath := config.GlobalDir()
	store := agent.NewGlobalAgentStore(globalPath)

	manifest, err := store.GetManifest()
	if err != nil {
		s.errorResponse(w, 404, err.Error())
		return
	}
	s.jsonResponse(w, manifest)
}

// handleSystemAgentDetail handles requests for system agent details
func (s *Server) handleSystemAgentDetail(w http.ResponseWriter, r *http.Request, id string, subResource string) {
	// Parse system agent ID: system-{name} or system-{category}-{name}
	parts := strings.TrimPrefix(id, "system-")
	
	var templatePath string
	var agentName string
	var agentType agentv2.AgentType
	var description string
	var capabilities []string
	
	if strings.HasPrefix(parts, "backend-") {
		name := strings.TrimPrefix(parts, "backend-")
		agentName = "backend/" + name
		templatePath = "agents/workers/backend/" + name + ".yaml"
		agentType = agentv2.TypeWorker
		description = "Backend " + name + " worker"
		capabilities = []string{"system", "backend"}
	} else if strings.HasPrefix(parts, "frontend-") {
		name := strings.TrimPrefix(parts, "frontend-")
		agentName = "frontend/" + name
		templatePath = "agents/workers/frontend/" + name + ".yaml"
		agentType = agentv2.TypeWorker
		description = "Frontend " + name + " worker"
		capabilities = []string{"system", "frontend"}
	} else {
		// Core agent
		agentName = parts
		templatePath = "agents/core/" + parts + ".yaml"
		
		// Determine type based on name
		switch parts {
		case "architect", "planner":
			agentType = agentv2.TypeSpec
		case "builder", "operator":
			agentType = agentv2.TypeOperator
		default:
			agentType = agentv2.TypeWorker
		}
		
		descriptions := map[string]string{
			"architect": "시스템 아키텍처 설계 및 기술 결정",
			"builder":   "파이프라인 실행 및 빌드 관리",
			"docs":      "문서화 및 API 문서 생성",
			"operator":  "작업 조율 및 태스크 배분",
			"planner":   "작업 계획 수립 및 분해",
			"reviewer":  "코드 리뷰 및 품질 검증",
			"support":   "이슈 해결 및 기술 지원",
		}
		description = descriptions[parts]
		if description == "" {
			description = "System " + parts + " agent"
		}
		capabilities = []string{"system"}
	}
	
	switch subResource {
	case "spec":
		// Get YAML spec
		yamlContent, err := agent.GetTemplate(templatePath)
		if err != nil {
			s.errorResponse(w, 404, "Template not found: "+templatePath)
			return
		}
		
		// Also try to get rules.md
		rulesPath := strings.TrimSuffix(templatePath, ".yaml") + ".rules.md"
		rulesContent, _ := agent.GetTemplate(rulesPath)
		
		var content string
		if len(rulesContent) > 0 {
			content = string(yamlContent) + "\n---\n" + string(rulesContent)
		} else {
			content = string(yamlContent)
		}
		
		s.jsonResponse(w, map[string]interface{}{
			"id":      id,
			"version": 1,
			"content": content,
		})
		
	default:
		// Return agent metadata
		s.jsonResponse(w, map[string]interface{}{
			"id":              id,
			"name":            agentName,
			"type":            agentType,
			"description":     description,
			"capabilities":    capabilities,
			"current_version": 1,
			"is_system":       true,
		})
	}
}
