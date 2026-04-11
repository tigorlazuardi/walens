package source_types

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	sourcetypessvc "github.com/walens/walens/internal/services/source_types"
)

type GetSourceTypeBody = sourcetypessvc.SourceTypeMetadata

// GetSourceTypeOperation returns the Huma operation metadata for GetSourceType.
func GetSourceTypeOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "GetSourceType",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/source_types/GetSourceType"),
		Summary:     "Get a source type by name",
		Description: "Returns metadata for a specific source type including type name, display name, default lookup count, and parameter schema.",
		Tags:        []string{"Source Types"},
	}
}

// GetSourceTypeInput describes the request body for GetSourceType.
type GetSourceTypeInput struct {
	Body struct {
		TypeName string `json:"type_name" required:"true" doc:"The source type implementation name (e.g., 'booru')."`
	}
}

// GetSourceTypeOutput describes the response body for GetSourceType.
type GetSourceTypeOutput struct {
	Body GetSourceTypeBody
}

// GetSourceType handles POST /api/v1/source_types/GetSourceType.
// Returns metadata for a specific source type by name.
func GetSourceType(ctx context.Context, input *GetSourceTypeInput, svc *sourcetypessvc.Service) (*GetSourceTypeOutput, error) {
	metadata, err := svc.GetSourceType(input.Body.TypeName)
	if err != nil {
		if errors.Is(err, sourcetypessvc.ErrRegistryUnavailable) {
			return nil, huma.Error503ServiceUnavailable("source registry unavailable")
		}
		if errors.Is(err, sourcetypessvc.ErrSourceTypeNotFound) {
			return nil, huma.Error404NotFound("source type not found: " + input.Body.TypeName)
		}
		return nil, huma.Error500InternalServerError("failed to get source type", err)
	}
	return &GetSourceTypeOutput{
		Body: *metadata,
	}, nil
}
