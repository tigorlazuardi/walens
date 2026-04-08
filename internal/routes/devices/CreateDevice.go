package devices

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	devicesvc "github.com/walens/walens/internal/services/devices"
)

// CreateDeviceOperation returns the Huma operation metadata for CreateDevice.
func CreateDeviceOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-devices-create-device",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/devices/CreateDevice"),
		Summary:     "Create a new device",
		Description: "Creates a new device row with screen constraints, image bounds, filesize bounds, and adult-content preferences.",
		Tags:        []string{"devices"},
	}
}

// CreateDeviceInput describes the request body for CreateDevice.
type CreateDeviceInput struct {
	Body devicesvc.CreateDeviceInput
}

// CreateDeviceOutput describes the response body for CreateDevice.
type CreateDeviceOutput struct {
	Body devicesvc.DeviceRow
}

// CreateDevice handles POST /api/v1/devices/CreateDevice.
// Creates a new device row.
func CreateDevice(ctx context.Context, input *CreateDeviceInput, svc *devicesvc.Service) (*CreateDeviceOutput, error) {
	dev, err := svc.CreateDevice(ctx, &input.Body)
	if err != nil {
		if errors.Is(err, devicesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
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
		return nil, huma.Error500InternalServerError("failed to create device", err)
	}

	return &CreateDeviceOutput{
		Body: *dev,
	}, nil
}
