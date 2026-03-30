package vfs

import "errors"

// Common VFS errors.
var (
	ErrNotFound          = errors.New("file or folder not found")
	ErrAlreadyExists     = errors.New("file or folder already exists")
	ErrInvalidPath       = errors.New("invalid path")
	ErrIsFolder          = errors.New("path is a folder, not a file")
	ErrIsFile            = errors.New("path is a file, not a folder")
	ErrNotEmpty          = errors.New("folder is not empty")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrInvalidName       = errors.New("invalid name")
	ErrParentNotFound    = errors.New("parent folder not found")
	ErrCrossDevice       = errors.New("cross-device move not supported")
	ErrRootDelete        = errors.New("cannot delete root folder")
	ErrQuotaExceeded     = errors.New("storage quota exceeded")
	ErrFileLocked        = errors.New("file is locked")
	ErrVersionNotFound   = errors.New("version not found")
	ErrTrashNotFound     = errors.New("item not found in trash")
	ErrInvalidOperation  = errors.New("invalid operation")
)
