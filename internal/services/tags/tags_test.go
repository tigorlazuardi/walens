package tags

import (
	"context"
	"database/sql"
	"log/slog"
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

// createTables creates the required tables for tag tests.
func createTables(t *testing.T, db *sql.DB) {
	tables := []string{
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
	}

	for _, table := range tables {
		_, err := db.Exec(table)
		if err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
}

// insertTestTag inserts a tag for testing.
func insertTestTag(t *testing.T, db *sql.DB, id, name, normalizedName string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO tags (id, name, normalized_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, name, normalizedName, now, now,
	)
	if err != nil {
		t.Fatalf("insert test tag: %v", err)
	}
}

// insertTestImageTag inserts an image-tag association for testing.
func insertTestImageTag(t *testing.T, db *sql.DB, id, imageID, tagID string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO image_tags (id, image_id, tag_id, created_at)
		VALUES (?, ?, ?, ?)`,
		id, imageID, tagID, now,
	)
	if err != nil {
		t.Fatalf("insert test image tag: %v", err)
	}
}

// --- Test Cases ---

// TestNormalizeTag tests tag normalization.
func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"   ", ""},
		{"Tag", "tag"},
		{"  Tag  ", "tag"},
		{"UPPER", "upper"},
		{"MixedCase", "mixedcase"},
		{"  lowercase  ", "lowercase"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := NormalizeTag(tc.input)
			if result != tc.expected {
				t.Errorf("NormalizeTag(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestEnsureTag_CreateNew tests creating a new tag.
func TestEnsureTag_CreateNew(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	tag, err := svc.EnsureTag(ctx, "MyTag")
	if err != nil {
		t.Fatalf("EnsureTag failed: %v", err)
	}
	if tag == nil {
		t.Fatal("expected tag, got nil")
	}
	if tag.Name != "MyTag" {
		t.Errorf("Name = %q, want %q", tag.Name, "MyTag")
	}
	if tag.NormalizedName != "mytag" {
		t.Errorf("NormalizedName = %q, want %q", tag.NormalizedName, "mytag")
	}
	if tag.ID.UUID.String() == "" {
		t.Error("ID should not be empty")
	}
}

// TestEnsureTag_Existing tests finding an existing tag.
func TestEnsureTag_Existing(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	tagID := dbtypes.MustNewUUIDV7()
	insertTestTag(t, db, tagID.UUID.String(), "OriginalName", "originalname")

	svc := NewService(db)
	ctx := context.Background()

	// Different case should find the same tag
	tag, err := svc.EnsureTag(ctx, "ORIGINALNAME")
	if err != nil {
		t.Fatalf("EnsureTag failed: %v", err)
	}
	if tag == nil {
		t.Fatal("expected tag, got nil")
	}
	if tag.ID.UUID.String() != tagID.UUID.String() {
		t.Errorf("ID = %q, want %q", tag.ID.UUID.String(), tagID.UUID.String())
	}
	// Should preserve original name from DB
	if tag.Name != "OriginalName" {
		t.Errorf("Name = %q, want %q", tag.Name, "OriginalName")
	}
}

// TestEnsureTag_BlankTags tests that blank tags are ignored.
func TestEnsureTag_BlankTags(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	tag, err := svc.EnsureTag(ctx, "")
	if err != nil {
		t.Fatalf("EnsureTag failed: %v", err)
	}
	if tag != nil {
		t.Errorf("EnsureTag(%q) returned tag, want nil", "")
	}

	tag, err = svc.EnsureTag(ctx, "   ")
	if err != nil {
		t.Fatalf("EnsureTag failed: %v", err)
	}
	if tag != nil {
		t.Errorf("EnsureTag(%q) returned tag, want nil", "   ")
	}
}

// TestEnsureImageTag_CreateNew tests creating a new image-tag association.
func TestEnsureImageTag_CreateNew(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	imageID := dbtypes.MustNewUUIDV7()

	// Create tag first
	_, err := svc.EnsureTag(ctx, "test-tag")
	if err != nil {
		t.Fatalf("EnsureTag failed: %v", err)
	}

	// Get the tag ID
	tag, err := svc.EnsureTag(ctx, "test-tag")
	if err != nil {
		t.Fatalf("EnsureTag failed: %v", err)
	}

	imgTag, err := svc.EnsureImageTag(ctx, imageID, tag.ID)
	if err != nil {
		t.Fatalf("EnsureImageTag failed: %v", err)
	}
	if imgTag == nil {
		t.Fatal("expected image tag, got nil")
	}
	if imgTag.ImageID.UUID.String() != imageID.UUID.String() {
		t.Errorf("ImageID = %q, want %q", imgTag.ImageID.UUID.String(), imageID.UUID.String())
	}
	if imgTag.TagID.UUID.String() != tag.ID.UUID.String() {
		t.Errorf("TagID = %q, want %q", imgTag.TagID.UUID.String(), tag.ID.UUID.String())
	}
}

// TestEnsureImageTag_Idempotent tests that EnsureImageTag is idempotent.
func TestEnsureImageTag_Idempotent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	svc := NewService(db)
	ctx := context.Background()

	imageID := dbtypes.MustNewUUIDV7()

	// Create tag first
	tag, err := svc.EnsureTag(ctx, "idempotent-tag")
	if err != nil {
		t.Fatalf("EnsureTag failed: %v", err)
	}

	// Create association twice
	imgTag1, err := svc.EnsureImageTag(ctx, imageID, tag.ID)
	if err != nil {
		t.Fatalf("EnsureImageTag failed: %v", err)
	}

	imgTag2, err := svc.EnsureImageTag(ctx, imageID, tag.ID)
	if err != nil {
		t.Fatalf("EnsureImageTag failed: %v", err)
	}

	// Should be the same association
	if imgTag1.ID.UUID.String() != imgTag2.ID.UUID.String() {
		t.Errorf("IDs differ: %q vs %q", imgTag1.ID.UUID.String(), imgTag2.ID.UUID.String())
	}
}

// TestSyncImageTags_Basic tests basic tag syncing.
func TestSyncImageTags_Basic(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	svc := NewService(db)
	ctx := context.Background()
	logger := slog.Default()

	imageID := dbtypes.MustNewUUIDV7()

	err := svc.SyncImageTags(ctx, imageID, []string{"tag1", "tag2", "tag3"}, logger)
	if err != nil {
		t.Fatalf("SyncImageTags failed: %v", err)
	}

	// Verify tags were created
	tag1, err := svc.getTagByNormalizedName(ctx, "tag1")
	if err != nil {
		t.Errorf("tag1 should exist: %v", err)
	}
	if tag1 == nil {
		t.Error("tag1 should not be nil")
	}

	// Verify image-tag associations were created
	imgTag, err := svc.getImageTag(ctx, imageID, tag1.ID)
	if err != nil {
		t.Errorf("image tag should exist: %v", err)
	}
	if imgTag == nil {
		t.Error("image tag should not be nil")
	}
}

// TestSyncImageTags_Dedupe tests that duplicate tags are handled.
func TestSyncImageTags_Dedupe(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	svc := NewService(db)
	ctx := context.Background()
	logger := slog.Default()

	imageID := dbtypes.MustNewUUIDV7()

	// Same tag with different cases
	err := svc.SyncImageTags(ctx, imageID, []string{"Tag1", "TAG1", "tag1"}, logger)
	if err != nil {
		t.Fatalf("SyncImageTags failed: %v", err)
	}

	// Should only have one tag
	tag, err := svc.getTagByNormalizedName(ctx, "tag1")
	if err != nil {
		t.Fatalf("tag should exist: %v", err)
	}

	// Should only have one image-tag association
	imgTag, err := svc.getImageTag(ctx, imageID, tag.ID)
	if err != nil {
		t.Fatalf("image tag should exist: %v", err)
	}
	if imgTag == nil {
		t.Error("image tag should not be nil")
	}
}

// TestSyncImageTags_Empty tests empty tag list.
func TestSyncImageTags_Empty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	svc := NewService(db)
	ctx := context.Background()
	logger := slog.Default()

	imageID := dbtypes.MustNewUUIDV7()

	err := svc.SyncImageTags(ctx, imageID, []string{}, logger)
	if err != nil {
		t.Fatalf("SyncImageTags failed: %v", err)
	}

	err = svc.SyncImageTags(ctx, imageID, nil, logger)
	if err != nil {
		t.Fatalf("SyncImageTags failed: %v", err)
	}
}

// TestSyncImageTags_BlankFiltered tests that blank tags are filtered.
func TestSyncImageTags_BlankFiltered(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	createTables(t, db)

	svc := NewService(db)
	ctx := context.Background()
	logger := slog.Default()

	imageID := dbtypes.MustNewUUIDV7()

	err := svc.SyncImageTags(ctx, imageID, []string{"valid", "", "   ", "another"}, logger)
	if err != nil {
		t.Fatalf("SyncImageTags failed: %v", err)
	}

	// Should have 2 tags: "valid" and "another"
	tag, err := svc.getTagByNormalizedName(ctx, "valid")
	if err != nil {
		t.Errorf("valid tag should exist: %v", err)
	}
	if tag == nil {
		t.Error("valid tag should not be nil")
	}

	tag, err = svc.getTagByNormalizedName(ctx, "another")
	if err != nil {
		t.Errorf("another tag should exist: %v", err)
	}
	if tag == nil {
		t.Error("another tag should not be nil")
	}

	// "normalized" empty tag should not exist
	_, err = svc.getTagByNormalizedName(ctx, "")
	if err != ErrTagNotFound {
		t.Errorf("empty tag should not exist, got: %v", err)
	}
}
