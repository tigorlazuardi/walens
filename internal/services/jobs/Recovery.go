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

// GetJobsForRecovery retrieves queued and running jobs using QRM.
func (s *Service) GetJobsForRecovery(ctx context.Context) ([]model.Jobs, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	statuses := []Expression{String(StatusQueued), String(StatusRunning)}
	var jobs []model.Jobs

	stmt := SELECT(Jobs.AllColumns).
		FROM(Jobs).
		WHERE(Jobs.Status.IN(statuses...)).
		ORDER_BY(Jobs.CreatedAt.ASC())

	if err := stmt.QueryContext(ctx, s.db, &jobs); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return []model.Jobs{}, nil
		}
		return nil, fmt.Errorf("query jobs for recovery: %w", err)
	}

	return jobs, nil
}

// RecoverRunningJobs resets running jobs to queued for reprocessing.
func (s *Service) RecoverRunningJobs(ctx context.Context) (int64, error) {
	if s.db == nil {
		return 0, ErrDBUnavailable
	}

	updated := model.Jobs{
		Status:     StatusQueued,
		StartedAt:  nil,
		DurationMs: nil,
		UpdatedAt:  dbtypes.NewUnixMilliTimeNow(),
	}

	stmt := Jobs.UPDATE(
		Jobs.Status,
		Jobs.StartedAt,
		Jobs.DurationMs,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.Status.EQ(String(StatusRunning)),
	)

	result, err := stmt.ExecContext(ctx, s.db)
	if err != nil {
		return 0, fmt.Errorf("recover running jobs: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return affected, nil
}

// MarkJobsForRecovery marks interrupted jobs with trigger_kind=recovery.
func (s *Service) MarkJobsForRecovery(ctx context.Context, jobIDs []dbtypes.UUID) (int64, error) {
	if s.db == nil {
		return 0, ErrDBUnavailable
	}
	if len(jobIDs) == 0 {
		return 0, nil
	}

	ids := make([]Expression, 0, len(jobIDs))
	for _, id := range jobIDs {
		ids = append(ids, String(id.UUID.String()))
	}

	updated := model.Jobs{
		TriggerKind: TriggerKindRecovery,
		UpdatedAt:   dbtypes.NewUnixMilliTimeNow(),
	}

	stmt := Jobs.UPDATE(
		Jobs.TriggerKind,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.IN(ids...),
	)

	result, err := stmt.ExecContext(ctx, s.db)
	if err != nil {
		return 0, fmt.Errorf("mark jobs for recovery: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return affected, nil
}

// GetRecentJobs retrieves recent jobs for a source using QRM.
func (s *Service) GetRecentJobs(ctx context.Context, sourceID dbtypes.UUID, limit int) ([]model.Jobs, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	var jobs []model.Jobs
	stmt := SELECT(Jobs.AllColumns).
		FROM(Jobs).
		WHERE(Jobs.SourceID.EQ(String(sourceID.UUID.String()))).
		ORDER_BY(Jobs.CreatedAt.DESC()).
		LIMIT(int64(limit))

	if err := stmt.QueryContext(ctx, s.db, &jobs); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return []model.Jobs{}, nil
		}
		return nil, fmt.Errorf("query recent jobs: %w", err)
	}

	return jobs, nil
}
