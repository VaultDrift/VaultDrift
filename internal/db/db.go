// Package db provides database access for VaultDrift using CobaltDB.
// CobaltDB is an embedded SQL database with B+Tree storage, WAL, and MVCC.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/vaultdrift/vaultdrift/cobaltdb" // CobaltDB driver
)

// Manager handles database connections and provides data access methods.
type Manager struct {
	db     *sql.DB
	path   string
	mu     sync.RWMutex
	closed bool
}

// Config holds database configuration.
type Config struct {
	Path string
}

// Open opens or creates a database at the specified path.
func Open(cfg Config) (*Manager, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("cobaltdb", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	m := &Manager{
		db:   db,
		path: cfg.Path,
	}

	// Run migrations
	if err := m.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Seed default data
	if err := m.seed(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to seed database: %w", err)
	}

	return m, nil
}

// Close closes the database connection.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	return m.db.Close()
}

// DB returns the underlying sql.DB for raw queries.
func (m *Manager) DB() *sql.DB {
	return m.db
}

// Transaction executes a function within a database transaction.
func (m *Manager) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Exec executes a query without returning rows.
func (m *Manager) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return m.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func (m *Manager) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return m.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row.
func (m *Manager) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return m.db.QueryRowContext(ctx, query, args...)
}

// GetVersion returns the current database schema version.
func (m *Manager) GetVersion() (int, error) {
	var version int
	err := m.db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return version, nil
}

// SetVersion sets the database schema version.
func (m *Manager) SetVersion(version int) error {
	_, err := m.db.Exec(
		"INSERT INTO schema_version (version, applied_at) VALUES (?, ?) "+
			"ON CONFLICT (id) DO UPDATE SET version = ?, applied_at = ?",
		version, time.Now().UTC().Format(time.RFC3339),
		version, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// Stats returns database statistics.
func (m *Manager) Stats() sql.DBStats {
	return m.db.Stats()
}

// Ping verifies the database connection.
func (m *Manager) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}
