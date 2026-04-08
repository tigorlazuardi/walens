package source_schedules

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
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
	Body struct {
		ID string `json:"id" doc:"Unique source schedule identifier (UUIDv7)."`
	}
}

// DeleteSourceSchedule handles POST /api/v1/source_schedules/DeleteSourceSchedule.
// Deletes a source schedule by ID.
func DeleteSourceSchedule(ctx context.Context, input *DeleteSourceScheduleInput, svc *schedulesvc.Service) (*struct{}, error) {
	id, err := dbtypes.NewUUIDFromString(input.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid source schedule ID format", err)
	}

	err = svc.DeleteSchedule(ctx, id)
	if err != nil {
		if errors.Is(err, schedulesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, schedulesvc.ErrScheduleNotFound) {
			return nil, huma.Error404NotFound("source schedule not found")
		}
		return nil, huma.Error500InternalServerError("failed to delete source schedule", err)
	}

	return nil, nil
}
