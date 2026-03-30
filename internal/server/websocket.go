package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// Message types for WebSocket protocol
const (
	WSMsgTypeAuth         = "auth"
	WSMsgTypeAuthSuccess  = "auth_success"
	WSMsgTypeAuthError    = "auth_error"
	WSMsgTypeSubscribe    = "subscribe"
	WSMsgTypeUnsubscribe  = "unsubscribe"
	WSMsgTypeEvent        = "event"
	WSMsgTypePing         = "ping"
	WSMsgTypePong         = "pong"
	WSMsgTypeSyncRequest  = "sync_request"
	WSMsgTypeSyncResponse = "sync_response"
	WSMsgTypeError        = "error"
)

// WebSocketMessage represents a message in the WebSocket protocol
type WebSocketMessage struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Error     string          `json:"error,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// WebSocketEvent represents a real-time event
type WebSocketEvent struct {
	Type      string      `json:"type"`
	UserID    string      `json:"user_id,omitempty"`
	FileID    string      `json:"file_id,omitempty"`
	FolderID  string      `json:"folder_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID                string
	UserID            string
	Conn              WebSocketConn
	Server            *WebSocketServer
	Send              chan *WebSocketMessage
	SubscribedFolders map[string]bool
	mu                sync.RWMutex
}

// WebSocketConn interface for WebSocket connections (allows testing)
type WebSocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	WriteControl(messageType int, data []byte, deadline time.Time) error
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetPongHandler(h func(appData string) error)
	SetPingHandler(h func(appData string) error)
}

// WebSocketServer manages WebSocket connections and event broadcasting
type WebSocketServer struct {
	vfs        *vfs.VFS
	db         *db.Manager
	jwtSecret  []byte

	clients    map[string]*WebSocketClient
	clientsMu  sync.RWMutex

	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan *WebSocketEvent

	upgrader   WebSocketUpgrader
}

// WebSocketUpgrader interface for upgrading HTTP to WebSocket
type WebSocketUpgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (WebSocketConn, error)
}

// defaultUpgrader wraps the actual WebSocket upgrade logic
type defaultUpgrader struct{}

func (u *defaultUpgrader) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (WebSocketConn, error) {
	return nil, fmt.Errorf("WebSocket support requires gorilla/websocket library")
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(vfsService *vfs.VFS, database *db.Manager, jwtSecret []byte) *WebSocketServer {
	s := &WebSocketServer{
		vfs:        vfsService,
		db:         database,
		jwtSecret:  jwtSecret,
		clients:    make(map[string]*WebSocketClient),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan *WebSocketEvent, 256),
		upgrader:   &defaultUpgrader{},
	}

	go s.run()

	return s
}

// SetUpgrader sets a custom WebSocket upgrader (for testing or different impl)
func (s *WebSocketServer) SetUpgrader(upgrader WebSocketUpgrader) {
	s.upgrader = upgrader
}

// RegisterRoutes registers WebSocket routes
func (s *WebSocketServer) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// WebSocket endpoint
	mux.Handle("GET /ws", auth.RequireAuth(http.HandlerFunc(s.handleWebSocket)))
}

// handleWebSocket handles WebSocket upgrade requests
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create client
	client := &WebSocketClient{
		ID:                generateClientID(),
		UserID:            userID,
		Conn:              conn,
		Server:            s,
		Send:              make(chan *WebSocketMessage, 256),
		SubscribedFolders: make(map[string]bool),
	}

	// Register client
	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// run is the main event loop for the WebSocket server
func (s *WebSocketServer) run() {
	for {
		select {
		case client := <-s.register:
			s.clientsMu.Lock()
			s.clients[client.ID] = client
			s.clientsMu.Unlock()

		case client := <-s.unregister:
			s.clientsMu.Lock()
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				close(client.Send)
			}
			s.clientsMu.Unlock()

		case event := <-s.broadcast:
			s.broadcastEvent(event)
		}
	}
}

// broadcastEvent sends an event to all subscribed clients
func (s *WebSocketServer) broadcastEvent(event *WebSocketEvent) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	message := &WebSocketMessage{
		Type:      WSMsgTypeEvent,
		Payload:   mustJSON(event),
		Timestamp: time.Now().Unix(),
	}

	for _, client := range s.clients {
		// Skip if event is for specific user and doesn't match
		if event.UserID != "" && client.UserID != event.UserID {
			continue
		}

		// Skip if client hasn't subscribed to this folder
		if event.FolderID != "" {
			client.mu.RLock()
			subscribed := client.SubscribedFolders[event.FolderID]
			client.mu.RUnlock()
			if !subscribed {
				continue
			}
		}

		select {
		case client.Send <- message:
		default:
			// Client buffer full, drop message
		}
	}
}

// Broadcast sends an event to all connected clients (or filtered by user)
func (s *WebSocketServer) Broadcast(event *WebSocketEvent) {
	select {
	case s.broadcast <- event:
	default:
		// Broadcast channel full, drop event
	}
}

// BroadcastToUser sends an event to a specific user
func (s *WebSocketServer) BroadcastToUser(userID string, eventType string, data interface{}) {
	s.Broadcast(&WebSocketEvent{
		Type:      eventType,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

// BroadcastFileEvent broadcasts a file-related event
func (s *WebSocketServer) BroadcastFileEvent(userID, folderID, fileID, eventType string, data interface{}) {
	s.Broadcast(&WebSocketEvent{
		Type:      eventType,
		UserID:    userID,
		FolderID:  folderID,
		FileID:    fileID,
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

// readPump pumps messages from the WebSocket connection to the server
func (c *WebSocketClient) readPump() {
	defer func() {
		c.Server.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		c.handleMessage(message)
	}
}

// writePump pumps messages from the server to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(8, []byte{}) // Close message
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				continue
			}

			if err := c.Conn.WriteMessage(1, data); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(9, []byte{}); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *WebSocketClient) handleMessage(data []byte) {
	var msg WebSocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("", "invalid_message", "Failed to parse message")
		return
	}

	switch msg.Type {
	case WSMsgTypePing:
		c.Send <- &WebSocketMessage{
			Type:      WSMsgTypePong,
			ID:        msg.ID,
			Timestamp: time.Now().Unix(),
		}

	case WSMsgTypeSubscribe:
		c.handleSubscribe(&msg)

	case WSMsgTypeUnsubscribe:
		c.handleUnsubscribe(&msg)

	case WSMsgTypeSyncRequest:
		c.handleSyncRequest(&msg)

	default:
		c.sendError(msg.ID, "unknown_type", fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleSubscribe handles folder subscription requests
func (c *WebSocketClient) handleSubscribe(msg *WebSocketMessage) {
	var payload struct {
		FolderID string `json:"folder_id"`
	}

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError(msg.ID, "invalid_payload", "Invalid subscription payload")
		return
	}

	c.mu.Lock()
	c.SubscribedFolders[payload.FolderID] = true
	c.mu.Unlock()

	c.Send <- &WebSocketMessage{
		Type:      WSMsgTypeSubscribe,
		ID:        msg.ID,
		Payload:   mustJSON(map[string]string{"status": "subscribed", "folder_id": payload.FolderID}),
		Timestamp: time.Now().Unix(),
	}
}

// handleUnsubscribe handles folder unsubscription requests
func (c *WebSocketClient) handleUnsubscribe(msg *WebSocketMessage) {
	var payload struct {
		FolderID string `json:"folder_id"`
	}

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError(msg.ID, "invalid_payload", "Invalid unsubscription payload")
		return
	}

	c.mu.Lock()
	delete(c.SubscribedFolders, payload.FolderID)
	c.mu.Unlock()

	c.Send <- &WebSocketMessage{
		Type:      WSMsgTypeUnsubscribe,
		ID:        msg.ID,
		Payload:   mustJSON(map[string]string{"status": "unsubscribed", "folder_id": payload.FolderID}),
		Timestamp: time.Now().Unix(),
	}
}

// handleSyncRequest handles sync requests from clients
func (c *WebSocketClient) handleSyncRequest(msg *WebSocketMessage) {
	var payload struct {
		FolderID string `json:"folder_id"`
		Since    int64  `json:"since"`
		FullSync bool   `json:"full_sync"`
	}

	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError(msg.ID, "invalid_payload", "Invalid sync request payload")
		return
	}

	ctx := context.Background()

	// Get folder contents
	opts := db.ListOpts{
		Limit:  1000,
		Offset: 0,
	}
	items, err := c.Server.vfs.ListDirectory(ctx, c.UserID, payload.FolderID, opts)
	if err != nil {
		c.sendError(msg.ID, "sync_failed", err.Error())
		return
	}

	// Filter by timestamp if incremental sync
	var changes []interface{}
	for _, item := range items {
		if !payload.FullSync && payload.Since > 0 {
			if item.UpdatedAt.Unix() < payload.Since {
				continue
			}
		}
		changes = append(changes, item)
	}

	response := &WebSocketMessage{
		Type:    WSMsgTypeSyncResponse,
		ID:      msg.ID,
		Payload: mustJSON(map[string]interface{}{
			"folder_id": payload.FolderID,
			"changes":   changes,
			"full_sync": payload.FullSync,
			"timestamp": time.Now().Unix(),
		}),
		Timestamp: time.Now().Unix(),
	}

	c.Send <- response
}

// sendError sends an error message to the client
func (c *WebSocketClient) sendError(id, code, message string) {
	c.Send <- &WebSocketMessage{
		Type:      WSMsgTypeError,
		ID:        id,
		Error:     fmt.Sprintf("%s: %s", code, message),
		Timestamp: time.Now().Unix(),
	}
}

func generateClientID() string {
	return "ws_" + generateRandomString(16)
}

func mustJSON(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
