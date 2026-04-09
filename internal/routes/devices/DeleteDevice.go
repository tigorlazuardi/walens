package devices

import (
	"context"
	"path"

	"github.com/danielgtaylor/huma/v2"
	devicesvc "github.com/walens/walens/internal/services/devices"
)

// DeleteDeviceOperation returns the Huma operation metadata for DeleteDevice.
func DeleteDeviceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID:   "post-devices-delete-device",
		Method:        "POST",
		Path:          path.Join(basePath, "/api/v1/devices/DeleteDevice"),
		DefaultStatus: 204,
		Summary:       "Delete a device by ID",
		Description:   "Deletes a device row by its ID. This also cascades to delete associated device_source_subscriptions and image_assignments.",
		Tags:          []string{"devices"},
	}
}

// DeleteDeviceInput describes the request body for DeleteDevice.
type DeleteDeviceInput struct {
	Body devicesvc.DeleteDeviceRequest
}

// DeleteDevice handles POST /api/v1/devices/DeleteDevice.
// Deletes a device row by ID.
func DeleteDevice(ctx context.Context, input *DeleteDeviceInput, svc *devicesvc.Service) (*struct{}, error) {
	_, err := svc.DeleteDevice(ctx, input.Body)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
