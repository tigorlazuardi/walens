package jobs

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	jobssvc "github.com/walens/walens/internal/services/jobs"
)

// ListJobsOperation returns the Huma operation metadata for ListJobs.
func ListJobsOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "ListJobs",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/jobs/ListJobs"),
		Summary:     "List jobs",
		Description: "Returns jobs matching the provided filters.",
		Tags:        []string{"Jobs"},
	}
}

// ListJobsInput describes the request body for ListJobs.
type ListJobsInput struct {
	Body jobssvc.ListJobsRequest
}

// ListJobsOutput describes the response body for ListJobs.
type ListJobsOutput struct {
	Body jobssvc.ListJobsResponse
}

// ListJobs handles POST /api/v1/jobs/ListJobs.
func ListJobs(ctx context.Context, input *ListJobsInput, svc *jobssvc.Service) (*ListJobsOutput, error) {
	resp, err := svc.ListJobs(ctx, input.Body)
	if err != nil {
		return nil, err
	}
	return &ListJobsOutput{Body: resp}, nil
}
