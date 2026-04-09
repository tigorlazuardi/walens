package source_schedules

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/robfig/cron/v3"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

var ErrScheduleNotFound = errors.New("source schedule not found")
var ErrSourceNotFound = errors.New("source not found")
var ErrInvalidCronExpr = errors.New("invalid cron expression")
var ErrSchedulerUnavailable = errors.New("scheduler unavailable")

type ScheduleRow = model.SourceSchedules

type ScheduleWithSource struct {
	ScheduleRow `alias:"source_schedules.*"`
	SourceName  string `alias:"source_name" json:"source_name" doc:"Name of the parent source for overlap grouping."`
}

type OverlapWarning struct {
	ScheduleID1  dbtypes.UUID `json:"schedule_id_1" doc:"First schedule ID."`
	ScheduleID2  dbtypes.UUID `json:"schedule_id_2" doc:"Second schedule ID."`
	SourceName   string       `json:"source_name" doc:"Source name that both schedules belong to."`
	Occurrence1  time.Time    `json:"occurrence_1" doc:"First occurrence that triggered the warning."`
	Occurrence2  time.Time    `json:"occurrence_2" doc:"Second occurrence that triggered the warning."`
	DistanceMins float64      `json:"distance_mins" doc:"Distance between occurrences in minutes."`
}

type Service struct {
	db        *sql.DB
	scheduler SchedulerInterface
}

type SchedulerInterface interface{ Reload() error }

func NewService(db *sql.DB, scheduler SchedulerInterface) *Service {
	return &Service{db: db, scheduler: scheduler}
}

func ValidateCronExpr(expr string) (cron.Schedule, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	return parser.Parse(expr)
}

func scheduleIDValue(id *dbtypes.UUID) (dbtypes.UUID, bool) {
	if id == nil {
		return dbtypes.UUID{}, false
	}
	return *id, true
}

func CheckOverlapWarnings(schedules []ScheduleWithSource, lookaheadDays int) ([]OverlapWarning, error) {
	if len(schedules) < 2 {
		return nil, nil
	}

	warnings := make([]OverlapWarning, 0)
	now := time.Now().UTC()
	end := now.AddDate(0, 0, lookaheadDays)
	minDistance := 5 * time.Minute

	bySource := make(map[string][]ScheduleWithSource)
	for _, sched := range schedules {
		bySource[sched.SourceName] = append(bySource[sched.SourceName], sched)
	}

	for sourceName, sourceSchedules := range bySource {
		if len(sourceSchedules) < 2 {
			continue
		}

		type parsedSchedule struct {
			schedule ScheduleWithSource
			parser   cron.Schedule
		}
		parsed := make([]parsedSchedule, 0, len(sourceSchedules))
		for _, s := range sourceSchedules {
			parser, err := ValidateCronExpr(s.CronExpr)
			if err != nil {
				continue
			}
			parsed = append(parsed, parsedSchedule{schedule: s, parser: parser})
		}
		if len(parsed) < 2 {
			continue
		}

		type occurrence struct {
			schedID  dbtypes.UUID
			t        time.Time
			schedIdx int
		}
		allOccurrences := make([]occurrence, 0)

		for idx, p := range parsed {
			scheduleID, ok := scheduleIDValue(p.schedule.ID)
			if !ok {
				continue
			}
			for t := p.parser.Next(now); t.Before(end); t = p.parser.Next(t) {
				allOccurrences = append(allOccurrences, occurrence{schedID: scheduleID, t: t, schedIdx: idx})
			}
		}

		sort.Slice(allOccurrences, func(i, j int) bool { return allOccurrences[i].t.Before(allOccurrences[j].t) })

		for i := 1; i < len(allOccurrences); i++ {
			prev := allOccurrences[i-1]
			curr := allOccurrences[i]
			if prev.schedIdx == curr.schedIdx {
				continue
			}
			dist := curr.t.Sub(prev.t)
			if dist >= minDistance {
				continue
			}

			warnings = append(warnings, OverlapWarning{
				ScheduleID1:  prev.schedID,
				ScheduleID2:  curr.schedID,
				SourceName:   sourceName,
				Occurrence1:  prev.t,
				Occurrence2:  curr.t,
				DistanceMins: dist.Minutes(),
			})
		}
	}

	return warnings, nil
}

func (s *Service) countSources(ctx context.Context, id dbtypes.UUID) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(Sources.ID).AS("count")).FROM(Sources).WHERE(Sources.ID.EQ(String(id.UUID.String())))
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("check source exists: %w", err)
	}
	return count.Count, nil
}
