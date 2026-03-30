package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// GetSetting retrieves a setting value by key.
func (m *Manager) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := m.db.QueryRowContext(ctx,
		"SELECT value FROM settings WHERE key = ?",
		key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("setting not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}

// GetSettingOrDefault retrieves a setting value or returns default if not found.
func (m *Manager) GetSettingOrDefault(ctx context.Context, key, defaultValue string) string {
	value, err := m.GetSetting(ctx, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// SetSetting creates or updates a setting.
func (m *Manager) SetSetting(ctx context.Context, key, value string) error {
	query := `INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at`

	_, err := m.db.ExecContext(ctx, query, key, value, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}
	return nil
}

// DeleteSetting removes a setting.
func (m *Manager) DeleteSetting(ctx context.Context, key string) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM settings WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("setting not found")
	}
	return nil
}

// GetAllSettings retrieves all settings.
func (m *Manager) GetAllSettings(ctx context.Context) (map[string]string, error) {
	query := "SELECT key, value FROM settings"

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings[key] = value
	}

	return settings, nil
}
