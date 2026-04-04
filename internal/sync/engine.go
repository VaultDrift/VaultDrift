package sync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
)

// CommitRequest is what the client sends to apply changes.
type CommitRequest struct {
	DeviceID    string       `json:"device_id"`
	VectorClock VectorClock  `json:"vector_clock"`
	Changes     []FileChange `json:"changes"`
}

// FileChange represents a single file change to commit.
type FileChange struct {
	Path        string      `json:"path"`
	Action      string      `json:"action"` // "add", "modify", "delete"
	SizeBytes   int64       `json:"size_bytes,omitempty"`
	MimeType    string      `json:"mime_type,omitempty"`
	ChunkHashes []string    `json:"chunk_hashes,omitempty"`
	ManifestID  string      `json:"manifest_id,omitempty"`
	VectorClock VectorClock `json:"vector_clock"`
}

// CommitResponse is what the server returns after committing changes.
type CommitResponse struct {
	Status       string         `json:"status"` // "committed", "conflict"
	NewRoot      string         `json:"new_root"`
	VectorClock  VectorClock    `json:"vector_clock"`
	Conflicts    []ConflictInfo `json:"conflicts,omitempty"`
	AppliedCount int            `json:"applied_count"`
}

// ConflictInfo describes a conflict that was detected.
type ConflictInfo struct {
	Path          string `json:"path"`
	LocalVersion  int    `json:"local_version"`
	RemoteVersion int    `json:"remote_version"`
	Resolution    string `json:"resolution"` // "server_wins", "client_wins", "conflict_copy"
}

// SyncStatus represents the current sync state for a device.
type SyncStatus struct {
	DeviceID    string      `json:"device_id"`
	LastSyncAt  *time.Time  `json:"last_sync_at"`
	MerkleRoot  string      `json:"merkle_root"`
	VectorClock VectorClock `json:"vector_clock"`
	FileCount   int         `json:"file_count"`
	PendingOps  int         `json:"pending_ops"`
}

// Engine orchestrates sync operations.
type Engine struct {
	db      *db.Manager
	storage storage.Backend
	vfs     PathResolver
}

// PathResolver abstracts VFS file listing for the sync engine.
type PathResolver interface {
	ListAllFiles(ctx context.Context, userID string) ([]FileInfo, error)
}

// NegotiateRequest is the request body for the negotiate endpoint.
type NegotiateRequest struct {
	DeviceID    string      `json:"device_id"`
	MerkleRoot  string      `json:"merkle_root"`
	VectorClock VectorClock `json:"vector_clock"`
	ClientFiles []FileInfo  `json:"client_files"`
}

// NegotiateResponse is the response from the negotiate endpoint.
type NegotiateResponse struct {
	Status       string      `json:"status"`
	ServerRoot   string      `json:"server_root"`
	VectorClock  VectorClock `json:"vector_clock"`
	Diff         *DiffResult `json:"diff,omitempty"`
	NeededChunks []string    `json:"needed_chunks,omitempty"`
	HaveChunks   []string    `json:"have_chunks,omitempty"`
}

// NewEngine creates a new sync engine.
func NewEngine(database *db.Manager, store storage.Backend) *Engine {
	return &Engine{
		db:      database,
		storage: store,
	}
}

// SetVFS sets the path resolver (called after VFS initialization).
func (e *Engine) SetVFS(vfs PathResolver) {
	e.vfs = vfs
}

// Negotiate compares client state with server state and returns a diff.
func (e *Engine) Negotiate(ctx context.Context, userID string, req NegotiateRequest) (*NegotiateResponse, error) {
	// Build server merkle tree from current files
	serverFiles, err := e.vfs.ListAllFiles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list server files: %w", err)
	}

	serverRoot := Build(serverFiles)
	serverRootHash := ""
	if serverRoot != nil {
		serverRootHash = serverRoot.Hash
	}

	// Quick check: if roots match, no diff needed
	if serverRootHash == req.MerkleRoot && req.MerkleRoot != "" {
		return &NegotiateResponse{
			Status:      "synced",
			ServerRoot:  serverRootHash,
			VectorClock: req.VectorClock,
		}, nil
	}

	// Build client tree from provided files
	clientRoot := Build(req.ClientFiles)

	// Compute diff
	var diff *DiffResult
	if clientRoot != nil && serverRoot != nil {
		diff = Diff(clientRoot, serverRoot)
	} else if serverRoot != nil {
		// Client is empty — everything is new for client
		diff = &DiffResult{
			Added:    collectAllPaths(serverRoot),
			Modified: []string{},
			Deleted:  []string{},
		}
	} else {
		// Server is empty — everything from client is new
		diff = &DiffResult{
			Added:    []string{},
			Modified: []string{},
			Deleted:  []string{},
		}
	}

	// Determine which chunks the server needs
	neededChunks := e.findNeededChunks(ctx, userID, diff.Modified)

	// Determine which chunks the client may need (for added/modified files)
	haveChunks := e.findAvailableChunks(ctx, userID, diff.Added, diff.Modified)

	// Merge vector clocks
	mergedClock := req.VectorClock.Copy()

	// Get server device's vector clock if available
	device, err := e.db.GetDeviceByID(ctx, req.DeviceID)
	if err == nil && device.VectorClock != "" {
		var serverClock VectorClock
		if err := json.Unmarshal([]byte(device.VectorClock), &serverClock); err == nil {
			mergedClock.Merge(serverClock)
		}
	}

	return &NegotiateResponse{
		Status:       "diff",
		ServerRoot:   serverRootHash,
		VectorClock:  mergedClock,
		Diff:         diff,
		NeededChunks: neededChunks,
		HaveChunks:   haveChunks,
	}, nil
}

// Commit applies a batch of file changes from a client.
func (e *Engine) Commit(ctx context.Context, userID string, req CommitRequest) (*CommitResponse, error) {
	appliedCount := 0
	var conflicts []ConflictInfo

	for _, change := range req.Changes {
		// Check for conflicts using vector clocks
		if conflict := e.detectConflict(ctx, userID, change); conflict != nil {
			conflicts = append(conflicts, *conflict)
			// Auto-resolve: server wins for concurrent modifications
			if change.Action == "modify" {
				log.Printf("sync: conflict on %s, server version retained", change.Path)
				continue
			}
		}

		// Apply the change
		if err := e.applyChange(ctx, userID, change); err != nil {
			log.Printf("sync: failed to apply %s on %s: %v", change.Action, change.Path, err)
			continue
		}
		appliedCount++
	}

	// Update device state
	e.updateDeviceState(ctx, req.DeviceID, req.VectorClock)

	// Build new server tree
	serverFiles, err := e.vfs.ListAllFiles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to rebuild tree: %w", err)
	}
	newRoot := Build(serverFiles)
	newRootHash := ""
	if newRoot != nil {
		newRootHash = newRoot.Hash
	}

	// Increment vector clock
	req.VectorClock.Increment(req.DeviceID)

	status := "committed"
	if len(conflicts) > 0 {
		status = "partial"
	}

	return &CommitResponse{
		Status:       status,
		NewRoot:      newRootHash,
		VectorClock:  req.VectorClock,
		Conflicts:    conflicts,
		AppliedCount: appliedCount,
	}, nil
}

// GetStatus returns the sync status for a device.
func (e *Engine) GetStatus(ctx context.Context, deviceID string) (*SyncStatus, error) {
	device, err := e.db.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	var vc VectorClock
	if device.VectorClock != "" {
		_ = json.Unmarshal([]byte(device.VectorClock), &vc)
	}

	merkleRoot := ""
	if device.MerkleRoot != nil {
		merkleRoot = *device.MerkleRoot
	}

	return &SyncStatus{
		DeviceID:    deviceID,
		LastSyncAt:  device.LastSyncAt,
		MerkleRoot:  merkleRoot,
		VectorClock: vc,
	}, nil
}

// RegisterDevice registers or updates a sync device.
func (e *Engine) RegisterDevice(ctx context.Context, device *db.Device) error {
	existing, err := e.db.GetDeviceByID(ctx, device.ID)
	if err == nil && existing != nil {
		// Update existing device
		return e.db.UpdateDevice(ctx, device.ID, map[string]any{
			"name":        device.Name,
			"device_type": device.DeviceType,
			"os":          device.OS,
			"sync_folder": device.SyncFolder,
			"is_active":   true,
		})
	}

	// Create new device
	vcJSON, _ := json.Marshal(NewVectorClock())
	device.VectorClock = string(vcJSON)
	device.IsActive = true
	return e.db.CreateDevice(ctx, device)
}

// findNeededChunks returns chunk hashes the server doesn't have yet.
// It looks up the manifests for modified files and checks which chunk
// hashes are missing from storage.
func (e *Engine) findNeededChunks(ctx context.Context, userID string, paths []string) []string {
	var needed []string
	seen := make(map[string]struct{})

	for _, filePath := range paths {
		chunks, err := e.collectChunkHashesForPath(ctx, userID, filePath)
		if err != nil {
			log.Printf("sync: findNeededChunks: failed to resolve %s: %v", filePath, err)
			continue
		}

		for _, hash := range chunks {
			if _, already := seen[hash]; already {
				continue
			}
			seen[hash] = struct{}{}

			exists, err := e.storage.Exists(ctx, hash)
			if err != nil {
				log.Printf("sync: findNeededChunks: failed to check chunk %s: %v", hash, err)
				continue
			}
			if !exists {
				needed = append(needed, hash)
			}
		}
	}

	return needed
}

// findAvailableChunks returns chunk hashes the server can provide to the client.
// It collects all chunk hashes from manifests of added and modified files that
// exist in storage.
func (e *Engine) findAvailableChunks(ctx context.Context, userID string, added, modified []string) []string {
	var available []string
	seen := make(map[string]struct{})

	allPaths := make([]string, 0, len(added)+len(modified))
	allPaths = append(allPaths, added...)
	allPaths = append(allPaths, modified...)

	for _, filePath := range allPaths {
		chunks, err := e.collectChunkHashesForPath(ctx, userID, filePath)
		if err != nil {
			log.Printf("sync: findAvailableChunks: failed to resolve %s: %v", filePath, err)
			continue
		}

		for _, hash := range chunks {
			if _, already := seen[hash]; already {
				continue
			}
			seen[hash] = struct{}{}

			exists, err := e.storage.Exists(ctx, hash)
			if err != nil {
				log.Printf("sync: findAvailableChunks: failed to check chunk %s: %v", hash, err)
				continue
			}
			if exists {
				available = append(available, hash)
			}
		}
	}

	return available
}

// collectChunkHashesForPath resolves a file path to its latest manifest chunk hashes.
func (e *Engine) collectChunkHashesForPath(ctx context.Context, userID, filePath string) ([]string, error) {
	dir := path.Dir(filePath)
	name := path.Base(filePath)

	parentID, err := e.resolveParentID(ctx, userID, dir)
	if err != nil {
		return nil, fmt.Errorf("resolve parent for %s: %w", filePath, err)
	}

	fileRec, err := e.db.GetFileByPath(ctx, userID, parentID, name)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	manifest, err := e.db.GetLatestManifest(ctx, fileRec.ID)
	if err != nil {
		return nil, fmt.Errorf("manifest not found for file %s: %w", filePath, err)
	}

	return manifest.Chunks, nil
}

// detectConflict checks if a change conflicts with server state.
func (e *Engine) detectConflict(ctx context.Context, userID string, change FileChange) *ConflictInfo {
	if change.Action == "add" {
		// No conflict for new files
		return nil
	}

	// For modify/delete, check if there's a concurrent modification
	// by comparing vector clocks
	if change.Action == "modify" || change.Action == "delete" {
		// Look up existing sync state for this file
		// If the server vector clock is concurrent with the client's, it's a conflict
		// For now, simple heuristic: if file was modified since last sync, it's a conflict
	}

	return nil
}

// applyChange applies a single file change to the server.
func (e *Engine) applyChange(ctx context.Context, userID string, change FileChange) error {
	switch change.Action {
	case "add":
		return e.applyAdd(ctx, userID, change)
	case "modify":
		return e.applyModify(ctx, userID, change)
	case "delete":
		return e.applyDelete(ctx, userID, change)
	default:
		return fmt.Errorf("unknown action: %s", change.Action)
	}
}

// applyAdd creates a new file with its manifest and chunk references.
func (e *Engine) applyAdd(ctx context.Context, userID string, change FileChange) error {
	dir, name := path.Dir(change.Path), path.Base(change.Path)

	// Resolve parent directory ID
	parentID, err := e.resolveParentID(ctx, userID, dir)
	if err != nil {
		return fmt.Errorf("resolve parent for %s: %w", change.Path, err)
	}

	// Create file metadata
	now := time.Now().UTC()
	fileRec := &db.File{
		ID:        generateSyncID("file"),
		UserID:    userID,
		ParentID:  parentID,
		Name:      name,
		Type:      "file",
		SizeBytes: change.SizeBytes,
		MimeType:  change.MimeType,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := e.db.CreateFile(ctx, fileRec); err != nil {
		return fmt.Errorf("create file %s: %w", change.Path, err)
	}

	// Create manifest with chunk hashes
	if len(change.ChunkHashes) > 0 {
		manifest := &db.Manifest{
			ID:         generateSyncID("manifest"),
			FileID:     fileRec.ID,
			Version:    1,
			SizeBytes:  change.SizeBytes,
			ChunkCount: len(change.ChunkHashes),
			Chunks:     change.ChunkHashes,
			Checksum:   change.ManifestID,
			DeviceID:   change.VectorClock.firstDeviceID(),
			CreatedAt:  now,
		}

		if err := e.db.CreateManifest(ctx, manifest); err != nil {
			return fmt.Errorf("create manifest for %s: %w", change.Path, err)
		}

		// Link manifest to file and update chunk ref counts
		manifestIDStr := manifest.ID
		if err := e.db.UpdateFile(ctx, fileRec.ID, map[string]any{
			"manifest_id": manifestIDStr,
			"version":     1,
		}); err != nil {
			return fmt.Errorf("link manifest to file %s: %w", change.Path, err)
		}

		if err := e.ensureChunkRefs(ctx, change.ChunkHashes); err != nil {
			return fmt.Errorf("ensure chunk refs for %s: %w", change.Path, err)
		}
	}

	return nil
}

// applyModify updates an existing file with new manifest and chunk references.
func (e *Engine) applyModify(ctx context.Context, userID string, change FileChange) error {
	existing, err := e.resolveFileByPath(ctx, userID, change.Path)
	if err != nil {
		return fmt.Errorf("resolve file %s: %w", change.Path, err)
	}

	newVersion := existing.Version + 1

	// Create new manifest
	manifest := &db.Manifest{
		ID:         generateSyncID("manifest"),
		FileID:     existing.ID,
		Version:    newVersion,
		SizeBytes:  change.SizeBytes,
		ChunkCount: len(change.ChunkHashes),
		Chunks:     change.ChunkHashes,
		Checksum:   change.ManifestID,
		DeviceID:   change.VectorClock.firstDeviceID(),
		CreatedAt:  time.Now().UTC(),
	}

	if err := e.db.CreateManifest(ctx, manifest); err != nil {
		return fmt.Errorf("create manifest for %s: %w", change.Path, err)
	}

	// Update file metadata
	manifestIDStr := manifest.ID
	updates := map[string]any{
		"manifest_id": manifestIDStr,
		"size_bytes":  change.SizeBytes,
		"mime_type":   change.MimeType,
		"version":     newVersion,
	}
	if err := e.db.UpdateFile(ctx, existing.ID, updates); err != nil {
		return fmt.Errorf("update file %s: %w", change.Path, err)
	}

	// Update chunk ref counts for new chunks
	if err := e.ensureChunkRefs(ctx, change.ChunkHashes); err != nil {
		return fmt.Errorf("ensure chunk refs for %s: %w", change.Path, err)
	}

	return nil
}

// applyDelete soft-deletes a file.
func (e *Engine) applyDelete(ctx context.Context, userID string, change FileChange) error {
	existing, err := e.resolveFileByPath(ctx, userID, change.Path)
	if err != nil {
		return fmt.Errorf("resolve file %s: %w", change.Path, err)
	}

	if err := e.db.SoftDelete(ctx, existing.ID); err != nil {
		return fmt.Errorf("soft delete %s: %w", change.Path, err)
	}

	return nil
}

// ensureChunkRefs creates chunk records if missing and increments ref counts.
func (e *Engine) ensureChunkRefs(ctx context.Context, hashes []string) error {
	for _, hash := range hashes {
		exists, err := e.db.ChunkExists(ctx, hash)
		if err != nil {
			return fmt.Errorf("check chunk %s: %w", hash, err)
		}
		if exists {
			if err := e.db.IncrementRefCount(ctx, hash); err != nil {
				return fmt.Errorf("increment ref for chunk %s: %w", hash, err)
			}
		} else {
			// Create new chunk record
			chunk := &db.Chunk{
				Hash:           hash,
				SizeBytes:      0, // Size will be updated when chunk data is pushed
				StorageBackend: e.storage.Type(),
				StoragePath:    hash,
				RefCount:       1,
				IsEncrypted:    false,
				CreatedAt:      time.Now().UTC(),
			}
			if err := e.db.CreateChunk(ctx, chunk); err != nil {
				return fmt.Errorf("create chunk %s: %w", hash, err)
			}
		}
	}
	return nil
}

// resolveParentID walks the directory path and returns the parent folder ID.
// Returns nil for root-level files.
func (e *Engine) resolveParentID(ctx context.Context, userID, dirPath string) (*string, error) {
	dirPath = strings.TrimPrefix(dirPath, "/")
	if dirPath == "" || dirPath == "." {
		return nil, nil
	}

	segments := strings.Split(dirPath, "/")
	var parentID *string

	for _, seg := range segments {
		if seg == "" {
			continue
		}
		folder, err := e.db.GetFileByPath(ctx, userID, parentID, seg)
		if err != nil {
			return nil, fmt.Errorf("folder %q not found: %w", seg, err)
		}
		id := folder.ID
		parentID = &id
	}

	return parentID, nil
}

// resolveFileByPath walks the path to find the file record.
func (e *Engine) resolveFileByPath(ctx context.Context, userID, filePath string) (*db.File, error) {
	dir := path.Dir(filePath)
	name := path.Base(filePath)

	parentID, err := e.resolveParentID(ctx, userID, dir)
	if err != nil {
		return nil, err
	}

	file, err := e.db.GetFileByPath(ctx, userID, parentID, name)
	if err != nil {
		return nil, fmt.Errorf("file not found at %s: %w", filePath, err)
	}
	return file, nil
}

// generateSyncID creates a unique ID with the given prefix using crypto/rand.
func generateSyncID(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UTC().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b)
}

// firstDeviceID returns the first device ID in the vector clock, or empty string.
func (v VectorClock) firstDeviceID() string {
	for id := range v {
		return id
	}
	return ""
}

// updateDeviceState updates the device's vector clock and merkle root after commit.
func (e *Engine) updateDeviceState(ctx context.Context, deviceID string, vc VectorClock) {
	vcJSON, err := json.Marshal(vc)
	if err != nil {
		log.Printf("sync: failed to marshal vector clock: %v", err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_ = e.db.UpdateDevice(ctx, deviceID, map[string]any{
		"vector_clock": string(vcJSON),
		"last_sync_at": now,
	})
}

// collectAllPaths returns all file paths in a merkle tree.
func collectAllPaths(node *MerkleNode) []string {
	if node == nil {
		return []string{}
	}
	var paths []string
	collectPathsRecursive(node, &paths)
	return paths
}

func collectPathsRecursive(node *MerkleNode, paths *[]string) {
	if node == nil {
		return
	}
	if node.IsFile {
		*paths = append(*paths, node.Path)
	}
	for _, child := range node.Children {
		collectPathsRecursive(child, paths)
	}
}
