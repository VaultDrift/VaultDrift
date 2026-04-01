package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

// Event types for WebSocket events
const (
	EventFileCreated   = "file.created"
	EventFileUpdated   = "file.updated"
	EventFileDeleted   = "file.deleted"
	EventFileMoved     = "file.moved"
	EventFolderCreated = "folder.created"
	EventFolderDeleted = "folder.deleted"
	EventSyncComplete  = "sync.complete"
	EventConflict      = "conflict.detected"
	EventShareReceived = "share.received"
	EventShareRevoked  = "share.revoked"
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
	DeviceID          string
	Conn              *websocket.Conn
	Server            *WebSocketServer
	Send              chan *WebSocketMessage
	SubscribedFolders map[string]bool
	mu                sync.RWMutex
}

// WebSocketServer manages WebSocket connections and event broadcasting
type WebSocketServer struct {
	vfs       *vfs.VFS
	db        *db.Manager
	jwtSecret []byte

	clients   map[string]*WebSocketClient
	clientsMu sync.RWMutex
	userIndex map[string][]string // userID -> clientIDs
	userMu    sync.RWMutex

	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan *WebSocketEvent

	upgrader websocket.Upgrader
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(vfsService *vfs.VFS, database *db.Manager, jwtSecret []byte) *WebSocketServer {
	s := &WebSocketServer{
		vfs:        vfsService,
		db:         database,
		jwtSecret:  jwtSecret,
		clients:    make(map[string]*WebSocketClient),
		userIndex:  make(map[string][]string),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan *WebSocketEvent, 256),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin in development
				// In production, this should be configured properly
				return true
			},
		},
	}

	go s.run()

	return s
}

// RegisterRoutes registers WebSocket routes
func (s *WebSocketServer) RegisterRoutes(mux *http.ServeMux, auth *AuthHandler) {
	// WebSocket endpoint with token-based auth via query parameter
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		s.handleWebSocket(w, r, auth)
	})
}

// handleWebSocket handles WebSocket upgrade requests
func (s *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request, auth *AuthHandler) {
	// Get token from query parameter or header
	token := r.URL.Query().Get("token")
	if token == "" {
		token = r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		http.Error(w, "Unauthorized: token required", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := auth.authSvc.ValidateAccessToken(token)
	if err != nil {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get device ID
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		deviceID = generateRandomString(16)
	}

	// Create client
	client := &WebSocketClient{
		ID:                generateClientID(),
		UserID:            claims.UserID,
		DeviceID:          deviceID,
		Conn:              conn,
		Server:            s,
		Send:              make(chan *WebSocketMessage, 256),
		SubscribedFolders: make(map[string]bool),
	}

	// Register client
	s.register <- client

	// Send initial auth success message
	client.Send <- &WebSocketMessage{
		Type:      WSMsgTypeAuthSuccess,
		Payload:   mustJSON(map[string]string{"device_id": deviceID}),
		Timestamp: time.Now().Unix(),
	}

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

			// Add to user index
			s.userMu.Lock()
			s.userIndex[client.UserID] = append(s.userIndex[client.UserID], client.ID)
			s.userMu.Unlock()

		case client := <-s.unregister:
			s.clientsMu.Lock()
			if _, ok := s.clients[client.ID]; ok {
				delete(s.clients, client.ID)
				close(client.Send)
			}
			s.clientsMu.Unlock()

			// Remove from user index
			s.userMu.Lock()
			if clients, ok := s.userIndex[client.UserID]; ok {
				for i, id := range clients {
					if id == client.ID {
						s.userIndex[client.UserID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
			}
			s.userMu.Unlock()

		case event := <-s.broadcast:
			s.broadcastEvent(event)
		}
	}
}

// broadcastEvent sends an event to all subscribed clients
func (s *WebSocketServer) broadcastEvent(event *WebSocketEvent) {
	// Build message once
	message := &WebSocketMessage{
		Type:      WSMsgTypeEvent,
		Payload:   mustJSON(event),
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	// Get clients for this user
	s.userMu.RLock()
	clientIDs := s.userIndex[event.UserID]
	s.userMu.RUnlock()

	if len(clientIDs) == 0 {
		return
	}

	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()

	for _, clientID := range clientIDs {
		client, ok := s.clients[clientID]
		if !ok {
			continue
		}

		// Skip if client hasn't subscribed to this folder (if specified)
		if event.FolderID != "" {
			client.mu.RLock()
			subscribed := client.SubscribedFolders[event.FolderID] || client.SubscribedFolders[""]
			client.mu.RUnlock()
			if !subscribed {
				continue
			}
		}

		// Send directly to avoid channel bottleneck for broadcast
		_ = client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			// Write failed, client will be cleaned up on next read
			_ = err // Explicitly ignore - client cleanup handled elsewhere
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
		_ = c.Conn.Close()
	}()

	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log unexpected close
			}
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
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				continue
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
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

	// Empty folder ID means subscribe to all
	folderID := payload.FolderID
	if folderID == "" {
		folderID = ""
	}

	c.mu.Lock()
	c.SubscribedFolders[folderID] = true
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
		Type: WSMsgTypeSyncResponse,
		ID:   msg.ID,
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
