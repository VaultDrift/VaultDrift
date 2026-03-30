package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CreateSession creates a new session.
func (m *Manager) CreateSession(ctx context.Context, session *Session) error {
	query := `INSERT INTO sessions (id, user_id, refresh_token, device_name, device_type,
		ip_address, user_agent, last_active_at, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.RefreshToken, session.DeviceName,
		session.DeviceType, session.IPAddress, session.UserAgent,
		session.LastActiveAt.Format(time.RFC3339), session.ExpiresAt.Format(time.RFC3339),
		session.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSessionByRefreshToken retrieves a session by refresh token.
func (m *Manager) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	query := `SELECT id, user_id, refresh_token, device_name, device_type,
		ip_address, user_agent, last_active_at, expires_at, created_at
	FROM sessions WHERE refresh_token = ?`

	session := &Session{}
	err := m.db.QueryRowContext(ctx, query, refreshToken).Scan(
		&session.ID, &session.UserID, &session.RefreshToken, &session.DeviceName,
		&session.DeviceType, &session.IPAddress, &session.UserAgent,
		&session.LastActiveAt, &session.ExpiresAt, &session.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// GetSessionsByUser retrieves all sessions for a user.
func (m *Manager) GetSessionsByUser(ctx context.Context, userID string) ([]*Session, error) {
	query := `SELECT id, user_id, refresh_token, device_name, device_type,
		ip_address, user_agent, last_active_at, expires_at, created_at
	FROM sessions WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]*Session, 0)
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(
			&session.ID, &session.UserID, &session.RefreshToken, &session.DeviceName,
			&session.DeviceType, &session.IPAddress, &session.UserAgent,
			&session.LastActiveAt, &session.ExpiresAt, &session.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// UpdateSessionActivity updates the last active time of a session.
func (m *Manager) UpdateSessionActivity(ctx context.Context, id string) error {
	_, err := m.db.ExecContext(ctx,
		"UPDATE sessions SET last_active_at = ? WHERE id = ?",
		time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// DeleteSession deletes a session.
func (m *Manager) DeleteSession(ctx context.Context, id string) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// DeleteSessionsByUser deletes all sessions for a user (logout everywhere).
func (m *Manager) DeleteSessionsByUser(ctx context.Context, userID string) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}
	return nil
}

// CleanupExpiredSessions deletes all expired sessions.
func (m *Manager) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	result, err := m.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE expires_at < ?",
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup sessions: %w", err)
	}

	count, _ := result.RowsAffected()
	return count, nil
}
