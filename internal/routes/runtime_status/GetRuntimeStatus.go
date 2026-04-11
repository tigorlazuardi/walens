package runtime_status

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
)

// RuntimeStatusOutput describes the runtime status response.
type RuntimeStatusOutput struct {
	Body struct {
		Status         string `json:"status" doc:"Overall runtime status: ok, degraded, stopping."`
		QueueSize      int    `json:"queue_size" doc:"Number of jobs currently in the in-memory queue."`
		SchedulerReady bool   `json:"scheduler_ready" doc:"Whether the scheduler has completed at least one successful reload."`
		ScheduleCount  int    `json:"schedule_count" doc:"Number of active cron schedules loaded."`
		RunnerActive   bool   `json:"runner_active" doc:"Whether the job runner is currently processing or waiting."`
		AuthEnabled    bool   `json:"auth_enabled" doc:"Whether bootstrap auth is enabled."`
	}
}

// GetRuntimeStatusOperation returns the Huma operation metadata for GetRuntimeStatus.
func GetRuntimeStatusOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "GetRuntimeStatus",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/runtime_status/GetRuntimeStatus"),
		Summary:     "Get runtime status",
		Description: "Returns current runtime status including queue size, scheduler state, and worker state.",
		Tags:        []string{"Runtime Status"},
	}
}

// RuntimeStatusDeps contains the dependencies needed for runtime status.
type RuntimeStatusDeps interface {
	QueueSize() int
	SchedulerReady() bool
	GetScheduleCount() int
	IsRunnerActive() bool
	IsAuthEnabled() bool
}

// GetRuntimeStatus handles POST /api/v1/runtime_status/GetRuntimeStatus.
func GetRuntimeStatus(ctx context.Context, input *struct{}, deps RuntimeStatusDeps) (*RuntimeStatusOutput, error) {
	output := &RuntimeStatusOutput{}
	output.Body.QueueSize = deps.QueueSize()
	output.Body.SchedulerReady = deps.SchedulerReady()
	output.Body.ScheduleCount = deps.GetScheduleCount()
	output.Body.RunnerActive = deps.IsRunnerActive()
	output.Body.AuthEnabled = deps.IsAuthEnabled()

	// Determine overall status
	if !deps.SchedulerReady() {
		output.Body.Status = "degraded"
	} else {
		output.Body.Status = "ok"
	}

	return output, nil
}
