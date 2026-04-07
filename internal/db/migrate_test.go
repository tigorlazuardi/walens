package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", sqliteDSN(dsn))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func TestRunMigrationsEmptyDB(t *testing.T) {
	db := openTestDB(t, ":memory:")

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	var configsTableExists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='configs'`).Scan(&configsTableExists); err != nil {
		t.Fatalf("check configs table: %v", err)
	}
	if configsTableExists != 1 {
		t.Fatalf("expected configs table to exist")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM configs`).Scan(&count); err != nil {
		t.Fatalf("count configs rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one default configs row, got %d", count)
	}

	var maxVersion int
	if err := db.QueryRow(`SELECT MAX(version_id) FROM goose_db_version`).Scan(&maxVersion); err != nil {
		t.Fatalf("read goose max version: %v", err)
	}
	if maxVersion != 3 {
		t.Fatalf("expected goose max version 3, got %d", maxVersion)
	}
}

func TestRunMigrationsRepeatedRunIsIdempotent(t *testing.T) {
	db := openTestDB(t, ":memory:")

	if err := RunMigrations(db); err != nil {
		t.Fatalf("first RunMigrations failed: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM configs`).Scan(&count); err != nil {
		t.Fatalf("count configs rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one configs row after repeated migrations, got %d", count)
	}

	var maxVersion int
	if err := db.QueryRow(`SELECT MAX(version_id) FROM goose_db_version`).Scan(&maxVersion); err != nil {
		t.Fatalf("read goose max version: %v", err)
	}
	if maxVersion != 3 {
		t.Fatalf("expected goose max version 3 after repeated migrations, got %d", maxVersion)
	}
}

func TestRunMigrationsFileBasedDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "walens.db")

	db := openTestDB(t, dbPath)
	if err := RunMigrations(db); err != nil {
		t.Fatalf("initial RunMigrations failed: %v", err)
	}
	if _, err := db.Exec(`UPDATE configs SET value = ?, updated_at = ? WHERE id = 1`, `{"hello":"world"}`, 123); err != nil {
		t.Fatalf("update configs row: %v", err)
	}
	_ = db.Close()

	db = openTestDB(t, dbPath)
	if err := RunMigrations(db); err != nil {
		t.Fatalf("reopened RunMigrations failed: %v", err)
	}

	var value string
	var updatedAt int
	if err := db.QueryRow(`SELECT value, updated_at FROM configs WHERE id = 1`).Scan(&value, &updatedAt); err != nil {
		t.Fatalf("query configs row: %v", err)
	}
	if value != `{"hello":"world"}` || updatedAt != 123 {
		t.Fatalf("expected config row preserved, got value=%s updated_at=%d", value, updatedAt)
	}
}

func TestBusinessSchemaTablesExist(t *testing.T) {
	db := openTestDB(t, ":memory:")

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	expectedTables := []string{
		"devices",
		"sources",
		"source_schedules",
		"device_source_subscriptions",
		"images",
		"tags",
		"image_tags",
		"image_assignments",
		"image_locations",
		"image_thumbnails",
		"image_blacklists",
		"jobs",
	}

	for _, table := range expectedTables {
		var exists int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if exists != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestBusinessSchemaIndexesExist(t *testing.T) {
	db := openTestDB(t, ":memory:")

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	expectedIndexes := []struct {
		name  string
		table string
	}{
		{"idx_devices_slug", "devices"},
		{"idx_sources_name", "sources"},
		{"idx_source_schedules_source_id", "source_schedules"},
		{"idx_device_source_subscriptions_device_source", "device_source_subscriptions"},
		{"idx_images_source_id", "images"},
		{"idx_images_unique_identifier", "images"},
		{"idx_tags_normalized_name", "tags"},
		{"idx_image_tags_image_tag", "image_tags"},
		{"idx_image_assignments_image_device", "image_assignments"},
		{"idx_image_locations_path", "image_locations"},
		{"idx_image_locations_image_id", "image_locations"},
		{"idx_image_locations_device_id", "image_locations"},
		{"idx_image_thumbnails_image_id", "image_thumbnails"},
		{"idx_image_thumbnails_path", "image_thumbnails"},
		{"idx_image_blacklists_source_identifier", "image_blacklists"},
		{"idx_jobs_status", "jobs"},
		{"idx_jobs_source_id", "jobs"},
		{"idx_jobs_run_after", "jobs"},
	}

	for _, idx := range expectedIndexes {
		var exists int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, idx.name).Scan(&exists); err != nil {
			t.Fatalf("check index %s: %v", idx.name, err)
		}
		if exists != 1 {
			t.Errorf("expected index %s on table %s to exist", idx.name, idx.table)
		}
	}
}

func TestBusinessSchemaColumnPresence(t *testing.T) {
	db := openTestDB(t, ":memory:")

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Spot-check key columns across tables to catch column name typos.
	checks := []struct {
		table    string
		column   string
		expected string
	}{
		// devices
		{"devices", "id", "TEXT"},
		{"devices", "slug", "TEXT"},
		{"devices", "screen_width", "INTEGER"},
		{"devices", "aspect_ratio_tolerance", "REAL"},
		{"devices", "is_adult_allowed", "INTEGER"},
		{"devices", "is_enabled", "INTEGER"},
		{"devices", "created_at", "INTEGER"},
		// sources
		{"sources", "id", "TEXT"},
		{"sources", "name", "TEXT"},
		{"sources", "source_type", "TEXT"},
		{"sources", "params", "TEXT"},
		{"sources", "lookup_count", "INTEGER"},
		{"sources", "is_enabled", "INTEGER"},
		// images
		{"images", "id", "TEXT"},
		{"images", "unique_identifier", "TEXT"},
		{"images", "source_item_identifier", "TEXT"},
		{"images", "original_identifier", "TEXT"},
		{"images", "is_adult", "INTEGER"},
		{"images", "is_favorite", "INTEGER"},
		{"images", "json_meta", "TEXT"},
		// jobs
		{"jobs", "id", "TEXT"},
		{"jobs", "job_type", "TEXT"},
		{"jobs", "status", "TEXT"},
		{"jobs", "trigger_kind", "TEXT"},
		{"jobs", "run_after", "INTEGER"},
		{"jobs", "duration_ms", "INTEGER"},
		{"jobs", "json_input", "TEXT"},
		{"jobs", "json_result", "TEXT"},
		// image_locations
		{"image_locations", "storage_kind", "TEXT"},
		{"image_locations", "is_primary", "INTEGER"},
		{"image_locations", "is_active", "INTEGER"},
	}

	for _, c := range checks {
		var colType string
		if err := db.QueryRow(`SELECT type FROM pragma_table_info(?) WHERE name=?`, c.table, c.column).Scan(&colType); err != nil {
			t.Errorf("column %s.%s: %v", c.table, c.column, err)
			continue
		}
		if colType != c.expected {
			t.Errorf("column %s.%s: expected type %q, got %q", c.table, c.column, c.expected, colType)
		}
	}
}

func TestBusinessSchemaForeignKeys(t *testing.T) {
	db := openTestDB(t, ":memory:")

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify foreign keys are enforced by testing cascade delete.
	// Insert a device, then a subscription, delete device, subscription should be gone.
	deviceInsert := "INSERT INTO devices (id, name, slug, screen_width, screen_height, is_enabled, created_at, updated_at) VALUES ('dev1', 'Test Device', 'test-device', 1920, 1080, 1, 0, 0)"
	if _, err := db.Exec(deviceInsert); err != nil {
		t.Fatalf("insert test device: %v", err)
	}
	sourceInsert := "INSERT INTO sources (id, name, source_type, created_at, updated_at) VALUES ('src1', 'Test Source', 'test', 0, 0)"
	if _, err := db.Exec(sourceInsert); err != nil {
		t.Fatalf("insert test source: %v", err)
	}
	subscriptionInsert := "INSERT INTO device_source_subscriptions (id, device_id, source_id, created_at, updated_at) VALUES ('sub1', 'dev1', 'src1', 0, 0)"
	if _, err := db.Exec(subscriptionInsert); err != nil {
		t.Fatalf("insert test subscription: %v", err)
	}

	// Delete the device, subscription should be cascade-deleted.
	deleteDevice := "DELETE FROM devices WHERE id = 'dev1'"
	if _, err := db.Exec(deleteDevice); err != nil {
		t.Fatalf("delete device: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM device_source_subscriptions WHERE id = 'sub1'").Scan(&count); err != nil {
		t.Fatalf("check subscription deleted: %v", err)
	}
	if count != 0 {
		t.Error("expected subscription to be cascade-deleted when device is deleted")
	}
}

func TestMigrationVersionIsThree(t *testing.T) {
	db := openTestDB(t, ":memory:")

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	var maxVersion int
	if err := db.QueryRow(`SELECT MAX(version_id) FROM goose_db_version`).Scan(&maxVersion); err != nil {
		t.Fatalf("read goose max version: %v", err)
	}
	if maxVersion != 3 {
		t.Errorf("expected goose version 3, got %d", maxVersion)
	}
}
