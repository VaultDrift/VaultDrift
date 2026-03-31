package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FederationPeer represents a federated server in the database
type FederationPeer struct {
	ID           string    `db:"id"`
	Name         string    `db:"name"`
	PublicURL    string    `db:"public_url"`
	PublicKey    string    `db:"public_key"`
	Status       string    `db:"status"`
	LastSeen     time.Time `db:"last_seen"`
	CreatedAt    time.Time `db:"created_at"`
	Capabilities string    `db:"capabilities"`
}

// FederationInvite represents an invitation to join federation
type FederationInvite struct {
	ID         string    `db:"id"`
	FromPeerID string    `db:"from_peer_id"`
	Token      string    `db:"token"`
	ExpiresAt  time.Time `db:"expires_at"`
	Used       bool      `db:"used"`
	CreatedAt  time.Time `db:"created_at"`
}

// FederatedShare represents a cross-server share
type FederatedShare struct {
	ID           string    `db:"id"`
	LocalFileID  string    `db:"local_file_id"`
	PeerID       string    `db:"peer_id"`
	RemoteUserID string    `db:"remote_user_id"`
	Permission   string    `db:"permission"`
	Token        string    `db:"token"`
	ExpiresAt    time.Time `db:"expires_at"`
	CreatedAt    time.Time `db:"created_at"`
}

// AddFederationPeer adds a federation peer to the database
func (m *Manager) AddFederationPeer(ctx context.Context, peer *FederationPeer) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO federation_peers (id, name, public_url, public_key, status, last_seen, created_at, capabilities)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			public_url = excluded.public_url,
			public_key = excluded.public_key,
			status = excluded.status,
			last_seen = excluded.last_seen,
			capabilities = excluded.capabilities
	`, peer.ID, peer.Name, peer.PublicURL, peer.PublicKey, peer.Status, peer.LastSeen, peer.CreatedAt, peer.Capabilities)

	return err
}

// GetFederationPeers returns all federation peers
func (m *Manager) GetFederationPeers(ctx context.Context) ([]*FederationPeer, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, name, public_url, public_key, status, last_seen, created_at, capabilities
		FROM federation_peers
		WHERE status != 'blocked'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*FederationPeer
	for rows.Next() {
		peer := &FederationPeer{}
		if err := rows.Scan(&peer.ID, &peer.Name, &peer.PublicURL, &peer.PublicKey, &peer.Status, &peer.LastSeen, &peer.CreatedAt, &peer.Capabilities); err != nil {
			return nil, err
		}
		peers = append(peers, peer)
	}

	return peers, rows.Err()
}

// GetFederationPeer returns a specific peer by ID
func (m *Manager) GetFederationPeer(ctx context.Context, peerID string) (*FederationPeer, error) {
	peer := &FederationPeer{}
	err := m.db.QueryRowContext(ctx, `
		SELECT id, name, public_url, public_key, status, last_seen, created_at, capabilities
		FROM federation_peers
		WHERE id = ?
	`, peerID).Scan(&peer.ID, &peer.Name, &peer.PublicURL, &peer.PublicKey, &peer.Status, &peer.LastSeen, &peer.CreatedAt, &peer.Capabilities)
	if err != nil {
		return nil, err
	}
	return peer, nil
}

// RemoveFederationPeer removes a federation peer
func (m *Manager) RemoveFederationPeer(ctx context.Context, peerID string) error {
	_, err := m.db.ExecContext(ctx, `DELETE FROM federation_peers WHERE id = ?`, peerID)
	return err
}

// UpdateFederationPeerStatus updates a peer's status
func (m *Manager) UpdateFederationPeerStatus(ctx context.Context, peerID, status string) error {
	_, err := m.db.ExecContext(ctx, `
		UPDATE federation_peers SET status = ?, last_seen = ? WHERE id = ?
	`, status, time.Now(), peerID)
	return err
}

// CreateFederationInvite creates a federation invitation
func (m *Manager) CreateFederationInvite(ctx context.Context, invite *FederationInvite) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO federation_invites (id, from_peer_id, token, expires_at, used, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, invite.ID, invite.FromPeerID, invite.Token, invite.ExpiresAt, invite.Used, invite.CreatedAt)
	return err
}

// GetFederationInvite returns an invite by token
func (m *Manager) GetFederationInvite(ctx context.Context, token string) (*FederationInvite, error) {
	invite := &FederationInvite{}
	err := m.db.QueryRowContext(ctx, `
		SELECT id, from_peer_id, token, expires_at, used, created_at
		FROM federation_invites
		WHERE token = ?
	`, token).Scan(&invite.ID, &invite.FromPeerID, &invite.Token, &invite.ExpiresAt, &invite.Used, &invite.CreatedAt)
	if err != nil {
		return nil, err
	}
	return invite, nil
}

// UseFederationInvite marks an invite as used
func (m *Manager) UseFederationInvite(ctx context.Context, inviteID string) error {
	_, err := m.db.ExecContext(ctx, `
		UPDATE federation_invites SET used = 1 WHERE id = ?
	`, inviteID)
	return err
}

// AddFederatedShare adds a federated share
func (m *Manager) AddFederatedShare(ctx context.Context, share *FederatedShare) error {
	_, err := m.db.ExecContext(ctx, `
		INSERT INTO federated_shares (id, local_file_id, peer_id, remote_user_id, permission, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, share.ID, share.LocalFileID, share.PeerID, share.RemoteUserID, share.Permission, share.Token, share.ExpiresAt, share.CreatedAt)
	return err
}

// GetFederatedShare returns a federated share by token
func (m *Manager) GetFederatedShare(ctx context.Context, token string) (*FederatedShare, error) {
	share := &FederatedShare{}
	err := m.db.QueryRowContext(ctx, `
		SELECT id, local_file_id, peer_id, remote_user_id, permission, token, expires_at, created_at
		FROM federated_shares
		WHERE token = ? AND (expires_at IS NULL OR expires_at > ?)
	`, token, time.Now()).Scan(&share.ID, &share.LocalFileID, &share.PeerID, &share.RemoteUserID, &share.Permission, &share.Token, &share.ExpiresAt, &share.CreatedAt)
	if err != nil {
		return nil, err
	}
	return share, nil
}

// joinCapabilities joins capability strings
func joinCapabilities(caps []string) string {
	return fmt.Sprintf("[%s]", strings.Join(caps, ","))
}
