package devices

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
)

func (s *Service) ListDevices(ctx context.Context) ([]DeviceRow, error) {
	if s.db == nil {
		return nil, ErrDBUnavailable
	}
	var items []model.Devices
	stmt := SELECT(Devices.AllColumns).FROM(Devices).ORDER_BY(Devices.Name.ASC())
	if err := stmt.QueryContext(ctx, s.db, &items); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return []DeviceRow{}, nil
		}
		return nil, fmt.Errorf("query devices: %w", err)
	}
	if items == nil {
		return []DeviceRow{}, nil
	}
	return items, nil
}
