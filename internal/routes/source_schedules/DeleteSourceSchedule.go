package source_schedules

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	schedulesvc "github.com/walens/walens/internal/services/source_schedules"
)

// DeleteSourceScheduleOperation returns the Huma operation metadata for DeleteSourceSchedule.
func DeleteSourceScheduleOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID:   "post-source_schedules-delete-source_schedule",
		Method:        "POST",
		Path:          path.Join(basePath, "/api/v1/source_schedules/DeleteSourceSchedule"),
		DefaultStatus: 204,
		Summary:       "Delete a source schedule by ID",
		Description:   "Deletes a source schedule by its ID.",
		Tags:          []string{"source_schedules"},
	}
}

// DeleteSourceScheduleInput describes the request body for DeleteSourceSchedule.
type DeleteSourceScheduleInput struct {
	Body schedulesvc.DeleteScheduleRequest
}

// DeleteSourceSchedule handles POST /api/v1/source_schedules/DeleteSourceSchedule.
// Deletes a source schedule by ID.
func DeleteSourceSchedule(ctx context.Context, input *DeleteSourceScheduleInput, svc *schedulesvc.Service) (*struct{}, error) {
	_, err := svc.DeleteSchedule(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
