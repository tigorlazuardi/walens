package images

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// EnsureImageLocationRequest defines the input for ensuring an image location record.
type EnsureImageLocationRequest struct {
	ImageID     dbtypes.UUID `json:"image_id" doc:"Reference to image"`
	DeviceID    dbtypes.UUID `json:"device_id" doc:"Reference to device this path is for"`
	Path        string       `json:"path" doc:"Absolute filesystem path to the image file"`
	StorageKind string       `json:"storage_kind" doc:"Storage type: canonical, hardlink, or copy"`
	IsPrimary   bool         `json:"is_primary" doc:"Whether this is the primary device location"`
	IsActive    bool         `json:"is_active" doc:"Whether this location is currently active"`
}

// EnsureImageLocation creates or updates a location record for (image_id, device_id).
// If a location already exists for this image+device, it updates the existing row
// (keeping the same row id) instead of inserting a new one.
func (s *Service) EnsureImageLocation(ctx context.Context, req EnsureImageLocationRequest) (*model.ImageLocations, error) {
	// Check if location already exists for this image+device
	existing, err := s.GetDeviceImageLocation(ctx, req.ImageID, req.DeviceID)
	if err != nil && !errors.Is(err, ErrLocationNotFound) {
		return nil, fmt.Errorf("check existing location: %w", err)
	}

	now := dbtypes.NewUnixMilliTimeNow()

	if errors.Is(err, ErrLocationNotFound) {
		// No existing location - create new one
		id := dbtypes.MustNewUUIDV7()
		row := model.ImageLocations{
			ID:          &id,
			ImageID:     req.ImageID,
			DeviceID:    req.DeviceID,
			Path:        req.Path,
			StorageKind: req.StorageKind,
			IsPrimary:   dbtypes.BoolInt(req.IsPrimary),
			IsActive:    dbtypes.BoolInt(req.IsActive),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		stmt := ImageLocations.INSERT(
			ImageLocations.ID, ImageLocations.ImageID, ImageLocations.DeviceID,
			ImageLocations.Path, ImageLocations.StorageKind, ImageLocations.IsPrimary,
			ImageLocations.IsActive, ImageLocations.CreatedAt, ImageLocations.UpdatedAt,
		).MODEL(row)

		if _, err := stmt.ExecContext(ctx, s.db); err != nil {
			return nil, fmt.Errorf("create image location: %w", err)
		}

		return &row, nil
	}

	// Location exists - update the existing row (keep same ID)
	updated := existing
	updated.Path = req.Path
	updated.StorageKind = req.StorageKind
	updated.IsPrimary = dbtypes.BoolInt(req.IsPrimary)
	updated.IsActive = dbtypes.BoolInt(req.IsActive)
	updated.UpdatedAt = now

	stmt := ImageLocations.UPDATE(
		ImageLocations.Path,
		ImageLocations.StorageKind,
		ImageLocations.IsPrimary,
		ImageLocations.IsActive,
		ImageLocations.UpdatedAt,
	).MODEL(updated).WHERE(
		ImageLocations.ID.EQ(String(existing.ID.UUID.String())),
	)

	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, fmt.Errorf("update image location: %w", err)
	}

	return updated, nil
}
