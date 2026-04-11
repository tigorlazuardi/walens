package images

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	imagesvc "github.com/walens/walens/internal/services/images"
)

// BlacklistImageOperation returns the Huma operation metadata for BlacklistImage.
func BlacklistImageOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "BlacklistImage",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/BlacklistImage"),
		Summary:     "Blacklist an image",
		Description: "Adds an image to the blacklist, preventing it from being downloaded again.",
		Tags:        []string{"Images"},
	}
}

// BlacklistImageInput describes the request body for BlacklistImage.
type BlacklistImageInput struct {
	Body imagesvc.BlacklistImageInput
}

// BlacklistImageOutput describes the response body for BlacklistImage.
type BlacklistImageOutput struct {
	Body imagesvc.BlacklistImageOutput
}

// BlacklistImage handles POST /api/v1/images/BlacklistImage.
func BlacklistImage(ctx context.Context, input *BlacklistImageInput, svc *imagesvc.Service) (*BlacklistImageOutput, error) {
	resp, err := svc.BlacklistImage(ctx, input.Body)
	if err != nil {
		if errors.Is(err, imagesvc.ErrImageNotFound) {
			return nil, huma.Error404NotFound("Image not found")
		}
		return nil, err
	}

	return &BlacklistImageOutput{
		Body: *resp,
	}, nil
}
