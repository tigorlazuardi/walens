package db

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -2000",
		"PRAGMA temp_store = MEMORY",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			t.Fatalf("apply pragma %q: %v", pragma, err)
		}
	}

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
	if maxVersion != 2 {
		t.Fatalf("expected goose max version 2, got %d", maxVersion)
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
	if maxVersion != 2 {
		t.Fatalf("expected goose max version 2 after repeated migrations, got %d", maxVersion)
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
