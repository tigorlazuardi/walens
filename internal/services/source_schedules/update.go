package source_schedules

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type UpdateScheduleInput struct {
	ID        string `json:"id" doc:"Unique source schedule identifier."`
	SourceID  string `json:"source_id" doc:"Reference to the parent source."`
	CronExpr  string `json:"cron_expr" doc:"Cron expression (5-field, minute hour day month weekday)."`
	IsEnabled bool   `json:"is_enabled" doc:"Whether this schedule is active."`
}

// UpdateSchedule updates an existing source schedule.
func (s *Service) UpdateSchedule(ctx context.Context, input *UpdateScheduleInput) (*ScheduleRow, []OverlapWarning, error) {
	if s.db == nil {
		return nil, nil, ErrDBUnavailable
	}
	if _, err := ValidateCronExpr(input.CronExpr); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidCronExpr, err)
	}
	id, err := dbtypes.NewUUIDFromString(input.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid schedule ID format: %w", err)
	}
	sourceID, err := dbtypes.NewUUIDFromString(input.SourceID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid source ID format: %w", err)
	}
	if _, err := s.GetSchedule(ctx, id); err != nil {
		return nil, nil, err
	}
	sourceCount, err := s.countSources(ctx, sourceID)
	if err != nil {
		return nil, nil, err
	}
	if sourceCount == 0 {
		return nil, nil, ErrSourceNotFound
	}
	updated := model.SourceSchedules{
		ID:        &id,
		SourceID:  sourceID,
		CronExpr:  input.CronExpr,
		IsEnabled: dbtypes.BoolInt(input.IsEnabled),
		UpdatedAt: dbtypes.NewUnixMilliTimeNow(),
	}
	stmt := SourceSchedules.UPDATE(
		SourceSchedules.SourceID,
		SourceSchedules.CronExpr,
		SourceSchedules.IsEnabled,
		SourceSchedules.UpdatedAt,
	).MODEL(updated).WHERE(SourceSchedules.ID.EQ(String(id.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, nil, fmt.Errorf("update schedule: %w", err)
	}
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}
	sched, err := s.GetSchedule(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	allSchedules, err := s.ListSchedulesWithSourceName(ctx)
	if err == nil {
		warnings, _ := CheckOverlapWarnings(allSchedules, 14)
		return sched, warnings, nil
	}
	return sched, nil, nil
}
