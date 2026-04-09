package source_schedules

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type ListSchedulesRequest struct {
	Pagination *dbtypes.CursorPaginationRequest `json:"pagination,omitempty"`
}

type ListSchedulesResponse struct {
	Items      []model.SourceSchedules           `json:"items" doc:"List of source schedules."`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
	Total      int64                             `json:"total" doc:"Total count of schedules matching filters, independent of pagination"`
}

// ListSchedules returns all source schedules ordered by creation time.
func (s *Service) ListSchedules(ctx context.Context, req ListSchedulesRequest) (ListSchedulesResponse, error) {
	var items []model.SourceSchedules
	baseCond := Bool(true)

	// Get total count before pagination filters
	total, err := s.countSchedules(ctx, baseCond)
	if err != nil {
		return ListSchedulesResponse{}, err
	}

	// Pagination - build condition with cursor filters
	cond := baseCond
	next := req.Pagination.NextToken()
	prev := req.Pagination.PrevToken()
	isPrev := next == "" && prev != ""
	if next != "" {
		cond = cond.AND(SourceSchedules.ID.GT(String(next)))
	}
	if isPrev {
		cond = cond.AND(SourceSchedules.ID.LT(String(prev)))
	}

	orderBy, err := req.Pagination.BuildOrderByClause(SourceSchedules.AllColumns)
	if err != nil {
		return ListSchedulesResponse{}, err
	}
	if len(orderBy) == 0 {
		orderBy = append(orderBy, SourceSchedules.CreatedAt.ASC())
	}
	if isPrev {
		orderBy = append(orderBy, SourceSchedules.ID.DESC())
	} else {
		orderBy = append(orderBy, SourceSchedules.ID.ASC())
	}

	limit := req.Pagination.GetLimitOrDefault(20, 100)
	stmt := SELECT(SourceSchedules.AllColumns).
		FROM(SourceSchedules).
		WHERE(cond).
		ORDER_BY(orderBy...).
		LIMIT(limit + 1).
		OFFSET(req.Pagination.GetOffset())
	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		return ListSchedulesResponse{}, huma.Error500InternalServerError("failed to list source schedules", err)
	}
	if len(items) == 0 {
		return ListSchedulesResponse{Items: []model.SourceSchedules{}, Total: total}, nil
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
	return ListSchedulesResponse{Items: items, Pagination: cursor, Total: total}, nil
}

// ListSchedulesWithSourceName returns all source schedules with their parent source name.
func (s *Service) ListSchedulesWithSourceName(ctx context.Context) ([]ScheduleWithSource, error) {
	var items []ScheduleWithSource
	stmt := SELECT(
		SourceSchedules.AllColumns,
		Sources.Name.AS("source_name"),
	).FROM(
		SourceSchedules.INNER_JOIN(Sources, Sources.ID.EQ(SourceSchedules.SourceID)),
	).ORDER_BY(SourceSchedules.CreatedAt.ASC())

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return []ScheduleWithSource{}, nil
		}
		return nil, fmt.Errorf("query source_schedules with source name: %w", err)
	}
	if items == nil {
		return []ScheduleWithSource{}, nil
	}
	return items, nil
}
