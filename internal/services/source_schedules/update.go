package source_schedules

import (
	"context"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type UpdateScheduleRequest struct {
	ID        dbtypes.UUID `json:"id" doc:"Unique source schedule identifier."`
	SourceID  dbtypes.UUID `json:"source_id" doc:"Reference to the parent source."`
	CronExpr  string       `json:"cron_expr" doc:"Cron expression (5-field, minute hour day month weekday)."`
	IsEnabled bool         `json:"is_enabled" doc:"Whether this schedule is active."`
}

type UpdateScheduleResponse struct {
	Schedule model.SourceSchedules `json:"schedule" doc:"The updated source schedule."`
	Warnings []OverlapWarning      `json:"warnings,omitempty" doc:"Overlap warnings if any schedules are less than 5 minutes apart within the same source."`
}

// UpdateSchedule updates an existing source schedule.
func (s *Service) UpdateSchedule(ctx context.Context, req UpdateScheduleRequest) (UpdateScheduleResponse, error) {
	if _, err := ValidateCronExpr(req.CronExpr); err != nil {
		errMsg := err.Error()
		if strings.HasPrefix(errMsg, "invalid cron expression: ") {
			errMsg = strings.TrimPrefix(errMsg, "invalid cron expression: ")
		}
		return UpdateScheduleResponse{}, huma.Error400BadRequest("invalid cron expression: "+errMsg, ErrInvalidCronExpr)
	}
	if _, err := s.GetSchedule(ctx, GetScheduleRequest{ID: req.ID}); err != nil {
		return UpdateScheduleResponse{}, err
	}
	sourceCount, err := s.countSources(ctx, req.SourceID)
	if err != nil {
		return UpdateScheduleResponse{}, huma.Error500InternalServerError("failed to validate source", err)
	}
	if sourceCount == 0 {
		return UpdateScheduleResponse{}, huma.Error400BadRequest("source not found", ErrSourceNotFound)
	}
	updated := model.SourceSchedules{
		ID:        &req.ID,
		SourceID:  req.SourceID,
		CronExpr:  req.CronExpr,
		IsEnabled: dbtypes.BoolInt(req.IsEnabled),
		UpdatedAt: dbtypes.NewUnixMilliTimeNow(),
	}
	stmt := SourceSchedules.UPDATE(
		SourceSchedules.SourceID,
		SourceSchedules.CronExpr,
		SourceSchedules.IsEnabled,
		SourceSchedules.UpdatedAt,
	).MODEL(updated).WHERE(SourceSchedules.ID.EQ(String(req.ID.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return UpdateScheduleResponse{}, huma.Error500InternalServerError("failed to update source schedule", err)
	}
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}
	sched, err := s.GetSchedule(ctx, GetScheduleRequest{ID: req.ID})
	if err != nil {
		return UpdateScheduleResponse{}, err
	}
	allSchedules, err := s.ListSchedulesWithSourceName(ctx)
	if err == nil {
		warnings, _ := CheckOverlapWarnings(allSchedules, 14)
		return UpdateScheduleResponse{Schedule: sched, Warnings: warnings}, nil
	}
	return UpdateScheduleResponse{Schedule: sched}, nil
}
