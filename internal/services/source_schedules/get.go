package source_schedules

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// GetSchedule returns a single source schedule by ID.
func (s *Service) GetSchedule(ctx context.Context, id dbtypes.UUID) (*ScheduleRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
	var sched model.SourceSchedules
	stmt := SELECT(SourceSchedules.AllColumns).
		FROM(SourceSchedules).
		WHERE(SourceSchedules.ID.EQ(String(id.UUID.String()))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &sched); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("query schedule: %w", err)
	}
	return &sched, nil
}
