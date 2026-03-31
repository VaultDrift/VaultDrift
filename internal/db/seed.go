package db

import (
	"time"

	"github.com/vaultdrift/vaultdrift/internal/util"
)

// seed populates the database with default data.
func (m *Manager) seed() error {
	// Check if we already have data
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM roles").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Already seeded
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Seed default roles
	roles := []struct {
		id          string
		name        string
		description string
		isSystem    bool
	}{
		{"role_admin", "admin", "Full system control", true},
		{"role_user", "user", "Standard user", true},
		{"role_guest", "guest", "Guest user with limited access", true},
	}

	for _, role := range roles {
		_, err := m.db.Exec(
			"INSERT INTO roles (id, name, description, is_system, created_at) VALUES (?, ?, ?, ?, ?)",
			role.id, role.name, role.description, role.isSystem, now,
		)
		if err != nil {
			return err
		}
	}

	// Seed admin role permissions
	adminPerms := []struct {
		resource string
		action   string
		scope    string
	}{
		{"file", "read", "all"},
		{"file", "write", "all"},
		{"file", "delete", "all"},
		{"file", "share", "all"},
		{"folder", "read", "all"},
		{"folder", "write", "all"},
		{"folder", "delete", "all"},
		{"folder", "share", "all"},
		{"user", "read", "all"},
		{"user", "write", "all"},
		{"user", "delete", "all"},
		{"user", "manage", "all"},
		{"share", "read", "all"},
		{"share", "write", "all"},
		{"share", "delete", "all"},
		{"system", "read", "all"},
		{"system", "write", "all"},
		{"system", "manage", "all"},
	}

	for _, perm := range adminPerms {
		id, _ := util.GenerateUUIDv7()
		_, err := m.db.Exec(
			"INSERT INTO permissions (id, role_id, resource, action, scope) VALUES (?, ?, ?, ?, ?)",
			id, "role_admin", perm.resource, perm.action, perm.scope,
		)
		if err != nil {
			return err
		}
	}

	// Seed user role permissions
	userPerms := []struct {
		resource string
		action   string
		scope    string
	}{
		{"file", "read", "own"},
		{"file", "write", "own"},
		{"file", "delete", "own"},
		{"file", "share", "own"},
		{"file", "read", "group"},
		{"folder", "read", "own"},
		{"folder", "write", "own"},
		{"folder", "delete", "own"},
		{"folder", "share", "own"},
		{"folder", "read", "group"},
		{"share", "read", "own"},
		{"share", "write", "own"},
		{"share", "delete", "own"},
	}

	for _, perm := range userPerms {
		id, _ := util.GenerateUUIDv7()
		_, err := m.db.Exec(
			"INSERT INTO permissions (id, role_id, resource, action, scope) VALUES (?, ?, ?, ?, ?)",
			id, "role_user", perm.resource, perm.action, perm.scope,
		)
		if err != nil {
			return err
		}
	}

	// Seed guest role permissions
	guestPerms := []struct {
		resource string
		action   string
		scope    string
	}{
		{"file", "read", "group"},
		{"folder", "read", "group"},
	}

	for _, perm := range guestPerms {
		id, _ := util.GenerateUUIDv7()
		_, err := m.db.Exec(
			"INSERT INTO permissions (id, role_id, resource, action, scope) VALUES (?, ?, ?, ?, ?)",
			id, "role_guest", perm.resource, perm.action, perm.scope,
		)
		if err != nil {
			return err
		}
	}

	// Seed default admin user (password: admin)
	// Pre-computed PHC hash for "admin"
	adminPasswordHash := "$argon2id$v=19$m=65536,t=3,p=1$6IOdsT/ZB3Yc39wdhST32A$0psObWN9M9SCF1zLWMsE9lef9JFfK2pqsY1o2M3x2zg"

	adminID, _ := util.GenerateUUIDv7()
	_, err = m.db.Exec(
		`INSERT INTO users (id, username, email, display_name, password_hash, role, quota_bytes, used_bytes,
		status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		adminID, "admin", "admin@localhost", "Administrator", adminPasswordHash, "admin",
		10737418240, 0, "active", now, now,
	)
	if err != nil {
		return err
	}

	return nil
}
