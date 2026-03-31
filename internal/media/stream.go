package media

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// StreamHandler handles video streaming requests
type StreamHandler struct {
	vfs      *vfs.VFS
	db       *db.Manager
	storage  storage.Backend
	cacheDir string
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(vfsService *vfs.VFS, database *db.Manager, store storage.Backend) *StreamHandler {
	cacheDir := os.TempDir() + "/vaultdrift-streams"
	os.MkdirAll(cacheDir, 0755)

	return &StreamHandler{
		vfs:      vfsService,
		db:       database,
		storage:  store,
		cacheDir: cacheDir,
	}
}

// RegisterRoutes registers streaming endpoints
func (h *StreamHandler) RegisterRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	// HLS playlist endpoint
	mux.Handle("GET /api/v1/media/{fileID}/playlist.m3u8", auth(http.HandlerFunc(h.handlePlaylist)))

	// HLS segment endpoint
	mux.Handle("GET /api/v1/media/{fileID}/{segment}", auth(http.HandlerFunc(h.handleSegment)))

	// Video metadata
	mux.Handle("GET /api/v1/media/{fileID}/info", auth(http.HandlerFunc(h.handleStreamInfo)))

	// Transcode status/check
	mux.Handle("GET /api/v1/media/{fileID}/status", auth(http.HandlerFunc(h.handleTranscodeStatus)))
}

// StreamInfo contains video metadata
type StreamInfo struct {
	Duration  float64 `json:"duration"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Bitrate   int64   `json:"bitrate"`
	Codec     string  `json:"codec"`
	FrameRate string  `json:"frame_rate"`
	HasHLS    bool    `json:"has_hls"`
	Ready     bool    `json:"ready"`
}

// TranscodeStatus represents transcoding progress
type TranscodeStatus struct {
	FileID    string    `json:"file_id"`
	Ready     bool      `json:"ready"`
	Progress  float64   `json:"progress"`
	Qualities []string  `json:"qualities"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *StreamHandler) handleStreamInfo(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Get file from VFS
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	if file.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if file is video
	if !isVideoFile(file.Name) {
		http.Error(w, "Not a video file", http.StatusBadRequest)
		return
	}

	// Get video metadata
	info, err := h.probeVideo(r.Context(), fileID)
	if err != nil {
		// Return basic info if probing fails
		info = &StreamInfo{
			Ready:  false,
			HasHLS: false,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (h *StreamHandler) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Verify ownership
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if file.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check if HLS is already generated
	playlistPath := filepath.Join(h.cacheDir, fileID, "playlist.m3u8")

	if _, err := os.Stat(playlistPath); os.IsNotExist(err) {
		// Generate HLS on-demand (async)
		go h.generateHLS(context.Background(), fileID)
		http.Error(w, "Stream not ready, generating...", http.StatusAccepted)
		return
	}

	// Serve playlist
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, playlistPath)
}

func (h *StreamHandler) handleSegment(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := r.PathValue("fileID")
	segment := r.PathValue("segment")

	if fileID == "" || segment == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate segment name to prevent directory traversal
	if strings.Contains(segment, "..") || strings.Contains(segment, "/") {
		http.Error(w, "Invalid segment", http.StatusBadRequest)
		return
	}

	// Verify ownership
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if file.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	segmentPath := filepath.Join(h.cacheDir, fileID, segment)

	// Check if segment exists
	if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
		http.Error(w, "Segment not found", http.StatusNotFound)
		return
	}

	// Serve segment with proper headers for streaming
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, segmentPath)
}

func (h *StreamHandler) handleTranscodeStatus(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	fileID := r.PathValue("fileID")
	if fileID == "" {
		http.Error(w, "File ID required", http.StatusBadRequest)
		return
	}

	// Verify ownership
	file, err := h.vfs.GetFile(r.Context(), fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	if file.UserID != userID {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check HLS status
	outputDir := filepath.Join(h.cacheDir, fileID)
	playlistPath := filepath.Join(outputDir, "playlist.m3u8")

	status := TranscodeStatus{
		FileID:    fileID,
		Ready:     false,
		Progress:  0,
		Qualities: []string{},
	}

	if info, err := os.Stat(playlistPath); err == nil {
		status.Ready = true
		status.CreatedAt = info.ModTime()
		status.Progress = 100

		// Detect available qualities
		qualities := []string{"480p", "720p", "1080p"}
		for _, q := range qualities {
			variantPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.m3u8", fileID, q))
			if _, err := os.Stat(variantPath); err == nil {
				status.Qualities = append(status.Qualities, q)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// generateHLS generates HLS segments from video file
func (h *StreamHandler) generateHLS(ctx context.Context, fileID string) error {
	// Create output directory
	outputDir := filepath.Join(h.cacheDir, fileID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get file metadata
	file, err := h.vfs.GetFile(ctx, fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	if file.ManifestID == nil || *file.ManifestID == "" {
		return fmt.Errorf("file has no content")
	}

	// Get manifest
	manifest, err := h.db.GetManifest(ctx, *file.ManifestID)
	if err != nil {
		return fmt.Errorf("failed to get manifest: %w", err)
	}

	// Assemble file to temp location
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("vaultdrift-video-%s.tmp", fileID))
	if err := h.assembleFile(ctx, manifest, tempFile); err != nil {
		return fmt.Errorf("failed to assemble file: %w", err)
	}
	defer os.Remove(tempFile)

	// Generate HLS using ffmpeg
	qualities := []struct {
		name    string
		width   int
		height  int
		bitrate string
	}{
		{"1080p", 1920, 1080, "5000k"},
		{"720p", 1280, 720, "2500k"},
		{"480p", 854, 480, "1000k"},
	}

	var variants []string
	for _, q := range qualities {
		variantPlaylist := fmt.Sprintf("%s_%s.m3u8", fileID, q.name)
		variantPath := filepath.Join(outputDir, variantPlaylist)

		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-i", tempFile,
			"-vf", fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease", q.width, q.height),
			"-c:a", "aac",
			"-b:a", "128k",
			"-c:v", "libx264",
			"-b:v", q.bitrate,
			"-preset", "fast",
			"-hls_time", "10",
			"-hls_playlist_type", "vod",
			"-hls_segment_filename", filepath.Join(outputDir, fmt.Sprintf("%s_%s_%%03d.ts", fileID, q.name)),
			variantPath,
		)

		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("Failed to generate %s variant: %v\n%s\n", q.name, err, output)
			continue
		}

		bandwidth := parseBitrate(q.bitrate)
		variants = append(variants, fmt.Sprintf(
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d\n%s",
			bandwidth, q.width, q.height, variantPlaylist,
		))
	}

	if len(variants) == 0 {
		return fmt.Errorf("failed to generate any quality variants")
	}

	// Generate master playlist
	masterPlaylist := "#EXTM3U\n#EXT-X-VERSION:3\n"
	for _, v := range variants {
		masterPlaylist += v + "\n"
	}

	masterPath := filepath.Join(outputDir, "playlist.m3u8")
	if err := os.WriteFile(masterPath, []byte(masterPlaylist), 0644); err != nil {
		return fmt.Errorf("failed to write master playlist: %w", err)
	}

	return nil
}

// assembleFile assembles chunks into a single file
func (h *StreamHandler) assembleFile(ctx context.Context, manifest *db.Manifest, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, hash := range manifest.Chunks {
		data, err := h.storage.Get(ctx, hash)
		if err != nil {
			return fmt.Errorf("failed to get chunk %s: %w", hash, err)
		}
		if _, err := f.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// probeVideo extracts video metadata using ffprobe
func (h *StreamHandler) probeVideo(ctx context.Context, fileID string) (*StreamInfo, error) {
	// Get file metadata
	file, err := h.vfs.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if file.ManifestID == nil || *file.ManifestID == "" {
		return nil, fmt.Errorf("file has no content")
	}

	manifest, err := h.db.GetManifest(ctx, *file.ManifestID)
	if err != nil {
		return nil, err
	}

	// Assemble to temp file
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("vaultdrift-probe-%s.tmp", fileID))
	if err := h.assembleFile(ctx, manifest, tempFile); err != nil {
		return nil, err
	}
	defer os.Remove(tempFile)

	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration,bit_rate",
		"-show_entries", "stream=width,height,codec_name,r_frame_rate",
		"-of", "json",
		tempFile,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probeData struct {
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
		Streams []struct {
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			CodecName string `json:"codec_name"`
			FrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, err
	}

	info := &StreamInfo{
		Ready: true,
	}

	if d, err := strconv.ParseFloat(probeData.Format.Duration, 64); err == nil {
		info.Duration = d
	}
	if b, err := strconv.ParseInt(probeData.Format.BitRate, 10, 64); err == nil {
		info.Bitrate = b
	}

	for _, s := range probeData.Streams {
		if s.Width > 0 && s.Height > 0 {
			info.Width = s.Width
			info.Height = s.Height
			info.Codec = s.CodecName
			info.FrameRate = s.FrameRate
			break
		}
	}

	playlistPath := filepath.Join(h.cacheDir, fileID, "playlist.m3u8")
	if _, err := os.Stat(playlistPath); err == nil {
		info.HasHLS = true
	}

	return info, nil
}

// CleanupOldStreams removes stream cache older than specified duration
func (h *StreamHandler) CleanupOldStreams(maxAge time.Duration) error {
	entries, err := os.ReadDir(h.cacheDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			dirPath := filepath.Join(h.cacheDir, entry.Name())
			os.RemoveAll(dirPath)
		}
	}

	return nil
}

// Helper functions

func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := []string{".mp4", ".mkv", ".avi", ".mov", ".webm", ".flv", ".wmv", ".m4v", ".ogv"}
	for _, ve := range videoExts {
		if ext == ve {
			return true
		}
	}
	return false
}

func parseBitrate(bitrate string) int {
	bitrate = strings.TrimSuffix(bitrate, "k")
	if v, err := strconv.Atoi(bitrate); err == nil {
		return v * 1000
	}
	return 1000000
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return userID
	}
	return ""
}

