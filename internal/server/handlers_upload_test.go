package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestUploadFlow tests complete upload session lifecycle
func TestUploadFlow(t *testing.T) {
	// Create upload handler with mock VFS
	uploadHandler := NewUploadHandler(nil)

	// Test 1: Create upload session
	t.Run("CreateUploadSession", func(t *testing.T) {
		reqBody := createUploadRequest{
			Name:     "test-file.txt",
			Size:     1024,
			MimeType: "text/plain",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/uploads", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		// Mock user ID using the same context key type
		ctx := context.WithValue(req.Context(), userIDKey, "test-user")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		uploadHandler.createUpload(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var respWrapper struct {
			Success bool                 `json:"success"`
			Data    createUploadResponse `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &respWrapper); err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}
		resp := respWrapper.Data

		if resp.SessionID == "" {
			t.Error("SessionID empty")
		}

		t.Logf("✅ Upload session created: %s", resp.SessionID)
	})

	// Test 2: Upload chunk
	t.Run("UploadChunk", func(t *testing.T) {
		// Create session first
		session := &UploadSession{
			ID:          "test-session-1",
			UserID:      "test-user",
			FileName:    "test.txt",
			Size:        100,
			TotalChunks: 1,
			Chunks:      make(map[int]*ChunkInfo),
			CreatedAt:   time.Now(),
			ExpiresAt:   time.Now().Add(SessionTTL),
		}
		uploadHandler.sessions[session.ID] = session

		// Chunk data
		chunkData := []byte("Hello, this is test chunk data!")

		req := httptest.NewRequest("PUT", "/api/v1/uploads/test-session-1/chunks/0", bytes.NewReader(chunkData))
		ctx := context.WithValue(req.Context(), userIDKey, "test-user")
		req = req.WithContext(ctx)
		// Set path values for Go 1.22+ pattern matching
		req.SetPathValue("id", "test-session-1")
		req.SetPathValue("index", "0")
		w := httptest.NewRecorder()

		uploadHandler.uploadChunk(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusCreated {
			t.Errorf("Expected 200/201, got %d: %s", w.Code, w.Body.String())
		}

		t.Logf("✅ Chunk uploaded successfully")
	})

	// Test 3: Complete upload (without VFS - will fail but tests flow)
	t.Run("CompleteUploadFlow", func(t *testing.T) {
		// Create session and add chunk
		session := &UploadSession{
			ID:          "test-session-complete",
			UserID:      "test-user",
			FileName:    "test-complete.txt",
			Size:        31,
			TotalChunks: 1,
			Chunks: map[int]*ChunkInfo{
				0: {Index: 0, Size: 31, UploadedAt: time.Now()},
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(SessionTTL),
			Status:    "completed",
		}
		uploadHandler.sessions[session.ID] = session

		// Get status first
		req := httptest.NewRequest("GET", "/api/v1/uploads/test-session-complete/status", nil)
		ctx := context.WithValue(req.Context(), userIDKey, "test-user")
		req = req.WithContext(ctx)
		req.SetPathValue("id", "test-session-complete")
		w := httptest.NewRecorder()

		uploadHandler.getUploadStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for status, got %d: %s", w.Code, w.Body.String())
		}

		var statusWrapper struct {
			Success bool                 `json:"success"`
			Data    uploadStatusResponse `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &statusWrapper)
		statusResp := statusWrapper.Data

		if statusResp.UploadedBytes != 31 {
			t.Errorf("Expected 31 bytes uploaded, got %d", statusResp.UploadedBytes)
		}

		t.Logf("✅ Upload status check passed: %d bytes uploaded", statusResp.UploadedBytes)
	})

	// Test 4: Cancel upload
	t.Run("CancelUpload", func(t *testing.T) {
		// Create session
		session := &UploadSession{
			ID:        "test-session-cancel",
			UserID:    "test-user",
			FileName:  "cancel-me.txt",
			Size:      100,
			Chunks:    make(map[int]*ChunkInfo),
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(SessionTTL),
		}
		uploadHandler.sessions[session.ID] = session

		req := httptest.NewRequest("DELETE", "/api/v1/uploads/test-session-cancel", nil)
		ctx := context.WithValue(req.Context(), userIDKey, "test-user")
		req = req.WithContext(ctx)
		req.SetPathValue("id", "test-session-cancel")
		w := httptest.NewRecorder()

		uploadHandler.cancelUpload(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
		}

		// Verify session deleted
		uploadHandler.sessionsMutex.RLock()
		_, exists := uploadHandler.sessions["test-session-cancel"]
		uploadHandler.sessionsMutex.RUnlock()

		if exists {
			t.Error("Session should have been deleted")
		}

		t.Logf("✅ Upload cancelled and session cleaned up")
	})
}

// TestUploadValidation tests input validation
func TestUploadValidation(t *testing.T) {
	handler := NewUploadHandler(nil)

	tests := []struct {
		name       string
		req        createUploadRequest
		wantStatus int
	}{
		{
			name:       "Empty name",
			req:        createUploadRequest{Name: "", Size: 100},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Zero size",
			req:        createUploadRequest{Name: "test.txt", Size: 0},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Negative size",
			req:        createUploadRequest{Name: "test.txt", Size: -1},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Valid request",
			req:        createUploadRequest{Name: "test.txt", Size: 1024, MimeType: "text/plain"},
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.req)
			req := httptest.NewRequest("POST", "/api/v1/uploads", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(req.Context(), userIDKey, "test-user")
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handler.createUpload(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

// TestUploadAuthorization tests authorization checks
func TestUploadAuthorization(t *testing.T) {
	handler := NewUploadHandler(nil)

	// Create a session for test-user
	session := &UploadSession{
		ID:          "auth-test-session",
		UserID:      "test-user",
		FileName:    "private.txt",
		Size:        100,
		TotalChunks: 1,
		Chunks:      make(map[int]*ChunkInfo),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(SessionTTL),
	}
	handler.sessions[session.ID] = session

	t.Run("AccessDeniedForOtherUser", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/uploads/auth-test-session/status", nil)
		ctx := context.WithValue(req.Context(), userIDKey, "other-user")
		req = req.WithContext(ctx)
		req.SetPathValue("id", "auth-test-session")
		w := httptest.NewRecorder()

		handler.getUploadStatus(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected 403, got %d", w.Code)
		}
	})

	t.Run("UnauthorizedRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/uploads/auth-test-session/status", nil)
		// No user context - simulating unauthenticated request
		req.SetPathValue("id", "auth-test-session")
		w := httptest.NewRecorder()

		handler.getUploadStatus(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", w.Code)
		}
	})
}

// TestSessionExpiration tests session expiration
func TestSessionExpiration(t *testing.T) {
	handler := NewUploadHandler(nil)

	// Create an expired session
	expiredSession := &UploadSession{
		ID:          "expired-session",
		UserID:      "test-user",
		FileName:    "expired.txt",
		Size:        100,
		TotalChunks: 1,
		Chunks:      make(map[int]*ChunkInfo),
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		ExpiresAt:   time.Now().Add(-24 * time.Hour), // Expired
	}
	handler.sessions[expiredSession.ID] = expiredSession

	t.Run("ExpiredSessionRejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/uploads/expired-session/status", nil)
		ctx := context.WithValue(req.Context(), userIDKey, "test-user")
		req = req.WithContext(ctx)
		req.SetPathValue("id", "expired-session")
		w := httptest.NewRecorder()

		handler.getUploadStatus(w, req)

		// Should get 410 Gone for expired session
		if w.Code != http.StatusGone {
			t.Errorf("Expected 410 for expired session, got %d", w.Code)
		}
	})
}

// TestChunkUploadValidation tests chunk upload validation
func TestChunkUploadValidation(t *testing.T) {
	handler := NewUploadHandler(nil)

	// Create session with 2 chunks expected
	session := &UploadSession{
		ID:          "chunk-test-session",
		UserID:      "test-user",
		FileName:    "chunks.txt",
		Size:        100,
		TotalChunks: 2,
		Chunks:      make(map[int]*ChunkInfo),
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(SessionTTL),
	}
	handler.sessions[session.ID] = session

	t.Run("InvalidChunkIndex", func(t *testing.T) {
		chunkData := []byte("test data")
		req := httptest.NewRequest("PUT", "/api/v1/uploads/chunk-test-session/chunks/5", bytes.NewReader(chunkData))
		ctx := context.WithValue(req.Context(), userIDKey, "test-user")
		req = req.WithContext(ctx)
		req.SetPathValue("id", "chunk-test-session")
		req.SetPathValue("index", "5")
		w := httptest.NewRecorder()

		handler.uploadChunk(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid chunk index, got %d", w.Code)
		}
	})

	t.Run("NegativeChunkIndex", func(t *testing.T) {
		chunkData := []byte("test data")
		req := httptest.NewRequest("PUT", "/api/v1/uploads/chunk-test-session/chunks/-1", bytes.NewReader(chunkData))
		ctx := context.WithValue(req.Context(), userIDKey, "test-user")
		req = req.WithContext(ctx)
		req.SetPathValue("id", "chunk-test-session")
		req.SetPathValue("index", "-1")
		w := httptest.NewRecorder()

		handler.uploadChunk(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for negative chunk index, got %d", w.Code)
		}
	})

	t.Run("ChunkAlreadyUploaded", func(t *testing.T) {
		// First upload chunk 0
		session.Chunks[0] = &ChunkInfo{Index: 0, Size: 10, UploadedAt: time.Now()}

		chunkData := []byte("test data")
		req := httptest.NewRequest("PUT", "/api/v1/uploads/chunk-test-session/chunks/0", bytes.NewReader(chunkData))
		ctx := context.WithValue(req.Context(), userIDKey, "test-user")
		req = req.WithContext(ctx)
		req.SetPathValue("id", "chunk-test-session")
		req.SetPathValue("index", "0")
		w := httptest.NewRecorder()

		handler.uploadChunk(w, req)

		// Should return OK with already_uploaded status
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for already uploaded chunk, got %d", w.Code)
		}

		var respWrapper struct {
			Success bool              `json:"success"`
			Data    map[string]string `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &respWrapper)
		resp := respWrapper.Data
		if resp["status"] != "already_uploaded" {
			t.Errorf("Expected 'already_uploaded' status, got %s", resp["status"])
		}
	})
}

// TestMissingChunks tests missing chunks calculation
func TestMissingChunks(t *testing.T) {
	handler := NewUploadHandler(nil)

	session := &UploadSession{
		ID:          "missing-chunks-session",
		UserID:      "test-user",
		FileName:    "multi.txt",
		Size:        1000,
		TotalChunks: 5,
		Chunks: map[int]*ChunkInfo{
			0: {Index: 0, Size: 200},
			2: {Index: 2, Size: 200},
			4: {Index: 4, Size: 200},
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(SessionTTL),
	}
	handler.sessions[session.ID] = session

	req := httptest.NewRequest("GET", "/api/v1/uploads/missing-chunks-session/status", nil)
	ctx := context.WithValue(req.Context(), userIDKey, "test-user")
	req = req.WithContext(ctx)
	req.SetPathValue("id", "missing-chunks-session")
	w := httptest.NewRecorder()

	handler.getUploadStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var respWrapper struct {
		Success bool                 `json:"success"`
		Data    uploadStatusResponse `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &respWrapper)
	resp := respWrapper.Data

	// Should report chunks 1 and 3 as missing
	if len(resp.MissingChunks) != 2 {
		t.Errorf("Expected 2 missing chunks, got %d", len(resp.MissingChunks))
	}

	expectedMissing := []int{1, 3}
	for i, chunk := range resp.MissingChunks {
		if chunk != expectedMissing[i] {
			t.Errorf("Expected missing chunk %d, got %d", expectedMissing[i], chunk)
		}
	}

	t.Logf("✅ Missing chunks correctly identified: %v", resp.MissingChunks)
}
