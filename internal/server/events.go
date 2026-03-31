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

// SSEEvent represents a server-sent event
type SSEEvent struct {
	Type      string      `json:"type"`
	UserID    string      `json:"user_id"`
	FolderID  string      `json:"folder_id,omitempty"`
	FileID    string      `json:"file_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// Event types for real-time notifications
const (
	SSEventFileCreated    = "file:created"
	SSEventFileUpdated    = "file:updated"
	SSEventFileDeleted    = "file:deleted"
	SSEventFileMoved      = "file:moved"
	SSEventFolderCreated  = "folder:created"
	SSEventFolderUpdated  = "folder:updated"
	SSEventFolderDeleted  = "folder:deleted"
	SSEventShareCreated   = "share:created"
	SSEventShareRevoked   = "share:revoked"
	SSEventUploadComplete = "upload:complete"
)

// EventNotifier handles real-time event notifications using SSE
type EventNotifier struct {
	vfs *vfs.VFS
	db  *db.Manager

	subscribers map[string]*EventSubscriber
	mu          sync.RWMutex
	eventChan   chan *SSEEvent
}

// EventSubscriber represents a subscriber to events
type EventSubscriber struct {
	ID        string
	UserID    string
	FolderIDs map[string]bool
	Events    chan *SSEEvent
}

// NewEventNotifier creates a new event notifier
func NewEventNotifier(vfsService *vfs.VFS, database *db.Manager) *EventNotifier {
	n := &EventNotifier{
		vfs:         vfsService,
		db:          database,
		subscribers: make(map[string]*EventSubscriber),
		eventChan:   make(chan *SSEEvent, 256),
	}

	go n.eventLoop()

	return n
}

// Subscribe creates a new event subscriber
func (n *EventNotifier) Subscribe(userID string) *EventSubscriber {
	sub := &EventSubscriber{
		ID:        generateSubscriberID(),
		UserID:    userID,
		FolderIDs: make(map[string]bool),
		Events:    make(chan *SSEEvent, 64),
	}

	n.mu.Lock()
	n.subscribers[sub.ID] = sub
	n.mu.Unlock()

	return sub
}

// Unsubscribe removes a subscriber
func (n *EventNotifier) Unsubscribe(subID string) {
	n.mu.Lock()
	if sub, ok := n.subscribers[subID]; ok {
		delete(n.subscribers, subID)
		close(sub.Events)
	}
	n.mu.Unlock()
}

// SubscribeFolder subscribes a subscriber to folder events
func (n *EventNotifier) SubscribeFolder(subID, folderID string) {
	n.mu.RLock()
	sub, ok := n.subscribers[subID]
	n.mu.RUnlock()

	if ok {
		sub.FolderIDs[folderID] = true
	}
}

// UnsubscribeFolder unsubscribes a subscriber from folder events
func (n *EventNotifier) UnsubscribeFolder(subID, folderID string) {
	n.mu.RLock()
	sub, ok := n.subscribers[subID]
	n.mu.RUnlock()

	if ok {
		delete(sub.FolderIDs, folderID)
	}
}

// Notify sends an event to relevant subscribers
func (n *EventNotifier) Notify(event *SSEEvent) {
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}

	select {
	case n.eventChan <- event:
	default:
		// Event channel full, drop event
	}
}

// NotifyUser sends an event to a specific user
func (n *EventNotifier) NotifyUser(userID, eventType string, data interface{}) {
	n.Notify(&SSEEvent{
		Type:   eventType,
		UserID: userID,
		Data:   data,
	})
}

// NotifyFileChange notifies about file changes in a folder
func (n *EventNotifier) NotifyFileChange(userID, folderID, fileID, eventType string, data interface{}) {
	n.Notify(&SSEEvent{
		Type:     eventType,
		UserID:   userID,
		FolderID: folderID,
		FileID:   fileID,
		Data:     data,
	})
}

// eventLoop processes events and distributes to subscribers
func (n *EventNotifier) eventLoop() {
	for event := range n.eventChan {
		n.mu.RLock()
		subscribers := make([]*EventSubscriber, 0, len(n.subscribers))
		for _, sub := range n.subscribers {
			// Filter by user
			if sub.UserID != event.UserID {
				continue
			}
			// Filter by folder if specified
			if event.FolderID != "" {
				if !sub.FolderIDs[event.FolderID] && !sub.FolderIDs["*"] {
					continue
				}
			}
			subscribers = append(subscribers, sub)
		}
		n.mu.RUnlock()

		// Send to filtered subscribers
		for _, sub := range subscribers {
			select {
			case sub.Events <- event:
			default:
				// Subscriber buffer full, drop event
			}
		}
	}
}

// RegisterRoutes registers SSE routes
func (n *EventNotifier) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// SSE endpoint for real-time events
	mux.Handle("GET /api/v1/events", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(n.HandleSSE))))

	// Long-polling fallback
	mux.Handle("GET /api/v1/events/poll", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(n.HandleLongPoll))))
}

// HandleSSE handles Server-Sent Events HTTP requests
func (n *EventNotifier) HandleSSE(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Get optional folder filter
	folderID := r.URL.Query().Get("folder")

	// Create subscriber
	sub := n.Subscribe(userID)
	if folderID != "" {
		n.SubscribeFolder(sub.ID, folderID)
	}
	defer n.Unsubscribe(sub.ID)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", `{"status":"connected","subscriber_id":"`+sub.ID+`"}`)
	flusher.Flush()

	// Heartbeat ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Handle client disconnect
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
			flusher.Flush()

		case event, ok := <-sub.Events:
			if !ok {
				return
			}

			// Format as SSE
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\nid: %d\ndata: %s\n\n", event.Type, event.Timestamp, string(data))
			flusher.Flush()
		}
	}
}

// HandleLongPoll handles long-polling for clients that don't support SSE
func (n *EventNotifier) HandleLongPoll(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get parameters
	folderID := r.URL.Query().Get("folder")
	timeoutStr := r.URL.Query().Get("timeout")
	timeout := 30 * time.Second
	if timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr + "s"); err == nil {
			timeout = t
		}
	}

	// Create subscriber
	sub := n.Subscribe(userID)
	if folderID != "" {
		n.SubscribeFolder(sub.ID, folderID)
	}
	defer n.Unsubscribe(sub.ID)

	// Wait for event or timeout
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	select {
	case event := <-sub.Events:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)

	case <-ctx.Done():
		// Timeout - return empty response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "timeout"})
	}
}

func generateSubscriberID() string {
	return "sub_" + generateRandomString(12)
}
