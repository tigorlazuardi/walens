package source_schedules

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type GetScheduleRequest struct {
	ID dbtypes.UUID `json:"id" required:"true" doc:"Unique source schedule identifier."`
}

type GetScheduleResponse = model.SourceSchedules

// GetSchedule returns a single source schedule by ID.
func (s *Service) GetSchedule(ctx context.Context, req GetScheduleRequest) (GetScheduleResponse, error) {
	var sched model.SourceSchedules
	stmt := SELECT(SourceSchedules.AllColumns).
		FROM(SourceSchedules).
		WHERE(SourceSchedules.ID.EQ(String(req.ID.UUID.String()))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &sched); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return GetScheduleResponse{}, huma.Error404NotFound("source schedule not found", ErrScheduleNotFound)
		}
		return GetScheduleResponse{}, huma.Error500InternalServerError("failed to get source schedule", err)
	}
	return sched, nil
}
