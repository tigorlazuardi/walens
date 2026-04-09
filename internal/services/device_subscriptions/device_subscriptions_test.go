package device_subscriptions

import (
	"context"
	"database/sql"
	"testing"

	"github.com/walens/walens/internal/dbtypes"
	_ "modernc.org/sqlite"
)

func testDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=temp_store(MEMORY)")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return db, func() { db.Close() }
}

func createTables(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS devices (
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
		)
	`)
	if err != nil {
		t.Fatalf("create devices table: %v", err)
	}

	_, err = db.Exec(`
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

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS device_source_subscriptions (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			source_id TEXT NOT NULL,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
			FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("create device_source_subscriptions table: %v", err)
	}

	_, err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_device_source_subscriptions_device_source
		ON device_source_subscriptions(device_id, source_id)
	`)
	if err != nil {
		t.Fatalf("create unique index: %v", err)
	}
}

func insertTestDevice(t *testing.T, db *sql.DB, id, name, slug string) {
	_, err := db.Exec(`
		INSERT INTO devices (id, name, slug, screen_width, screen_height, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, 1920, 1080, 1, 1000, 1000)`,
		id, name, slug,
	)
	if err != nil {
		t.Fatalf("insert test device: %v", err)
	}
}

func insertTestSource(t *testing.T, db *sql.DB, id, name string) {
	_, err := db.Exec(`
		INSERT INTO sources (id, name, source_type, params, lookup_count, is_enabled, created_at, updated_at)
		VALUES (?, ?, 'booru', '{}', 0, 1, 1000, 1000)`,
		id, name,
	)
	if err != nil {
		t.Fatalf("insert test source: %v", err)
	}
}

// --- ListSubscriptions Tests ---

func TestServiceListSubscriptionsEmpty(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	svc := NewService(db)
	items, err := svc.ListSubscriptions(context.Background(), ListSubscriptionsRequest{})
	if err != nil {
		t.Fatalf("ListSubscriptions failed: %v", err)
	}
	if len(items.Items) != 0 {
		t.Errorf("expected 0 subscriptions, got %d", len(items.Items))
	}
}

func TestServiceListSubscriptionsWithData(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	_, err := db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000002",
		1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test subscription: %v", err)
	}

	svc := NewService(db)
	items, err := svc.ListSubscriptions(context.Background(), ListSubscriptionsRequest{})
	if err != nil {
		t.Fatalf("ListSubscriptions failed: %v", err)
	}
	if len(items.Items) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(items.Items))
	}
	if bool(items.Items[0].IsEnabled) != true {
		t.Errorf("expected is_enabled true, got %v", bool(items.Items[0].IsEnabled))
	}
}

// --- GetSubscription Tests ---

func TestServiceGetSubscription(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	_, err := db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000002",
		1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test subscription: %v", err)
	}

	svc := NewService(db)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	item, err := svc.GetSubscription(context.Background(), GetSubscriptionRequest{ID: id})
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}
	if bool(item.IsEnabled) != true {
		t.Errorf("expected is_enabled true, got %v", bool(item.IsEnabled))
	}
}

func TestServiceGetSubscriptionNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	svc := NewService(db)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	_, err := svc.GetSubscription(context.Background(), GetSubscriptionRequest{ID: id})
	if err == nil {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

// --- CreateSubscription Tests ---

func TestServiceCreateSubscription(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	svc := NewService(db)
	input := CreateSubscriptionRequest{
		DeviceID:  "01800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000002",
		IsEnabled: true,
	}

	item, err := svc.CreateSubscription(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}
	if bool(item.IsEnabled) != true {
		t.Errorf("expected is_enabled true, got %v", bool(item.IsEnabled))
	}
	if item.DeviceID.UUID.String() != "01800000-0000-0000-0000-000000000001" {
		t.Errorf("expected device_id match, got %v", item.DeviceID.UUID.String())
	}
	if item.SourceID.UUID.String() != "01800000-0000-0000-0000-000000000002" {
		t.Errorf("expected source_id match, got %v", item.SourceID.UUID.String())
	}
}

func TestServiceCreateSubscriptionDeviceNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	svc := NewService(db)
	input := CreateSubscriptionRequest{
		DeviceID:  "01800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000002",
		IsEnabled: true,
	}

	_, err := svc.CreateSubscription(context.Background(), input)
	if err == nil {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestServiceCreateSubscriptionSourceNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")

	svc := NewService(db)
	input := CreateSubscriptionRequest{
		DeviceID:  "01800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000002",
		IsEnabled: true,
	}

	_, err := svc.CreateSubscription(context.Background(), input)
	if err == nil {
		t.Errorf("expected ErrSourceNotFound, got %v", err)
	}
}

func TestServiceCreateSubscriptionDuplicate(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	svc := NewService(db)
	input := CreateSubscriptionRequest{
		DeviceID:  "01800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000002",
		IsEnabled: true,
	}

	_, err := svc.CreateSubscription(context.Background(), input)
	if err != nil {
		t.Fatalf("first CreateSubscription failed: %v", err)
	}

	// Try creating the same subscription again
	_, err = svc.CreateSubscription(context.Background(), input)
	if err == nil {
		t.Errorf("expected ErrDuplicateSubscription, got %v", err)
	}
}

func TestServiceCreateSubscriptionInvalidDeviceID(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	svc := NewService(db)
	input := CreateSubscriptionRequest{
		DeviceID:  "not-a-uuid",
		SourceID:  "01800000-0000-0000-0000-000000000002",
		IsEnabled: true,
	}

	_, err := svc.CreateSubscription(context.Background(), input)
	if err == nil {
		t.Error("expected error for invalid device ID, got nil")
	}
}

func TestServiceCreateSubscriptionInvalidSourceID(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	svc := NewService(db)
	input := CreateSubscriptionRequest{
		DeviceID:  "01800000-0000-0000-0000-000000000001",
		SourceID:  "not-a-uuid",
		IsEnabled: true,
	}

	_, err := svc.CreateSubscription(context.Background(), input)
	if err == nil {
		t.Error("expected error for invalid source ID, got nil")
	}
}

// --- UpdateSubscription Tests ---

func TestServiceUpdateSubscription(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	_, err := db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000002",
		1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test subscription: %v", err)
	}

	svc := NewService(db)
	input := UpdateSubscriptionRequest{
		ID:        "02800000-0000-0000-0000-000000000001",
		DeviceID:  "01800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000002",
		IsEnabled: false,
	}

	item, err := svc.UpdateSubscription(context.Background(), input)
	if err != nil {
		t.Fatalf("UpdateSubscription failed: %v", err)
	}
	if bool(item.IsEnabled) != false {
		t.Errorf("expected is_enabled false, got %v", bool(item.IsEnabled))
	}
}

func TestServiceUpdateSubscriptionNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	svc := NewService(db)
	input := UpdateSubscriptionRequest{
		ID:        "02800000-0000-0000-0000-000000000001",
		DeviceID:  "01800000-0000-0000-0000-000000000001",
		SourceID:  "01800000-0000-0000-0000-000000000002",
		IsEnabled: true,
	}

	_, err := svc.UpdateSubscription(context.Background(), input)
	if err == nil {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestServiceUpdateSubscriptionDeviceNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	_, err := db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000002",
		1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test subscription: %v", err)
	}

	svc := NewService(db)
	input := UpdateSubscriptionRequest{
		ID:        "02800000-0000-0000-0000-000000000001",
		DeviceID:  "01800000-0000-0000-0000-000000000099", // non-existent device
		SourceID:  "01800000-0000-0000-0000-000000000002",
		IsEnabled: true,
	}

	_, err = svc.UpdateSubscription(context.Background(), input)
	if err == nil {
		t.Errorf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestServiceUpdateSubscriptionDuplicateCheck(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Device 1", "device-1")
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000002", "Device 2", "device-2")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000003", "Test Source")

	// Create subscription for device 1 -> source 3
	_, err := db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000003",
		1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test subscription 1: %v", err)
	}

	// Create subscription for device 2 -> source 3
	_, err = db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000002",
		"01800000-0000-0000-0000-000000000002",
		"01800000-0000-0000-0000-000000000003",
		1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test subscription 2: %v", err)
	}

	svc := NewService(db)
	// Try to update subscription 2 to point to device 1 -> source 3 (which already exists as subscription 1)
	input := UpdateSubscriptionRequest{
		ID:        "02800000-0000-0000-0000-000000000002",
		DeviceID:  "01800000-0000-0000-0000-000000000001", // Change device to device 1
		SourceID:  "01800000-0000-0000-0000-000000000003",
		IsEnabled: true,
	}

	_, err = svc.UpdateSubscription(context.Background(), input)
	if err == nil {
		t.Errorf("expected ErrDuplicateSubscription, got %v", err)
	}
}

// --- DeleteSubscription Tests ---

func TestServiceDeleteSubscription(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)
	insertTestDevice(t, db, "01800000-0000-0000-0000-000000000001", "Test Device", "test-device")
	insertTestSource(t, db, "01800000-0000-0000-0000-000000000002", "Test Source")

	_, err := db.Exec(`
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"02800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000001",
		"01800000-0000-0000-0000-000000000002",
		1, 1000, 1000,
	)
	if err != nil {
		t.Fatalf("insert test subscription: %v", err)
	}

	svc := NewService(db)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	_, err = svc.DeleteSubscription(context.Background(), DeleteSubscriptionRequest{ID: id})
	if err != nil {
		t.Fatalf("DeleteSubscription failed: %v", err)
	}

	// Verify deleted
	_, err = svc.GetSubscription(context.Background(), GetSubscriptionRequest{ID: id})
	if err == nil {
		t.Errorf("expected ErrSubscriptionNotFound after delete, got %v", err)
	}
}

func TestServiceDeleteSubscriptionNotFound(t *testing.T) {
	db, cleanup := testDB(t)
	defer cleanup()
	createTables(t, db)

	svc := NewService(db)
	id, _ := dbtypes.NewUUIDFromString("02800000-0000-0000-0000-000000000001")
	_, err := svc.DeleteSubscription(context.Background(), DeleteSubscriptionRequest{ID: id})
	if err == nil {
		t.Errorf("expected ErrSubscriptionNotFound, got %v", err)
	}
}
