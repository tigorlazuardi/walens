package source_schedules

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	schedulesvc "github.com/walens/walens/internal/services/source_schedules"
)

// UpdateSourceScheduleOperation returns the Huma operation metadata for UpdateSourceSchedule.
func UpdateSourceScheduleOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-source_schedules-update-source_schedule",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/source_schedules/UpdateSourceSchedule"),
		Summary:     "Update an existing source schedule",
		Description: "Updates an existing source schedule with full-object update semantics. All fields are required. Cron expressions are validated at the API boundary.",
		Tags:        []string{"source_schedules"},
	}
}

// UpdateSourceScheduleInput describes the request body for UpdateSourceSchedule.
type UpdateSourceScheduleInput struct {
	Body schedulesvc.UpdateScheduleRequest
}

// UpdateSourceScheduleOutput describes the response body for UpdateSourceSchedule.
type UpdateSourceScheduleOutput struct {
	Body struct {
		Schedule schedulesvc.ScheduleRow      `json:"schedule" doc:"The updated source schedule."`
		Warnings []schedulesvc.OverlapWarning `json:"warnings,omitempty" doc:"Overlap warnings if any schedules are less than 5 minutes apart within the same source."`
	}
}

// UpdateSourceSchedule handles POST /api/v1/source_schedules/UpdateSourceSchedule.
// Updates an existing source schedule with full-object update semantics.
func UpdateSourceSchedule(ctx context.Context, input *UpdateSourceScheduleInput, svc *schedulesvc.Service) (*UpdateSourceScheduleOutput, error) {
	resp, err := svc.UpdateSchedule(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &UpdateSourceScheduleOutput{
		Body: struct {
			Schedule schedulesvc.ScheduleRow      `json:"schedule" doc:"The updated source schedule."`
			Warnings []schedulesvc.OverlapWarning `json:"warnings,omitempty" doc:"Overlap warnings if any schedules are less than 5 minutes apart within the same source."`
		}{
			Schedule: resp.Schedule,
			Warnings: resp.Warnings,
		},
	}, nil
}
