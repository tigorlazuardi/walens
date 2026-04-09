package jobs

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// PrecheckResult contains the result of the job precheck
type PrecheckResult struct {
	CanProceed        bool
	SourceEnabled     bool
	HasEnabledDevices bool
	Message           string
}

// CheckSourceAndSubscriptions performs pre-run checks for a job.
// It verifies:
// 1. The source is enabled
// 2. At least one enabled device is subscribed to the source
//
// Returns PrecheckResult with CanProceed=false if checks fail,
// along with an informational message explaining why.
func (s *Service) CheckSourceAndSubscriptions(ctx context.Context, sourceID dbtypes.UUID) (*PrecheckResult, error) {
	result := &PrecheckResult{
		CanProceed:        false,
		SourceEnabled:     false,
		HasEnabledDevices: false,
	}

	// Check if source is enabled
	var source struct {
		IsEnabled dbtypes.BoolInt `alias:"is_enabled"`
	}
	stmt := SELECT(Sources.IsEnabled.AS("is_enabled")).
		FROM(Sources).
		WHERE(Sources.ID.EQ(String(sourceID.UUID.String()))).
		LIMIT(1)

	if err := stmt.QueryContext(ctx, s.db, &source); err != nil {
		return nil, fmt.Errorf("check source enabled: %w", err)
	}

	result.SourceEnabled = bool(source.IsEnabled)
	if !result.SourceEnabled {
		result.Message = "Source is disabled; job skipped"
		return result, nil
	}

	// Check if at least one enabled device is subscribed
	var count struct {
		Count int64 `alias:"count"`
	}
	countStmt := SELECT(
		COUNT(DeviceSourceSubscriptions.ID).AS("count"),
	).FROM(
		DeviceSourceSubscriptions.
			INNER_JOIN(Devices, Devices.ID.EQ(DeviceSourceSubscriptions.DeviceID)),
	).WHERE(
		DeviceSourceSubscriptions.SourceID.EQ(String(sourceID.UUID.String())).
			AND(DeviceSourceSubscriptions.IsEnabled.EQ(Int(1))).
			AND(Devices.IsEnabled.EQ(Int(1))),
	)

	if err := countStmt.QueryContext(ctx, s.db, &count); err != nil {
		return nil, fmt.Errorf("check enabled subscriptions: %w", err)
	}

	result.HasEnabledDevices = count.Count > 0
	if !result.HasEnabledDevices {
		result.Message = "No enabled devices subscribed to source; job skipped"
		return result, nil
	}

	result.CanProceed = true
	result.Message = "Precheck passed"
	return result, nil
}

// PrecheckAndStartJob performs precheck, starts the job if passed,
// or completes it with an informational message if failed.
// Returns (job, canProceed, error)
func (s *Service) PrecheckAndStartJob(ctx context.Context, jobID dbtypes.UUID) (*model.Jobs, bool, error) {
	// Get the job details
	job, err := s.GetJob(ctx, GetJobRequest{ID: jobID})
	if err != nil {
		return nil, false, fmt.Errorf("get job for precheck: %w", err)
	}

	// Only perform precheck for source_sync jobs with a source_id
	if job.JobType != JobTypeSourceSync || job.SourceID == nil {
		// Non-source jobs or jobs without source can proceed
		started, err := s.StartJob(ctx, StartJobRequest{ID: jobID})
		if err != nil {
			return nil, false, fmt.Errorf("start job: %w", err)
		}
		return &started, true, nil
	}

	// Perform precheck
	precheck, err := s.CheckSourceAndSubscriptions(ctx, *job.SourceID)
	if err != nil {
		return nil, false, fmt.Errorf("precheck failed: %w", err)
	}

	// Rule 1: If source is disabled, complete job with message and exit
	if !precheck.SourceEnabled {
		// Start the job
		started, err := s.StartJob(ctx, StartJobRequest{ID: jobID})
		if err != nil {
			return nil, false, fmt.Errorf("start job: %w", err)
		}

		// Complete immediately with informational message
		msg := "Source is disabled; job skipped"
		completed, err := s.CompleteJob(ctx, CompleteJobRequest{
			ID:      jobID,
			Message: &msg,
		})
		if err != nil {
			return &started, false, fmt.Errorf("complete skipped job: %w", err)
		}
		return &completed, false, nil
	}

	// Rule 3: Source is enabled but no subscribed devices
	if !precheck.HasEnabledDevices {
		// Start the job
		started, err := s.StartJob(ctx, StartJobRequest{ID: jobID})
		if err != nil {
			return nil, false, fmt.Errorf("start job: %w", err)
		}

		// Complete immediately with informational message
		msg := "No enabled devices subscribed to source; job skipped"
		completed, err := s.CompleteJob(ctx, CompleteJobRequest{
			ID:      jobID,
			Message: &msg,
		})
		if err != nil {
			return &started, false, fmt.Errorf("complete skipped job: %w", err)
		}
		return &completed, false, nil
	}

	// Rule 4: Source is enabled and has devices, start the job normally
	started, err := s.StartJob(ctx, StartJobRequest{ID: jobID})
	if err != nil {
		return nil, false, fmt.Errorf("start job: %w", err)
	}
	return &started, true, nil
}
