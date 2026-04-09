package devices

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-jet/jet/v2/qrm"
	. "github.com/go-jet/jet/v2/sqlite"
	"github.com/walens/walens/internal/db/generated/model"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type GetDeviceRequest struct {
	ID dbtypes.UUID `json:"id" doc:"unique identifier of the device"`
}

type GetDeviceResponse = model.Devices

func (s *Service) GetDevice(ctx context.Context, req GetDeviceRequest) (GetDeviceResponse, error) {
	var dev model.Devices
	stmt := SELECT(Devices.AllColumns).
		FROM(Devices).
		WHERE(Devices.ID.EQ(String(req.ID.String()))).
		LIMIT(1)
	if err := stmt.QueryContext(ctx, s.db, &dev); err != nil {
		if errors.Is(err, qrm.ErrNoRows) {
			return GetDeviceResponse{}, huma.Error404NotFound("device not found", err)
		}
		return GetDeviceResponse{}, huma.Error500InternalServerError("failed to get device", err)
	}
	return dev, nil
}
