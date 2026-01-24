package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SSEServer manages Server-Sent Events connections
type SSEServer struct {
	clients    map[string]*SSEClient
	register   chan *SSEClient
	unregister chan string
	broadcast  chan *Event
	mu         sync.RWMutex
	running    bool
}

// SSEClient represents a connected client
type SSEClient struct {
	ID        string
	Events    chan *Event
	Filters   []EventType  // Event types to receive (empty = all)
	SessionID string       // Only receive events for this session (empty = all)
	done      chan struct{}
}

// NewSSEServer creates a new SSE server
func NewSSEServer() *SSEServer {
	s := &SSEServer{
		clients:    make(map[string]*SSEClient),
		register:   make(chan *SSEClient),
		unregister: make(chan string),
		broadcast:  make(chan *Event, 100),
	}
	return s
}

// Start starts the SSE server event loop
func (s *SSEServer) Start() {
	s.running = true
	go s.run()
}

// Stop stops the SSE server
func (s *SSEServer) Stop() {
	s.running = false
	// Close all client connections
	s.mu.Lock()
	for _, client := range s.clients {
		close(client.done)
	}
	s.mu.Unlock()
}

func (s *SSEServer) run() {
	for s.running {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client.ID] = client
			s.mu.Unlock()

		case clientID := <-s.unregister:
			s.mu.Lock()
			if client, ok := s.clients[clientID]; ok {
				close(client.Events)
				delete(s.clients, clientID)
			}
			s.mu.Unlock()

		case event := <-s.broadcast:
			s.mu.RLock()
			for _, client := range s.clients {
				if s.shouldSend(client, event) {
					select {
					case client.Events <- event:
					default:
						// Client buffer full, skip
					}
				}
			}
			s.mu.RUnlock()
		}
	}
}

// shouldSend checks if client should receive this event
func (s *SSEServer) shouldSend(client *SSEClient, event *Event) bool {
	// Session filter
	if client.SessionID != "" && event.SessionID != "" && client.SessionID != event.SessionID {
		return false
	}

	// Event type filter
	if len(client.Filters) > 0 {
		for _, f := range client.Filters {
			if f == event.Type {
				return true
			}
		}
		return false
	}

	return true
}

// Broadcast sends an event to all connected clients
func (s *SSEServer) Broadcast(event *Event) {
	if !s.running {
		return
	}
	select {
	case s.broadcast <- event:
	default:
		// Buffer full, drop event
	}
}

// SendTo sends an event to a specific client
func (s *SSEServer) SendTo(clientID string, event *Event) {
	s.mu.RLock()
	client, ok := s.clients[clientID]
	s.mu.RUnlock()

	if ok {
		select {
		case client.Events <- event:
		default:
		}
	}
}

// ClientCount returns the number of connected clients
func (s *SSEServer) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// ServeHTTP handles SSE connections
func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse filters from query params
	filterParam := r.URL.Query().Get("filter")
	var filters []EventType
	if filterParam != "" {
		for _, f := range strings.Split(filterParam, ",") {
			filters = append(filters, EventType(strings.TrimSpace(f)))
		}
	}

	sessionID := r.URL.Query().Get("session_id")

	// Create client
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := &SSEClient{
		ID:        clientID,
		Events:    make(chan *Event, 50),
		Filters:   filters,
		SessionID: sessionID,
		done:      make(chan struct{}),
	}

	// Register client
	s.register <- client

	// Ensure cleanup on disconnect
	defer func() {
		s.unregister <- clientID
	}()

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	connectEvent := &Event{
		Type:      "connection:established",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"client_id": clientID,
			"filters":   filters,
		},
	}
	s.sendEvent(w, flusher, connectEvent)

	// Keep-alive ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Event loop
	for {
		select {
		case <-r.Context().Done():
			return

		case <-client.done:
			return

		case event := <-client.Events:
			s.sendEvent(w, flusher, event)

		case <-ticker.C:
			// Send keep-alive comment
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

func (s *SSEServer) sendEvent(w http.ResponseWriter, flusher http.Flusher, event *Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
