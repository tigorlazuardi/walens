package device_subscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/walens/walens/internal/dbtypes"
)

// ErrDBUnavailable is returned when the database is not available.
var ErrDBUnavailable = errors.New("database unavailable")

// ErrSubscriptionNotFound is returned when the requested subscription does not exist.
var ErrSubscriptionNotFound = errors.New("device subscription not found")

// ErrDeviceNotFound is returned when the referenced device does not exist.
var ErrDeviceNotFound = errors.New("device not found")

// ErrSourceNotFound is returned when the referenced source does not exist.
var ErrSourceNotFound = errors.New("source not found")

// ErrDuplicateSubscription is returned when a device is already subscribed to the source.
var ErrDuplicateSubscription = errors.New("device is already subscribed to this source")

// SubscriptionRow represents a device_source_subscriptions row in the database.
type SubscriptionRow struct {
	ID        dbtypes.UUID          `json:"id" doc:"Unique subscription identifier (UUIDv7)."`
	DeviceID  dbtypes.UUID          `json:"device_id" doc:"Reference to the subscribed device."`
	SourceID  dbtypes.UUID          `json:"source_id" doc:"Reference to the subscribed source."`
	IsEnabled dbtypes.BoolInt       `json:"is_enabled" doc:"Whether this subscription is active."`
	CreatedAt dbtypes.UnixMilliTime `json:"created_at" doc:"Subscription creation timestamp."`
	UpdatedAt dbtypes.UnixMilliTime `json:"updated_at" doc:"Last modification timestamp."`
}

// CreateSubscriptionInput contains the fields needed to create a new device subscription.
type CreateSubscriptionInput struct {
	DeviceID  string `json:"device_id" doc:"Reference to the device to subscribe."`
	SourceID  string `json:"source_id" doc:"Reference to the source to subscribe to."`
	IsEnabled bool   `json:"is_enabled" doc:"Whether this subscription is active."`
}

// UpdateSubscriptionInput contains the fields needed to update an existing device subscription.
// All fields are required for full-object update semantics.
type UpdateSubscriptionInput struct {
	ID        string `json:"id" doc:"Unique subscription identifier."`
	DeviceID  string `json:"device_id" doc:"Reference to the device."`
	SourceID  string `json:"source_id" doc:"Reference to the source."`
	IsEnabled bool   `json:"is_enabled" doc:"Whether this subscription is active."`
}

// Service provides CRUD operations for device source subscriptions.
type Service struct {
	db *sql.DB
}

// NewService creates a new device_subscriptions service.
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// ListSubscriptions returns all device source subscriptions.
func (s *Service) ListSubscriptions(ctx context.Context) ([]SubscriptionRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, device_id, source_id, is_enabled, created_at, updated_at
		FROM device_source_subscriptions
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query device_source_subscriptions: %w", err)
	}
	defer rows.Close()

	results := make([]SubscriptionRow, 0)
	for rows.Next() {
		var sub SubscriptionRow
		if err := rows.Scan(
			&sub.ID, &sub.DeviceID, &sub.SourceID,
			&sub.IsEnabled, &sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan subscription row: %w", err)
		}
		results = append(results, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// GetSubscription returns a single device source subscription by ID.
func (s *Service) GetSubscription(ctx context.Context, id dbtypes.UUID) (*SubscriptionRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	query := `
		SELECT id, device_id, source_id, is_enabled, created_at, updated_at
		FROM device_source_subscriptions
		WHERE id = ?
	`

	var sub SubscriptionRow
	err := s.db.QueryRowContext(ctx, query, id.UUID.String()).Scan(
		&sub.ID, &sub.DeviceID, &sub.SourceID,
		&sub.IsEnabled, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query subscription: %w", err)
	}

	return &sub, nil
}

// deviceExists checks if a device with the given ID exists.
func (s *Service) deviceExists(ctx context.Context, deviceID dbtypes.UUID) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM devices WHERE id = ?`, deviceID.UUID.String()).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check device exists: %w", err)
	}
	return true, nil
}

// sourceExists checks if a source with the given ID exists.
func (s *Service) sourceExists(ctx context.Context, sourceID dbtypes.UUID) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM sources WHERE id = ?`, sourceID.UUID.String()).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check source exists: %w", err)
	}
	return true, nil
}

// CreateSubscription creates a new device source subscription.
func (s *Service) CreateSubscription(ctx context.Context, input *CreateSubscriptionInput) (*SubscriptionRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	// Parse and validate device ID
	deviceID, err := dbtypes.NewUUIDFromString(input.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device ID format: %w", err)
	}

	// Parse and validate source ID
	sourceID, err := dbtypes.NewUUIDFromString(input.SourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid source ID format: %w", err)
	}

	// Check device exists
	exists, err := s.deviceExists(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrDeviceNotFound
	}

	// Check source exists
	exists, err = s.sourceExists(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSourceNotFound
	}

	// Check for duplicate subscription
	var existingID string
	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM device_source_subscriptions WHERE device_id = ? AND source_id = ?`,
		deviceID.UUID.String(), sourceID.UUID.String(),
	).Scan(&existingID)
	if err == nil {
		return nil, ErrDuplicateSubscription
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("check duplicate subscription: %w", err)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, fmt.Errorf("generate UUIDv7: %w", err)
	}

	isEnabled := dbtypes.BoolInt(input.IsEnabled)

	query := `
		INSERT INTO device_source_subscriptions (id, device_id, source_id, is_enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		id.UUID.String(), deviceID.UUID.String(), sourceID.UUID.String(),
		isEnabled, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert subscription: %w", err)
	}

	return &SubscriptionRow{
		ID:        id,
		DeviceID:  deviceID,
		SourceID:  sourceID,
		IsEnabled: isEnabled,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateSubscription updates an existing device source subscription with full-object update semantics.
func (s *Service) UpdateSubscription(ctx context.Context, input *UpdateSubscriptionInput) (*SubscriptionRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}

	// Parse IDs
	id, err := dbtypes.NewUUIDFromString(input.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription ID format: %w", err)
	}

	deviceID, err := dbtypes.NewUUIDFromString(input.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device ID format: %w", err)
	}

	sourceID, err := dbtypes.NewUUIDFromString(input.SourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid source ID format: %w", err)
	}

	// Check subscription exists
	existing, err := s.GetSubscription(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check device exists
	exists, err := s.deviceExists(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrDeviceNotFound
	}

	// Check source exists
	exists, err = s.sourceExists(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSourceNotFound
	}

	// Check for duplicate subscription (excluding current subscription)
	var otherID string
	err = s.db.QueryRowContext(ctx,
		`SELECT id FROM device_source_subscriptions WHERE device_id = ? AND source_id = ? AND id != ?`,
		deviceID.UUID.String(), sourceID.UUID.String(), id.UUID.String(),
	).Scan(&otherID)
	if err == nil {
		return nil, ErrDuplicateSubscription
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("check duplicate subscription: %w", err)
	}

	now := dbtypes.NewUnixMilliTimeNow()
	isEnabled := dbtypes.BoolInt(input.IsEnabled)

	query := `
		UPDATE device_source_subscriptions
		SET device_id = ?, source_id = ?, is_enabled = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.ExecContext(ctx, query,
		deviceID.UUID.String(), sourceID.UUID.String(), isEnabled, now, id.UUID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("update subscription: %w", err)
	}

	return &SubscriptionRow{
		ID:        id,
		DeviceID:  deviceID,
		SourceID:  sourceID,
		IsEnabled: isEnabled,
		CreatedAt: existing.CreatedAt,
		UpdatedAt: now,
	}, nil
}

// DeleteSubscription deletes a device source subscription by ID.
func (s *Service) DeleteSubscription(ctx context.Context, id dbtypes.UUID) error {
	if s.db == nil {
		return ErrDBUnavailable
	}

	// Check subscription exists
	_, err := s.GetSubscription(ctx, id)
	if err != nil {
		return err
	}

	query := `DELETE FROM device_source_subscriptions WHERE id = ?`
	_, err = s.db.ExecContext(ctx, query, id.UUID.String())
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}

	return nil
}
