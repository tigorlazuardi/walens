package storage

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/walens/walens/internal/dbtypes"
)

// createTestImage creates a test JPEG image at the given path.
func createTestImage(t *testing.T, path string, width, height int) {
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	defer file.Close()

	// Create a simple colored image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a gradient
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}

	if err := jpeg.Encode(file, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
}

// TestGenerateThumbnail_Basic tests that thumbnail generation works.
func TestGenerateThumbnail_Basic(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "walens-thumbnail-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create storage service
	svc := NewService(Config{BaseDir: tempDir})

	// Create a test source image
	sourcePath := filepath.Join(tempDir, "source.jpg")
	createTestImage(t, sourcePath, 1920, 1080)

	// Generate thumbnail
	imageID := dbtypes.MustNewUUIDV7()
	result, err := svc.GenerateThumbnail(sourcePath, imageID)
	if err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Check result
	if result.Path == "" {
		t.Error("expected non-empty path")
	}
	if result.Width <= 0 {
		t.Errorf("expected positive width, got %d", result.Width)
	}
	if result.Height <= 0 {
		t.Errorf("expected positive height, got %d", result.Height)
	}
	if result.FileSizeBytes <= 0 {
		t.Errorf("expected positive file size, got %d", result.FileSizeBytes)
	}

	// Verify file exists
	if !svc.FileExists(result.Path) {
		t.Errorf("expected thumbnail file to exist at %s", result.Path)
	}

	// Verify path format
	expectedPath := filepath.Join(tempDir, "thumbnails", imageID.UUID.String()+".jpg")
	if result.Path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, result.Path)
	}
}

// TestGenerateThumbnail_AspectRatio tests that aspect ratio is preserved.
func TestGenerateThumbnail_AspectRatio(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "walens-thumbnail-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc := NewService(Config{BaseDir: tempDir})

	// Test portrait image (1080x1920 - 9:16 ratio)
	sourcePath := filepath.Join(tempDir, "portrait.jpg")
	createTestImage(t, sourcePath, 1080, 1920)

	imageID := dbtypes.MustNewUUIDV7()
	result, err := svc.GenerateThumbnail(sourcePath, imageID)
	if err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Portrait image should have height > width after scaling to fit 512x512
	if result.Height <= result.Width {
		t.Errorf("expected portrait thumbnail (height > width), got width=%d height=%d", result.Width, result.Height)
	}

	// Test landscape image (1920x1080 - 16:9 ratio)
	sourcePath2 := filepath.Join(tempDir, "landscape.jpg")
	createTestImage(t, sourcePath2, 1920, 1080)

	imageID2 := dbtypes.MustNewUUIDV7()
	result2, err := svc.GenerateThumbnail(sourcePath2, imageID2)
	if err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Landscape image should have width > height
	if result2.Width <= result2.Height {
		t.Errorf("expected landscape thumbnail (width > height), got width=%d height=%d", result2.Width, result2.Height)
	}
}

// TestGenerateThumbnail_MaxDimensions tests that thumbnail fits within 512x512.
func TestGenerateThumbnail_MaxDimensions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "walens-thumbnail-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc := NewService(Config{BaseDir: tempDir})

	// Create a large image
	sourcePath := filepath.Join(tempDir, "large.jpg")
	createTestImage(t, sourcePath, 4000, 3000)

	imageID := dbtypes.MustNewUUIDV7()
	result, err := svc.GenerateThumbnail(sourcePath, imageID)
	if err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Both dimensions should be <= 512
	if result.Width > 512 {
		t.Errorf("expected width <= 512, got %d", result.Width)
	}
	if result.Height > 512 {
		t.Errorf("expected height <= 512, got %d", result.Height)
	}
}

// TestGenerateThumbnail_Overwrite tests that generating a thumbnail overwrites existing one.
func TestGenerateThumbnail_Overwrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "walens-thumbnail-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc := NewService(Config{BaseDir: tempDir})

	// Create a test source image
	sourcePath := filepath.Join(tempDir, "source.jpg")
	createTestImage(t, sourcePath, 1920, 1080)

	imageID := dbtypes.MustNewUUIDV7()

	// Generate first thumbnail
	result1, err := svc.GenerateThumbnail(sourcePath, imageID)
	if err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Generate second thumbnail (should overwrite)
	result2, err := svc.GenerateThumbnail(sourcePath, imageID)
	if err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	// Paths should be the same
	if result1.Path != result2.Path {
		t.Errorf("expected same path, got %s and %s", result1.Path, result2.Path)
	}

	// File should still exist
	if !svc.FileExists(result2.Path) {
		t.Errorf("expected thumbnail file to exist at %s", result2.Path)
	}
}

// TestThumbnailPath tests that ThumbnailPath returns correct path.
func TestThumbnailPath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "walens-thumbnail-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc := NewService(Config{BaseDir: tempDir})

	imageID := dbtypes.MustNewUUIDV7()
	path := svc.ThumbnailPath(imageID)

	expected := filepath.Join(tempDir, "thumbnails", imageID.UUID.String()+".jpg")
	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

// TestThumbnailDir tests that ThumbnailDir returns correct directory.
func TestThumbnailDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "walens-thumbnail-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	svc := NewService(Config{BaseDir: tempDir})

	dir := svc.ThumbnailDir()
	expected := filepath.Join(tempDir, "thumbnails")
	if dir != expected {
		t.Errorf("expected dir %s, got %s", expected, dir)
	}
}
