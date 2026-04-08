package source_schedules

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	schedulesvc "github.com/walens/walens/internal/services/source_schedules"
)

// ListSourceSchedulesOperation returns the Huma operation metadata for ListSourceSchedules.
func ListSourceSchedulesOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-source_schedules-list-source_schedules",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/source_schedules/ListSourceSchedules"),
		Summary:     "List all source schedules",
		Description: "Returns all source schedule rows.",
		Tags:        []string{"source_schedules"},
	}
}

// ListSourceSchedulesInput describes the request body for ListSourceSchedules.
type ListSourceSchedulesInput struct {
	Body struct{}
}

// ListSourceSchedulesOutput describes the response body for ListSourceSchedules.
type ListSourceSchedulesOutput struct {
	Body struct {
		Items []schedulesvc.ScheduleRow `json:"items" doc:"List of source schedules."`
	}
}

// ListSourceSchedules handles POST /api/v1/source_schedules/ListSourceSchedules.
// Returns all source schedule rows.
func ListSourceSchedules(ctx context.Context, input *ListSourceSchedulesInput, svc *schedulesvc.Service) (*ListSourceSchedulesOutput, error) {
	items, err := svc.ListSchedules(ctx)
	if err != nil {
		if errors.Is(err, schedulesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		return nil, huma.Error500InternalServerError("failed to list source schedules", err)
	}

	return &ListSourceSchedulesOutput{
		Body: struct {
			Items []schedulesvc.ScheduleRow `json:"items" doc:"List of source schedules."`
		}{
			Items: items,
		},
	}, nil
}
