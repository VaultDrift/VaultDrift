package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// CreateAuditEntry creates a new audit log entry.
func (m *Manager) CreateAuditEntry(ctx context.Context, entry *AuditEntry) error {
	query := `INSERT INTO audit_log (id, user_id, action, resource_type, resource_id,
		details, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		entry.ID, entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID,
		entry.Details, entry.IPAddress, entry.UserAgent,
		entry.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create audit entry: %w", err)
	}

	return nil
}

// GetAuditEntries retrieves audit log entries with filtering.
func (m *Manager) GetAuditEntries(ctx context.Context, userID *string, action, resourceType string, start, end time.Time, limit, offset int) ([]*AuditEntry, int, error) {
	whereClauses := []string{"1 = 1"}
	args := []any{}

	if userID != nil {
		whereClauses = append(whereClauses, "user_id = ?")
		args = append(args, *userID)
	}

	if action != "" {
		whereClauses = append(whereClauses, "action = ?")
		args = append(args, action)
	}

	if resourceType != "" {
		whereClauses = append(whereClauses, "resource_type = ?")
		args = append(args, resourceType)
	}

	if !start.IsZero() {
		whereClauses = append(whereClauses, "created_at >= ?")
		args = append(args, start.Format(time.RFC3339))
	}

	if !end.IsZero() {
		whereClauses = append(whereClauses, "created_at <= ?")
		args = append(args, end.Format(time.RFC3339))
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM audit_log WHERE " + strings.Join(whereClauses, " AND ")
	var total int
	if err := m.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count audit entries: %w", err)
	}

	// Get entries
	query := `SELECT id, user_id, action, resource_type, resource_id,
		details, ip_address, user_agent, created_at
	FROM audit_log WHERE ` + strings.Join(whereClauses, " AND ") +
		` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	args = append(args, limit, offset)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list audit entries: %w", err)
	}
	defer rows.Close()

	entries := make([]*AuditEntry, 0)
	for rows.Next() {
		entry := &AuditEntry{}
		var createdAt sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.UserID, &entry.Action, &entry.ResourceType, &entry.ResourceID,
			&entry.Details, &entry.IPAddress, &entry.UserAgent, &createdAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if createdAt.Valid {
			entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
		}

		entries = append(entries, entry)
	}

	return entries, total, nil
}

// CleanupOldAuditEntries removes audit entries older than the specified duration.
func (m *Manager) CleanupOldAuditEntries(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	result, err := m.db.ExecContext(ctx,
		"DELETE FROM audit_log WHERE created_at < ?",
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup audit entries: %w", err)
	}

	count, _ := result.RowsAffected()
	return count, nil
}
