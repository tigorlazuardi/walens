package sources

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
	sourcesvc "github.com/walens/walens/internal/services/sources"
)

// GetSourceOperation returns the Huma operation metadata for GetSource.
func GetSourceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-sources-get-source",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/sources/GetSource"),
		Summary:     "Get a source by ID",
		Description: "Returns a single configured source row by its ID.",
		Tags:        []string{"sources"},
	}
}

// GetSourceInput describes the request body for GetSource.
type GetSourceInput struct {
	Body struct {
		ID string `json:"id" doc:"Unique source identifier (UUIDv7)."`
	}
}

// GetSourceOutput describes the response body for GetSource.
type GetSourceOutput struct {
	Body sourcesvc.SourceRow
}

// GetSource handles POST /api/v1/sources/GetSource.
// Returns a single configured source row by ID.
func GetSource(ctx context.Context, input *GetSourceInput, svc *sourcesvc.Service) (*GetSourceOutput, error) {
	id, err := dbtypes.NewUUIDFromString(input.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid source ID format", err)
	}

	src, err := svc.GetSource(ctx, id)
	if err != nil {
		if errors.Is(err, sourcesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, sourcesvc.ErrSourceNotFound) {
			return nil, huma.Error404NotFound("source not found")
		}
		return nil, huma.Error500InternalServerError("failed to get source", err)
	}

	return &GetSourceOutput{
		Body: *src,
	}, nil
}
