package devices

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	devicesvc "github.com/walens/walens/internal/services/devices"
)

// UpdateDeviceOperation returns the Huma operation metadata for UpdateDevice.
func UpdateDeviceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-devices-update-device",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/devices/UpdateDevice"),
		Summary:     "Update an existing device",
		Description: "Updates an existing device row with full-object update semantics.",
		Tags:        []string{"devices"},
	}
}

// UpdateDeviceInput describes the request body for UpdateDevice.
type UpdateDeviceInput struct {
	Body devicesvc.UpdateDeviceRequest
}

// UpdateDeviceOutput describes the response body for UpdateDevice.
type UpdateDeviceOutput struct {
	Body devicesvc.CreateDeviceResponse
}

// UpdateDevice handles POST /api/v1/devices/UpdateDevice.
// Updates an existing device row.
func UpdateDevice(ctx context.Context, input *UpdateDeviceInput, svc *devicesvc.Service) (*UpdateDeviceOutput, error) {
	dev, err := svc.UpdateDevice(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &UpdateDeviceOutput{
		Body: dev,
	}, nil
}
