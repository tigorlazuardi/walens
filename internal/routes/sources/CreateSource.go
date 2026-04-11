package sources

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	sourcesvc "github.com/walens/walens/internal/services/sources"
)

// CreateSourceOperation returns the Huma operation metadata for CreateSource.
func CreateSourceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "CreateSource",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/sources/CreateSource"),
		Summary:     "Create a new configured source",
		Description: "Creates a new configured source row. The source_type must be a registered source implementation. Params are validated against the source implementation's schema.",
		Tags:        []string{"Sources"},
	}
}

// CreateSourceInput describes the request body for CreateSource.
type CreateSourceInput struct {
	Body sourcesvc.CreateSourceRequest
}

// CreateSourceOutput describes the response body for CreateSource.
type CreateSourceOutput struct {
	Body sourcesvc.SourceRow
}

// CreateSource handles POST /api/v1/sources/CreateSource.
// Creates a new configured source row.
func CreateSource(ctx context.Context, input *CreateSourceInput, svc *sourcesvc.Service) (*CreateSourceOutput, error) {
	src, err := svc.CreateSource(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &CreateSourceOutput{
		Body: src,
	}, nil
}
