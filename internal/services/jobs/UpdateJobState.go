package jobs

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// StartJob transitions a job from queued to running.
func (s *Service) StartJob(ctx context.Context, req StartJobRequest) (JobResponse, error) {
	job, err := s.GetJob(ctx, GetJobRequest{ID: req.ID})
	if err != nil {
		return JobResponse{}, err
	}
	if !isValidTransition(job.Status, StatusRunning) {
		return JobResponse{}, huma.Error400BadRequest(fmt.Sprintf("cannot transition from %s to %s", job.Status, StatusRunning), ErrInvalidTransition)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	updated := job
	updated.Status = StatusRunning
	updated.StartedAt = &now
	updated.UpdatedAt = now

	stmt := Jobs.UPDATE(
		Jobs.Status,
		Jobs.StartedAt,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(req.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to start job", err)
	}

	return s.GetJob(ctx, GetJobRequest{ID: req.ID})
}

// CompleteJob transitions a job to succeeded state.
func (s *Service) CompleteJob(ctx context.Context, req CompleteJobRequest) (JobResponse, error) {
	job, err := s.GetJob(ctx, GetJobRequest{ID: req.ID})
	if err != nil {
		return JobResponse{}, err
	}
	if !isValidTransition(job.Status, StatusSucceeded) {
		return JobResponse{}, huma.Error400BadRequest(fmt.Sprintf("cannot transition from %s to %s", job.Status, StatusSucceeded), ErrInvalidTransition)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	updated := job
	updated.Status = StatusSucceeded
	updated.FinishedAt = &now
	updated.Message = req.Message
	updated.JSONResult = ensureJSON(req.JSONResult)
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
		Jobs.ID.EQ(String(req.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to complete job", err)
	}

	return s.GetJob(ctx, GetJobRequest{ID: req.ID})
}

// FailJob transitions a job to failed state.
func (s *Service) FailJob(ctx context.Context, req FailJobRequest) (JobResponse, error) {
	job, err := s.GetJob(ctx, GetJobRequest{ID: req.ID})
	if err != nil {
		return JobResponse{}, err
	}
	if !isValidTransition(job.Status, StatusFailed) {
		return JobResponse{}, huma.Error400BadRequest(fmt.Sprintf("cannot transition from %s to %s", job.Status, StatusFailed), ErrInvalidTransition)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	updated := job
	updated.Status = StatusFailed
	updated.FinishedAt = &now
	updated.ErrorMessage = &req.ErrorMessage
	updated.JSONResult = ensureJSON(req.JSONResult)
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
		Jobs.ID.EQ(String(req.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to fail job", err)
	}

	return s.GetJob(ctx, GetJobRequest{ID: req.ID})
}

// CancelJob transitions a job to cancelled state.
func (s *Service) CancelJob(ctx context.Context, req CancelJobRequest) (JobResponse, error) {
	job, err := s.GetJob(ctx, GetJobRequest{ID: req.ID})
	if err != nil {
		return JobResponse{}, err
	}
	if !isValidTransition(job.Status, StatusCancelled) {
		return JobResponse{}, huma.Error400BadRequest(fmt.Sprintf("cannot transition from %s to %s", job.Status, StatusCancelled), ErrInvalidTransition)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	updated := job
	updated.Status = StatusCancelled
	updated.Message = req.Message
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
		Jobs.ID.EQ(String(req.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to cancel job", err)
	}

	return s.GetJob(ctx, GetJobRequest{ID: req.ID})
}

// UpdateJobState updates job status with validation.
func (s *Service) UpdateJobState(ctx context.Context, req UpdateJobStateRequest) (JobResponse, error) {
	if err := validateStatus(req.Status); err != nil {
		return JobResponse{}, huma.Error400BadRequest(err.Error(), err)
	}

	switch req.Status {
	case StatusRunning:
		return s.StartJob(ctx, StartJobRequest{ID: req.ID})
	case StatusSucceeded:
		return s.CompleteJob(ctx, CompleteJobRequest{ID: req.ID})
	case StatusFailed:
		return s.FailJob(ctx, FailJobRequest{ID: req.ID, ErrorMessage: ""})
	case StatusCancelled:
		return s.CancelJob(ctx, CancelJobRequest{ID: req.ID})
	default:
		job, err := s.GetJob(ctx, GetJobRequest{ID: req.ID})
		if err != nil {
			return JobResponse{}, err
		}
		if !isValidTransition(job.Status, req.Status) {
			return JobResponse{}, huma.Error400BadRequest(fmt.Sprintf("cannot transition from %s to %s", job.Status, req.Status), ErrInvalidTransition)
		}
		updated := job
		updated.Status = req.Status
		updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

		stmt := Jobs.UPDATE(Jobs.Status, Jobs.UpdatedAt).MODEL(updated).WHERE(
			Jobs.ID.EQ(String(req.ID.UUID.String())),
		)

		if _, err := stmt.ExecContext(ctx, s.db); err != nil {
			return JobResponse{}, huma.Error500InternalServerError("failed to update job state", err)
		}

		return s.GetJob(ctx, GetJobRequest{ID: req.ID})
	}
}
