package sources

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	sourcesvc "github.com/walens/walens/internal/services/sources"
)

// ListSourcesOperation returns the Huma operation metadata for ListSources.
func ListSourcesOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-sources-list-sources",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/sources/ListSources"),
		Summary:     "List all configured sources",
		Description: "Returns all configured source rows, ordered by name.",
		Tags:        []string{"sources"},
	}
}

// ListSourcesInput describes the request body for ListSources.
type ListSourcesInput struct {
	Body sourcesvc.ListSourcesRequest
}

// ListSourcesOutput describes the response body for ListSources.
type ListSourcesOutput struct {
	Body sourcesvc.ListSourcesResponse
}

// ListSources handles POST /api/v1/sources/ListSources.
// Returns all configured source rows.
func ListSources(ctx context.Context, input *ListSourcesInput, svc *sourcesvc.Service) (*ListSourcesOutput, error) {
	resp, err := svc.ListSources(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &ListSourcesOutput{
		Body: resp,
	}, nil
}
