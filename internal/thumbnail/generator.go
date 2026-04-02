package thumbnail

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sync"

	"github.com/vaultdrift/vaultdrift/internal/storage"
	"golang.org/x/image/draw"
)

// Sizes defines thumbnail sizes
type Size struct {
	Name   string
	Width  int
	Height int
}

var (
	// SizeSmall is a small icon (used in file lists)
	SizeSmall = Size{Name: "small", Width: 64, Height: 64}

	// SizeMedium is a medium preview (used in grid view)
	SizeMedium = Size{Name: "medium", Width: 256, Height: 256}

	// SizeLarge is a large preview (used in preview modal)
	SizeLarge = Size{Name: "large", Width: 1024, Height: 1024}

	// AllSizes contains all thumbnail sizes
	AllSizes = []Size{SizeSmall, SizeMedium, SizeLarge}
)

// Generator generates thumbnails
type Generator struct {
	storage    storage.Backend
	cacheDir   string
	mu         sync.RWMutex
	generating map[string]bool // Track in-progress generations
}

// NewGenerator creates a new thumbnail generator
func NewGenerator(store storage.Backend, cacheDir string) *Generator {
	return &Generator{
		storage:    store,
		cacheDir:   cacheDir,
		generating: make(map[string]bool),
	}
}

// Init initializes the thumbnail cache directory
func (g *Generator) Init() error {
	for _, size := range AllSizes {
		dir := filepath.Join(g.cacheDir, size.Name)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create thumbnail dir %s: %w", dir, err)
		}
	}
	return nil
}

// Generate generates thumbnails for an image file
func (g *Generator) Generate(fileID string, mimeType string, data []byte) (map[string]string, error) {
	// Check if generation is already in progress
	g.mu.Lock()
	if g.generating[fileID] {
		g.mu.Unlock()
		return nil, fmt.Errorf("thumbnail generation already in progress")
	}
	g.generating[fileID] = true
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		delete(g.generating, fileID)
		g.mu.Unlock()
	}()

	// Check if this is an image we can process
	if !g.isSupportedImage(mimeType) {
		return nil, fmt.Errorf("unsupported image type: %s", mimeType)
	}

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	results := make(map[string]string)

	// Generate thumbnails for each size
	for _, size := range AllSizes {
		thumbPath := g.thumbnailPath(fileID, size.Name)

		// Generate thumbnail
		thumb := g.resizeImage(img, size.Width, size.Height)

		// Save thumbnail
		if err := g.saveThumbnail(thumb, thumbPath, format); err != nil {
			return nil, fmt.Errorf("failed to save thumbnail %s: %w", size.Name, err)
		}

		results[size.Name] = thumbPath
	}

	return results, nil
}

// Get retrieves a thumbnail path if it exists
func (g *Generator) Get(fileID string, size string) (string, bool) {
	thumbPath := g.thumbnailPath(fileID, size)
	if _, err := os.Stat(thumbPath); err == nil {
		return thumbPath, true
	}
	return "", false
}

// Delete deletes all thumbnails for a file
func (g *Generator) Delete(fileID string) error {
	for _, size := range AllSizes {
		thumbPath := g.thumbnailPath(fileID, size.Name)
		if err := os.Remove(thumbPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// isSupportedImage checks if the mime type is a supported image
func (g *Generator) isSupportedImage(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

// resizeImage resizes an image to fit within the given dimensions
func (g *Generator) resizeImage(src image.Image, maxWidth, maxHeight int) image.Image {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	width, height := srcWidth, srcHeight

	if width > maxWidth {
		height = height * maxWidth / width
		width = maxWidth
	}
	if height > maxHeight {
		width = width * maxHeight / height
		height = maxHeight
	}

	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// Use Lanczos resampling for high quality
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	return dst
}

// saveThumbnail saves a thumbnail to disk
func (g *Generator) saveThumbnail(img image.Image, path string, format string) error {
	file, err := os.Create(path) // #nosec G304 - path constructed from sanitized fileID
	if err != nil {
		return err
	}
	defer file.Close()

	switch format {
	case "jpeg", "jpg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 85})
	case "gif":
		return gif.Encode(file, img, nil)
	default:
		return png.Encode(file, img)
	}
}

// thumbnailPath returns the path for a thumbnail
func (g *Generator) thumbnailPath(fileID string, size string) string {
	return filepath.Join(g.cacheDir, size, fileID+".png")
}

// CanGenerate checks if thumbnails can be generated for the given mime type
func (g *Generator) CanGenerate(mimeType string) bool {
	return g.isSupportedImage(mimeType)
}
func GetSupportedTypes() []string {
	return []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}
}
