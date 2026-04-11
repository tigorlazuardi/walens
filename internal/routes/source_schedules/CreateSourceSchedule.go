package source_schedules

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	schedulesvc "github.com/walens/walens/internal/services/source_schedules"
)

// CreateSourceScheduleOperation returns the Huma operation metadata for CreateSourceSchedule.
func CreateSourceScheduleOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "CreateSourceSchedule",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/source_schedules/CreateSourceSchedule"),
		Summary:     "Create a new source schedule",
		Description: "Creates a new source schedule for a configured source. Cron expressions are validated at the API boundary.",
		Tags:        []string{"Source Schedules"},
	}
}

// CreateSourceScheduleInput describes the request body for CreateSourceSchedule.
type CreateSourceScheduleInput struct {
	Body schedulesvc.CreateScheduleRequest
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
	resp, err := svc.CreateSchedule(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &CreateSourceScheduleOutput{
		Body: struct {
			Schedule schedulesvc.ScheduleRow      `json:"schedule" doc:"The created source schedule."`
			Warnings []schedulesvc.OverlapWarning `json:"warnings,omitempty" doc:"Overlap warnings if any schedules are less than 5 minutes apart within the same source."`
		}{
			Schedule: resp.Schedule,
			Warnings: resp.Warnings,
		},
	}, nil
}
