package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/sources"
	"github.com/walens/walens/internal/sources/booru"
	_ "modernc.org/sqlite"
)

func stringPtr(v string) *string                    { return &v }
func int64Ptr(v int64) *int64                       { return &v }
func boolPtr(v bool) *bool                          { return &v }
func rawJSONPtr(v json.RawMessage) *json.RawMessage { return &v }

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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	items, err := svc.ListSources(context.Background(), ListSourcesRequest{})
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if len(items.Items) != 0 {
		t.Errorf("expected 0 sources, got %d", len(items.Items))
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	items, err := svc.ListSources(context.Background(), ListSourcesRequest{})
	if err != nil {
		t.Fatalf("ListSources failed: %v", err)
	}
	if len(items.Items) != 1 {
		t.Errorf("expected 1 source, got %d", len(items.Items))
	}
	if items.Items[0].Name != "test-source" {
		t.Errorf("expected name 'test-source', got %q", items.Items[0].Name)
	}
	if items.Items[0].SourceType != "booru" {
		t.Errorf("expected source_type 'booru', got %q", items.Items[0].SourceType)
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	src, err := svc.GetSource(context.Background(), GetSourceRequest{ID: id})
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	_, err := svc.GetSource(context.Background(), GetSourceRequest{ID: id})
	if err == nil {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestServiceCreateSource(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	input := CreateSourceRequest{
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	input := CreateSourceRequest{
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	input := CreateSourceRequest{
		Name:        "existing-source",
		SourceType:  "booru",
		Params:      json.RawMessage(`{"tags":["nature"]}`),
		LookupCount: 50,
		IsEnabled:   true,
	}

	_, err = svc.CreateSource(context.Background(), input)
	if err == nil {
		t.Errorf("expected ErrDuplicateSourceName, got %v", err)
	}
}

func TestServiceCreateSourceInvalidType(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	input := CreateSourceRequest{
		Name:        "my-source",
		SourceType:  "nonexistent",
		Params:      json.RawMessage(`{}`),
		LookupCount: 50,
		IsEnabled:   true,
	}

	_, err := svc.CreateSource(context.Background(), input)
	if err == nil {
		t.Errorf("expected ErrInvalidSourceType, got %v", err)
	}
}

func TestServiceCreateSourceInvalidParams(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	// booru requires at least one tag or booru_host
	input := CreateSourceRequest{
		Name:        "my-source",
		SourceType:  "booru",
		Params:      json.RawMessage(`{"rating":"safe"}`), // Missing tags
		LookupCount: 50,
		IsEnabled:   true,
	}

	_, err := svc.CreateSource(context.Background(), input)
	if err == nil {
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	input := UpdateSourceRequest{
		ID:          id,
		Name:        stringPtr("updated-source"),
		SourceType:  stringPtr("booru"),
		Params:      rawJSONPtr(json.RawMessage(`{"tags":["nature","mountain"]}`)),
		LookupCount: int64Ptr(200),
		IsEnabled:   boolPtr(false),
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	input := UpdateSourceRequest{
		ID:          id,
		Name:        stringPtr("updated-source"),
		SourceType:  stringPtr("booru"),
		Params:      rawJSONPtr(json.RawMessage(`{"tags":["nature"]}`)),
		LookupCount: int64Ptr(50),
		IsEnabled:   boolPtr(true),
	}

	_, err := svc.UpdateSource(context.Background(), input)
	if err == nil {
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	input := UpdateSourceRequest{
		ID:          id,
		Name:        stringPtr("source-2"), // Try to rename to existing name
		SourceType:  stringPtr("booru"),
		Params:      rawJSONPtr(json.RawMessage(`{"tags":["landscape"]}`)),
		LookupCount: int64Ptr(100),
		IsEnabled:   boolPtr(true),
	}

	_, err = svc.UpdateSource(context.Background(), input)
	if err == nil {
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

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	_, err = svc.DeleteSource(context.Background(), DeleteSourceRequest{ID: id})
	if err != nil {
		t.Fatalf("DeleteSource failed: %v", err)
	}

	// Verify deleted
	_, err = svc.GetSource(context.Background(), GetSourceRequest{ID: id})
	if err == nil {
		t.Errorf("expected ErrSourceNotFound after delete, got %v", err)
	}
}

func TestServiceDeleteSourceNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})
	id, _ := dbtypes.NewUUIDFromString("01800000-0000-0000-0000-000000000001")
	_, err := svc.DeleteSource(context.Background(), DeleteSourceRequest{ID: id})
	if err == nil {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestServicePanicsWhenRegistryIsNil(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when registry is nil")
		}
	}()

	_ = NewService(ServiceDependencies{DB: db})
}

// TestListSources_TotalCount verifies that Total reflects all matching rows independent of pagination.
func TestListSources_TotalCount(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	// Insert 5 sources with "alpha" in name
	for i := 0; i < 5; i++ {
		_, err := db.Exec(`
			INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("01800000-0000-0000-0000-%012d", i),
			fmt.Sprintf("alpha-source-%d", i), "booru",
			`{}`, 100, 1, 1000, 1000,
		)
		if err != nil {
			t.Fatalf("insert test source: %v", err)
		}
	}

	// Insert 3 sources with "beta" in name
	for i := 5; i < 8; i++ {
		_, err := db.Exec(`
			INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("01800000-0000-0000-0000-%012d", i),
			fmt.Sprintf("beta-source-%d", i), "booru",
			`{}`, 100, 1, 1000, 1000,
		)
		if err != nil {
			t.Fatalf("insert test source: %v", err)
		}
	}

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})

	// Test: Total should be 5 for search matching "alpha"
	search := "alpha"
	resp, err := svc.ListSources(context.Background(), ListSourcesRequest{
		Search: &search,
	})
	if err != nil {
		t.Fatalf("ListSources with search failed: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("expected Total=5 for search 'alpha', got %d", resp.Total)
	}

	// Test: Total should be 8 when no search filter (all sources)
	respAll, err := svc.ListSources(context.Background(), ListSourcesRequest{})
	if err != nil {
		t.Fatalf("ListSources without search failed: %v", err)
	}
	if respAll.Total != 8 {
		t.Errorf("expected Total=8 for all sources, got %d", respAll.Total)
	}
}

// TestListSources_TotalNotAffectedByPagination verifies Total is independent of Limit.
func TestListSources_TotalNotAffectedByPagination(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createSourcesTable(t, db)

	// Insert 10 sources
	for i := 0; i < 10; i++ {
		_, err := db.Exec(`
			INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("01800000-0000-0000-0000-%012d", i),
			fmt.Sprintf("source-%d", i), "booru",
			`{}`, 100, 1, 1000, 1000,
		)
		if err != nil {
			t.Fatalf("insert test source: %v", err)
		}
	}

	svc := NewService(ServiceDependencies{DB: db, Registry: testRegistry()})

	// First page with small limit
	limit := 3
	resp, err := svc.ListSources(context.Background(), ListSourcesRequest{
		Pagination: &dbtypes.CursorPaginationRequest{Limit: &limit},
	})
	if err != nil {
		t.Fatalf("ListSources with limit failed: %v", err)
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
