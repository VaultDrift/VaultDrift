package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/federation"
)

// FederationHandler handles federation API endpoints
type FederationHandler struct {
	fed *federation.Manager
}

// NewFederationHandler creates a new federation handler
func NewFederationHandler(fed *federation.Manager) *FederationHandler {
	return &FederationHandler{fed: fed}
}

// RegisterRoutes registers federation routes
func (h *FederationHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	// Internal federation endpoints (require auth)
	mux.Handle("GET /api/v1/federation/config", auth(http.HandlerFunc(h.handleGetConfig)))
	mux.Handle("GET /api/v1/federation/peers", auth(http.HandlerFunc(h.handleListPeers)))
	mux.Handle("POST /api/v1/federation/peers", auth(http.HandlerFunc(h.handleAddPeer)))
	mux.Handle("DELETE /api/v1/federation/peers/{peerID}", auth(http.HandlerFunc(h.handleRemovePeer)))
	mux.Handle("POST /api/v1/federation/invites", auth(http.HandlerFunc(h.handleCreateInvite)))

	// Inter-server federation endpoints (no auth, signature verification)
	mux.HandleFunc("GET /federation/v1/health", h.handleHealth)
	mux.HandleFunc("POST /federation/v1/discover", h.handleDiscover)
	mux.HandleFunc("POST /federation/v1/message", h.handleMessage)
	mux.HandleFunc("POST /federation/v1/join", h.handleJoin)
}

func (h *FederationHandler) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	if !h.fed.IsEnabled() {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	config := h.fed.GetConfig()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":        true,
		"server_id":      config.ServerID,
		"public_url":     config.PublicURL,
		"public_key":     config.PublicKey,
		"trusted_peers":  config.TrustedPeers,
		"auto_discovery": config.AutoDiscovery,
	})
}

func (h *FederationHandler) handleListPeers(w http.ResponseWriter, r *http.Request) {
	if !h.fed.IsEnabled() {
		http.Error(w, "Federation not enabled", http.StatusServiceUnavailable)
		return
	}

	peers := h.fed.GetPeers()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"peers": peers,
	})
}

func (h *FederationHandler) handleAddPeer(w http.ResponseWriter, r *http.Request) {
	if !h.fed.IsEnabled() {
		http.Error(w, "Federation not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		PublicURL    string   `json:"public_url"`
		PublicKey    string   `json:"public_key"`
		Capabilities []string `json:"capabilities"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.ID == "" || req.PublicURL == "" || req.PublicKey == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	peer := &db.FederationPeer{
		ID:           req.ID,
		Name:         req.Name,
		PublicURL:    req.PublicURL,
		PublicKey:    req.PublicKey,
		Status:       "active",
		Capabilities: strings.Join(req.Capabilities, ","),
	}

	if err := h.fed.AddPeer(r.Context(), peer); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, peer)
}

func (h *FederationHandler) handleRemovePeer(w http.ResponseWriter, r *http.Request) {
	if !h.fed.IsEnabled() {
		http.Error(w, "Federation not enabled", http.StatusServiceUnavailable)
		return
	}

	peerID := r.PathValue("peerID")
	if peerID == "" {
		http.Error(w, "Peer ID required", http.StatusBadRequest)
		return
	}

	if err := h.fed.RemovePeer(r.Context(), peerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *FederationHandler) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	if !h.fed.IsEnabled() {
		http.Error(w, "Federation not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ExpiresInHours int `json:"expires_in_hours"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.ExpiresInHours = 24 // Default 24 hours
	}

	if req.ExpiresInHours < 1 {
		req.ExpiresInHours = 1
	}
	if req.ExpiresInHours > 168 { // Max 7 days
		req.ExpiresInHours = 168
	}

	invite, err := h.fed.CreateInvite(r.Context(), "", time.Duration(req.ExpiresInHours)*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config := h.fed.GetConfig()

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"invite_token": invite.Token,
		"expires_at":   invite.ExpiresAt,
		"server_id":    config.ServerID,
		"public_url":   config.PublicURL,
	})
}

// Inter-server endpoints

func (h *FederationHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "healthy",
		"server_id":  h.fed.GetConfig().ServerID,
		"federation": h.fed.IsEnabled(),
	})
}

func (h *FederationHandler) handleDiscover(w http.ResponseWriter, r *http.Request) {
	if !h.fed.IsEnabled() {
		http.Error(w, "Federation not enabled", http.StatusServiceUnavailable)
		return
	}

	var announcement federation.PeerAnnouncement
	if err := json.NewDecoder(r.Body).Decode(&announcement); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify the announcement has required fields
	if announcement.ServerID == "" || announcement.PublicURL == "" || announcement.PublicKey == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Don't add self
	if announcement.ServerID == h.fed.GetConfig().ServerID {
		http.Error(w, "Cannot add self", http.StatusBadRequest)
		return
	}

	// Add as peer (if auto-discovery is enabled or manually approved)
	// For now, add automatically
	peer := &db.FederationPeer{
		ID:           announcement.ServerID,
		Name:         announcement.Name,
		PublicURL:    announcement.PublicURL,
		PublicKey:    announcement.PublicKey,
		Status:       "active",
		Capabilities: strings.Join(announcement.Capabilities, ","),
	}

	if err := h.fed.AddPeer(r.Context(), peer); err != nil {
		// Peer might already exist, which is fine
		if !strings.Contains(err.Error(), "already exists") {
			log.Printf("Failed to add peer from discovery: %v", err)
		}
	}

	// Return our own announcement
	config := h.fed.GetConfig()
	writeJSON(w, http.StatusOK, federation.PeerAnnouncement{
		ServerID:     config.ServerID,
		Name:         config.ServerID,
		PublicURL:    config.PublicURL,
		PublicKey:    config.PublicKey,
		Capabilities: federation.DefaultCapabilities(),
	})
}

func (h *FederationHandler) handleMessage(w http.ResponseWriter, r *http.Request) {
	if !h.fed.IsEnabled() {
		http.Error(w, "Federation not enabled", http.StatusServiceUnavailable)
		return
	}

	var msg federation.FederationMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify the message signature
	if err := h.fed.VerifyIncomingMessage(&msg); err != nil {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Process the message based on type
	switch msg.Type {
	case "share_request":
		h.handleShareRequest(w, r, &msg)
	case "sync_request":
		h.handleSyncRequest(w, r, &msg)
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
	}
}

func (h *FederationHandler) handleJoin(w http.ResponseWriter, r *http.Request) {
	if !h.fed.IsEnabled() {
		http.Error(w, "Federation not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Token    string                      `json:"token"`
		PeerInfo federation.PeerAnnouncement `json:"peer_info"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.fed.AcceptInvite(r.Context(), req.Token, &req.PeerInfo); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}

func (h *FederationHandler) handleShareRequest(w http.ResponseWriter, r *http.Request, msg *federation.FederationMessage) {
	// Handle share request from peer
	// This would validate the request and return a download URL
	writeJSON(w, http.StatusOK, map[string]string{"status": "processing"})
}

func (h *FederationHandler) handleSyncRequest(w http.ResponseWriter, r *http.Request, msg *federation.FederationMessage) {
	// Handle sync request from peer
	writeJSON(w, http.StatusOK, map[string]string{"status": "processing"})
}
