package images

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/walens/walens/internal/db"
	"github.com/walens/walens/internal/dbtypes"
)

func setupTestDB(t *testing.T) *sql.DB {
	dbPath := t.TempDir() + "/test.db"
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.RunMigrations(database); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
	return database
}

func TestListImages_BasicPagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	req := ListImagesRequest{
		Pagination: &dbtypes.CursorPaginationRequest{},
	}

	resp, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages failed: %v", err)
	}

	// Should return empty list with no images
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
	// Pagination may be nil when there are no items
}

func TestListImages_AdultFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First create an image to test filtering
	// Note: This is a basic test that the service doesn't crash with filters
	svc := NewService(db)
	ctx := context.Background()

	adultTrue := true
	adultFalse := false

	// Test with adult=true filter
	req := ListImagesRequest{
		Adult:      &adultTrue,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with adult=true failed: %v", err)
	}

	// Test with adult=false filter
	req = ListImagesRequest{
		Adult:      &adultFalse,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err = svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with adult=false failed: %v", err)
	}
}

func TestListImages_FavoriteFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	favTrue := true
	favFalse := false

	// Test with favorite=true filter
	req := ListImagesRequest{
		Favorite:   &favTrue,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with favorite=true failed: %v", err)
	}

	// Test with favorite=false filter
	req = ListImagesRequest{
		Favorite:   &favFalse,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err = svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with favorite=false failed: %v", err)
	}
}

func TestListImages_SearchFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create source and images with different metadata for search testing
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	// Image with specific uploader
	img1ID := dbtypes.MustNewUUIDV7()
	insertImageWithMetadata(t, db, img1ID.UUID.String(), sourceID.UUID.String(), "img-001", "artist_alpha", "", "", "", 1920, 1080, 100000, false)

	// Image with specific artist
	img2ID := dbtypes.MustNewUUIDV7()
	insertImageWithMetadata(t, db, img2ID.UUID.String(), sourceID.UUID.String(), "img-002", "", "painter_bravo", "", "", 1920, 1080, 100000, false)

	// Image with specific origin URL
	img3ID := dbtypes.MustNewUUIDV7()
	insertImageWithMetadata(t, db, img3ID.UUID.String(), sourceID.UUID.String(), "img-003", "", "", "https://example.com/gallery", "", 1920, 1080, 100000, false)

	// Image with specific source item identifier
	img4ID := dbtypes.MustNewUUIDV7()
	insertImageWithMetadata(t, db, img4ID.UUID.String(), sourceID.UUID.String(), "img-004", "", "", "", "item_12345", 1920, 1080, 100000, false)

	// Image with tag
	img5ID := dbtypes.MustNewUUIDV7()
	insertImageWithMetadata(t, db, img5ID.UUID.String(), sourceID.UUID.String(), "img-005", "", "", "", "", 1920, 1080, 100000, false)
	tag1ID := dbtypes.MustNewUUIDV7()
	insertTag(t, db, tag1ID.UUID.String(), "TagOne", "tagone")
	imgTag1ID := dbtypes.MustNewUUIDV7()
	insertImageTag(t, db, imgTag1ID.UUID.String(), img5ID.UUID.String(), tag1ID.UUID.String())

	// Test search by uploader
	searchUploader := "artist_alpha"
	req := ListImagesRequest{
		Search:     &searchUploader,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with uploader search failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image matching uploader 'artist_alpha', got %d", len(resp.Items))
	}
	if len(resp.Items) > 0 && resp.Items[0].ID.String() != img1ID.UUID.String() {
		t.Errorf("expected image ID %s, got %s", img1ID.UUID.String(), resp.Items[0].ID.String())
	}

	// Test search by artist
	searchArtist := "painter_bravo"
	req = ListImagesRequest{
		Search:     &searchArtist,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with artist search failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image matching artist 'painter_bravo', got %d", len(resp.Items))
	}

	// Test search by origin URL
	searchOrigin := "example.com"
	req = ListImagesRequest{
		Search:     &searchOrigin,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with origin URL search failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image matching origin URL 'example.com', got %d", len(resp.Items))
	}

	// Test search by source item identifier
	searchSourceItem := "item_12345"
	req = ListImagesRequest{
		Search:     &searchSourceItem,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with source item identifier search failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image matching source item 'item_12345', got %d", len(resp.Items))
	}

	// Test search by tag name (normalized)
	searchTag := "TagOne"
	req = ListImagesRequest{
		Search:     &searchTag,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with tag search failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image matching tag 'TagOne', got %d", len(resp.Items))
	}

	// Test search that matches multiple fields
	searchMulti := "1920" // doesn't match any text field but will test partial
	req = ListImagesRequest{
		Search:     &searchMulti,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with multi search failed: %v", err)
	}
	// Should return 0 since no image has "1920" in uploader/artist/origin_url/source_item_identifier
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 images for search '1920' (no text match), got %d", len(resp.Items))
	}

	// Test empty/whitespace search is ignored
	searchWhitespace := "   "
	req = ListImagesRequest{
		Search:     &searchWhitespace,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with whitespace search failed: %v", err)
	}
	// Should return all 5 images since whitespace search is ignored
	if len(resp.Items) != 5 {
		t.Errorf("expected 5 images with whitespace search (ignored), got %d", len(resp.Items))
	}
}

func TestListImages_DimensionFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	minW := int64(100)
	maxW := int64(1920)
	minH := int64(100)
	maxH := int64(1080)

	req := ListImagesRequest{
		MinWidth:   &minW,
		MaxWidth:   &maxW,
		MinHeight:  &minH,
		MaxHeight:  &maxH,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with dimension filters failed: %v", err)
	}
}

func TestListImages_FileSizeFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	minF := int64(1000)
	maxF := int64(10000000)

	req := ListImagesRequest{
		MinFileSizeBytes: &minF,
		MaxFileSizeBytes: &maxF,
		Pagination:       &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with filesize filters failed: %v", err)
	}
}

func TestListImages_TextFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Test with combined search (replaces separate text filters)
	search := "artist example.com 12345"
	req := ListImagesRequest{
		Search:     &search,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with search failed: %v", err)
	}
}

func TestListDeviceImages_DeviceNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	deviceID := dbtypes.MustNewUUIDV7()

	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListDeviceImages(ctx, req)
	if err == nil {
		t.Error("expected error for non-existent device")
	}
}

func TestListDeviceImages_DisabledDevice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create a disabled device directly in the database for testing
	deviceID := dbtypes.MustNewUUIDV7()

	// Note: This is a basic test to ensure the service doesn't crash
	// A full integration test would require creating actual device and subscription records
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListDeviceImages(ctx, req)
	// Device not found should return 404
	if err == nil {
		t.Error("expected error for non-existent device")
	}
}

func TestListDeviceImages_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	deviceID := dbtypes.MustNewUUIDV7()
	adultTrue := true

	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Adult:      &adultTrue,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListDeviceImages(ctx, req)
	// Should get a 404 since device doesn't exist
	if err == nil {
		t.Error("expected error for non-existent device")
	}
}

// Test helper to insert a device
func insertDevice(t *testing.T, db *sql.DB, id, name, slug string, screenW, screenH int64, isEnabled, isAdultAllowed bool) {
	now := time.Now().UnixMilli()
	enabledVal := int64(0)
	if isEnabled {
		enabledVal = 1
	}
	adultVal := int64(0)
	if isAdultAllowed {
		adultVal = 1
	}
	_, err := db.Exec(`
		INSERT INTO devices (id, name, slug, screen_width, screen_height, is_enabled, is_adult_allowed, aspect_ratio_tolerance, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0.15, ?, ?)`,
		id, name, slug, screenW, screenH, enabledVal, adultVal, now, now)
	if err != nil {
		t.Fatalf("failed to insert device: %v", err)
	}
}

// Test helper to insert a source
func insertSource(t *testing.T, db *sql.DB, id, name, sourceType string, isEnabled bool) {
	now := time.Now().UnixMilli()
	enabledVal := int64(0)
	if isEnabled {
		enabledVal = 1
	}
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, '{}', 0, ?, ?, ?)`,
		id, name, sourceType, enabledVal, now, now)
	if err != nil {
		t.Fatalf("failed to insert source: %v", err)
	}
}

// Test helper to insert a device-source subscription
func insertDeviceSubscription(t *testing.T, db *sql.DB, id, deviceID, sourceID string, isEnabled bool) {
	now := time.Now().UnixMilli()
	enabledVal := int64(0)
	if isEnabled {
		enabledVal = 1
	}
	_, err := db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, deviceID, sourceID, enabledVal, now, now)
	if err != nil {
		t.Fatalf("failed to insert device subscription: %v", err)
	}
}

// Test helper to insert an image
func insertImage(t *testing.T, db *sql.DB, id, sourceID, uniqueID string, width, height, fileSize int64, isAdult bool) {
	now := time.Now().UnixMilli()
	adultVal := int64(0)
	if isAdult {
		adultVal = 1
	}
	aspectRatio := float64(width) / float64(height)
	_, err := db.Exec(`
		INSERT INTO images (id, source_id, unique_identifier, source_type, mime_type, width, height, aspect_ratio, file_size_bytes, is_adult, is_favorite, json_meta, created_at, updated_at)
		VALUES (?, ?, ?, 'test', 'image/jpeg', ?, ?, ?, ?, ?, 0, '{}', ?, ?)`,
		id, sourceID, uniqueID, width, height, aspectRatio, fileSize, adultVal, now, now)
	if err != nil {
		t.Fatalf("failed to insert image: %v", err)
	}
}

// Test helper to insert an image assignment
func insertImageAssignment(t *testing.T, db *sql.DB, id, imageID, deviceID string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO image_assignments (id, image_id, device_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, imageID, deviceID, now, now)
	if err != nil {
		t.Fatalf("failed to insert image assignment: %v", err)
	}
}

// Test helper to insert an image location
func insertImageLocation(t *testing.T, db *sql.DB, id, imageID, deviceID, path string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO image_locations (id, image_id, device_id, path, storage_kind, is_primary, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'local', 1, 1, ?, ?)`,
		id, imageID, deviceID, path, now, now)
	if err != nil {
		t.Fatalf("failed to insert image location: %v", err)
	}
}

// Test helper to insert a tag
func insertTag(t *testing.T, db *sql.DB, id, name, normalizedName string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO tags (id, name, normalized_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		id, name, normalizedName, now, now)
	if err != nil {
		t.Fatalf("failed to insert tag: %v", err)
	}
}

// Test helper to insert an image tag association
func insertImageTag(t *testing.T, db *sql.DB, id, imageID, tagID string) {
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
		INSERT INTO image_tags (id, image_id, tag_id, created_at)
		VALUES (?, ?, ?, ?)`,
		id, imageID, tagID, now)
	if err != nil {
		t.Fatalf("failed to insert image tag: %v", err)
	}
}

// Test helper to insert an image with full metadata
func insertImageWithMetadata(t *testing.T, db *sql.DB, id, sourceID, uniqueID, uploader, artist, originURL, sourceItemID string, width, height, fileSize int64, isAdult bool) {
	now := time.Now().UnixMilli()
	adultVal := int64(0)
	if isAdult {
		adultVal = 1
	}
	aspectRatio := float64(width) / float64(height)
	_, err := db.Exec(`
		INSERT INTO images (id, source_id, unique_identifier, source_type, mime_type, uploader, artist, origin_url, source_item_identifier, width, height, aspect_ratio, file_size_bytes, is_adult, is_favorite, json_meta, created_at, updated_at)
		VALUES (?, ?, ?, 'test', 'image/jpeg', ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, '{}', ?, ?)`,
		id, sourceID, uniqueID, uploader, artist, originURL, sourceItemID, width, height, aspectRatio, fileSize, adultVal, now, now)
	if err != nil {
		t.Fatalf("failed to insert image with metadata: %v", err)
	}
}

// TestListDeviceImages_DisabledDeviceWithAssignment tests that a disabled device
// still returns images that were previously assigned to it.
func TestListDeviceImages_DisabledDeviceWithAssignment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create a disabled device
	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Disabled Device", "disabled-device", 1920, 1080, false, false)

	// Create a source and subscription (even though device is disabled)
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	subscriptionID := dbtypes.MustNewUUIDV7()
	insertDeviceSubscription(t, db, subscriptionID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	// Create an image and assign it to the disabled device
	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-001", 1920, 1080, 100000, false)

	assignmentID := dbtypes.MustNewUUIDV7()
	insertImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Query images for the disabled device - should still return the assigned image
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}

	resp, err := svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image (assigned to disabled device), got %d", len(resp.Items))
	}

	if len(resp.Items) > 0 && resp.Items[0].ID.String() != imageID.UUID.String() {
		t.Errorf("expected image ID %s, got %s", imageID.UUID.String(), resp.Items[0].ID.String())
	}
}

// TestListDeviceImages_DisabledSubscriptionWithAssignment tests that a disabled
// subscription still returns images that were previously assigned.
func TestListDeviceImages_DisabledSubscriptionWithAssignment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create an enabled device
	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Enabled Device", "enabled-device", 1920, 1080, true, false)

	// Create a source
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	// Create a DISABLED subscription
	subscriptionID := dbtypes.MustNewUUIDV7()
	insertDeviceSubscription(t, db, subscriptionID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), false)

	// Create an image and assign it to the device (via subscription is disabled)
	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-002", 1920, 1080, 100000, false)

	assignmentID := dbtypes.MustNewUUIDV7()
	insertImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Query images - should still return the assigned image even though subscription is disabled
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}

	resp, err := svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image (assigned with disabled subscription), got %d", len(resp.Items))
	}
}

// TestListDeviceImages_DisabledSubscriptionWithLocation tests that a disabled
// subscription still returns images that have a location record.
func TestListDeviceImages_DisabledSubscriptionWithLocation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create an enabled device
	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Enabled Device", "enabled-device-loc", 1920, 1080, true, false)

	// Create a source
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	// Create a DISABLED subscription
	subscriptionID := dbtypes.MustNewUUIDV7()
	insertDeviceSubscription(t, db, subscriptionID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), false)

	// Create an image with a location record (no assignment)
	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-003", 1920, 1080, 100000, false)

	locationID := dbtypes.MustNewUUIDV7()
	insertImageLocation(t, db, locationID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String(), "/images/device-loc/img-003.jpg")

	// Query images - should return the image with location even though subscription is disabled
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}

	resp, err := svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image (with location, disabled subscription), got %d", len(resp.Items))
	}
}

// TestListDeviceImages_NonAssociatedImageNotReturnedWhenEligibilityBlocked tests
// that an image which only qualifies through current matching (not historical)
// is NOT returned when device/subscription eligibility is disabled.
func TestListDeviceImages_NonAssociatedImageNotReturnedWhenEligibilityBlocked(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create a DISABLED device
	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Disabled Device", "disabled-device-no-assoc", 1920, 1080, false, false)

	// Create a source and subscription
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	subscriptionID := dbtypes.MustNewUUIDV7()
	insertDeviceSubscription(t, db, subscriptionID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	// Create an image that would match the device constraints but has NO assignment/location
	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-004", 1920, 1080, 100000, false)

	// Query images for the disabled device - should NOT return the non-associated image
	// because current eligibility is blocked (device is disabled) and there's no historical record
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}

	resp, err := svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages failed: %v", err)
	}

	if len(resp.Items) != 0 {
		t.Errorf("expected 0 images (device disabled, no historical association), got %d", len(resp.Items))
	}
}

// TestListDeviceImages_NonAssociatedImageWithEnabledDevice tests that an image
// which matches device constraints is returned when the device is enabled.
func TestListDeviceImages_NonAssociatedImageWithEnabledDevice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create an ENABLED device
	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Enabled Device", "enabled-device-yes-assoc", 1920, 1080, true, false)

	// Create a source and ENABLED subscription
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	subscriptionID := dbtypes.MustNewUUIDV7()
	insertDeviceSubscription(t, db, subscriptionID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	// Create an image that matches the device constraints (no assignment/location)
	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-005", 1920, 1080, 100000, false)

	// Query images for the enabled device - should return the matching image
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}

	resp, err := svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image (matches enabled device constraints), got %d", len(resp.Items))
	}
}

// TestListDeviceImages_HistoricalAndCurrentBothExist tests that when an image
// has both historical association AND current eligibility, it's returned once.
func TestListDeviceImages_HistoricalAndCurrentBothExist(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create an enabled device
	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Enabled Device", "enabled-device-both", 1920, 1080, true, false)

	// Create a source and enabled subscription
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	subscriptionID := dbtypes.MustNewUUIDV7()
	insertDeviceSubscription(t, db, subscriptionID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	// Create an image that has assignment AND matches current criteria
	imageID := dbtypes.MustNewUUIDV7()
	insertImage(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-006", 1920, 1080, 100000, false)

	assignmentID := dbtypes.MustNewUUIDV7()
	insertImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Query images - should return exactly 1 (not duplicates)
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}

	resp, err := svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages failed: %v", err)
	}

	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image (deduplicated), got %d", len(resp.Items))
	}
}

// TestListDeviceImages_DisabledDeviceWithHistoricalAssignmentSearch tests that search
// still applies to historical images for a disabled device.
func TestListDeviceImages_DisabledDeviceWithHistoricalAssignmentSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create a disabled device
	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Disabled Device", "disabled-device-search", 1920, 1080, false, false)

	// Create a source and subscription
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	subscriptionID := dbtypes.MustNewUUIDV7()
	insertDeviceSubscription(t, db, subscriptionID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), true)

	// Create an image with assignment to the disabled device
	imageID := dbtypes.MustNewUUIDV7()
	insertImageWithMetadata(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-search-001", "searchable_artist", "", "", "", 1920, 1080, 100000, false)

	assignmentID := dbtypes.MustNewUUIDV7()
	insertImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// Without search, should return 1 image
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err := svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages without search failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image without search, got %d", len(resp.Items))
	}

	// With search matching uploader, should return 1 image
	searchArtist := "searchable_artist"
	req = ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Search:     &searchArtist,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages with search failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image matching search 'searchable_artist', got %d", len(resp.Items))
	}

	// With search NOT matching, should return 0 images
	searchNonMatch := "nonexistent"
	req = ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Search:     &searchNonMatch,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages with non-matching search failed: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 images for non-matching search, got %d", len(resp.Items))
	}
}

// TestListDeviceImages_DisabledSubscriptionWithHistoricalImageSearch tests that search
// applies to historical images when subscription is disabled.
func TestListDeviceImages_DisabledSubscriptionWithHistoricalImageSearch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Create an enabled device
	deviceID := dbtypes.MustNewUUIDV7()
	insertDevice(t, db, deviceID.UUID.String(), "Enabled Device", "enabled-device-search", 1920, 1080, true, false)

	// Create a source
	sourceID := dbtypes.MustNewUUIDV7()
	insertSource(t, db, sourceID.UUID.String(), "Test Source", "booru", true)

	// Create a DISABLED subscription
	subscriptionID := dbtypes.MustNewUUIDV7()
	insertDeviceSubscription(t, db, subscriptionID.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(), false)

	// Create an image with assignment
	imageID := dbtypes.MustNewUUIDV7()
	insertImageWithMetadata(t, db, imageID.UUID.String(), sourceID.UUID.String(), "img-search-002", "", "famous_painter", "", "", 1920, 1080, 100000, false)

	assignmentID := dbtypes.MustNewUUIDV7()
	insertImageAssignment(t, db, assignmentID.UUID.String(), imageID.UUID.String(), deviceID.UUID.String())

	// With search matching artist, should return 1 image
	searchArtist := "famous_painter"
	req := ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Search:     &searchArtist,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err := svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages with search failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 image matching search 'famous_painter', got %d", len(resp.Items))
	}

	// With search NOT matching, should return 0 images
	searchNonMatch := "unknown_artist"
	req = ListDeviceImagesRequest{
		DeviceID:   deviceID,
		Search:     &searchNonMatch,
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	resp, err = svc.ListDeviceImages(ctx, req)
	if err != nil {
		t.Fatalf("ListDeviceImages with non-matching search failed: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Errorf("expected 0 images for non-matching search, got %d", len(resp.Items))
	}
}
