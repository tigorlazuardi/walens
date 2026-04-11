package sources

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	sourcesvc "github.com/walens/walens/internal/services/sources"
)

// GetSourceOperation returns the Huma operation metadata for GetSource.
func GetSourceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "GetSource",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/sources/GetSource"),
		Summary:     "Get a source by ID",
		Description: "Returns a single configured source row by its ID.",
		Tags:        []string{"Sources"},
	}
}

// GetSourceInput describes the request body for GetSource.
type GetSourceInput struct {
	Body sourcesvc.GetSourceRequest
}

// GetSourceOutput describes the response body for GetSource.
type GetSourceOutput struct {
	Body sourcesvc.SourceRow
}

// GetSource handles POST /api/v1/sources/GetSource.
// Returns a single configured source row by ID.
func GetSource(ctx context.Context, input *GetSourceInput, svc *sourcesvc.Service) (*GetSourceOutput, error) {
	src, err := svc.GetSource(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &GetSourceOutput{
		Body: src,
	}, nil
}
