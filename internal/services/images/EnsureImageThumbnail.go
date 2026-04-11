package images

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// EnsureImageThumbnailRequest defines the input for ensuring a thumbnail record.
type EnsureImageThumbnailRequest struct {
	ImageID       dbtypes.UUID `json:"image_id" doc:"Reference to source image"`
	Path          string       `json:"path" doc:"Absolute filesystem path to the thumbnail"`
	Width         int64        `json:"width" doc:"Thumbnail width in pixels"`
	Height        int64        `json:"height" doc:"Thumbnail height in pixels"`
	FileSizeBytes *int64       `json:"file_size_bytes" doc:"Thumbnail file size in bytes"`
}

// EnsureImageThumbnail creates or updates a thumbnail record for an image.
// If a thumbnail already exists for this image, it updates the existing row
// (keeping the same row id) instead of inserting a new one.
func (s *Service) EnsureImageThumbnail(ctx context.Context, req EnsureImageThumbnailRequest) (*model.ImageThumbnails, error) {
	// Check if thumbnail already exists for this image
	existing, err := s.GetImageThumbnail(ctx, req.ImageID)
	if err != nil && !errors.Is(err, ErrThumbnailNotFound) {
		return nil, fmt.Errorf("check existing thumbnail: %w", err)
	}

	now := dbtypes.NewUnixMilliTimeNow()

	if errors.Is(err, ErrThumbnailNotFound) {
		// No existing thumbnail - create new one
		id := dbtypes.MustNewUUIDV7()
		row := model.ImageThumbnails{
			ID:            id,
			ImageID:       req.ImageID,
			Path:          req.Path,
			Width:         req.Width,
			Height:        req.Height,
			FileSizeBytes: req.FileSizeBytes,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		stmt := ImageThumbnails.INSERT(
			ImageThumbnails.ID, ImageThumbnails.ImageID, ImageThumbnails.Path,
			ImageThumbnails.Width, ImageThumbnails.Height, ImageThumbnails.FileSizeBytes,
			ImageThumbnails.CreatedAt, ImageThumbnails.UpdatedAt,
		).MODEL(row)

		if _, err := stmt.ExecContext(ctx, s.db); err != nil {
			return nil, fmt.Errorf("create image thumbnail: %w", err)
		}

		return &row, nil
	}

	// Thumbnail exists - update the existing row (keep same ID)
	updated := existing
	updated.Path = req.Path
	updated.Width = req.Width
	updated.Height = req.Height
	updated.FileSizeBytes = req.FileSizeBytes
	updated.UpdatedAt = now

	stmt := ImageThumbnails.UPDATE(
		ImageThumbnails.Path,
		ImageThumbnails.Width,
		ImageThumbnails.Height,
		ImageThumbnails.FileSizeBytes,
		ImageThumbnails.UpdatedAt,
	).MODEL(updated).WHERE(
		ImageThumbnails.ID.EQ(String(existing.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrThumbnailNotFound
		}
		return nil, fmt.Errorf("update image thumbnail: %w", err)
	}

	return updated, nil
}
