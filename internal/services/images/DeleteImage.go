package images

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// DeleteImageInput defines the input for DeleteImage.
type DeleteImageInput struct {
	ID dbtypes.UUID `json:"id" doc:"Image ID to delete"`
}

// DeleteImageResponse describes the result of a delete operation.
type DeleteImageResponse struct {
	DeletedLocationCount int64    `json:"deleted_location_count" doc:"Number of image location rows deleted"`
	DeletedThumbnail     bool     `json:"deleted_thumbnail" doc:"Whether the thumbnail row was deleted"`
	DeletedImage         bool     `json:"deleted_image" doc:"Whether the image row was deleted"`
	FailedPaths          []string `json:"failed_paths" doc:"Paths that could not be deleted from disk"`
}

// DeleteImage removes an image and its associated data.
// It attempts to delete all tracked file paths from disk before removing DB records.
// If any file deletions fail, the operation returns partial success with FailedPaths.
func (s *Service) DeleteImage(ctx context.Context, input DeleteImageInput) (*DeleteImageResponse, error) {
	resp := &DeleteImageResponse{
		FailedPaths: []string{},
	}

	// 1. Load the image by ID (404 if missing)
	var img model.Images
	getStmt := SELECT(Images.AllColumns).
		FROM(Images).
		WHERE(Images.ID.EQ(String(input.ID.UUID.String()))).
		LIMIT(1)
	if err := getStmt.QueryContext(ctx, s.db, &img); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("get image for delete: %w", err)
	}

	// 2. Load all image_locations for the image
	locations, err := s.GetImageLocations(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("get image locations: %w", err)
	}

	// 3. Attempt to remove each tracked file path from disk
	for _, loc := range locations {
		if err := os.Remove(loc.Path); err != nil {
			if !os.IsNotExist(err) {
				resp.FailedPaths = append(resp.FailedPaths, loc.Path)
			}
			// os.IsNotExist means file already gone - not a failure
		}
	}

	// 4. Attempt to remove thumbnail file if thumbnail row exists
	thumbnail, thumbErr := s.GetImageThumbnail(ctx, input.ID)
	if thumbErr == nil && thumbnail != nil {
		if err := os.Remove(thumbnail.Path); err != nil {
			if !os.IsNotExist(err) {
				resp.FailedPaths = append(resp.FailedPaths, thumbnail.Path)
			}
		}
		// Note: we still delete the thumbnail DB row even if file removal fails
		// because the DB record is now orphaned anyway
		resp.DeletedThumbnail = true
	} else if thumbErr != nil && !errors.Is(thumbErr, ErrThumbnailNotFound) {
		// Unexpected error when fetching thumbnail
		return nil, fmt.Errorf("check thumbnail: %w", thumbErr)
	}

	// 5. If any file deletions failed, return partial success without deleting DB rows
	// This keeps DB state consistent with actual disk state
	if len(resp.FailedPaths) > 0 {
		return resp, nil
	}

	// 6. All files cleaned (or already absent), delete DB rows in order

	// Delete thumbnail row if it existed
	if resp.DeletedThumbnail {
		thumbDeleteStmt := ImageThumbnails.DELETE().WHERE(ImageThumbnails.ImageID.EQ(String(input.ID.UUID.String())))
		if _, err := thumbDeleteStmt.ExecContext(ctx, s.db); err != nil {
			return nil, fmt.Errorf("delete thumbnail row: %w", err)
		}
	}

	// Delete image_locations rows
	if len(locations) > 0 {
		locDeleteStmt := ImageLocations.DELETE().WHERE(ImageLocations.ImageID.EQ(String(input.ID.UUID.String())))
		if _, err := locDeleteStmt.ExecContext(ctx, s.db); err != nil {
			return nil, fmt.Errorf("delete image locations: %w", err)
		}
		resp.DeletedLocationCount = int64(len(locations))
	}

	// Delete image_assignments rows
	assignDeleteStmt := ImageAssignments.DELETE().WHERE(ImageAssignments.ImageID.EQ(String(input.ID.UUID.String())))
	if _, err := assignDeleteStmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("delete image assignments: %w", err)
	}

	// Delete image_tags rows
	tagsDeleteStmt := ImageTags.DELETE().WHERE(ImageTags.ImageID.EQ(String(input.ID.UUID.String())))
	if _, err := tagsDeleteStmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("delete image tags: %w", err)
	}

	// Finally, delete the image row
	imgDeleteStmt := Images.DELETE().WHERE(Images.ID.EQ(String(input.ID.UUID.String())))
	if _, err := imgDeleteStmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("delete image row: %w", err)
	}
	resp.DeletedImage = true

	// Clean up any orphaned directories (best effort)
	for _, loc := range locations {
		dir := filepath.Dir(loc.Path)
		// Only attempt to remove if it's a device-specific subdirectory
		// and might be empty now
		_ = os.Remove(dir)
	}

	return resp, nil
}
