package source_schedules

import (
	"context"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/scheduler"
)

type enabledScheduleRow struct {
	SourceID   string `alias:"source_id"`
	SourceName string `alias:"source_name"`
	SourceType string `alias:"source_type"`
	ScheduleID string `alias:"schedule_id"`
	CronExpr   string `alias:"cron_expr"`
}

// LoadEnabledSourceSchedules returns all enabled source schedules for runtime scheduling.
func (s *Service) LoadEnabledSourceSchedules(ctx context.Context) ([]scheduler.SourceSchedule, error) {
	if s == nil || s.db == nil {
		return []scheduler.SourceSchedule{}, nil
	}

	var items []enabledScheduleRow
	stmt := SELECT(
		Sources.ID.AS("source_id"),
		Sources.Name.AS("source_name"),
		Sources.SourceType.AS("source_type"),
		SourceSchedules.ID.AS("schedule_id"),
		SourceSchedules.CronExpr.AS("cron_expr"),
	).
		FROM(SourceSchedules.INNER_JOIN(Sources, Sources.ID.EQ(SourceSchedules.SourceID))).
		WHERE(Bool(true).AND(Sources.IsEnabled.EQ(Int64(1))).AND(SourceSchedules.IsEnabled.EQ(Int64(1)))).
		ORDER_BY(Sources.ID.ASC(), SourceSchedules.ID.ASC())

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		if err == qrm.ErrNoRows {
			return []scheduler.SourceSchedule{}, nil
		}
		return nil, fmt.Errorf("load enabled source schedules: %w", err)
	}

	results := make([]scheduler.SourceSchedule, 0, len(items))
	for _, item := range items {
		sourceID, err := dbtypes.NewUUIDFromString(item.SourceID)
		if err != nil {
			continue
		}
		scheduleID, err := dbtypes.NewUUIDFromString(item.ScheduleID)
		if err != nil {
			continue
		}
		results = append(results, scheduler.SourceSchedule{
			SourceID:   sourceID,
			SourceName: item.SourceName,
			SourceType: item.SourceType,
			ScheduleID: scheduleID,
			CronExpr:   item.CronExpr,
		})
	}

	return results, nil
}
