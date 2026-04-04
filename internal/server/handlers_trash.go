package server

import (
	"net/http"
	"strconv"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// TrashHandler handles trash/recycle bin API requests.
type TrashHandler struct {
	vfs *vfs.VFS
	db  *db.Manager
}

// NewTrashHandler creates a new trash handler.
func NewTrashHandler(vfsService *vfs.VFS, database *db.Manager) *TrashHandler {
	return &TrashHandler{vfs: vfsService, db: database}
}

// RegisterRoutes registers the trash routes.
func (h *TrashHandler) RegisterRoutes(mux *http.ServeMux, auth *AuthMiddleware) {
	// List trashed items
	mux.Handle("GET /api/v1/trash", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.listTrash))))

	// Restore item from trash
	mux.Handle("POST /api/v1/trash/{id}/restore", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.restoreItem))))

	// Permanently delete item
	mux.Handle("DELETE /api/v1/trash/{id}", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.permanentDelete))))

	// Empty trash
	mux.Handle("DELETE /api/v1/trash", auth.Authenticate(auth.RequireAuth(http.HandlerFunc(h.emptyTrash))))
}

// listTrash lists all items in the user's trash.
func (h *TrashHandler) listTrash(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Parse pagination
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

	items, err := h.vfs.ListTrash(r.Context(), userID, limit, offset)
	if err != nil {
		InternalErrorResponse(w, err)
		return
	}

	total, err := h.vfs.CountTrash(r.Context(), userID)
	if err != nil {
		InternalErrorResponse(w, err)
		return
	}

	SuccessResponse(w, map[string]any{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// restoreItem restores an item from trash to its original location.
func (h *TrashHandler) restoreItem(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	itemID := r.PathValue("id")
	if itemID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Item ID required")
		return
	}

	// Verify ownership
	item, err := h.vfs.GetFile(r.Context(), itemID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "Item not found")
			return
		}
		InternalErrorResponse(w, err)
		return
	}

	if item.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := h.vfs.Restore(r.Context(), itemID); err != nil {
		InternalErrorResponse(w, err)
		return
	}

	SuccessResponse(w, map[string]string{"status": "restored"})
}

// permanentDelete permanently deletes an item from trash.
func (h *TrashHandler) permanentDelete(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	itemID := r.PathValue("id")
	if itemID == "" {
		ErrorResponse(w, http.StatusBadRequest, "Item ID required")
		return
	}

	// Verify ownership
	item, err := h.vfs.GetFile(r.Context(), itemID)
	if err != nil {
		if err == vfs.ErrNotFound {
			ErrorResponse(w, http.StatusNotFound, "Item not found")
			return
		}
		InternalErrorResponse(w, err)
		return
	}

	if item.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := h.vfs.DeletePermanent(r.Context(), itemID); err != nil {
		InternalErrorResponse(w, err)
		return
	}

	SuccessResponse(w, map[string]string{"status": "deleted"})
}

// emptyTrash permanently deletes all items in the user's trash using batched deletion.
func (h *TrashHandler) emptyTrash(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	deletedCount := 0
	failedCount := 0
	const batchSize = 100
	offset := 0

	for {
		items, err := h.vfs.ListTrash(r.Context(), userID, batchSize, offset)
		if err != nil {
			InternalErrorResponse(w, err)
			return
		}
		if len(items) == 0 {
			break
		}

		for _, item := range items {
			if err := h.vfs.DeletePermanent(r.Context(), item.ID); err != nil {
				failedCount++
			} else {
				deletedCount++
			}
		}

		// If we got fewer than batchSize items, we've reached the end.
		if len(items) < batchSize {
			break
		}
		offset += batchSize
	}

	SuccessResponse(w, map[string]any{
		"status":  "emptied",
		"deleted": deletedCount,
		"failed":  failedCount,
	})
}
