package devices

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
	devicesvc "github.com/walens/walens/internal/services/devices"
)

// GetDeviceOperation returns the Huma operation metadata for GetDevice.
func GetDeviceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-devices-get-device",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/devices/GetDevice"),
		Summary:     "Get a device by ID",
		Description: "Returns a single device row by its ID.",
		Tags:        []string{"devices"},
	}
}

// GetDeviceInput describes the request body for GetDevice.
type GetDeviceInput struct {
	Body struct {
		ID string `json:"id" doc:"Unique device identifier (UUIDv7)."`
	}
}

// GetDeviceOutput describes the response body for GetDevice.
type GetDeviceOutput struct {
	Body devicesvc.DeviceRow
}

// GetDevice handles POST /api/v1/devices/GetDevice.
// Returns a single device row by ID.
func GetDevice(ctx context.Context, input *GetDeviceInput, svc *devicesvc.Service) (*GetDeviceOutput, error) {
	id, err := dbtypes.NewUUIDFromString(input.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid device ID format", err)
	}

	dev, err := svc.GetDevice(ctx, id)
	if err != nil {
		if errors.Is(err, devicesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, devicesvc.ErrDeviceNotFound) {
			return nil, huma.Error404NotFound("device not found")
		}
		return nil, huma.Error500InternalServerError("failed to get device", err)
	}

	return &GetDeviceOutput{
		Body: *dev,
	}, nil
}
