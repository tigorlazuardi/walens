package images

import (
	"context"
	"fmt"

	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// CreateImageLocationRequest defines the input for creating a new image location record.
type CreateImageLocationRequest struct {
	ImageID     dbtypes.UUID `json:"image_id" doc:"Reference to image"`
	DeviceID    dbtypes.UUID `json:"device_id" doc:"Reference to device this path is for"`
	Path        string       `json:"path" doc:"Absolute filesystem path to the image file"`
	StorageKind string       `json:"storage_kind" doc:"Storage type: canonical, hardlink, or copy"`
	IsPrimary   bool         `json:"is_primary" doc:"Whether this is the primary device location"`
	IsActive    bool         `json:"is_active" doc:"Whether this location is currently active"`
}

// CreateImageLocation inserts a new location record and returns the created location.
func (s *Service) CreateImageLocation(ctx context.Context, req CreateImageLocationRequest) (*model.ImageLocations, error) {
	now := dbtypes.NewUnixMilliTimeNow()
	id := dbtypes.MustNewUUIDV7()

	row := model.ImageLocations{
		ID:          id,
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
