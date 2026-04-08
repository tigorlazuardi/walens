package jobs

import (
	"context"
	"fmt"

	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// CreateJob creates a new job row using the generated Go-Jet model.
func (s *Service) CreateJob(ctx context.Context, input *CreateJobInput) (*model.Jobs, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	if err := validateJobType(input.JobType); err != nil {
		return nil, err
	}
	if err := validateTriggerKind(input.TriggerKind); err != nil {
		return nil, err
	}

	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, fmt.Errorf("generate UUIDv7: %w", err)
	}

	job := model.Jobs{
		ID:                   &id,
		JobType:              input.JobType,
		SourceID:             input.SourceID,
		SourceName:           strPtr(input.SourceName),
		SourceType:           strPtr(input.SourceType),
		Status:               StatusQueued,
		TriggerKind:          input.TriggerKind,
		RunAfter:             dbtypes.NewUnixMilliTime(input.RunAfter),
		RequestedImageCount:  input.RequestedImageCount,
		DownloadedImageCount: 0,
		ReusedImageCount:     0,
		HardlinkedImageCount: 0,
		CopiedImageCount:     0,
		StoredImageCount:     0,
		SkippedImageCount:    0,
		JSONInput:            ensureJSON(input.JSONInput),
		JSONResult:           dbtypes.RawJSON([]byte("{}")),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	stmt := Jobs.INSERT(
		Jobs.ID,
		Jobs.JobType,
		Jobs.SourceID,
		Jobs.SourceName,
		Jobs.SourceType,
		Jobs.Status,
		Jobs.TriggerKind,
		Jobs.RunAfter,
		Jobs.RequestedImageCount,
		Jobs.DownloadedImageCount,
		Jobs.ReusedImageCount,
		Jobs.HardlinkedImageCount,
		Jobs.CopiedImageCount,
		Jobs.StoredImageCount,
		Jobs.SkippedImageCount,
		Jobs.JSONInput,
		Jobs.JSONResult,
		Jobs.CreatedAt,
		Jobs.UpdatedAt,
	).MODEL(job)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("insert job: %w", err)
	}

	return s.GetJob(ctx, id)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
