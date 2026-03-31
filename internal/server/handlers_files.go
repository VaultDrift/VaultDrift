package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// FileHandler handles file API requests.
type FileHandler struct {
	vfs    *vfs.VFS
	db     *db.Manager
	events *EventNotifier
}

// NewFileHandler creates a new file handler.
func NewFileHandler(vfsService *vfs.VFS, database *db.Manager, events *EventNotifier) *FileHandler {
	return &FileHandler{vfs: vfsService, db: database, events: events}
}

// RegisterRoutes registers the file routes.
func (h *FileHandler) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// List files in a folder
	mux.Handle("GET /api/v1/files", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.listFiles))))

	// Get file details
	mux.Handle("GET /api/v1/files/{id}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.getFile))))

	// Create new file entry (metadata only)
	mux.Handle("POST /api/v1/files", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.createFile))))

	// Update file (rename/move)
	mux.Handle("PUT /api/v1/files/{id}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.updateFile))))

	// Delete file (move to trash)
	mux.Handle("DELETE /api/v1/files/{id}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.deleteFile))))

	// Search files
	mux.Handle("GET /api/v1/files/search", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.searchFiles))))

	// Get recent files
	mux.Handle("GET /api/v1/files/recent", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.recentFiles))))
}

// listFiles lists files in a folder.
func (h *FileHandler) listFiles(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get parent folder ID from query
	parentID := r.URL.Query().Get("parent_id")

	// Pagination
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	opts := db.ListOpts{
		Limit:  limit,
		Offset: offset,
	}

	files, err := h.vfs.ListDirectory(r.Context(), userID, parentID, opts)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(w, map[string]interface{}{
		"files":  files,
		"limit":  limit,
		"offset": offset,
	})
}

// getFile returns file details.
func (h *FileHandler) getFile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("id")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Verify ownership
	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	SuccessResponse(w, file)
}

// createFileRequest represents a create file request.
type createFileRequest struct {
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

// createFile creates a new file entry.
func (h *FileHandler) createFile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req createFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		ErrorResponse(w, http.StatusBadRequest, "Name is required")
		return
	}

	file, err := h.vfs.CreateFile(r.Context(), userID, req.ParentID, req.Name, req.MimeType, req.Size)
	if err != nil {
		if err == vfs.ErrAlreadyExists {
			ErrorResponse(w, http.StatusConflict, "File already exists")
			return
		}
		if err == vfs.ErrInvalidName {
			ErrorResponse(w, http.StatusBadRequest, "Invalid file name")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	SuccessResponse(w, file)

	// Emit event for real-time sync
	if h.events != nil {
		parentID := req.ParentID
		if parentID == "" {
			parentID = "root"
		}
		h.events.NotifyFileChange(userID, parentID, file.ID, SSEventFileCreated, map[string]any{
			"file": file,
		})
	}
}

// updateFileRequest represents an update file request.
type updateFileRequest struct {
	Name       string `json:"name,omitempty"`
	ParentID   string `json:"parent_id,omitempty"`
}

// updateFile updates a file (rename or move).
func (h *FileHandler) updateFile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("id")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	// Verify ownership
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	var req updateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Handle move
	if req.ParentID != "" {
		if err := h.vfs.Move(r.Context(), fileID, req.ParentID, req.Name); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if req.Name != "" {
		// Just rename
		if err := h.vfs.Rename(r.Context(), fileID, req.Name); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	SuccessResponse(w, map[string]string{"status": "updated"})

	// Emit event for real-time sync
	if h.events != nil {
		eventType := SSEventFileUpdated
		if req.ParentID != "" && req.ParentID != *file.ParentID {
			eventType = SSEventFileMoved
		}
		h.events.NotifyFileChange(userID, *file.ParentID, fileID, eventType, map[string]any{
			"file_id":   fileID,
			"new_name":  req.Name,
			"new_parent": req.ParentID,
		})
	}
}

// deleteFile moves a file to trash.
func (h *FileHandler) deleteFile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileID := r.PathValue("id")
	if fileID == "" {
		ErrorResponse(w, http.StatusBadRequest, "File ID required")
		return
	}

	// Verify ownership
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := h.vfs.Delete(r.Context(), fileID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(w, map[string]string{"status": "deleted"})

	// Emit event for real-time sync
	if h.events != nil {
		h.events.NotifyFileChange(userID, *file.ParentID, fileID, SSEventFileDeleted, map[string]any{
			"file_id": fileID,
			"name":    file.Name,
		})
	}
}

// searchFiles searches for files.
func (h *FileHandler) searchFiles(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		ErrorResponse(w, http.StatusBadRequest, "Query required")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	files, err := h.vfs.Search(r.Context(), userID, query, limit)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(w, map[string]interface{}{
		"files": files,
		"query": query,
	})
}

// recentFiles returns recently modified files.
func (h *FileHandler) recentFiles(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	files, err := h.vfs.Recent(r.Context(), userID, limit)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(w, map[string]interface{}{
		"files": files,
	})
}
