package source_schedules

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type CreateScheduleRequest struct {
	SourceID  dbtypes.UUID `json:"source_id" required:"true" doc:"Reference to the parent source."`
	CronExpr  string       `json:"cron_expr" required:"true" doc:"Cron expression (5-field, minute hour day month weekday)."`
	IsEnabled bool         `json:"is_enabled" required:"true" doc:"Whether this schedule is active."`
}

type CreateScheduleResponse struct {
	Schedule model.SourceSchedules `json:"schedule" doc:"The created source schedule."`
	Warnings []OverlapWarning      `json:"warnings,omitempty" doc:"Overlap warnings if any schedules are less than 5 minutes apart within the same source."`
}

// CreateSchedule creates a new source schedule.
func (s *Service) CreateSchedule(ctx context.Context, req CreateScheduleRequest) (CreateScheduleResponse, error) {
	if _, err := ValidateCronExpr(req.CronExpr); err != nil {
		return CreateScheduleResponse{}, huma.Error400BadRequest("invalid cron expression: "+err.Error(), ErrInvalidCronExpr)
	}
	sourceCount, err := s.countSources(ctx, req.SourceID)
	if err != nil {
		return CreateScheduleResponse{}, huma.Error500InternalServerError("failed to validate source", err)
	}
	if sourceCount == 0 {
		return CreateScheduleResponse{}, huma.Error400BadRequest("source not found", ErrSourceNotFound)
	}
	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return CreateScheduleResponse{}, huma.Error500InternalServerError("failed to generate schedule id", err)
	}
	row := model.SourceSchedules{
		ID:        id,
		SourceID:  req.SourceID,
		CronExpr:  req.CronExpr,
		IsEnabled: dbtypes.BoolInt(req.IsEnabled),
		CreatedAt: now,
		UpdatedAt: now,
	}
	stmt := SourceSchedules.INSERT(
		SourceSchedules.ID,
		SourceSchedules.SourceID,
		SourceSchedules.CronExpr,
		SourceSchedules.IsEnabled,
		SourceSchedules.CreatedAt,
		SourceSchedules.UpdatedAt,
	).MODEL(row)
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return CreateScheduleResponse{}, huma.Error500InternalServerError("failed to create source schedule", err)
	}
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}
	created, err := s.GetSchedule(ctx, GetScheduleRequest{ID: id})
	if err != nil {
		return CreateScheduleResponse{}, err
	}
	allSchedules, err := s.ListSchedulesWithSourceName(ctx)
	if err == nil {
		warnings, _ := CheckOverlapWarnings(allSchedules, 14)
		return CreateScheduleResponse{Schedule: created, Warnings: warnings}, nil
	}
	return CreateScheduleResponse{Schedule: created}, nil
}
