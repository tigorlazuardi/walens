// Package storage provides file operations for the runner.
package storage

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/walens/walens/internal/dbtypes"

	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/png"
)

// ThumbnailResult contains the outcome of thumbnail generation.
type ThumbnailResult struct {
	Path          string
	Width         int64
	Height        int64
	FileSizeBytes int64
}

// ThumbnailMaxWidth is the maximum width for thumbnails.
const ThumbnailMaxWidth = 512

// ThumbnailMaxHeight is the maximum height for thumbnails.
const ThumbnailMaxHeight = 512

// ThumbnailTargetSize is the target file size in bytes (~40KB).
const ThumbnailTargetSize = 40 * 1024

// ThumbnailDir returns the directory path for thumbnail storage.
func (s *Service) ThumbnailDir() string {
	return filepath.Join(s.cfg.BaseDir, "thumbnails")
}

// ThumbnailPath returns the expected thumbnail path for an image.
func (s *Service) ThumbnailPath(imageID dbtypes.UUID) string {
	return filepath.Join(s.ThumbnailDir(), fmt.Sprintf("%s.jpg", imageID.UUID.String()))
}

// GenerateThumbnail creates a thumbnail from the source image file.
// The thumbnail is always JPEG format, preserves aspect ratio, and fits within 512x512.
// It attempts to target around 40KB file size via quality adjustment.
// Supports JPEG, PNG, GIF, and WebP source images.
func (s *Service) GenerateThumbnail(sourcePath string, imageID dbtypes.UUID) (ThumbnailResult, error) {
	// Open the source image
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return ThumbnailResult{}, fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	// Decode the image - uses registered decoders (jpeg, png, gif, webp via blank imports)
	img, _, err := image.Decode(srcFile)
	if err != nil {
		return ThumbnailResult{}, fmt.Errorf("decode image: %w", err)
	}

	// Calculate target dimensions maintaining aspect ratio
	bounds := img.Bounds()
	srcWidth := int64(bounds.Dx())
	srcHeight := int64(bounds.Dy())

	// Handle zero-dimension edge case
	if srcWidth == 0 || srcHeight == 0 {
		return ThumbnailResult{}, fmt.Errorf("image has zero dimensions")
	}

	// Calculate scaled dimensions to fit within 512x512
	var thumbWidth, thumbHeight int64
	if srcWidth > srcHeight {
		// Landscape - scale by width
		thumbWidth = ThumbnailMaxWidth
		thumbHeight = (srcHeight * ThumbnailMaxWidth) / srcWidth
	} else {
		// Portrait or square - scale by height
		thumbHeight = ThumbnailMaxHeight
		thumbWidth = (srcWidth * ThumbnailMaxHeight) / srcHeight
	}

	// Ensure minimum dimensions of 1
	if thumbWidth < 1 {
		thumbWidth = 1
	}
	if thumbHeight < 1 {
		thumbHeight = 1
	}

	// Scale the image
	scaled := scaleImage(img, int(thumbWidth), int(thumbHeight))

	// Ensure thumbnail directory exists
	thumbDir := s.ThumbnailDir()
	if err := s.EnsureDir(thumbDir); err != nil {
		return ThumbnailResult{}, fmt.Errorf("ensure thumbnail dir: %w", err)
	}

	// Generate thumbnail path
	thumbPath := s.ThumbnailPath(imageID)

	// Encode to JPEG with quality loop targeting ~40KB
	if err := encodeJPEGWithQuality(scaled, thumbPath); err != nil {
		return ThumbnailResult{}, fmt.Errorf("encode thumbnail: %w", err)
	}

	// Get file size
	stat, err := os.Stat(thumbPath)
	if err != nil {
		return ThumbnailResult{}, fmt.Errorf("stat thumbnail: %w", err)
	}

	return ThumbnailResult{
		Path:          thumbPath,
		Width:         thumbWidth,
		Height:        thumbHeight,
		FileSizeBytes: stat.Size(),
	}, nil
}

// scaleImage scales an image to the specified dimensions using nearest neighbor
// (fast and appropriate for thumbnails).
func scaleImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// Calculate scaling ratios
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Skip if source dimensions are zero
	if srcWidth == 0 || srcHeight == 0 {
		return dst
	}

	// Simple nearest-neighbor scaling
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Map target pixel to source pixel
			srcX := (x * srcWidth) / width
			srcY := (y * srcHeight) / height

			// Ensure we don't exceed source bounds
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}
			if srcY >= srcHeight {
				srcY = srcHeight - 1
			}

			dst.Set(x, y, src.At(srcX+srcBounds.Min.X, srcY+srcBounds.Min.Y))
		}
	}

	return dst
}

// encodeJPEGWithQuality encodes an image to JPEG, adjusting quality to
// try to get close to the target file size.
func encodeJPEGWithQuality(img image.Image, path string) error {
	// Start with high quality and reduce if needed
	quality := 85

	for quality >= 50 {
		tmpPath := path + ".tmp"

		file, err := os.Create(tmpPath)
		if err != nil {
			return fmt.Errorf("create temp file: %w", err)
		}

		err = jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
		file.Close()

		if err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("encode jpeg: %w", err)
		}

		// Check file size
		stat, err := os.Stat(tmpPath)
		if err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("stat temp file: %w", err)
		}

		if stat.Size() <= ThumbnailTargetSize || quality <= 50 {
			// Size is acceptable or we've hit minimum quality
			os.Rename(tmpPath, path)
			return nil
		}

		os.Remove(tmpPath)
		quality -= 10
	}

	// Final encode at minimum quality
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create final file: %w", err)
	}
	defer file.Close()

	return jpeg.Encode(file, img, &jpeg.Options{Quality: quality})
}

// NewUUIDV7 generates a new UUIDv7.
func NewUUIDV7() (uuid.UUID, error) {
	return uuid.NewV7()
}
