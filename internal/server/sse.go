package server

import (
	"net/http"

	"github.com/n0roo/pal-kit/internal/server/events"
)

// SSEHub wraps the SSE server for compatibility
type SSEHub struct {
	server *events.SSEServer
}

// NewSSEHub creates a new SSE hub
func NewSSEHub() *SSEHub {
	return &SSEHub{
		server: events.NewSSEServer(),
	}
}

// Run starts the SSE hub
func (h *SSEHub) Run() {
	h.server.Start()

	// Also register with global publisher
	publisher := events.GetPublisher()
	publisher.SetSSEServer(h.server)
}

// Stop stops the SSE hub
func (h *SSEHub) Stop() {
	h.server.Stop()
}

// Broadcast sends an event to all clients
func (h *SSEHub) Broadcast(event *events.Event) {
	h.server.Broadcast(event)
}

// ClientCount returns number of connected clients
func (h *SSEHub) ClientCount() int {
	return h.server.ClientCount()
}

// ServeHTTP implements http.Handler
func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.server.ServeHTTP(w, r)
}

// RegisterSSERoutes registers SSE routes
func (s *Server) RegisterSSERoutes(mux *http.ServeMux, hub *SSEHub) {
	// Main SSE stream endpoint
	mux.Handle("/api/v2/events", hub)
	mux.Handle("/api/v2/events/stream", hub)

	// SSE status endpoint
	mux.HandleFunc("/api/v2/events/status", s.withCORS(func(w http.ResponseWriter, r *http.Request) {
		s.jsonResponse(w, map[string]interface{}{
			"connected_clients": hub.ClientCount(),
			"status":            "running",
		})
	}))

	// Test endpoint to emit events (for development)
	mux.HandleFunc("/api/v2/events/test", s.withCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			s.errorResponse(w, 405, "Method not allowed")
			return
		}

		// Emit a test event
		publisher := events.GetPublisher()
		publisher.Publish(events.NewEvent("test:ping", map[string]interface{}{
			"message": "Test event",
		}))

		s.jsonResponse(w, map[string]string{"status": "event sent"})
	}))
}
