package devices

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
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
	Body devicesvc.GetDeviceRequest
}

// GetDeviceOutput describes the response body for GetDevice.
type GetDeviceOutput struct {
	Body devicesvc.GetDeviceResponse
}

// GetDevice handles POST /api/v1/devices/GetDevice.
// Returns a single device row by ID.
func GetDevice(ctx context.Context, input *GetDeviceInput, svc *devicesvc.Service) (*GetDeviceOutput, error) {
	dev, err := svc.GetDevice(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &GetDeviceOutput{Body: dev}, nil
}
