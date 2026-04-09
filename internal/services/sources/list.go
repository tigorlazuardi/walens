package sources

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
)

type ListSourcesRequest struct{}

type ListSourcesResponse struct {
	Items []model.Sources `json:"items" doc:"List of configured sources."`
}

// ListSources returns all configured source rows.
func (s *Service) ListSources(ctx context.Context, _ ListSourcesRequest) (ListSourcesResponse, error) {
	var results []model.Sources
	stmt := SELECT(Sources.AllColumns).
		FROM(Sources).
		ORDER_BY(Sources.Name.ASC())

	if err := stmt.QueryContext(ctx, s.db, &results); err != nil {
		return ListSourcesResponse{}, huma.Error500InternalServerError("failed to list sources", err)
	}
	if results == nil {
		return ListSourcesResponse{Items: []model.Sources{}}, nil
	}

	return ListSourcesResponse{Items: results}, nil
}
