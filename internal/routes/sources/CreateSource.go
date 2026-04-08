package sources

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	sourcesvc "github.com/walens/walens/internal/services/sources"
)

// CreateSourceOperation returns the Huma operation metadata for CreateSource.
func CreateSourceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-sources-create-source",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/sources/CreateSource"),
		Summary:     "Create a new configured source",
		Description: "Creates a new configured source row. The source_type must be a registered source implementation. Params are validated against the source implementation's schema.",
		Tags:        []string{"sources"},
	}
}

// CreateSourceInput describes the request body for CreateSource.
type CreateSourceInput struct {
	Body sourcesvc.CreateSourceInput
}

// CreateSourceOutput describes the response body for CreateSource.
type CreateSourceOutput struct {
	Body sourcesvc.SourceRow
}

// CreateSource handles POST /api/v1/sources/CreateSource.
// Creates a new configured source row.
func CreateSource(ctx context.Context, input *CreateSourceInput, svc *sourcesvc.Service) (*CreateSourceOutput, error) {
	src, err := svc.CreateSource(ctx, &input.Body)
	if err != nil {
		if errors.Is(err, sourcesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, sourcesvc.ErrRegistryUnavailable) {
			return nil, huma.Error503ServiceUnavailable("source registry unavailable")
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
		return nil, huma.Error500InternalServerError("failed to create source", err)
	}

	return &CreateSourceOutput{
		Body: *src,
	}, nil
}
