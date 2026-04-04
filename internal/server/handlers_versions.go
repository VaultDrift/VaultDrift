package server

import (
	"net/http"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// VersionHandler handles file versioning API requests.
type VersionHandler struct {
	vfs *vfs.VFS
	db  *db.Manager
}

// NewVersionHandler creates a new version handler.
func NewVersionHandler(vfsService *vfs.VFS, database *db.Manager) *VersionHandler {
	return &VersionHandler{vfs: vfsService, db: database}
}

// RegisterRoutes registers the version routes.
func (h *VersionHandler) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// Get current version
	mux.Handle("GET /api/v1/files/{id}/version", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.getVersion))))

	// Increment version (manual version bump)
	mux.Handle("POST /api/v1/files/{id}/version", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.incrementVersion))))
}

// getVersion returns the current version of a file.
func (h *VersionHandler) getVersion(w http.ResponseWriter, r *http.Request) {
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

	// Verify file ownership
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		InternalErrorResponse(w, err)
		return
	}

	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	SuccessResponse(w, map[string]any{
		"file_id": file.ID,
		"version": file.Version,
		"name":    file.Name,
	})
}

// incrementVersion manually increments the version number.
func (h *VersionHandler) incrementVersion(w http.ResponseWriter, r *http.Request) {
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

	// Verify file ownership
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "File not found")
			return
		}
		InternalErrorResponse(w, err)
		return
	}

	if file.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	// Parse request body for optional comment
	var req struct {
		Comment string `json:"comment,omitempty"`
	}
	if err := DecodeJSON(r, &req); err != nil {
			// Optional body, ignore parse errors
	}

	_ = r.Body.Close()

	// Create versioning service and increment version
	vs := vfs.NewVersioningService(h.vfs, h.db)
	newVersion, err := vs.IncrementVersion(r.Context(), fileID)
	if err != nil {
		InternalErrorResponse(w, err)
		return
	}

	SuccessResponse(w, map[string]any{
		"file_id":          fileID,
		"version":          newVersion,
		"previous_version": file.Version,
		"comment":          req.Comment,
	})
}
