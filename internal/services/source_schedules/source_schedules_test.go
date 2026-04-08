package source_schedules

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/walens/walens/internal/dbtypes"
	_ "modernc.org/sqlite"
)

func uuidPtr(id dbtypes.UUID) *dbtypes.UUID { return &id }

// mockScheduler implements SchedulerInterface for testing.
type mockScheduler struct {
	reloadCalled int
	reloadErr    error
}

func (m *mockScheduler) Reload() error {
	m.reloadCalled++
	return m.reloadErr
}

func testDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=temp_store(MEMORY)")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return db, func() { db.Close() }
}

func createTables(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sources (
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

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS source_schedules (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			cron_expr TEXT NOT NULL,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("create source_schedules table: %v", err)
	}
}

func insertTestSource(t *testing.T, db *sql.DB, id, name string) {
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, 'booru', '{}', 0, 1, 1000, 1000)`,
		id, name,
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}
}

// --- Cron Validation Tests ---

func TestValidateCronExprValid(t *testing.T) {
	tests := []struct {
		expr    string
		wantErr bool
	}{
		{"* * * * *", false},     // Every minute
		{"0 * * * *", false},     // Every hour at minute 0
		{"0 0 * * *", false},     // Every day at midnight
		{"0 9 * * 1", false},     // Every Monday at 9am
		{"*/5 * * * *", false},   // Every 5 minutes
		{"0 0 1 * *", false},     // First day of every month
		{"30 4 1,15 * *", false}, // 4:30am on 1st and 15th
		{"0 0 * * 0", false},     // Every Sunday at midnight
		{"0 0 * * 6,0", false},   // Sat and Sun at midnight
	}

	for _, tt := range tests {
		_, err := ValidateCronExpr(tt.expr)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateCronExpr(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
		}
	}
}

func TestValidateCronExprInvalid(t *testing.T) {
	tests := []string{
		"",
		"* * * *",
		"* * * * * *",
		"not a cron",
		"60 * * * *", // minute out of range
		"* 25 * * *", // hour out of range
		"* * 32 * *", // day out of range
		"* * * 13 *", // month out of range
		"* * * * 8",  // weekday out of range
	}

	for _, expr := range tests {
		_, err := ValidateCronExpr(expr)
		if err == nil {
			t.Errorf("ValidateCronExpr(%q) expected error, got nil", expr)
		}
	}
}

// --- Service CRUD Tests ---

func TestServiceListSchedulesEmpty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	sched := NewService(db, nil)
	items, err := sched.ListSchedules(context.Background())
	if err != nil {
		t.Fatalf("ListSchedules failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 schedules, got %d", len(items))
	}
}

func TestServiceListSchedulesWithData(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	_, err := db.Exec(`
		INSERT INTO source_schedules (id, source_id, cron_expr, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001", "01800000-0000-0000-0000-000000000001",
		"0 * * * *", 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test schedule: %v", err)
	}

	sched := NewService(db, nil)
	items, err := sched.ListSchedules(context.Background())
	if err != nil {
		t.Fatalf("ListSchedules failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(items))
	}
	if items[0].CronExpr != "0 * * * *" {
		t.Errorf("expected cron_expr '0 * * * *', got %q", items[0].CronExpr)
	}
}

func TestServiceGetSchedule(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	_, err := db.Exec(`
		INSERT INTO source_schedules (id, source_id, cron_expr, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001", "01800000-0000-0000-0000-000000000001",
		"0 * * * *", 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test schedule: %v", err)
	}

	sched := NewService(db, nil)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	item, err := sched.GetSchedule(context.Background(), id)
	if err != nil {
		t.Fatalf("GetSchedule failed: %v", err)
	}
	if item.CronExpr != "0 * * * *" {
		t.Errorf("expected cron_expr '0 * * * *', got %q", item.CronExpr)
	}
}

func TestServiceGetScheduleNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	sched := NewService(db, nil)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	_, err := sched.GetSchedule(context.Background(), id)
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestServiceCreateSchedule(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	sched := NewService(db, nil)
	input := &CreateScheduleInput{
		SourceID:  "01800000-0000-0000-0000-000000000001",
		CronExpr:  "0 * * * *",
		IsEnabled: true,
	}

	item, warnings, err := sched.CreateSchedule(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}
	if item.CronExpr != "0 * * * *" {
		t.Errorf("expected cron_expr '0 * * * *', got %q", item.CronExpr)
	}
	if bool(item.IsEnabled) != true {
		t.Errorf("expected is_enabled true, got %v", bool(item.IsEnabled))
	}
	if warnings == nil {
		t.Log("warnings may be nil if no overlaps, which is OK")
	}
}

func TestServiceCreateScheduleInvalidCron(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	sched := NewService(db, nil)
	input := &CreateScheduleInput{
		SourceID:  "01800000-0000-0000-0000-000000000001",
		CronExpr:  "invalid",
		IsEnabled: true,
	}

	_, _, err := sched.CreateSchedule(context.Background(), input)
	if !errors.Is(err, ErrInvalidCronExpr) {
		t.Errorf("expected ErrInvalidCronExpr, got %v", err)
	}
}

func TestServiceCreateScheduleSourceNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	sched := NewService(db, nil)
	input := &CreateScheduleInput{
		SourceID:  "01800000-0000-0000-0000-000000000001",
		CronExpr:  "0 * * * *",
		IsEnabled: true,
	}

	_, _, err := sched.CreateSchedule(context.Background(), input)
	if !errors.Is(err, ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestServiceCreateScheduleCallsSchedulerReload(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	mock := &mockScheduler{}
	sched := NewService(db, mock)
	input := &CreateScheduleInput{
		SourceID:  "01800000-0000-0000-0000-000000000001",
		CronExpr:  "0 * * * *",
		IsEnabled: true,
	}

	_, _, err := sched.CreateSchedule(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateSchedule failed: %v", err)
	}
	if mock.reloadCalled != 1 {
		t.Errorf("expected Reload called once, got %d", mock.reloadCalled)
	}
}

func TestServiceUpdateSchedule(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	_, err := db.Exec(`
		INSERT INTO source_schedules (id, source_id, cron_expr, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001", "01800000-0000-0000-0000-000000000001",
		"0 * * * *", 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test schedule: %v", err)
	}

	sched := NewService(db, nil)
	input := &UpdateScheduleInput{
		ID:        "02800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000001",
		CronExpr:  "*/5 * * * *",
		IsEnabled: false,
	}

	item, _, err := sched.UpdateSchedule(context.Background(), input)
	if err != nil {
		t.Fatalf("UpdateSchedule failed: %v", err)
	}
	if item.CronExpr != "*/5 * * * *" {
		t.Errorf("expected cron_expr '*/5 * * * *', got %q", item.CronExpr)
	}
	if bool(item.IsEnabled) != false {
		t.Errorf("expected is_enabled false, got %v", bool(item.IsEnabled))
	}
}

func TestServiceUpdateScheduleNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	sched := NewService(db, nil)
	input := &UpdateScheduleInput{
		ID:        "02800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000001",
		CronExpr:  "0 * * * *",
		IsEnabled: true,
	}

	_, _, err := sched.UpdateSchedule(context.Background(), input)
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestServiceUpdateScheduleSourceNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	_, err := db.Exec(`
		INSERT INTO source_schedules (id, source_id, cron_expr, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001", "01800000-0000-0000-0000-000000000001",
		"0 * * * *", 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test schedule: %v", err)
	}

	sched := NewService(db, nil)
	input := &UpdateScheduleInput{
		ID:        "02800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000002", // different source
		CronExpr:  "0 * * * *",
		IsEnabled: true,
	}

	_, _, err = sched.UpdateSchedule(context.Background(), input)
	if !errors.Is(err, ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestServiceUpdateScheduleCallsSchedulerReload(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	_, err := db.Exec(`
		INSERT INTO source_schedules (id, source_id, cron_expr, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001", "01800000-0000-0000-0000-000000000001",
		"0 * * * *", 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test schedule: %v", err)
	}

	mock := &mockScheduler{}
	sched := NewService(db, mock)
	input := &UpdateScheduleInput{
		ID:        "02800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000001",
		CronExpr:  "*/5 * * * *",
		IsEnabled: false,
	}

	_, _, err = sched.UpdateSchedule(context.Background(), input)
	if err != nil {
		t.Fatalf("UpdateSchedule failed: %v", err)
	}
	if mock.reloadCalled != 1 {
		t.Errorf("expected Reload called once, got %d", mock.reloadCalled)
	}
}

func TestServiceDeleteSchedule(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	_, err := db.Exec(`
		INSERT INTO source_schedules (id, source_id, cron_expr, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001", "01800000-0000-0000-0000-000000000001",
		"0 * * * *", 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test schedule: %v", err)
	}

	sched := NewService(db, nil)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	err = sched.DeleteSchedule(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteSchedule failed: %v", err)
	}

	// Verify deleted
	_, err = sched.GetSchedule(context.Background(), id)
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound after delete, got %v", err)
	}
}

func TestServiceDeleteScheduleNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	sched := NewService(db, nil)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	err := sched.DeleteSchedule(context.Background(), id)
	if !errors.Is(err, ErrScheduleNotFound) {
		t.Errorf("expected ErrScheduleNotFound, got %v", err)
	}
}

func TestServiceDeleteScheduleCallsSchedulerReload(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000001", "test-source")

	_, err := db.Exec(`
		INSERT INTO source_schedules (id, source_id, cron_expr, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001", "01800000-0000-0000-0000-000000000001",
		"0 * * * *", 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test schedule: %v", err)
	}

	mock := &mockScheduler{}
	sched := NewService(db, mock)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	err = sched.DeleteSchedule(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteSchedule failed: %v", err)
	}
	if mock.reloadCalled != 1 {
		t.Errorf("expected Reload called once, got %d", mock.reloadCalled)
	}
}

func TestServiceErrDBUnavailableWhenDBIsNil(t *testing.T) {
	sched := NewService(nil, nil)

	_, err := sched.ListSchedules(context.Background())
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}

	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	_, err = sched.GetSchedule(context.Background(), id)
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}

	_, _, err = sched.CreateSchedule(context.Background(), &CreateScheduleInput{})
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}

	_, _, err = sched.UpdateSchedule(context.Background(), &UpdateScheduleInput{})
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}

	err = sched.DeleteSchedule(context.Background(), id)
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}
}

// --- Overlap Warning Tests ---

func TestCheckOverlapWarningsNoSchedules(t *testing.T) {
	warnings, err := CheckOverlapWarnings(nil, 14)
	if err != nil {
		t.Fatalf("CheckOverlapWarnings failed: %v", err)
	}
	if warnings != nil {
		t.Errorf("expected nil warnings for nil input, got %v", warnings)
	}
}

func TestCheckOverlapWarningsSingleSchedule(t *testing.T) {
	schedules := []ScheduleWithSource{
		{
			ScheduleRow: ScheduleRow{
				ID:       uuidPtr(mustNewUUID("02800000-0000-0000-0000-000000000001")),
				CronExpr: "0 * * * *",
			},
			SourceName: "source-a",
		},
	}
	warnings, err := CheckOverlapWarnings(schedules, 14)
	if err != nil {
		t.Fatalf("CheckOverlapWarnings failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for single schedule, got %d", len(warnings))
	}
}

func TestCheckOverlapWarningsNoOverlap(t *testing.T) {
	schedules := []ScheduleWithSource{
		{
			ScheduleRow: ScheduleRow{
				ID:       uuidPtr(mustNewUUID("02800000-0000-0000-0000-000000000001")),
				CronExpr: "0 * * * *", // Every hour
			},
			SourceName: "source-a",
		},
		{
			ScheduleRow: ScheduleRow{
				ID:       uuidPtr(mustNewUUID("02800000-0000-0000-0000-000000000002")),
				CronExpr: "30 * * * *", // Every hour at minute 30
			},
			SourceName: "source-a",
		},
	}

	warnings, err := CheckOverlapWarnings(schedules, 14)
	if err != nil {
		t.Fatalf("CheckOverlapWarnings failed: %v", err)
	}
	// 30 minutes apart is > 5 minutes, so no warning
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for schedules 30 mins apart, got %d", len(warnings))
	}
}

func TestCheckOverlapWarningsWithOverlap(t *testing.T) {
	schedules := []ScheduleWithSource{
		{
			ScheduleRow: ScheduleRow{
				ID:       uuidPtr(mustNewUUID("02800000-0000-0000-0000-000000000001")),
				CronExpr: "0 * * * *", // Every hour at minute 0
			},
			SourceName: "source-a",
		},
		{
			ScheduleRow: ScheduleRow{
				ID:       uuidPtr(mustNewUUID("02800000-0000-0000-0000-000000000002")),
				CronExpr: "2 * * * *", // Every hour at minute 2 (2 mins after, < 5 min warning threshold)
			},
			SourceName: "source-a",
		},
	}

	warnings, err := CheckOverlapWarnings(schedules, 14)
	if err != nil {
		t.Fatalf("CheckOverlapWarnings failed: %v", err)
	}
	if len(warnings) == 0 {
		t.Error("expected at least 1 warning for schedules 2 mins apart")
	}
	// Check the warning is about the correct source
	if warnings[0].SourceName != "source-a" {
		t.Errorf("expected source_name 'source-a', got %q", warnings[0].SourceName)
	}
	if warnings[0].DistanceMins >= 5 {
		t.Errorf("expected distance < 5 mins, got %f", warnings[0].DistanceMins)
	}
}

func TestCheckOverlapWarningsDifferentSources(t *testing.T) {
	// Two different sources - should not warn even if times are close
	schedules := []ScheduleWithSource{
		{
			ScheduleRow: ScheduleRow{
				ID:       uuidPtr(mustNewUUID("02800000-0000-0000-0000-000000000001")),
				CronExpr: "0 * * * *",
			},
			SourceName: "source-a",
		},
		{
			ScheduleRow: ScheduleRow{
				ID:       uuidPtr(mustNewUUID("02800000-0000-0000-0000-000000000002")),
				CronExpr: "1 * * * *", // 1 minute after, but different source
			},
			SourceName: "source-b", // Different source
		},
	}

	warnings, err := CheckOverlapWarnings(schedules, 14)
	if err != nil {
		t.Fatalf("CheckOverlapWarnings failed: %v", err)
	}
	// Overlaps are only checked within the same source
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings for different sources, got %d", len(warnings))
	}
}

func mustNewUUID(s string) dbtypes.UUID {
	id, err := dbtypes.NewUUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return id
}
