package sources

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	sourcesvc "github.com/walens/walens/internal/services/sources"
)

// DeleteSourceOperation returns the Huma operation metadata for DeleteSource.
func DeleteSourceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID:   "DeleteSource",
		Method:        "POST",
		Path:          path.Join(basePath, "/api/v1/sources/DeleteSource"),
		DefaultStatus: 204,
		Summary:       "Delete a source by ID",
		Description:   "Deletes a configured source row by its ID. This also cascades to delete associated source_schedules and device_source_subscriptions.",
		Tags:          []string{"Sources"},
	}
}

// DeleteSourceInput describes the request body for DeleteSource.
type DeleteSourceInput struct {
	Body sourcesvc.DeleteSourceRequest
}

// DeleteSource handles POST /api/v1/sources/DeleteSource.
// Deletes a configured source row by ID.
func DeleteSource(ctx context.Context, input *DeleteSourceInput, svc *sourcesvc.Service) (*struct{}, error) {
	_, err := svc.DeleteSource(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
