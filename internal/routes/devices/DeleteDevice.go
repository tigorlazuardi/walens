package devices

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	"github.com/walens/walens/internal/dbtypes"
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
	Body struct {
		ID string `json:"id" doc:"Unique device identifier (UUIDv7)."`
	}
}

// DeleteDevice handles POST /api/v1/devices/DeleteDevice.
// Deletes a device row by ID.
func DeleteDevice(ctx context.Context, input *DeleteDeviceInput, svc *devicesvc.Service) (*struct{}, error) {
	id, err := dbtypes.NewUUIDFromString(input.Body.ID)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid device ID format", err)
	}

	err = svc.DeleteDevice(ctx, id)
	if err != nil {
		if errors.Is(err, devicesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, devicesvc.ErrDeviceNotFound) {
			return nil, huma.Error404NotFound("device not found")
		}
		return nil, huma.Error500InternalServerError("failed to delete device", err)
	}

	return nil, nil
}
