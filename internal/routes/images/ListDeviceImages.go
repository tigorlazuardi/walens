package images

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	imagesvc "github.com/walens/walens/internal/services/images"
)

// ListDeviceImagesOperation returns the Huma operation metadata for ListDeviceImages.
func ListDeviceImagesOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "ListDeviceImages",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/ListDeviceImages"),
		Summary:     "List images for a specific device",
		Description: "Returns images that match a specific device according to the device's subscription, dimension, filesize, and adult constraints.",
		Tags:        []string{"Images"},
	}
}

// ListDeviceImagesInput describes the request body for ListDeviceImages.
type ListDeviceImagesInput struct {
	Body imagesvc.ListDeviceImagesRequest
}

// ListDeviceImagesOutput describes the response body for ListDeviceImages.
type ListDeviceImagesOutput struct {
	Body imagesvc.ListDeviceImagesResponse
}

// ListDeviceImages handles POST /api/v1/images/ListDeviceImages.
// Returns images matching the device constraints and optional filters.
func ListDeviceImages(ctx context.Context, input *ListDeviceImagesInput, svc *imagesvc.Service) (*ListDeviceImagesOutput, error) {
	resp, err := svc.ListDeviceImages(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &ListDeviceImagesOutput{
		Body: resp,
	}, nil
}
