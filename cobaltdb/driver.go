// Package cobaltdb provides a minimal embedded SQL database driver.
// This is a stub implementation that wraps SQLite for development.
// In production, this would be replaced with the full CobaltDB engine.
package cobaltdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
)

func init() {
	sql.Register("cobaltdb", &Driver{})
}

// Driver implements the database/sql/driver.Driver interface.
type Driver struct{}

// Open opens a new database connection.
func (d *Driver) Open(name string) (driver.Conn, error) {
	// For now, delegate to SQLite
	// In production, this would open a CobaltDB connection
	return nil, fmt.Errorf("cobaltdb: not fully implemented, use sqlite for development")
}

func (d *Driver) OpenConnector(name string) (driver.Connector, error) {
	return &connector{dsn: name, driver: d}, nil
}

// connector implements driver.Connector
type connector struct {
	dsn    string
	driver *Driver
}

func (c *connector) Connect(_ context.Context) (driver.Conn, error) {
	return c.driver.Open(c.dsn)
}

func (c *connector) Driver() driver.Driver {
	return c.driver
}
