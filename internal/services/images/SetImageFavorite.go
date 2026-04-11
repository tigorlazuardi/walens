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

// SetImageFavoriteInput defines the input for SetImageFavorite.
type SetImageFavoriteInput struct {
	ID         dbtypes.UUID `json:"id" required:"true" doc:"Image ID"`
	IsFavorite bool         `json:"is_favorite" required:"true" doc:"Whether the image should be marked as favorite"`
}

// SetImageFavoriteOutput defines the output for SetImageFavorite.
type SetImageFavoriteOutput struct {
	Image *model.Images `json:"image"`
}

// SetImageFavorite updates the is_favorite flag and updated_at timestamp for an image.
func (s *Service) SetImageFavorite(ctx context.Context, input SetImageFavoriteInput) (*SetImageFavoriteOutput, error) {
	now := dbtypes.NewUnixMilliTimeNow()

	// First verify the image exists
	var img model.Images
	checkStmt := SELECT(Images.AllColumns).
		FROM(Images).
		WHERE(Images.ID.EQ(String(input.ID.UUID.String()))).
		LIMIT(1)
	if err := checkStmt.QueryContext(ctx, s.db, &img); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("check image exists: %w", err)
	}

	// Update the favorite flag and updated_at
	updateStmt := Images.UPDATE(Images.IsFavorite, Images.UpdatedAt).
		WHERE(Images.ID.EQ(String(input.ID.UUID.String()))).
		MODEL(model.Images{
			IsFavorite: dbtypes.BoolInt(input.IsFavorite),
			UpdatedAt:  now,
		})
	if _, err := updateStmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("set image favorite: %w", err)
	}

	// Reload and return the updated image
	updatedImg, err := s.GetImage(ctx, GetImageInput{ID: input.ID})
	if err != nil {
		return nil, fmt.Errorf("reload image: %w", err)
	}

	return &SetImageFavoriteOutput{Image: updatedImg}, nil
}
