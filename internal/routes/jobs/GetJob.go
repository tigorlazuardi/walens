package jobs

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	jobssvc "github.com/walens/walens/internal/services/jobs"
)

// GetJobOperation returns the Huma operation metadata for GetJob.
func GetJobOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "GetJob",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/jobs/GetJob"),
		Summary:     "Get a job",
		Description: "Returns a single job by ID.",
		Tags:        []string{"Jobs"},
	}
}

// GetJobInput describes the request body for GetJob.
type GetJobInput struct {
	Body jobssvc.GetJobRequest
}

// GetJobOutput describes the response body for GetJob.
type GetJobOutput struct {
	Body jobssvc.JobResponse
}

// GetJob handles POST /api/v1/jobs/GetJob.
func GetJob(ctx context.Context, input *GetJobInput, svc *jobssvc.Service) (*GetJobOutput, error) {
	resp, err := svc.GetJob(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return &GetJobOutput{Body: resp}, nil
}
