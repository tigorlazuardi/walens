package configs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func TestGetConfigReturnsExisting(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Create the configs table and insert a config
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS configs (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			value TEXT NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create configs table: %v", err)
	}

	persistedCfg := &PersistedConfig{
		DataDir:  "/existing/data",
		LogLevel: "warn",
	}
	valueBytes, _ := json.Marshal(persistedCfg)
	_, err = db.Exec(`INSERT INTO configs (id, value, updated_at) VALUES (1, ?, 1000)`, string(valueBytes))
	if err != nil {
		t.Fatalf("insert config: %v", err)
	}

	svc := NewService(db)
	loaded, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if loaded.DataDir != "/existing/data" {
		t.Errorf("expected DataDir '/existing/data', got: %q", loaded.DataDir)
	}
	if loaded.LogLevel != "warn" {
		t.Errorf("expected LogLevel 'warn', got: %q", loaded.LogLevel)
	}
}

func TestGetConfigInitializesDefaultsWhenAbsent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Create empty configs table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS configs (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			value TEXT NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create configs table: %v", err)
	}

	svc := NewService(db)
	loaded, err := svc.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	// Should return defaults
	if loaded.DataDir != "./data" {
		t.Errorf("expected default DataDir './data', got: %q", loaded.DataDir)
	}
	if loaded.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got: %q", loaded.LogLevel)
	}
}

func TestGetConfigReturnsErrDBUnavailableWhenDBIsNil(t *testing.T) {
	svc := NewService(nil)
	_, err := svc.GetConfig(context.Background())
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got: %v", err)
	}
}

func TestUpdateConfigStoresConfig(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Create configs table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS configs (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			value TEXT NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create configs table: %v", err)
	}

	svc := NewService(db)
	newCfg := &PersistedConfig{
		DataDir:  "/new/data",
		LogLevel: "debug",
	}

	storedCfg, err := svc.UpdateConfig(context.Background(), newCfg)
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	if storedCfg.DataDir != "/new/data" {
		t.Errorf("expected DataDir '/new/data', got: %q", storedCfg.DataDir)
	}
	if storedCfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got: %q", storedCfg.LogLevel)
	}

	// Verify it was actually stored
	loaded, err := svc.Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.DataDir != "/new/data" {
		t.Errorf("expected stored DataDir '/new/data', got: %q", loaded.DataDir)
	}
}

func TestUpdateConfigReturnsErrDBUnavailableWhenDBIsNil(t *testing.T) {
	svc := NewService(nil)
	cfg := &PersistedConfig{DataDir: "/data", LogLevel: "info"}
	_, err := svc.UpdateConfig(context.Background(), cfg)
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got: %v", err)
	}
}
