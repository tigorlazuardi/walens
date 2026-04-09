package sources

import (
	"context"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type ListSourcesRequest struct {
	Search     *string                          `json:"search,omitempty" doc:"Search sources by name"`
	Pagination *dbtypes.CursorPaginationRequest `json:"pagination,omitempty"`
}

type ListSourcesResponse struct {
	Items      []model.Sources                   `json:"items" doc:"List of configured sources."`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
}

// ListSources returns configured source rows.
func (s *Service) ListSources(ctx context.Context, req ListSourcesRequest) (ListSourcesResponse, error) {
	var items []model.Sources
	cond := Bool(true)
	if req.Search != nil && *req.Search != "" {
		pattern := String("%" + *req.Search + "%")
		cond = cond.AND(Sources.Name.LIKE(pattern))
	}

	next := req.Pagination.NextToken()
	prev := req.Pagination.PrevToken()
	isPrev := next == "" && prev != ""
	if next != "" {
		cond = cond.AND(Sources.ID.GT(String(next)))
	}
	if isPrev {
		cond = cond.AND(Sources.ID.LT(String(prev)))
	}

	orderBy, err := req.Pagination.BuildOrderByClause(Sources.AllColumns)
	if err != nil {
		return ListSourcesResponse{}, err
	}
	if len(orderBy) == 0 {
		orderBy = append(orderBy, Sources.Name.ASC())
	}
	if isPrev {
		orderBy = append(orderBy, Sources.ID.DESC())
	} else {
		orderBy = append(orderBy, Sources.ID.ASC())
	}

	limit := req.Pagination.GetLimitOrDefault(20, 100)
	stmt := SELECT(Sources.AllColumns).
		FROM(Sources).
		WHERE(cond).
		ORDER_BY(orderBy...).
		LIMIT(limit + 1).
		OFFSET(req.Pagination.GetOffset())

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		return ListSourcesResponse{}, huma.Error500InternalServerError("failed to list sources", err)
	}
	if len(items) == 0 {
		return ListSourcesResponse{Items: []model.Sources{}}, nil
	}

	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	cursor := &dbtypes.CursorPaginationResponse{}
	if isPrev {
		slices.Reverse(items)
	}
	if hasMore {
		cursor.Next = items[len(items)-1].ID
	}
	if next != "" {
		cursor.Prev = items[0].ID
	}

	return ListSourcesResponse{Items: items, Pagination: cursor}, nil
}
