// Package jobs provides job persistence, state transitions, and recovery for source-triggered work.
package jobs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/walens/walens/internal/db/generated/model"
	"github.com/walens/walens/internal/dbtypes"
)

// Job type constants.
const (
	JobTypeSourceSync     = "source_sync"
	JobTypeSourceDownload = "source_download"
)

// Status constants.
const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Trigger kind constants.
const (
	TriggerKindManual   = "manual"
	TriggerKindSchedule = "schedule"
	TriggerKindRecovery = "recovery"
)

// Errors.
var (
	ErrDBUnavailable     = errors.New("database unavailable")
	ErrJobNotFound       = errors.New("job not found")
	ErrInvalidState      = errors.New("invalid job state")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrJobNotRunning     = errors.New("job is not running")
)

// CreateJobInput contains the fields needed to create a new job.
type CreateJobInput struct {
	JobType             string          `json:"job_type" doc:"Job type: source_sync or source_download."`
	SourceID            *dbtypes.UUID   `json:"source_id,omitempty" doc:"Reference to source this job is for."`
	SourceName          string          `json:"source_name" doc:"Denormalized source name for job history."`
	SourceType          string          `json:"source_type" doc:"Denormalized source type for job history."`
	TriggerKind         string          `json:"trigger_kind" doc:"What triggered this job: manual, schedule, or recovery."`
	RunAfter            time.Time       `json:"run_after" doc:"When the job should run."`
	RequestedImageCount int64           `json:"requested_image_count" doc:"How many images were requested to fetch."`
	JSONInput           json.RawMessage `json:"json_input" doc:"Job input parameters as JSON."`
}

// UpdateJobStateInput contains fields for updating job state.
type UpdateJobStateInput struct {
	ID     dbtypes.UUID `json:"id" doc:"Job ID."`
	Status string       `json:"status" doc:"New status."`
}

// UpdateJobCountersInput contains fields for updating job counters.
type UpdateJobCountersInput struct {
	ID                   dbtypes.UUID `json:"id" doc:"Job ID."`
	DownloadedImageCount *int64       `json:"downloaded_image_count,omitempty" doc:"Delta to add to downloaded count."`
	ReusedImageCount     *int64       `json:"reused_image_count,omitempty" doc:"Delta to add to reused count."`
	HardlinkedImageCount *int64       `json:"hardlinked_image_count,omitempty" doc:"Delta to add to hardlinked count."`
	CopiedImageCount     *int64       `json:"copied_image_count,omitempty" doc:"Delta to add to copied count."`
	StoredImageCount     *int64       `json:"stored_image_count,omitempty" doc:"Delta to add to stored count."`
	SkippedImageCount    *int64       `json:"skipped_image_count,omitempty" doc:"Delta to add to skipped count."`
}

// SetJobResultInput contains fields for setting job result.
type SetJobResultInput struct {
	ID           dbtypes.UUID    `json:"id" doc:"Job ID."`
	Message      *string         `json:"message,omitempty" doc:"Informational message about the job result."`
	ErrorMessage *string         `json:"error_message,omitempty" doc:"Error message if the job failed."`
	JSONResult   json.RawMessage `json:"json_result,omitempty" doc:"Job result metadata as JSON."`
}

// ListJobsInput contains filters for listing jobs.
type ListJobsInput struct {
	Status      *string       `json:"status,omitempty" doc:"Filter by status."`
	JobType     *string       `json:"job_type,omitempty" doc:"Filter by job type."`
	SourceID    *dbtypes.UUID `json:"source_id,omitempty" doc:"Filter by source ID."`
	TriggerKind *string       `json:"trigger_kind,omitempty" doc:"Filter by trigger kind."`
	Limit       int           `json:"limit" doc:"Maximum number of jobs to return."`
	Offset      int           `json:"offset" doc:"Offset for pagination."`
}

// ListJobsResponse contains the list of jobs and total count.
type ListJobsResponse struct {
	Items []model.Jobs `json:"items" doc:"List of jobs."`
	Total int64        `json:"total" doc:"Total number of jobs matching filters."`
}

// Service provides job persistence and state management.
type Service struct {
	db *sql.DB
}

// NewService creates a new jobs service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func ensureJSON(raw json.RawMessage) dbtypes.RawJSON {
	if len(raw) == 0 {
		return dbtypes.RawJSON([]byte("{}"))
	}
	return dbtypes.RawJSON(raw)
}

// validateJobType checks if the job type is valid.
func validateJobType(jobType string) error {
	switch jobType {
	case JobTypeSourceSync, JobTypeSourceDownload:
		return nil
	default:
		return fmt.Errorf("%w: invalid job_type: %s", ErrInvalidState, jobType)
	}
}

// validateStatus checks if the status is valid.
func validateStatus(status string) error {
	switch status {
	case StatusQueued, StatusRunning, StatusSucceeded, StatusFailed, StatusCancelled:
		return nil
	default:
		return fmt.Errorf("%w: invalid status: %s", ErrInvalidState, status)
	}
}

// validateTriggerKind checks if the trigger kind is valid.
func validateTriggerKind(kind string) error {
	switch kind {
	case TriggerKindManual, TriggerKindSchedule, TriggerKindRecovery:
		return nil
	default:
		return fmt.Errorf("%w: invalid trigger_kind: %s", ErrInvalidState, kind)
	}
}

// isValidTransition checks if a state transition is valid.
func isValidTransition(from, to string) bool {
	switch from {
	case StatusQueued:
		return to == StatusRunning || to == StatusCancelled
	case StatusRunning:
		return to == StatusSucceeded || to == StatusFailed || to == StatusCancelled
	case StatusSucceeded, StatusFailed, StatusCancelled:
		return false
	default:
		return false
	}
}
