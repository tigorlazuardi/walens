package source_schedules

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type DeleteScheduleRequest struct {
	ID dbtypes.UUID `json:"id" required:"true" doc:"Unique source schedule identifier."`
}

type DeleteScheduleResponse struct{}

// DeleteSchedule deletes a source schedule by ID.
func (s *Service) DeleteSchedule(ctx context.Context, req DeleteScheduleRequest) (DeleteScheduleResponse, error) {
	if _, err := s.GetSchedule(ctx, GetScheduleRequest{ID: req.ID}); err != nil {
		return DeleteScheduleResponse{}, err
	}
	stmt := SourceSchedules.DELETE().WHERE(SourceSchedules.ID.EQ(String(req.ID.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return DeleteScheduleResponse{}, huma.Error500InternalServerError("failed to delete source schedule", err)
	}
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}
	return DeleteScheduleResponse{}, nil
}
