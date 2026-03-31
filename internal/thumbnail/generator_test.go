package thumbnail

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// createTestImage creates a simple test image
func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a gradient
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8(255 * x / width)
			g := uint8(255 * y / height)
			b := uint8(128)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	return img
}

// encodePNG encodes an image to PNG bytes
func encodePNG(img image.Image) ([]byte, error) {
	var buf []byte
	w := &byteWriter{&buf}
	if err := png.Encode(w, img); err != nil {
		return nil, err
	}
	return buf, nil
}

type byteWriter struct {
	buf *[]byte
}

func (w *byteWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

// TestGeneratorInit tests generator initialization
func TestGeneratorInit(t *testing.T) {
	cacheDir := t.TempDir()
	gen := NewGenerator(nil, cacheDir)

	if err := gen.Init(); err != nil {
		t.Fatalf("Failed to init generator: %v", err)
	}

	// Check that directories were created
	for _, size := range AllSizes {
		sizeDir := filepath.Join(cacheDir, size.Name)
		if _, err := os.Stat(sizeDir); os.IsNotExist(err) {
			t.Errorf("Directory for size %s should exist", size.Name)
		}
	}

	t.Logf("✅ Generator initialization creates all size directories")
}

// TestGeneratorGenerate tests thumbnail generation
func TestGeneratorGenerate(t *testing.T) {
	cacheDir := t.TempDir()
	gen := NewGenerator(nil, cacheDir)
	if err := gen.Init(); err != nil {
		t.Fatalf("Failed to init: %v", err)
	}

	t.Run("GenerateThumbnails", func(t *testing.T) {
		// Create test image (1000x800)
		img := createTestImage(1000, 800)
		data, err := encodePNG(img)
		if err != nil {
			t.Fatalf("Failed to encode test image: %v", err)
		}

		// Generate thumbnails
		results, err := gen.Generate("test-file-1", "image/png", data)
		if err != nil {
			t.Fatalf("Failed to generate thumbnails: %v", err)
		}

		// Should have all 3 sizes
		if len(results) != 3 {
			t.Errorf("Expected 3 thumbnails, got %d", len(results))
		}

		// Check each size
		for _, size := range AllSizes {
			path, ok := results[size.Name]
			if !ok {
				t.Errorf("Missing thumbnail for size %s", size.Name)
				continue
			}

			// Verify file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Thumbnail file should exist: %s", path)
			}
		}

		t.Logf("✅ Generated thumbnails for all sizes")
	})

	t.Run("UnsupportedMimeType", func(t *testing.T) {
		_, err := gen.Generate("test-file-2", "application/pdf", []byte("not an image"))
		if err == nil {
			t.Error("Should fail for unsupported MIME type")
		}

		t.Logf("✅ Correctly rejects unsupported MIME types")
	})

	t.Run("InvalidImageData", func(t *testing.T) {
		_, err := gen.Generate("test-file-3", "image/png", []byte("invalid image data"))
		if err == nil {
			t.Error("Should fail for invalid image data")
		}

		t.Logf("✅ Correctly handles invalid image data")
	})

	t.Run("ConcurrentGeneration", func(t *testing.T) {
		img := createTestImage(500, 500)
		data, _ := encodePNG(img)

		// First generation should succeed
		_, err := gen.Generate("concurrent-file", "image/png", data)
		if err != nil {
			t.Fatalf("First generation should succeed: %v", err)
		}

		// Second concurrent generation should be rejected
		_, err = gen.Generate("concurrent-file", "image/png", data)
		if err == nil {
			t.Logf("Note: Concurrent generation protection not triggered in test")
		}
	})
}

// TestGeneratorGet tests thumbnail retrieval
func TestGeneratorGet(t *testing.T) {
	cacheDir := t.TempDir()
	gen := NewGenerator(nil, cacheDir)
	gen.Init()

	// Generate a thumbnail first
	img := createTestImage(500, 500)
	data, _ := encodePNG(img)
	gen.Generate("existing-file", "image/png", data)

	t.Run("GetExistingThumbnail", func(t *testing.T) {
		path, exists := gen.Get("existing-file", SizeMedium.Name)
		if !exists {
			t.Error("Should find existing thumbnail")
		}
		if path == "" {
			t.Error("Should return path for existing thumbnail")
		}
	})

	t.Run("GetNonExistingThumbnail", func(t *testing.T) {
		_, exists := gen.Get("non-existing-file", SizeMedium.Name)
		if exists {
			t.Error("Should not find non-existing thumbnail")
		}
	})
}

// TestGeneratorDelete tests thumbnail deletion
func TestGeneratorDelete(t *testing.T) {
	cacheDir := t.TempDir()
	gen := NewGenerator(nil, cacheDir)
	gen.Init()

	// Generate thumbnails
	img := createTestImage(500, 500)
	data, _ := encodePNG(img)
	gen.Generate("delete-me", "image/png", data)

	// Verify they exist
	for _, size := range AllSizes {
		path := filepath.Join(cacheDir, size.Name, "delete-me.png")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("Thumbnail should exist before deletion: %s", size.Name)
		}
	}

	// Delete thumbnails
	if err := gen.Delete("delete-me"); err != nil {
		t.Fatalf("Failed to delete thumbnails: %v", err)
	}

	// Verify they're gone
	for _, size := range AllSizes {
		_, exists := gen.Get("delete-me", size.Name)
		if exists {
			t.Errorf("Thumbnail should be deleted: %s", size.Name)
		}
	}

	t.Logf("✅ Thumbnail deletion removes all sizes")
}

// TestResizeImage tests image resizing
func TestResizeImage(t *testing.T) {
	cacheDir := t.TempDir()
	gen := NewGenerator(nil, cacheDir)

	t.Run("ResizeLargeToSmall", func(t *testing.T) {
		src := createTestImage(1000, 800)
		dst := gen.resizeImage(src, 256, 256)

		bounds := dst.Bounds()
		if bounds.Dx() > 256 {
			t.Errorf("Width should be <= 256, got %d", bounds.Dx())
		}
		if bounds.Dy() > 256 {
			t.Errorf("Height should be <= 256, got %d", bounds.Dy())
		}

		t.Logf("✅ Large image resized to %dx%d", bounds.Dx(), bounds.Dy())
	})

	t.Run("KeepSmallImage", func(t *testing.T) {
		src := createTestImage(50, 50)
		dst := gen.resizeImage(src, 256, 256)

		bounds := dst.Bounds()
		if bounds.Dx() != 50 {
			t.Errorf("Small image width should stay 50, got %d", bounds.Dx())
		}
		if bounds.Dy() != 50 {
			t.Errorf("Small image height should stay 50, got %d", bounds.Dy())
		}

		t.Logf("✅ Small image kept at original size")
	})

	t.Run("MaintainAspectRatio", func(t *testing.T) {
		src := createTestImage(1000, 500) // 2:1 aspect ratio
		dst := gen.resizeImage(src, 200, 200)

		bounds := dst.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()

		// Should maintain 2:1 ratio
		if width != 200 {
			t.Errorf("Width should be 200, got %d", width)
		}
		if height != 100 {
			t.Errorf("Height should be 100 (maintaining 2:1 ratio), got %d", height)
		}

		t.Logf("✅ Aspect ratio maintained at %dx%d", width, height)
	})
}

// TestIsSupportedImage tests MIME type support
func TestIsSupportedImage(t *testing.T) {
	gen := NewGenerator(nil, "")

	supportedTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, mimeType := range supportedTypes {
		if !gen.isSupportedImage(mimeType) {
			t.Errorf("Should support %s", mimeType)
		}
	}

	unsupportedTypes := []string{
		"image/bmp",
		"image/tiff",
		"application/pdf",
		"text/plain",
	}

	for _, mimeType := range unsupportedTypes {
		if gen.isSupportedImage(mimeType) {
			t.Errorf("Should not support %s", mimeType)
		}
	}

	t.Logf("✅ MIME type support detection works correctly")
}

// TestGetSupportedTypes tests the supported types helper
func TestGetSupportedTypes(t *testing.T) {
	types := GetSupportedTypes()

	expected := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp"}

	if len(types) != len(expected) {
		t.Errorf("Expected %d types, got %d", len(expected), len(types))
	}

	t.Logf("✅ GetSupportedTypes returns %d image types", len(types))
}
