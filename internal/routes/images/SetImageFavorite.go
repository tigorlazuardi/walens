package images

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	imagesvc "github.com/walens/walens/internal/services/images"
)

// SetImageFavoriteOperation returns the Huma operation metadata for SetImageFavorite.
func SetImageFavoriteOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "SetImageFavorite",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/SetImageFavorite"),
		Summary:     "Set image favorite status",
		Description: "Updates the is_favorite flag on an image.",
		Tags:        []string{"Images"},
	}
}

// SetImageFavoriteInput describes the request body for SetImageFavorite.
type SetImageFavoriteInput struct {
	Body imagesvc.SetImageFavoriteInput
}

// SetImageFavoriteOutput describes the response body for SetImageFavorite.
type SetImageFavoriteOutput struct {
	Body imagesvc.SetImageFavoriteOutput
}

// SetImageFavorite handles POST /api/v1/images/SetImageFavorite.
func SetImageFavorite(ctx context.Context, input *SetImageFavoriteInput, svc *imagesvc.Service) (*SetImageFavoriteOutput, error) {
	resp, err := svc.SetImageFavorite(ctx, input.Body)
	if err != nil {
		if errors.Is(err, imagesvc.ErrImageNotFound) {
			return nil, huma.Error404NotFound("Image not found")
		}
		return nil, err
	}

	return &SetImageFavoriteOutput{
		Body: *resp,
	}, nil
}
