package source_schedules

import (
	"context"
	"errors"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
)

type ListSchedulesRequest struct{}

type ListSchedulesResponse struct {
	Items []model.SourceSchedules `json:"items" doc:"List of source schedules."`
}

// ListSchedules returns all source schedules ordered by creation time.
func (s *Service) ListSchedules(ctx context.Context, _ ListSchedulesRequest) (ListSchedulesResponse, error) {
	var items []model.SourceSchedules
	stmt := SELECT(SourceSchedules.AllColumns).FROM(SourceSchedules).ORDER_BY(SourceSchedules.CreatedAt.ASC())
	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return ListSchedulesResponse{Items: []model.SourceSchedules{}}, nil
		}
		return ListSchedulesResponse{}, huma.Error500InternalServerError("failed to list source schedules", err)
	}
	if items == nil {
		return ListSchedulesResponse{Items: []model.SourceSchedules{}}, nil
	}
	return ListSchedulesResponse{Items: items}, nil
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
