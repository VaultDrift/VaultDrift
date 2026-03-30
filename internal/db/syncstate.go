package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CreateDevice creates a new device.
func (m *Manager) CreateDevice(ctx context.Context, device *Device) error {
	query := `INSERT INTO devices (id, user_id, name, device_type, os, sync_folder,
		last_sync_at, vector_clock, merkle_root, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := m.db.ExecContext(ctx, query,
		device.ID, device.UserID, device.Name, device.DeviceType, device.OS,
		device.SyncFolder, device.LastSyncAt, device.VectorClock, device.MerkleRoot,
		boolToInt(device.IsActive), device.CreatedAt.Format(time.RFC3339), device.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}

	return nil
}

// GetDeviceByID retrieves a device by ID.
func (m *Manager) GetDeviceByID(ctx context.Context, id string) (*Device, error) {
	query := `SELECT id, user_id, name, device_type, os, sync_folder,
		last_sync_at, vector_clock, merkle_root, is_active, created_at, updated_at
	FROM devices WHERE id = ?`

	device := &Device{}
	var lastSyncAt, merkleRoot sql.NullString

	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&device.ID, &device.UserID, &device.Name, &device.DeviceType, &device.OS,
		&device.SyncFolder, &lastSyncAt, &device.VectorClock, &merkleRoot,
		&device.IsActive, &device.CreatedAt, &device.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	if lastSyncAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastSyncAt.String)
		device.LastSyncAt = &t
	}
	if merkleRoot.Valid {
		device.MerkleRoot = &merkleRoot.String
	}

	return device, nil
}

// GetDevicesByUser retrieves all devices for a user.
func (m *Manager) GetDevicesByUser(ctx context.Context, userID string) ([]*Device, error) {
	query := `SELECT id, user_id, name, device_type, os, sync_folder,
		last_sync_at, vector_clock, merkle_root, is_active, created_at, updated_at
	FROM devices WHERE user_id = ? ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	devices := make([]*Device, 0)
	for rows.Next() {
		device := &Device{}
		var lastSyncAt, merkleRoot sql.NullString

		err := rows.Scan(
			&device.ID, &device.UserID, &device.Name, &device.DeviceType, &device.OS,
			&device.SyncFolder, &lastSyncAt, &device.VectorClock, &merkleRoot,
			&device.IsActive, &device.CreatedAt, &device.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}

		if lastSyncAt.Valid {
			t, _ := time.Parse(time.RFC3339, lastSyncAt.String)
			device.LastSyncAt = &t
		}
		if merkleRoot.Valid {
			device.MerkleRoot = &merkleRoot.String
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// UpdateDevice updates a device.
func (m *Manager) UpdateDevice(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	setClauses := make([]string, 0, len(updates))
	args := make([]any, 0, len(updates)+2)

	for field, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", field))
		if b, ok := value.(bool); ok {
			args = append(args, boolToInt(b))
		} else {
			args = append(args, value)
		}
	}

	setClauses = append(setClauses, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE devices SET %s WHERE id = ?", setClauses)

	result, err := m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("device not found")
	}

	return nil
}

// DeleteDevice deletes a device.
func (m *Manager) DeleteDevice(ctx context.Context, id string) error {
	result, err := m.db.ExecContext(ctx, "DELETE FROM devices WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("device not found")
	}

	return nil
}

// CreateOrUpdateSyncState creates or updates sync state.
func (m *Manager) CreateOrUpdateSyncState(ctx context.Context, state *SyncState) error {
	query := `INSERT INTO sync_state (id, device_id, file_id, manifest_id, vector_clock, synced_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (device_id, file_id) DO UPDATE SET
			manifest_id = excluded.manifest_id,
			vector_clock = excluded.vector_clock,
			synced_at = excluded.synced_at`

	_, err := m.db.ExecContext(ctx, query,
		state.ID, state.DeviceID, state.FileID, state.ManifestID,
		state.VectorClock, state.SyncedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create/update sync state: %w", err)
	}

	return nil
}

// GetSyncState retrieves sync state for a device and file.
func (m *Manager) GetSyncState(ctx context.Context, deviceID, fileID string) (*SyncState, error) {
	query := `SELECT id, device_id, file_id, manifest_id, vector_clock, synced_at
	FROM sync_state WHERE device_id = ? AND file_id = ?`

	state := &SyncState{}
	err := m.db.QueryRowContext(ctx, query, deviceID, fileID).Scan(
		&state.ID, &state.DeviceID, &state.FileID, &state.ManifestID,
		&state.VectorClock, &state.SyncedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sync state not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync state: %w", err)
	}

	return state, nil
}

// GetSyncStatesByDevice retrieves all sync states for a device.
func (m *Manager) GetSyncStatesByDevice(ctx context.Context, deviceID string) ([]*SyncState, error) {
	query := `SELECT id, device_id, file_id, manifest_id, vector_clock, synced_at
	FROM sync_state WHERE device_id = ? ORDER BY synced_at DESC`

	rows, err := m.db.QueryContext(ctx, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sync states: %w", err)
	}
	defer rows.Close()

	states := make([]*SyncState, 0)
	for rows.Next() {
		state := &SyncState{}
		err := rows.Scan(
			&state.ID, &state.DeviceID, &state.FileID, &state.ManifestID,
			&state.VectorClock, &state.SyncedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sync state: %w", err)
		}
		states = append(states, state)
	}

	return states, nil
}
