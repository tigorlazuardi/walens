package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func Open(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to make directory at %s: %w", dbPath, err)
	}

	dsn := sqliteDSN(dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite at %s: %w", dsn, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to execute ping to %s: %w", dbPath, err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	// Apply SQLite pragmas that should persist at the database level.
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to execute pragma %s: %w", p, err)
		}
	}

	return db, nil
}

func sqliteDSN(dbPath string) string {
	values := url.Values{}
	values.Add("_pragma", "foreign_keys(1)")
	values.Add("_pragma", "busy_timeout(5000)")
	values.Add("_pragma", "temp_store(MEMORY)")
	values.Add("_pragma", "cache_size(-2000)")
	if dbPath == ":memory:" {
		values.Add("mode", "memory")
		return "file::memory:?" + values.Encode()
	}

	// Convert relative paths to absolute to ensure valid file URI.
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		// Fallback to original path if Abs fails (should not happen in practice).
		absPath = dbPath
	}

	return (&url.URL{
		Scheme:   "file",
		Path:     filepath.ToSlash(absPath),
		RawQuery: values.Encode(),
	}).String()
}

// Ping checks database connectivity.
func Ping(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
