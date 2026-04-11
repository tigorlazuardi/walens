package devices

import (
	"context"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type ListDevicesRequest struct {
	Search     *string                          `json:"search" doc:"Search devices by name or slug"`
	Pagination *dbtypes.CursorPaginationRequest `json:"pagination,omitempty"`
}

type ListDevicesResponse struct {
	Items      []model.Devices                   `json:"items" doc:"List of devices"`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
	Total      int64                             `json:"total" doc:"Total count of devices matching filters, independent of pagination"`
}

func (s *Service) ListDevices(ctx context.Context, req ListDevicesRequest) (ListDevicesResponse, error) {
	var items []model.Devices
	baseCond := Bool(true)
	if s := ptrStr(req.Search); s != "" {
		pattern := String("%" + s + "%")
		baseCond = baseCond.AND(
			Devices.Name.LIKE(pattern).OR(
				Devices.Slug.LIKE(pattern),
			),
		)
	}

	// Get total count before pagination filters
	total, err := s.countDevices(ctx, baseCond)
	if err != nil {
		return ListDevicesResponse{}, err
	}

	// Pagination - build condition with cursor filters
	cond := baseCond
	next := req.Pagination.NextToken()
	prev := req.Pagination.PrevToken()
	isPrev := next == "" && prev != ""
	if next != "" {
		cond = cond.AND(Devices.ID.GT(String(next)))
	}
	if isPrev {
		cond = cond.AND(Devices.ID.LT(String(prev)))
	}
	orderBy, err := req.Pagination.BuildOrderByClause(Devices.AllColumns)
	if err != nil {
		return ListDevicesResponse{}, err
	}
	if len(orderBy) == 0 {
		orderBy = append(orderBy, Devices.Name.ASC())
	}
	if isPrev {
		orderBy = append(orderBy, Devices.ID.DESC())
	} else {
		orderBy = append(orderBy, Devices.ID.ASC())
	}
	limit := req.Pagination.GetLimitOrDefault(20, 100)
	stmt := SELECT(Devices.AllColumns).
		FROM(Devices).
		WHERE(cond).
		ORDER_BY(orderBy...).
		LIMIT(limit + 1).
		OFFSET(req.Pagination.GetOffset())

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		return ListDevicesResponse{}, huma.Error500InternalServerError("failed to list devices", err)
	}
	if len(items) == 0 {
		return ListDevicesResponse{Items: []model.Devices{}, Total: total}, nil
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
		nextID := items[len(items)-1].ID
		cursor.Next = &nextID
	}
	if next != "" {
		prevID := items[0].ID
		cursor.Prev = &prevID
	}
	return ListDevicesResponse{Items: items, Pagination: cursor, Total: total}, nil
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getLimitPtr(i *int, def int64, max int64) int64 {
	if i == nil {
		return def
	}
	return getLimit(*i, def, max)
}

func getLimit(val int, def int64, maximum int64) int64 {
	if val <= 0 {
		return def
	}
	return min(maximum, int64(val))
}
