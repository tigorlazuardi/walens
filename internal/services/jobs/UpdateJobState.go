package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// StartJob transitions a job from queued to running.
func (s *Service) StartJob(ctx context.Context, id dbtypes.UUID) (*model.Jobs, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isValidTransition(job.Status, StatusRunning) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, job.Status, StatusRunning)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	updated := *job
	updated.Status = StatusRunning
	updated.StartedAt = &now
	updated.UpdatedAt = now

	stmt := Jobs.UPDATE(
		Jobs.Status,
		Jobs.StartedAt,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(id.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("start job: %w", err)
	}

	return s.GetJob(ctx, id)
}

// CompleteJob transitions a job to succeeded state.
func (s *Service) CompleteJob(ctx context.Context, id dbtypes.UUID, message *string, jsonResult json.RawMessage) (*model.Jobs, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isValidTransition(job.Status, StatusSucceeded) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, job.Status, StatusSucceeded)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	updated := *job
	updated.Status = StatusSucceeded
	updated.FinishedAt = &now
	updated.Message = message
	updated.JSONResult = ensureJSON(jsonResult)
	updated.UpdatedAt = now
	if job.StartedAt != nil {
		d := dbtypes.NewUnixMilliDuration(now.Time.Sub(job.StartedAt.Time))
		updated.DurationMs = &d
	}

	stmt := Jobs.UPDATE(
		Jobs.Status,
		Jobs.FinishedAt,
		Jobs.DurationMs,
		Jobs.Message,
		Jobs.JSONResult,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(id.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("complete job: %w", err)
	}

	return s.GetJob(ctx, id)
}

// FailJob transitions a job to failed state.
func (s *Service) FailJob(ctx context.Context, id dbtypes.UUID, errorMessage string, jsonResult json.RawMessage) (*model.Jobs, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isValidTransition(job.Status, StatusFailed) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, job.Status, StatusFailed)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	updated := *job
	updated.Status = StatusFailed
	updated.FinishedAt = &now
	updated.ErrorMessage = &errorMessage
	updated.JSONResult = ensureJSON(jsonResult)
	updated.UpdatedAt = now
	if job.StartedAt != nil {
		d := dbtypes.NewUnixMilliDuration(now.Time.Sub(job.StartedAt.Time))
		updated.DurationMs = &d
	}

	stmt := Jobs.UPDATE(
		Jobs.Status,
		Jobs.FinishedAt,
		Jobs.DurationMs,
		Jobs.ErrorMessage,
		Jobs.JSONResult,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(id.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("fail job: %w", err)
	}

	return s.GetJob(ctx, id)
}

// CancelJob transitions a job to cancelled state.
func (s *Service) CancelJob(ctx context.Context, id dbtypes.UUID, message *string) (*model.Jobs, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if !isValidTransition(job.Status, StatusCancelled) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, job.Status, StatusCancelled)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	updated := *job
	updated.Status = StatusCancelled
	updated.Message = message
	updated.UpdatedAt = now
	if job.StartedAt != nil {
		updated.FinishedAt = &now
		d := dbtypes.NewUnixMilliDuration(now.Time.Sub(job.StartedAt.Time))
		updated.DurationMs = &d
	}

	stmt := Jobs.UPDATE(
		Jobs.Status,
		Jobs.FinishedAt,
		Jobs.DurationMs,
		Jobs.Message,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(id.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("cancel job: %w", err)
	}

	return s.GetJob(ctx, id)
}

// UpdateJobState updates job status with validation.
func (s *Service) UpdateJobState(ctx context.Context, input *UpdateJobStateInput) (*model.Jobs, error) {
	if err := validateStatus(input.Status); err != nil {
		return nil, err
	}

	switch input.Status {
	case StatusRunning:
		return s.StartJob(ctx, input.ID)
	case StatusSucceeded:
		return s.CompleteJob(ctx, input.ID, nil, nil)
	case StatusFailed:
		return s.FailJob(ctx, input.ID, "", nil)
	case StatusCancelled:
		return s.CancelJob(ctx, input.ID, nil)
	default:
		job, err := s.GetJob(ctx, input.ID)
		if err != nil {
			return nil, err
		}
		if !isValidTransition(job.Status, input.Status) {
			return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, job.Status, input.Status)
		}
		updated := *job
		updated.Status = input.Status
		updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

		stmt := Jobs.UPDATE(Jobs.Status, Jobs.UpdatedAt).MODEL(updated).WHERE(
			Jobs.ID.EQ(String(input.ID.UUID.String())),
		)

		if _, err := stmt.ExecContext(ctx, s.db); err != nil {
			return nil, fmt.Errorf("update job state: %w", err)
		}

		return s.GetJob(ctx, input.ID)
	}
}
