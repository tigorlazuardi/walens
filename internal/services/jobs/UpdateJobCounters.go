package jobs

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// IncrementJobCounters atomically increments job counters using MODEL update semantics.
func (s *Service) IncrementJobCounters(ctx context.Context, id dbtypes.UUID, deltas UpdateJobCountersInput) (*model.Jobs, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.Status != StatusRunning {
		return nil, ErrJobNotRunning
	}

	updated := *job
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

	if deltas.DownloadedImageCount != nil {
		updated.DownloadedImageCount += *deltas.DownloadedImageCount
	}
	if deltas.ReusedImageCount != nil {
		updated.ReusedImageCount += *deltas.ReusedImageCount
	}
	if deltas.HardlinkedImageCount != nil {
		updated.HardlinkedImageCount += *deltas.HardlinkedImageCount
	}
	if deltas.CopiedImageCount != nil {
		updated.CopiedImageCount += *deltas.CopiedImageCount
	}
	if deltas.StoredImageCount != nil {
		updated.StoredImageCount += *deltas.StoredImageCount
	}
	if deltas.SkippedImageCount != nil {
		updated.SkippedImageCount += *deltas.SkippedImageCount
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
		Jobs.ID.EQ(String(id.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("increment job counters: %w", err)
	}

	return s.GetJob(ctx, id)
}

// SetJobMessage updates the informational message for a job.
func (s *Service) SetJobMessage(ctx context.Context, id dbtypes.UUID, message *string) (*model.Jobs, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return nil, err
	}

	updated := *job
	updated.Message = message
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

	stmt := Jobs.UPDATE(Jobs.Message, Jobs.UpdatedAt).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(id.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("set job message: %w", err)
	}

	return s.GetJob(ctx, id)
}

// SetJobResult sets the job result metadata and optional message.
func (s *Service) SetJobResult(ctx context.Context, input *SetJobResultInput) (*model.Jobs, error) {
	job, err := s.GetJob(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	updated := *job
	updated.Message = input.Message
	updated.ErrorMessage = input.ErrorMessage
	updated.JSONResult = ensureJSON(input.JSONResult)
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()

	stmt := Jobs.UPDATE(
		Jobs.Message,
		Jobs.ErrorMessage,
		Jobs.JSONResult,
		Jobs.UpdatedAt,
	).MODEL(updated).WHERE(
		Jobs.ID.EQ(String(input.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("set job result: %w", err)
	}

	return s.GetJob(ctx, input.ID)
}
