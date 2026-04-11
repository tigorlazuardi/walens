package devices

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	. "github.com/go-jet/jet/v2/sqlite"
	. "github.com/walens/walens/internal/db/generated/table"
	"github.com/walens/walens/internal/dbtypes"
)

type DeleteDeviceRequest struct {
	ID dbtypes.UUID `json:"id" required:"true" doc:"unique identifier of the device"`
}

type DeleteDeviceResponse struct{}

func (s *Service) DeleteDevice(ctx context.Context, req DeleteDeviceRequest) (DeleteDeviceResponse, error) {
	if _, err := s.GetDevice(ctx, GetDeviceRequest{ID: req.ID}); err != nil {
		return DeleteDeviceResponse{}, err
	}
	stmt := Devices.DELETE().WHERE(Devices.ID.EQ(String(req.ID.UUID.String())))
	if _, err := stmt.ExecContext(ctx, s.db); err != nil {
		return DeleteDeviceResponse{}, huma.Error500InternalServerError("failed to delete device", err)
	}
	return DeleteDeviceResponse{}, nil
}
