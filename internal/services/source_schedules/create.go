package source_schedules

import (
	"context"
	"fmt"

	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type CreateScheduleInput struct {
	SourceID  string `json:"source_id" doc:"Reference to the parent source."`
	CronExpr  string `json:"cron_expr" doc:"Cron expression (5-field, minute hour day month weekday)."`
	IsEnabled bool   `json:"is_enabled" doc:"Whether this schedule is active."`
}

// CreateSchedule creates a new source schedule.
func (s *Service) CreateSchedule(ctx context.Context, input *CreateScheduleInput) (*ScheduleRow, []OverlapWarning, error) {
	if s.db == nil {
		return nil, nil, ErrDBUnavailable
	}
	if _, err := ValidateCronExpr(input.CronExpr); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidCronExpr, err)
	}
	sourceID, err := dbtypes.NewUUIDFromString(input.SourceID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid source ID format: %w", err)
	}
	sourceCount, err := s.countSources(ctx, sourceID)
	if err != nil {
		return nil, nil, err
	}
	if sourceCount == 0 {
		return nil, nil, ErrSourceNotFound
	}
	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, nil, fmt.Errorf("generate UUIDv7: %w", err)
	}
	row := model.SourceSchedules{
		ID:        &id,
		SourceID:  sourceID,
		CronExpr:  input.CronExpr,
		IsEnabled: dbtypes.BoolInt(input.IsEnabled),
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
		return nil, nil, fmt.Errorf("insert schedule: %w", err)
	}
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}
	created, err := s.GetSchedule(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	allSchedules, err := s.ListSchedulesWithSourceName(ctx)
	if err == nil {
		warnings, _ := CheckOverlapWarnings(allSchedules, 14)
		return created, warnings, nil
	}
	return created, nil, nil
}
