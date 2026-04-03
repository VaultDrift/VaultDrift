package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// FolderHandler handles folder API requests.
type FolderHandler struct {
	vfs    *vfs.VFS
	events *EventNotifier
}

// NewFolderHandler creates a new folder handler.
func NewFolderHandler(vfsService *vfs.VFS, events *EventNotifier) *FolderHandler {
	return &FolderHandler{vfs: vfsService, events: events}
}

// RegisterRoutes registers the folder routes.
func (h *FolderHandler) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// List folder contents
	mux.Handle("GET /api/v1/folders", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.listFolders))))

	// Get folder details
	mux.Handle("GET /api/v1/folders/{id}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.getFolder))))

	// Create new folder
	mux.Handle("POST /api/v1/folders", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.createFolder))))

	// Update folder (rename/move)
	mux.Handle("PUT /api/v1/folders/{id}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.updateFolder))))

	// Delete folder
	mux.Handle("DELETE /api/v1/folders/{id}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.deleteFolder))))

	// Get breadcrumbs for a folder
	mux.Handle("GET /api/v1/folders/{id}/breadcrumbs", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.getBreadcrumbs))))
}

// listFolders lists folders in a parent folder.
func (h *FolderHandler) listFolders(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Get parent folder ID from query
	parentID := r.URL.Query().Get("parent_id")

	// Pagination
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 100 {
			limit = 50
		}
	}
	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}
	}

	opts := db.ListOpts{
		Limit:  limit,
		Offset: offset,
	}

	files, err := h.vfs.ListDirectory(r.Context(), userID, parentID, opts)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Filter for folders only
	var folders []interface{}
	for _, f := range files {
		if f.Type == "folder" {
			folders = append(folders, f)
		}
	}

	SuccessResponse(w, map[string]interface{}{
		"folders": folders,
		"limit":   limit,
		"offset":  offset,
	})
}

// getFolder returns folder details.
func (h *FolderHandler) getFolder(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	folderID := r.PathValue("id")
	if folderID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Folder ID required")
		return
	}

	folder, err := h.vfs.GetFile(r.Context(), folderID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "Folder not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Verify ownership
	if folder.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Verify it's a folder
	if folder.Type != "folder" {
		ErrorResponse(w, http.StatusBadRequest, "Not a folder")
		return
	}

	SuccessResponse(w, folder)
}

// createFolderRequest represents a create folder request.
type createFolderRequest struct {
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
}

// createFolder creates a new folder.
func (h *FolderHandler) createFolder(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req createFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		ErrorResponse(w, http.StatusBadRequest, "Name is required")
		return
	}

	folder, err := h.vfs.CreateFolder(r.Context(), userID, req.ParentID, req.Name)
	if err != nil {
		if err == vfs.ErrAlreadyExists {
			ErrorResponse(w, http.StatusConflict, "Folder already exists")
			return
		}
		if err == vfs.ErrInvalidName {
			ErrorResponse(w, http.StatusBadRequest, "Invalid folder name")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	SuccessResponse(w, folder)

	// Emit event for real-time sync
	if h.events != nil {
		parentID := req.ParentID
		if parentID == "" {
			parentID = "root"
		}
		h.events.Notify(&SSEEvent{
			Type:     SSEventFolderCreated,
			UserID:   userID,
			FolderID: parentID,
			FileID:   folder.ID,
			Data: map[string]any{
				"folder": folder,
			},
		})
	}
}

// updateFolderRequest represents an update folder request.
type updateFolderRequest struct {
	Name     string `json:"name,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
}

// updateFolder updates a folder (rename or move).
func (h *FolderHandler) updateFolder(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	folderID := r.PathValue("id")
	if folderID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Folder ID required")
		return
	}

	// Verify ownership
	folder, err := h.vfs.GetFile(r.Context(), folderID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "Folder not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if folder.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if folder.Type != "folder" {
		ErrorResponse(w, http.StatusBadRequest, "Not a folder")
		return
	}

	var req updateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Handle move
	if req.ParentID != "" {
		if err := h.vfs.Move(r.Context(), folderID, req.ParentID, req.Name); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if req.Name != "" {
		// Just rename
		if err := h.vfs.Rename(r.Context(), folderID, req.Name); err != nil {
			ErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	SuccessResponse(w, map[string]string{"status": "updated"})

	// Emit event for real-time sync
	if h.events != nil {
		eventType := SSEventFolderUpdated
		if req.ParentID != "" && folder.ParentID != nil && req.ParentID != *folder.ParentID {
			eventType = SSEventFileMoved
		}
		h.events.Notify(&SSEEvent{
			Type:     eventType,
			UserID:   userID,
			FolderID: folderID,
			Data: map[string]any{
				"folder_id":  folderID,
				"new_name":   req.Name,
				"new_parent": req.ParentID,
			},
		})
	}
}

// deleteFolder moves a folder to trash.
func (h *FolderHandler) deleteFolder(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	folderID := r.PathValue("id")
	if folderID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Folder ID required")
		return
	}

	// Verify ownership
	folder, err := h.vfs.GetFile(r.Context(), folderID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "Folder not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if folder.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if folder.Type != "folder" {
		ErrorResponse(w, http.StatusBadRequest, "Not a folder")
		return
	}

	if err := h.vfs.Delete(r.Context(), folderID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(w, map[string]string{"status": "deleted"})

	// Emit event for real-time sync
	if h.events != nil {
		h.events.Notify(&SSEEvent{
			Type:     SSEventFolderDeleted,
			UserID:   userID,
			FolderID: folderID,
			Data: map[string]any{
				"folder_id": folderID,
				"name":      folder.Name,
			},
		})
	}
}

// getBreadcrumbs returns breadcrumb trail for a folder.
func (h *FolderHandler) getBreadcrumbs(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	folderID := r.PathValue("id")
	if folderID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Folder ID required")
		return
	}

	// Verify ownership
	folder, err := h.vfs.GetFile(r.Context(), folderID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "Folder not found")
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if folder.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if folder.Type != "folder" {
		ErrorResponse(w, http.StatusBadRequest, "Not a folder")
		return
	}

	breadcrumbs, err := h.vfs.GetBreadcrumbs(r.Context(), folderID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(w, map[string]interface{}{
		"breadcrumbs": breadcrumbs,
	})
}
