package db

import (
	"context"
	"time"
)

// CountActiveShares returns the number of non-expired, non-revoked shares.
func (m *Manager) CountActiveShares(ctx context.Context) (int64, error) {
	var count int64
	err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM shares
		WHERE is_active = 1 AND (expires_at IS NULL OR expires_at > ?)
	`, time.Now().UTC()).Scan(&count)
	return count, err
}
