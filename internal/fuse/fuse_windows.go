//go:build windows
// +build windows

package fuse

import (
	"errors"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// ErrNotSupported is returned on Windows where FUSE is not supported
var ErrNotSupported = errors.New("FUSE filesystem is not supported on Windows")

// VaultDriftFS is a stub for Windows
type VaultDriftFS struct{}

// Mount is not supported on Windows
func Mount(vfsService *vfs.VFS, database *db.Manager, userID, mountPoint string) error {
	return ErrNotSupported
}

// Unmount is not supported on Windows
func Unmount(mountPoint string) error {
	return ErrNotSupported
}
