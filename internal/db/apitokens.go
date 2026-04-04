package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CreateAPIToken creates a new API token.
func (m *Manager) CreateAPIToken(ctx context.Context, token *APIToken) error {
	// Serialize permissions as JSON array
	permsJSONBytes, err := json.Marshal(token.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}
	permsJSON := string(permsJSONBytes)

	query := `INSERT INTO api_tokens (
		id, user_id, name, token_hash, permissions, last_used_at, expires_at, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	var lastUsedAt, expiresAt *string
	if token.LastUsedAt != nil {
		s := token.LastUsedAt.Format(time.RFC3339)
		lastUsedAt = &s
	}
	if token.ExpiresAt != nil {
		s := token.ExpiresAt.Format(time.RFC3339)
		expiresAt = &s
	}

	_, err = m.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.Name, token.TokenHash,
		permsJSON, lastUsedAt, expiresAt, token.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create API token: %w", err)
	}

	return nil
}

// GetAPITokenByHash retrieves an API token by its hash.
func (m *Manager) GetAPITokenByHash(ctx context.Context, tokenHash string) (*APIToken, error) {
	query := `SELECT id, user_id, name, token_hash, permissions, last_used_at, expires_at, created_at
		FROM api_tokens WHERE token_hash = ?`

	token := &APIToken{}
	var permsJSON string
	var lastUsedAt, expiresAt sql.NullString
	var createdAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.Name, &token.TokenHash,
		&permsJSON, &lastUsedAt, &expiresAt, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API token: %w", err)
	}

	// Parse permissions
	token.Permissions = parsePermissionsJSON(permsJSON)

	if lastUsedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastUsedAt.String)
		token.LastUsedAt = &t
	}
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		token.ExpiresAt = &t
	}
	if createdAt.Valid {
		token.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}

	return token, nil
}

// GetAPITokenByID retrieves an API token by ID.
func (m *Manager) GetAPITokenByID(ctx context.Context, id string) (*APIToken, error) {
	query := `SELECT id, user_id, name, token_hash, permissions, last_used_at, expires_at, created_at
		FROM api_tokens WHERE id = ?`

	token := &APIToken{}
	var permsJSON string
	var lastUsedAt, expiresAt sql.NullString
	var createdAt sql.NullString

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&token.ID, &token.UserID, &token.Name, &token.TokenHash,
		&permsJSON, &lastUsedAt, &expiresAt, &createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API token: %w", err)
	}

	token.Permissions = parsePermissionsJSON(permsJSON)

	if lastUsedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastUsedAt.String)
		token.LastUsedAt = &t
	}
	if expiresAt.Valid {
		t, _ := time.Parse(time.RFC3339, expiresAt.String)
		token.ExpiresAt = &t
	}
	if createdAt.Valid {
		token.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	}

	return token, nil
}

// ListAPITokensByUser lists all API tokens for a user.
func (m *Manager) ListAPITokensByUser(ctx context.Context, userID string) ([]*APIToken, error) {
	query := `SELECT id, user_id, name, token_hash, permissions, last_used_at, expires_at, created_at
		FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API tokens: %w", err)
	}
	defer rows.Close()

	tokens := make([]*APIToken, 0)
	for rows.Next() {
		token := &APIToken{}
		var permsJSON string
		var lastUsedAt, expiresAt sql.NullString
		var createdAt sql.NullString

		err := rows.Scan(
			&token.ID, &token.UserID, &token.Name, &token.TokenHash,
			&permsJSON, &lastUsedAt, &expiresAt, &createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API token: %w", err)
		}

		token.Permissions = parsePermissionsJSON(permsJSON)

		if lastUsedAt.Valid {
			t, _ := time.Parse(time.RFC3339, lastUsedAt.String)
			token.LastUsedAt = &t
		}
		if expiresAt.Valid {
			t, _ := time.Parse(time.RFC3339, expiresAt.String)
			token.ExpiresAt = &t
		}
		if createdAt.Valid {
			token.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}

		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate API tokens: %w", err)
	}

	return tokens, nil
}

// UpdateAPITokenLastUsed updates the last used timestamp.
func (m *Manager) UpdateAPITokenLastUsed(ctx context.Context, id string, usedAt time.Time) error {
	query := "UPDATE api_tokens SET last_used_at = ? WHERE id = ?"
	_, err := m.db.ExecContext(ctx, query, usedAt.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("failed to update API token: %w", err)
	}
	return nil
}

// DeleteAPIToken deletes an API token by ID.
func (m *Manager) DeleteAPIToken(ctx context.Context, id string) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM api_tokens WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete API token: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("token not found")
	}

	return nil
}

// DeleteAPITokensByUser deletes all API tokens for a user.
func (m *Manager) DeleteAPITokensByUser(ctx context.Context, userID string) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM api_tokens WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete API tokens: %w", err)
	}
	return nil
}

// CleanupExpiredAPITokens removes expired API tokens.
func (m *Manager) CleanupExpiredAPITokens(ctx context.Context) (int64, error) {
	result, err := m.db.ExecContext(ctx,
		"DELETE FROM api_tokens WHERE expires_at IS NOT NULL AND expires_at < ?",
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup API tokens: %w", err)
	}

	count, _ := result.RowsAffected()
	return count, nil
}

// parsePermissionsJSON parses a JSON array of permissions.
func parsePermissionsJSON(raw string) []string {
	var perms []string
	if err := json.Unmarshal([]byte(raw), &perms); err != nil {
		return []string{}
	}
	return perms
}
