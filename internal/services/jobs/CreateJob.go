package jobs

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// CreateJob creates a new job row using the generated Go-Jet model.
func (s *Service) CreateJob(ctx context.Context, req CreateJobRequest) (JobResponse, error) {
	if err := validateJobType(req.JobType); err != nil {
		return JobResponse{}, huma.Error400BadRequest(err.Error(), err)
	}
	if err := validateTriggerKind(req.TriggerKind); err != nil {
		return JobResponse{}, huma.Error400BadRequest(err.Error(), err)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return JobResponse{}, huma.Error500InternalServerError("failed to generate job id", err)
	}

	job := model.Jobs{
		ID:                   id,
		JobType:              req.JobType,
		SourceID:             req.SourceID,
		SourceName:           strPtr(req.SourceName),
		SourceType:           strPtr(req.SourceType),
		Status:               StatusQueued,
		TriggerKind:          req.TriggerKind,
		RunAfter:             dbtypes.NewUnixMilliTime(req.RunAfter),
		RequestedImageCount:  req.RequestedImageCount,
		DownloadedImageCount: 0,
		ReusedImageCount:     0,
		HardlinkedImageCount: 0,
		CopiedImageCount:     0,
		StoredImageCount:     0,
		SkippedImageCount:    0,
		JSONInput:            ensureJSON(req.JSONInput),
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
		return JobResponse{}, huma.Error500InternalServerError("failed to create job", err)
	}

	return s.GetJob(ctx, GetJobRequest{ID: id})
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
