package federation

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// FederationConfig holds federation settings
type FederationConfig struct {
	Enabled       bool     `yaml:"enabled" json:"enabled"`
	ServerID      string   `yaml:"server_id" json:"server_id"`
	PublicURL     string   `yaml:"public_url" json:"public_url"`
	PrivateKey    string   `yaml:"private_key" json:"-"`
	PublicKey     string   `yaml:"public_key" json:"public_key"`
	TrustedPeers  []string `yaml:"trusted_peers" json:"trusted_peers"`
	AutoDiscovery bool     `yaml:"auto_discovery" json:"auto_discovery"`
}

// FederationMessage represents an inter-server message
type FederationMessage struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`      // Server ID
	To        string    `json:"to"`        // Target Server ID
	Type      string    `json:"type"`      // peer_announce, share_request, sync_request, etc.
	Payload   []byte    `json:"payload"`
	Signature []byte    `json:"signature"`
	Timestamp time.Time `json:"timestamp"`
}

// PeerAnnouncement is sent when a server joins the federation
type PeerAnnouncement struct {
	ServerID     string   `json:"server_id"`
	Name         string   `json:"name"`
	PublicURL    string   `json:"public_url"`
	PublicKey    string   `json:"public_key"`
	Capabilities []string `json:"capabilities"`
}

// ShareRequest is sent to request access to a shared file
type ShareRequest struct {
	ShareID        string `json:"share_id"`
	FileID         string `json:"file_id"`
	RequesterID    string `json:"requester_id"`
	RequesterEmail string `json:"requester_email"`
	Permission     string `json:"permission"`
}

// ShareResponse is the response to a share request
type ShareResponse struct {
	ShareID     string `json:"share_id"`
	Approved    bool   `json:"approved"`
	DownloadURL string `json:"download_url,omitempty"`
	Error       string `json:"error,omitempty"`
}

// GenerateKeyPair generates a new Ed25519 key pair for federation
func GenerateKeyPair() (publicKey, privateKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	return base64.StdEncoding.EncodeToString(pub),
		base64.StdEncoding.EncodeToString(priv),
		nil
}

// SignMessage signs a message with the private key
func SignMessage(privateKeyBase64 string, message []byte) ([]byte, error) {
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}

	return ed25519.Sign(privateKey, message), nil
}

// VerifyMessage verifies a message signature
func VerifyMessage(publicKeyBase64 string, message, signature []byte) error {
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size")
	}

	if !ed25519.Verify(publicKey, message, signature) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// DefaultCapabilities returns the default federation capabilities
func DefaultCapabilities() []string {
	return []string{
		"file_share",
		"user_sync",
		"health_check",
		"peer_discovery",
	}
}
