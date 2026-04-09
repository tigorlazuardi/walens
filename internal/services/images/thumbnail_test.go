package images

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/walens/walens/internal/dbtypes"
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

// createTestTables creates the required tables for thumbnail tests.
func createTestTables(t *testing.T, db *sql.DB) {
	tables := []string{
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

// insertTestThumbnail inserts a thumbnail for testing.
func insertTestThumbnail(t *testing.T, db *sql.DB, id, imageID, path string, width, height int64, fileSizeBytes *int64) {
	now := time.Now().UnixMilli()
	var fs interface{}
	if fileSizeBytes != nil {
		fs = *fileSizeBytes
	}
	_, err := db.Exec(`
		INSERT INTO image_thumbnails (id, image_id, path, width, height, file_size_bytes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, imageID, path, width, height, fs, now, now,
	)
	if err != nil {
		t.Fatalf("insert test thumbnail: %v", err)
	}
}

// TestGetImageThumbnail_NotFound tests that GetImageThumbnail returns ErrThumbnailNotFound when no thumbnail exists.
func TestGetImageThumbnail_NotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTestTables(t, db)

	ctx := context.Background()
	svc := NewService(db)

	imageID := dbtypes.MustNewUUIDV7()

	_, err := svc.GetImageThumbnail(ctx, imageID)
	if err != ErrThumbnailNotFound {
		t.Errorf("expected ErrThumbnailNotFound, got %v", err)
	}
}

// TestGetImageThumbnail_Found tests that GetImageThumbnail returns the thumbnail when it exists.
func TestGetImageThumbnail_Found(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTestTables(t, db)

	ctx := context.Background()
	svc := NewService(db)

	imageID := dbtypes.MustNewUUIDV7()
	thumbnailID := dbtypes.MustNewUUIDV7()
	path := "/path/to/thumbnail.jpg"
	width := int64(512)
	height := int64(384)
	fileSize := int64(40960)

	insertTestImage(t, db, imageID.UUID.String(), "", "unique-1", "mock")
	insertTestThumbnail(t, db, thumbnailID.UUID.String(), imageID.UUID.String(), path, width, height, &fileSize)

	thumbnail, err := svc.GetImageThumbnail(ctx, imageID)
	if err != nil {
		t.Fatalf("GetImageThumbnail failed: %v", err)
	}

	if thumbnail.Path != path {
		t.Errorf("expected path %s, got %s", path, thumbnail.Path)
	}
	if thumbnail.Width != width {
		t.Errorf("expected width %d, got %d", width, thumbnail.Width)
	}
	if thumbnail.Height != height {
		t.Errorf("expected height %d, got %d", height, thumbnail.Height)
	}
}

// TestEnsureImageThumbnail_Create tests that EnsureImageThumbnail creates a new thumbnail when none exists.
func TestEnsureImageThumbnail_Create(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTestTables(t, db)

	ctx := context.Background()
	svc := NewService(db)

	imageID := dbtypes.MustNewUUIDV7()
	path := "/path/to/thumbnail.jpg"
	width := int64(512)
	height := int64(384)
	fileSize := int64(40960)

	insertTestImage(t, db, imageID.UUID.String(), "", "unique-1", "mock")

	thumbnail, err := svc.EnsureImageThumbnail(ctx, EnsureImageThumbnailRequest{
		ImageID:       imageID,
		Path:          path,
		Width:         width,
		Height:        height,
		FileSizeBytes: &fileSize,
	})
	if err != nil {
		t.Fatalf("EnsureImageThumbnail failed: %v", err)
	}

	if thumbnail.Path != path {
		t.Errorf("expected path %s, got %s", path, thumbnail.Path)
	}
	if thumbnail.Width != width {
		t.Errorf("expected width %d, got %d", width, thumbnail.Width)
	}
	if thumbnail.Height != height {
		t.Errorf("expected height %d, got %d", height, thumbnail.Height)
	}
	if thumbnail.FileSizeBytes == nil || *thumbnail.FileSizeBytes != fileSize {
		t.Errorf("expected file_size_bytes %d, got %v", fileSize, thumbnail.FileSizeBytes)
	}
}

// TestEnsureImageThumbnail_Update tests that EnsureImageThumbnail updates an existing thumbnail.
func TestEnsureImageThumbnail_Update(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTestTables(t, db)

	ctx := context.Background()
	svc := NewService(db)

	imageID := dbtypes.MustNewUUIDV7()
	thumbnailID := dbtypes.MustNewUUIDV7()
	oldPath := "/path/to/old-thumbnail.jpg"
	newPath := "/path/to/new-thumbnail.jpg"
	oldWidth := int64(256)
	newWidth := int64(512)
	oldHeight := int64(192)
	newHeight := int64(384)
	oldSize := int64(20480)
	newSize := int64(40960)

	insertTestImage(t, db, imageID.UUID.String(), "", "unique-1", "mock")
	insertTestThumbnail(t, db, thumbnailID.UUID.String(), imageID.UUID.String(), oldPath, oldWidth, oldHeight, &oldSize)

	// Ensure updated thumbnail
	thumbnail, err := svc.EnsureImageThumbnail(ctx, EnsureImageThumbnailRequest{
		ImageID:       imageID,
		Path:          newPath,
		Width:         newWidth,
		Height:        newHeight,
		FileSizeBytes: &newSize,
	})
	if err != nil {
		t.Fatalf("EnsureImageThumbnail failed: %v", err)
	}

	// ID should remain the same
	if thumbnail.ID.UUID != thumbnailID.UUID {
		t.Errorf("expected thumbnail ID %s, got %s", thumbnailID.UUID.String(), thumbnail.ID.UUID.String())
	}

	// Path should be updated
	if thumbnail.Path != newPath {
		t.Errorf("expected path %s, got %s", newPath, thumbnail.Path)
	}

	// Dimensions should be updated
	if thumbnail.Width != newWidth {
		t.Errorf("expected width %d, got %d", newWidth, thumbnail.Width)
	}
	if thumbnail.Height != newHeight {
		t.Errorf("expected height %d, got %d", newHeight, thumbnail.Height)
	}
}

// TestEnsureImageThumbnail_NoFileSize tests that EnsureImageThumbnail handles nil file_size_bytes.
func TestEnsureImageThumbnail_NoFileSize(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTestTables(t, db)

	ctx := context.Background()
	svc := NewService(db)

	imageID := dbtypes.MustNewUUIDV7()
	path := "/path/to/thumbnail.jpg"
	width := int64(512)
	height := int64(384)

	insertTestImage(t, db, imageID.UUID.String(), "", "unique-1", "mock")

	thumbnail, err := svc.EnsureImageThumbnail(ctx, EnsureImageThumbnailRequest{
		ImageID:       imageID,
		Path:          path,
		Width:         width,
		Height:        height,
		FileSizeBytes: nil,
	})
	if err != nil {
		t.Fatalf("EnsureImageThumbnail failed: %v", err)
	}

	if thumbnail.FileSizeBytes != nil {
		t.Errorf("expected nil file_size_bytes, got %v", *thumbnail.FileSizeBytes)
	}
}
