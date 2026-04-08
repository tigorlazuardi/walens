package devices

import (
	"context"
	"errors"
	"path"

	"github.com/danielgtaylor/huma/v2"
	devicesvc "github.com/walens/walens/internal/services/devices"
)

// ListDevicesOperation returns the Huma operation metadata for ListDevices.
func ListDevicesOperation(basePath string) huma.Operation {
	return huma.Operation{
		OperationID: "post-devices-list-devices",
		Method:      "POST",
		Path:        path.Join(basePath, "/api/v1/devices/ListDevices"),
		Summary:     "List all devices",
		Description: "Returns all device rows, ordered by name.",
		Tags:        []string{"devices"},
	}
}

// ListDevicesInput describes the request body for ListDevices.
type ListDevicesInput struct {
	Body struct{}
}

// ListDevicesOutput describes the response body for ListDevices.
type ListDevicesOutput struct {
	Body struct {
		Items []devicesvc.DeviceRow `json:"items" doc:"List of devices."`
	}
}

// ListDevices handles POST /api/v1/devices/ListDevices.
// Returns all device rows.
func ListDevices(ctx context.Context, input *ListDevicesInput, svc *devicesvc.Service) (*ListDevicesOutput, error) {
	items, err := svc.ListDevices(ctx)
	if err != nil {
		if errors.Is(err, devicesvc.ErrDBUnavailable) {
			return nil, huma.Error503ServiceUnavailable("database unavailable")
		}
		return nil, huma.Error500InternalServerError("failed to list devices", err)
	}

	return &ListDevicesOutput{
		Body: struct {
			Items []devicesvc.DeviceRow `json:"items" doc:"List of devices."`
		}{
			Items: items,
		},
	}, nil
}
