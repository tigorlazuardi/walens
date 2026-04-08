package devices

import (
	"context"
	"errors"
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
	Body devicesvc.UpdateDeviceInput
}

// UpdateDeviceOutput describes the response body for UpdateDevice.
type UpdateDeviceOutput struct {
	Body devicesvc.DeviceRow
}

// UpdateDevice handles POST /api/v1/devices/UpdateDevice.
// Updates an existing device row.
func UpdateDevice(ctx context.Context, input *UpdateDeviceInput, svc *devicesvc.Service) (*UpdateDeviceOutput, error) {
	dev, err := svc.UpdateDevice(ctx, &input.Body)
	if err != nil {
		if errors.Is(err, devicesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		if errors.Is(err, devicesvc.ErrDeviceNotFound) {
			return nil, huma.Error404NotFound("device not found")
		}
		if errors.Is(err, devicesvc.ErrDuplicateDeviceSlug) {
			return nil, huma.Error409Conflict("device with this slug already exists")
		}
		if errors.Is(err, devicesvc.ErrInvalidSlug) {
			return nil, huma.Error400BadRequest("invalid slug: must contain only lowercase letters, numbers, and hyphens")
		}
		if errors.Is(err, devicesvc.ErrInvalidScreenDimensions) {
			return nil, huma.Error400BadRequest("screen width and height must be positive")
		}
		if errors.Is(err, devicesvc.ErrInvalidImageBounds) {
			return nil, huma.Error400BadRequest("min image dimensions cannot exceed max dimensions")
		}
		if errors.Is(err, devicesvc.ErrInvalidFilesizeBounds) {
			return nil, huma.Error400BadRequest("min filesize cannot exceed max filesize")
		}
		if errors.Is(err, devicesvc.ErrInvalidAspectRatioTolerance) {
			return nil, huma.Error400BadRequest("aspect ratio tolerance must be between 0 and 1")
		}
		return nil, huma.Error500InternalServerError("failed to update device", err)
	}

	return &UpdateDeviceOutput{
		Body: *dev,
	}, nil
}
