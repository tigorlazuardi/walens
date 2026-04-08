package source_schedules

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// DeleteSchedule deletes a source schedule by ID.
func (s *Service) DeleteSchedule(ctx context.Context, id dbtypes.UUID) error {
	if s.db == nil {
		return ErrDBUnavailable
	}
	if _, err := s.GetSchedule(ctx, id); err != nil {
		return err
	}
	stmt := SourceSchedules.DELETE().WHERE(SourceSchedules.ID.EQ(String(id.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	if s.scheduler != nil {
		_ = s.scheduler.Reload()
	}
	return nil
}
