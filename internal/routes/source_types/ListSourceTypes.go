package source_types

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	sourcetypessvc "github.com/walens/walens/internal/services/source_types"
)

type SourceTypeMetadata = sourcetypessvc.SourceTypeMetadata

// ListSourceTypesOperation returns the Huma operation metadata for ListSourceTypes.
func ListSourceTypesOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "ListSourceTypes",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/source_types/ListSourceTypes"),
		Summary:     "List all registered source types",
		Description: "Returns metadata for all source types registered in the system, including type name, display name, default lookup count, and parameter schema.",
		Tags:        []string{"Source Types"},
	}
}

// ListSourceTypesInput describes the request body for ListSourceTypes.
type ListSourceTypesInput struct {
	Body struct{}
}

// ListSourceTypesOutput describes the response body for ListSourceTypes.
type ListSourceTypesOutput struct {
	Body struct {
		Items []SourceTypeMetadata `json:"items" doc:"List of registered source types."`
	}
}

// ListSourceTypes handles POST /api/v1/source_types/ListSourceTypes.
// Returns metadata for all registered source types.
func ListSourceTypes(ctx context.Context, input *ListSourceTypesInput, svc *sourcetypessvc.Service) (*ListSourceTypesOutput, error) {
	items, err := svc.ListSourceTypes()
	if err != nil {
		if errors.Is(err, sourcetypessvc.ErrRegistryUnavailable) {
			return nil, huma.Error503ServiceUnavailable("source registry unavailable")
		}
		return nil, huma.Error500InternalServerError("failed to list source types", err)
	}

	return &ListSourceTypesOutput{
		Body: struct {
			Items []SourceTypeMetadata `json:"items" doc:"List of registered source types."`
		}{
			Items: items,
		},
	}, nil
}
