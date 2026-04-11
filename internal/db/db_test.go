package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteDSN_Memory(t *testing.T) {
	dsn := sqliteDSN(":memory:")
	if !strings.HasPrefix(dsn, "file::memory:?") {
		t.Errorf("expected memory DSN to start with 'file::memory:?', got %q", dsn)
	}
	if !strings.Contains(dsn, "mode=memory") {
		t.Errorf("expected memory DSN to contain 'mode=memory', got %q", dsn)
	}
}

func TestSQLiteDSN_RelativePath(t *testing.T) {
	dsn := sqliteDSN("artifacts/sqlite.db")
	// DSN should be an absolute path, not file://artifacts/...
	if !strings.HasPrefix(dsn, "file:///") {
		t.Errorf("expected relative path DSN to start with 'file:///', got %q", dsn)
	}
	// Should not be file://artifacts/... (with artifacts as host)
	if strings.HasPrefix(dsn, "file://artifacts/") {
		t.Errorf("expected relative path DSN to NOT have artifacts as host, got %q", dsn)
	}
	// Should still have pragmas
	if !strings.Contains(dsn, "foreign_keys") {
		t.Errorf("expected DSN to contain foreign_keys pragma, got %q", dsn)
	}
}

func TestSQLiteDSN_AbsolutePath(t *testing.T) {
	absPath := "/tmp/walens/test.db"
	dsn := sqliteDSN(absPath)
	if !strings.HasPrefix(dsn, "file:///tmp/walens/test.db") {
		t.Errorf("expected absolute path DSN to start with 'file:///tmp/walens/test.db', got %q", dsn)
	}
	if !strings.Contains(dsn, "foreign_keys") {
		t.Errorf("expected DSN to contain foreign_keys pragma, got %q", dsn)
	}
}

func TestOpen_RelativePath(t *testing.T) {
	// Change to a temp directory to ensure path is relative
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get current dir: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to temp dir: %v", err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	db, err := Open("data/test.db")
	if err != nil {
		t.Fatalf("Open with relative path failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify we can execute a simple query
	var count int
	if err := db.QueryRow("SELECT 1").Scan(&count); err != nil {
		t.Fatalf("query after open: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count=1, got %d", count)
	}
}

func TestOpen_AbsolutePath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "walens_absolute.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open with absolute path failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	var count int
	if err := db.QueryRow("SELECT 1").Scan(&count); err != nil {
		t.Fatalf("query after open: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count=1, got %d", count)
	}
}

func TestOpen_Memory(t *testing.T) {
	// Note: Open doesn't special-case :memory:, but sqliteDSN does
	// For true in-memory, we use sql.Open directly with sqliteDSN
	// The dir creation will succeed but :memory: ignores it
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open with :memory: failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	var count int
	if err := db.QueryRow("SELECT 1").Scan(&count); err != nil {
		t.Fatalf("query after open: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count=1, got %d", count)
	}
}
