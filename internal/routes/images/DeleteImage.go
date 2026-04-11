package images

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	imagesvc "github.com/walens/walens/internal/services/images"
)

// DeleteImageOperation returns the Huma operation metadata for DeleteImage.
func DeleteImageOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "DeleteImage",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/DeleteImage"),
		Summary:     "Delete an image",
		Description: "Deletes an image and its associated files and database records.",
		Tags:        []string{"Images"},
	}
}

// DeleteImageInput describes the request body for DeleteImage.
type DeleteImageInput struct {
	Body imagesvc.DeleteImageInput
}

// DeleteImageOutput describes the response body for DeleteImage.
type DeleteImageOutput struct {
	Body imagesvc.DeleteImageResponse
}

// DeleteImage handles POST /api/v1/images/DeleteImage.
func DeleteImage(ctx context.Context, input *DeleteImageInput, svc *imagesvc.Service) (*DeleteImageOutput, error) {
	resp, err := svc.DeleteImage(ctx, input.Body)
	if err != nil {
		if errors.Is(err, imagesvc.ErrImageNotFound) {
			return nil, huma.Error404NotFound("Image not found")
		}
		return nil, err
	}

	return &DeleteImageOutput{
		Body: *resp,
	}, nil
}
