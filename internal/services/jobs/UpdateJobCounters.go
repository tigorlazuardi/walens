package jobs

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// IncrementJobCounters atomically increments job counters using MODEL update semantics.
func (s *Service) IncrementJobCounters(ctx context.Context, req IncrementJobCountersRequest) (JobResponse, error) {
	job, err := s.GetJob(ctx, GetJobRequest{ID: req.ID})
	if err != nil {
		return JobResponse{}, err
	}
	if job.Status != StatusRunning {
		return JobResponse{}, huma.Error400BadRequest("job is not running", ErrJobNotRunning)
	}

	updated := job
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

	if req.Deltas.DownloadedImageCount != nil {
		updated.DownloadedImageCount += *req.Deltas.DownloadedImageCount
	}
	if req.Deltas.ReusedImageCount != nil {
		updated.ReusedImageCount += *req.Deltas.ReusedImageCount
	}
	if req.Deltas.HardlinkedImageCount != nil {
		updated.HardlinkedImageCount += *req.Deltas.HardlinkedImageCount
	}
	if req.Deltas.CopiedImageCount != nil {
		updated.CopiedImageCount += *req.Deltas.CopiedImageCount
	}
	if req.Deltas.StoredImageCount != nil {
		updated.StoredImageCount += *req.Deltas.StoredImageCount
	}
	if req.Deltas.SkippedImageCount != nil {
		updated.SkippedImageCount += *req.Deltas.SkippedImageCount
	}

	stmt := Jobs.UPDATE(
		Jobs.DownloadedImageCount,
		Jobs.ReusedImageCount,
		Jobs.HardlinkedImageCount,
		Jobs.CopiedImageCount,
		Jobs.StoredImageCount,
		Jobs.SkippedImageCount,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(req.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to increment job counters", err)
	}

	return s.GetJob(ctx, GetJobRequest{ID: req.ID})
}

// SetJobMessage updates the informational message for a job.
func (s *Service) SetJobMessage(ctx context.Context, req SetJobMessageRequest) (JobResponse, error) {
	job, err := s.GetJob(ctx, GetJobRequest{ID: req.ID})
	if err != nil {
		return JobResponse{}, err
	}

	updated := job
	updated.Message = req.Message
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

	stmt := Jobs.UPDATE(Jobs.Message, Jobs.UpdatedAt).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(req.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to set job message", err)
	}

	return s.GetJob(ctx, GetJobRequest{ID: req.ID})
}

// SetJobResult sets the job result metadata and optional message.
func (s *Service) SetJobResult(ctx context.Context, req SetJobResultRequest) (JobResponse, error) {
	job, err := s.GetJob(ctx, GetJobRequest{ID: req.ID})
	if err != nil {
		return JobResponse{}, err
	}

	updated := job
	updated.Message = req.Message
	updated.ErrorMessage = req.ErrorMessage
	updated.JSONResult = ensureJSON(req.JSONResult)
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

	stmt := Jobs.UPDATE(
		Jobs.Message,
		Jobs.ErrorMessage,
		Jobs.JSONResult,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(req.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to set job result", err)
	}

	return s.GetJob(ctx, GetJobRequest{ID: req.ID})
}
