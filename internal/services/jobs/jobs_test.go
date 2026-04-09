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

// Helper functions for precheck tests
func createPrecheckTables(t *testing.T, db *sql.DB) {
	// Create devices table
	_, err := db.Exec(`
		CREATE TABLE devices (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			screen_width INTEGER NOT NULL,
			screen_height INTEGER NOT NULL,
			min_image_width INTEGER NOT NULL DEFAULT 0,
			max_image_width INTEGER NOT NULL DEFAULT 0,
			min_image_height INTEGER NOT NULL DEFAULT 0,
			max_image_height INTEGER NOT NULL DEFAULT 0,
			min_filesize INTEGER NOT NULL DEFAULT 0,
			max_filesize INTEGER NOT NULL DEFAULT 0,
			is_adult_allowed INTEGER NOT NULL DEFAULT 0,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			aspect_ratio_tolerance REAL NOT NULL DEFAULT 0.15,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create devices table: %v", err)
	}

	// Create sources table
	_, err = db.Exec(`
		CREATE TABLE sources (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			source_type TEXT NOT NULL,
			params TEXT NOT NULL DEFAULT '{}',
			lookup_count INTEGER NOT NULL DEFAULT 0,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create sources table: %v", err)
	}

	// Create device_source_subscriptions table
	_, err = db.Exec(`
		CREATE TABLE device_source_subscriptions (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			source_id TEXT NOT NULL,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
			FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("create device_source_subscriptions table: %v", err)
	}

	_, err = db.Exec(`
		CREATE UNIQUE INDEX idx_device_source_subscriptions_device_source 
		ON device_source_subscriptions(device_id, source_id)
	`)
	if err != nil {
		t.Fatalf("create subscription index: %v", err)
	}
}

func insertTestDevice(t *testing.T, db *sql.DB, id, name, slug string, isEnabled bool) {
	enabled := 0
	if isEnabled {
		enabled = 1
	}
	_, err := db.Exec(`
		INSERT INTO devices (id, name, slug, screen_width, screen_height, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, 1920, 1080, ?, ?, ?)`,
		id, name, slug, enabled, time.Now().UnixMilli(), time.Now().UnixMilli(),
	)
	if err != nil {
		t.Fatalf("insert test device: %v", err)
	}
}

func insertTestSource(t *testing.T, db *sql.DB, id, name string, isEnabled bool) {
	enabled := 0
	if isEnabled {
		enabled = 1
	}
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, 'booru', '{}', 0, ?, ?, ?)`,
		id, name, enabled, time.Now().UnixMilli(), time.Now().UnixMilli(),
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}
}

func insertTestSubscription(t *testing.T, db *sql.DB, id, deviceID, sourceID string, isEnabled bool) {
	enabled := 0
	if isEnabled {
		enabled = 1
	}
	_, err := db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, deviceID, sourceID, enabled, time.Now().UnixMilli(), time.Now().UnixMilli(),
	)
	if err != nil {
		t.Fatalf("insert test subscription: %v", err)
	}
}

// --- Precheck Tests ---

func TestCheckSourceAndSubscriptions_SourceDisabled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create a disabled source
	sourceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "disabled-source", false)

	result, err := svc.CheckSourceAndSubscriptions(ctx, sourceID)
	if err != nil {
		t.Fatalf("CheckSourceAndSubscriptions failed: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed to be false for disabled source")
	}
	if result.SourceEnabled {
		t.Error("expected SourceEnabled to be false")
	}
	if result.HasEnabledDevices {
		t.Error("expected HasEnabledDevices to be false")
	}
	if result.Message != "Source is disabled; job skipped" {
		t.Errorf("expected message 'Source is disabled; job skipped', got '%s'", result.Message)
	}
}

func TestCheckSourceAndSubscriptions_NoSubscriptions(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create an enabled source with no subscriptions
	sourceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "enabled-source", true)

	result, err := svc.CheckSourceAndSubscriptions(ctx, sourceID)
	if err != nil {
		t.Fatalf("CheckSourceAndSubscriptions failed: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed to be false when no subscriptions exist")
	}
	if !result.SourceEnabled {
		t.Error("expected SourceEnabled to be true")
	}
	if result.HasEnabledDevices {
		t.Error("expected HasEnabledDevices to be false")
	}
	if result.Message != "No enabled devices subscribed to source; job skipped" {
		t.Errorf("expected message 'No enabled devices subscribed to source; job skipped', got '%s'", result.Message)
	}
}

func TestCheckSourceAndSubscriptions_SubscriptionDisabled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create enabled source and device
	sourceID, _ := dbtypes.NewUUIDV7()
	deviceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "enabled-source", true)
	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device", true)

	// Create a disabled subscription
	subID, _ := dbtypes.NewUUIDV7()
	insertTestSubscription(t, db, subID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), false)

	result, err := svc.CheckSourceAndSubscriptions(ctx, sourceID)
	if err != nil {
		t.Fatalf("CheckSourceAndSubscriptions failed: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed to be false when subscription is disabled")
	}
	if !result.SourceEnabled {
		t.Error("expected SourceEnabled to be true")
	}
	if result.HasEnabledDevices {
		t.Error("expected HasEnabledDevices to be false when subscription is disabled")
	}
}

func TestCheckSourceAndSubscriptions_DeviceDisabled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create enabled source and disabled device
	sourceID, _ := dbtypes.NewUUIDV7()
	deviceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "enabled-source", true)
	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device", false)

	// Create an enabled subscription to disabled device
	subID, _ := dbtypes.NewUUIDV7()
	insertTestSubscription(t, db, subID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	result, err := svc.CheckSourceAndSubscriptions(ctx, sourceID)
	if err != nil {
		t.Fatalf("CheckSourceAndSubscriptions failed: %v", err)
	}

	if result.CanProceed {
		t.Error("expected CanProceed to be false when device is disabled")
	}
	if !result.SourceEnabled {
		t.Error("expected SourceEnabled to be true")
	}
	if result.HasEnabledDevices {
		t.Error("expected HasEnabledDevices to be false when device is disabled")
	}
}

func TestCheckSourceAndSubscriptions_Passes(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create enabled source and device
	sourceID, _ := dbtypes.NewUUIDV7()
	deviceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "enabled-source", true)
	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device", true)

	// Create an enabled subscription
	subID, _ := dbtypes.NewUUIDV7()
	insertTestSubscription(t, db, subID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	result, err := svc.CheckSourceAndSubscriptions(ctx, sourceID)
	if err != nil {
		t.Fatalf("CheckSourceAndSubscriptions failed: %v", err)
	}

	if !result.CanProceed {
		t.Error("expected CanProceed to be true when source and subscription are enabled")
	}
	if !result.SourceEnabled {
		t.Error("expected SourceEnabled to be true")
	}
	if !result.HasEnabledDevices {
		t.Error("expected HasEnabledDevices to be true")
	}
	if result.Message != "Precheck passed" {
		t.Errorf("expected message 'Precheck passed', got '%s'", result.Message)
	}
}

func TestCheckSourceAndSubscriptions_MissingSource(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Try to check a non-existent source
	fakeSourceID, _ := dbtypes.NewUUIDV7()
	_, err := svc.CheckSourceAndSubscriptions(ctx, fakeSourceID)
	if err == nil {
		t.Error("expected error for missing source, got nil")
	}
}

func TestPrecheckAndStartJob_SkippedForNonSourceSyncJobs(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create a source_download job (non-source_sync)
	input := &CreateJobRequest{
		JobType:     JobTypeSourceDownload,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Precheck should skip and start the job directly
	started, canProceed, err := svc.PrecheckAndStartJob(ctx, *job.ID)
	if err != nil {
		t.Fatalf("PrecheckAndStartJob failed: %v", err)
	}

	if !canProceed {
		t.Error("expected canProceed to be true for non-source_sync jobs")
	}
	if started.Status != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, started.Status)
	}
}

func TestPrecheckAndStartJob_SkippedForSourceSyncWithoutSourceID(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create a source_sync job without source_id
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Precheck should skip and start the job directly (no source_id)
	started, canProceed, err := svc.PrecheckAndStartJob(ctx, *job.ID)
	if err != nil {
		t.Fatalf("PrecheckAndStartJob failed: %v", err)
	}

	if !canProceed {
		t.Error("expected canProceed to be true for source_sync jobs without source_id")
	}
	if started.Status != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, started.Status)
	}
}

func TestPrecheckAndStartJob_FailsAndCompletesWhenSourceDisabled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create a disabled source
	sourceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "disabled-source", false)

	// Create a source_sync job for the disabled source
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		SourceID:    &sourceID,
		SourceName:  "disabled-source",
		SourceType:  "booru",
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Precheck should fail and complete the job immediately
	completed, canProceed, err := svc.PrecheckAndStartJob(ctx, *job.ID)
	if err != nil {
		t.Fatalf("PrecheckAndStartJob failed: %v", err)
	}

	if canProceed {
		t.Error("expected canProceed to be false for disabled source")
	}
	if completed.Status != StatusSucceeded {
		t.Errorf("expected status %s, got %s", StatusSucceeded, completed.Status)
	}
	if completed.Message == nil || *completed.Message != "Source is disabled; job skipped" {
		msg := ""
		if completed.Message != nil {
			msg = *completed.Message
		}
		t.Errorf("expected message 'Source is disabled; job skipped', got '%s'", msg)
	}
}

func TestPrecheckAndStartJob_FailsAndCompletesWhenNoSubscriptions(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create an enabled source with no subscriptions
	sourceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "enabled-source", true)

	// Create a source_sync job for the source
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		SourceID:    &sourceID,
		SourceName:  "enabled-source",
		SourceType:  "booru",
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Precheck should fail and complete the job immediately
	completed, canProceed, err := svc.PrecheckAndStartJob(ctx, *job.ID)
	if err != nil {
		t.Fatalf("PrecheckAndStartJob failed: %v", err)
	}

	if canProceed {
		t.Error("expected canProceed to be false when no subscriptions exist")
	}
	if completed.Status != StatusSucceeded {
		t.Errorf("expected status %s, got %s", StatusSucceeded, completed.Status)
	}
	if completed.Message == nil || *completed.Message != "No enabled devices subscribed to source; job skipped" {
		msg := ""
		if completed.Message != nil {
			msg = *completed.Message
		}
		t.Errorf("expected message 'No enabled devices subscribed to source; job skipped', got '%s'", msg)
	}
}

func TestPrecheckAndStartJob_PassesAndStartsWhenAllEnabled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create enabled source and device
	sourceID, _ := dbtypes.NewUUIDV7()
	deviceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "enabled-source", true)
	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device", true)

	// Create an enabled subscription
	subID, _ := dbtypes.NewUUIDV7()
	insertTestSubscription(t, db, subID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	// Create a source_sync job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		SourceID:    &sourceID,
		SourceName:  "enabled-source",
		SourceType:  "booru",
		TriggerKind: TriggerKindManual,
		RunAfter:    time.Now().UTC(),
	}
	job, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Precheck should pass and start the job
	started, canProceed, err := svc.PrecheckAndStartJob(ctx, *job.ID)
	if err != nil {
		t.Fatalf("PrecheckAndStartJob failed: %v", err)
	}

	if !canProceed {
		t.Error("expected canProceed to be true when source and subscription are enabled")
	}
	if started.Status != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, started.Status)
	}
}

func TestPrecheckAndStartJob_JobNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Try to precheck a non-existent job
	fakeJobID, _ := dbtypes.NewUUIDV7()
	_, _, err := svc.PrecheckAndStartJob(ctx, fakeJobID)
	if err == nil {
		t.Error("expected error for missing job, got nil")
	}
}

// --- Schedule-Triggered Precheck Tests ---

func TestPrecheckAndStartJob_ScheduleTrigger_SkippedWhenNoEnabledDevices(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create an enabled source with no subscriptions
	sourceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "enabled-source", true)

	// Create a scheduled source_sync job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		SourceID:    &sourceID,
		SourceName:  "enabled-source",
		SourceType:  "booru",
		TriggerKind: TriggerKindSchedule,
		RunAfter:    time.Now().UTC(),
	}
	job, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Precheck should fail and complete the job immediately
	completed, canProceed, err := svc.PrecheckAndStartJob(ctx, *job.ID)
	if err != nil {
		t.Fatalf("PrecheckAndStartJob failed: %v", err)
	}

	if canProceed {
		t.Error("expected canProceed to be false when no enabled devices are subscribed")
	}
	if completed.Status != StatusSucceeded {
		t.Errorf("expected status %s, got %s", StatusSucceeded, completed.Status)
	}
	if completed.Message == nil || *completed.Message != "No enabled devices subscribed to source; job skipped" {
		msg := ""
		if completed.Message != nil {
			msg = *completed.Message
		}
		t.Errorf("expected message 'No enabled devices subscribed to source; job skipped', got '%s'", msg)
	}
}

func TestPrecheckAndStartJob_ScheduleTrigger_PassesWhenAllEnabled(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)
	createPrecheckTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create enabled source and device
	sourceID, _ := dbtypes.NewUUIDV7()
	deviceID, _ := dbtypes.NewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "enabled-source", true)
	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device", true)

	// Create an enabled subscription
	subID, _ := dbtypes.NewUUIDV7()
	insertTestSubscription(t, db, subID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	// Create a scheduled source_sync job
	input := &CreateJobRequest{
		JobType:     JobTypeSourceSync,
		SourceID:    &sourceID,
		SourceName:  "enabled-source",
		SourceType:  "booru",
		TriggerKind: TriggerKindSchedule,
		RunAfter:    time.Now().UTC(),
	}
	job, err := svc.CreateJob(ctx, *input)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Precheck should pass and start the job
	started, canProceed, err := svc.PrecheckAndStartJob(ctx, *job.ID)
	if err != nil {
		t.Fatalf("PrecheckAndStartJob failed: %v", err)
	}

	if !canProceed {
		t.Error("expected canProceed to be true when source and subscription are enabled")
	}
	if started.Status != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, started.Status)
	}
}

// --- Existing Tests ---

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

// TestListJobs_TotalCount verifies Total reflects all matching rows independent of pagination.
func TestListJobs_TotalCount(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create 5 queued jobs
	for i := 0; i < 5; i++ {
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

	// Create 3 running jobs
	for i := 0; i < 3; i++ {
		input := &CreateJobRequest{
			JobType:     JobTypeSourceSync,
			TriggerKind: TriggerKindManual,
			RunAfter:    time.Now().UTC(),
		}
		job, _ := svc.CreateJob(ctx, *input)
		svc.StartJob(ctx, StartJobRequest{ID: *job.ID})
	}

	// Test: Total should be 8 for all jobs
	respAll, err := svc.ListJobs(ctx, ListJobsRequest{
		Pagination: &dbtypes.CursorPaginationRequest{Limit: intPtr(10), Offset: intPtr(0)},
	})
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if respAll.Total != 8 {
		t.Errorf("expected Total=8 for all jobs, got %d", respAll.Total)
	}

	// Test: Total should be 5 when filtering by StatusQueued
	queuedStatus := StatusQueued
	respQueued, err := svc.ListJobs(ctx, ListJobsRequest{
		Status:     &queuedStatus,
		Pagination: &dbtypes.CursorPaginationRequest{Limit: intPtr(10), Offset: intPtr(0)},
	})
	if err != nil {
		t.Fatalf("ListJobs with status filter failed: %v", err)
	}
	if respQueued.Total != 5 {
		t.Errorf("expected Total=5 for queued jobs, got %d", respQueued.Total)
	}
}

// TestListJobs_TotalNotAffectedByPagination verifies Total is independent of Limit.
func TestListJobs_TotalNotAffectedByPagination(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createJobsTable(t, db)

	svc := NewService(db)
	ctx := context.Background()

	// Create 10 jobs
	for i := 0; i < 10; i++ {
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

	// First page with small limit
	limit := 3
	resp, err := svc.ListJobs(ctx, ListJobsRequest{
		Pagination: &dbtypes.CursorPaginationRequest{Limit: intPtr(limit), Offset: intPtr(0)},
	})
	if err != nil {
		t.Fatalf("ListJobs with limit failed: %v", err)
	}

	// Items should be limited to 3
	if len(resp.Items) > 3 {
		t.Errorf("expected at most 3 items, got %d", len(resp.Items))
	}
	// But Total should still be 10
	if resp.Total != 10 {
		t.Errorf("expected Total=10 (independent of pagination), got %d", resp.Total)
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
