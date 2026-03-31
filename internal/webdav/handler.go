package webdav

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// WebDAV HTTP methods
const (
	MethodPropFind  = "PROPFIND"
	MethodPropPatch = "PROPPATCH"
	MethodMkCol     = "MKCOL"
	MethodCopy      = "COPY"
	MethodMove      = "MOVE"
	MethodLock      = "LOCK"
	MethodUnlock    = "UNLOCK"
)

// Handler implements a WebDAV Class 2 compliant server
type Handler struct {
	vfs       *vfs.VFS
	db        *db.Manager
	lockStore *LockStore
	basePath  string
}

// NewHandler creates a new WebDAV handler
func NewHandler(vfsService *vfs.VFS, database *db.Manager, basePath string) *Handler {
	return &Handler{
		vfs:       vfsService,
		db:        database,
		lockStore: NewLockStore(),
		basePath:  basePath,
	}
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip base path
	webdavPath := strings.TrimPrefix(r.URL.Path, h.basePath)
	if webdavPath == "" {
		webdavPath = "/"
	}

	// Set DAV headers
	w.Header().Set("DAV", "1, 2") // Class 1 and 2 compliance
	w.Header().Set("MS-Author-Via", "DAV")

	switch r.Method {
	case http.MethodOptions:
		h.handleOptions(w, r)
	case http.MethodGet, http.MethodHead:
		h.handleGet(w, r, webdavPath)
	case http.MethodPut:
		h.handlePut(w, r, webdavPath)
	case http.MethodDelete:
		h.handleDelete(w, r, webdavPath)
	case MethodPropFind:
		h.handlePropFind(w, r, webdavPath)
	case MethodPropPatch:
		h.handlePropPatch(w, r, webdavPath)
	case MethodMkCol:
		h.handleMkCol(w, r, webdavPath)
	case MethodCopy:
		h.handleCopy(w, r, webdavPath)
	case MethodMove:
		h.handleMove(w, r, webdavPath)
	case MethodLock:
		h.handleLock(w, r, webdavPath)
	case MethodUnlock:
		h.handleUnlock(w, r, webdavPath)
	default:
		h.sendError(w, http.StatusNotImplemented, fmt.Sprintf("Method %s not implemented", r.Method))
	}
}

// handleOptions handles OPTIONS requests
func (h *Handler) handleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, COPY, MOVE, LOCK, UNLOCK")
	w.Header().Set("DAV", "1, 2")
	w.WriteHeader(http.StatusOK)
}

// handleGet handles GET requests for file download
func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Parse path to get file
	file, err := h.getFileByPath(r.Context(), userID, webdavPath)
	if err != nil {
		if err == vfs.ErrNotFound {
			h.sendError(w, http.StatusNotFound, "Resource not found")
			return
		}
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if file.Type == "folder" {
		h.sendError(w, http.StatusMethodNotAllowed, "Cannot GET a folder")
		return
	}

	// Set headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	w.Header().Set("Last-Modified", file.UpdatedAt.Format(http.TimeFormat))
	if file.Checksum != nil {
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, *file.Checksum))
	}

	// Handle conditional requests
	if match := r.Header.Get("If-None-Match"); match != "" && file.Checksum != nil {
		if match == fmt.Sprintf(`"%s"`, *file.Checksum) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// For now, return placeholder - actual implementation would stream chunks
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "[File content would be streamed here]")
}

// handlePut handles PUT requests for file upload
func (h *Handler) handlePut(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get parent directory path
	dirPath := path.Dir(webdavPath)
	fileName := path.Base(webdavPath)

	// Get or create parent folder
	var parentID string
	if dirPath != "/" && dirPath != "." {
		parent, err := h.getFileByPath(r.Context(), userID, dirPath)
		if err != nil {
			if err == vfs.ErrNotFound {
				h.sendError(w, http.StatusConflict, "Parent directory does not exist")
				return
			}
			h.sendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		parentID = parent.ID
	}

	// Check if file already exists
	existing, _ := h.getFileByPath(r.Context(), userID, webdavPath)
	isNew := existing == nil

	// Read upload content
	data, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to read upload")
		return
	}

	// Calculate checksum
	checksum := sha256.Sum256(data)
	checksumStr := fmt.Sprintf("%x", checksum)

	if isNew {
		// Create new file using VFS
		_, err := h.vfs.CreateFile(r.Context(), userID, parentID, fileName, detectContentType(fileName, data), int64(len(data)))
		if err != nil {
			h.sendError(w, http.StatusInternalServerError, "Failed to create file")
			return
		}
	} else {
		// Update existing file
		updates := map[string]any{
			"size_bytes": int64(len(data)),
			"checksum":   checksumStr,
			"mime_type":  detectContentType(fileName, data),
		}
		if err := h.db.UpdateFile(r.Context(), existing.ID, updates); err != nil {
			h.sendError(w, http.StatusInternalServerError, "Failed to update file")
			return
		}
	}

	if isNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleDelete handles DELETE requests
func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Resolve path
	file, err := h.getFileByPath(r.Context(), userID, webdavPath)
	if err != nil {
		if err == vfs.ErrNotFound {
			h.sendError(w, http.StatusNotFound, "Resource not found")
			return
		}
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check locks
	if h.lockStore.IsLocked(file.ID) {
		h.sendError(w, http.StatusLocked, "Resource is locked")
		return
	}

	// Delete (soft delete to trash)
	if err := h.vfs.Delete(r.Context(), file.ID); err != nil {
		h.sendError(w, http.StatusInternalServerError, "Failed to delete")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handlePropFind handles PROPFIND requests
func (h *Handler) handlePropFind(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Parse depth header
	depth := r.Header.Get("Depth")
	if depth == "" {
		depth = "infinity"
	}

	// Get resource
	resource, err := h.getFileByPath(r.Context(), userID, webdavPath)
	if err != nil {
		if err == vfs.ErrNotFound {
			// Return 207 with empty multistatus for non-existent resource
			h.sendMultiStatus(w, []Response{})
			return
		}
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build response
	responses := []Response{h.buildPropResponse(resource, r.URL.Path)}

	// If depth > 0 and resource is a folder, add children
	if resource.Type == "folder" && depth != "0" {
		opts := db.ListOpts{Limit: 1000}
		children, err := h.vfs.ListDirectory(r.Context(), userID, resource.ID, opts)
		if err == nil {
			for _, child := range children {
				childPath := path.Join(r.URL.Path, child.Name)
				responses = append(responses, h.buildPropResponse(child, childPath))
			}
		}
	}

	h.sendMultiStatus(w, responses)
}

// handlePropPatch handles PROPPATCH requests
func (h *Handler) handlePropPatch(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get resource
	resource, err := h.getFileByPath(r.Context(), userID, webdavPath)
	if err != nil {
		if err == vfs.ErrNotFound {
			h.sendError(w, http.StatusNotFound, "Resource not found")
			return
		}
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build response (we don't actually store custom properties)
	responses := []Response{h.buildPropResponse(resource, r.URL.Path)}
	h.sendMultiStatus(w, responses)
}

// handleMkCol handles MKCOL requests (create directory)
func (h *Handler) handleMkCol(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Check if already exists
	existing, _ := h.getFileByPath(r.Context(), userID, webdavPath)
	if existing != nil {
		h.sendError(w, http.StatusMethodNotAllowed, "Resource already exists")
		return
	}

	// Get parent path
	dirPath := path.Dir(webdavPath)
	folderName := path.Base(webdavPath)

	// Get parent folder
	var parentID string
	if dirPath != "/" && dirPath != "." {
		parent, err := h.getFileByPath(r.Context(), userID, dirPath)
		if err != nil {
			h.sendError(w, http.StatusConflict, "Parent directory does not exist")
			return
		}
		parentID = parent.ID
	}

	// Create folder using VFS
	if _, err := h.vfs.CreateFolder(r.Context(), userID, parentID, folderName); err != nil {
		if err == vfs.ErrAlreadyExists {
			h.sendError(w, http.StatusMethodNotAllowed, "Resource already exists")
			return
		}
		h.sendError(w, http.StatusInternalServerError, "Failed to create folder")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// handleCopy handles COPY requests
func (h *Handler) handleCopy(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	destination := r.Header.Get("Destination")
	if destination == "" {
		h.sendError(w, http.StatusBadRequest, "Destination header required")
		return
	}

	// Parse destination URL
	destPath := h.parseDestination(destination)
	if destPath == "" {
		h.sendError(w, http.StatusBadRequest, "Invalid destination")
		return
	}

	// Get source resource
	source, err := h.getFileByPath(r.Context(), userID, webdavPath)
	if err != nil {
		if err == vfs.ErrNotFound {
			h.sendError(w, http.StatusNotFound, "Source not found")
			return
		}
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get destination parent
	dirPath := path.Dir(destPath)
	destName := path.Base(destPath)

	var destParentID string
	if dirPath != "/" && dirPath != "." {
		parent, err := h.getFileByPath(r.Context(), userID, dirPath)
		if err != nil {
			h.sendError(w, http.StatusConflict, "Destination parent does not exist")
			return
		}
		destParentID = parent.ID
	}

	// VFS doesn't have Copy method yet - return not implemented
	_ = source
	_ = destParentID
	_ = destName
	h.sendError(w, http.StatusNotImplemented, "Copy not yet implemented")
}

// handleMove handles MOVE requests
func (h *Handler) handleMove(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	destination := r.Header.Get("Destination")
	if destination == "" {
		h.sendError(w, http.StatusBadRequest, "Destination header required")
		return
	}

	// Parse destination URL
	destPath := h.parseDestination(destination)
	if destPath == "" {
		h.sendError(w, http.StatusBadRequest, "Invalid destination")
		return
	}

	// Get source resource
	source, err := h.getFileByPath(r.Context(), userID, webdavPath)
	if err != nil {
		if err == vfs.ErrNotFound {
			h.sendError(w, http.StatusNotFound, "Source not found")
			return
		}
		h.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check locks
	if h.lockStore.IsLocked(source.ID) {
		h.sendError(w, http.StatusLocked, "Resource is locked")
		return
	}

	// Get destination path
	dirPath := path.Dir(destPath)
	destName := path.Base(destPath)

	var destParentID string
	if dirPath != "/" && dirPath != "." {
		parent, err := h.getFileByPath(r.Context(), userID, dirPath)
		if err != nil {
			h.sendError(w, http.StatusConflict, "Destination parent does not exist")
			return
		}
		destParentID = parent.ID
	}

	// Move using VFS
	if err := h.vfs.Move(r.Context(), source.ID, destParentID, destName); err != nil {
		h.sendError(w, http.StatusInternalServerError, "Move failed")
		return
	}

	overwrite := r.Header.Get("Overwrite") != "F"
	if overwrite {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
}

// handleLock handles LOCK requests (Class 2 WebDAV)
func (h *Handler) handleLock(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Get or create lock token
	lockToken := r.Header.Get("If")
	if lockToken == "" {
		// Create new lock
		lockToken = generateLockToken()
	}

	// Parse lock info if present
	var lockInfo LockInfo
	if r.ContentLength > 0 {
		xml.NewDecoder(r.Body).Decode(&lockInfo)
	}

	// Set default timeout
	timeout := 30 * time.Minute
	if lockInfo.Timeout != "" {
		if t, err := parseTimeout(lockInfo.Timeout); err == nil {
			timeout = t
		}
	}

	// Get resource
	resource, err := h.getFileByPath(r.Context(), userID, webdavPath)
	if err != nil {
		if err == vfs.ErrNotFound {
			// Lock on non-existent resource (lock-null resource)
			h.lockStore.Lock(webdavPath, lockToken, userID, timeout, true)
		} else {
			h.sendError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		h.lockStore.Lock(resource.ID, lockToken, userID, timeout, false)
	}

	// Build lock discovery response
	lockDiscovery := h.buildLockDiscovery(lockToken, timeout)

	w.Header().Set("Lock-Token", fmt.Sprintf("<%s>", lockToken))
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	xml.NewEncoder(w).Encode(lockDiscovery)
}

// handleUnlock handles UNLOCK requests
func (h *Handler) handleUnlock(w http.ResponseWriter, r *http.Request, webdavPath string) {
	userID := h.getUserID(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	lockToken := r.Header.Get("Lock-Token")
	if lockToken == "" {
		h.sendError(w, http.StatusBadRequest, "Lock-Token header required")
		return
	}

	// Remove angle brackets if present
	lockToken = strings.Trim(lockToken, "<>")

	if !h.lockStore.Unlock(lockToken, userID) {
		h.sendError(w, http.StatusForbidden, "Unlock failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (h *Handler) getUserID(r *http.Request) string {
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return ""
}

func (h *Handler) getFileByPath(ctx context.Context, userID, webdavPath string) (*db.File, error) {
	if webdavPath == "/" || webdavPath == "" {
		// Root directory - return a synthetic folder
		return &db.File{
			ID:        "",
			UserID:    userID,
			Name:      "root",
			Type:      "folder",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}

	// Split path into parts
	parts := strings.Split(strings.Trim(webdavPath, "/"), "/")
	if len(parts) == 0 {
		return nil, vfs.ErrNotFound
	}

	// Walk the path
	var parentID string
	for i, part := range parts {
		isLast := i == len(parts)-1

		file, err := h.vfs.GetFileByPath(ctx, userID, parentID, part)
		if err != nil {
			return nil, err
		}

		if isLast {
			return file, nil
		}

		if file.Type != "folder" {
			return nil, vfs.ErrNotFound
		}
		parentID = file.ID
	}

	return nil, vfs.ErrNotFound
}

func (h *Handler) parseDestination(dest string) string {
	// Simple parsing - extract path after host
	parts := strings.SplitN(dest, "://", 2)
	if len(parts) != 2 {
		return ""
	}
	pathParts := strings.SplitN(parts[1], "/", 2)
	if len(pathParts) != 2 {
		return ""
	}
	return "/" + pathParts[1]
}

func (h *Handler) sendError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(message))
}

func (h *Handler) sendMultiStatus(w http.ResponseWriter, responses []Response) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)

	multistatus := MultiStatus{
		XMLNS:     "DAV:",
		Responses: responses,
	}

	w.Write([]byte(xml.Header))
	xml.NewEncoder(w).Encode(multistatus)
}

func (h *Handler) buildPropResponse(resource *db.File, href string) Response {
	props := []Property{
		{Name: "displayname", Value: resource.Name},
		{Name: "getlastmodified", Value: resource.UpdatedAt.Format(http.TimeFormat)},
		{Name: "creationdate", Value: resource.CreatedAt.Format(time.RFC3339)},
	}

	if resource.Checksum != nil {
		props = append(props, Property{Name: "getetag", Value: fmt.Sprintf(`"%s"`, *resource.Checksum)})
	}

	if resource.Type == "file" {
		props = append(props,
			Property{Name: "getcontentlength", Value: strconv.FormatInt(resource.SizeBytes, 10)},
			Property{Name: "getcontenttype", Value: resource.MimeType},
			Property{Name: "resourcetype", IsResourceType: false},
		)
	} else {
		props = append(props,
			Property{Name: "resourcetype", IsResourceType: true},
		)
	}

	// Add lock discovery if locked
	if h.lockStore.IsLocked(resource.ID) {
		lockToken, _ := h.lockStore.GetLockToken(resource.ID)
		props = append(props, Property{
			Name:  "lockdiscovery",
			Value: lockToken,
		})
	}

	return Response{
		Href:     href,
		Property: props,
	}
}

func (h *Handler) buildLockDiscovery(token string, timeout time.Duration) *LockDiscovery {
	return &LockDiscovery{
		ActiveLock: ActiveLock{
			LockType:  "write",
			LockScope: "exclusive",
			Depth:     "0",
			Timeout:   fmt.Sprintf("Second-%d", int(timeout.Seconds())),
			LockToken: LockToken{Href: token},
		},
	}
}

// Utility functions

func detectContentType(filename string, data []byte) string {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".pdf":
		return "application/pdf"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func generateLockToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based if crypto/rand fails
		for i := range b {
			b[i] = byte(time.Now().UnixNano() % 256)
		}
	}
	return "opaquelocktoken:" + base64.URLEncoding.EncodeToString(b)
}

func parseTimeout(s string) (time.Duration, error) {
	if strings.HasPrefix(s, "Second-") {
		seconds, err := strconv.Atoi(s[7:])
		if err != nil {
			return 0, err
		}
		return time.Duration(seconds) * time.Second, nil
	}
	if s == "Infinite" {
		return 30 * time.Minute, nil
	}
	return 30 * time.Minute, nil
}

// LockStore manages WebDAV locks
type LockStore struct {
	locks map[string]*Lock // resourceID -> Lock
	mu    sync.RWMutex
}

type Lock struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	IsNull    bool
}

func NewLockStore() *LockStore {
	store := &LockStore{
		locks: make(map[string]*Lock),
	}
	go store.cleanupLoop()
	return store
}

func (s *LockStore) Lock(resourceID, token, userID string, duration time.Duration, isNull bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locks[resourceID] = &Lock{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(duration),
		IsNull:    isNull,
	}
}

func (s *LockStore) Unlock(token, userID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for resourceID, lock := range s.locks {
		if lock.Token == token && lock.UserID == userID {
			delete(s.locks, resourceID)
			return true
		}
	}
	return false
}

func (s *LockStore) IsLocked(resourceID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lock, ok := s.locks[resourceID]
	if !ok {
		return false
	}
	return time.Now().Before(lock.ExpiresAt)
}

func (s *LockStore) GetLockToken(resourceID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	lock, ok := s.locks[resourceID]
	if !ok {
		return "", false
	}
	return lock.Token, time.Now().Before(lock.ExpiresAt)
}

func (s *LockStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for resourceID, lock := range s.locks {
			if now.After(lock.ExpiresAt) {
				delete(s.locks, resourceID)
			}
		}
		s.mu.Unlock()
	}
}
