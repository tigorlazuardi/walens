package devices

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

func (s *Service) GetDevice(ctx context.Context, id dbtypes.UUID) (*DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
	var dev model.Devices
	stmt := SELECT(Devices.AllColumns).FROM(Devices).WHERE(Devices.ID.EQ(String(id.UUID.String()))).LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &dev); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("query device: %w", err)
	}
	return &dev, nil
}

func (s *Service) GetDeviceBySlug(ctx context.Context, slug string) (*DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
	var dev model.Devices
	stmt := SELECT(Devices.AllColumns).FROM(Devices).WHERE(Devices.Slug.EQ(String(normalizeSlug(slug)))).LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &dev); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("query device by slug: %w", err)
	}
	return &dev, nil
}
