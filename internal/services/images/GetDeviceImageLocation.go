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

// GetDeviceImageLocation retrieves the location record for a specific image and device combination.
func (s *Service) GetDeviceImageLocation(ctx context.Context, imageID, deviceID dbtypes.UUID) (*model.ImageLocations, error) {
	var location model.ImageLocations
	stmt := SELECT(ImageLocations.AllColumns).
		FROM(ImageLocations).
		WHERE(ImageLocations.ImageID.EQ(String(imageID.String())).
			AND(ImageLocations.DeviceID.EQ(String(deviceID.String())))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &location); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return nil, ErrLocationNotFound
		}
		return nil, fmt.Errorf("get device image location: %w", err)
	}
	return &location, nil
}
