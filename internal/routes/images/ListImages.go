package images

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	imagesvc "github.com/walens/walens/internal/services/images"
)

// ListImagesOperation returns the Huma operation metadata for ListImages.
func ListImagesOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "ListImages",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/images/ListImages"),
		Summary:     "List all images",
		Description: "Returns all images matching the provided filters.",
		Tags:        []string{"Images"},
	}
}

// ListImagesInput describes the request body for ListImages.
type ListImagesInput struct {
	Body imagesvc.ListImagesRequest
}

// ListImagesOutput describes the response body for ListImages.
type ListImagesOutput struct {
	Body imagesvc.ListImagesResponse
}

// ListImages handles POST /api/v1/images/ListImages.
// Returns images matching the provided filters.
func ListImages(ctx context.Context, input *ListImagesInput, svc *imagesvc.Service) (*ListImagesOutput, error) {
	resp, err := svc.ListImages(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &ListImagesOutput{
		Body: resp,
	}, nil
}
