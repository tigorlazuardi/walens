package images

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/walens/walens/internal/dbtypes"
)

// TestGetImage_NotFound tests that GetImage returns ErrImageNotFound when the image doesn't exist.
func TestGetImage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	nonExistentID := dbtypes.MustNewUUIDV7()

	_, err := svc.GetImage(ctx, GetImageInput{ID: nonExistentID})
	if err != ErrImageNotFound {
		t.Errorf("expected ErrImageNotFound, got %v", err)
	}
}

// TestGetImage_Found tests that GetImage returns the image when it exists.
func TestGetImage_Found(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create source and image
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "unique-img-1", 1920, 1080, 100000, false)

	resp, err := svc.GetImage(ctx, GetImageInput{ID: imageID})
	if err != nil {
		t.Fatalf("GetImage failed: %v", err)
	}

	if resp.ID.UUID.String() != imageID.UUID.String() {
		t.Errorf("expected image ID %s, got %s", imageID.UUID.String(), resp.ID.UUID.String())
	}
}

// TestSetImageFavorite_UpdatesIsFavorite tests that SetImageFavorite updates the is_favorite flag.
func TestSetImageFavorite_UpdatesIsFavorite(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create source and image
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "unique-img-fav", 1920, 1080, 100000, false)

	// Set favorite to true
	resp, err := svc.SetImageFavorite(ctx, SetImageFavoriteInput{
		ID:         imageID,
		IsFavorite: true,
	})
	if err != nil {
		t.Fatalf("SetImageFavorite failed: %v", err)
	}

	if !bool(resp.Image.IsFavorite) {
		t.Errorf("expected IsFavorite to be true")
	}

	// Set favorite back to false
	resp, err = svc.SetImageFavorite(ctx, SetImageFavoriteInput{
		ID:         imageID,
		IsFavorite: false,
	})
	if err != nil {
		t.Fatalf("SetImageFavorite failed: %v", err)
	}

	if bool(resp.Image.IsFavorite) {
		t.Errorf("expected IsFavorite to be false")
	}
}

// TestSetImageFavorite_NotFound tests that SetImageFavorite returns ErrImageNotFound when the image doesn't exist.
func TestSetImageFavorite_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	nonExistentID := dbtypes.MustNewUUIDV7()

	_, err := svc.SetImageFavorite(ctx, SetImageFavoriteInput{
		ID:         nonExistentID,
		IsFavorite: true,
	})
	if err != ErrImageNotFound {
		t.Errorf("expected ErrImageNotFound, got %v", err)
	}
}

// TestBlacklistImage_CreatesBlacklistRow tests that BlacklistImage creates a blacklist entry.
func TestBlacklistImage_CreatesBlacklistRow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create source and image
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "unique-img-bl", 1920, 1080, 100000, false)

	reason := "test reason"
	resp, err := svc.BlacklistImage(ctx, BlacklistImageInput{
		ImageID: imageID,
		Reason:  &reason,
	})
	if err != nil {
		t.Fatalf("BlacklistImage failed: %v", err)
	}

	if resp.Blacklist == nil {
		t.Fatalf("expected blacklist entry, got nil")
	}
	if resp.Blacklist.SourceID.UUID.String() != sourceID.UUID.String() {
		t.Errorf("expected source ID %s, got %s", sourceID.UUID.String(), resp.Blacklist.SourceID.UUID.String())
	}
	if resp.Blacklist.UniqueIdentifier != "unique-img-bl" {
		t.Errorf("expected unique_identifier 'unique-img-bl', got '%s'", resp.Blacklist.UniqueIdentifier)
	}
	if resp.Blacklist.Reason == nil || *resp.Blacklist.Reason != reason {
		t.Errorf("expected reason '%s', got %v", reason, resp.Blacklist.Reason)
	}
}

// TestBlacklistImage_Idempotent tests that BlacklistImage is idempotent.
func TestBlacklistImage_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create source and image
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "unique-img-bl-idemp", 1920, 1080, 100000, false)

	reason := "first reason"

	// First call
	resp1, err := svc.BlacklistImage(ctx, BlacklistImageInput{
		ImageID: imageID,
		Reason:  &reason,
	})
	if err != nil {
		t.Fatalf("BlacklistImage first call failed: %v", err)
	}

	// Second call with different reason - should return the same entry
	reason2 := "different reason"
	resp2, err := svc.BlacklistImage(ctx, BlacklistImageInput{
		ImageID: imageID,
		Reason:  &reason2,
	})
	if err != nil {
		t.Fatalf("BlacklistImage second call failed: %v", err)
	}

	// Should be the same entry (idempotent)
	if resp1.Blacklist.ID.UUID.String() != resp2.Blacklist.ID.UUID.String() {
		t.Errorf("expected same blacklist ID, got %s and %s", resp1.Blacklist.ID.UUID.String(), resp2.Blacklist.ID.UUID.String())
	}

	// Reason should still be the original
	if resp2.Blacklist.Reason == nil || *resp2.Blacklist.Reason != reason {
		t.Errorf("expected reason '%s', got %v", reason, resp2.Blacklist.Reason)
	}
}

// TestBlacklistImage_NotFound tests that BlacklistImage returns ErrImageNotFound when the image doesn't exist.
func TestBlacklistImage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	nonExistentID := dbtypes.MustNewUUIDV7()

	_, err := svc.BlacklistImage(ctx, BlacklistImageInput{
		ImageID: nonExistentID,
	})
	if err != ErrImageNotFound {
		t.Errorf("expected ErrImageNotFound, got %v", err)
	}
}

// TestDeleteImage_HappyPath tests that DeleteImage removes image rows and file-backed location rows.
func TestDeleteImage_HappyPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a temp directory for image files
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "test-image.jpg")

	// Create a real file
	if err := os.WriteFile(imagePath, []byte("test image content"), 0644); err != nil {
		t.Fatalf("failed to create test image file: %v", err)
	}

	svc := NewService(db)
	ctx := context.Background()

	// Create source, device, and image
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device", 1920, 1080, true, false)

	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "unique-img-del", 1920, 1080, 100000, false)

	// Create location pointing to the real file
	locationID := dbtypes.MustNewUUIDV7()
	insertImageLocation(t, db, locationID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String(), imagePath)

	// Create assignment
	assignmentID := dbtypes.MustNewUUIDV7()
	insertImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Delete the image
	resp, err := svc.DeleteImage(ctx, DeleteImageInput{ID: imageID})
	if err != nil {
		t.Fatalf("DeleteImage failed: %v", err)
	}

	if !resp.DeletedImage {
		t.Errorf("expected DeletedImage to be true")
	}
	if resp.DeletedLocationCount != 1 {
		t.Errorf("expected DeletedLocationCount to be 1, got %d", resp.DeletedLocationCount)
	}
	if len(resp.FailedPaths) != 0 {
		t.Errorf("expected no failed paths, got %v", resp.FailedPaths)
	}

	// Verify file was deleted
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Errorf("expected image file to be deleted")
	}

	// Verify image is gone from DB
	_, err = svc.GetImage(ctx, GetImageInput{ID: imageID})
	if err != ErrImageNotFound {
		t.Errorf("expected ErrImageNotFound, got %v", err)
	}
}

// TestDeleteImage_ConservesStateOnFailure tests that DeleteImage keeps DB rows if file deletion fails.
func TestDeleteImage_ConservesStateOnFailure(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create source, device, and image
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device-fail", 1920, 1080, true, false)

	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "unique-img-del-fail", 1920, 1080, 100000, false)

	// Create a temp directory with a file inside (making it non-empty, so os.Remove fails)
	tempDir := t.TempDir()
	if err := os.WriteFile(tempDir+"/somefile.txt", []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Use the temp directory path as the "image location" - os.Remove on a non-empty dir fails
	locationID := dbtypes.MustNewUUIDV7()
	insertImageLocation(t, db, locationID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String(), tempDir)

	// Delete the image - should return partial success because tempDir cannot be deleted via os.Remove
	resp, err := svc.DeleteImage(ctx, DeleteImageInput{ID: imageID})
	if err != nil {
		t.Fatalf("DeleteImage failed: %v", err)
	}

	// Should indicate failure in failed paths since tempDir is a non-empty directory
	if len(resp.FailedPaths) != 1 {
		t.Errorf("expected 1 failed path, got %d", len(resp.FailedPaths))
	}

	// Image should NOT be deleted from DB since file deletion failed
	_, err = svc.GetImage(ctx, GetImageInput{ID: imageID})
	if err != nil {
		t.Errorf("expected image to still exist in DB after partial failure, but got error: %v", err)
	}
}

// TestDeleteImage_NotFound tests that DeleteImage returns ErrImageNotFound when the image doesn't exist.
func TestDeleteImage_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	nonExistentID := dbtypes.MustNewUUIDV7()

	_, err := svc.DeleteImage(ctx, DeleteImageInput{ID: nonExistentID})
	if err != ErrImageNotFound {
		t.Errorf("expected ErrImageNotFound, got %v", err)
	}
}

// TestDeleteImage_WithThumbnail tests that DeleteImage also handles thumbnail deletion.
func TestDeleteImage_WithThumbnail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a temp directory for files
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "test-image.jpg")
	thumbPath := filepath.Join(tempDir, "test-thumb.jpg")

	// Create real files
	if err := os.WriteFile(imagePath, []byte("test image content"), 0644); err != nil {
		t.Fatalf("failed to create test image file: %v", err)
	}
	if err := os.WriteFile(thumbPath, []byte("test thumb content"), 0644); err != nil {
		t.Fatalf("failed to create test thumb file: %v", err)
	}

	svc := NewService(db)
	ctx := context.Background()

	// Create source, device, and image
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Test Device", "test-device", 1920, 1080, true, false)

	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "unique-img-thumb", 1920, 1080, 100000, false)

	// Create location
	locationID := dbtypes.MustNewUUIDV7()
	insertImageLocation(t, db, locationID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String(), imagePath)

	// Create thumbnail
	thumbID := dbtypes.MustNewUUIDV7()
	fileSize := int64(40960)
	insertTestThumbnail(t, db, thumbID.UUID.String(), imageID.UUID.String(), thumbPath, 512, 384, &fileSize)

	// Delete the image
	resp, err := svc.DeleteImage(ctx, DeleteImageInput{ID: imageID})
	if err != nil {
		t.Fatalf("DeleteImage failed: %v", err)
	}

	if !resp.DeletedImage {
		t.Errorf("expected DeletedImage to be true")
	}
	if !resp.DeletedThumbnail {
		t.Errorf("expected DeletedThumbnail to be true")
	}

	// Verify files were deleted
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Errorf("expected image file to be deleted")
	}
	if _, err := os.Stat(thumbPath); !os.IsNotExist(err) {
		t.Errorf("expected thumbnail file to be deleted")
	}
}
