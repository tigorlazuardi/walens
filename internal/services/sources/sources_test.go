package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/sources"
	"github.com/walens/walens/internal/sources/booru"
	_ "modernc.org/sqlite"
)

func testRegistry() *sources.Registry {
	registry := sources.NewRegistry()
	registry.Register(booru.New())
	return registry
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=temp_store(MEMORY)")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return db
}

func createSourcesTable(t *testing.T, db *sql.DB) {
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
}

func TestServiceListSourcesEmpty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, testRegistry())
	items, err := svc.ListSources(context.Background())
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 sources, got %d", len(items))
	}
}

func TestServiceListSourcesWithData(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"01800000-0000-0000-0000-000000000001", "test-source", "booru",
		`{"tags":["landscape"]}`, 100, 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}

	svc := NewService(db, testRegistry())
	items, err := svc.ListSources(context.Background())
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 source, got %d", len(items))
	}
	if items[0].Name != "test-source" {
		t.Errorf("expected name 'test-source', got %q", items[0].Name)
	}
	if items[0].SourceType != "booru" {
		t.Errorf("expected source_type 'booru', got %q", items[0].SourceType)
	}
}

func TestServiceGetSource(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"01800000-0000-0000-0000-000000000001", "test-source", "booru",
		`{"tags":["landscape"]}`, 100, 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}

	svc := NewService(db, testRegistry())
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	src, err := svc.GetSource(context.Background(), id)
	if err != nil {
		t.Fatalf("GetSource failed: %v", err)
	}
	if src.Name != "test-source" {
		t.Errorf("expected name 'test-source', got %q", src.Name)
	}
}

func TestServiceGetSourceNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, testRegistry())
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	_, err := svc.GetSource(context.Background(), id)
	if !errors.Is(err, ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestServiceCreateSource(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, testRegistry())
	input := &CreateSourceInput{
		Name:        "my-source",
		SourceType:  "booru",
		Params:      json.RawMessage(`{"tags":["nature"]}`),
		LookupCount: 50,
		IsEnabled:   true,
	}

	src, err := svc.CreateSource(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateSource failed: %v", err)
	}
	if src.Name != "my-source" {
		t.Errorf("expected name 'my-source', got %q", src.Name)
	}
	if src.SourceType != "booru" {
		t.Errorf("expected source_type 'booru', got %q", src.SourceType)
	}
	if src.LookupCount != 50 {
		t.Errorf("expected lookup_count 50, got %d", src.LookupCount)
	}
}

func TestServiceCreateSourceKeepsZeroLookupCount(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, testRegistry())
	input := &CreateSourceInput{
		Name:        "my-source",
		SourceType:  "booru",
		Params:      json.RawMessage(`{"tags":["nature"]}`),
		LookupCount: 0,
		IsEnabled:   true,
	}

	src, err := svc.CreateSource(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateSource failed: %v", err)
	}
	if src.LookupCount != 0 {
		t.Errorf("expected lookup_count 0 to be preserved, got %d", src.LookupCount)
	}
}

func TestServiceCreateSourceDuplicateName(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	// Insert existing source
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"01800000-0000-0000-0000-000000000001", "existing-source", "booru",
		`{"tags":["landscape"]}`, 100, 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}

	svc := NewService(db, testRegistry())
	input := &CreateSourceInput{
		Name:        "existing-source",
		SourceType:  "booru",
		Params:      json.RawMessage(`{"tags":["nature"]}`),
		LookupCount: 50,
		IsEnabled:   true,
	}

	_, err = svc.CreateSource(context.Background(), input)
	if !errors.Is(err, ErrDuplicateSourceName) {
		t.Errorf("expected ErrDuplicateSourceName, got %v", err)
	}
}

func TestServiceCreateSourceInvalidType(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, testRegistry())
	input := &CreateSourceInput{
		Name:        "my-source",
		SourceType:  "nonexistent",
		Params:      json.RawMessage(`{}`),
		LookupCount: 50,
		IsEnabled:   true,
	}

	_, err := svc.CreateSource(context.Background(), input)
	if !errors.Is(err, ErrInvalidSourceType) {
		t.Errorf("expected ErrInvalidSourceType, got %v", err)
	}
}

func TestServiceCreateSourceInvalidParams(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, testRegistry())
	// booru requires at least one tag or booru_host
	input := &CreateSourceInput{
		Name:        "my-source",
		SourceType:  "booru",
		Params:      json.RawMessage(`{"rating":"safe"}`), // Missing tags
		LookupCount: 50,
		IsEnabled:   true,
	}

	_, err := svc.CreateSource(context.Background(), input)
	if !errors.Is(err, ErrInvalidParams) {
		t.Errorf("expected ErrInvalidParams, got %v", err)
	}
}

func TestServiceUpdateSource(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"01800000-0000-0000-0000-000000000001", "test-source", "booru",
		`{"tags":["landscape"]}`, 100, 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}

	svc := NewService(db, testRegistry())
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	input := &UpdateSourceInput{
		ID:          id,
		Name:        "updated-source",
		SourceType:  "booru",
		Params:      json.RawMessage(`{"tags":["nature","mountain"]}`),
		LookupCount: 200,
		IsEnabled:   false,
	}

	src, err := svc.UpdateSource(context.Background(), input)
	if err != nil {
		t.Fatalf("UpdateSource failed: %v", err)
	}
	if src.Name != "updated-source" {
		t.Errorf("expected name 'updated-source', got %q", src.Name)
	}
	if src.LookupCount != 200 {
		t.Errorf("expected lookup_count 200, got %d", src.LookupCount)
	}
	if bool(src.IsEnabled) != false {
		t.Errorf("expected is_enabled false, got %v", bool(src.IsEnabled))
	}
}

func TestServiceUpdateSourceNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, testRegistry())
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	input := &UpdateSourceInput{
		ID:          id,
		Name:        "updated-source",
		SourceType:  "booru",
		Params:      json.RawMessage(`{"tags":["nature"]}`),
		LookupCount: 50,
		IsEnabled:   true,
	}

	_, err := svc.UpdateSource(context.Background(), input)
	if !errors.Is(err, ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestServiceUpdateSourceDuplicateName(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	// Insert two sources
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"01800000-0000-0000-0000-000000000001", "source-1", "booru",
		`{"tags":["landscape"]}`, 100, 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test source 1: %v", err)
	}
	_, err = db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"01800000-0000-0000-0000-000000000002", "source-2", "booru",
		`{"tags":["nature"]}`, 100, 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test source 2: %v", err)
	}

	svc := NewService(db, testRegistry())
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	input := &UpdateSourceInput{
		ID:          id,
		Name:        "source-2", // Try to rename to existing name
		SourceType:  "booru",
		Params:      json.RawMessage(`{"tags":["landscape"]}`),
		LookupCount: 100,
		IsEnabled:   true,
	}

	_, err = svc.UpdateSource(context.Background(), input)
	if !errors.Is(err, ErrDuplicateSourceName) {
		t.Errorf("expected ErrDuplicateSourceName, got %v", err)
	}
}

func TestServiceDeleteSource(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"01800000-0000-0000-0000-000000000001", "test-source", "booru",
		`{"tags":["landscape"]}`, 100, 1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}

	svc := NewService(db, testRegistry())
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	err = svc.DeleteSource(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteSource failed: %v", err)
	}

	// Verify deleted
	_, err = svc.GetSource(context.Background(), id)
	if !errors.Is(err, ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound after delete, got %v", err)
	}
}

func TestServiceDeleteSourceNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, testRegistry())
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	err := svc.DeleteSource(context.Background(), id)
	if !errors.Is(err, ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestServiceReturnsErrDBUnavailableWhenDBIsNil(t *testing.T) {
	svc := NewService(nil, testRegistry())

	_, err := svc.ListSources(context.Background())
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}

	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	_, err = svc.GetSource(context.Background(), id)
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}

	_, err = svc.CreateSource(context.Background(), &CreateSourceInput{})
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}

	_, err = svc.UpdateSource(context.Background(), &UpdateSourceInput{ID: id})
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}

	err = svc.DeleteSource(context.Background(), id)
	if !errors.Is(err, ErrDBUnavailable) {
		t.Errorf("expected ErrDBUnavailable, got %v", err)
	}
}

func TestServiceReturnsErrRegistryUnavailableWhenRegistryIsNil(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(db, nil)
	_, err := svc.CreateSource(context.Background(), &CreateSourceInput{
		Name:       "my-source",
		SourceType: "booru",
		Params:     json.RawMessage(`{"tags":["nature"]}`),
	})
	if !errors.Is(err, ErrRegistryUnavailable) {
		t.Errorf("expected ErrRegistryUnavailable, got %v", err)
	}
}
