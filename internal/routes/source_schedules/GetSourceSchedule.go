package source_schedules

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
	schedulesvc "github.com/walens/walens/internal/services/source_schedules"
)

// GetSourceScheduleOperation returns the Huma operation metadata for GetSourceSchedule.
func GetSourceScheduleOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-source_schedules-get-source_schedule",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/source_schedules/GetSourceSchedule"),
		Summary:     "Get a source schedule by ID",
		Description: "Returns a single source schedule by its ID.",
		Tags:        []string{"source_schedules"},
	}
}

// GetSourceScheduleInput describes the request body for GetSourceSchedule.
type GetSourceScheduleInput struct {
	Body struct {
		ID string `json:"id" doc:"Unique source schedule identifier (UUIDv7)."`
	}
}

// GetSourceScheduleOutput describes the response body for GetSourceSchedule.
type GetSourceScheduleOutput struct {
	Body schedulesvc.ScheduleRow
}

// GetSourceSchedule handles POST /api/v1/source_schedules/GetSourceSchedule.
// Returns a single source schedule by ID.
func GetSourceSchedule(ctx context.Context, input *GetSourceScheduleInput, svc *schedulesvc.Service) (*GetSourceScheduleOutput, error) {
	id, err := dbtypes.NewUUIDFromString(input.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid source schedule ID format", err)
	}

	sched, err := svc.GetSchedule(ctx, id)
	if err != nil {
		if errors.Is(err, schedulesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, schedulesvc.ErrScheduleNotFound) {
			return nil, huma.Error404NotFound("source schedule not found")
		}
		return nil, huma.Error500InternalServerError("failed to get source schedule", err)
	}

	return &GetSourceScheduleOutput{
		Body: *sched,
	}, nil
}
