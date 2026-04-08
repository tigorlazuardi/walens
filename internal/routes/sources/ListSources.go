package sources

import (
	"context"
	"errors"
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
	Body struct{}
}

// ListSourcesOutput describes the response body for ListSources.
type ListSourcesOutput struct {
	Body struct {
		Items []sourcesvc.SourceRow `json:"items" doc:"List of configured sources."`
	}
}

// ListSources handles POST /api/v1/sources/ListSources.
// Returns all configured source rows.
func ListSources(ctx context.Context, input *ListSourcesInput, svc *sourcesvc.Service) (*ListSourcesOutput, error) {
	items, err := svc.ListSources(ctx)
	if err != nil {
		if errors.Is(err, sourcesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		return nil, huma.Error500InternalServerError("failed to list sources", err)
	}

	return &ListSourcesOutput{
		Body: struct {
			Items []sourcesvc.SourceRow `json:"items" doc:"List of configured sources."`
		}{
			Items: items,
		},
	}, nil
}
