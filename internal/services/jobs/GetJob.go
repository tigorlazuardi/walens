package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// GetJob retrieves a single job by ID using QRM mapping into the generated model.
func (s *Service) GetJob(ctx context.Context, id dbtypes.UUID) (*model.Jobs, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	stmt := SELECT(Jobs.AllColumns).
		FROM(Jobs).
		WHERE(Jobs.ID.EQ(String(id.UUID.String()))).
		LIMIT(1)

	var dest model.Jobs
	err := stmt.QueryContext(ctx, s.db, &dest)
	if errors.Is(err, qrm.ErrNoRows) {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query job: %w", err)
	}

	return &dest, nil
}

// ListJobs retrieves jobs with optional filtering using Jet dynamic conditions and QRM.
func (s *Service) ListJobs(ctx context.Context, input *ListJobsInput) (*ListJobsResponse, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	condition := Bool(true)

	if input.Status != nil && *input.Status != "" {
		condition = condition.AND(Jobs.Status.EQ(String(*input.Status)))
	}
	if input.JobType != nil && *input.JobType != "" {
		condition = condition.AND(Jobs.JobType.EQ(String(*input.JobType)))
	}
	if input.SourceID != nil {
		condition = condition.AND(Jobs.SourceID.EQ(String(input.SourceID.UUID.String())))
	}
	if input.TriggerKind != nil && *input.TriggerKind != "" {
		condition = condition.AND(Jobs.TriggerKind.EQ(String(*input.TriggerKind)))
	}

	var countDest struct {
		Count int64
	}

	countStmt := SELECT(COUNT(Jobs.ID).AS("count")).
		FROM(Jobs).
		WHERE(condition)

	if err := countStmt.QueryContext(ctx, s.db, &countDest); err != nil {
		return nil, fmt.Errorf("count jobs: %w", err)
	}

	var items []model.Jobs
	stmt := SELECT(Jobs.AllColumns).
		FROM(Jobs).
		WHERE(condition).
		ORDER_BY(Jobs.CreatedAt.DESC()).
		LIMIT(int64(input.Limit)).
		OFFSET(int64(input.Offset))

	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			items = []model.Jobs{}
		} else {
			return nil, fmt.Errorf("query jobs: %w", err)
		}
	}

	return &ListJobsResponse{Items: items, Total: countDest.Count}, nil
}
