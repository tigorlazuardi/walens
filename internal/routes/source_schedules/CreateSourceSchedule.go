package source_schedules

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	schedulesvc "github.com/walens/walens/internal/services/source_schedules"
)

// CreateSourceScheduleOperation returns the Huma operation metadata for CreateSourceSchedule.
func CreateSourceScheduleOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-source_schedules-create-source_schedule",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/source_schedules/CreateSourceSchedule"),
		Summary:     "Create a new source schedule",
		Description: "Creates a new source schedule for a configured source. Cron expressions are validated at the API boundary.",
		Tags:        []string{"source_schedules"},
	}
}

// CreateSourceScheduleInput describes the request body for CreateSourceSchedule.
type CreateSourceScheduleInput struct {
	Body schedulesvc.CreateScheduleInput
}

// CreateSourceScheduleOutput describes the response body for CreateSourceSchedule.
type CreateSourceScheduleOutput struct {
	Body struct {
		Schedule schedulesvc.ScheduleRow      `json:"schedule" doc:"The created source schedule."`
		Warnings []schedulesvc.OverlapWarning `json:"warnings,omitempty" doc:"Overlap warnings if any schedules are less than 5 minutes apart within the same source."`
	}
}

// CreateSourceSchedule handles POST /api/v1/source_schedules/CreateSourceSchedule.
// Creates a new source schedule.
func CreateSourceSchedule(ctx context.Context, input *CreateSourceScheduleInput, svc *schedulesvc.Service) (*CreateSourceScheduleOutput, error) {
	sched, warnings, err := svc.CreateSchedule(ctx, &input.Body)
	if err != nil {
		if errors.Is(err, schedulesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, schedulesvc.ErrSourceNotFound) {
			return nil, huma.Error400BadRequest("source not found")
		}
		if errors.Is(err, schedulesvc.ErrInvalidCronExpr) {
			// Strip the prefix for cleaner error message
			errMsg := err.Error()
			if strings.HasPrefix(errMsg, "invalid cron expression: ") {
				errMsg = strings.TrimPrefix(errMsg, "invalid cron expression: ")
			}
			return nil, huma.Error400BadRequest("invalid cron expression: "+errMsg, err)
		}
		return nil, huma.Error500InternalServerError("failed to create source schedule", err)
	}

	return &CreateSourceScheduleOutput{
		Body: struct {
			Schedule schedulesvc.ScheduleRow      `json:"schedule" doc:"The created source schedule."`
			Warnings []schedulesvc.OverlapWarning `json:"warnings,omitempty" doc:"Overlap warnings if any schedules are less than 5 minutes apart within the same source."`
		}{
			Schedule: *sched,
			Warnings: warnings,
		},
	}, nil
}
