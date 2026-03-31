package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockAuthHandler is a minimal mock for auth handling
type mockAuthHandler struct {
	authSvc *mockAuthService
}

type mockAuthService struct {
	validTokens map[string]*mockClaims
}

type mockClaims struct {
	UserID   string
	Username string
	Roles    []string
	DeviceID string
}

func (m *mockAuthService) ValidateAccessToken(token string) (*mockClaims, error) {
	if claims, ok := m.validTokens[token]; ok {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

var ErrInvalidToken = &testError{msg: "invalid token"}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }

// TestWebSocketConnection tests WebSocket connection establishment
func TestWebSocketConnection(t *testing.T) {
	// Create mock auth handler
	mockAuth := &mockAuthHandler{
		authSvc: &mockAuthService{
			validTokens: map[string]*mockClaims{
				"valid-token": {
					UserID:   "test-user",
					Username: "testuser",
					Roles:    []string{"user"},
				},
			},
		},
	}

	// Create WebSocket server (without VFS and DB for now)
	wsServer := NewWebSocketServer(nil, nil, []byte("test-secret"))

	// Create test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check token
		token := r.URL.Query().Get("token")
		if token == "" {
			token = r.Header.Get("Authorization")
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
		}

		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := mockAuth.authSvc.ValidateAccessToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Upgrade to WebSocket
		conn, err := wsServer.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Create and register client
		client := &WebSocketClient{
			ID:                generateClientID(),
			UserID:            claims.UserID,
			DeviceID:          "test-device",
			Conn:              conn,
			Server:            wsServer,
			Send:              make(chan *WebSocketMessage, 256),
			SubscribedFolders: make(map[string]bool),
		}

		wsServer.register <- client
		defer func() { wsServer.unregister <- client }()

		// Send auth success
		client.Send <- &WebSocketMessage{
			Type:      WSMsgTypeAuthSuccess,
			Payload:   mustJSON(map[string]string{"device_id": "test-device"}),
			Timestamp: time.Now().Unix(),
		}

		// Start pumps
		go client.writePump()
		client.readPump()
	}))
	defer srv.Close()

	t.Run("ConnectWithValidToken", func(t *testing.T) {
		wsURL := strings.Replace(srv.URL, "http", "ws", 1) + "?token=valid-token"
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("Expected 101, got %d", resp.StatusCode)
		}

		// Read auth success message
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read auth success: %v", err)
		}

		var authMsg WebSocketMessage
		if err := json.Unmarshal(msg, &authMsg); err != nil {
			t.Fatalf("Failed to unmarshal auth message: %v", err)
		}

		if authMsg.Type != WSMsgTypeAuthSuccess {
			t.Errorf("Expected auth_success, got %s", authMsg.Type)
		}

		t.Logf("✅ WebSocket connected and authenticated")
	})

	t.Run("ConnectWithoutToken", func(t *testing.T) {
		wsURL := strings.Replace(srv.URL, "http", "ws", 1)
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Error("Expected connection to fail without token")
		}
		if resp != nil && resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}

		t.Logf("✅ Connection correctly rejected without token")
	})

	t.Run("ConnectWithInvalidToken", func(t *testing.T) {
		wsURL := strings.Replace(srv.URL, "http", "ws", 1) + "?token=invalid-token"
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Error("Expected connection to fail with invalid token")
		}
		if resp != nil && resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}

		t.Logf("✅ Connection correctly rejected with invalid token")
	})
}

// TestWebSocketPingPong tests ping/pong messages
func TestWebSocketPingPong(t *testing.T) {
	wsServer := NewWebSocketServer(nil, nil, []byte("test-secret"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsServer.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := &WebSocketClient{
			ID:                generateClientID(),
			UserID:            "test-user",
			DeviceID:          "test-device",
			Conn:              conn,
			Server:            wsServer,
			Send:              make(chan *WebSocketMessage, 256),
			SubscribedFolders: make(map[string]bool),
		}

		wsServer.register <- client
		defer func() { wsServer.unregister <- client }()

		go client.writePump()
		client.readPump()
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http", "ws", 1)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Send ping
	pingMsg := WebSocketMessage{
		Type:      WSMsgTypePing,
		ID:        "ping-1",
		Timestamp: time.Now().Unix(),
	}
	if err := conn.WriteJSON(pingMsg); err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Read pong
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read pong: %v", err)
	}

	var pongMsg WebSocketMessage
	if err := json.Unmarshal(msg, &pongMsg); err != nil {
		t.Fatalf("Failed to unmarshal pong: %v", err)
	}

	if pongMsg.Type != WSMsgTypePong {
		t.Errorf("Expected pong, got %s", pongMsg.Type)
	}

	if pongMsg.ID != "ping-1" {
		t.Errorf("Expected pong ID to match ping ID, got %s", pongMsg.ID)
	}

	t.Logf("✅ Ping/pong working correctly")
}

// TestWebSocketSubscription tests folder subscription
func TestWebSocketSubscription(t *testing.T) {
	wsServer := NewWebSocketServer(nil, nil, []byte("test-secret"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsServer.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := &WebSocketClient{
			ID:                generateClientID(),
			UserID:            "test-user",
			DeviceID:          "test-device",
			Conn:              conn,
			Server:            wsServer,
			Send:              make(chan *WebSocketMessage, 256),
			SubscribedFolders: make(map[string]bool),
		}

		wsServer.register <- client
		defer func() { wsServer.unregister <- client }()

		go client.writePump()
		client.readPump()
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http", "ws", 1)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	t.Run("SubscribeToFolder", func(t *testing.T) {
		subscribeMsg := WebSocketMessage{
			Type: "subscribe",
			ID:   "sub-1",
			Payload: mustJSON(map[string]string{
				"folder_id": "folder-123",
			}),
			Timestamp: time.Now().Unix(),
		}

		if err := conn.WriteJSON(subscribeMsg); err != nil {
			t.Fatalf("Failed to send subscribe: %v", err)
		}

		// Read response
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read subscribe response: %v", err)
		}

		var resp WebSocketMessage
		if err := json.Unmarshal(msg, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Type != WSMsgTypeSubscribe {
			t.Errorf("Expected subscribe response, got %s", resp.Type)
		}

		t.Logf("✅ Successfully subscribed to folder")
	})

	t.Run("UnsubscribeFromFolder", func(t *testing.T) {
		unsubscribeMsg := WebSocketMessage{
			Type: "unsubscribe",
			ID:   "unsub-1",
			Payload: mustJSON(map[string]string{
				"folder_id": "folder-123",
			}),
			Timestamp: time.Now().Unix(),
		}

		if err := conn.WriteJSON(unsubscribeMsg); err != nil {
			t.Fatalf("Failed to send unsubscribe: %v", err)
		}

		// Read response
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read unsubscribe response: %v", err)
		}

		var resp WebSocketMessage
		if err := json.Unmarshal(msg, &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Type != WSMsgTypeUnsubscribe {
			t.Errorf("Expected unsubscribe response, got %s", resp.Type)
		}

		t.Logf("✅ Successfully unsubscribed from folder")
	})
}

// TestWebSocketEventBroadcast tests event broadcasting
func TestWebSocketEventBroadcast(t *testing.T) {
	wsServer := NewWebSocketServer(nil, nil, []byte("test-secret"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsServer.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		client := &WebSocketClient{
			ID:                generateClientID(),
			UserID:            "test-user",
			DeviceID:          "test-device",
			Conn:              conn,
			Server:            wsServer,
			Send:              make(chan *WebSocketMessage, 256),
			SubscribedFolders: make(map[string]bool),
		}

		// Subscribe to root folder
		client.SubscribedFolders[""] = true

		wsServer.register <- client
		defer func() { wsServer.unregister <- client }()

		go client.writePump()
		client.readPump()
	}))
	defer srv.Close()

	// Connect first client
	wsURL := strings.Replace(srv.URL, "http", "ws", 1)
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer conn1.Close()

	// Give server time to register client
	time.Sleep(100 * time.Millisecond)

	// Broadcast an event
	event := &WebSocketEvent{
		Type:      EventFileCreated,
		UserID:    "test-user",
		FolderID:  "",
		FileID:    "file-123",
		Data:      map[string]string{"name": "test.txt"},
		Timestamp: time.Now().Unix(),
	}

	wsServer.Broadcast(event)

	// Read event on client 1
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read event: %v", err)
	}

	var receivedMsg WebSocketMessage
	if err := json.Unmarshal(msg, &receivedMsg); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if receivedMsg.Type != WSMsgTypeEvent {
		t.Errorf("Expected event type, got %s", receivedMsg.Type)
	}

	var receivedEvent WebSocketEvent
	if err := json.Unmarshal(receivedMsg.Payload, &receivedEvent); err != nil {
		t.Fatalf("Failed to unmarshal event payload: %v", err)
	}

	if receivedEvent.Type != EventFileCreated {
		t.Errorf("Expected file.created event, got %s", receivedEvent.Type)
	}

	if receivedEvent.FileID != "file-123" {
		t.Errorf("Expected file ID file-123, got %s", receivedEvent.FileID)
	}

	t.Logf("✅ Event broadcast received correctly")
}

// TestWebSocketBroadcastToUser tests user-specific broadcasting
func TestWebSocketBroadcastToUser(t *testing.T) {
	wsServer := NewWebSocketServer(nil, nil, []byte("test-secret"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsServer.upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = "default-user"
		}

		client := &WebSocketClient{
			ID:                generateClientID(),
			UserID:            userID,
			DeviceID:          "test-device",
			Conn:              conn,
			Server:            wsServer,
			Send:              make(chan *WebSocketMessage, 256),
			SubscribedFolders: make(map[string]bool),
		}

		wsServer.register <- client
		defer func() { wsServer.unregister <- client }()

		go client.writePump()
		client.readPump()
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http", "ws", 1)

	// Connect user 1
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL+"?user_id=user-1", nil)
	if err != nil {
		t.Fatalf("Failed to connect user 1: %v", err)
	}
	defer conn1.Close()

	// Connect user 2
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL+"?user_id=user-2", nil)
	if err != nil {
		t.Fatalf("Failed to connect user 2: %v", err)
	}
	defer conn2.Close()

	// Give server time to register clients
	time.Sleep(100 * time.Millisecond)

	// Broadcast event only to user-1
	wsServer.BroadcastToUser("user-1", EventFileUpdated, map[string]string{"file": "doc.txt"})

	// User 1 should receive the event
	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("User 1 should have received event: %v", err)
	}

	var receivedMsg WebSocketMessage
	if err := json.Unmarshal(msg, &receivedMsg); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if receivedMsg.Type != WSMsgTypeEvent {
		t.Errorf("Expected event type, got %s", receivedMsg.Type)
	}

	// User 2 should not receive anything (set short deadline)
	conn2.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err = conn2.ReadMessage()
	if err == nil {
		t.Error("User 2 should not have received the event")
	}

	t.Logf("✅ User-specific broadcast working correctly")
}
