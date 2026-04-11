package images

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/db/generated/model"
	"github.com/walens/walens/internal/dbtypes"
	imagesvc "github.com/walens/walens/internal/services/images"
)

// GetImageThumbnailOperation returns the Huma operation metadata for GetImageThumbnail.
func GetImageThumbnailOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "GetImageThumbnail",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/GetImageThumbnail"),
		Summary:     "Get an image thumbnail",
		Description: "Retrieves the thumbnail record for a given image.",
		Tags:        []string{"Images"},
	}
}

// GetImageThumbnailInput describes the request body for GetImageThumbnail.
type GetImageThumbnailInput struct {
	Body struct {
		ImageID dbtypes.UUID `json:"image_id" required:"true" doc:"Image ID"`
	}
}

// GetImageThumbnailOutput describes the response body for GetImageThumbnail.
type GetImageThumbnailOutput struct {
	Body *model.ImageThumbnails
}

// GetImageThumbnail handles POST /api/v1/images/GetImageThumbnail.
// Returns the thumbnail for a given image.
func GetImageThumbnail(ctx context.Context, input *GetImageThumbnailInput, svc *imagesvc.Service) (*GetImageThumbnailOutput, error) {
	resp, err := svc.GetImageThumbnail(ctx, input.Body.ImageID)
	if err != nil {
		if errors.Is(err, imagesvc.ErrThumbnailNotFound) {
			return nil, huma.Error404NotFound("Image thumbnail not found")
		}
		return nil, err
	}

	return &GetImageThumbnailOutput{
		Body: resp,
	}, nil
}
