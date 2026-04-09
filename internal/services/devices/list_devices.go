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
	Pagination *dbtypes.CursorPaginationRequest `json:"pagination"`
}

type ListDevicesResponse struct {
	Items      []model.Devices                   `json:"items" doc:"List of devices"`
	Pagination *dbtypes.CursorPaginationResponse `json:"pagination"`
}

func (s *Service) ListDevices(ctx context.Context, req ListDevicesRequest) (ListDevicesResponse, error) {
	var items []model.Devices
	cond := Bool(true)
	if s := ptrStr(req.Search); s != "" {
		pattern := String("%" + s + "%")
		cond = cond.AND(
			Devices.Name.LIKE(pattern).OR(
				Devices.Slug.LIKE(pattern),
			),
		)
	}
	next := req.Pagination.NextToken()
	prev := req.Pagination.PrevToken()
	isPrev := next == "" && prev != ""
	if next != "" {
		cond = cond.AND(Devices.ID.GT(String(next)))
	}
	if isPrev {
		cond = cond.AND(Devices.ID.GT(String(prev)))
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
		return ListDevicesResponse{Items: []model.Devices{}}, nil
	}
	hasMore := len(items) > int(limit)
	items = items[:limit]
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
	return ListDevicesResponse{Items: items, Pagination: cursor}, nil
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
