package federation

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// Manager handles federation operations
type Manager struct {
	config     FederationConfig
	db         *db.Manager
	httpClient *http.Client
	peers      map[string]*db.FederationPeer
	peersMu    sync.RWMutex
	stopChan   chan struct{}
}

// NewManager creates a new federation manager
func NewManager(config FederationConfig, database *db.Manager) (*Manager, error) {
	if config.ServerID == "" {
		config.ServerID = generateServerID()
	}

	if config.PrivateKey == "" || config.PublicKey == "" {
		pub, priv, err := GenerateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate federation keys: %w", err)
		}
		config.PublicKey = pub
		config.PrivateKey = priv
	}

	return &Manager{
		config: config,
		db:     database,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		peers:    make(map[string]*db.FederationPeer),
		stopChan: make(chan struct{}),
	}, nil
}

// Start begins federation operations
func (m *Manager) Start(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}

	log.Println("Starting federation manager...")

	// Load existing peers
	if err := m.loadPeers(ctx); err != nil {
		log.Printf("Failed to load peers: %v", err)
	}

	// Start health check loop
	go m.healthCheckLoop(ctx)

	// Announce to trusted peers
	for _, peerURL := range m.config.TrustedPeers {
		if err := m.discoverPeer(ctx, peerURL); err != nil {
			log.Printf("Failed to discover peer %s: %v", peerURL, err)
		}
	}

	return nil
}

// Stop stops federation operations
func (m *Manager) Stop() {
	close(m.stopChan)
}

// IsEnabled returns true if federation is enabled
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled
}

// GetConfig returns the federation configuration
func (m *Manager) GetConfig() FederationConfig {
	return m.config
}

// GetPeers returns all known peers
func (m *Manager) GetPeers() []*db.FederationPeer {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	peers := make([]*db.FederationPeer, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, peer)
	}
	return peers
}

// GetPeer returns a specific peer by ID
func (m *Manager) GetPeer(peerID string) (*db.FederationPeer, error) {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	peer, ok := m.peers[peerID]
	if !ok {
		return nil, fmt.Errorf("peer not found: %s", peerID)
	}
	return peer, nil
}

// AddPeer adds a new federation peer
func (m *Manager) AddPeer(ctx context.Context, peer *db.FederationPeer) error {
	if peer.ID == m.config.ServerID {
		return fmt.Errorf("cannot add self as peer")
	}

	peer.LastSeen = time.Now()
	peer.CreatedAt = time.Now()
	if peer.Capabilities == "" {
		peer.Capabilities = strings.Join(DefaultCapabilities(), ",")
	}

	// Save to database
	if err := m.db.AddFederationPeer(ctx, peer); err != nil {
		return fmt.Errorf("failed to save peer: %w", err)
	}

	m.peersMu.Lock()
	m.peers[peer.ID] = peer
	m.peersMu.Unlock()

	log.Printf("Added federation peer: %s (%s)", peer.Name, peer.PublicURL)
	return nil
}

// RemovePeer removes a federation peer
func (m *Manager) RemovePeer(ctx context.Context, peerID string) error {
	if err := m.db.RemoveFederationPeer(ctx, peerID); err != nil {
		return err
	}

	m.peersMu.Lock()
	delete(m.peers, peerID)
	m.peersMu.Unlock()

	return nil
}

// CreateInvite creates a federation invitation
func (m *Manager) CreateInvite(ctx context.Context, peerID string, expiresIn time.Duration) (*db.FederationInvite, error) {
	invite := &db.FederationInvite{
		ID:         generateInviteID(),
		FromPeerID: m.config.ServerID,
		Token:      generateSecureToken(),
		ExpiresAt:  time.Now().Add(expiresIn),
		CreatedAt:  time.Now(),
	}

	if err := m.db.CreateFederationInvite(ctx, invite); err != nil {
		return nil, err
	}

	return invite, nil
}

// VerifyInvite verifies a federation invitation token
func (m *Manager) VerifyInvite(ctx context.Context, token string) (*db.FederationInvite, error) {
	invite, err := m.db.GetFederationInvite(ctx, token)
	if err != nil {
		return nil, err
	}

	if invite.Used {
		return nil, fmt.Errorf("invite already used")
	}

	if time.Now().After(invite.ExpiresAt) {
		return nil, fmt.Errorf("invite expired")
	}

	return invite, nil
}

// AcceptInvite accepts a federation invitation
func (m *Manager) AcceptInvite(ctx context.Context, token string, peerInfo *PeerAnnouncement) error {
	invite, err := m.VerifyInvite(ctx, token)
	if err != nil {
		return err
	}

	// Mark invite as used
	if err := m.db.UseFederationInvite(ctx, invite.ID); err != nil {
		return err
	}

	// Add the peer
	peer := &db.FederationPeer{
		ID:           peerInfo.ServerID,
		Name:         peerInfo.Name,
		PublicURL:    peerInfo.PublicURL,
		PublicKey:    peerInfo.PublicKey,
		Status:       "active",
		Capabilities: strings.Join(peerInfo.Capabilities, ","),
	}

	return m.AddPeer(ctx, peer)
}

// SendMessage sends a signed message to a peer
func (m *Manager) SendMessage(ctx context.Context, peerID string, msgType string, payload interface{}) error {
	peer, err := m.GetPeer(peerID)
	if err != nil {
		return err
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := FederationMessage{
		ID:        generateMessageID(),
		From:      m.config.ServerID,
		To:        peerID,
		Type:      msgType,
		Payload:   payloadBytes,
		Timestamp: time.Now(),
	}

	// Sign the message
	sig, err := SignMessage(m.config.PrivateKey, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}
	msg.Signature = sig

	// Send to peer
	url := fmt.Sprintf("%s/federation/v1/message", peer.PublicURL)
	return m.sendHTTPRequest(ctx, url, msg)
}

// VerifyIncomingMessage verifies an incoming federation message
func (m *Manager) VerifyIncomingMessage(msg *FederationMessage) error {
	peer, err := m.GetPeer(msg.From)
	if err != nil {
		return fmt.Errorf("unknown peer: %w", err)
	}

	if peer.Status != "active" {
		return fmt.Errorf("peer is not active")
	}

	return VerifyMessage(peer.PublicKey, msg.Payload, msg.Signature)
}

// healthCheckLoop periodically checks peer health
func (m *Manager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkPeersHealth(ctx)
		}
	}
}

// checkPeersHealth checks the health of all peers
func (m *Manager) checkPeersHealth(ctx context.Context) {
	m.peersMu.RLock()
	peers := make([]*db.FederationPeer, 0, len(m.peers))
	for _, peer := range m.peers {
		peers = append(peers, peer)
	}
	m.peersMu.RUnlock()

	for _, peer := range peers {
		go m.checkPeerHealth(ctx, peer)
	}
}

// checkPeerHealth checks the health of a single peer
func (m *Manager) checkPeerHealth(ctx context.Context, peer *db.FederationPeer) {
	url := fmt.Sprintf("%s/federation/v1/health", peer.PublicURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		m.updatePeerStatus(peer.ID, "inactive")
		return
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.updatePeerStatus(peer.ID, "inactive")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		m.updatePeerStatus(peer.ID, "active")
		m.updatePeerLastSeen(peer.ID)
	} else {
		m.updatePeerStatus(peer.ID, "inactive")
	}
}

// updatePeerStatus updates a peer's status
func (m *Manager) updatePeerStatus(peerID, status string) {
	m.peersMu.Lock()
	if peer, ok := m.peers[peerID]; ok {
		peer.Status = status
	}
	m.peersMu.Unlock()

	// Update in database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m.db.UpdateFederationPeerStatus(ctx, peerID, status)
}

// updatePeerLastSeen updates a peer's last seen time
func (m *Manager) updatePeerLastSeen(peerID string) {
	m.peersMu.Lock()
	if peer, ok := m.peers[peerID]; ok {
		peer.LastSeen = time.Now()
	}
	m.peersMu.Unlock()
}

// discoverPeer attempts to discover and add a peer by URL
func (m *Manager) discoverPeer(ctx context.Context, peerURL string) error {
	// Send announcement to peer
	announcement := PeerAnnouncement{
		ServerID:     m.config.ServerID,
		Name:         m.config.ServerID, // Could be configured
		PublicURL:    m.config.PublicURL,
		PublicKey:    m.config.PublicKey,
		Capabilities: DefaultCapabilities(),
	}

	url := fmt.Sprintf("%s/federation/v1/discover", peerURL)
	return m.sendHTTPRequest(ctx, url, announcement)
}

// sendHTTPRequest sends an HTTP request to a peer
func (m *Manager) sendHTTPRequest(ctx context.Context, url string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Federation-ID", m.config.ServerID)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("peer returned error: %d", resp.StatusCode)
	}

	return nil
}

// loadPeers loads peers from database
func (m *Manager) loadPeers(ctx context.Context) error {
	peers, err := m.db.GetFederationPeers(ctx)
	if err != nil {
		return err
	}

	m.peersMu.Lock()
	for _, peer := range peers {
		m.peers[peer.ID] = peer
	}
	m.peersMu.Unlock()

	log.Printf("Loaded %d federation peers", len(peers))
	return nil
}

// Helper functions
func generateServerID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateInviteID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateSecureToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateMessageID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
