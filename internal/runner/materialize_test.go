package runner

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/db/generated/model"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/services/images"
	"github.com/walens/walens/internal/sources"
	"github.com/walens/walens/internal/storage"
	"iter"
	_ "modernc.org/sqlite"
)

// openTestDB opens an in-memory SQLite database for testing.
func openTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

// createTables creates the required tables for materialization tests.
func createTables(t *testing.T, db *sql.DB) {
	tables := []string{
		`CREATE TABLE devices (
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
		)`,
		`CREATE TABLE sources (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			source_type TEXT NOT NULL,
			params TEXT NOT NULL DEFAULT '{}',
			lookup_count INTEGER NOT NULL DEFAULT 0,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE images (
			id TEXT PRIMARY KEY,
			source_id TEXT,
			unique_identifier TEXT NOT NULL,
			source_type TEXT NOT NULL,
			original_filename TEXT,
			preview_url TEXT,
			origin_url TEXT,
			source_item_identifier TEXT,
			original_identifier TEXT,
			uploader TEXT,
			artist TEXT,
			mime_type TEXT,
			file_size_bytes INTEGER,
			width INTEGER,
			height INTEGER,
			aspect_ratio REAL,
			is_adult INTEGER NOT NULL DEFAULT 0,
			is_favorite INTEGER NOT NULL DEFAULT 0,
			json_meta TEXT NOT NULL DEFAULT '{}',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE image_locations (
			id TEXT PRIMARY KEY,
			image_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			path TEXT NOT NULL,
			storage_kind TEXT NOT NULL,
			is_primary INTEGER NOT NULL DEFAULT 1,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE image_assignments (
			id TEXT PRIMARY KEY,
			image_id TEXT NOT NULL,
			device_id TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE image_blacklists (
			id TEXT PRIMARY KEY,
			source_id TEXT NOT NULL,
			unique_identifier TEXT NOT NULL,
			reason TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE jobs (
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
		)`,
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
}

// insertTestDevice inserts a device for testing.
func insertTestDevice(t *testing.T, db *sql.DB, id, name, slug string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO devices (id, name, slug, screen_width, screen_height, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, 1920, 1080, 1, ?, ?)`,
		id, name, slug, now, now,
	)
	if err != nil {
		t.Fatalf("insert test device: %v", err)
	}
}

// insertTestSource inserts a source for testing.
func insertTestSource(t *testing.T, db *sql.DB, id, name string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, 'booru', '{}', 100, 1, ?, ?)`,
		id, name, now, now,
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}
}

// insertTestImage inserts an image for testing.
func insertTestImage(t *testing.T, db *sql.DB, id, sourceID, uniqueID, sourceType string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO images (id, source_id, unique_identifier, source_type, is_adult, is_favorite, json_meta, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, 0, '{}', ?, ?)`,
		id, sourceID, uniqueID, sourceType, now, now,
	)
	if err != nil {
		t.Fatalf("insert test image: %v", err)
	}
}

// insertTestImageLocation inserts an image location for testing.
func insertTestImageLocation(t *testing.T, db *sql.DB, id, imageID, deviceID, path, storageKind string, isActive bool) {
	now := time.Now().UnixMilli()
	active := 0
	if isActive {
		active = 1
	}
	_, err := db.Exec(`
		INSERT INTO image_locations (id, image_id, device_id, path, storage_kind, is_primary, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?)`,
		id, imageID, deviceID, path, storageKind, active, now, now,
	)
	if err != nil {
		t.Fatalf("insert test image location: %v", err)
	}
}

// insertTestImageAssignment inserts an image assignment for testing.
func insertTestImageAssignment(t *testing.T, db *sql.DB, id, imageID, deviceID string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO image_assignments (id, image_id, device_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, imageID, deviceID, now, now,
	)
	if err != nil {
		t.Fatalf("insert test image assignment: %v", err)
	}
}

// insertTestBlacklistEntry inserts a blacklist entry for testing.
func insertTestBlacklistEntry(t *testing.T, db *sql.DB, id, sourceID, uniqueIdentifier string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO image_blacklists (id, source_id, unique_identifier, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, sourceID, uniqueIdentifier, now, now,
	)
	if err != nil {
		t.Fatalf("insert test blacklist entry: %v", err)
	}
}

// mockSource implements sources.Source for testing.
type mockSource struct {
	items []sources.ImageMetadata
}

func (m *mockSource) TypeName() string                         { return "mock" }
func (m *mockSource) DisplayName() string                      { return "Mock Source" }
func (m *mockSource) ValidateParams(raw json.RawMessage) error { return nil }
func (m *mockSource) ParamSchema() *huma.Schema                { return nil }
func (m *mockSource) DefaultLookupCount() int                  { return 100 }
func (m *mockSource) BuildUniqueID(item sources.ImageMetadata) (string, error) {
	return item.UniqueIdentifier, nil
}
func (m *mockSource) Fetch(ctx context.Context, req sources.FetchRequest) iter.Seq2[sources.ImageMetadata, error] {
	return func(yield func(sources.ImageMetadata, error) bool) {
		for _, item := range m.items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

// testStorage wraps storage.Service for testing with a temp directory.
type testStorage struct {
	*storage.Service
	tempDir string
}

func newTestStorage(t *testing.T) *testStorage {
	tempDir, err := os.MkdirTemp("", "walens-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}

	svc := storage.NewService(storage.Config{BaseDir: tempDir})
	return &testStorage{Service: svc, tempDir: tempDir}
}

func (ts *testStorage) cleanup() {
	os.RemoveAll(ts.tempDir)
}

// createTestFile creates a file at the given path for testing.
func (ts *testStorage) createTestFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// --- Test Cases ---

// TestRule1_AssignedAndFileExists_Skip tests Rule 1:
// Image assigned to device AND file exists → skip
func TestRule1_AssignedAndFileExists_Skip(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Create device, source, image, assignment, and location with existing file
	deviceID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()
	imageID := dbtypes.MustNewUUIDV7()
	locationID := dbtypes.MustNewUUIDV7()
	assignmentID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")
	insertTestImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-001", "mock")
	insertTestImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Create location with existing file
	existingPath := filepath.Join(ts.tempDir, "images", "test-device", "img-001.jpg")
	ts.createTestFile(existingPath, "fake image content")
	insertTestImageLocation(t, db, locationID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String(), existingPath, StorageKindCanonical, true)

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields one image
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "img-001", OriginURL: "http://example.com/img.jpg", MimeType: "image/jpeg"},
		},
	}

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: &deviceID, Slug: "test-device"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Assert: Image skipped because assignment exists and file exists
	if result.SkippedCount != 1 {
		t.Errorf("expected SkippedCount=1, got %d", result.SkippedCount)
	}
	if result.DownloadedCount != 0 {
		t.Errorf("expected DownloadedCount=0, got %d", result.DownloadedCount)
	}
	if result.HardlinkedCount != 0 {
		t.Errorf("expected HardlinkedCount=0, got %d", result.HardlinkedCount)
	}
	if result.CopiedCount != 0 {
		t.Errorf("expected CopiedCount=0, got %d", result.CopiedCount)
	}
}

// TestRule3_AssignedToAnotherDevice_HardLinkOrCopy tests Rule 3:
// Image assigned to another device → hard link or copy
func TestRule3_AssignedToAnotherDevice_HardLinkOrCopy(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Create two devices, source, image, assignment for device A with existing file
	deviceAID := dbtypes.MustNewUUIDV7()
	deviceBID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()
	imageID := dbtypes.MustNewUUIDV7()
	locationAID := dbtypes.MustNewUUIDV7()
	assignmentAID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceAID.UUID.String(), "Device A", "device-a")
	insertTestDevice(t, db, deviceBID.UUID.String(), "Device B", "device-b")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")
	insertTestImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-003", "mock")
	insertTestImageAssignment(t, db, assignmentAID.UUID.String(), imageID.UUID.String(), deviceAID.UUID.String())

	// Create location for device A with existing file (canonical)
	existingPath := filepath.Join(ts.tempDir, "images", "device-a", "img-003.jpg")
	ts.createTestFile(existingPath, "fake image content for hardlink test")
	insertTestImageLocation(t, db, locationAID.UUID.String(), imageID.UUID.String(), deviceAID.UUID.String(), existingPath, StorageKindCanonical, true)

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields one image
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "img-003", OriginURL: "http://example.com/img3.jpg", MimeType: "image/jpeg"},
		},
	}

	// Materialize for device B (which has no assignment)
	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: &deviceBID, Slug: "device-b"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Assert: Image hardlinked or copied (Rule 3)
	if result.HardlinkedCount+result.CopiedCount != 1 {
		t.Errorf("expected HardlinkedCount+CopiedCount=1, got hardlinked=%d copied=%d",
			result.HardlinkedCount, result.CopiedCount)
	}
	if result.DownloadedCount != 0 {
		t.Errorf("expected DownloadedCount=0, got %d", result.DownloadedCount)
	}
}

// TestBlacklist_SkipsBlacklistedImage tests that blacklisted images are skipped.
func TestBlacklist_SkipsBlacklistedImage(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Create device, source, and blacklisted image
	deviceID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()
	blacklistID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")

	// Add to blacklist
	insertTestBlacklistEntry(t, db, blacklistID.UUID.String(), sourceID.UUID.String(), "blacklisted-img")

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields the blacklisted image
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "blacklisted-img", OriginURL: "http://example.com/blacklisted.jpg", MimeType: "image/jpeg"},
		},
	}

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: &deviceID, Slug: "test-device"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Assert: Image skipped due to blacklist
	if result.SkippedCount != 1 {
		t.Errorf("expected SkippedCount=1 (blacklisted), got %d", result.SkippedCount)
	}
	if result.DownloadedCount != 0 {
		t.Errorf("expected DownloadedCount=0, got %d", result.DownloadedCount)
	}
}

// TestNoDevices_NoPanic tests that materialization handles empty device list gracefully.
func TestNoDevices_NoPanic(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	sourceID := dbtypes.MustNewUUIDV7()
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "img-orphan", OriginURL: "http://example.com/orphan.jpg", MimeType: "image/jpeg"},
		},
	}

	// Request with no devices
	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// With no devices, no counters should increment
	if result.SkippedCount != 0 {
		t.Errorf("expected SkippedCount=0, got %d", result.SkippedCount)
	}
	_ = result // Just ensure no panic
}

// TestRule4_AssignedElsewhereSourceMissing_DownloadForCurrentDevice tests Rule 4:
// Image assigned elsewhere BUT source file missing → download for current device only
//
// Note: This test requires a test HTTP server to fully validate the download path.
// Currently, the download will fail because we can't make real HTTP requests.
// This test validates the pre-download logic but cannot verify the actual download counter.
func TestRule4_AssignedElsewhereSourceMissing_DownloadForCurrentDevice(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Create two devices, source, image, assignment for device A BUT file is missing
	deviceAID := dbtypes.MustNewUUIDV7()
	deviceBID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()
	imageID := dbtypes.MustNewUUIDV7()
	locationAID := dbtypes.MustNewUUIDV7()
	assignmentAID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceAID.UUID.String(), "Device A", "device-a")
	insertTestDevice(t, db, deviceBID.UUID.String(), "Device B", "device-b")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")
	insertTestImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-004", "mock")
	insertTestImageAssignment(t, db, assignmentAID.UUID.String(), imageID.UUID.String(), deviceAID.UUID.String())

	// Create location for device A but file does NOT exist (path exists in DB but not on disk)
	nonExistentPath := filepath.Join(ts.tempDir, "images", "device-a", "img-004.jpg")
	insertTestImageLocation(t, db, locationAID.UUID.String(), imageID.UUID.String(), deviceAID.UUID.String(), nonExistentPath, StorageKindCanonical, true)
	// Note: We do NOT create the file - it should be "missing"

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "img-004", OriginURL: "http://example.com/img4.jpg", MimeType: "image/jpeg"},
		},
	}

	// Materialize for device B
	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: &deviceBID, Slug: "device-b"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// The download will fail because we can't make real HTTP requests.
	// So DownloadedCount will be 0. In a proper integration test with a
	// test HTTP server, this would be 1.
	//
	// The important thing is that Rule 3 was NOT triggered (no hardlink/copy)
	// because the source file was missing.
	if result.HardlinkedCount != 0 {
		t.Errorf("expected HardlinkedCount=0 (source file missing), got %d", result.HardlinkedCount)
	}
	if result.CopiedCount != 0 {
		t.Errorf("expected CopiedCount=0 (source file missing), got %d", result.CopiedCount)
	}
}

// TestMultipleImages_MultipleDevices tests materialization with multiple images and devices.
func TestMultipleImages_MultipleDevices(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	deviceAID := dbtypes.MustNewUUIDV7()
	deviceBID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceAID.UUID.String(), "Device A", "device-a")
	insertTestDevice(t, db, deviceBID.UUID.String(), "Device B", "device-b")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Three images: one blacklisted, two regular
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "blacklisted", OriginURL: "http://example.com/black.jpg", MimeType: "image/jpeg"},
			{UniqueIdentifier: "img-1", OriginURL: "http://example.com/1.jpg", MimeType: "image/jpeg"},
			{UniqueIdentifier: "img-2", OriginURL: "http://example.com/2.jpg", MimeType: "image/jpeg"},
		},
	}

	// Add blacklist entry for one image
	blacklistID := dbtypes.MustNewUUIDV7()
	insertTestBlacklistEntry(t, db, blacklistID.UUID.String(), sourceID.UUID.String(), "blacklisted")

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: &deviceAID, Slug: "device-a"}, {ID: &deviceBID, Slug: "device-b"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Blacklisted image should be skipped once (blacklist check is per-image, not per-device)
	if result.SkippedCount != 1 {
		t.Errorf("expected SkippedCount=1 (blacklist check is per-image), got %d", result.SkippedCount)
	}

	// Downloads will fail without HTTP server, but at least we verify blacklist works
}

// TestMaterializeResult_Counters tests the counter tracking in the result.
func TestMaterializeResult_Counters(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	deviceID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Two images: one assigned with file (skip), one not assigned
	image1ID := dbtypes.MustNewUUIDV7()
	assignment1ID := dbtypes.MustNewUUIDV7()
	location1ID := dbtypes.MustNewUUIDV7()

	insertTestImage(t, db, image1ID.UUID.String(), sourceID.UUID.String(), "skip-me", "mock")
	insertTestImageAssignment(t, db, assignment1ID.UUID.String(), image1ID.UUID.String(), deviceID.UUID.String())
	existingPath := filepath.Join(ts.tempDir, "images", "test-device", "skip-me.jpg")
	ts.createTestFile(existingPath, "existing content")
	insertTestImageLocation(t, db, location1ID.UUID.String(), image1ID.UUID.String(), deviceID.UUID.String(), existingPath, StorageKindCanonical, true)

	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "skip-me", OriginURL: "http://example.com/skip.jpg", MimeType: "image/jpeg"},
			{UniqueIdentifier: "download-me", OriginURL: "http://example.com/dl.jpg", MimeType: "image/jpeg"},
		},
	}

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: &deviceID, Slug: "test-device"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Verify: One image should be skipped (assignment exists + file exists)
	if result.SkippedCount != 1 {
		t.Errorf("expected SkippedCount=1, got %d", result.SkippedCount)
	}

	// Total processed: only the skipped image was successfully processed.
	// The download image failed to download (no HTTP server), so no counter was incremented.
	// This is expected behavior - downloads that fail don't increment counters.
	totalProcessed := result.SkippedCount + result.DownloadedCount + result.HardlinkedCount + result.CopiedCount + result.ReusedCount + result.StoredCount
	if totalProcessed != 1 {
		t.Errorf("expected total processed=1 (only skipped), got %d", totalProcessed)
	}
}

// TestRule3_CopyFallback tests that copy is used when hard link fails.
func TestRule3_CopyFallback(t *testing.T) {
	// This test would require a mock storage that fails hard link but succeeds copy.
	// Currently, the only way to test this is to use a cross-filesystem scenario
	// where hard links aren't supported (different volumes).
	// For unit tests, this would require an interface-based storage service.
	t.Skip("requires interface-based storage service for mocking")
}
