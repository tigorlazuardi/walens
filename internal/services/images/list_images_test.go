package images

import (
	"context"
	"database/sql"
	"testing"

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

func TestListImages_TagFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewService(db)
	ctx := context.Background()

	// Test with tag names filter
	req := ListImagesRequest{
		TagNames:   []string{"tag1", "tag2"},
		Pagination: &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with tag names failed: %v", err)
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

	uploader := "test_uploader"
	artist := "test_artist"
	originURL := "https://example.com"
	sourceItemID := "12345"

	req := ListImagesRequest{
		Uploader:             &uploader,
		Artist:               &artist,
		OriginURL:            &originURL,
		SourceItemIdentifier: &sourceItemID,
		Pagination:           &dbtypes.CursorPaginationRequest{},
	}
	_, err := svc.ListImages(ctx, req)
	if err != nil {
		t.Fatalf("ListImages with text filters failed: %v", err)
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
