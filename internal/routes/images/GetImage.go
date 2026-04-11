package images

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/db/generated/model"
	imagesvc "github.com/walens/walens/internal/services/images"
)

// GetImageOperation returns the Huma operation metadata for GetImage.
func GetImageOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "GetImage",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/GetImage"),
		Summary:     "Get a single image by ID",
		Description: "Retrieves a single image by its unique identifier.",
		Tags:        []string{"Images"},
	}
}

// GetImageInput describes the request body for GetImage.
type GetImageInput struct {
	Body imagesvc.GetImageInput
}

// GetImageOutput describes the response body for GetImage.
type GetImageOutput struct {
	Body *model.Images
}

// GetImage handles POST /api/v1/images/GetImage.
// Returns a single image by ID.
func GetImage(ctx context.Context, input *GetImageInput, svc *imagesvc.Service) (*GetImageOutput, error) {
	resp, err := svc.GetImage(ctx, input.Body)
	if err != nil {
		if errors.Is(err, imagesvc.ErrImageNotFound) {
			return nil, huma.Error404NotFound("Image not found")
		}
		return nil, err
	}

	return &GetImageOutput{
		Body: resp,
	}, nil
}
