package sources

import (
	"context"
	"errors"
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
	Body sourcesvc.UpdateSourceInput
}

// UpdateSourceOutput describes the response body for UpdateSource.
type UpdateSourceOutput struct {
	Body sourcesvc.SourceRow
}

// UpdateSource handles POST /api/v1/sources/UpdateSource.
// Updates an existing configured source with full-object update semantics.
func UpdateSource(ctx context.Context, input *UpdateSourceInput, svc *sourcesvc.Service) (*UpdateSourceOutput, error) {
	src, err := svc.UpdateSource(ctx, &input.Body)
	if err != nil {
		if errors.Is(err, sourcesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, sourcesvc.ErrRegistryUnavailable) {
			return nil, huma.Error503ServiceUnavailable("source registry unavailable")
		}
		if errors.Is(err, sourcesvc.ErrSourceNotFound) {
			return nil, huma.Error404NotFound("source not found")
		}
		if errors.Is(err, sourcesvc.ErrDuplicateSourceName) {
			return nil, huma.Error409Conflict("source with this name already exists")
		}
		if errors.Is(err, sourcesvc.ErrInvalidSourceType) {
			return nil, huma.Error400BadRequest("invalid source type: not registered")
		}
		if errors.Is(err, sourcesvc.ErrInvalidParams) {
			return nil, huma.Error400BadRequest("invalid params for source type", err)
		}
		return nil, huma.Error500InternalServerError("failed to update source", err)
	}

	return &UpdateSourceOutput{
		Body: *src,
	}, nil
}
