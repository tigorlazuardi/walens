package images

import (
	"context"
	"fmt"

	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

// GetSubscribedDevices returns all enabled devices that are subscribed to a given source.
func (s *Service) GetSubscribedDevices(ctx context.Context, sourceID dbtypes.UUID) ([]model.Devices, error) {
	var devices []model.Devices
	stmt := SELECT(Devices.AllColumns).
		FROM(Devices.INNER_JOIN(DeviceSourceSubscriptions, DeviceSourceSubscriptions.DeviceID.EQ(Devices.ID))).
		WHERE(
			DeviceSourceSubscriptions.SourceID.EQ(String(sourceID.String())).
				AND(DeviceSourceSubscriptions.IsEnabled.EQ(Int(1))).
				AND(Devices.IsEnabled.EQ(Int(1))),
		)
	if err := stmt.QueryContext(ctx, s.db, &devices); err != nil {
		return nil, fmt.Errorf("get subscribed devices: %w", err)
	}
	if len(devices) == 0 {
		return nil, ErrNoSubscribedDevices
	}
	return devices, nil
}
