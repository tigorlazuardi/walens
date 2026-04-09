package sources

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	sourcesvc "github.com/walens/walens/internal/services/sources"
)

// UpdateSourceOperation returns the Huma operation metadata for UpdateSource.
func UpdateSourceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-sources-update-source",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/sources/UpdateSource"),
		Summary:     "Update an existing source",
		Description: "Updates an existing configured source row with full-object update semantics. All fields are required. The source_type must be a registered source implementation and params are validated against it.",
		Tags:        []string{"sources"},
	}
}

// UpdateSourceInput describes the request body for UpdateSource.
type UpdateSourceInput struct {
	Body sourcesvc.UpdateSourceRequest
}

// UpdateSourceOutput describes the response body for UpdateSource.
type UpdateSourceOutput struct {
	Body sourcesvc.SourceRow
}

// UpdateSource handles POST /api/v1/sources/UpdateSource.
// Updates an existing configured source with full-object update semantics.
func UpdateSource(ctx context.Context, input *UpdateSourceInput, svc *sourcesvc.Service) (*UpdateSourceOutput, error) {
	src, err := svc.UpdateSource(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &UpdateSourceOutput{
		Body: src,
	}, nil
}
