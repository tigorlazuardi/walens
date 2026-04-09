package jobs

import (
	"context"
	"errors"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
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
	baseCond := Bool(true)

	if req.Status != nil && *req.Status != "" {
		baseCond = baseCond.AND(Jobs.Status.EQ(String(*req.Status)))
	}
	if req.JobType != nil && *req.JobType != "" {
		baseCond = baseCond.AND(Jobs.JobType.EQ(String(*req.JobType)))
	}
	if req.SourceID != nil {
		baseCond = baseCond.AND(Jobs.SourceID.EQ(String(req.SourceID.UUID.String())))
	}
	if req.TriggerKind != nil && *req.TriggerKind != "" {
		baseCond = baseCond.AND(Jobs.TriggerKind.EQ(String(*req.TriggerKind)))
	}

	// Get total count before pagination filters
	total, err := s.countJobs(ctx, baseCond)
	if err != nil {
		return ListJobsResponse{}, err
	}

	// Pagination - build condition with cursor filters
	cond := baseCond
	next := req.Pagination.NextToken()
	prev := req.Pagination.PrevToken()
	isPrev := next == "" && prev != ""
	if next != "" {
		cond = cond.AND(Jobs.ID.GT(String(next)))
	}
	if isPrev {
		cond = cond.AND(Jobs.ID.LT(String(prev)))
	}

	orderBy, err := req.Pagination.BuildOrderByClause(Jobs.AllColumns)
	if err != nil {
		return ListJobsResponse{}, err
	}
	if len(orderBy) == 0 {
		orderBy = append(orderBy, Jobs.CreatedAt.DESC())
	}
	if isPrev {
		orderBy = append(orderBy, Jobs.ID.DESC())
	} else {
		orderBy = append(orderBy, Jobs.ID.ASC())
	}

	limit := req.Pagination.GetLimitOrDefault(20, 100)

	var items []model.Jobs
	stmt := SELECT(Jobs.AllColumns).
		FROM(Jobs).
		WHERE(cond).
		ORDER_BY(orderBy...).
		LIMIT(limit + 1).
		OFFSET(req.Pagination.GetOffset())

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		return ListJobsResponse{}, huma.Error500InternalServerError("failed to list jobs", err)
	}
	if len(items) == 0 {
		return ListJobsResponse{Items: []model.Jobs{}, Total: total}, nil
	}

	hasMore := len(items) > int(limit)
	if hasMore {
		items = items[:limit]
	}
	cursor := &dbtypes.CursorPaginationResponse{}
	if isPrev {
		slices.Reverse(items)
	}
	if hasMore {
		cursor.Next = items[len(items)-1].ID
	}
	if next != "" {
		cursor.Prev = items[0].ID
	}

	return ListJobsResponse{Items: items, Pagination: cursor, Total: total}, nil
}
