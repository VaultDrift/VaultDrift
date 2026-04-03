package vfs

import "errors"

// Common VFS errors.
var (
	ErrNotFound         = errors.New("file or folder not found")
	ErrAlreadyExists    = errors.New("file or folder already exists")
	ErrInvalidPath      = errors.New("invalid path")
	ErrIsFile           = errors.New("path is a file, not a folder")
	ErrPermissionDenied = errors.New("permission denied")
	ErrInvalidName      = errors.New("invalid name")
)
