package devices

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	devicesvc "github.com/walens/walens/internal/services/devices"
)

// CreateDeviceOperation returns the Huma operation metadata for CreateDevice.
func CreateDeviceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "CreateDevice",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/devices/CreateDevice"),
		Summary:     "Create a new device",
		Description: "Creates a new device row with screen constraints, image bounds, filesize bounds, and adult-content preferences.",
		Tags:        []string{"Devices"},
	}
}

// CreateDeviceInput describes the request body for CreateDevice.
type CreateDeviceInput struct {
	Body devicesvc.CreateDeviceRequest
}

// CreateDeviceOutput describes the response body for CreateDevice.
type CreateDeviceOutput struct {
	Body devicesvc.CreateDeviceResponse
}

// CreateDevice handles POST /api/v1/devices/CreateDevice.
// Creates a new device row.
func CreateDevice(ctx context.Context, input *CreateDeviceInput, svc *devicesvc.Service) (*CreateDeviceOutput, error) {
	dev, err := svc.CreateDevice(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return &CreateDeviceOutput{
		Body: dev,
	}, nil
}
