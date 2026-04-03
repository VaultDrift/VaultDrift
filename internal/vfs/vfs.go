package vfs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
)

// VFS implements a virtual filesystem layer on top of the database.
type VFS struct {
	db *db.Manager
}

// NewVFS creates a new virtual filesystem.
func NewVFS(db *db.Manager) *VFS {
	return &VFS{db: db}
}

// CreateFolder creates a new folder.
func (v *VFS) CreateFolder(ctx context.Context, userID, parentID, name string) (*db.File, error) {
	// Validate name
	if !isValidName(name) {
		return nil, ErrInvalidName
	}

	// Check if folder already exists in this parent
	_, err := v.db.GetFileByPath(ctx, userID, &parentID, name)
	if err == nil {
		return nil, ErrAlreadyExists
	}

	// Create folder record
	now := time.Now().UTC()
	folder := &db.File{
		ID:        generateID("folder"),
		UserID:    userID,
		ParentID:  &parentID,
		Name:      name,
		Type:      "folder",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if parentID == "" {
		folder.ParentID = nil // Root level
	}

	if err := v.db.CreateFile(ctx, folder); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	return folder, nil
}

// CreateFile creates a new file entry (without content).
func (v *VFS) CreateFile(ctx context.Context, userID, parentID, name, mimeType string, size int64) (*db.File, error) {
	// Validate name
	if !isValidName(name) {
		return nil, ErrInvalidName
	}

	// Check if file already exists in this parent
	_, err := v.db.GetFileByPath(ctx, userID, &parentID, name)
	if err == nil {
		return nil, ErrAlreadyExists
	}

	// Create file record
	now := time.Now().UTC()
	file := &db.File{
		ID:        generateID("file"),
		UserID:    userID,
		ParentID:  &parentID,
		Name:      name,
		Type:      "file",
		SizeBytes: size,
		MimeType:  mimeType,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if parentID == "" {
		file.ParentID = nil
	}

	if err := v.db.CreateFile(ctx, file); err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}

// GetFile retrieves a file by ID.
func (v *VFS) GetFile(ctx context.Context, fileID string) (*db.File, error) {
	file, err := v.db.GetFileByID(ctx, fileID)
	if err != nil {
		return nil, ErrNotFound
	}
	return file, nil
}

// GetFileByPath retrieves a file by user, parent, and name.
func (v *VFS) GetFileByPath(ctx context.Context, userID, parentID, name string) (*db.File, error) {
	file, err := v.db.GetFileByPath(ctx, userID, &parentID, name)
	if err != nil {
		return nil, ErrNotFound
	}
	return file, nil
}

// ListDirectory lists files and folders in a directory.
func (v *VFS) ListDirectory(ctx context.Context, userID, parentID string, opts db.ListOpts) ([]*db.File, error) {
	files, err := v.db.ListDirectory(ctx, userID, &parentID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}
	return files, nil
}

// Rename renames a file or folder.
func (v *VFS) Rename(ctx context.Context, fileID, newName string) error {
	if !isValidName(newName) {
		return ErrInvalidName
	}

	updates := map[string]any{
		"name": newName,
	}

	if err := v.db.UpdateFile(ctx, fileID, updates); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	return nil
}

// Move moves a file or folder to a new parent.
func (v *VFS) Move(ctx context.Context, fileID, newParentID, newName string) error {
	if newName != "" && !isValidName(newName) {
		return ErrInvalidName
	}

	if newName == "" {
		// Just move, keep same name
		file, err := v.db.GetFileByID(ctx, fileID)
		if err != nil {
			return ErrNotFound
		}
		newName = file.Name
	}

	if err := v.db.MoveFile(ctx, fileID, newParentID, newName); err != nil {
		return fmt.Errorf("failed to move: %w", err)
	}

	return nil
}

// Delete moves a file or folder to trash (soft delete).
func (v *VFS) Delete(ctx context.Context, fileID string) error {
	if err := v.db.SoftDelete(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

// DeletePermanent permanently deletes a file or folder.
func (v *VFS) DeletePermanent(ctx context.Context, fileID string) error {
	if err := v.db.PermanentDelete(ctx, fileID); err != nil {
		return fmt.Errorf("failed to permanently delete: %w", err)
	}
	return nil
}

// Restore restores a file or folder from trash.
func (v *VFS) Restore(ctx context.Context, fileID string) error {
	if err := v.db.RestoreFromTrash(ctx, fileID); err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}
	return nil
}

// ListTrash lists trashed files for a user.
func (v *VFS) ListTrash(ctx context.Context, userID string) ([]*db.File, error) {
	files, err := v.db.ListTrash(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list trash: %w", err)
	}
	return files, nil
}

// Search searches files by name for a user.
func (v *VFS) Search(ctx context.Context, userID, query string, limit int) ([]*db.File, error) {
	if limit <= 0 {
		limit = 100
	}

	files, err := v.db.SearchFiles(ctx, userID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	return files, nil
}

// Recent returns recently modified files for a user.
func (v *VFS) Recent(ctx context.Context, userID string, limit int) ([]*db.File, error) {
	if limit <= 0 {
		limit = 50
	}

	files, err := v.db.RecentFiles(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent files: %w", err)
	}
	return files, nil
}

// GetFolderPath returns the full path of a folder (for display purposes).
func (v *VFS) GetFolderPath(ctx context.Context, folderID string) (string, error) {
	if folderID == "" {
		return "/", nil
	}

	folder, err := v.db.GetFileByID(ctx, folderID)
	if err != nil {
		return "", ErrNotFound
	}

	if folder.Type != "folder" {
		return "", ErrIsFile
	}

	// Build path by traversing up
	parts := []string{folder.Name}
	currentID := folder.ParentID

	for currentID != nil && *currentID != "" {
		parent, err := v.db.GetFileByID(ctx, *currentID)
		if err != nil {
			break
		}
		parts = append([]string{parent.Name}, parts...)
		currentID = parent.ParentID
	}

	return "/" + joinPath(parts), nil
}

// GetBreadcrumbs returns the breadcrumb trail for a folder.
func (v *VFS) GetBreadcrumbs(ctx context.Context, folderID string) ([]Breadcrumb, error) {
	var crumbs []Breadcrumb

	currentID := &folderID
	for currentID != nil && *currentID != "" {
		folder, err := v.db.GetFileByID(ctx, *currentID)
		if err != nil {
			break
		}

		crumbs = append([]Breadcrumb{{ID: folder.ID, Name: folder.Name}}, crumbs...)
		currentID = folder.ParentID
	}

	// Add root
	crumbs = append([]Breadcrumb{{ID: "", Name: "root"}}, crumbs...)
	return crumbs, nil
}

// Breadcrumb represents a breadcrumb entry.
type Breadcrumb struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Copy copies a file or folder to a new parent with optional new name.
func (v *VFS) Copy(ctx context.Context, userID, fileID, newParentID, newName string) error {
	// Get source file
	source, err := v.db.GetFileByID(ctx, fileID)
	if err != nil {
		return ErrNotFound
	}

	// Verify ownership
	if source.UserID != userID {
		return ErrPermissionDenied
	}

	// Use original name if no new name provided
	if newName == "" {
		newName = source.Name
	}

	// Validate new name
	if !isValidName(newName) {
		return ErrInvalidName
	}

	// Check if destination already exists
	_, err = v.db.GetFileByPath(ctx, userID, &newParentID, newName)
	if err == nil {
		return ErrAlreadyExists
	}

	// Create the copy
	now := time.Now().UTC()
	copyFile := &db.File{
		ID:           generateID(source.Type),
		UserID:       userID,
		ParentID:     &newParentID,
		Name:         newName,
		Type:         source.Type,
		SizeBytes:    source.SizeBytes,
		MimeType:     source.MimeType,
		ManifestID:   source.ManifestID,
		Checksum:     source.Checksum,
		IsEncrypted:  source.IsEncrypted,
		EncryptedKey: source.EncryptedKey,
		Version:      1, // Reset version for copy
		IsTrashed:    false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if newParentID == "" {
		copyFile.ParentID = nil
	}

	if err := v.db.CreateFile(ctx, copyFile); err != nil {
		return fmt.Errorf("failed to create copy: %w", err)
	}

	// If it's a folder, recursively copy contents
	if source.Type == "folder" {
		if err := v.copyFolderContents(ctx, userID, fileID, copyFile.ID); err != nil {
			return fmt.Errorf("failed to copy folder contents: %w", err)
		}
	}

	return nil
}

// copyFolderContents recursively copies folder contents.
func (v *VFS) copyFolderContents(ctx context.Context, userID, sourceFolderID, destFolderID string) error {
	// List all items in source folder
	opts := db.ListOpts{Limit: 1000}
	items, err := v.db.ListDirectory(ctx, userID, &sourceFolderID, opts)
	if err != nil {
		return err
	}

	for _, item := range items {
		now := time.Now().UTC()
		copyItem := &db.File{
			ID:           generateID(item.Type),
			UserID:       userID,
			ParentID:     &destFolderID,
			Name:         item.Name,
			Type:         item.Type,
			SizeBytes:    item.SizeBytes,
			MimeType:     item.MimeType,
			ManifestID:   item.ManifestID,
			Checksum:     item.Checksum,
			IsEncrypted:  item.IsEncrypted,
			EncryptedKey: item.EncryptedKey,
			Version:      item.Version,
			IsTrashed:    false,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := v.db.CreateFile(ctx, copyItem); err != nil {
			return fmt.Errorf("failed to copy item %s: %w", item.Name, err)
		}

		// Recursively copy subfolders
		if item.Type == "folder" {
			if err := v.copyFolderContents(ctx, userID, item.ID, copyItem.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

// Helper functions

func isValidName(name string) bool {
	if name == "" {
		return false
	}
	if name == "." || name == ".." {
		return false
	}
	// Check for invalid characters
	for _, c := range name {
		// Reject path separators, null bytes, newlines, and control characters
		if c == '/' || c == '\x00' || c == '\n' || c == '\r' || c == '\t' || c < 32 {
			return false
		}
	}
	return true
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func joinPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(parts[0])
	for _, p := range parts[1:] {
		b.WriteByte('/')
		b.WriteString(p)
	}
	return b.String()
}
