package sources

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
	sourcesvc "github.com/walens/walens/internal/services/sources"
)

// DeleteSourceOperation returns the Huma operation metadata for DeleteSource.
func DeleteSourceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-sources-delete-source",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/sources/DeleteSource"),
		Summary:     "Delete a source by ID",
		Description: "Deletes a configured source row by its ID. This also cascades to delete associated source_schedules and device_source_subscriptions.",
		Tags:        []string{"sources"},
	}
}

// DeleteSourceInput describes the request body for DeleteSource.
type DeleteSourceInput struct {
	Body struct {
		ID string `json:"id" doc:"Unique source identifier (UUIDv7)."`
	}
}

// DeleteSourceOutput describes the response body for DeleteSource.
type DeleteSourceOutput struct {
	Body struct{}
}

// DeleteSource handles POST /api/v1/sources/DeleteSource.
// Deletes a configured source row by ID.
func DeleteSource(ctx context.Context, input *DeleteSourceInput, svc *sourcesvc.Service) (*DeleteSourceOutput, error) {
	id, err := dbtypes.NewUUIDFromString(input.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid source ID format", err)
	}

	err = svc.DeleteSource(ctx, id)
	if err != nil {
		if errors.Is(err, sourcesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, sourcesvc.ErrSourceNotFound) {
			return nil, huma.Error404NotFound("source not found")
		}
		return nil, huma.Error500InternalServerError("failed to delete source", err)
	}

	return &DeleteSourceOutput{
		Body: struct{}{},
	}, nil
}
