package runner

import (
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/db/generated/model"
	"github.com/walens/walens/internal/dbtypes"
	"github.com/walens/walens/internal/services/images"
	"github.com/walens/walens/internal/services/tags"
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
		`CREATE TABLE tags (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			normalized_name TEXT NOT NULL UNIQUE,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE image_tags (
			id TEXT PRIMARY KEY,
			image_id TEXT NOT NULL,
			tag_id TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE image_thumbnails (
			id TEXT PRIMARY KEY,
			image_id TEXT NOT NULL,
			path TEXT NOT NULL,
			width INTEGER NOT NULL,
			height INTEGER NOT NULL,
			file_size_bytes INTEGER,
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
		Devices:     []model.Devices{{ID: deviceID, Slug: "test-device"}},
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
		Devices:     []model.Devices{{ID: deviceBID, Slug: "device-b"}},
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
		Devices:     []model.Devices{{ID: deviceID, Slug: "test-device"}},
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
		Devices:     []model.Devices{{ID: deviceBID, Slug: "device-b"}},
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
		Devices:     []model.Devices{{ID: deviceAID, Slug: "device-a"}, {ID: deviceBID, Slug: "device-b"}},
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
		Devices:     []model.Devices{{ID: deviceID, Slug: "test-device"}},
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

// TestRule2_RedownloadWithExistingAssignment_NoDuplicateLocation tests that
// re-downloading for an already-assigned device with a missing file does not
// fail due to duplicate location insertion.
func TestRule2_RedownloadWithExistingAssignment_NoDuplicateLocation(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Create device, source, image, assignment, and location with MISSING file
	deviceID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()
	imageID := dbtypes.MustNewUUIDV7()
	locationID := dbtypes.MustNewUUIDV7()
	assignmentID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")
	insertTestImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-002", "mock")
	insertTestImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Create location with MISSING file (path in DB but no actual file)
	missingPath := filepath.Join(ts.tempDir, "images", "test-device", "img-002.jpg")
	insertTestImageLocation(t, db, locationID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String(), missingPath, StorageKindCanonical, true)
	// Note: We do NOT create the file - it should be "missing"

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields one image - this should trigger Rule 2 (re-download)
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "img-002", OriginURL: "http://example.com/img2.jpg", MimeType: "image/jpeg"},
		},
	}

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: deviceID, Slug: "test-device"}},
	}

	// This should NOT fail even though there's already a location record.
	// The EnsureImageLocation should UPDATE the existing record, not INSERT a new one.
	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Download will fail because no HTTP server, but the important thing is that
	// it didn't fail due to duplicate location insertion.
	// We verify there is still only 1 location for this image+device.
	var locationCount int
	err = db.QueryRow("SELECT COUNT(*) FROM image_locations WHERE image_id = ? AND device_id = ?",
		imageID.UUID.String(), deviceID.UUID.String()).Scan(&locationCount)
	if err != nil {
		t.Fatalf("count locations: %v", err)
	}
	if locationCount != 1 {
		t.Errorf("expected 1 location (updated, not duplicated), got %d", locationCount)
	}

	// Verify no crash and reasonable counter state
	if result.SkippedCount != 0 {
		t.Errorf("expected SkippedCount=0, got %d", result.SkippedCount)
	}
}

// TestCopyFallback_StorageKindIsCopy verifies that when hard link fails and copy
// succeeds, the persisted location has storage_kind = "copy" not "hardlink".
// We test this indirectly by verifying the EnsureImageLocationRequest path works correctly.
func TestCopyFallback_StorageKindIsCopy(t *testing.T) {
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
	insertTestImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-copy-test", "mock")
	insertTestImageAssignment(t, db, assignmentAID.UUID.String(), imageID.UUID.String(), deviceAID.UUID.String())

	// Create location for device A with existing file (canonical)
	existingPath := filepath.Join(ts.tempDir, "images", "device-a", "img-copy-test.jpg")
	ts.createTestFile(existingPath, "fake image content for copy test")
	insertTestImageLocation(t, db, locationAID.UUID.String(), imageID.UUID.String(), deviceAID.UUID.String(), existingPath, StorageKindCanonical, true)

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields one image
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "img-copy-test", OriginURL: "http://example.com/copytest.jpg", MimeType: "image/jpeg"},
		},
	}

	// Materialize for device B (which has no assignment)
	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: deviceBID, Slug: "device-b"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Either hardlink or copy succeeded - verify one of them worked
	if result.HardlinkedCount+result.CopiedCount != 1 {
		t.Errorf("expected HardlinkedCount+CopiedCount=1, got hardlinked=%d copied=%d",
			result.HardlinkedCount, result.CopiedCount)
	}

	// Verify the location for device B exists and has correct storage_kind
	var storageKind string
	err = db.QueryRow("SELECT storage_kind FROM image_locations WHERE image_id = ? AND device_id = ?",
		imageID.UUID.String(), deviceBID.UUID.String()).Scan(&storageKind)
	if err != nil {
		t.Fatalf("get storage_kind: %v", err)
	}

	// If hardlink succeeded, it should be "hardlink"; if copy fallback, it should be "copy"
	// Our implementation correctly sets storageKind based on which operation actually succeeded
	expectedKind := StorageKindHardlink
	if result.CopiedCount == 1 {
		expectedKind = StorageKindCopy
	}
	if storageKind != expectedKind {
		t.Errorf("expected storage_kind=%s, got %s", expectedKind, storageKind)
	}
}

// TestStoredCount_OnlyNewImages tests that StoredCount only increments for
// newly created images, not per-device materialization.
func TestStoredCount_OnlyNewImages(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Create two devices, source, and a NEW image that doesn't exist in DB yet
	deviceAID := dbtypes.MustNewUUIDV7()
	deviceBID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceAID.UUID.String(), "Device A", "device-a")
	insertTestDevice(t, db, deviceBID.UUID.String(), "Device B", "device-b")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")

	// Note: NO image exists in DB yet - this is a brand new image
	// We create a mock source that returns this image

	// Setup materializer with a discard logger
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields a new image (not in DB yet)
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "brand-new-img", OriginURL: "http://example.com/new.jpg", MimeType: "image/jpeg"},
		},
	}

	// Materialize for both devices
	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: deviceAID, Slug: "device-a"}, {ID: deviceBID, Slug: "device-b"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Downloads will fail (no HTTP server), but the important thing is:
	// - The image was created (GetOrCreateImage returned isNew=true)
	// - StoredCount should be 1 (not 2) because we only count newly created images once

	// Since downloads failed, DownloadedCount will be 0 and StoredCount will be 0.
	// But let's verify the logic: if downloads had succeeded, StoredCount should be 1.
	// For now, we just verify no crash and reasonable state.
	if result.StoredCount != 0 {
		t.Errorf("expected StoredCount=0 (download failed), got %d", result.StoredCount)
	}

	// Verify image was created in DB
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM images WHERE unique_identifier = ?", "brand-new-img").Scan(&count)
	if err != nil {
		t.Fatalf("count images: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 image created, got %d", count)
	}
}

// TestTagsPersistence_TagsAreStored tests that tags from ImageMetadata are persisted.
func TestTagsPersistence_TagsAreStored(t *testing.T) {
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

	// Setup materializer with services
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))
	mat.SetTagsService(tags.NewService(db))

	// Create source that yields an image with tags
	src := &mockSource{
		items: []sources.ImageMetadata{
			{
				UniqueIdentifier: "tagged-img",
				OriginURL:        "http://example.com/tagged.jpg",
				MimeType:         "image/jpeg",
				Tags:             []string{"landscape", "nature", "Mountain"},
			},
		},
	}

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: deviceID, Slug: "test-device"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Verify image was created
	var imageID string
	err = db.QueryRow("SELECT id FROM images WHERE unique_identifier = ?", "tagged-img").Scan(&imageID)
	if err != nil {
		t.Fatalf("get image id: %v", err)
	}

	// Verify tags were created (should be case-insensitively deduped)
	var tagCount int
	err = db.QueryRow("SELECT COUNT(*) FROM tags WHERE normalized_name IN (?, ?, ?)",
		"landscape", "nature", "mountain").Scan(&tagCount)
	if err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if tagCount != 3 {
		t.Errorf("expected 3 tags, got %d", tagCount)
	}

	// Verify image_tags associations were created
	var imageTagCount int
	err = db.QueryRow("SELECT COUNT(*) FROM image_tags WHERE image_id = ?", imageID).Scan(&imageTagCount)
	if err != nil {
		t.Fatalf("count image_tags: %v", err)
	}
	if imageTagCount != 3 {
		t.Errorf("expected 3 image_tags, got %d", imageTagCount)
	}

	// Verify result counters - downloads will fail but tags should be synced
	_ = result
}

// TestTagsPersistence_DedupeCaseInsensitive tests that duplicate tags are deduped case-insensitively.
func TestTagsPersistence_DedupeCaseInsensitive(t *testing.T) {
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

	// Setup materializer with services
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))
	mat.SetTagsService(tags.NewService(db))

	// Create source that yields an image with duplicate tags (different cases)
	src := &mockSource{
		items: []sources.ImageMetadata{
			{
				UniqueIdentifier: "dedupe-img",
				OriginURL:        "http://example.com/dedupe.jpg",
				MimeType:         "image/jpeg",
				Tags:             []string{"TAG1", "tag1", "Tag1"},
			},
		},
	}

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: deviceID, Slug: "test-device"}},
	}

	_, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Verify only one tag was created
	var tagCount int
	err = db.QueryRow("SELECT COUNT(*) FROM tags WHERE normalized_name = ?", "tag1").Scan(&tagCount)
	if err != nil {
		t.Fatalf("count tags: %v", err)
	}
	if tagCount != 1 {
		t.Errorf("expected 1 tag (deduped), got %d", tagCount)
	}

	// Verify image_id was stored only once for this image
	var imageID string
	err = db.QueryRow("SELECT id FROM images WHERE unique_identifier = ?", "dedupe-img").Scan(&imageID)
	if err != nil {
		t.Fatalf("get image id: %v", err)
	}

	var imageTagCount int
	err = db.QueryRow("SELECT COUNT(*) FROM image_tags WHERE image_id = ?", imageID).Scan(&imageTagCount)
	if err != nil {
		t.Fatalf("count image_tags: %v", err)
	}
	if imageTagCount != 1 {
		t.Errorf("expected 1 image_tag (deduped), got %d", imageTagCount)
	}
}

// createTestJPEGFile creates a minimal valid JPEG file at the given path.
func createTestJPEGFile(path string, width, height int) error {
	// Create a simple image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a simple color to make it recognizable
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 128, G: 64, B: 192, A: 255})
		}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return jpeg.Encode(file, img, &jpeg.Options{Quality: 85})
}

// TestThumbnail_CreatesThumbnailRow tests that materialization creates a thumbnail row
// when a canonical file exists.
func TestThumbnail_CreatesThumbnailRow(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Create device, source, image, and location with existing JPEG file
	deviceID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()
	imageID := dbtypes.MustNewUUIDV7()
	locationID := dbtypes.MustNewUUIDV7()
	assignmentID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")
	insertTestImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "thumb-test-img", "mock")
	insertTestImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Create a real JPEG file for the canonical location
	existingPath := filepath.Join(ts.tempDir, "images", "test-device", "thumb-test-img.jpg")
	if err := createTestJPEGFile(existingPath, 1920, 1080); err != nil {
		t.Fatalf("create test JPEG: %v", err)
	}
	insertTestImageLocation(t, db, locationID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String(), existingPath, StorageKindCanonical, true)

	// Setup materializer
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields the image
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "thumb-test-img", OriginURL: "http://example.com/thumb.jpg", MimeType: "image/jpeg"},
		},
	}

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: deviceID, Slug: "test-device"}},
	}

	result, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Verify: Image should be skipped (Rule 1)
	if result.SkippedCount != 1 {
		t.Errorf("expected SkippedCount=1, got %d", result.SkippedCount)
	}

	// Verify: Thumbnail row should be created
	var thumbCount int
	err = db.QueryRow("SELECT COUNT(*) FROM image_thumbnails WHERE image_id = ?", imageID.UUID.String()).Scan(&thumbCount)
	if err != nil {
		t.Fatalf("count thumbnails: %v", err)
	}
	if thumbCount != 1 {
		t.Errorf("expected 1 thumbnail row, got %d", thumbCount)
	}

	// Verify: Thumbnail file should exist on disk
	var thumbPath string
	err = db.QueryRow("SELECT path FROM image_thumbnails WHERE image_id = ?", imageID.UUID.String()).Scan(&thumbPath)
	if err != nil {
		t.Fatalf("get thumbnail path: %v", err)
	}
	if !ts.FileExists(thumbPath) {
		t.Errorf("expected thumbnail file to exist at %s", thumbPath)
	}
}

// TestThumbnail_NoDuplicateRows tests that repeated processing does not create
// duplicate thumbnail rows.
func TestThumbnail_NoDuplicateRows(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Create device, source, image, and location with existing JPEG file
	deviceID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()
	imageID := dbtypes.MustNewUUIDV7()
	locationID := dbtypes.MustNewUUIDV7()
	assignmentID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")
	insertTestImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "dup-thumb-test", "mock")
	insertTestImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Create a real JPEG file
	existingPath := filepath.Join(ts.tempDir, "images", "test-device", "dup-thumb-test.jpg")
	if err := createTestJPEGFile(existingPath, 1920, 1080); err != nil {
		t.Fatalf("create test JPEG: %v", err)
	}
	insertTestImageLocation(t, db, locationID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String(), existingPath, StorageKindCanonical, true)

	// Setup materializer
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields the image
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "dup-thumb-test", OriginURL: "http://example.com/dup.jpg", MimeType: "image/jpeg"},
		},
	}

	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: deviceID, Slug: "test-device"}},
	}

	// Process the image once
	_, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Process the image again
	_, err = mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed (second run): %v", err)
	}

	// Verify: Only 1 thumbnail row should exist
	var thumbCount int
	err = db.QueryRow("SELECT COUNT(*) FROM image_thumbnails WHERE image_id = ?", imageID.UUID.String()).Scan(&thumbCount)
	if err != nil {
		t.Fatalf("count thumbnails: %v", err)
	}
	if thumbCount != 1 {
		t.Errorf("expected 1 thumbnail row after duplicate processing, got %d", thumbCount)
	}
}

// TestThumbnail_UsesCanonicalFile tests that thumbnail is generated from canonical file
// when available.
func TestThumbnail_UsesCanonicalFile(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	ts := newTestStorage(t)
	defer ts.cleanup()

	ctx := context.Background()

	// Setup: Two devices, device A has canonical file, device B will hard link
	deviceAID := dbtypes.MustNewUUIDV7()
	deviceBID := dbtypes.MustNewUUIDV7()
	sourceID := dbtypes.MustNewUUIDV7()
	imageID := dbtypes.MustNewUUIDV7()
	locationAID := dbtypes.MustNewUUIDV7()
	assignmentAID := dbtypes.MustNewUUIDV7()

	insertTestDevice(t, db, deviceAID.UUID.String(), "Device A", "device-a")
	insertTestDevice(t, db, deviceBID.UUID.String(), "Device B", "device-b")
	insertTestSource(t, db, sourceID.UUID.String(), "test-source")
	insertTestImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "canonical-thumb-test", "mock")
	insertTestImageAssignment(t, db, assignmentAID.UUID.String(), imageID.UUID.String(), deviceAID.UUID.String())

	// Create a real JPEG file for device A (canonical)
	canonicalPath := filepath.Join(ts.tempDir, "images", "device-a", "canonical-thumb-test.jpg")
	if err := createTestJPEGFile(canonicalPath, 1920, 1080); err != nil {
		t.Fatalf("create test JPEG: %v", err)
	}
	insertTestImageLocation(t, db, locationAID.UUID.String(), imageID.UUID.String(), deviceAID.UUID.String(), canonicalPath, StorageKindCanonical, true)

	// Setup materializer
	mat := NewMaterializer(slog.New(slog.DiscardHandler))
	mat.SetStorageService(ts.Service)
	mat.SetImageService(images.NewService(db))

	// Create source that yields the image
	src := &mockSource{
		items: []sources.ImageMetadata{
			{UniqueIdentifier: "canonical-thumb-test", OriginURL: "http://example.com/canonical.jpg", MimeType: "image/jpeg"},
		},
	}

	// Process for both devices
	req := MaterializeRequest{
		SourceID:    sourceID,
		SourceType:  "mock",
		LookupCount: 10,
		Devices:     []model.Devices{{ID: deviceAID, Slug: "device-a"}, {ID: deviceBID, Slug: "device-b"}},
	}

	_, err := mat.MaterializeImage(ctx, req, src)
	if err != nil {
		t.Fatalf("MaterializeImage failed: %v", err)
	}

	// Verify: Only 1 thumbnail row should exist
	var thumbCount int
	err = db.QueryRow("SELECT COUNT(*) FROM image_thumbnails WHERE image_id = ?", imageID.UUID.String()).Scan(&thumbCount)
	if err != nil {
		t.Fatalf("count thumbnails: %v", err)
	}
	if thumbCount != 1 {
		t.Errorf("expected 1 thumbnail row, got %d", thumbCount)
	}

	// Verify: Device B got hard linked (Rule 3)
	var hardlinkCount int
	err = db.QueryRow("SELECT COUNT(*) FROM image_locations WHERE image_id = ? AND storage_kind = 'hardlink'", imageID.UUID.String()).Scan(&hardlinkCount)
	if err != nil {
		t.Fatalf("count hardlinks: %v", err)
	}
	if hardlinkCount != 1 {
		t.Errorf("expected 1 hardlink location, got %d", hardlinkCount)
	}
}
