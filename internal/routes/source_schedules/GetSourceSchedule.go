package source_schedules

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	schedulesvc "github.com/walens/walens/internal/services/source_schedules"
)

// GetSourceScheduleOperation returns the Huma operation metadata for GetSourceSchedule.
func GetSourceScheduleOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "GetSourceSchedule",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/source_schedules/GetSourceSchedule"),
		Summary:     "Get a source schedule by ID",
		Description: "Returns a single source schedule by its ID.",
		Tags:        []string{"Source Schedules"},
	}
}

// GetSourceScheduleInput describes the request body for GetSourceSchedule.
type GetSourceScheduleInput struct {
	Body schedulesvc.GetScheduleRequest
}

// GetSourceScheduleOutput describes the response body for GetSourceSchedule.
type GetSourceScheduleOutput struct {
	Body schedulesvc.ScheduleRow
}

// GetSourceSchedule handles POST /api/v1/source_schedules/GetSourceSchedule.
// Returns a single source schedule by ID.
func GetSourceSchedule(ctx context.Context, input *GetSourceScheduleInput, svc *schedulesvc.Service) (*GetSourceScheduleOutput, error) {
	sched, err := svc.GetSchedule(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &GetSourceScheduleOutput{
		Body: sched,
	}, nil
}
