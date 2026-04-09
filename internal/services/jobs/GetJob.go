package jobs

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
)

// GetJob retrieves a single job by ID using QRM mapping into the generated model.
func (s *Service) GetJob(ctx context.Context, req GetJobRequest) (JobResponse, error) {
	stmt := SELECT(Jobs.AllColumns).
		FROM(Jobs).
		WHERE(Jobs.ID.EQ(String(req.ID.UUID.String()))).
		LIMIT(1)

	var dest model.Jobs
	err := stmt.QueryContext(ctx, s.db, &dest)
	if errors.Is(err, qrm.ErrNoRows) {
		return JobResponse{}, huma.Error404NotFound("job not found", ErrJobNotFound)
	}
	if err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to get job", err)
	}

	return dest, nil
}

// ListJobs retrieves jobs with optional filtering using Jet dynamic conditions and QRM.
func (s *Service) ListJobs(ctx context.Context, req ListJobsRequest) (ListJobsResponse, error) {
	condition := Bool(true)

	if req.Status != nil && *req.Status != "" {
		condition = condition.AND(Jobs.Status.EQ(String(*req.Status)))
	}
	if req.JobType != nil && *req.JobType != "" {
		condition = condition.AND(Jobs.JobType.EQ(String(*req.JobType)))
	}
	if req.SourceID != nil {
		condition = condition.AND(Jobs.SourceID.EQ(String(req.SourceID.UUID.String())))
	}
	if req.TriggerKind != nil && *req.TriggerKind != "" {
		condition = condition.AND(Jobs.TriggerKind.EQ(String(*req.TriggerKind)))
	}

	var countDest struct {
		Count int64
	}

	countStmt := SELECT(COUNT(Jobs.ID).AS("count")).
		FROM(Jobs).
		WHERE(condition)

	if err := countStmt.QueryContext(ctx, s.db, &countDest); err != nil {
		return ListJobsResponse{}, huma.Error500InternalServerError("failed to count jobs", err)
	}

	var items []model.Jobs
	stmt := SELECT(Jobs.AllColumns).
		FROM(Jobs).
		WHERE(condition).
		ORDER_BY(Jobs.CreatedAt.DESC()).
		LIMIT(int64(req.Limit)).
		OFFSET(int64(req.Offset))

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			items = []model.Jobs{}
		} else {
			return ListJobsResponse{}, huma.Error500InternalServerError("failed to list jobs", err)
		}
	}

	return ListJobsResponse{Items: items, Total: countDest.Count}, nil
}
