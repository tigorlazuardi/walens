package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
	_ "modernc.org/sqlite"
)

func assertHumaErrorStatus(t *testing.T, err error, expectedStatus int) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with status %d, got nil", expectedStatus)
	}

	var humaErr *huma.ErrorModel
	if !errors.As(err, &humaErr) {
		t.Fatalf("expected *huma.ErrorModel, got %T: %v", err, err)
	}

	if humaErr.GetStatus() != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, humaErr.GetStatus())
	}
}

func intPtr(v int) *int { return &v }

func openTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func createJobsTable(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			job_type TEXT NOT NULL,
			source_id TEXT,
			source_name TEXT,
			source_type TEXT,
			status TEXT NOT NULL,
			trigger_kind TEXT NOT NULL,
			run_after INTEGER NOT NULL,
			started_at INTEGER,
			finished_at INTEGER,
			duration_ms INTEGER,
			requested_image_count INTEGER NOT NULL DEFAULT 0,
			downloaded_image_count INTEGER NOT NULL DEFAULT 0,
			reused_image_count INTEGER NOT NULL DEFAULT 0,
			hardlinked_image_count INTEGER NOT NULL DEFAULT 0,
			copied_image_count INTEGER NOT NULL DEFAULT 0,
			stored_image_count INTEGER NOT NULL DEFAULT 0,
			skipped_image_count INTEGER NOT NULL DEFAULT 0,
			message TEXT,
			error_message TEXT,
			json_input TEXT NOT NULL DEFAULT '{}',
			json_result TEXT NOT NULL DEFAULT '{}',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create jobs table: %v", err)
	}
}

func TestCreateJob(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	sourceID, _ := dbtypes.NewUUIDV7()
	input := &CreateJobRequest{
		JobType:             JobTypeSourceSync,
		SourceID:            &sourceID,
		SourceName:          "test-source",
		SourceType:          "booru",
		TriggerKind:         TriggerKindManual,
		RunAfter:            time.Now().UTC(),
		RequestedImageCount: 100,
		JSONInput:           json.RawMessage(`{"tags":["landscape"]}`),
	}

	job, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	if job.JobType != JobTypeSourceSync {
		t.Errorf("expected job_type %s, got %s", JobTypeSourceSync, job.JobType)
	}
	if job.Status != StatusQueued {
		t.Errorf("expected status %s, got %s", StatusQueued, job.Status)
	}
	if job.TriggerKind != TriggerKindManual {
		t.Errorf("expected trigger_kind %s, got %s", TriggerKindManual, job.TriggerKind)
	}
	if job.SourceID == nil || job.SourceID.UUID != sourceID.UUID {
		t.Error("expected source_id to match")
	}
	if job.RequestedImageCount != 100 {
		t.Errorf("expected requested_image_count 100, got %d", job.RequestedImageCount)
	}
}

func TestCreateJobInvalidJobType(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	input := &CreateJobRequest{
		JobType:     "invalid_type",
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}

	_, err := svc.CreateJob(ctx, *input)
	assertHumaErrorStatus(t, err, 400)
}

func TestCreateJobInvalidTriggerKind(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: "invalid_trigger",
		RunAfter:    time.Now().UTC(),
	}

	_, err := svc.CreateJob(ctx, *input)
	assertHumaErrorStatus(t, err, 400)
}

func TestGetJob(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create a job first
	input := &CreateJobRequest{
		JobType:     JobTypeSourceDownload,
		TriggerKind: TriggerKindSchedule,
		RunAfter:    time.Now().UTC(),
	}
	created, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Get the job
	job, err := svc.GetJob(ctx, GetJobRequest{ID: *created.ID})
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}

	if job.ID == nil || created.ID == nil || job.ID.UUID != created.ID.UUID {
		t.Error("job ID mismatch")
	}
	if job.JobType != JobTypeSourceDownload {
		t.Errorf("expected job_type %s, got %s", JobTypeSourceDownload, job.JobType)
	}
}

func TestGetJobNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Try to get non-existent job
	fakeID, _ := dbtypes.NewUUIDV7()
	_, err := svc.GetJob(ctx, GetJobRequest{ID: fakeID})
	assertHumaErrorStatus(t, err, 404)
}

func TestListJobs(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create multiple jobs
	for i := 0; i < 3; i++ {
		input := &CreateJobRequest{
			JobType:     JobTypeSourceSync,
			TriggerKind: TriggerKindManual,
			RunAfter:    time.Now().UTC(),
		}
		_, err := svc.CreateJob(ctx, *input)
		if err != nil {
			t.Fatalf("CreateJob failed: %v", err)
		}
	}

	// List all jobs
	resp, err := svc.ListJobs(ctx, ListJobsRequest{
		Pagination: &dbtypes.CursorPaginationRequest{Limit: intPtr(10), Offset: intPtr(0)},
	})
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(resp.Items))
	}
}

func TestListJobsWithFilter(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create jobs with different statuses
	statuses := []string{StatusQueued, StatusRunning, StatusSucceeded}
	for _, status := range statuses {
		input := &CreateJobRequest{
			JobType:     JobTypeSourceSync,
			TriggerKind: TriggerKindManual,
			RunAfter:    time.Now().UTC(),
		}
		job, _ := svc.CreateJob(ctx, *input)

		switch status {
		case StatusRunning:
			_, _ = svc.StartJob(ctx, StartJobRequest{ID: *job.ID})
		case StatusSucceeded:
			_, _ = svc.StartJob(ctx, StartJobRequest{ID: *job.ID})
			_, _ = svc.CompleteJob(ctx, CompleteJobRequest{ID: *job.ID, Message: nil, JSONResult: nil})
		}
	}

	// List only queued jobs
	queuedStatus := StatusQueued
	resp, err := svc.ListJobs(ctx, ListJobsRequest{
		Status:     &queuedStatus,
		Pagination: &dbtypes.CursorPaginationRequest{Limit: intPtr(10), Offset: intPtr(0)},
	})
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 queued item, got %d", len(resp.Items))
	}
}

func TestStartJob(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create a job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, _ := svc.CreateJob(ctx, *input)

	// Start the job
	started, err := svc.StartJob(ctx, StartJobRequest{ID: *job.ID})
	if err != nil {
		t.Fatalf("StartJob failed: %v", err)
	}

	if started.Status != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, started.Status)
	}
	if started.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
}

func TestStartJobInvalidTransition(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create, start, and complete a job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, _ := svc.CreateJob(ctx, *input)
	svc.StartJob(ctx, StartJobRequest{ID: *job.ID})
	svc.CompleteJob(ctx, CompleteJobRequest{ID: *job.ID, Message: nil, JSONResult: nil})

	// Try to start completed job
	_, err := svc.StartJob(ctx, StartJobRequest{ID: *job.ID})
	assertHumaErrorStatus(t, err, 400)
}

func TestCompleteJob(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create and start a job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, _ := svc.CreateJob(ctx, *input)
	svc.StartJob(ctx, StartJobRequest{ID: *job.ID})

	// Complete the job
	message := "Job completed successfully"
	result := json.RawMessage(`{"images_processed": 10}`)
	completed, err := svc.CompleteJob(ctx, CompleteJobRequest{ID: *job.ID, Message: &message, JSONResult: result})
	if err != nil {
		t.Fatalf("CompleteJob failed: %v", err)
	}

	if completed.Status != StatusSucceeded {
		t.Errorf("expected status %s, got %s", StatusSucceeded, completed.Status)
	}
	if completed.FinishedAt == nil {
		t.Error("expected finished_at to be set")
	}
	if completed.DurationMs == nil {
		t.Error("expected duration_ms to be set")
	}
	if completed.Message == nil || *completed.Message != message {
		t.Error("expected message to match")
	}
}

func TestFailJob(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create and start a job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, _ := svc.CreateJob(ctx, *input)
	svc.StartJob(ctx, StartJobRequest{ID: *job.ID})

	// Fail the job
	errorMsg := "Network timeout"
	result := json.RawMessage(`{"error_code": "TIMEOUT"}`)
	failed, err := svc.FailJob(ctx, FailJobRequest{ID: *job.ID, ErrorMessage: errorMsg, JSONResult: result})
	if err != nil {
		t.Fatalf("FailJob failed: %v", err)
	}

	if failed.Status != StatusFailed {
		t.Errorf("expected status %s, got %s", StatusFailed, failed.Status)
	}
	if failed.ErrorMessage == nil || *failed.ErrorMessage != errorMsg {
		t.Error("expected error_message to match")
	}
}

func TestCancelJob(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create a queued job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, _ := svc.CreateJob(ctx, *input)

	// Cancel the job
	message := "Cancelled by user"
	cancelled, err := svc.CancelJob(ctx, CancelJobRequest{ID: *job.ID, Message: &message})
	if err != nil {
		t.Fatalf("CancelJob failed: %v", err)
	}

	if cancelled.Status != StatusCancelled {
		t.Errorf("expected status %s, got %s", StatusCancelled, cancelled.Status)
	}
}

func TestIncrementJobCounters(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create and start a job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, _ := svc.CreateJob(ctx, *input)
	svc.StartJob(ctx, StartJobRequest{ID: *job.ID})

	// Increment counters
	downloaded := int64(5)
	reused := int64(3)
	updated, err := svc.IncrementJobCounters(ctx, IncrementJobCountersRequest{ID: *job.ID, Deltas: UpdateJobCountersRequest{
		DownloadedImageCount: &downloaded,
		ReusedImageCount:     &reused,
	}})
	if err != nil {
		t.Fatalf("IncrementJobCounters failed: %v", err)
	}

	if updated.DownloadedImageCount != 5 {
		t.Errorf("expected downloaded_image_count 5, got %d", updated.DownloadedImageCount)
	}
	if updated.ReusedImageCount != 3 {
		t.Errorf("expected reused_image_count 3, got %d", updated.ReusedImageCount)
	}

	// Increment again
	downloaded2 := int64(2)
	updated2, err := svc.IncrementJobCounters(ctx, IncrementJobCountersRequest{ID: *job.ID, Deltas: UpdateJobCountersRequest{
		DownloadedImageCount: &downloaded2,
	}})
	if err != nil {
		t.Fatalf("IncrementJobCounters failed: %v", err)
	}

	if updated2.DownloadedImageCount != 7 {
		t.Errorf("expected downloaded_image_count 7, got %d", updated2.DownloadedImageCount)
	}
}

func TestIncrementJobCountersNotRunning(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create a queued job (not running)
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, _ := svc.CreateJob(ctx, *input)

	// Try to increment counters
	downloaded := int64(5)
	_, err := svc.IncrementJobCounters(ctx, IncrementJobCountersRequest{ID: *job.ID, Deltas: UpdateJobCountersRequest{
		DownloadedImageCount: &downloaded,
	}})
	assertHumaErrorStatus(t, err, 400)
}

func TestRecoverRunningJobs(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create jobs in different states
	for i := 0; i < 2; i++ {
		input := &CreateJobRequest{
			JobType:     JobTypeSourceSync,
			TriggerKind: TriggerKindManual,
			RunAfter:    time.Now().UTC(),
		}
		job, _ := svc.CreateJob(ctx, *input)
		if i == 0 {
			// Start one job
			svc.StartJob(ctx, StartJobRequest{ID: *job.ID})
		}
	}

	// Recover running jobs
	affected, err := svc.RecoverRunningJobs(ctx)
	if err != nil {
		t.Fatalf("RecoverRunningJobs failed: %v", err)
	}

	if affected != 1 {
		t.Errorf("expected 1 job recovered, got %d", affected)
	}

	// Verify the job is now queued
	resp, _ := svc.ListJobs(ctx, ListJobsRequest{
		Pagination: &dbtypes.CursorPaginationRequest{Limit: intPtr(10), Offset: intPtr(0)},
	})
	for _, job := range resp.Items {
		if job.Status == StatusRunning {
			t.Error("found running job after recovery")
		}
	}
}

func TestGetJobsForRecovery(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create jobs
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	queuedJob, _ := svc.CreateJob(ctx, *input)
	runningJob, _ := svc.CreateJob(ctx, *input)
	svc.StartJob(ctx, StartJobRequest{ID: *runningJob.ID})

	// Get jobs for recovery
	jobs, err := svc.GetJobsForRecovery(ctx)
	if err != nil {
		t.Fatalf("GetJobsForRecovery failed: %v", err)
	}

	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs for recovery, got %d", len(jobs))
	}

	// Verify IDs match
	foundQueued := false
	foundRunning := false
	for _, job := range jobs {
		if job.ID != nil && queuedJob.ID != nil && job.ID.UUID.String() == queuedJob.ID.UUID.String() {
			foundQueued = true
		}
		if job.ID != nil && runningJob.ID != nil && job.ID.UUID.String() == runningJob.ID.UUID.String() {
			foundRunning = true
		}
	}
	if !foundQueued {
		t.Error("queued job not found in recovery list")
	}
	if !foundRunning {
		t.Error("running job not found in recovery list")
	}
}
