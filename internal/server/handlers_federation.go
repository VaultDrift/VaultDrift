package server

import (
	"encoding/base64"
	"fmt"
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

// isAdmin checks if the current user has admin role.
func (h *FederationHandler) isAdmin(r *http.Request) bool {
	return IsAdmin(r)
}

// requireAdmin checks admin role and returns true if authorized.
func (h *FederationHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !h.isAdmin(r) {
		ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return false
	}
	return true
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
	if !h.requireAdmin(w, r) {
		return
	}

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
	if !h.requireAdmin(w, r) {
		return
	}

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
	if !h.requireAdmin(w, r) {
		return
	}

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

	if err := DecodeJSON(r, &req); err != nil {
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, peer)
}

func (h *FederationHandler) handleRemovePeer(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *FederationHandler) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	if !h.fed.IsEnabled() {
		http.Error(w, "Federation not enabled", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ExpiresInHours int `json:"expires_in_hours"`
	}

	if err := DecodeJSON(r, &req); err != nil {
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
	if err := DecodeJSON(r, &announcement); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify the announcement has required fields
	if announcement.ServerID == "" || announcement.PublicURL == "" || announcement.PublicKey == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Require a valid signature
	if announcement.Signature == "" || announcement.Timestamp == 0 {
		http.Error(w, "Missing signature or timestamp", http.StatusBadRequest)
		return
	}

	// Verify timestamp is within 5 minutes to prevent replay attacks
	now := time.Now().Unix()
	const maxSkew int64 = 300 // 5 minutes
	if announcement.Timestamp > now+maxSkew || announcement.Timestamp < now-maxSkew {
		http.Error(w, "Stale or future timestamp", http.StatusBadRequest)
		return
	}

	// Verify the Ed25519 signature over "server_id:public_url:timestamp"
	sigBytes, err := base64.StdEncoding.DecodeString(announcement.Signature)
	if err != nil {
		http.Error(w, "Invalid signature encoding", http.StatusBadRequest)
		return
	}
	message := []byte(fmt.Sprintf("%s:%s:%d", announcement.ServerID, announcement.PublicURL, announcement.Timestamp))
	if err := federation.VerifyMessage(announcement.PublicKey, message, sigBytes); err != nil {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Don't add self
	if announcement.ServerID == h.fed.GetConfig().ServerID {
		http.Error(w, "Cannot add self", http.StatusBadRequest)
		return
	}

	// Check if auto-discovery is enabled
	if !h.fed.GetConfig().AutoDiscovery {
		http.Error(w, "Auto-discovery not enabled", http.StatusForbidden)
		return
	}

	// Add as peer
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
	if err := DecodeJSON(r, &msg); err != nil {
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

	if err := DecodeJSON(r, &req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.fed.AcceptInvite(r.Context(), req.Token, &req.PeerInfo); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}

func (h *FederationHandler) handleShareRequest(w http.ResponseWriter, r *http.Request, msg *federation.FederationMessage) {
	// Federation share requests are not yet implemented
	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"status":  "not_implemented",
		"message": "federation share requests are not yet supported",
	})
}

func (h *FederationHandler) handleSyncRequest(w http.ResponseWriter, r *http.Request, msg *federation.FederationMessage) {
	// Federation sync requests are not yet implemented
	writeJSON(w, http.StatusNotImplemented, map[string]string{
		"status":  "not_implemented",
		"message": "federation sync requests are not yet supported",
	})
}
