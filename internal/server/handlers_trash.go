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
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	items, err := h.vfs.ListTrash(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Apply pagination manually (in production, do this in DB query)
	start := offset
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	paginatedItems := items[start:end]

	SuccessResponse(w, map[string]any{
		"items":  paginatedItems,
		"total":  len(items),
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
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if item.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := h.vfs.Restore(r.Context(), itemID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
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
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	if item.UserID != userID {
		ErrorResponse(w, http.StatusForbidden, "Access denied")
		return
	}

	if err := h.vfs.DeletePermanent(r.Context(), itemID); err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	SuccessResponse(w, map[string]string{"status": "deleted"})
}

// emptyTrash permanently deletes all items in the user's trash.
func (h *TrashHandler) emptyTrash(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromRequest(r)
	if userID == "" {
		ErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	items, err := h.vfs.ListTrash(r.Context(), userID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	deletedCount := 0
	failedCount := 0

	for _, item := range items {
		if err := h.vfs.DeletePermanent(r.Context(), item.ID); err != nil {
			failedCount++
		} else {
			deletedCount++
		}
	}

	SuccessResponse(w, map[string]any{
		"status":        "emptied",
		"deleted":       deletedCount,
		"failed":        failedCount,
		"total":         len(items),
	})
}
