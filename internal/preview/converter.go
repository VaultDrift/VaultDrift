package preview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vaultdrift/vaultdrift/internal/db"
	"github.com/vaultdrift/vaultdrift/internal/storage"
	"github.com/vaultdrift/vaultdrift/internal/util"
	"github.com/vaultdrift/vaultdrift/internal/vfs"
)

// DocumentConverter handles office document preview generation
type DocumentConverter struct {
	vfs      *vfs.VFS
	db       *db.Manager
	storage  storage.Backend
	cacheDir string
	enabled  bool
}

// NewDocumentConverter creates a new document converter
func NewDocumentConverter(vfsService *vfs.VFS, database *db.Manager, store storage.Backend) *DocumentConverter {
	cacheDir := os.TempDir() + "/vaultdrift-previews"
	_ = os.MkdirAll(cacheDir, 0750)

	// Check if LibreOffice is available
	enabled := false
	if _, err := exec.LookPath("soffice"); err == nil {
		enabled = true
	} else if _, err := exec.LookPath("libreoffice"); err == nil {
		enabled = true
	}

	return &DocumentConverter{
		vfs:      vfsService,
		db:       database,
		storage:  store,
		cacheDir: cacheDir,
		enabled:  enabled,
	}
}

// IsEnabled returns true if document conversion is available
func (c *DocumentConverter) IsEnabled() bool {
	return c.enabled
}

// SupportedFormats returns list of supported document MIME types
func (c *DocumentConverter) SupportedFormats() []string {
	return []string{
		// Microsoft Office
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document", // DOCX
		"application/msword", // DOC
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", // XLSX
		"application/vnd.ms-excel", // XLS
		"application/vnd.openxmlformats-officedocument.presentationml.presentation", // PPTX
		"application/vnd.ms-powerpoint",                                             // PPT
		// OpenDocument
		"application/vnd.oasis.opendocument.text",         // ODT
		"application/vnd.oasis.opendocument.spreadsheet",  // ODS
		"application/vnd.oasis.opendocument.presentation", // ODP
		// Other
		"application/rtf",
		"text/plain",
		"text/csv",
	}
}

// CanPreview checks if a file type can be previewed
func (c *DocumentConverter) CanPreview(mimeType string) bool {
	if !c.enabled {
		return false
	}
	for _, supported := range c.SupportedFormats() {
		if supported == mimeType {
			return true
		}
	}
	return false
}

// PreviewResult contains the generated preview info
type PreviewResult struct {
	FileID      string    `json:"file_id"`
	PreviewPath string    `json:"preview_path"`
	PreviewType string    `json:"preview_type"` // "pdf", "html", "txt"
	PageCount   int       `json:"page_count"`
	GeneratedAt time.Time `json:"generated_at"`
}

// GeneratePreview creates a preview for a document
func (c *DocumentConverter) GeneratePreview(ctx context.Context, fileID string) (*PreviewResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("document conversion not available - install LibreOffice")
	}

	// Sanitize fileID to prevent path traversal
	fileID, err := util.SanitizeFileID(fileID)
	if err != nil {
		return nil, err
	}

	// Get file metadata
	file, err := c.vfs.GetFile(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	if !c.CanPreview(file.MimeType) {
		return nil, fmt.Errorf("unsupported file type: %s", file.MimeType)
	}

	// Check if preview already exists
	previewPath := filepath.Join(c.cacheDir, fileID+".pdf") // #nosec G703 - fileID sanitized at start
	if info, err := os.Stat(previewPath); err == nil { // #nosec G703
		return &PreviewResult{
			FileID:      fileID,
			PreviewPath: previewPath,
			PreviewType: "pdf",
			GeneratedAt: info.ModTime(),
		}, nil
	}

	// Get manifest
	if file.ManifestID == nil || *file.ManifestID == "" {
		return nil, fmt.Errorf("file has no content")
	}

	manifest, err := c.db.GetManifest(ctx, *file.ManifestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %w", err)
	}

	// Assemble file to temp location
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("vaultdrift-doc-%s.tmp", fileID)) // #nosec G703 - fileID sanitized
	if err := c.assembleFile(ctx, manifest, tempFile); err != nil {
		return nil, fmt.Errorf("failed to assemble file: %w", err)
	}
	defer os.Remove(tempFile)

	// Convert to PDF using LibreOffice
	outputDir := filepath.Join(c.cacheDir, fileID+"_work") // #nosec G703 - fileID sanitized
	_ = os.MkdirAll(outputDir, 0750) // #nosec G703
	defer os.RemoveAll(outputDir)

	sofficePath, _ := exec.LookPath("soffice")
	if sofficePath == "" {
		sofficePath, _ = exec.LookPath("libreoffice")
	}

	cmd := exec.CommandContext(ctx, sofficePath, // #nosec G204 G702 - fileID sanitized, tempFile/outputDir system-generated
		"--headless",
		"--convert-to", "pdf",
		"--outdir", outputDir,
		"--norestore",
		"--writer",
		tempFile,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("conversion failed: %w\n%s", err, output)
	}

	// Find the generated PDF
	baseName := strings.TrimSuffix(filepath.Base(tempFile), filepath.Ext(tempFile))
	generatedPDF := filepath.Join(outputDir, baseName+".pdf")

	// Move to cache location
	if err := os.Rename(generatedPDF, previewPath); err != nil { // #nosec G703 - previewPath uses sanitized fileID
		return nil, fmt.Errorf("failed to move preview: %w", err)
	}

	return &PreviewResult{
		FileID:      fileID,
		PreviewPath: previewPath,
		PreviewType: "pdf",
		GeneratedAt: time.Now(),
	}, nil
}

// GenerateHTML creates an HTML preview for web viewing
func (c *DocumentConverter) GenerateHTML(ctx context.Context, fileID string) (string, error) {
	if !c.enabled {
		return "", fmt.Errorf("document conversion not available")
	}

	// Sanitize fileID to prevent path traversal
	fileID, err := util.SanitizeFileID(fileID)
	if err != nil {
		return "", err
	}

	// First generate PDF
	result, err := c.GeneratePreview(ctx, fileID)
	if err != nil {
		return "", err
	}

	// Convert PDF to HTML using pdftotext or pdf2html
	htmlPath := filepath.Join(c.cacheDir, fileID+".html") // #nosec G703 - fileID sanitized at start

	// Check if pdftotext is available
	if _, err := exec.LookPath("pdftotext"); err == nil {
		cmd := exec.CommandContext(ctx, "pdftotext", // #nosec G204 G702 - fileID sanitized, paths controlled
			"-htmlmeta",
			result.PreviewPath,
			htmlPath,
		)
		if err := cmd.Run(); err != nil {
			// Fall back to simple iframe embed
			return c.generateSimpleHTML(fileID, result.PreviewPath), nil
		}
		return htmlPath, nil
	}

	// Simple HTML wrapper for PDF
	return c.generateSimpleHTML(fileID, result.PreviewPath), nil
}

// generateSimpleHTML creates a simple HTML wrapper for PDF viewing
func (c *DocumentConverter) generateSimpleHTML(fileID, pdfPath string) string {
	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Document Preview</title>
    <style>
        body { margin: 0; padding: 0; overflow: hidden; }
        #pdf-viewer { width: 100%%; height: 100vh; }
    </style>
</head>
<body>
    <iframe id="pdf-viewer" src="/api/v1/preview/%s/pdf" type="application/pdf"></iframe>
</body>
</html>`, fileID)

	htmlPath := filepath.Join(c.cacheDir, fileID+".html") // #nosec G703 - fileID sanitized in caller
	_ = os.WriteFile(htmlPath, []byte(htmlContent), 0600) // #nosec G703
	return htmlPath
}

// GetPreview returns the preview file path, generating if needed
func (c *DocumentConverter) GetPreview(ctx context.Context, fileID string) (*PreviewResult, error) {
	// Sanitize fileID to prevent path traversal
	fileID, err := util.SanitizeFileID(fileID)
	if err != nil {
		return nil, err
	}

	previewPath := filepath.Join(c.cacheDir, fileID+".pdf") // #nosec G703 - fileID sanitized above

	// Check cache
	if info, err := os.Stat(previewPath); err == nil { // #nosec G703
		return &PreviewResult{
			FileID:      fileID,
			PreviewPath: previewPath,
			PreviewType: "pdf",
			GeneratedAt: info.ModTime(),
		}, nil
	}

	// Generate new preview
	return c.GeneratePreview(ctx, fileID)
}

// CleanupOldPreviews removes previews older than specified duration
func (c *DocumentConverter) CleanupOldPreviews(maxAge time.Duration) error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(c.cacheDir, entry.Name()))
		}
	}

	return nil
}

// assembleFile assembles chunks into a single file
func (c *DocumentConverter) assembleFile(ctx context.Context, manifest *db.Manifest, outputPath string) error {
	f, err := os.Create(outputPath) // #nosec G304 G703 - outputPath constructed from sanitized fileID
	if err != nil {
		return err
	}
	defer f.Close()

	for _, hash := range manifest.Chunks {
		data, err := c.storage.Get(ctx, hash)
		if err != nil {
			return fmt.Errorf("failed to get chunk %s: %w", hash, err)
		}
		if _, err := f.Write(data); err != nil {
			return err
		}
	}

	return nil
}
