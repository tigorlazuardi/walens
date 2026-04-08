package device_subscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

var ErrDBUnavailable = errors.New("database unavailable")
var ErrSubscriptionNotFound = errors.New("device subscription not found")
var ErrDeviceNotFound = errors.New("device not found")
var ErrSourceNotFound = errors.New("source not found")
var ErrDuplicateSubscription = errors.New("device is already subscribed to this source")

type SubscriptionRow = model.DeviceSourceSubscriptions

type CreateSubscriptionInput struct {
	DeviceID  string `json:"device_id" doc:"Reference to the device to subscribe."`
	SourceID  string `json:"source_id" doc:"Reference to the source to subscribe to."`
	IsEnabled bool   `json:"is_enabled" doc:"Whether this subscription is active."`
}

type UpdateSubscriptionInput struct {
	ID        string `json:"id" doc:"Unique subscription identifier."`
	DeviceID  string `json:"device_id" doc:"Reference to the device."`
	SourceID  string `json:"source_id" doc:"Reference to the source."`
	IsEnabled bool   `json:"is_enabled" doc:"Whether this subscription is active."`
}

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) countDevices(ctx context.Context, id dbtypes.UUID) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(Devices.ID).AS("count")).FROM(Devices).WHERE(Devices.ID.EQ(String(id.UUID.String())))
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("check device exists: %w", err)
	}
	return count.Count, nil
}

func (s *Service) countSources(ctx context.Context, id dbtypes.UUID) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(Sources.ID).AS("count")).FROM(Sources).WHERE(Sources.ID.EQ(String(id.UUID.String())))
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("check source exists: %w", err)
	}
	return count.Count, nil
}

func (s *Service) countSubscriptions(ctx context.Context, condition BoolExpression) (int64, error) {
	var count struct {
		Count int64 `alias:"count"`
	}
	stmt := SELECT(COUNT(DeviceSourceSubscriptions.ID).AS("count")).FROM(DeviceSourceSubscriptions).WHERE(condition)
	if err := stmt.QueryContext(ctx, s.db, &count); err != nil {
		return 0, fmt.Errorf("count subscriptions: %w", err)
	}
	return count.Count, nil
}

func (s *Service) ListSubscriptions(ctx context.Context) ([]SubscriptionRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
	var items []model.DeviceSourceSubscriptions
	stmt := SELECT(DeviceSourceSubscriptions.AllColumns).FROM(DeviceSourceSubscriptions).ORDER_BY(DeviceSourceSubscriptions.CreatedAt.ASC())
	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return []SubscriptionRow{}, nil
		}
		return nil, fmt.Errorf("query device_source_subscriptions: %w", err)
	}
	return items, nil
}

func (s *Service) GetSubscription(ctx context.Context, id dbtypes.UUID) (*SubscriptionRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
	var sub model.DeviceSourceSubscriptions
	stmt := SELECT(DeviceSourceSubscriptions.AllColumns).
		FROM(DeviceSourceSubscriptions).
		WHERE(DeviceSourceSubscriptions.ID.EQ(String(id.UUID.String()))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &sub); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("query subscription: %w", err)
	}
	return &sub, nil
}

func (s *Service) CreateSubscription(ctx context.Context, input *CreateSubscriptionInput) (*SubscriptionRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
	deviceID, err := dbtypes.NewUUIDFromString(input.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid device ID format: %w", err)
	}
	sourceID, err := dbtypes.NewUUIDFromString(input.SourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid source ID format: %w", err)
	}
	deviceCount, err := s.countDevices(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if deviceCount == 0 {
		return nil, ErrDeviceNotFound
	}
	sourceCount, err := s.countSources(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if sourceCount == 0 {
		return nil, ErrSourceNotFound
	}
	duplicateCount, err := s.countSubscriptions(ctx,
		DeviceSourceSubscriptions.DeviceID.EQ(String(deviceID.UUID.String())).
			AND(DeviceSourceSubscriptions.SourceID.EQ(String(sourceID.UUID.String()))),
	)
	if err != nil {
		return nil, fmt.Errorf("check duplicate subscription: %w", err)
	}
	if duplicateCount > 0 {
		return nil, ErrDuplicateSubscription
	}
	now := dbtypes.NewUnixMilliTimeNow()
	id, err := dbtypes.NewUUIDV7()
	if err != nil {
		return nil, fmt.Errorf("generate UUIDv7: %w", err)
	}
	row := model.DeviceSourceSubscriptions{
		ID:        &id,
		DeviceID:  deviceID,
		SourceID:  sourceID,
		IsEnabled: dbtypes.BoolInt(input.IsEnabled),
		CreatedAt: now,
		UpdatedAt: now,
	}
	stmt := DeviceSourceSubscriptions.INSERT(
		DeviceSourceSubscriptions.ID,
		DeviceSourceSubscriptions.DeviceID,
		DeviceSourceSubscriptions.SourceID,
		DeviceSourceSubscriptions.IsEnabled,
		DeviceSourceSubscriptions.CreatedAt,
		DeviceSourceSubscriptions.UpdatedAt,
	).MODEL(row)
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("insert subscription: %w", err)
	}
	return s.GetSubscription(ctx, id)
}

func (s *Service) UpdateSubscription(ctx context.Context, input *UpdateSubscriptionInput) (*SubscriptionRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
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
	existing, err := s.GetSubscription(ctx, id)
	if err != nil {
		return nil, err
	}
	deviceCount, err := s.countDevices(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if deviceCount == 0 {
		return nil, ErrDeviceNotFound
	}
	sourceCount, err := s.countSources(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if sourceCount == 0 {
		return nil, ErrSourceNotFound
	}
	duplicateCount, err := s.countSubscriptions(ctx,
		DeviceSourceSubscriptions.DeviceID.EQ(String(deviceID.UUID.String())).
			AND(DeviceSourceSubscriptions.SourceID.EQ(String(sourceID.UUID.String()))).
			AND(DeviceSourceSubscriptions.ID.NOT_EQ(String(id.UUID.String()))),
	)
	if err != nil {
		return nil, fmt.Errorf("check duplicate subscription: %w", err)
	}
	if duplicateCount > 0 {
		return nil, ErrDuplicateSubscription
	}
	updated := *existing
	updated.DeviceID = deviceID
	updated.SourceID = sourceID
	updated.IsEnabled = dbtypes.BoolInt(input.IsEnabled)
	updated.UpdatedAt = dbtypes.NewUnixMilliTimeNow()
	stmt := DeviceSourceSubscriptions.UPDATE(
		DeviceSourceSubscriptions.DeviceID,
		DeviceSourceSubscriptions.SourceID,
		DeviceSourceSubscriptions.IsEnabled,
		DeviceSourceSubscriptions.UpdatedAt,
	).MODEL(updated).WHERE(DeviceSourceSubscriptions.ID.EQ(String(id.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return nil, fmt.Errorf("update subscription: %w", err)
	}
	return s.GetSubscription(ctx, id)
}

func (s *Service) DeleteSubscription(ctx context.Context, id dbtypes.UUID) error {
	if s.db == nil {
		return ErrDBUnavailable
	}
	if _, err := s.GetSubscription(ctx, id); err != nil {
		return err
	}
	stmt := DeviceSourceSubscriptions.DELETE().WHERE(DeviceSourceSubscriptions.ID.EQ(String(id.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	return nil
}
