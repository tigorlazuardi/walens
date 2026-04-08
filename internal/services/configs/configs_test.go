package configs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/walens/walens/internal/config"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", sqliteDSN(":memory:"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return db
}

func sqliteDSN(dbPath string) string {
	if dbPath == ":memory:" {
		return "file::memory:?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=temp_store(MEMORY)"
	}
	return "file:" + dbPath + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
}

func TestLoadConfigNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Insert the configs table structure without a row
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
	_, err = svc.Load(context.Background())

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound, got: %v", err)
	}
}

func TestLoadConfigEmptyValue(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	// Insert the configs table with empty value
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

	_, err = db.Exec(`INSERT INTO configs (id, value, updated_at) VALUES (1, '{}', 0)`)
	if err != nil {
		t.Fatalf("insert empty config: %v", err)
	}

	svc := NewService(db)
	_, err = svc.Load(context.Background())

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound for empty JSON, got: %v", err)
	}
}

func TestLoadConfigWhitespaceValue(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

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

	_, err = db.Exec(`INSERT INTO configs (id, value, updated_at) VALUES (1, '   ', 0)`)
	if err != nil {
		t.Fatalf("insert whitespace config: %v", err)
	}

	svc := NewService(db)
	_, err = svc.Load(context.Background())

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound for whitespace-only JSON, got: %v", err)
	}
}

func TestLoadConfigSuccess(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

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
		Server: PersistedServerConfig{
			BasePath: "/walens",
		},
		DataDir:  "/data",
		LogLevel: "debug",
	}
	valueBytes, _ := json.Marshal(persistedCfg)
	_, err = db.Exec(`INSERT INTO configs (id, value, updated_at) VALUES (1, ?, 1000)`, string(valueBytes))
	if err != nil {
		t.Fatalf("insert config: %v", err)
	}

	svc := NewService(db)
	loaded, err := svc.Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Server.BasePath != "/walens" {
		t.Errorf("expected BasePath '/walens', got: %q", loaded.Server.BasePath)
	}
	if loaded.DataDir != "/data" {
		t.Errorf("expected DataDir '/data', got: %q", loaded.DataDir)
	}
	if loaded.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got: %q", loaded.LogLevel)
	}
}

func TestStoreConfigAtomicReplace(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

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

	// Insert initial config
	initialCfg := &PersistedConfig{
		Server: PersistedServerConfig{
			BasePath: "/",
		},
		DataDir:  "/initial",
		LogLevel: "info",
	}
	initialBytes, _ := json.Marshal(initialCfg)
	_, err = db.Exec(`INSERT INTO configs (id, value, updated_at) VALUES (1, ?, 0)`, string(initialBytes))
	if err != nil {
		t.Fatalf("insert initial config: %v", err)
	}

	// Store new config
	svc := NewService(db)
	newCfg := &PersistedConfig{
		Server: PersistedServerConfig{
			BasePath: "/walens",
		},
		DataDir:  "/newdata",
		LogLevel: "debug",
	}
	if err := svc.Store(context.Background(), newCfg); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify atomic replacement
	var value string
	var updatedAt int64
	err = db.QueryRow(`SELECT value, updated_at FROM configs WHERE id = 1`).Scan(&value, &updatedAt)
	if err != nil {
		t.Fatalf("query config: %v", err)
	}

	if updatedAt == 0 {
		t.Error("expected updated_at to be set to a non-zero timestamp")
	}

	var loaded PersistedConfig
	if err := json.Unmarshal([]byte(value), &loaded); err != nil {
		t.Fatalf("unmarshal stored config: %v", err)
	}

	if loaded.Server.BasePath != "/walens" {
		t.Errorf("expected BasePath '/walens', got: %q", loaded.Server.BasePath)
	}
	if loaded.DataDir != "/newdata" {
		t.Errorf("expected DataDir '/newdata', got: %q", loaded.DataDir)
	}
	if loaded.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got: %q", loaded.LogLevel)
	}
}

func TestBootstrapDefaultInsertsWhenAbsent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

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
	defaultCfg := &PersistedConfig{
		Server: PersistedServerConfig{
			BasePath: "/",
		},
		DataDir:  "/default/data",
		LogLevel: "info",
	}

	bootstrapCfg, err := svc.BootstrapDefault(context.Background(), defaultCfg)
	if err != nil {
		t.Fatalf("BootstrapDefault failed: %v", err)
	}

	if bootstrapCfg.DataDir != "/default/data" {
		t.Errorf("expected DataDir '/default/data', got: %q", bootstrapCfg.DataDir)
	}

	// Verify row was actually inserted
	var value string
	err = db.QueryRow(`SELECT value FROM configs WHERE id = 1`).Scan(&value)
	if err != nil {
		t.Fatalf("query config after bootstrap: %v", err)
	}

	var stored PersistedConfig
	if err := json.Unmarshal([]byte(value), &stored); err != nil {
		t.Fatalf("unmarshal stored config: %v", err)
	}
	if stored.DataDir != "/default/data" {
		t.Errorf("expected stored DataDir '/default/data', got: %q", stored.DataDir)
	}
}

func TestBootstrapDefaultReturnsExisting(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

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

	// Pre-populate with existing config
	existingCfg := &PersistedConfig{
		Server: PersistedServerConfig{
			BasePath: "/existing",
		},
		DataDir:  "/existing/data",
		LogLevel: "warn",
	}
	existingBytes, _ := json.Marshal(existingCfg)
	_, err = db.Exec(`INSERT INTO configs (id, value, updated_at) VALUES (1, ?, 500)`, string(existingBytes))
	if err != nil {
		t.Fatalf("insert existing config: %v", err)
	}

	svc := NewService(db)
	defaultCfg := &PersistedConfig{
		Server: PersistedServerConfig{
			BasePath: "/",
		},
		DataDir:  "/default/data",
		LogLevel: "info",
	}

	bootstrapCfg, err := svc.BootstrapDefault(context.Background(), defaultCfg)
	if err != nil {
		t.Fatalf("BootstrapDefault failed: %v", err)
	}

	// Should return existing, not default
	if bootstrapCfg.DataDir != "/existing/data" {
		t.Errorf("expected DataDir '/existing/data', got: %q", bootstrapCfg.DataDir)
	}
	if bootstrapCfg.LogLevel != "warn" {
		t.Errorf("expected LogLevel 'warn', got: %q", bootstrapCfg.LogLevel)
	}
}

func TestBootstrapDefaultWithEmptyRow(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

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

	// Insert empty config row
	_, err = db.Exec(`INSERT INTO configs (id, value, updated_at) VALUES (1, '{}', 0)`)
	if err != nil {
		t.Fatalf("insert empty config: %v", err)
	}

	svc := NewService(db)
	defaultCfg := &PersistedConfig{
		Server: PersistedServerConfig{
			BasePath: "/",
		},
		DataDir:  "/default/data",
		LogLevel: "info",
	}

	bootstrapCfg, err := svc.BootstrapDefault(context.Background(), defaultCfg)
	if err != nil {
		t.Fatalf("BootstrapDefault failed: %v", err)
	}

	// Should inject defaults for empty row
	if bootstrapCfg.DataDir != "/default/data" {
		t.Errorf("expected DataDir '/default/data', got: %q", bootstrapCfg.DataDir)
	}
}

func TestDefaultPersistedConfig(t *testing.T) {
	persistedCfg := DefaultPersistedConfig()

	if persistedCfg.Server.BasePath != "/" {
		t.Errorf("expected default BasePath '/', got: %q", persistedCfg.Server.BasePath)
	}
	if persistedCfg.DataDir != "./data" {
		t.Errorf("expected default DataDir './data', got: %q", persistedCfg.DataDir)
	}
	if persistedCfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got: %q", persistedCfg.LogLevel)
	}
}

func TestPersistedConfigApplyBootstrapConfig(t *testing.T) {
	bootstrapCfg := &config.Config{
		Server: config.ServerConfig{
			Host:     "0.0.0.0",
			Port:     8080,
			BasePath: "/walens",
		},
		Database: config.DatabaseConfig{
			Path: "./data/walens.db",
		},
		DataDir:  "/opt/walens/data",
		LogLevel: "debug",
	}

	persistedCfg := DefaultPersistedConfig()
	persistedCfg.ApplyBootstrapConfig(bootstrapCfg)

	if persistedCfg.Server.BasePath != "/walens" {
		t.Errorf("expected BasePath '/walens', got: %q", persistedCfg.Server.BasePath)
	}
	if persistedCfg.DataDir != "/opt/walens/data" {
		t.Errorf("expected DataDir '/opt/walens/data', got: %q", persistedCfg.DataDir)
	}
	if persistedCfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got: %q", persistedCfg.LogLevel)
	}
}

func TestPersistedConfigJSONSerialization(t *testing.T) {
	cfg := &PersistedConfig{
		Server: PersistedServerConfig{
			BasePath: "/custom",
		},
		DataDir:  "/custom/data",
		LogLevel: "trace",
	}

	bytes, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var loaded PersistedConfig
	if err := json.Unmarshal(bytes, &loaded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if loaded.Server.BasePath != "/custom" {
		t.Errorf("expected BasePath '/custom', got: %q", loaded.Server.BasePath)
	}
	if loaded.DataDir != "/custom/data" {
		t.Errorf("expected DataDir '/custom/data', got: %q", loaded.DataDir)
	}
	if loaded.LogLevel != "trace" {
		t.Errorf("expected LogLevel 'trace', got: %q", loaded.LogLevel)
	}
}

func TestStoreConfigUpdatesTimestamp(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

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

	// Insert initial config with old timestamp
	initialCfg := &PersistedConfig{Server: PersistedServerConfig{BasePath: "/"}}
	initialBytes, _ := json.Marshal(initialCfg)
	_, err = db.Exec(`INSERT INTO configs (id, value, updated_at) VALUES (1, ?, 1000)`, string(initialBytes))
	if err != nil {
		t.Fatalf("insert initial config: %v", err)
	}

	svc := NewService(db)
	newCfg := &PersistedConfig{Server: PersistedServerConfig{BasePath: "/new"}}
	if err := svc.Store(context.Background(), newCfg); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify timestamp was updated
	var updatedAt int64
	err = db.QueryRow(`SELECT updated_at FROM configs WHERE id = 1`).Scan(&updatedAt)
	if err != nil {
		t.Fatalf("query updated_at: %v", err)
	}

	if updatedAt <= 1000 {
		t.Errorf("expected updated_at > 1000 after store, got: %d", updatedAt)
	}
}

func TestConfigRepoWithFileDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "walens.db")

	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS configs (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			value TEXT NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create configs table: %v", err)
	}

	// Simulate first boot: bootstrap defaults
	svc := NewService(db)
	defaultCfg := &PersistedConfig{
		Server:   PersistedServerConfig{BasePath: "/"},
		DataDir:  "./data",
		LogLevel: "info",
	}

	persistedCfg, err := svc.BootstrapDefault(context.Background(), defaultCfg)
	if err != nil {
		t.Fatalf("BootstrapDefault failed: %v", err)
	}

	// Simulate second boot: load existing config
	persistedCfg2, err := svc.BootstrapDefault(context.Background(), defaultCfg)
	if err != nil {
		t.Fatalf("BootstrapDefault on second boot failed: %v", err)
	}

	if persistedCfg2.DataDir != persistedCfg.DataDir {
		t.Errorf("expected DataDir to persist: got %q, want %q", persistedCfg2.DataDir, persistedCfg.DataDir)
	}

	// Simulate config update
	newCfg := &PersistedConfig{
		Server:   PersistedServerConfig{BasePath: "/walens"},
		DataDir:  "/new/data",
		LogLevel: "debug",
	}
	if err := svc.Store(context.Background(), newCfg); err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// Verify update persisted
	loaded, err := svc.Load(context.Background())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.DataDir != "/new/data" {
		t.Errorf("expected DataDir '/new/data', got: %q", loaded.DataDir)
	}
}
